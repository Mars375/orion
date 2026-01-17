# External Integrations

## Message Bus

### Redis Streams
**Status**: Active, implemented in Phase 1

**Role**: Central event bus for all module-to-module communication

**Connection**:
- Endpoint: `localhost:6379` (localhost only)
- Client library: `redis==5.0.1` (Python)
- Port: `127.0.0.1:6379:6379`

**Configuration** (docker-compose):
- Persistent storage with AOF enabled
- `appendfsync everysec` for durability
- Dangerous commands disabled (FLUSHDB, FLUSHALL, CONFIG)
- Data: `/mnt/orion-data/orion/redis`

**Streams**:
- `event` - Raw observations
- `incident` - Correlated events
- `decision` - Decisions from Brain
- `approval_request` - Approval requests (N3)
- `approval_decision` - Approval responses
- `action` - Action execution proposals
- `outcome` - Execution results

**Usage**: All cross-module communication flows through Redis Streams with contract validation

## Communication

### Telegram Bot Integration
**Status**: Designed (not implemented)

**Planned for N3 mode**: Approval requests via Telegram

**Module**: `orion-approval-telegram`
- Sends approval requests to authorized users
- Collects approval/deny responses
- Enforces timeouts (explicit expiration)
- Emits approval decision contracts
- Strict authorization (configured admin only)

**Flow**:
1. Brain emits approval request
2. Telegram module sends message to admin
3. Admin approves/denies via bot
4. Module emits approval_decision contract
5. Commander validates and executes/rejects

**Note**: Phase 4 supports approval coordinator but telegram integration is future work

## Storage

### Persistent Storage (HDD)
**Status**: Fully designed, documented, not deployed

**Layout** (`/mnt/orion-data`):
```
/mnt/orion-data/
├── media/                  # Media library
│   ├── movies/
│   ├── tv/
│   └── music/
├── downloads/              # Download staging
│   ├── complete/
│   └── incomplete/
├── config/                 # Service configurations
│   ├── jellyfin/
│   ├── radarr/
│   ├── sonarr/
│   ├── prometheus/
│   └── grafana/
├── logs/                   # Centralized logs
│   ├── media/
│   ├── monitoring/
│   └── orion/
└── orion/                  # ORION state
    ├── memory/             # Audit trail (JSONL)
    └── redis/              # Event bus persistence
```

**Critical Rules**:
- **HDD ONLY**: All persistent data on HDD
- **No SD Writes**: SD cards for OS only
- **Fail Closed**: If HDD unavailable, services fail to start

**Permissions**:
- Owner: `orion:orion` (UID:GID 1000:1000)
- Media dirs: `755`
- Config dirs: `700`
- Logs: `755`
- ORION data: `700`

## Monitoring

### Prometheus (Metrics Collection)
**Status**: Documented, observe-only configured

**Endpoint**: `http://localhost:9090`

**Configuration**:
- No alertmanager
- No alert rules
- Metrics collection only
- TSDB retention: 30 days or 10GB
- Storage: `/mnt/orion-data/config/prometheus/data`

**Scraped Metrics**:
- `prometheus` (self-monitoring)
- `node` (node-exporter port 9100)
- `cadvisor` (container metrics port 8081)
- `redis` (event bus port 6379)

**ORION Integration**: Queries Prometheus API, emits events to Redis based on metric values. NO alertmanager feedback loop.

**Pattern**:
```
Poll Prometheus → Detect anomaly → Emit event → Guardian correlates → Brain decides
```

### Grafana (Dashboards)
**Status**: Documented, observe-only

**Endpoint**: `http://localhost:3000`

**Configuration**:
- Admin user: `admin` / password: `orion`
- Analytics disabled
- Sign-up disabled
- No alert notifications
- Storage: `/mnt/orion-data/config/grafana`

**Purpose**: Human observation only, no alerting or automation

### Node Exporter (System Metrics)
**Endpoint**: `http://node-exporter:9100`

**Exposed Metrics**:
- CPU, memory, disk usage
- Network statistics
- Process information
- Filesystem metrics

### cAdvisor (Container Metrics)
**Endpoint**: `http://cadvisor:8080` (published to `8081:8080`)

**Exposed Metrics**:
- Per-container CPU/memory/network/disk
- Container lifecycle events

## Hardware

### Freenove Hexapod Robot
**Status**: Planned for Phase 6

**Location**: `edge/freenove_hexapod/`

**Safety Rules**:
- Default to stop
- Loss of network → safe mode
- No destructive commands without approval

**Planned Integration**:
- Edge device operates offline-first
- MQTT for telemetry
- Edge agent sends telemetry to ORION
- ORION never controls edge directly
- Edge must function without ORION

## APIs and Protocols

### HTTP Endpoints (Inspection Only)
**Status**: Designed for future phases

**Module**: `orion-api`

**Characteristics**:
- Read-only inspection
- Authentication required
- Rate limiting enforced
- NO write operations
- NO side effects

**Example Endpoints** (not implemented):
- `GET /incidents`
- `GET /decisions`
- `GET /actions`
- `GET /health`

### JSON Schema Validation
**Status**: Active in all communication

**Contract Schemas** (`bus/contracts/`):
- `event.schema.json`
- `incident.schema.json`
- `decision.schema.json`
- `approval_request.schema.json`
- `approval_decision.schema.json`
- `action.schema.json`
- `outcome.schema.json`

**Validation**: All messages validated at bus boundary

## Media Stack (Opaque to ORION)

### Services on orion-hub
**Status**: Documented configurations available

**Note**: ORION observes metrics only, does NOT control these services

**Services** (`deploy/hub/docker-compose.yml`):
- **Jellyfin** (8096) - Media server
- **Radarr** (7878) - Movie management
- **Sonarr** (8989) - TV management
- **Prowlarr** (9696) - Indexer manager
- **qBittorrent** (8080) - Torrent client

**Integration**: Prometheus scrapes container metrics, ORION queries Prometheus, emits observation events. No direct control.

## Planned Integrations (Future Phases)

### Phase 5: AI Council
**Vision**: Multiple LLM models evaluate decisions together
- Multi-model reasoning
- Confidence scoring
- Model critique

**Questions**:
- External LLM APIs? (OpenAI, Claude)
- Local model serving (Ollama)?
- Voting/consensus mechanism?

### Phase 6: Edge Integration
**Planned Systems**:
- MQTT telemetry from edge devices
- Edge agent autonomy (offline-first)
- Hexapod robot control
- Sensor integration

**Pattern**:
```
Edge Device (MQTT) → ORION Bus → Guardian → Brain → Commander
```

### Phase 7: Compute Expansion
**Planned Workers**:
- Optional GPU compute nodes
- Worker pool coordination
- Task distribution

## Integration Patterns

### Event-Driven Architecture
**Core Pattern**: All integration flows through Redis Streams

```
External System
    ↓
Module (Observer/Emitter)
    ↓
Event Contract (validated)
    ↓
Redis Streams
    ↓
Consumer Module
    ↓
Decision/Action/Outcome
```

### Contract-First Design
**Principle**: Contracts define the integration contract

**Flow**:
1. Define schema (what data moves)
2. Validate messages against schema
3. Reject invalid messages
4. Producers emit facts, consumers reason

### Observe-Only Pattern (N0 Mode)
**Current deployment model**

```
Prometheus → Query → ORION Watcher → Event → Redis → Guardian → Brain
                                                          ↓
                                                   Decisions: NO_ACTION
```

### Safety-Gate Pattern (N2/N3 Mode)
**For future autonomous execution**

```
Event → Guardian → Brain → Approve (N3) → Commander → Outcome
                     ↓ (safety gates)
                  Policies
                  Cooldowns
                  Circuit breaker
                  Approval (N3)
```

## Notes

### Current State

**Integrated**:
- Redis Streams (event bus)
- Prometheus (metrics observation)
- Docker/containers (cAdvisor metrics)
- File system storage (JSONL audit)
- System resources (psutil)

**NOT Integrated**:
- Telegram (designed, not implemented)
- LLMs (Phase 5)
- MQTT (Phase 6)
- Direct Docker control (disabled)
- External webhooks
- Kubernetes

### Integration Principles (from CLAUDE.md)

1. **No shared memory**: All interaction via explicit contracts
2. **No implicit behavior**: Defaults explicitly documented and safe
3. **Contract validation**: Every message validated at boundaries
4. **Fail closed**: Missing dependencies cause failures
5. **No coupling**: Modules replaceable without affecting others

### Safety Constraints

**ORION NEVER**:
- Restarts services (without approval in N3)
- Kills processes
- Scales containers
- Deletes files
- Modifies configurations
- Executes shell commands
- Controls hardware directly
- Sends notifications autonomously

**ORION ONLY**:
- Observes metrics
- Emits observation events
- Correlates events
- Makes decisions (within autonomy level)
- Executes SAFE actions (N2+)
- Requests approvals (N3)
- Stores audit trail

### Operational Considerations

**Deployment**: Manual via docker-compose
- No auto-deploy scripts
- No orchestration tools
- No Kubernetes

**Observability**: Human-centric
- Prometheus for metrics
- Grafana for dashboards
- JSONL audit trail
- No automated alerting

**Resilience**: Fail-safe design
- ORION down = infrastructure unaffected
- Media stack independent
- Monitoring independent
- Human can always intervene

---

**Document Generated**: Analysis of Phases 0-4 (complete through N3 autonomy)
**Future Phases**: Phase 5-7 concepts outlined but not yet implemented
