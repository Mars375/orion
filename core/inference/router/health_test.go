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

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
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

func TestNewHealthReader(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	// Test with default key prefix
	reader := NewHealthReader(client, "")
	if reader.keyPrefix != DefaultHealthHashKey {
		t.Errorf("keyPrefix = %v, want %v", reader.keyPrefix, DefaultHealthHashKey)
	}

	// Test with custom key prefix
	reader = NewHealthReader(client, "custom:health")
	if reader.keyPrefix != "custom:health" {
		t.Errorf("keyPrefix = %v, want %v", reader.keyPrefix, "custom:health")
	}
}

func TestGetHealthyNodes(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	reader := NewHealthReader(client, "")

	// Add test nodes
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "pi-8g",
			RAMPercent:  70.0,
			TempCelsius: 55.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "pi-16g",
			RAMPercent:  40.0,
			TempCelsius: 50.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Get healthy nodes
	result, err := reader.GetHealthyNodes(ctx)
	if err != nil {
		t.Fatalf("GetHealthyNodes() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("GetHealthyNodes() returned %d nodes, want 2", len(result))
	}

	// Verify sorted by RAM percent ascending
	if result[0].NodeID != "pi-16g" {
		t.Errorf("First node should be pi-16g (40%% RAM), got %s (%.1f%%)", result[0].NodeID, result[0].RAMPercent)
	}
	if result[1].NodeID != "pi-8g" {
		t.Errorf("Second node should be pi-8g (70%% RAM), got %s (%.1f%%)", result[1].NodeID, result[1].RAMPercent)
	}
}

func TestGetHealthyNodes_FiltersStaleNodes(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	reader := NewHealthReader(client, "")

	// Add fresh and stale nodes
	freshNode := contracts.NodeHealth{
		NodeID:      "fresh",
		RAMPercent:  50.0,
		TempCelsius: 50.0,
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}
	staleNode := contracts.NodeHealth{
		NodeID:      "stale",
		RAMPercent:  50.0,
		TempCelsius: 50.0,
		Available:   true,
		LastSeen:    time.Now().Add(-30 * time.Second).UTC(), // Stale
	}

	freshData, _ := json.Marshal(freshNode)
	staleData, _ := json.Marshal(staleNode)
	client.HSet(ctx, DefaultHealthHashKey, "fresh", string(freshData))
	client.HSet(ctx, DefaultHealthHashKey, "stale", string(staleData))

	// Get healthy nodes - should only return fresh
	result, err := reader.GetHealthyNodes(ctx)
	if err != nil {
		t.Fatalf("GetHealthyNodes() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("GetHealthyNodes() returned %d nodes, want 1 (stale filtered)", len(result))
	}

	if result[0].NodeID != "fresh" {
		t.Errorf("Expected fresh node, got %s", result[0].NodeID)
	}
}

func TestGetHealthyNodes_FiltersUnavailableNodes(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	reader := NewHealthReader(client, "")

	// Add available and unavailable nodes
	availableNode := contracts.NodeHealth{
		NodeID:      "available",
		RAMPercent:  50.0,
		TempCelsius: 50.0,
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}
	unavailableNode := contracts.NodeHealth{
		NodeID:      "unavailable",
		RAMPercent:  50.0,
		TempCelsius: 50.0,
		Available:   false, // Unavailable
		LastSeen:    time.Now().UTC(),
	}

	availableData, _ := json.Marshal(availableNode)
	unavailableData, _ := json.Marshal(unavailableNode)
	client.HSet(ctx, DefaultHealthHashKey, "available", string(availableData))
	client.HSet(ctx, DefaultHealthHashKey, "unavailable", string(unavailableData))

	// Get healthy nodes - should only return available
	result, err := reader.GetHealthyNodes(ctx)
	if err != nil {
		t.Fatalf("GetHealthyNodes() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("GetHealthyNodes() returned %d nodes, want 1 (unavailable filtered)", len(result))
	}

	if result[0].NodeID != "available" {
		t.Errorf("Expected available node, got %s", result[0].NodeID)
	}
}

func TestGetHealthyNodes_FiltersUnhealthyNodes(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	reader := NewHealthReader(client, "")

	// Add nodes with different health statuses
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "healthy",
			RAMPercent:  50.0,
			TempCelsius: 50.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "hot",
			RAMPercent:  50.0,
			TempCelsius: 80.0, // Over temperature threshold
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "overloaded",
			RAMPercent:  95.0, // Over RAM threshold
			TempCelsius: 50.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Get healthy nodes - should only return the healthy one
	result, err := reader.GetHealthyNodes(ctx)
	if err != nil {
		t.Fatalf("GetHealthyNodes() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("GetHealthyNodes() returned %d nodes, want 1 (unhealthy filtered)", len(result))
	}

	if result[0].NodeID != "healthy" {
		t.Errorf("Expected healthy node, got %s", result[0].NodeID)
	}
}

func TestGetNodeHealth(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	reader := NewHealthReader(client, "")

	// Add a node
	health := contracts.NodeHealth{
		NodeID:      "pi-8g",
		RAMPercent:  60.0,
		TempCelsius: 55.0,
		Available:   true,
		LastSeen:    time.Now().UTC(),
	}
	data, _ := json.Marshal(health)
	client.HSet(ctx, DefaultHealthHashKey, "pi-8g", string(data))

	// Get specific node
	result, err := reader.GetNodeHealth(ctx, "pi-8g")
	if err != nil {
		t.Fatalf("GetNodeHealth() error = %v", err)
	}

	if result == nil {
		t.Fatal("GetNodeHealth() returned nil")
	}

	if result.NodeID != "pi-8g" {
		t.Errorf("NodeID = %v, want %v", result.NodeID, "pi-8g")
	}

	// Get non-existent node
	result, err = reader.GetNodeHealth(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetNodeHealth() error = %v", err)
	}
	if result != nil {
		t.Error("GetNodeHealth() should return nil for non-existent node")
	}
}

func TestGetNodeHealth_ReturnsNilForStale(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	reader := NewHealthReader(client, "")

	// Add a stale node
	health := contracts.NodeHealth{
		NodeID:      "stale-node",
		RAMPercent:  60.0,
		TempCelsius: 55.0,
		Available:   true,
		LastSeen:    time.Now().Add(-30 * time.Second).UTC(), // Stale
	}
	data, _ := json.Marshal(health)
	client.HSet(ctx, DefaultHealthHashKey, "stale-node", string(data))

	// Get stale node - should return nil
	result, err := reader.GetNodeHealth(ctx, "stale-node")
	if err != nil {
		t.Fatalf("GetNodeHealth() error = %v", err)
	}

	if result != nil {
		t.Error("GetNodeHealth() should return nil for stale node")
	}
}

func TestIsNodeHealthy(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	reader := NewHealthReader(client, "")

	tests := []struct {
		name     string
		health   contracts.NodeHealth
		expected bool
	}{
		{
			name: "healthy node",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  60.0,
				Available:   true,
				LastSeen:    time.Now().UTC(),
			},
			expected: true,
		},
		{
			name: "high temperature",
			health: contracts.NodeHealth{
				TempCelsius: 80.0,
				RAMPercent:  60.0,
				Available:   true,
				LastSeen:    time.Now().UTC(),
			},
			expected: false,
		},
		{
			name: "high RAM",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  95.0,
				Available:   true,
				LastSeen:    time.Now().UTC(),
			},
			expected: false,
		},
		{
			name: "unavailable",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  60.0,
				Available:   false,
				LastSeen:    time.Now().UTC(),
			},
			expected: false,
		},
		{
			name: "stale",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  60.0,
				Available:   true,
				LastSeen:    time.Now().Add(-30 * time.Second).UTC(),
			},
			expected: false,
		},
		{
			name: "exactly at temp threshold",
			health: contracts.NodeHealth{
				TempCelsius: 75.0,
				RAMPercent:  60.0,
				Available:   true,
				LastSeen:    time.Now().UTC(),
			},
			expected: true, // 75 is not > 75
		},
		{
			name: "exactly at RAM threshold",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  90.0,
				Available:   true,
				LastSeen:    time.Now().UTC(),
			},
			expected: true, // 90 is not > 90
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reader.IsNodeHealthy(tt.health); got != tt.expected {
				t.Errorf("IsNodeHealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetHealthyNodes_SortedByRAM(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	reader := NewHealthReader(client, "")

	// Add nodes with varying RAM usage
	nodes := []contracts.NodeHealth{
		{
			NodeID:      "high-ram",
			RAMPercent:  80.0,
			TempCelsius: 50.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "low-ram",
			RAMPercent:  30.0,
			TempCelsius: 50.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
		{
			NodeID:      "medium-ram",
			RAMPercent:  55.0,
			TempCelsius: 50.0,
			Available:   true,
			LastSeen:    time.Now().UTC(),
		},
	}

	for _, health := range nodes {
		data, _ := json.Marshal(health)
		client.HSet(ctx, DefaultHealthHashKey, health.NodeID, string(data))
	}

	// Get healthy nodes - should be sorted by RAM ascending
	result, err := reader.GetHealthyNodes(ctx)
	if err != nil {
		t.Fatalf("GetHealthyNodes() error = %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("GetHealthyNodes() returned %d nodes, want 3", len(result))
	}

	expectedOrder := []string{"low-ram", "medium-ram", "high-ram"}
	for i, expected := range expectedOrder {
		if result[i].NodeID != expected {
			t.Errorf("Node at position %d: got %s, want %s", i, result[i].NodeID, expected)
		}
	}
}
