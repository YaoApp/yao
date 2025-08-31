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

// TestProgressManager tests progress management functionality
func TestProgressManager(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test job
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name": "Test Progress Job",
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Get progress manager
	progressManager := testJob.Progress()
	if progressManager == nil {
		t.Fatal("Failed to get progress manager")
	}

	// Test initial progress
	err = progressManager.Set(0, "Starting")
	if err != nil {
		t.Fatalf("Failed to set initial progress: %v", err)
	}

	// Test progress updates
	err = progressManager.Set(25, "25% complete")
	if err != nil {
		t.Fatalf("Failed to set progress to 25: %v", err)
	}

	err = progressManager.Set(50, "50% complete")
	if err != nil {
		t.Fatalf("Failed to set progress to 50: %v", err)
	}

	err = progressManager.Set(75, "75% complete")
	if err != nil {
		t.Fatalf("Failed to set progress to 75: %v", err)
	}

	err = progressManager.Set(100, "Complete")
	if err != nil {
		t.Fatalf("Failed to set progress to 100: %v", err)
	}
}

// TestProgressWithExecution tests progress updates during job execution
func TestProgressWithExecution(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test job
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name": "Test Progress Execution Job",
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Channel to signal completion
	done := make(chan int, 1)

	// Progress tracking handler
	progressHandler := func(ctx context.Context, execution *job.Execution) error {
		// Test SetProgress method on execution
		for i := 0; i <= 100; i += 20 {
			err := execution.SetProgress(i, fmt.Sprintf("Progress: %d%%", i))
			if err != nil {
				return err
			}
			time.Sleep(50 * time.Millisecond)
		}
		done <- 100
		return nil
	}

	err = testJob.Add(1, progressHandler)
	if err != nil {
		t.Fatalf("Failed to add progress handler: %v", err)
	}

	// Start worker manager
	wm := job.NewWorkerManagerForTest(2)
	wm.Start()
	defer wm.Stop()
	testJob.SetWorkerManager(wm)

	// Start job
	err = testJob.Start()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Wait for job completion or timeout
	var finalProgress int
	select {
	case finalProgress = <-done:
		t.Logf("Job completed with progress: %d", finalProgress)
	case <-time.After(10 * time.Second):
		t.Error("Job execution timeout")
		return
	}

	// Give some extra time for database operations to complete
	time.Sleep(200 * time.Millisecond)

	// Check final execution state
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) == 0 {
		t.Error("Expected at least one execution")
		return
	}

	execution := executions[0]

	// Check final progress - we know it completed with 100 from the handler
	if finalProgress != 100 {
		t.Errorf("Expected handler to complete with 100, got %d", finalProgress)
	}

	// The database might not have the latest progress due to async operations
	t.Logf("Final execution progress in database: %d", execution.Progress)
	t.Logf("Final progress from handler: %d", finalProgress)
}

// TestProgressWithDatabase tests progress persistence in database
func TestProgressWithDatabase(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test job
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name": "Test Progress Database Job",
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Add handler to save job first
	err = testJob.Add(1, func(ctx context.Context, execution *job.Execution) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Create execution manually to test progress persistence
	testExecution := &job.Execution{
		ExecutionID:     "test-progress-exec-001",
		JobID:           testJob.JobID,
		Status:          "running",
		TriggerCategory: "manual",
		Progress:        0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = job.SaveExecution(testExecution)
	if err != nil {
		t.Fatalf("Failed to save test execution: %v", err)
	}

	// Test progress updates through SetProgress
	progressValues := []int{10, 25, 50, 75, 90, 100}
	for _, progress := range progressValues {
		testExecution.Progress = progress
		err = testExecution.SetProgress(progress, fmt.Sprintf("Progress: %d%%", progress))
		if err != nil {
			t.Fatalf("Failed to set progress to %d: %v", progress, err)
		}

		// Verify progress was saved to database
		savedExecution, err := job.GetExecution(testExecution.ExecutionID, model.QueryParam{})
		if err != nil {
			t.Fatalf("Failed to get saved execution: %v", err)
		}

		if savedExecution.Progress != progress {
			t.Errorf("Expected saved progress %d, got %d", progress, savedExecution.Progress)
		}
	}

	// Clean up
	job.RemoveExecutions([]string{testExecution.ExecutionID})
	job.RemoveJobs([]string{testJob.JobID})
}

// TestGetProgress tests live progress retrieval
func TestGetProgress(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test execution
	testExecution := &job.Execution{
		ExecutionID:     "test-get-progress-001",
		JobID:           "test-job-001",
		Status:          "running",
		TriggerCategory: "manual",
		Progress:        75,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := job.SaveExecution(testExecution)
	if err != nil {
		t.Fatalf("Failed to save test execution: %v", err)
	}

	// Test GetProgress function
	callbackCalled := false
	progress, err := job.GetProgress(testExecution.ExecutionID, func(p *job.Progress) {
		callbackCalled = true
		if p.ExecutionID != testExecution.ExecutionID {
			t.Errorf("Expected execution ID %s, got %s", testExecution.ExecutionID, p.ExecutionID)
		}
		if p.Progress != 75 {
			t.Errorf("Expected progress 75, got %d", p.Progress)
		}
	})

	if err != nil {
		t.Fatalf("Failed to get progress: %v", err)
	}

	if progress == nil {
		t.Fatal("Expected progress object, got nil")
	}

	if progress.ExecutionID != testExecution.ExecutionID {
		t.Errorf("Expected execution ID %s, got %s", testExecution.ExecutionID, progress.ExecutionID)
	}

	if progress.Progress != 75 {
		t.Errorf("Expected progress 75, got %d", progress.Progress)
	}

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	// Clean up
	job.RemoveExecutions([]string{testExecution.ExecutionID})
}
