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

func setupStickyTest(t *testing.T) (*miniredis.Miniredis, *redis.Client, *StickyRouter) {
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

	return mr, client, stickyRouter
}

func addTestNodes(t *testing.T, client *redis.Client, nodes []contracts.NodeHealth) {
	t.Helper()
	ctx := context.Background()
	for _, health := range nodes {
		data, err := json.Marshal(health)
		if err != nil {
			t.Fatalf("Failed to marshal health: %v", err)
		}
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(data))
	}
}

func TestNewStickyRouter(t *testing.T) {
	mr, client, router := setupStickyTest(t)
	defer mr.Close()
	defer client.Close()

	if router.healthReader == nil {
		t.Error("StickyRouter healthReader should not be nil")
	}
}

func TestSelectNode_StickyHit(t *testing.T) {
	mr, client, router := setupStickyTest(t)
	defer mr.Close()
	defer client.Close()

	// Add nodes - one with model loaded, one without
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "pi-8g",
			RAMPercent:  60.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-16g",
			RAMPercent:  40.0, // Lower RAM, but no model
			TempCelsius: 45.0,
			Models:      []string{}, // No models loaded
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}
	addTestNodes(t, client, nodes)

	ctx := context.Background()

	// Select node for gemma3:1b - should get pi-8g (sticky hit)
	nodeID, err := router.SelectNode(ctx, "gemma3:1b")
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if nodeID != "pi-8g" {
		t.Errorf("SelectNode(gemma3:1b) = %v, want pi-8g (sticky hit)", nodeID)
	}
}

func TestSelectNode_FallbackToLeastLoaded(t *testing.T) {
	mr, client, router := setupStickyTest(t)
	defer mr.Close()
	defer client.Close()

	// Add nodes - neither has the requested model
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
			RAMPercent:  40.0, // Lower RAM
			TempCelsius: 45.0,
			Models:      []string{"different-model"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}
	addTestNodes(t, client, nodes)

	ctx := context.Background()

	// Select node for llama3.2:3b - should fall back to least loaded (pi-16g)
	nodeID, err := router.SelectNode(ctx, "llama3.2:3b")
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if nodeID != "pi-16g" {
		t.Errorf("SelectNode(llama3.2:3b) = %v, want pi-16g (least loaded fallback)", nodeID)
	}
}

func TestSelectNode_NoAvailableNodes(t *testing.T) {
	mr, client, router := setupStickyTest(t)
	defer mr.Close()
	defer client.Close()

	// Don't add any nodes
	ctx := context.Background()

	_, err := router.SelectNode(ctx, "any-model")
	if err != ErrNoAvailableNodes {
		t.Errorf("SelectNode() error = %v, want ErrNoAvailableNodes", err)
	}
}

func TestSelectNode_PreferLeastLoadedAmongModelResidents(t *testing.T) {
	mr, client, router := setupStickyTest(t)
	defer mr.Close()
	defer client.Close()

	// Add multiple nodes with the same model - should pick least loaded
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "high-load",
			RAMPercent:  80.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "low-load",
			RAMPercent:  30.0, // Least loaded
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "medium-load",
			RAMPercent:  55.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}
	addTestNodes(t, client, nodes)

	ctx := context.Background()

	// Select node for gemma3:1b - should get low-load (least loaded with model)
	nodeID, err := router.SelectNode(ctx, "gemma3:1b")
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if nodeID != "low-load" {
		t.Errorf("SelectNode(gemma3:1b) = %v, want low-load (least loaded with model)", nodeID)
	}
}

func TestSelectNode_IgnoresUnhealthyNodesWithModel(t *testing.T) {
	mr, client, router := setupStickyTest(t)
	defer mr.Close()
	defer client.Close()

	// Add nodes - one with model but unhealthy, one without model but healthy
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "unhealthy-with-model",
			RAMPercent:  95.0, // Over threshold
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "healthy-no-model",
			RAMPercent:  50.0,
			TempCelsius: 50.0,
			Models:      []string{}, // No models
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}
	addTestNodes(t, client, nodes)

	ctx := context.Background()

	// Select node for gemma3:1b - should NOT get unhealthy-with-model
	nodeID, err := router.SelectNode(ctx, "gemma3:1b")
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if nodeID != "healthy-no-model" {
		t.Errorf("SelectNode(gemma3:1b) = %v, want healthy-no-model (unhealthy filtered)", nodeID)
	}
}

func TestGetModelResidency(t *testing.T) {
	mr, client, router := setupStickyTest(t)
	defer mr.Close()
	defer client.Close()

	nodes := []contracts.NodeHealth{
		{
			NodeID:      "pi-8g",
			RAMPercent:  60.0,
			TempCelsius: 50.0,
			Models:      []string{"gemma3:1b", "llama3.2:3b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-16g",
			RAMPercent:  40.0,
			TempCelsius: 45.0,
			Models:      []string{"llama3.2:3b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-4g",
			RAMPercent:  30.0,
			TempCelsius: 48.0,
			Models:      []string{}, // No models
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}
	addTestNodes(t, client, nodes)

	ctx := context.Background()

	// Get nodes with gemma3:1b
	residency, err := router.GetModelResidency(ctx, "gemma3:1b")
	if err != nil {
		t.Fatalf("GetModelResidency() error = %v", err)
	}

	if len(residency) != 1 {
		t.Errorf("GetModelResidency(gemma3:1b) returned %d nodes, want 1", len(residency))
	}
	if len(residency) > 0 && residency[0] != "pi-8g" {
		t.Errorf("GetModelResidency(gemma3:1b) = %v, want [pi-8g]", residency)
	}

	// Get nodes with llama3.2:3b
	residency, err = router.GetModelResidency(ctx, "llama3.2:3b")
	if err != nil {
		t.Fatalf("GetModelResidency() error = %v", err)
	}

	if len(residency) != 2 {
		t.Errorf("GetModelResidency(llama3.2:3b) returned %d nodes, want 2", len(residency))
	}
}

func TestGetModelResidencyFromNodes(t *testing.T) {
	nodes := []contracts.NodeHealth{
		{NodeID: "node1", Models: []string{"gemma3:1b", "llama3.2:3b"}},
		{NodeID: "node2", Models: []string{"llama3.2:3b"}},
		{NodeID: "node3", Models: []string{}},
	}

	// Test gemma3:1b
	residency := GetModelResidencyFromNodes(nodes, "gemma3:1b")
	if len(residency) != 1 || residency[0] != "node1" {
		t.Errorf("GetModelResidencyFromNodes(gemma3:1b) = %v, want [node1]", residency)
	}

	// Test llama3.2:3b
	residency = GetModelResidencyFromNodes(nodes, "llama3.2:3b")
	if len(residency) != 2 {
		t.Errorf("GetModelResidencyFromNodes(llama3.2:3b) returned %d, want 2", len(residency))
	}

	// Test non-existent model
	residency = GetModelResidencyFromNodes(nodes, "nonexistent")
	if len(residency) != 0 {
		t.Errorf("GetModelResidencyFromNodes(nonexistent) = %v, want []", residency)
	}
}
