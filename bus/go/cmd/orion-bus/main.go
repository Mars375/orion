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
	"github.com/yourusername/orion/bus/go/internal/bus"
	"github.com/yourusername/orion/bus/go/internal/shutdown"
)

const version = "0.1.0-alpha"

func main() {
	// Parse command-line flags
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis server address")
	contractsDir := flag.String("contracts-dir", "../../../contracts", "Path to JSON Schema contracts directory")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	streamPrefix := flag.String("stream-prefix", "orion", "Stream name prefix")
	maxStreamLen := flag.Int64("max-stream-len", 10000, "Maximum stream length (approximate)")
	httpPort := flag.String("http-port", "8080", "HTTP server port")
	flag.Parse()

	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("ORION Bus (Go) v%s starting on :%s", version, *httpPort)
	log.Printf("Redis address: %s", *redisAddr)
	log.Printf("Contracts directory: %s", *contractsDir)
	log.Printf("Stream prefix: %s", *streamPrefix)
	log.Printf("Log level: %s", *logLevel)

	// Set up graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create Redis client with connection pool configuration
	redisClient := redis.NewClient(&redis.Options{
		Addr:            *redisAddr,
		PoolSize:        100,
		MinIdleConns:    10,
		ConnMaxLifetime: 1 * time.Hour,
	})

	// Ping Redis to verify connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("FATAL: Failed to connect to Redis at %s: %v", *redisAddr, err)
	}
	log.Printf("INFO: Connected to Redis at %s", *redisAddr)

	// Create EventBus with contract validation
	eventBus, err := bus.NewEventBus(redisClient, *contractsDir, *streamPrefix, *maxStreamLen)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize EventBus: %v", err)
	}
	log.Printf("INFO: EventBus initialized with contract validation")

	// Create HTTP server with health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "orion-bus-go",
			"version": version,
		})
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", *httpPort),
		Handler: mux,
	}

	// Start HTTP server in background
	go func() {
		log.Printf("INFO: HTTP server listening on :%s", *httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ERROR: HTTP server failed: %v", err)
		}
	}()

	// Wait for shutdown signal and coordinate graceful shutdown
	coordinator := shutdown.NewCoordinator(25*time.Second, log.Default())
	if err := coordinator.WaitForShutdown(ctx, func(cleanupCtx context.Context) error {
		// Shutdown HTTP server
		log.Printf("INFO: Shutting down HTTP server...")
		if err := httpServer.Shutdown(cleanupCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}
		return nil
	}, func(cleanupCtx context.Context) error {
		// Close Redis client
		log.Printf("INFO: Closing Redis connection...")
		if err := redisClient.Close(); err != nil {
			return fmt.Errorf("Redis close failed: %w", err)
		}
		return nil
	}); err != nil {
		log.Printf("ERROR: Shutdown errors occurred: %v", err)
		os.Exit(1)
	}

	// Suppress unused variable warning for eventBus
	// In future, eventBus will be used for Subscribe/Publish operations
	_ = eventBus

	log.Printf("INFO: ORION Bus (Go) stopped cleanly")
}
