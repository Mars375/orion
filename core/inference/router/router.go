package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/contracts"
)

const (
	// DefaultRequestStream is the main stream for incoming inference requests.
	DefaultRequestStream = "orion:inference:requests"

	// DefaultConsumerGroup is the consumer group name for the router.
	DefaultConsumerGroup = "orion-router"

	// DefaultMaxStreamLen is the maximum length for worker request streams.
	DefaultMaxStreamLen = 1000

	// DefaultBlockTimeout is the timeout for blocking reads from streams.
	DefaultBlockTimeout = 5 * time.Second
)

// RoutingStats tracks routing decisions for observability.
type RoutingStats struct {
	TotalRouted   int64            // Total requests routed
	StickyHits    int64            // Requests routed to node with model resident
	Fallbacks     int64            // Requests routed to least-loaded node
	Errors        int64            // Routing errors
	NodeRouteCounts map[string]int64 // Requests per node
}

// InferenceRouter routes inference requests to worker nodes via Redis Streams.
// It uses sticky routing to prefer nodes with model already loaded.
type InferenceRouter struct {
	redis        *redis.Client
	sticky       *StickyRouter
	streamPrefix string
	consumerName string

	// Stats (accessed atomically)
	stats RoutingStats
}

// NewInferenceRouter creates a new inference router.
// streamPrefix is used to construct stream names (e.g., "orion:inference").
func NewInferenceRouter(redisClient *redis.Client, sticky *StickyRouter, streamPrefix string) *InferenceRouter {
	if streamPrefix == "" {
		streamPrefix = "orion:inference"
	}
	return &InferenceRouter{
		redis:        redisClient,
		sticky:       sticky,
		streamPrefix: streamPrefix,
		consumerName: fmt.Sprintf("router-%d", time.Now().UnixNano()),
		stats: RoutingStats{
			NodeRouteCounts: make(map[string]int64),
		},
	}
}

// RouteRequest routes an inference request to the optimal worker node.
// It selects the node via sticky routing and dispatches to the node's request stream.
func (r *InferenceRouter) RouteRequest(ctx context.Context, req contracts.InferenceRequest) error {
	// Select optimal node
	nodeID, err := r.sticky.SelectNode(ctx, req.Model)
	if err != nil {
		atomic.AddInt64(&r.stats.Errors, 1)
		return fmt.Errorf("failed to select node for model %s: %w", req.Model, err)
	}

	// Check if this was a sticky hit (model was resident)
	residency, _ := r.sticky.GetModelResidency(ctx, req.Model)
	isStickyHit := contains(residency, nodeID)

	// Dispatch to worker's stream
	workerStream := fmt.Sprintf("%s:requests:%s", r.streamPrefix, nodeID)

	data, err := json.Marshal(req)
	if err != nil {
		atomic.AddInt64(&r.stats.Errors, 1)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = r.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: workerStream,
		MaxLen: DefaultMaxStreamLen,
		Approx: true,
		Values: map[string]interface{}{
			"data":       string(data),
			"request_id": req.RequestID,
			"model":      req.Model,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		},
	}).Result()

	if err != nil {
		atomic.AddInt64(&r.stats.Errors, 1)
		return fmt.Errorf("failed to dispatch request to %s: %w", workerStream, err)
	}

	// Update stats
	atomic.AddInt64(&r.stats.TotalRouted, 1)
	if isStickyHit {
		atomic.AddInt64(&r.stats.StickyHits, 1)
	} else {
		atomic.AddInt64(&r.stats.Fallbacks, 1)
	}
	// Note: NodeRouteCounts not thread-safe, but acceptable for observability

	log.Printf("INFO: Routing request %s for model %s to node %s (sticky=%v)",
		req.RequestID, req.Model, nodeID, isStickyHit)

	return nil
}

// ConsumeRequests starts consuming from the main request stream and routing requests.
// Blocks until context is cancelled.
func (r *InferenceRouter) ConsumeRequests(ctx context.Context) error {
	streamName := fmt.Sprintf("%s:requests", r.streamPrefix)

	// Create consumer group if it doesn't exist
	err := r.redis.XGroupCreateMkStream(ctx, streamName, DefaultConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("WARN: Could not create consumer group: %v", err)
	}

	log.Printf("INFO: Starting request consumer on stream %s", streamName)

	for {
		select {
		case <-ctx.Done():
			log.Printf("INFO: Stopping request consumer")
			return ctx.Err()
		default:
		}

		// Read from stream with blocking
		streams, err := r.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    DefaultConsumerGroup,
			Consumer: r.consumerName,
			Streams:  []string{streamName, ">"},
			Count:    10,
			Block:    DefaultBlockTimeout,
		}).Result()

		if err == redis.Nil {
			continue // No messages, continue polling
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("WARN: Error reading from stream: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Process messages
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				r.processMessage(ctx, streamName, msg)
			}
		}
	}
}

// processMessage handles a single message from the request stream.
func (r *InferenceRouter) processMessage(ctx context.Context, streamName string, msg redis.XMessage) {
	// Extract request data
	data, ok := msg.Values["data"].(string)
	if !ok {
		log.Printf("ERROR: Message %s missing data field", msg.ID)
		r.ackMessage(ctx, streamName, msg.ID)
		return
	}

	var req contracts.InferenceRequest
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		log.Printf("ERROR: Failed to unmarshal request from message %s: %v", msg.ID, err)
		r.ackMessage(ctx, streamName, msg.ID)
		return
	}

	// Route the request
	if err := r.RouteRequest(ctx, req); err != nil {
		log.Printf("ERROR: Failed to route request %s: %v", req.RequestID, err)
		// Don't ACK - leave for retry or dead-letter processing
		return
	}

	r.ackMessage(ctx, streamName, msg.ID)
}

// ackMessage acknowledges a message in the consumer group.
func (r *InferenceRouter) ackMessage(ctx context.Context, streamName, msgID string) {
	if err := r.redis.XAck(ctx, streamName, DefaultConsumerGroup, msgID).Err(); err != nil {
		log.Printf("WARN: Failed to ACK message %s: %v", msgID, err)
	}
}

// GetRoutingStats returns current routing statistics.
func (r *InferenceRouter) GetRoutingStats() map[string]interface{} {
	return map[string]interface{}{
		"total_routed": atomic.LoadInt64(&r.stats.TotalRouted),
		"sticky_hits":  atomic.LoadInt64(&r.stats.StickyHits),
		"fallbacks":    atomic.LoadInt64(&r.stats.Fallbacks),
		"errors":       atomic.LoadInt64(&r.stats.Errors),
	}
}

// GetWorkerStreamName returns the stream name for a worker node.
func (r *InferenceRouter) GetWorkerStreamName(nodeID string) string {
	return fmt.Sprintf("%s:requests:%s", r.streamPrefix, nodeID)
}

// GetRequestStreamName returns the main request stream name.
func (r *InferenceRouter) GetRequestStreamName() string {
	return fmt.Sprintf("%s:requests", r.streamPrefix)
}

// contains checks if a string slice contains a value.
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
