# ORION Approval System

**Phase 4: Human Authority & Approvals**

## Purpose

Implements strict, auditable, single-admin human approval system for RISKY actions.

Enables **N3 autonomy level**:
- SAFE actions execute automatically (like N2)
- RISKY actions require explicit ADMIN approval
- Silence or timeout is NEVER permission
- All approvals are time-limited and audited

## Components

### AdminIdentity (`admin_identity.py`)

Validates admin identity for approval decisions.

**Invariants:**
- Exactly ONE admin identity per system
- Identity must be explicitly configured
- Unknown identity = rejected
- No implicit defaults, no delegation, no quorum

**Configuration:**
- Requires `config/admin.yaml` with single ADMIN identity
- Supports two channels: Telegram (chat ID) and CLI (username/UID)
- At least one channel must be configured

**Methods:**
- `verify_telegram(chat_id)` - Verify Telegram identity matches ADMIN
- `verify_cli(identity)` - Verify CLI identity matches ADMIN

### ApprovalCoordinator (`approval_coordinator.py`)

Central approval request tracking and timeout handling.

**Responsibilities:**
1. Receives approval requests from brain
2. Tracks pending approvals with expiration
3. Validates admin decisions (approve/deny/force)
4. Emits approval_decision contracts
5. Handles timeouts (escalate, never execute)

**Invariants:**
- Silence is NEVER permission
- Timeout = escalation, never execution
- All approvals expire (time-limited)
- Expired approval = invalid approval
- Only ADMIN can approve

**Methods:**
- `handle_approval_request(request)` - Track new approval request
- `approve(...)` - ADMIN approves RISKY action
- `deny(...)` - ADMIN denies RISKY action
- `force(...)` - ADMIN forces action with optional override flags
- `check_expired_approvals()` - Escalate expired requests

## Inputs (Contracts Consumed)

### approval_request.schema.json
- **Emitted by**: `orion-brain` (N3 mode)
- **Purpose**: Request approval for RISKY action
- **Fields**: approval_request_id, decision_id, action_type, risk_level, requested_action, expires_at

## Outputs (Contracts Emitted)

### approval_decision.schema.json
- **Emitted by**: `orion-approval-telegram` or `orion-approval-cli`
- **Purpose**: Admin decision on approval request
- **Fields**: approval_id, decision (approve/deny/force), admin_identity, reason (mandatory), expires_at
- **Override flags**: override_circuit_breaker, override_cooldown (force only)

## Approval Flow (N3 Mode)

1. **Brain** detects incident requiring RISKY action
2. **Brain** emits `decision` (REQUEST_APPROVAL) + `approval_request`
3. **Approval Coordinator** receives request, tracks expiration
4. **Approval Channel** (Telegram/CLI) presents request to ADMIN
5. **ADMIN** makes decision: approve, deny, or force
6. **Approval Coordinator** validates identity and emits `approval_decision`
7. **Commander** receives approval, validates expiration, executes action
8. **Commander** emits `outcome` for audit trail

## Approval Decisions

### APPROVE
- Allow specific RISKY action
- Approval is time-limited (expires_at)
- One-time use (consumed after execution)
- ADMIN identity required
- Mandatory reason for audit trail

### DENY
- Permanently cancel action
- Does not expire (denial is final)
- ADMIN identity required
- Mandatory reason for audit trail

### FORCE
- Allow execution even if safety checks fail
- Can override circuit breaker
- Can override cooldown
- Approval is time-limited
- Requires strong reason (>= 10 characters)
- Highest level of override

## Timeout & Escalation

**If approval request times out:**
1. Log escalation (ERROR level)
2. Notify ADMIN (future: via configured channels)
3. Action is NOT executed
4. Remove from pending approvals

**Timeout behavior:**
- Repeated timeouts do NOT increase autonomy
- Timeout is treated as: human unavailable, system in safe inaction
- NEVER infer permission from silence

## Expiration Rules

**All approvals MUST expire:**
- Default: 300 seconds (5 minutes)
- Maximum: 3600 seconds (1 hour)
- Configured per action type in `policies/approvals.yaml`

**Expiration enforcement:**
- Coordinator checks expiration before emitting decision
- Commander validates expiration before execution
- Expired approval = rejection
- Automatic cleanup of expired approvals

**If override expires during execution:**
- Execution must stop or roll back safely
- No continuation without renewed approval

## Audit Trail

**All approval activity is audited:**
- Approval requests
- Approval decisions (approve/deny/force)
- Overrides (circuit breaker, cooldown)
- Expirations
- Forced executions
- Timeout escalations

**Audit trail allows reconstructing:**
```
incident → decision → approval → action → outcome
```

## Invariants (MUST Always Hold)

1. **Only ADMIN can approve** - Single admin identity, strictly validated
2. **Silence is never permission** - No response = NO_ACTION
3. **Expired approval = invalid** - Expiration always enforced
4. **Unknown human = no authority** - Identity must match ADMIN
5. **Unknown action = NO_ACTION** - Fail closed on uncertainty
6. **Policy always overrides heuristics** - Policies are source of truth
7. **Safety > availability** - Prefer inaction over risky action
8. **Inaction > risky action** - When uncertain, do nothing

## Failure Modes

### Admin identity mismatch
- **Symptom**: Approval rejected
- **Cause**: Identity doesn't match configured ADMIN
- **Resolution**: Verify admin.yaml configuration

### Approval expired
- **Symptom**: Action not executed despite approval
- **Cause**: Approval timeout elapsed before execution
- **Resolution**: Approve again with fresh timeout

### Approval timeout
- **Symptom**: No action taken, escalation logged
- **Cause**: ADMIN did not respond within timeout
- **Resolution**: Check ADMIN availability, review escalation

### Circuit breaker override expires
- **Symptom**: Execution stopped mid-action
- **Cause**: Force approval with override expired
- **Resolution**: Renew approval if execution should continue

## Explicit Non-Responsibilities

**This module does NOT:**
- Execute actions (handled by commander)
- Make decisions (handled by brain)
- Store long-term state (approvals are ephemeral)
- Implement notification channels (delegated to submodules)
- Allow multi-admin or delegation
- Allow approval reuse (one-time use only)
- Infer approval from silence or timeout

## Configuration

### config/admin.yaml (required)
```yaml
version: "1.0"
admin:
  telegram_chat_id: "YOUR_CHAT_ID"  # Optional
  cli_identity: "YOUR_USERNAME"     # Optional
  # At least one must be configured
```

**Security:**
- Keep `admin.yaml` secure
- Add to `.gitignore`
- Use `admin.yaml.example` as template

### policies/approvals.yaml
```yaml
approval_settings:
  default_timeout: "300s"
  max_timeout: "3600s"
  timeout_behavior: "deny"

action_overrides:
  - action_type: "restart_service"
    timeout: "300s"
```

## Testing

**Test coverage:** 31 tests (all passing)

**Areas tested:**
- Admin identity verification (11 tests)
- Approval coordinator (9 tests)
- Brain N3 mode (4 tests)
- Commander approval validation (3 tests)
- Invariants (4 tests)

**Run tests:**
```bash
pytest tests/test_approval_phase4.py -v
```

## Future Work (Post-Phase 4)

- Telegram bot integration (orion-approval-telegram)
- CLI approval interface (orion-approval-cli)
- Admin notification on timeout
- Approval request queuing
- Multi-step approval workflows (if ever needed)

---

**Remember:** ORION approves nothing in silence. Only explicit ADMIN approval permits RISKY actions.
