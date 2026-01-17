# Testing Approach

## Framework

**pytest** (version 8.0.0) with plugins:
- `pytest-mock` (3.12.0) - Mocking utilities
- `pytest-cov` (4.1.0) - Code coverage
- `pytest-asyncio` (0.21.1) - Async support
- `fakeredis` (2.21.1) - In-memory Redis mock

**Configuration**: `pytest.ini`
- Test discovery: `test_*.py` and `*_test.py`
- Python path: Root directory
- Output: Verbose with short tracebacks
- Async mode: Auto

## Test Organization

**Directory Structure**:
```
tests/
├── conftest.py                    # Global fixtures
├── test_approval_phase4.py        # Approval system (28 tests)
├── test_brain.py                  # Brain N0 mode (24 tests)
├── test_brain_n2.py               # Brain N2 mode (27 tests)
├── test_bus.py                    # Event bus (18 tests)
├── test_circuit_breaker.py        # Failure protection (27 tests)
├── test_commander.py              # Action execution (15 tests)
├── test_contracts.py              # Schema validation (24 tests)
├── test_cooldown_tracker.py       # Rate limiting (16 tests)
├── test_guardian.py               # Event correlation (20 tests)
├── test_memory.py                 # Append-only storage (15 tests)
├── test_policies.py               # Policy consistency (17 tests)
└── test_policy_loader.py          # Policy loading (8 tests)
```

## Coverage by Module

| Module | Test File | Tests | Coverage |
|--------|-----------|-------|----------|
| Contracts | test_contracts.py | 24 | Complete |
| Event Bus | test_bus.py | 18 | Comprehensive |
| Brain (N0) | test_brain.py | 24 | Complete |
| Brain (N2) | test_brain_n2.py | 27 | Complete |
| Circuit Breaker | test_circuit_breaker.py | 27 | Comprehensive |
| Cooldown Tracker | test_cooldown_tracker.py | 16 | Comprehensive |
| Guardian | test_guardian.py | 20 | Comprehensive |
| Memory | test_memory.py | 15 | Comprehensive |
| Commander | test_commander.py | 15 | Comprehensive |
| Policies | test_policies.py | 17 | Comprehensive |
| Policy Loader | test_policy_loader.py | 8 | Complete |
| Approval Phase 4 | test_approval_phase4.py | 28 | Comprehensive |

**Total: 238 tests, all passing**

## Testing Patterns

### 1. Contract Validation Tests
**File**: `test_contracts.py` (24 tests)

Tests verify:
- Schema validity using `Draft202012Validator.check_schema()`
- Valid instance acceptance
- Required field enforcement
- Unknown field rejection (`additionalProperties: false`)
- Version validation
- Source validation (enforcing correct module origin)

**Classes**:
- `TestEventContract`: 5 tests
- `TestIncidentContract`: 4 tests
- `TestDecisionContract`: 4 tests
- `TestActionContract`: 3 tests
- `TestApprovalContract`: 3 tests
- `TestOutcomeContract`: 5 tests

### 2. Unit Tests
**Marked with @pytest.mark.unit** (166 tests)

**Core logic tests**:
- Brain decision making (N0, N2 modes)
- Circuit breaker state management
- Cooldown tracking and rate limiting
- Policy loading and classification
- Memory store append-only invariants
- Guardian incident detection

**Patterns**:
- Test classes group related scenarios
- Test names encode expected behavior
- Each test isolated with fresh fixtures
- Edge cases thoroughly covered

### 3. Policy Consistency Tests
**File**: `test_policies.py` (17 tests, @pytest.mark.policy)

Tests ensure:
- SAFE/RISKY classification completeness (no overlap)
- Required documentation fields present
- Conservative criteria met (blast radius, reversibility)
- Cooldown definitions for all RISKY actions
- Circuit breaker enabled
- Default autonomy level is N0
- Approval settings enforce deny-on-timeout
- Approvals are one-time use

### 4. Integration Tests
**Files**: test_bus.py, test_commander.py, test_brain_n2.py

**Patterns**:
- Use `fakeredis` for Redis Stream testing
- Test EventBus publish/read/subscribe flows
- Validate contract validation at boundaries
- Test module-to-module event flow
- Memory storage integration
- Decision → Action → Outcome chain

### 5. Safety-Critical Tests

**Brain N0 Enforcement** (test_brain.py):
- `test_never_suggests_action`: NO_ACTION always returned
- `test_never_requests_approval`: Approval never requested
- `test_decision_type_is_always_no_action`: All incidents return NO_ACTION

**Brain N2 Safety** (test_brain_n2.py):
- `test_never_executes_risky`: RISKY actions blocked
- `test_fail_closed_on_unknown`: Unknown actions blocked
- `test_all_actions_auditable`: Reasoning provided

**Commander Safety** (test_commander.py):
- `test_only_safe_actions_execute`: Validates SAFE classification
- `test_risky_never_executes`: Blocks RISKY actions
- `test_all_outcomes_auditable`: Outcomes logged

**Circuit Breaker** (test_circuit_breaker.py):
- State tracking: CLOSED, OPEN, timeout recovery
- Per-action type independence
- Failure window expiration
- Multiple open/close cycles

**Cooldown Tracker** (test_cooldown_tracker.py):
- Per-entity tracking
- Long cooldowns (86400s+)
- Zero/negative cooldown handling
- Rapid success/failure cycles

## Fixtures & Mocking

### Global Fixtures (`conftest.py`)
```python
@pytest.fixture
def valid_event_v1()         # Valid event contract v1.0
def valid_incident_v1()      # Valid incident contract v1.0
def valid_decision_v1()      # Valid decision contract v1.0
def valid_action_v1()        # Valid action contract v1.0
def valid_approval_v1()      # Valid approval contract v1.0
def valid_outcome_v1()       # Valid outcome contract v1.0
```

### Local Fixtures (per test file)
```python
@pytest.fixture
def redis_client()           # fakeredis.FakeRedis()
def event_bus()              # EventBus with test contracts
def brain()                  # Brain instance (N0, N2, or N3)
def guardian()               # Guardian with defaults
def commander()              # Commander with memory store
def memory_store()           # MemoryStore in temp directory
def policy_loader()          # PolicyLoader from test policies/
```

### Mocking Strategy
- **Redis**: Mocked with `fakeredis.FakeRedis(decode_responses=False)`
- **Time**: Real `time.sleep()` for cooldown/circuit breaker tests
- **File I/O**: Real files in `tmp_path` for memory store tests

## Safety Testing

**Safety-Critical Test Coverage**:

1. **Approval System** (28 tests):
   - Admin identity verification (Telegram, CLI)
   - Approval request handling and expiration
   - Silence is never permission
   - Expired approvals rejected
   - Only admins can approve

2. **Decision Logic**:
   - N0: Always NO_ACTION
   - N2: Only SAFE actions execute
   - N3: SAFE automatic, RISKY requires approval
   - All decisions include reasoning (min 10 chars)

3. **Action Execution**:
   - Safety classification enforced
   - Contract validation on boundaries
   - Cooldown enforcement
   - Circuit breaker prevents cascades
   - Outcomes immutably logged

4. **Policy Enforcement**:
   - No overlap between SAFE and RISKY
   - All RISKY actions require approval
   - Fail-closed on unknown actions

## Current Status

**Test Results** (Latest Run):
```
Total Tests: 238
Passed: 238 (100%)
Failed: 0
Skipped: 0
Runtime: ~15 seconds
```

**Test Breakdown by Marker**:
- Unit (@pytest.mark.unit): 166 tests
- Contract (@pytest.mark.contract): 24 tests
- Policy (@pytest.mark.policy): 17 tests
- Integration: Integration patterns in unit tests
- Slow: Circuit breaker and cooldown tests with time.sleep()

**Key Quality Metrics**:
- All safety-critical paths tested
- All contracts validated with schema tests
- All policies consistency-checked
- All modules have integration tests
- 100% of main code paths covered
- Edge cases documented in test names

## Notes

**Testing Philosophy** (from CLAUDE.md):
1. **Contracts are sacred**: All inputs validated against schemas
2. **Tests alongside code**: Features implemented with tests
3. **Safety-critical logic tested**: Decision logic, policies, cooldowns, circuit breaker
4. **No test, no action**: Code that triggers actions must be tested
5. **Test quality**: Names encode safety expectations

**Testing Requirements Met**:
- ✅ Contract validation (test_contracts.py)
- ✅ Unit testing for safety-critical paths
- ✅ Integration testing at module boundaries
- ✅ Fixtures for common contract instances
- ✅ Mocking of external dependencies
- ✅ Policy consistency validation
- ✅ Autonomy level enforcement (N0, N2, N3)
- ✅ Approval flow testing
- ✅ Cooldown and circuit breaker enforcement

**Running Tests**:
```bash
# All tests
pytest

# By marker
pytest -m unit                    # Fast unit tests
pytest -m integration             # With mocks
pytest -m contract                # Contract validation
pytest -m policy                  # Policy tests

# Specific module
pytest tests/test_brain.py -v

# With coverage
pytest --cov=core --cov-report=html
```

---

**Test Organization Pattern**:
Each test file follows:
1. Fixture setup (redis_client, event_bus, module instances)
2. Class-based test grouping by feature
3. Pytest markers (@pytest.mark.unit, @pytest.mark.contract)
4. Clear assertion messages
5. Edge case coverage
