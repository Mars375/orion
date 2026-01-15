# orion-guardian

**Language:** Python

## Purpose
Correlates events into incidents using temporal logic and pattern detection. Guardian observes event streams and identifies meaningful situations that require attention, without making decisions about how to respond.

## Inputs (Contracts Consumed)
- `event.schema.json` — Raw events from any ORION module or edge device

## Outputs (Contracts Emitted)
- `incident.schema.json` — Correlated incidents representing meaningful situations

## Invariants (MUST Always Hold)
1. **Event-driven only**: Guardian only reacts to events, never polls or queries external systems
2. **Correlation windows are explicit**: Every incident MUST include correlation_window showing time range
3. **Event IDs preserved**: Incidents MUST reference all contributing event_ids for traceability
4. **No decisions**: Guardian detects and correlates, but never decides actions (that's orion-brain's job)
5. **State tracking**: Guardian tracks incident state (open/acknowledged/resolved) but does not execute state changes
6. **Idempotency**: Processing the same event multiple times MUST NOT create duplicate incidents
7. **Time-bounded**: Correlation windows have maximum duration to prevent unbounded memory growth

## Failure Modes
- **Event schema violation**: If event fails validation, guardian logs error and discards event
- **Correlation state loss**: If Redis fails, guardian continues processing new events but loses correlation history
- **Time drift**: If system clock skews significantly, correlation windows may be inaccurate
- **Event flood**: If event rate exceeds capacity, guardian may drop events (logs dropped count)

## Explicit Non-Responsibilities (What Guardian NEVER Does)
- **Never decides actions**: Action decisions are orion-brain's responsibility
- **Never executes actions**: Action execution is orion-commander's responsibility
- **Never emits events**: Guardian consumes events but does not emit them (only incidents)
- **Never queries external systems**: Guardian is event-driven only, no active polling
- **Never stores long-term history**: Historical analysis is orion-memory's responsibility
- **Never exposes HTTP APIs**: API exposure is orion-api's responsibility
