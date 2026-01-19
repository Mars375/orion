// Package router implements inference request routing with sticky model placement.
package router

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/contracts"
)

const (
	// DefaultHealthHashKey is the Redis hash key for node health data.
	DefaultHealthHashKey = "orion:inference:health"

	// DefaultStaleDuration is the duration after which health data is considered stale.
	DefaultStaleDuration = 15 * time.Second
)

// HealthReader reads node health information from Redis for routing decisions.
// Health data is always read fresh (no caching) to ensure routing accuracy.
type HealthReader struct {
	redis         *redis.Client
	keyPrefix     string
	staleDuration time.Duration
}

// NewHealthReader creates a new HealthReader.
// keyPrefix can be empty to use the default "orion:inference:health" hash key.
func NewHealthReader(redis *redis.Client, keyPrefix string) *HealthReader {
	if keyPrefix == "" {
		keyPrefix = DefaultHealthHashKey
	}
	return &HealthReader{
		redis:         redis,
		keyPrefix:     keyPrefix,
		staleDuration: DefaultStaleDuration,
	}
}

// GetHealthyNodes returns all healthy and available nodes sorted by RAM usage (ascending).
// Filters out stale nodes (LastSeen > staleDuration) and unavailable nodes (Available == false).
// Also filters nodes that don't pass health thresholds (temp > 75°C or RAM > 90%).
func (h *HealthReader) GetHealthyNodes(ctx context.Context) ([]contracts.NodeHealth, error) {
	// Read all entries from the health hash
	result, err := h.redis.HGetAll(ctx, h.keyPrefix).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read health data: %w", err)
	}

	nodes := make([]contracts.NodeHealth, 0, len(result))

	for _, data := range result {
		var health contracts.NodeHealth
		if err := json.Unmarshal([]byte(data), &health); err != nil {
			continue // Skip malformed entries
		}

		// Filter out nodes that don't meet criteria
		if !h.IsNodeHealthy(health) {
			continue
		}

		nodes = append(nodes, health)
	}

	// Sort by RAM percent ascending (least loaded first)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].RAMPercent < nodes[j].RAMPercent
	})

	return nodes, nil
}

// GetNodeHealth returns health data for a specific node.
// Returns nil if the node is not found or if its data is stale.
func (h *HealthReader) GetNodeHealth(ctx context.Context, nodeID string) (*contracts.NodeHealth, error) {
	data, err := h.redis.HGet(ctx, h.keyPrefix, nodeID).Result()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get health for node %s: %w", nodeID, err)
	}

	var health contracts.NodeHealth
	if err := json.Unmarshal([]byte(data), &health); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health for node %s: %w", nodeID, err)
	}

	// Check staleness
	if health.IsStale(h.staleDuration) {
		return nil, nil // Stale data treated as missing
	}

	return &health, nil
}

// IsNodeHealthy checks if a node meets all health criteria for routing:
// - TempCelsius <= 75°C
// - RAMPercent <= 90%
// - LastSeen within staleDuration
// - Available == true
func (h *HealthReader) IsNodeHealthy(health contracts.NodeHealth) bool {
	// Check staleness first
	if health.IsStale(h.staleDuration) {
		return false
	}

	// Check availability flag
	if !health.Available {
		return false
	}

	// Check health thresholds (delegated to NodeHealth.IsHealthy)
	if !health.IsHealthy() {
		return false
	}

	return true
}

// SetStaleDuration allows customizing the stale threshold for testing.
func (h *HealthReader) SetStaleDuration(d time.Duration) {
	h.staleDuration = d
}
