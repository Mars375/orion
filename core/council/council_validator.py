"""
ORION Council Validator - Local SLM validation interface.

Interfaces with Ollama-served Gemma-2 2B model to validate Brain decisions.
Designed for Raspberry Pi 5 constraints: 8GB RAM, ARM64, thermal limits.

Hardware Requirements:
- Minimum 4GB free RAM before model loading
- Gemma-2 2B requires ~3GB RAM during inference
- Sequential model loading only (no parallel inference)
- 30-second timeout per validation to prevent hangs

Safety Invariants:
- Fail-closed: Any error or timeout returns (0.0, "ERROR: ...")
- Resource checks before model loading (delegated to MemoryManager)
- Never blocks system resources indefinitely
"""

import logging
from typing import Dict, Tuple, Any

import ollama
from ollama import ResponseError


logger = logging.getLogger(__name__)


# Default model to use for validation
DEFAULT_MODEL = "gemma2:2b"

# Timeout for validation requests (seconds)
VALIDATION_TIMEOUT = 30


class CouncilValidator:
    """
    Validates Brain decisions using local SLM via Ollama.

    The validator sends decisions to a local Gemma-2 2B model to:
    1. Check if SAFE/RISKY classification aligns with policies
    2. Verify reasoning makes sense for the incident
    3. Self-report confidence score (0.0-1.0)

    Invariants:
    - Fail-closed: Errors/timeouts return (0.0, "ERROR: ...")
    - Resource-aware: Integrates with MemoryManager (when provided)
    - Timeout-enforced: 30-second limit per validation
    - Sequential: One validation at a time
    """

    def __init__(
        self,
        model: str = DEFAULT_MODEL,
        timeout: int = VALIDATION_TIMEOUT,
    ) -> None:
        """
        Initialize the council validator.

        Args:
            model: Ollama model to use for validation (default: gemma2:2b)
            timeout: Timeout in seconds for validation requests (default: 30)
        """
        self.model = model
        self.timeout = timeout
        self._memory_manager = None

        logger.info(
            f"CouncilValidator initialized with model={model}, timeout={timeout}s"
        )

    def set_memory_manager(self, memory_manager: Any) -> None:
        """
        Set the memory manager for resource checks.

        Args:
            memory_manager: MemoryManager instance for resource monitoring
        """
        self._memory_manager = memory_manager
        logger.debug("MemoryManager attached to CouncilValidator")

    def _build_validation_prompt(self, decision: Dict[str, Any]) -> str:
        """
        Build prompt for validation request.

        Args:
            decision: Decision dict containing incident context and Brain reasoning

        Returns:
            Formatted prompt string for the model
        """
        incident_type = decision.get("incident_type", "unknown")
        severity = decision.get("severity", "unknown")
        classification = decision.get("safety_classification", "unknown")
        reasoning = decision.get("reasoning", "No reasoning provided")
        decision_type = decision.get("decision_type", "unknown")

        prompt = f"""You are a safety validator for an autonomous system called ORION.

TASK: Evaluate if this decision is correctly classified and reasoned.

INCIDENT CONTEXT:
- Type: {incident_type}
- Severity: {severity}

BRAIN DECISION:
- Classification: {classification}
- Decision Type: {decision_type}
- Reasoning: {reasoning}

EVALUATE:
1. Is the SAFE/RISKY classification appropriate for this incident?
2. Does the reasoning logically follow from the incident context?
3. Are there any safety concerns with this decision?

RESPOND IN THIS EXACT FORMAT:
CONFIDENCE: [0.0-1.0 score]
CRITIQUE: [Your brief evaluation in 1-2 sentences]

Be conservative - when uncertain, report lower confidence. Safety is paramount."""

        return prompt

    def _parse_response(self, response_text: str) -> Tuple[float, str]:
        """
        Parse model response to extract confidence and critique.

        Args:
            response_text: Raw response from model

        Returns:
            Tuple of (confidence_score, critique_text)
        """
        confidence = 0.0
        critique = response_text.strip()

        lines = response_text.strip().split("\n")

        for line in lines:
            line_upper = line.upper().strip()

            if line_upper.startswith("CONFIDENCE:"):
                try:
                    value_part = line.split(":", 1)[1].strip()
                    # Handle formats like "0.8", "0.8/1.0", "80%"
                    value_part = value_part.replace("%", "").split("/")[0].strip()
                    parsed = float(value_part)
                    # Normalize if given as percentage
                    if parsed > 1.0:
                        parsed = parsed / 100.0
                    confidence = max(0.0, min(1.0, parsed))
                except (ValueError, IndexError):
                    logger.warning(f"Failed to parse confidence from: {line}")
                    confidence = 0.0

            elif line_upper.startswith("CRITIQUE:"):
                try:
                    critique = line.split(":", 1)[1].strip()
                except IndexError:
                    critique = response_text.strip()

        return (confidence, critique)

    def validate(self, decision: Dict[str, Any]) -> Tuple[float, str]:
        """
        Validate a Brain decision using local SLM.

        Sends the decision to Ollama-served model for independent evaluation.
        Returns confidence score and critique text.

        Args:
            decision: Dict containing incident context and Brain reasoning.
                Expected keys:
                - incident_type: Type of incident
                - severity: Incident severity
                - safety_classification: SAFE or RISKY
                - decision_type: NO_ACTION, EXECUTE_SAFE_ACTION, etc.
                - reasoning: Brain's reasoning for the decision

        Returns:
            Tuple of (confidence_score, critique_text):
            - confidence_score: 0.0-1.0, where 1.0 means fully agrees with decision
            - critique_text: Brief evaluation or error message

        Raises:
            TimeoutError: If validation exceeds timeout limit

        Safety:
            - Fail-closed: Any error returns (0.0, "ERROR: ...")
            - Resource checks performed if MemoryManager is attached
        """
        # Check resources if memory manager is attached
        if self._memory_manager is not None:
            can_load, reason = self._memory_manager.check_resources_before_load()
            if not can_load:
                logger.warning(f"Resource check failed: {reason}")
                return (0.0, f"BLOCKED: {reason}")

        prompt = self._build_validation_prompt(decision)

        logger.debug(f"Sending validation request to {self.model}")

        try:
            # Use context manager for monitoring if available
            if self._memory_manager is not None:
                with self._memory_manager.monitor_during_inference():
                    response = ollama.generate(
                        model=self.model,
                        prompt=prompt,
                        options={"timeout": self.timeout},
                    )
            else:
                response = ollama.generate(
                    model=self.model,
                    prompt=prompt,
                    options={"timeout": self.timeout},
                )

            response_text = response.get("response", "")

            if not response_text:
                logger.warning("Empty response from model")
                return (0.0, "ERROR: Empty response from model")

            confidence, critique = self._parse_response(response_text)

            logger.info(
                f"Validation complete: confidence={confidence:.2f}, "
                f"critique_length={len(critique)}"
            )

            return (confidence, critique)

        except ResponseError as e:
            logger.error(f"Ollama response error: {e}")
            return (0.0, f"ERROR: Model unavailable - {e}")

        except TimeoutError:
            logger.error(f"Validation timed out after {self.timeout}s")
            raise

        except Exception as e:
            logger.error(f"Unexpected validation error: {e}", exc_info=True)
            return (0.0, f"ERROR: Validation failed - {type(e).__name__}")
