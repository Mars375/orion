# CLAUDE.md — ORION Development Guide

## What is ORION

ORION is a modular, autonomous homelab system built around **safety**, **SRE principles**, and **controlled intelligence**. It observes, reasons, decides, and acts—but only when safe.

**Core invariants:**
- Observation precedes action
- Inaction is preferred to risky action
- No automation without explicit rules, evidence, and auditability
- Decisions must be explainable
- Human-in-the-loop for anything risky

## Node Roles

- **orion-core**: reasoning, memory, decision logic
- **orion-hub**: external access, media, storage
- **orion-edge**: autonomous edge devices (robot)
- **orion-worker**: optional compute expansion

Nodes are replaceable. Roles are stable.
Edge nodes are never trusted dependencies and must remain safe and functional without ORION.

---

## Technology Stack

**Mixed stack: Python + Go**

The language never defines the architecture. Contracts define everything.

### Python — cognitive complexity
- `orion-brain` (reasoning, policies, decisions)
- `orion-guardian` (correlation, temporal logic)
- `orion-memory` (post-mortems, embeddings)
- `orion-api` (logic exposure)
- `orion-commander` (initial implementation)
- `orion-approval-telegram`

Use Python for: AI, reasoning, SRE logic, rapid iteration.

### Go — reliability and performance
- `orion-bus` (Redis Streams client)
- `orion-edge-agent` (robot, MQTT, offline safety)
- System agents (metrics, watchers)
- CLI tools
- Future workers

Use Go for: long-running services, low memory, static binaries, ARM/x86 cross-compile.

### Communication (no cross-language imports)
- `orion-contracts` (JSON Schema) — language-agnostic
- Redis Streams — event bus
- MQTT — edge telemetry
- HTTP — only for explicit, human-facing or inspection APIs (never for internal control flow)

No shared memory, shared volumes, or implicit state between modules.
All cross-module interaction must occur through explicit, versioned interfaces.

If a Python module becomes performance-critical, it can be rewritten in Go without touching other modules.

---

## Safety Philosophy

ORION prefers to do nothing rather than do the wrong thing.

### Three Pillars (all required, never skip one)

**1. Conservative by default**
- When uncertain, choose the safer and simpler option
- Capability is secondary to safety
- "We can add this later" is acceptable
- Over-optimization is a liability
- If a behavior cannot be safely reversed or stopped, it must remain disabled or classified as RISKY

**2. Ask first**
- Ask before any change affecting autonomy level (N0 → N1 → N2 → N3)
- Ask before altering SAFE vs RISKY classification
- Ask before modifying execution, rollback, or approvals
- Ask before changing time-based reasoning (cooldowns, thresholds)
- If a change could alter how ORION behaves under failure, stop and ask
- If approval or clarification is required, no implementation or enablement work may proceed until it is explicitly granted

**3. Dry-run by default**
- All new behavior starts in observe-only or dry-run mode
- Progression: Observe → Dry-run → Restricted → Full
- Restricted means explicitly scoped enablement (limited targets, limited frequency, explicit rollback, heightened logging)
- Promotion requires explicit approval and evidence
- Dry-run is not optional—it is the default state

### Assumptions Claude must hold
- ORION may run unattended
- Mistakes have real consequences
- False positives are worse than missed actions
- Human trust is fragile and must be preserved
- Long-term stability beats short-term cleverness

### What Claude must never do
- Remove or weaken guardrails without explicit ADR
- Auto-enable behavior because "it seems safe"
- Silently expand scope
- Optimize away approvals
- Treat uncertainty as acceptable risk

---

## Testing Doctrine

Test-alongside: write tests as part of implementation, not before or after.

### Rule 1 — Contracts are sacred
- Validate all inputs against `orion-contracts`
- Reject invalid events explicitly
- Tests must include: valid contract, invalid contract, backward-compatible version
- Backward-compatible tests must verify that older schema versions are accepted without changing semantics or safety behavior

### Rule 2 — Tests alongside code
- Implement a feature, immediately add tests
- Cover: normal path, failure path, edge cases
- Never leave logic untested "temporarily"

### Rule 3 — What must be unit-tested
Safety-critical logic requires unit tests:
- Decision logic (brain)
- Policy evaluation (SAFE vs RISKY)
- Cooldown / circuit breaker logic
- Correlation windows (guardian)
- Approval state machines
- Rollback eligibility logic

### Rule 4 — Integration tests at boundaries
- Test bus ↔ module boundaries
- Test module ↔ module boundaries via events
- Mock Redis, MQTT, Telegram
- Never rely on live infrastructure

### Rule 5 — No test, no action
If code can trigger an action, escalate, or change state—it must be tested or it does not ship.

Manual testing, log inspection, or interactive validation does not count as testing for the purpose of shipping code.

### Test quality
- Tests must be readable and describe behavior
- Test names should encode safety expectations
- Bad: `assert foo(bar) == 3`
- Good: `test_risky_action_is_never_executed_without_approval`

---

## Git Workflow

Module-scoped branches with conventional commits on `main`.

### Branch rules
- One module per branch: `module/<module-name>`
- Examples: `module/orion-brain`, `module/orion-bus`, `module/orion-guardian`
- Forbidden: touching multiple modules in one branch, "while I'm here" changes

### Commits on branches
- Commit freely with descriptive messages
- No strict format required during development

### Commits on main
- All merges are squash merges
- Squash commit must use conventional format:
  ```
  <type>(<module>): <summary>
  ```
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
- Examples:
  - `feat(contracts): introduce versioned event schemas`
  - `fix(bus): enforce schema validation on publish`
  - `test(brain): cover safe vs risky decision paths`

A branch is merge-ready only when:
- All tests pass locally
- Contract validations pass
- Module README is updated if behavior changed
- No TODOs remain in safety-critical paths

If a change legitimately impacts multiple modules (e.g., contracts), the scope must reflect the owning module (e.g., `contracts`) and the commit body must list affected modules explicitly.

### Safety rules for commits
- No behavior change without tests in the same branch
- No contract change without version bump and backward compatibility notes
- No direct commits to `main`—all work through module branches and squash merge

---

## Coding & Documentation Standards

Code must be readable by a human in 10 years, explainable to an auditor, and safe under autonomous execution.

### Docstrings
**Required for:**
- Public functions, classes, module entry points
- Anything that makes decisions, triggers actions, changes state, or enforces safety

**Style:** Short, precise. Focus on what, why, constraints. Not restating code.

**Not required for:** Trivial getters, obvious transformations, private helpers with clear intent.

### Comments
- Comments explain WHY, never WHAT
- Allowed: explaining safety guards, threshold choices, temporal windows, why something is NOT done
- Forbidden: `# increment counter` above `counter += 1`

### Type hints
- Mandatory everywhere (Python and Go)
- No untyped public functions
- Use dataclasses, TypedDict, pydantic where appropriate

### Runtime validation
- Mandatory for anything crossing module boundaries
- Mandatory for input from: Redis Streams, MQTT, Telegram, external APIs
- Type hints are not enough—external input must be validated against contracts
- Validation must occur at module boundaries, before any internal logic or decision-making is executed

### ADRs
Write an ADR only when:
- Changing a safety invariant
- Changing autonomy level
- Changing execution rules
- Introducing architectural coupling
- Changing how decisions are made

Do not write ADRs for: refactoring, bug fixes, performance improvements, adding tests.

### Module READMEs
Every module must have a README containing:
- Purpose (1 paragraph max)
- Inputs / outputs
- Invariants
- Failure modes
- What the module explicitly does NOT do

### What Claude must not do
- Add "just in case" comments
- Document obvious code
- Duplicate ADR text in docstrings
- Add TODOs without context
- Introduce implicit behavior without explanation
- Introduce implicit defaults for safety- or decision-related behavior without documenting and justifying them

---

## Quick Reference

### Before any change, ask yourself:
1. Does this affect autonomy, safety, or decision-making? → Ask first
2. Can this be reversed or stopped safely? → If not, classify as RISKY
3. Is this tested? → No test, no ship
4. Is this validated against contracts? → Validate at boundaries
5. Am I touching multiple modules? → STOP. One module per branch, no exceptions.

### Module mapping

| Module                     | Language |
|---------------------------|----------|
| `orion-contracts`         | Agnostic |
| `orion-bus`               | Go       |
| `orion-guardian`          | Python   |
| `orion-brain`             | Python   |
| `orion-commander`         | Python   |
| `orion-approval-telegram` | Python   |
| `orion-memory`            | Python   |
| `orion-edge-agent`        | Go       |
| `orion-api`               | Python   |
| `orion-ui`                | TBD      |

### Autonomy levels
- **N0**: Observe only, no action
- **N1**: Suggest actions, human executes
- **N2**: Execute SAFE actions, suggest RISKY
- **N3**: Execute with approval flow for RISKY

Default state for any new behavior is **N0 (Observe only)**.
Promotion between levels requires explicit approval and evidence.
