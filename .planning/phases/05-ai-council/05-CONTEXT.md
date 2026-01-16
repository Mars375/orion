# Phase 5: AI Council - Context

**Gathered:** 2026-01-16
**Status:** Ready for research

<vision>
## How This Should Work

**Sequential Consensus & Hybrid Critique Architecture**

The AI Council is a validation layer that operates within Raspberry Pi 5 hardware constraints (16GB RAM, no parallel model execution). The flow is:

1. **Brain proposes**: Lightweight Python logic or small local SLM (Phi-3/Gemma-2b) generates initial draft decision based on policy
2. **Council validates**: Proposal sent to validators sequentially (cannot load multiple LLMs simultaneously)
   - External APIs for high intelligence (Claude/OpenAI)
   - Local models loaded sequentially to conserve RAM
3. **Consensus aggregates**: Brain collects critiques and applies consensus rules
   - If confidence drops below threshold → action blocked or escalated to Admin
   - If models disagree on safety → action blocked or escalated to Admin

**Key Constraint:** No parallel local model execution. Sequential validation only.

Council members evaluate both:
- **Safety classification**: Is Brain's SAFE/RISKY assessment correct per policies?
- **Reasoning quality**: Does Brain's reasoning make sense given the incident context?

</vision>

<essential>
## What Must Be Nailed

- **Safety Veto Authority**: Council can block Brain decisions if any model flags safety concerns with high confidence, regardless of majority vote
- **Deterministic Validation**: Council critiques and votes (Approve/Reject) but never proposes alternative actions
- **Hardware Viability**: Sequential model execution that respects Pi 5 RAM limits (no OOM crashes)

The Council is a safety backstop, not an advisory board. If Council raises red flags, action does not proceed.

</essential>

<boundaries>
## What's Out of Scope

- **NO Council proposing alternatives**: Council validates only, doesn't create. This prevents "negotiation loops" and keeps logic deterministic.
- **NO Model training or fine-tuning**: Not viable on Pi 5 without compromising stability. Use frozen, pre-trained models or external APIs only.
- **NO Real-time model updates**: No dynamic switching of model weights during execution to avoid OOM crashes and unpredictable latency.
- **NO Local Vision Models**: Keep Council focused on text/logs reasoning to preserve RAM for core SRE functions.

</boundaries>

<specifics>
## Specific Ideas

**Staged Validation Strategy:**
- Start with cheap local model validator (e.g., Phi-3 or Gemma-2b loaded sequentially)
- Escalate to expensive external API (Claude/OpenAI) only if:
  - Local model is uncertain (low confidence)
  - Local model flags potential safety concern
  - Decision is RISKY (requires higher scrutiny)

This optimizes for cost and latency while maintaining safety rigor.

**Hybrid Model Pool:**
- External APIs: Claude (high reasoning), OpenAI (diverse perspective)
- Local SLMs: Phi-3, Gemma-2b (quick validation, loaded one at a time)
- Sequential execution: Load model → evaluate → unload → next model

</specifics>

<notes>
## Additional Context

**Hardware Constraints (Critical):**
- Raspberry Pi 5 (16GB RAM)
- Cannot load multiple LLMs simultaneously
- Must avoid OOM crashes that could destabilize core SRE functions
- Sequential validation is mandatory, not optional

**Safety Philosophy:**
- Council as validator fits ORION's "conservative by default" principle
- Multiple independent evaluations reduce single-point-of-failure risk
- Safety veto ensures no single Brain mistake bypasses scrutiny

**Integration Points:**
- Memory embeddings (orion-memory) - future enhancement for Council to reference similar past incidents
- Approval system (Phase 4) - Council rejection may still allow Admin override in N3 mode

</notes>

---

*Phase: 05-ai-council*
*Context gathered: 2026-01-16*
