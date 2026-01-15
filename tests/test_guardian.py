"""
Guardian tests.

Tests event correlation and incident detection.
"""

from datetime import datetime, timezone, timedelta
from pathlib import Path
import pytest
import fakeredis

from bus.python.orion_bus import EventBus
from core.guardian import Guardian


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
def guardian(event_bus):
    """Provide guardian instance."""
    return Guardian(event_bus, correlation_window=60)


@pytest.mark.unit
class TestGuardianInit:
    """Test guardian initialization."""

    def test_initializes_with_defaults(self, event_bus):
        """Guardian initializes with default settings."""
        guardian = Guardian(event_bus)

        assert guardian.correlation_window == 60
        assert guardian.source_name == "orion-guardian"
        assert len(guardian._event_buffer) == 0

    def test_initializes_with_custom_settings(self, event_bus):
        """Guardian accepts custom settings."""
        guardian = Guardian(
            event_bus,
            correlation_window=120,
            source_name="test-guardian",
        )

        assert guardian.correlation_window == 120
        assert guardian.source_name == "test-guardian"


@pytest.mark.unit
class TestGuardianFingerprint:
    """Test event fingerprinting for deduplication."""

    def test_calculate_fingerprint(self, guardian, valid_event_v1):
        """Fingerprint is calculated from event fields."""
        fingerprint = guardian._calculate_fingerprint(valid_event_v1)

        assert fingerprint is not None
        assert isinstance(fingerprint, str)
        assert len(fingerprint) == 16  # Truncated SHA256

    def test_same_event_same_fingerprint(self, guardian, valid_event_v1):
        """Same event produces same fingerprint."""
        fp1 = guardian._calculate_fingerprint(valid_event_v1)
        fp2 = guardian._calculate_fingerprint(valid_event_v1)

        assert fp1 == fp2

    def test_different_event_different_fingerprint(self, guardian, valid_event_v1):
        """Different event produces different fingerprint."""
        event2 = valid_event_v1.copy()
        event2["event_type"] = "service_up"

        fp1 = guardian._calculate_fingerprint(valid_event_v1)
        fp2 = guardian._calculate_fingerprint(event2)

        assert fp1 != fp2


@pytest.mark.unit
class TestGuardianIncidentType:
    """Test incident type determination."""

    def test_service_down_creates_outage(self, guardian):
        """service_down event creates service_outage incident."""
        events = [{"event_type": "service_down"}]

        incident_type = guardian._determine_incident_type(events)

        assert incident_type == "service_outage"

    def test_metric_threshold_creates_anomaly(self, guardian):
        """metric_threshold_exceeded creates metric_anomaly."""
        events = [{"event_type": "metric_threshold_exceeded"}]

        incident_type = guardian._determine_incident_type(events)

        assert incident_type == "metric_anomaly"

    def test_edge_offline_creates_failure(self, guardian):
        """edge_device_offline creates edge_device_failure."""
        events = [{"event_type": "edge_device_offline"}]

        incident_type = guardian._determine_incident_type(events)

        assert incident_type == "edge_device_failure"

    def test_unknown_creates_correlation(self, guardian):
        """Unknown event type creates correlation_detected."""
        events = [{"event_type": "unknown_type"}]

        incident_type = guardian._determine_incident_type(events)

        assert incident_type == "correlation_detected"


@pytest.mark.unit
class TestGuardianSeverity:
    """Test incident severity determination."""

    def test_never_escalates_severity(self, guardian):
        """Incident severity never exceeds event severity."""
        # All info events -> low incident
        events = [{"severity": "info"}]
        assert guardian._determine_severity(events) == "low"

        # Warning event -> medium incident
        events = [{"severity": "warning"}]
        assert guardian._determine_severity(events) == "medium"

        # Error event -> high incident
        events = [{"severity": "error"}]
        assert guardian._determine_severity(events) == "high"

        # Critical event -> critical incident
        events = [{"severity": "critical"}]
        assert guardian._determine_severity(events) == "critical"

    def test_uses_highest_event_severity(self, guardian):
        """Multiple events use highest severity."""
        events = [
            {"severity": "info"},
            {"severity": "warning"},
            {"severity": "error"},
        ]

        severity = guardian._determine_severity(events)

        assert severity == "high"  # From error


@pytest.mark.unit
class TestGuardianCorrelation:
    """Test event correlation logic."""

    def test_should_create_incident_for_warning(self, guardian):
        """Warning events should create incident."""
        events = [{"severity": "warning"}]

        assert guardian._should_create_incident(events) is True

    def test_should_not_create_incident_for_info(self, guardian):
        """Info events should not create incident."""
        events = [{"severity": "info"}]

        assert guardian._should_create_incident(events) is False

    def test_should_create_incident_for_error(self, guardian):
        """Error events should create incident."""
        events = [{"severity": "error"}]

        assert guardian._should_create_incident(events) is True

    def test_empty_events_no_incident(self, guardian):
        """Empty event list should not create incident."""
        assert guardian._should_create_incident([]) is False


@pytest.mark.unit
class TestGuardianIncidentCreation:
    """Test incident creation."""

    def test_create_incident_from_events(self, guardian, valid_event_v1):
        """Incident is created from correlated events."""
        now = datetime.now(timezone.utc)
        start = now - timedelta(seconds=60)

        incident = guardian._create_incident([valid_event_v1], start, now)

        assert incident["version"] == "1.0"
        assert "incident_id" in incident
        assert incident["source"] == "orion-guardian"
        assert incident["state"] == "open"
        assert valid_event_v1["event_id"] in incident["event_ids"]
        assert incident["correlation_window"]["start"] == start.isoformat()
        assert incident["correlation_window"]["end"] == now.isoformat()

    def test_incident_includes_all_event_ids(self, guardian, valid_event_v1):
        """Incident includes all correlated event IDs."""
        event2 = valid_event_v1.copy()
        event2["event_id"] = "different-id"

        now = datetime.now(timezone.utc)
        start = now - timedelta(seconds=60)

        incident = guardian._create_incident([valid_event_v1, event2], start, now)

        assert len(incident["event_ids"]) == 2
        assert valid_event_v1["event_id"] in incident["event_ids"]
        assert event2["event_id"] in incident["event_ids"]


@pytest.mark.unit
class TestGuardianEventHandling:
    """Test event handling and correlation."""

    def test_handle_event_adds_to_buffer(self, guardian, valid_event_v1):
        """Handling event adds it to buffer."""
        assert len(guardian._event_buffer) == 0

        guardian.handle_event(valid_event_v1)

        assert len(guardian._event_buffer) == 1
        assert guardian._event_buffer[0] == valid_event_v1

    def test_handle_event_bounds_buffer_size(self, guardian, valid_event_v1):
        """Event buffer is bounded to 100 events."""
        # Add 150 events
        for i in range(150):
            event = valid_event_v1.copy()
            event["event_id"] = f"event-{i}"
            guardian.handle_event(event)

        assert len(guardian._event_buffer) == 100

    def test_handle_warning_event_creates_incident(self, guardian, event_bus):
        """Warning event triggers incident creation and publishing."""
        event = {
            "version": "1.0",
            "event_id": str(__import__("uuid").uuid4()),
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": "test",
            "event_type": "service_down",
            "severity": "warning",
            "data": {},
        }

        guardian.handle_event(event)

        # Check that incident was published
        incidents = event_bus.read_stream("incident")
        assert len(incidents) == 1
        assert incidents[0]["incident_type"] == "service_outage"
        assert incidents[0]["severity"] == "medium"

    def test_duplicate_incident_not_created(self, guardian, event_bus):
        """Duplicate incident (same fingerprint) is not created."""
        event = {
            "version": "1.0",
            "event_id": str(__import__("uuid").uuid4()),
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": "test",
            "event_type": "service_down",
            "severity": "warning",
            "data": {},
        }

        # Handle same event twice
        guardian.handle_event(event)
        guardian.handle_event(event)

        # Only one incident should be created
        incidents = event_bus.read_stream("incident")
        assert len(incidents) == 1
