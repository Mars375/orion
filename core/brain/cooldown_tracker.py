"""
Cooldown tracker for action execution.

Prevents rapid repeated execution of the same action.
"""

import logging
import time
from typing import Dict, Tuple, Optional


logger = logging.getLogger(__name__)


class CooldownTracker:
    """
    Tracks cooldowns for actions to prevent loops.

    Invariants:
    - Same action cannot execute repeatedly within cooldown period
    - Cooldowns are per (action_type, applies_per_key) tuple
    - If cooldown active, action MUST NOT execute
    """

    def __init__(self):
        """Initialize cooldown tracker."""
        # Key: (action_type, applies_per_key), Value: timestamp of last execution
        self._last_execution: Dict[Tuple[str, str], float] = {}

    def check_cooldown(
        self,
        action_type: str,
        cooldown_seconds: int,
        applies_per_key: Optional[str] = None,
    ) -> bool:
        """
        Check if action is in cooldown.

        Args:
            action_type: Type of action
            cooldown_seconds: Cooldown duration in seconds
            applies_per_key: Optional key for per-entity cooldown

        Returns:
            True if action can execute (not in cooldown), False otherwise
        """
        if cooldown_seconds == 0:
            # No cooldown
            return True

        key = (action_type, applies_per_key or "")
        last_exec = self._last_execution.get(key)

        if last_exec is None:
            # Never executed
            return True

        elapsed = time.time() - last_exec
        if elapsed >= cooldown_seconds:
            # Cooldown expired
            return True

        # Still in cooldown
        logger.warning(
            f"Action {action_type} in cooldown: {elapsed:.1f}s elapsed, "
            f"{cooldown_seconds}s required"
        )
        return False

    def record_execution(
        self,
        action_type: str,
        applies_per_key: Optional[str] = None,
    ) -> None:
        """
        Record that action was executed.

        Args:
            action_type: Type of action
            applies_per_key: Optional key for per-entity cooldown
        """
        key = (action_type, applies_per_key or "")
        self._last_execution[key] = time.time()
        logger.debug(f"Recorded execution of {action_type} (key={applies_per_key})")

    def get_remaining_cooldown(
        self,
        action_type: str,
        cooldown_seconds: int,
        applies_per_key: Optional[str] = None,
    ) -> float:
        """
        Get remaining cooldown time.

        Args:
            action_type: Type of action
            cooldown_seconds: Cooldown duration in seconds
            applies_per_key: Optional key for per-entity cooldown

        Returns:
            Remaining cooldown in seconds (0 if no cooldown)
        """
        if cooldown_seconds == 0:
            return 0.0

        key = (action_type, applies_per_key or "")
        last_exec = self._last_execution.get(key)

        if last_exec is None:
            return 0.0

        elapsed = time.time() - last_exec
        remaining = max(0.0, cooldown_seconds - elapsed)
        return remaining

    def clear(self) -> None:
        """Clear all cooldown state (for testing)."""
        self._last_execution.clear()
