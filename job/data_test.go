package job_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/test"
)

// TestJobCRUD tests job CRUD operations
func TestJobCRUD(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a test category first
	category, err := job.GetOrCreateCategory("test-crud-category", "Test category for CRUD operations")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Test job creation with unique ID
	timestamp := time.Now().UnixNano()
	testJob := &job.Job{
		JobID:         fmt.Sprintf("test-job-crud-%d", timestamp),
		Name:          "Test CRUD Job",
		CategoryID:    category.CategoryID,
		Status:        "draft",
		Mode:          job.GOROUTINE,
		ScheduleType:  string(job.ScheduleTypeOnce),
		MaxWorkerNums: 1,
		MaxRetryCount: 0,
		Priority:      5,
		CreatedBy:     "test-user",
		Enabled:       true,
		System:        false,
		Readonly:      false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Ensure cleanup even if test fails
	defer func() {
		job.RemoveJobs([]string{testJob.JobID})
		job.RemoveCategories([]string{category.CategoryID})
	}()

	// Test SaveJob (Create)
	err = job.SaveJob(testJob)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
	if testJob.ID == 0 {
		t.Error("Expected job ID to be set after creation")
	}

	// Test GetJob (Read)
	retrievedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if retrievedJob.Name != testJob.Name {
		t.Errorf("Expected job name '%s', got '%s'", testJob.Name, retrievedJob.Name)
	}
	if retrievedJob.Priority != testJob.Priority {
		t.Errorf("Expected priority %d, got %d", testJob.Priority, retrievedJob.Priority)
	}

	// Test SaveJob (Update)
	retrievedJob.Name = "Updated CRUD Job"
	retrievedJob.Priority = 10
	err = job.SaveJob(retrievedJob)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	// Verify update
	updatedJob, err := job.GetJob(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}
	if updatedJob.Name != "Updated CRUD Job" {
		t.Errorf("Expected updated name 'Updated CRUD Job', got '%s'", updatedJob.Name)
	}
	if updatedJob.Priority != 10 {
		t.Errorf("Expected updated priority 10, got %d", updatedJob.Priority)
	}

	// Test ListJobs
	jobs, err := job.ListJobs(model.QueryParam{}, 1, 10)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if jobs["total"].(int) == 0 {
		t.Error("Expected at least one job in list")
	}

	// Test CountJobs
	count, err := job.CountJobs(model.QueryParam{})
	if err != nil {
		t.Fatalf("Failed to count jobs: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one job in count")
	}

	// Test GetActiveJobs
	retrievedJob.Status = "ready"
	job.SaveJob(retrievedJob)
	activeJobs, err := job.GetActiveJobs()
	if err != nil {
		t.Fatalf("Failed to get active jobs: %v", err)
	}
	found := false
	for _, activeJob := range activeJobs {
		if activeJob.JobID == testJob.JobID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find the test job in active jobs")
	}

	// Test RemoveJobs (Delete)
	err = job.RemoveJobs([]string{testJob.JobID})
	if err != nil {
		t.Fatalf("Failed to remove job: %v", err)
	}

	// Verify deletion
	_, err = job.GetJob(testJob.JobID)
	if err == nil {
		t.Error("Expected error when getting deleted job")
	}
}

// TestCategoryCRUD tests category CRUD operations
func TestCategoryCRUD(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Test category creation with unique ID
	timestamp := time.Now().UnixNano()
	testCategory := &job.Category{
		CategoryID:  fmt.Sprintf("test-category-crud-%d", timestamp),
		Name:        "Test CRUD Category",
		Description: stringPtr("Test category for CRUD operations"),
		Sort:        1,
		System:      false,
		Enabled:     true,
		Readonly:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Ensure cleanup even if test fails
	defer func() {
		job.RemoveCategories([]string{testCategory.CategoryID})
	}()

	// Test SaveCategory (Create)
	err := job.SaveCategory(testCategory)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}
	if testCategory.ID == 0 {
		t.Error("Expected category ID to be set after creation")
	}

	// Test GetCategories (Read)
	categories, err := job.GetCategories(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "category_id", Value: testCategory.CategoryID},
		},
	})
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}
	if len(categories) == 0 {
		t.Fatal("Expected to find the test category")
	}
	if categories[0].Name != testCategory.Name {
		t.Errorf("Expected category name '%s', got '%s'", testCategory.Name, categories[0].Name)
	}

	// Test SaveCategory (Update)
	testCategory.Name = "Updated CRUD Category"
	testCategory.Sort = 5
	err = job.SaveCategory(testCategory)
	if err != nil {
		t.Fatalf("Failed to update category: %v", err)
	}

	// Verify update
	updatedCategories, err := job.GetCategories(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "category_id", Value: testCategory.CategoryID},
		},
	})
	if err != nil {
		t.Fatalf("Failed to get updated categories: %v", err)
	}
	if len(updatedCategories) == 0 {
		t.Error("Expected to find the updated category")
	}
	if updatedCategories[0].Name != "Updated CRUD Category" {
		t.Errorf("Expected updated name 'Updated CRUD Category', got '%s'", updatedCategories[0].Name)
	}

	// Test CountCategories
	count, err := job.CountCategories(model.QueryParam{})
	if err != nil {
		t.Fatalf("Failed to count categories: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one category in count")
	}

	// Test GetOrCreateCategory
	existingCategory, err := job.GetOrCreateCategory("Updated CRUD Category", "Should find existing")
	if err != nil {
		t.Fatalf("Failed to get existing category: %v", err)
	}
	if existingCategory.CategoryID != testCategory.CategoryID {
		t.Error("Expected to get the existing category")
	}

	newCategory, err := job.GetOrCreateCategory("Brand New Category", "Should create new")
	if err != nil {
		t.Fatalf("Failed to create new category: %v", err)
	}
	if newCategory.CategoryID == testCategory.CategoryID {
		t.Error("Expected to create a new category")
	}

	// Test RemoveCategories (Delete)
	err = job.RemoveCategories([]string{testCategory.CategoryID, newCategory.CategoryID})
	if err != nil {
		t.Fatalf("Failed to remove categories: %v", err)
	}

	// Verify deletion
	deletedCategories, err := job.GetCategories(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "category_id", OP: "in", Value: []string{testCategory.CategoryID, newCategory.CategoryID}},
		},
	})
	if err != nil {
		t.Fatalf("Failed to check deleted categories: %v", err)
	}
	if len(deletedCategories) != 0 {
		t.Error("Expected categories to be deleted")
	}
}

// TestExecutionCRUD tests execution CRUD operations
func TestExecutionCRUD(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test job first with unique ID
	timestamp := time.Now().UnixNano()
	testJob := &job.Job{
		JobID:         fmt.Sprintf("test-execution-job-%d", timestamp),
		Name:          "Test Execution Job",
		CategoryID:    "default",
		Status:        "ready",
		Mode:          job.GOROUTINE,
		ScheduleType:  string(job.ScheduleTypeOnce),
		MaxWorkerNums: 1,
		CreatedBy:     "test-user",
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err := job.SaveJob(testJob)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Ensure cleanup even if test fails
	defer func() {
		job.RemoveJobs([]string{testJob.JobID})
	}()

	// Test execution creation with unique ID
	executionTimestamp := time.Now().UnixNano() + 1 // Ensure different from job timestamp
	testExecution := &job.Execution{
		ExecutionID:     fmt.Sprintf("test-execution-crud-%d", executionTimestamp),
		JobID:           testJob.JobID,
		Status:          "queued",
		TriggerCategory: "manual",
		RetryAttempt:    0,
		Progress:        0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Test SaveExecution (Create)
	err = job.SaveExecution(testExecution)
	if err != nil {
		t.Fatalf("Failed to create execution: %v", err)
	}
	if testExecution.ID == 0 {
		t.Error("Expected execution ID to be set after creation")
	}

	// Test GetExecution (Read)
	retrievedExecution, err := job.GetExecution(testExecution.ExecutionID, model.QueryParam{})
	if err != nil {
		t.Fatalf("Failed to get execution: %v", err)
	}
	if retrievedExecution.Status != testExecution.Status {
		t.Errorf("Expected execution status '%s', got '%s'", testExecution.Status, retrievedExecution.Status)
	}

	// Test SaveExecution (Update)
	retrievedExecution.Status = "running"
	retrievedExecution.Progress = 50
	now := time.Now()
	retrievedExecution.StartedAt = &now
	err = job.SaveExecution(retrievedExecution)
	if err != nil {
		t.Fatalf("Failed to update execution: %v", err)
	}

	// Verify update
	updatedExecution, err := job.GetExecution(testExecution.ExecutionID, model.QueryParam{})
	if err != nil {
		t.Fatalf("Failed to get updated execution: %v", err)
	}
	if updatedExecution.Status != "running" {
		t.Errorf("Expected updated status 'running', got '%s'", updatedExecution.Status)
	}
	if updatedExecution.Progress != 50 {
		t.Errorf("Expected updated progress 50, got %d", updatedExecution.Progress)
	}

	// Test GetExecutions
	executions, err := job.GetExecutions(testJob.JobID)
	if err != nil {
		t.Fatalf("Failed to get executions: %v", err)
	}
	if len(executions) == 0 {
		t.Error("Expected at least one execution")
	}

	// Test CountExecutions
	count, err := job.CountExecutions(testJob.JobID, model.QueryParam{})
	if err != nil {
		t.Fatalf("Failed to count executions: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one execution in count")
	}

	// Test RemoveExecutions (Delete)
	err = job.RemoveExecutions([]string{testExecution.ExecutionID})
	if err != nil {
		t.Fatalf("Failed to remove execution: %v", err)
	}

	// Verify deletion
	_, err = job.GetExecution(testExecution.ExecutionID, model.QueryParam{})
	if err == nil {
		t.Error("Expected error when getting deleted execution")
	}

	// Clean up job
	job.RemoveJobs([]string{testJob.JobID})
}

// TestLogCRUD tests log CRUD operations
func TestLogCRUD(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test job first with unique ID
	timestamp := time.Now().UnixNano()
	testJob := &job.Job{
		JobID:         fmt.Sprintf("test-log-job-%d", timestamp),
		Name:          "Test Log Job",
		CategoryID:    "default",
		Status:        "ready",
		Mode:          job.GOROUTINE,
		ScheduleType:  string(job.ScheduleTypeOnce),
		MaxWorkerNums: 1,
		CreatedBy:     "test-user",
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err := job.SaveJob(testJob)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Ensure cleanup even if test fails
	defer func() {
		job.RemoveJobs([]string{testJob.JobID})
	}()

	// Test log creation
	testLog := &job.Log{
		JobID:     testJob.JobID,
		Level:     "info",
		Message:   "Test log message for CRUD operations",
		Timestamp: time.Now(),
		Sequence:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test SaveLog (Create)
	err = job.SaveLog(testLog)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}
	if testLog.ID == 0 {
		t.Error("Expected log ID to be set after creation")
	}

	// Create more logs for testing
	for i := 2; i <= 5; i++ {
		log := &job.Log{
			JobID:     testJob.JobID,
			Level:     "debug",
			Message:   fmt.Sprintf("Test log message %d", i),
			Timestamp: time.Now(),
			Sequence:  i,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		job.SaveLog(log)
	}

	// Test ListLogs (Read with pagination)
	logs, err := job.ListLogs(testJob.JobID, model.QueryParam{}, 1, 10)
	if err != nil {
		t.Fatalf("Failed to list logs: %v", err)
	}
	if logs["total"].(int) < 5 {
		t.Errorf("Expected at least 5 logs, got %d", logs["total"].(int))
	}

	// Test logs with filtering
	filteredLogs, err := job.ListLogs(testJob.JobID, model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "level", Value: "info"},
		},
	}, 1, 10)
	if err != nil {
		t.Fatalf("Failed to list filtered logs: %v", err)
	}
	if filteredLogs["total"].(int) < 1 {
		t.Errorf("Expected at least 1 info log, got %d", filteredLogs["total"].(int))
	}

	// Test RemoveLogs (Delete) - get some log IDs first
	allLogs, _ := job.ListLogs(testJob.JobID, model.QueryParam{}, 1, 100)
	if items, ok := allLogs["items"].([]interface{}); ok && len(items) > 0 {
		// Remove first log
		if firstLog, ok := items[0].(map[string]interface{}); ok {
			if id, ok := firstLog["id"]; ok {
				err = job.RemoveLogs([]string{fmt.Sprintf("%v", id)})
				if err != nil {
					t.Fatalf("Failed to remove log: %v", err)
				}

				// Verify deletion
				remainingLogs, _ := job.ListLogs(testJob.JobID, model.QueryParam{}, 1, 100)
				if remainingLogs["total"].(int) >= allLogs["total"].(int) {
					t.Error("Expected fewer logs after deletion")
				}
			}
		}
	}

	// Clean up job (this should cascade delete logs)
	job.RemoveJobs([]string{testJob.JobID})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
