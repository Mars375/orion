# Phase 7 Plan 02: Router Service Summary

**Implemented inference router service with sticky model-aware routing and health-based node selection via Redis Streams.**

## Accomplishments

- Created HealthReader for reading node health from Redis with stale detection and availability filtering
- Implemented StickyRouter algorithm that prioritizes nodes with model already loaded (sticky hits)
- Built InferenceRouter that consumes requests from main stream and dispatches to worker-specific streams
- Added routing statistics for observability (total routed, sticky hits, fallbacks, errors)

## Files Created/Modified

- `core/inference/router/health.go` - HealthReader for fetching and filtering node health data
- `core/inference/router/health_test.go` - Unit tests for health filtering, stale detection, sorting
- `core/inference/router/sticky.go` - StickyRouter with model-resident preference and load-based fallback
- `core/inference/router/sticky_test.go` - Unit tests for sticky routing algorithm
- `core/inference/router/router.go` - InferenceRouter with Redis Streams dispatch and consumer group
- `core/inference/router/router_test.go` - Unit tests for routing and stream operations

## Decisions Made

- Health data is always read fresh (no caching) to ensure accurate routing decisions
- Nodes are sorted by RAM percent ascending (least loaded first) before model residency check
- When multiple nodes have the model loaded, the least loaded is selected (best of both worlds)
- Stream naming convention: `orion:inference:requests` for incoming, `orion:inference:requests:{nodeID}` for workers
- Consumer group pattern used for reliable message processing with acknowledgments

## Issues Encountered

- Miniredis doesn't properly handle context cancellation in XReadGroup blocking mode, so consumer loop testing was simplified to test the routing logic directly

## Next Step

Ready for 07-03-PLAN.md (Integration & Testing)
