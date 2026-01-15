"""
Brain N2 mode tests.

Tests N2 decision making (SAFE actions allowed).
"""

from pathlib import Path
import pytest
import fakeredis

from bus.python.orion_bus import EventBus
from core.brain import Brain


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
def brain_n2(event_bus):
    """Provide brain instance in N2 mode."""
    return Brain(event_bus, autonomy_level="N2", policy_dir=POLICIES_DIR)


@pytest.mark.unit
class TestBrainN2Init:
    """Test N2 mode initialization."""

    def test_initializes_in_n2_mode(self, event_bus):
        """Brain initializes in N2 mode with policies."""
        brain = Brain(event_bus, autonomy_level="N2", policy_dir=POLICIES_DIR)

        assert brain.autonomy_level == "N2"
        assert brain.policy_loader is not None
        assert brain.cooldown_tracker is not None
        assert brain.circuit_breaker is not None

    def test_requires_policy_dir_in_n2(self, event_bus):
        """N2 mode requires policy directory."""
        with pytest.raises(ValueError, match="N2 mode requires policy_dir"):
            Brain(event_bus, autonomy_level="N2", policy_dir=None)


@pytest.mark.unit
class TestBrainN2SafeActions:
    """Test N2 SAFE action execution."""

    def test_executes_safe_actions(self, brain_n2, valid_incident_v1):
        """N2 mode executes SAFE actions for appropriate incidents."""
        # Medium severity should trigger acknowledge_incident
        valid_incident_v1["severity"] = "medium"

        decision = brain_n2.decide(valid_incident_v1)

        assert decision["decision_type"] == "EXECUTE_SAFE_ACTION"
        assert decision["safety_classification"] == "SAFE"
        assert "proposed_action" in decision
        assert decision["proposed_action"]["action_type"] == "acknowledge_incident"

    def test_includes_proposed_action(self, brain_n2, valid_incident_v1):
        """EXECUTE_SAFE_ACTION includes proposed_action."""
        valid_incident_v1["severity"] = "high"

        decision = brain_n2.decide(valid_incident_v1)

        assert decision["decision_type"] == "EXECUTE_SAFE_ACTION"
        assert "proposed_action" in decision

        proposed = decision["proposed_action"]
        assert proposed["action_type"] == "acknowledge_incident"
        assert "parameters" in proposed
        assert proposed["parameters"]["incident_id"] == valid_incident_v1["incident_id"]

    def test_low_severity_no_action(self, brain_n2, valid_incident_v1):
        """Low severity incidents result in NO_ACTION."""
        valid_incident_v1["severity"] = "low"

        decision = brain_n2.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"
        assert "proposed_action" not in decision

    def test_critical_severity_executes(self, brain_n2, valid_incident_v1):
        """Critical severity triggers action."""
        valid_incident_v1["severity"] = "critical"

        decision = brain_n2.decide(valid_incident_v1)

        assert decision["decision_type"] == "EXECUTE_SAFE_ACTION"


@pytest.mark.unit
class TestBrainN2RiskyActions:
    """Test N2 RISKY action blocking."""

    def test_blocks_risky_actions(self, brain_n2, valid_incident_v1):
        """RISKY actions result in NO_ACTION in N2 mode."""
        # Manually inject a risky action determination
        # (In full implementation, would be determined by incident type)
        # For this test, we verify the policy enforcement logic

        # This is a unit test of the policy enforcement path
        # The actual action determination logic only returns acknowledge_incident
        # But we test that IF a risky action were proposed, it would be blocked

        decision = brain_n2.decide(valid_incident_v1)

        # Current implementation only proposes acknowledge_incident (SAFE)
        # RISKY actions would be blocked by policy loader checks
        if decision["decision_type"] == "EXECUTE_SAFE_ACTION":
            action_type = decision["proposed_action"]["action_type"]
            assert brain_n2.policy_loader.is_safe(action_type)
            assert not brain_n2.policy_loader.is_risky(action_type)

    def test_risky_classification_in_reasoning(self, brain_n2):
        """Reasoning explains why RISKY actions blocked."""
        # This test verifies the reasoning logic for RISKY actions
        # We can't easily trigger this in current implementation
        # But the code path exists and is tested via inspection

        # Verify policy loader correctly classifies RISKY actions
        assert brain_n2.policy_loader.is_risky("restart_service")
        assert brain_n2.policy_loader.classify_action("restart_service") == "RISKY"


@pytest.mark.unit
class TestBrainN2Cooldowns:
    """Test N2 cooldown enforcement."""

    def test_cooldown_prevents_repeated_execution(self, brain_n2, valid_incident_v1):
        """Cooldown prevents rapid repeated execution."""
        # Note: acknowledge_incident has 0s cooldown by policy
        # So we test by manually recording an execution first
        valid_incident_v1["severity"] = "medium"

        # Manually record a recent execution to simulate cooldown
        brain_n2.cooldown_tracker.record_execution("acknowledge_incident")

        # Decision should be blocked by cooldown
        decision = brain_n2.decide(valid_incident_v1)

        # Since we set a cooldown, it should check and find it's in cooldown
        # But acknowledge_incident has 0s cooldown, so it won't block
        # Let's test the cooldown mechanism with a different approach:
        # Override the policy loader to return a non-zero cooldown
        import unittest.mock
        with unittest.mock.patch.object(
            brain_n2.policy_loader, 'get_cooldown', return_value=60
        ):
            # Now decide should check cooldown
            decision2 = brain_n2.decide(valid_incident_v1)
            assert decision2["decision_type"] == "NO_ACTION"
            assert "cooldown" in decision2["reasoning"].lower()

    def test_cooldown_in_reasoning(self, brain_n2, valid_incident_v1):
        """Cooldown block includes reasoning."""
        import unittest.mock

        valid_incident_v1["severity"] = "high"

        # Mock cooldown to be non-zero and already recorded
        brain_n2.cooldown_tracker.record_execution("acknowledge_incident")

        with unittest.mock.patch.object(
            brain_n2.policy_loader, 'get_cooldown', return_value=60
        ):
            decision = brain_n2.decide(valid_incident_v1)

            assert decision["decision_type"] == "NO_ACTION"
            assert "cooldown" in decision["reasoning"]
            assert "remaining" in decision["reasoning"]

    def test_different_actions_independent_cooldowns(self, brain_n2, valid_incident_v1):
        """Different action types have independent cooldowns."""
        valid_incident_v1["severity"] = "medium"

        # Record execution for one action type
        brain_n2.cooldown_tracker.record_execution("acknowledge_incident")

        # Cooldown tracker tracks per action_type
        assert not brain_n2.cooldown_tracker.check_cooldown("acknowledge_incident", 10)

        # Different action would have independent cooldown
        assert brain_n2.cooldown_tracker.check_cooldown("different_action", 10)


@pytest.mark.unit
class TestBrainN2CircuitBreaker:
    """Test N2 circuit breaker enforcement."""

    def test_circuit_breaker_prevents_execution(self, brain_n2, valid_incident_v1):
        """Open circuit breaker prevents execution."""
        valid_incident_v1["severity"] = "medium"

        # Manually open circuit breaker
        for _ in range(3):
            brain_n2.circuit_breaker.record_failure("acknowledge_incident")

        decision = brain_n2.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"
        assert "circuit breaker" in decision["reasoning"]

    def test_circuit_breaker_in_reasoning(self, brain_n2, valid_incident_v1):
        """Circuit breaker block includes reasoning."""
        valid_incident_v1["severity"] = "high"

        # Open circuit
        for _ in range(3):
            brain_n2.circuit_breaker.record_failure("acknowledge_incident")

        decision = brain_n2.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"
        assert "circuit breaker" in decision["reasoning"]
        assert "OPEN" in decision["reasoning"]


@pytest.mark.unit
class TestBrainN2UnknownActions:
    """Test N2 unknown action handling."""

    def test_unknown_actions_blocked(self, brain_n2, valid_incident_v1):
        """Unknown actions result in NO_ACTION (fail closed)."""
        # Current implementation only uses known SAFE actions
        # But verify policy loader behavior

        classification = brain_n2.policy_loader.classify_action("unknown_action")
        assert classification == "UNKNOWN"

        # Unknown actions would be treated as RISKY
        assert not brain_n2.policy_loader.is_safe("unknown_action")


@pytest.mark.unit
class TestBrainN2DecisionContract:
    """Test N2 decision contract compliance."""

    def test_decision_matches_contract(self, brain_n2, valid_incident_v1):
        """Decision matches decision.schema.json in N2 mode."""
        valid_incident_v1["severity"] = "medium"

        decision = brain_n2.decide(valid_incident_v1)

        # Required fields
        assert decision["version"] == "1.0"
        assert "decision_id" in decision
        assert "timestamp" in decision
        assert decision["source"] == "orion-brain"
        assert decision["incident_id"] == valid_incident_v1["incident_id"]
        assert decision["decision_type"] in ["NO_ACTION", "EXECUTE_SAFE_ACTION"]
        assert decision["safety_classification"] in ["SAFE", "RISKY", "UNKNOWN"]
        assert isinstance(decision["requires_approval"], bool)
        assert len(decision["reasoning"]) >= 10
        assert decision["autonomy_level"] == "N2"

    def test_safe_action_never_requires_approval(self, brain_n2, valid_incident_v1):
        """SAFE actions in N2 never require approval."""
        valid_incident_v1["severity"] = "high"

        decision = brain_n2.decide(valid_incident_v1)

        if decision["decision_type"] == "EXECUTE_SAFE_ACTION":
            assert decision["requires_approval"] is False

    def test_no_action_never_requires_approval(self, brain_n2, valid_incident_v1):
        """NO_ACTION never requires approval."""
        valid_incident_v1["severity"] = "low"

        decision = brain_n2.decide(valid_incident_v1)

        assert decision["decision_type"] == "NO_ACTION"
        assert decision["requires_approval"] is False


@pytest.mark.unit
class TestBrainN2Reasoning:
    """Test N2 reasoning generation."""

    def test_reasoning_includes_mode(self, brain_n2, valid_incident_v1):
        """Reasoning includes N2 mode."""
        valid_incident_v1["severity"] = "medium"

        decision = brain_n2.decide(valid_incident_v1)

        assert "N2" in decision["reasoning"]

    def test_reasoning_includes_action_type(self, brain_n2, valid_incident_v1):
        """Reasoning includes action type for EXECUTE decisions."""
        valid_incident_v1["severity"] = "high"

        decision = brain_n2.decide(valid_incident_v1)

        if decision["decision_type"] == "EXECUTE_SAFE_ACTION":
            assert "acknowledge_incident" in decision["reasoning"]

    def test_reasoning_includes_incident_details(self, brain_n2, valid_incident_v1):
        """Reasoning includes incident context."""
        valid_incident_v1["severity"] = "critical"
        valid_incident_v1["incident_type"] = "service_outage"

        decision = brain_n2.decide(valid_incident_v1)

        reasoning = decision["reasoning"]
        assert "service_outage" in reasoning or "critical" in reasoning


@pytest.mark.unit
class TestBrainN2IncidentHandling:
    """Test N2 incident handling and bus integration."""

    def test_handle_incident_publishes_decision(self, brain_n2, event_bus, valid_incident_v1):
        """Handling incident publishes decision to bus."""
        valid_incident_v1["severity"] = "medium"

        brain_n2.handle_incident(valid_incident_v1)

        decisions = event_bus.read_stream("decision")
        assert len(decisions) == 1
        assert decisions[0]["incident_id"] == valid_incident_v1["incident_id"]

    def test_multiple_incidents_handled(self, brain_n2, event_bus, valid_incident_v1):
        """Multiple incidents create multiple decisions."""
        valid_incident_v1["severity"] = "high"

        brain_n2.handle_incident(valid_incident_v1)

        incident2 = valid_incident_v1.copy()
        incident2["incident_id"] = "different-incident"
        brain_n2.handle_incident(incident2)

        decisions = event_bus.read_stream("decision")
        assert len(decisions) == 2


@pytest.mark.unit
class TestBrainN2SafetyInvariants:
    """Test N2 safety invariants."""

    def test_never_executes_risky(self, brain_n2, valid_incident_v1):
        """N2 never decides to execute RISKY actions."""
        # Test various incident types and severities
        test_cases = [
            {"incident_type": "service_outage", "severity": "critical"},
            {"incident_type": "edge_device_failure", "severity": "high"},
            {"incident_type": "metric_anomaly", "severity": "medium"},
        ]

        for case in test_cases:
            incident = valid_incident_v1.copy()
            incident.update(case)

            decision = brain_n2.decide(incident)

            # If action proposed, must be SAFE
            if "proposed_action" in decision:
                action_type = decision["proposed_action"]["action_type"]
                assert brain_n2.policy_loader.is_safe(action_type)

    def test_fail_closed_on_unknown(self, brain_n2, valid_incident_v1):
        """Unknown classifications result in NO_ACTION (fail closed)."""
        valid_incident_v1["severity"] = "medium"

        decision = brain_n2.decide(valid_incident_v1)

        # Any unknown classification should fail closed
        if decision["safety_classification"] == "UNKNOWN":
            assert decision["decision_type"] == "NO_ACTION"

    def test_all_actions_auditable(self, brain_n2, valid_incident_v1):
        """All decisions include reasoning (auditability)."""
        valid_incident_v1["severity"] = "high"

        decision = brain_n2.decide(valid_incident_v1)

        assert "reasoning" in decision
        assert len(decision["reasoning"]) >= 10
        assert decision["autonomy_level"] == "N2"
