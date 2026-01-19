package config

import (
	"flag"
	"fmt"
)

// Config holds the configuration for the orion-edge agent.
type Config struct {
	// DeviceID is the unique identifier for this edge device (e.g., "hexapod-1")
	DeviceID string

	// RedisAddr is the address of the Brain's Redis server
	RedisAddr string

	// RedisPassword is the Redis authentication password (empty if none)
	RedisPassword string

	// MQTTBrokerURL is the MQTT broker URL (e.g., "tcp://192.168.1.100:1883")
	MQTTBrokerURL string

	// MQTTClientID is derived from DeviceID for MQTT connections
	MQTTClientID string

	// HeartbeatIntervalSec is the interval between health heartbeats
	HeartbeatIntervalSec int

	// WatchdogTimeoutSec is the Dead Man's Switch timeout
	WatchdogTimeoutSec int

	// StreamPrefix is the prefix for Redis stream names
	StreamPrefix string

	// HTTPPort is the port for the health endpoint
	HTTPPort string
}

// LoadFromFlags parses command-line flags and returns a Config.
func LoadFromFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.DeviceID, "device-id", "", "Edge device identifier (required)")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", "localhost:6379", "Brain's Redis address")
	flag.StringVar(&cfg.RedisPassword, "redis-password", "", "Redis password (empty if none)")
	flag.StringVar(&cfg.MQTTBrokerURL, "mqtt-broker", "tcp://localhost:1883", "MQTT broker URL")
	flag.IntVar(&cfg.HeartbeatIntervalSec, "heartbeat-interval", 1, "Heartbeat interval in seconds")
	flag.IntVar(&cfg.WatchdogTimeoutSec, "watchdog-timeout", 5, "Dead Man's Switch timeout in seconds")
	flag.StringVar(&cfg.StreamPrefix, "stream-prefix", "orion", "Redis stream name prefix")
	flag.StringVar(&cfg.HTTPPort, "http-port", "8081", "Health endpoint HTTP port")

	flag.Parse()

	// Derive MQTT client ID from device ID
	if cfg.DeviceID != "" {
		cfg.MQTTClientID = fmt.Sprintf("orion-edge-%s", cfg.DeviceID)
	}

	return cfg
}

// Validate checks that all required configuration fields are set.
func (c *Config) Validate() error {
	if c.DeviceID == "" {
		return fmt.Errorf("--device-id is required")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("--redis-addr is required")
	}
	if c.MQTTBrokerURL == "" {
		return fmt.Errorf("--mqtt-broker is required")
	}
	if c.HeartbeatIntervalSec <= 0 {
		return fmt.Errorf("--heartbeat-interval must be positive")
	}
	if c.WatchdogTimeoutSec <= 0 {
		return fmt.Errorf("--watchdog-timeout must be positive")
	}
	return nil
}
