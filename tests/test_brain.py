"""
Brain tests.

Tests N0 decision making (NO_ACTION enforcement).
"""

from pathlib import Path
import pytest
import fakeredis

from bus.python.orion_bus import EventBus
from core.brain import Brain


CONTRACTS_DIR = Path(__file__).parent.parent / "bus" / "contracts"


@pytest.fixture
def redis_client():
    """Provide fake Redis client."""
    return fakeredis.FakeRedis(decode_responses=False)


@pytest.fixture
def event_bus(redis_client):
    """Provide event bus."""
    return EventBus(redis_client, CONTRACTS_DIR)


@pytest.fixture
def brain(event_bus):
    """Provide brain instance in N0 mode."""
    return Brain(event_bus, autonomy_level="N0")


@pytest.mark.unit
class TestBrainInit:
    """Test brain initialization."""

    def test_initializes_in_n0_mode(self, event_bus):
        """Brain initializes in N0 mode."""
        brain = Brain(event_bus, autonomy_level="N0")

        assert brain.autonomy_level == "N0"
        assert brain.source_name == "orion-brain"

    def test_rejects_unsupported_modes(self, event_bus):
        """Brain rejects unsupported autonomy levels (N1, N4+)."""
        with pytest.raises(ValueError, match="Only N0, N2, and N3 modes supported"):
            Brain(event_bus, autonomy_level="N1")

        with pytest.raises(ValueError, match="Only N0, N2, and N3 modes supported"):
            Brain(event_bus, autonomy_level="N4")


@pytest.mark.unit
class TestBrainReasoning:
    """Test reasoning generation."""

    def test_generates_reasoning(self, brain, valid_incident_v1):
        """Brain generates reasoning for decisions."""
        reasoning = brain._generate_reasoning_n0(valid_incident_v1)

        assert len(reasoning) >= 10  # Minimum required by contract
        assert "N0" in reasoning
        assert "observe only" in reasoning
        assert valid_incident_v1["incident_type"] in reasoning

    def test_reasoning_includes_incident_details(self, brain, valid_incident_v1):
        """Reasoning includes incident type and severity."""
        reasoning = brain._generate_reasoning_n0(valid_incident_v1)

        assert valid_incident_v1["incident_type"] in reasoning
        assert valid_incident_v1["severity"] in reasoning


@pytest.mark.unit
class TestBrainDecisions:
    """Test decision making in N0 mode."""

    def test_decide_always_returns_no_action(self, brain, valid_incident_v1):
        """In N0 mode, brain always decides NO_ACTION."""
        decision = brain.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"

    def test_decision_matches_contract(self, brain, valid_incident_v1):
        """Decision matches decision.schema.json contract."""
        decision = brain.decide(valid_incident_v1)

        # Required fields
        assert decision["version"] == "1.0"
        assert "decision_id" in decision
        assert "timestamp" in decision
        assert decision["source"] == "orion-brain"
        assert decision["incident_id"] == valid_incident_v1["incident_id"]
        assert decision["decision_type"] == "NO_ACTION"
        assert decision["safety_classification"] in ["SAFE", "RISKY", "UNKNOWN"]
        assert isinstance(decision["requires_approval"], bool)
        assert len(decision["reasoning"]) >= 10
        assert decision["autonomy_level"] == "N0"

    def test_no_action_is_safe(self, brain, valid_incident_v1):
        """NO_ACTION is always classified as SAFE."""
        decision = brain.decide(valid_incident_v1)

        assert decision["safety_classification"] == "SAFE"

    def test_no_action_never_requires_approval(self, brain, valid_incident_v1):
        """NO_ACTION never requires approval."""
        decision = brain.decide(valid_incident_v1)

        assert decision["requires_approval"] is False

    def test_decision_includes_reasoning(self, brain, valid_incident_v1):
        """Decision includes explicit reasoning."""
        decision = brain.decide(valid_incident_v1)

        assert "reasoning" in decision
        assert len(decision["reasoning"]) >= 10

    def test_decision_for_critical_incident_still_no_action(self, brain, valid_incident_v1):
        """Even critical incidents result in NO_ACTION in N0 mode."""
        valid_incident_v1["severity"] = "critical"

        decision = brain.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"

    def test_decision_for_service_outage_still_no_action(self, brain, valid_incident_v1):
        """Service outage still results in NO_ACTION in N0 mode."""
        valid_incident_v1["incident_type"] = "service_outage"
        valid_incident_v1["severity"] = "high"

        decision = brain.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"


@pytest.mark.unit
class TestBrainIncidentHandling:
    """Test incident handling."""

    def test_handle_incident_creates_decision(self, brain, event_bus, valid_incident_v1):
        """Handling incident creates and publishes decision."""
        brain.handle_incident(valid_incident_v1)

        # Check that decision was published
        decisions = event_bus.read_stream("decision")
        assert len(decisions) == 1
        assert decisions[0]["decision_type"] == "NO_ACTION"
        assert decisions[0]["incident_id"] == valid_incident_v1["incident_id"]

    def test_handle_multiple_incidents(self, brain, event_bus, valid_incident_v1):
        """Multiple incidents create multiple decisions."""
        brain.handle_incident(valid_incident_v1)

        incident2 = valid_incident_v1.copy()
        incident2["incident_id"] = "different-incident-id"
        brain.handle_incident(incident2)

        decisions = event_bus.read_stream("decision")
        assert len(decisions) == 2
        assert all(d["decision_type"] == "NO_ACTION" for d in decisions)


@pytest.mark.unit
class TestBrainN0Enforcement:
    """Test N0 mode invariant enforcement."""

    def test_never_suggests_action(self, brain, valid_incident_v1):
        """Brain never suggests actions in N0 mode."""
        decision = brain.decide(valid_incident_v1)

        assert decision["decision_type"] != "SUGGEST_ACTION"
        assert "proposed_action" not in decision

    def test_never_executes_action(self, brain, valid_incident_v1):
        """Brain never decides to execute actions in N0 mode."""
        decision = brain.decide(valid_incident_v1)

        assert decision["decision_type"] != "EXECUTE_SAFE_ACTION"
        assert "proposed_action" not in decision

    def test_never_requests_approval(self, brain, valid_incident_v1):
        """Brain never requests approval in N0 mode."""
        decision = brain.decide(valid_incident_v1)

        assert decision["decision_type"] != "REQUEST_APPROVAL"
        assert decision["requires_approval"] is False

    def test_decision_type_is_always_no_action(self, brain, valid_incident_v1):
        """Decision type is ALWAYS NO_ACTION in N0 mode."""
        # Test with different incident types and severities
        test_cases = [
            {"incident_type": "service_outage", "severity": "critical"},
            {"incident_type": "edge_device_failure", "severity": "high"},
            {"incident_type": "metric_anomaly", "severity": "medium"},
            {"incident_type": "repeated_failures", "severity": "low"},
        ]

        for case in test_cases:
            incident = valid_incident_v1.copy()
            incident.update(case)

            decision = brain.decide(incident)

            assert decision["decision_type"] == "NO_ACTION", \
                f"Expected NO_ACTION for {case}, got {decision['decision_type']}"
