package safety

import (
	"errors"
	"log"
	"sync"
)

// ErrNotInSafeMode is returned when trying to exit safe mode when not in it.
var ErrNotInSafeMode = errors.New("not currently in safe mode")

// SafeStateManager handles the "Sit & Freeze" safe state for edge devices.
//
// When triggered:
//  1. Stops all movement immediately
//  2. Moves to safe position (stub in Phase 6)
//  3. Disables autonomous movement
//  4. Publishes health with state=SAFE_MODE
//
// Resumption requires explicit RESUME command from Brain after reconnection.
type SafeStateManager struct {
	inSafeMode  bool
	mu          sync.Mutex
	logger      *log.Logger
	onEnterSafe func() // Callback for kinematics (stub)
	onExitSafe  func() // Callback for resumption (stub)
}

// NewSafeStateManager creates a new safe state manager with the given callbacks.
// onEnter is called when entering safe mode (kinematics: stop and sit).
// onExit is called when exiting safe mode (kinematics: resume).
func NewSafeStateManager(onEnter, onExit func()) *SafeStateManager {
	return &SafeStateManager{
		inSafeMode:  false,
		logger:      log.Default(),
		onEnterSafe: onEnter,
		onExitSafe:  onExit,
	}
}

// EnterSafeMode enters the "Sit & Freeze" safe state.
// Idempotent - calling multiple times has no additional effect.
// Thread-safe.
func (s *SafeStateManager) EnterSafeMode() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.inSafeMode {
		// Already in safe mode - idempotent
		return
	}

	s.inSafeMode = true
	s.logger.Printf("SAFETY: Entering safe mode - Sit & Freeze")

	if s.onEnterSafe != nil {
		s.onEnterSafe()
	}
}

// ExitSafeMode exits safe mode and allows resumption.
// Only exits if currently in safe mode.
// Returns error if not in safe mode.
// Thread-safe.
func (s *SafeStateManager) ExitSafeMode() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.inSafeMode {
		return ErrNotInSafeMode
	}

	s.inSafeMode = false
	s.logger.Printf("SAFETY: Exiting safe mode - resuming operations")

	if s.onExitSafe != nil {
		s.onExitSafe()
	}

	return nil
}

// IsInSafeMode returns true if currently in safe mode.
// Thread-safe.
func (s *SafeStateManager) IsInSafeMode() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inSafeMode
}

// GetSafePosition returns stub position data for the "Sit & Freeze" posture.
// In Phase 6, this is a stub. Actual kinematics will be implemented later.
func (s *SafeStateManager) GetSafePosition() map[string]interface{} {
	return map[string]interface{}{
		"state":       "sit_freeze",
		"height":      0.0,
		"legs_folded": true,
	}
}
