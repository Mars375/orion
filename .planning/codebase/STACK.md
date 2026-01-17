# Technology Stack

## Languages

- **Python 3.12**: Core cognitive modules (Brain, Guardian, Memory, Commander, Approval Coordinator)
- **YAML**: Policy configuration and cooldown settings

## Frameworks & Libraries

### Python Runtime Dependencies
- `redis==5.0.1` - Redis client for event bus communication
- `jsonschema==4.21.1` - JSON Schema validation for contracts
- `pyyaml==6.0.1` - Policy file parsing
- `psutil==5.9.8` - System resource monitoring

### Python Testing Dependencies
- `pytest==8.0.0` - Test framework
- `pytest-cov==4.1.0` - Code coverage reporting
- `pytest-mock==3.12.0` - Mocking utilities
- `fakeredis==2.21.1` - In-memory Redis mock for testing
- `pytest-asyncio==0.21.1` - Async test support

## Infrastructure & Services

- **Redis 7 (Alpine)** - Event bus backbone (Redis Streams)
- **Prometheus (latest)** - Metrics collection (observe-only, no alerting)
- **Grafana (latest)** - Metrics visualization
- **Node Exporter (latest)** - System metrics exposure
- **cAdvisor (latest)** - Container metrics
- **Docker & Docker Compose** - Containerization and orchestration

### Media Stack (Optional)
- **Jellyfin** - Media server
- **Radarr** - Movie management
- **Sonarr** - TV series management
- **Prowlarr** - Indexer aggregator
- **qBittorrent** - Torrent client

## Communication Protocols & Data Formats

- **Redis Streams** - Core event bus (no AMQP, no Kafka)
- **MQTT** - Edge device telemetry (planned, not implemented)
- **HTTP REST** - API exposure (Prometheus, Grafana, ORION services)
- **JSON** - Message format throughout
- **JSON Schema** - Contract definitions and validation

## Configuration & Storage

- **Environment Variables** (.env format) - Runtime configuration
- **JSON Schema Files** - Versioned contracts for inter-module communication
- **YAML Policy Files** - Safety classification and cooldown configuration
- **JSONL Format** - Append-only audit trail (events, incidents, decisions)
- **SQLite/File-based** - Memory store for audit trail

## Development Tools

### Testing Framework
- `pytest` with markers: `unit`, `integration`, `contract`, `policy`, `slow`
- Async test support via `pytest-asyncio`
- Mock Redis via `fakeredis` for isolated testing
- Test discovery patterns: `test_*.py` and `*_test.py`

### Code Quality
- Type hints throughout (Python 3.12 style)
- Dataclass-based contracts
- Comprehensive docstrings on public APIs

### Build & Deployment
- Docker Compose for orchestration
- Volume binding to HDD storage (`/mnt/orion-data`)
- Health checks on all services
- Memory limits per container

## Autonomy Levels

- **N0** - Observe only, no action
- **N2** - Execute SAFE actions with rate limiting and circuit breakers
- **N3** - Execute SAFE actions + REQUEST approval for RISKY actions

## Core Modules & Their Tech Stack

| Module | Language | Key Libraries |
|--------|----------|---------------|
| `orion-brain` | Python 3.12 | redis, jsonschema, pyyaml (policies) |
| `orion-guardian` | Python 3.12 | redis, jsonschema |
| `orion-commander` | Python 3.12 | redis, jsonschema, pyyaml (policies) |
| `orion-memory` | Python 3.12 | jsonl (file-based) |
| `orion-approval` | Python 3.12 | redis, jsonschema, pyyaml |
| `orion-bus` (Python) | Python 3.12 | redis, jsonschema |
| `orion-watcher` | Python 3.12 | psutil, redis |

## System Architecture Notes

1. **Event-Driven**: Redis Streams as single event bus
2. **Contract-First**: Strict JSON Schema validation at message boundaries
3. **Fail-Closed**: Unknown actions treated as RISKY by default
4. **Audit Trail**: All decisions and outcomes logged in JSONL format
5. **Safety Guards**: Cooldown tracking, circuit breaker pattern
6. **Rate Limiting**: Configurable via YAML policies
7. **Approval System**: Time-limited, admin-only approval mechanism

## Deployment Topology

- **orion-core** (docker-compose): Redis only (containerized)
- **ORION services**: Run on host as systemd services or manually
- **Monitoring stack**: Prometheus, Grafana, Node Exporter, cAdvisor
- **Media stack**: Optional services on hub node

## Storage & Persistence

- Redis AOF (Append-Only File) persistence enabled
- HDD-only storage (no SD card writes permitted)
- 30-day retention for Prometheus metrics
- Immutable JSONL audit trail

## Network Architecture

- Localhost-only Redis and Prometheus
- Docker bridge networks: `orion-core`, `orion-monitoring`, `orion-media`
- MQTT for edge device communication (planned)

## Notes

This stack reflects ORION's philosophy: **conservative by default, explicit communication, safety before capability, and complete auditability**.
