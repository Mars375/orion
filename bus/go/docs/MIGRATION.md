# ORION Bus Migration Guide: Python → Go

## Overview

This document describes the three-phase migration strategy for transitioning from the Python event bus implementation to the Go implementation. The migration prioritizes **zero downtime**, **no message loss**, and **safe rollback** at every stage.

**Migration Timeline**: 3 weeks (1 week per phase)

**Key Invariants**:
- Both implementations use Redis Streams with consumer groups (architecture-compatible)
- Contract validation enforced at all times (JSON Schema Draft 2020-12)
- Consumer groups are independent (no coordination required for rollback)
- Messages are never lost (Redis Streams persistence)

---

## Phase 1: Parallel Operation (Week 1)

### Goal
Deploy Go bus alongside Python bus for observation and validation without affecting production traffic.

### Actions

1. **Deploy Go bus**:
   ```bash
   cd /home/orion/orion/bus/go
   make build
   ./bin/orion-bus \
     --redis-addr localhost:6379 \
     --contracts-dir ../../../contracts \
     --stream-prefix orion \
     --max-stream-len 10000 \
     --http-port 8080
   ```

2. **Configure systemd service** (optional, for production):
   - Create `/etc/systemd/system/orion-bus-go.service`
   - Enable and start service
   - Verify health endpoint: `curl http://localhost:8080/health`

3. **Observation mode**:
   - Go bus runs but does NOT consume messages yet (no subscribers)
   - Python bus continues normal operation (all publishers + consumers)
   - Both buses coexist using same Redis Streams infrastructure

4. **Daily validation**:
   ```bash
   cd /home/orion/orion/bus/go
   ./scripts/migration-validate.sh
   ```
   - Run validation script daily
   - Check for contract violations (should be zero)
   - Monitor Go bus memory usage (target: <100MB)
   - Verify health endpoint availability

### Success Criteria

- ✅ Go bus runs for 7 days without crashes
- ✅ Health endpoint responds 100% of the time
- ✅ Memory usage <100MB (Python bus: ~150MB)
- ✅ Zero contract validation errors
- ✅ migration-validate.sh passes all tests daily

### Rollback

If Go bus shows instability:
1. Stop Go bus service: `systemctl stop orion-bus-go`
2. Python bus continues unaffected (no changes made)
3. Investigate logs: `/var/log/orion-bus-go.log`

**No rollback complexity** - Go bus is observe-only in this phase.

---

## Phase 2: Consumer Migration (Week 2)

### Goal
Migrate consumers from Python bus to Go bus one module at a time, with 24-hour monitoring between migrations.

### Migration Order (Lowest Risk → Highest Risk)

1. **orion-memory** (Day 1-2)
   - Risk: Low (read-only, post-mortem storage)
   - Impact: No immediate effect on decisions

2. **orion-guardian** (Day 3-4)
   - Risk: Medium (correlation logic, temporal windows)
   - Impact: Affects incident detection

3. **orion-brain** (Day 5-6)
   - Risk: High (decision logic, core reasoning)
   - Impact: Affects all autonomous actions

4. **orion-commander** (Day 7)
   - Risk: High (action execution)
   - Impact: Direct system changes

5. **orion-approval** (Day 8)
   - Risk: Medium (approval coordination)
   - Impact: Human-in-the-loop timing

### Migration Procedure (Per Module)

For each module, follow these steps:

#### Step 1: Update Consumer Configuration

**Python bus consumer (old)**:
```python
bus.subscribe("event", "orion-memory-group", handler)
```

**Go bus consumer (new)**:
```python
# Update to subscribe from Go bus endpoint or update consumer group name
bus.subscribe("event", "orion-memory-group-go", handler)
```

**Note**: Consumer groups are independent - both Python and Go groups can exist simultaneously.

#### Step 2: Run Integration Tests

```bash
# Module-specific tests
cd /home/orion/orion/<module-name>
pytest tests/integration/test_bus_integration.py -v

# Verify consumer group exists
redis-cli XINFO GROUPS orion:events | grep "orion-<module>-group-go"
```

#### Step 3: Deploy and Monitor (24 hours)

- Deploy updated module configuration
- Monitor consumer group lag:
  ```bash
  redis-cli XPENDING orion:events orion-<module>-group-go
  ```
- Check for processing errors in module logs
- Verify message acknowledgment (pending count should be 0)

#### Step 4: Verify or Rollback

**Verification criteria** (all must pass):
- ✅ Consumer group lag <100 messages (check hourly)
- ✅ No processing errors in logs
- ✅ Module behavior unchanged (compare with Python bus metrics)
- ✅ Pending message count returns to 0 after processing

**Rollback procedure** (if criteria fail):
```bash
# Revert module to Python bus consumer group
# Update config: orion-<module>-group-go → orion-<module>-group
# Restart module
systemctl restart orion-<module>
```

**Rollback verification**:
- Old consumer group receives messages
- Processing resumes normally
- Investigate Go bus issue before retrying

### Success Criteria (Phase 2 Complete)

- ✅ All 5 modules consuming from Go bus
- ✅ Python bus consumers stopped (no active consumer groups)
- ✅ Zero message loss (verify via Redis Stream message IDs)
- ✅ Consumer lag <100 messages across all groups
- ✅ No increase in processing errors (compare with baseline)

### Monitoring During Phase 2

**Daily checks**:
```bash
# Consumer group lag (all modules)
for group in memory guardian brain commander approval; do
  echo "Checking orion-$group-group-go:"
  redis-cli XPENDING orion:events "orion-$group-group-go" | head -5
done

# Go bus health
curl http://localhost:8080/health

# Go bus memory
ps aux | grep orion-bus | awk '{print $6/1024 " MB"}'

# Contract validation errors (should be zero)
grep "contract validation failed" /var/log/orion-bus-go.log | wc -l
```

---

## Phase 3: Full Cutover (Week 3)

### Goal
Complete migration by switching publishers to Go bus and removing Python bus code.

### Actions

#### Day 1-2: Publisher Migration

1. **Update publishers to Go bus**:
   - Modules that publish events: orion-brain, orion-guardian, orion-commander, orion-edge-agent
   - Update configuration to use Go bus endpoint (if applicable)
   - If publishers use Python bus library directly, update import to Go bus client

2. **Verify published messages**:
   ```bash
   # Publish test event
   # Verify appears in stream with valid contract
   redis-cli XRANGE orion:events - + COUNT 1
   ```

3. **Stop Python bus publishers** (but keep Python bus running for rollback):
   - Update systemd services to disable Python bus
   - Keep Python bus installed but not running

#### Day 3-7: Burn-in Period

- Monitor Go bus for 7 days with 100% traffic
- Check for anomalies:
  - Memory leaks (should stay <100MB)
  - Consumer lag (should stay <100 messages)
  - Processing errors (should match baseline)
  - Contract validation errors (should be zero)

#### Day 8: Remove Python Bus Code

**Only after successful burn-in**:

1. **Archive Python bus code**:
   ```bash
   cd /home/orion/orion/bus/python
   git mv orion-bus orion-bus-archived
   git commit -m "chore(bus): archive Python bus (migration complete)"
   ```

2. **Update systemd services**:
   - Remove orion-bus-python.service
   - Ensure orion-bus-go.service is enabled

3. **Update documentation**:
   - Mark Python bus as deprecated in README
   - Update architecture diagrams to show Go bus only

### Success Criteria (Phase 3 Complete)

- ✅ Go bus handles 100% of publish + subscribe traffic
- ✅ Python bus code archived (not deleted, for reference)
- ✅ Zero downtime during migration
- ✅ No message loss (verify via message count)
- ✅ Go bus memory <100MB (target achieved)
- ✅ Go bus latency p95 <10ms (Python: ~50ms)
- ✅ 7-day burn-in period successful

### Rollback (Emergency Only)

**When to rollback**:
- Contract validation failure rate >1%
- Memory usage >150MB (same as Python bus)
- Consumer lag >1000 messages sustained
- Critical processing errors

**Rollback procedure**:
1. Restart Python bus service:
   ```bash
   systemctl start orion-bus-python
   ```

2. Revert all modules to Python bus consumer groups:
   ```bash
   # Update config for each module
   # Restart all consumers
   for module in memory guardian brain commander approval; do
     systemctl restart orion-$module
   done
   ```

3. Stop Go bus:
   ```bash
   systemctl stop orion-bus-go
   ```

4. Verify Python bus operation:
   ```bash
   # Check consumer groups
   redis-cli XINFO GROUPS orion:events

   # Verify message processing
   tail -f /var/log/orion-*.log
   ```

**Recovery time**: ~15 minutes (all services restart, messages reprocessed)

---

## Monitoring and Metrics

### Key Metrics

| Metric | Target | Alert Threshold | Command |
|--------|--------|-----------------|---------|
| Go bus memory | <100MB | >150MB | `ps aux \| grep orion-bus \| awk '{print $6/1024}'` |
| Consumer lag | <100 msgs | >1000 msgs | `redis-cli XPENDING orion:events <group>` |
| Contract errors | 0 | >1% of traffic | `grep "contract validation failed" /var/log/orion-bus-go.log \| wc -l` |
| Health endpoint | 100% uptime | <99% uptime | `curl -f http://localhost:8080/health` |
| Message loss | 0 | Any loss | `redis-cli XLEN orion:events` (compare before/after) |

### Dashboards (If Using Prometheus/Grafana)

**Recommended panels**:
- Go bus memory (RSS) over time
- Consumer group lag by group
- Message publish rate
- Contract validation error rate
- HTTP /health response time

### Log Locations

- **Go bus logs**: `/var/log/orion-bus-go.log`
- **Python bus logs**: `/var/log/orion-bus-python.log` (Phase 1-2 only)
- **Module logs**: `/var/log/orion-<module>.log`
- **Redis logs**: `/var/log/redis/redis-server.log`

---

## Emergency Stop Criteria

**Halt migration immediately if**:

1. **Contract validation failures >1% of traffic**
   - Indicates incompatibility between Python/Go implementations
   - Action: Stop migration, investigate schema differences

2. **Memory usage >150MB sustained**
   - Go bus should be more efficient than Python bus
   - Action: Profile memory, investigate leaks

3. **Consumer lag >1000 messages for >5 minutes**
   - Indicates Go bus cannot keep up with traffic
   - Action: Check Go bus performance, verify consumer logic

4. **Processing errors increase >50% vs baseline**
   - Indicates behavioral differences between implementations
   - Action: Compare error logs, verify message handling

5. **Any data loss detected**
   - Zero tolerance for message loss
   - Action: Immediate rollback, full investigation

---

## Validation Tools

### Pre-Migration Checklist

Before starting Phase 1:
- [ ] Go bus builds successfully (`make build`)
- [ ] All 25 tests pass (`make test`)
- [ ] Validation script passes (`./scripts/migration-validate.sh`)
- [ ] Contracts directory accessible from Go bus
- [ ] Redis Streams healthy (XINFO STREAM orion:events)
- [ ] Python bus baseline metrics captured

### Migration Validation Script

Run after each phase:
```bash
cd /home/orion/orion/bus/go
./scripts/migration-validate.sh
```

**What it validates**:
- Redis availability
- Go bus health endpoint
- Consumer group creation
- Stream read/write
- Contract schema loading

**Exit codes**:
- 0: All tests passed
- 1: Prerequisites failed (Redis/Go bus not running)
- 2: Message roundtrip failed
- 3: Consumer group test failed
- 4: Validation test failed

---

## Rollback Decision Matrix

| Issue | Phase 1 | Phase 2 | Phase 3 | Action |
|-------|---------|---------|---------|--------|
| Go bus crashes | Stop Go bus | Revert module | Full rollback | Investigate logs |
| Memory >150MB | Monitor | Revert module | Full rollback | Profile memory |
| Contract errors >1% | Stop migration | Revert module | Full rollback | Fix schemas |
| Consumer lag >1000 | N/A | Revert module | Full rollback | Check performance |
| Processing errors increase | N/A | Revert module | Full rollback | Compare behavior |

**Decision rule**: When in doubt, rollback. ORION prefers safety over speed.

---

## Success Metrics (Final)

After Phase 3 completion and 7-day burn-in:

| Metric | Baseline (Python) | Target (Go) | Actual |
|--------|------------------|-------------|--------|
| Memory usage | ~150MB | <100MB | ___ |
| Publish latency (p95) | ~50ms | <10ms | ___ |
| Subscribe latency (p95) | ~50ms | <10ms | ___ |
| Uptime (7 days) | 99.9% | 99.9% | ___ |
| Contract validation errors | 0 | 0 | ___ |
| Message loss | 0 | 0 | ___ |
| Consumer lag (avg) | <50 msgs | <50 msgs | ___ |

**Fill in "Actual" column after 7-day burn-in to document migration success.**

---

## Contact and Support

- **Logs**: `/var/log/orion-bus-go.log`, `/var/log/orion-*.log`
- **Health check**: `curl http://localhost:8080/health`
- **Validation**: `./bus/go/scripts/migration-validate.sh`
- **Rollback**: See Phase-specific rollback procedures above

For issues during migration, consult:
1. This MIGRATION.md document
2. Go bus README.md
3. ORION CLAUDE.md (safety invariants)
4. Redis Streams documentation (consumer groups)

---

## Appendix: Consumer Group Management

### List all consumer groups:
```bash
redis-cli XINFO GROUPS orion:events
```

### Check pending messages:
```bash
redis-cli XPENDING orion:events <group-name>
```

### Delete consumer group (rollback):
```bash
redis-cli XGROUP DESTROY orion:events <group-name>
```

### Claim pending messages (if consumer dies):
```bash
redis-cli XCLAIM orion:events <group> <consumer> <min-idle-time> <message-id>
```

### Acknowledge stuck message manually:
```bash
redis-cli XACK orion:events <group> <message-id>
```

---

**Document Version**: 1.0
**Last Updated**: 2026-01-17
**Migration Status**: Pre-Phase 1 (Go bus ready for deployment)
