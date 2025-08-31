package job_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/job"
)

// TestJobTypes tests job type definitions and constants
func TestJobTypes(t *testing.T) {
	// Test ModeType constants
	if job.GOROUTINE != "GOROUTINE" {
		t.Errorf("Expected GOROUTINE mode to be 'GOROUTINE', got '%s'", job.GOROUTINE)
	}
	if job.PROCESS != "PROCESS" {
		t.Errorf("Expected PROCESS mode to be 'PROCESS', got '%s'", job.PROCESS)
	}

	// Test ScheduleType constants
	if job.ScheduleTypeOnce != "once" {
		t.Errorf("Expected ScheduleTypeOnce to be 'once', got '%s'", job.ScheduleTypeOnce)
	}
	if job.ScheduleTypeCron != "cron" {
		t.Errorf("Expected ScheduleTypeCron to be 'cron', got '%s'", job.ScheduleTypeCron)
	}
	if job.ScheduleTypeDaemon != "daemon" {
		t.Errorf("Expected ScheduleTypeDaemon to be 'daemon', got '%s'", job.ScheduleTypeDaemon)
	}

	// Test LogLevel constants
	expectedLevels := map[job.LogLevel]string{
		job.Debug: "debug",
		job.Info:  "info",
		job.Warn:  "warn",
		job.Error: "error",
		job.Fatal: "fatal",
		job.Panic: "panic",
		job.Trace: "trace",
	}

	for level, name := range expectedLevels {
		if level > 6 {
			t.Errorf("LogLevel %s has invalid value %d", name, level)
		}
	}
}

// TestJobStructure tests Job struct serialization and deserialization
func TestJobStructure(t *testing.T) {
	// Create a test job
	now := time.Now()
	description := "Test job description"
	timeout := 300
	nextRun := now.Add(time.Hour)
	lastRun := now.Add(-time.Hour)
	currentExecID := "exec-123"

	testJob := &job.Job{
		ID:                 1,
		JobID:              "test-job-001",
		Name:               "Test Job",
		Icon:               &description,
		Description:        &description,
		CategoryID:         "test-category",
		MaxWorkerNums:      2,
		Status:             "ready",
		Mode:               job.GOROUTINE,
		ScheduleType:       string(job.ScheduleTypeOnce),
		ScheduleExpression: nil,
		MaxRetryCount:      3,
		DefaultTimeout:     &timeout,
		Priority:           5,
		CreatedBy:          "test-user",
		NextRunAt:          &nextRun,
		LastRunAt:          &lastRun,
		CurrentExecutionID: &currentExecID,
		Config:             map[string]interface{}{"key": "value"},
		Sort:               1,
		Enabled:            true,
		System:             false,
		Readonly:           false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(testJob)
	if err != nil {
		t.Fatalf("Failed to marshal job to JSON: %v", err)
	}

	// Test JSON deserialization
	var deserializedJob job.Job
	err = json.Unmarshal(jsonData, &deserializedJob)
	if err != nil {
		t.Fatalf("Failed to unmarshal job from JSON: %v", err)
	}

	// Verify key fields
	if deserializedJob.JobID != testJob.JobID {
		t.Errorf("Expected JobID '%s', got '%s'", testJob.JobID, deserializedJob.JobID)
	}
	if deserializedJob.Name != testJob.Name {
		t.Errorf("Expected Name '%s', got '%s'", testJob.Name, deserializedJob.Name)
	}
	if deserializedJob.Mode != testJob.Mode {
		t.Errorf("Expected Mode '%s', got '%s'", testJob.Mode, deserializedJob.Mode)
	}
	if deserializedJob.Priority != testJob.Priority {
		t.Errorf("Expected Priority %d, got %d", testJob.Priority, deserializedJob.Priority)
	}
}

// TestCategoryStructure tests Category struct
func TestCategoryStructure(t *testing.T) {
	now := time.Now()
	description := "Test category description"

	testCategory := &job.Category{
		ID:          1,
		CategoryID:  "test-category-001",
		Name:        "Test Category",
		Icon:        &description,
		Description: &description,
		Sort:        1,
		System:      false,
		Enabled:     true,
		Readonly:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(testCategory)
	if err != nil {
		t.Fatalf("Failed to marshal category to JSON: %v", err)
	}

	// Test JSON deserialization
	var deserializedCategory job.Category
	err = json.Unmarshal(jsonData, &deserializedCategory)
	if err != nil {
		t.Fatalf("Failed to unmarshal category from JSON: %v", err)
	}

	// Verify key fields
	if deserializedCategory.CategoryID != testCategory.CategoryID {
		t.Errorf("Expected CategoryID '%s', got '%s'", testCategory.CategoryID, deserializedCategory.CategoryID)
	}
	if deserializedCategory.Name != testCategory.Name {
		t.Errorf("Expected Name '%s', got '%s'", testCategory.Name, deserializedCategory.Name)
	}
	if deserializedCategory.Enabled != testCategory.Enabled {
		t.Errorf("Expected Enabled %v, got %v", testCategory.Enabled, deserializedCategory.Enabled)
	}
}

// TestExecutionStructure tests Execution struct
func TestExecutionStructure(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-time.Minute)
	endedAt := now
	timeout := 300
	duration := 60000
	parentExecID := "parent-exec-001"
	workerID := "worker-001"
	processID := "process-001"

	testExecution := &job.Execution{
		ID:                1,
		ExecutionID:       "test-execution-001",
		JobID:             "test-job-001",
		Status:            "completed",
		TriggerCategory:   "manual",
		TriggerSource:     &workerID,
		ScheduledAt:       &startedAt,
		WorkerID:          &workerID,
		ProcessID:         &processID,
		RetryAttempt:      0,
		ParentExecutionID: &parentExecID,
		StartedAt:         &startedAt,
		EndedAt:           &endedAt,
		TimeoutSeconds:    &timeout,
		Duration:          &duration,
		Progress:          100,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(testExecution)
	if err != nil {
		t.Fatalf("Failed to marshal execution to JSON: %v", err)
	}

	// Test JSON deserialization
	var deserializedExecution job.Execution
	err = json.Unmarshal(jsonData, &deserializedExecution)
	if err != nil {
		t.Fatalf("Failed to unmarshal execution from JSON: %v", err)
	}

	// Verify key fields
	if deserializedExecution.ExecutionID != testExecution.ExecutionID {
		t.Errorf("Expected ExecutionID '%s', got '%s'", testExecution.ExecutionID, deserializedExecution.ExecutionID)
	}
	if deserializedExecution.JobID != testExecution.JobID {
		t.Errorf("Expected JobID '%s', got '%s'", testExecution.JobID, deserializedExecution.JobID)
	}
	if deserializedExecution.Status != testExecution.Status {
		t.Errorf("Expected Status '%s', got '%s'", testExecution.Status, deserializedExecution.Status)
	}
	if deserializedExecution.Progress != testExecution.Progress {
		t.Errorf("Expected Progress %d, got %d", testExecution.Progress, deserializedExecution.Progress)
	}
}

// TestLogStructure tests Log struct
func TestLogStructure(t *testing.T) {
	now := time.Now()
	executionID := "test-execution-001"
	source := "test-handler"
	step := "initialization"
	progress := 50
	duration := 1000
	errorCode := "ERR001"
	stackTrace := "stack trace here"
	workerID := "worker-001"
	processID := "process-001"

	testLog := &job.Log{
		ID:          1,
		JobID:       "test-job-001",
		Level:       "info",
		Message:     "Test log message",
		Source:      &source,
		ExecutionID: &executionID,
		Step:        &step,
		Progress:    &progress,
		Duration:    &duration,
		ErrorCode:   &errorCode,
		StackTrace:  &stackTrace,
		WorkerID:    &workerID,
		ProcessID:   &processID,
		Timestamp:   now,
		Sequence:    1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(testLog)
	if err != nil {
		t.Fatalf("Failed to marshal log to JSON: %v", err)
	}

	// Test JSON deserialization
	var deserializedLog job.Log
	err = json.Unmarshal(jsonData, &deserializedLog)
	if err != nil {
		t.Fatalf("Failed to unmarshal log from JSON: %v", err)
	}

	// Verify key fields
	if deserializedLog.JobID != testLog.JobID {
		t.Errorf("Expected JobID '%s', got '%s'", testLog.JobID, deserializedLog.JobID)
	}
	if deserializedLog.Level != testLog.Level {
		t.Errorf("Expected Level '%s', got '%s'", testLog.Level, deserializedLog.Level)
	}
	if deserializedLog.Message != testLog.Message {
		t.Errorf("Expected Message '%s', got '%s'", testLog.Message, deserializedLog.Message)
	}
	if deserializedLog.Sequence != testLog.Sequence {
		t.Errorf("Expected Sequence %d, got %d", testLog.Sequence, deserializedLog.Sequence)
	}
}

// TestProgressStructure tests Progress struct
func TestProgressStructure(t *testing.T) {
	testProgress := &job.Progress{
		ExecutionID: "test-execution-001",
		Progress:    75,
		Message:     "75% complete",
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(testProgress)
	if err != nil {
		t.Fatalf("Failed to marshal progress to JSON: %v", err)
	}

	// Test JSON deserialization
	var deserializedProgress job.Progress
	err = json.Unmarshal(jsonData, &deserializedProgress)
	if err != nil {
		t.Fatalf("Failed to unmarshal progress from JSON: %v", err)
	}

	// Verify fields
	if deserializedProgress.ExecutionID != testProgress.ExecutionID {
		t.Errorf("Expected ExecutionID '%s', got '%s'", testProgress.ExecutionID, deserializedProgress.ExecutionID)
	}
	if deserializedProgress.Progress != testProgress.Progress {
		t.Errorf("Expected Progress %d, got %d", testProgress.Progress, deserializedProgress.Progress)
	}
	if deserializedProgress.Message != testProgress.Message {
		t.Errorf("Expected Message '%s', got '%s'", testProgress.Message, deserializedProgress.Message)
	}
}

// TestHandlerFunc tests HandlerFunc type
func TestHandlerFunc(t *testing.T) {
	// Test handler function signature
	var handler job.HandlerFunc = func(ctx context.Context, execution *job.Execution) error {
		if ctx == nil {
			return fmt.Errorf("context is nil")
		}
		if execution == nil {
			return fmt.Errorf("execution is nil")
		}
		execution.Info("Handler executed successfully")
		return nil
	}

	// Test handler function signature
	if handler == nil {
		t.Error("Handler function should not be nil")
	}
}
