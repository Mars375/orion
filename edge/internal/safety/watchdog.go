package safety

import (
	"log"
	"sync"
	"time"
)

// DeadManSwitch monitors Brain connectivity and triggers safe state on timeout.
//
// Invariants:
//   - Watchdog MUST be Reset() on every heartbeat/command from Brain
//   - Timeout triggers safe state callback (non-reversible without explicit resume)
//   - Safe state callback runs ONCE per trigger (idempotent)
//   - Thread-safe (concurrent Reset calls from multiple goroutines)
type DeadManSwitch struct {
	timeout   time.Duration
	timer     *time.Timer
	mu        sync.Mutex
	triggered bool
	stopped   bool
	onTrigger func() // Called when watchdog expires
	logger    *log.Logger
	startTime time.Time // Track when timer was last reset
}

// NewDeadManSwitch creates a new watchdog that triggers onTrigger after timeout.
// The timer starts immediately upon creation.
func NewDeadManSwitch(timeout time.Duration, onTrigger func()) *DeadManSwitch {
	dms := &DeadManSwitch{
		timeout:   timeout,
		triggered: false,
		stopped:   false,
		onTrigger: onTrigger,
		logger:    log.Default(),
		startTime: time.Now(),
	}

	dms.timer = time.AfterFunc(timeout, dms.expire)
	dms.logger.Printf("INFO: Dead Man's Switch armed with %v timeout", timeout)

	return dms
}

// expire is called when the timer expires.
func (d *DeadManSwitch) expire() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.triggered || d.stopped {
		// Already triggered or stopped, don't call again
		return
	}

	d.triggered = true
	d.logger.Printf("CRITICAL: Dead Man's Switch TRIGGERED - entering safe state")

	if d.onTrigger != nil {
		d.onTrigger()
	}
}

// Reset resets the timer to the full timeout duration.
// No-op if already triggered (safe state is sticky) or stopped.
// Thread-safe with mutex.
func (d *DeadManSwitch) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.triggered {
		// Safe state is sticky - don't reset
		d.logger.Printf("WARN: Reset called but Dead Man's Switch already triggered - ignoring")
		return
	}

	if d.stopped {
		return
	}

	// Stop the current timer and create a new one
	d.timer.Stop()
	d.startTime = time.Now()
	d.timer = time.AfterFunc(d.timeout, d.expire)
}

// Stop stops the timer (for cleanup).
func (d *DeadManSwitch) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stopped = true
	d.timer.Stop()
	d.logger.Printf("INFO: Dead Man's Switch stopped")
}

// IsTriggered returns true if safe state was triggered.
func (d *DeadManSwitch) IsTriggered() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.triggered
}

// ClearTriggered allows resumption (called only on explicit RESUME command from Brain).
// Restarts the watchdog timer.
func (d *DeadManSwitch) ClearTriggered() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.triggered {
		return
	}

	d.triggered = false
	d.stopped = false
	d.startTime = time.Now()
	d.timer = time.AfterFunc(d.timeout, d.expire)
	d.logger.Printf("WARN: Dead Man's Switch cleared - resuming operations")
}

// RemainingMs returns the approximate milliseconds remaining before trigger.
// Returns 0 if triggered or stopped.
func (d *DeadManSwitch) RemainingMs() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.triggered || d.stopped {
		return 0
	}

	elapsed := time.Since(d.startTime)
	remaining := d.timeout - elapsed
	if remaining < 0 {
		return 0
	}
	return int(remaining.Milliseconds())
}
