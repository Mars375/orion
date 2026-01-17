package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/orion/bus/go/internal/validator"
)

// EventBus provides Redis Streams-based event bus functionality.
//
// Invariants:
//   - All messages MUST validate against JSON Schema contracts before publish
//   - Invalid messages are rejected immediately (fail-fast)
//   - All messages published to Redis Streams
//   - Consumer groups guarantee at-least-once delivery
//   - Bounded memory usage via maxlen trimming
//   - Context cancellation stops all operations cleanly
type EventBus struct {
	client       *redis.Client
	validator    *validator.ContractValidator
	streamPrefix string
	maxLen       int64
	logger       *log.Logger
}

// NewEventBus creates a new EventBus instance with contract validation.
//
// Parameters:
//   - client: Configured Redis client with connection pool
//   - contractsDir: Path to directory containing *.schema.json files
//   - streamPrefix: Prefix for stream names (e.g., "orion" -> "orion:events")
//   - maxLen: Maximum stream length for memory bounding (approximate trimming)
//
// Returns:
//   - Initialized EventBus ready for Publish and Subscribe operations
//   - Error if contract schemas cannot be loaded
func NewEventBus(client *redis.Client, contractsDir string, streamPrefix string, maxLen int64) (*EventBus, error) {
	// Load contract validator
	v, err := validator.NewContractValidator(contractsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract validator: %w", err)
	}

	return &EventBus{
		client:       client,
		validator:    v,
		streamPrefix: streamPrefix,
		maxLen:       maxLen,
		logger:       log.Default(),
	}, nil
}

// getStreamName returns the Redis stream name for a contract type.
//
// Example: contractType "event" -> "orion:events"
func (b *EventBus) getStreamName(contractType string) string {
	return fmt.Sprintf("%s:%ss", b.streamPrefix, contractType)
}

// Publish publishes a message to the event bus after validating against JSON Schema.
//
// The message is validated against its contract schema BEFORE publishing to Redis.
// Invalid messages are rejected immediately (fail-fast). Valid messages are marshaled
// to JSON and published to the Redis stream with approximate maxlen trimming.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - message: Message data as map (must conform to JSON Schema)
//   - contractType: Contract type (e.g., "event", "incident", "decision")
//
// Returns:
//   - Message ID from Redis (e.g., "1234567890123-0")
//   - Error if validation fails, Redis operation fails, or context is cancelled
//
// Contract validation is mandatory - no message reaches Redis without validation.
func (b *EventBus) Publish(ctx context.Context, message map[string]interface{}, contractType string) (string, error) {
	// Validate message against contract schema (fail-fast)
	if err := b.validator.Validate(message, contractType); err != nil {
		b.logger.Printf("ERROR: Contract validation failed for %s: %v", contractType, err)
		return "", fmt.Errorf("contract validation failed: %w", err)
	}

	// Get stream name
	streamName := b.getStreamName(contractType)

	// Marshal message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to Redis Stream with approximate maxlen trimming
	args := &redis.XAddArgs{
		Stream: streamName,
		MaxLen: b.maxLen,
		Approx: true, // Approximate trimming for performance
		Values: map[string]interface{}{
			"data": messageJSON,
		},
	}

	messageID, err := b.client.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("failed to publish to %s: %w", streamName, err)
	}

	b.logger.Printf("DEBUG: Published %s to %s: %s", contractType, streamName, messageID)

	return messageID, nil
}

// Subscribe subscribes to messages from the event bus using consumer groups.
//
// Creates a consumer group if it doesn't exist, then enters a blocking loop
// reading messages with XReadGroup. For each message, the handler is called.
// If the handler succeeds, the message is acknowledged with XAck.
//
// Parameters:
//   - ctx: Context for cancellation (exits cleanly when ctx.Done())
//   - contractType: Contract type to subscribe to
//   - consumerGroup: Redis consumer group name
//   - consumerName: Consumer name within the group
//   - handler: Function to process each message (receives unmarshaled JSON)
//
// Returns:
//   - nil when context is cancelled
//   - error if consumer group creation fails or Redis operations fail
//
// Handler errors: If handler returns an error, the message is NOT acknowledged
// and will be redelivered to the consumer group.
func (b *EventBus) Subscribe(ctx context.Context, contractType string, consumerGroup string, consumerName string, handler func(map[string]interface{}) error) error {
	streamName := b.getStreamName(contractType)

	// Create consumer group if it doesn't exist
	// Use "0" to consume from beginning (for testing), "$" for production (new messages only)
	err := b.client.XGroupCreateMkStream(ctx, streamName, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group %s: %w", consumerGroup, err)
	}

	b.logger.Printf("INFO: Starting subscription: %s (group=%s, consumer=%s)", streamName, consumerGroup, consumerName)

	// Read loop
	for {
		// Read messages from consumer group with context
		streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{streamName, ">"},
			Count:    10,
			Block:    1000, // 1 second block timeout
		}).Result()

		// Check context cancellation first
		if ctx.Err() != nil {
			b.logger.Printf("INFO: Subscription stopped: %s (group=%s, consumer=%s)", streamName, consumerGroup, consumerName)
			return nil
		}

		if err != nil {
			// Timeout is expected - continue loop
			if err == redis.Nil {
				continue
			}
			// Other errors
			return fmt.Errorf("failed to read from %s: %w", streamName, err)
		}

		// Process messages
		for _, stream := range streams {
			for _, message := range stream.Messages {
				// Extract message data
				dataJSON, ok := message.Values["data"].(string)
				if !ok {
					b.logger.Printf("WARN: Message %s missing 'data' field", message.ID)
					continue
				}

				// Unmarshal JSON
				var messageData map[string]interface{}
				if err := json.Unmarshal([]byte(dataJSON), &messageData); err != nil {
					b.logger.Printf("ERROR: Failed to unmarshal message %s: %v", message.ID, err)
					continue
				}

				// Call handler
				if err := handler(messageData); err != nil {
					b.logger.Printf("ERROR: Handler failed for message %s: %v", message.ID, err)
					// Do NOT acknowledge - message will be redelivered
					continue
				}

				// Acknowledge message
				if err := b.client.XAck(ctx, streamName, consumerGroup, message.ID).Err(); err != nil {
					b.logger.Printf("ERROR: Failed to acknowledge message %s: %v", message.ID, err)
				}
			}
		}
	}
}
