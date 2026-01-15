# ORION Project Context

**Last Updated**: 2026-01-15
**Current Phase**: Phase 3 Complete
**Autonomy Level**: N2 Capable (N0 or N2 configurable)

## What ORION Is

ORION is a safety-first autonomous homelab system that observes infrastructure, reasons about incidents, and makes decisions—but only when safe. Core principle: observation precedes action, and inaction is always preferred to risky action. SAFE actions can execute automatically in N2 mode, but RISKY actions never execute without human approval (N3+). Built around SRE principles with explicit safety invariants that cannot be bypassed.

## Current Project State

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 0 | COMPLETE | Foundation & Governance (contracts, policies, tests) |
| Phase 1 | COMPLETE | Core Observability (event bus, guardian, brain N0, memory) |
| Phase 2 | COMPLETE | Hub Infrastructure (media stack, monitoring, deployment docs) |
| Phase 3 | COMPLETE | Controlled Autonomy (N2 mode, Commander, SAFE actions only) |
| Phase 4+ | NOT STARTED | Future phases require explicit approval |

**Autonomy Levels Supported**:
- **N0** (Observe Only): All decisions are NO_ACTION, no execution
- **N2** (SAFE Actions): SAFE actions execute automatically, RISKY actions blocked

**Actions That Can Execute** (N2 mode only):
- `acknowledge_incident`: Updates incident state (idempotent, SAFE)

**Actions That NEVER Execute**:
- Any RISKY action (restart_service, scale_service, stop_edge_device, etc.)
- Unknown or unclassified actions

## What Exists Today

### Core Components (Functional)

- **Event Bus** (`orion-bus`): Redis Streams with JSON Schema validation
- **Guardian** (`orion-guardian`): Event correlation into incidents
- **Brain** (`orion-brain`): Decision logic (N0 + N2 modes, policy enforcement)
  - PolicyLoader: SAFE/RISKY classification from YAML
  - CooldownTracker: Rate limiting and repeated execution prevention
  - CircuitBreaker: Failure protection (stops after repeated failures)
- **Commander** (`orion-commander`): Action execution engine (SAFE actions only)
  - Executes `acknowledge_incident` action
  - Automatic rollback on failure
  - Emits outcome contracts for audit trail
- **Memory** (`orion-memory`): Append-only JSONL audit trail
- **Watcher** (`orion-watcher`): System resource observer

### Infrastructure (Documented, Not Deployed)

- **Media Stack**: Jellyfin, Radarr, Sonarr, Prowlarr, qBittorrent
- **Monitoring**: Prometheus (metrics only), Grafana (dashboards only)
- **Deployment**: docker-compose configurations for manual deployment
- **Storage**: HDD-only persistent storage layout

### Tests

- 207 tests passing (Phases 0-3)
- Contract validation (24 tests)
- Policy consistency (17 tests)
- Bus functionality (16 tests)
- Memory immutability (14 tests)
- Guardian correlation (21 tests)
- Brain N0 enforcement (17 tests)
- Brain N2 behavior (21 tests)
- Policy loader (7 tests)
- Cooldown tracker (16 tests)
- Circuit breaker (19 tests)
- Commander execution (26 tests)

### Documentation

- `CLAUDE.md`: Development contract and safety rules
- `deploy/ARCHITECTURE.md`: Hardware topology and deployment
- `deploy/STORAGE.md`: HDD storage layout and rules
- `deploy/INTEGRATION.md`: ORION observe-only integration
- `docs/ROADMAP.md`: 8-phase plan
- `bus/contracts/`: JSON schemas for all message types

## What Is Explicitly NOT Allowed

### RISKY Actions ORION Cannot Execute (Even in N2 Mode)

- Restart services or containers (`restart_service` - RISKY)
- Stop or kill processes (`stop_edge_device` - RISKY)
- Scale services (`scale_service` - RISKY)
- Delete files or clean up storage
- Modify configurations
- Execute shell commands
- Control Docker directly
- Control hardware (fans, power, network)
- Deploy or update services

### SAFE Actions ORION Can Execute (N2 Mode Only)

- Acknowledge incidents (`acknowledge_incident` - SAFE, idempotent)
- Send notifications (`send_notification` - SAFE, rate-limited) [not yet implemented]
- Run diagnostics (`run_diagnostic` - SAFE, rate-limited) [not yet implemented]

### Behaviors That Are Forbidden

- Autonomous remediation at any autonomy level
- Alert-triggered actions (no alertmanager)
- Implicit defaults for safety-critical behavior
- Cross-module coupling (all communication via contracts)
- SD card writes (except OS)
- Bypassing contract validation
- Weakening safety policies

### Code Changes That Require ADR

- Changing autonomy level
- Modifying SAFE/RISKY classifications
- Changing execution rules
- Introducing coupling between modules
- Modifying decision logic

## Hardware & Storage Rules

### Hardware Topology

- **Node 1** (orion-hub): Raspberry Pi 5 (16GB RAM)
  - Media stack (Jellyfin, *arr, qBittorrent)
  - Shared storage

- **Node 2** (orion-core): Raspberry Pi 5 (8GB RAM)
  - ORION services (Guardian, Brain, Memory)
  - Monitoring (Prometheus, Grafana)

- **Storage**: 3TB HDD mounted at `/mnt/orion-data`
  - ALL persistent data
  - ALL logs
  - ALL configurations

### Storage Rules (CRITICAL)

1. **No SD card writes**: SD cards are OS-only
2. **HDD for everything**: All volumes map to `/mnt/orion-data`
3. **Fail closed**: If HDD unavailable, services MUST NOT start
4. **tmpfs for transients**: Use tmpfs for truly temporary data only
5. **Bounded growth**: Logs and data must have retention policies

### Directory Structure

```
/mnt/orion-data/
├── media/          # Media library
├── downloads/      # Download staging
├── config/         # Service configurations
├── logs/           # Centralized logs
└── orion/          # ORION persistent state
    ├── memory/     # Audit trail (JSONL)
    └── redis/      # Event bus persistence
```

## Rules for Contributors (CRITICAL)

### Before Making Any Change

1. **Read `CLAUDE.md`** (non-negotiable)
2. **Understand contracts** (`bus/contracts/`)
3. **Check current policies** (`policies/`)
4. **Verify autonomy level** (must be N0)
5. **Run all tests** (must pass)

### Changes That Require Explicit Approval

- Promoting autonomy level (N0 → N1 → N2 → N3)
- Changing SAFE vs RISKY classification
- Modifying execution, rollback, or approval logic
- Changing time-based reasoning (cooldowns, thresholds)
- Any change that could alter behavior under failure

### Development Workflow

- One module per branch: `module/<module-name>`
- Small commits during development
- Squash merge to main with conventional commit format
- All work through module branches (no direct commits to main)
- Tests must pass before merge

### Safety Invariants (Never Violate)

1. **Conservative by default**: When uncertain, choose safer option
2. **Ask first**: Get approval before changing autonomy or safety
3. **Dry-run by default**: New behavior starts in observe-only mode
4. **Test alongside code**: No feature without tests
5. **Contracts are sacred**: Validate all cross-module communication

## Where to Look Next

### Essential Reading (In Order)

1. **`CLAUDE.md`**: Development contract, safety philosophy, coding standards
2. **`docs/ROADMAP.md`**: 8-phase development plan
3. **`deploy/ARCHITECTURE.md`**: Hardware, deployment, service topology
4. **`bus/contracts/README.md`**: Message contracts and validation
5. **`deploy/STORAGE.md`**: Storage layout and HDD rules

### Module Documentation

Each module has a README with:
- Purpose (what it does)
- Inputs/outputs (contracts consumed/emitted)
- Invariants (must always hold)
- Failure modes (what can go wrong)
- Explicit non-responsibilities (what it never does)

### Key Directories

- `bus/contracts/`: JSON schemas (version 1.0)
- `policies/`: SAFE/RISKY action classifications, cooldowns, approvals
- `tests/`: All test code (109 tests)
- `core/`: ORION modules (brain, guardian, memory, etc.)
- `deploy/`: Infrastructure configurations
- `watchers/`: Event producers (system resources, etc.)

### Understanding the Data Flow

```
Watchers → Events → Guardian → Incidents → Brain → Decisions (NO_ACTION)
                                                          ↓
                                                      Memory (audit)
```

All data flows through Redis Streams event bus with contract validation.

## Current Limitations

### By Design

- **N0 only**: No autonomous actions until explicitly enabled
- **No hardware control**: Robot/edge devices not integrated
- **Manual deployment**: Infrastructure configs exist but not deployed
- **No UI**: Command-line and file-based interaction only

### Technical

- **In-memory correlation**: Guardian state not persisted (Phase 1)
- **No metrics polling**: System watcher exists but not integrated with Prometheus
- **No log tailing**: Only system metrics observed
- **Single-node ORION**: Core services not distributed

## Getting Started

### For Developers

```bash
# 1. Read the contract
cat CLAUDE.md

# 2. Set up Python environment
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements-test.txt

# 3. Run tests
pytest tests/ -v

# 4. Understand current state
cat docs/ROADMAP.md
ls -la bus/contracts/
```

### For Operators

```bash
# 1. Review deployment docs
cat deploy/ARCHITECTURE.md
cat deploy/STORAGE.md

# 2. Prepare storage (if deploying)
# See deploy/STORAGE.md for commands

# 3. Deploy (manual, not automated)
# See deploy/ARCHITECTURE.md for steps
```

---

**Remember**: ORION observes. Humans decide. Humans act. Always.
