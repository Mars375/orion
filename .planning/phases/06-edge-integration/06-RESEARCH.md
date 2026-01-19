# Phase 6: Edge Integration - Research

**Researched:** 2026-01-17
**Status:** Complete

## Executive Summary

This research covers the technical foundation for integrating a Freenove Hexapod robot as an edge node in the ORION system. The architecture involves two Raspberry Pis: Pi 16GB (orion-core/Brain) running the Go Bus and AI Council, and Pi 4 (orion-edge) handling kinematics and local safety. Key findings support the viability of Go for the edge agent, MQTT for telemetry, Redis Streams for event bus integration, and a watchdog-based Dead Man's Switch for fail-safe behavior.

---

## 1. Freenove Hexapod Hardware & SDK

### 1.1 Hardware Overview

The Freenove Big Hexapod Robot Kit is a 6-legged robot with 18 servos (3 per leg) designed for Raspberry Pi. Each leg has three degrees of freedom (DOF) controlled by position-controlled servos.

**Key Components:**
- **Servo Controller:** PCA9685 16-channel, 12-bit PWM driver via I2C
- **Servos:** 18 standard servos (some kits use 2x PCA9685 boards for 18+ channels)
- **Communication:** I2C bus (GPIO 2/SDA, GPIO 3/SCL)
- **Power:** External 5-6V supply for servos (separate from Pi power)

**Sources:**
- [Freenove GitHub Repository](https://github.com/Freenove/Freenove_Big_Hexapod_Robot_Kit_for_Raspberry_Pi)
- [Freenove Store Product Page](https://store.freenove.com/products/fnk0052)

### 1.2 Existing SDK Structure

The Freenove SDK is primarily Python (52.2%) with C (46.5%) for low-level operations:
- Python handles high-level robot control and orchestration
- C manages performance-critical hardware interactions
- License: Creative Commons Attribution-NonCommercial-ShareAlike 3.0

**Implication for ORION:** We will NOT use the Python SDK directly. Instead, orion-edge-agent will be written in Go for consistency with the bus and better ARM cross-compilation. Go can interface with PCA9685 via I2C libraries.

### 1.3 Servo Control via PCA9685

**I2C Wiring:**
```
PCA9685 VCC  → Pi 3.3V
PCA9685 GND  → Pi GND
PCA9685 SCL  → GPIO 3 (SCL)
PCA9685 SDA  → GPIO 2 (SDA)
V+ (servos)  → External 5-6V supply
```

**PWM Configuration:**
- Frequency: 50-60 Hz for servos
- Pulse range: ~150 (min) to ~600 (max) out of 4096 for servo positioning
- Multiple PCA9685 boards can be addressed by soldering A0 jumper (addresses 0x40, 0x41)

**Go Libraries for PCA9685:**
- `periph.io/x/conn/v3/i2c` - Low-level I2C access
- `periph.io/x/devices/v3/pca9685` - Direct PCA9685 driver

**Sources:**
- [Adafruit PCA9685 with Raspberry Pi](https://learn.adafruit.com/adafruit-16-channel-servo-driver-with-raspberry-pi)
- [PCA9685 Setup Guide](https://www.kevsrobots.com/learn/pca9685/05_setting_up_the_pca9685_with_raspberry_pi.html)

### 1.4 Inverse Kinematics & Gait Patterns

**Tripod Gait:**
The hexapod uses a tripod gait where 3 legs move at a time while the other 3 maintain ground contact. This ensures stability by keeping the center of mass within the support triangle.

**Inverse Kinematics (IK) Approach:**
Instead of frame-by-frame servo angle sequences, IK calculates joint angles from high-level goals:
1. Define gait timing (duty factor for stance vs swing phase)
2. Model leg structure (link lengths, leg positions relative to body center)
3. Calculate foot trajectories (arc for swing, line for stance)
4. Apply trigonometric IK solver to compute servo angles
5. Update in tight control loop and send to servos

**Key Insight:** The Pi 4 edge agent handles ALL kinematics locally. The Brain sends high-level commands like "move forward at 5 cm/s" or "turn left 30 degrees" - never individual servo angles.

**Sources:**
- [Hackster: Tripod Gait Biology to IK](https://www.hackster.io/HiwonderRobot/tripod-gait-for-hexapod-robots-biology-to-inverse-kinematic-1d94b0)
- [IEEE: Hexapod Kinematics and Tripod Gait](https://ieeexplore.ieee.org/document/8355009/)

---

## 2. Go on Raspberry Pi ARM

### 2.1 Cross-Compilation

Go excels at cross-compilation for ARM targets. For Pi 4 (ARM64/aarch64):

**Pure Go (no CGO):**
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o orion-edge main.go
```

This produces a statically linked binary that runs directly on Pi 4 with no dependencies.

**With CGO (if needed for hardware):**
```bash
CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o orion-edge main.go
```

**Recommendation:** Prefer pure Go with `periph.io` for I2C/GPIO access to avoid CGO complexity.

**Sources:**
- [Go Wiki: Go on ARM](https://go.dev/wiki/GoArm)
- [Medium: Cross-compiling Go for Raspberry Pi](https://medium.com/@chrischdi/cross-compiling-go-for-raspberry-pi-dc09892dc745)
- [Opensource.com: Cross-compiling Go](https://opensource.com/article/21/1/go-cross-compiling)

### 2.2 Memory Footprint

Go binaries with `-ldflags="-s -w"` (strip symbols) are typically 5-15MB. The Pi 4 with 4GB RAM has plenty of headroom for:
- orion-edge-agent
- Redis client connection
- MQTT client connection
- Kinematics calculations
- Local safety logic

---

## 3. MQTT for Edge Telemetry & Commands

### 3.1 Topic Design Patterns

**Telemetry Topics (Edge → Brain):**
```
orion/edge/{device-id}/telemetry/position    # Current robot position
orion/edge/{device-id}/telemetry/battery     # Battery level
orion/edge/{device-id}/telemetry/temperature # Motor temperatures
orion/edge/{device-id}/telemetry/health      # Heartbeat/health status
```

**Command Topics (Brain → Edge):**
```
orion/edge/{device-id}/cmd/move              # Movement commands
orion/edge/{device-id}/cmd/stop              # Emergency stop
orion/edge/{device-id}/cmd/calibrate         # Servo calibration
orion/edge/{device-id}/res/{request-id}      # Command responses
```

**Best Practices:**
- Never subscribe to `#` (all topics) on devices
- Use single-level wildcards (`+`) for device subscriptions
- Separate telemetry from command topics
- Include device-id in topic for multi-device future

**Sources:**
- [AWS: MQTT Design Best Practices](https://docs.aws.amazon.com/whitepapers/latest/designing-mqtt-topics-aws-iot-core/mqtt-design-best-practices.html)
- [AWS: MQTT Communication Patterns](https://docs.aws.amazon.com/whitepapers/latest/designing-mqtt-topics-aws-iot-core/mqtt-communication-patterns.html)

### 3.2 QoS Selection

| Message Type | QoS | Rationale |
|--------------|-----|-----------|
| Telemetry (high-freq) | 0 | Fire-and-forget, occasional loss acceptable |
| Health heartbeat | 1 | Must be delivered for Dead Man's Switch |
| Movement commands | 1 | Delivery confirmation needed |
| Emergency stop | 2 | Exactly-once critical command |

### 3.3 Go MQTT Library: Eclipse Paho

**Recommended Package:** `github.com/eclipse/paho.golang/autopaho`

Key features:
- Automatic reconnection with exponential backoff
- Connection loss callbacks (critical for Dead Man's Switch)
- File-based queue for offline messages
- QoS 0/1/2 support

**Example Connection with Reconnection Handling:**
```go
cliCfg := autopaho.ClientConfig{
    ServerUrls:        []*url.URL{serverURL},
    KeepAlive:         20,
    ReconnectBackoff:  exponentialBackoff,
    OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
        fmt.Println("Connected - resuming operations")
    },
    OnConnectionDown: func() bool {
        triggerDeadManSwitch()  // CRITICAL: Trigger safety mode
        return true             // Continue reconnection attempts
    },
}
```

**Sources:**
- [Eclipse Paho Go Documentation](https://context7.com/eclipse-paho/paho.golang)

---

## 4. Redis Remote Connection

### 4.1 Network Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Home Network (LAN)                        │
│                                                             │
│   ┌─────────────────┐              ┌──────────────────┐    │
│   │  Pi 16GB        │              │  Pi 4            │    │
│   │  (orion-core)   │◄────────────►│  (orion-edge)    │    │
│   │                 │   Network    │                  │    │
│   │  Redis Server   │              │  Redis Client    │    │
│   │  Port 6379      │              │  go-redis        │    │
│   │                 │              │                  │    │
│   │  MQTT Broker    │              │  MQTT Client     │    │
│   │  (optional)     │              │  paho.golang     │    │
│   └─────────────────┘              └──────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Redis Security Configuration

**On Pi 16GB (redis.conf):**
```conf
# Bind to private network interface (not 0.0.0.0)
bind 192.168.x.x 127.0.0.1

# Require password authentication
requirepass <strong-password>

# Or use ACL for granular control (Redis 6+)
user orion-edge on >edgepassword ~orion:* +@read +@write +@pubsub

# Optional: TLS (recommended for sensitive environments)
# tls-port 6379
# port 0
# tls-cert-file /path/to/redis.crt
# tls-key-file /path/to/redis.key
```

**Firewall (ufw on Pi 16GB):**
```bash
sudo ufw allow from 192.168.x.0/24 to any port 6379
```

**Sources:**
- [Redis Security Documentation](https://redis.io/docs/latest/operate/oss_and_stack/management/security/)
- [Redis TLS Configuration](https://redis.io/docs/latest/operate/rs/7.4/security/encryption/tls/)

### 4.3 Go Redis Client: go-redis

**Recommended Package:** `github.com/redis/go-redis/v9`

**Remote Connection Example:**
```go
rdb := redis.NewClient(&redis.Options{
    Addr:     "192.168.x.x:6379",  // Pi 16GB address
    Password: "edgepassword",
    DB:       0,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

**Redis Streams for Event Bus:**
The edge agent can subscribe to Redis Streams for event bus integration:
```go
// Subscribe to command stream
streams, err := rdb.XRead(ctx, &redis.XReadArgs{
    Streams: []string{"orion:commands", "$"},
    Block:   5 * time.Second,
}).Result()

// Publish telemetry to stream
rdb.XAdd(ctx, &redis.XAddArgs{
    Stream: "orion:telemetry",
    Values: map[string]interface{}{
        "device":  "hexapod-1",
        "battery": 87,
    },
})
```

**Sources:**
- [go-redis Official Guide](https://redis.io/docs/latest/develop/clients/go/)
- [go-redis GitHub](https://github.com/redis/go-redis)

---

## 5. Dead Man's Switch Implementation

### 5.1 Concept

A Dead Man's Switch ensures the robot enters a safe state if communication with the Brain is lost. This is **local logic on the Pi 4** that does not depend on any remote command.

**Trigger Conditions:**
1. MQTT connection lost for > N seconds
2. Redis connection lost for > N seconds
3. No heartbeat received from Brain for > N seconds
4. Any combination of the above

**Safe State ("Sit & Freeze"):**
1. Stop all movement immediately
2. Lower body to ground (predefined safe servo positions)
3. Fold legs into stable configuration
4. Disable all autonomous movement
5. Continue attempting reconnection
6. Only resume operation when connection restored AND explicit "resume" command received

### 5.2 Go Implementation Pattern

**Watchdog Timer with Heartbeat:**
```go
type DeadManSwitch struct {
    timeout     time.Duration
    timer       *time.Timer
    mu          sync.Mutex
    triggered   bool
    onTrigger   func()  // Called when switch triggers
}

func NewDeadManSwitch(timeout time.Duration, onTrigger func()) *DeadManSwitch {
    dms := &DeadManSwitch{
        timeout:   timeout,
        onTrigger: onTrigger,
    }
    dms.timer = time.AfterFunc(timeout, dms.trigger)
    return dms
}

// Reset must be called periodically (on each heartbeat/command)
func (d *DeadManSwitch) Reset() {
    d.mu.Lock()
    defer d.mu.Unlock()
    if !d.triggered {
        d.timer.Reset(d.timeout)
    }
}

func (d *DeadManSwitch) trigger() {
    d.mu.Lock()
    d.triggered = true
    d.mu.Unlock()
    d.onTrigger()  // Execute safety action
}
```

**Integration with MQTT Connection Callbacks:**
```go
cliCfg := autopaho.ClientConfig{
    OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
        deadManSwitch.Reset()
        // Resume normal operation only after explicit command
    },
    OnConnectionDown: func() bool {
        // Timer will trigger if reconnection takes too long
        log.Warn("Connection lost - Dead Man's Switch armed")
        return true
    },
}
```

**Sources:**
- [Wikipedia: Dead Man's Switch](https://en.wikipedia.org/wiki/Dead_man's_switch)
- [Standard Bots: Dead Man's Switch Explained](https://standardbots.com/blog/dead-mans-switch)
- [Medium: Heartbeats in Golang](https://medium.com/geekculture/heartbeats-in-golang-1a12c4c366f)

### 5.3 Timeout Recommendations

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Heartbeat interval | 1 second | Frequent enough to detect issues quickly |
| Dead Man's timeout | 3-5 seconds | Allows for brief network hiccups |
| Reconnection backoff | 1s → 60s max | Exponential to avoid overwhelming network |
| Resume lockout | Manual only | Never auto-resume after safety trigger |

---

## 6. Communication Protocol Design

### 6.1 Dual-Channel Architecture

The edge agent uses TWO communication channels:

**Channel 1: MQTT (Primary for Commands)**
- Low-latency command delivery
- QoS guarantees
- Native connection loss detection
- Used for: Movement commands, emergency stop, status requests

**Channel 2: Redis Streams (Event Bus Integration)**
- Integration with ORION event bus
- Persistent message history
- Consumer groups for reliability
- Used for: Telemetry logging, incident correlation, audit trail

### 6.2 Message Flow

```
Brain (Pi 16GB)                           Edge (Pi 4)
     │                                         │
     │──── MQTT: cmd/move ────────────────────►│
     │                                         │ (Execute kinematics)
     │◄─── MQTT: res/move (ack) ──────────────│
     │                                         │
     │◄─── Redis: telemetry/position ─────────│
     │◄─── Redis: telemetry/battery ──────────│
     │                                         │
     │──── MQTT: cmd/stop (emergency) ────────►│
     │                                         │ (Immediate halt)
```

### 6.3 Contract Schemas

Edge integration will use ORION bus contracts. Proposed schemas:

**edge.command.schema.json:**
```json
{
  "type": "object",
  "required": ["command_id", "command_type", "timestamp", "source"],
  "properties": {
    "command_id": { "type": "string", "format": "uuid" },
    "command_type": { "enum": ["MOVE", "STOP", "CALIBRATE", "STATUS"] },
    "parameters": { "type": "object" },
    "timestamp": { "type": "string", "format": "date-time" },
    "source": { "type": "string" }
  }
}
```

**edge.telemetry.schema.json:**
```json
{
  "type": "object",
  "required": ["device_id", "telemetry_type", "value", "timestamp"],
  "properties": {
    "device_id": { "type": "string" },
    "telemetry_type": { "enum": ["POSITION", "BATTERY", "TEMPERATURE", "HEALTH"] },
    "value": { "type": "object" },
    "timestamp": { "type": "string", "format": "date-time" }
  }
}
```

---

## 7. Recommended Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Edge Agent | Go | Matches bus, static binaries, excellent ARM support |
| MQTT Client | paho.golang/autopaho | Auto-reconnect, connection callbacks |
| Redis Client | go-redis/v9 | Official client, Streams support |
| I2C/Servo | periph.io | Pure Go hardware access |
| Kinematics | Custom Go | Simple trig-based IK, no external deps |
| Safety Logic | Custom Go | Watchdog timer, local-only |

---

## 8. Risk Assessment

### 8.1 High Priority Risks

| Risk | Mitigation |
|------|------------|
| Network partition during movement | Dead Man's Switch triggers immediate safe state |
| Servo malfunction | Temperature monitoring, current limiting (hardware) |
| Pi 4 crash/hang | Hardware watchdog timer (bcm2835_wdt) |
| Redis unavailable | MQTT-only fallback mode for critical commands |

### 8.2 Medium Priority Risks

| Risk | Mitigation |
|------|------------|
| I2C bus errors | Retry logic, error logging, graceful degradation |
| Power brownout | Battery monitoring, early warning via telemetry |
| Clock drift | NTP sync, relative timestamps for local logic |

---

## 9. Implementation Recommendations

### 9.1 Phase 6 Should Include

1. **orion-edge-agent** (Go binary):
   - MQTT client with connection callbacks
   - Redis Streams client for bus integration
   - Dead Man's Switch watchdog
   - Basic "Sit & Freeze" safety state
   - Stub/mock for kinematics (actual IK can be Phase 7)

2. **Bus contracts**:
   - edge.command.schema.json
   - edge.telemetry.schema.json
   - edge.health.schema.json

3. **Redis configuration**:
   - Remote access from Pi 4
   - ACL user for edge agent
   - Firewall rules

4. **Integration tests**:
   - Connection loss triggers safety
   - Reconnection resumes only with explicit command
   - Telemetry flows to bus

### 9.2 Out of Scope for Phase 6

- Actual servo control (hardware interfacing)
- Full inverse kinematics implementation
- Physical robot testing
- MQTT broker deployment (can use existing or local dev broker)

---

## 10. References

### Hardware & Robotics
- [Freenove Big Hexapod GitHub](https://github.com/Freenove/Freenove_Big_Hexapod_Robot_Kit_for_Raspberry_Pi)
- [Adafruit PCA9685 Guide](https://learn.adafruit.com/adafruit-16-channel-servo-driver-with-raspberry-pi)
- [Hackster: Tripod Gait IK](https://www.hackster.io/HiwonderRobot/tripod-gait-for-hexapod-robots-biology-to-inverse-kinematic-1d94b0)

### Go Development
- [Go Wiki: Go on ARM](https://go.dev/wiki/GoArm)
- [periph.io - Go Hardware Library](https://periph.io/)
- [go-redis Documentation](https://redis.io/docs/latest/develop/clients/go/)
- [Eclipse Paho Go](https://github.com/eclipse/paho.golang)

### Communication
- [AWS MQTT Best Practices](https://docs.aws.amazon.com/whitepapers/latest/designing-mqtt-topics-aws-iot-core/mqtt-design-best-practices.html)
- [Redis Security](https://redis.io/docs/latest/operate/oss_and_stack/management/security/)

### Safety Patterns
- [Dead Man's Switch - Wikipedia](https://en.wikipedia.org/wiki/Dead_man's_switch)
- [Standard Bots: Dead Man's Switch](https://standardbots.com/blog/dead-mans-switch)
- [Heartbeats in Golang](https://medium.com/geekculture/heartbeats-in-golang-1a12c4c366f)

---

*Phase: 06-edge-integration*
*Research completed: 2026-01-17*
