# System Architecture

## Overview

ORION is a **safety-first autonomous homelab system** built around three core principles:
1. **Observation precedes action** - Understand system state before any decision
2. **Inaction is preferred to risky action** - Conservative by default
3. **Controlled intelligence** - Explicit rules, evidence-based decisions, full auditability

The system flows through a strict pipeline: observation → correlation → decision → action, with comprehensive audit trails at every stage.

## Modules

### orion-bus (Event Infrastructure)
**Language**: Go (planned), Python (current)
**Purpose**: Redis Streams client with JSON Schema validation
**Key Invariant**: Contract validation - every message MUST validate against schema
**Failure Mode**: Schema violation = immediate rejection (fail-fast)

### orion-guardian (Event Correlation)
**Language**: Python
**Purpose**: Real-time event correlation into meaningful incidents
**Consumes**: `event.schema.json`
**Emits**: `incident.schema.json`
**Key Mechanisms**:
- Fingerprint-based deduplication
- Time-bounded correlation windows (60s default)
- Preserves all contributing event IDs

### orion-brain (Decision Engine)
**Language**: Python
**Purpose**: Policy-driven decision making
**Consumes**: `incident.schema.json`
**Emits**: `decision.schema.json`, `approval_request.schema.json` (N3)
**Autonomy Levels**:
- **N0**: Every decision MUST be `NO_ACTION`
- **N2**: SAFE actions → `EXECUTE_SAFE_ACTION`, RISKY → `NO_ACTION`
- **N3**: SAFE auto-execute, RISKY → `REQUEST_APPROVAL`
**Safety Components**:
- PolicyLoader: SAFE/RISKY classification from YAML
- CooldownTracker: Rate limiting
- CircuitBreaker: Failure protection

### orion-commander (Action Execution)
**Language**: Python
**Purpose**: Execute actions with rollback and audit
**Consumes**: `decision.schema.json`
**Emits**: `outcome.schema.json`
**Validation Flow**:
1. Filter for `EXECUTE_SAFE_ACTION` decisions
2. Validate action type against SAFE policy
3. For RISKY: check approval validity and expiration
4. Execute or reject
5. Emit outcome

### orion-approval (Human Authority - Phase 4)
**Language**: Python
**Purpose**: Single-admin approval system for N3 mode
**Components**:
- AdminIdentity: Single-admin identity verification
- ApprovalCoordinator: Tracks pending approvals, handles expiration
**Consumes**: `approval_request.schema.json`
**Emits**: `approval_decision.schema.json`
**Critical Invariants**:
- Silence is NEVER permission
- Expiration always enforced
- Only ADMIN can approve
- Timeout = escalation, never execution

### orion-memory (Audit Trail)
**Language**: Python
**Purpose**: Append-only JSONL audit trail
**Storage**: `/mnt/orion-data/orion/memory/`
**Invariants**:
- Append-only (never deletes/modifies)
- Immutable history
- No real-time decision-making

### orion-api (Inspection Interface)
**Language**: Python
**Purpose**: HTTP endpoints for human inspection
**Type**: Read-only queries
**Invariants**:
- No write operations
- No side effects
- Authentication required

## Communication Patterns

### Event Bus (Redis Streams)
**Philosophy**: "Contracts are law. Modules are citizens. Violations are rejected."

**Stream Organization**:
- `event` → raw observations
- `incident` → correlated events
- `decision` → decisions about incidents
- `approval_request` → approval requests (N3)
- `approval_decision` → approval responses
- `action` → commands to execute
- `outcome` → execution results

### Contracts (JSON Schema)
All messages MUST include:
- `version` (const "1.0")
- `<entity>_id` (UUID format)
- `timestamp` (ISO 8601 date-time)
- `source` (module name)

**Message Types**:
1. **Event**: Raw observations
2. **Incident**: Correlated situations (from guardian)
3. **Decision**: Reasoning output (from brain)
4. **Approval Request**: Human approval needed (from brain, N3)
5. **Approval Decision**: Human response (from approval system)
6. **Action**: Command to execute
7. **Outcome**: Execution result (from commander)

## Data Flow

### Complete Pipeline Flow

```
System Events
    ↓
orion-watcher
    ↓
Redis Streams (event bus)
    ↓
orion-guardian (correlate events)
    ↓ [emits incident]
Redis Streams
    ↓
orion-brain (decide response)
    ├─→ N0: NO_ACTION only
    ├─→ N2: SAFE actions execute, RISKY blocked
    └─→ N3: SAFE auto-execute, RISKY → REQUEST_APPROVAL
    ↓
Redis Streams
    ├─→ decision
    └─→ approval_request (if RISKY in N3)
    ↓
orion-approval (validate expiration, admin identity)
    ↓ [emits approval_decision]
orion-commander (validate approval, execute, emit outcome)
    ↓
Redis Streams (outcome)
    ↓
orion-memory (audit trail)
```

### Key Invariants at Boundaries
1. **Bus Entry**: Schema validation (reject invalid)
2. **Guardian Output**: All incidents include correlation_window and event_ids
3. **Brain Output**: All decisions include 10+ char reasoning
4. **Brain to Approval**: RISKY actions only, includes expires_at
5. **Approval Expiration**: Coordinator and Commander both validate
6. **Commander Output**: All outcomes include execution_time_ms
7. **Memory Storage**: Append-only, never modified

## Autonomy Levels

### N0 Mode (Observe Only) - Current Default
- Brain: Every decision MUST be `NO_ACTION`
- Commander: Inactive
- Approval: Not applicable
- Use Case: Safe observation without risk

### N2 Mode (Safe Actions)
- Brain: SAFE → `EXECUTE_SAFE_ACTION`, RISKY → `NO_ACTION`
- Commander: Executes SAFE actions only
- Safety: Complete - only idempotent, reversible actions
- Implemented SAFE Actions:
  - `acknowledge_incident`
  - `send_notification` (planned)
  - `run_diagnostic` (planned)

### N3 Mode (Approved RISKY Actions) - Enabled
- Brain: SAFE auto-execute, RISKY → `REQUEST_APPROVAL`
- Approval System: Single-admin identity, time-limited approvals
- Commander: SAFE automatic, RISKY only with valid approval
- RISKY Actions (require approval):
  - `restart_service`
  - `scale_service`
  - `stop_edge_device`
  - `restart_edge_device`

## Safety Mechanisms

### 1. Policy Enforcement
- **SAFE Actions**: Explicitly allowlisted in `policies/actions_safe.yaml`
- **RISKY Actions**: Explicitly classified in `policies/actions_risky.yaml`
- **Unknown Actions**: Treated as RISKY (fail-closed)

### 2. Cooldown Tracker
- Per-action cooldowns prevent rapid repeated execution
- Configurable per action type
- Scope: Per resource (service_name, device_id)
- Enforcement: Brain checks before proposing action

### 3. Circuit Breaker
- Activation: After 3 failures within 300s window
- Behavior: Stops execution, returns NO_ACTION
- Open Duration: 600s before retry
- Override: ADMIN can FORCE with override flag (N3)

### 4. Approval Expiration (N3)
- Default: 300s (5 minutes) per action type
- Maximum: 3600s (1 hour)
- Enforcement: Coordinator + Commander both validate
- Behavior: Never executes expired approval

### 5. Admin Identity Validation (N3)
- Single ADMIN in `config/admin.yaml`
- Channels: Telegram (chat_id) and/or CLI (username/UID)
- Unknown identity: Rejected immediately

### 6. Silent Rejection
- "Silence is NEVER permission"
- Timeout: Escalate (ERROR log), never execute
- Result: Inaction, not execution

### 7. Contract Validation
- Bus validates every message on publish
- Enforcement: additionalProperties: false
- UUIDs, timestamps, required fields strictly enforced

## Hardware & Deployment

### Node Topology
```
orion-hub (Pi 5, 16GB)          orion-core (Pi 5, 8GB)
├── Jellyfin (media)            ├── Redis (event bus)
├── Radarr (movies)             ├── Guardian
├── Sonarr (TV)                 ├── Brain
├── Prowlarr (indexing)         ├── Memory
├── qBittorrent (torrents)      ├── Approval
└── Shared volumes              ├── Prometheus
                                └── Grafana

        ↓
    3TB HDD @ /mnt/orion-data
```

### Storage Rules (Critical)
- **OS Only**: SD cards store OS + code only
- **HDD Everything**: ALL persistent data on `/mnt/orion-data`
- **Fail Closed**: If HDD unavailable, services MUST NOT start

## Testing Architecture

**Total: 238 tests passing**
- Contract Validation: 24 tests
- Policy Consistency: 17 tests
- Bus Functionality: 16 tests
- Memory Immutability: 14 tests
- Guardian Correlation: 21 tests
- Brain N0 Enforcement: 17 tests
- Brain N2 Behavior: 21 tests
- Brain N3 Approvals: 4 tests
- Policy Loader: 7 tests
- Cooldown Tracker: 16 tests
- Circuit Breaker: 19 tests
- Commander Execution: 26 tests
- Approval System: 31 tests

## Architecture Principles

1. **Conservative by Default**: Uncertain → NO_ACTION
2. **Contracts Over Coupling**: All communication via JSON Schema
3. **Audit Trail First**: Every decision, approval, action in memory
4. **Fail-Closed Safety**: Timeout = escalation, never execution
5. **Human Authority (N3)**: Only explicit ADMIN approval permits RISKY actions
6. **Rate Limiting & Protection**: Cooldowns + circuit breaker

## What ORION Does NOT Do

### Actions Never Executed
- Delete files or storage
- Modify configurations
- Execute shell commands
- Control Docker directly
- Control hardware (fans, power, network)
- Deploy or update services
- Any action without explicit classification

### Behaviors Never Executed
- Autonomous remediation (without approval)
- Alert-triggered actions
- Implicit defaults for safety-critical behavior
- Cross-module coupling
- SD card writes (except OS)
- Schema validation bypass

## Key Metrics

- **Autonomy Levels Supported**: 3 (N0, N2, N3)
- **Modules**: 8 core + infrastructure
- **Contracts**: 9 schema types
- **SAFE Actions**: 3 implemented
- **RISKY Actions**: 4 classified
- **Tests**: 238 (all passing)
- **Nodes**: 2 Raspberry Pi 5s
- **Storage**: 3TB HDD centralized

## Current Limitations (By Design)

1. **N0 Default**: No autonomous actions until explicitly enabled
2. **No Hardware Control**: Edge devices not integrated (Phase 6)
3. **In-Memory Correlation**: Guardian state not persisted
4. **Manual Deployment**: Infrastructure configs exist but not auto-deployed
5. **Single-Node ORION**: Core services not distributed

---

This architecture achieves ORION's core mission: **Safe autonomy through strict rules, explicit policies, comprehensive audit, and human authority over high-risk decisions.**
