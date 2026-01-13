# Codebase Structure

**Analysis Date:** 2026-01-13

## Directory Layout

```
orion/
├── bus/                    # Event bus & contracts (centralized)
│   ├── README.md          # Bus philosophy
│   └── contracts/         # JSON Schema event definitions (empty)
├── core/                   # orion-core node (reasoning & coordination)
│   ├── api/               # HTTP inspection API (Python, stub)
│   ├── brain/             # Decision logic & policies (Python, stub)
│   ├── commander/         # Action orchestration (Python, stub)
│   └── guardian/          # Correlation & incident detection (Python, stub)
├── deploy/                 # Infrastructure deployment
│   ├── core/              # orion-core docker-compose (empty)
│   ├── hub/               # orion-hub docker-compose (empty)
│   ├── edge/              # Edge node deployment (stub)
│   └── lab/               # Lab setup documentation (stub)
├── edge/                   # Edge devices & autonomous agents
│   └── freenove_hexapod/  # Hexapod robot (Raspberry Pi 4)
├── policies/               # Safety & execution policies (YAML, all empty)
├── docs/                   # Architecture & roadmap
├── CLAUDE.md              # Development contract & governance rules
├── README.md              # Project overview
└── LICENSE                # License
```

## Directory Purposes

**bus/**
- Purpose: Event bus client and contract definitions
- Contains: Redis Streams client code (Go, not implemented), JSON Schema contracts
- Key files:
  - `bus/README.md` (21 lines) - Bus philosophy
  - `bus/contracts/event.schema.json` (0 bytes, empty)
  - `bus/contracts/incident.schema.json` (0 bytes, empty)
  - `bus/contracts/decision.schema.json` (0 bytes, empty)
  - `bus/contracts/action.schema.json` (0 bytes, empty)
  - `bus/contracts/outcome.schema.json` (0 bytes, empty)
- Subdirectories: `contracts/` for JSON Schema definitions

**core/**
- Purpose: orion-core node modules (reasoning, coordination, decision-making)
- Contains: Python modules for AI reasoning and SRE logic
- Key files:
  - `core/api/README.md` (0 bytes, empty stub)
  - `core/brain/README.md` (0 bytes, empty stub)
  - `core/commander/README.md` (0 bytes, empty stub)
  - `core/guardian/README.md` (0 bytes, empty stub)
- Subdirectories: Module-specific directories awaiting implementation

**deploy/**
- Purpose: Infrastructure deployment configurations
- Contains: Docker compose files, environment templates
- Key files:
  - `deploy/core/docker-compose.yml` (0 bytes, empty)
  - `deploy/core/.env.example` (0 bytes, empty)
  - `deploy/hub/docker-compose.yml` (0 bytes, empty)
  - `deploy/hub/.env.example` (0 bytes, empty)
  - `deploy/edge/README.md` (0 bytes, empty)
  - `deploy/lab/README.md` (0 bytes, empty)
- Subdirectories: Node-specific deployment configs

**edge/**
- Purpose: Edge devices and autonomous agents
- Contains: Hardware-specific code and safety rules
- Key files:
  - `edge/freenove_hexapod/README.md` (6 lines) - Edge node overview
  - `edge/freenove_hexapod/safety.md` (6 lines) - Offline safety rules
- Subdirectories: Device-specific implementations

**policies/**
- Purpose: Safety and execution policies (YAML configuration)
- Contains: Action classifications, approval rules, temporal constraints
- Key files:
  - `policies/actions_safe.yaml` (0 bytes, empty)
  - `policies/actions_risky.yaml` (0 bytes, empty)
  - `policies/approvals.yaml` (0 bytes, empty)
  - `policies/cooldowns.yaml` (0 bytes, empty)
- Subdirectories: None (flat structure)

**docs/**
- Purpose: Architecture documentation and roadmap
- Contains: Design documents, phase plans, security model
- Key files:
  - `docs/ARCHITECTURE.md` (55 lines) - Core design invariants
  - `docs/BUS_AND_CONTRACTS.md` (19 lines) - Event types and rules
  - `docs/EDGE_DEVICES.md` (29 lines) - Edge node design
  - `docs/EXTENSIBILITY.md` (10 lines) - Adding new nodes
  - `docs/NODES.md` (29 lines) - Node types and roles
  - `docs/ORION_BRAIN.md` (13 lines) - Decision pipeline
  - `docs/PHASE_0_RESET.md` (8 lines) - Clean reset objective
  - `docs/PHASE_1_ORION_HUB.md` (9 lines) - Hub deployment
  - `docs/PHASE_2_ORION_CORE.md` (10 lines) - Core observability
  - `docs/PHASE_3_AUTONOMY.md` (6 lines) - Safe autonomy
  - `docs/PHASE_4_APPROVALS.md` (5 lines) - Telegram approvals
  - `docs/PHASE_5_AI_COUNCIL.md` (5 lines) - Multi-model reasoning
  - `docs/ROADMAP.md` (101 lines) - 7-phase execution plan
  - `docs/RUNBOOKS.md` (8 lines) - Operational procedures (stub)
  - `docs/SECURITY.md` (21 lines) - Zero-trust and secrets policy
- Subdirectories: None (flat structure)

## Key File Locations

**Entry Points:**
- Not yet implemented - No source code files exist
- Planned: `orion-brain`, `orion-guardian`, `orion-commander`, `orion-api`, `orion-approval-telegram`, `orion-bus`, `orion-edge-agent`

**Configuration:**
- `CLAUDE.md` (279 lines) - Development contract and governance rules
- `README.md` (25 lines) - Project overview
- `.gitignore` (22 lines) - Environment files, Python cache, logs, OS/editor files
- `deploy/*/.env.example` (0 bytes, empty templates)

**Core Logic:**
- Not yet implemented
- Expected locations based on CLAUDE.md:
  - `core/brain/` - Decision logic and policy evaluation
  - `core/guardian/` - Correlation and temporal logic
  - `core/commander/` - Action orchestration and rollback
  - `core/api/` - HTTP inspection endpoints

**Testing:**
- Not yet implemented - No test files detected
- Test-alongside doctrine specified in `CLAUDE.md` lines 108-141

**Documentation:**
- `CLAUDE.md` (279 lines) - Primary governance document
- `docs/` directory - Architecture and phase documentation
- Module READMEs - Mostly empty stubs awaiting implementation

## Naming Conventions

**Files:**
- Markdown: kebab-case (e.g., `orion-brain.md`, `bus-and-contracts.md`)
- Important project files: UPPERCASE (e.g., `CLAUDE.md`, `README.md`, `LICENSE`)
- Schemas: `{event_type}.schema.json` (e.g., `event.schema.json`, `incident.schema.json`)
- Policies: `{category}_{classification}.yaml` (e.g., `actions_safe.yaml`, `actions_risky.yaml`)

**Directories:**
- kebab-case for all directories
- Module names match directory structure: `orion-brain` → `core/brain/`
- Node types: `core/`, `hub/`, `edge/`, `deploy/`

**Special Patterns:**
- Phase documents: `PHASE_{N}_{NAME}.md` (e.g., `PHASE_2_ORION_CORE.md`)
- Environment templates: `.env.example` (gitignored actual `.env` files)
- Contract schemas: `*.schema.json` in `bus/contracts/`

## Where to Add New Code

**New Module:**
- Primary code: `core/{module-name}/` for Python, `bus/` or edge location for Go
- Contracts: `bus/contracts/{event-type}.schema.json`
- Tests: Co-located with source (test-alongside per `CLAUDE.md`)
- Documentation: Module README with purpose, inputs/outputs, invariants, failure modes

**New Policy:**
- Implementation: `policies/{category}_{type}.yaml`
- Documentation: Reference in `docs/ARCHITECTURE.md` or module README

**New Edge Device:**
- Implementation: `edge/{device-name}/`
- Safety rules: `edge/{device-name}/safety.md`
- Deployment: `deploy/edge/{device-name}/`

**New Contract:**
- Schema: `bus/contracts/{event-type}.schema.json`
- Version: Schemas are versioned, backward compatibility required
- Documentation: Update `docs/BUS_AND_CONTRACTS.md`

**Utilities:**
- Not yet established - No utility pattern exists
- Future: Module-specific utilities within module directories (no shared utilities)

## Special Directories

**.planning/codebase/**
- Purpose: Codebase mapping documents (this analysis)
- Source: Generated by `/gsd:map-codebase` command
- Committed: Yes (reference documentation)

**policies/**
- Purpose: Safety policy configuration (YAML)
- Source: Manual definitions of SAFE/RISKY classifications
- Committed: Yes (explicit rules required per `CLAUDE.md`)

**bus/contracts/**
- Purpose: Event schema definitions (JSON Schema)
- Source: Versioned contract definitions
- Committed: Yes (single source of truth for module boundaries)

---

*Structure analysis: 2026-01-13*
*Update when directory structure changes*
