# ORION Contracts

**Language-agnostic JSON Schema definitions**

## Purpose

Defines the strict, versioned contracts for all inter-module communication in ORION. Every message on the event bus MUST conform to a contract. Contracts are the single source of truth for message structure and validation.

## Contracts Defined

### Event (`event.schema.json`)
- **Emitted by**: Any ORION module
- **Purpose**: Observations of facts (service down, metric threshold exceeded, telemetry received)
- **No decisions**: Events are pure observations with no action attached
- **Required fields**: version, event_id, timestamp, source, event_type, severity, data

### Incident (`incident.schema.json`)
- **Emitted by**: `orion-guardian` ONLY
- **Purpose**: Correlated events representing a meaningful situation requiring attention
- **Correlation window**: Tracks time window during which events were correlated
- **Required fields**: version, incident_id, timestamp, source (must be "orion-guardian"), incident_type, severity, event_ids (minimum 1), correlation_window, state

### Decision (`decision.schema.json`)
- **Emitted by**: `orion-brain` ONLY
- **Purpose**: Reasoning outcome about how to respond to an incident
- **Decision types**: NO_ACTION, SUGGEST_ACTION, EXECUTE_SAFE_ACTION, REQUEST_APPROVAL
- **Explainability**: Every decision MUST include reasoning (minimum 10 characters)
- **Safety classification**: SAFE, RISKY, or UNKNOWN (UNKNOWN treated as RISKY)
- **Required fields**: version, decision_id, timestamp, source (must be "orion-brain"), incident_id, decision_type, safety_classification, requires_approval, reasoning

### Action (`action.schema.json`)
- **Emitted by**: `orion-brain` or `orion-approval-telegram`
- **Purpose**: Command to execute an action (MUST have passed safety checks before emission)
- **Safety invariant**: safety_classification can ONLY be SAFE or RISKY (never UNKNOWN)
- **States**: pending, executing, succeeded, failed, rolled_back
- **Required fields**: version, action_id, timestamp, source, decision_id, action_type, safety_classification, state, parameters

### Approval Request (`approval_request.schema.json`)
- **Emitted by**: `orion-brain` ONLY
- **Purpose**: Request for human approval of a RISKY action (N3 autonomy level)
- **Expiration**: All requests must include expiration timestamp (timeout)
- **Required fields**: version, approval_request_id, timestamp, source (must be "orion-brain"), decision_id, action_type, risk_level, requested_action, expires_at

### Approval Decision (`approval_decision.schema.json`)
- **Emitted by**: `orion-approval-telegram` or `orion-approval-cli`
- **Purpose**: Admin decision on approval request (approve/deny/force)
- **Admin identity**: Must include admin_identity for audit trail
- **Expiration**: All approvals are time-limited with expires_at
- **Mandatory reason**: Admin must provide reason for all decisions
- **Required fields**: version, approval_id, timestamp, source, approval_request_id, decision_id, decision, admin_identity, reason, issued_at, expires_at

### Approval (`approval.schema.json`) [DEPRECATED - Phase 3]
- **Status**: Deprecated in favor of approval_request + approval_decision
- **Migration**: Use approval_decision.schema.json for new implementations

### Outcome (`outcome.schema.json`)
- **Emitted by**: `orion-commander` ONLY
- **Purpose**: Result of an executed action (success, failure, or rollback)
- **Execution tracking**: Includes execution time in milliseconds
- **Error handling**: Failed outcomes MUST include error object with code and message
- **Required fields**: version, outcome_id, timestamp, source (must be "orion-commander"), action_id, status, execution_time_ms

## Contract Invariants (MUST Always Hold)

1. **Strict validation**: `additionalProperties: false` on all schemas (no unknown fields permitted)
2. **Versioning**: All contracts include version field (currently "1.0")
3. **UUID identifiers**: All entity IDs use UUID format
4. **ISO 8601 timestamps**: All timestamps use ISO 8601 date-time format
5. **Source enforcement**: Many contracts enforce specific source modules (e.g., incidents MUST come from orion-guardian)
6. **No implicit defaults**: All required fields must be explicitly provided
7. **Backward compatibility**: Schema changes MUST be backward compatible or require version increment

## Schema Validation Rules

- **Required fields**: Cannot be omitted or null
- **Enum fields**: Must exactly match one of the specified values
- **Format validation**: UUIDs and timestamps must match specified formats
- **Pattern validation**: Source fields use regex patterns to enforce module names
- **Minimum constraints**: Some fields have minimum length (e.g., reasoning >= 10 characters)
- **Array constraints**: Some arrays require minimum items (e.g., event_ids >= 1)

## Contract Evolution

### Adding a new field (backward compatible)
1. Add field as optional (not in `required` array)
2. Document default behavior if field is absent
3. Add tests verifying old messages still validate

### Changing required fields (breaking change)
1. Increment version (e.g., "1.0" → "2.0")
2. Update `$id` to include version
3. Support both versions during transition
4. Document migration path in ADR

### Adding a new contract
1. Create `<name>.schema.json` in this directory
2. Use JSON Schema draft 2020-12
3. Set `additionalProperties: false`
4. Include version field with const value
5. Add comprehensive test coverage in `tests/test_contracts.py`
6. Add fixture in `tests/conftest.py`
7. Document in this README

## Testing

All contracts are validated by:
- `tests/test_contracts.py` — Schema validity, validation rules, rejection of invalid messages
- Tests MUST cover: valid case, missing required field, unknown field, invalid version

## Failure Modes

- **Schema violation**: Message is rejected at bus publish time (fail fast)
- **Unknown schema**: Message is rejected (no implicit schema creation)
- **Version mismatch**: Message is rejected if version is not supported
- **Missing required field**: Message is rejected with validation error
- **Invalid format**: UUIDs, timestamps, enums validated and rejected if invalid

## Explicit Non-Responsibilities (What Contracts NEVER Do)

- **Never contain business logic**: Contracts are pure structure definitions
- **Never execute validation**: Validation is performed by orion-bus or module code, not by schemas themselves
- **Never transform data**: Contracts define structure, not transformations
- **Never store data**: Contracts are documentation and validation, not storage
- **Never imply behavior**: Behavior is defined by module code and policies, not contracts

---

## Philosophy

Contracts are law.
Modules are citizens.
Violations are rejected.
