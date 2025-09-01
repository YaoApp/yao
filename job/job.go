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

// Start start the job
func (j *Job) Start() error {
	// Get executions for this job
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

	// Set default values if not provided
	if job.CategoryID == "" {
		job.CategoryID = "default"
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
