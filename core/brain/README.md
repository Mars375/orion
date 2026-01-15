# orion-brain

**Language:** Python

## Purpose
Makes decisions about how to respond to incidents by applying safety policies, evaluating action classifications, and enforcing cooldowns. Brain is the reasoning module—it decides WHAT to do (or not do), but never executes actions itself.

## Inputs (Contracts Consumed)
- `incident.schema.json` — Incidents from orion-guardian requiring decision-making

## Outputs (Contracts Emitted)
- `decision.schema.json` — Decisions about incidents (NO_ACTION, SUGGEST_ACTION, EXECUTE_SAFE_ACTION, REQUEST_APPROVAL)

## Invariants (MUST Always Hold)
1. **Default to NO_ACTION**: If uncertain, brain decides NO_ACTION
2. **Safety over capability**: UNKNOWN actions are treated as RISKY, never executed
3. **Cooldowns enforced**: Brain MUST check cooldown state before proposing any action
4. **Policies are truth**: Brain applies policies from `policies/` as single source of truth
5. **Explainability**: Every decision MUST include reasoning field (minimum 10 characters)
6. **Autonomy level respected**: Brain only proposes actions permitted by current autonomy level (N0/N1/N2/N3)
7. **No side effects**: Brain never executes actions, only emits decisions

## Failure Modes
- **Policy load failure**: If policies cannot be loaded, brain defaults to NO_ACTION for all incidents
- **Invalid incident**: If incident fails schema validation, brain logs error and ignores incident
- **Cooldown state unavailable**: If cooldown state cannot be checked (Redis down), brain defaults to NO_ACTION
- **Unknown action type**: If proposed action is not in SAFE or RISKY lists, brain classifies as RISKY

## Explicit Non-Responsibilities (What Brain NEVER Does)
- **Never executes actions**: Brain only decides, orion-commander executes
- **Never correlates events**: Event correlation is orion-guardian's responsibility
- **Never approves actions**: Human approval comes from orion-approval-telegram
- **Never stores long-term memory**: Post-mortems and memory are orion-memory's responsibility
- **Never exposes HTTP APIs**: API exposure is orion-api's responsibility
- **Never interacts with hardware**: Edge control is orion-edge-agent's responsibility
