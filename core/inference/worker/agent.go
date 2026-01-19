package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/contracts"
)

const (
	// DefaultHealthInterval is the interval for publishing health updates.
	DefaultHealthInterval = 5 * time.Second

	// DefaultConsumerGroup is the consumer group for worker request streams.
	DefaultConsumerGroup = "orion-worker"

	// DefaultBlockTimeout is the timeout for blocking reads.
	DefaultBlockTimeout = 1 * time.Second
)

// WorkerAgent processes inference requests using Ollama.
// It publishes health metrics and consumes requests from its dedicated stream.
type WorkerAgent struct {
	nodeID       string
	ollamaHost   string
	streamPrefix string
	consumerName string

	redis        *redis.Client
	ollama       *api.Client
	metrics      *HealthCollector
	registry     *HealthRegistry

	mu      sync.Mutex
	running bool
}

// NewWorkerAgent creates a new worker agent.
func NewWorkerAgent(nodeID, ollamaHost string, redisClient *redis.Client, streamPrefix string) (*WorkerAgent, error) {
	// Parse Ollama host URL
	u, err := url.Parse(ollamaHost)
	if err != nil {
		return nil, fmt.Errorf("invalid ollama host: %w", err)
	}

	// Create Ollama client
	ollamaClient := api.NewClient(u, http.DefaultClient)

	// Create health collector
	metrics, err := NewHealthCollector(nodeID, ollamaHost)
	if err != nil {
		return nil, fmt.Errorf("failed to create health collector: %w", err)
	}

	// Create health registry
	registry := NewHealthRegistry(redisClient, nodeID, "")

	if streamPrefix == "" {
		streamPrefix = "orion:inference"
	}

	return &WorkerAgent{
		nodeID:       nodeID,
		ollamaHost:   ollamaHost,
		streamPrefix: streamPrefix,
		consumerName: fmt.Sprintf("worker-%s-%d", nodeID, time.Now().UnixNano()),
		redis:        redisClient,
		ollama:       ollamaClient,
		metrics:      metrics,
		registry:     registry,
	}, nil
}

// Start begins the worker agent's operation.
// It starts health publishing and request consumption goroutines.
// Blocks until context is cancelled.
func (w *WorkerAgent) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("worker agent already running")
	}
	w.running = true
	w.mu.Unlock()

	log.Printf("INFO: Worker agent %s starting", w.nodeID)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Start health publisher
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.publishHealthLoop(ctx)
	}()

	// Start request consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.consumeRequests(ctx); err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		log.Printf("INFO: Worker agent %s stopping", w.nodeID)
	case err := <-errCh:
		log.Printf("ERROR: Worker agent %s error: %v", w.nodeID, err)
	}

	// Wait for goroutines to finish
	wg.Wait()

	w.mu.Lock()
	w.running = false
	w.mu.Unlock()

	return nil
}

// Stop gracefully stops the worker agent and removes it from the health registry.
func (w *WorkerAgent) Stop(ctx context.Context) error {
	log.Printf("INFO: Removing worker %s from health registry", w.nodeID)
	return w.registry.RemoveNode(ctx)
}

// publishHealthLoop periodically publishes health metrics to Redis.
func (w *WorkerAgent) publishHealthLoop(ctx context.Context) {
	ticker := time.NewTicker(DefaultHealthInterval)
	defer ticker.Stop()

	// Publish immediately on start
	w.publishHealth(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Printf("INFO: Health publisher stopped for %s", w.nodeID)
			return
		case <-ticker.C:
			w.publishHealth(ctx)
		}
	}
}

// publishHealth collects and publishes health metrics.
func (w *WorkerAgent) publishHealth(ctx context.Context) {
	health, err := w.metrics.CollectHealth(ctx)
	if err != nil {
		log.Printf("WARN: Failed to collect health for %s: %v", w.nodeID, err)
		return
	}

	if err := w.registry.PublishHealth(ctx, health); err != nil {
		log.Printf("WARN: Failed to publish health for %s: %v", w.nodeID, err)
	}
}

// consumeRequests reads and processes inference requests from the worker's stream.
func (w *WorkerAgent) consumeRequests(ctx context.Context) error {
	streamName := fmt.Sprintf("%s:requests:%s", w.streamPrefix, w.nodeID)

	// Create consumer group if it doesn't exist
	err := w.redis.XGroupCreateMkStream(ctx, streamName, DefaultConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Printf("WARN: Could not create consumer group: %v", err)
	}

	log.Printf("INFO: Worker %s consuming from stream %s", w.nodeID, streamName)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read from stream with blocking
		streams, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    DefaultConsumerGroup,
			Consumer: w.consumerName,
			Streams:  []string{streamName, ">"},
			Count:    1,
			Block:    DefaultBlockTimeout,
		}).Result()

		if err == redis.Nil {
			continue // No messages
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
				w.processMessage(ctx, streamName, msg)
			}
		}
	}
}

// processMessage handles a single inference request message.
func (w *WorkerAgent) processMessage(ctx context.Context, streamName string, msg redis.XMessage) {
	data, ok := msg.Values["data"].(string)
	if !ok {
		log.Printf("ERROR: Message %s missing data field", msg.ID)
		w.ackMessage(ctx, streamName, msg.ID)
		return
	}

	var req contracts.InferenceRequest
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		log.Printf("ERROR: Failed to unmarshal request from message %s: %v", msg.ID, err)
		w.ackMessage(ctx, streamName, msg.ID)
		return
	}

	log.Printf("INFO: Processing inference request %s for model %s", req.RequestID, req.Model)

	// Process the inference request
	resp := w.processInference(ctx, req)

	// Publish response to callback stream if specified
	if req.Callback != "" {
		if err := w.publishResponse(ctx, req.Callback, resp); err != nil {
			log.Printf("ERROR: Failed to publish response to %s: %v", req.Callback, err)
		}
	}

	w.ackMessage(ctx, streamName, msg.ID)
}

// processInference executes inference using Ollama and returns the response.
func (w *WorkerAgent) processInference(ctx context.Context, req contracts.InferenceRequest) contracts.InferenceResponse {
	startTime := time.Now()

	resp := contracts.InferenceResponse{
		Version:   "1.0",
		RequestID: req.RequestID,
		Model:     req.Model,
		Timestamp: time.Now().UTC(),
		Source:    fmt.Sprintf("orion-inference-worker-%s", w.nodeID),
	}

	// Convert messages to Ollama format
	messages := make([]api.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = api.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	// Build chat request
	keepAlive := req.KeepAliveDuration()
	chatReq := &api.ChatRequest{
		Model:     req.Model,
		Messages:  messages,
		KeepAlive: &api.Duration{Duration: keepAlive},
		Stream:    boolPtr(false), // Non-streaming for simplicity
	}

	// Execute inference
	var fullResponse strings.Builder
	var loadDuration int64
	var promptTokens, completionTokens int

	err := w.ollama.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
		fullResponse.WriteString(resp.Message.Content)
		loadDuration = int64(resp.LoadDuration / time.Millisecond)
		promptTokens = resp.PromptEvalCount
		completionTokens = resp.EvalCount
		return nil
	})

	if err != nil {
		log.Printf("ERROR: Ollama inference failed for request %s: %v", req.RequestID, err)
		resp.Error = fmt.Sprintf("inference failed: %v", err)
		return resp
	}

	resp.Response = fullResponse.String()
	resp.LoadDurationMs = loadDuration
	resp.PromptTokens = promptTokens
	resp.CompletionTokens = completionTokens
	resp.TotalDurationMs = time.Since(startTime).Milliseconds()

	log.Printf("INFO: Completed inference request %s in %dms (load: %dms, tokens: %d/%d)",
		req.RequestID, resp.TotalDurationMs, resp.LoadDurationMs, resp.PromptTokens, resp.CompletionTokens)

	return resp
}

// publishResponse publishes an inference response to the callback stream.
func (w *WorkerAgent) publishResponse(ctx context.Context, callback string, resp contracts.InferenceResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	_, err = w.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: callback,
		MaxLen: 1000,
		Approx: true,
		Values: map[string]interface{}{
			"data":       string(data),
			"request_id": resp.RequestID,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		},
	}).Result()

	return err
}

// ackMessage acknowledges a message in the consumer group.
func (w *WorkerAgent) ackMessage(ctx context.Context, streamName, msgID string) {
	if err := w.redis.XAck(ctx, streamName, DefaultConsumerGroup, msgID).Err(); err != nil {
		log.Printf("WARN: Failed to ACK message %s: %v", msgID, err)
	}
}

// GetNodeID returns the worker's node ID.
func (w *WorkerAgent) GetNodeID() string {
	return w.nodeID
}

// GetStreamName returns the worker's request stream name.
func (w *WorkerAgent) GetStreamName() string {
	return fmt.Sprintf("%s:requests:%s", w.streamPrefix, w.nodeID)
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}
