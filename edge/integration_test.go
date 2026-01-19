//go:build integration

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/yourusername/orion/edge/internal/client"
	"github.com/yourusername/orion/edge/internal/safety"
)

// TestAgentStartsAndPublishesHealth verifies the agent starts cleanly,
// health endpoint returns OK, and heartbeat messages are published to Redis.
func TestAgentStartsAndPublishesHealth(t *testing.T) {
	// Start miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create components (like main.go does)
	safeState := safety.NewSafeStateManager(
		func() {},
		func() {},
	)

	watchdog := safety.NewDeadManSwitch(
		5*time.Second,
		func() { safeState.EnterSafeMode() },
	)
	defer watchdog.Stop()

	// Create Redis client
	redisClient := client.NewRedisClient(mr.Addr(), "", "orion", "test-device")
	if err := redisClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Reset watchdog on connection
	watchdog.Reset()

	// Create HTTP health endpoint (like main.go)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "ok",
			"service":            "orion-edge",
			"device_id":          "test-device",
			"safe_mode":          safeState.IsInSafeMode(),
			"watchdog_triggered": watchdog.IsTriggered(),
		})
	})

	// Start test server
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Test health endpoint returns OK
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", health["status"])
	}

	if health["safe_mode"] != false {
		t.Errorf("Expected safe_mode false, got %v", health["safe_mode"])
	}

	if health["watchdog_triggered"] != false {
		t.Errorf("Expected watchdog_triggered false, got %v", health["watchdog_triggered"])
	}

	// Publish a telemetry message (simulating heartbeat)
	telemetry := map[string]interface{}{
		"device_id": "test-device",
		"state":     "RUNNING",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if err := redisClient.PublishTelemetry(ctx, telemetry); err != nil {
		t.Fatalf("Failed to publish telemetry: %v", err)
	}

	// Verify message was published to Redis stream
	msgs, err := mr.Stream("orion:edge:telemetry")
	if err != nil {
		t.Fatalf("Failed to read stream: %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("Expected 1 message in stream, got %d", len(msgs))
	}
}

// TestWatchdogTriggersOnNoReset verifies the Dead Man's Switch triggers safe mode
// when not reset within the timeout period.
func TestWatchdogTriggersOnNoReset(t *testing.T) {
	// Start miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track callback invocation
	var safeModeEntered atomic.Bool

	safeState := safety.NewSafeStateManager(
		func() { safeModeEntered.Store(true) },
		func() {},
	)

	// Create watchdog with short timeout (100ms for testing)
	watchdog := safety.NewDeadManSwitch(
		100*time.Millisecond,
		func() { safeState.EnterSafeMode() },
	)
	defer watchdog.Stop()

	// Connect to Redis
	redisClient := client.NewRedisClient(mr.Addr(), "", "orion", "test-device")
	if err := redisClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Create health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"safe_mode":          safeState.IsInSafeMode(),
			"watchdog_triggered": watchdog.IsTriggered(),
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// DO NOT reset the watchdog - wait for it to trigger
	time.Sleep(200 * time.Millisecond)

	// Verify safe mode was entered
	if !safeModeEntered.Load() {
		t.Error("Expected safe mode to be entered after watchdog timeout")
	}

	if !safeState.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be true")
	}

	if !watchdog.IsTriggered() {
		t.Error("Expected IsTriggered() to be true")
	}

	// Verify health endpoint reflects safe mode
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["safe_mode"] != true {
		t.Errorf("Expected health safe_mode true, got %v", health["safe_mode"])
	}

	if health["watchdog_triggered"] != true {
		t.Errorf("Expected health watchdog_triggered true, got %v", health["watchdog_triggered"])
	}
}

// TestCommandResetsWatchdog verifies that receiving commands resets the watchdog
// and prevents safe mode from being triggered.
func TestCommandResetsWatchdog(t *testing.T) {
	// Start miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track callback invocation
	var safeModeEntered atomic.Bool

	safeState := safety.NewSafeStateManager(
		func() { safeModeEntered.Store(true) },
		func() {},
	)

	// Create watchdog with short timeout (100ms for testing)
	watchdog := safety.NewDeadManSwitch(
		100*time.Millisecond,
		func() { safeState.EnterSafeMode() },
	)
	defer watchdog.Stop()

	// Connect to Redis
	redisClient := client.NewRedisClient(mr.Addr(), "", "orion", "test-device")
	if err := redisClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Create health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"safe_mode":          safeState.IsInSafeMode(),
			"watchdog_triggered": watchdog.IsTriggered(),
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Simulate receiving commands by resetting watchdog periodically
	for i := 0; i < 5; i++ {
		time.Sleep(50 * time.Millisecond)
		// Command received - reset watchdog (simulates handleCommands behavior)
		watchdog.Reset()
	}

	// Wait a bit more but less than timeout since last reset
	time.Sleep(50 * time.Millisecond)

	// Verify safe mode was NOT entered
	if safeModeEntered.Load() {
		t.Error("Expected safe mode to NOT be entered when watchdog is reset by commands")
	}

	if safeState.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be false")
	}

	if watchdog.IsTriggered() {
		t.Error("Expected IsTriggered() to be false")
	}

	// Verify health endpoint reflects normal state
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["safe_mode"] != false {
		t.Errorf("Expected health safe_mode false, got %v", health["safe_mode"])
	}

	if health["watchdog_triggered"] != false {
		t.Errorf("Expected health watchdog_triggered false, got %v", health["watchdog_triggered"])
	}
}

// TestResumeCommandExitsSafeMode verifies that the RESUME command clears the
// watchdog triggered state and exits safe mode.
func TestResumeCommandExitsSafeMode(t *testing.T) {
	// Start miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track callback invocations
	var safeModeEntered atomic.Bool
	var safeModeExited atomic.Bool

	safeState := safety.NewSafeStateManager(
		func() { safeModeEntered.Store(true) },
		func() { safeModeExited.Store(true) },
	)

	// Create watchdog with short timeout (100ms for testing)
	watchdog := safety.NewDeadManSwitch(
		100*time.Millisecond,
		func() { safeState.EnterSafeMode() },
	)
	defer watchdog.Stop()

	// Connect to Redis
	redisClient := client.NewRedisClient(mr.Addr(), "", "orion", "test-device")
	if err := redisClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Create health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"safe_mode":          safeState.IsInSafeMode(),
			"watchdog_triggered": watchdog.IsTriggered(),
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Wait for watchdog to trigger safe mode
	time.Sleep(200 * time.Millisecond)

	// Verify safe mode was entered
	if !safeModeEntered.Load() {
		t.Fatal("Expected safe mode to be entered after watchdog timeout")
	}

	if !safeState.IsInSafeMode() {
		t.Fatal("Expected IsInSafeMode() to be true")
	}

	// Verify health shows safe mode
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	resp.Body.Close()

	// Simulate RESUME command handling (from handleResumeCommand in main.go)
	// This clears the watchdog and exits safe mode
	watchdog.ClearTriggered()
	if err := safeState.ExitSafeMode(); err != nil {
		t.Fatalf("Failed to exit safe mode: %v", err)
	}

	// Verify safe mode was exited
	if !safeModeExited.Load() {
		t.Error("Expected safe mode exit callback to be called")
	}

	if safeState.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be false after RESUME")
	}

	if watchdog.IsTriggered() {
		t.Error("Expected IsTriggered() to be false after RESUME")
	}

	// Verify health endpoint reflects resumed state
	resp, err = http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["safe_mode"] != false {
		t.Errorf("Expected health safe_mode false after RESUME, got %v", health["safe_mode"])
	}

	if health["watchdog_triggered"] != false {
		t.Errorf("Expected health watchdog_triggered false after RESUME, got %v", health["watchdog_triggered"])
	}
}
