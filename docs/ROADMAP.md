# ORION — Roadmap

## Current State

**Phases 0, 1, 2, 3, and 4 are COMPLETE.**

ORION supports **N0 (observe-only), N2 (SAFE actions), and N3 (approved RISKY actions) modes**. The system observes infrastructure, correlates events into incidents, and makes decisions. In N2 mode, SAFE actions execute automatically. In N3 mode, SAFE actions auto-execute and RISKY actions require explicit ADMIN approval. All approvals are time-limited with strict expiration enforcement. Silence or timeout is NEVER permission.

**Phase 4 Implementation**:
- N3 autonomy level with human authority
- Single-admin identity model (strictly enforced)
- Approval system (coordinator, admin identity validation)
- Approval contracts (request, decision)
- Brain emits approval requests for RISKY actions
- Commander validates approval expiration before execution
- Support for approve/deny/force decisions with overrides
- Timeout escalation (never executes on timeout)
- 238 tests passing (all green)

---

## Phases

Phase 0: Documentation & reset — **DONE**
Phase 1: orion-hub — **DONE**
Phase 2: orion-core — **DONE**
Phase 3: Safe autonomy — **DONE**
Phase 4: Human approvals — **DONE**
Phase 5: AI council
Phase 6: Edge integration
Phase 7: Compute expansion
