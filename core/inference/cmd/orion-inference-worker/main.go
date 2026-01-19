package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orion/core/inference/worker"
)

const version = "0.1.0"

func main() {
	// Parse command-line flags
	nodeID := flag.String("node-id", "", "Unique node identifier (required)")
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis server address")
	redisPassword := flag.String("redis-password", "", "Redis password")
	ollamaHost := flag.String("ollama-host", "http://localhost:11434", "Ollama server URL")
	httpPort := flag.String("http-port", "8081", "HTTP health endpoint port")
	streamPrefix := flag.String("stream-prefix", "orion:inference", "Redis stream prefix")
	flag.Parse()

	if *nodeID == "" {
		fmt.Fprintln(os.Stderr, "Error: --node-id is required")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("[%s] ", *nodeID))

	log.Printf("ORION Inference Worker v%s starting (node: %s)", version, *nodeID)
	log.Printf("Redis: %s", *redisAddr)
	log.Printf("Ollama: %s", *ollamaHost)
	log.Printf("HTTP port: %s", *httpPort)

	// Set up graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:            *redisAddr,
		Password:        *redisPassword,
		PoolSize:        10,
		MinIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("FATAL: Failed to connect to Redis at %s: %v", *redisAddr, err)
	}
	log.Printf("INFO: Connected to Redis at %s", *redisAddr)

	// Create worker agent
	agent, err := worker.NewWorkerAgent(*nodeID, *ollamaHost, redisClient, *streamPrefix)
	if err != nil {
		log.Fatalf("FATAL: Failed to create worker agent: %v", err)
	}

	// Create HTTP server with health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "ok",
			"service":  "orion-inference-worker",
			"node_id":  *nodeID,
			"version":  version,
			"ollama":   *ollamaHost,
		})
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", *httpPort),
		Handler: mux,
	}

	// Start HTTP server in background
	go func() {
		log.Printf("INFO: HTTP health endpoint listening on :%s", *httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ERROR: HTTP server failed: %v", err)
		}
	}()

	// Start worker agent in background
	agentDone := make(chan error, 1)
	go func() {
		agentDone <- agent.Start(ctx)
	}()

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Printf("INFO: Shutdown signal received")
	case err := <-agentDone:
		if err != nil {
			log.Printf("ERROR: Worker agent stopped with error: %v", err)
		}
	}

	// Graceful shutdown with 25s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer shutdownCancel()

	// Stop worker agent (removes from health registry)
	log.Printf("INFO: Stopping worker agent...")
	if err := agent.Stop(shutdownCtx); err != nil {
		log.Printf("ERROR: Worker agent stop failed: %v", err)
	}

	// Shutdown HTTP server
	log.Printf("INFO: Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: HTTP server shutdown failed: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("ERROR: Redis close failed: %v", err)
	}

	log.Printf("INFO: ORION Inference Worker stopped cleanly")
}
