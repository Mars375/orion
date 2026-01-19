package safety

import (
	"sync/atomic"
	"testing"
)

func TestEnterSafeModeSetsFLag(t *testing.T) {
	enterCount := int32(0)

	ssm := NewSafeStateManager(
		func() { atomic.AddInt32(&enterCount, 1) },
		func() {},
	)

	if ssm.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be false initially")
	}

	ssm.EnterSafeMode()

	if !ssm.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be true after EnterSafeMode()")
	}

	if atomic.LoadInt32(&enterCount) != 1 {
		t.Errorf("Expected onEnterSafe callback to be called once, but was called %d times", enterCount)
	}
}

func TestExitSafeModeClearsFlag(t *testing.T) {
	exitCount := int32(0)

	ssm := NewSafeStateManager(
		func() {},
		func() { atomic.AddInt32(&exitCount, 1) },
	)

	// Enter safe mode first
	ssm.EnterSafeMode()

	if !ssm.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be true after EnterSafeMode()")
	}

	// Exit safe mode
	err := ssm.ExitSafeMode()
	if err != nil {
		t.Errorf("Expected no error from ExitSafeMode(), got: %v", err)
	}

	if ssm.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be false after ExitSafeMode()")
	}

	if atomic.LoadInt32(&exitCount) != 1 {
		t.Errorf("Expected onExitSafe callback to be called once, but was called %d times", exitCount)
	}
}

func TestExitWhenNotSafeReturnsError(t *testing.T) {
	ssm := NewSafeStateManager(func() {}, func() {})

	err := ssm.ExitSafeMode()
	if err == nil {
		t.Error("Expected error from ExitSafeMode() when not in safe mode")
	}

	if err != ErrNotInSafeMode {
		t.Errorf("Expected ErrNotInSafeMode, got: %v", err)
	}
}

func TestGetSafePositionReturnsStub(t *testing.T) {
	ssm := NewSafeStateManager(func() {}, func() {})

	pos := ssm.GetSafePosition()

	if pos == nil {
		t.Fatal("Expected non-nil position map")
	}

	if state, ok := pos["state"]; !ok || state != "sit_freeze" {
		t.Errorf("Expected state to be 'sit_freeze', got: %v", pos["state"])
	}

	if height, ok := pos["height"]; !ok || height != 0.0 {
		t.Errorf("Expected height to be 0.0, got: %v", pos["height"])
	}

	if legsFolded, ok := pos["legs_folded"]; !ok || legsFolded != true {
		t.Errorf("Expected legs_folded to be true, got: %v", pos["legs_folded"])
	}
}

func TestEnterSafeModeIdempotent(t *testing.T) {
	enterCount := int32(0)

	ssm := NewSafeStateManager(
		func() { atomic.AddInt32(&enterCount, 1) },
		func() {},
	)

	// Enter multiple times
	ssm.EnterSafeMode()
	ssm.EnterSafeMode()
	ssm.EnterSafeMode()

	if atomic.LoadInt32(&enterCount) != 1 {
		t.Errorf("Expected onEnterSafe callback to be called exactly once, but was called %d times", enterCount)
	}

	if !ssm.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be true")
	}
}

func TestSafeStateManagerWithNilCallbacks(t *testing.T) {
	ssm := NewSafeStateManager(nil, nil)

	// Should not panic with nil callbacks
	ssm.EnterSafeMode()
	if !ssm.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be true")
	}

	err := ssm.ExitSafeMode()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if ssm.IsInSafeMode() {
		t.Error("Expected IsInSafeMode() to be false")
	}
}
