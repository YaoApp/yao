package job

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
)

// init initializes the job package
func init() {
	// Initialize and start health checker
	initHealthChecker()

	// Initialize and start data cleaner
	initDataCleaner()

	log.Info("Job package initialized with health checker and data cleaner")
}

// initHealthChecker initializes the health checker
func initHealthChecker() {
	// Get health check interval from configuration or use default
	interval := getHealthCheckInterval()
	globalHealthChecker = NewHealthChecker(interval)

	// Start health check goroutine
	go globalHealthChecker.Start()

	log.Info("Job health checker started with %v interval", interval)
}

// getHealthCheckInterval returns the configured health check interval or default
func getHealthCheckInterval() time.Duration {
	// Default interval: 5 minutes (balanced between detection speed and resource usage)
	// This is suitable for most job monitoring scenarios:
	// - Short jobs (< 5min): Health check won't interfere much
	// - Medium jobs (5min - 1h): Good detection without excessive overhead
	// - Long jobs (> 1h): Timely detection of issues
	defaultInterval := 5 * time.Minute

	// TODO: Add configuration support from environment variables or config file
	// For example:
	// if envInterval := os.Getenv("YAO_JOB_HEALTH_CHECK_INTERVAL"); envInterval != "" {
	//     if duration, err := time.ParseDuration(envInterval); err == nil {
	//         return duration
	//     }
	// }

	return defaultInterval
}

// StopHealthChecker stops the health checker
func StopHealthChecker() {
	if globalHealthChecker != nil {
		globalHealthChecker.Stop()
		log.Info("Job health checker stopped")
	}
}

// RestartHealthChecker restarts the health checker with a new interval
// This is useful for testing or dynamic configuration changes
func RestartHealthChecker(interval time.Duration) {
	// Stop existing health checker
	StopHealthChecker()

	// Create and start new health checker with specified interval
	globalHealthChecker = NewHealthChecker(interval)
	go globalHealthChecker.Start()

	log.Info("Job health checker restarted with %v interval", interval)
}

// initDataCleaner initializes the data cleaner
func initDataCleaner() {
	// Create data cleaner with 90 days retention
	retentionDays := 90
	globalDataCleaner = NewDataCleaner(retentionDays)

	// Start data cleaner goroutine
	go globalDataCleaner.Start()

	log.Info("Job data cleaner started with %d days retention", retentionDays)
}

// StopDataCleaner stops the data cleaner
func StopDataCleaner() {
	if globalDataCleaner != nil {
		globalDataCleaner.Stop()
		log.Info("Job data cleaner stopped")
	}
}

// ForceCleanup forces an immediate data cleanup (useful for testing)
func ForceCleanup() error {
	if globalDataCleaner != nil {
		return globalDataCleaner.performCleanup()
	}
	return fmt.Errorf("data cleaner not initialized")
}

// Once create a new job
func Once(mode ModeType, data map[string]interface{}) (*Job, error) {
	data["mode"] = mode
	data["schedule_type"] = ScheduleTypeOnce
	raw, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	return makeJob(raw)
}

// OnceAndSave create a new job and save it immediately
func OnceAndSave(mode ModeType, data map[string]interface{}) (*Job, error) {
	job, err := Once(mode, data)
	if err != nil {
		return nil, err
	}

	err = SaveJob(job)
	if err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	return job, nil
}

// Cron create a new job
func Cron(mode ModeType, data map[string]interface{}, expression string) (*Job, error) {
	data["mode"] = mode
	data["schedule_type"] = ScheduleTypeCron
	data["schedule_expression"] = expression
	raw, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	return makeJob(raw)
}

// CronAndSave create a new cron job and save it immediately
func CronAndSave(mode ModeType, data map[string]interface{}, expression string) (*Job, error) {
	job, err := Cron(mode, data, expression)
	if err != nil {
		return nil, err
	}

	err = SaveJob(job)
	if err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	return job, nil
}

// Daemon create a new job
func Daemon(mode ModeType, data map[string]interface{}) (*Job, error) {
	data["mode"] = mode
	data["schedule_type"] = ScheduleTypeDaemon
	raw, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	return makeJob(raw)
}

// DaemonAndSave create a new daemon job and save it immediately
func DaemonAndSave(mode ModeType, data map[string]interface{}) (*Job, error) {
	job, err := Daemon(mode, data)
	if err != nil {
		return nil, err
	}

	err = SaveJob(job)
	if err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	return job, nil
}

// Push pushes the job to execution queue (renamed from Start for better semantics)
func (j *Job) Push() error {
	// Get executions from database
	// For ExecutionTypeFunc, the function is stored in global registry (funcRegistry)
	// and will be looked up by FuncID (ExecutionID) during execution
	executions, err := j.GetExecutions()
	if err != nil {
		return fmt.Errorf("failed to get executions: %w", err)
	}

	if len(executions) == 0 {
		return fmt.Errorf("no executions found for job %s", j.JobID)
	}

	// Sort executions by priority (higher priority first)
	sort.Slice(executions, func(i, j int) bool {
		priorityI := 0
		if executions[i].ExecutionOptions != nil {
			priorityI = executions[i].ExecutionOptions.Priority
		}
		priorityJ := 0
		if executions[j].ExecutionOptions != nil {
			priorityJ = executions[j].ExecutionOptions.Priority
		}
		return priorityI > priorityJ
	})

	// Initialize job context for cancellation
	if j.ctx == nil {
		j.ctx, j.cancel = context.WithCancel(context.Background())
	}

	// Initialize execution contexts map
	if j.executionContexts == nil {
		j.executionContexts = make(map[string]context.CancelFunc)
	}

	// Update job status to ready
	j.Status = "ready"
	if err := SaveJob(j); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Get global worker manager (should be already started)
	wm := GetWorkerManager()

	// Submit executions and ensure all are added successfully
	var submitErrors []string
	for _, execution := range executions {
		// Create execution-specific context derived from job context
		execCtx, execCancel := context.WithCancel(j.ctx)

		// Store execution cancel function
		j.executionMutex.Lock()
		j.executionContexts[execution.ExecutionID] = execCancel
		j.executionMutex.Unlock()

		// Submit execution (non-blocking)
		if err := wm.SubmitJob(execCtx, j, execution); err != nil {
			// Clean up on error
			execCancel()
			j.executionMutex.Lock()
			delete(j.executionContexts, execution.ExecutionID)
			j.executionMutex.Unlock()

			submitErrors = append(submitErrors, fmt.Sprintf("execution %s: %v", execution.ExecutionID, err))
			log.Error("Failed to submit execution %s: %v", execution.ExecutionID, err)
		}
	}

	// Return error if any submissions failed
	if len(submitErrors) > 0 {
		return fmt.Errorf("failed to submit some executions: %s", strings.Join(submitErrors, "; "))
	}

	return nil
}

// Stop stops the job and cancels all running executions
func (j *Job) Stop() error {
	// Update job status
	j.Status = "disabled"
	if err := SaveJob(j); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Cancel all running executions using job context
	if j.cancel != nil {
		j.cancel()
		log.Info("Job %s cancelled, all executions will be stopped", j.JobID)
	}

	// Cancel individual execution contexts and clean up
	j.executionMutex.Lock()
	for executionID, cancelFunc := range j.executionContexts {
		cancelFunc()

		// Update execution status in database
		execution, err := GetExecution(executionID, model.QueryParam{})
		if err == nil && (execution.Status == "queued" || execution.Status == "running") {
			execution.Status = "cancelled"
			execution.EndedAt = &time.Time{}
			*execution.EndedAt = time.Now()
			SaveExecution(execution)

			// Log cancellation
			logEntry := &Log{
				JobID:       j.JobID,
				Level:       "info",
				Message:     "Job execution cancelled by user",
				ExecutionID: &executionID,
				Timestamp:   time.Now(),
				Sequence:    0,
			}
			SaveLog(logEntry)
		}
	}
	// Clear execution contexts
	j.executionContexts = make(map[string]context.CancelFunc)
	j.executionMutex.Unlock()

	return nil
}

// Destroy destroys the job and cleans up all resources
func (j *Job) Destroy() error {
	// Stop the job first
	if err := j.Stop(); err != nil {
		log.Warn("Failed to stop job during destroy: %v", err)
	}

	// Handlers are now stored with executions, no global registry to clean

	// Update job status to deleted
	j.Status = "deleted"
	if err := SaveJob(j); err != nil {
		log.Warn("Failed to update job status to deleted: %v", err)
	}

	// Clear job references
	if j.cancel != nil {
		j.cancel()
		j.cancel = nil
	}

	log.Info("Job %s destroyed successfully", j.JobID)
	return nil
}

// SetData set the data of the job
func (j *Job) SetData(data map[string]interface{}) *Job {
	return j
}

// SetConfig set the config of the job
func (j *Job) SetConfig(config map[string]interface{}) *Job {
	j.Config = config
	return j
}

// SetName set the name of the job
func (j *Job) SetName(name string) *Job {
	j.Name = name
	return j
}

// SetDescription set the description of the job
func (j *Job) SetDescription(description string) *Job {
	j.Description = &description
	return j
}

// SetCategory set the category of the job
func (j *Job) SetCategory(category string) *Job {
	j.CategoryID = category
	return j
}

// SetMaxWorkerNums set the max worker nums of the job
func (j *Job) SetMaxWorkerNums(maxWorkerNums int) *Job {
	j.MaxWorkerNums = maxWorkerNums
	return j
}

// SetStatus set the status of the job
func (j *Job) SetStatus(status string) *Job {
	j.Status = status
	return j
}

// SetMaxRetryCount set the max retry count of the job
func (j *Job) SetMaxRetryCount(maxRetryCount int) *Job {
	j.MaxRetryCount = maxRetryCount
	return j
}

// SetDefaultTimeout set the default timeout of the job
func (j *Job) SetDefaultTimeout(defaultTimeout int) *Job {
	j.DefaultTimeout = &defaultTimeout
	return j
}

// SetMode set the mode of the job
func (j *Job) SetMode(mode ModeType) {
	j.Mode = mode
}

func makeJob(data []byte) (*Job, error) {
	var job Job
	err := jsoniter.Unmarshal(data, &job)
	if err != nil {
		return nil, err
	}

	// Set default CategoryName if both CategoryID and CategoryName are empty
	if job.CategoryID == "" && job.CategoryName == "" {
		job.CategoryName = "Default"
	}
	if job.Status == "" {
		job.Status = "draft"
	}
	if job.MaxWorkerNums == 0 {
		job.MaxWorkerNums = 1 // Default to 1 worker
	}
	if job.Priority == 0 {
		job.Priority = 1 // Default job priority
	}
	if job.CreatedBy == "" {
		job.CreatedBy = "system"
	}

	// If YaoCreatedBy is set, use it as the created by
	if job.YaoCreatedBy != "" {
		job.CreatedBy = job.YaoCreatedBy
	}

	// Set default enabled to true if not specified
	// Note: Go's zero value for bool is false, so we need to explicitly check if it was set
	// Since we can't distinguish between explicitly set false and zero value,
	// we'll assume new jobs should be enabled by default
	if !job.System && !job.Readonly {
		job.Enabled = true
	}

	return &job, nil
}

// RestoreJobsFromDatabase restores jobs from database on system startup
func RestoreJobsFromDatabase() ([]*Job, error) {
	// Get all active jobs from database
	activeJobs, err := GetActiveJobs()
	if err != nil {
		return nil, fmt.Errorf("failed to get active jobs: %w", err)
	}

	// No need to restore handlers since we only use Yao processes and commands
	// Both are fully serializable and self-contained

	log.Info("Restored %d jobs from database", len(activeJobs))
	return activeJobs, nil
}
