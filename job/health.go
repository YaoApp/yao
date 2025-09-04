package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
)

// HealthChecker manages job health monitoring
type HealthChecker struct {
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

var globalHealthChecker *HealthChecker

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the health check goroutine
func (hc *HealthChecker) Start() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	log.Info("Job health checker started with interval: %v", hc.interval)

	for {
		select {
		case <-ticker.C:
			if err := hc.performHealthCheck(); err != nil {
				log.Error("Health check failed: %v", err)
			}
		case <-hc.ctx.Done():
			log.Info("Job health checker stopped")
			return
		}
	}
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	if hc.cancel != nil {
		hc.cancel()
	}
}

// performHealthCheck performs health check
func (hc *HealthChecker) performHealthCheck() error {
	log.Debug("Starting health check...")

	// 1. Query all running jobs
	runningJobs, err := hc.getRunningJobs()
	if err != nil {
		return fmt.Errorf("failed to get running jobs: %w", err)
	}

	if len(runningJobs) == 0 {
		log.Debug("No running jobs found")
		return nil
	}

	log.Debug("Found %d running jobs to check", len(runningJobs))

	// 2. Check worker status for each job
	wm := GetWorkerManager()
	for _, job := range runningJobs {
		if err := hc.checkJobHealth(job, wm); err != nil {
			log.Error("Failed to check job %s health: %v", job.JobID, err)
		}
	}

	return nil
}

// getRunningJobs gets all jobs with running status
func (hc *HealthChecker) getRunningJobs() ([]*Job, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return nil, fmt.Errorf("job model not found")
	}

	param := model.QueryParam{
		Select: JobFields,
		Wheres: []model.QueryWhere{
			{Column: "status", Value: "running"},
			{Column: "enabled", Value: true},
		},
	}

	results, err := mod.Get(param)
	if err != nil {
		// If database is closed during testing, return empty slice instead of error
		if err.Error() == "sql: database is closed" {
			log.Debug("Database is closed, returning empty job list")
			return []*Job{}, nil
		}
		return nil, err
	}

	jobs := make([]*Job, 0, len(results))
	for _, result := range results {
		job := &Job{}
		if err := mapToStruct(result, job); err != nil {
			log.Warn("Failed to parse job data: %v", err)
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// checkJobHealth checks the health status of a single job
func (hc *HealthChecker) checkJobHealth(job *Job, wm *WorkerManager) error {
	// Get current execution record for the job
	if job.CurrentExecutionID == nil || *job.CurrentExecutionID == "" {
		// No current execution ID but status is running, this is abnormal
		return hc.markJobAsFailed(job, "Job status is running but no current execution ID found")
	}

	execution, err := GetExecution(*job.CurrentExecutionID, model.QueryParam{})
	if err != nil {
		return hc.markJobAsFailed(job, fmt.Sprintf("Failed to get execution: %v", err))
	}

	// Check execution status
	if execution.Status != "running" {
		// Execution status is not running but job status is running, this is inconsistent
		return hc.markJobAsFailed(job, fmt.Sprintf("Job status is running but execution status is %s", execution.Status))
	}

	// Check if worker exists and is working
	if execution.WorkerID == nil || *execution.WorkerID == "" {
		return hc.markJobAsFailed(job, "Execution is running but no worker ID assigned")
	}

	// Check if worker still exists in active workers
	if !hc.isWorkerActive(*execution.WorkerID, wm) {
		return hc.markJobAsFailed(job, fmt.Sprintf("Worker %s is no longer active", *execution.WorkerID))
	}

	// Check if execution has timed out
	if hc.isExecutionTimeout(execution) {
		return hc.markJobAsFailed(job, "Execution has timed out")
	}

	log.Debug("Job %s health check passed", job.JobID)
	return nil
}

// isWorkerActive checks if worker is still active
func (hc *HealthChecker) isWorkerActive(workerID string, wm *WorkerManager) bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	_, exists := wm.activeWorkers[workerID]
	return exists
}

// isExecutionTimeout checks if execution has timed out
func (hc *HealthChecker) isExecutionTimeout(execution *Execution) bool {
	if execution.StartedAt == nil {
		return false // No start time, cannot determine timeout
	}

	// Only check timeout if timeout is explicitly set
	if execution.TimeoutSeconds == nil || *execution.TimeoutSeconds <= 0 {
		return false // No timeout configured, execution can run indefinitely
	}

	timeoutDuration := time.Duration(*execution.TimeoutSeconds) * time.Second
	return time.Since(*execution.StartedAt) > timeoutDuration
}

// markJobAsFailed marks a job as failed
func (hc *HealthChecker) markJobAsFailed(job *Job, reason string) error {
	log.Warn("Marking job %s as failed: %s", job.JobID, reason)

	// Update job status
	job.Status = "failed"
	if err := SaveJob(job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update execution status (if exists)
	if job.CurrentExecutionID != nil && *job.CurrentExecutionID != "" {
		execution, err := GetExecution(*job.CurrentExecutionID, model.QueryParam{})
		if err == nil {
			execution.Status = "failed"
			now := time.Now()
			execution.EndedAt = &now

			// Set error information
			errorInfo := map[string]interface{}{
				"error":  reason,
				"time":   now,
				"source": "health_checker",
			}
			errorData, _ := jsoniter.Marshal(errorInfo)
			execution.ErrorInfo = (*json.RawMessage)(&errorData)

			if err := SaveExecution(execution); err != nil {
				log.Error("Failed to update execution status: %v", err)
			}
		}
	}

	// Record log entry
	logEntry := &Log{
		JobID:       job.JobID,
		Level:       "error",
		Message:     fmt.Sprintf("Job marked as failed by health checker: %s", reason),
		ExecutionID: job.CurrentExecutionID,
		Source:      stringPtr("health_checker"),
		Timestamp:   time.Now(),
		Sequence:    0,
	}

	if err := SaveLog(logEntry); err != nil {
		log.Error("Failed to save health check log: %v", err)
	}

	// Clear current execution ID
	job.CurrentExecutionID = nil
	if err := SaveJob(job); err != nil {
		log.Error("Failed to clear current execution ID: %v", err)
	}

	return nil
}

// stringPtr returns a string pointer
func stringPtr(s string) *string {
	return &s
}

// GetHealthChecker gets the global health checker (if needed)
// Note: Health checker is now started in job.go init() function
func GetHealthChecker() *HealthChecker {
	return globalHealthChecker
}

// ========================
// Data Cleaner - Independent cleanup functionality
// ========================

// DataCleaner manages cleanup of old job data
type DataCleaner struct {
	ctx             context.Context
	cancel          context.CancelFunc
	retentionDays   int
	lastCleanupTime time.Time
}

var globalDataCleaner *DataCleaner

// NewDataCleaner creates a new data cleaner
func NewDataCleaner(retentionDays int) *DataCleaner {
	ctx, cancel := context.WithCancel(context.Background())
	return &DataCleaner{
		ctx:             ctx,
		cancel:          cancel,
		retentionDays:   retentionDays,
		lastCleanupTime: time.Now(), // Initialize to avoid immediate cleanup on startup
	}
}

// Start starts the daily data cleanup routine
func (dc *DataCleaner) Start() {
	// Check every hour if daily cleanup is needed
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Info("Data cleaner started with %d days retention", dc.retentionDays)

	for {
		select {
		case <-ticker.C:
			if dc.shouldRunCleanup() {
				if err := dc.performCleanup(); err != nil {
					log.Error("Data cleanup failed: %v", err)
				} else {
					dc.lastCleanupTime = time.Now()
				}
			}
		case <-dc.ctx.Done():
			log.Info("Data cleaner stopped")
			return
		}
	}
}

// Stop stops the data cleaner
func (dc *DataCleaner) Stop() {
	if dc.cancel != nil {
		dc.cancel()
	}
}

// shouldRunCleanup checks if cleanup should run (once per day)
func (dc *DataCleaner) shouldRunCleanup() bool {
	return time.Since(dc.lastCleanupTime) >= 24*time.Hour
}

// performCleanup performs the actual data cleanup
func (dc *DataCleaner) performCleanup() error {
	log.Info("Starting daily data cleanup...")

	cutoffTime := time.Now().AddDate(0, 0, -dc.retentionDays)

	// Clean up jobs (excluding running jobs)
	deletedJobs, err := dc.cleanupJobs(cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup jobs: %w", err)
	}

	// Clean up executions
	deletedExecutions, err := dc.cleanupExecutions(cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup executions: %w", err)
	}

	// Clean up logs
	deletedLogs, err := dc.cleanupLogs(cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup logs: %w", err)
	}

	log.Info("Data cleanup completed: %d jobs, %d executions, %d logs deleted",
		deletedJobs, deletedExecutions, deletedLogs)

	return nil
}

// cleanupJobs removes old jobs that are not running
func (dc *DataCleaner) cleanupJobs(cutoffTime time.Time) (int, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return 0, fmt.Errorf("job model not found")
	}

	// Get jobs to delete (older than cutoff and not running)
	param := model.QueryParam{
		Select: []interface{}{"job_id"},
		Wheres: []model.QueryWhere{
			{Column: "created_at", OP: "<", Value: cutoffTime},
			{Column: "status", OP: "!=", Value: "running"},
		},
	}

	results, err := mod.Get(param)
	if err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, nil
	}

	// Extract job IDs
	jobIDs := make([]string, 0, len(results))
	for _, result := range results {
		if jobID, ok := result["job_id"].(string); ok {
			jobIDs = append(jobIDs, jobID)
		}
	}

	if len(jobIDs) == 0 {
		return 0, nil
	}

	// Delete jobs
	deleteParam := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "job_id", OP: "in", Value: jobIDs},
		},
	}

	deleted, err := mod.DeleteWhere(deleteParam)
	if err != nil {
		return 0, err
	}

	return deleted, nil
}

// cleanupExecutions removes old executions
func (dc *DataCleaner) cleanupExecutions(cutoffTime time.Time) (int, error) {
	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return 0, fmt.Errorf("job execution model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "created_at", OP: "<", Value: cutoffTime},
		},
	}

	deleted, err := mod.DeleteWhere(param)
	if err != nil {
		return 0, err
	}

	return deleted, nil
}

// cleanupLogs removes old logs
func (dc *DataCleaner) cleanupLogs(cutoffTime time.Time) (int, error) {
	mod := model.Select("__yao.job.log")
	if mod == nil {
		return 0, fmt.Errorf("job log model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "created_at", OP: "<", Value: cutoffTime},
		},
	}

	deleted, err := mod.DeleteWhere(param)
	if err != nil {
		return 0, err
	}

	return deleted, nil
}

// GetDataCleaner gets the global data cleaner
func GetDataCleaner() *DataCleaner {
	return globalDataCleaner
}
