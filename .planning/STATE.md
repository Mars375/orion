# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-01-19)

**Core value:** Safety-first autonomous homelab - observation precedes action, inaction preferred to risky action, no automation without explicit rules
**Current focus:** v1.0 Core Platform shipped - ready for production deployment

## Current Position

Phase: 7 of 7 (Compute Expansion)
Plan: Complete
Status: v1.0 Milestone Shipped
Last activity: 2026-01-19 — v1.0 Core Platform milestone complete

Progress: ██████████ 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 15 (Phases 4.1-7)
- Average duration: ~6 hours/plan
- Total execution time: ~90 hours (6 days)

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 4.1 Bus Migration | 4 | ~24h | ~6h |
| 5 AI Council | 4 | ~24h | ~6h |
| 6 Edge Integration | 4 | ~24h | ~6h |
| 7 Compute Expansion | 3 | ~18h | ~6h |

**Recent Trend:**
- Last 5 plans: Consistent ~6h/plan
- Trend: Stable velocity

## Accumulated Context

### Decisions

Decisions are logged in CLAUDE.md and PROJECT.md.
Recent decisions affecting current work:

- Go for event bus (performance, memory, type safety)
- santhosh-tekuri/jsonschema for validation (kaptinlin required Go 1.25)
- Gemma-2 2B for local SLM (lower RAM than Phi-3)
- Claude 3.5 Sonnet for external validation (superior reasoning)
- Sticky routing for inference (prefer nodes with model loaded)
- 5-second default watchdog timeout for edge devices
- Health thresholds: temp > 75°C, RAM > 90%
- Fail-closed throughout

### Roadmap Evolution

Phase insertions and modifications:

- **Phase 4.1 inserted after Phase 4** (2026-01-16): TECHNICAL PIVOT: Bus Migration to Go
  - **Reason**: URGENT - Must rewrite orion-bus from Python to Go before Phase 5 AI Council
  - **Performance**: AI Council will saturate CPU with LLM inference. Bus needs high-concurrency event dispatching.
  - **Memory**: Pi 5 has 16GB. LLMs are memory-intensive. Go minimizes bus footprint.
  - **Safety**: Go's type system provides compile-time contract validation guarantees.
  - **Impact**: Phase 5 AI Council now depends on completion of Phase 4.1
  - **Outcome**: Completed successfully 2026-01-17

### Critical Gaps (resolved)

All critical gaps from initial codebase mapping resolved:
- Policy files populated (policies/*.yaml)
- Contract schemas defined (bus/contracts/*.json)
- Module READMEs complete
- Dependency specifications present (requirements.txt, go.mod)
- Test infrastructure established (274+ tests)

### Deferred Issues

- Code generation for Go structs (requires Go 1.23+, have 1.22)
- Actual servo control for edge robot (requires hardware)
- Inverse kinematics implementation (future phase)
- Phase 0.1 Hardware Clean Reset (can happen during deployment)

### Pending Todos

None - milestone complete.

### Blockers/Concerns

**For production deployment:**
- Ollama must be installed on Pi nodes with Gemma-2 2B pulled
- API keys required for external validation (ANTHROPIC_API_KEY, OPENAI_API_KEY)
- Redis must be accessible from all nodes
- MQTT broker required for edge devices

## Session Continuity

Last session: 2026-01-19
Stopped at: v1.0 milestone complete
Resume file: None

## Milestone History

- **v1.0 Core Platform** (2026-01-19): Phases 0-7, 15 plans, 6 days
  - Go Event Bus, AI Council, Edge Integration, Distributed Inference
  - See .planning/MILESTONES.md for details
