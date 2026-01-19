// Package worker implements the ORION inference worker agent.
// Workers run on Pi nodes and handle LLM inference requests via Ollama.
package worker

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/sensors"

	"github.com/orion/core/inference/contracts"
)

// HealthCollector gathers system health metrics for routing decisions.
type HealthCollector struct {
	nodeID      string
	ollamaHost  string
	ollamaClient *api.Client
}

// NewHealthCollector creates a new HealthCollector.
// ollamaHost should be the Ollama server URL (e.g., "http://localhost:11434").
func NewHealthCollector(nodeID, ollamaHost string) (*HealthCollector, error) {
	// Parse the Ollama host URL
	u, err := url.Parse(ollamaHost)
	if err != nil {
		return nil, err
	}

	// Create Ollama client
	client := api.NewClient(u, http.DefaultClient)

	return &HealthCollector{
		nodeID:       nodeID,
		ollamaHost:   ollamaHost,
		ollamaClient: client,
	}, nil
}

// CollectHealth gathers current system health metrics.
// Returns NodeHealth with CPU, RAM, temperature, and loaded models.
// Does not fail if Ollama is temporarily unreachable.
func (h *HealthCollector) CollectHealth(ctx context.Context) (contracts.NodeHealth, error) {
	health := contracts.NodeHealth{
		NodeID:   h.nodeID,
		LastSeen: time.Now().UTC(),
	}

	// Collect CPU usage (1 second sample)
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		log.Printf("WARN: Failed to collect CPU metrics: %v", err)
	} else if len(cpuPercent) > 0 {
		health.CPUPercent = cpuPercent[0]
	}

	// Collect memory usage
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		log.Printf("WARN: Failed to collect memory metrics: %v", err)
	} else {
		health.RAMPercent = vmem.UsedPercent
		health.RAMUsedMB = int64(vmem.Used / 1024 / 1024)
		health.RAMTotalMB = int64(vmem.Total / 1024 / 1024)
	}

	// Collect CPU temperature
	health.TempCelsius = h.getCPUTemperature(ctx)

	// Get loaded models from Ollama
	health.Models = h.getLoadedModels(ctx)

	// Determine availability based on health thresholds
	health.Available = health.IsHealthy()

	return health, nil
}

// getCPUTemperature returns the CPU temperature in Celsius.
// Tries multiple methods: gopsutil sensors, then /sys/class/thermal fallback.
// Returns 0 if temperature cannot be read (optional metric).
func (h *HealthCollector) getCPUTemperature(ctx context.Context) float64 {
	// Try gopsutil sensors first (moved to sensors package in gopsutil v4)
	temps, err := sensors.TemperaturesWithContext(ctx)
	if err == nil {
		for _, t := range temps {
			// Look for common CPU temperature sensor names
			key := strings.ToLower(t.SensorKey)
			if key == "cpu_thermal" || key == "cpu-thermal" ||
			   strings.Contains(key, "coretemp") || strings.Contains(key, "k10temp") {
				return t.Temperature
			}
		}
	}

	// Fallback to reading /sys/class/thermal directly (common on Raspberry Pi)
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err == nil {
		tempStr := strings.TrimSpace(string(data))
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			// Temperature is in millidegrees Celsius
			return temp / 1000.0
		}
	}

	// Temperature unavailable - return 0 (optional metric)
	return 0
}

// getLoadedModels queries Ollama for currently loaded models.
// Returns empty slice if Ollama is unreachable (resilient to temporary failures).
func (h *HealthCollector) getLoadedModels(ctx context.Context) []string {
	if h.ollamaClient == nil {
		return []string{}
	}

	// Use the ps endpoint to get running models
	resp, err := h.ollamaClient.ListRunning(ctx)
	if err != nil {
		// Don't fail health collection if Ollama is temporarily unreachable
		log.Printf("DEBUG: Could not query Ollama for loaded models: %v", err)
		return []string{}
	}

	models := make([]string, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, m.Name)
	}

	return models
}

// IsAvailable checks if the node should accept new inference requests.
// Returns false if temperature > 75°C or RAM > 90%.
func (h *HealthCollector) IsAvailable(health contracts.NodeHealth) bool {
	return health.IsHealthy()
}
