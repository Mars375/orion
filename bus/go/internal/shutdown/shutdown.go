package shutdown

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Coordinator handles graceful shutdown with timeout.
//
// Implements graceful shutdown pattern from RESEARCH.md:
// - Shutdown timeout < Kubernetes terminationGracePeriodSeconds (30s default)
// - Use 25s timeout to allow 5s buffer for SIGKILL
// - Propagate context cancellation to all cleanup functions
type Coordinator struct {
	timeout time.Duration
	logger  *log.Logger
}

// NewCoordinator creates a new shutdown coordinator.
//
// Parameters:
//   - timeout: Maximum time to wait for cleanup functions (default: 25s)
//   - logger: Logger for shutdown events
//
// Returns initialized Coordinator ready for WaitForShutdown.
func NewCoordinator(timeout time.Duration, logger *log.Logger) *Coordinator {
	if timeout == 0 {
		timeout = 25 * time.Second // Default: 25s (5s buffer before K8s SIGKILL at 30s)
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Coordinator{
		timeout: timeout,
		logger:  logger,
	}
}

// WaitForShutdown blocks until context is cancelled, then executes cleanup functions.
//
// This function waits for the provided context to be cancelled (typically by
// signal.NotifyContext on SIGTERM/SIGINT), then executes all cleanup functions
// with a timeout context. Cleanup functions are executed sequentially in the
// order provided.
//
// Parameters:
//   - ctx: Context to wait on (typically from signal.NotifyContext)
//   - cleanupFuncs: Functions to execute during shutdown (e.g., bus.Stop, server.Shutdown)
//
// Returns:
//   - nil if all cleanup functions succeed
//   - error if any cleanup function fails or timeout occurs
//
// Example:
//
//	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
//	defer stop()
//
//	coordinator := NewCoordinator(25*time.Second, logger)
//	err := coordinator.WaitForShutdown(ctx,
//	    func(ctx context.Context) error { return bus.Stop(ctx) },
//	    func(ctx context.Context) error { return server.Shutdown(ctx) },
//	)
func (c *Coordinator) WaitForShutdown(ctx context.Context, cleanupFuncs ...func(context.Context) error) error {
	// Wait for shutdown signal
	<-ctx.Done()
	c.logger.Println("INFO: Shutdown signal received, starting graceful shutdown")

	// Create timeout context for cleanup
	cleanupCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// Execute cleanup functions
	var errors []error
	for i, cleanupFunc := range cleanupFuncs {
		c.logger.Printf("INFO: Executing cleanup function %d/%d", i+1, len(cleanupFuncs))
		if err := cleanupFunc(cleanupCtx); err != nil {
			c.logger.Printf("ERROR: Cleanup function %d failed: %v", i+1, err)
			errors = append(errors, fmt.Errorf("cleanup %d: %w", i+1, err))
		}
	}

	// Check for timeout
	if cleanupCtx.Err() == context.DeadlineExceeded {
		c.logger.Printf("ERROR: Shutdown timeout exceeded (%v)", c.timeout)
		errors = append(errors, fmt.Errorf("shutdown timeout exceeded: %w", cleanupCtx.Err()))
	}

	// Log completion
	if len(errors) == 0 {
		c.logger.Println("INFO: Graceful shutdown completed successfully")
		return nil
	}

	c.logger.Printf("ERROR: Graceful shutdown completed with %d error(s)", len(errors))
	return fmt.Errorf("shutdown errors: %v", errors)
}
