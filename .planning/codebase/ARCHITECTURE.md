# Architecture

**Analysis Date:** 2026-01-13

## Pattern Overview

**Overall:** Event-Driven Microservices with Safety-First Design

**Key Characteristics:**
- Contract-first: JSON Schema defines all module boundaries
- Language-agnostic: Modules freely choose Python or Go; communicate via events
- Zero-trust: Tailscale mandatory, no open ports, explicit ACLs
- Safety by design: Observation precedes action, inaction preferred to risky action
- Auditability: Every event immutable, decisions logged, outcomes tracked

## Layers

**Presentation Layer:**
- Purpose: Human interaction and approval workflows
- Contains: Telegram approvals (`orion-approval-telegram`), API endpoints (`orion-api`), dashboards (planned for hub)
- Depends on: Event bus for decision notifications
- Used by: Human operators, external monitoring systems

**Reasoning Layer:**
- Purpose: Decision-making, policy evaluation, correlation
- Contains: `orion-brain` (reasoning, policies, decisions), `orion-guardian` (correlation, temporal logic)
- Location: `core/brain/`, `core/guardian/` (empty READMEs, not yet implemented)
- Depends on: Event bus for observations and incidents
- Used by: Commander for action execution

**Orchestration Layer:**
- Purpose: Action execution, rollback, state transitions
- Contains: `orion-commander` (initial implementation)
- Location: `core/commander/` (empty README, not yet implemented)
- Depends on: Decision events from brain, approval confirmations from Telegram
- Used by: Edge agents, system agents

**Event Bus Layer:**
- Purpose: Immutable event log, inter-module communication
- Contains: `orion-bus` (Redis Streams client, Go)
- Location: `bus/` directory
- Depends on: Redis Streams infrastructure
- Used by: All modules for communication
- Event types: Observation, Incident, Decision, Action, Outcome per `docs/BUS_AND_CONTRACTS.md`

**Persistence & Memory Layer:**
- Purpose: Long-term state, post-mortems, audit trails
- Contains: `orion-memory` (post-mortems, embeddings, Python)
- Depends on: Database (not yet specified), event bus
- Used by: Brain for historical context, audit systems

**Edge & System Layer:**
- Purpose: Physical device control, offline safety, system monitoring
- Contains: `orion-edge-agent` (robot, MQTT, offline safety, Go), system agents (metrics, watchers)
- Location: `edge/freenove_hexapod/` (Raspberry Pi 4 + Freenove Hexapod)
- Depends on: MQTT broker, offline safety protocols
- Used by: Commander for action execution on hardware

## Data Flow

**Observation → Decision → Action Lifecycle:**

1. **Observation (Input)** - Monitoring agents, edge devices, external systems emit observations
2. **Normalization** - Transform to canonical event form, emit to Redis Streams
3. **Correlation** - Guardian analyzes patterns, temporal windows per `docs/ORION_BRAIN.md`
4. **Incident Detection** - Emit incident event if threshold exceeded
5. **Policy Evaluation** - Brain applies SAFE/RISKY classification from `policies/*.yaml` (empty)
6. **Decision** - Brain reasons over policies, emit decision event
7. **Approval Check** - If RISKY, Telegram approval required with expiration per `docs/SECURITY.md`
8. **Action** - Commander executes approved action, emit action event
9. **Outcome** - System reports result, emit outcome event
10. **Audit** - Memory stores decision rationale and outcome

**Key Rule:** All cross-module communication occurs through Redis Streams events. No shared memory, no direct function calls between modules per `CLAUDE.md`.

**State Management:**
- Event-sourced: All state changes tracked as immutable events
- No shared state: Each module maintains local state derived from events
- Edge autonomy: Edge nodes operate offline, sync when connected

## Key Abstractions

**Event Contracts (JSON Schema versioned):**
- Location: `bus/contracts/*.schema.json` (empty files, schemas not yet defined)
- Types: `event.schema.json`, `incident.schema.json`, `decision.schema.json`, `action.schema.json`, `outcome.schema.json`
- Purpose: Language-agnostic module boundaries, runtime validation mandatory
- Pattern: All producers validate before publish, all consumers validate on receipt

**Safety Classification:**
- **SAFE**: Allowlisted actions in `policies/actions_safe.yaml` (empty), reversal guaranteed, no approval required
- **RISKY**: Restricted actions in `policies/actions_risky.yaml` (empty), requires Telegram approval, time-limited, audit mandatory
- Purpose: Prevent autonomous system from unsafe behavior

**Autonomy Levels (Progression model per CLAUDE.md):**
- **N0** (Observe only): Default for all new behavior, no actions executed
- **N1** (Suggest): System suggests actions, human executes manually
- **N2** (Execute SAFE): Autonomous execution of allowlisted safe actions only
- **N3** (Full with approval): Execute SAFE + RISKY (with Telegram approval flow)
- Promotion: Requires explicit approval and evidence, never automatic

**Policies (YAML configuration in `policies/` - all empty):**
- `actions_safe.yaml`: Allowlist of autonomous actions
- `actions_risky.yaml`: Actions requiring approval
- `approvals.yaml`: Approval requirements and expiration
- `cooldowns.yaml`: Rate limiting and circuit breaker logic

## Entry Points

**Not yet implemented**, but planned:

**Python Entry Points:**
- `orion-brain` - Main reasoning loop subscribing to incident stream
- `orion-guardian` - Correlation engine reading event stream
- `orion-commander` - Action executor subscribing to decision stream
- `orion-api` - HTTP server exposing decision logic (read-only inspection)
- `orion-approval-telegram` - Telegram bot listening for approvals

**Go Entry Points:**
- `orion-bus` - Redis Streams client library (not standalone service)
- `orion-edge-agent` - MQTT + offline safety loop on Raspberry Pi

**Deployment Entry Points:**
- `deploy/core/docker-compose.yml` - Starts orion-core services (empty)
- `deploy/hub/docker-compose.yml` - Starts orion-hub services (empty)

## Error Handling

**Strategy:** Conservative by default - inaction preferred to risky action per `CLAUDE.md`

**Patterns:**
- Explicit rules: No automation without explicit rules, evidence, and auditability
- Dry-run by default: New behavior starts as N0 (observe-only), progresses: Observe → Dry-run → Restricted → Full
- Rollback required: Safe actions must be reversible
- Approval expiration: Risky action approvals time-limited

**Edge Safety:**
- Default to stop: Hardware enters safe state on uncertainty per `edge/freenove_hexapod/safety.md`
- Network loss → safe mode: Edge nodes autonomous offline
- No destructive commands without approval: Hardware safety enforced

## Cross-Cutting Concerns

**Logging:**
- Loki for log aggregation per `docs/ARCHITECTURE.md` (not configured)
- Audit trail mandatory for all decisions and actions

**Validation:**
- JSON Schema validation at all module boundaries per `CLAUDE.md`
- Type hints mandatory everywhere (Python and Go)
- Runtime validation for external inputs (Redis Streams, MQTT, Telegram)

**Authentication:**
- Tailscale zero-trust networking per `docs/SECURITY.md`
- No open inbound ports, explicit ACLs, identity per node

---

*Architecture analysis: 2026-01-13*
*Update when major patterns change*
