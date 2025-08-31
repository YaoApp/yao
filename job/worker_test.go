package job_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/test"
)

// TestWorkerManagerLifecycle tests worker manager lifecycle
func TestWorkerManagerLifecycle(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a new worker manager for testing (not singleton)
	wm := job.NewWorkerManagerForTest(2)
	if wm == nil {
		t.Fatal("Failed to create worker manager")
	}

	// Test initial state
	if wm.GetActiveWorkers() != 0 {
		t.Error("Expected 0 active workers initially")
	}

	// Test start
	wm.Start()
	if wm.GetActiveWorkers() == 0 {
		t.Error("Expected active workers after start")
	}

	// Test stop
	wm.Stop()
	// Note: Workers might still be active briefly after stop due to cleanup time
	time.Sleep(100 * time.Millisecond)

	// Test restart
	wm.Start()
	if wm.GetActiveWorkers() == 0 {
		t.Error("Expected active workers after restart")
	}

	// Final cleanup
	wm.Stop()
}

// TestWorkerJobSubmission tests job submission to workers
func TestWorkerJobSubmission(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test job
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name": "Test Worker Submission Job",
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Test handler
	executed := make(chan bool, 1)
	testHandler := func(ctx context.Context, execution *job.Execution) error {
		execution.Info("Test handler executed")
		executed <- true
		return nil
	}

	// Add handler and save job
	err = testJob.Add(1, testHandler)
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Create worker manager for testing
	wm := job.NewWorkerManagerForTest(2)
	wm.Start()
	defer wm.Stop()

	// Submit job
	err = wm.SubmitJob(testJob, testHandler)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for execution
	select {
	case <-executed:
		t.Log("Job executed successfully")
	case <-time.After(5 * time.Second):
		t.Error("Job execution timeout")
	}

	// Check executions
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) == 0 {
		t.Error("Expected at least one execution")
	}
}

// TestWorkerModes tests different worker execution modes
func TestWorkerModes(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	wm := job.NewWorkerManagerForTest(4)
	wm.Start()
	defer wm.Stop()

	// Test GOROUTINE mode
	goroutineJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name": "Test Goroutine Mode Job",
	})
	if err != nil {
		t.Fatalf("Failed to create goroutine job: %v", err)
	}

	goroutineExecuted := make(chan bool, 1)
	goroutineHandler := func(ctx context.Context, execution *job.Execution) error {
		if execution.Job != nil && execution.Job.Mode != job.GOROUTINE {
			t.Errorf("Expected GOROUTINE mode, got %v", execution.Job.Mode)
		}
		goroutineExecuted <- true
		return nil
	}

	err = goroutineJob.Add(1, goroutineHandler)
	if err != nil {
		t.Fatalf("Failed to add goroutine handler: %v", err)
	}

	goroutineJob.SetWorkerManager(wm)

	err = goroutineJob.Start()
	if err != nil {
		t.Fatalf("Failed to start goroutine job: %v", err)
	}

	// Test PROCESS mode
	processJob, err := job.Once(job.PROCESS, map[string]interface{}{
		"name": "Test Process Mode Job",
	})
	if err != nil {
		t.Fatalf("Failed to create process job: %v", err)
	}

	processExecuted := make(chan bool, 1)
	processHandler := func(ctx context.Context, execution *job.Execution) error {
		if execution.Job != nil && execution.Job.Mode != job.PROCESS {
			t.Errorf("Expected PROCESS mode, got %v", execution.Job.Mode)
		}
		processExecuted <- true
		return nil
	}

	err = processJob.Add(1, processHandler)
	if err != nil {
		t.Fatalf("Failed to add process handler: %v", err)
	}

	processJob.SetWorkerManager(wm)

	err = processJob.Start()
	if err != nil {
		t.Fatalf("Failed to start process job: %v", err)
	}

	// Wait for both executions
	timeout := time.After(10 * time.Second)
	goroutineDone := false
	processDone := false

	for !goroutineDone || !processDone {
		select {
		case <-goroutineExecuted:
			goroutineDone = true
			t.Log("Goroutine job executed successfully")
		case <-processExecuted:
			processDone = true
			t.Log("Process job executed successfully")
		case <-timeout:
			t.Error("Jobs execution timeout")
			return
		}
	}

	// Give extra time for database operations to complete
	time.Sleep(500 * time.Millisecond)
}

// TestWorkerErrorHandling tests worker error handling
func TestWorkerErrorHandling(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test job
	errorJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name": "Test Error Handling Job",
	})
	if err != nil {
		t.Fatalf("Failed to create error job: %v", err)
	}

	// Error handler
	errorHandler := func(ctx context.Context, execution *job.Execution) error {
		execution.Error("Test error occurred")
		return fmt.Errorf("intentional test error")
	}

	err = errorJob.Add(1, errorHandler)
	if err != nil {
		t.Fatalf("Failed to add error handler: %v", err)
	}

	wm := job.NewWorkerManagerForTest(4)
	wm.Start()
	defer wm.Stop()
	errorJob.SetWorkerManager(wm)

	err = errorJob.Start()
	if err != nil {
		t.Fatalf("Failed to start error job: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(2 * time.Second)

	// Give extra time for database operations to complete
	time.Sleep(500 * time.Millisecond)

	// Check execution status
	executions, err := errorJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) == 0 {
		t.Error("Expected at least one execution")
		return
	}

	execution := executions[0]
	if execution.Status != "failed" {
		t.Errorf("Expected execution status 'failed', got '%s'", execution.Status)
	}

	if execution.ErrorInfo == nil {
		t.Error("Expected error info to be set")
	}
}

// TestWorkerConcurrency tests worker concurrency
func TestWorkerConcurrency(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	wm := job.NewWorkerManagerForTest(4)
	wm.Start()
	defer wm.Stop()

	const numJobs = 5
	executed := make(chan bool, numJobs)

	// Create and submit multiple jobs concurrently
	for i := 0; i < numJobs; i++ {
		testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
			"name": fmt.Sprintf("Concurrent Test Job %d", i+1),
		})
		if err != nil {
			t.Fatalf("Failed to create test job %d: %v", i+1, err)
		}

		jobID := i + 1
		testHandler := func(ctx context.Context, execution *job.Execution) error {
			execution.Info("Concurrent job %d executed", jobID)
			time.Sleep(100 * time.Millisecond) // Simulate work
			executed <- true
			return nil
		}

		err = testJob.Add(1, testHandler)
		if err != nil {
			t.Fatalf("Failed to add handler for job %d: %v", i+1, err)
		}

		testJob.SetWorkerManager(wm)

		err = testJob.Start()
		if err != nil {
			t.Fatalf("Failed to start job %d: %v", i+1, err)
		}
	}

	// Wait for all jobs to complete
	completed := 0
	timeout := time.After(10 * time.Second)

	for completed < numJobs {
		select {
		case <-executed:
			completed++
			t.Logf("Job %d completed", completed)
		case <-timeout:
			t.Errorf("Timeout: only %d out of %d jobs completed", completed, numJobs)
			return
		}
	}

	if completed != numJobs {
		t.Errorf("Expected %d jobs to complete, got %d", numJobs, completed)
	}

	// Give extra time for database operations to complete
	time.Sleep(500 * time.Millisecond)
}
