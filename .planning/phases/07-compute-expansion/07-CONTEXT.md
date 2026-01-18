# Phase 7: Compute Expansion - Context

**Gathered:** 2026-01-18
**Status:** Ready for planning

<vision>
## How This Should Work

ORION expands compute capacity by treating the local Pi cluster as a distributed inference grid. Different models run on different nodes based on their specs, with the Brain routing inference requests intelligently without caring which specific node handles them.

The system uses health-aware routing — it knows what each node can handle (CPU, RAM, temperature) and routes intelligently based on current conditions. When a node is overloaded or overheating, requests automatically flow elsewhere.

The key insight is **sticky routing**: prioritize nodes that already have the required model resident in memory. This avoids cold-start latencies from loading models repeatedly. The Brain asks for inference, the system figures out where to run it.

</vision>

<essential>
## What Must Be Nailed

- **Resource awareness** — System knows what each node can handle and routes intelligently based on health metrics
- **Sticky routing** — Prioritize nodes with models already loaded to avoid cold-start latency
- **Safety backoff** — Nodes automatically marked unavailable when temperature > 75°C or RAM > 90%

</essential>

<boundaries>
## What's Out of Scope

- **Cloud integration** — ORION must remain 100% local/air-gapped. No AWS/GCP/Azure dependencies.
- **GPU orchestration** — CPU-only distribution (Ollama on ARM). GPU can come later.
- **Auto-scaling** — Fixed cluster of 3 nodes. No dynamic provisioning or spin-up/down.
- **Pi 4G inference** — Robot node (orion-edge) excluded from LLM tasks, reserved for kinematics only.

</boundaries>

<specifics>
## Specific Ideas

**Health-Aware Ollama Cluster:**
- Use Go Bus (Redis) as central registry
- Each node reports health: CPU, RAM, Temperature
- Distributed inference grid across the cluster

**Hardware Roles:**
- Pi 16GB (Master): Primary validator + Orchestrator
- Pi 8GB (Worker): Dedicated to heavy LLM analysis (Tier 2)
- Pi 4GB (Robot): EXCLUDED from LLM tasks (kinematics only)

**Inference Strategy:**
- Sticky routing: prefer nodes with model already resident
- Health-aware: route based on node health, memory pressure, load
- Safety backoff: auto-mark unavailable at temp > 75°C or RAM > 90%

**Communication:**
- Redis Streams for async request/response
- Decouples compute from real-time bus logic
- Ollama-based workers, Brain routes to whichever has the right model

**Resource Budget:**
- Maximize 24GB RAM available across local cluster
- No external API dependencies for core inference

</specifics>

<notes>
## Additional Context

This phase completes the compute architecture for ORION. With Phase 6 providing edge integration (robot control) and Phase 5 providing AI Council (multi-model reasoning), Phase 7 enables those models to run distributed across available hardware.

The goal is making the homelab cluster feel like a single inference endpoint while being aware of the physical constraints of each node.

</notes>

---

*Phase: 07-compute-expansion*
*Context gathered: 2026-01-18*
