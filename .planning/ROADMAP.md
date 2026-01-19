# Roadmap: ORION

## Overview

ORION is a modular, autonomous homelab system built around safety, SRE principles, and controlled intelligence. The journey progresses from logical foundation (contracts, policies, tests) through core observability, then infrastructure deployment, followed by carefully controlled autonomy with human oversight, ultimately building toward multi-model reasoning and edge integration.

## Milestones

- **v1.0 Core Platform** — Phases 0-7 (shipped 2026-01-19) [Details](milestones/v1.0-ROADMAP.md)

## Current State

**v1.0 Core Platform SHIPPED** (2026-01-19)

ORION supports **N0 (observe-only), N2 (SAFE actions), and N3 (approved RISKY actions) modes** with multi-model AI Council validation. The system observes infrastructure, correlates events into incidents, makes decisions with confidence-weighted voting, and executes actions with safety controls.

**Implementation Status:**
- Event bus (Go with Redis Streams) with contract validation
- Guardian (event correlation into incidents)
- Brain (N0/N2/N3 decision logic with policy enforcement)
- Commander (action execution with rollback)
- Approval system (coordinator, admin identity validation)
- AI Council (local SLM + external APIs, safety veto)
- Edge agent (Dead Man's Switch, safe state)
- Inference cluster (sticky routing, health-aware load balancing)
- 274+ tests passing

**Next:** Production deployment to Raspberry Pi cluster

## Domain Expertise

None (specialized homelab/SRE system with custom architecture)

## Phases

**Phase Numbering:**
- Integer phases (0, 1, 2, 3...): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>v1.0 Core Platform (Phases 0-7) — SHIPPED 2026-01-19</summary>

- [x] **Phase 0: Foundation & Governance** - Contracts, policies, tests (no runtime) - completed 2026-01-15
- [ ] **Phase 0.1: Hardware Clean Reset (INSERTED)** - Backup, flash OS, hardware prep - Deferred
- [x] **Phase 1: Core Observability** - Event bus, guardian, memory (N0 only) - completed 2026-01-15
- [x] **Phase 2: Hub Infrastructure** - Media, access, storage, dashboards - completed 2026-01-15
- [x] **Phase 3: Controlled Autonomy** - Safe actions with allowlists, cooldowns, rollback - completed 2026-01-15
- [x] **Phase 4: Telegram Approvals** - Risky action approval workflow - completed 2026-01-16
- [x] **Phase 4.1: Bus Migration to Go (INSERTED)** - Rewrite orion-bus in Go - completed 2026-01-17
- [x] **Phase 5: AI Council** - Multi-model reasoning with confidence scoring - completed 2026-01-17
- [x] **Phase 6: Edge Integration** - Autonomous edge devices (robot) - completed 2026-01-18
- [x] **Phase 7: Compute Expansion** - Distributed inference cluster - completed 2026-01-19

</details>

## Progress

**Execution Order:**
Phases execute in numeric order: 0 → 0.1 → 1 → 2 → 3 → 4 → 4.1 → 5 → 6 → 7

Note: Phase 0.1 (Hardware Clean Reset) was deferred during development. Hardware prep can happen during production deployment.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 0. Foundation & Governance | v1.0 | Complete | Complete | 2026-01-15 |
| 0.1. Hardware Clean Reset | v1.0 | 0/? | Deferred | - |
| 1. Core Observability | v1.0 | Complete | Complete | 2026-01-15 |
| 2. Hub Infrastructure | v1.0 | Complete | Complete | 2026-01-15 |
| 3. Controlled Autonomy | v1.0 | Complete | Complete | 2026-01-15 |
| 4. Telegram Approvals | v1.0 | Complete | Complete | 2026-01-16 |
| 4.1. Bus Migration to Go | v1.0 | 4/4 | Complete | 2026-01-17 |
| 5. AI Council | v1.0 | 4/4 | Complete | 2026-01-17 |
| 6. Edge Integration | v1.0 | 4/4 | Complete | 2026-01-18 |
| 7. Compute Expansion | v1.0 | 3/3 | Complete | 2026-01-19 |
