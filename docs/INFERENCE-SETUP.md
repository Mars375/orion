# ORION Inference Subsystem Setup Guide

This guide covers deployment and configuration of the ORION inference subsystem, which provides distributed LLM inference with sticky model routing and health-aware load balancing.

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │                  Brain                   │
                    │            (Request Source)              │
                    └──────────────────┬──────────────────────┘
                                       │ XADD
                                       ▼
                    ┌─────────────────────────────────────────┐
                    │         orion:inference:requests         │
                    │           (Main Request Stream)          │
                    └──────────────────┬──────────────────────┘
                                       │ XReadGroup
                                       ▼
                    ┌─────────────────────────────────────────┐
                    │          Inference Router                │
                    │    (Sticky Routing + Health-Aware)       │
                    └────────────┬────────────────┬───────────┘
                                 │                │
              XADD               │                │              XADD
         ┌───────────────────────┘                └───────────────────────┐
         ▼                                                                ▼
┌─────────────────────────┐                          ┌─────────────────────────┐
│ orion:inference:        │                          │ orion:inference:        │
│ requests:pi-8g          │                          │ requests:pi-16g         │
└──────────┬──────────────┘                          └──────────┬──────────────┘
           │ XReadGroup                                         │ XReadGroup
           ▼                                                    ▼
┌─────────────────────────┐                          ┌─────────────────────────┐
│   Worker (pi-8g)        │                          │   Worker (pi-16g)       │
│   └── Ollama            │                          │   └── Ollama            │
│   └── gemma3:1b ●       │                          │   └── llama3.2:3b ●     │
└─────────────────────────┘                          └─────────────────────────┘
           │                                                    │
           │ Health Publish (5s)                               │ Health Publish (5s)
           └────────────────────────┬───────────────────────────┘
                                    ▼
                    ┌─────────────────────────────────────────┐
                    │         orion:inference:health           │
                    │           (Health Registry Hash)         │
                    └─────────────────────────────────────────┘
```

## Prerequisites

### Required Software

1. **Redis** (v7.0+) - Event bus and health registry
   ```bash
   # Verify Redis is running
   redis-cli ping
   ```

2. **Ollama** (v0.3+) - LLM runtime on worker nodes
   ```bash
   # Verify Ollama is running
   curl http://localhost:11434/api/tags
   ```

3. **Go** (1.22+) - For building from source

### Hardware Requirements

| Node | RAM | Role | Notes |
|------|-----|------|-------|
| Pi 16GB | 16GB | Router + Worker | Primary validator, runs orchestrator |
| Pi 8GB | 8GB | Worker only | Dedicated to heavy LLM analysis |
| Pi 4GB | 4GB | EXCLUDED | Kinematics only, no LLM tasks |

## Build Instructions

```bash
cd core/inference

# Build both binaries
make build

# Build for ARM64 (Raspberry Pi)
make build-arm

# Run tests
make test-unit
make test-integration
```

Built binaries will be in `bin/`:
- `orion-inference-worker`
- `orion-inference-router`

## Configuration

### Worker Configuration

```bash
orion-inference-worker \
  --node-id pi-8g \
  --redis-addr redis.orion.local:6379 \
  --redis-password your-password \
  --ollama-host http://localhost:11434 \
  --http-port 8081
```

| Flag | Default | Description |
|------|---------|-------------|
| `--node-id` | (required) | Unique node identifier |
| `--redis-addr` | localhost:6379 | Redis server address |
| `--redis-password` | "" | Redis password |
| `--ollama-host` | http://localhost:11434 | Ollama server URL |
| `--http-port` | 8081 | HTTP health endpoint port |
| `--stream-prefix` | orion:inference | Redis stream prefix |

### Router Configuration

```bash
orion-inference-router \
  --redis-addr redis.orion.local:6379 \
  --redis-password your-password \
  --http-port 8080
```

| Flag | Default | Description |
|------|---------|-------------|
| `--redis-addr` | localhost:6379 | Redis server address |
| `--redis-password` | "" | Redis password |
| `--http-port` | 8080 | HTTP endpoints port |
| `--stream-prefix` | orion:inference | Redis stream prefix |

## Deployment

### Systemd Service (Worker)

Create `/etc/systemd/system/orion-inference-worker.service`:

```ini
[Unit]
Description=ORION Inference Worker
After=network.target ollama.service redis.service
Wants=ollama.service

[Service]
Type=simple
User=orion
WorkingDirectory=/opt/orion
ExecStart=/opt/orion/bin/orion-inference-worker \
  --node-id=%H \
  --redis-addr=redis.orion.local:6379 \
  --ollama-host=http://localhost:11434 \
  --http-port=8081
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Systemd Service (Router)

Create `/etc/systemd/system/orion-inference-router.service`:

```ini
[Unit]
Description=ORION Inference Router
After=network.target redis.service

[Service]
Type=simple
User=orion
WorkingDirectory=/opt/orion
ExecStart=/opt/orion/bin/orion-inference-router \
  --redis-addr=redis.orion.local:6379 \
  --http-port=8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable orion-inference-worker
sudo systemctl start orion-inference-worker
```

## Health Endpoints

### Worker Health

```bash
curl http://worker-host:8081/health
```

Response:
```json
{
  "status": "ok",
  "service": "orion-inference-worker",
  "node_id": "pi-8g",
  "version": "0.1.0",
  "ollama": "http://localhost:11434"
}
```

### Router Health

```bash
curl http://router-host:8080/health
```

Response:
```json
{
  "status": "ok",
  "service": "orion-inference-router",
  "version": "0.1.0"
}
```

### Router Stats

```bash
curl http://router-host:8080/stats
```

Response:
```json
{
  "total_routed": 142,
  "sticky_hits": 128,
  "fallbacks": 14,
  "errors": 0
}
```

### Available Nodes

```bash
curl http://router-host:8080/nodes
```

Response:
```json
[
  {
    "node_id": "pi-8g",
    "cpu_percent": 45.2,
    "ram_percent": 62.1,
    "temp_celsius": 55.0,
    "models": ["gemma3:1b"],
    "available": true,
    "last_seen": "2026-01-19T12:00:00Z"
  }
]
```

## Redis Stream Names

| Stream | Purpose |
|--------|---------|
| `orion:inference:requests` | Main incoming request stream (router consumes) |
| `orion:inference:requests:{nodeID}` | Per-worker request streams |
| `orion:inference:health` | Health registry (Redis hash) |
| `orion:inference:health:{nodeID}` | Per-node health with TTL (backup for stale detection) |

## Routing Algorithm

### Sticky Routing

1. Get all healthy nodes (sorted by RAM usage ascending)
2. **First pass**: Find node with model already loaded → return immediately (sticky hit)
3. **Second pass**: Return least-loaded healthy node (fallback)

### Health Thresholds

Nodes are excluded from routing if:
- Temperature > 75°C
- RAM usage > 90%
- Last health update > 15 seconds ago (stale)
- `available` flag is false

## Troubleshooting

### Worker not appearing in health registry

1. Check Redis connectivity:
   ```bash
   redis-cli -h redis.orion.local HGETALL orion:inference:health
   ```

2. Verify worker is publishing health:
   ```bash
   journalctl -u orion-inference-worker -f
   ```

3. Check for stale detection:
   - Health must be published within 15 seconds
   - Default publish interval is 5 seconds

### Requests not being routed

1. Check router is consuming:
   ```bash
   journalctl -u orion-inference-router -f
   ```

2. Verify healthy nodes exist:
   ```bash
   curl http://router:8080/nodes
   ```

3. Check main request stream:
   ```bash
   redis-cli XLEN orion:inference:requests
   ```

### High latency on inference

1. Check if model needs loading (load_duration_ms in response)
2. Verify sticky routing is working (check /stats for sticky_hits)
3. Consider pre-loading models:
   ```bash
   curl http://worker:11434/api/generate -d '{"model":"gemma3:1b","keep_alive":"24h"}'
   ```

### Node marked unavailable

Check health thresholds:
```bash
# View raw health data
redis-cli HGET orion:inference:health pi-8g | jq

# Check temperature
cat /sys/class/thermal/thermal_zone0/temp  # Divide by 1000 for °C

# Check RAM
free -m
```

## Contract Schemas

JSON Schema contracts are defined in `bus/contracts/`:
- `inference.request.schema.json` - Request format
- `inference.response.schema.json` - Response format

Validate requests:
```bash
# Using ajv-cli
npx ajv validate -s bus/contracts/inference.request.schema.json -d request.json
```
