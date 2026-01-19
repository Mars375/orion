// Package worker implements the ORION inference worker agent.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/contracts"
)

const (
	// DefaultHealthHashKey is the Redis hash key storing all worker health data.
	DefaultHealthHashKey = "orion:inference:health"

	// DefaultHealthTTL is the TTL for individual health entries.
	DefaultHealthTTL = 30 * time.Second

	// DefaultStaleThreshold is the duration after which health data is considered stale.
	DefaultStaleThreshold = 15 * time.Second
)

// HealthRegistry manages worker health data in Redis.
// It publishes health updates and retrieves health information for routing decisions.
type HealthRegistry struct {
	redis    *redis.Client
	nodeID   string
	hashKey  string
}

// NewHealthRegistry creates a new HealthRegistry.
// hashKey is the Redis hash key for storing health data (use DefaultHealthHashKey if empty).
func NewHealthRegistry(redisClient *redis.Client, nodeID, hashKey string) *HealthRegistry {
	if hashKey == "" {
		hashKey = DefaultHealthHashKey
	}
	return &HealthRegistry{
		redis:   redisClient,
		nodeID:  nodeID,
		hashKey: hashKey,
	}
}

// PublishHealth publishes the node's health to Redis.
// Health is stored in a hash for efficient multi-node retrieval and
// also in a separate key with TTL as a backup for stale detection.
func (r *HealthRegistry) PublishHealth(ctx context.Context, health contracts.NodeHealth) error {
	// Serialize health to JSON
	data, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("failed to marshal health: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := r.redis.Pipeline()

	// Store in hash for efficient retrieval of all nodes
	pipe.HSet(ctx, r.hashKey, r.nodeID, string(data))

	// Store individual key with TTL as backup for stale detection
	individualKey := fmt.Sprintf("%s:%s", r.hashKey, r.nodeID)
	pipe.SetEx(ctx, individualKey, string(data), DefaultHealthTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish health: %w", err)
	}

	log.Printf("DEBUG: Published health for node %s", r.nodeID)
	return nil
}

// GetAllHealth retrieves health data for all registered nodes.
// Filters out nodes with stale data (last_seen > staleThreshold ago).
func (r *HealthRegistry) GetAllHealth(ctx context.Context) ([]contracts.NodeHealth, error) {
	// Get all entries from the health hash
	result, err := r.redis.HGetAll(ctx, r.hashKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get health data: %w", err)
	}

	nodes := make([]contracts.NodeHealth, 0, len(result))
	staleNodes := []string{}

	for nodeID, data := range result {
		var health contracts.NodeHealth
		if err := json.Unmarshal([]byte(data), &health); err != nil {
			log.Printf("WARN: Failed to unmarshal health for node %s: %v", nodeID, err)
			continue
		}

		// Filter out stale entries
		if health.IsStale(DefaultStaleThreshold) {
			staleNodes = append(staleNodes, nodeID)
			continue
		}

		nodes = append(nodes, health)
	}

	// Clean up stale entries asynchronously
	if len(staleNodes) > 0 {
		go func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for _, nodeID := range staleNodes {
				if err := r.redis.HDel(cleanupCtx, r.hashKey, nodeID).Err(); err != nil {
					log.Printf("WARN: Failed to clean up stale node %s: %v", nodeID, err)
				}
			}
		}()
	}

	return nodes, nil
}

// GetNodeHealth retrieves health data for a specific node.
// Returns nil if the node is not found or its data is stale.
func (r *HealthRegistry) GetNodeHealth(ctx context.Context, nodeID string) (*contracts.NodeHealth, error) {
	data, err := r.redis.HGet(ctx, r.hashKey, nodeID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get health for node %s: %w", nodeID, err)
	}

	var health contracts.NodeHealth
	if err := json.Unmarshal([]byte(data), &health); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health for node %s: %w", nodeID, err)
	}

	// Check if data is stale
	if health.IsStale(DefaultStaleThreshold) {
		return nil, nil
	}

	return &health, nil
}

// RemoveNode removes the node's health entry from Redis.
// Called during graceful shutdown.
func (r *HealthRegistry) RemoveNode(ctx context.Context) error {
	pipe := r.redis.Pipeline()

	// Remove from hash
	pipe.HDel(ctx, r.hashKey, r.nodeID)

	// Remove individual key
	individualKey := fmt.Sprintf("%s:%s", r.hashKey, r.nodeID)
	pipe.Del(ctx, individualKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove node %s: %w", r.nodeID, err)
	}

	log.Printf("INFO: Removed node %s from health registry", r.nodeID)
	return nil
}

// GetAvailableNodes returns all healthy and available nodes.
// Filters by IsHealthy() and Available flag.
func (r *HealthRegistry) GetAvailableNodes(ctx context.Context) ([]contracts.NodeHealth, error) {
	all, err := r.GetAllHealth(ctx)
	if err != nil {
		return nil, err
	}

	available := make([]contracts.NodeHealth, 0, len(all))
	for _, health := range all {
		if health.Available && health.IsHealthy() {
			available = append(available, health)
		}
	}

	return available, nil
}

// GetNodesWithModel returns all available nodes that have the specified model loaded.
// Used for sticky routing to avoid model loading latency.
func (r *HealthRegistry) GetNodesWithModel(ctx context.Context, model string) ([]contracts.NodeHealth, error) {
	available, err := r.GetAvailableNodes(ctx)
	if err != nil {
		return nil, err
	}

	withModel := make([]contracts.NodeHealth, 0)
	for _, health := range available {
		if health.HasModel(model) {
			withModel = append(withModel, health)
		}
	}

	return withModel, nil
}
