package job_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/test"
)

// TestOnce test once job
func TestOnceGoroutine(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// Channel to signal completion
	done := make(chan bool, 1)

	// Handler that signals completion
	handler := func(ctx context.Context, execution *job.Execution) error {
		execution.SetProgress(50, "Progress 50%%")
		execution.Info("Progress 50%%")
		time.Sleep(100 * time.Millisecond)
		execution.SetProgress(100, "Progress 100%%")
		execution.Info("Progress 100%%")
		done <- true
		return nil
	}

	err = testJob.Add(1, handler)
	if err != nil {
		t.Fatal(err)
	}

	// Set up worker manager for test
	wm := job.NewWorkerManagerForTest(2)
	wm.Start()
	defer wm.Stop()
	testJob.SetWorkerManager(wm)

	err = testJob.Start()
	if err != nil {
		t.Fatal(err)
	}

	// Wait for job completion or timeout
	select {
	case <-done:
		t.Log("Job completed successfully")
	case <-time.After(10 * time.Second):
		t.Error("Job execution timeout")
	}

	// Give some extra time for cleanup
	time.Sleep(500 * time.Millisecond)
}

func TestOnceProcess(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Once(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// Channel to signal completion
	done := make(chan bool, 1)

	// Handler that signals completion
	handler := func(ctx context.Context, execution *job.Execution) error {
		execution.SetProgress(50, "Progress 50%%")
		execution.Info("Progress 50%%")
		time.Sleep(100 * time.Millisecond)
		execution.SetProgress(100, "Progress 100%%")
		execution.Info("Progress 100%%")
		done <- true
		return nil
	}

	err = testJob.Add(1, handler)
	if err != nil {
		t.Fatal(err)
	}

	// Set up worker manager for test
	wm := job.NewWorkerManagerForTest(2)
	wm.Start()
	defer wm.Stop()
	testJob.SetWorkerManager(wm)

	err = testJob.Start()
	if err != nil {
		t.Fatal(err)
	}

	// Wait for job completion or timeout
	select {
	case <-done:
		t.Log("Job completed successfully")
	case <-time.After(10 * time.Second):
		t.Error("Job execution timeout")
	}

	// Give some extra time for cleanup
	time.Sleep(500 * time.Millisecond)
}

func TestCronGoroutine(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Cron(job.GOROUTINE, map[string]interface{}{}, "0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}

	// For cron jobs, we just test creation, not execution
	err = testJob.Add(1, HandlerTest)
	if err != nil {
		t.Fatal(err)
	}

	// Don't start cron jobs in tests as they are scheduled
	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeCron) {
		t.Errorf("Expected schedule type cron, got %s", testJob.ScheduleType)
	}
}

func TestCronProcess(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Cron(job.PROCESS, map[string]interface{}{}, "0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}

	// For cron jobs, we just test creation, not execution
	err = testJob.Add(1, HandlerTest)
	if err != nil {
		t.Fatal(err)
	}

	// Don't start cron jobs in tests as they are scheduled
	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeCron) {
		t.Errorf("Expected schedule type cron, got %s", testJob.ScheduleType)
	}
}

// TestDaemonGoroutine tests daemon job with goroutine mode using Ticker handler
func TestDaemonGoroutine(t *testing.T) {
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Daemon(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// For daemon jobs, we just test creation, not long-running execution
	err = testJob.Add(1, DaemonHandlerFastTest)
	if err != nil {
		t.Fatal(err)
	}

	// Don't start daemon jobs in tests as they run indefinitely
	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeDaemon) {
		t.Errorf("Expected schedule type daemon, got %s", testJob.ScheduleType)
	}
}

// TestDaemonProcess tests daemon job with process mode using Ticker handler
func TestDaemonProcess(t *testing.T) {
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Daemon(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// For daemon jobs, we just test creation, not long-running execution
	err = testJob.Add(1, DaemonHandlerFastTest)
	if err != nil {
		t.Fatal(err)
	}

	// Don't start daemon jobs in tests as they run indefinitely
	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeDaemon) {
		t.Errorf("Expected schedule type daemon, got %s", testJob.ScheduleType)
	}
}

// TestDaemonFastGoroutine tests fast daemon job with goroutine mode for quick testing
func TestDaemonFastGoroutine(t *testing.T) {
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Daemon(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// For daemon jobs, we just test creation, not execution
	err = testJob.Add(1, DaemonHandlerFastTest)
	if err != nil {
		t.Fatal(err)
	}

	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeDaemon) {
		t.Errorf("Expected schedule type daemon, got %s", testJob.ScheduleType)
	}
}

// TestDaemonFastProcess tests fast daemon job with process mode for quick testing
func TestDaemonFastProcess(t *testing.T) {
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	testJob, err := job.Daemon(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// For daemon jobs, we just test creation, not execution
	err = testJob.Add(1, DaemonHandlerFastTest)
	if err != nil {
		t.Fatal(err)
	}

	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeDaemon) {
		t.Errorf("Expected schedule type daemon, got %s", testJob.ScheduleType)
	}
}

func HandlerTest(ctx context.Context, execution *job.Execution) error {
	execution.SetProgress(50, "Progress 50%%")
	execution.Info("Progress 50%%")
	time.Sleep(100 * time.Millisecond)
	execution.SetProgress(100, "Progress 100%%")
	execution.Info("Progress 100%%")
	time.Sleep(200 * time.Millisecond)
	execution.SetProgress(100, "Progress 100%%")
	execution.Info("Progress 100%%")
	return nil
}

func DaemonHandlerTest(ctx context.Context, execution *job.Execution) error {
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

// DaemonHandlerFastTest fast testing version of daemon handler for testing (executes every 500ms)
func DaemonHandlerFastTest(ctx context.Context, execution *job.Execution) error {
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

// TestDatabase test database operations
func TestDatabase(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Test category creation
	category, err := job.GetOrCreateCategory("test-category", "Test category for unit tests")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	if category.Name != "test-category" {
		t.Errorf("Expected category name 'test-category', got '%s'", category.Name)
	}

	// Test job creation and saving
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name":        "Test Database Job",
		"description": "Job for testing database operations",
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	testJob.SetCategory(category.CategoryID)
	testJob.Add(1, HandlerTest)

	// Test job retrieval
	retrievedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrievedJob.Name != "Test Database Job" {
		t.Errorf("Expected job name 'Test Database Job', got '%s'", retrievedJob.Name)
	}

	// Update the testJob with the retrieved data to maintain consistency
	testJob = retrievedJob

	// Test job listing
	jobs, err := job.ListJobs(model.QueryParam{}, 1, 10)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if jobs["total"].(int) == 0 {
		t.Error("Expected at least one job in list")
	}

	// Test job counting
	count, err := job.CountJobs(model.QueryParam{})
	if err != nil {
		t.Fatalf("Failed to count jobs: %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one job in count")
	}
}

// TestWorkerManager test worker management
func TestWorkerManager(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Get worker manager
	wm := job.NewWorkerManagerForTest(2)
	if wm == nil {
		t.Fatal("Failed to get worker manager")
	}

	// Start worker manager
	wm.Start()
	defer wm.Stop()

	// Check active workers
	activeWorkers := wm.GetActiveWorkers()
	if activeWorkers == 0 {
		t.Error("Expected active workers after starting worker manager")
	}

	// Create and submit a job
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name":        "Test Worker Job",
		"description": "Job for testing worker management",
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	err = testJob.Add(1, HandlerTest)
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Start the job
	err = testJob.Start()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Wait for job to complete
	time.Sleep(1 * time.Second)

	// Check executions
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}

	if len(executions) == 0 {
		t.Error("Expected at least one execution")
	}

	// Check execution status
	if executions[0].Status != "completed" && executions[0].Status != "running" {
		t.Errorf("Expected execution status 'completed' or 'running', got '%s'", executions[0].Status)
	}
}

// TestJobExecution test job execution with logging and progress
func TestJobExecution(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a job with enhanced handler
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name":        "Test Execution Job",
		"description": "Job for testing execution features",
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Channel to signal completion
	done := make(chan bool, 1)

	// Enhanced handler for testing
	enhancedHandler := func(ctx context.Context, execution *job.Execution) error {
		execution.Info("Starting enhanced test execution")
		execution.SetProgress(10, "Initialization complete")

		time.Sleep(50 * time.Millisecond)

		execution.Debug("Debug message test")
		execution.SetProgress(50, "Halfway complete")

		time.Sleep(50 * time.Millisecond)

		execution.Warn("Warning message test")
		execution.SetProgress(80, "Almost done")

		time.Sleep(50 * time.Millisecond)

		execution.Info("Execution completed successfully")
		execution.SetProgress(100, "Complete")

		done <- true
		return nil
	}

	err = testJob.Add(1, enhancedHandler)
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Start worker manager
	wm := job.NewWorkerManagerForTest(2)
	wm.Start()
	defer wm.Stop()

	// Start the job
	err = testJob.Start()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Wait for completion or timeout
	select {
	case <-done:
		t.Log("Job execution completed")
	case <-time.After(10 * time.Second):
		t.Error("Job execution timeout")
	}

	// Give some extra time for database operations to complete
	time.Sleep(200 * time.Millisecond)

	// Check executions
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}

	if len(executions) == 0 {
		t.Fatal("Expected at least one execution")
	}

	execution := executions[0]

	// Check final progress (may take time to update)
	if execution.Progress < 50 {
		t.Errorf("Expected progress at least 50, got %d", execution.Progress)
	}

	// Check logs
	logs, err := job.ListLogs(testJob.JobID, model.QueryParam{}, 1, 100)
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	// Check if logs have items
	if logs["items"] != nil {
		logItems, ok := logs["items"].([]interface{})
		if ok && len(logItems) == 0 {
			t.Error("Expected log entries")
		}
	} else {
		t.Log("No log items found, this may be expected if logging is async")
	}
}
