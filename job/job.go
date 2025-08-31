package job

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
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

// SetWorkerManager sets a custom worker manager for this job (for testing)
func (j *Job) SetWorkerManager(wm *WorkerManager) {
	j.workerManager = wm
}

// Start start the job
func (j *Job) Start() error {
	// Get handler from registry (thread-safe)
	handler, exists := getHandler(j.JobID)
	if !exists {
		return fmt.Errorf("no handler registered for job %s", j.JobID)
	}

	// Update job status to ready
	j.Status = "ready"
	if err := SaveJob(j); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Submit to worker manager (use custom one if set, otherwise global)
	var wm *WorkerManager
	if j.workerManager != nil {
		wm = j.workerManager
	} else {
		wm = GetWorkerManager()
		// Start worker manager if not already started
		if wm.GetActiveWorkers() == 0 {
			wm.Start()
		}
	}

	return wm.SubmitJob(j, handler)
}

// Cancel cancel the job
func (j *Job) Cancel() error {
	// Update job status
	j.Status = "disabled"
	if err := SaveJob(j); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// If there's a current execution, mark it as cancelled
	if j.CurrentExecutionID != nil {
		execution, err := GetExecution(*j.CurrentExecutionID, model.QueryParam{})
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
				ExecutionID: j.CurrentExecutionID,
				Timestamp:   time.Now(),
				Sequence:    0,
			}
			SaveLog(logEntry)
		}
	}

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
	return &job, nil
}
