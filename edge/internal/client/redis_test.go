package client

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisClientConnectSuccess(t *testing.T) {
	// Start miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := NewRedisClient(mr.Addr(), "", "orion", "hexapod-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Expected successful connection, got error: %v", err)
	}

	// Verify we can ping
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Expected ping to succeed, got error: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Errorf("Expected close to succeed, got error: %v", err)
	}
}

func TestRedisClientConnectFailure(t *testing.T) {
	// Use an address that will fail to connect
	client := NewRedisClient("localhost:59999", "", "orion", "hexapod-1")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to fail with invalid address")
		client.Close()
		return
	}

	// Error should mention connection failure
	if !containsAny(err.Error(), "failed to connect", "connection refused", "refused") {
		t.Logf("Got error (expected): %v", err)
	}
}

func TestRedisPublishTelemetryFormat(t *testing.T) {
	// Start miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := NewRedisClient(mr.Addr(), "", "orion", "hexapod-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Publish telemetry
	telemetry := map[string]interface{}{
		"cpu_temp":     45.2,
		"memory_usage": 512,
		"safe_mode":    false,
	}

	err = client.PublishTelemetry(ctx, telemetry)
	if err != nil {
		t.Fatalf("Failed to publish telemetry: %v", err)
	}

	// Verify the stream was created with correct name
	streamName := "orion:edge:telemetry"
	msgs, err := mr.Stream(streamName)
	if err != nil {
		t.Fatalf("Failed to get stream: %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("Expected 1 message in stream, got %d", len(msgs))
	}

	// Convert Values slice (key-value pairs) to map
	msgValues := make(map[string]string)
	for i := 0; i < len(msgs[0].Values)-1; i += 2 {
		msgValues[msgs[0].Values[i]] = msgs[0].Values[i+1]
	}

	// Verify device_id field
	if deviceID, ok := msgValues["device_id"]; !ok || deviceID != "hexapod-1" {
		t.Errorf("Expected device_id 'hexapod-1', got %v", msgValues["device_id"])
	}

	// Verify data field contains JSON
	data, ok := msgValues["data"]
	if !ok {
		t.Fatal("Expected 'data' field in message")
	}

	var parsedData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsedData); err != nil {
		t.Fatalf("Failed to parse data JSON: %v", err)
	}

	// Verify telemetry values
	if cpuTemp, ok := parsedData["cpu_temp"].(float64); !ok || cpuTemp != 45.2 {
		t.Errorf("Expected cpu_temp 45.2, got %v", parsedData["cpu_temp"])
	}

	// Verify timestamp field exists
	if _, ok := msgValues["timestamp"]; !ok {
		t.Error("Expected 'timestamp' field in message")
	}
}

func TestRedisSubscribeCommandsHandler(t *testing.T) {
	// Start miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := NewRedisClient(mr.Addr(), "", "orion", "hexapod-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Channel to receive handled messages
	receivedMsgs := make(chan map[string]interface{}, 10)

	// Start subscriber in background
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	go func() {
		client.SubscribeCommands(subCtx, func(data map[string]interface{}) error {
			receivedMsgs <- data
			return nil
		})
	}()

	// Give subscriber time to start
	time.Sleep(100 * time.Millisecond)

	// Add a command to the stream
	streamName := "orion:edge:commands:hexapod-1"
	mr.XAdd(streamName, "*", []string{
		"command", "RESUME",
		"source", "brain",
	})

	// Wait for message to be received
	select {
	case msg := <-receivedMsgs:
		if cmd, ok := msg["command"]; !ok || cmd != "RESUME" {
			t.Errorf("Expected command 'RESUME', got %v", msg["command"])
		}
		if source, ok := msg["source"]; !ok || source != "brain" {
			t.Errorf("Expected source 'brain', got %v", msg["source"])
		}
	case <-time.After(3 * time.Second):
		t.Error("Timed out waiting for command message")
	}

	// Cancel subscriber
	subCancel()
}
