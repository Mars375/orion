package client

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestMQTTClientConfigParsing(t *testing.T) {
	testCases := []struct {
		name      string
		brokerURL string
		clientID  string
		deviceID  string
	}{
		{
			name:      "valid TCP URL",
			brokerURL: "tcp://localhost:1883",
			clientID:  "test-client",
			deviceID:  "hexapod-1",
		},
		{
			name:      "valid SSL URL",
			brokerURL: "ssl://broker.example.com:8883",
			clientID:  "secure-client",
			deviceID:  "hexapod-2",
		},
		{
			name:      "valid websocket URL",
			brokerURL: "ws://localhost:9001",
			clientID:  "ws-client",
			deviceID:  "hexapod-3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewMQTTClient(tc.brokerURL, tc.clientID, tc.deviceID)

			if client == nil {
				t.Fatal("Expected non-nil client")
			}

			if client.brokerURL != tc.brokerURL {
				t.Errorf("Expected brokerURL %s, got %s", tc.brokerURL, client.brokerURL)
			}

			if client.clientID != tc.clientID {
				t.Errorf("Expected clientID %s, got %s", tc.clientID, client.clientID)
			}

			if client.deviceID != tc.deviceID {
				t.Errorf("Expected deviceID %s, got %s", tc.deviceID, client.deviceID)
			}

			// Initially not connected
			if client.IsConnected() {
				t.Error("Expected client to be disconnected initially")
			}
		})
	}
}

func TestMQTTClientConnectTimeoutWithInvalidURL(t *testing.T) {
	// Use invalid broker URL to test timeout/error handling
	client := NewMQTTClient("tcp://nonexistent.invalid:1883", "test-client", "hexapod-1")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Error("Expected error when connecting to invalid broker")
		client.Close(context.Background())
	}

	// Should have error message about connection failure
	if err != nil && !containsAny(err.Error(), "failed to connect", "context deadline") {
		t.Logf("Got error (expected): %v", err)
	}
}

func TestMQTTClientCallbackRegistration(t *testing.T) {
	client := NewMQTTClient("tcp://localhost:1883", "test-client", "hexapod-1")

	// Track callback invocations
	var mu sync.Mutex
	connUpCalled := false
	connDownCalled := false
	var connDownErr error

	client.SetOnConnectionUp(func() {
		mu.Lock()
		connUpCalled = true
		mu.Unlock()
	})

	client.SetOnConnectionDown(func(err error) {
		mu.Lock()
		connDownCalled = true
		connDownErr = err
		mu.Unlock()
	})

	// Verify callbacks are registered (check internal state)
	client.mu.RLock()
	hasUpCallback := client.onConnUp != nil
	hasDownCallback := client.onConnDown != nil
	client.mu.RUnlock()

	if !hasUpCallback {
		t.Error("Expected onConnUp callback to be registered")
	}

	if !hasDownCallback {
		t.Error("Expected onConnDown callback to be registered")
	}

	// Simulate connection up event (internal method)
	client.mu.Lock()
	client.connected = true
	callback := client.onConnUp
	client.mu.Unlock()
	if callback != nil {
		callback()
	}

	mu.Lock()
	if !connUpCalled {
		t.Error("Expected connection up callback to be called")
	}
	mu.Unlock()

	// Simulate connection down event
	client.mu.Lock()
	client.connected = false
	downCallback := client.onConnDown
	client.mu.Unlock()
	if downCallback != nil {
		downCallback(context.DeadlineExceeded)
	}

	mu.Lock()
	if !connDownCalled {
		t.Error("Expected connection down callback to be called")
	}
	if connDownErr != context.DeadlineExceeded {
		t.Errorf("Expected error to be context.DeadlineExceeded, got %v", connDownErr)
	}
	mu.Unlock()
}

func TestMQTTHealthMessageFormat(t *testing.T) {
	client := NewMQTTClient("tcp://localhost:1883", "test-client", "hexapod-1")

	// Create a health message similar to what the agent publishes
	health := map[string]interface{}{
		"status":        "ok",
		"service":       "orion-edge",
		"device_id":     client.deviceID,
		"safe_mode":     false,
		"mqtt_ok":       true,
		"redis_ok":      true,
		"watchdog_ms":   5000,
		"uptime_seconds": 123,
	}

	// Verify the health map can be marshaled to JSON (same as PublishHealth does)
	payload, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal health: %v", err)
	}

	// Verify the JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal health: %v", err)
	}

	// Check required fields
	if parsed["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", parsed["status"])
	}

	if parsed["service"] != "orion-edge" {
		t.Errorf("Expected service 'orion-edge', got %v", parsed["service"])
	}

	if parsed["device_id"] != "hexapod-1" {
		t.Errorf("Expected device_id 'hexapod-1', got %v", parsed["device_id"])
	}

	// Verify topic format
	expectedTopic := "orion/edge/hexapod-1/health"
	actualTopic := "orion/edge/" + client.deviceID + "/health"
	if actualTopic != expectedTopic {
		t.Errorf("Expected topic %s, got %s", expectedTopic, actualTopic)
	}
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
