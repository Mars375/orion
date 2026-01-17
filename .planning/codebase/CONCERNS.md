# Concerns & Risks

## Safety Concerns

### Critical - In-Memory State Loss (Guardian)
- Guardian maintains in-memory correlation state (`_event_buffer`, `_incident_fingerprints`)
- NOT persisted to Redis or disk
- If Guardian crashes, all correlation history is lost
- Documented as Phase 1 limitation: "Guardian state not persisted"
- **Risk**: Correlated incidents could be duplicated if Guardian restarts
- **Impact**: False incidents violate "no escalation beyond observed data" invariant

### Critical - In-Memory Approval Tracking (ApprovalCoordinator)
- Pending approvals tracked in `self.pending_approvals` (in-memory Dict)
- If coordinator crashes, all pending RISKY action requests are lost
- No persistence to Redis or disk
- **Risk**: RISKY action could execute without fresh approval after restart
- **Impact**: "Silence is never permission" invariant could be violated

### Critical - In-Memory Safety State (CircuitBreaker, CooldownTracker)
- Both use in-memory state (`_failures`, `_circuit_opened`, `_last_execution`)
- No persistence across restarts
- **Risk**: After crash, safety mechanisms reset
- **Impact**: Runaway action execution if service fails and restarts

### High - Type Hint Bug (CircuitBreaker)
- Line 117 in `circuit_breaker.py`: `def get_state(self, action_type: str) -> Dict[str, any]:`
- Uses lowercase `any` instead of `Any` from typing module
- Runtime error if type checking is enforced
- **Impact**: Type validation will fail

### High - Admin Notification Not Implemented
- Line 438 in `approval_coordinator.py`: `# TODO: Notify admin via configured channels`
- Approval timeout logs escalation but doesn't notify ADMIN
- **Risk**: ADMIN unaware that RISKY action is waiting
- **Impact**: Approval timeouts could be silent failures

### High - Expiration During Execution Risk
- Commander validates approval BEFORE execution but not DURING long-running actions
- If action takes longer than approval window, execution continues with expired approval
- Documented but not enforced in code
- **Impact**: Could violate approval time-limit invariant

## Technical Debt

### Medium - Guardian State Recovery Not Designed
- No mechanism to recover lost correlation state from Redis
- Cannot replay events or rebuild deduplication state
- Would need to scan entire event history (unbounded work)
- **Impact**: Unplanned incident duplication on restart

### Medium - No Distributed State Options
- All safety-critical state is single-instance in-memory
- No Redis-backed state management
- Documented: "Single-node ORION: Core services not distributed"
- **Impact**: Future blocker for high-availability

### Medium - Hardcoded Defaults
- Default approval timeout: 300 seconds (hardcoded)
- CircuitBreaker: 3 failures in 300s, 600s open duration
- Cooldown values from policies only (no programmatic fallback)
- **Risk**: Hard to customize for different scenarios

### Medium - Error Handling Patterns
- Commander raises ValueError for unknown action types
- Most modules catch exceptions and log but don't propagate
- No structured error response contracts
- **Impact**: Failure diagnosis requires log inspection

### Low - Guardian Event Buffer Naive Deduplication
- Deduplication uses 16-character SHA256 prefix
- Stores all event_ids but deduplication key is limited
- Very low collision risk but theoretically possible
- **Impact**: Duplicates from different causes could be masked

## Known Limitations

### By Design (Documented)
- N0 only by default (autonomy must be explicitly enabled)
- No hardware control (edge devices not integrated)
- Manual deployment (no auto-deploy)
- No UI (CLI and file-based only)
- No autonomous remediation at any autonomy level

### Technical Limitations (Documented)
- Guardian correlation not persisted (Phase 1, not fixed)
- No metrics polling (watcher exists, not integrated)
- No log tailing (system metrics only)
- Single-node ORION (not distributed)
- Approval channels (Telegram/CLI) not implemented

## TODOs in Code

### Critical - In Approval System
- `approval_coordinator.py:438`: "TODO: Notify admin via configured channels"
  - Affects timeout escalation
  - Admin won't be alerted if approval expires
  - **Status**: Blocks full N3 deployment safety

## Future Refactoring Needs

### High Priority
1. **Persist safety state to Redis**:
   - Guardian correlation state
   - Approval coordinator pending requests
   - CircuitBreaker failure history
   - CooldownTracker execution timestamps

2. **Implement admin notification**:
   - Telegram approval notifications
   - CLI approval interface
   - Timeout escalation alerts
   - Needed for unattended N3 operation

3. **Add distributed state management**:
   - Redis-backed state for Guardian
   - Distributed approval coordinator
   - Shared circuit breaker across instances

### Medium Priority
1. **Fix type hints**: Use `Any` instead of `any` in CircuitBreaker

2. **Improve error handling**:
   - Structured error contracts
   - Better error propagation
   - Fallback behaviors

3. **Persistence recovery**:
   - Event replay for Guardian reconstruction
   - Approval state restoration from audit trail
   - State validation on startup

## Security Considerations

### High - In-Memory State Exposure
- Pending approvals in memory are not encrypted
- Cooldown/CircuitBreaker state accessible if process memory compromised
- **Risk**: Attacker could clear state to bypass safety
- **Mitigation**: Run in secure container with isolation

### Medium - Admin Identity Storage
- Admin identity loaded from `config/admin.yaml`
- Should be in .gitignore but could be accidentally committed
- **Risk**: Credentials in git history
- **Mitigation**: Git hooks + .gitignore enforcement

### Medium - No Input Validation Contracts
- API README claims "authentication required" but no implementation
- No validation that approval decisions come from verified ADMIN in all paths
- **Risk**: Unauthorized approvals if channels integrated without identity verification
- **Mitigation**: Telegram/CLI modules must validate identity

### Low - Policy File Permissions
- Policy files not documented for permission requirements
- Sensitive approval timeouts in readable YAML
- **Risk**: World-readable policies expose classifications
- **Mitigation**: Set permissions to 644 or 600

## Performance Concerns

### Medium - Guardian Event Buffer Unbounded Growth
- Buffer limited to 100 events (reasonable)
- `_incident_fingerprints` Dict grows without bound
- After weeks/months, memory usage could be significant
- **Risk**: Memory leak in long-running deployments
- **Mitigation**: Implement fingerprint TTL or circular buffer

### Low - Cooldown/CircuitBreaker Memory Usage
- Both use `time.time()` for timestamps (float storage)
- CooldownTracker stores per (action_type, applies_per_key) tuple
- Unlikely bottleneck but Dict could grow
- **Impact**: Minimal for current use cases

## Scalability Issues

### High - Single Redis Instance
- All modules depend on single Redis
- No clustering or replication configured
- Documented as "Single-node ORION"
- **Risk**: Redis failure = complete system failure
- **Impact**: Not suitable for critical deployments

### Medium - In-Memory Correlation
- Guardian can't scale to high event rates
- Buffer limited to 100 events (hard limit)
- Correlation windows fixed at 60 seconds
- **Risk**: Events could be dropped at high rates
- **Impact**: Incident detection gaps possible

### Medium - No Async/Concurrent Processing
- Code uses synchronous I/O
- No thread pools or async/await patterns
- Single stream consumer per module
- **Risk**: Processing latency could block event stream

## Testing & Quality Concerns

### Good Coverage (238 tests passing)
- Contract validation (24 tests)
- Policy consistency (17 tests)
- Safety-critical paths well tested
- Guardian, Brain, Commander comprehensive

### Gaps
- Integration tests use mocked Redis (fakeredis)
- No tests for state persistence/recovery
- No tests for crashed service restart scenarios
- No concurrent operation tests
- Error handling could have more edge cases

## Notes & Context

### Risk Tolerance
- ORION designed for homelab SRE (not critical infrastructure)
- Conservative: Prefer inaction over risky action
- Current N0 mode is safe (no autonomous actions)
- N2/N3 modes add execution risk mitigated by policies + approvals

### Priority Ordering
1. **Immediate**: Fix type hints, document notification implementation
2. **Phase 5+**: Implement persistence when distributed deployment planned
3. **Enhancement**: Improve error handling and state recovery
4. **Future**: Address scalability for larger deployments

### Blocking Issues
- None for N0 (current mode)
- Admin notification TODO blocks full N3 safety for unattended operation
- None preventing Phase 5 planning

---

**Document describes concerns as of current codebase state. For implementation priority, review with CLAUDE.md's directive: "Ask first" before addressing safety-critical changes.**
