# Roadmap: ORION

## Overview

ORION is a modular, autonomous homelab system built around safety, SRE principles, and controlled intelligence. The journey progresses from clean infrastructure through observability, then carefully controlled autonomy with human oversight, ultimately building toward multi-model reasoning and edge integration.

## Domain Expertise

None (specialized homelab/SRE system with custom architecture)

## Phases

**Phase Numbering:**
- Integer phases (0, 1, 2, 3...): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 0: Clean Reset** - Start from known, reproducible state
- [ ] **Phase 1: orion-hub** - Media, access, storage, dashboards
- [ ] **Phase 2: orion-core** - Observability, event bus, memory (no actions)
- [ ] **Phase 3: Controlled Autonomy** - Safe actions with allowlists, cooldowns, rollback
- [ ] **Phase 4: Telegram Approvals** - Risky action approval workflow
- [ ] **Phase 5: AI Council** - Multi-model reasoning with confidence scoring
- [ ] **Phase 6: Edge Integration** - Autonomous edge devices (robot)
- [ ] **Phase 7: Compute Expansion** - Optional worker node capacity

## Phase Details

### Phase 0: Clean Reset
**Goal**: Start from a known, reproducible state
**Depends on**: Nothing (first phase)
**Research**: Unlikely (infrastructure setup following documented procedures)
**Plans**: TBD

**Objectives:**
- Backup what matters
- Flash clean OS
- No partial reuse
- Document everything

Plans:
- [ ] 00-01: TBD during planning

### Phase 1: orion-hub
**Goal**: Media, access, storage, and dashboards infrastructure
**Depends on**: Phase 0
**Research**: Likely (deployment configuration, Tailscale setup)
**Research topics**: Docker deployment patterns, Tailscale ACL configuration, storage solutions, dashboard frameworks
**Plans**: TBD

**Success criteria**: Reboot safe + remote access + no open ports

**Objectives:**
- Media server functionality
- Remote access via Tailscale
- Storage management
- Monitoring dashboards

Plans:
- [ ] 01-01: TBD during planning

### Phase 2: orion-core
**Goal**: Observability, event bus, and memory (no autonomous actions)
**Depends on**: Phase 1
**Research**: Likely (Redis Streams, event-driven architecture, observability stack)
**Research topics**: Redis Streams patterns, JSON Schema validation, event sourcing, Prometheus/Loki setup
**Plans**: TBD

**Success criteria**: Incidents detected, nothing auto-fixed

**Objectives:**
- Event bus (Redis Streams)
- Observability infrastructure
- Memory and audit trails
- Guardian correlation logic
- Brain reasoning (observe-only mode, N0)

Plans:
- [ ] 02-01: TBD during planning

### Phase 3: Controlled Autonomy
**Goal**: Execute safe actions with allowlists, cooldowns, and rollback
**Depends on**: Phase 2
**Research**: Unlikely (internal patterns established in Phase 2)
**Plans**: TBD

**Success criteria**: Safe actions execute autonomously, risky actions blocked

**Objectives:**
- Define SAFE action allowlist (policies/actions_safe.yaml)
- Implement cooldown logic
- Build rollback capabilities
- Commander orchestration (N2 autonomy level)

Plans:
- [ ] 03-01: TBD during planning

### Phase 4: Telegram Approvals
**Goal**: Human approval workflow for risky actions
**Depends on**: Phase 3
**Research**: Likely (Telegram Bot API integration)
**Research topics**: Telegram Bot API, webhook patterns, approval state machines, time-limited approvals
**Plans**: TBD

**Success criteria**: Risky actions require approval, expiration enforced, deny = inaction

**Objectives:**
- Telegram bot integration (orion-approval-telegram)
- Approval state machine
- Time-limited approval expiration
- Full audit trail
- N3 autonomy level activation

Plans:
- [ ] 04-01: TBD during planning

### Phase 5: AI Council
**Goal**: Multi-model reasoning with confidence scoring
**Depends on**: Phase 4
**Research**: Likely (AI model integration, multi-agent patterns)
**Research topics**: Claude API, OpenAI API, multi-model orchestration, confidence scoring algorithms, embeddings for memory
**Plans**: TBD

**Success criteria**: Multiple models critique decisions, no single model trusted

**Objectives:**
- Multi-model integration
- Confidence scoring system
- Model disagreement resolution
- Memory embeddings (orion-memory)

Plans:
- [ ] 05-01: TBD during planning

### Phase 6: Edge Integration
**Goal**: Autonomous edge devices with offline safety
**Depends on**: Phase 4
**Research**: Likely (MQTT, edge autonomy, robot hardware)
**Research topics**: MQTT patterns, Freenove Hexapod SDK, offline safety protocols, ARM cross-compilation
**Plans**: TBD

**Success criteria**: Edge nodes operate autonomously offline, safe mode on network loss

**Objectives:**
- MQTT telemetry and commands
- orion-edge-agent (Go, Raspberry Pi)
- Freenove Hexapod integration
- Offline safety enforcement
- Default to stop behavior

Plans:
- [ ] 06-01: TBD during planning

### Phase 7: Compute Expansion
**Goal**: Optional worker node capacity
**Depends on**: Phase 6
**Research**: Unlikely (established patterns from earlier phases)
**Plans**: TBD

**Success criteria**: Worker nodes scale compute capacity without architectural changes

**Objectives:**
- orion-worker node role definition
- Horizontal scaling patterns
- Workload distribution

Plans:
- [ ] 07-01: TBD during planning

## Progress

**Execution Order:**
Phases execute in numeric order: 0 → 1 → 2 → 3 → 4 → 5 → 6 → 7

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 0. Clean Reset | 0/? | Not started | - |
| 1. orion-hub | 0/? | Not started | - |
| 2. orion-core | 0/? | Not started | - |
| 3. Controlled Autonomy | 0/? | Not started | - |
| 4. Telegram Approvals | 0/? | Not started | - |
| 5. AI Council | 0/? | Not started | - |
| 6. Edge Integration | 0/? | Not started | - |
| 7. Compute Expansion | 0/? | Not started | - |
