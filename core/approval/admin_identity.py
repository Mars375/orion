"""
Admin Identity Management.

Enforces single ADMIN identity model:
- Exactly ONE human authority in the system
- ADMIN identity must be explicitly configured
- Any non-matching identity is rejected
- No delegation, no quorum, no multi-admin
"""

import logging
from typing import Optional
from pathlib import Path
import yaml


logger = logging.getLogger(__name__)


class AdminIdentity:
    """
    Validates admin identity for approval decisions.

    Invariants:
    - Only ONE admin identity per system
    - Identity must be explicitly configured
    - Unknown identity = rejected
    - No implicit defaults
    """

    def __init__(self, config_file: Optional[Path] = None):
        """
        Initialize admin identity validator.

        Args:
            config_file: Path to admin config file (required)
        """
        if config_file is None:
            raise ValueError("Admin identity requires explicit configuration file")

        if not config_file.exists():
            raise ValueError(f"Admin config file not found: {config_file}")

        self.config_file = config_file
        self._load_config()

    def _load_config(self) -> None:
        """Load admin configuration from YAML file."""
        try:
            with open(self.config_file, 'r') as f:
                config = yaml.safe_load(f)

            if not config or 'admin' not in config:
                raise ValueError("Config must contain 'admin' section")

            admin_config = config['admin']

            # Telegram admin identity
            self.telegram_chat_id = admin_config.get('telegram_chat_id')

            # CLI admin identity (username or UID)
            self.cli_identity = admin_config.get('cli_identity')

            # At least one channel must be configured
            if not self.telegram_chat_id and not self.cli_identity:
                raise ValueError("At least one admin identity (telegram or cli) must be configured")

            logger.info(
                f"Admin identity loaded: "
                f"telegram={bool(self.telegram_chat_id)}, "
                f"cli={bool(self.cli_identity)}"
            )

        except Exception as e:
            logger.error(f"Failed to load admin config: {e}")
            raise

    def verify_telegram(self, chat_id: str) -> bool:
        """
        Verify Telegram identity matches ADMIN.

        Args:
            chat_id: Telegram chat ID to verify

        Returns:
            True if matches ADMIN, False otherwise
        """
        if self.telegram_chat_id is None:
            logger.warning("Telegram admin not configured, rejecting")
            return False

        is_valid = str(chat_id) == str(self.telegram_chat_id)

        if not is_valid:
            logger.warning(
                f"Telegram identity mismatch: got {chat_id}, "
                f"expected {self.telegram_chat_id}"
            )

        return is_valid

    def verify_cli(self, identity: str) -> bool:
        """
        Verify CLI identity matches ADMIN.

        Args:
            identity: CLI identity to verify (username or UID)

        Returns:
            True if matches ADMIN, False otherwise
        """
        if self.cli_identity is None:
            logger.warning("CLI admin not configured, rejecting")
            return False

        is_valid = identity == self.cli_identity

        if not is_valid:
            logger.warning(
                f"CLI identity mismatch: got {identity}, "
                f"expected {self.cli_identity}"
            )

        return is_valid

    def get_admin_identity(self, channel: str) -> Optional[str]:
        """
        Get configured admin identity for channel.

        Args:
            channel: Channel name ("telegram" or "cli")

        Returns:
            Admin identity string or None if not configured
        """
        if channel == "telegram":
            return self.telegram_chat_id
        elif channel == "cli":
            return self.cli_identity
        else:
            logger.error(f"Unknown channel: {channel}")
            return None
