# ORION — Architecture

ORION follows a **logic-first architecture**.

Infrastructure executes.
ORION observes, reasons, decides, and acts.

## Design Invariants

- Event-driven system
- Role-based nodes
- Zero-trust networking
- Explicit decision pipeline
- Human approval for risky actions

## Communication

- Redis Streams: core event bus
- MQTT: edge telemetry and commands
- Prometheus: metrics
- Loki: logs

No direct synchronous dependency is allowed for critical flows.
