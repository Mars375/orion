"""
Policy loader for SAFE/RISKY action classification.

Reads policy files and enforces action safety classifications.
"""

import logging
from pathlib import Path
from typing import Dict, Any, Set, Optional

import yaml


logger = logging.getLogger(__name__)


class PolicyLoader:
    """
    Loads and enforces action policies.

    Policies are the single source of truth for SAFE vs RISKY classification.
    """

    def __init__(self, policy_dir: Path):
        """
        Initialize policy loader.

        Args:
            policy_dir: Directory containing policy YAML files
        """
        self.policy_dir = policy_dir
        self._safe_actions: Set[str] = set()
        self._risky_actions: Set[str] = set()
        self._cooldowns: Dict[str, int] = {}
        self._load_policies()

    def _load_policies(self) -> None:
        """Load all policy files."""
        try:
            # Load SAFE actions
            safe_policy_path = self.policy_dir / "actions_safe.yaml"
            with open(safe_policy_path) as f:
                safe_policy = yaml.safe_load(f)
                for action in safe_policy.get("safe_actions", []):
                    self._safe_actions.add(action["action_type"])

            # Load RISKY actions
            risky_policy_path = self.policy_dir / "actions_risky.yaml"
            with open(risky_policy_path) as f:
                risky_policy = yaml.safe_load(f)
                for action in risky_policy.get("risky_actions", []):
                    self._risky_actions.add(action["action_type"])

            # Load cooldowns
            cooldown_policy_path = self.policy_dir / "cooldowns.yaml"
            with open(cooldown_policy_path) as f:
                cooldown_policy = yaml.safe_load(f)
                for action_cooldown in cooldown_policy.get("action_cooldowns", []):
                    action_type = action_cooldown["action_type"]
                    cooldown_str = action_cooldown["cooldown"]
                    # Parse cooldown string (e.g., "60s" -> 60)
                    cooldown_seconds = self._parse_duration(cooldown_str)
                    self._cooldowns[action_type] = cooldown_seconds

            logger.info(
                f"Policies loaded: {len(self._safe_actions)} SAFE, "
                f"{len(self._risky_actions)} RISKY, "
                f"{len(self._cooldowns)} cooldowns"
            )

        except Exception as e:
            logger.error(f"Failed to load policies: {e}", exc_info=True)
            # Fail closed: if policies can't be loaded, no actions are safe
            self._safe_actions = set()
            self._risky_actions = set()
            self._cooldowns = {}

    def _parse_duration(self, duration_str: str) -> int:
        """
        Parse duration string to seconds.

        Args:
            duration_str: Duration string (e.g., "60s", "5m")

        Returns:
            Duration in seconds
        """
        duration_str = duration_str.strip()
        if duration_str.endswith("s"):
            return int(duration_str[:-1])
        elif duration_str.endswith("m"):
            return int(duration_str[:-1]) * 60
        elif duration_str.endswith("h"):
            return int(duration_str[:-1]) * 3600
        else:
            # Default to seconds if no unit
            return int(duration_str)

    def is_safe(self, action_type: str) -> bool:
        """
        Check if action is classified as SAFE.

        Args:
            action_type: Action type to check

        Returns:
            True if action is SAFE, False otherwise
        """
        return action_type in self._safe_actions

    def is_risky(self, action_type: str) -> bool:
        """
        Check if action is classified as RISKY.

        Args:
            action_type: Action type to check

        Returns:
            True if action is RISKY, False otherwise
        """
        return action_type in self._risky_actions

    def classify_action(self, action_type: str) -> str:
        """
        Classify action as SAFE, RISKY, or UNKNOWN.

        Args:
            action_type: Action type to classify

        Returns:
            Classification: "SAFE", "RISKY", or "UNKNOWN"
        """
        if self.is_safe(action_type):
            return "SAFE"
        elif self.is_risky(action_type):
            return "RISKY"
        else:
            # Unknown actions are treated as RISKY (fail closed)
            return "UNKNOWN"

    def get_cooldown(self, action_type: str) -> Optional[int]:
        """
        Get cooldown duration for action type.

        Args:
            action_type: Action type

        Returns:
            Cooldown in seconds, or None if no cooldown defined
        """
        return self._cooldowns.get(action_type)

    def get_safe_actions(self) -> Set[str]:
        """Get set of all SAFE action types."""
        return self._safe_actions.copy()

    def get_risky_actions(self) -> Set[str]:
        """Get set of all RISKY action types."""
        return self._risky_actions.copy()
