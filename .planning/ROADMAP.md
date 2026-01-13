# Roadmap: ORION

## Overview

ORION is a modular, autonomous homelab system built around safety, SRE principles, and controlled intelligence. The journey progresses from logical foundation (contracts, policies, tests) through core observability, then infrastructure deployment, followed by carefully controlled autonomy with human oversight, ultimately building toward multi-model reasoning and edge integration.

## Domain Expertise

None (specialized homelab/SRE system with custom architecture)

## Phases

**Phase Numbering:**
- Integer phases (0, 1, 2, 3...): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 0: Foundation & Governance** - Contracts, policies, tests (no runtime)
- [ ] **Phase 0.1: Hardware Clean Reset (INSERTED)** - Backup, flash OS, hardware prep
- [ ] **Phase 1: Core Observability** - Event bus, guardian, memory (N0 only)
- [ ] **Phase 2: Hub Infrastructure** - Media, access, storage, dashboards
- [ ] **Phase 3: Controlled Autonomy** - Safe actions with allowlists, cooldowns, rollback
- [ ] **Phase 4: Telegram Approvals** - Risky action approval workflow
- [ ] **Phase 5: AI Council** - Multi-model reasoning with confidence scoring
- [ ] **Phase 6: Edge Integration** - Autonomous edge devices (robot)
- [ ] **Phase 7: Compute Expansion** - Optional worker node capacity

## Phase Details

### Phase 0: Foundation & Governance
**Goal**: Establish contracts, policies, and test infrastructure (zero runtime)
**Depends on**: Nothing (first phase)
**Research**: Unlikely (internal documentation and schema definition)
**Plans**: TBD

**Success criteria**: All contracts defined, all policies populated, test infrastructure ready, module READMEs complete

**Objectives:**
- Define all contract schemas (bus/contracts/*.json)
- Populate policy files (policies/*.yaml: SAFE, RISKY, approvals, cooldowns)
- Write all module READMEs (purpose, inputs/outputs, invariants, failure modes)
- Set up test infrastructure (pytest, Go testing, contract validation fixtures)
- Document environment variables (.env.example files)
- Zero services running, zero hardware dependency

Plans:
- [ ] 00-01: TBD during planning

### Phase 0.1: Hardware Clean Reset (INSERTED)
**Goal**: Start hardware from known, reproducible state
**Depends on**: Phase 0 (logical foundation must exist first)
**Research**: Unlikely (infrastructure setup following documented procedures)
**Plans**: TBD

**Success criteria**: Clean OS installed, documented baseline, ready for Phase 1 deployment

**Objectives:**
- Backup what matters (existing data, configurations)
- Flash clean OS on all nodes (orion-core, orion-hub)
- No partial reuse of old configurations
- Document hardware baseline (versions, network config, storage layout)
- Verify Tailscale connectivity

**Execution constraint**: Cannot execute until hardware physically available

Plans:
- [ ] 00.1-01: TBD during planning

### Phase 1: Core Observability
**Goal**: Event bus, correlation, and memory (N0 observe-only, no autonomous actions)
**Depends on**: Phase 0.1 (hardware ready) or Phase 0 (can develop without hardware)
**Research**: Likely (Redis Streams, event-driven architecture, observability stack)
**Research topics**: Redis Streams patterns, JSON Schema validation runtime, event sourcing, Prometheus/Loki setup, correlation windows
**Plans**: TBD

**Success criteria**: Events flowing, incidents detected, nothing auto-fixed, full audit trail

**Objectives:**
- Deploy orion-bus (Go, Redis Streams client with contract validation)
- Implement orion-guardian (Python, correlation, temporal logic, incident detection)
- Implement orion-memory (Python, audit trails, post-mortems, embeddings stub)
- Implement orion-brain (Python, reasoning, N0 observe-only mode)
- Deploy Redis Streams, Prometheus, Loki
- All modules validate contracts at runtime
- Zero autonomous actions (N0 level enforced)

Plans:
- [ ] 01-01: TBD during planning

### Phase 2: Hub Infrastructure
**Goal**: Media, access, storage, and dashboards
**Depends on**: Phase 1 (core observability operational)
**Research**: Likely (deployment configuration, Tailscale setup, media stack)
**Research topics**: Docker deployment patterns, Tailscale ACL configuration, storage solutions, dashboard frameworks, media server options
**Plans**: TBD

**Success criteria**: Reboot safe + remote access + no open ports + media functional

**Objectives:**
- Deploy orion-hub services (media, storage, dashboards)
- Configure Tailscale zero-trust networking (ACLs, magic DNS)
- Set up storage management (ZFS, backups, snapshots)
- Deploy monitoring dashboards (Grafana, alerting)
- Ensure reboot safety (systemd units, health checks)
- Verify no exposed ports (Tailscale-only access)

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
Phases execute in numeric order: 0 → 0.1 → 1 → 2 → 3 → 4 → 5 → 6 → 7

Note: Phase 0.1 (Hardware Clean Reset) can be skipped during early development if hardware is unavailable. Phase 1 can proceed with local development environment.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 0. Foundation & Governance | 0/? | Not started | - |
| 0.1. Hardware Clean Reset | 0/? | Not started | - |
| 1. Core Observability | 0/? | Not started | - |
| 2. Hub Infrastructure | 0/? | Not started | - |
| 3. Controlled Autonomy | 0/? | Not started | - |
| 4. Telegram Approvals | 0/? | Not started | - |
| 5. AI Council | 0/? | Not started | - |
| 6. Edge Integration | 0/? | Not started | - |
| 7. Compute Expansion | 0/? | Not started | - |
