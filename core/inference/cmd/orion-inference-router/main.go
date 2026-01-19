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

	"github.com/orion/core/inference/router"
)

const version = "0.1.0"

func main() {
	// Parse command-line flags
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis server address")
	redisPassword := flag.String("redis-password", "", "Redis password")
	httpPort := flag.String("http-port", "8080", "HTTP health endpoint port")
	streamPrefix := flag.String("stream-prefix", "orion:inference", "Redis stream prefix")
	flag.Parse()

	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[router] ")

	log.Printf("ORION Inference Router v%s starting", version)
	log.Printf("Redis: %s", *redisAddr)
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

	// Create health reader, sticky router, and inference router
	healthReader := router.NewHealthReader(redisClient, "")
	stickyRouter := router.NewStickyRouter(healthReader)
	inferenceRouter := router.NewInferenceRouter(redisClient, stickyRouter, *streamPrefix)

	// Create HTTP server with health and stats endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "orion-inference-router",
			"version": version,
		})
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(inferenceRouter.GetRoutingStats())
	})

	mux.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes, err := healthReader.GetHealthyNodes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", *httpPort),
		Handler: mux,
	}

	// Start HTTP server in background
	go func() {
		log.Printf("INFO: HTTP endpoints listening on :%s", *httpPort)
		log.Printf("INFO: Available endpoints: /health, /stats, /nodes")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ERROR: HTTP server failed: %v", err)
		}
	}()

	// Start router consumer in background
	routerDone := make(chan error, 1)
	go func() {
		routerDone <- inferenceRouter.ConsumeRequests(ctx)
	}()

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Printf("INFO: Shutdown signal received")
	case err := <-routerDone:
		if err != nil && err != context.Canceled {
			log.Printf("ERROR: Router stopped with error: %v", err)
		}
	}

	// Graceful shutdown with 25s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	log.Printf("INFO: Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: HTTP server shutdown failed: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("ERROR: Redis close failed: %v", err)
	}

	// Print final stats
	stats := inferenceRouter.GetRoutingStats()
	log.Printf("INFO: Final routing stats: %v", stats)

	log.Printf("INFO: ORION Inference Router stopped cleanly")
}
