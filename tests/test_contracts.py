"""
Contract validation tests.

Validates that all JSON schemas:
1. Are valid JSON Schema documents
2. Accept valid contract instances
3. Reject invalid contract instances
4. Enforce additionalProperties: false (no unknown fields)
"""

import json
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator, ValidationError


CONTRACTS_DIR = Path(__file__).parent.parent / "bus" / "contracts"


def load_schema(name):
    """Load a JSON schema from bus/contracts/."""
    schema_path = CONTRACTS_DIR / name
    with open(schema_path) as f:
        return json.load(f)


@pytest.mark.contract
class TestEventContract:
    """Test event.schema.json validation."""

    def test_schema_is_valid(self):
        """event.schema.json is a valid JSON Schema."""
        schema = load_schema("event.schema.json")
        Draft202012Validator.check_schema(schema)

    def test_valid_event_accepted(self, valid_event_v1):
        """Valid event passes validation."""
        schema = load_schema("event.schema.json")
        validator = Draft202012Validator(schema)
        validator.validate(valid_event_v1)  # Raises if invalid

    def test_missing_required_field_rejected(self, valid_event_v1):
        """Event missing required field is rejected."""
        schema = load_schema("event.schema.json")
        validator = Draft202012Validator(schema)

        # Remove required field
        del valid_event_v1["event_id"]

        with pytest.raises(ValidationError):
            validator.validate(valid_event_v1)

    def test_unknown_field_rejected(self, valid_event_v1):
        """Event with unknown field is rejected (additionalProperties: false)."""
        schema = load_schema("event.schema.json")
        validator = Draft202012Validator(schema)

        # Add unknown field
        valid_event_v1["unknown_field"] = "should_fail"

        with pytest.raises(ValidationError):
            validator.validate(valid_event_v1)

    def test_invalid_version_rejected(self, valid_event_v1):
        """Event with wrong version is rejected."""
        schema = load_schema("event.schema.json")
        validator = Draft202012Validator(schema)

        valid_event_v1["version"] = "2.0"  # Only 1.0 is valid

        with pytest.raises(ValidationError):
            validator.validate(valid_event_v1)


@pytest.mark.contract
class TestIncidentContract:
    """Test incident.schema.json validation."""

    def test_schema_is_valid(self):
        """incident.schema.json is a valid JSON Schema."""
        schema = load_schema("incident.schema.json")
        Draft202012Validator.check_schema(schema)

    def test_valid_incident_accepted(self, valid_incident_v1):
        """Valid incident passes validation."""
        schema = load_schema("incident.schema.json")
        validator = Draft202012Validator(schema)
        validator.validate(valid_incident_v1)

    def test_source_must_be_guardian(self, valid_incident_v1):
        """Incident source must be orion-guardian."""
        schema = load_schema("incident.schema.json")
        validator = Draft202012Validator(schema)

        valid_incident_v1["source"] = "orion-brain"  # Wrong source

        with pytest.raises(ValidationError):
            validator.validate(valid_incident_v1)

    def test_event_ids_not_empty(self, valid_incident_v1):
        """Incident must have at least one event_id."""
        schema = load_schema("incident.schema.json")
        validator = Draft202012Validator(schema)

        valid_incident_v1["event_ids"] = []  # Empty not allowed

        with pytest.raises(ValidationError):
            validator.validate(valid_incident_v1)


@pytest.mark.contract
class TestDecisionContract:
    """Test decision.schema.json validation."""

    def test_schema_is_valid(self):
        """decision.schema.json is a valid JSON Schema."""
        schema = load_schema("decision.schema.json")
        Draft202012Validator.check_schema(schema)

    def test_valid_decision_accepted(self, valid_decision_v1):
        """Valid decision passes validation."""
        schema = load_schema("decision.schema.json")
        validator = Draft202012Validator(schema)
        validator.validate(valid_decision_v1)

    def test_source_must_be_brain(self, valid_decision_v1):
        """Decision source must be orion-brain."""
        schema = load_schema("decision.schema.json")
        validator = Draft202012Validator(schema)

        valid_decision_v1["source"] = "orion-guardian"  # Wrong source

        with pytest.raises(ValidationError):
            validator.validate(valid_decision_v1)

    def test_reasoning_min_length(self, valid_decision_v1):
        """Decision reasoning must be at least 10 characters."""
        schema = load_schema("decision.schema.json")
        validator = Draft202012Validator(schema)

        valid_decision_v1["reasoning"] = "Too short"  # Less than 10 chars

        with pytest.raises(ValidationError):
            validator.validate(valid_decision_v1)


@pytest.mark.contract
class TestActionContract:
    """Test action.schema.json validation."""

    def test_schema_is_valid(self):
        """action.schema.json is a valid JSON Schema."""
        schema = load_schema("action.schema.json")
        Draft202012Validator.check_schema(schema)

    def test_valid_action_accepted(self, valid_action_v1):
        """Valid action passes validation."""
        schema = load_schema("action.schema.json")
        validator = Draft202012Validator(schema)
        validator.validate(valid_action_v1)

    def test_safety_classification_no_unknown(self, valid_action_v1):
        """Action safety_classification cannot be UNKNOWN."""
        schema = load_schema("action.schema.json")
        validator = Draft202012Validator(schema)

        valid_action_v1["safety_classification"] = "UNKNOWN"  # Not in enum

        with pytest.raises(ValidationError):
            validator.validate(valid_action_v1)


@pytest.mark.contract
class TestApprovalContract:
    """Test approval.schema.json validation."""

    def test_schema_is_valid(self):
        """approval.schema.json is a valid JSON Schema."""
        schema = load_schema("approval.schema.json")
        Draft202012Validator.check_schema(schema)

    def test_valid_approval_accepted(self, valid_approval_v1):
        """Valid approval passes validation."""
        schema = load_schema("approval.schema.json")
        validator = Draft202012Validator(schema)
        validator.validate(valid_approval_v1)

    def test_source_must_be_approval_telegram(self, valid_approval_v1):
        """Approval source must be orion-approval-telegram."""
        schema = load_schema("approval.schema.json")
        validator = Draft202012Validator(schema)

        valid_approval_v1["source"] = "orion-brain"  # Wrong source

        with pytest.raises(ValidationError):
            validator.validate(valid_approval_v1)


@pytest.mark.contract
class TestOutcomeContract:
    """Test outcome.schema.json validation."""

    def test_schema_is_valid(self):
        """outcome.schema.json is a valid JSON Schema."""
        schema = load_schema("outcome.schema.json")
        Draft202012Validator.check_schema(schema)

    def test_valid_outcome_accepted(self, valid_outcome_v1):
        """Valid outcome passes validation."""
        schema = load_schema("outcome.schema.json")
        validator = Draft202012Validator(schema)
        validator.validate(valid_outcome_v1)

    def test_source_must_be_commander(self, valid_outcome_v1):
        """Outcome source must be orion-commander."""
        schema = load_schema("outcome.schema.json")
        validator = Draft202012Validator(schema)

        valid_outcome_v1["source"] = "orion-brain"  # Wrong source

        with pytest.raises(ValidationError):
            validator.validate(valid_outcome_v1)

    def test_execution_time_cannot_be_negative(self, valid_outcome_v1):
        """Outcome execution_time_ms must be >= 0."""
        schema = load_schema("outcome.schema.json")
        validator = Draft202012Validator(schema)

        valid_outcome_v1["execution_time_ms"] = -1  # Invalid

        with pytest.raises(ValidationError):
            validator.validate(valid_outcome_v1)

    def test_unknown_field_rejected(self, valid_outcome_v1):
        """Outcome with unknown field is rejected."""
        schema = load_schema("outcome.schema.json")
        validator = Draft202012Validator(schema)

        valid_outcome_v1["unknown_field"] = "should_fail"

        with pytest.raises(ValidationError):
            validator.validate(valid_outcome_v1)
