package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the go-redis client for edge agent operations.
type RedisClient struct {
	client       *redis.Client
	streamPrefix string
	deviceID     string
}

// NewRedisClient creates a new RedisClient for the edge agent.
func NewRedisClient(addr, password, streamPrefix, deviceID string) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		PoolSize:        10,
		MinIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
	})

	return &RedisClient{
		client:       client,
		streamPrefix: streamPrefix,
		deviceID:     deviceID,
	}
}

// Connect establishes connection to Redis with a timeout.
func (r *RedisClient) Connect(ctx context.Context) error {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := r.client.Ping(connectCtx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("INFO: Connected to Redis")
	return nil
}

// Close closes the Redis connection.
func (r *RedisClient) Close() error {
	log.Printf("INFO: Closing Redis connection...")
	return r.client.Close()
}

// PublishTelemetry publishes telemetry data to the telemetry stream.
func (r *RedisClient) PublishTelemetry(ctx context.Context, telemetry map[string]interface{}) error {
	streamName := fmt.Sprintf("%s:edge:telemetry", r.streamPrefix)

	// Serialize telemetry to JSON for storage
	data, err := json.Marshal(telemetry)
	if err != nil {
		return fmt.Errorf("failed to marshal telemetry: %w", err)
	}

	_, err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{
			"device_id": r.deviceID,
			"data":      string(data),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to publish telemetry: %w", err)
	}

	return nil
}

// SubscribeCommands subscribes to the command stream and calls handler for each message.
// This method blocks until the context is cancelled.
func (r *RedisClient) SubscribeCommands(ctx context.Context, handler func(map[string]interface{}) error) error {
	streamName := fmt.Sprintf("%s:edge:commands:%s", r.streamPrefix, r.deviceID)
	groupName := fmt.Sprintf("edge-%s", r.deviceID)
	consumerName := fmt.Sprintf("edge-%s-consumer", r.deviceID)

	// Create consumer group if it doesn't exist
	err := r.client.XGroupCreateMkStream(ctx, streamName, groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("WARN: Could not create consumer group: %v", err)
	}

	log.Printf("INFO: Subscribing to commands on stream %s", streamName)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read from stream with blocking
		streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{streamName, ">"},
			Count:    1,
			Block:    1 * time.Second,
		}).Result()

		if err == redis.Nil {
			// No messages, continue polling
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("WARN: Error reading commands: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Process messages
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				// Parse message data
				data := make(map[string]interface{})
				for k, v := range msg.Values {
					data[k] = v
				}
				data["_message_id"] = msg.ID

				if err := handler(data); err != nil {
					log.Printf("ERROR: Failed to handle command %s: %v", msg.ID, err)
				}

				// Acknowledge message
				r.client.XAck(ctx, streamName, groupName, msg.ID)
			}
		}
	}
}

// Ping checks the Redis connection.
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
