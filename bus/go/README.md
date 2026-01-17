# orion-bus (Go)

**Module**: orion-bus (Go)
**Phase**: 4.1 (In Progress)
**Language**: Go 1.22+

## Purpose

High-performance Redis Streams event bus with contract validation, migrated from Python to support Phase 5 AI Council concurrent LLM inference requirements.

## Inputs

- Messages from publishers (Python and Go modules)
- JSON Schema contracts from `bus/contracts/`

## Outputs

- Validated messages published to Redis Streams
- Consumer group message delivery
- Validation errors for invalid messages

## Invariants

- **Contract validation is mandatory at all boundaries**
- Fail-fast on invalid messages
- Messages must conform to JSON Schema Draft 2020-12
- Consumer groups guarantee at-least-once delivery
- Graceful shutdown ensures no message loss

## Failure Modes

1. **Redis connection loss**: Automatic reconnection with exponential backoff
2. **Invalid message schema**: Message rejected, error logged, no retry
3. **Consumer processing failure**: Message redelivered to consumer group
4. **Graceful shutdown timeout**: Force shutdown after timeout, log unprocessed messages

## Implementation Details

**Dependencies:**
- `github.com/redis/go-redis/v9` - Official Redis client with Streams support
- `github.com/santhosh-tekuri/jsonschema/v6` - JSON Schema Draft 2020-12 validation
- `github.com/google/uuid` - UUID generation for message IDs
- `github.com/stretchr/testify` - Testing assertions

**Architecture:**
- Contract-first design with runtime validation
- Worker pool pattern for bounded concurrency
- Context-based cancellation for graceful shutdown

## What This Module Does NOT Do

- Does NOT modify message content (pass-through only)
- Does NOT persist messages (Redis handles persistence)
- Does NOT implement business logic (pure transport layer)
- Does NOT automatically upgrade schema versions
- Does NOT support schema-less messages
