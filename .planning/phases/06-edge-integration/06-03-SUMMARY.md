---
phase: 06-edge-integration
plan: 03
type: summary
status: complete
---

# Plan 06-03 Summary: Dead Man's Switch and Safe State

## Completed Tasks

### Task 1: DeadManSwitch Watchdog Timer ✓

**edge/internal/safety/watchdog.go:**

Implemented the Dead Man's Switch watchdog with the following invariants:
- Watchdog MUST be Reset() on every heartbeat/command from Brain
- Timeout triggers safe state callback (non-reversible without explicit resume)
- Safe state callback runs ONCE per trigger (idempotent)
- Thread-safe (concurrent Reset calls from multiple goroutines)

**Methods:**
- `NewDeadManSwitch(timeout, onTrigger)` - Creates and arms watchdog immediately
- `Reset()` - Resets timer; no-op if already triggered (sticky state)
- `Stop()` - Stops timer for cleanup
- `IsTriggered() bool` - Returns triggered state
- `ClearTriggered()` - Allows resumption (RESUME command only)
- `RemainingMs() int` - Returns milliseconds until trigger

**Tests (9 tests):**
1. TestWatchdogTriggersAfterTimeout
2. TestWatchdogResetPreventsTrigger
3. TestWatchdogTriggeredStateSticky
4. TestWatchdogClearAllowsResume
5. TestWatchdogCallbackRunsOnce
6. TestWatchdogConcurrentResetSafe
7. TestWatchdogRemainingMs
8. TestWatchdogRemainingMsWhenTriggered
9. TestWatchdogStop

### Task 2: SafeStateManager for "Sit & Freeze" ✓

**edge/internal/safety/safe_state.go:**

Implemented the safe state manager for "Sit & Freeze" behavior:
- Stops all movement immediately when triggered
- Moves to safe position (stub in Phase 6)
- Disables autonomous movement
- Resumption requires explicit RESUME command

**Methods:**
- `NewSafeStateManager(onEnter, onExit)` - Creates manager with callbacks
- `EnterSafeMode()` - Enters safe state (idempotent)
- `ExitSafeMode() error` - Exits safe state (returns error if not in safe mode)
- `IsInSafeMode() bool` - Returns current state
- `GetSafePosition()` - Returns stub position data

**Tests (6 tests):**
1. TestEnterSafeModeSetsFLag
2. TestExitSafeModeClearsFlag
3. TestExitWhenNotSafeReturnsError
4. TestGetSafePositionReturnsStub
5. TestEnterSafeModeIdempotent
6. TestSafeStateManagerWithNilCallbacks

### Task 3: Main Agent Integration ✓

**edge/cmd/orion-edge/main.go:**

Integrated safety components into the main agent lifecycle:

1. **SafeStateManager creation** with stub kinematics callbacks
2. **DeadManSwitch creation** with configured timeout
3. **MQTT connection callbacks** wired:
   - OnConnectionUp: Reset watchdog, log "Brain connection restored"
   - OnConnectionDown: Log warning, watchdog continues
4. **Watchdog reset points**:
   - On successful Redis connection
   - On successful MQTT connection
   - On reconnection (OnConnectionUp)
   - On any command received
5. **Health endpoint** updated with safety fields:
   - `safe_mode`: boolean
   - `watchdog_triggered`: boolean
6. **Health messages** include full safety_state:
   - `dead_man_switch_active`: watchdog.IsTriggered()
   - `watchdog_remaining_ms`: watchdog.RemainingMs()
   - `in_safe_position`: safeState.IsInSafeMode()
7. **Command handler** implemented:
   - RESUME: Clears watchdog and exits safe mode
   - STOP: Logs command (stub)
   - MOVE: Rejected if in safe mode
   - CALIBRATE: Rejected if in safe mode
   - STATUS: Reports via heartbeat

## Verification Results

| Check | Status |
|-------|--------|
| `go test -v -race ./internal/safety/...` passes | ✓ (15 tests) |
| DeadManSwitch triggers callback after timeout | ✓ |
| DeadManSwitch.Reset() prevents trigger | ✓ |
| Triggered state is sticky (no auto-resume) | ✓ |
| SafeStateManager enters/exits safe mode correctly | ✓ |
| Main agent compiles with safety integration | ✓ |
| Health endpoint includes safe_mode and watchdog_triggered | ✓ |
| RESUME command handling implemented | ✓ |

## Files Created/Modified

**Created:**
- `edge/internal/safety/watchdog.go` - DeadManSwitch implementation
- `edge/internal/safety/watchdog_test.go` - 9 watchdog tests
- `edge/internal/safety/safe_state.go` - SafeStateManager implementation
- `edge/internal/safety/safe_state_test.go` - 6 safe state tests

**Modified:**
- `edge/cmd/orion-edge/main.go` - Integrated safety components

## Design Decisions

1. **Sticky triggered state**: Once the Dead Man's Switch triggers, Reset() calls are ignored. Only explicit ClearTriggered() (via RESUME command) can restore normal operation. This prevents accidental auto-resume on reconnection.

2. **Watchdog reset on reconnection**: The watchdog is reset when MQTT reconnects, but the triggered state is NOT cleared. This ensures the device stays in safe mode until Brain explicitly confirms it's safe to resume.

3. **Commands rejected in safe mode**: MOVE and CALIBRATE commands are rejected when in safe mode. Only RESUME can transition out of safe mode.

4. **Health messages include remaining time**: The `watchdog_remaining_ms` field allows Brain to monitor how close the edge device is to triggering safe mode.

5. **Stub kinematics**: The onEnterSafe/onExitSafe callbacks log stub messages. Actual servo control will be implemented in Phase 7 or later.

## Safety Invariants Enforced

1. **No auto-resume**: After connection loss triggers safe mode, the device stays frozen until explicit RESUME command
2. **Fail-closed**: If watchdog expires, device enters safe state immediately
3. **Idempotent triggers**: Multiple timeouts only call the callback once
4. **Thread-safe**: Concurrent Reset() calls from heartbeats and commands are safe
5. **Conservative defaults**: 5-second watchdog timeout by default

## Next Steps

Plan 06-04 will implement MQTT topic routing and QoS handling for commands and telemetry.
