"""
Contract validation for ORION event bus.

Validates messages against JSON schemas before publishing.
"""

import json
from pathlib import Path
from typing import Dict, Any

from jsonschema import Draft202012Validator, ValidationError


class ContractValidator:
    """Validates messages against ORION contracts."""

    def __init__(self, contracts_dir: Path):
        """
        Initialize validator with contracts directory.

        Args:
            contracts_dir: Path to directory containing JSON schema files
        """
        self.contracts_dir = contracts_dir
        self._schemas: Dict[str, Dict[str, Any]] = {}
        self._validators: Dict[str, Draft202012Validator] = {}
        self._load_schemas()

    def _load_schemas(self) -> None:
        """Load all JSON schemas from contracts directory."""
        if not self.contracts_dir.exists():
            raise ValueError(f"Contracts directory does not exist: {self.contracts_dir}")

        for schema_file in self.contracts_dir.glob("*.schema.json"):
            schema_name = schema_file.stem  # e.g., "event.schema"
            with open(schema_file) as f:
                schema = json.load(f)
                self._schemas[schema_name] = schema
                self._validators[schema_name] = Draft202012Validator(schema)

    def validate(self, message: Dict[str, Any], schema_name: str) -> None:
        """
        Validate message against specified schema.

        Args:
            message: Message to validate
            schema_name: Schema name (e.g., "event.schema", "incident.schema")

        Raises:
            ValueError: If schema not found
            ValidationError: If message doesn't match schema
        """
        if schema_name not in self._validators:
            raise ValueError(f"Unknown schema: {schema_name}")

        validator = self._validators[schema_name]
        validator.validate(message)  # Raises ValidationError if invalid

    def get_schema(self, schema_name: str) -> Dict[str, Any]:
        """
        Get loaded schema by name.

        Args:
            schema_name: Schema name

        Returns:
            Schema dictionary

        Raises:
            ValueError: If schema not found
        """
        if schema_name not in self._schemas:
            raise ValueError(f"Unknown schema: {schema_name}")
        return self._schemas[schema_name]
