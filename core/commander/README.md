# orion-commander

**Language:** Python (initial implementation)

## Purpose
Orchestrates action execution, rollback, and state tracking. Commander is the only module that executes actions—it translates decisions into actual system changes while maintaining rollback capability and auditable state.

## Inputs (Contracts Consumed)
- `action.schema.json` — Approved actions from orion-brain or orion-approval-telegram

## Outputs (Contracts Emitted)
- `outcome.schema.json` — Results of action execution (success, failure, rolled back)

## Invariants (MUST Always Hold)
1. **Action validation**: Commander MUST validate action against schema before execution
2. **Safety check**: Commander MUST verify action has explicit safety_classification (SAFE or RISKY, never UNKNOWN)
3. **State tracking**: Commander tracks action state (pending → executing → succeeded/failed/rolled_back)
4. **Idempotency**: Executing same action_id multiple times MUST NOT cause duplicate execution
5. **Rollback integrity**: If rollback_enabled is true and action fails, commander MUST attempt rollback
6. **Dry-run support**: If dry_run is true, commander simulates action but does not execute
7. **No autonomous decisions**: Commander executes actions but never decides which actions to execute

## Failure Modes
- **Action execution failure**: If action fails, commander emits outcome with state=failed and attempts rollback if enabled
- **Rollback failure**: If rollback fails, commander emits outcome with state=failed and logs rollback error
- **Unknown action_type**: If action_type is not recognized, commander rejects action and emits failure outcome
- **External dependency unavailable**: If target system (e.g., Docker API) is unreachable, commander emits failure outcome
- **Timeout**: If action exceeds timeout, commander terminates execution and emits failure outcome

## Explicit Non-Responsibilities (What Commander NEVER Does)
- **Never decides actions**: Decision-making is orion-brain's responsibility
- **Never correlates events**: Event correlation is orion-guardian's responsibility
- **Never approves actions**: Approval is orion-approval-telegram's responsibility
- **Never classifies safety**: Safety classification is defined in policies, applied by orion-brain
- **Never stores long-term history**: Historical tracking is orion-memory's responsibility
- **Never exposes HTTP APIs**: API exposure is orion-api's responsibility
