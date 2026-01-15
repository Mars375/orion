"""
ORION Brain - Decision making logic.

Reasons about incidents and produces decisions.
N0 mode: Every decision MUST be NO_ACTION.
"""

import logging
import uuid
from datetime import datetime, timezone
from typing import Dict, Any

from bus.python.orion_bus import EventBus


logger = logging.getLogger(__name__)


class Brain:
    """
    Makes decisions about how to respond to incidents.

    N0 Mode Invariants:
    - Every decision MUST be NO_ACTION
    - Reasoning must be explicit
    - No policy evaluation (Phase 1)
    - No action triggers
    - No cooldowns (not needed in N0)

    This is reasoning only, not execution.
    """

    def __init__(
        self,
        event_bus: EventBus,
        autonomy_level: str = "N0",
        source_name: str = "orion-brain",
    ):
        """
        Initialize brain.

        Args:
            event_bus: Event bus for pub/sub
            autonomy_level: Current autonomy level (must be N0 for Phase 1)
            source_name: Source identifier for decisions
        """
        if autonomy_level != "N0":
            raise ValueError(f"Phase 1 only supports N0 mode, got: {autonomy_level}")

        self.bus = event_bus
        self.autonomy_level = autonomy_level
        self.source_name = source_name

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

    def _generate_reasoning(self, incident: Dict[str, Any]) -> str:
        """
        Generate reasoning for NO_ACTION decision.

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

    def decide(self, incident: Dict[str, Any]) -> Dict[str, Any]:
        """
        Make decision about incident.

        In N0 mode, always returns NO_ACTION.

        Args:
            incident: Incident from guardian

        Returns:
            Decision (always NO_ACTION in N0)
        """
        # In N0 mode, always decide NO_ACTION
        reasoning = self._generate_reasoning(incident)

        decision = self._create_decision(
            incident=incident,
            decision_type="NO_ACTION",
            reasoning=reasoning,
            safety_classification="SAFE",  # NO_ACTION is always safe
        )

        logger.info(
            f"Decision {decision['decision_id']} for incident "
            f"{incident['incident_id']}: {decision['decision_type']}"
        )
        logger.debug(f"Reasoning: {reasoning}")

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
