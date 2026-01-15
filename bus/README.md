# orion-bus

**Language:** Go

## Purpose
Provides Redis Streams client library for event bus communication between ORION modules. Bus enforces contract validation and ensures all messages conform to schemas defined in `bus/contracts/`.

## Inputs (Contracts Consumed)
- None (bus is infrastructure, does not consume business events)

## Outputs (Contracts Emitted)
- None (bus transports events but does not emit business events)

## Invariants (MUST Always Hold)
1. **Contract validation**: Bus MUST validate all messages against schemas before publishing
2. **Schema rejection**: Bus MUST reject messages that do not match their declared schema
3. **Version enforcement**: Bus MUST enforce version compatibility rules
4. **No message transformation**: Bus transports messages without modification
5. **Ordering guarantees**: Messages within a stream MUST maintain publication order
6. **No business logic**: Bus is pure infrastructure, contains no decision-making or correlation logic

## Failure Modes
- **Redis unavailable**: If Redis is down, publish operations fail with error (no silent failures)
- **Schema validation failure**: If message violates schema, publish is rejected with validation error
- **Unknown schema**: If message references unknown schema, publish is rejected
- **Network partition**: If network partitions, some consumers may lag behind (eventual consistency)

## Explicit Non-Responsibilities (What Bus NEVER Does)
- **Never makes decisions**: Decision logic is orion-brain's responsibility
- **Never correlates events**: Correlation is orion-guardian's responsibility
- **Never executes actions**: Action execution is orion-commander's responsibility
- **Never transforms messages**: Bus transports messages as-is, no business logic
- **Never stores long-term history**: Bus is a transport layer, not a data store (use orion-memory for history)
- **Never implements retry logic**: Consumers are responsible for their own retry behavior

---

## Philosophy

Contracts define truth.
Producers emit facts.
Consumers reason.
