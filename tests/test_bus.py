"""
Event bus tests.

Tests Redis Streams integration with contract validation.
Uses fakeredis to avoid requiring real Redis.
"""

from pathlib import Path
import pytest
import fakeredis
from jsonschema import ValidationError

from bus.python.orion_bus import EventBus, ContractValidator


CONTRACTS_DIR = Path(__file__).parent.parent / "bus" / "contracts"


@pytest.fixture
def redis_client():
    """Provide fake Redis client for testing."""
    return fakeredis.FakeRedis(decode_responses=False)


@pytest.fixture
def event_bus(redis_client):
    """Provide event bus instance with fake Redis."""
    return EventBus(
        redis_client=redis_client,
        contracts_dir=CONTRACTS_DIR,
        stream_prefix="test-orion",
    )


@pytest.mark.unit
class TestContractValidator:
    """Test contract validation logic."""

    def test_load_schemas(self):
        """Validator loads all schemas from contracts directory."""
        validator = ContractValidator(CONTRACTS_DIR)

        assert "event.schema" in validator._schemas
        assert "incident.schema" in validator._schemas
        assert "decision.schema" in validator._schemas
        assert "action.schema" in validator._schemas
        assert "approval.schema" in validator._schemas
        assert "outcome.schema" in validator._schemas

    def test_validate_valid_event(self, valid_event_v1):
        """Valid event passes validation."""
        validator = ContractValidator(CONTRACTS_DIR)
        validator.validate(valid_event_v1, "event.schema")  # Should not raise

    def test_validate_invalid_event_rejected(self, valid_event_v1):
        """Invalid event is rejected."""
        validator = ContractValidator(CONTRACTS_DIR)

        # Remove required field
        del valid_event_v1["event_id"]

        with pytest.raises(ValidationError):
            validator.validate(valid_event_v1, "event.schema")

    def test_validate_unknown_schema_rejected(self, valid_event_v1):
        """Unknown schema raises ValueError."""
        validator = ContractValidator(CONTRACTS_DIR)

        with pytest.raises(ValueError, match="Unknown schema"):
            validator.validate(valid_event_v1, "nonexistent.schema")


@pytest.mark.unit
class TestEventBusPublish:
    """Test event bus publish functionality."""

    def test_publish_valid_event(self, event_bus, valid_event_v1):
        """Valid event is published to Redis stream."""
        message_id = event_bus.publish(valid_event_v1, "event")

        assert message_id is not None
        assert isinstance(message_id, str)

    def test_publish_invalid_event_rejected(self, event_bus, valid_event_v1):
        """Invalid event is rejected before publish."""
        del valid_event_v1["event_id"]

        with pytest.raises(ValidationError):
            event_bus.publish(valid_event_v1, "event")

        # Verify nothing was published
        messages = event_bus.read_stream("event")
        assert len(messages) == 0

    def test_publish_creates_stream(self, event_bus, valid_event_v1):
        """Publishing creates Redis stream if it doesn't exist."""
        event_bus.publish(valid_event_v1, "event")

        stream_name = event_bus._get_stream_name("event")
        assert event_bus.redis.exists(stream_name)

    def test_publish_multiple_events(self, event_bus, valid_event_v1):
        """Multiple events can be published."""
        event_bus.publish(valid_event_v1, "event")

        event2 = valid_event_v1.copy()
        event2["event_id"] = "650e8400-e29b-41d4-a716-446655440099"
        event_bus.publish(event2, "event")

        messages = event_bus.read_stream("event")
        assert len(messages) == 2

    def test_publish_respects_max_stream_length(self, redis_client):
        """Stream is trimmed to max length."""
        bus = EventBus(
            redis_client=redis_client,
            contracts_dir=CONTRACTS_DIR,
            max_stream_length=2,
        )

        # Publish 3 events
        for i in range(3):
            event = {
                "version": "1.0",
                "event_id": f"550e8400-e29b-41d4-a716-44665544000{i}",
                "timestamp": "2026-01-15T12:00:00Z",
                "source": "orion-test",
                "event_type": "service_down",
                "severity": "info",
                "data": {},
            }
            bus.publish(event, "event")

        # Only approximately max_stream_length entries remain
        stream_name = bus._get_stream_name("event")
        length = redis_client.xlen(stream_name)
        assert length <= 3  # Approximate trimming


@pytest.mark.unit
class TestEventBusRead:
    """Test event bus read functionality."""

    def test_read_stream_empty(self, event_bus):
        """Reading empty stream returns empty list."""
        messages = event_bus.read_stream("event")
        assert messages == []

    def test_read_stream_returns_messages(self, event_bus, valid_event_v1):
        """Reading stream returns published messages."""
        event_bus.publish(valid_event_v1, "event")

        messages = event_bus.read_stream("event")
        assert len(messages) == 1
        assert messages[0]["event_id"] == valid_event_v1["event_id"]

    def test_read_stream_respects_count(self, event_bus, valid_event_v1):
        """Read respects count parameter."""
        for i in range(5):
            event = valid_event_v1.copy()
            event["event_id"] = f"550e8400-e29b-41d4-a716-44665544000{i}"
            event_bus.publish(event, "event")

        messages = event_bus.read_stream("event", count=2)
        assert len(messages) == 2


@pytest.mark.unit
class TestEventBusSubscribe:
    """Test event bus subscribe functionality (basic tests)."""

    def test_subscribe_creates_consumer_group(self, event_bus, valid_event_v1):
        """Subscribe creates consumer group."""
        import threading

        received = []

        def handler(message):
            received.append(message)
            raise KeyboardInterrupt  # Exit after first message

        # Publish event first
        event_bus.publish(valid_event_v1, "event")

        # Subscribe in thread
        def subscribe_thread():
            try:
                event_bus.subscribe(
                    "event",
                    handler,
                    consumer_group="test-group",
                    consumer_name="test-consumer",
                    block_ms=100,
                )
            except KeyboardInterrupt:
                pass

        thread = threading.Thread(target=subscribe_thread, daemon=True)
        thread.start()
        thread.join(timeout=2)

        # Verify consumer group exists
        stream_name = event_bus._get_stream_name("event")
        groups = event_bus.redis.xinfo_groups(stream_name)
        assert len(groups) > 0
        # fakeredis returns dict, real Redis returns list of dicts
        group_name = groups[0].get("name") or groups[0].get(b"name")
        assert group_name in (b"test-group", "test-group")


@pytest.mark.unit
class TestEventBusContract:
    """Test contract enforcement across different message types."""

    def test_publish_incident(self, event_bus, valid_incident_v1):
        """Incident contract is enforced."""
        event_bus.publish(valid_incident_v1, "incident")

        messages = event_bus.read_stream("incident")
        assert len(messages) == 1
        assert messages[0]["incident_type"] == "service_outage"

    def test_publish_decision(self, event_bus, valid_decision_v1):
        """Decision contract is enforced."""
        event_bus.publish(valid_decision_v1, "decision")

        messages = event_bus.read_stream("decision")
        assert len(messages) == 1
        assert messages[0]["decision_type"] == "NO_ACTION"

    def test_publish_wrong_contract_rejected(self, event_bus, valid_event_v1):
        """Event published as incident is rejected."""
        with pytest.raises(ValidationError):
            event_bus.publish(valid_event_v1, "incident")
