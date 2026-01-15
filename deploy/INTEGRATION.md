# ORION Integration with Infrastructure

## Overview

ORION integrates with infrastructure in **observe-only** mode.

**Key Principle**: ORION watches but never controls.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    ORION Observe Loop                    │
│                                                           │
│  Prometheus ──► ORION Watcher ──► Event Bus ──► Guardian│
│  (metrics)      (observe)         (Redis)      (correlate)
│                                                     │     │
│                                                     ▼     │
│                                   Brain ──► Memory       │
│                                   (decide)  (audit)      │
│                                     │                     │
│                                     ▼                     │
│                                 NO_ACTION                 │
└─────────────────────────────────────────────────────────┘

Media Stack (Jellyfin, *arr, qBittorrent)
  │
  ├─► cAdvisor ──► Prometheus
  └─► Node Exporter ──► Prometheus

NO control path from ORION → Media Stack
```

## Integration Points

### 1. Metrics Observation

**Source**: Prometheus
**Consumer**: ORION Watcher (future implementation)
**Protocol**: HTTP (Prometheus API)
**Frequency**: Polling (every 60s)

**Metrics of Interest**:
- Container health (cAdvisor)
- System resources (node_exporter)
- Disk usage
- Memory pressure
- Network errors

**Example Query**:
```python
import requests

# Query Prometheus
response = requests.get(
    "http://localhost:9090/api/v1/query",
    params={"query": "up{job='cadvisor'}"}
)

# Emit event if service down
if response.json()["data"]["result"][0]["value"][1] == "0":
    bus.publish({
        "event_type": "service_down",
        "severity": "error",
        "data": {"service": "cadvisor"}
    }, "event")
```

### 2. Log Observation (Future)

**Source**: Docker container logs
**Consumer**: ORION Log Watcher (not implemented in Phase 2)
**Protocol**: Docker API or file tailing

**Not Implemented**: Phase 2 focuses on metrics only.

### 3. Event Emission

**ORION Behavior**:

1. **Observe**: Poll Prometheus metrics
2. **Detect**: Threshold crossing, service down, errors
3. **Emit**: Event to Redis Streams
4. **Correlate**: Guardian creates incident
5. **Decide**: Brain decides NO_ACTION
6. **Store**: Memory persists audit trail

**No Actions**: ORION never:
- Restarts services
- Kills containers
- Modifies configuration
- Sends alerts (beyond logging)

## Data Flow

### Normal Operation

```
1. Container running
   ↓
2. cAdvisor exports metrics
   ↓
3. Prometheus scrapes metrics
   ↓
4. ORION queries Prometheus (60s poll)
   ↓
5. ORION: "All normal" (no event emitted)
```

### Anomaly Detection

```
1. Container crashes
   ↓
2. cAdvisor reports container_last_seen decreasing
   ↓
3. Prometheus records metric
   ↓
4. ORION queries Prometheus
   ↓
5. ORION detects: container down
   ↓
6. ORION emits: event (service_down, severity=error)
   ↓
7. Guardian receives event
   ↓
8. Guardian correlates: creates incident
   ↓
9. Brain receives incident
   ↓
10. Brain decides: NO_ACTION
   ↓
11. Memory stores: event + incident + decision
   ↓
12. Human reviews audit trail (Grafana or CLI)
```

## Configuration

### Prometheus (Observe-Only)

**No alert rules**:
```yaml
# ❌ NOT in prometheus.yml
# rule_files:
#   - /etc/prometheus/alerts/*.yml
```

**No alertmanager**:
```yaml
# ❌ NOT in prometheus.yml
# alerting:
#   alertmanagers:
#     - static_configs:
#       - targets: ['alertmanager:9093']
```

**Metrics collection only**:
```yaml
# ✓ prometheus.yml
scrape_configs:
  - job_name: 'cadvisor'
    static_configs:
      - targets: ['cadvisor:8080']
```

### Grafana (Dashboards Only)

**No alert notifications**:
- Dashboards display metrics
- No email/Slack/PagerDuty
- Human observes, human acts

**Dashboard Examples**:
1. **ORION Health**: Event rate, incident count, decision types
2. **Media Stack**: Container status, resource usage
3. **System**: CPU, memory, disk, network

### ORION Configuration

**Environment Variables**:
```bash
# Autonomy level (MUST be N0)
ORION_AUTONOMY_LEVEL=N0

# Prometheus endpoint
PROMETHEUS_URL=http://localhost:9090

# Poll interval (seconds)
ORION_POLL_INTERVAL=60

# Redis (event bus)
REDIS_HOST=localhost
REDIS_PORT=6379
```

## Failure Modes

### Prometheus Down

**Behavior**:
- ORION cannot query metrics
- ORION emits: event (external_api_failure)
- Guardian correlates: prometheus_outage incident
- Brain decides: NO_ACTION
- **Result**: ORION observes its own failure, cannot fix

**Recovery**: Manual restart of Prometheus.

### Media Stack Down

**Behavior**:
- Prometheus reports container down
- ORION detects via metrics
- ORION emits: event (service_down)
- Guardian correlates: service_outage incident
- Brain decides: NO_ACTION
- **Result**: ORION observes, logs, does nothing

**Recovery**: Docker restarts container (unless-stopped policy).

### ORION Down

**Behavior**:
- Media stack continues unaffected
- Monitoring continues (Prometheus/Grafana)
- **Result**: Infrastructure is independent of ORION

**Recovery**: Restart ORION services manually.

## Non-Integration (Explicit)

### What ORION Does NOT Do

1. **No Docker Control**:
   - Cannot `docker restart`
   - Cannot `docker stop`
   - Cannot `docker rm`

2. **No File System Control**:
   - Cannot delete downloads
   - Cannot clean logs
   - Cannot modify configs

3. **No Network Control**:
   - Cannot block ports
   - Cannot restart networking
   - Cannot modify firewall

4. **No Notification Control**:
   - Cannot send emails
   - Cannot send Slack messages
   - Cannot trigger webhooks

### Media Stack Opacity

**Media stack is unaware of ORION**:
- No ORION agents in containers
- No ORION environment variables
- No ORION volume mounts
- No ORION network dependencies

**ORION is unaware of media internals**:
- No access to Radarr/Sonarr databases
- No access to qBittorrent state
- No access to Jellyfin sessions
- Only observes via Prometheus metrics

## Testing Integration

### Verify Observe-Only

```bash
# 1. Start infrastructure
docker-compose -f deploy/lab/docker-compose.monitoring.yml up -d

# 2. Verify Prometheus accessible
curl http://localhost:9090/api/v1/targets

# 3. Verify no alertmanager
curl http://localhost:9090/api/v1/alertmanagers
# Should return empty

# 4. Verify Grafana accessible
curl http://localhost:3000/api/health

# 5. Start ORION (manual)
python3 -m watchers.system_resources

# 6. Verify events emitted
# (Check Redis or memory logs)

# 7. Verify no actions taken
# (Check docker ps - containers unchanged)
```

### Verify N0 Enforcement

```bash
# Check autonomy level
grep AUTONOMY_LEVEL .env
# Should be: N0

# Check brain decisions
cat /mnt/orion-data/orion/memory/decisions.jsonl | jq '.decision_type'
# All should be: "NO_ACTION"

# Verify no action contracts emitted
cat /mnt/orion-data/orion/memory/actions.jsonl
# File should not exist or be empty
```

## Future Phases

### Phase 3+ May Add:

- Log tailing (observe container logs)
- Docker events (observe container lifecycle)
- Network metrics (observe bandwidth)
- **Still N0**: No actions, only richer observation

### NOT in Scope (Ever):

- Autonomous restarts (requires N2+)
- Autonomous cleanup (requires N2+)
- Autonomous scaling (requires N2+)

---

**Remember**: ORION observes. Humans act. Always.
