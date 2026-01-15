# orion-api

**Language:** Python

## Purpose
Exposes HTTP inspection endpoints for human operators to query ORION state, view incidents, decisions, and actions. API is read-only and does not accept control commands—it is for observation and debugging only.

## Inputs (Contracts Consumed)
- None (API queries Redis/state directly, does not consume bus events)

## Outputs (Contracts Emitted)
- None (API returns HTTP responses, does not emit bus events)

## Invariants (MUST Always Hold)
1. **Read-only**: API MUST NOT accept write operations or control commands
2. **No side effects**: API queries MUST NOT trigger actions, decisions, or state changes
3. **Authentication required**: All endpoints MUST require authentication (no public access)
4. **Rate limiting**: API MUST enforce rate limits to prevent abuse
5. **CORS disabled**: API is internal-only, no cross-origin access permitted
6. **No direct control flow**: API never emits bus events or triggers ORION behavior

## Failure Modes
- **Redis unavailable**: If state backend is down, API returns 503 Service Unavailable
- **Invalid query parameters**: API returns 400 Bad Request with validation errors
- **Authentication failure**: API returns 401 Unauthorized
- **Rate limit exceeded**: API returns 429 Too Many Requests

## Explicit Non-Responsibilities (What API NEVER Does)
- **Never executes actions**: Action execution is orion-commander's responsibility
- **Never makes decisions**: Decision-making is orion-brain's responsibility
- **Never emits bus events**: API is read-only, does not produce events
- **Never controls ORION behavior**: API is for inspection only, not control
- **Never stores state**: API reads state but does not modify it
- **Never approves actions**: Approval is orion-approval-telegram's responsibility
