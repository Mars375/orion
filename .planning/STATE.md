# Project State

## Project Reference

See: CLAUDE.md (development contract), docs/ (architecture and phases)

**Core value:** Safety-first autonomous homelab - observation precedes action, inaction preferred to risky action, no automation without explicit rules
**Current focus:** Phase 0 — Foundation & Governance (logical foundation, zero runtime)

## Current Position

Phase: 0 of 7 (Foundation & Governance)
Plan: Not started
Status: Ready to plan
Last activity: 2026-01-13 — Roadmap corrected: logical foundation before hardware

Progress: ░░░░░░░░░░ 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 0
- Average duration: —
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| — | — | — | — |

**Recent Trend:**
- Last 5 plans: —
- Trend: —

## Accumulated Context

### Decisions

Decisions are logged in CLAUDE.md and docs/.
Recent decisions affecting current work:

- Mixed Python/Go stack (Python for cognitive complexity, Go for reliability)
- Event-driven architecture with Redis Streams as event bus
- Contract-first design with JSON Schema validation
- Zero-trust networking via Tailscale
- Three-pillar safety: Conservative by default, Ask first, Dry-run by default
- Autonomy levels: N0 (observe) → N1 (suggest) → N2 (safe actions) → N3 (with approvals)

### Roadmap Evolution

Phase insertions and modifications:

- **Phase 4.1 inserted after Phase 4** (2026-01-16): TECHNICAL PIVOT: Bus Migration to Go
  - **Reason**: URGENT - Must rewrite orion-bus from Python to Go before Phase 5 AI Council
  - **Performance**: AI Council will saturate CPU with LLM inference. Bus needs high-concurrency event dispatching.
  - **Memory**: Pi 5 has 16GB. LLMs are memory-intensive. Go minimizes bus footprint.
  - **Safety**: Go's type system provides compile-time contract validation guarantees.
  - **Impact**: Phase 5 AI Council now depends on completion of Phase 4.1

### Critical Gaps (from codebase mapping)

- All policy files empty (policies/*.yaml) - SAFE/RISKY classifications undefined
- All contract schemas empty (bus/contracts/*.json) - validation rules missing
- All module READMEs empty - no implementation guidance
- No dependency specifications (requirements.txt, go.mod)
- No test infrastructure

### Deferred Issues

None yet.

### Pending Todos

None yet.

### Blockers/Concerns

**Before Phase 0:**
- None (logical foundation, no dependencies)

**Before Phase 0.1:**
- Hardware must be physically available (can be skipped for early dev)

**Before Phase 1:**
- Contract schemas must be defined (Phase 0 deliverable)
- Policy files must be populated (Phase 0 deliverable)
- Test infrastructure must be established (Phase 0 deliverable)
- Module READMEs must be complete (Phase 0 deliverable)

**Before Phase 2:**
- Core observability operational (Phase 1 deliverable)
- Deployment configuration defined (.env.example, docker-compose files)

**Before Phase 3:**
- SAFE action allowlist validated in production (Phase 1-2 data)

## Session Continuity

Last session: 2026-01-13 14:00
Stopped at: Codebase mapping complete, roadmap created
Resume file: None
