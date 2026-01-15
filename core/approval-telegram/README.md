# orion-approval-telegram

**Language:** Python

## Purpose
Manages human approval workflow for RISKY actions via Telegram bot. Sends approval requests to authorized users, collects responses, and emits approval events to the bus. This is the only human-in-the-loop component for action approval.

## Inputs (Contracts Consumed)
- `decision.schema.json` — Decisions requiring approval (where requires_approval is true)

## Outputs (Contracts Emitted)
- `approval.schema.json` — Approval responses (approved, denied, expired)

## Invariants (MUST Always Hold)
1. **Authorization required**: Only authorized Telegram user IDs can approve actions
2. **Timeout enforcement**: Approval requests MUST expire after configured timeout
3. **One approval per decision**: Each decision_id can only be approved or denied once
4. **Explicit approval**: Approval MUST be explicit user action, never implicit or automated
5. **Audit trail**: All approval decisions MUST be logged with user ID and timestamp
6. **No auto-approval**: Module NEVER auto-approves, even if timeout occurs (timeout → denied/expired)

## Failure Modes
- **Telegram API unavailable**: If Telegram API is down, approval requests are queued and retried
- **Unauthorized user**: If unauthorized user attempts approval, request is rejected and logged
- **Timeout expiry**: If approval not received within timeout, decision expires and action is not executed
- **Network partition**: If network partitions, approval may be delayed but will not auto-approve

## Explicit Non-Responsibilities (What Approval-Telegram NEVER Does)
- **Never makes decisions**: Decision-making is orion-brain's responsibility
- **Never executes actions**: Action execution is orion-commander's responsibility
- **Never classifies safety**: Safety classification is defined in policies
- **Never auto-approves**: All approvals require explicit human action
- **Never modifies decisions**: Approval module only approves/denies, does not change decision content
- **Never bypasses approval**: No emergency bypass or auto-approval mechanism exists
