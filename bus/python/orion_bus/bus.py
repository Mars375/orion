"""
Redis Streams-based event bus implementation.

Enforces contract validation and fail-fast semantics.
"""

import json
import logging
from typing import Dict, Any, List, Optional, Callable
from pathlib import Path

import redis
from jsonschema import ValidationError

from .validator import ContractValidator


logger = logging.getLogger(__name__)


class EventBus:
    """
    Redis Streams-based event bus with contract validation.

    Invariants:
    - All messages MUST validate against contracts before publish
    - Invalid messages are rejected (fail fast)
    - No retries that could amplify events
    - Bounded memory usage (Redis Streams with maxlen)
    """

    def __init__(
        self,
        redis_client: redis.Redis,
        contracts_dir: Path,
        stream_prefix: str = "orion",
        max_stream_length: int = 10000,
    ):
        """
        Initialize event bus.

        Args:
            redis_client: Redis client instance
            contracts_dir: Path to contracts directory
            stream_prefix: Prefix for Redis stream names
            max_stream_length: Maximum entries per stream (for bounded memory)
        """
        self.redis = redis_client
        self.validator = ContractValidator(contracts_dir)
        self.stream_prefix = stream_prefix
        self.max_stream_length = max_stream_length

    def _get_stream_name(self, contract_type: str) -> str:
        """
        Get Redis stream name for contract type.

        Args:
            contract_type: Contract type (event, incident, decision, etc.)

        Returns:
            Stream name (e.g., "orion:events")
        """
        return f"{self.stream_prefix}:{contract_type}s"

    def publish(self, message: Dict[str, Any], contract_type: str) -> str:
        """
        Publish message to event bus after validation.

        Args:
            message: Message to publish
            contract_type: Contract type (event, incident, decision, etc.)

        Returns:
            Message ID from Redis

        Raises:
            ValidationError: If message doesn't match contract
            ValueError: If contract type unknown
            redis.RedisError: If Redis operation fails
        """
        # Validate against contract (fail fast)
        schema_name = f"{contract_type}.schema"
        self.validator.validate(message, schema_name)

        # Publish to Redis Stream
        stream_name = self._get_stream_name(contract_type)
        message_id = self.redis.xadd(
            stream_name,
            {"data": json.dumps(message)},
            maxlen=self.max_stream_length,
            approximate=True,  # Allow approximate trimming for performance
        )

        logger.debug(
            f"Published {contract_type} to {stream_name}: {message_id}",
            extra={"contract_type": contract_type, "message_id": message_id},
        )

        return message_id.decode("utf-8") if isinstance(message_id, bytes) else message_id

    def subscribe(
        self,
        contract_type: str,
        handler: Callable[[Dict[str, Any]], None],
        consumer_group: str,
        consumer_name: str,
        block_ms: int = 1000,
        count: int = 10,
    ) -> None:
        """
        Subscribe to contract type and process messages.

        Args:
            contract_type: Contract type to subscribe to
            handler: Callback function to handle each message
            consumer_group: Redis consumer group name
            consumer_name: Consumer name within group
            block_ms: Block timeout in milliseconds
            count: Maximum messages to read per call

        Note:
            This is a blocking call that processes messages in a loop.
            Consumer must handle exceptions within handler.
        """
        stream_name = self._get_stream_name(contract_type)

        # Create consumer group if it doesn't exist
        try:
            self.redis.xgroup_create(stream_name, consumer_group, id="0", mkstream=True)
            logger.info(f"Created consumer group {consumer_group} for {stream_name}")
        except redis.ResponseError as e:
            if "BUSYGROUP" not in str(e):
                raise
            logger.debug(f"Consumer group {consumer_group} already exists for {stream_name}")

        logger.info(
            f"Starting subscription: {stream_name} (group={consumer_group}, consumer={consumer_name})"
        )

        while True:
            try:
                # Read from stream as consumer group
                messages = self.redis.xreadgroup(
                    consumer_group,
                    consumer_name,
                    {stream_name: ">"},
                    count=count,
                    block=block_ms,
                )

                if not messages:
                    continue

                for stream, entries in messages:
                    for message_id, fields in entries:
                        try:
                            # Parse message
                            data = json.loads(fields[b"data"])

                            # Handle message
                            handler(data)

                            # Acknowledge message
                            self.redis.xack(stream_name, consumer_group, message_id)

                        except Exception as e:
                            logger.error(
                                f"Error processing message {message_id}: {e}",
                                exc_info=True,
                                extra={"message_id": message_id, "stream": stream_name},
                            )
                            # Do NOT retry - fail closed, not open
                            # Acknowledge to prevent blocking the consumer group
                            self.redis.xack(stream_name, consumer_group, message_id)

            except KeyboardInterrupt:
                logger.info(f"Stopping subscription to {stream_name}")
                break
            except Exception as e:
                logger.error(f"Error in subscription loop: {e}", exc_info=True)
                # Continue processing - transient errors shouldn't stop the bus

    def read_stream(
        self,
        contract_type: str,
        start_id: str = "0",
        count: int = 100,
    ) -> List[Dict[str, Any]]:
        """
        Read messages from stream (for testing and inspection).

        Args:
            contract_type: Contract type to read
            start_id: Start reading from this ID
            count: Maximum messages to read

        Returns:
            List of messages
        """
        stream_name = self._get_stream_name(contract_type)
        entries = self.redis.xrange(stream_name, min=start_id, max="+", count=count)

        messages = []
        for message_id, fields in entries:
            data = json.loads(fields[b"data"])
            messages.append(data)

        return messages
