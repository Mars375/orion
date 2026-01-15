"""
Policy loader tests.

Tests SAFE/RISKY classification and cooldown enforcement.
"""

from pathlib import Path
import pytest

from core.brain.policy_loader import PolicyLoader


POLICIES_DIR = Path(__file__).parent.parent / "policies"


@pytest.mark.unit
class TestPolicyLoader:
    """Test policy loading and classification."""

    def test_loads_policies(self):
        """Policy loader loads SAFE and RISKY actions."""
        loader = PolicyLoader(POLICIES_DIR)

        assert len(loader.get_safe_actions()) > 0
        assert len(loader.get_risky_actions()) > 0

    def test_safe_action_classification(self):
        """SAFE actions are correctly classified."""
        loader = PolicyLoader(POLICIES_DIR)

        # From policies/actions_safe.yaml
        assert loader.is_safe("send_notification")
        assert loader.is_safe("acknowledge_incident")
        assert loader.is_safe("run_diagnostic")

        assert loader.classify_action("send_notification") == "SAFE"
        assert loader.classify_action("acknowledge_incident") == "SAFE"

    def test_risky_action_classification(self):
        """RISKY actions are correctly classified."""
        loader = PolicyLoader(POLICIES_DIR)

        # From policies/actions_risky.yaml
        assert loader.is_risky("restart_service")
        assert loader.is_risky("scale_service")
        assert loader.is_risky("stop_edge_device")

        assert loader.classify_action("restart_service") == "RISKY"
        assert loader.classify_action("stop_edge_device") == "RISKY"

    def test_unknown_action_classification(self):
        """Unknown actions are classified as UNKNOWN (fail closed)."""
        loader = PolicyLoader(POLICIES_DIR)

        assert not loader.is_safe("unknown_action")
        assert not loader.is_risky("unknown_action")
        assert loader.classify_action("unknown_action") == "UNKNOWN"

    def test_action_not_in_both_lists(self):
        """No action appears in both SAFE and RISKY lists."""
        loader = PolicyLoader(POLICIES_DIR)

        safe = loader.get_safe_actions()
        risky = loader.get_risky_actions()

        overlap = safe & risky
        assert len(overlap) == 0, f"Actions in both SAFE and RISKY: {overlap}"

    def test_cooldown_loading(self):
        """Cooldowns are loaded from policy."""
        loader = PolicyLoader(POLICIES_DIR)

        # From policies/cooldowns.yaml
        cooldown = loader.get_cooldown("restart_service")
        assert cooldown == 300  # 5 minutes

        cooldown = loader.get_cooldown("send_notification")
        assert cooldown == 10  # 10 seconds

    def test_missing_cooldown_returns_none(self):
        """Missing cooldown returns None."""
        loader = PolicyLoader(POLICIES_DIR)

        cooldown = loader.get_cooldown("nonexistent_action")
        assert cooldown is None

    def test_parse_duration(self):
        """Duration strings are correctly parsed."""
        loader = PolicyLoader(POLICIES_DIR)

        assert loader._parse_duration("60s") == 60
        assert loader._parse_duration("5m") == 300
        assert loader._parse_duration("1h") == 3600
        assert loader._parse_duration("30") == 30  # No unit = seconds
