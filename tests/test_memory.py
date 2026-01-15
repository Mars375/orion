"""
Memory store tests.

Tests append-only persistence and immutability.
"""

import json
from pathlib import Path
import pytest

from core.memory import MemoryStore


@pytest.fixture
def temp_storage(tmp_path):
    """Provide temporary storage directory."""
    return tmp_path / "memory"


@pytest.fixture
def memory_store(temp_storage):
    """Provide memory store instance."""
    return MemoryStore(temp_storage)


@pytest.mark.unit
class TestMemoryStoreInit:
    """Test memory store initialization."""

    def test_creates_storage_directory(self, temp_storage):
        """Memory store creates storage directory if it doesn't exist."""
        assert not temp_storage.exists()

        store = MemoryStore(temp_storage)

        assert temp_storage.exists()
        assert temp_storage.is_dir()

    def test_initializes_log_paths(self, memory_store):
        """Memory store initializes log file paths."""
        assert memory_store.event_log.name == "events.jsonl"
        assert memory_store.incident_log.name == "incidents.jsonl"
        assert memory_store.decision_log.name == "decisions.jsonl"


@pytest.mark.unit
class TestMemoryStoreEvents:
    """Test event storage."""

    def test_store_event(self, memory_store, valid_event_v1):
        """Event is appended to events.jsonl."""
        memory_store.store_event(valid_event_v1)

        assert memory_store.event_log.exists()

        # Verify JSONL format
        with open(memory_store.event_log) as f:
            line = f.readline()
            stored = json.loads(line)
            assert stored["event_id"] == valid_event_v1["event_id"]

    def test_store_multiple_events(self, memory_store, valid_event_v1):
        """Multiple events are appended."""
        memory_store.store_event(valid_event_v1)

        event2 = valid_event_v1.copy()
        event2["event_id"] = "different-id"
        memory_store.store_event(event2)

        events = memory_store.read_events()
        assert len(events) == 2
        assert events[0]["event_id"] == valid_event_v1["event_id"]
        assert events[1]["event_id"] == "different-id"

    def test_read_events_empty(self, memory_store):
        """Reading empty log returns empty list."""
        events = memory_store.read_events()
        assert events == []

    def test_read_events_with_limit(self, memory_store, valid_event_v1):
        """Read respects limit parameter."""
        for i in range(5):
            event = valid_event_v1.copy()
            event["event_id"] = f"event-{i}"
            memory_store.store_event(event)

        events = memory_store.read_events(limit=2)
        assert len(events) == 2

    def test_read_events_with_since(self, memory_store, valid_event_v1):
        """Read respects since parameter."""
        event1 = valid_event_v1.copy()
        event1["timestamp"] = "2026-01-01T00:00:00Z"
        memory_store.store_event(event1)

        event2 = valid_event_v1.copy()
        event2["event_id"] = "event-2"
        event2["timestamp"] = "2026-01-02T00:00:00Z"
        memory_store.store_event(event2)

        events = memory_store.read_events(since="2026-01-02T00:00:00Z")
        assert len(events) == 1
        assert events[0]["event_id"] == "event-2"

    def test_count_events(self, memory_store, valid_event_v1):
        """Count returns correct number of events."""
        assert memory_store.count_events() == 0

        memory_store.store_event(valid_event_v1)
        assert memory_store.count_events() == 1

        memory_store.store_event(valid_event_v1)
        assert memory_store.count_events() == 2


@pytest.mark.unit
class TestMemoryStoreIncidents:
    """Test incident storage."""

    def test_store_incident(self, memory_store, valid_incident_v1):
        """Incident is appended to incidents.jsonl."""
        memory_store.store_incident(valid_incident_v1)

        assert memory_store.incident_log.exists()

        incidents = memory_store.read_incidents()
        assert len(incidents) == 1
        assert incidents[0]["incident_id"] == valid_incident_v1["incident_id"]

    def test_count_incidents(self, memory_store, valid_incident_v1):
        """Count returns correct number of incidents."""
        assert memory_store.count_incidents() == 0

        memory_store.store_incident(valid_incident_v1)
        assert memory_store.count_incidents() == 1


@pytest.mark.unit
class TestMemoryStoreDecisions:
    """Test decision storage."""

    def test_store_decision(self, memory_store, valid_decision_v1):
        """Decision is appended to decisions.jsonl."""
        memory_store.store_decision(valid_decision_v1)

        assert memory_store.decision_log.exists()

        decisions = memory_store.read_decisions()
        assert len(decisions) == 1
        assert decisions[0]["decision_id"] == valid_decision_v1["decision_id"]

    def test_count_decisions(self, memory_store, valid_decision_v1):
        """Count returns correct number of decisions."""
        assert memory_store.count_decisions() == 0

        memory_store.store_decision(valid_decision_v1)
        assert memory_store.count_decisions() == 1


@pytest.mark.unit
class TestMemoryStoreImmutability:
    """Test append-only invariants."""

    def test_append_only_no_updates(self, memory_store, valid_event_v1):
        """Storing same event ID twice creates duplicate (no update)."""
        memory_store.store_event(valid_event_v1)
        memory_store.store_event(valid_event_v1)

        events = memory_store.read_events()
        assert len(events) == 2  # Both stored, no update

    def test_separate_logs_for_contracts(self, memory_store, valid_event_v1, valid_incident_v1):
        """Different contract types use separate log files."""
        memory_store.store_event(valid_event_v1)
        memory_store.store_incident(valid_incident_v1)

        assert memory_store.count_events() == 1
        assert memory_store.count_incidents() == 1

        # Events and incidents in separate files
        assert memory_store.event_log.exists()
        assert memory_store.incident_log.exists()
        assert memory_store.event_log != memory_store.incident_log
