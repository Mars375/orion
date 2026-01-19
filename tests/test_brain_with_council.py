"""
Integration tests for Brain + AI Council.

Tests the integration between Brain decision making and Council validation,
verifying that Council blocking changes decisions to NO_ACTION and that
backward compatibility is maintained when council=None.
"""

from pathlib import Path
from unittest.mock import MagicMock, AsyncMock, patch
import pytest
import fakeredis

from bus.python.orion_bus import EventBus
from core.brain import Brain
from core.council import ConsensusAggregator, CouncilValidator, ExternalValidator


CONTRACTS_DIR = Path(__file__).parent.parent / "bus" / "contracts"
POLICIES_DIR = Path(__file__).parent.parent / "policies"


@pytest.fixture
def redis_client():
    """Provide fake Redis client."""
    return fakeredis.FakeRedis(decode_responses=False)


@pytest.fixture
def event_bus(redis_client):
    """Provide event bus."""
    return EventBus(redis_client, CONTRACTS_DIR)


@pytest.fixture
def mock_council():
    """Provide a mock ConsensusAggregator."""
    council = MagicMock(spec=ConsensusAggregator)
    return council


@pytest.fixture
def mock_council_validator():
    """Provide a mock CouncilValidator."""
    validator = MagicMock(spec=CouncilValidator)
    return validator


@pytest.fixture
def mock_external_validator():
    """Provide a mock ExternalValidator."""
    validator = MagicMock(spec=ExternalValidator)
    return validator


@pytest.mark.integration
class TestBrainWithoutCouncil:
    """Tests for Brain behavior when council=None (backward compatibility)."""

    def test_brain_without_council_unchanged(self, event_bus, valid_incident_v1):
        """Brain with council=None behaves identically to before."""
        brain = Brain(event_bus, autonomy_level="N0", council=None)

        decision = brain.decide(valid_incident_v1)

        # Should work exactly as before
        assert decision["decision_type"] == "NO_ACTION"
        assert "N0" in decision["reasoning"]
        assert "BLOCKED BY COUNCIL" not in decision["reasoning"]

    def test_brain_n0_without_council(self, event_bus, valid_incident_v1):
        """N0 mode still works without council."""
        brain = Brain(event_bus, autonomy_level="N0")

        decision = brain.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"
        assert brain.council is None


@pytest.mark.integration
class TestBrainCouncilApproval:
    """Tests for Brain + Council when council APPROVES decisions."""

    def test_brain_council_approves_safe_decision(
        self,
        event_bus,
        valid_incident_v1,
        mock_council,
        mock_council_validator,
        mock_external_validator,
    ):
        """Council APPROVED allows decision to proceed unchanged."""
        # Configure mock council to approve
        mock_council.validate_decision = AsyncMock(
            return_value=("APPROVED", 0.9, "Decision validated successfully")
        )

        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=mock_council,
            council_validator=mock_council_validator,
            external_validator=mock_external_validator,
        )

        # Make decision
        decision = brain.decide(valid_incident_v1)

        # Validate with council via handle_incident pathway
        decision = brain._validate_with_council(decision, valid_incident_v1)

        # Decision should proceed unchanged
        assert decision["decision_type"] == "NO_ACTION"  # N0 always NO_ACTION
        assert "BLOCKED BY COUNCIL" not in decision["reasoning"]


@pytest.mark.integration
class TestBrainCouncilBlocking:
    """Tests for Brain + Council when council BLOCKS decisions."""

    def test_brain_council_blocks_unsafe_decision(
        self,
        event_bus,
        valid_incident_v1,
        mock_council,
        mock_council_validator,
        mock_external_validator,
    ):
        """Council BLOCKED changes decision to NO_ACTION."""
        # Configure mock council to block
        mock_council.validate_decision = AsyncMock(
            return_value=("BLOCKED", 0.0, "Safety veto triggered")
        )

        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=mock_council,
            council_validator=mock_council_validator,
            external_validator=mock_external_validator,
        )

        # Make decision
        decision = brain.decide(valid_incident_v1)

        # Validate with council
        decision = brain._validate_with_council(decision, valid_incident_v1)

        # Decision should be changed to NO_ACTION
        assert decision["decision_type"] == "NO_ACTION"
        assert "BLOCKED BY COUNCIL" in decision["reasoning"]
        assert "Safety veto" in decision["reasoning"]

    def test_brain_council_block_removes_proposed_action(
        self,
        event_bus,
        valid_incident_v1,
        mock_council,
        mock_council_validator,
        mock_external_validator,
    ):
        """Council blocking removes proposed_action from decision."""
        mock_council.validate_decision = AsyncMock(
            return_value=("BLOCKED", 0.0, "Unsafe action blocked")
        )

        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=mock_council,
            council_validator=mock_council_validator,
            external_validator=mock_external_validator,
        )

        # Create a decision with proposed_action
        decision = brain.decide(valid_incident_v1)
        decision["proposed_action"] = {"action_type": "test_action"}

        # Validate with council
        decision = brain._validate_with_council(decision, valid_incident_v1)

        # proposed_action should be removed
        assert "proposed_action" not in decision


@pytest.mark.integration
class TestBrainCouncilEscalation:
    """Tests for Council escalation behavior."""

    def test_council_escalates_risky_to_external_apis(
        self,
        event_bus,
        valid_incident_v1,
        mock_council_validator,
        mock_external_validator,
    ):
        """RISKY decisions escalate to external APIs."""
        # Use real ConsensusAggregator but mock validators
        council = ConsensusAggregator(confidence_threshold=0.7)

        # Mock local validator returns low confidence
        mock_council_validator.validate.return_value = (
            0.5,
            "Local is uncertain about this safe decision"
        )

        # Mock external validator
        mock_external_validator.validate_parallel = AsyncMock(
            return_value=[
                (0.9, "Claude approved"),
                (0.85, "OpenAI approved")
            ]
        )

        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=council,
            council_validator=mock_council_validator,
            external_validator=mock_external_validator,
        )

        decision = brain.decide(valid_incident_v1)
        decision = brain._validate_with_council(decision, valid_incident_v1)

        # External validator should have been called (escalation)
        mock_external_validator.validate_parallel.assert_called_once()


@pytest.mark.integration
class TestBrainCouncilFailClosed:
    """Tests for fail-closed behavior on Council errors."""

    def test_council_validation_failure_fails_closed(
        self,
        event_bus,
        valid_incident_v1,
        mock_council,
        mock_council_validator,
        mock_external_validator,
    ):
        """Council validation error results in BLOCKED (fail-closed)."""
        # Configure mock council to raise exception
        mock_council.validate_decision = AsyncMock(
            side_effect=Exception("Connection failed")
        )

        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=mock_council,
            council_validator=mock_council_validator,
            external_validator=mock_external_validator,
        )

        decision = brain.decide(valid_incident_v1)
        decision = brain._validate_with_council(decision, valid_incident_v1)

        # Should be blocked due to error
        assert decision["decision_type"] == "NO_ACTION"
        assert "BLOCKED BY COUNCIL" in decision["reasoning"]
        assert "Validation error" in decision["reasoning"]

    def test_council_missing_validators_skips_validation(
        self,
        event_bus,
        valid_incident_v1,
        mock_council,
    ):
        """Missing validators causes validation to be skipped."""
        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=mock_council,
            council_validator=None,  # Missing!
            external_validator=None,  # Missing!
        )

        original_decision = brain.decide(valid_incident_v1)
        validated_decision = brain._validate_with_council(
            original_decision.copy(),
            valid_incident_v1
        )

        # Should skip validation, not block
        assert validated_decision["decision_type"] == original_decision["decision_type"]
        assert "BLOCKED BY COUNCIL" not in validated_decision["reasoning"]


@pytest.mark.integration
class TestBrainCouncilHandleIncident:
    """Tests for full handle_incident flow with Council."""

    def test_handle_incident_calls_council_validation(
        self,
        event_bus,
        valid_incident_v1,
        mock_council,
        mock_council_validator,
        mock_external_validator,
    ):
        """handle_incident calls Council validation before publishing."""
        mock_council.validate_decision = AsyncMock(
            return_value=("APPROVED", 0.9, "Validated")
        )

        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=mock_council,
            council_validator=mock_council_validator,
            external_validator=mock_external_validator,
        )

        # Mock the bus publish
        with patch.object(brain.bus, "publish") as mock_publish:
            brain.handle_incident(valid_incident_v1)

            # Council should have been called
            mock_council.validate_decision.assert_called_once()

            # Decision should have been published
            mock_publish.assert_called_once()

    def test_handle_incident_publishes_blocked_decision(
        self,
        event_bus,
        valid_incident_v1,
        mock_council,
        mock_council_validator,
        mock_external_validator,
    ):
        """handle_incident publishes BLOCKED decision when Council blocks."""
        mock_council.validate_decision = AsyncMock(
            return_value=("BLOCKED", 0.0, "Blocked for safety")
        )

        brain = Brain(
            event_bus,
            autonomy_level="N0",
            council=mock_council,
            council_validator=mock_council_validator,
            external_validator=mock_external_validator,
        )

        with patch.object(brain.bus, "publish") as mock_publish:
            brain.handle_incident(valid_incident_v1)

            # Check the published decision
            published_decision = mock_publish.call_args[0][0]
            assert published_decision["decision_type"] == "NO_ACTION"
            assert "BLOCKED BY COUNCIL" in published_decision["reasoning"]
