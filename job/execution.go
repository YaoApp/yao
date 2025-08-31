package job

import (
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
)

// handlerRegistry stores registered handlers for jobs
var handlerRegistry = make(map[string]HandlerFunc)
var handlerRegistryMutex sync.RWMutex

// SetHandler sets a handler for a job ID (for testing)
func SetHandler(jobID string, handler HandlerFunc) {
	handlerRegistryMutex.Lock()
	defer handlerRegistryMutex.Unlock()
	handlerRegistry[jobID] = handler
}

// getHandler gets a handler for a job ID (thread-safe)
func getHandler(jobID string) (HandlerFunc, bool) {
	handlerRegistryMutex.RLock()
	defer handlerRegistryMutex.RUnlock()
	handler, exists := handlerRegistry[jobID]
	return handler, exists
}

// Add add a new execution to the job with handler
func (j *Job) Add(priority int, handler HandlerFunc) error {
	// Set job priority
	j.Priority = priority

	// Auto-create or get category if not set
	if j.CategoryID == "" {
		category, err := GetOrCreateCategory("default", "Default job category")
		if err != nil {
			log.Warn("Failed to create default category: %v", err)
			j.CategoryID = "default"
		} else {
			j.CategoryID = category.CategoryID
		}
	}

	// Set default values
	if j.Status == "" {
		j.Status = "draft"
	}
	if j.MaxWorkerNums == 0 {
		j.MaxWorkerNums = 1
	}
	if j.CreatedBy == "" {
		j.CreatedBy = "system"
	}

	// Save job to database first
	err := SaveJob(j)
	if err != nil {
		return err
	}

	// Store handler in registry using the final JobID (thread-safe)
	handlerRegistryMutex.Lock()
	handlerRegistry[j.JobID] = handler
	handlerRegistryMutex.Unlock()

	return nil
}

// GetExecutions get executions for this job
func (j *Job) GetExecutions() ([]*Execution, error) {
	return GetExecutions(j.JobID)
}

// GetExecution get specific execution for this job
func (j *Job) GetExecution(executionID string) (*Execution, error) {
	return GetExecution(executionID, model.QueryParam{})
}

// Log log with execution context
func (e *Execution) Log(level LogLevel, format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)

	// Map LogLevel to string
	levelStr := ""
	switch level {
	case Debug:
		levelStr = "debug"
	case Info:
		levelStr = "info"
	case Warn:
		levelStr = "warning"
	case Error:
		levelStr = "error"
	case Fatal:
		levelStr = "fatal"
	case Panic:
		levelStr = "fatal"
	case Trace:
		levelStr = "debug"
	default:
		levelStr = "info"
	}

	// Create log entry
	logEntry := &Log{
		JobID:       e.JobID,
		Level:       levelStr,
		Message:     message,
		ExecutionID: &e.ExecutionID,
		WorkerID:    e.WorkerID,
		ProcessID:   e.ProcessID,
		Progress:    &e.Progress,
		Timestamp:   time.Now(),
		Sequence:    0, // TODO: implement sequence tracking
	}

	// Save to database
	err := SaveLog(logEntry)
	if err != nil {
		log.Error("Failed to save log entry: %v", err)
	}

	// Also log to system logger
	switch level {
	case Debug, Trace:
		log.Debug("[Job:%s][Exec:%s] %s", e.JobID, e.ExecutionID, message)
	case Info:
		log.Info("[Job:%s][Exec:%s] %s", e.JobID, e.ExecutionID, message)
	case Warn:
		log.Warn("[Job:%s][Exec:%s] %s", e.JobID, e.ExecutionID, message)
	case Error, Fatal, Panic:
		log.Error("[Job:%s][Exec:%s] %s", e.JobID, e.ExecutionID, message)
	}

	return err
}

// Info info log
func (e *Execution) Info(format string, args ...interface{}) error {
	return e.Log(Info, format, args...)
}

// Debug debug log
func (e *Execution) Debug(format string, args ...interface{}) error {
	return e.Log(Debug, format, args...)
}

// Warn warn log
func (e *Execution) Warn(format string, args ...interface{}) error {
	return e.Log(Warn, format, args...)
}

// Error error log
func (e *Execution) Error(format string, args ...interface{}) error {
	return e.Log(Error, format, args...)
}

// Fatal fatal log
func (e *Execution) Fatal(format string, args ...interface{}) error {
	return e.Log(Fatal, format, args...)
}

// Panic panic log
func (e *Execution) Panic(format string, args ...interface{}) error {
	return e.Log(Panic, format, args...)
}

// Trace trace log
func (e *Execution) Trace(format string, args ...interface{}) error {
	return e.Log(Trace, format, args...)
}

// SetProgress set the progress
func (e *Execution) SetProgress(progress int, message string) error {
	// Update execution progress
	e.Progress = progress

	// Save to database (gracefully handle database closure)
	err := SaveExecution(e)
	if err != nil {
		log.Warn("Failed to update execution progress (database may be closed): %v", err)
	}

	// Log progress update (gracefully handle database closure)
	if logErr := e.Info("Progress: %d%% - %s", progress, message); logErr != nil {
		log.Warn("Failed to log progress update (database may be closed): %v", logErr)
	}

	return nil // Don't propagate database errors as they might be due to test cleanup
}
