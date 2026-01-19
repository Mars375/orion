package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/orion/edge/internal/client"
	"github.com/yourusername/orion/edge/internal/config"
	"github.com/yourusername/orion/edge/internal/safety"
)

const version = "0.1.0"

func main() {
	// Parse command-line flags
	cfg := config.LoadFromFlags()

	// Initialize logger with device ID prefix
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("[%s] ", cfg.DeviceID))

	log.Printf("ORION Edge Agent v%s starting (device: %s)", version, cfg.DeviceID)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("FATAL: Invalid configuration: %v", err)
	}

	log.Printf("Redis address: %s", cfg.RedisAddr)
	log.Printf("MQTT broker: %s", cfg.MQTTBrokerURL)
	log.Printf("Heartbeat interval: %ds", cfg.HeartbeatIntervalSec)
	log.Printf("Watchdog timeout: %ds", cfg.WatchdogTimeoutSec)

	// Set up graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create SafeStateManager with stub callbacks (kinematics not implemented in Phase 6)
	safeState := safety.NewSafeStateManager(
		func() { log.Println("STUB: Would move to Sit & Freeze position") },
		func() { log.Println("STUB: Would resume normal operations") },
	)

	// Create DeadManSwitch with configured timeout
	watchdog := safety.NewDeadManSwitch(
		time.Duration(cfg.WatchdogTimeoutSec)*time.Second,
		func() {
			log.Println("SAFETY: Dead Man's Switch triggered!")
			safeState.EnterSafeMode()
		},
	)
	defer watchdog.Stop()

	// Create Redis client and connect (fail-fast if Brain unreachable)
	redisClient := client.NewRedisClient(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.StreamPrefix,
		cfg.DeviceID,
	)

	if err := redisClient.Connect(ctx); err != nil {
		log.Fatalf("FATAL: Failed to connect to Redis at %s: %v", cfg.RedisAddr, err)
	}
	log.Printf("INFO: Connected to Redis at %s", cfg.RedisAddr)

	// Reset watchdog on successful Redis connection
	watchdog.Reset()

	// Create MQTT client and connect (fail-fast if broker unreachable)
	mqttClient := client.NewMQTTClient(
		cfg.MQTTBrokerURL,
		cfg.MQTTClientID,
		cfg.DeviceID,
	)

	// Wire MQTT connection callbacks for Dead Man's Switch
	mqttClient.SetOnConnectionUp(func() {
		log.Printf("INFO: Brain connection restored via MQTT")
		// Reset watchdog on reconnection, but do NOT clear triggered state
		// Explicit RESUME command required to exit safe mode
		watchdog.Reset()
	})
	mqttClient.SetOnConnectionDown(func(err error) {
		log.Printf("WARN: Brain connection lost via MQTT: %v - watchdog active", err)
		// Watchdog timer continues running, will trigger safe state if not reset
	})

	if err := mqttClient.Connect(ctx); err != nil {
		log.Fatalf("FATAL: Failed to connect to MQTT broker at %s: %v", cfg.MQTTBrokerURL, err)
	}
	log.Printf("INFO: Connected to MQTT broker at %s", cfg.MQTTBrokerURL)

	// Reset watchdog on successful MQTT connection
	watchdog.Reset()

	// Create HTTP server with health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "ok",
			"service":            "orion-edge",
			"device_id":          cfg.DeviceID,
			"version":            version,
			"mqtt_connected":     mqttClient.IsConnected(),
			"safe_mode":          safeState.IsInSafeMode(),
			"watchdog_triggered": watchdog.IsTriggered(),
		})
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: mux,
	}

	// Start HTTP server in background
	go func() {
		log.Printf("INFO: HTTP health endpoint listening on :%s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ERROR: HTTP server failed: %v", err)
		}
	}()

	// Start heartbeat publisher goroutine
	var wg sync.WaitGroup
	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		runHeartbeat(heartbeatCtx, cfg, mqttClient, redisClient, watchdog, safeState)
	}()

	// Start command handler goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		handleCommands(ctx, mqttClient, redisClient, watchdog, safeState)
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Printf("INFO: Shutdown signal received")

	// Graceful shutdown with 15s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// Stop heartbeat and command handler
	log.Printf("INFO: Stopping background goroutines...")
	heartbeatCancel()
	wg.Wait()

	// Shutdown HTTP server
	log.Printf("INFO: Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: HTTP server shutdown failed: %v", err)
	}

	// Close MQTT connection
	if err := mqttClient.Close(shutdownCtx); err != nil {
		log.Printf("ERROR: MQTT close failed: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("ERROR: Redis close failed: %v", err)
	}

	log.Printf("INFO: ORION Edge Agent stopped cleanly")
}

// handleCommands subscribes to commands and processes them.
func handleCommands(ctx context.Context, mqtt *client.MQTTClient, redis *client.RedisClient, watchdog *safety.DeadManSwitch, safeState *safety.SafeStateManager) {
	// Subscribe to MQTT commands
	err := mqtt.SubscribeCommands(ctx, func(topic string, payload []byte) {
		log.Printf("INFO: Received command on topic %s", topic)

		// Reset watchdog on any command from Brain
		watchdog.Reset()

		// Parse command
		var cmd map[string]interface{}
		if err := json.Unmarshal(payload, &cmd); err != nil {
			log.Printf("ERROR: Failed to parse command: %v", err)
			return
		}

		cmdType, ok := cmd["command_type"].(string)
		if !ok {
			log.Printf("ERROR: Command missing command_type field")
			return
		}

		// Handle commands
		switch cmdType {
		case "RESUME":
			handleResumeCommand(watchdog, safeState)
		case "STOP":
			handleStopCommand(safeState, cmd)
		case "MOVE":
			handleMoveCommand(safeState, cmd)
		case "CALIBRATE":
			handleCalibrateCommand(safeState, cmd)
		case "STATUS":
			log.Printf("INFO: Received STATUS command - reporting via heartbeat")
		default:
			log.Printf("WARN: Unknown command type: %s", cmdType)
		}
	})

	if err != nil {
		log.Printf("ERROR: Failed to subscribe to commands: %v", err)
	}

	// Block until context is cancelled
	<-ctx.Done()
	log.Printf("INFO: Command handler stopped")
}

// handleResumeCommand processes RESUME commands to exit safe mode.
func handleResumeCommand(watchdog *safety.DeadManSwitch, safeState *safety.SafeStateManager) {
	log.Printf("INFO: Received RESUME command")

	if !safeState.IsInSafeMode() {
		log.Printf("INFO: Not in safe mode, RESUME command ignored")
		return
	}

	// Clear watchdog triggered state and exit safe mode
	watchdog.ClearTriggered()
	if err := safeState.ExitSafeMode(); err != nil {
		log.Printf("ERROR: Failed to exit safe mode: %v", err)
		return
	}

	log.Printf("INFO: Exited safe mode via RESUME command")
}

// handleStopCommand processes STOP commands.
func handleStopCommand(safeState *safety.SafeStateManager, cmd map[string]interface{}) {
	reason := ""
	if params, ok := cmd["parameters"].(map[string]interface{}); ok {
		if r, ok := params["reason"].(string); ok {
			reason = r
		}
	}

	if safeState.IsInSafeMode() {
		log.Printf("INFO: STOP command received but already in safe mode")
		return
	}

	log.Printf("INFO: STOP command received (reason: %s) - STUB: would stop movement", reason)
}

// handleMoveCommand processes MOVE commands.
func handleMoveCommand(safeState *safety.SafeStateManager, cmd map[string]interface{}) {
	if safeState.IsInSafeMode() {
		log.Printf("WARN: MOVE command rejected - in safe mode (requires RESUME first)")
		return
	}

	params, _ := cmd["parameters"].(map[string]interface{})
	direction, _ := params["direction"].(string)
	speed, _ := params["speed"].(float64)

	log.Printf("INFO: MOVE command received (direction: %s, speed: %.2f) - STUB: would execute movement", direction, speed)
}

// handleCalibrateCommand processes CALIBRATE commands.
func handleCalibrateCommand(safeState *safety.SafeStateManager, cmd map[string]interface{}) {
	if safeState.IsInSafeMode() {
		log.Printf("WARN: CALIBRATE command rejected - in safe mode (requires RESUME first)")
		return
	}

	params, _ := cmd["parameters"].(map[string]interface{})
	calibrationType, _ := params["calibration_type"].(string)

	log.Printf("INFO: CALIBRATE command received (type: %s) - STUB: would execute calibration", calibrationType)
}

// runHeartbeat publishes periodic health messages.
func runHeartbeat(ctx context.Context, cfg *config.Config, mqtt *client.MQTTClient, redis *client.RedisClient, watchdog *safety.DeadManSwitch, safeState *safety.SafeStateManager) {
	ticker := time.NewTicker(time.Duration(cfg.HeartbeatIntervalSec) * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Printf("INFO: Heartbeat publisher stopped")
			return
		case <-ticker.C:
			health := buildHealthMessage(cfg, mqtt, redis, watchdog, safeState, startTime)
			if err := mqtt.PublishHealth(ctx, health); err != nil {
				log.Printf("WARN: Failed to publish health via MQTT: %v", err)
			}
		}
	}
}

// buildHealthMessage creates a health message following edge.health.schema.json.
func buildHealthMessage(cfg *config.Config, mqtt *client.MQTTClient, redis *client.RedisClient, watchdog *safety.DeadManSwitch, safeState *safety.SafeStateManager, startTime time.Time) map[string]interface{} {
	now := time.Now().UTC()

	// Determine state based on connections and safety
	state := "RUNNING"
	if safeState.IsInSafeMode() {
		state = "SAFE_MODE"
	} else if !mqtt.IsConnected() {
		state = "ERROR"
	}

	// Check Redis connection
	redisConnected := true
	if err := redis.Ping(context.Background()); err != nil {
		redisConnected = false
		if state == "RUNNING" {
			state = "ERROR"
		}
	}

	// Collect errors
	var errors []string
	if !mqtt.IsConnected() {
		errors = append(errors, "mqtt_disconnected")
	}
	if !redisConnected {
		errors = append(errors, "redis_disconnected")
	}
	if watchdog.IsTriggered() {
		errors = append(errors, "watchdog_triggered")
	}

	return map[string]interface{}{
		"version":        "1.0",
		"health_id":      uuid.New().String(),
		"timestamp":      now.Format(time.RFC3339),
		"source":         fmt.Sprintf("orion-edge-%s", cfg.DeviceID),
		"device_id":      cfg.DeviceID,
		"state":          state,
		"uptime_seconds": int(now.Sub(startTime).Seconds()),
		"connection_status": map[string]interface{}{
			"mqtt_connected":     mqtt.IsConnected(),
			"redis_connected":    redisConnected,
			"last_brain_contact": now.Format(time.RFC3339),
		},
		"safety_state": map[string]interface{}{
			"dead_man_switch_active": watchdog.IsTriggered(),
			"watchdog_remaining_ms":  watchdog.RemainingMs(),
			"in_safe_position":       safeState.IsInSafeMode(),
		},
		"errors": errors,
	}
}
