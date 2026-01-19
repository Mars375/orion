package worker

import (
	"context"
	"testing"
	"time"

	"github.com/orion/core/inference/contracts"
)

func TestNewHealthCollector(t *testing.T) {
	tests := []struct {
		name        string
		nodeID      string
		ollamaHost  string
		wantErr     bool
	}{
		{
			name:       "valid configuration",
			nodeID:     "test-node",
			ollamaHost: "http://localhost:11434",
			wantErr:    false,
		},
		{
			name:       "invalid URL",
			nodeID:     "test-node",
			ollamaHost: "://invalid",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc, err := NewHealthCollector(tt.nodeID, tt.ollamaHost)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHealthCollector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hc == nil {
				t.Error("NewHealthCollector() returned nil without error")
			}
			if !tt.wantErr && hc.nodeID != tt.nodeID {
				t.Errorf("NewHealthCollector() nodeID = %v, want %v", hc.nodeID, tt.nodeID)
			}
		})
	}
}

func TestCollectHealth_SystemMetrics(t *testing.T) {
	// This test verifies that system metrics can be collected
	// We don't mock gopsutil - we test actual system collection
	hc, err := NewHealthCollector("test-node", "http://localhost:11434")
	if err != nil {
		t.Fatalf("Failed to create HealthCollector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := hc.CollectHealth(ctx)
	if err != nil {
		t.Fatalf("CollectHealth() error = %v", err)
	}

	// Verify node ID is set
	if health.NodeID != "test-node" {
		t.Errorf("NodeID = %v, want %v", health.NodeID, "test-node")
	}

	// Verify LastSeen is recent
	if time.Since(health.LastSeen) > 5*time.Second {
		t.Errorf("LastSeen too old: %v", health.LastSeen)
	}

	// Verify RAM metrics are populated (should always work)
	if health.RAMTotalMB == 0 {
		t.Error("RAMTotalMB should be > 0")
	}

	// CPU percent should be 0-100 range
	if health.CPUPercent < 0 || health.CPUPercent > 100 {
		t.Errorf("CPUPercent out of range: %v", health.CPUPercent)
	}

	// RAM percent should be 0-100 range
	if health.RAMPercent < 0 || health.RAMPercent > 100 {
		t.Errorf("RAMPercent out of range: %v", health.RAMPercent)
	}
}

func TestIsAvailable_Thresholds(t *testing.T) {
	tests := []struct {
		name     string
		health   contracts.NodeHealth
		expected bool
	}{
		{
			name: "healthy node",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  60.0,
			},
			expected: true,
		},
		{
			name: "temperature too high",
			health: contracts.NodeHealth{
				TempCelsius: 80.0,
				RAMPercent:  60.0,
			},
			expected: false,
		},
		{
			name: "RAM too high",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  95.0,
			},
			expected: false,
		},
		{
			name: "exactly at temperature threshold",
			health: contracts.NodeHealth{
				TempCelsius: 75.0,
				RAMPercent:  60.0,
			},
			expected: true, // 75 is not > 75
		},
		{
			name: "just over temperature threshold",
			health: contracts.NodeHealth{
				TempCelsius: 75.1,
				RAMPercent:  60.0,
			},
			expected: false,
		},
		{
			name: "exactly at RAM threshold",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  90.0,
			},
			expected: true, // 90 is not > 90
		},
		{
			name: "just over RAM threshold",
			health: contracts.NodeHealth{
				TempCelsius: 50.0,
				RAMPercent:  90.1,
			},
			expected: false,
		},
		{
			name: "temperature at zero (unavailable metric)",
			health: contracts.NodeHealth{
				TempCelsius: 0,
				RAMPercent:  60.0,
			},
			expected: true, // 0 temp is acceptable (optional metric)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc, _ := NewHealthCollector("test", "http://localhost:11434")
			if got := hc.IsAvailable(tt.health); got != tt.expected {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNodeHealth_HasModel(t *testing.T) {
	health := contracts.NodeHealth{
		Models: []string{"gemma3:1b", "llama3.2:3b"},
	}

	tests := []struct {
		model    string
		expected bool
	}{
		{"gemma3:1b", true},
		{"llama3.2:3b", true},
		{"mistral:7b", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := health.HasModel(tt.model); got != tt.expected {
			t.Errorf("HasModel(%q) = %v, want %v", tt.model, got, tt.expected)
		}
	}
}

func TestNodeHealth_IsStale(t *testing.T) {
	tests := []struct {
		name     string
		lastSeen time.Time
		maxAge   time.Duration
		expected bool
	}{
		{
			name:     "fresh data",
			lastSeen: time.Now(),
			maxAge:   15 * time.Second,
			expected: false,
		},
		{
			name:     "stale data",
			lastSeen: time.Now().Add(-30 * time.Second),
			maxAge:   15 * time.Second,
			expected: true,
		},
		{
			name:     "just under threshold",
			lastSeen: time.Now().Add(-14 * time.Second),
			maxAge:   15 * time.Second,
			expected: false, // Not stale when under threshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := contracts.NodeHealth{LastSeen: tt.lastSeen}
			if got := health.IsStale(tt.maxAge); got != tt.expected {
				t.Errorf("IsStale() = %v, want %v", got, tt.expected)
			}
		})
	}
}
