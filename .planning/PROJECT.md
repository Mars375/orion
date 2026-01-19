# Project: ORION

## What This Is

A modular, autonomous homelab system built around safety, SRE principles, and controlled intelligence. ORION observes, reasons, decides, and acts - but only when safe. The system supports N0 (observe-only), N2 (SAFE actions), and N3 (approved RISKY actions) autonomy modes with multi-model AI validation.

## Core Value

Safety-first autonomous homelab - observation precedes action, inaction preferred to risky action, no automation without explicit rules, evidence, and auditability.

## Requirements

### Validated

- Go Event Bus with Redis Streams and contract validation - v1.0
- AI Council multi-model validation (local SLM + external APIs) - v1.0
- Confidence-weighted voting with safety veto - v1.0
- Edge device integration with Dead Man's Switch - v1.0
- "Sit & Freeze" safe state for edge devices - v1.0
- Distributed inference cluster with sticky routing - v1.0
- Health-aware load balancing (temp > 75C, RAM > 90% thresholds) - v1.0
- N0/N2/N3 autonomy levels with approval flow - v1.0

### Active

- [ ] Production deployment to Raspberry Pi cluster
- [ ] 7-day burn-in period
- [ ] Actual servo control for edge robot (kinematics)
- [ ] Code generation for Go structs (after Go 1.23 upgrade)

### Out of Scope

- Mobile app - web-first approach
- Cloud hosting - homelab only
- Multi-tenant - single homelab instance
- Real-time video streaming - edge devices report telemetry only

## Current State

**Shipped v1.0 Core Platform** (2026-01-19)

Tech stack:
- Python 3.12: ~8,714 LOC (brain, guardian, memory, commander, council)
- Go 1.22: ~7,456 LOC (bus, edge-agent, inference worker/router)
- Redis Streams: Event bus
- MQTT: Edge telemetry
- JSON Schema: Contract validation (Draft 2020-12)

Test coverage:
- 274+ tests across all modules
- Unit tests with mocks (miniredis, mocked APIs)
- Integration tests with build tags

Deployable binaries:
- `orion-bus` - Go event bus
- `orion-edge` - Edge agent (ARM64 static)
- `orion-inference-worker` - Inference worker
- `orion-inference-router` - Inference router

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go for event bus | Performance, memory, type safety | Good - 78%+ test coverage |
| santhosh-tekuri/jsonschema | kaptinlin required Go 1.25 | Good - full Draft 2020-12 support |
| Gemma-2 2B for local SLM | Lower RAM (3GB vs 5GB) | Good - fits Pi 5 constraints |
| Sticky routing for inference | Avoid model loading latency | Good - prefer resident models |
| Dead Man's Switch timeout 5s | Balance safety vs network jitter | Good - configurable |
| Fail-closed throughout | Safety invariant | Good - no silent failures |

## Constraints

- Raspberry Pi 5 16GB - primary compute
- Raspberry Pi 4 4GB - edge only (excluded from inference)
- No cloud dependencies (except optional external AI APIs)
- Must operate autonomously during network partitions
- Human approval required for RISKY actions

---
*Last updated: 2026-01-19 after v1.0 milestone*
