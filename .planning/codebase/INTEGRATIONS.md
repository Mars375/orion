# External Integrations

**Analysis Date:** 2026-01-13

## APIs & External Services

**Telegram Bot API:**
- Purpose: Human-in-the-loop approvals for risky autonomous actions
- Module: `orion-approval-telegram` (Python, not yet implemented)
- Auth: Bot token (stored in environment variables)
- Mandatory: Required for N3 autonomy level per `docs/SECURITY.md`
- Time-limited: Approvals expire, full audit trail required

**No other external APIs detected**

## Data Storage

**Redis Streams:**
- Purpose: Core event bus for inter-module communication
- Connection: Via environment variables (not yet configured)
- Client: `orion-bus` (Go module, not yet implemented)
- Event types: Observation, Incident, Decision, Action, Outcome per `docs/BUS_AND_CONTRACTS.md`
- Rules: Events immutable, schemas versioned, decisions auditable

**Database:**
- Type: Not specified
- Purpose: State storage, memory, post-mortems (implied)
- Module: `orion-memory` (Python, not yet implemented)

**File Storage:**
- Not detected

## Authentication & Identity

**Tailscale:**
- Purpose: Zero-trust networking for all ORION nodes per `docs/SECURITY.md`
- Implementation: Mandatory, no open inbound ports
- Policy: Explicit ACLs, identity per node
- Configuration: Not yet defined in deployment files

**Auth Provider:**
- Not applicable - System-to-system authentication only via Tailscale

## Monitoring & Observability

**Prometheus:**
- Purpose: Metrics collection per `docs/ARCHITECTURE.md`
- Integration: Planned but not configured
- Retention: Not specified

**Loki:**
- Purpose: Log aggregation per `docs/ARCHITECTURE.md`
- Integration: Planned but not configured
- Retention: Not specified

**Error Tracking:**
- Not configured
- Audit trail: Mandatory for all decisions and actions per `CLAUDE.md`

**Analytics:**
- Not applicable (internal SRE system)

## CI/CD & Deployment

**Hosting:**
- Platform: Docker containers on physical hardware (homelab)
- Nodes: orion-core, orion-hub, orion-edge
- Deployment: `deploy/core/docker-compose.yml`, `deploy/hub/docker-compose.yml` (empty)
- Environment vars: Not yet configured

**CI Pipeline:**
- Not configured
- Expected: Git hooks for conventional commits, branch naming enforcement per `CLAUDE.md`
- Testing: Test-alongside doctrine, contract validation required

## Edge & IoT Integration

**MQTT:**
- Purpose: Edge telemetry and commands per `docs/ARCHITECTURE.md`
- Client: `orion-edge-agent` (Go, not yet implemented)
- Topics: Not specified
- QoS: Not specified
- Offline behavior: Edge nodes must operate autonomously per `edge/freenove_hexapod/safety.md`

**Robot Hardware:**
- Device: Freenove Hexapod on Raspberry Pi 4 Model B
- Location: `edge/freenove_hexapod/`
- Safety: Default to stop, loss of network → safe mode, no destructive commands without approval

## Environment Configuration

**Development:**
- Required env vars: Not documented (`.env.example` files empty)
- Secrets location: `.env` files (gitignored per `.gitignore`)
- Mock/stub services: Redis, MQTT, Telegram per `CLAUDE.md` testing doctrine

**Staging:**
- Not defined

**Production:**
- Secrets management: Environment files, backed up encrypted per `docs/SECURITY.md`
- Failover/redundancy: Not specified
- Edge resilience: Devices must survive core node failure

## Webhooks & Callbacks

**Incoming:**
- Telegram webhook: For approval responses (planned but not implemented)
- MQTT messages: From edge devices

**Outgoing:**
- Telegram notifications: For risky action approval requests
- MQTT commands: To edge devices

## Communication Protocols

**No cross-language imports:**
- All modules communicate via Redis Streams events
- Contracts defined in `bus/contracts/*.json` (empty files)
- No shared memory, shared volumes, or implicit state
- HTTP only for explicit human-facing or inspection APIs (never internal control flow)

---

*Integration audit: 2026-01-13*
*Update when adding/removing external services*
