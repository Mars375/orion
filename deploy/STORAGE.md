# ORION Storage Layout

## Critical Rules

1. **HDD ONLY**: All persistent data MUST live on HDD (`/mnt/orion-data`)
2. **No SD Writes**: SD cards are read-only for services (OS writes only)
3. **tmpfs for transients**: Use tmpfs for temporary/transient logs
4. **Fail Closed**: If HDD unavailable, services MUST NOT start

## Directory Structure

```
/mnt/orion-data/
├── media/                  # Media library (Jellyfin)
│   ├── movies/            # Movie files
│   ├── tv/                # TV show files
│   └── music/             # Music files
│
├── downloads/              # Download staging
│   ├── complete/          # Completed downloads
│   └── incomplete/        # In-progress downloads
│
├── config/                 # Service configurations
│   ├── jellyfin/          # Jellyfin config & database
│   ├── radarr/            # Radarr config & database
│   ├── sonarr/            # Sonarr config & database
│   ├── prowlarr/          # Prowlarr config & database
│   ├── qbittorrent/       # qBittorrent config
│   ├── prometheus/        # Prometheus config & TSDB
│   └── grafana/           # Grafana config & dashboards
│
├── logs/                   # Centralized persistent logs
│   ├── media/             # Media stack logs
│   │   ├── jellyfin/
│   │   ├── radarr/
│   │   ├── sonarr/
│   │   ├── prowlarr/
│   │   └── qbittorrent/
│   ├── monitoring/        # Monitoring stack logs
│   │   ├── prometheus/
│   │   └── grafana/
│   └── orion/             # ORION service logs
│       ├── guardian/
│       ├── brain/
│       ├── memory/
│       └── watcher/
│
└── orion/                  # ORION persistent state
    ├── memory/            # Audit trail (JSONL files)
    │   ├── events.jsonl
    │   ├── incidents.jsonl
    │   └── decisions.jsonl
    ├── redis/             # Redis persistence (RDB/AOF)
    └── backups/           # Configuration backups
        └── YYYY-MM-DD/
```

## Permissions

### Owner/Group

All directories owned by `orion:orion` (UID:GID = 1000:1000)

```bash
sudo chown -R 1000:1000 /mnt/orion-data
```

### Directory Permissions

| Path | Mode | Reason |
|------|------|--------|
| `/mnt/orion-data` | `755` | Root accessible |
| `media/*` | `755` | Readable by media services |
| `downloads/*` | `755` | Writable by qBittorrent, readable by *arr |
| `config/*` | `700` | Service-specific (no cross-access) |
| `logs/*` | `755` | Readable for debugging |
| `orion/memory` | `700` | ORION-only access |
| `orion/redis` | `700` | Redis-only access |

### Setup Commands

```bash
# Create structure
sudo mkdir -p /mnt/orion-data/{media/{movies,tv,music},downloads/{complete,incomplete}}
sudo mkdir -p /mnt/orion-data/config/{jellyfin,radarr,sonarr,prowlarr,qbittorrent,prometheus,grafana}
sudo mkdir -p /mnt/orion-data/logs/{media/{jellyfin,radarr,sonarr,prowlarr,qbittorrent},monitoring/{prometheus,grafana},orion/{guardian,brain,memory,watcher}}
sudo mkdir -p /mnt/orion-data/orion/{memory,redis,backups}

# Set ownership
sudo chown -R 1000:1000 /mnt/orion-data

# Set permissions
chmod 755 /mnt/orion-data
chmod -R 755 /mnt/orion-data/media
chmod -R 755 /mnt/orion-data/downloads
chmod -R 700 /mnt/orion-data/config
chmod -R 755 /mnt/orion-data/logs
chmod -R 700 /mnt/orion-data/orion
```

## Volume Mounts (Docker)

All Docker volumes MUST map to HDD paths.

### Media Stack

```yaml
volumes:
  - /mnt/orion-data/config/jellyfin:/config
  - /mnt/orion-data/media:/media:ro
  - /mnt/orion-data/logs/media/jellyfin:/logs
```

**Never**:
```yaml
volumes:
  - ./config:/config  # ❌ SD card!
  - /var/lib/jellyfin:/config  # ❌ SD card!
```

### ORION Services

```yaml
volumes:
  - /mnt/orion-data/orion/memory:/data/memory
  - /mnt/orion-data/orion/redis:/data/redis
  - /mnt/orion-data/logs/orion:/logs
```

## Transient Storage (tmpfs)

For truly transient data (not audit trail):

```yaml
tmpfs:
  - /tmp:size=1G,mode=1777
  - /var/tmp:size=512M,mode=1777
```

**Use Cases**:
- Temporary file processing
- Cache that can be lost
- PIDs, sockets

**Never**:
- Audit logs
- Configuration
- Metrics history

## Backup Strategy

### What to Backup

1. **Critical** (daily):
   - `/mnt/orion-data/config/*`
   - `/mnt/orion-data/orion/memory/*`

2. **Important** (weekly):
   - `/mnt/orion-data/orion/redis/*`
   - Prometheus data (if long-term retention needed)

3. **Optional**:
   - Media files (can be re-downloaded)
   - Logs (can be truncated)

### Backup Location

**Not on HDD**: Backups should go to separate device or NFS.

Example:
```bash
# Backup to NAS
rsync -av /mnt/orion-data/config/ nas:/backups/orion/config/
rsync -av /mnt/orion-data/orion/memory/ nas:/backups/orion/memory/
```

## Monitoring Disk Usage

### Prometheus Metrics

Monitor HDD usage via node_exporter:

```promql
# Disk usage
node_filesystem_avail_bytes{mountpoint="/mnt/orion-data"}

# Disk full alert (observe only, no action)
(node_filesystem_avail_bytes{mountpoint="/mnt/orion-data"}
 / node_filesystem_size_bytes{mountpoint="/mnt/orion-data"}) < 0.1
```

### Manual Checks

```bash
# Disk usage summary
df -h /mnt/orion-data

# Directory sizes
du -sh /mnt/orion-data/*

# Largest directories
du -h /mnt/orion-data | sort -rh | head -20
```

## Disk Full Handling

### ORION Behavior (N0 Mode)

If disk full:
1. **Observe**: ORION detects via metrics
2. **Emit Event**: disk_full event
3. **Correlate**: Guardian creates incident
4. **Decide**: Brain decides NO_ACTION
5. **Store**: Memory writes to... wait, disk is full!

### Failure Mode

**Disk full = graceful degradation**:
- Media services continue (read-only)
- Downloads stop (qBittorrent pauses)
- ORION memory writes fail (logged, not retried)
- Metrics collection may degrade

### Recovery (Manual)

1. Free space (delete old downloads, compress logs)
2. Restart affected services
3. ORION resumes automatically

**No auto-cleanup**: Human decides what to delete.

## Verification Checklist

Before deploying:

- [ ] HDD mounted at `/mnt/orion-data`
- [ ] Directory structure created
- [ ] Permissions set (orion:orion, 700/755)
- [ ] No Docker volumes pointing to SD card
- [ ] Backup strategy defined
- [ ] Monitoring configured

---

**Remember**: If it persists, it lives on HDD. No exceptions.
