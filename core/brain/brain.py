"""
ORION Brain - Decision making logic.

Reasons about incidents and produces decisions.
- N0 mode: Every decision MUST be NO_ACTION
- N2 mode: SAFE actions may execute, RISKY actions result in NO_ACTION
"""

import logging
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Any, Optional

from bus.python.orion_bus import EventBus
from .policy_loader import PolicyLoader
from .cooldown_tracker import CooldownTracker
from .circuit_breaker import CircuitBreaker


logger = logging.getLogger(__name__)


class Brain:
    """
    Makes decisions about how to respond to incidents.

    N0 Mode Invariants:
    - Every decision MUST be NO_ACTION
    - Reasoning must be explicit

    N2 Mode Invariants:
    - SAFE actions may execute automatically
    - RISKY actions MUST result in NO_ACTION
    - All execution is rate-limited (cooldowns)
    - Circuit breaker prevents failure loops
    - Policies are source of truth
    """

    def __init__(
        self,
        event_bus: EventBus,
        autonomy_level: str = "N0",
        policy_dir: Optional[Path] = None,
        source_name: str = "orion-brain",
    ):
        """
        Initialize brain.

        Args:
            event_bus: Event bus for pub/sub
            autonomy_level: Current autonomy level ("N0" or "N2")
            policy_dir: Directory containing policy files (required for N2)
            source_name: Source identifier for decisions
        """
        if autonomy_level not in ("N0", "N2"):
            raise ValueError(f"Only N0 and N2 modes supported, got: {autonomy_level}")

        self.bus = event_bus
        self.autonomy_level = autonomy_level
        self.source_name = source_name

        # N2 mode requires policy enforcement
        if autonomy_level == "N2":
            if policy_dir is None:
                raise ValueError("N2 mode requires policy_dir")
            self.policy_loader = PolicyLoader(policy_dir)
            self.cooldown_tracker = CooldownTracker()
            self.circuit_breaker = CircuitBreaker()
        else:
            self.policy_loader = None
            self.cooldown_tracker = None
            self.circuit_breaker = None

    def _create_decision(
        self,
        incident: Dict[str, Any],
        decision_type: str,
        reasoning: str,
        safety_classification: str = "UNKNOWN",
    ) -> Dict[str, Any]:
        """
        Create decision contract.

        Args:
            incident: Incident to decide about
            decision_type: Decision type
            reasoning: Explanation for decision
            safety_classification: Safety classification

        Returns:
            Decision matching decision.schema.json
        """
        decision = {
            "version": "1.0",
            "decision_id": str(uuid.uuid4()),
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": self.source_name,
            "incident_id": incident["incident_id"],
            "decision_type": decision_type,
            "safety_classification": safety_classification,
            "requires_approval": False,  # N0 never requires approval (never acts)
            "reasoning": reasoning,
            "autonomy_level": self.autonomy_level,
        }

        return decision

    def _generate_reasoning_n0(self, incident: Dict[str, Any]) -> str:
        """
        Generate reasoning for NO_ACTION decision in N0 mode.

        Args:
            incident: Incident to reason about

        Returns:
            Reasoning string (minimum 10 characters)
        """
        incident_type = incident.get("incident_type", "unknown")
        severity = incident.get("severity", "unknown")
        event_count = len(incident.get("event_ids", []))

        reasoning = (
            f"N0 mode (observe only): Detected {incident_type} "
            f"(severity={severity}, events={event_count}). "
            f"No action taken as per autonomy level N0 policy."
        )

        return reasoning

    def _determine_action_type(self, incident: Dict[str, Any]) -> Optional[str]:
        """
        Determine appropriate action type for incident.

        Args:
            incident: Incident to analyze

        Returns:
            Action type or None if no action appropriate
        """
        # For Phase 3, only acknowledge_incident is implemented
        # This is a SAFE action that updates incident state
        incident_type = incident.get("incident_type")

        # Only acknowledge incidents that are meaningful
        if incident.get("severity") in ("medium", "high", "critical"):
            return "acknowledge_incident"

        return None

    def decide(self, incident: Dict[str, Any]) -> Dict[str, Any]:
        """
        Make decision about incident.

        - N0 mode: Always returns NO_ACTION
        - N2 mode: May return EXECUTE_SAFE_ACTION or NO_ACTION

        Args:
            incident: Incident from guardian

        Returns:
            Decision contract
        """
        if self.autonomy_level == "N0":
            return self._decide_n0(incident)
        elif self.autonomy_level == "N2":
            return self._decide_n2(incident)
        else:
            # Fail closed
            return self._decide_n0(incident)

    def _decide_n0(self, incident: Dict[str, Any]) -> Dict[str, Any]:
        """
        Make decision in N0 mode (always NO_ACTION).

        Args:
            incident: Incident to decide about

        Returns:
            Decision (always NO_ACTION)
        """
        reasoning = self._generate_reasoning_n0(incident)

        decision = self._create_decision(
            incident=incident,
            decision_type="NO_ACTION",
            reasoning=reasoning,
            safety_classification="SAFE",
        )

        logger.info(
            f"Decision {decision['decision_id']} for incident "
            f"{incident['incident_id']}: NO_ACTION (N0)"
        )

        return decision

    def _decide_n2(self, incident: Dict[str, Any]) -> Dict[str, Any]:
        """
        Make decision in N2 mode (SAFE actions allowed).

        Args:
            incident: Incident to decide about

        Returns:
            Decision contract
        """
        # Determine if action is appropriate
        action_type = self._determine_action_type(incident)

        if action_type is None:
            # No action needed
            reasoning = (
                f"N2 mode: Incident {incident.get('incident_type')} detected "
                f"but no action required (severity={incident.get('severity')})."
            )
            return self._create_decision(
                incident=incident,
                decision_type="NO_ACTION",
                reasoning=reasoning,
                safety_classification="SAFE",
            )

        # Check policy classification
        classification = self.policy_loader.classify_action(action_type)

        if classification == "RISKY":
            # RISKY actions forbidden in N2 (would require N3 with approval)
            reasoning = (
                f"N2 mode: Action {action_type} is RISKY and requires approval. "
                f"No approval mechanism available. No action taken."
            )
            return self._create_decision(
                incident=incident,
                decision_type="NO_ACTION",
                reasoning=reasoning,
                safety_classification="RISKY",
            )

        if classification == "UNKNOWN":
            # Unknown actions treated as RISKY (fail closed)
            reasoning = (
                f"N2 mode: Action {action_type} classification unknown. "
                f"Treating as RISKY. No action taken."
            )
            return self._create_decision(
                incident=incident,
                decision_type="NO_ACTION",
                reasoning=reasoning,
                safety_classification="UNKNOWN",
            )

        # Action is SAFE - check cooldown
        cooldown = self.policy_loader.get_cooldown(action_type)
        if cooldown and not self.cooldown_tracker.check_cooldown(action_type, cooldown):
            remaining = self.cooldown_tracker.get_remaining_cooldown(action_type, cooldown)
            reasoning = (
                f"N2 mode: Action {action_type} is SAFE but in cooldown "
                f"({remaining:.1f}s remaining). No action taken."
            )
            return self._create_decision(
                incident=incident,
                decision_type="NO_ACTION",
                reasoning=reasoning,
                safety_classification="SAFE",
            )

        # Check circuit breaker
        if self.circuit_breaker.is_open(action_type):
            reasoning = (
                f"N2 mode: Action {action_type} circuit breaker is OPEN "
                f"(too many recent failures). No action taken."
            )
            return self._create_decision(
                incident=incident,
                decision_type="NO_ACTION",
                reasoning=reasoning,
                safety_classification="SAFE",
            )

        # All checks passed - execute SAFE action
        reasoning = (
            f"N2 mode: Executing SAFE action {action_type} for incident "
            f"{incident.get('incident_type')} (severity={incident.get('severity')})."
        )

        decision = self._create_decision(
            incident=incident,
            decision_type="EXECUTE_SAFE_ACTION",
            reasoning=reasoning,
            safety_classification="SAFE",
        )

        # Add proposed action
        decision["proposed_action"] = {
            "action_type": action_type,
            "parameters": {
                "incident_id": incident["incident_id"],
            },
        }

        logger.info(
            f"Decision {decision['decision_id']}: EXECUTE_SAFE_ACTION "
            f"({action_type}) for incident {incident['incident_id']}"
        )

        # Record cooldown
        if cooldown:
            self.cooldown_tracker.record_execution(action_type)

        return decision

    def handle_incident(self, incident: Dict[str, Any]) -> None:
        """
        Handle incoming incident.

        Args:
            incident: Incident from bus
        """
        logger.debug(
            f"Received incident {incident.get('incident_id')} "
            f"type={incident.get('incident_type')}"
        )

        # Make decision
        decision = self.decide(incident)

        # Publish decision to bus
        try:
            self.bus.publish(decision, "decision")
            logger.info(f"Published decision {decision['decision_id']}")
        except Exception as e:
            logger.error(f"Failed to publish decision: {e}", exc_info=True)

    def run(self) -> None:
        """
        Run brain subscription loop.

        Subscribes to incidents and makes decisions.
        """
        logger.info(f"Starting brain decision loop (autonomy={self.autonomy_level})")

        self.bus.subscribe(
            contract_type="incident",
            handler=self.handle_incident,
            consumer_group="brain",
            consumer_name="brain-1",
        )
