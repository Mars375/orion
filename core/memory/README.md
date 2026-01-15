# orion-memory

**Language:** Python

## Purpose
Stores long-term history, post-mortems, and incident analysis using embeddings for semantic search. Memory enables ORION to learn from past incidents without modifying core decision logic.

## Inputs (Contracts Consumed)
- `incident.schema.json` — Incidents for historical storage
- `decision.schema.json` — Decisions for historical analysis
- `action.schema.json` — Actions for post-mortem correlation
- `outcome.schema.json` — Outcomes for learning from failures

## Outputs (Contracts Emitted)
- None (memory is a storage layer, does not emit real-time events)

## Invariants (MUST Always Hold)
1. **Append-only**: Memory never deletes or modifies historical data (immutable)
2. **Semantic search**: Memory provides embedding-based search for similar incidents
3. **No real-time control**: Memory informs future decisions but never triggers immediate actions
4. **Privacy-aware**: Memory MUST NOT store sensitive data (credentials, PII) in embeddings
5. **Retention policy**: Memory enforces retention limits to prevent unbounded growth
6. **No decision-making**: Memory provides context but does not make decisions

## Failure Modes
- **Storage unavailable**: If database is down, memory write operations fail but ORION continues operating
- **Embedding service unavailable**: If embedding API is down, new memories are stored without embeddings (degraded mode)
- **Search failure**: If search fails, ORION falls back to policy-based decisions without historical context
- **Retention cleanup failure**: If cleanup fails, memory growth continues but does not affect ORION operation

## Explicit Non-Responsibilities (What Memory NEVER Does)
- **Never makes decisions**: Decision-making is orion-brain's responsibility
- **Never correlates real-time events**: Real-time correlation is orion-guardian's responsibility
- **Never executes actions**: Action execution is orion-commander's responsibility
- **Never provides real-time event bus**: Memory is for historical analysis, not real-time messaging
- **Never modifies history**: Memory is append-only, historical data is immutable
- **Never auto-tunes policies**: Policy changes require explicit human updates, not automated learning
