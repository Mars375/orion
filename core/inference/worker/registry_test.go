package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/contracts"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
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

func TestNewHealthRegistry(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	// Test with default hash key
	registry := NewHealthRegistry(client, "test-node", "")
	if registry.hashKey != DefaultHealthHashKey {
		t.Errorf("hashKey = %v, want %v", registry.hashKey, DefaultHealthHashKey)
	}

	// Test with custom hash key
	registry = NewHealthRegistry(client, "test-node", "custom:health")
	if registry.hashKey != "custom:health" {
		t.Errorf("hashKey = %v, want %v", registry.hashKey, "custom:health")
	}
}

func TestPublishHealth(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	registry := NewHealthRegistry(client, "pi-8g", "")

	health := contracts.NodeHealth{
		NodeID:      "pi-8g",
		CPUPercent:  45.5,
		RAMPercent:  62.3,
		RAMUsedMB:   5000,
		RAMTotalMB:  8192,
		TempCelsius: 55.0,
		Models:      []string{"gemma3:1b"},
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}

	err := registry.PublishHealth(ctx, health)
	if err != nil {
		t.Fatalf("PublishHealth() error = %v", err)
	}

	// Verify hash entry
	data, err := client.HGet(ctx, DefaultHealthHashKey, "pi-8g").Result()
	if err != nil {
		t.Fatalf("Failed to get hash entry: %v", err)
	}

	var stored contracts.NodeHealth
	if err := json.Unmarshal([]byte(data), &stored); err != nil {
		t.Fatalf("Failed to unmarshal stored health: %v", err)
	}

	if stored.NodeID != "pi-8g" {
		t.Errorf("stored.NodeID = %v, want %v", stored.NodeID, "pi-8g")
	}
	if stored.CPUPercent != 45.5 {
		t.Errorf("stored.CPUPercent = %v, want %v", stored.CPUPercent, 45.5)
	}
	if len(stored.Models) != 1 || stored.Models[0] != "gemma3:1b" {
		t.Errorf("stored.Models = %v, want [gemma3:1b]", stored.Models)
	}

	// Verify individual key with TTL exists
	individualKey := DefaultHealthHashKey + ":pi-8g"
	exists, err := client.Exists(ctx, individualKey).Result()
	if err != nil {
		t.Fatalf("Failed to check individual key: %v", err)
	}
	if exists != 1 {
		t.Error("Individual key should exist")
	}
}

func TestGetAllHealth(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	registry := NewHealthRegistry(client, "pi-8g", "")

	// Add multiple nodes
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "pi-8g",
			CPUPercent:  45.5,
			RAMPercent:  62.3,
			Models:      []string{"gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-16g",
			CPUPercent:  30.0,
			RAMPercent:  40.0,
			Models:      []string{"llama3.2:3b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	// Publish health for each node
	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Retrieve all health
	result, err := registry.GetAllHealth(ctx)
	if err != nil {
		t.Fatalf("GetAllHealth() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("GetAllHealth() returned %d nodes, want 2", len(result))
	}
}

func TestGetAllHealth_FilterStale(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	registry := NewHealthRegistry(client, "pi-8g", "")

	// Add fresh node
	freshHealth := contracts.NodeHealth{
		NodeID:   "fresh-node",
		LastSeen: time.Now().UTC(),
	}
	freshData, _ := json.Marshal(freshHealth)
	client.HSet(ctx, DefaultHealthHashKey, "fresh-node", string(freshData))

	// Add stale node (last seen 30 seconds ago)
	staleHealth := contracts.NodeHealth{
		NodeID:   "stale-node",
		LastSeen: time.Now().Add(-30 * time.Second).UTC(),
	}
	staleData, _ := json.Marshal(staleHealth)
	client.HSet(ctx, DefaultHealthHashKey, "stale-node", string(staleData))

	// Retrieve all health - should only return fresh node
	result, err := registry.GetAllHealth(ctx)
	if err != nil {
		t.Fatalf("GetAllHealth() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("GetAllHealth() returned %d nodes, want 1 (stale filtered)", len(result))
	}

	if len(result) > 0 && result[0].NodeID != "fresh-node" {
		t.Errorf("GetAllHealth() returned %v, want fresh-node", result[0].NodeID)
	}
}

func TestGetNodeHealth(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	registry := NewHealthRegistry(client, "pi-8g", "")

	// Add a node
	health := contracts.NodeHealth{
		NodeID:      "pi-8g",
		CPUPercent:  45.5,
		RAMPercent:  62.3,
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}
	data, _ := json.Marshal(health)
	client.HSet(ctx, DefaultHealthHashKey, "pi-8g", string(data))

	// Retrieve specific node
	result, err := registry.GetNodeHealth(ctx, "pi-8g")
	if err != nil {
		t.Fatalf("GetNodeHealth() error = %v", err)
	}

	if result == nil {
		t.Fatal("GetNodeHealth() returned nil")
	}

	if result.NodeID != "pi-8g" {
		t.Errorf("NodeID = %v, want %v", result.NodeID, "pi-8g")
	}

	// Retrieve non-existent node
	result, err = registry.GetNodeHealth(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetNodeHealth() error = %v", err)
	}
	if result != nil {
		t.Error("GetNodeHealth() should return nil for non-existent node")
	}
}

func TestRemoveNode(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	registry := NewHealthRegistry(client, "pi-8g", "")

	// Add a node
	health := contracts.NodeHealth{
		NodeID:   "pi-8g",
		LastSeen: time.Now().UTC(),
	}
	data, _ := json.Marshal(health)
	client.HSet(ctx, DefaultHealthHashKey, "pi-8g", string(data))
	client.Set(ctx, DefaultHealthHashKey+":pi-8g", string(data), 0)

	// Remove the node
	err := registry.RemoveNode(ctx)
	if err != nil {
		t.Fatalf("RemoveNode() error = %v", err)
	}

	// Verify hash entry is removed
	exists, err := client.HExists(ctx, DefaultHealthHashKey, "pi-8g").Result()
	if err != nil {
		t.Fatalf("Failed to check hash entry: %v", err)
	}
	if exists {
		t.Error("Hash entry should be removed")
	}

	// Verify individual key is removed
	keyExists, err := client.Exists(ctx, DefaultHealthHashKey+":pi-8g").Result()
	if err != nil {
		t.Fatalf("Failed to check individual key: %v", err)
	}
	if keyExists != 0 {
		t.Error("Individual key should be removed")
	}
}

func TestGetAvailableNodes(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	registry := NewHealthRegistry(client, "pi-8g", "")

	// Add nodes with different availability
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "available-healthy",
			RAMPercent:  60.0,
			TempCelsius: 50.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "unavailable",
			RAMPercent:  60.0,
			TempCelsius: 50.0,
			Available:   false, // Marked unavailable
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "unhealthy-temp",
			RAMPercent:  60.0,
			TempCelsius: 80.0, // Over threshold
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Get available nodes
	result, err := registry.GetAvailableNodes(ctx)
	if err != nil {
		t.Fatalf("GetAvailableNodes() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("GetAvailableNodes() returned %d nodes, want 1", len(result))
	}

	if len(result) > 0 && result[0].NodeID != "available-healthy" {
		t.Errorf("GetAvailableNodes() returned %v, want available-healthy", result[0].NodeID)
	}
}

func TestGetNodesWithModel(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	registry := NewHealthRegistry(client, "pi-8g", "")

	// Add nodes with different models
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
			RAMPercent:  40.0,
			TempCelsius: 45.0,
			Models:      []string{"llama3.2:3b", "gemma3:1b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-other",
			RAMPercent:  50.0,
			TempCelsius: 48.0,
			Models:      []string{"mistral:7b"},
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Get nodes with gemma3:1b
	result, err := registry.GetNodesWithModel(ctx, "gemma3:1b")
	if err != nil {
		t.Fatalf("GetNodesWithModel() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("GetNodesWithModel(gemma3:1b) returned %d nodes, want 2", len(result))
	}

	// Get nodes with mistral:7b
	result, err = registry.GetNodesWithModel(ctx, "mistral:7b")
	if err != nil {
		t.Fatalf("GetNodesWithModel() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("GetNodesWithModel(mistral:7b) returned %d nodes, want 1", len(result))
	}

	// Get nodes with non-existent model
	result, err = registry.GetNodesWithModel(ctx, "nonexistent:0b")
	if err != nil {
		t.Fatalf("GetNodesWithModel() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("GetNodesWithModel(nonexistent) returned %d nodes, want 0", len(result))
	}
}
