"""
Circuit breaker for action execution.

Stops execution after repeated failures to prevent damage.
"""

import logging
import time
from collections import defaultdict
from typing import Dict, List, Tuple


logger = logging.getLogger(__name__)


class CircuitBreaker:
    """
    Circuit breaker to prevent repeated failing actions.

    Invariants:
    - If failure threshold exceeded, circuit opens
    - While circuit open, ALL execution blocked
    - Circuit closes after timeout expires
    - State is observable and auditable
    """

    def __init__(
        self,
        failure_threshold: int = 3,
        failure_window: int = 300,
        circuit_open_duration: int = 600,
    ):
        """
        Initialize circuit breaker.

        Args:
            failure_threshold: Number of failures before opening (default: 3)
            failure_window: Time window for counting failures in seconds (default: 300)
            circuit_open_duration: How long circuit stays open in seconds (default: 600)
        """
        self.failure_threshold = failure_threshold
        self.failure_window = failure_window
        self.circuit_open_duration = circuit_open_duration

        # Per action type: list of failure timestamps
        self._failures: Dict[str, List[float]] = defaultdict(list)

        # Per action type: timestamp when circuit opened
        self._circuit_opened: Dict[str, float] = {}

    def record_failure(self, action_type: str) -> None:
        """
        Record action failure.

        Args:
            action_type: Type of action that failed
        """
        now = time.time()
        self._failures[action_type].append(now)

        # Clean old failures outside window
        cutoff = now - self.failure_window
        self._failures[action_type] = [
            ts for ts in self._failures[action_type] if ts > cutoff
        ]

        # Check if threshold exceeded
        if len(self._failures[action_type]) >= self.failure_threshold:
            if action_type not in self._circuit_opened:
                logger.error(
                    f"Circuit breaker OPENED for {action_type}: "
                    f"{len(self._failures[action_type])} failures in "
                    f"{self.failure_window}s"
                )
                self._circuit_opened[action_type] = now

    def record_success(self, action_type: str) -> None:
        """
        Record action success.

        Clears failure history for that action type.

        Args:
            action_type: Type of action that succeeded
        """
        self._failures[action_type].clear()
        logger.debug(f"Circuit breaker: success recorded for {action_type}")

    def is_open(self, action_type: str) -> bool:
        """
        Check if circuit is open for action type.

        Args:
            action_type: Type of action

        Returns:
            True if circuit is open (execution blocked), False otherwise
        """
        if action_type not in self._circuit_opened:
            return False

        opened_at = self._circuit_opened[action_type]
        elapsed = time.time() - opened_at

        if elapsed >= self.circuit_open_duration:
            # Circuit timeout expired, close circuit
            logger.info(
                f"Circuit breaker CLOSED for {action_type} after {elapsed:.1f}s"
            )
            del self._circuit_opened[action_type]
            self._failures[action_type].clear()
            return False

        # Circuit still open
        return True

    def get_state(self, action_type: str) -> Dict[str, any]:
        """
        Get circuit breaker state for action type.

        Args:
            action_type: Type of action

        Returns:
            State dictionary with circuit status and failure count
        """
        is_open = self.is_open(action_type)
        failure_count = len(self._failures.get(action_type, []))

        state = {
            "action_type": action_type,
            "circuit_open": is_open,
            "failure_count": failure_count,
            "failure_threshold": self.failure_threshold,
        }

        if is_open:
            opened_at = self._circuit_opened[action_type]
            elapsed = time.time() - opened_at
            remaining = max(0.0, self.circuit_open_duration - elapsed)
            state["circuit_opened_at"] = opened_at
            state["circuit_remaining_seconds"] = remaining

        return state

    def clear(self) -> None:
        """Clear all circuit breaker state (for testing)."""
        self._failures.clear()
        self._circuit_opened.clear()
