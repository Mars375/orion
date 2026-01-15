"""
ORION Guardian - Event correlation and incident detection.

Subscribes to events, correlates them, emits incidents.
N0 mode: No decisions, no actions, pure correlation.
"""

import hashlib
import logging
import uuid
from collections import defaultdict
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, List, Optional

from bus.python.orion_bus import EventBus


logger = logging.getLogger(__name__)


class Guardian:
    """
    Correlates events into incidents.

    Responsibilities:
    - Correlate related events
    - Deduplicate using fingerprints
    - Enrich incidents with context
    - Never escalate severity beyond observed data

    N0 Invariants:
    - No action decisions
    - No execution
    - Outputs only incident contracts
    """

    def __init__(
        self,
        event_bus: EventBus,
        correlation_window: int = 60,
        source_name: str = "orion-guardian",
    ):
        """
        Initialize guardian.

        Args:
            event_bus: Event bus for pub/sub
            correlation_window: Seconds to correlate events (default: 60)
            source_name: Source identifier for incidents
        """
        self.bus = event_bus
        self.correlation_window = correlation_window
        self.source_name = source_name

        # Correlation state (in-memory for Phase 1)
        self._event_buffer: List[Dict[str, Any]] = []
        self._incident_fingerprints: Dict[str, str] = {}  # fingerprint -> incident_id

    def _calculate_fingerprint(self, event: Dict[str, Any]) -> str:
        """
        Calculate fingerprint for event deduplication.

        Args:
            event: Event to fingerprint

        Returns:
            Fingerprint hash
        """
        # Fingerprint based on event type and key data fields
        fingerprint_data = {
            "event_type": event.get("event_type"),
            "source": event.get("source"),
            "severity": event.get("severity"),
        }

        # Include relevant data fields for deduplication
        data = event.get("data", {})
        if "service_name" in data:
            fingerprint_data["service_name"] = data["service_name"]
        if "resource_type" in data:
            fingerprint_data["resource_type"] = data["resource_type"]

        # Create hash
        fingerprint_str = str(sorted(fingerprint_data.items()))
        return hashlib.sha256(fingerprint_str.encode()).hexdigest()[:16]

    def _determine_incident_type(self, events: List[Dict[str, Any]]) -> str:
        """
        Determine incident type from correlated events.

        Args:
            events: Correlated events

        Returns:
            Incident type
        """
        # For Phase 1, simple mapping
        event_types = {e.get("event_type") for e in events}

        if "service_down" in event_types:
            return "service_outage"
        elif "metric_threshold_exceeded" in event_types:
            return "metric_anomaly"
        elif "edge_device_offline" in event_types:
            return "edge_device_failure"
        else:
            return "correlation_detected"

    def _determine_severity(self, events: List[Dict[str, Any]]) -> str:
        """
        Determine incident severity from events.

        Never escalate beyond observed data.

        Args:
            events: Correlated events

        Returns:
            Severity (low, medium, high, critical)
        """
        severities = [e.get("severity", "info") for e in events]

        # Map event severity to incident severity
        # Never escalate beyond observed
        if "critical" in severities:
            return "critical"
        elif "error" in severities:
            return "high"
        elif "warning" in severities:
            return "medium"
        else:
            return "low"

    def _create_incident(
        self,
        events: List[Dict[str, Any]],
        correlation_start: datetime,
        correlation_end: datetime,
    ) -> Dict[str, Any]:
        """
        Create incident from correlated events.

        Args:
            events: Correlated events
            correlation_start: Start of correlation window
            correlation_end: End of correlation window

        Returns:
            Incident matching incident.schema.json
        """
        event_ids = [e["event_id"] for e in events]
        incident_type = self._determine_incident_type(events)
        severity = self._determine_severity(events)

        # Create description from events
        description = f"Correlated {len(events)} event(s): {incident_type}"

        incident = {
            "version": "1.0",
            "incident_id": str(uuid.uuid4()),
            "timestamp": correlation_end.isoformat(),
            "source": self.source_name,
            "incident_type": incident_type,
            "severity": severity,
            "event_ids": event_ids,
            "correlation_window": {
                "start": correlation_start.isoformat(),
                "end": correlation_end.isoformat(),
            },
            "state": "open",
            "description": description,
        }

        return incident

    def _should_create_incident(self, events: List[Dict[str, Any]]) -> bool:
        """
        Determine if events should create an incident.

        Args:
            events: Potential events for incident

        Returns:
            True if incident should be created
        """
        if not events:
            return False

        # For Phase 1: create incident for any non-info events
        # This is conservative - only correlate meaningful events
        for event in events:
            if event.get("severity") in ("warning", "error", "critical"):
                return True

        return False

    def correlate_events(self) -> Optional[Dict[str, Any]]:
        """
        Correlate buffered events into incident.

        Returns:
            Incident if correlation detected, None otherwise
        """
        if not self._should_create_incident(self._event_buffer):
            return None

        now = datetime.now(timezone.utc)
        window_start = now - timedelta(seconds=self.correlation_window)

        # Filter events within correlation window
        recent_events = [
            e
            for e in self._event_buffer
            if datetime.fromisoformat(e["timestamp"].replace("Z", "+00:00"))
            >= window_start
        ]

        if not recent_events:
            return None

        # Check fingerprint for deduplication
        fingerprint = self._calculate_fingerprint(recent_events[0])
        if fingerprint in self._incident_fingerprints:
            logger.debug(f"Incident already exists for fingerprint {fingerprint}")
            return None

        # Create incident
        incident = self._create_incident(recent_events, window_start, now)

        # Store fingerprint
        self._incident_fingerprints[fingerprint] = incident["incident_id"]

        logger.info(
            f"Created incident {incident['incident_id']}: "
            f"{incident['incident_type']} (severity={incident['severity']}, "
            f"events={len(recent_events)})"
        )

        return incident

    def handle_event(self, event: Dict[str, Any]) -> None:
        """
        Handle incoming event.

        Args:
            event: Event from bus
        """
        logger.debug(f"Received event {event.get('event_id')} type={event.get('event_type')}")

        # Add to buffer
        self._event_buffer.append(event)

        # Keep buffer bounded (last 100 events)
        if len(self._event_buffer) > 100:
            self._event_buffer = self._event_buffer[-100:]

        # Attempt correlation
        incident = self.correlate_events()

        if incident:
            # Publish incident to bus
            try:
                self.bus.publish(incident, "incident")
                logger.info(f"Published incident {incident['incident_id']}")
            except Exception as e:
                logger.error(f"Failed to publish incident: {e}", exc_info=True)

    def run(self) -> None:
        """
        Run guardian subscription loop.

        Subscribes to events and correlates them into incidents.
        """
        logger.info("Starting guardian event correlation")

        self.bus.subscribe(
            contract_type="event",
            handler=self.handle_event,
            consumer_group="guardian",
            consumer_name="guardian-1",
        )
