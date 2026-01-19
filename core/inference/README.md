# ORION Inference Subsystem

Distributed LLM inference with sticky model routing and health-aware load balancing.

## Purpose

The inference subsystem provides:
- Distributed Ollama cluster across Raspberry Pi nodes
- Sticky routing to avoid cold-start model loading latency
- Health-aware backoff when nodes are overloaded or overheating
- Async request/response via Redis Streams

## Inputs/Outputs

**Inputs:**
- `orion:inference:requests` Redis stream - Inference requests from Brain

**Outputs:**
- `orion:inference:requests:{nodeID}` Redis streams - Per-worker request queues
- `{callback}` Redis stream - Inference responses (if callback specified)
- `orion:inference:health` Redis hash - Worker health data

## Invariants

1. **Sticky routing**: Requests prefer nodes with model already loaded
2. **Health thresholds**: Nodes excluded if temp > 75°C or RAM > 90%
3. **Stale detection**: Nodes excluded if last health update > 15 seconds ago
4. **Graceful shutdown**: Workers remove themselves from health registry on stop

## Failure Modes

### No available nodes
- Router returns `ErrNoAvailableNodes`
- Request remains in pending state for retry
- Brain should implement backoff/retry logic

### Ollama unreachable
- Worker continues health publishing (models list empty)
- Inference requests fail with error in response
- Worker remains available for routing (may recover)

### Redis disconnected
- Worker fails to start
- Health publishing fails (logged, not fatal)
- Graceful degradation when connection restored

## What This Module Does NOT Do

- **Model management**: Ollama handles model pulling/caching
- **Request queuing**: Brain handles retry/backoff logic
- **Response processing**: Consumers handle response stream reading
- **Authentication**: Assumes trusted internal network

## Directory Structure

```
core/inference/
├── contracts/           # Shared data types
│   └── inference.go
├── worker/              # Worker agent implementation
│   ├── agent.go         # WorkerAgent with Ollama integration
│   ├── metrics.go       # HealthCollector using gopsutil
│   └── registry.go      # HealthRegistry for Redis publishing
├── router/              # Router implementation
│   ├── health.go        # HealthReader for node health
│   ├── sticky.go        # StickyRouter algorithm
│   └── router.go        # InferenceRouter with Redis Streams
├── cmd/
│   ├── orion-inference-worker/
│   └── orion-inference-router/
├── integration_test.go  # Integration tests (requires Redis)
├── Makefile
└── README.md
```

## Quick Start

```bash
# Build
make build

# Run router (on Pi 16GB)
./bin/orion-inference-router --redis-addr localhost:6379

# Run worker (on each Pi)
./bin/orion-inference-worker --node-id pi-8g --redis-addr localhost:6379
```

See `docs/INFERENCE-SETUP.md` for full deployment guide.
