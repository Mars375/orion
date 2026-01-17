# Roadmap: ORION

## Overview

ORION is a modular, autonomous homelab system built around safety, SRE principles, and controlled intelligence. The journey progresses from logical foundation (contracts, policies, tests) through core observability, then infrastructure deployment, followed by carefully controlled autonomy with human oversight, ultimately building toward multi-model reasoning and edge integration.

## Current State

**Phases 0-4 Complete** ✅ (2026-01-15 through 2026-01-16)

ORION supports **N0 (observe-only), N2 (SAFE actions), and N3 (approved RISKY actions) modes**. The system observes infrastructure, correlates events into incidents, and makes decisions. In N2 mode, SAFE actions execute automatically. In N3 mode, SAFE actions auto-execute and RISKY actions require explicit ADMIN approval.

**Implementation Status:**
- ✅ Event bus (Redis Streams) with contract validation
- ✅ Guardian (event correlation into incidents)
- ✅ Brain (N0/N2/N3 decision logic with policy enforcement)
- ✅ Commander (action execution with rollback)
- ✅ Approval system (coordinator, admin identity validation)
- ✅ Safety mechanisms (cooldowns, circuit breaker, approval expiration)
- ✅ 238 tests passing (all green)

**Next Phase:** Phase 4.1 (Bus Migration to Go) - TECHNICAL PIVOT before AI Council

## Domain Expertise

None (specialized homelab/SRE system with custom architecture)

## Phases

**Phase Numbering:**
- Integer phases (0, 1, 2, 3...): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 0: Foundation & Governance** - Contracts, policies, tests (no runtime) ✅
- [ ] **Phase 0.1: Hardware Clean Reset (INSERTED)** - Backup, flash OS, hardware prep
- [x] **Phase 1: Core Observability** - Event bus, guardian, memory (N0 only) ✅
- [x] **Phase 2: Hub Infrastructure** - Media, access, storage, dashboards ✅
- [x] **Phase 3: Controlled Autonomy** - Safe actions with allowlists, cooldowns, rollback ✅
- [x] **Phase 4: Telegram Approvals** - Risky action approval workflow ✅
- [ ] **Phase 4.1: TECHNICAL PIVOT: Bus Migration to Go (INSERTED)** - Rewrite orion-bus in Go
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

### Phase 4.1: TECHNICAL PIVOT: Bus Migration to Go (INSERTED)
**Goal**: Rewrite orion-bus from Python to Go for performance, memory efficiency, and type safety
**Depends on**: Phase 4 (Telegram Approvals)
**Research**: Likely (Go Redis Streams client, concurrent dispatch patterns, contract validation in Go)
**Plans**: 0 plans

**Success criteria**: orion-bus rewritten in Go, all existing functionality preserved, performance and memory targets met, zero downtime migration path established

**Objectives:**
- Rewrite orion-bus in Go with Redis Streams client
- Implement strict contract validation using Go's type system
- Achieve sub-millisecond event routing performance
- Minimize memory footprint (critical for Pi 5 with 16GB + LLMs)
- High-concurrency event dispatching (prepare for AI Council load)
- Maintain backward compatibility with existing Python consumers
- Zero-downtime migration strategy
- Full test coverage (unit + integration)

**Rationale (TECHNICAL PIVOT):**
- **Performance**: Phase 5 AI Council will saturate CPU with multiple LLM inference calls. Bus must handle high-concurrency event dispatching without blocking.
- **Memory**: Pi 5 has 16GB total. LLMs are memory-intensive. Minimizing bus footprint leaves more room for AI models.
- **Safety**: Go's type system provides compile-time guarantees for contract validation. Python's runtime validation is good, but Go eliminates entire classes of contract violations at build time.
- **Reliability**: Long-running process with strict SLA. Go's memory model and GC behavior are more predictable under sustained load.

**Migration approach:**
- Implement Go bus alongside existing Python bus
- Run both in parallel with event duplication
- Validate Go bus behavior matches Python bus
- Gradual cutover with rollback capability
- Remove Python bus only after burn-in period

Plans:
- [ ] TBD (run /gsd:plan-phase 4.1 to break down)

### Phase 5: AI Council
**Goal**: Multi-model reasoning with confidence scoring
**Depends on**: Phase 4.1 (Bus Migration to Go)
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
Phases execute in numeric order: 0 → 0.1 → 1 → 2 → 3 → 4 → 4.1 → 5 → 6 → 7

Note: Phase 0.1 (Hardware Clean Reset) can be skipped during early development if hardware is unavailable. Phase 1 can proceed with local development environment.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 0. Foundation & Governance | Complete | ✅ Complete | 2026-01-15 |
| 0.1. Hardware Clean Reset | 0/? | Deferred | - |
| 1. Core Observability | Complete | ✅ Complete | 2026-01-15 |
| 2. Hub Infrastructure | Complete | ✅ Complete | 2026-01-15 |
| 3. Controlled Autonomy | Complete | ✅ Complete | 2026-01-15 |
| 4. Telegram Approvals | Complete | ✅ Complete | 2026-01-16 |
| 4.1. Bus Migration to Go | 0/? | Not started | - |
| 5. AI Council | 0/? | Not started | - |
| 6. Edge Integration | 0/? | Not started | - |
| 7. Compute Expansion | 0/? | Not started | - |
