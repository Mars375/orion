# Phase 5: AI Council - Research

**Research conducted:** 2026-01-16
**Status:** Complete
**Context file:** 05-CONTEXT.md

---

## Executive Summary

Phase 5 implements a **Sequential Consensus & Hybrid Critique** architecture where multiple AI models validate Brain decisions. Key constraint: **Raspberry Pi 5 (16GB RAM)** cannot load multiple LLMs simultaneously.

**Viable approach confirmed:**
- Local SLMs (Phi-3 Mini 3.8B, Gemma-2 2B) run sequentially on Pi 5
- External APIs (Claude, OpenAI) for high-confidence validation
- Sophisticated consensus mechanisms beyond simple majority voting
- Proper memory management prevents OOM crashes

---

## 1. Core Technology: Small Language Model Serving

### Phi-3 Mini (3.8B Parameters)

**Performance on Raspberry Pi 5:**
- **5-7 tokens/sec** achievable with proper optimization
- **Q4_K_M quantization** recommended (balances quality/size)
- Requires **thermal management** (heatsinks, PWM tuning)
- Observed: 3.42 tokens/s with phi3.5-3.8b-q4 in practice

**Deployment Options:**
- Ollama: Easiest setup, good Python client
- ONNX Runtime: Microsoft's optimized inference
- llama.cpp: Maximum performance, lower-level control

**Sources:**
- [Phi-3 Mini Raspberry Pi 5 deployment guide](https://medium.com/)
- [DFRobot Phi-3 optimization](https://www.dfrobot.com/)
- [Towards Data Science performance analysis](https://towardsdatascience.com/)

### Gemma-2 2B (2.6B Parameters)

**Performance on Raspberry Pi 5:**
- **Only 3GB RAM** usage on Pi 5
- **Q4_0 quantization** → 1.6GB model size
- **Better tokens/sec than Phi-3** (smaller model)
- Excellent response quality reported in testing

**Framework Compatibility:**
- Ollama: Full support
- llama.cpp: Better performance than Ollama
- Native ARM64 support

**Sources:**
- [llama.cpp GitHub](https://github.com/ggerganov/llama.cpp)
- [Gemma-2 Raspberry Pi deployment](https://medium.com/)
- [DFRobot benchmarks](https://www.dfrobot.com/)
- [Hugging Face model card](https://huggingface.co/)

### Key Takeaways

✅ **Both models viable** on Pi 5 with sequential loading
✅ **Gemma-2 2B preferred** for lower RAM usage (3GB vs ~5GB)
✅ **Q4 quantization** (Q4_K_M or Q4_0) optimal for Pi 5
⚠️ **Thermal management critical** for sustained inference
⚠️ **3-7 tokens/sec** means validation takes seconds (not milliseconds)

---

## 2. Ecosystem: Model Serving Frameworks

### Ollama (Recommended)

**Pros:**
- Simplest setup (`ollama serve`)
- Official Python client (`ollama-python`)
- Model lifecycle: `pull`, `create`, `rm`, `ps`
- REST API: `http://localhost:11434/api/generate`

**Memory Management:**
- **8GB RAM minimum** for 7B models
- **16GB RAM minimum** for 13B models
- `ollama ps` lists loaded models
- `ollama stop` halts running models (but no explicit unload API documented)

**Resource Constraints:**
- Select **1B-3B parameter models** for minimal systems
- Docker deployment option
- Embedding-focused models (`embeddinggemma`) for lighter workloads

**Sources:**
- [Ollama GitHub](https://github.com/ollama/ollama)

### llama.cpp

**Pros:**
- **Maximum performance** (lower-level control)
- Aggressive quantization: 1.5-bit to 8-bit
- Python bindings: `abetlen/llama-cpp-python`
- ARM NEON optimization (Apple Silicon first-class)

**Cons:**
- More complex setup
- No explicit model unloading API documented
- Raspberry Pi-specific benchmarks not published

**Quantization for Low-RAM:**
- 2-bit and 3-bit formats minimize footprint
- 4-bit (Q4_K_M) balances quality/size

**Sources:**
- [llama.cpp GitHub](https://github.com/ggerganov/llama.cpp)

### Decision: Ollama for Phase 5

**Rationale:**
- Simpler API surface
- Official Python client
- Model lifecycle management built-in
- Trade-off: Slightly lower tokens/sec than llama.cpp acceptable for validation use case

---

## 3. Patterns: Multi-Model Consensus & Confidence Scoring

### State-of-the-Art (2025-2026)

Recent research shows **sophisticated approaches far beyond majority voting**:

#### ReConcile Framework (2025)

**Multi-agent round-table conference:**
- Each LLM agent discusses over multiple rounds
- **Confidence-weighted voting** mechanism
- Agents estimate uncertainty via confidence prompts
- Once agents converge, weighted vote computed as team answer

**Sources:**
- [ReConcile paper (arXiv)](https://arxiv.org/abs/2309.13007)
- [OpenReview discussion](https://openreview.net/forum?id=Yol6nUVIJD)

#### Multi-Model Consensus Reasoning Engine (January 2026)

**Supervised meta-learning approach:**
- Each LLM self-reports confidence score
- For open-weight models, average log-probability of answer tokens computed
- Features: semantic, structural, reasoning, confidence, model-specific
- Gradient-boosted trees, ranking models, graph neural networks
- **Significant improvements** in accuracy, calibration, hallucination robustness

**Sources:**
- [Learning to Trust the Crowd (arXiv)](https://arxiv.org/html/2601.07245)

#### Advanced Ensemble Techniques (August 2025)

**Integrates GPT-4, LLaMA 3.3, Claude 3:**
- Non-parametric confidence calibration using **isotonic regression**
- **8.4% improvement** over simple majority voting
- Validated across multiple domains

**Sources:**
- [MDPI ensemble classification paper](https://www.mdpi.com/2079-9292/14/17/3404)

#### Beyond Majority Voting (October 2025)

**Two new algorithms:**
- **Optimal Weight (OW)**: Leverages first-order and second-order information
- **Inverse Surprising Popularity (ISP)**: Uses higher-order information
- Provably mitigates limitations of majority voting
- More reliable collective decisions

**Sources:**
- [Beyond Majority Voting (arXiv)](https://arxiv.org/html/2510.01499v1)

### Object Detection Consensus (December 2025)

**Applicable pattern for AI Council:**
- Multiple models process same input
- Each produces independent prediction + confidence score
- Merge using consensus algorithms:
  - **Weighted Boxes Fusion (WBF)**
  - **Agglomerative Late Fusion Algorithm (ALFA)**
  - **Probabilistic Ensembling (ProbEn)**

**Sources:**
- [Multiple Large AI Models' Consensus (MDPI)](https://www.mdpi.com/2076-3417/15/24/12961)

### Recommended Approach for ORION

**Staged Validation with Confidence Weighting:**

1. **Local SLM validates** (Gemma-2 2B):
   - Self-reported confidence score
   - Log-probability of answer tokens
   - Fast (3-7 seconds on Pi 5)

2. **Escalate to external API if:**
   - Local model confidence < threshold (e.g., 0.7)
   - Local model flags safety concern
   - Decision is RISKY classification

3. **Consensus aggregation:**
   - **Confidence-weighted voting** (not simple majority)
   - **Safety veto**: Any model flagging safety concern blocks action
   - **Isotonic regression** for confidence calibration (future enhancement)

**Implementation patterns:**
- Use Kinde LLM Fan-Out patterns for async validation
- Leverage higher-order information (model agreement/disagreement patterns)
- Avoid "negotiation loops" (Council validates, never proposes)

**Sources:**
- [Kinde LLM Fan-Out 101](https://kinde.com/learn/ai-for-software-engineering/workflows/llm-fan-out-101-self-consistency-consensus-and-voting-patterns/)
- [LLM Voting: Human Choices and AI (arXiv)](https://arxiv.org/html/2402.01766v2)

---

## 4. Memory Management: Sequential Model Loading

### HuggingFace Transformers Patterns

**Loading Models:**
- `device_map="sequential"`: Fits what it can on GPU 0, then GPU 1, etc.
- `device_map="auto"`: Evenly splits across all GPUs
- Accelerate's Big Model Inference for sharded checkpoints
- Fast initialization with lower bit data types

**Releasing RAM (CPU):**

**Linux:**
```python
import ctypes
import gc

# Delete model
del model
gc.collect()

# Trim malloc
libc = ctypes.CDLL("libc.so.6")
libc.malloc_trim(0)
```

**Alternative (no ctypes):**
```bash
export MALLOC_TRIM_THRESHOLD_=-1
```

**Releasing VRAM (GPU):**
```python
import torch

# Delete model
del model
torch.cuda.empty_cache()  # Release Reserved Memory
gc.collect()
```

**Known Issue:**
- GPU memory may not return to baseline after sequential training jobs
- Proper cleanup procedures critical

**Sources:**
- [How to Release HuggingFace Models from RAM/VRAM](https://mjunya.com/en/posts/2025-01-27-hf-torch-clear-memory/)
- [HuggingFace transformers #13208](https://github.com/huggingface/transformers/issues/13208)
- [HuggingFace OOM Issue #1742](https://github.com/huggingface/transformers/issues/1742)
- [Loading big models into memory](https://huggingface.co/docs/accelerate/en/concept_guides/big_model_inference)

### Ollama Memory Management

**Model Lifecycle:**
```bash
ollama pull gemma2:2b    # Download model
ollama ps                # List loaded models
ollama stop <model>      # Halt running model
ollama rm <model>        # Remove from disk
```

**Python Client:**
```python
import ollama

# Generate (loads model if needed)
response = ollama.generate(model='gemma2:2b', prompt='...')

# Chat API
response = ollama.chat(model='gemma2:2b', messages=[...])

# Embeddings
response = ollama.embeddings(model='embeddinggemma', prompt='...')
```

**RAM Guidelines:**
- 8GB RAM: 7B models
- 16GB RAM: 13B models (but Pi 5 should stick to 2-3B)
- 32GB RAM: 33B models

**Sources:**
- [Ollama GitHub](https://github.com/ollama/ollama)

### Sequential Orchestration Pattern (ORION AI Council)

**Proposed Flow:**

```python
import ollama
import gc

async def validate_with_local_slm(decision):
    """Load Gemma-2 2B, validate, unload."""
    # Ollama auto-loads on first request
    response = ollama.generate(
        model='gemma2:2b',
        prompt=f"Validate this decision: {decision}..."
    )

    # Extract confidence and critique
    confidence = extract_confidence(response)
    critique = response['response']

    # Ollama stop command (manual cleanup if needed)
    # Note: Ollama may keep model warm for performance

    return confidence, critique

async def validate_with_external_api(decision):
    """Call Claude or OpenAI API."""
    import anthropic

    client = anthropic.Anthropic()
    response = client.messages.create(
        model="claude-3-5-sonnet-20241022",
        messages=[{"role": "user", "content": f"Validate: {decision}"}]
    )

    return extract_confidence(response), response.content[0].text

async def council_validate(decision):
    """Sequential validation with escalation."""
    # Stage 1: Local SLM
    local_conf, local_critique = await validate_with_local_slm(decision)

    # Stage 2: Escalate if uncertain or RISKY
    if local_conf < 0.7 or decision['classification'] == 'RISKY':
        api_conf, api_critique = await validate_with_external_api(decision)

        # Confidence-weighted consensus
        if local_conf < 0.5 or api_conf < 0.5:
            return "BLOCKED", [local_critique, api_critique]

    return "APPROVED", [local_critique]
```

**Key Principles:**
- One model active at a time
- Ollama handles load/unload implicitly
- Manual cleanup with `ollama stop` if needed
- External APIs only for escalation (cost optimization)

---

## 5. External API Clients

### Anthropic Claude API (2026)

**Official Python SDK:**
```bash
pip install anthropic
```

**Setup:**
```python
import anthropic

client = anthropic.Anthropic(api_key=os.environ["ANTHROPIC_API_KEY"])
```

**Best Practices:**
- Use **native Claude API** (not OpenAI compatibility layer)
- OpenAI SDK compatibility for testing only (not production)
- Structured Outputs: Define schemas as "tools"
- PDF processing, citations, extended thinking, prompt caching

**Performance Highlights:**
- **200K token context window** (handles entire codebases)
- **40%+ error rate reduction** in workflows
- **95%+ consistency** for JSON structured outputs

**Sources:**
- [OpenAI SDK compatibility](https://platform.claude.com/docs/en/api/openai-sdk)
- [Anthropic Academy](https://www.anthropic.com/learn/build-with-claude)
- [Claude Agent SDK](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)
- [Anthropic SDK Python GitHub](https://github.com/anthropics/anthropic-sdk-python)

### OpenAI API (2026)

**Official Python SDK:**
```bash
pip install openai
```

**Setup:**
```python
import openai

client = openai.OpenAI(api_key=os.environ["OPENAI_API_KEY"])
```

**Usage for Council:**
- Provides **diverse perspective** (complement Claude)
- Structured outputs via function calling
- JSON mode for deterministic responses

**Sources:**
- [Structured outputs guide](https://agenta.ai/blog/the-guide-to-structured-outputs-and-function-calling-with-llms)

### Recommended Integration

**Hybrid Model Pool:**
```python
class AICouncil:
    def __init__(self):
        self.local_model = "gemma2:2b"  # Ollama
        self.anthropic = anthropic.Anthropic()
        self.openai = openai.OpenAI()

    async def validate(self, decision):
        """Sequential validation with escalation."""
        # Stage 1: Local (always)
        local_result = await self._validate_local(decision)

        # Stage 2: External API (if needed)
        if self._should_escalate(local_result, decision):
            external_results = await asyncio.gather(
                self._validate_claude(decision),
                self._validate_openai(decision)
            )

            return self._aggregate_consensus([local_result, *external_results])

        return local_result
```

**Cost Optimization:**
- Local SLM: Free (after model download)
- Claude API: Pay per token (use for RISKY only)
- OpenAI API: Pay per token (tie-breaker or second opinion)

---

## 6. LLM Orchestration Frameworks

### CrewAI

**Features:**
- Opinionated Crew → Task → Agent hierarchy
- Built-in **asyncio** support
- Sequential or hierarchical processes
- Automates complex multi-agent projects

**Sources:**
- [LLM Orchestration comparison](https://research.aimultiple.com/llm-orchestration/)
- [Top LLM frameworks](https://www.secondtalent.com/resources/top-llm-frameworks-for-building-ai-agents/)

### LangGraph

**Features:**
- Hierarchical LLM workflows
- Parallel or sequential task passing
- Collaborative work between specialized models

**Sources:**
- [AI orchestration tools](https://www.cudocompute.com/blog/llms-ai-orchestration-toolkits-comparison)

### JustLLMs

**Features:**
- Production-ready multi-provider orchestration
- Python library
- Unified interface for Claude, OpenAI, etc.

**Sources:**
- [JustLLMs GitHub](https://github.com/just-llms/justllms)

### Asyncio Best Practices (Python)

**For LLM concurrency:**
- Use `asyncio.gather()` for parallel external API calls
- Use `asyncio.create_task()` for fire-and-forget
- Sequential local model loading (cannot parallelize on Pi 5)

**Sources:**
- [Python Asyncio for LLM Concurrency](https://www.newline.co/@zaoyang/python-asyncio-for-llm-concurrency-best-practices--bc079176)

### Recommendation for ORION

**Don't use heavyweight frameworks (CrewAI, LangGraph) yet:**
- ORION Council has specific constraints (sequential local, hybrid external)
- Simple asyncio orchestration sufficient for Phase 5
- Consider frameworks in Phase 6+ if complexity grows

**Rationale:**
- Frameworks add complexity and dependencies
- ORION's deterministic validation doesn't need agent negotiation
- Custom orchestration gives full control over memory management

---

## 7. Common Pitfalls & Mitigations

### Pitfall 1: OOM Crashes on Pi 5

**Risk:**
- Loading multiple LLMs simultaneously exhausts 16GB RAM
- Background services (Redis, Guardian, Brain) also use RAM
- OOM kills destabilize core SRE functions

**Mitigation:**
- **Sequential validation only** (enforced in architecture)
- Monitor RAM with `psutil` before loading models
- Set Ollama memory limits if possible
- Use smallest viable models (Gemma-2 2B preferred over Phi-3 3.8B)

### Pitfall 2: Thermal Throttling

**Risk:**
- Sustained inference generates heat
- Pi 5 throttles CPU/GPU when overheating
- Tokens/sec drops, validation latency increases

**Mitigation:**
- Install **proper heatsinks** (active cooling if needed)
- Monitor temperature with `vcgencmd measure_temp`
- Tune PWM for fan control
- Limit inference duration (timeout after 30 seconds)

**Sources:**
- [DFRobot Phi-3 optimization](https://www.dfrobot.com/)

### Pitfall 3: Autoscaling Ineffectiveness

**Risk:**
- Traditional CPU/memory autoscaling doesn't work for LLMs
- Resource usage highly unpredictable
- Cold start latency (minutes to load model into memory)

**Mitigation:**
- ORION is single-node (no autoscaling)
- Keep Gemma-2 2B "warm" in Ollama if possible
- Accept 3-7 second validation latency as baseline

**Sources:**
- [LLM Orchestration challenges](https://research.aimultiple.com/llm-orchestration/)

### Pitfall 4: Confidence Score Miscalibration

**Risk:**
- LLMs overconfident in wrong answers
- Simple confidence scores misleading
- Consensus voting may amplify errors

**Mitigation:**
- Use **isotonic regression** for calibration (future enhancement)
- Weight models differently (Claude > local SLM for safety)
- **Safety veto**: Any high-confidence safety flag blocks action

**Sources:**
- [Advanced Ensemble Techniques (MDPI)](https://www.mdpi.com/2079-9292/14/17/3404)

### Pitfall 5: Memory Leaks in Sequential Loading

**Risk:**
- Python GC doesn't release model memory
- VRAM not cleared after `del model`
- Long-running process accumulates memory usage

**Mitigation:**
- Explicit cleanup: `del model; gc.collect(); libc.malloc_trim(0)`
- Restart Council process periodically (systemd watchdog)
- Monitor memory trends with Prometheus

**Sources:**
- [HuggingFace memory issues](https://github.com/huggingface/transformers/issues/13208)

### Pitfall 6: Network Dependency for External APIs

**Risk:**
- External API calls fail if network unavailable
- Validation blocked waiting for timeout
- Defeats "conservative by default" principle

**Mitigation:**
- **Fail-closed on API timeout**: Treat as "BLOCKED"
- Local SLM always runs first (offline-capable)
- Admin override available in N3 mode if API unavailable

---

## 8. Don't Hand-Roll These

### ❌ Don't Build: Model Serving Infrastructure

**Use:**
- **Ollama** for local model serving
- Existing REST API (`localhost:11434`)
- Official Python client (`ollama-python`)

**Why:**
- Model loading, quantization, optimization already solved
- Thermal management, memory mapping, GGUF format handling complex
- Community-tested on Raspberry Pi

### ❌ Don't Build: External API Clients

**Use:**
- **anthropic-sdk-python** for Claude API
- **openai-sdk-python** for OpenAI API
- Official SDKs handle retries, rate limits, streaming

**Why:**
- API changes handled by vendor
- Error handling, backoff, timeouts already implemented
- Structured outputs, JSON mode, function calling built-in

### ❌ Don't Build: Quantization Tools

**Use:**
- Pre-quantized models from Ollama library
- GGUF models from Hugging Face
- `gemma2:2b` and `phi3:3.8b` tags in Ollama

**Why:**
- Quantization quality depends on calibration datasets
- Q4_K_M and Q4_0 formats already optimized
- Community validation of quality/size tradeoffs

### ✅ Do Build: Council Orchestration Logic

**Custom implementation required:**
- Sequential validation flow (local → escalate → aggregate)
- Confidence-weighted consensus algorithm
- Safety veto logic (any model flags concern → block)
- Integration with ORION Brain, Guardian, Commander

**Why:**
- ORION-specific constraints (N0/N2/N3 modes, SAFE/RISKY policies)
- Deterministic validation (no negotiation loops)
- Integration with existing contracts and event bus

### ✅ Do Build: Monitoring and Safeguards

**Custom implementation required:**
- RAM monitoring before model loading
- Temperature monitoring during inference
- Validation timeout enforcement (30 seconds max)
- OOM prevention checks

**Why:**
- Pi 5 resource constraints unique to ORION
- Integration with Prometheus metrics
- Fail-closed behavior aligned with safety philosophy

---

## 9. SOTA Check (2026)

### Current State-of-the-Art

**Small Language Models:**
- **Gemma-2 2B** (Google, 2.6B parameters): Best tokens/sec on Pi 5, 3GB RAM
- **Phi-3 Mini** (Microsoft, 3.8B parameters): Strong reasoning, 5GB RAM
- **LLaMA 3.3** (Meta): Larger (7B+), too heavy for Pi 5
- **Qwen 2.5** (Alibaba): Competitive with Gemma-2, less Pi 5 testing

**Consensus Mechanisms:**
- **Confidence-weighted voting** (2025-2026 research)
- **Isotonic regression calibration** (proven 8.4% improvement)
- **Higher-order information** (OW, ISP algorithms)
- Moving beyond simple majority voting

**Inference Frameworks:**
- **Ollama**: Dominant for ease of use
- **llama.cpp**: Best raw performance
- **vLLM**: High-throughput server (overkill for Pi 5)
- **ONNX Runtime**: Microsoft optimization

**External APIs (January 2026):**
- **Claude 3.7 Sonnet**: Latest model (superior reasoning)
- **GPT-4 Turbo**: Strong baseline
- **Gemini 1.5 Pro**: Google alternative

**Orchestration:**
- **CrewAI**, **LangGraph**: Mature agent frameworks
- **JustLLMs**: Multi-provider abstraction
- Custom asyncio orchestration still viable for deterministic workflows

### Technology Choices for ORION Phase 5

| Component | Choice | Rationale |
|-----------|--------|-----------|
| **Local SLM** | Gemma-2 2B | Lowest RAM (3GB), good tokens/sec, proven on Pi 5 |
| **Inference** | Ollama | Simplest API, official Python client, lifecycle management |
| **Quantization** | Q4_0 or Q4_K_M | Optimal quality/size for Pi 5 |
| **External API #1** | Claude 3.7 Sonnet | Superior reasoning, structured outputs, 200K context |
| **External API #2** | OpenAI GPT-4 Turbo | Diverse perspective, tie-breaker |
| **Consensus** | Confidence-weighted voting | SOTA research (2025-2026), proven improvements |
| **Orchestration** | Custom asyncio | Deterministic, no heavyweight framework overhead |
| **Safety Veto** | Custom logic | ORION-specific policies and N0/N2/N3 modes |

---

## 10. Implementation Recommendations

### Phase 5 Architecture (Detailed)

**Module: `orion-council`** (Python)

**Components:**

1. **CouncilValidator** (local SLM interface)
   - Wraps Ollama API
   - Loads Gemma-2 2B on-demand
   - Extracts confidence scores
   - Handles timeouts (30s max)

2. **ExternalValidator** (API interface)
   - Wraps Claude + OpenAI SDKs
   - Parallel API calls (asyncio.gather)
   - Retry logic with exponential backoff
   - Fail-closed on timeout

3. **ConsensusAggregator** (voting logic)
   - Confidence-weighted voting algorithm
   - Safety veto enforcement
   - Escalation logic (local → external)
   - Threshold tuning (default: 0.7)

4. **MemoryManager** (resource monitoring)
   - Check RAM before loading model
   - Monitor temperature during inference
   - OOM prevention (block if < 4GB free)
   - Cleanup after validation

**Event Flow:**

```
Brain → Decision (event) → Redis Streams
         ↓
      Council subscribes
         ↓
      CouncilValidator (local SLM)
         ↓ (if uncertain or RISKY)
      ExternalValidator (Claude + OpenAI)
         ↓
      ConsensusAggregator
         ↓
      Validation Result → Redis Streams
         ↓
      Commander checks validation before execution
```

### Testing Strategy

**Unit Tests:**
- CouncilValidator with mocked Ollama
- ExternalValidator with mocked API responses
- ConsensusAggregator voting logic
- MemoryManager OOM prevention

**Integration Tests:**
- End-to-end validation flow with fakeredis
- Timeout handling (30s enforcement)
- Safety veto scenarios (any model blocks → action blocked)
- Escalation triggers (low confidence, RISKY classification)

**Load Tests:**
- Sequential validations (10 decisions in a row)
- Monitor RAM usage over time
- Check for memory leaks
- Thermal throttling detection

### Performance Targets

**Latency:**
- Local SLM validation: 3-7 seconds
- External API validation: 2-5 seconds (parallel)
- Total worst-case: 12 seconds (local + external)

**Accuracy:**
- False positive rate: < 5% (blocking safe actions)
- False negative rate: < 1% (approving unsafe actions)
- Safety veto effectiveness: 100% (any concern blocks)

**Resource Usage:**
- Peak RAM: < 12GB (leave 4GB for other services)
- CPU temperature: < 70°C sustained
- No OOM crashes over 48-hour run

### Safety Invariants

**Council must enforce:**
- ✅ **Safety veto**: Any model flags concern → block action
- ✅ **Deterministic validation**: No negotiation loops
- ✅ **Timeout enforcement**: Max 30 seconds per validation
- ✅ **Fail-closed**: Errors/timeouts → block action
- ✅ **Sequential loading**: Never load multiple local models
- ✅ **RAM protection**: Don't load if < 4GB free

**Council must NOT:**
- ❌ Propose alternative actions (Brain's job)
- ❌ Modify decisions (validate only)
- ❌ Auto-approve on uncertainty (escalate or block)
- ❌ Train or fine-tune models
- ❌ Load vision models
- ❌ Bypass Brain policies

---

## 11. Open Questions for Planning Phase

**1. Confidence Threshold Tuning:**
- What threshold triggers escalation? (Proposed: 0.7)
- Should threshold vary by decision classification (SAFE vs RISKY)?
- How to calibrate thresholds empirically?

**2. Safety Veto Weighting:**
- Is any model flagging concern sufficient to block?
- Should we weight Claude > local SLM for safety concerns?
- What constitutes "high confidence" safety flag?

**3. Model Selection:**
- Start with Gemma-2 2B only, or support Phi-3 as fallback?
- Should we use different models for different decision types?
- When to add third local model (e.g., Qwen 2.5)?

**4. Cost Management:**
- Daily budget for Claude/OpenAI API calls?
- Metrics to track: API calls per day, cost per validation
- Circuit breaker for API costs (max $X/day)?

**5. Memory Embeddings Integration:**
- Should Council reference similar past incidents (orion-memory)?
- How to retrieve relevant context without slowing validation?
- Future enhancement or Phase 5 requirement?

**6. Admin Override:**
- If Council blocks action, can Admin override in N3 mode?
- Should overrides be logged differently (higher scrutiny)?
- What happens if Admin overrides Council repeatedly?

**7. Observability:**
- What metrics to expose? (validation latency, consensus votes, RAM usage)
- Grafana dashboard for Council health?
- Alerts on: OOM risk, thermal throttling, API failures

---

## 12. Next Steps

**Immediate (Planning Phase):**
1. Review RESEARCH.md with stakeholders
2. Answer open questions above
3. Create detailed implementation plans
4. Define contracts for Council validation messages
5. Update policies (thresholds, escalation rules)

**Phase 5 Implementation Order:**
1. Implement CouncilValidator (local SLM interface)
2. Implement ExternalValidator (API interface)
3. Implement ConsensusAggregator (voting logic)
4. Implement MemoryManager (resource monitoring)
5. Integrate with Brain decision flow
6. Add Commander validation checks
7. Write comprehensive tests (unit, integration, load)
8. Deploy to Pi 5 and tune thresholds

**Success Criteria:**
- ✅ All 238 existing tests still pass
- ✅ Council tests added (50+ new tests)
- ✅ No OOM crashes in 48-hour run
- ✅ Validation latency < 12 seconds (worst case)
- ✅ Safety veto blocks unsafe actions (100% effectiveness)
- ✅ False positive rate < 5%

---

## Appendix: Sources

### Multi-Model Consensus Research
- [Learning to Trust the Crowd: Multi-Model Consensus Reasoning Engine](https://arxiv.org/html/2601.07245)
- [Multiple Large AI Models' Consensus for Object Detection](https://www.mdpi.com/2076-3417/15/24/12961)
- [ReConcile: Round-Table Conference Improves Reasoning via Consensus](https://arxiv.org/abs/2309.13007)
- [ReConcile OpenReview Discussion](https://openreview.net/forum?id=Yol6nUVIJD)
- [Kinde LLM Fan-Out 101: Self-Consistency, Consensus, and Voting Patterns](https://kinde.com/learn/ai-for-software-engineering/workflows/llm-fan-out-101-self-consistency-consensus-and-voting-patterns/)
- [LLM Voting: Human Choices and AI Collective Decision-Making](https://arxiv.org/html/2402.01766v2)
- [Beyond Majority Voting: Leveraging Higher-Order Information](https://arxiv.org/html/2510.01499v1)
- [Integrated Survey Classification via LLMs: Ensemble Approach](https://www.mdpi.com/2079-9292/14/17/3404)

### Memory Management & Model Loading
- [How to Release HuggingFace Models from RAM/VRAM](https://mjunya.com/en/posts/2025-01-27-hf-torch-clear-memory/)
- [HuggingFace transformers Issue #13208: Loading takes much RAM](https://github.com/huggingface/transformers/issues/13208)
- [HuggingFace transformers Issue #1742: OOM when repeatedly running models](https://github.com/huggingface/transformers/issues/1742)
- [HuggingFace Docs: Loading models](https://huggingface.co/docs/transformers/models)
- [HuggingFace Docs: Loading big models into memory](https://huggingface.co/docs/accelerate/en/concept_guides/big_model_inference)

### API Clients & SDKs
- [OpenAI SDK compatibility - Claude Docs](https://platform.claude.com/docs/en/api/openai-sdk)
- [Anthropic Academy: Claude API Development Guide](https://www.anthropic.com/learn/build-with-claude)
- [Structured outputs and function calling with LLMs guide](https://agenta.ai/blog/the-guide-to-structured-outputs-and-function-calling-with-llms)
- [Anthropic Claude API Review 2026](https://hackceleration.com/anthropic-review/)
- [GitHub: anthropic-sdk-python](https://github.com/anthropics/anthropic-sdk-python)
- [Building agents with Claude Agent SDK](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)

### LLM Orchestration
- [Compare Top 12 LLM Orchestration Frameworks 2026](https://research.aimultiple.com/llm-orchestration/)
- [2026 Leading AI Orchestration Tools Coordinate Multiple LLMs](https://www.prompts.ai/en/blog/leading-ai-orchestration-tools-coordinate-multiple-llms)
- [Python Asyncio for LLM Concurrency: Best Practices](https://www.newline.co/@zaoyang/python-asyncio-for-llm-concurrency-best-practices--bc079176)
- [Best LLM & AI orchestration toolkits comparison](https://www.cudocompute.com/blog/llms-ai-orchestration-toolkits-comparison)
- [Top 8 LLM Frameworks for Building AI Agents 2026](https://www.secondtalent.com/resources/top-llm-frameworks-for-building-ai-agents/)
- [GitHub: justllms - Multi-provider LLM orchestration](https://github.com/just-llms/justllms)

### Model Serving & Inference
- [Ollama GitHub Repository](https://github.com/ollama/ollama)
- [llama.cpp GitHub Repository](https://github.com/ggerganov/llama.cpp)
- [vLLM GitHub: High-throughput LLM inference](https://github.com/vllm-project/vllm)

---

**Research Status:** Complete
**Next Phase:** `/gsd:plan-phase 5` to create implementation plans
**Blockers:** None identified
