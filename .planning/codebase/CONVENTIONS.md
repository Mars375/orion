# Coding Conventions

## Overview

ORION emphasizes **safety**, **SRE principles**, and **controlled intelligence**. All code follows strict conventions ensuring readability, auditability, and safety under autonomous execution.

**Language Mix**: Python (reasoning, policies, decisions) + Go (reliability, performance, services)

## Code Style

### Python
- **Style Guide**: PEP 8
- **Line length**: 79 chars for code, 99 for docs
- **Indentation**: 4 spaces (never tabs)
- **String quotes**: Double quotes (`"string"`)
- **Imports**: Three groups (stdlib, third-party, local) with blank lines between

**Example Import Organization**:
```python
import json
import logging
from datetime import datetime, timezone
from typing import Dict, Any, Optional

import redis
from jsonschema import ValidationError

from bus.python.orion_bus import EventBus
from .policy_loader import PolicyLoader
```

**Logging Standards**:
- Module-level logger: `logger = logging.getLogger(__name__)`
- Log levels: DEBUG (flow), INFO (events), WARNING (recoverable), ERROR (failures)
- Include context in all log statements
- Use `exc_info=True` for exceptions

### Go
- Use `gofmt` for formatting
- Exported: PascalCase
- Unexported: camelCase
- Constants: UPPER_SNAKE_CASE

## Naming Conventions

### Modules & Directories
**Pattern**: `orion-<module-name>` with hyphenated names
- `/core/brain/` → `orion-brain`
- `/core/guardian/` → `orion-guardian`
- `/core/approval/` → `orion-approval`

### Classes
**Pattern**: PascalCase for all classes
- `Brain`, `Guardian`, `ApprovalCoordinator`, `EventBus`, `CooldownTracker`, `CircuitBreaker`

### Functions & Methods
**Pattern**: snake_case for all functions
- Private: Prefix with underscore (`_calculate_fingerprint()`)
- Public: No prefix (`handle_incident()`, `decide()`)

### Variables & Constants
- Local variables: snake_case (`action_type`, `approval_timeout`)
- Instance variables: snake_case (`self.autonomy_level`)
- Constants: UPPER_SNAKE_CASE (`FAILURE_THRESHOLD = 3`)
- Protected data: Prefix with underscore (`self._event_buffer`)

**Special Rules**:
- Contract types omit `.schema.json` suffix (File: `decision.schema.json`, Code: `contract_type="decision"`)
- Autonomy levels: Uppercase (`N0`, `N2`, `N3`)

## Documentation Standards

### Module Docstrings
**Required for**: All modules

**Format**:
```python
"""
<One-line summary>.

<Detailed purpose and context if complex.>
"""
```

### Class Docstrings
**Required for**: All public classes

**Format**:
```python
class ClassName:
    """
    Brief description of class purpose.

    Longer explanation of responsibilities and key invariants.
    
    Invariants:
    - Specific guarantee 1
    - Specific guarantee 2
    """
```

### Function/Method Docstrings
**Required for**:
- Public functions and methods
- Anything that makes decisions, triggers actions, or changes state
- Safety-critical logic

**Format**: NumPy-style with Args, Returns, Raises

```python
def decide(self, incident: Dict[str, Any]) -> Dict[str, Any]:
    """
    Make decision about incident.

    Args:
        incident: Incident from guardian

    Returns:
        Decision contract

    Raises:
        ValidationError: If incident invalid
    """
```

### Comments
**Philosophy**: Comments explain WHY, never WHAT

**Allowed**:
- Safety guards and justification
- Threshold choices
- Why something is NOT done

**Forbidden**:
- Repeating code logic
- TODO without context
- FIXME without explanation

### README Standards
**Required for**: Every module

**Structure**:
1. Module name and language
2. Phase status
3. Purpose (1 paragraph max)
4. Inputs (Contracts Consumed)
5. Outputs (Contracts Emitted)
6. Invariants (MUST Always Hold)
7. Failure Modes
8. Explicit Non-Responsibilities

## Type Safety

### Type Hints
**Mandatory everywhere** - Python and Go

**Python Pattern**:
```python
from typing import Dict, Any, Optional, List, Callable

def handle_incident(self, incident: Dict[str, Any]) -> None:
    """Handle incoming incident."""

def decide(self, incident: Dict[str, Any]) -> Dict[str, Any]:
    """Make decision about incident."""
```

### Type Validation
**Required for**:
- Anything crossing module boundaries
- Input from: Redis Streams, MQTT, Telegram, external APIs

**Pattern**:
```python
from jsonschema import ValidationError

def validate(self, message: Dict[str, Any], schema_name: str) -> None:
    """Validate message against schema."""
    if schema_name not in self._validators:
        raise ValueError(f"Unknown schema: {schema_name}")
    
    validator = self._validators[schema_name]
    validator.validate(message)
```

## Error Handling

### Exception Patterns

**Pattern 1: Explicit Rejection**:
```python
if autonomy_level not in ("N0", "N2", "N3"):
    raise ValueError(f"Only N0, N2, and N3 modes supported, got: {autonomy_level}")
```

**Pattern 2: Validation Errors**:
```python
try:
    self.validator.validate(message, schema_name)
except ValidationError as e:
    raise ValidationError(f"Message doesn't match contract: {e}")
```

**Pattern 3: Safety-Critical Errors (Escalate)**:
```python
def _escalate_timeout(self, request: Dict[str, Any]) -> None:
    """
    Escalate timed-out approval request.
    NEVER executes action on timeout.
    """
    logger.error(
        f"ESCALATION: Approval request {request_id} timed out. "
        f"Action NOT executed. Human unavailable, system in safe inaction."
    )
```

## Module Structure

### Standard Module Layout
```
orion-brain/
├── __init__.py              # Public exports
├── brain.py                 # Main class
├── policy_loader.py         # Helpers
├── cooldown_tracker.py      # Helpers
├── circuit_breaker.py       # Helpers
└── README.md                # Module documentation
```

### Module `__init__.py` Pattern
```python
"""ORION Brain - Decision making."""

from .brain import Brain

__all__ = ["Brain"]
```

## Testing Doctrine

### Test File Organization
**Pattern**: `tests/test_<module>.py` mirrors `core/<module>/<module>.py`

### Test Markers
```python
@pytest.mark.unit              # Fast tests, no external deps
@pytest.mark.integration       # May use Redis/MQTT mocks
@pytest.mark.contract          # Schema compliance
@pytest.mark.policy            # SAFE/RISKY classification
@pytest.mark.slow              # Tests taking > 1 second
```

### Fixtures
**Shared fixtures in `tests/conftest.py`**:
```python
@pytest.fixture
def valid_incident_v1()         # Valid incident contract
def redis_client()              # fakeredis.FakeRedis()
def event_bus()                 # EventBus with test contracts
```

### Test Naming
**Format**: `test_<behavior>_<condition>`

**Examples**:
- `test_brain_always_decides_no_action_in_n0_mode`
- `test_risky_action_is_never_executed_without_approval`
- `test_cooldown_prevents_rapid_repeated_execution`

## Git Workflow

### Branch Naming
**Pattern**: `module/<module-name>`
- `module/orion-brain`
- `module/orion-guardian`
- `module/orion-approval`

**Rules**:
- One module per branch (strict)
- No cross-module changes

### Commits on Main (Squash Merge)
**Format**: Conventional Commits
```
<type>(<scope>): <subject>

<optional body>

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

**Types**: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

**Example**:
```
feat(brain): add N2 autonomy with policy enforcement

Implement decision-making for N2 with policy-based classification.
- Add PolicyLoader for SAFE/RISKY classification
- Add CooldownTracker to prevent rapid repeated execution
- Add CircuitBreaker to stop failure cascades

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

## Safety-Critical Code Patterns

### Fail Closed Pattern
```python
if classification == "UNKNOWN":
    # Unknown actions treated as RISKY (fail closed)
    return self._create_decision(
        decision_type="NO_ACTION",
        reasoning=f"Action {action_type} classification unknown. Treating as RISKY.",
        safety_classification="UNKNOWN",
    )
```

### Explicit Reasoning Pattern
```python
def _generate_reasoning_n0(self, incident: Dict[str, Any]) -> str:
    """Generate reasoning for NO_ACTION decision in N0 mode."""
    incident_type = incident.get("incident_type", "unknown")
    severity = incident.get("severity", "unknown")
    
    reasoning = (
        f"N0 mode (observe only): Detected {incident_type} "
        f"(severity={severity}). No action taken as per N0 policy."
    )
    
    return reasoning  # Minimum 10 characters enforced by contract
```

### Timeout and Escalation Pattern
```python
def check_expired_approvals(self) -> None:
    """Check for expired approval requests and escalate."""
    now = datetime.now(timezone.utc)
    expired = []
    
    for request_id, request in self.pending_approvals.items():
        expires_at = datetime.fromisoformat(request["expires_at"])
        if now >= expires_at:
            expired.append(request_id)
    
    for request_id in expired:
        request = self.pending_approvals[request_id]
        self._escalate_timeout(request)  # NEVER execute on timeout
        del self.pending_approvals[request_id]
```

## Common Mistakes to Avoid

1. **Implicit Defaults**: Always validate and raise errors
2. **Silent Failures**: Always log exceptions
3. **Missing Type Hints**: All functions must have type hints
4. **Unclear Reasoning**: Reasoning must be 10+ chars and specific
5. **Crossing Boundaries Without Validation**: Validate at module edges
6. **TODO Without Context**: Add phase/context to all TODOs

## Autonomy Levels in Code

### N0 Mode - Observe Only
```python
def _decide_n0(self, incident: Dict[str, Any]) -> Dict[str, Any]:
    """Make decision in N0 mode (always NO_ACTION)."""
    reasoning = self._generate_reasoning_n0(incident)
    
    return self._create_decision(
        incident=incident,
        decision_type="NO_ACTION",
        reasoning=reasoning,
        safety_classification="SAFE",
    )
```

### N2 Mode - SAFE Autonomous Execution
```python
def _decide_n2(self, incident: Dict[str, Any]) -> Dict[str, Any]:
    """Make decision in N2 mode (SAFE actions allowed)."""
    action_type = self._determine_action_type(incident)
    classification = self.policy_loader.classify_action(action_type)
    
    if classification == "RISKY":
        return self._create_decision(..., decision_type="NO_ACTION", ...)
    
    # Check cooldown and circuit breaker
    if not self.cooldown_tracker.check_cooldown(action_type, cooldown):
        return self._create_decision(..., decision_type="NO_ACTION", ...)
    
    # Execute SAFE action
    return decision
```

### N3 Mode - Human Authority
```python
def _decide_n3(self, incident: Dict[str, Any]) -> Dict[str, Any]:
    """Make decision in N3 mode (SAFE auto-execute, RISKY request approval)."""
    classification = self.policy_loader.classify_action(action_type)
    
    if classification == "RISKY":
        decision["decision_type"] = "REQUEST_APPROVAL"
        decision["requires_approval"] = True
        self._emit_approval_request(decision, incident)
        return decision
    
    # SAFE actions: check cooldown and circuit breaker (same as N2)
```

## Summary

ORION coding conventions prioritize:
1. **Safety** - Fail closed, explicit reasoning, validated inputs
2. **Clarity** - Readable by humans and auditors
3. **Auditability** - Complete logging, explicit approvals, versioned contracts
4. **Modularity** - Clear boundaries, explicit contracts, no shared state
5. **Testability** - All safety logic tested, mocked external dependencies

Follow CLAUDE.md as authoritative guide. When conventions conflict with safety, safety wins.
