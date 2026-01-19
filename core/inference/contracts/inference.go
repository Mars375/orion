// Package contracts defines shared data types for the ORION inference subsystem.
// These types are used by both workers and routers for health reporting and request routing.
package contracts

import (
	"time"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", or "system"
	Content string `json:"content"` // Message content
}

// InferenceRequest represents a request for LLM inference.
// Submitted by Brain to the inference routing system.
type InferenceRequest struct {
	// Required fields (per inference.request.schema.json)
	Version   string    `json:"version"`    // Schema version, always "1.0"
	RequestID string    `json:"request_id"` // UUID for request tracking
	Model     string    `json:"model"`      // Model name (e.g., "gemma3:1b", "llama3.2:3b")
	Messages  []Message `json:"messages"`   // Conversation messages
	Timestamp time.Time `json:"timestamp"`  // Request timestamp
	Source    string    `json:"source"`     // Originating module (e.g., "orion-brain")

	// Optional fields
	KeepAliveSeconds int    `json:"keep_alive_seconds,omitempty"` // Model residency in seconds (default 600)
	Callback         string `json:"callback,omitempty"`           // Response stream name for async response
}

// KeepAliveDuration returns the KeepAlive as a time.Duration.
// Defaults to 10 minutes if not specified.
func (r *InferenceRequest) KeepAliveDuration() time.Duration {
	if r.KeepAliveSeconds <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(r.KeepAliveSeconds) * time.Second
}

// InferenceResponse represents the response from an LLM inference request.
type InferenceResponse struct {
	// Required fields (per inference.response.schema.json)
	Version   string    `json:"version"`    // Schema version, always "1.0"
	RequestID string    `json:"request_id"` // Matches InferenceRequest.RequestID
	Model     string    `json:"model"`      // Model that processed the request
	Timestamp time.Time `json:"timestamp"`  // Response timestamp
	Source    string    `json:"source"`     // Responding worker (e.g., "orion-inference-worker-pi8g")

	// Response content
	Response string `json:"response,omitempty"` // Generated text content

	// Performance metrics
	PromptTokens     int   `json:"prompt_tokens,omitempty"`     // Input tokens processed
	CompletionTokens int   `json:"completion_tokens,omitempty"` // Output tokens generated
	LoadDurationMs   int64 `json:"load_duration_ms,omitempty"`  // Model load time (0 if already resident)
	TotalDurationMs  int64 `json:"total_duration_ms,omitempty"` // Total inference time

	// Error handling
	Error string `json:"error,omitempty"` // Error message if inference failed (empty on success)
}

// IsSuccess returns true if the response completed without error.
func (r *InferenceResponse) IsSuccess() bool {
	return r.Error == ""
}

// NodeHealth represents the health status of an inference worker node.
// Published to Redis for routing decisions.
type NodeHealth struct {
	// Node identification
	NodeID string `json:"node_id"` // Unique node identifier (e.g., "pi-16g", "pi-8g")

	// System metrics
	CPUPercent float64 `json:"cpu_percent"` // CPU usage percentage (0-100)
	RAMPercent float64 `json:"ram_percent"` // RAM usage percentage (0-100)
	RAMUsedMB  int64   `json:"ram_used_mb"` // RAM used in megabytes
	RAMTotalMB int64   `json:"ram_total_mb"` // Total RAM in megabytes
	TempCelsius float64 `json:"temp_celsius"` // CPU temperature in Celsius (0 if unavailable)

	// Model residency (for sticky routing)
	Models []string `json:"models"` // Currently loaded model names

	// Availability
	Available bool      `json:"available"` // Whether node accepts new requests
	LastSeen  time.Time `json:"last_seen"` // Last health update timestamp
}

// IsHealthy checks if the node meets health thresholds for routing.
// Returns false if temperature > 75°C or RAM > 90%.
func (h *NodeHealth) IsHealthy() bool {
	if h.TempCelsius > 75 {
		return false
	}
	if h.RAMPercent > 90 {
		return false
	}
	return true
}

// HasModel checks if the node has the specified model loaded in memory.
func (h *NodeHealth) HasModel(model string) bool {
	for _, m := range h.Models {
		if m == model {
			return true
		}
	}
	return false
}

// IsStale checks if the health data is older than the given duration.
func (h *NodeHealth) IsStale(maxAge time.Duration) bool {
	return time.Since(h.LastSeen) > maxAge
}
