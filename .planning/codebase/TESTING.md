# Testing Patterns

**Analysis Date:** 2026-01-13

## Test Framework

**Runner:**
- Not yet configured - No test files or framework detected
- Expected: pytest for Python, testing package for Go

**Assertion Library:**
- Not yet configured

**Run Commands:**
- Not yet defined - No package.json or test scripts exist

## Test File Organization

**Location:**
- Test-alongside doctrine per `CLAUDE.md` lines 108-110
- Tests written as part of implementation, not before or after
- Expected: Tests co-located with source files

**Naming:**
- Not yet defined - No test files exist
- Expected: Python `test_*.py`, Go `*_test.go`

**Structure:**
- Not yet implemented

## Test Structure

**Suite Organization:**
- Not yet implemented

**Patterns:**
- Test-alongside: Write tests during implementation per `CLAUDE.md` lines 108-110
- Test names encode safety expectations per `CLAUDE.md` lines 134-137
- Good example: `test_risky_action_is_never_executed_without_approval`
- Bad example: `assert foo(bar) == 3`

## Mocking

**Framework:**
- Expected: Mock Redis, MQTT, Telegram per `CLAUDE.md` line 119
- Never rely on live infrastructure per `CLAUDE.md` line 120

**Patterns:**
- Not yet implemented

**What to Mock:**
- Redis Streams (event bus)
- MQTT broker (edge telemetry)
- Telegram Bot API (approvals)
- External APIs

**What NOT to Mock:**
- Internal module logic
- Pure functions
- Policy evaluation logic (test actual policies)

## Fixtures and Factories

**Test Data:**
- Not yet implemented

**Location:**
- Expected: Co-located with tests per test-alongside doctrine

## Coverage

**Requirements:**
- No specific coverage target defined
- Focus on safety-critical logic per `CLAUDE.md` lines 123-130

**Critical Areas Requiring Tests:**
- Decision logic (brain)
- Policy evaluation (SAFE vs RISKY classification)
- Cooldown/circuit breaker logic
- Correlation windows (guardian)
- Approval state machines
- Rollback eligibility logic

**Configuration:**
- Not yet configured

## Test Types

**Unit Tests:**
- Scope: Safety-critical logic must be unit-tested per `CLAUDE.md` lines 123-130
- Mocking: Mock all external dependencies (Redis, MQTT, Telegram)
- Speed: Must be fast for rapid feedback

**Integration Tests:**
- Scope: Test bus ↔ module boundaries per `CLAUDE.md` lines 132-135
- Mocking: Mock Redis, MQTT, Telegram (no live infrastructure)
- Focus: Module-to-module communication via events

**Contract Tests:**
- Scope: Validate against JSON Schema contracts per `CLAUDE.md` lines 111-115
- Required tests: Valid contract, invalid contract, backward-compatible version
- Purpose: Ensure schema compliance at module boundaries
- Location: `bus/contracts/*.schema.json` (empty files, not yet defined)

**E2E Tests:**
- Not defined

## Common Patterns

**Contract Validation:**
- Validate all inputs against `orion-contracts` per `CLAUDE.md` line 111
- Reject invalid events explicitly per `CLAUDE.md` line 112
- Test backward-compatible versions per `CLAUDE.md` line 114

**Async Testing:**
- Not yet defined - No async code exists

**Error Testing:**
- Focus on safety-critical failure modes
- Test edge safety: Default to stop, network loss → safe mode

**Snapshot Testing:**
- Not defined

## Testing Doctrine

**Rule 1 - Contracts are sacred:**
- Validate all inputs against `orion-contracts`
- Reject invalid events explicitly
- Tests must include: valid contract, invalid contract, backward-compatible version

**Rule 2 - Tests alongside code:**
- Implement feature, immediately add tests
- Cover: normal path, failure path, edge cases
- Never leave logic untested "temporarily"

**Rule 3 - What must be unit-tested:**
- Decision logic (brain)
- Policy evaluation (SAFE vs RISKY)
- Cooldown/circuit breaker logic
- Correlation windows (guardian)
- Approval state machines
- Rollback eligibility logic

**Rule 4 - Integration tests at boundaries:**
- Test bus ↔ module boundaries
- Test module ↔ module boundaries via events
- Mock Redis, MQTT, Telegram
- Never rely on live infrastructure

**Rule 5 - No test, no action:**
- Code triggering actions, escalations, or state changes must be tested
- Manual testing, log inspection, or interactive validation does not count
- Untested code cannot ship per `CLAUDE.md` lines 138-141

## Test Quality Standards

**Test names encode safety expectations:**
- Good: `test_risky_action_is_never_executed_without_approval`
- Bad: `assert foo(bar) == 3`

**Tests must be readable:**
- Describe behavior clearly
- Test names should be self-documenting

**Safety-focused:**
- Tests verify safety invariants
- Tests cover failure modes
- Tests ensure SAFE/RISKY classification respected

## Current State

**No tests exist:**
- Zero test files detected in codebase
- No test framework configured
- No test infrastructure set up
- Test-alongside doctrine defined but not yet applied

**Next Steps:**
- Set up pytest for Python modules
- Set up Go testing for bus and edge agent
- Create contract test fixtures for JSON Schema validation
- Mock Redis Streams, MQTT, Telegram for integration tests

---

*Testing analysis: 2026-01-13*
*Update when test patterns change*
