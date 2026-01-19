package safety

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchdogTriggersAfterTimeout(t *testing.T) {
	triggered := make(chan struct{})

	dms := NewDeadManSwitch(50*time.Millisecond, func() {
		close(triggered)
	})
	defer dms.Stop()

	select {
	case <-triggered:
		// Expected - callback was called
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Watchdog did not trigger within expected time")
	}

	if !dms.IsTriggered() {
		t.Error("Expected IsTriggered() to return true after timeout")
	}
}

func TestWatchdogResetPreventsTrigger(t *testing.T) {
	callCount := int32(0)

	dms := NewDeadManSwitch(100*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
	})
	defer dms.Stop()

	// Reset multiple times before timeout
	for i := 0; i < 5; i++ {
		time.Sleep(30 * time.Millisecond)
		dms.Reset()
	}

	// Wait a bit more but less than timeout since last reset
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("Expected callback not to be called, but was called %d times", callCount)
	}

	if dms.IsTriggered() {
		t.Error("Expected IsTriggered() to return false when reset prevents trigger")
	}
}

func TestWatchdogTriggeredStateSticky(t *testing.T) {
	triggered := make(chan struct{})
	callCount := int32(0)

	dms := NewDeadManSwitch(50*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
		close(triggered)
	})
	defer dms.Stop()

	// Wait for trigger
	<-triggered

	// Try to reset after trigger - should be no-op
	dms.Reset()

	if !dms.IsTriggered() {
		t.Error("Expected IsTriggered() to remain true after Reset() - triggered state should be sticky")
	}

	// Wait to ensure callback isn't called again
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("Expected callback to be called exactly once, but was called %d times", callCount)
	}
}

func TestWatchdogClearAllowsResume(t *testing.T) {
	callCount := int32(0)
	triggered := make(chan struct{}, 2)

	dms := NewDeadManSwitch(50*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
		triggered <- struct{}{}
	})
	defer dms.Stop()

	// Wait for first trigger
	<-triggered

	if !dms.IsTriggered() {
		t.Error("Expected IsTriggered() to be true after trigger")
	}

	// Clear the triggered state
	dms.ClearTriggered()

	if dms.IsTriggered() {
		t.Error("Expected IsTriggered() to be false after ClearTriggered()")
	}

	// Wait for second trigger
	select {
	case <-triggered:
		// Expected - watchdog restarted and triggered again
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Watchdog did not trigger again after ClearTriggered()")
	}

	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("Expected callback to be called twice, but was called %d times", callCount)
	}
}

func TestWatchdogCallbackRunsOnce(t *testing.T) {
	callCount := int32(0)

	dms := NewDeadManSwitch(50*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
	})
	defer dms.Stop()

	// Wait well beyond the timeout
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("Expected callback to be called exactly once, but was called %d times", callCount)
	}
}

func TestWatchdogConcurrentResetSafe(t *testing.T) {
	callCount := int32(0)

	dms := NewDeadManSwitch(100*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
	})
	defer dms.Stop()

	// Spawn multiple goroutines that concurrently call Reset
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				dms.Reset()
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Watchdog should not have triggered due to continuous resets
	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("Expected callback not to be called during concurrent resets, but was called %d times", callCount)
	}

	if dms.IsTriggered() {
		t.Error("Expected IsTriggered() to be false during concurrent resets")
	}
}

func TestWatchdogRemainingMs(t *testing.T) {
	dms := NewDeadManSwitch(200*time.Millisecond, func() {})
	defer dms.Stop()

	// Initially should be close to 200ms
	remaining := dms.RemainingMs()
	if remaining < 150 || remaining > 200 {
		t.Errorf("Expected remaining to be ~200ms, got %dms", remaining)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	remaining = dms.RemainingMs()
	if remaining < 50 || remaining > 150 {
		t.Errorf("Expected remaining to be ~100ms after waiting, got %dms", remaining)
	}

	// Reset should restore to full timeout
	dms.Reset()
	remaining = dms.RemainingMs()
	if remaining < 150 || remaining > 200 {
		t.Errorf("Expected remaining to be ~200ms after reset, got %dms", remaining)
	}
}

func TestWatchdogRemainingMsWhenTriggered(t *testing.T) {
	triggered := make(chan struct{})

	dms := NewDeadManSwitch(50*time.Millisecond, func() {
		close(triggered)
	})
	defer dms.Stop()

	<-triggered

	remaining := dms.RemainingMs()
	if remaining != 0 {
		t.Errorf("Expected remaining to be 0 when triggered, got %dms", remaining)
	}
}

func TestWatchdogStop(t *testing.T) {
	callCount := int32(0)

	dms := NewDeadManSwitch(50*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
	})

	// Stop before timeout
	dms.Stop()

	// Wait beyond the original timeout
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("Expected callback not to be called after Stop, but was called %d times", callCount)
	}
}
