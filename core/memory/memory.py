"""
ORION Memory - Immutable audit trail.

Append-only persistence for events, incidents, and decisions.
"""

import json
import logging
from pathlib import Path
from typing import Dict, Any, List, Optional
from datetime import datetime, timezone


logger = logging.getLogger(__name__)


class MemoryStore:
    """
    Append-only storage for ORION audit trail.

    Invariants:
    - Append-only (no updates or deletes)
    - JSONL format (one JSON object per line)
    - Never mutates historical data
    - Designed for auditability, not performance
    """

    def __init__(self, storage_dir: Path):
        """
        Initialize memory store.

        Args:
            storage_dir: Directory to store JSONL files
        """
        self.storage_dir = storage_dir
        self.storage_dir.mkdir(parents=True, exist_ok=True)

        # Separate files for each contract type
        self.event_log = self.storage_dir / "events.jsonl"
        self.incident_log = self.storage_dir / "incidents.jsonl"
        self.decision_log = self.storage_dir / "decisions.jsonl"

        logger.info(f"Memory store initialized at {self.storage_dir}")

    def _append(self, log_file: Path, data: Dict[str, Any]) -> None:
        """
        Append entry to log file.

        Args:
            log_file: Log file path
            data: Data to append
        """
        with open(log_file, "a") as f:
            f.write(json.dumps(data) + "\n")

    def store_event(self, event: Dict[str, Any]) -> None:
        """
        Store event in audit trail.

        Args:
            event: Event matching event.schema.json
        """
        self._append(self.event_log, event)
        logger.debug(f"Stored event {event.get('event_id')} to audit trail")

    def store_incident(self, incident: Dict[str, Any]) -> None:
        """
        Store incident in audit trail.

        Args:
            incident: Incident matching incident.schema.json
        """
        self._append(self.incident_log, incident)
        logger.debug(f"Stored incident {incident.get('incident_id')} to audit trail")

    def store_decision(self, decision: Dict[str, Any]) -> None:
        """
        Store decision in audit trail.

        Args:
            decision: Decision matching decision.schema.json
        """
        self._append(self.decision_log, decision)
        logger.debug(f"Stored decision {decision.get('decision_id')} to audit trail")

    def read_events(
        self,
        limit: Optional[int] = None,
        since: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Read events from audit trail.

        Args:
            limit: Maximum entries to return
            since: ISO timestamp to filter from

        Returns:
            List of events
        """
        return self._read_log(self.event_log, limit, since)

    def read_incidents(
        self,
        limit: Optional[int] = None,
        since: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Read incidents from audit trail.

        Args:
            limit: Maximum entries to return
            since: ISO timestamp to filter from

        Returns:
            List of incidents
        """
        return self._read_log(self.incident_log, limit, since)

    def read_decisions(
        self,
        limit: Optional[int] = None,
        since: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Read decisions from audit trail.

        Args:
            limit: Maximum entries to return
            since: ISO timestamp to filter from

        Returns:
            List of decisions
        """
        return self._read_log(self.decision_log, limit, since)

    def _read_log(
        self,
        log_file: Path,
        limit: Optional[int] = None,
        since: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Read entries from log file.

        Args:
            log_file: Log file path
            limit: Maximum entries to return
            since: ISO timestamp to filter from

        Returns:
            List of entries
        """
        if not log_file.exists():
            return []

        entries = []
        with open(log_file) as f:
            for line in f:
                if not line.strip():
                    continue

                entry = json.loads(line)

                # Filter by timestamp if specified
                if since and entry.get("timestamp", "") < since:
                    continue

                entries.append(entry)

                # Respect limit
                if limit and len(entries) >= limit:
                    break

        return entries

    def count_events(self) -> int:
        """Count total events in audit trail."""
        return self._count_lines(self.event_log)

    def count_incidents(self) -> int:
        """Count total incidents in audit trail."""
        return self._count_lines(self.incident_log)

    def count_decisions(self) -> int:
        """Count total decisions in audit trail."""
        return self._count_lines(self.decision_log)

    def _count_lines(self, log_file: Path) -> int:
        """Count lines in log file."""
        if not log_file.exists():
            return 0

        count = 0
        with open(log_file) as f:
            for line in f:
                if line.strip():
                    count += 1
        return count
