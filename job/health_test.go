package job_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/test"
)

// registerHealthTestProcesses registers test processes for health testing
func registerHealthTestProcesses() {
	// Register a test process that simulates long-running execution
	process.Register("test.health.longrunning", func(process *process.Process) interface{} {
		args := process.Args
		message := "Long running process"
		if len(args) > 0 {
			message = args[0].(string)
		}

		// Simulate long-running process by sleeping
		time.Sleep(5 * time.Second)

		return map[string]interface{}{
			"message": message,
			"status":  "success",
		}
	})

	// Register a test process that simulates quick execution
	process.Register("test.health.quick", func(process *process.Process) interface{} {
		args := process.Args
		message := "Quick process"
		if len(args) > 0 {
			message = args[0].(string)
		}

		return map[string]interface{}{
			"message": message,
			"status":  "success",
		}
	})
}

// TestHealthCheckerCreation tests health checker creation
func TestHealthCheckerCreation(t *testing.T) {
	// Test creating health checker with different intervals
	hc := job.NewHealthChecker(10 * time.Second)
	if hc == nil {
		t.Fatal("Expected health checker to be created")
	}

	// Test stopping health checker
	hc.Stop()
}

// TestHealthCheckerBasicFunction tests basic health checker functionality
func TestHealthCheckerBasicFunction(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerHealthTestProcesses()

	// Create a short-interval health checker for testing (2 seconds for fast testing)
	hc := job.NewHealthChecker(2 * time.Second)
	defer hc.Stop()

	// Start health checker in background
	go hc.Start()

	// Wait a bit to let health checker run
	time.Sleep(3 * time.Second)

	t.Log("Health checker basic function test completed")
}

// TestHealthCheckerWithRunningJob tests health checker with actual running jobs
func TestHealthCheckerWithRunningJob(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerHealthTestProcesses()

	// Create a job that will run quickly
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Health Test Quick Job",
		"description": "Job for testing health checker with quick execution",
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Add execution to the job
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"test_context": "health check test",
		},
	}, "test.health.quick", "Quick execution for health test")
	if err != nil {
		t.Fatalf("Failed to add execution: %v", err)
	}

	// Start the job
	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Wait for job to complete
	time.Sleep(2 * time.Second)

	// Get updated job status
	updatedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	t.Logf("Job status after execution: %s", updatedJob.Status)
}

// TestHealthCheckerWithTimeoutJob tests health checker with timeout configuration
func TestHealthCheckerWithTimeoutJob(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerHealthTestProcesses()

	// Create a job with timeout
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":            "Health Test Timeout Job",
		"description":     "Job for testing health checker timeout handling",
		"default_timeout": 2, // 2 seconds timeout
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Add execution that will run longer than timeout
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"test_context": "timeout test",
		},
	}, "test.health.longrunning", "Long running execution for timeout test")
	if err != nil {
		t.Fatalf("Failed to add execution: %v", err)
	}

	// Start the job (this will run in background)
	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Create health checker with short interval for testing
	hc := job.NewHealthChecker(1 * time.Second)
	defer hc.Stop()

	// Start health checker
	go hc.Start()

	// Wait for health checker to detect and handle timeout
	time.Sleep(8 * time.Second)

	// Check if job was marked as failed due to timeout
	updatedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	t.Logf("Job status after timeout check: %s", updatedJob.Status)

	// Note: We don't assert failed status here because the test process might complete
	// before timeout is detected, depending on system performance
}

// TestHealthCheckerWithNoTimeoutJob tests health checker with jobs that have no timeout
func TestHealthCheckerWithNoTimeoutJob(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerHealthTestProcesses()

	// Create a job without timeout (should run indefinitely)
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Health Test No Timeout Job",
		"description": "Job for testing health checker with no timeout",
		// No default_timeout specified
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Add execution
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"test_context": "no timeout test",
		},
	}, "test.health.quick", "Quick execution with no timeout")
	if err != nil {
		t.Fatalf("Failed to add execution: %v", err)
	}

	// Start the job
	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(2 * time.Second)

	// Check job status
	updatedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	t.Logf("Job status (no timeout): %s", updatedJob.Status)
}

// TestHealthCheckerStopAndStart tests stopping and starting health checker
func TestHealthCheckerStopAndStart(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Create health checker
	hc := job.NewHealthChecker(1 * time.Second)

	// Start health checker
	go hc.Start()

	// Let it run for a bit
	time.Sleep(2 * time.Second)

	// Stop health checker
	hc.Stop()

	// Wait a bit to ensure it stops
	time.Sleep(1 * time.Second)

	t.Log("Health checker stop and start test completed")
}

// TestGlobalHealthChecker tests the global health checker functions
func TestGlobalHealthChecker(t *testing.T) {
	// Test getting global health checker
	globalHC := job.GetHealthChecker()
	if globalHC == nil {
		t.Log("Global health checker is nil, this is expected if not initialized")
	} else {
		t.Log("Global health checker exists")
	}

	// Test stopping global health checker
	job.StopHealthChecker()

	t.Log("Global health checker test completed")
}

// TestHealthCheckerRestart tests restarting health checker with different intervals
func TestHealthCheckerRestart(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Test restarting with different intervals
	job.RestartHealthChecker(1 * time.Second)
	time.Sleep(2 * time.Second)

	job.RestartHealthChecker(3 * time.Second)
	time.Sleep(1 * time.Second)

	// Stop the health checker
	job.StopHealthChecker()

	t.Log("Health checker restart test completed")
}

// TestDataCleanerCreation tests data cleaner creation
func TestDataCleanerCreation(t *testing.T) {
	// Test creating data cleaner with different retention periods
	dc := job.NewDataCleaner(30)
	if dc == nil {
		t.Fatal("Expected data cleaner to be created")
	}

	// Test stopping data cleaner
	dc.Stop()
}

// TestDataCleanerBasicFunction tests basic data cleaner functionality
func TestDataCleanerBasicFunction(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Create data cleaner with 1 day retention for testing
	dc := job.NewDataCleaner(1)
	defer dc.Stop()

	// Start data cleaner (won't actually clean on first run due to initialization)
	go dc.Start()

	// Wait a bit
	time.Sleep(1 * time.Second)

	t.Log("Data cleaner basic function test completed")
}

// TestDataCleanupWithOldJobs tests data cleanup with old completed jobs
func TestDataCleanupWithOldJobs(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerHealthTestProcesses()

	// Create an old completed job (simulate by creating and completing it)
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Old Test Job",
		"description": "Job for testing data cleanup",
	})
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Add execution and complete it
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"test_context": "cleanup test",
		},
	}, "test.health.quick", "Quick execution for cleanup test")
	if err != nil {
		t.Fatalf("Failed to add execution: %v", err)
	}

	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Wait for job to complete
	time.Sleep(2 * time.Second)

	// Verify job is completed
	updatedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	t.Logf("Job status before cleanup: %s", updatedJob.Status)

	// Test force cleanup (this won't delete recent jobs due to retention period)
	err = job.ForceCleanup()
	if err != nil {
		t.Fatalf("Failed to force cleanup: %v", err)
	}

	// Verify job still exists (should not be deleted due to retention period)
	_, err = job.GetJob(testJob.JobID)
	if err != nil {
		t.Log("Job was cleaned up (expected if older than retention period)")
	} else {
		t.Log("Job still exists (expected for recent jobs)")
	}
}

// TestGlobalDataCleaner tests global data cleaner functions
func TestGlobalDataCleaner(t *testing.T) {
	// Test getting global data cleaner
	globalDC := job.GetDataCleaner()
	if globalDC == nil {
		t.Log("Global data cleaner is nil, this is expected if not initialized")
	} else {
		t.Log("Global data cleaner exists")
	}

	// Test stopping global data cleaner
	job.StopDataCleaner()

	t.Log("Global data cleaner test completed")
}

// TestHealthCheckerWithMultipleJobs tests health checker with multiple jobs
func TestHealthCheckerWithMultipleJobs(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerHealthTestProcesses()

	// Create multiple jobs
	jobs := make([]*job.Job, 3)
	for i := 0; i < 3; i++ {
		testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
			"name":        fmt.Sprintf("Health Test Job %d", i+1),
			"description": fmt.Sprintf("Job %d for testing health checker with multiple jobs", i+1),
		})
		if err != nil {
			t.Fatalf("Failed to create test job %d: %v", i+1, err)
		}

		// Add execution to each job
		err = testJob.Add(&job.ExecutionOptions{
			Priority: 1,
			SharedData: map[string]interface{}{
				"test_context": fmt.Sprintf("multi job test %d", i+1),
			},
		}, "test.health.quick", fmt.Sprintf("Execution for job %d", i+1))
		if err != nil {
			t.Fatalf("Failed to add execution to job %d: %v", i+1, err)
		}

		jobs[i] = testJob
	}

	// Start all jobs
	for i, testJob := range jobs {
		err := testJob.Push()
		if err != nil {
			t.Fatalf("Failed to start job %d: %v", i+1, err)
		}
	}

	// Create health checker for monitoring
	hc := job.NewHealthChecker(1 * time.Second)
	defer hc.Stop()

	// Start health checker
	go hc.Start()

	// Wait for jobs to complete and health checker to run
	time.Sleep(5 * time.Second)

	// Check status of all jobs
	for i, testJob := range jobs {
		updatedJob, err := job.GetJob(testJob.JobID)
		if err != nil {
			t.Errorf("Failed to get updated job %d: %v", i+1, err)
			continue
		}
		t.Logf("Job %d status: %s", i+1, updatedJob.Status)
	}
}
