# orion-commander

**Language:** Python (initial implementation)
**Phase 3 Status:** Implemented (SAFE actions only)

## Purpose
Orchestrates action execution, rollback, and state tracking. Commander is the only module that executes actions—it translates decisions into actual system changes while maintaining rollback capability and auditable state.

**Phase 3 Scope:** Executes SAFE actions (`acknowledge_incident`). RISKY actions are NEVER executed.

## Inputs (Contracts Consumed)
- `decision.schema.json` — Decisions from orion-brain (subscribes to decision stream)
- Executes only `EXECUTE_SAFE_ACTION` decisions
- Verifies action is SAFE via policy loader before execution

## Outputs (Contracts Emitted)
- `outcome.schema.json` — Results of action execution (succeeded, failed, rolled_back)

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

## Phase 3 Implementation

### Implemented Actions
- `acknowledge_incident`: Updates incident state in memory (idempotent, SAFE)

### Safety Enforcement
- Verifies every decision has `EXECUTE_SAFE_ACTION` type
- Checks proposed action is SAFE via PolicyLoader
- Rejects RISKY actions (logged and ignored)
- Rejects unknown action types (logged and failed)

### Execution Flow
1. Subscribe to `decision` stream
2. Filter for `EXECUTE_SAFE_ACTION` decisions
3. Verify proposed action is SAFE
4. Create action contract from decision
5. Execute action
6. Emit outcome to bus
7. Store outcome in memory (audit trail)

### Rollback
- All SAFE actions support rollback
- Rollback triggered automatically on failure
- Rollback for `acknowledge_incident` is logged (idempotent)

## Explicit Non-Responsibilities (What Commander NEVER Does)
- **Never decides actions**: Decision-making is orion-brain's responsibility
- **Never correlates events**: Event correlation is orion-guardian's responsibility
- **Never approves actions**: Approval is orion-approval-telegram's responsibility
- **Never classifies safety**: Safety classification is defined in policies, applied by orion-brain
- **Never stores long-term history**: Historical tracking is orion-memory's responsibility
- **Never exposes HTTP APIs**: API exposure is orion-api's responsibility
- **NEVER executes RISKY actions**: In N2 mode, RISKY actions must not execute
