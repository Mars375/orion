# Technology Stack

**Analysis Date:** 2026-01-13

## Languages

**Primary:**
- Python - AI reasoning, SRE logic, decision systems (planned for brain, guardian, memory, api, commander, approval-telegram)
- Go - Long-running services, reliability, performance (planned for bus, edge-agent, system agents)

**Secondary:**
- JSON Schema - Language-agnostic contract definitions in `bus/contracts/`

## Runtime

**Environment:**
- Python (version not specified) - For cognitive modules
- Go (version not specified) - For reliability-critical services
- Raspberry Pi 4 Model B - Edge devices (`edge/freenove_hexapod/`)

**Package Manager:**
- Not detected (no package.json, requirements.txt, go.mod, or Cargo.toml found)
- Project is in early architectural phase without implementation

## Frameworks

**Core:**
- Not yet implemented - No framework dependencies detected

**Testing:**
- Test-alongside doctrine specified in `CLAUDE.md`
- Framework TBD - Must support Python and Go with contract validation

**Build/Dev:**
- Docker (planned) - Empty docker-compose.yml files in `deploy/core/` and `deploy/hub/`
- Git - Conventional commits with module-scoped branches

## Key Dependencies

**Critical (Planned but not installed):**
- Redis Streams - Core event bus for inter-module communication
- MQTT broker - Edge telemetry and commands for robot devices
- Telegram Bot API - Human approval workflow for risky actions
- JSON Schema validator - Contract validation at module boundaries

**Infrastructure (Planned):**
- Tailscale - Zero-trust networking, mandatory for all nodes
- Prometheus - Metrics collection
- Loki - Log aggregation
- Supabase or similar - Database (implied but not specified)

## Configuration

**Environment:**
- `.env` files (gitignored) - Secrets management per `CLAUDE.md`
- `.env.example` templates present but empty in `deploy/core/` and `deploy/hub/`
- Secrets never committed, backed up encrypted per `docs/SECURITY.md`

**Build:**
- Not yet configured
- Docker Compose expected in `deploy/` directories (currently empty)

## Platform Requirements

**Development:**
- Linux/macOS/Windows (WSL2 detected from system path)
- Docker for containerized services
- Git with conventional commit enforcement

**Production:**
- Docker containers - orion-core and orion-hub nodes
- Raspberry Pi 4 Model B - Edge devices (`edge/freenove_hexapod/`)
- Tailscale VPN - Mandatory for node communication (`docs/SECURITY.md`)
- Hardware: Freenove Hexapod Robot on Raspberry Pi

---

*Stack analysis: 2026-01-13*
*Update after major dependency changes*
