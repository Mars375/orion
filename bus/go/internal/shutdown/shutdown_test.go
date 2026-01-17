package shutdown

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForShutdown_AllCleanupSucceeds(t *testing.T) {
	t.Parallel()

	// Create coordinator with short timeout for testing
	coordinator := NewCoordinator(5*time.Second, log.Default())

	// Create context that we'll cancel to simulate shutdown signal
	ctx, cancel := context.WithCancel(context.Background())

	// Track which cleanup functions were called
	var called []int
	cleanup1 := func(ctx context.Context) error {
		called = append(called, 1)
		return nil
	}
	cleanup2 := func(ctx context.Context) error {
		called = append(called, 2)
		return nil
	}
	cleanup3 := func(ctx context.Context) error {
		called = append(called, 3)
		return nil
	}

	// Start WaitForShutdown in background
	done := make(chan error, 1)
	go func() {
		done <- coordinator.WaitForShutdown(ctx, cleanup1, cleanup2, cleanup3)
	}()

	// Give it time to start waiting
	time.Sleep(10 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for completion
	err := <-done
	require.NoError(t, err)

	// Verify all cleanup functions were called in order
	assert.Equal(t, []int{1, 2, 3}, called)
}

func TestWaitForShutdown_CleanupTimeout(t *testing.T) {
	t.Parallel()

	// Create coordinator with very short timeout
	coordinator := NewCoordinator(100*time.Millisecond, log.Default())

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cleanup function that sleeps longer than timeout
	cleanupSlow := func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	// Start WaitForShutdown in background
	done := make(chan error, 1)
	go func() {
		done <- coordinator.WaitForShutdown(ctx, cleanupSlow)
	}()

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for completion
	err := <-done
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestWaitForShutdown_CleanupError(t *testing.T) {
	t.Parallel()

	// Create coordinator
	coordinator := NewCoordinator(5*time.Second, log.Default())

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cleanup function that returns an error
	expectedError := errors.New("cleanup failed")
	cleanupFail := func(ctx context.Context) error {
		return expectedError
	}

	// Start WaitForShutdown in background
	done := make(chan error, 1)
	go func() {
		done <- coordinator.WaitForShutdown(ctx, cleanupFail)
	}()

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for completion
	err := <-done
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup failed")
}

func TestWaitForShutdown_MultipleErrorsCollected(t *testing.T) {
	t.Parallel()

	// Create coordinator
	coordinator := NewCoordinator(5*time.Second, log.Default())

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Multiple cleanup functions that fail
	cleanup1 := func(ctx context.Context) error {
		return errors.New("error 1")
	}
	cleanup2 := func(ctx context.Context) error {
		return errors.New("error 2")
	}

	// Start WaitForShutdown in background
	done := make(chan error, 1)
	go func() {
		done <- coordinator.WaitForShutdown(ctx, cleanup1, cleanup2)
	}()

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for completion
	err := <-done
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error 1")
	assert.Contains(t, err.Error(), "error 2")
}

func TestNewCoordinator_DefaultTimeout(t *testing.T) {
	t.Parallel()

	// Create coordinator with zero timeout (should use default)
	coordinator := NewCoordinator(0, nil)

	// Verify default timeout is 25 seconds
	assert.Equal(t, 25*time.Second, coordinator.timeout)
}

func TestNewCoordinator_DefaultLogger(t *testing.T) {
	t.Parallel()

	// Create coordinator with nil logger (should use default)
	coordinator := NewCoordinator(1*time.Second, nil)

	// Verify logger is not nil
	assert.NotNil(t, coordinator.logger)
}
