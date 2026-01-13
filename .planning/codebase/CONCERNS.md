# Codebase Concerns

**Analysis Date:** 2026-01-13

## Project Status

**Early-stage architectural project (Phase 0)** with **no implementation code**. The codebase consists of documentation and planning files with contracts and policies defined but empty.

---

## Tech Debt

**No implementation exists yet** - All modules are placeholder directories with empty READMEs.

---

## Known Bugs

**N/A** - No code to have bugs

---

## Security Considerations

**Policy files empty:**
- Risk: Safety classifications not defined, no SAFE/RISKY rules exist
- Files: `policies/actions_safe.yaml` (0 bytes), `policies/actions_risky.yaml` (0 bytes), `policies/approvals.yaml` (0 bytes), `policies/cooldowns.yaml` (0 bytes)
- Current mitigation: None - policies must be populated before any autonomous behavior
- Recommendations: Define explicit SAFE action allowlist, RISKY action restrictions, approval workflows before implementing brain or commander modules

**Contract schemas empty:**
- Risk: No validation rules defined, module boundaries not enforced
- Files: `bus/contracts/event.schema.json` (0 bytes), `bus/contracts/incident.schema.json` (0 bytes), `bus/contracts/decision.schema.json` (0 bytes), `bus/contracts/action.schema.json` (0 bytes), `bus/contracts/outcome.schema.json` (0 bytes)
- Current mitigation: None - CLAUDE.md Rule 1 states "Validate all inputs against orion-contracts" but contracts don't exist
- Recommendations: Define all event schemas before implementing any module communication

**Environment configuration missing:**
- Risk: Secrets management undefined, no documented required variables
- Files: `deploy/core/.env.example` (0 bytes), `deploy/hub/.env.example` (0 bytes)
- Current mitigation: `.env` files gitignored per `.gitignore`, but no documentation of what's needed
- Recommendations: Document all required environment variables (Redis, MQTT, Telegram, Tailscale credentials) with examples

**Robot safety underdefined:**
- Risk: Offline safety mechanisms not specified in detail
- File: `edge/freenove_hexapod/safety.md` (only 6 lines: "Default to stop, Loss of network → safe mode, No destructive commands without explicit approval")
- Current mitigation: Safety principles stated but not operationalized
- Recommendations: Define exact safe mode behavior (power down, hold position, limp), network loss timeout threshold, hardware safety limits (servo torque, battery voltage, tilt angles)

---

## Performance Bottlenecks

**N/A** - No implementation exists yet

---

## Fragile Areas

**Deployment configuration:**
- Why fragile: Docker compose files completely empty
- Files: `deploy/core/docker-compose.yml` (0 bytes), `deploy/hub/docker-compose.yml` (0 bytes)
- Common failures: Cannot deploy anything - no service definitions
- Safe modification: Define service structure with health checks, resource limits, restart policies
- Test coverage: None

**Module READMEs:**
- Why fragile: All empty stubs, no module specifications
- Files: `core/api/README.md` (0 bytes), `core/brain/README.md` (0 bytes), `core/commander/README.md` (0 bytes), `core/guardian/README.md` (0 bytes)
- Common failures: No implementation guidance, unclear module boundaries
- Safe modification: Populate READMEs with purpose, inputs/outputs, invariants, failure modes per CLAUDE.md requirements
- Test coverage: None

---

## Scaling Limits

**N/A** - No implementation to scale

---

## Dependencies at Risk

**No dependencies defined:**
- Risk: No version pinning, no dependency specifications
- Current state: Zero package.json, requirements.txt, go.mod, or Cargo.toml files
- Impact: Cannot install or run anything
- Migration plan: Create dependency specifications with version pinning before implementation begins

**Critical dependencies mentioned but not specified:**
- Redis Streams (version unknown, configuration unknown)
- MQTT broker (version unknown, authentication unknown)
- Telegram Bot API (version unknown, rate limits unknown)
- Tailscale (version unknown, ACL configuration unknown)
- Prometheus/Loki (versions unknown, retention policies unknown)

---

## Missing Critical Features

**All features missing** - Project is in documentation phase:

**Core modules not implemented:**
- Problem: No `orion-brain`, `orion-guardian`, `orion-commander`, `orion-api`, `orion-approval-telegram`, `orion-bus`, `orion-edge-agent` code
- Current workaround: N/A - nothing operational
- Blocks: Cannot deploy, test, or operate system
- Implementation complexity: High - requires Python/Go implementations following safety doctrine

**Policy definitions missing:**
- Problem: No SAFE/RISKY action classifications exist
- Files: `policies/actions_safe.yaml` (0 bytes), `policies/actions_risky.yaml` (0 bytes), `policies/approvals.yaml` (0 bytes), `policies/cooldowns.yaml` (0 bytes)
- Current workaround: N/A - no autonomous behavior possible
- Blocks: Cannot implement brain module (no policies to evaluate)
- Implementation complexity: Medium - requires careful safety analysis

**Contract schemas missing:**
- Problem: No event validation rules defined
- Files: All `bus/contracts/*.schema.json` files are 0 bytes
- Current workaround: N/A - no module communication possible
- Blocks: Cannot implement any module (no validated communication)
- Implementation complexity: Medium - requires JSON Schema definitions for 5 event types

**Runbooks incomplete:**
- Problem: Operational procedures listed but not documented
- File: `docs/RUNBOOKS.md` lists 5 runbooks (Restore access, Restore backups, Recover from disk full, Handle network split, Rebuild a node) with zero procedures
- Current workaround: N/A - no operational procedures exist
- Blocks: Cannot handle production incidents safely
- Implementation complexity: Low - requires documentation of procedures

**Phase documentation minimal:**
- Problem: Phase plans are 5-10 line outlines, not detailed implementation plans
- Files: `docs/PHASE_*.md` files are minimal stubs
- Current workaround: N/A - insufficient detail for implementation
- Blocks: Unclear acceptance criteria, implementation steps undefined
- Implementation complexity: Low - requires detailed planning documentation

---

## Test Coverage Gaps

**Zero test coverage:**
- What's not tested: Everything - no tests exist
- Risk: No safety validation, cannot verify behavior
- Priority: CRITICAL - CLAUDE.md states "No test, no action"
- Difficulty to test: Cannot test until implementation exists

**Missing test infrastructure:**
- What's not tested: No test framework, no fixtures, no mocks
- Risk: Cannot write or run tests when implementation begins
- Priority: HIGH - prerequisite for any module development
- Difficulty to test: Must set up pytest (Python), Go testing, contract validation fixtures

---

## Documentation Gaps

**Error handling strategy missing:**
- File: `CLAUDE.md` extensively documents safety philosophy but zero guidance on error handling
- Impact: Unclear how to handle failures in critical modules, recovery procedures undefined
- Recommendation: Document error handling for brain disagreements, corrupted state, timeout/retry strategies

**Dependency version specifications missing:**
- Impact: Cannot reproduce environment, unclear compatibility requirements
- Recommendation: Create requirements.txt (Python), go.mod (Go) with justified version pins

**CI/CD enforcement missing:**
- Impact: CLAUDE.md rules not enforced (conventional commits, branch naming, module isolation)
- Recommendation: Add pre-commit hooks, GitHub Actions workflows

---

## Summary

This is a **well-architected project at the planning stage** with:
- Strong safety principles defined in CLAUDE.md
- Clear module boundaries and event-driven architecture
- Zero enforcement mechanisms in place
- All implementation files empty or missing

**Critical path to first implementation:**
1. Populate policy files (`policies/*.yaml`)
2. Define contract schemas (`bus/contracts/*.json`)
3. Document environment variables (`.env.example` files)
4. Set up test infrastructure (pytest, Go testing, mocks)
5. Implement first module with tests following CLAUDE.md doctrine

No security vulnerabilities exist because no code exists. The architecture is sound; execution is the next phase.

---

*Concerns audit: 2026-01-13*
*Update as issues are fixed or new ones discovered*
