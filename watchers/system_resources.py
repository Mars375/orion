"""
System resource watcher.

Observes CPU, memory, and disk usage and emits events.
N0 mode: Pure observation, no thresholds, no actions.
"""

import logging
import time
import uuid
from datetime import datetime, timezone
from typing import Dict, Any
from pathlib import Path

import psutil

from bus.python.orion_bus import EventBus


logger = logging.getLogger(__name__)


class SystemResourceWatcher:
    """
    Watches system resources and emits observation events.

    N0 Invariants:
    - No thresholds (just observe)
    - No heuristics
    - No decisions
    - Emit fewer events rather than too many
    """

    def __init__(
        self,
        event_bus: EventBus,
        poll_interval: int = 60,
        source_name: str = "orion-watcher-system",
    ):
        """
        Initialize system resource watcher.

        Args:
            event_bus: Event bus to publish to
            poll_interval: Seconds between polls (default: 60, minimum: 30)
            source_name: Source identifier for events
        """
        self.bus = event_bus
        self.poll_interval = max(poll_interval, 30)  # Minimum 30 seconds
        self.source_name = source_name

    def _create_event(
        self,
        event_type: str,
        severity: str,
        data: Dict[str, Any],
    ) -> Dict[str, Any]:
        """
        Create event contract.

        Args:
            event_type: Event type
            severity: Severity level
            data: Event-specific data

        Returns:
            Event matching event.schema.json
        """
        return {
            "version": "1.0",
            "event_id": str(uuid.uuid4()),
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": self.source_name,
            "event_type": event_type,
            "severity": severity,
            "data": data,
        }

    def _get_cpu_stats(self) -> Dict[str, Any]:
        """Get CPU usage statistics."""
        cpu_percent = psutil.cpu_percent(interval=1)
        cpu_count = psutil.cpu_count()

        return {
            "cpu_percent": cpu_percent,
            "cpu_count": cpu_count,
        }

    def _get_memory_stats(self) -> Dict[str, Any]:
        """Get memory usage statistics."""
        mem = psutil.virtual_memory()

        return {
            "total_bytes": mem.total,
            "available_bytes": mem.available,
            "used_bytes": mem.used,
            "percent": mem.percent,
        }

    def _get_disk_stats(self) -> Dict[str, Any]:
        """Get disk usage statistics for root partition."""
        disk = psutil.disk_usage("/")

        return {
            "total_bytes": disk.total,
            "used_bytes": disk.used,
            "free_bytes": disk.free,
            "percent": disk.percent,
        }

    def poll_once(self) -> None:
        """
        Poll system resources once and emit events.

        Emits a single event with all resource data.
        No thresholds - just raw observation.
        """
        try:
            # Gather all stats
            cpu_stats = self._get_cpu_stats()
            memory_stats = self._get_memory_stats()
            disk_stats = self._get_disk_stats()

            # Create single event with all resource data
            event = self._create_event(
                event_type="edge_telemetry",  # Using existing event_type
                severity="info",  # Always info - no thresholds
                data={
                    "resource_type": "system",
                    "cpu": cpu_stats,
                    "memory": memory_stats,
                    "disk": disk_stats,
                },
            )

            # Publish to bus
            self.bus.publish(event, "event")

            logger.debug(
                f"Published system resource event: CPU={cpu_stats['cpu_percent']}% "
                f"MEM={memory_stats['percent']}% DISK={disk_stats['percent']}%"
            )

        except Exception as e:
            logger.error(f"Error polling system resources: {e}", exc_info=True)
            # Do not retry - fail closed

    def run(self) -> None:
        """
        Run watcher loop.

        Polls system resources at configured interval.
        """
        logger.info(
            f"Starting system resource watcher (poll_interval={self.poll_interval}s)"
        )

        while True:
            try:
                self.poll_once()
                time.sleep(self.poll_interval)

            except KeyboardInterrupt:
                logger.info("Stopping system resource watcher")
                break
            except Exception as e:
                logger.error(f"Error in watcher loop: {e}", exc_info=True)
                time.sleep(self.poll_interval)  # Continue after errors
