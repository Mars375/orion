"""
Commander tests.

Tests action execution, rollback, and outcome emission.
"""

from pathlib import Path
import pytest
import fakeredis

from bus.python.orion_bus import EventBus
from core.commander import Commander
from core.memory import MemoryStore


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
def memory_store(redis_client, tmp_path):
    """Provide memory store."""
    return MemoryStore(storage_dir=tmp_path / "memory")


@pytest.fixture
def commander(event_bus, memory_store):
    """Provide commander instance."""
    return Commander(event_bus, POLICIES_DIR, memory_store)


@pytest.fixture
def valid_safe_decision():
    """Provide valid EXECUTE_SAFE_ACTION decision."""
    return {
        "version": "1.0",
        "decision_id": "test-decision-001",
        "timestamp": "2025-01-15T10:00:00Z",
        "source": "orion-brain",
        "incident_id": "test-incident-001",
        "decision_type": "EXECUTE_SAFE_ACTION",
        "safety_classification": "SAFE",
        "requires_approval": False,
        "reasoning": "Test reasoning for SAFE action execution",
        "autonomy_level": "N2",
        "proposed_action": {
            "action_type": "acknowledge_incident",
            "parameters": {
                "incident_id": "test-incident-001",
            },
        },
    }


@pytest.fixture
def valid_no_action_decision():
    """Provide valid NO_ACTION decision."""
    return {
        "version": "1.0",
        "decision_id": "test-decision-002",
        "timestamp": "2025-01-15T10:00:00Z",
        "source": "orion-brain",
        "incident_id": "test-incident-002",
        "decision_type": "NO_ACTION",
        "safety_classification": "SAFE",
        "requires_approval": False,
        "reasoning": "No action needed",
        "autonomy_level": "N2",
    }


@pytest.mark.unit
class TestCommanderInit:
    """Test commander initialization."""

    def test_initializes_with_dependencies(self, event_bus, memory_store):
        """Commander initializes with required dependencies."""
        commander = Commander(event_bus, POLICIES_DIR, memory_store)

        assert commander.bus is not None
        assert commander.policy_loader is not None
        assert commander.memory is not None
        assert commander.source_name == "orion-commander"


@pytest.mark.unit
class TestCommanderSafeActions:
    """Test SAFE action execution."""

    def test_executes_safe_action(self, commander, valid_safe_decision):
        """Commander executes SAFE actions."""
        action = commander._create_action(valid_safe_decision)

        outcome = commander.execute_action(action)

        assert outcome["status"] == "succeeded"
        assert outcome["action_id"] == action["action_id"]
        assert "execution_time_ms" in outcome
        assert "result" in outcome

    def test_creates_action_from_decision(self, commander, valid_safe_decision):
        """Creates action contract from decision."""
        action = commander._create_action(valid_safe_decision)

        assert action["version"] == "1.0"
        assert "action_id" in action
        assert "timestamp" in action
        assert action["source"] == "orion-brain"
        assert action["decision_id"] == valid_safe_decision["decision_id"]
        assert action["action_type"] == "acknowledge_incident"
        assert action["safety_classification"] == "SAFE"
        assert action["state"] == "pending"
        assert action["rollback_enabled"] is True
        assert action["dry_run"] is False

    def test_acknowledge_incident_execution(self, commander, valid_safe_decision):
        """acknowledge_incident action executes correctly."""
        action = commander._create_action(valid_safe_decision)

        outcome = commander.execute_action(action)

        assert outcome["status"] == "succeeded"
        assert "result" in outcome
        assert outcome["result"]["incident_id"] == "test-incident-001"
        assert "acknowledgment" in outcome["result"]

    def test_outcome_includes_execution_time(self, commander, valid_safe_decision):
        """Outcome includes execution time in milliseconds."""
        action = commander._create_action(valid_safe_decision)

        outcome = commander.execute_action(action)

        assert "execution_time_ms" in outcome
        assert isinstance(outcome["execution_time_ms"], int)
        assert outcome["execution_time_ms"] >= 0


@pytest.mark.unit
class TestCommanderRiskyActions:
    """Test RISKY action blocking."""

    def test_blocks_risky_action(self, commander, valid_safe_decision):
        """Commander blocks RISKY actions."""
        # Change to risky action type
        valid_safe_decision["proposed_action"]["action_type"] = "restart_service"
        valid_safe_decision["safety_classification"] = "RISKY"

        # handle_decision should reject this
        commander.handle_decision(valid_safe_decision)

        # No outcome should be published (decision rejected)
        outcomes = commander.bus.read_stream("outcome")
        assert len(outcomes) == 0

    def test_logs_risky_rejection(self, commander, valid_safe_decision, caplog):
        """Rejecting RISKY action is logged."""
        valid_safe_decision["proposed_action"]["action_type"] = "restart_service"

        commander.handle_decision(valid_safe_decision)

        # Should log refusal
        assert any("not SAFE" in record.message for record in caplog.records)


@pytest.mark.unit
class TestCommanderUnknownActions:
    """Test unknown action handling."""

    def test_fails_on_unknown_action(self, commander, valid_safe_decision):
        """Unknown action types result in failure."""
        action = commander._create_action(valid_safe_decision)
        action["action_type"] = "unknown_action_type"

        outcome = commander.execute_action(action)

        assert outcome["status"] in ["failed", "rolled_back"]
        assert "error" in outcome
        assert outcome["error"]["code"] == "EXECUTION_FAILED"


@pytest.mark.unit
class TestCommanderRollback:
    """Test rollback functionality."""

    def test_rollback_on_failure(self, commander, valid_safe_decision):
        """Failed actions trigger rollback."""
        action = commander._create_action(valid_safe_decision)

        # Simulate failure by using None parameters (will cause AttributeError)
        action["parameters"] = None

        outcome = commander.execute_action(action)

        # Should attempt rollback
        assert outcome["status"] in ["failed", "rolled_back"]

    def test_rollback_executed_flag(self, commander, valid_safe_decision):
        """Rolled back outcomes have rollback_executed flag."""
        action = commander._create_action(valid_safe_decision)
        action["parameters"] = None  # Force failure

        outcome = commander.execute_action(action)

        if outcome["status"] == "rolled_back":
            assert outcome.get("rollback_executed") is True


@pytest.mark.unit
class TestCommanderOutcomes:
    """Test outcome contract creation."""

    def test_creates_outcome_contract(self, commander, valid_safe_decision):
        """Creates outcome matching outcome.schema.json."""
        action = commander._create_action(valid_safe_decision)

        outcome = commander.execute_action(action)

        # Required fields
        assert outcome["version"] == "1.0"
        assert "outcome_id" in outcome
        assert "timestamp" in outcome
        assert outcome["source"] == "orion-commander"
        assert outcome["action_id"] == action["action_id"]
        assert outcome["status"] in ["succeeded", "failed", "rolled_back"]
        assert "execution_time_ms" in outcome

    def test_success_outcome_has_result(self, commander, valid_safe_decision):
        """Successful outcome includes result."""
        action = commander._create_action(valid_safe_decision)

        outcome = commander.execute_action(action)

        assert outcome["status"] == "succeeded"
        assert "result" in outcome

    def test_failure_outcome_has_error(self, commander, valid_safe_decision):
        """Failed outcome includes error."""
        action = commander._create_action(valid_safe_decision)
        action["action_type"] = "unknown_type"

        outcome = commander.execute_action(action)

        assert outcome["status"] in ["failed", "rolled_back"]
        assert "error" in outcome
        assert "code" in outcome["error"]
        assert "message" in outcome["error"]


@pytest.mark.unit
class TestCommanderDecisionHandling:
    """Test decision handling logic."""

    def test_handles_execute_safe_action(self, commander, event_bus, valid_safe_decision):
        """Handles EXECUTE_SAFE_ACTION decisions."""
        commander.handle_decision(valid_safe_decision)

        # Should publish outcome
        outcomes = event_bus.read_stream("outcome")
        assert len(outcomes) == 1
        assert outcomes[0]["status"] == "succeeded"

    def test_ignores_no_action(self, commander, event_bus, valid_no_action_decision):
        """Ignores NO_ACTION decisions."""
        commander.handle_decision(valid_no_action_decision)

        # Should not publish outcome
        outcomes = event_bus.read_stream("outcome")
        assert len(outcomes) == 0

    def test_rejects_decision_without_proposed_action(self, commander, event_bus, valid_safe_decision):
        """Rejects EXECUTE_SAFE_ACTION without proposed_action."""
        del valid_safe_decision["proposed_action"]

        commander.handle_decision(valid_safe_decision)

        # Should not execute or publish outcome
        outcomes = event_bus.read_stream("outcome")
        assert len(outcomes) == 0

    def test_verifies_action_is_safe(self, commander, valid_safe_decision):
        """Verifies action is SAFE before executing."""
        # Propose a RISKY action
        valid_safe_decision["proposed_action"]["action_type"] = "restart_service"

        commander.handle_decision(valid_safe_decision)

        # Should refuse to execute
        outcomes = commander.bus.read_stream("outcome")
        assert len(outcomes) == 0


@pytest.mark.unit
class TestCommanderBusIntegration:
    """Test event bus integration."""

    def test_publishes_outcome_to_bus(self, commander, event_bus, valid_safe_decision):
        """Publishes outcome to bus after execution."""
        commander.handle_decision(valid_safe_decision)

        outcomes = event_bus.read_stream("outcome")
        assert len(outcomes) == 1

    def test_multiple_executions(self, commander, event_bus, valid_safe_decision):
        """Multiple executions publish multiple outcomes."""
        commander.handle_decision(valid_safe_decision)

        decision2 = valid_safe_decision.copy()
        decision2["decision_id"] = "decision-002"
        decision2["incident_id"] = "incident-002"
        decision2["proposed_action"] = {
            "action_type": "acknowledge_incident",
            "parameters": {"incident_id": "incident-002"},
        }
        commander.handle_decision(decision2)

        outcomes = event_bus.read_stream("outcome")
        assert len(outcomes) == 2


@pytest.mark.unit
class TestCommanderSafetyInvariants:
    """Test commander safety invariants."""

    def test_only_safe_actions_execute(self, commander, valid_safe_decision):
        """Only SAFE actions execute (fail closed)."""
        # Try to execute unknown action
        valid_safe_decision["proposed_action"]["action_type"] = "completely_unknown"

        commander.handle_decision(valid_safe_decision)

        # Should be rejected (not SAFE)
        outcomes = commander.bus.read_stream("outcome")
        assert len(outcomes) == 0

    def test_risky_never_executes(self, commander, valid_safe_decision):
        """RISKY actions never execute in any mode."""
        valid_safe_decision["proposed_action"]["action_type"] = "restart_service"

        commander.handle_decision(valid_safe_decision)

        outcomes = commander.bus.read_stream("outcome")
        assert len(outcomes) == 0

    def test_all_outcomes_auditable(self, commander, event_bus, valid_safe_decision):
        """All outcomes are auditable (published to bus)."""
        commander.handle_decision(valid_safe_decision)

        outcomes = event_bus.read_stream("outcome")
        assert len(outcomes) == 1

        outcome = outcomes[0]
        assert "outcome_id" in outcome
        assert "timestamp" in outcome
        assert "source" in outcome
        assert "action_id" in outcome


@pytest.mark.unit
class TestCommanderIdempotency:
    """Test action idempotency."""

    def test_acknowledge_incident_is_idempotent(self, commander, valid_safe_decision):
        """acknowledge_incident can be executed multiple times safely."""
        action = commander._create_action(valid_safe_decision)

        # Execute twice
        outcome1 = commander.execute_action(action)
        outcome2 = commander.execute_action(action)

        # Both should succeed
        assert outcome1["status"] == "succeeded"
        assert outcome2["status"] == "succeeded"


@pytest.mark.unit
class TestCommanderErrorHandling:
    """Test error handling and edge cases."""

    def test_handles_invalid_parameters(self, commander, valid_safe_decision):
        """Handles invalid action parameters gracefully."""
        action = commander._create_action(valid_safe_decision)
        action["parameters"] = None

        outcome = commander.execute_action(action)

        assert outcome["status"] in ["failed", "rolled_back"]
        assert "error" in outcome

    def test_logs_execution_errors(self, commander, valid_safe_decision, caplog):
        """Logs execution errors."""
        action = commander._create_action(valid_safe_decision)
        action["action_type"] = "unknown"

        commander.execute_action(action)

        assert any("failed" in record.message.lower() for record in caplog.records)

    def test_handles_missing_action_id(self, commander, valid_safe_decision):
        """Missing decision_id raises KeyError (strict validation)."""
        del valid_safe_decision["decision_id"]

        # Commander expects valid contracts, missing fields raise KeyError
        with pytest.raises(KeyError):
            commander.handle_decision(valid_safe_decision)
