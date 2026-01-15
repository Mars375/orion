"""
Pytest configuration for ORION test suite.

Fixtures and configuration shared across all tests.
"""

import pytest


@pytest.fixture
def valid_event_v1():
    """Returns a valid event matching event.schema.json v1.0."""
    return {
        "version": "1.0",
        "event_id": "550e8400-e29b-41d4-a716-446655440000",
        "timestamp": "2026-01-14T12:00:00Z",
        "source": "orion-guardian",
        "event_type": "service_down",
        "severity": "error",
        "data": {
            "service_name": "test-service"
        }
    }


@pytest.fixture
def valid_incident_v1():
    """Returns a valid incident matching incident.schema.json v1.0."""
    return {
        "version": "1.0",
        "incident_id": "650e8400-e29b-41d4-a716-446655440001",
        "timestamp": "2026-01-14T12:01:00Z",
        "source": "orion-guardian",
        "incident_type": "service_outage",
        "severity": "high",
        "event_ids": ["550e8400-e29b-41d4-a716-446655440000"],
        "correlation_window": {
            "start": "2026-01-14T12:00:00Z",
            "end": "2026-01-14T12:01:00Z"
        },
        "state": "open"
    }


@pytest.fixture
def valid_decision_v1():
    """Returns a valid decision matching decision.schema.json v1.0."""
    return {
        "version": "1.0",
        "decision_id": "750e8400-e29b-41d4-a716-446655440002",
        "timestamp": "2026-01-14T12:02:00Z",
        "source": "orion-brain",
        "incident_id": "650e8400-e29b-41d4-a716-446655440001",
        "decision_type": "NO_ACTION",
        "safety_classification": "SAFE",
        "requires_approval": False,
        "reasoning": "Service down detected but within acceptable downtime window"
    }


@pytest.fixture
def valid_action_v1():
    """Returns a valid action matching action.schema.json v1.0."""
    return {
        "version": "1.0",
        "action_id": "850e8400-e29b-41d4-a716-446655440003",
        "timestamp": "2026-01-14T12:03:00Z",
        "source": "orion-brain",
        "decision_id": "750e8400-e29b-41d4-a716-446655440002",
        "action_type": "send_notification",
        "safety_classification": "SAFE",
        "state": "pending",
        "parameters": {
            "message": "Service test-service is down"
        }
    }


@pytest.fixture
def valid_approval_v1():
    """Returns a valid approval matching approval.schema.json v1.0."""
    return {
        "version": "1.0",
        "approval_id": "950e8400-e29b-41d4-a716-446655440004",
        "timestamp": "2026-01-14T12:04:00Z",
        "source": "orion-approval-telegram",
        "decision_id": "750e8400-e29b-41d4-a716-446655440002",
        "status": "approved",
        "approved_by": "user_12345"
    }


@pytest.fixture
def valid_outcome_v1():
    """Returns a valid outcome matching outcome.schema.json v1.0."""
    return {
        "version": "1.0",
        "outcome_id": "a50e8400-e29b-41d4-a716-446655440005",
        "timestamp": "2026-01-14T12:05:00Z",
        "source": "orion-commander",
        "action_id": "850e8400-e29b-41d4-a716-446655440003",
        "status": "succeeded",
        "execution_time_ms": 1250
    }
