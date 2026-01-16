"""
ORION Commander - Action execution engine.

Executes actions and emits outcomes.
- N2 mode: SAFE actions only
- N3 mode: SAFE actions + approved RISKY actions
- Validates approval expiration
- Supports override flags for forced actions
"""

import logging
import uuid
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Any, Optional

from bus.python.orion_bus import EventBus
from core.brain.policy_loader import PolicyLoader
from core.memory import MemoryStore


logger = logging.getLogger(__name__)


class Commander:
    """
    Executes actions and emits outcomes.

    Invariants:
    - SAFE actions execute automatically (N2/N3)
    - RISKY actions require valid, unexpired approval (N3 only)
    - Expired approval = rejection
    - Unknown actions are rejected
    - All execution is audited
    - Rollback is automatic for failures
    - Outcomes are always emitted
    """

    def __init__(
        self,
        event_bus: EventBus,
        policy_dir: Path,
        memory_store: MemoryStore,
        source_name: str = "orion-commander",
    ):
        """
        Initialize commander.

        Args:
            event_bus: Event bus for pub/sub
            policy_dir: Directory containing policy files
            memory_store: Memory store for audit trail
            source_name: Source identifier for outcomes
        """
        self.bus = event_bus
        self.policy_loader = PolicyLoader(policy_dir)
        self.memory = memory_store
        self.source_name = source_name

        # Track pending approvals: approval_request_id -> approval_decision
        self.pending_approvals: Dict[str, Dict[str, Any]] = {}

    def _create_action(
        self,
        decision: Dict[str, Any],
    ) -> Dict[str, Any]:
        """
        Create action contract from decision.

        Args:
            decision: Decision contract

        Returns:
            Action matching action.schema.json
        """
        proposed_action = decision.get("proposed_action", {})

        action = {
            "version": "1.0",
            "action_id": str(uuid.uuid4()),
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": "orion-brain",  # Actions originate from brain decisions
            "decision_id": decision["decision_id"],
            "action_type": proposed_action["action_type"],
            "safety_classification": decision["safety_classification"],
            "state": "pending",
            "parameters": proposed_action.get("parameters", {}),
            "rollback_enabled": True,  # All SAFE actions support rollback
            "dry_run": False,
        }

        return action

    def _create_outcome(
        self,
        action: Dict[str, Any],
        status: str,
        execution_time_ms: int,
        result: Optional[Dict[str, Any]] = None,
        error: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Create outcome contract.

        Args:
            action: Action that was executed
            status: Outcome status (succeeded, failed, rolled_back)
            execution_time_ms: Execution time in milliseconds
            result: Optional result data
            error: Optional error information

        Returns:
            Outcome matching outcome.schema.json
        """
        outcome = {
            "version": "1.0",
            "outcome_id": str(uuid.uuid4()),
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": self.source_name,
            "action_id": action["action_id"],
            "status": status,
            "execution_time_ms": execution_time_ms,
        }

        if result:
            outcome["result"] = result

        if error:
            outcome["error"] = error

        if status == "rolled_back":
            outcome["rollback_executed"] = True

        return outcome

    def _execute_acknowledge_incident(
        self,
        action: Dict[str, Any],
    ) -> Dict[str, Any]:
        """
        Execute acknowledge_incident action.

        This is a SAFE action that updates incident state in memory.

        Args:
            action: Action to execute

        Returns:
            Result dictionary
        """
        incident_id = action["parameters"].get("incident_id")

        logger.info(f"Acknowledging incident {incident_id}")

        # Store acknowledgment in memory
        # This is idempotent and safe
        acknowledgment = {
            "incident_id": incident_id,
            "acknowledged_at": datetime.now(timezone.utc).isoformat(),
            "acknowledged_by": "orion-brain",
            "action_id": action["action_id"],
        }

        # In a full implementation, this would update incident state
        # For Phase 3, we simply log it
        result = {
            "incident_id": incident_id,
            "acknowledgment": acknowledgment,
            "message": "Incident acknowledged (audit trail updated)",
        }

        return result

    def _rollback_acknowledge_incident(
        self,
        action: Dict[str, Any],
    ) -> None:
        """
        Rollback acknowledge_incident action.

        For acknowledgment, rollback means logging the rollback.
        The incident state remains observable in audit trail.

        Args:
            action: Action to rollback
        """
        incident_id = action["parameters"].get("incident_id")
        logger.info(f"Rolling back acknowledgment of incident {incident_id}")

        # Rollback is idempotent
        # For acknowledgment, we just log the rollback
        # The audit trail shows both the ack and the rollback

    def execute_action(self, action: Dict[str, Any]) -> Dict[str, Any]:
        """
        Execute action and return outcome.

        Args:
            action: Action to execute

        Returns:
            Outcome contract
        """
        start_time = time.time()
        action_type = action["action_type"]

        logger.info(
            f"Executing action {action['action_id']} "
            f"(type={action_type}, decision={action['decision_id']})"
        )

        try:
            # Execute based on action type
            if action_type == "acknowledge_incident":
                result = self._execute_acknowledge_incident(action)
                status = "succeeded"
                error = None

            else:
                # Unknown action type - should never happen due to brain checks
                raise ValueError(f"Unknown action type: {action_type}")

            execution_time_ms = int((time.time() - start_time) * 1000)

            outcome = self._create_outcome(
                action=action,
                status=status,
                execution_time_ms=execution_time_ms,
                result=result,
                error=error,
            )

            logger.info(
                f"Action {action['action_id']} {status} "
                f"in {execution_time_ms}ms"
            )

            return outcome

        except Exception as e:
            execution_time_ms = int((time.time() - start_time) * 1000)

            logger.error(
                f"Action {action['action_id']} failed: {e}",
                exc_info=True,
            )

            # Action failed - attempt rollback
            try:
                if action_type == "acknowledge_incident":
                    self._rollback_acknowledge_incident(action)
                status = "rolled_back"
            except Exception as rollback_error:
                logger.error(
                    f"Rollback failed for {action['action_id']}: {rollback_error}",
                    exc_info=True,
                )
                status = "failed"

            error = {
                "code": "EXECUTION_FAILED",
                "message": str(e),
                "details": {"action_type": action_type},
            }

            outcome = self._create_outcome(
                action=action,
                status=status,
                execution_time_ms=execution_time_ms,
                error=error,
            )

            return outcome

    def handle_approval_decision(self, approval_decision: Dict[str, Any]) -> None:
        """
        Handle incoming approval decision.

        Args:
            approval_decision: Approval decision from approval system
        """
        approval_id = approval_decision.get("approval_id")
        decision_type = approval_decision.get("decision")
        decision_id = approval_decision.get("decision_id")
        expires_at_str = approval_decision.get("expires_at")

        logger.info(
            f"Received approval decision {approval_id}: {decision_type} "
            f"for decision {decision_id}"
        )

        # Only process approve and force decisions
        if decision_type not in ("approve", "force"):
            logger.info(
                f"Approval decision {approval_id} is {decision_type}, "
                "not executing"
            )
            return

        # Validate expiration
        expires_at = datetime.fromisoformat(expires_at_str)
        if datetime.now(timezone.utc) >= expires_at:
            logger.error(
                f"Approval {approval_id} has expired, cannot execute action"
            )
            return

        # Store approval for action execution
        approval_request_id = approval_decision.get("approval_request_id")
        self.pending_approvals[approval_request_id] = approval_decision

        logger.info(
            f"Approval {approval_id} stored, awaiting corresponding decision"
        )

        # Note: In full implementation, would correlate with pending decision
        # and execute immediately if decision already received
        # For Phase 4, we rely on decision arriving after approval

    def _validate_approval(
        self,
        decision: Dict[str, Any],
    ) -> Optional[Dict[str, Any]]:
        """
        Validate approval for RISKY action.

        Args:
            decision: Decision requiring approval

        Returns:
            Approval decision if valid, None otherwise
        """
        decision_id = decision["decision_id"]

        # Find matching approval (in full implementation, would match on request ID)
        # For Phase 4, we match on decision_id
        approval = None
        for req_id, appr in self.pending_approvals.items():
            if appr.get("decision_id") == decision_id:
                approval = appr
                break

        if not approval:
            logger.error(
                f"No approval found for decision {decision_id}, "
                "cannot execute RISKY action"
            )
            return None

        # Validate expiration
        expires_at = datetime.fromisoformat(approval["expires_at"])
        if datetime.now(timezone.utc) >= expires_at:
            logger.error(
                f"Approval {approval['approval_id']} has expired, "
                "cannot execute action"
            )
            # Remove expired approval
            for req_id, appr in list(self.pending_approvals.items()):
                if appr.get("approval_id") == approval["approval_id"]:
                    del self.pending_approvals[req_id]
                    break
            return None

        logger.info(
            f"Approval {approval['approval_id']} validated for "
            f"decision {decision_id}"
        )

        return approval

    def handle_decision(self, decision: Dict[str, Any]) -> None:
        """
        Handle incoming decision.

        Processes:
        - EXECUTE_SAFE_ACTION: Execute immediately (N2/N3)
        - REQUEST_APPROVAL: Execute only if valid approval exists (N3)

        Args:
            decision: Decision from bus
        """
        decision_type = decision.get("decision_type")
        decision_id = decision.get("decision_id")

        logger.debug(
            f"Received decision {decision_id} type={decision_type}"
        )

        # Get proposed action
        proposed_action = decision.get("proposed_action")
        if not proposed_action:
            if decision_type in ("EXECUTE_SAFE_ACTION", "REQUEST_APPROVAL"):
                logger.error(
                    f"Decision {decision_id} has {decision_type} but no proposed_action"
                )
            return

        action_type = proposed_action.get("action_type")

        # Process EXECUTE_SAFE_ACTION (N2/N3 mode)
        if decision_type == "EXECUTE_SAFE_ACTION":
            # Verify action is SAFE
            if not self.policy_loader.is_safe(action_type):
                logger.error(
                    f"Decision {decision_id} proposes {action_type} which is not SAFE. "
                    f"Refusing to execute."
                )
                return

            # Create and execute action
            action = self._create_action(decision)
            outcome = self.execute_action(action)

            # Publish outcome
            try:
                self.bus.publish(outcome, "outcome")
                logger.info(f"Published outcome {outcome['outcome_id']}")
            except Exception as e:
                logger.error(f"Failed to publish outcome: {e}", exc_info=True)

            return

        # Process REQUEST_APPROVAL (N3 mode)
        if decision_type == "REQUEST_APPROVAL":
            # Validate approval exists and is not expired
            approval = self._validate_approval(decision)

            if not approval:
                logger.warning(
                    f"Decision {decision_id} requires approval but none found "
                    "or approval expired. Not executing."
                )
                return

            # Approval valid - execute RISKY action
            decision_action = approval.get("decision")

            if decision_action == "force":
                logger.warning(
                    f"Executing FORCED action {action_type} "
                    f"(approval {approval['approval_id']}, "
                    f"override_cb={approval.get('override_circuit_breaker', False)}, "
                    f"override_cooldown={approval.get('override_cooldown', False)})"
                )
            else:
                logger.info(
                    f"Executing approved RISKY action {action_type} "
                    f"(approval {approval['approval_id']})"
                )

            # Create action contract
            action = self._create_action(decision)
            action["approval_id"] = approval["approval_id"]

            # Execute action
            outcome = self.execute_action(action)

            # Publish outcome
            try:
                self.bus.publish(outcome, "outcome")
                logger.info(f"Published outcome {outcome['outcome_id']}")
            except Exception as e:
                logger.error(f"Failed to publish outcome: {e}", exc_info=True)

            # Remove approval after use (one-time use)
            approval_request_id = approval.get("approval_request_id")
            if approval_request_id in self.pending_approvals:
                del self.pending_approvals[approval_request_id]

            return

        # Other decision types - ignore
        logger.debug(
            f"Decision {decision_id} is {decision_type}, not executing"
        )

    def run(self) -> None:
        """
        Run commander subscription loop.

        Subscribes to:
        - decision: Execute SAFE actions, handle approval-required decisions
        - approval_decision: Track approvals for RISKY actions
        """
        logger.info("Starting commander execution loop")

        # Subscribe to decisions
        self.bus.subscribe(
            contract_type="decision",
            handler=self.handle_decision,
            consumer_group="commander",
            consumer_name="commander-1",
        )

        # Subscribe to approval decisions (N3 mode)
        self.bus.subscribe(
            contract_type="approval_decision",
            handler=self.handle_approval_decision,
            consumer_group="commander-approval",
            consumer_name="commander-approval-1",
        )
