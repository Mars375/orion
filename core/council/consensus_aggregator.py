"""
ORION Consensus Aggregator - Confidence-weighted voting with safety veto.

Combines validations from local SLM and external APIs using confidence-weighted
voting (not simple majority), implements safety veto for high-confidence concerns,
and orchestrates staged validation (local → external escalation).

Voting Algorithm (based on ReConcile research):
- Confidence-weighted average: sum(conf_i * vote_i) / sum(conf_i)
- Each critique is parsed for keywords to determine vote (1.0=APPROVE, 0.0=BLOCK)
- Safety veto: Any high-confidence safety concern blocks action immediately
- Escalation: Uncertain or RISKY decisions escalate to external APIs

Safety Invariants:
- Safety veto: If ANY validator flags concern with confidence > 0.8 → BLOCK
- Fail-closed: All errors result in BLOCKED status
- Conservative: Default to blocking when uncertain
- No negotiation: Council validates only, never proposes alternatives
"""

import logging
import re
from typing import Any, Dict, List, Optional, Tuple, TYPE_CHECKING

if TYPE_CHECKING:
    from .council_validator import CouncilValidator
    from .external_validator import ExternalValidator


logger = logging.getLogger(__name__)


# Default thresholds
DEFAULT_CONFIDENCE_THRESHOLD = 0.7
DEFAULT_SAFETY_VETO_THRESHOLD = 0.8

# Keywords for parsing critique text
APPROVE_KEYWORDS = ["approve", "approved", "safe", "correct", "valid", "agree", "confident"]
BLOCK_KEYWORDS = ["block", "blocked", "unsafe", "risky", "concern", "reject", "invalid", "dangerous", "error"]


class ConsensusAggregator:
    """
    Aggregates validations using confidence-weighted voting with safety veto.

    Implements SOTA voting algorithm from ReConcile research:
    - Confidence-weighted voting (not simple majority)
    - Safety veto for high-confidence concerns
    - Staged validation: local first, escalate if needed

    Invariants:
    - Conservative: Default to BLOCKED when uncertain
    - Safety veto: High-confidence concern always blocks
    - Fail-closed: Errors return BLOCKED status
    - No negotiation: Validates only, never proposes alternatives
    """

    def __init__(
        self,
        confidence_threshold: float = DEFAULT_CONFIDENCE_THRESHOLD,
        safety_veto_threshold: float = DEFAULT_SAFETY_VETO_THRESHOLD,
    ) -> None:
        """
        Initialize consensus aggregator.

        Args:
            confidence_threshold: Minimum weighted confidence to approve (default: 0.7)
                Below this threshold, escalate to external APIs or block.
            safety_veto_threshold: Confidence threshold for safety veto (default: 0.8)
                If any validator flags concern with confidence >= this, block immediately.
        """
        self.confidence_threshold = confidence_threshold
        self.safety_veto_threshold = safety_veto_threshold

        logger.info(
            f"ConsensusAggregator initialized: "
            f"conf_threshold={confidence_threshold}, "
            f"safety_veto_threshold={safety_veto_threshold}"
        )

    def _parse_critique_vote(self, critique: str) -> float:
        """
        Parse critique text to determine vote (1.0=APPROVE, 0.0=BLOCK).

        Uses naive keyword matching to classify critique sentiment.

        Args:
            critique: Critique text from validator

        Returns:
            Vote value: 1.0 if critique suggests approval, 0.0 if suggests blocking
        """
        critique_lower = critique.lower()

        # Check for block keywords first (safety priority)
        for keyword in BLOCK_KEYWORDS:
            if keyword in critique_lower:
                logger.debug(f"Critique contains block keyword '{keyword}': {critique[:50]}...")
                return 0.0

        # Check for approve keywords
        for keyword in APPROVE_KEYWORDS:
            if keyword in critique_lower:
                logger.debug(f"Critique contains approve keyword '{keyword}': {critique[:50]}...")
                return 1.0

        # Default to conservative (block) when uncertain
        logger.debug(f"No keywords found in critique, defaulting to block: {critique[:50]}...")
        return 0.0

    def _has_safety_concern(self, critique: str) -> bool:
        """
        Check if critique contains safety-related concerns.

        Args:
            critique: Critique text from validator

        Returns:
            True if critique contains safety concern keywords
        """
        critique_lower = critique.lower()
        safety_keywords = ["unsafe", "risky", "concern", "dangerous", "violation", "hazard"]

        for keyword in safety_keywords:
            if keyword in critique_lower:
                return True

        return False

    def aggregate_votes(
        self, validations: List[Tuple[float, str]]
    ) -> Tuple[str, float, str]:
        """
        Aggregate validations using confidence-weighted voting.

        Algorithm:
        1. Parse each critique to determine vote (1.0=APPROVE, 0.0=BLOCK)
        2. Calculate weighted average: sum(conf_i * vote_i) / sum(conf_i)
        3. If weighted_avg >= confidence_threshold: APPROVED
        4. Otherwise: BLOCKED

        Args:
            validations: List of (confidence, critique) tuples from validators

        Returns:
            Tuple of (result, weighted_confidence, combined_critique):
            - result: "APPROVED" or "BLOCKED"
            - weighted_confidence: Confidence-weighted average (0.0-1.0)
            - combined_critique: Combined critique from all validators
        """
        if not validations:
            logger.warning("No validations to aggregate")
            return ("BLOCKED", 0.0, "No validations provided")

        # Filter out zero-confidence validations (errors)
        valid_validations = [(c, t) for c, t in validations if c > 0.0]

        if not valid_validations:
            # All validators failed
            combined = "; ".join([critique for _, critique in validations])
            logger.warning(f"All validations failed: {combined}")
            return ("BLOCKED", 0.0, f"All validators failed: {combined}")

        # Calculate confidence-weighted vote
        total_weight = sum(conf for conf, _ in valid_validations)
        weighted_sum = sum(
            conf * self._parse_critique_vote(critique)
            for conf, critique in valid_validations
        )

        weighted_avg = weighted_sum / total_weight if total_weight > 0 else 0.0

        # Combine critiques
        combined_critique = " | ".join(
            f"[{conf:.2f}] {critique[:100]}"
            for conf, critique in valid_validations
        )

        # Determine result
        if weighted_avg >= self.confidence_threshold:
            result = "APPROVED"
            logger.info(
                f"Vote aggregation: APPROVED (weighted_avg={weighted_avg:.2f} >= {self.confidence_threshold})"
            )
        else:
            result = "BLOCKED"
            logger.info(
                f"Vote aggregation: BLOCKED (weighted_avg={weighted_avg:.2f} < {self.confidence_threshold})"
            )

        return (result, weighted_avg, combined_critique)

    def safety_veto(
        self, validations: List[Tuple[float, str]]
    ) -> Optional[str]:
        """
        Check if any validator has flagged a high-confidence safety concern.

        If ANY validator expresses safety concerns with confidence >= safety_veto_threshold,
        the action is vetoed (blocked) regardless of other votes.

        Args:
            validations: List of (confidence, critique) tuples from validators

        Returns:
            Veto reason string if safety veto triggered, None otherwise
        """
        for i, (confidence, critique) in enumerate(validations):
            if confidence >= self.safety_veto_threshold and self._has_safety_concern(critique):
                veto_reason = (
                    f"BLOCKED: Safety veto triggered by validator {i + 1} "
                    f"(confidence={confidence:.2f}): {critique[:100]}"
                )
                logger.warning(veto_reason)
                return veto_reason

        return None

    def should_escalate(
        self, local_confidence: float, decision_classification: str
    ) -> bool:
        """
        Determine if decision should escalate to external APIs.

        Escalation criteria:
        1. Local confidence is below threshold (uncertain)
        2. Decision is classified as RISKY (requires higher scrutiny)

        Args:
            local_confidence: Confidence score from local SLM (0.0-1.0)
            decision_classification: Safety classification ("SAFE" or "RISKY")

        Returns:
            True if external API validation should be requested
        """
        # Escalate if local confidence is too low
        if local_confidence < self.confidence_threshold:
            logger.info(
                f"Escalating: local confidence {local_confidence:.2f} < {self.confidence_threshold}"
            )
            return True

        # Escalate if decision is RISKY
        if decision_classification.upper() == "RISKY":
            logger.info(f"Escalating: decision classified as RISKY")
            return True

        logger.debug(
            f"No escalation needed: confidence={local_confidence:.2f}, "
            f"classification={decision_classification}"
        )
        return False

    async def validate_decision(
        self,
        decision: Dict[str, Any],
        local_validator: "CouncilValidator",
        external_validator: "ExternalValidator",
    ) -> Tuple[str, float, str]:
        """
        Orchestrate staged validation flow for a decision.

        Stages:
        1. Call local SLM validator
        2. Check if escalation needed (uncertain or RISKY)
        3. If escalating, call external APIs in parallel
        4. Check safety veto
        5. Aggregate votes and return result

        Args:
            decision: Decision dict containing incident context and Brain reasoning
            local_validator: CouncilValidator instance for local SLM
            external_validator: ExternalValidator instance for cloud APIs

        Returns:
            Tuple of (result, confidence, critique):
            - result: "APPROVED", "BLOCKED", or "ESCALATE_TO_ADMIN"
            - confidence: Final weighted confidence score
            - critique: Combined critique from all validators

        Safety:
            - Fail-closed: Any error returns ("BLOCKED", 0.0, error_message)
            - Safety veto: High-confidence concern blocks immediately
            - Conservative: Default to blocking when uncertain
        """
        all_validations: List[Tuple[float, str]] = []
        classification = decision.get("safety_classification", "UNKNOWN")

        logger.info(f"Starting validation for decision (classification={classification})")

        # Stage 1: Local SLM validation
        logger.info("Stage 1: Local SLM validation")
        try:
            local_confidence, local_critique = local_validator.validate(decision)
            all_validations.append((local_confidence, f"[Local] {local_critique}"))
            logger.info(f"Local validation: confidence={local_confidence:.2f}")
        except Exception as e:
            logger.error(f"Local validation failed: {e}", exc_info=True)
            all_validations.append((0.0, f"[Local] ERROR: {type(e).__name__}"))
            local_confidence = 0.0

        # Stage 2: Check if escalation needed
        logger.info("Stage 2: Checking escalation")
        if self.should_escalate(local_confidence, classification):
            logger.info("Stage 2a: Escalating to external APIs")
            try:
                external_results = await external_validator.validate_parallel(decision)
                for i, (ext_conf, ext_critique) in enumerate(external_results):
                    validator_name = "Claude" if i == 0 else "OpenAI"
                    all_validations.append((ext_conf, f"[{validator_name}] {ext_critique}"))
                    logger.info(f"{validator_name} validation: confidence={ext_conf:.2f}")
            except Exception as e:
                logger.error(f"External validation failed: {e}", exc_info=True)
                all_validations.append((0.0, f"[External] ERROR: {type(e).__name__}"))
        else:
            logger.info("Stage 2: No escalation needed")

        # Stage 3: Safety veto check
        logger.info("Stage 3: Safety veto check")
        veto_reason = self.safety_veto(all_validations)
        if veto_reason:
            return ("BLOCKED", 0.0, veto_reason)

        # Stage 4: Aggregate votes
        logger.info("Stage 4: Aggregating votes")
        result, confidence, critique = self.aggregate_votes(all_validations)

        # Check if result requires admin escalation
        # If RISKY decision is approved with moderate confidence, escalate to admin
        if (
            result == "APPROVED"
            and classification.upper() == "RISKY"
            and confidence < 0.9  # High bar for RISKY auto-approval
        ):
            logger.warning(
                f"RISKY decision approved with confidence {confidence:.2f} < 0.9, "
                f"escalating to admin"
            )
            return ("ESCALATE_TO_ADMIN", confidence, critique)

        logger.info(f"Validation complete: {result} (confidence={confidence:.2f})")
        return (result, confidence, critique)
