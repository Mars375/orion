package router

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/contracts"
)

func setupRouterTest(t *testing.T) (*miniredis.Miniredis, *redis.Client, *InferenceRouter, *StickyRouter) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	healthReader := NewHealthReader(client, "")
	stickyRouter := NewStickyRouter(healthReader)
	inferenceRouter := NewInferenceRouter(client, stickyRouter, "orion:inference")

	return mr, client, inferenceRouter, stickyRouter
}

func TestNewInferenceRouter(t *testing.T) {
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	if router.streamPrefix != "orion:inference" {
		t.Errorf("streamPrefix = %v, want orion:inference", router.streamPrefix)
	}

	if router.sticky == nil {
		t.Error("StickyRouter should not be nil")
	}
}

func TestNewInferenceRouter_DefaultPrefix(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	healthReader := NewHealthReader(client, "")
	stickyRouter := NewStickyRouter(healthReader)
	router := NewInferenceRouter(client, stickyRouter, "")

	if router.streamPrefix != "orion:inference" {
		t.Errorf("Default streamPrefix = %v, want orion:inference", router.streamPrefix)
	}
}

func TestRouteRequest_Success(t *testing.T) {
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Add a healthy node
	health := contracts.NodeHealth{
		NodeID:      "pi-8g",
		RAMPercent:  50.0,
		TempCelsius: 50.0,
		Models:      []string{"gemma3:1b"},
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}
	healthData, _ := json.Marshal(health)
	client.HSet(ctx, DefaultHealthHashKey, "pi-8g", string(healthData))

	// Create a request
	req := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "test-123",
		Model:     "gemma3:1b",
		Messages:  []contracts.Message{{Role: "user", Content: "Hello"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}

	// Route the request
	err := router.RouteRequest(ctx, req)
	if err != nil {
		t.Fatalf("RouteRequest() error = %v", err)
	}

	// Verify request was dispatched to worker stream
	workerStream := router.GetWorkerStreamName("pi-8g")
	messages, err := client.XRange(ctx, workerStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("Failed to read worker stream: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message in worker stream, got %d", len(messages))
	}

	// Verify message content
	msg := messages[0]
	if msg.Values["request_id"] != "test-123" {
		t.Errorf("request_id = %v, want test-123", msg.Values["request_id"])
	}
	if msg.Values["model"] != "gemma3:1b" {
		t.Errorf("model = %v, want gemma3:1b", msg.Values["model"])
	}
}

func TestRouteRequest_NoAvailableNodes(t *testing.T) {
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Don't add any nodes
	req := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "test-456",
		Model:     "gemma3:1b",
		Messages:  []contracts.Message{{Role: "user", Content: "Hello"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}

	// Route should fail
	err := router.RouteRequest(ctx, req)
	if err == nil {
		t.Error("RouteRequest() should fail when no nodes available")
	}

	// Check stats
	stats := router.GetRoutingStats()
	if stats["errors"].(int64) != 1 {
		t.Errorf("errors = %v, want 1", stats["errors"])
	}
}

func TestRouteRequest_StickyHitStats(t *testing.T) {
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Add a node with the model
	health := contracts.NodeHealth{
		NodeID:      "pi-8g",
		RAMPercent:  50.0,
		TempCelsius: 50.0,
		Models:      []string{"gemma3:1b"},
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}
	healthData, _ := json.Marshal(health)
	client.HSet(ctx, DefaultHealthHashKey, "pi-8g", string(healthData))

	// Route a request
	req := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "test-sticky",
		Model:     "gemma3:1b",
		Messages:  []contracts.Message{{Role: "user", Content: "Hello"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}

	err := router.RouteRequest(ctx, req)
	if err != nil {
		t.Fatalf("RouteRequest() error = %v", err)
	}

	// Check stats - should be a sticky hit
	stats := router.GetRoutingStats()
	if stats["total_routed"].(int64) != 1 {
		t.Errorf("total_routed = %v, want 1", stats["total_routed"])
	}
	if stats["sticky_hits"].(int64) != 1 {
		t.Errorf("sticky_hits = %v, want 1", stats["sticky_hits"])
	}
	if stats["fallbacks"].(int64) != 0 {
		t.Errorf("fallbacks = %v, want 0", stats["fallbacks"])
	}
}

func TestRouteRequest_FallbackStats(t *testing.T) {
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Add a node WITHOUT the model
	health := contracts.NodeHealth{
		NodeID:      "pi-8g",
		RAMPercent:  50.0,
		TempCelsius: 50.0,
		Models:      []string{}, // No models
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}
	healthData, _ := json.Marshal(health)
	client.HSet(ctx, DefaultHealthHashKey, "pi-8g", string(healthData))

	// Route a request for model not loaded
	req := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "test-fallback",
		Model:     "llama3.2:3b",
		Messages:  []contracts.Message{{Role: "user", Content: "Hello"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}

	err := router.RouteRequest(ctx, req)
	if err != nil {
		t.Fatalf("RouteRequest() error = %v", err)
	}

	// Check stats - should be a fallback
	stats := router.GetRoutingStats()
	if stats["total_routed"].(int64) != 1 {
		t.Errorf("total_routed = %v, want 1", stats["total_routed"])
	}
	if stats["sticky_hits"].(int64) != 0 {
		t.Errorf("sticky_hits = %v, want 0", stats["sticky_hits"])
	}
	if stats["fallbacks"].(int64) != 1 {
		t.Errorf("fallbacks = %v, want 1", stats["fallbacks"])
	}
}

func TestGetWorkerStreamName(t *testing.T) {
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	streamName := router.GetWorkerStreamName("pi-8g")
	expected := "orion:inference:requests:pi-8g"

	if streamName != expected {
		t.Errorf("GetWorkerStreamName() = %v, want %v", streamName, expected)
	}
}

func TestGetRequestStreamName(t *testing.T) {
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	streamName := router.GetRequestStreamName()
	expected := "orion:inference:requests"

	if streamName != expected {
		t.Errorf("GetRequestStreamName() = %v, want %v", streamName, expected)
	}
}

func TestRouteRequest_MultipleRequests(t *testing.T) {
	// This test verifies the router can handle multiple requests
	// and routes them to the correct worker streams.
	mr, client, router, _ := setupRouterTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Add two healthy nodes
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "pi-16g",
			RAMPercent:  40.0,
			TempCelsius: 45.0,
			Models:      []string{"llama3.2:3b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-8g",
			RAMPercent:  60.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		healthData, _ := json.Marshal(health)
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(healthData))
	}

	// Route request for gemma3:1b - should go to pi-8g (sticky)
	req1 := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "test-1",
		Model:     "gemma3:1b",
		Messages:  []contracts.Message{{Role: "user", Content: "Test 1"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}
	err := router.RouteRequest(ctx, req1)
	if err != nil {
		t.Fatalf("RouteRequest(gemma3:1b) error = %v", err)
	}

	// Route request for llama3.2:3b - should go to pi-16g (sticky)
	req2 := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "test-2",
		Model:     "llama3.2:3b",
		Messages:  []contracts.Message{{Role: "user", Content: "Test 2"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}
	err = router.RouteRequest(ctx, req2)
	if err != nil {
		t.Fatalf("RouteRequest(llama3.2:3b) error = %v", err)
	}

	// Verify each request went to the correct stream
	pi8gStream := router.GetWorkerStreamName("pi-8g")
	pi16gStream := router.GetWorkerStreamName("pi-16g")

	pi8gMsgs, _ := client.XRange(ctx, pi8gStream, "-", "+").Result()
	pi16gMsgs, _ := client.XRange(ctx, pi16gStream, "-", "+").Result()

	if len(pi8gMsgs) != 1 {
		t.Errorf("pi-8g stream has %d messages, want 1", len(pi8gMsgs))
	}
	if len(pi16gMsgs) != 1 {
		t.Errorf("pi-16g stream has %d messages, want 1", len(pi16gMsgs))
	}

	// Check stats
	stats := router.GetRoutingStats()
	if stats["total_routed"].(int64) != 2 {
		t.Errorf("total_routed = %v, want 2", stats["total_routed"])
	}
	if stats["sticky_hits"].(int64) != 2 {
		t.Errorf("sticky_hits = %v, want 2", stats["sticky_hits"])
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice    []string
		val      string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}

	for _, tt := range tests {
		if got := contains(tt.slice, tt.val); got != tt.expected {
			t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.val, got, tt.expected)
		}
	}
}
