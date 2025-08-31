package job_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/job"
)

// TestOnce test once job
func TestOnceGoroutine(t *testing.T) {
	test, err := job.Once(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestHandler)
	test.Start()
}

func TestOnceProcess(t *testing.T) {
	test, err := job.Once(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestHandler)
	test.Start()
}

func TestCronGoroutine(t *testing.T) {
	test, err := job.Cron(job.GOROUTINE, map[string]interface{}{}, "0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestHandler)
	test.Start()
}

func TestCronProcess(t *testing.T) {
	test, err := job.Cron(job.PROCESS, map[string]interface{}{}, "0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestHandler)
	test.Start()
}

// TestDaemonGoroutine tests daemon job with goroutine mode using Ticker handler
func TestDaemonGoroutine(t *testing.T) {
	test, err := job.Daemon(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestDaemonHandler)
	test.Start()
}

// TestDaemonProcess tests daemon job with process mode using Ticker handler
func TestDaemonProcess(t *testing.T) {
	test, err := job.Daemon(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestDaemonHandler)
	test.Start()
}

// TestDaemonFastGoroutine tests fast daemon job with goroutine mode for quick testing
func TestDaemonFastGoroutine(t *testing.T) {
	test, err := job.Daemon(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestDaemonHandlerFast)
	test.Start()
}

// TestDaemonFastProcess tests fast daemon job with process mode for quick testing
func TestDaemonFastProcess(t *testing.T) {
	test, err := job.Daemon(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	test.Add(1, TestDaemonHandlerFast)
	test.Start()
}

func TestHandler(ctx context.Context, execution *job.Execution) error {
	execution.SetProgress(50, "Progress 50%")
	execution.Info("Progress 50%")
	time.Sleep(100 * time.Millisecond)
	execution.SetProgress(100, "Progress 100%")
	execution.Info("Progress 100%")
	time.Sleep(200 * time.Millisecond)
	execution.SetProgress(100, "Progress 100%")
	execution.Info("Progress 100%")
	return nil
}

func TestDaemonHandler(ctx context.Context, execution *job.Execution) error {
	// Build a daemon handler using Ticker that executes tasks every 5 seconds continuously
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	counter := 0

	execution.Info("Daemon handler started, running continuously...")
	execution.SetProgress(0, "Daemon initialized and ready")

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, graceful shutdown
			execution.Info("Daemon handler received cancellation signal after %d iterations, exiting...", counter)
			execution.SetProgress(100, fmt.Sprintf("Daemon stopped gracefully after %d iterations", counter))
			return ctx.Err()
		case <-ticker.C:
			counter++

			// Daemon doesn't need specific completion progress, show running status instead
			execution.SetProgress(50, fmt.Sprintf("Running - completed %d iterations", counter))
			execution.Info("Daemon tick %d: Processing periodic task...", counter)

			// Simulate periodic tasks execution
			// e.g.: cleanup temp files, health checks, data synchronization, etc.
			time.Sleep(500 * time.Millisecond) // Simulate task execution time

			execution.Debug("Daemon iteration %d completed successfully", counter)

			// Output statistics every 10 iterations
			if counter%10 == 0 {
				execution.Info("Daemon health check: %d iterations completed, still running...", counter)
			}
		}
	}
}

// TestDaemonHandlerFast fast testing version of daemon handler for testing (executes every 500ms)
func TestDaemonHandlerFast(ctx context.Context, execution *job.Execution) error {
	// Use shorter interval for testing
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	counter := 0

	execution.Info("Fast daemon handler started for testing, running continuously...")
	execution.SetProgress(0, "Fast daemon initialized")

	for {
		select {
		case <-ctx.Done():
			execution.Info("Fast daemon handler stopped after %d iterations", counter)
			execution.SetProgress(100, fmt.Sprintf("Fast daemon stopped after %d iterations", counter))
			return ctx.Err()
		case <-ticker.C:
			counter++

			execution.SetProgress(50, fmt.Sprintf("Fast daemon: %d iterations", counter))
			execution.Debug("Fast daemon tick %d: Quick task execution", counter)

			// Quick task simulation
			time.Sleep(50 * time.Millisecond)

			// Output info every 5 iterations (due to higher frequency)
			if counter%5 == 0 {
				execution.Info("Fast daemon: %d iterations completed", counter)
			}
		}
	}
}
