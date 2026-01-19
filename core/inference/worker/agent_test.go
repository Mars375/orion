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

func setupAgentTest(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
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

func TestNewWorkerAgent(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	agent, err := NewWorkerAgent("test-node", "http://localhost:11434", client, "")
	if err != nil {
		t.Fatalf("NewWorkerAgent() error = %v", err)
	}

	if agent.nodeID != "test-node" {
		t.Errorf("nodeID = %v, want %v", agent.nodeID, "test-node")
	}

	if agent.streamPrefix != "orion:inference" {
		t.Errorf("streamPrefix = %v, want orion:inference", agent.streamPrefix)
	}
}

func TestNewWorkerAgent_InvalidOllamaHost(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	_, err := NewWorkerAgent("test-node", "://invalid", client, "")
	if err == nil {
		t.Error("NewWorkerAgent() should fail with invalid Ollama host")
	}
}

func TestWorkerAgent_GetNodeID(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	agent, _ := NewWorkerAgent("pi-8g", "http://localhost:11434", client, "")

	if agent.GetNodeID() != "pi-8g" {
		t.Errorf("GetNodeID() = %v, want pi-8g", agent.GetNodeID())
	}
}

func TestWorkerAgent_GetStreamName(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	agent, _ := NewWorkerAgent("pi-8g", "http://localhost:11434", client, "orion:inference")

	expected := "orion:inference:requests:pi-8g"
	if agent.GetStreamName() != expected {
		t.Errorf("GetStreamName() = %v, want %v", agent.GetStreamName(), expected)
	}
}

func TestWorkerAgent_PublishHealth(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	agent, err := NewWorkerAgent("test-node", "http://localhost:11434", client, "")
	if err != nil {
		t.Fatalf("NewWorkerAgent() error = %v", err)
	}

	ctx := context.Background()

	// Manually trigger health publish
	agent.publishHealth(ctx)

	// Verify health was published to Redis
	data, err := client.HGet(ctx, DefaultHealthHashKey, "test-node").Result()
	if err != nil {
		t.Fatalf("Failed to get health data: %v", err)
	}

	var health contracts.NodeHealth
	if err := json.Unmarshal([]byte(data), &health); err != nil {
		t.Fatalf("Failed to unmarshal health: %v", err)
	}

	if health.NodeID != "test-node" {
		t.Errorf("Health NodeID = %v, want test-node", health.NodeID)
	}

	// RAM should be populated (non-zero on any system)
	if health.RAMTotalMB == 0 {
		t.Error("Health RAMTotalMB should be > 0")
	}
}

func TestWorkerAgent_Stop(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	agent, err := NewWorkerAgent("test-node", "http://localhost:11434", client, "")
	if err != nil {
		t.Fatalf("NewWorkerAgent() error = %v", err)
	}

	ctx := context.Background()

	// Publish health first
	agent.publishHealth(ctx)

	// Verify it's in Redis
	exists, _ := client.HExists(ctx, DefaultHealthHashKey, "test-node").Result()
	if !exists {
		t.Fatal("Health entry should exist before Stop()")
	}

	// Stop the agent
	if err := agent.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Verify it's removed from Redis
	exists, _ = client.HExists(ctx, DefaultHealthHashKey, "test-node").Result()
	if exists {
		t.Error("Health entry should be removed after Stop()")
	}
}

func TestWorkerAgent_HealthPublishLoop(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	agent, err := NewWorkerAgent("loop-test", "http://localhost:11434", client, "")
	if err != nil {
		t.Fatalf("NewWorkerAgent() error = %v", err)
	}

	// Use a context with enough time for CPU sampling (1 second) plus margin
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the health loop in background
	done := make(chan struct{})
	go func() {
		agent.publishHealthLoop(ctx)
		close(done)
	}()

	// Wait for context to expire
	<-ctx.Done()

	// Wait for goroutine to finish
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Health loop did not stop")
	}

	// Verify health was published at least once
	exists, _ := client.HExists(context.Background(), DefaultHealthHashKey, "loop-test").Result()
	if !exists {
		t.Error("Health should have been published at least once")
	}
}

func TestWorkerAgent_StartPreventsDoubleStart(t *testing.T) {
	mr, client := setupAgentTest(t)
	defer mr.Close()
	defer client.Close()

	agent, err := NewWorkerAgent("double-start", "http://localhost:11434", client, "")
	if err != nil {
		t.Fatalf("NewWorkerAgent() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start first instance
	startDone := make(chan error)
	go func() {
		startDone <- agent.Start(ctx)
	}()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Try to start again - should fail
	err = agent.Start(ctx)
	if err == nil {
		t.Error("Start() should fail when already running")
	}

	// Clean up
	cancel()
	<-startDone
}

func TestBoolPtr(t *testing.T) {
	truePtr := boolPtr(true)
	falsePtr := boolPtr(false)

	if *truePtr != true {
		t.Error("boolPtr(true) should return pointer to true")
	}
	if *falsePtr != false {
		t.Error("boolPtr(false) should return pointer to false")
	}
}
