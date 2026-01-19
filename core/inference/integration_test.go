//go:build integration

package inference

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/contracts"
	"github.com/orion/core/inference/router"
	"github.com/orion/core/inference/worker"
)

func setupIntegrationTest(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

// TestWorkerHealthPublishing verifies that workers publish health to Redis correctly.
func TestWorkerHealthPublishing(t *testing.T) {
	mr, client := setupIntegrationTest(t)
	defer mr.Close()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create worker agent
	agent, err := worker.NewWorkerAgent("pi-8g", "http://localhost:11434", client, "orion:inference")
	if err != nil {
		t.Fatalf("Failed to create worker agent: %v", err)
	}

	// Start agent in background
	go agent.Start(ctx)

	// Wait for health to be published (CPU metrics take ~1 second to collect)
	time.Sleep(2 * time.Second)

	// Verify health was published to Redis
	data, err := client.HGet(ctx, worker.DefaultHealthHashKey, "pi-8g").Result()
	if err == redis.Nil {
		t.Fatal("Health entry not found in Redis")
	}
	if err != nil {
		t.Fatalf("Failed to get health from Redis: %v", err)
	}

	var health contracts.NodeHealth
	if err := json.Unmarshal([]byte(data), &health); err != nil {
		t.Fatalf("Failed to unmarshal health: %v", err)
	}

	// Verify health data
	if health.NodeID != "pi-8g" {
		t.Errorf("NodeID = %v, want pi-8g", health.NodeID)
	}

	if health.RAMTotalMB == 0 {
		t.Error("RAMTotalMB should be > 0")
	}

	// Clean up
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestRouterStickyRouting verifies sticky routing prefers nodes with model resident.
func TestRouterStickyRouting(t *testing.T) {
	mr, client := setupIntegrationTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Seed health registry with two nodes
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "pi-8g",
			RAMPercent:  60.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"}, // Has model loaded
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-16g",
			RAMPercent:  40.0, // Lower RAM (would be preferred without sticky)
			TempCelsius: 45.0,
			Models:      []string{}, // No models
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, router.DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Create router components
	healthReader := router.NewHealthReader(client, "")
	stickyRouter := router.NewStickyRouter(healthReader)
	inferenceRouter := router.NewInferenceRouter(client, stickyRouter, "orion:inference")

	// Submit request for gemma3:1b
	req := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "sticky-test-123",
		Model:     "gemma3:1b",
		Messages:  []contracts.Message{{Role: "user", Content: "Hello"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}

	err := inferenceRouter.RouteRequest(ctx, req)
	if err != nil {
		t.Fatalf("RouteRequest() error = %v", err)
	}

	// Verify request went to pi-8g (has model) not pi-16g (lower RAM)
	pi8gStream := inferenceRouter.GetWorkerStreamName("pi-8g")
	pi16gStream := inferenceRouter.GetWorkerStreamName("pi-16g")

	pi8gMsgs, _ := client.XRange(ctx, pi8gStream, "-", "+").Result()
	pi16gMsgs, _ := client.XRange(ctx, pi16gStream, "-", "+").Result()

	if len(pi8gMsgs) != 1 {
		t.Errorf("pi-8g should have 1 message (sticky hit), got %d", len(pi8gMsgs))
	}
	if len(pi16gMsgs) != 0 {
		t.Errorf("pi-16g should have 0 messages, got %d", len(pi16gMsgs))
	}
}

// TestRouterFallbackRouting verifies fallback to least-loaded when model not resident.
func TestRouterFallbackRouting(t *testing.T) {
	mr, client := setupIntegrationTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Seed health registry - neither has the requested model
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "pi-8g",
			RAMPercent:  70.0,
			TempCelsius: 50.0,
			Models:      []string{"other-model"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-16g",
			RAMPercent:  40.0, // Lower RAM - should be selected
			TempCelsius: 45.0,
			Models:      []string{"different-model"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, router.DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Create router
	healthReader := router.NewHealthReader(client, "")
	stickyRouter := router.NewStickyRouter(healthReader)
	inferenceRouter := router.NewInferenceRouter(client, stickyRouter, "orion:inference")

	// Submit request for model not loaded anywhere
	req := contracts.InferenceRequest{
		Version:   "1.0",
		RequestID: "fallback-test-456",
		Model:     "llama3.2:3b",
		Messages:  []contracts.Message{{Role: "user", Content: "Hello"}},
		Timestamp: time.Now().UTC(),
		Source:    "test",
	}

	err := inferenceRouter.RouteRequest(ctx, req)
	if err != nil {
		t.Fatalf("RouteRequest() error = %v", err)
	}

	// Verify request went to pi-16g (least loaded)
	pi16gStream := inferenceRouter.GetWorkerStreamName("pi-16g")
	pi16gMsgs, _ := client.XRange(ctx, pi16gStream, "-", "+").Result()

	if len(pi16gMsgs) != 1 {
		t.Errorf("pi-16g should have 1 message (fallback), got %d", len(pi16gMsgs))
	}
}

// TestHealthThresholdEnforcement verifies nodes are excluded when thresholds exceeded.
func TestHealthThresholdEnforcement(t *testing.T) {
	mr, client := setupIntegrationTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	t.Run("high temperature excludes node", func(t *testing.T) {
		// Clear any existing data
		client.FlushDB(ctx)

		// Add node with high temperature
		hotNode := contracts.NodeHealth{
			NodeID:      "hot-node",
			RAMPercent:  50.0,
			TempCelsius: 80.0, // Over 75°C threshold
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		}
		hotData, _ := json.Marshal(hotNode)
		client.HSet(ctx, router.DefaultHealthHashKey, "hot-node", string(hotData))

		// Create router
		healthReader := router.NewHealthReader(client, "")
		stickyRouter := router.NewStickyRouter(healthReader)

		// Try to select node
		_, err := stickyRouter.SelectNode(ctx, "gemma3:1b")
		if err != router.ErrNoAvailableNodes {
			t.Errorf("SelectNode() should return ErrNoAvailableNodes for hot node, got %v", err)
		}
	})

	t.Run("high RAM excludes node", func(t *testing.T) {
		// Clear any existing data
		client.FlushDB(ctx)

		// Add node with high RAM usage
		overloadedNode := contracts.NodeHealth{
			NodeID:      "overloaded-node",
			RAMPercent:  95.0, // Over 90% threshold
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		}
		overloadedData, _ := json.Marshal(overloadedNode)
		client.HSet(ctx, router.DefaultHealthHashKey, "overloaded-node", string(overloadedData))

		// Create router
		healthReader := router.NewHealthReader(client, "")
		stickyRouter := router.NewStickyRouter(healthReader)

		// Try to select node
		_, err := stickyRouter.SelectNode(ctx, "gemma3:1b")
		if err != router.ErrNoAvailableNodes {
			t.Errorf("SelectNode() should return ErrNoAvailableNodes for overloaded node, got %v", err)
		}
	})

	t.Run("unavailable flag excludes node", func(t *testing.T) {
		// Clear any existing data
		client.FlushDB(ctx)

		// Add unavailable node
		unavailableNode := contracts.NodeHealth{
			NodeID:      "unavailable-node",
			RAMPercent:  50.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   false, // Marked unavailable
			LastSeen:    time.Now().UTC(),
		}
		unavailableData, _ := json.Marshal(unavailableNode)
		client.HSet(ctx, router.DefaultHealthHashKey, "unavailable-node", string(unavailableData))

		// Create router
		healthReader := router.NewHealthReader(client, "")
		stickyRouter := router.NewStickyRouter(healthReader)

		// Try to select node
		_, err := stickyRouter.SelectNode(ctx, "gemma3:1b")
		if err != router.ErrNoAvailableNodes {
			t.Errorf("SelectNode() should return ErrNoAvailableNodes for unavailable node, got %v", err)
		}
	})

	t.Run("stale data excludes node", func(t *testing.T) {
		// Clear any existing data
		client.FlushDB(ctx)

		// Add stale node
		staleNode := contracts.NodeHealth{
			NodeID:      "stale-node",
			RAMPercent:  50.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().Add(-30 * time.Second).UTC(), // Stale
		}
		staleData, _ := json.Marshal(staleNode)
		client.HSet(ctx, router.DefaultHealthHashKey, "stale-node", string(staleData))

		// Create router
		healthReader := router.NewHealthReader(client, "")
		stickyRouter := router.NewStickyRouter(healthReader)

		// Try to select node
		_, err := stickyRouter.SelectNode(ctx, "gemma3:1b")
		if err != router.ErrNoAvailableNodes {
			t.Errorf("SelectNode() should return ErrNoAvailableNodes for stale node, got %v", err)
		}
	})
}

// TestEndToEndRouting verifies full routing flow from request to worker stream.
func TestEndToEndRouting(t *testing.T) {
	mr, client := setupIntegrationTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set up two healthy workers
	workers := []contracts.NodeHealth{
		{
			NodeID:      "worker-a",
			RAMPercent:  50.0,
			TempCelsius: 45.0,
			Models:      []string{"model-a", "model-b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "worker-b",
			RAMPercent:  60.0,
			TempCelsius: 48.0,
			Models:      []string{"model-c"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, w := range workers {
		data, _ := json.Marshal(w)
		client.HSet(ctx, router.DefaultHealthHashKey, w.NodeID, string(data))
	}

	// Create router
	healthReader := router.NewHealthReader(client, "")
	stickyRouter := router.NewStickyRouter(healthReader)
	inferenceRouter := router.NewInferenceRouter(client, stickyRouter, "orion:inference")

	// Route multiple requests
	requests := []struct {
		model          string
		expectedWorker string
	}{
		{"model-a", "worker-a"}, // Sticky to worker-a
		{"model-b", "worker-a"}, // Sticky to worker-a
		{"model-c", "worker-b"}, // Sticky to worker-b
		{"model-d", "worker-a"}, // Fallback to least loaded (worker-a at 50%)
	}

	for i, test := range requests {
		req := contracts.InferenceRequest{
			Version:   "1.0",
			RequestID: "e2e-test-" + string(rune('0'+i)),
			Model:     test.model,
			Messages:  []contracts.Message{{Role: "user", Content: "Test"}},
			Timestamp: time.Now().UTC(),
			Source:    "test",
		}

		err := inferenceRouter.RouteRequest(ctx, req)
		if err != nil {
			t.Fatalf("RouteRequest(%s) error = %v", test.model, err)
		}
	}

	// Verify distribution
	workerAStream := inferenceRouter.GetWorkerStreamName("worker-a")
	workerBStream := inferenceRouter.GetWorkerStreamName("worker-b")

	workerAMsgs, _ := client.XRange(ctx, workerAStream, "-", "+").Result()
	workerBMsgs, _ := client.XRange(ctx, workerBStream, "-", "+").Result()

	// worker-a should have 3 messages (model-a, model-b, model-d fallback)
	// worker-b should have 1 message (model-c)
	if len(workerAMsgs) != 3 {
		t.Errorf("worker-a should have 3 messages, got %d", len(workerAMsgs))
	}
	if len(workerBMsgs) != 1 {
		t.Errorf("worker-b should have 1 message, got %d", len(workerBMsgs))
	}
}
