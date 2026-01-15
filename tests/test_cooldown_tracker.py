"""
Cooldown tracker tests.

Tests rate limiting and cooldown enforcement.
"""

import time
import pytest

from core.brain.cooldown_tracker import CooldownTracker


@pytest.mark.unit
class TestCooldownTracker:
    """Test cooldown tracking and enforcement."""

    def test_initializes_empty(self):
        """Cooldown tracker initializes with no tracked actions."""
        tracker = CooldownTracker()

        # First check should always pass
        assert tracker.check_cooldown("any_action", 60)

    def test_allows_first_execution(self):
        """First execution of action is always allowed."""
        tracker = CooldownTracker()

        assert tracker.check_cooldown("send_notification", 10)

    def test_blocks_rapid_repeated_execution(self):
        """Repeated execution within cooldown is blocked."""
        tracker = CooldownTracker()

        # First execution allowed
        assert tracker.check_cooldown("send_notification", 10)
        tracker.record_execution("send_notification")

        # Second execution immediately blocked
        assert not tracker.check_cooldown("send_notification", 10)

    def test_allows_execution_after_cooldown(self):
        """Execution allowed after cooldown expires."""
        tracker = CooldownTracker()
        cooldown = 1  # 1 second cooldown

        # First execution
        assert tracker.check_cooldown("send_notification", cooldown)
        tracker.record_execution("send_notification")

        # Blocked immediately
        assert not tracker.check_cooldown("send_notification", cooldown)

        # Wait for cooldown to expire
        time.sleep(cooldown + 0.1)

        # Now allowed
        assert tracker.check_cooldown("send_notification", cooldown)

    def test_tracks_per_action_type(self):
        """Cooldowns are tracked independently per action type."""
        tracker = CooldownTracker()

        # Execute action A
        tracker.record_execution("action_a")

        # Action A blocked
        assert not tracker.check_cooldown("action_a", 10)

        # Action B still allowed
        assert tracker.check_cooldown("action_b", 10)

    def test_tracks_per_entity(self):
        """Cooldowns can be tracked per entity."""
        tracker = CooldownTracker()

        # Execute for entity 1
        tracker.record_execution("restart_service", applies_per_key="service-1")

        # Blocked for entity 1
        assert not tracker.check_cooldown(
            "restart_service", 10, applies_per_key="service-1"
        )

        # Allowed for entity 2
        assert tracker.check_cooldown(
            "restart_service", 10, applies_per_key="service-2"
        )

    def test_returns_remaining_cooldown(self):
        """Returns accurate remaining cooldown time."""
        tracker = CooldownTracker()
        cooldown = 10

        tracker.record_execution("send_notification")

        remaining = tracker.get_remaining_cooldown("send_notification", cooldown)

        assert 0 < remaining <= cooldown

    def test_remaining_cooldown_zero_after_expiry(self):
        """Remaining cooldown is zero after expiry."""
        tracker = CooldownTracker()
        cooldown = 1

        tracker.record_execution("send_notification")

        # Wait for cooldown to expire
        time.sleep(cooldown + 0.1)

        remaining = tracker.get_remaining_cooldown("send_notification", cooldown)

        assert remaining == 0

    def test_remaining_cooldown_for_never_executed(self):
        """Remaining cooldown is zero for never-executed actions."""
        tracker = CooldownTracker()

        remaining = tracker.get_remaining_cooldown("never_executed", 10)

        assert remaining == 0

    def test_different_cooldowns_per_action(self):
        """Different actions can have different cooldowns."""
        tracker = CooldownTracker()

        # Short cooldown action
        tracker.record_execution("fast_action")
        assert not tracker.check_cooldown("fast_action", 1)

        # Long cooldown action
        tracker.record_execution("slow_action")
        assert not tracker.check_cooldown("slow_action", 300)

        # Wait for fast action cooldown
        time.sleep(1.1)

        # Fast action now allowed
        assert tracker.check_cooldown("fast_action", 1)

        # Slow action still blocked
        assert not tracker.check_cooldown("slow_action", 300)

    def test_multiple_executions_extend_cooldown(self):
        """Recording execution updates the cooldown timestamp."""
        tracker = CooldownTracker()
        cooldown = 2

        # First execution
        tracker.record_execution("send_notification")
        time.sleep(1)

        # Second execution (should update timestamp)
        tracker.record_execution("send_notification")

        # Should still be in cooldown
        remaining = tracker.get_remaining_cooldown("send_notification", cooldown)
        assert remaining > 1  # More than 1 second left


@pytest.mark.unit
class TestCooldownTrackerEdgeCases:
    """Test edge cases and corner cases."""

    def test_zero_cooldown(self):
        """Zero cooldown always allows execution."""
        tracker = CooldownTracker()

        tracker.record_execution("no_cooldown")

        # Even with zero cooldown, should allow
        assert tracker.check_cooldown("no_cooldown", 0)

    def test_negative_cooldown_treated_as_zero(self):
        """Negative cooldown treated as no cooldown."""
        tracker = CooldownTracker()

        tracker.record_execution("negative")

        assert tracker.check_cooldown("negative", -10)

    def test_very_long_cooldown(self):
        """Very long cooldowns work correctly."""
        tracker = CooldownTracker()
        cooldown = 86400  # 1 day

        tracker.record_execution("daily_action")

        assert not tracker.check_cooldown("daily_action", cooldown)

        remaining = tracker.get_remaining_cooldown("daily_action", cooldown)
        assert remaining > 86000  # Almost full day remaining

    def test_concurrent_different_actions(self):
        """Multiple different actions can execute concurrently."""
        tracker = CooldownTracker()

        # Execute multiple actions rapidly
        for i in range(10):
            action = f"action_{i}"
            assert tracker.check_cooldown(action, 10)
            tracker.record_execution(action)

    def test_none_applies_per_key(self):
        """None and missing applies_per_key are equivalent."""
        tracker = CooldownTracker()

        tracker.record_execution("test_action", applies_per_key=None)

        # Should be blocked with explicit None
        assert not tracker.check_cooldown("test_action", 10, applies_per_key=None)

        # Should also be blocked without key
        assert not tracker.check_cooldown("test_action", 10)
