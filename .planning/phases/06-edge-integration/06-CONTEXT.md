# Phase 6: Edge Integration - Context

**Gathered:** 2026-01-17
**Status:** Ready for research

<vision>
## How This Should Work

The hexapod acts as an extension of ORION — another node in the system. ORION Brain sends high-level goals ("go check the garage") and the robot figures out the movement to accomplish them. Telemetry flows back to ORION for monitoring and decision-making.

**Critical Architecture:**
- **Distributed Hardware**: The hexapod is NOT controlled directly by the Pi 16GB. It has its own Raspberry Pi 4 Model B (4GB).
- **Role Separation**:
  - Pi 16GB (orion-core): Runs Go Bus and AI Council (Gemma-2) — the "Brain"
  - Pi 4 (orion-edge): Handles only kinematics and local sensors — the "Body"
- **Connectivity**: The Pi 4 connects remotely to the Redis Bus on the Pi 16GB via network

When network is lost, the robot freezes in place immediately. This is non-negotiable — the edge node must have local safety logic that doesn't depend on the Brain being reachable.

</vision>

<essential>
## What Must Be Nailed

- **Safety first** — The robot must never do something dangerous, even if commands are wrong or network is flaky
- **Dead Man's Switch** — Pi 4 must independently trigger "Sit & Freeze" safety state if it loses network connection to Brain. This logic runs locally on the edge, not dependent on any remote command.
- **Clean role separation** — Brain thinks, Edge moves. No AI inference on the Pi 4, no kinematics on the Pi 16GB.

</essential>

<boundaries>
## What's Out of Scope

- Computer vision — No camera processing, object detection, or visual navigation
- Voice control — No speech commands to the robot
- Multi-robot coordination — Just one hexapod, no swarm behavior
- AI on the edge — All reasoning stays on Pi 16GB, edge is pure execution
- Complex autonomy — Robot receives goals and executes, doesn't make strategic decisions

</boundaries>

<specifics>
## Specific Ideas

- Freenove Hexapod as the hardware platform
- orion-edge-agent written in Go (matches bus, good for ARM cross-compilation)
- MQTT for telemetry and commands between edge and core
- Redis Streams subscription from Pi 4 to Pi 16GB for event bus integration
- "Sit & Freeze" as the universal safe state — legs fold, robot lowers to ground, waits

</specifics>

<notes>
## Additional Context

The Pi 4 has only 4GB RAM — cannot run LLMs or heavy processing. It's purely a real-time kinematics controller with safety logic. All the "thinking" happens on orion-core (Pi 16GB) where the AI Council and Brain run.

This is a true distributed system: two separate Raspberry Pis communicating over the network. The edge must be resilient to network partitions and always fail-safe.

</notes>

---

*Phase: 06-edge-integration*
*Context gathered: 2026-01-17*
