"""
Approval Coordinator - Central approval request tracking and timeout handling.

Manages approval lifecycle:
1. Receives approval requests from brain
2. Tracks pending approvals with expiration
3. Routes to approval channels (Telegram, CLI)
4. Validates admin decisions
5. Emits approval_decision contracts
6. Handles timeouts (escalate, never execute)
"""

import logging
import uuid
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, Optional
from pathlib import Path

from bus.python.orion_bus import EventBus
from .admin_identity import AdminIdentity


logger = logging.getLogger(__name__)


class ApprovalCoordinator:
    """
    Coordinates approval requests and decisions.

    Invariants:
    - Silence is NEVER permission
    - Timeout = escalation, never execution
    - All approvals expire (time-limited)
    - Expired approval = invalid approval
    - Only ADMIN can approve
    """

    def __init__(
        self,
        event_bus: EventBus,
        admin_identity: AdminIdentity,
        source_name: str = "orion-approval-coordinator",
        default_approval_timeout: int = 300,  # 5 minutes
    ):
        """
        Initialize approval coordinator.

        Args:
            event_bus: Event bus for pub/sub
            admin_identity: Admin identity validator
            source_name: Source identifier
            default_approval_timeout: Default timeout in seconds
        """
        self.bus = event_bus
        self.admin_identity = admin_identity
        self.source_name = source_name
        self.default_approval_timeout = default_approval_timeout

        # Track pending approvals: approval_request_id -> request
        self.pending_approvals: Dict[str, Dict[str, Any]] = {}

        # Track approval decisions: approval_request_id -> decision
        self.decisions: Dict[str, Dict[str, Any]] = {}

    def handle_approval_request(self, request: Dict[str, Any]) -> None:
        """
        Handle incoming approval request from brain.

        Args:
            request: Approval request contract
        """
        request_id = request["approval_request_id"]
        action_type = request["action_type"]
        expires_at = datetime.fromisoformat(request["expires_at"])

        logger.info(
            f"Received approval request {request_id} for "
            f"action={action_type}, expires={expires_at}"
        )

        # Check if already expired (should not happen, but fail closed)
        if datetime.now(timezone.utc) >= expires_at:
            logger.error(
                f"Approval request {request_id} already expired, escalating"
            )
            self._escalate_timeout(request)
            return

        # Store pending approval
        self.pending_approvals[request_id] = request

        # Route to approval channels
        # For now, just log - channels will be implemented separately
        logger.info(
            f"Approval request {request_id} pending, awaiting ADMIN decision"
        )

    def approve(
        self,
        approval_request_id: str,
        admin_identity: str,
        channel: str,
        reason: str,
        approval_timeout: Optional[int] = None,
    ) -> Optional[Dict[str, Any]]:
        """
        Admin approves a RISKY action.

        Args:
            approval_request_id: Request being approved
            admin_identity: Identity of admin making decision
            channel: Channel ("telegram" or "cli")
            reason: Mandatory reason for approval
            approval_timeout: Optional custom approval timeout in seconds

        Returns:
            Approval decision contract or None if rejected
        """
        # Verify admin identity
        if channel == "telegram":
            if not self.admin_identity.verify_telegram(admin_identity):
                logger.error(
                    f"Approval rejected: Telegram identity {admin_identity} "
                    "does not match ADMIN"
                )
                return None
        elif channel == "cli":
            if not self.admin_identity.verify_cli(admin_identity):
                logger.error(
                    f"Approval rejected: CLI identity {admin_identity} "
                    "does not match ADMIN"
                )
                return None
        else:
            logger.error(f"Unknown approval channel: {channel}")
            return None

        # Check if request exists
        if approval_request_id not in self.pending_approvals:
            logger.error(
                f"Approval request {approval_request_id} not found or already processed"
            )
            return None

        request = self.pending_approvals[approval_request_id]

        # Check if request expired
        expires_at = datetime.fromisoformat(request["expires_at"])
        if datetime.now(timezone.utc) >= expires_at:
            logger.error(
                f"Approval request {approval_request_id} expired, cannot approve"
            )
            self._escalate_timeout(request)
            del self.pending_approvals[approval_request_id]
            return None

        # Validate reason
        if not reason or len(reason.strip()) == 0:
            logger.error("Approval rejected: reason is mandatory")
            return None

        # Create approval decision
        timeout = approval_timeout or self.default_approval_timeout
        decision = self._create_approval_decision(
            request=request,
            decision_type="approve",
            admin_identity=admin_identity,
            channel=channel,
            reason=reason,
            approval_timeout=timeout,
        )

        # Store decision
        self.decisions[approval_request_id] = decision

        # Remove from pending
        del self.pending_approvals[approval_request_id]

        # Publish decision to bus
        try:
            self.bus.publish(decision, "approval_decision")
            logger.info(
                f"Approval decision {decision['approval_id']} published: "
                f"APPROVE action={request['action_type']}"
            )
        except Exception as e:
            logger.error(f"Failed to publish approval decision: {e}", exc_info=True)

        return decision

    def deny(
        self,
        approval_request_id: str,
        admin_identity: str,
        channel: str,
        reason: str,
    ) -> Optional[Dict[str, Any]]:
        """
        Admin denies a RISKY action.

        Args:
            approval_request_id: Request being denied
            admin_identity: Identity of admin making decision
            channel: Channel ("telegram" or "cli")
            reason: Mandatory reason for denial

        Returns:
            Approval decision contract or None if rejected
        """
        # Verify admin identity
        if channel == "telegram":
            if not self.admin_identity.verify_telegram(admin_identity):
                logger.error(
                    f"Denial rejected: Telegram identity {admin_identity} "
                    "does not match ADMIN"
                )
                return None
        elif channel == "cli":
            if not self.admin_identity.verify_cli(admin_identity):
                logger.error(
                    f"Denial rejected: CLI identity {admin_identity} "
                    "does not match ADMIN"
                )
                return None
        else:
            logger.error(f"Unknown approval channel: {channel}")
            return None

        # Check if request exists
        if approval_request_id not in self.pending_approvals:
            logger.error(
                f"Approval request {approval_request_id} not found or already processed"
            )
            return None

        request = self.pending_approvals[approval_request_id]

        # Validate reason
        if not reason or len(reason.strip()) == 0:
            logger.error("Denial rejected: reason is mandatory")
            return None

        # Create denial decision
        decision = self._create_approval_decision(
            request=request,
            decision_type="deny",
            admin_identity=admin_identity,
            channel=channel,
            reason=reason,
            approval_timeout=0,  # Denials don't need expiration
        )

        # Store decision
        self.decisions[approval_request_id] = decision

        # Remove from pending
        del self.pending_approvals[approval_request_id]

        # Publish decision to bus
        try:
            self.bus.publish(decision, "approval_decision")
            logger.info(
                f"Approval decision {decision['approval_id']} published: "
                f"DENY action={request['action_type']}"
            )
        except Exception as e:
            logger.error(f"Failed to publish approval decision: {e}", exc_info=True)

        return decision

    def force(
        self,
        approval_request_id: str,
        admin_identity: str,
        channel: str,
        reason: str,
        override_circuit_breaker: bool = False,
        override_cooldown: bool = False,
        approval_timeout: Optional[int] = None,
    ) -> Optional[Dict[str, Any]]:
        """
        Admin forces a RISKY action (bypass safety checks).

        Args:
            approval_request_id: Request being forced
            admin_identity: Identity of admin making decision
            channel: Channel ("telegram" or "cli")
            reason: Mandatory reason for force
            override_circuit_breaker: Temporarily disable circuit breaker
            override_cooldown: Temporarily disable cooldown
            approval_timeout: Optional custom approval timeout in seconds

        Returns:
            Approval decision contract or None if rejected
        """
        # Verify admin identity
        if channel == "telegram":
            if not self.admin_identity.verify_telegram(admin_identity):
                logger.error(
                    f"Force rejected: Telegram identity {admin_identity} "
                    "does not match ADMIN"
                )
                return None
        elif channel == "cli":
            if not self.admin_identity.verify_cli(admin_identity):
                logger.error(
                    f"Force rejected: CLI identity {admin_identity} "
                    "does not match ADMIN"
                )
                return None
        else:
            logger.error(f"Unknown approval channel: {channel}")
            return None

        # Check if request exists
        if approval_request_id not in self.pending_approvals:
            logger.error(
                f"Approval request {approval_request_id} not found or already processed"
            )
            return None

        request = self.pending_approvals[approval_request_id]

        # Validate reason (force requires strong justification)
        if not reason or len(reason.strip()) < 10:
            logger.error("Force rejected: reason must be at least 10 characters")
            return None

        # Create force decision
        timeout = approval_timeout or self.default_approval_timeout
        decision = self._create_approval_decision(
            request=request,
            decision_type="force",
            admin_identity=admin_identity,
            channel=channel,
            reason=reason,
            approval_timeout=timeout,
            override_circuit_breaker=override_circuit_breaker,
            override_cooldown=override_cooldown,
        )

        # Store decision
        self.decisions[approval_request_id] = decision

        # Remove from pending
        del self.pending_approvals[approval_request_id]

        # Publish decision to bus
        try:
            self.bus.publish(decision, "approval_decision")
            logger.warning(
                f"Approval decision {decision['approval_id']} published: "
                f"FORCE action={request['action_type']} "
                f"(CB_override={override_circuit_breaker}, "
                f"cooldown_override={override_cooldown})"
            )
        except Exception as e:
            logger.error(f"Failed to publish approval decision: {e}", exc_info=True)

        return decision

    def _create_approval_decision(
        self,
        request: Dict[str, Any],
        decision_type: str,
        admin_identity: str,
        channel: str,
        reason: str,
        approval_timeout: int,
        override_circuit_breaker: bool = False,
        override_cooldown: bool = False,
    ) -> Dict[str, Any]:
        """
        Create approval decision contract.

        Args:
            request: Approval request being responded to
            decision_type: "approve", "deny", or "force"
            admin_identity: ADMIN identity
            channel: Channel ("telegram" or "cli")
            reason: Decision reason
            approval_timeout: Approval timeout in seconds
            override_circuit_breaker: Override circuit breaker
            override_cooldown: Override cooldown

        Returns:
            Approval decision contract
        """
        now = datetime.now(timezone.utc)
        expires_at = now + timedelta(seconds=approval_timeout)

        source = f"orion-approval-{channel}"

        decision = {
            "version": "1.0",
            "approval_id": str(uuid.uuid4()),
            "timestamp": now.isoformat(),
            "source": source,
            "approval_request_id": request["approval_request_id"],
            "decision_id": request["decision_id"],
            "decision": decision_type,
            "admin_identity": admin_identity,
            "reason": reason,
            "issued_at": now.isoformat(),
            "expires_at": expires_at.isoformat(),
        }

        # Add overrides only for force decisions
        if decision_type == "force":
            decision["override_circuit_breaker"] = override_circuit_breaker
            decision["override_cooldown"] = override_cooldown

        # Add action_id for approve/force decisions
        if decision_type in ("approve", "force"):
            decision["action_id"] = str(uuid.uuid4())

        return decision

    def _escalate_timeout(self, request: Dict[str, Any]) -> None:
        """
        Escalate timed-out approval request.

        Logs escalation and optionally notifies admin.
        NEVER executes action on timeout.

        Args:
            request: Timed-out approval request
        """
        request_id = request["approval_request_id"]
        action_type = request["action_type"]

        logger.error(
            f"ESCALATION: Approval request {request_id} timed out. "
            f"Action {action_type} NOT executed. "
            f"Human unavailable, system in safe inaction."
        )

        # TODO: Notify admin via configured channels
        # For now, just log

    def check_expired_approvals(self) -> None:
        """
        Check for expired approval requests and escalate.

        Should be called periodically.
        """
        now = datetime.now(timezone.utc)
        expired = []

        for request_id, request in self.pending_approvals.items():
            expires_at = datetime.fromisoformat(request["expires_at"])
            if now >= expires_at:
                expired.append(request_id)

        for request_id in expired:
            request = self.pending_approvals[request_id]
            logger.warning(
                f"Approval request {request_id} expired, escalating"
            )
            self._escalate_timeout(request)
            del self.pending_approvals[request_id]

    def run(self) -> None:
        """
        Run approval coordinator subscription loop.

        Subscribes to approval_request contracts from brain.
        """
        logger.info("Starting approval coordinator")

        self.bus.subscribe(
            contract_type="approval_request",
            handler=self.handle_approval_request,
            consumer_group="approval-coordinator",
            consumer_name="coordinator-1",
        )
