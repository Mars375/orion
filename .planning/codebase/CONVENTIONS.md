# Coding Conventions

**Analysis Date:** 2026-01-13

## Naming Patterns

**Files:**
- Module READMEs: Single file per module directory (e.g., `core/brain/README.md`)
- Markdown docs: kebab-case (e.g., `orion-brain.md`, `bus-and-contracts.md`)
- Important project files: UPPERCASE (e.g., `CLAUDE.md`, `README.md`, `LICENSE`)
- Schemas: `{event_type}.schema.json` (e.g., `event.schema.json`, `incident.schema.json`)
- Policies: `{category}_{classification}.yaml` (e.g., `actions_safe.yaml`, `actions_risky.yaml`)

**Functions:**
- Not yet implemented - No source code exists
- Expected: Python snake_case, Go camelCase per language conventions

**Variables:**
- Not yet implemented
- Expected: Python snake_case, Go camelCase

**Types:**
- Type hints mandatory everywhere (Python and Go) per `CLAUDE.md` lines 209-212
- Use dataclasses, TypedDict, pydantic for Python type definitions
- No untyped public functions allowed

**Modules:**
- Module names: kebab-case with `orion-` prefix (e.g., `orion-brain`, `orion-bus`, `orion-guardian`)
- Directory structure mirrors module names: `orion-brain` → `core/brain/`

## Code Style

**Formatting:**
- Not yet configured - No .prettierrc, .eslintrc, or similar config files
- Expected: Follow language defaults (PEP 8 for Python, gofmt for Go)

**Linting:**
- Not configured
- Expected: Python linters (pylint, flake8), Go linters (golint, staticcheck)

**Type Enforcement:**
- Type hints mandatory everywhere per `CLAUDE.md` lines 209-212
- Runtime validation required for module boundaries per `CLAUDE.md` lines 214-218

## Import Organization

**Order:**
- Not yet defined - No source code to analyze
- Expected: Standard library, third-party, local modules (Python), standard import grouping (Go)

**Grouping:**
- No cross-language imports allowed per `CLAUDE.md`
- All inter-module communication via Redis Streams events
- HTTP only for explicit human-facing APIs (never internal control flow)

**Path Aliases:**
- Not configured

## Error Handling

**Patterns:**
- Conservative by default: Inaction preferred to risky action per `CLAUDE.md`
- Explicit rules required: No automation without explicit rules, evidence, auditability
- Dry-run by default: New behavior starts as N0 (observe-only)

**Error Types:**
- Edge safety: Default to stop, network loss → safe mode per `edge/freenove_hexapod/safety.md`
- Approval expiration: Risky actions time-limited per `docs/SECURITY.md`

## Logging

**Framework:**
- Loki for log aggregation per `docs/ARCHITECTURE.md` (not configured)

**Patterns:**
- Audit trail mandatory for all decisions and actions per `CLAUDE.md`
- Full audit trail for approvals per `docs/SECURITY.md`

## Comments

**When to Comment:**
- Comments explain **WHY**, never **WHAT** per `CLAUDE.md` lines 204-207
- Example forbidden: `# increment counter` above `counter += 1`
- Allowed: Explaining safety guards, threshold choices, temporal windows, why something is NOT done

**Docstrings:**
- Required for: Public functions, classes, module entry points per `CLAUDE.md` lines 195-202
- Required for: Anything that makes decisions, triggers actions, changes state, or enforces safety
- Style: Short, precise, focus on what/why/constraints
- Not required for: Trivial getters, obvious transformations, private helpers with clear intent

**Module READMEs:**
- Every module must have README containing per `CLAUDE.md` lines 230-236:
  - Purpose (1 paragraph max)
  - Inputs / outputs
  - Invariants
  - Failure modes
  - What the module explicitly does NOT do

**ADRs (Architecture Decision Records):**
- Write only when per `CLAUDE.md` lines 224-228:
  - Changing a safety invariant
  - Changing autonomy level
  - Changing execution rules
  - Introducing architectural coupling
  - Changing how decisions are made
- Do not write for: refactoring, bug fixes, performance improvements, adding tests

## Function Design

**Size:**
- Keep code auditable and explainable per `CLAUDE.md`
- Code must be explainable to an auditor

**Parameters:**
- Not yet defined - No source code exists

**Return Values:**
- Not yet defined

## Module Design

**Exports:**
- No cross-language imports per `CLAUDE.md`
- No shared memory, volumes, or implicit state between modules
- All communication via Redis Streams events with JSON Schema validation

**Barrel Files:**
- Not applicable - Modules communicate via event bus, not imports

**Module Boundaries:**
- Validated against `bus/contracts/*.json` schemas per `CLAUDE.md` lines 111-115
- Mandatory runtime validation for: Redis Streams, MQTT, Telegram, external APIs
- Type hints insufficient - external input must be validated

## Git Workflow

**Branch Naming:**
- Module-scoped branches: `module/<module-name>` per `CLAUDE.md` lines 155-157
- Examples: `module/orion-brain`, `module/orion-bus`, `module/orion-guardian`
- Forbidden: Touching multiple modules in one branch

**Commits on Branches:**
- Commit freely with descriptive messages during development
- No strict format required on feature branches

**Commits on Main:**
- All merges are squash merges per `CLAUDE.md` lines 167-172
- Format: `<type>(<module>): <summary>`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
- Examples from history:
  - `docs(governance): add CLAUDE.md development contract`
  - `chore: initial ORION architecture, PRD, ADR, and repository layout`

**Merge Requirements:**
- All tests pass locally
- Contract validations pass
- Module README updated if behavior changed
- No TODOs remain in safety-critical paths

## Safety & Governance

**Three Pillars (CLAUDE.md lines 41-69):**
1. **Conservative by default**: When uncertain, choose safer option; capability secondary to safety
2. **Ask first**: Before autonomy changes, safety classification changes, scope expansion
3. **Dry-run by default**: New behavior starts N0 (observe-only); progression: Observe → Dry-run → Restricted → Full

**Autonomy Levels:**
- **N0**: Observe only, no action (default for all new behavior)
- **N1**: Suggest actions, human executes
- **N2**: Execute SAFE actions, suggest RISKY
- **N3**: Execute with approval flow for RISKY
- Promotion requires explicit approval and evidence

**Testing Doctrine (CLAUDE.md lines 105-141):**
- Test-alongside: Write tests as part of implementation, not before or after
- No test, no ship: Untested code cannot trigger actions
- Manual testing doesn't count toward shipping tests
- Contract validation required: Valid contract, invalid contract, backward-compatible version

**What Claude Must Never Do:**
- Remove or weaken guardrails without explicit ADR
- Auto-enable behavior because "it seems safe"
- Silently expand scope
- Optimize away approvals
- Treat uncertainty as acceptable risk

---

*Convention analysis: 2026-01-13*
*Update when patterns change*
