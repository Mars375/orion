"""
ORION External Validator - Cloud API validation interface.

Interfaces with Claude 3.5 Sonnet and OpenAI GPT-4 Turbo APIs for high-confidence
validation when local SLM is uncertain or decision is RISKY.

API Integration:
- Claude 3.5 Sonnet: Superior reasoning, 200K context, structured outputs
- OpenAI GPT-4 Turbo: Diverse perspective, tie-breaker
- Parallel execution via asyncio.gather for 2-5 second total latency

Safety Invariants:
- Fail-closed: Any error returns (0.0, "ERROR: ..."), never raises
- Missing API keys: Log warning, skip that API, continue
- Timeout: Return (0.0, "ERROR: timeout") after 10 seconds
- Network errors: Retry up to 2 times with exponential backoff
- Auth errors: Fail immediately (misconfiguration, not transient)
- Rate limit errors: Fail immediately (don't hammer APIs)

Environment Variables:
- ANTHROPIC_API_KEY: API key for Claude
- OPENAI_API_KEY: API key for OpenAI GPT-4
"""

import asyncio
import logging
import os
from typing import Any, Callable, Dict, List, Tuple

import anthropic
import openai


logger = logging.getLogger(__name__)


# API timeout in seconds
API_TIMEOUT = 10

# Retry configuration
MAX_RETRIES = 2
INITIAL_RETRY_DELAY = 1.0


class ExternalValidator:
    """
    Validates Brain decisions using external cloud APIs.

    Calls Claude 3.5 Sonnet and/or OpenAI GPT-4 Turbo for high-confidence
    validation. Both APIs are called in parallel when available.

    Invariants:
    - Fail-closed: Errors/timeouts return (0.0, "ERROR: ..."), never raise
    - Retry transient errors: Max 2 retries with exponential backoff
    - No retry for auth/rate limit: These are not transient
    - Missing keys: Skip that API, log warning
    - Parallel execution: asyncio.gather for concurrent API calls
    """

    def __init__(self) -> None:
        """
        Initialize external validator.

        Loads API keys from environment variables. Missing keys are handled
        gracefully - validation methods will return fail-closed errors.
        """
        self._anthropic_key = os.environ.get("ANTHROPIC_API_KEY")
        self._openai_key = os.environ.get("OPENAI_API_KEY")

        if not self._anthropic_key:
            logger.warning("ANTHROPIC_API_KEY not set - Claude validation unavailable")
        if not self._openai_key:
            logger.warning("OPENAI_API_KEY not set - OpenAI validation unavailable")

        # Initialize clients if keys available
        self._anthropic_client = (
            anthropic.Anthropic(api_key=self._anthropic_key)
            if self._anthropic_key
            else None
        )
        self._openai_client = (
            openai.OpenAI(api_key=self._openai_key)
            if self._openai_key
            else None
        )

        logger.info(
            f"ExternalValidator initialized: "
            f"claude={'available' if self._anthropic_key else 'unavailable'}, "
            f"openai={'available' if self._openai_key else 'unavailable'}"
        )

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

    async def _retry_with_backoff(
        self,
        async_func: Callable[[], Any],
        api_name: str,
        max_retries: int = MAX_RETRIES,
        initial_delay: float = INITIAL_RETRY_DELAY,
    ) -> Tuple[float, str]:
        """
        Execute async function with retry logic and exponential backoff.

        Only retries on connection errors. Authentication and rate limit
        errors fail immediately (not transient).

        Args:
            async_func: Async function to execute (should return Tuple[float, str])
            api_name: Name of API for logging (e.g., "Claude", "OpenAI")
            max_retries: Maximum number of retries (default: 2)
            initial_delay: Initial delay in seconds before first retry (default: 1.0)

        Returns:
            Tuple of (confidence_score, critique_text)

        Retry Strategy:
        - Connection errors: Retry with exponential backoff
        - Auth errors: Fail immediately (misconfiguration)
        - Rate limit errors: Fail immediately (don't hammer API)
        - Timeout errors: Retry with backoff
        - Other errors: Fail immediately
        """
        delay = initial_delay
        last_error = None

        for attempt in range(max_retries + 1):
            try:
                return await async_func()

            except anthropic.AuthenticationError as e:
                logger.error(f"{api_name} authentication failed: {e}")
                return (0.0, f"ERROR: {api_name} authentication failed")

            except openai.AuthenticationError as e:
                logger.error(f"{api_name} authentication failed: {e}")
                return (0.0, f"ERROR: {api_name} authentication failed")

            except anthropic.RateLimitError as e:
                logger.warning(f"{api_name} rate limited: {e}")
                return (0.0, f"ERROR: {api_name} rate limited")

            except openai.RateLimitError as e:
                logger.warning(f"{api_name} rate limited: {e}")
                return (0.0, f"ERROR: {api_name} rate limited")

            except (anthropic.APIConnectionError, openai.APIConnectionError) as e:
                last_error = e
                if attempt < max_retries:
                    logger.info(
                        f"{api_name} connection error, retrying in {delay}s "
                        f"(attempt {attempt + 1}/{max_retries + 1}): {e}"
                    )
                    await asyncio.sleep(delay)
                    delay *= 2
                else:
                    logger.warning(
                        f"{api_name} connection failed after {max_retries + 1} attempts: {e}"
                    )

            except asyncio.TimeoutError:
                last_error = "timeout"
                if attempt < max_retries:
                    logger.info(
                        f"{api_name} timeout, retrying in {delay}s "
                        f"(attempt {attempt + 1}/{max_retries + 1})"
                    )
                    await asyncio.sleep(delay)
                    delay *= 2
                else:
                    logger.warning(
                        f"{api_name} timeout after {max_retries + 1} attempts"
                    )

            except Exception as e:
                logger.error(f"{api_name} unexpected error: {e}", exc_info=True)
                return (0.0, f"ERROR: {api_name} failed - {type(e).__name__}")

        # All retries exhausted
        if last_error == "timeout":
            return (0.0, f"ERROR: {api_name} timeout")
        return (0.0, f"ERROR: {api_name} connection failed")

    async def validate_with_claude(
        self, decision: Dict[str, Any]
    ) -> Tuple[float, str]:
        """
        Validate decision using Claude 3.5 Sonnet API.

        Args:
            decision: Dict containing incident context and Brain reasoning

        Returns:
            Tuple of (confidence_score, critique_text):
            - confidence_score: 0.0-1.0
            - critique_text: Brief evaluation or error message

        Safety:
            - Fail-closed: Returns (0.0, "ERROR: ...") on any failure
            - Retries connection errors with exponential backoff
            - Does not retry auth/rate limit errors
        """
        if not self._anthropic_client:
            return (0.0, "ERROR: Claude API not configured")

        prompt = self._build_validation_prompt(decision)

        async def _call_claude() -> Tuple[float, str]:
            # Run synchronous API call in executor to make it async
            loop = asyncio.get_event_loop()

            def _sync_call() -> anthropic.types.Message:
                return self._anthropic_client.messages.create(
                    model="claude-3-5-sonnet-20241022",
                    max_tokens=1024,
                    temperature=0.0,
                    messages=[{"role": "user", "content": prompt}],
                    timeout=API_TIMEOUT,
                )

            response = await asyncio.wait_for(
                loop.run_in_executor(None, _sync_call),
                timeout=API_TIMEOUT + 2,  # Buffer for executor overhead
            )

            response_text = response.content[0].text if response.content else ""

            if not response_text:
                return (0.0, "ERROR: Empty response from Claude")

            return self._parse_response(response_text)

        result = await self._retry_with_backoff(_call_claude, "Claude")
        logger.info(f"Claude validation: confidence={result[0]:.2f}")
        return result

    async def validate_with_openai(
        self, decision: Dict[str, Any]
    ) -> Tuple[float, str]:
        """
        Validate decision using OpenAI GPT-4 Turbo API.

        Args:
            decision: Dict containing incident context and Brain reasoning

        Returns:
            Tuple of (confidence_score, critique_text):
            - confidence_score: 0.0-1.0
            - critique_text: Brief evaluation or error message

        Safety:
            - Fail-closed: Returns (0.0, "ERROR: ...") on any failure
            - Retries connection errors with exponential backoff
            - Does not retry auth/rate limit errors
        """
        if not self._openai_client:
            return (0.0, "ERROR: OpenAI API not configured")

        prompt = self._build_validation_prompt(decision)

        async def _call_openai() -> Tuple[float, str]:
            # Run synchronous API call in executor to make it async
            loop = asyncio.get_event_loop()

            def _sync_call() -> openai.types.chat.ChatCompletion:
                return self._openai_client.chat.completions.create(
                    model="gpt-4-turbo",
                    max_tokens=1024,
                    temperature=0.0,
                    messages=[{"role": "user", "content": prompt}],
                    timeout=API_TIMEOUT,
                )

            response = await asyncio.wait_for(
                loop.run_in_executor(None, _sync_call),
                timeout=API_TIMEOUT + 2,  # Buffer for executor overhead
            )

            response_text = (
                response.choices[0].message.content
                if response.choices and response.choices[0].message.content
                else ""
            )

            if not response_text:
                return (0.0, "ERROR: Empty response from OpenAI")

            return self._parse_response(response_text)

        result = await self._retry_with_backoff(_call_openai, "OpenAI")
        logger.info(f"OpenAI validation: confidence={result[0]:.2f}")
        return result

    async def validate_parallel(
        self, decision: Dict[str, Any]
    ) -> List[Tuple[float, str]]:
        """
        Validate decision using all available external APIs in parallel.

        Calls Claude and OpenAI APIs concurrently using asyncio.gather.
        Skips APIs with missing keys.

        Args:
            decision: Dict containing incident context and Brain reasoning

        Returns:
            List of (confidence_score, critique_text) tuples from each API.
            Empty list if no APIs configured, error tuple if all fail.

        Safety:
            - Fail-closed: Each API failure returns (0.0, "ERROR: ...")
            - Missing API keys: Skip that API, log INFO
            - Both missing: Return [(0.0, "ERROR: No external APIs configured")]
        """
        tasks = []
        api_names = []

        if self._anthropic_client:
            tasks.append(self.validate_with_claude(decision))
            api_names.append("Claude")
        else:
            logger.info("Skipping Claude validation (no API key)")

        if self._openai_client:
            tasks.append(self.validate_with_openai(decision))
            api_names.append("OpenAI")
        else:
            logger.info("Skipping OpenAI validation (no API key)")

        if not tasks:
            logger.warning("No external APIs configured for validation")
            return [(0.0, "ERROR: No external APIs configured")]

        logger.info(f"Running parallel validation with: {', '.join(api_names)}")

        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Convert any exceptions to fail-closed results
        processed_results: List[Tuple[float, str]] = []
        for i, result in enumerate(results):
            if isinstance(result, Exception):
                logger.error(
                    f"{api_names[i]} validation raised exception: {result}",
                    exc_info=True
                )
                processed_results.append(
                    (0.0, f"ERROR: {api_names[i]} failed - {type(result).__name__}")
                )
            else:
                processed_results.append(result)

        logger.info(
            f"Parallel validation complete: "
            f"{[f'{api_names[i]}={r[0]:.2f}' for i, r in enumerate(processed_results)]}"
        )

        return processed_results
