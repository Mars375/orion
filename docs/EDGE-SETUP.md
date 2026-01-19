# ORION Edge Device Setup Guide

This guide covers setting up a Raspberry Pi 4 as an ORION edge device (orion-edge) that connects to orion-core for robot control.

## Overview

The ORION edge architecture:

```
[Pi 16GB: orion-core]              [Pi 4: orion-edge]
+------------------+               +------------------+
|  Redis Streams   |<--telemetry---|  orion-edge     |
|  MQTT Broker     |<--health------|  agent          |
|  orion-brain     |---commands--->|                 |
+------------------+               +------------------+
        |                                   |
        +---- LAN (192.168.x.x) -----------+
```

- **orion-core** (Pi 16GB): Runs Redis, MQTT broker, and orion-brain
- **orion-edge** (Pi 4): Runs orion-edge-agent for robot control

## Prerequisites

- Raspberry Pi 4 (4GB or 8GB recommended)
- Raspberry Pi OS 64-bit (Bookworm or later)
- Network connectivity to orion-core
- `orion-edge` binary built for ARM64

## 1. Build the Edge Agent

On a development machine (or cross-compile):

```bash
cd /home/orion/orion/edge

# Build for ARM64 (Raspberry Pi 4)
make build-arm64

# Binary is at: bin/orion-edge-arm64
```

Or build natively on the Pi 4:

```bash
# Install Go 1.22+ on Pi 4 first
cd /home/orion/orion/edge
go build -o orion-edge ./cmd/orion-edge
```

## 2. Configure Redis on orion-core (Pi 16GB)

### 2.1 Edit Redis Configuration

```bash
sudo nano /etc/redis/redis.conf
```

Add the following settings (from `config/redis-edge.conf`):

```conf
# Bind to LAN interface (replace with actual IP of Pi 16GB)
bind 127.0.0.1 192.168.1.100

# Require password for edge devices
requirepass your-strong-password-here

# Memory limit (leave room for LLMs)
maxmemory 512mb
maxmemory-policy allkeys-lru
```

### 2.2 Restart Redis

```bash
sudo systemctl restart redis-server
sudo systemctl status redis-server
```

### 2.3 Test Local Connection

```bash
redis-cli -h 192.168.1.100 -a your-strong-password-here ping
# Should return: PONG
```

## 3. Configure Firewall on orion-core

Allow Redis connections from local network only:

```bash
# Enable UFW if not already
sudo ufw enable

# Allow Redis from LAN only
sudo ufw allow from 192.168.0.0/16 to any port 6379

# Allow MQTT from LAN only
sudo ufw allow from 192.168.0.0/16 to any port 1883

# Verify rules
sudo ufw status
```

Expected output:
```
6379                       ALLOW       192.168.0.0/16
1883                       ALLOW       192.168.0.0/16
```

## 4. Deploy orion-edge to Pi 4

### 4.1 Copy Binary

```bash
# From development machine
scp bin/orion-edge-arm64 pi@pi4-hostname:/home/pi/orion-edge

# On Pi 4
chmod +x /home/pi/orion-edge
```

### 4.2 Test Connection to orion-core

```bash
# Test Redis connectivity from Pi 4
redis-cli -h 192.168.1.100 -a your-password ping

# Test MQTT connectivity (if mosquitto_pub installed)
mosquitto_pub -h 192.168.1.100 -t test -m "hello"
```

## 5. Run orion-edge

### 5.1 Basic Command

```bash
./orion-edge \
  --device-id hexapod-1 \
  --redis-addr 192.168.1.100:6379 \
  --redis-password your-strong-password-here \
  --mqtt-broker tcp://192.168.1.100:1883 \
  --watchdog-timeout 5 \
  --heartbeat-interval 1
```

### 5.2 Full Options

| Flag | Description | Default |
|------|-------------|---------|
| `--device-id` | Unique identifier for this device | `hexapod-1` |
| `--redis-addr` | Redis server address | `localhost:6379` |
| `--redis-password` | Redis password | (empty) |
| `--mqtt-broker` | MQTT broker URL | `tcp://localhost:1883` |
| `--watchdog-timeout` | Dead Man's Switch timeout (seconds) | `5` |
| `--heartbeat-interval` | Health publish interval (seconds) | `1` |
| `--http-port` | HTTP health endpoint port | `8081` |

### 5.3 Verify Operation

```bash
# Check health endpoint
curl http://localhost:8081/health

# Expected response:
{
  "status": "ok",
  "service": "orion-edge",
  "device_id": "hexapod-1",
  "version": "0.1.0",
  "mqtt_connected": true,
  "safe_mode": false,
  "watchdog_triggered": false
}
```

## 6. Safety Behavior: Dead Man's Switch

The orion-edge agent implements a Dead Man's Switch for safety:

### When Safe Mode Activates

Safe mode ("Sit & Freeze") activates when:
- MQTT connection lost for longer than `--watchdog-timeout`
- No commands received from Brain for longer than `--watchdog-timeout`

### Safe Mode Behavior

When in safe mode:
1. Robot enters "Sit & Freeze" position (kinematics stub in v0.1)
2. All movement commands are rejected
3. Health reports `safe_mode: true`
4. Watchdog shows `dead_man_switch_active: true`

### Exiting Safe Mode

Safe mode is **sticky** and requires explicit action:

1. Restore network connectivity
2. Send RESUME command from orion-brain
3. Agent clears watchdog and exits safe mode

The agent will NOT automatically exit safe mode when connectivity is restored.

## 7. Run as systemd Service

### 7.1 Create Service File

```bash
sudo nano /etc/systemd/system/orion-edge.service
```

```ini
[Unit]
Description=ORION Edge Agent
After=network.target

[Service]
Type=simple
User=pi
ExecStart=/home/pi/orion-edge \
  --device-id hexapod-1 \
  --redis-addr 192.168.1.100:6379 \
  --redis-password your-password \
  --mqtt-broker tcp://192.168.1.100:1883 \
  --watchdog-timeout 5 \
  --heartbeat-interval 1
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 7.2 Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable orion-edge
sudo systemctl start orion-edge
sudo systemctl status orion-edge
```

### 7.3 View Logs

```bash
journalctl -u orion-edge -f
```

## 8. Troubleshooting

### "Connection refused" to Redis

1. Verify Redis bind address includes orion-core's LAN IP
2. Verify firewall allows the connection: `sudo ufw status`
3. Verify password is correct
4. Test from Pi 4: `redis-cli -h <ip> -a <password> ping`

### Watchdog Keeps Triggering

1. Check MQTT broker is running on orion-core
2. Verify orion-brain is sending heartbeats/commands
3. Check network latency: `ping orion-core-ip`
4. Increase `--watchdog-timeout` if network is slow

### Safe Mode Won't Exit

1. Verify MQTT connection is established (check health endpoint)
2. Send RESUME command from orion-brain
3. Check agent logs for errors: `journalctl -u orion-edge`

### Agent Fails to Start

1. Check all required flags are provided
2. Verify Redis and MQTT are reachable
3. Check for port conflicts on 8081 (HTTP health)
4. Run manually first to see error messages

## 9. Network Security Notes

### For Home LAN

The current setup assumes a trusted home LAN. For additional security:

- Use strong, unique passwords for Redis
- Consider Redis ACLs for fine-grained access control
- Keep firewall rules restricted to LAN only

### For External Access (Tailscale)

If accessing over Tailscale or external networks:

- Use Redis TLS (requires additional configuration)
- Use MQTT TLS (port 8883)
- Consider mTLS for mutual authentication
- Never expose Redis port 6379 to the internet

## 10. Redis Stream Names

The agent uses these Redis streams:

| Stream | Direction | Purpose |
|--------|-----------|---------|
| `orion:edge:telemetry` | Edge → Core | Telemetry data |
| `orion:edge:commands:<device-id>` | Core → Edge | Commands |

Health messages are published via MQTT to `orion/edge/<device-id>/health`.
