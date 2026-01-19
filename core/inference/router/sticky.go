package router

import (
	"context"
	"errors"

	"github.com/orion/core/inference/contracts"
)

// ErrNoAvailableNodes indicates no healthy inference nodes are available for routing.
var ErrNoAvailableNodes = errors.New("no available inference nodes")

// StickyRouter implements model-aware sticky routing for inference requests.
// It prioritizes nodes that already have the requested model loaded to avoid
// cold-start latency from model loading.
type StickyRouter struct {
	healthReader *HealthReader
}

// NewStickyRouter creates a new StickyRouter with the given health reader.
func NewStickyRouter(healthReader *HealthReader) *StickyRouter {
	return &StickyRouter{
		healthReader: healthReader,
	}
}

// SelectNode chooses the best node for processing a request for the given model.
// Algorithm:
//  1. Get all healthy nodes (filtered and sorted by RAM usage ascending)
//  2. First pass: Return first node that has the model already loaded (sticky hit)
//  3. Second pass: Return first node in the list (least loaded)
//
// Returns ErrNoAvailableNodes if no healthy nodes are available.
func (s *StickyRouter) SelectNode(ctx context.Context, model string) (string, error) {
	nodes, err := s.healthReader.GetHealthyNodes(ctx)
	if err != nil {
		return "", err
	}

	if len(nodes) == 0 {
		return "", ErrNoAvailableNodes
	}

	// First pass: Look for node with model already loaded (sticky routing)
	// Iterate in RAM-sorted order to prefer least-loaded among nodes with model
	for _, node := range nodes {
		if node.HasModel(model) {
			return node.NodeID, nil
		}
	}

	// Second pass: No node has model loaded, return least-loaded node
	// nodes is already sorted by RAMPercent ascending
	return nodes[0].NodeID, nil
}

// GetModelResidency returns a list of nodeIDs that currently have the model loaded.
// Useful for metrics and debugging sticky routing effectiveness.
func (s *StickyRouter) GetModelResidency(ctx context.Context, model string) ([]string, error) {
	nodes, err := s.healthReader.GetHealthyNodes(ctx)
	if err != nil {
		return nil, err
	}

	residency := make([]string, 0)
	for _, node := range nodes {
		if node.HasModel(model) {
			residency = append(residency, node.NodeID)
		}
	}

	return residency, nil
}

// GetModelResidencyFromNodes returns nodeIDs that have the model from a given list.
// Useful when you already have a list of nodes and want to check residency.
func GetModelResidencyFromNodes(nodes []contracts.NodeHealth, model string) []string {
	residency := make([]string, 0)
	for _, node := range nodes {
		if node.HasModel(model) {
			residency = append(residency, node.NodeID)
		}
	}
	return residency
}
