"""
Unit tests for ORION AI Council components.

Tests CouncilValidator, MemoryManager, ExternalValidator, and ConsensusAggregator
with all external dependencies mocked.
"""

import asyncio
from unittest.mock import MagicMock, patch, AsyncMock
import pytest

from core.council import (
    CouncilValidator,
    ExternalValidator,
    MemoryManager,
    ConsensusAggregator,
)


# ============================================================================
# MemoryManager Tests
# ============================================================================


@pytest.mark.unit
class TestMemoryManager:
    """Tests for MemoryManager resource monitoring."""

    def test_check_resources_blocks_below_4gb(self):
        """MemoryManager blocks when RAM is below 4GB threshold."""
        mm = MemoryManager(min_free_ram_gb=4.0)

        with patch("psutil.virtual_memory") as mock_vm:
            # Mock 2GB available RAM
            mock_vm.return_value = MagicMock(available=2 * 1024**3)
            can_proceed, message = mm.check_resources_before_load()

            assert can_proceed is False
            assert "Insufficient RAM" in message
            assert "2.0GB" in message

    def test_check_resources_allows_sufficient_ram(self):
        """MemoryManager allows when RAM is above threshold."""
        mm = MemoryManager(min_free_ram_gb=4.0)

        with patch("psutil.virtual_memory") as mock_vm:
            # Mock 6GB available RAM
            mock_vm.return_value = MagicMock(available=6 * 1024**3)

            with patch.object(mm, "get_cpu_temperature", return_value=(50.0, True)):
                can_proceed, message = mm.check_resources_before_load()

            assert can_proceed is True
            assert "Resources OK" in message

    def test_temperature_monitoring_optional(self):
        """Temperature monitoring is optional (vcgencmd may not be available)."""
        mm = MemoryManager()

        with patch("psutil.virtual_memory") as mock_vm:
            mock_vm.return_value = MagicMock(available=6 * 1024**3)

            # Simulate vcgencmd not available
            with patch("subprocess.run", side_effect=FileNotFoundError()):
                with patch("psutil.sensors_temperatures", return_value={}):
                    temp, available = mm.get_cpu_temperature()

            assert available is False
            # Should still work for resource check
            with patch.object(mm, "get_cpu_temperature", return_value=(0.0, False)):
                can_proceed, message = mm.check_resources_before_load()

            assert can_proceed is True
            assert "temp monitoring unavailable" in message

    def test_get_free_ram_gb(self):
        """get_free_ram_gb returns correct value."""
        mm = MemoryManager()

        with patch("psutil.virtual_memory") as mock_vm:
            mock_vm.return_value = MagicMock(available=8 * 1024**3)
            free_ram = mm.get_free_ram_gb()

            assert free_ram == 8.0


# ============================================================================
# CouncilValidator Tests
# ============================================================================


@pytest.mark.unit
class TestCouncilValidator:
    """Tests for CouncilValidator local SLM interface."""

    def test_validate_returns_confidence_and_critique(self):
        """validate() returns tuple of (confidence, critique)."""
        cv = CouncilValidator()

        # Mock MemoryManager to allow loading
        mock_mm = MagicMock()
        mock_mm.check_resources_before_load.return_value = (True, "OK")
        mock_mm.monitor_during_inference.return_value.__enter__ = MagicMock()
        mock_mm.monitor_during_inference.return_value.__exit__ = MagicMock()
        cv.set_memory_manager(mock_mm)

        with patch("ollama.generate") as mock_generate:
            mock_generate.return_value = {
                "response": "CONFIDENCE: 0.85\nCRITIQUE: Decision looks correct."
            }

            confidence, critique = cv.validate({"decision_type": "TEST"})

            assert confidence == 0.85
            assert "correct" in critique

    def test_validate_blocks_on_insufficient_ram(self):
        """validate() blocks when MemoryManager reports insufficient RAM."""
        cv = CouncilValidator()

        mock_mm = MagicMock()
        mock_mm.check_resources_before_load.return_value = (
            False,
            "Insufficient RAM: 2GB"
        )
        cv.set_memory_manager(mock_mm)

        confidence, critique = cv.validate({"decision_type": "TEST"})

        assert confidence == 0.0
        assert "BLOCKED" in critique
        assert "Insufficient RAM" in critique

    def test_validate_fails_closed_on_ollama_error(self):
        """validate() fails closed when Ollama raises an error."""
        cv = CouncilValidator()

        mock_mm = MagicMock()
        mock_mm.check_resources_before_load.return_value = (True, "OK")
        mock_mm.monitor_during_inference.return_value.__enter__ = MagicMock()
        mock_mm.monitor_during_inference.return_value.__exit__ = MagicMock()
        cv.set_memory_manager(mock_mm)

        with patch("ollama.generate") as mock_generate:
            from ollama import ResponseError
            mock_generate.side_effect = ResponseError("Model not found")

            confidence, critique = cv.validate({"decision_type": "TEST"})

            assert confidence == 0.0
            assert "ERROR" in critique

    def test_parse_response_extracts_confidence(self):
        """_parse_response correctly extracts confidence from response."""
        cv = CouncilValidator()

        response = "CONFIDENCE: 0.75\nCRITIQUE: Looks good"
        confidence, critique = cv._parse_response(response)

        assert confidence == 0.75
        assert critique == "Looks good"

    def test_parse_response_handles_percentage(self):
        """_parse_response handles percentage format."""
        cv = CouncilValidator()

        response = "CONFIDENCE: 80%\nCRITIQUE: Acceptable"
        confidence, critique = cv._parse_response(response)

        assert confidence == 0.8


# ============================================================================
# ExternalValidator Tests
# ============================================================================


@pytest.mark.unit
class TestExternalValidator:
    """Tests for ExternalValidator cloud API interface."""

    def test_missing_api_keys_handled_gracefully(self):
        """ExternalValidator handles missing API keys without raising."""
        with patch.dict("os.environ", {}, clear=True):
            ev = ExternalValidator()

            # Should not raise, but log warnings
            assert ev._anthropic_client is None
            assert ev._openai_client is None

    @pytest.mark.asyncio
    async def test_validate_with_claude_returns_confidence(self):
        """validate_with_claude returns (confidence, critique) tuple."""
        with patch.dict("os.environ", {"ANTHROPIC_API_KEY": "test-key"}):
            ev = ExternalValidator()

            # Mock the Anthropic client
            mock_response = MagicMock()
            mock_response.content = [
                MagicMock(text="CONFIDENCE: 0.9\nCRITIQUE: Approved")
            ]

            with patch.object(
                ev._anthropic_client.messages,
                "create",
                return_value=mock_response
            ):
                confidence, critique = await ev.validate_with_claude(
                    {"decision_type": "TEST"}
                )

            assert confidence == 0.9
            assert "Approved" in critique

    @pytest.mark.asyncio
    async def test_validate_with_openai_returns_confidence(self):
        """validate_with_openai returns (confidence, critique) tuple."""
        with patch.dict("os.environ", {"OPENAI_API_KEY": "test-key"}):
            ev = ExternalValidator()

            # Mock the OpenAI client
            mock_response = MagicMock()
            mock_response.choices = [
                MagicMock(
                    message=MagicMock(
                        content="CONFIDENCE: 0.85\nCRITIQUE: Valid decision"
                    )
                )
            ]

            with patch.object(
                ev._openai_client.chat.completions,
                "create",
                return_value=mock_response
            ):
                confidence, critique = await ev.validate_with_openai(
                    {"decision_type": "TEST"}
                )

            assert confidence == 0.85
            assert "Valid" in critique

    @pytest.mark.asyncio
    async def test_validate_parallel_calls_both_apis(self):
        """validate_parallel calls both APIs and returns combined results."""
        with patch.dict(
            "os.environ",
            {"ANTHROPIC_API_KEY": "test1", "OPENAI_API_KEY": "test2"}
        ):
            ev = ExternalValidator()

            # Mock both validators
            async def mock_claude(decision):
                return (0.9, "Claude approved")

            async def mock_openai(decision):
                return (0.8, "OpenAI approved")

            with patch.object(ev, "validate_with_claude", side_effect=mock_claude):
                with patch.object(ev, "validate_with_openai", side_effect=mock_openai):
                    results = await ev.validate_parallel({"decision_type": "TEST"})

            assert len(results) == 2
            assert results[0] == (0.9, "Claude approved")
            assert results[1] == (0.8, "OpenAI approved")

    @pytest.mark.asyncio
    async def test_validate_parallel_no_apis_configured(self):
        """validate_parallel returns error when no APIs configured."""
        with patch.dict("os.environ", {}, clear=True):
            ev = ExternalValidator()

            results = await ev.validate_parallel({"decision_type": "TEST"})

            assert len(results) == 1
            assert results[0][0] == 0.0
            assert "No external APIs configured" in results[0][1]


# ============================================================================
# ConsensusAggregator Tests
# ============================================================================


@pytest.mark.unit
class TestConsensusAggregator:
    """Tests for ConsensusAggregator voting and orchestration."""

    def test_confidence_weighted_voting_approves_high_confidence(self):
        """aggregate_votes approves when weighted average is high."""
        ca = ConsensusAggregator(confidence_threshold=0.7)

        validations = [
            (0.9, "This is a safe and correct decision"),
            (0.85, "I approve this decision"),
        ]

        result, confidence, critique = ca.aggregate_votes(validations)

        assert result == "APPROVED"
        assert confidence > 0.7

    def test_confidence_weighted_voting_blocks_low_confidence(self):
        """aggregate_votes blocks when weighted average is low."""
        ca = ConsensusAggregator(confidence_threshold=0.7)

        validations = [
            (0.9, "This is risky and concerning"),
            (0.8, "I have concerns about this unsafe action"),
        ]

        result, confidence, critique = ca.aggregate_votes(validations)

        assert result == "BLOCKED"

    def test_safety_veto_blocks_on_high_confidence_concern(self):
        """safety_veto blocks when validator has high-confidence safety concern."""
        ca = ConsensusAggregator(safety_veto_threshold=0.8)

        validations = [
            (0.85, "This is dangerous and unsafe"),
        ]

        veto_reason = ca.safety_veto(validations)

        assert veto_reason is not None
        assert "Safety veto" in veto_reason

    def test_safety_veto_allows_low_confidence_concern(self):
        """safety_veto doesn't trigger for low-confidence concerns."""
        ca = ConsensusAggregator(safety_veto_threshold=0.8)

        validations = [
            (0.5, "This might be unsafe"),  # Below threshold
        ]

        veto_reason = ca.safety_veto(validations)

        assert veto_reason is None

    def test_should_escalate_on_low_local_confidence(self):
        """should_escalate returns True for low local confidence."""
        ca = ConsensusAggregator(confidence_threshold=0.7)

        should = ca.should_escalate(0.5, "SAFE")

        assert should is True

    def test_should_escalate_on_risky_classification(self):
        """should_escalate returns True for RISKY classification."""
        ca = ConsensusAggregator(confidence_threshold=0.7)

        should = ca.should_escalate(0.9, "RISKY")

        assert should is True

    def test_should_not_escalate_on_high_confidence_safe(self):
        """should_escalate returns False for high-confidence SAFE."""
        ca = ConsensusAggregator(confidence_threshold=0.7)

        should = ca.should_escalate(0.9, "SAFE")

        assert should is False

    @pytest.mark.asyncio
    async def test_staged_validation_skips_external_on_high_local_confidence(self):
        """validate_decision skips external APIs when local confidence is high."""
        ca = ConsensusAggregator(confidence_threshold=0.7)

        # Mock validators
        mock_local = MagicMock()
        mock_local.validate.return_value = (0.9, "Local approved this safe decision")

        mock_external = MagicMock()
        mock_external.validate_parallel = AsyncMock()

        decision = {
            "safety_classification": "SAFE",
            "decision_type": "TEST",
            "reasoning": "Test reasoning"
        }

        result, confidence, critique = await ca.validate_decision(
            decision, mock_local, mock_external
        )

        # External validator should NOT be called
        mock_external.validate_parallel.assert_not_called()
        assert result == "APPROVED"

    @pytest.mark.asyncio
    async def test_staged_validation_escalates_on_low_confidence(self):
        """validate_decision escalates to external APIs on low confidence."""
        ca = ConsensusAggregator(confidence_threshold=0.7)

        # Mock validators
        mock_local = MagicMock()
        mock_local.validate.return_value = (0.5, "Local uncertain about this safe decision")

        mock_external = MagicMock()
        mock_external.validate_parallel = AsyncMock(
            return_value=[
                (0.9, "Claude approved"),
                (0.85, "OpenAI approved")
            ]
        )

        decision = {
            "safety_classification": "SAFE",
            "decision_type": "TEST",
            "reasoning": "Test reasoning"
        }

        result, confidence, critique = await ca.validate_decision(
            decision, mock_local, mock_external
        )

        # External validator SHOULD be called
        mock_external.validate_parallel.assert_called_once()

    @pytest.mark.asyncio
    async def test_staged_validation_safety_veto(self):
        """validate_decision triggers safety veto on high-confidence concern."""
        ca = ConsensusAggregator(
            confidence_threshold=0.7,
            safety_veto_threshold=0.8
        )

        # Mock validators
        mock_local = MagicMock()
        mock_local.validate.return_value = (0.85, "This is dangerous and unsafe")

        mock_external = MagicMock()

        decision = {
            "safety_classification": "SAFE",
            "decision_type": "TEST",
            "reasoning": "Test reasoning"
        }

        result, confidence, critique = await ca.validate_decision(
            decision, mock_local, mock_external
        )

        assert result == "BLOCKED"
        assert "Safety veto" in critique

    def test_aggregate_votes_handles_empty_list(self):
        """aggregate_votes handles empty validation list."""
        ca = ConsensusAggregator()

        result, confidence, critique = ca.aggregate_votes([])

        assert result == "BLOCKED"
        assert confidence == 0.0
        assert "No validations" in critique

    def test_aggregate_votes_handles_all_errors(self):
        """aggregate_votes handles when all validators return errors."""
        ca = ConsensusAggregator()

        validations = [
            (0.0, "ERROR: Model unavailable"),
            (0.0, "ERROR: API timeout"),
        ]

        result, confidence, critique = ca.aggregate_votes(validations)

        assert result == "BLOCKED"
        assert confidence == 0.0
        assert "failed" in critique.lower()
