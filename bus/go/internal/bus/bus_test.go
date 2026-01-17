package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const contractsDir = "../../../contracts"

// setupTestBus creates a miniredis server and EventBus for testing
func setupTestBus(t *testing.T) (*EventBus, *redis.Client, *miniredis.Miniredis) {
	t.Helper()

	// Create miniredis server
	mr := miniredis.RunT(t)

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Cleanup
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	// Create EventBus with contracts directory
	bus, err := NewEventBus(client, contractsDir, "orion", 10000)
	if err != nil {
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	return bus, client, mr
}

func TestPublish_ValidMessage_ReturnsMessageID(t *testing.T) {
	t.Parallel()

	bus, client, _ := setupTestBus(t)
	ctx := context.Background()

	// Publish valid event message
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data":       map[string]interface{}{},
	}

	messageID, err := bus.Publish(ctx, message, "event")
	require.NoError(t, err)
	assert.NotEmpty(t, messageID)

	// Verify message exists in Redis
	streamName := "orion:events"
	messages, err := client.XRange(ctx, streamName, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	// Verify message content
	dataJSON := messages[0].Values["data"].(string)
	var storedMessage map[string]interface{}
	err = json.Unmarshal([]byte(dataJSON), &storedMessage)
	require.NoError(t, err)
	assert.Equal(t, message["event_type"], storedMessage["event_type"])
}

func TestPublish_RedisDown_ReturnsError(t *testing.T) {
	t.Parallel()

	bus, _, mr := setupTestBus(t)
	ctx := context.Background()

	// Close miniredis server to simulate Redis being down
	mr.Close()

	// Attempt to publish valid message
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data":       map[string]interface{}{},
	}

	_, err := bus.Publish(ctx, message, "event")
	assert.Error(t, err)
	// Error should be from Redis, not validation
	assert.NotContains(t, err.Error(), "contract validation failed")
}

func TestSubscribe_ConsumerGroupCreated(t *testing.T) {
	t.Parallel()

	bus, client, _ := setupTestBus(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start subscription in background
	handlerCalled := false
	handler := func(msg map[string]interface{}) error {
		handlerCalled = true
		return nil
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- bus.Subscribe(ctx, "event", "test-group", "consumer1", handler)
	}()

	// Give it time to create consumer group
	time.Sleep(100 * time.Millisecond)

	// Verify consumer group exists
	streamName := "orion:events"
	groups, err := client.XInfoGroups(ctx, streamName).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "test-group", groups[0].Name)

	// Cancel context and verify clean shutdown
	cancel()
	err = <-errChan
	assert.NoError(t, err)
	assert.False(t, handlerCalled) // No messages published, handler should not be called
}

func TestSubscribe_HandlerCalled_MessageAcknowledged(t *testing.T) {
	t.Parallel()

	bus, client, _ := setupTestBus(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Publish valid event message first
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data":       map[string]interface{}{"test": "data"},
	}
	_, err := bus.Publish(ctx, message, "event")
	require.NoError(t, err)

	// Subscribe with handler that records calls
	var receivedMessages []map[string]interface{}
	handler := func(msg map[string]interface{}) error {
		receivedMessages = append(receivedMessages, msg)
		// Delay briefly to allow XAck to complete, then cancel
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		return nil
	}

	// Start subscription
	err = bus.Subscribe(ctx, "event", "test-group", "consumer1", handler)
	assert.NoError(t, err)

	// Verify handler was called
	require.Len(t, receivedMessages, 1)
	assert.Equal(t, "service_up", receivedMessages[0]["event_type"])
	assert.NotNil(t, receivedMessages[0]["data"])

	// Verify message was acknowledged (no pending messages)
	// Use background context since our ctx is cancelled
	bgCtx := context.Background()
	streamName := "orion:events"
	pending, err := client.XPending(bgCtx, streamName, "test-group").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending.Count)
}

func TestSubscribe_ContextCancellation_ReturnsCleanly(t *testing.T) {
	t.Parallel()

	bus, _, _ := setupTestBus(t)
	ctx, cancel := context.WithCancel(context.Background())

	// Start subscription in background
	handler := func(msg map[string]interface{}) error {
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- bus.Subscribe(ctx, "event", "test-group", "consumer1", handler)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Verify Subscribe returns cleanly when context is cancelled
	// Note: With miniredis, network-level timeouts can cause delays up to 10s
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(12 * time.Second):
		t.Fatal("Subscribe did not return within 12 seconds after context cancellation")
	}
}

func TestSubscribe_HandlerError_MessageNotAcknowledged(t *testing.T) {
	t.Parallel()

	bus, client, _ := setupTestBus(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Publish valid event message
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data":       map[string]interface{}{},
	}
	_, err := bus.Publish(ctx, message, "event")
	require.NoError(t, err)

	// Subscribe with handler that always fails
	callCount := 0
	handler := func(msg map[string]interface{}) error {
		callCount++
		return assert.AnError // Always fail - message should not be ack'd
	}

	// Start subscription (will timeout after 2 seconds)
	_ = bus.Subscribe(ctx, "event", "test-group", "consumer1", handler)

	// Handler should be called at least once
	assert.GreaterOrEqual(t, callCount, 1)

	// Message should still be pending (not acknowledged)
	bgCtx := context.Background()
	streamName := "orion:events"
	pending, err := client.XPending(bgCtx, streamName, "test-group").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), pending.Count) // One message still pending
}

func TestPublish_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	bus, _, _ := setupTestBus(t)
	ctx := context.Background()

	// Create valid message structure but with channel in data field
	// This will pass validation but fail JSON marshaling
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data": map[string]interface{}{
			"channel": make(chan int), // This will cause JSON marshal error
		},
	}

	_, err := bus.Publish(ctx, message, "event")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal")
}

func TestPublish_InvalidMessage_ReturnsValidationError(t *testing.T) {
	t.Parallel()

	bus, client, _ := setupTestBus(t)
	ctx := context.Background()

	// Message missing required field "event_id"
	message := map[string]interface{}{
		"version":    "1.0",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data":       map[string]interface{}{},
	}

	_, err := bus.Publish(ctx, message, "event")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contract validation failed")

	// Verify message was NOT published to Redis
	streamName := "orion:events"
	messages, err := client.XRange(ctx, streamName, "-", "+").Result()
	require.NoError(t, err)
	assert.Len(t, messages, 0, "Invalid message should not be published")
}
