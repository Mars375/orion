# ORION — Roadmap

## Current State

**Phases 0, 1, 2, and 3 are COMPLETE.**

ORION supports **N0 (observe-only) and N2 (SAFE actions) modes**. The system observes infrastructure, correlates events into incidents, and makes decisions. In N2 mode, SAFE actions (like `acknowledge_incident`) execute automatically with cooldown and circuit breaker protection. RISKY actions never execute without human approval. Media stack (Jellyfin, Radarr, Sonarr, Prowlarr, qBittorrent) and monitoring (Prometheus, Grafana) are deployed on 2x Raspberry Pi 5 with 3TB HDD storage.

**Phase 3 Implementation**:
- N2 autonomy level with policy enforcement
- Commander action execution engine
- Cooldown tracker (rate limiting)
- Circuit breaker (failure protection)
- 207 tests passing (all green)

---

## Phases

Phase 0: Documentation & reset — **DONE**
Phase 1: orion-hub — **DONE**
Phase 2: orion-core — **DONE**
Phase 3: Safe autonomy — **DONE**
Phase 4: Human approvals
Phase 5: AI council
Phase 6: Edge integration
Phase 7: Compute expansion
