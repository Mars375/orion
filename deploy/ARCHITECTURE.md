# ORION Deployment Architecture

## Overview

ORION is deployed across multiple Raspberry Pi 5 nodes with centralized storage on HDD.

**Key Principle**: SD cards are for OS and code only. ALL persistent data lives on HDD.

## Hardware Topology

### Nodes

| Node | Hardware | Role | Responsibilities |
|------|----------|------|------------------|
| `orion-hub` | Pi 5 (16GB RAM) | Media & Storage Hub | Jellyfin, *arr stack, qBittorrent, shared storage |
| `orion-core` | Pi 5 (8GB RAM) | ORION Control Plane | Guardian, Brain, Memory, Monitoring |

### Storage

| Device | Mount | Purpose | Writes |
|--------|-------|---------|--------|
| SD Card (hub) | `/` | OS, code, containers | OS only |
| SD Card (core) | `/` | OS, code, containers | OS only |
| **HDD 3TB** | `/mnt/orion-data` | ALL persistent data | Everything else |

**Critical**: No service may write to SD cards except the OS itself.

## Storage Layout (HDD)

All paths are relative to HDD mount point `/mnt/orion-data`:

```
/mnt/orion-data/
├── media/              # Media library (Jellyfin source)
│   ├── movies/
│   ├── tv/
│   └── music/
├── downloads/          # Download staging area
│   ├── complete/
│   └── incomplete/
├── config/             # Service configurations
│   ├── jellyfin/
│   ├── radarr/
│   ├── sonarr/
│   ├── prowlarr/
│   ├── qbittorrent/
│   ├── prometheus/
│   └── grafana/
├── logs/               # Centralized logs
│   ├── media/
│   ├── monitoring/
│   └── orion/
└── orion/              # ORION persistent state
    ├── memory/         # Audit trail (JSONL)
    ├── redis/          # Event bus persistence
    └── backups/        # Configuration backups
```

### Permissions

- **Owner**: `orion:orion` (UID/GID: 1000:1000)
- **Media dirs**: `755` (readable by media services)
- **Config dirs**: `700` (service-specific access)
- **Logs**: `755` (readable for debugging)
- **ORION data**: `700` (ORION-only access)

## Service Deployment

### Media Stack (orion-hub)

Deployed via `deploy/hub/docker-compose.yml`:

- **Jellyfin**: Media server (port 8096)
- **Radarr**: Movie management (port 7878)
- **Sonarr**: TV management (port 8989)
- **Prowlarr**: Indexer manager (port 9696)
- **qBittorrent**: Torrent client (port 8080)

**Volumes**: All volumes MUST map to `/mnt/orion-data` (HDD).

**No ORION Integration**: Media stack is opaque to ORION. ORION observes metrics only.

### Monitoring Stack (orion-core)

Deployed via `deploy/lab/docker-compose.monitoring.yml`:

- **Prometheus**: Metrics collection (port 9090)
- **Grafana**: Dashboards (port 3000)
- **Node Exporter**: System metrics (port 9100)
- **cAdvisor**: Container metrics (port 8080)

**Configuration**:
- Prometheus scrapes metrics only (no alertmanager)
- Grafana displays dashboards (no alert execution)
- ORION subscribes to Prometheus metrics via API

### ORION Core Services (orion-core)

Deployed via `deploy/lab/docker-compose.orion.yml`:

- **Redis**: Event bus (port 6379)
- **orion-watcher**: System resource observer
- **orion-guardian**: Event correlation
- **orion-brain**: Decision logic (N0 only)
- **orion-memory**: Audit trail writer

**Autonomy Level**: N0 (observe only)

**No Actions**: No execution paths wired.

## Network Architecture

### Ports (External Access)

| Service | Port | Access |
|---------|------|--------|
| Jellyfin | 8096 | LAN |
| Grafana | 3000 | LAN |
| Prometheus | 9090 | localhost only |
| Redis | 6379 | ORION services only |

### Service Discovery

Services discover each other via Docker network or explicit hostnames.

**No dynamic discovery**: Explicit configuration required.

## Observability

### Metrics Flow

```
System/Containers → Prometheus → Grafana (human)
                              ↓
                           ORION (observe)
```

### Logs Flow

```
Services → HDD logs → ORION memory (audit trail)
```

### ORION Integration

ORION observes but NEVER acts:

1. **Prometheus Metrics**: ORION reads metrics via Prometheus API
2. **Event Emission**: ORION emits observations as events
3. **Incident Detection**: Guardian correlates events
4. **Decision**: Brain decides NO_ACTION (N0 mode)
5. **Audit**: Memory stores all events/incidents/decisions

**No Control Paths**: ORION cannot restart, stop, or modify services.

## Deployment Process (MANUAL)

Phase 2 provides **documentation and configurations**, not auto-deploy.

### Prerequisites

1. HDD mounted at `/mnt/orion-data` on both nodes
2. Directory structure created with correct permissions
3. Docker and docker-compose installed

### Deployment Steps

1. **Prepare Storage**:
   ```bash
   sudo mkdir -p /mnt/orion-data/{media,downloads,config,logs,orion}
   sudo chown -R orion:orion /mnt/orion-data
   ```

2. **Deploy Media Stack** (on orion-hub):
   ```bash
   cd deploy/hub
   docker-compose up -d
   ```

3. **Deploy Monitoring** (on orion-core):
   ```bash
   cd deploy/lab
   docker-compose -f docker-compose.monitoring.yml up -d
   ```

4. **Deploy ORION** (on orion-core):
   ```bash
   cd deploy/lab
   docker-compose -f docker-compose.orion.yml up -d
   ```

5. **Verify**:
   - Jellyfin: http://orion-hub:8096
   - Grafana: http://orion-core:3000
   - Check logs: `docker-compose logs`

## Safety Invariants

1. **No SD Writes**: All persistent data on HDD
2. **N0 Autonomy**: ORION observes, never acts
3. **Media Opacity**: Media stack unaware of ORION
4. **Fail Safe**: Missing HDD = services stop (not fail open)
5. **No Auto-Actions**: No restart/kill/cleanup logic

## Failure Modes

| Failure | Behavior | Recovery |
|---------|----------|----------|
| HDD unavailable | Containers fail to start | Mount HDD, restart containers |
| SD card full | OS may fail | Clear OS logs, NOT service data |
| Service crash | Container restarts (Docker policy) | Manual investigation |
| ORION failure | Media stack unaffected | Debug ORION, media continues |

**Critical**: Media services MUST remain functional even if ORION fails.

## Phase 2 Constraints

- **No auto-remediation**: ORION observes only
- **No alertmanager**: Metrics collection, no execution
- **No approvals**: N0 mode requires no approvals
- **No policy changes**: SAFE/RISKY unchanged from Phase 1

---

**Status**: Phase 2 - Documentation and passive infrastructure only.
