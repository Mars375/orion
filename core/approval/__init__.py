"""ORION Approval System - Human authority and approvals for RISKY actions."""

from .approval_coordinator import ApprovalCoordinator
from .admin_identity import AdminIdentity

__all__ = ["ApprovalCoordinator", "AdminIdentity"]
