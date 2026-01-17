# Directory Structure

## Overview

ORION follows a **modular, event-driven architecture** organized around:
- **Core modules** (Python cognitive components): Brain, Guardian, Memory, Commander, Approval
- **Message bus** (Redis Streams): Event transport with JSON Schema contracts
- **Configuration & Policy**: YAML-based safety policies
- **Testing & Planning**: Centralized test suite and phase-based planning

## Top-Level Directories

```
/home/orion/orion/
├── README.md                          # Project overview
├── CLAUDE.md                          # Development contract (CRITICAL)
├── LICENSE
├── requirements-test.txt              # Test dependencies
├── pytest.ini                         # Pytest configuration
├── .env.example                       # Environment variables template
│
├── core/                              # Python cognitive modules
├── bus/                               # Event bus infrastructure
├── tests/                             # Centralized test suite
├── docs/                              # Architecture and phase documentation
├── config/                            # Configuration files
├── policies/                          # Safety and execution policies (YAML)
├── deploy/                            # Deployment configurations
├── edge/                              # Edge device implementations
├── watchers/                          # System monitoring and telemetry
└── .planning/                         # Development planning
```

## Core Modules

Python cognitive components for reasoning, decisions, and actions:

```
core/
├── brain/                             # Decision-making engine
│   ├── README.md
│   ├── __init__.py
│   ├── brain.py                       # Main decision logic
│   ├── policy_loader.py               # SAFE/RISKY classification
│   ├── cooldown_tracker.py            # Rate limiting
│   └── circuit_breaker.py             # Failure protection
│
├── guardian/                          # Event correlation
│   ├── README.md
│   ├── __init__.py
│   └── guardian.py                    # Incident detection
│
├── memory/                            # Long-term audit trail
│   ├── README.md
│   ├── __init__.py
│   └── memory.py                      # JSONL storage
│
├── commander/                         # Action execution
│   ├── README.md
│   ├── __init__.py
│   └── commander.py                   # Execute with rollback
│
├── approval/                          # Human authority (Phase 4)
│   ├── README.md
│   ├── __init__.py
│   ├── admin_identity.py              # Single-admin verification
│   └── approval_coordinator.py        # Approval tracking
│
├── approval-telegram/                 # Telegram bot (planned)
│   └── README.md
│
└── api/                               # HTTP API (planned)
    └── README.md
```

**Module Structure Pattern**:
Each module contains:
- `README.md` - Purpose, inputs/outputs, invariants, failure modes
- `__init__.py` - Public exports
- `<module>.py` - Main implementation
- `*_helper.py` - Supporting classes

## Contracts

JSON Schema definitions for all inter-module communication:

```
bus/
├── README.md                          # Bus design philosophy
├── contracts/                         # JSON Schema contracts
│   ├── README.md                      # Contract versioning rules
│   ├── event.schema.json              # Raw observations
│   ├── incident.schema.json           # Correlated events
│   ├── decision.schema.json           # Brain reasoning
│   ├── action.schema.json             # Commands to execute
│   ├── outcome.schema.json            # Execution results
│   ├── approval_request.schema.json   # Approval requests (N3)
│   ├── approval_decision.schema.json  # Approval responses
│   └── approval.schema.json           # [DEPRECATED]
│
└── python/                            # Python Redis Streams client
    ├── requirements.txt
    └── orion_bus/
        ├── __init__.py
        ├── bus.py                     # Publish/subscribe
        └── validator.py               # Contract validation
```

## Tests

Centralized test suite (238 tests, all passing):

```
tests/
├── conftest.py                        # Global fixtures
├── test_contracts.py                  # Contract validation (24 tests)
├── test_bus.py                        # Event bus (18 tests)
├── test_brain.py                      # Brain N0 mode (24 tests)
├── test_brain_n2.py                   # Brain N2 mode (27 tests)
├── test_commander.py                  # Action execution (15 tests)
├── test_guardian.py                   # Event correlation (20 tests)
├── test_memory.py                     # Memory operations (15 tests)
├── test_approval_phase4.py            # Approval system (28 tests)
├── test_circuit_breaker.py            # Failure protection (27 tests)
├── test_cooldown_tracker.py           # Rate limiting (16 tests)
├── test_policy_loader.py              # Policy loading (8 tests)
└── test_policies.py                   # Policy consistency (17 tests)
```

## Policies

YAML files defining execution rules and safety classifications:

```
policies/
├── actions_safe.yaml                  # SAFE action allowlist
├── actions_risky.yaml                 # RISKY action flaglist
├── approvals.yaml                     # Approval flow rules
└── cooldowns.yaml                     # Rate limiting windows
```

## Documentation

Architecture and phase documentation:

```
docs/
├── CONTEXT.md                         # Authoritative entry point (START HERE)
├── ROADMAP.md                         # 7-phase plan
├── ARCHITECTURE.md                    # System architecture
├── NODES.md                           # Node roles
├── BUS_AND_CONTRACTS.md               # Event bus design
├── SECURITY.md                        # Security model
├── RUNBOOKS.md                        # Operational procedures
├── PHASE_*.md                         # Phase specifications
└── [other design docs]
```

## Configuration

```
config/
└── admin.yaml.example                 # Admin identity template
```

## Deployment

Deployment configurations for all nodes:

```
deploy/
├── ARCHITECTURE.md                    # Deployment architecture
├── INTEGRATION.md                     # Integration guide
├── STORAGE.md                         # Storage configuration
│
├── core/                              # Core node deployment
│   ├── .env.example
│   └── docker-compose.yml
│
├── hub/                               # Hub node (docker-compose)
│   ├── .env.example
│   └── docker-compose.yml             # Media stack
│
├── edge/                              # Edge node deployment
│   └── README.md
│
└── lab/                               # Local lab setup
    ├── docker-compose.monitoring.yml  # Prometheus, Grafana
    ├── docker-compose.orion.yml       # Full ORION stack
    └── prometheus.yml
```

## Edge Devices

```
edge/
└── freenove_hexapod/                  # Hexapod robot integration
    ├── safety.md                      # Safety constraints
    └── README.md
```

## Watchers

System monitoring and telemetry:

```
watchers/
├── system_resources.py                # CPU, memory, disk watchers
└── requirements.txt
```

## Planning

Development planning and progress tracking:

```
.planning/
├── STATE.md                           # Current project state
├── ROADMAP.md                         # Execution roadmap
│
├── codebase/                          # Codebase documentation
│   ├── STACK.md                       # Technology stack
│   ├── ARCHITECTURE.md                # System architecture
│   ├── STRUCTURE.md                   # Directory structure (this file)
│   ├── CONVENTIONS.md                 # Coding conventions
│   ├── TESTING.md                     # Testing approach
│   ├── INTEGRATIONS.md                # External integrations
│   └── CONCERNS.md                    # Technical debt and risks
│
└── phases/                            # Phase-specific planning
    ├── 00-foundation-governance/
    ├── 00.1-hardware-clean-reset/
    ├── 01-core-observability/
    ├── 02-hub-infrastructure/
    ├── 03-controlled-autonomy/
    ├── 04-telegram-approvals/
    ├── 05-ai-council/
    ├── 06-edge-integration/
    └── 07-compute-expansion/
```

## Notes

### Critical Files for Development

**Essential Reading (In Order)**:
1. `CLAUDE.md` - Development contract (READ THIS FIRST)
2. `docs/CONTEXT.md` - Authoritative project entry point
3. `README.md` - Project overview
4. `bus/contracts/README.md` - Contract philosophy
5. Module READMEs - Each module's purpose and invariants

### Key Directories

- `bus/contracts/` - JSON schemas (version 1.0)
- `policies/` - SAFE/RISKY classifications
- `tests/` - All test code (238 tests)
- `core/` - ORION modules
- `deploy/` - Infrastructure configurations
- `.planning/` - Development planning and progress

### Directory Conventions

1. **Module-scoped branches**: `module/<module-name>`
2. **One module per branch**: No cross-module changes
3. **README in every module**: Documenting purpose, inputs, outputs, invariants
4. **Contracts before code**: JSON Schemas are source of truth
5. **Tests alongside code**: Features implemented with tests

---

This structure reflects ORION's core principle: **contracts define truth, modules are citizens, violations are rejected**.
