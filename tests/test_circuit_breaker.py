"""
Circuit breaker tests.

Tests failure protection and circuit breaker logic.
"""

import time
import pytest

from core.brain.circuit_breaker import CircuitBreaker


@pytest.mark.unit
class TestCircuitBreaker:
    """Test circuit breaker functionality."""

    def test_initializes_closed(self):
        """Circuit breaker initializes in CLOSED state."""
        breaker = CircuitBreaker()

        assert not breaker.is_open("any_action")

    def test_stays_closed_with_successes(self):
        """Circuit stays CLOSED with successful executions."""
        breaker = CircuitBreaker()

        for _ in range(10):
            breaker.record_success("stable_action")

        assert not breaker.is_open("stable_action")

    def test_opens_after_threshold_failures(self):
        """Circuit OPENS after failure threshold reached."""
        breaker = CircuitBreaker(
            failure_threshold=3,
            failure_window=60,
            circuit_open_duration=60,
        )

        # Record failures up to threshold
        for _ in range(3):
            breaker.record_failure("failing_action")

        # Circuit should now be OPEN
        assert breaker.is_open("failing_action")

    def test_stays_closed_below_threshold(self):
        """Circuit stays CLOSED below failure threshold."""
        breaker = CircuitBreaker(failure_threshold=3)

        # Record failures below threshold
        breaker.record_failure("action")
        breaker.record_failure("action")

        # Still closed
        assert not breaker.is_open("action")

    def test_tracks_per_action_type(self):
        """Circuit breaker tracks state independently per action."""
        breaker = CircuitBreaker(failure_threshold=2)

        # Fail action A
        breaker.record_failure("action_a")
        breaker.record_failure("action_a")

        # Action A circuit open
        assert breaker.is_open("action_a")

        # Action B circuit still closed
        assert not breaker.is_open("action_b")

    def test_success_resets_failure_count(self):
        """Success resets failure count."""
        breaker = CircuitBreaker(failure_threshold=3)

        # Record some failures
        breaker.record_failure("action")
        breaker.record_failure("action")

        # Record success
        breaker.record_success("action")

        # Record more failures (should start from zero)
        breaker.record_failure("action")
        breaker.record_failure("action")

        # Still closed (only 2 failures since last success)
        assert not breaker.is_open("action")

    def test_closes_after_timeout(self):
        """Circuit CLOSES after timeout expires."""
        breaker = CircuitBreaker(
            failure_threshold=2,
            circuit_open_duration=1,
        )

        # Open the circuit
        breaker.record_failure("action")
        breaker.record_failure("action")
        assert breaker.is_open("action")

        # Wait for timeout
        time.sleep(1.1)

        # Circuit should now be CLOSED
        assert not breaker.is_open("action")

    def test_old_failures_expire(self):
        """Failures outside window don't count."""
        breaker = CircuitBreaker(
            failure_threshold=2,
            failure_window=1,
        )

        # Record old failure
        breaker.record_failure("action")

        # Wait for window to pass
        time.sleep(1.1)

        # Record new failure (old one expired)
        breaker.record_failure("action")

        # Still closed (only 1 failure in window)
        assert not breaker.is_open("action")

    def test_get_state_closed(self):
        """get_state returns correct info for CLOSED circuit."""
        breaker = CircuitBreaker(failure_threshold=3)

        state = breaker.get_state("action")

        assert state["circuit_open"] is False
        assert state["failure_count"] == 0
        assert "circuit_opened_at" not in state

    def test_get_state_open(self):
        """get_state returns correct info for OPEN circuit."""
        breaker = CircuitBreaker(failure_threshold=2)

        breaker.record_failure("action")
        breaker.record_failure("action")

        state = breaker.get_state("action")

        assert state["circuit_open"] is True
        assert state["failure_count"] == 2
        assert "circuit_opened_at" in state

    def test_get_state_with_failures(self):
        """get_state shows failure count below threshold."""
        breaker = CircuitBreaker(failure_threshold=5)

        breaker.record_failure("action")
        breaker.record_failure("action")

        state = breaker.get_state("action")

        assert state["circuit_open"] is False
        assert state["failure_count"] == 2

    def test_multiple_cycles(self):
        """Circuit can open and close multiple times."""
        breaker = CircuitBreaker(
            failure_threshold=2,
            circuit_open_duration=1,
        )

        # First cycle: open
        breaker.record_failure("action")
        breaker.record_failure("action")
        assert breaker.is_open("action")

        # Wait for close
        time.sleep(1.1)
        assert not breaker.is_open("action")

        # Second cycle: open again
        breaker.record_failure("action")
        breaker.record_failure("action")
        assert breaker.is_open("action")


@pytest.mark.unit
class TestCircuitBreakerEdgeCases:
    """Test edge cases and corner cases."""

    def test_threshold_of_one(self):
        """Circuit with threshold=1 opens on first failure."""
        breaker = CircuitBreaker(failure_threshold=1)

        breaker.record_failure("action")

        assert breaker.is_open("action")

    def test_very_high_threshold(self):
        """Circuit with very high threshold works correctly."""
        breaker = CircuitBreaker(failure_threshold=1000)

        # Record many failures
        for _ in range(999):
            breaker.record_failure("action")

        # Still closed
        assert not breaker.is_open("action")

        # One more opens it
        breaker.record_failure("action")
        assert breaker.is_open("action")

    def test_zero_open_duration(self):
        """Zero open duration immediately closes circuit."""
        breaker = CircuitBreaker(
            failure_threshold=2,
            circuit_open_duration=0,
        )

        breaker.record_failure("action")
        breaker.record_failure("action")

        # With 0 duration, circuit opens but closes on first check
        # The is_open() call checks elapsed time and sees duration expired
        assert not breaker.is_open("action")

        # Can execute again immediately
        assert not breaker.is_open("action")

    def test_very_long_window(self):
        """Very long failure window works correctly."""
        breaker = CircuitBreaker(
            failure_threshold=2,
            failure_window=86400,  # 1 day
        )

        breaker.record_failure("action")
        time.sleep(1)
        breaker.record_failure("action")

        # Both failures still in window
        assert breaker.is_open("action")

    def test_concurrent_different_actions(self):
        """Multiple actions tracked independently."""
        breaker = CircuitBreaker(failure_threshold=2)

        # Open circuits for multiple actions
        for i in range(5):
            action = f"action_{i}"
            breaker.record_failure(action)
            breaker.record_failure(action)

        # All should be open
        for i in range(5):
            action = f"action_{i}"
            assert breaker.is_open(action)

        # Different action should be closed
        assert not breaker.is_open("different_action")

    def test_rapid_success_failure_cycles(self):
        """Rapid success/failure cycles handled correctly."""
        breaker = CircuitBreaker(failure_threshold=3)

        # Alternating success and failure
        for _ in range(10):
            breaker.record_failure("action")
            breaker.record_success("action")

        # Success after each failure should keep it closed
        assert not breaker.is_open("action")

    def test_state_never_tracked_action(self):
        """get_state for never-tracked action returns CLOSED."""
        breaker = CircuitBreaker()

        state = breaker.get_state("never_used")

        assert state["circuit_open"] is False
        assert state["failure_count"] == 0

    def test_success_on_open_circuit_resets(self):
        """Recording success clears failures but doesn't close open circuit."""
        breaker = CircuitBreaker(
            failure_threshold=2,
            circuit_open_duration=60,
        )

        # Open the circuit
        breaker.record_failure("action")
        breaker.record_failure("action")
        assert breaker.is_open("action")

        # Record success (clears failure count)
        breaker.record_success("action")

        # Circuit is still open (timeout not expired)
        assert breaker.is_open("action")

        state = breaker.get_state("action")
        assert state["failure_count"] == 0  # Failures cleared
        assert "circuit_opened_at" in state  # But still open


@pytest.mark.unit
class TestCircuitBreakerIntegration:
    """Test circuit breaker in realistic scenarios."""

    def test_intermittent_failures(self):
        """Intermittent failures eventually open circuit."""
        breaker = CircuitBreaker(
            failure_threshold=5,
            failure_window=10,
        )

        # Simulate intermittent failures
        for _ in range(5):
            breaker.record_failure("flaky_action")
            time.sleep(0.1)

        # Circuit should open
        assert breaker.is_open("flaky_action")

    def test_recovery_scenario(self):
        """Circuit can recover after failures stop."""
        breaker = CircuitBreaker(
            failure_threshold=3,
            circuit_open_duration=1,
        )

        # Failure spike opens circuit
        for _ in range(3):
            breaker.record_failure("action")
        assert breaker.is_open("action")

        # Wait for recovery
        time.sleep(1.1)
        assert not breaker.is_open("action")

        # Successful executions keep it closed
        for _ in range(5):
            breaker.record_success("action")
        assert not breaker.is_open("action")

    def test_prevents_execution_when_open(self):
        """Simulates preventing execution when circuit open."""
        breaker = CircuitBreaker(failure_threshold=2)

        # Open the circuit
        breaker.record_failure("action")
        breaker.record_failure("action")

        # Simulate execution check
        if breaker.is_open("action"):
            execution_allowed = False
        else:
            execution_allowed = True

        assert not execution_allowed
