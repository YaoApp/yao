package job_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/test"
)

// registerTestProcesses registers test processes for job testing
func registerTestProcesses() {
	// Register test.job.echo process
	process.Register("test.job.echo", func(process *process.Process) interface{} {
		args := process.Args
		if len(args) > 0 {
			message := args[0]

			// Simulate progress updates
			if process.Callback != nil {
				// Report 25% progress
				process.Callback(process, map[string]interface{}{
					"type":     "progress",
					"progress": 25,
					"message":  "Starting echo process",
				})

				// Report 50% progress
				process.Callback(process, map[string]interface{}{
					"type":     "progress",
					"progress": 50,
					"message":  "Processing message",
				})

				// Report 75% progress
				process.Callback(process, map[string]interface{}{
					"type":     "progress",
					"progress": 75,
					"message":  "Finalizing echo",
				})

				// Report 100% progress
				process.Callback(process, map[string]interface{}{
					"type":     "progress",
					"progress": 100,
					"message":  "Echo completed",
				})
			}

			return map[string]interface{}{
				"message": message,
				"echo":    "Echo: " + message.(string),
				"status":  "success",
			}
		}

		return map[string]interface{}{
			"message": "No message provided",
			"status":  "error",
		}
	})

	// Register test.job.cron process
	process.Register("test.job.cron", func(process *process.Process) interface{} {
		args := process.Args
		message := "Cron job executed"
		if len(args) > 0 {
			message = args[0].(string)
		}

		return map[string]interface{}{
			"message":   message,
			"timestamp": time.Now().Unix(),
			"status":    "success",
		}
	})

	// Register test.job.daemon process
	process.Register("test.job.daemon", func(process *process.Process) interface{} {
		args := process.Args
		message := "Daemon process executed"
		if len(args) > 0 {
			message = args[0].(string)
		}

		return map[string]interface{}{
			"message": message,
			"status":  "success",
			"daemon":  true,
		}
	})

	// Register test.job.database process
	process.Register("test.job.database", func(process *process.Process) interface{} {
		args := process.Args
		message := "Database operation executed"
		if len(args) > 0 {
			message = args[0].(string)
		}

		return map[string]interface{}{
			"message":   message,
			"operation": "test",
			"status":    "success",
		}
	})

	// Register test.job.execution process with enhanced features
	process.Register("test.job.execution", func(process *process.Process) interface{} {
		args := process.Args
		message := "Execution test"
		if len(args) > 0 {
			message = args[0].(string)
		}

		// Simulate progress updates with callback
		if process.Callback != nil {
			// Report progress incrementally
			for i := 10; i <= 100; i += 10 {
				process.Callback(process, map[string]interface{}{
					"type":     "progress",
					"progress": i,
					"message":  fmt.Sprintf("Processing step %d/10", i/10),
				})
				time.Sleep(10 * time.Millisecond) // Small delay to simulate work
			}
		}

		return map[string]interface{}{
			"message":   message,
			"progress":  100,
			"status":    "success",
			"test_data": "execution completed",
		}
	})
}

// TestOnce test once job
func TestOnceGoroutine(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// Use a test Yao process (this would need to be defined in your Yao app)
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"test_data": "Hello from test",
		},
	}, "test.job.echo", "Hello from test")
	if err != nil {
		t.Fatal(err)
	}

	err = testJob.Push()
	if err != nil {
		t.Fatal(err)
	}

	// Give some time for execution
	time.Sleep(2 * time.Second)

	t.Log("Job started successfully")
}

func TestOnceProcess(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	testJob, err := job.Once(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// Use a test Yao process
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"test_data": "Hello from process test",
		},
	}, "test.job.echo", "Hello from process test")
	if err != nil {
		t.Fatal(err)
	}

	err = testJob.Push()
	if err != nil {
		t.Fatal(err)
	}

	// Give some time for execution
	time.Sleep(2 * time.Second)

	t.Log("Process job started successfully")
}

func TestCronGoroutine(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	testJob, err := job.Cron(job.GOROUTINE, map[string]interface{}{}, "0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}

	// For cron jobs, we just test creation, not execution
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"cron_context": "scheduled execution",
		},
	}, "test.job.cron", "Cron test execution")
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

	// Register test processes
	registerTestProcesses()

	testJob, err := job.Cron(job.PROCESS, map[string]interface{}{}, "0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}

	// For cron jobs, we just test creation, not execution
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"cron_context": "scheduled process execution",
		},
	}, "test.job.cron", "Cron process test execution")
	if err != nil {
		t.Fatal(err)
	}

	// Don't start cron jobs in tests as they are scheduled
	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeCron) {
		t.Errorf("Expected schedule type cron, got %s", testJob.ScheduleType)
	}
}

// TestDaemonGoroutine tests daemon job with goroutine mode
func TestDaemonGoroutine(t *testing.T) {
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	testJob, err := job.Daemon(job.GOROUTINE, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// For daemon jobs, we just test creation, not long-running execution
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"daemon_context": "background service",
		},
	}, "test.job.daemon", "Daemon test execution")
	if err != nil {
		t.Fatal(err)
	}

	// Don't start daemon jobs in tests as they run indefinitely
	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeDaemon) {
		t.Errorf("Expected schedule type daemon, got %s", testJob.ScheduleType)
	}
}

// TestDaemonProcess tests daemon job with process mode
func TestDaemonProcess(t *testing.T) {
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	testJob, err := job.Daemon(job.PROCESS, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// For daemon jobs, we just test creation, not long-running execution
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"daemon_context": "background process service",
		},
	}, "test.job.daemon", "Daemon process test execution")
	if err != nil {
		t.Fatal(err)
	}

	// Don't start daemon jobs in tests as they run indefinitely
	// Just verify the job was created properly
	if testJob.ScheduleType != string(job.ScheduleTypeDaemon) {
		t.Errorf("Expected schedule type daemon, got %s", testJob.ScheduleType)
	}
}

// TestCommand test command execution
func TestCommand(t *testing.T) {
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name":        "Test Command Job",
		"description": "Job for testing command execution",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Test system command
	err = testJob.AddCommand(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"command_context": "test execution",
		},
	}, "echo", []string{"Hello from command test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = testJob.Push()
	if err != nil {
		t.Fatal(err)
	}

	// Give some time for execution
	time.Sleep(2 * time.Second)

	t.Log("Command job started successfully")
}

// TestDatabase test database operations
func TestDatabase(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

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
	testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"database_context": "test operation",
		},
	}, "test.job.database", "Database test execution")

	// Save the job to database before retrieving it
	err = job.SaveJob(testJob)
	if err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

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

// TestJobExecution test job execution with logging and progress
func TestJobExecution(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	// Create a job with enhanced handler
	testJob, err := job.Once(job.GOROUTINE, map[string]interface{}{
		"name":        "Test Execution Job",
		"description": "Job for testing execution features",
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Save the job to database first so it has a valid ID
	err = job.SaveJob(testJob)
	if err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Use a test Yao process for execution testing with chained options
	err = testJob.Add(
		job.NewExecutionOptions().
			WithPriority(1).
			AddSharedData("execution_context", "enhanced test").
			AddSharedData("user_id", "test_user_123").
			AddSharedData("session", map[string]interface{}{
				"token":   "test_token",
				"expires": "2024-12-31",
			}),
		"test.job.execution", "Enhanced execution test")
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Start the job
	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Give some time for execution
	time.Sleep(2 * time.Second)
	t.Log("Job execution started")

	// Check executions
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}

	if len(executions) == 0 {
		t.Fatal("Expected at least one execution")
	}

	execution := executions[0]
	t.Logf("Initial execution progress: %d", execution.Progress)

	// Get fresh execution data from database to check final progress
	freshExecution, err := job.GetExecution(execution.ExecutionID, model.QueryParam{})
	if err != nil {
		t.Fatalf("Failed to get fresh execution: %v", err)
	}

	t.Logf("Fresh execution progress: %d, status: %s", freshExecution.Progress, freshExecution.Status)
	if freshExecution.ErrorInfo != nil && len(*freshExecution.ErrorInfo) > 0 {
		t.Logf("Execution error: %s", string(*freshExecution.ErrorInfo))
	}
	if freshExecution.Result != nil && len(*freshExecution.Result) > 0 {
		t.Logf("Execution result: %s", string(*freshExecution.Result))
	}

	// Check final progress (may take time to update)
	if freshExecution.Progress < 50 {
		t.Errorf("Expected progress at least 50, got %d", freshExecution.Progress)
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

// TestOnceAndSave test OnceAndSave method
func TestOnceAndSave(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	// Test OnceAndSave - should create and save job in one step
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Test OnceAndSave Job",
		"description": "Job created and saved with OnceAndSave method",
	})
	if err != nil {
		t.Fatalf("Failed to create and save job: %v", err)
	}

	// Job should have a valid JobID after OnceAndSave
	if testJob.JobID == "" {
		t.Error("Expected job to have JobID after OnceAndSave")
	}

	// Verify job was saved to database
	retrievedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to retrieve saved job: %v", err)
	}

	if retrievedJob.Name != "Test OnceAndSave Job" {
		t.Errorf("Expected job name 'Test OnceAndSave Job', got '%s'", retrievedJob.Name)
	}

	// Add execution and push
	err = testJob.Add(&job.ExecutionOptions{
		Priority: 1,
		SharedData: map[string]interface{}{
			"test_data": "OnceAndSave test",
		},
	}, "test.job.echo", "OnceAndSave test")
	if err != nil {
		t.Fatal(err)
	}

	err = testJob.Push()
	if err != nil {
		t.Fatal(err)
	}

	// Give some time for execution
	time.Sleep(2 * time.Second)

	t.Log("OnceAndSave job completed successfully")
}

// TestCronAndSave test CronAndSave method
func TestCronAndSave(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	// Test CronAndSave - should create and save cron job in one step
	testJob, err := job.CronAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Test CronAndSave Job",
		"description": "Cron job created and saved with CronAndSave method",
	}, "0 0 * * *")
	if err != nil {
		t.Fatalf("Failed to create and save cron job: %v", err)
	}

	// Job should have a valid JobID after CronAndSave
	if testJob.JobID == "" {
		t.Error("Expected cron job to have JobID after CronAndSave")
	}

	// Verify cron job was saved to database
	retrievedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to retrieve saved cron job: %v", err)
	}

	if retrievedJob.ScheduleType != string(job.ScheduleTypeCron) {
		t.Errorf("Expected schedule type cron, got %s", retrievedJob.ScheduleType)
	}

	if retrievedJob.Name != "Test CronAndSave Job" {
		t.Errorf("Expected job name 'Test CronAndSave Job', got '%s'", retrievedJob.Name)
	}

	t.Log("CronAndSave job created and saved successfully")
}

// TestDaemonAndSave test DaemonAndSave method
func TestDaemonAndSave(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Register test processes
	registerTestProcesses()

	// Test DaemonAndSave - should create and save daemon job in one step
	testJob, err := job.DaemonAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Test DaemonAndSave Job",
		"description": "Daemon job created and saved with DaemonAndSave method",
	})
	if err != nil {
		t.Fatalf("Failed to create and save daemon job: %v", err)
	}

	// Job should have a valid JobID after DaemonAndSave
	if testJob.JobID == "" {
		t.Error("Expected daemon job to have JobID after DaemonAndSave")
	}

	// Verify daemon job was saved to database
	retrievedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to retrieve saved daemon job: %v", err)
	}

	if retrievedJob.ScheduleType != string(job.ScheduleTypeDaemon) {
		t.Errorf("Expected schedule type daemon, got %s", retrievedJob.ScheduleType)
	}

	if retrievedJob.Name != "Test DaemonAndSave Job" {
		t.Errorf("Expected job name 'Test DaemonAndSave Job', got '%s'", retrievedJob.Name)
	}

	t.Log("DaemonAndSave job created and saved successfully")
}

// TestAddFunc tests the AddFunc method for adding Go functions as job executions
func TestAddFunc(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Create a job
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Test AddFunc Job",
		"description": "Testing Go function execution",
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Track if function was called
	funcCalled := false
	funcArgs := make(map[string]interface{})

	// Add a Go function execution
	err = testJob.AddFunc(&job.ExecutionOptions{
		Priority: 1,
	}, "test.func", func(ctx *job.ExecutionContext) error {
		funcCalled = true
		funcArgs = ctx.Args
		t.Logf("Function executed with args: %v", ctx.Args)
		return nil
	}, map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})
	if err != nil {
		t.Fatalf("Failed to add function execution: %v", err)
	}

	// Get the execution to verify it was saved
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) != 1 {
		t.Fatalf("Expected 1 execution, got %d", len(executions))
	}

	// Verify function is registered in global registry
	funcID := executions[0].ExecutionID
	fn, ok := job.GetFunc(funcID)
	if !ok || fn == nil {
		t.Error("Expected function to be registered in global registry")
	}

	// Push the job
	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to push job: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(2 * time.Second)

	// Verify function was called
	if !funcCalled {
		t.Error("Expected function to be called")
	}

	// Verify args were passed
	if funcArgs["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", funcArgs["key1"])
	}
	// Note: JSON unmarshaling converts numbers to float64
	key2Val, ok := funcArgs["key2"].(float64)
	if !ok {
		// Try int in case it wasn't serialized
		if intVal, ok := funcArgs["key2"].(int); ok {
			key2Val = float64(intVal)
		} else {
			t.Errorf("Expected key2 to be a number, got %T: %v", funcArgs["key2"], funcArgs["key2"])
		}
	}
	if key2Val != 42 {
		t.Errorf("Expected key2=42, got %v", key2Val)
	}

	// Verify function was cleaned up from registry after execution
	fn, ok = job.GetFunc(funcID)
	if ok || fn != nil {
		t.Error("Expected function to be removed from global registry after execution")
	}

	t.Log("AddFunc test completed successfully")
}

// TestAddFuncMemoryCleanup tests that memory is properly cleaned up after function execution
func TestAddFuncMemoryCleanup(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Create a job
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Test AddFunc Memory Cleanup",
		"description": "Testing memory cleanup after function execution",
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create a large closure to make memory leak more detectable
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	executed := false

	// Add a Go function with large closure
	err = testJob.AddFunc(&job.ExecutionOptions{
		Priority: 1,
	}, "test.cleanup", func(ctx *job.ExecutionContext) error {
		// Use largeData to ensure it's captured in closure
		_ = len(largeData)
		executed = true
		return nil
	}, map[string]interface{}{
		"test": "cleanup",
	})
	if err != nil {
		t.Fatalf("Failed to add function execution: %v", err)
	}

	// Get the execution to verify FuncID is set
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) != 1 {
		t.Fatalf("Expected 1 execution, got %d", len(executions))
	}
	funcID := executions[0].ExecutionID
	t.Logf("FuncID (ExecutionID): %s", funcID)

	// Verify function is registered in global registry before execution
	fn, ok := job.GetFunc(funcID)
	if !ok || fn == nil {
		t.Error("Expected function to be registered in global registry before execution")
	}

	// Push the job
	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to push job: %v", err)
	}

	// Wait for execution to complete with polling
	maxWait := 10 * time.Second
	pollInterval := 200 * time.Millisecond
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		if executed {
			break
		}
		time.Sleep(pollInterval)
	}

	// Verify function was executed
	if !executed {
		t.Error("Expected function to be executed")
	}

	// Wait for execution to complete in database
	var finalStatus string
	for time.Since(startTime) < maxWait {
		executions, err := testJob.GetExecutions()
		if err == nil && len(executions) > 0 {
			finalStatus = executions[0].Status
			t.Logf("Execution status: %s", finalStatus)
			if finalStatus == "completed" || finalStatus == "failed" {
				break
			}
		}
		time.Sleep(pollInterval)
	}

	// Wait a bit more for cleanup to complete
	time.Sleep(500 * time.Millisecond)

	// Verify memory cleanup: function should be removed from global registry
	fn, ok = job.GetFunc(funcID)
	if ok || fn != nil {
		t.Errorf("Expected function to be removed from global registry after completion")
	}

	// Verify execution status in database
	executions, err = testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}

	if len(executions) != 1 {
		t.Errorf("Expected 1 execution in database, got %d", len(executions))
	}

	if executions[0].Status != "completed" {
		t.Errorf("Expected execution status 'completed', got '%s'", executions[0].Status)
	}

	t.Log("AddFunc memory cleanup test completed successfully")
}

// TestAddFuncError tests error handling in AddFunc execution
func TestAddFuncError(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Create a job
	testJob, err := job.OnceAndSave(job.GOROUTINE, map[string]interface{}{
		"name":        "Test AddFunc Error",
		"description": "Testing error handling in function execution",
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Add a Go function that returns an error
	err = testJob.AddFunc(&job.ExecutionOptions{
		Priority: 1,
	}, "test.error", func(ctx *job.ExecutionContext) error {
		return fmt.Errorf("intentional test error")
	}, nil)
	if err != nil {
		t.Fatalf("Failed to add function execution: %v", err)
	}

	// Push the job
	err = testJob.Push()
	if err != nil {
		t.Fatalf("Failed to push job: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(2 * time.Second)

	// Verify execution failed
	executions, err := testJob.GetExecutions()
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}

	if len(executions) != 1 {
		t.Errorf("Expected 1 execution, got %d", len(executions))
	}

	if executions[0].Status != "failed" {
		t.Errorf("Expected execution status 'failed', got '%s'", executions[0].Status)
	}

	// Verify memory cleanup even on error: function should be removed from global registry
	// Get the execution ID first
	if len(executions) > 0 {
		funcID := executions[0].ExecutionID
		fn, ok := job.GetFunc(funcID)
		if ok || fn != nil {
			t.Errorf("Expected function to be removed from global registry after failure")
		}
	}

	t.Log("AddFunc error handling test completed successfully")
}
