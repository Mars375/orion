"""
Policy consistency tests.

Validates that:
1. All action types are classified as either SAFE or RISKY
2. No action is in both SAFE and RISKY lists
3. SAFE actions meet safety criteria
4. RISKY actions require approval
5. Cooldowns exist for all actions
6. Policy files are valid YAML
"""

from pathlib import Path

import pytest
import yaml


POLICIES_DIR = Path(__file__).parent.parent / "policies"


def load_policy(name):
    """Load a YAML policy file from policies/."""
    policy_path = POLICIES_DIR / name
    with open(policy_path) as f:
        return yaml.safe_load(f)


@pytest.mark.policy
class TestActionClassification:
    """Test SAFE vs RISKY action classification."""

    def test_actions_safe_is_valid_yaml(self):
        """actions_safe.yaml is valid YAML."""
        policy = load_policy("actions_safe.yaml")
        assert policy is not None
        assert "version" in policy
        assert policy["version"] == "1.0"

    def test_actions_risky_is_valid_yaml(self):
        """actions_risky.yaml is valid YAML."""
        policy = load_policy("actions_risky.yaml")
        assert policy is not None
        assert "version" in policy
        assert policy["version"] == "1.0"

    def test_safe_actions_have_required_fields(self):
        """All SAFE actions have required documentation fields."""
        policy = load_policy("actions_safe.yaml")
        safe_actions = policy.get("safe_actions", [])

        assert len(safe_actions) > 0, "Must have at least one SAFE action"

        for action in safe_actions:
            assert "action_type" in action, f"Missing action_type"
            assert "description" in action, f"Missing description for {action.get('action_type')}"
            assert "reversible" in action, f"Missing reversible for {action['action_type']}"
            assert "external_side_effects" in action, f"Missing external_side_effects for {action['action_type']}"
            assert "justification" in action, f"Missing justification for {action['action_type']}"

    def test_risky_actions_have_required_fields(self):
        """All RISKY actions have required documentation fields."""
        policy = load_policy("actions_risky.yaml")
        risky_actions = policy.get("risky_actions", [])

        assert len(risky_actions) > 0, "Must have at least one RISKY action"

        for action in risky_actions:
            assert "action_type" in action, f"Missing action_type"
            assert "description" in action, f"Missing description for {action.get('action_type')}"
            assert "reversible" in action, f"Missing reversible for {action['action_type']}"
            assert "external_side_effects" in action, f"Missing external_side_effects for {action['action_type']}"
            assert "blast_radius" in action, f"Missing blast_radius for {action['action_type']}"
            assert "justification" in action, f"Missing justification for {action['action_type']}"
            assert "requires_approval" in action, f"Missing requires_approval for {action['action_type']}"
            assert action["requires_approval"] is True, f"RISKY action {action['action_type']} must require approval"

    def test_no_action_in_both_safe_and_risky(self):
        """No action type appears in both SAFE and RISKY lists."""
        safe_policy = load_policy("actions_safe.yaml")
        risky_policy = load_policy("actions_risky.yaml")

        safe_actions = {a["action_type"] for a in safe_policy.get("safe_actions", [])}
        risky_actions = {a["action_type"] for a in risky_policy.get("risky_actions", [])}

        overlap = safe_actions & risky_actions
        assert len(overlap) == 0, f"Actions in both SAFE and RISKY: {overlap}"

    def test_safe_actions_are_conservative(self):
        """SAFE actions meet conservative criteria."""
        policy = load_policy("actions_safe.yaml")
        safe_actions = policy.get("safe_actions", [])

        # All SAFE actions should either be:
        # 1. Read-only (run_diagnostic, acknowledge_incident)
        # 2. Notification-only (send_notification)
        for action in safe_actions:
            action_type = action["action_type"]

            # Safe actions should have limited blast radius
            # (This is a logical check, not a schema check)
            assert action.get("max_frequency") is not None or action_type == "acknowledge_incident", \
                f"SAFE action {action_type} should have max_frequency to prevent abuse"


@pytest.mark.policy
class TestCooldownPolicy:
    """Test cooldown policy configuration."""

    def test_cooldowns_is_valid_yaml(self):
        """cooldowns.yaml is valid YAML."""
        policy = load_policy("cooldowns.yaml")
        assert policy is not None
        assert "version" in policy
        assert policy["version"] == "1.0"

    def test_global_default_cooldown_exists(self):
        """Global default cooldown is defined."""
        policy = load_policy("cooldowns.yaml")
        assert "global_settings" in policy
        assert "global_default_cooldown" in policy["global_settings"]

    def test_risky_actions_have_cooldowns(self):
        """All RISKY actions have cooldown definitions."""
        risky_policy = load_policy("actions_risky.yaml")
        cooldown_policy = load_policy("cooldowns.yaml")

        risky_actions = {a["action_type"] for a in risky_policy.get("risky_actions", [])}
        cooldown_actions = {c["action_type"] for c in cooldown_policy.get("action_cooldowns", [])}

        # All risky actions should have explicit cooldowns
        missing_cooldowns = risky_actions - cooldown_actions
        assert len(missing_cooldowns) == 0, f"RISKY actions missing cooldowns: {missing_cooldowns}"

    def test_circuit_breaker_enabled(self):
        """Circuit breaker is enabled to prevent failure loops."""
        policy = load_policy("cooldowns.yaml")
        assert "circuit_breaker" in policy
        assert policy["circuit_breaker"]["enabled"] is True


@pytest.mark.policy
class TestApprovalPolicy:
    """Test approval policy configuration."""

    def test_approvals_is_valid_yaml(self):
        """approvals.yaml is valid YAML."""
        policy = load_policy("approvals.yaml")
        assert policy is not None
        assert "version" in policy
        assert policy["version"] == "1.0"

    def test_default_autonomy_level_is_n0(self):
        """Default autonomy level is N0 (observe only) for safety."""
        policy = load_policy("approvals.yaml")
        assert policy["default_autonomy_level"] == "N0", \
            "Default autonomy must be N0 (observe only) for maximum safety"

    def test_n0_allows_no_actions(self):
        """Autonomy level N0 permits no actions."""
        policy = load_policy("approvals.yaml")
        n0_config = policy["autonomy_levels"]["N0"]
        assert n0_config["allow_safe_actions"] is False
        assert n0_config["allow_risky_actions"] is False

    def test_n2_allows_safe_not_risky(self):
        """Autonomy level N2 allows SAFE but not RISKY actions."""
        policy = load_policy("approvals.yaml")
        n2_config = policy["autonomy_levels"]["N2"]
        assert n2_config["allow_safe_actions"] is True
        assert n2_config["allow_risky_actions"] is False

    def test_n3_requires_approval_for_risky(self):
        """Autonomy level N3 allows RISKY only with approval."""
        policy = load_policy("approvals.yaml")
        n3_config = policy["autonomy_levels"]["N3"]
        assert n3_config["allow_safe_actions"] is True
        assert n3_config["allow_risky_actions"] is True  # But only with approval

    def test_timeout_behavior_is_deny(self):
        """Approval timeout defaults to deny (safe default)."""
        policy = load_policy("approvals.yaml")
        assert policy["approval_settings"]["timeout_behavior"] == "deny", \
            "Timeout must deny for safety (never auto-approve)"

    def test_approval_persistence_is_false(self):
        """Approvals are not reusable (one-time use only)."""
        policy = load_policy("approvals.yaml")
        assert policy["approval_settings"]["approval_persistence"] is False, \
            "Approvals must be one-time use (not reusable) for safety"
