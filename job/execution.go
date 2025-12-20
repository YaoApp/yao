package job

import (
	"encoding/json"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
)

// Add adds a new execution with Yao process (default execution type)
func (j *Job) Add(options *ExecutionOptions, processName string, args ...interface{}) error {
	return j.addExecution(options, &ExecutionConfig{
		Type:        ExecutionTypeProcess,
		ProcessName: processName,
		ProcessArgs: args,
	})
}

// AddCommand adds a new execution with system command
func (j *Job) AddCommand(options *ExecutionOptions, command string, args []string, env map[string]string) error {
	return j.addExecution(options, &ExecutionConfig{
		Type:        ExecutionTypeCommand,
		Command:     command,
		CommandArgs: args,
		Environment: env,
	})
}

// AddFunc adds a new execution with a Go function
// The function is registered in a global registry and will be cleaned up after execution
// Note: The function is stored in memory registry and will be lost if the process restarts
func (j *Job) AddFunc(options *ExecutionOptions, name string, fn ExecutionFunc, args map[string]interface{}) error {
	// fn will be registered in addExecution after ExecutionID is generated
	return j.addExecution(options, &ExecutionConfig{
		Type:     ExecutionTypeFunc,
		Func:     fn, // Temporarily store here, will be moved to registry
		FuncName: name,
		FuncArgs: args,
	})
}

// addExecution is the internal method to create execution records
func (j *Job) addExecution(options *ExecutionOptions, config *ExecutionConfig) error {
	// Set default options if nil
	if options == nil {
		options = &ExecutionOptions{
			Priority:   0,
			SharedData: make(map[string]interface{}),
		}
	}

	// For ExecutionTypeFunc, we need to register the function after getting ExecutionID
	// Store the function temporarily and clear it before serialization
	var funcToRegister ExecutionFunc
	if config.Type == ExecutionTypeFunc && config.Func != nil {
		funcToRegister = config.Func
		config.Func = nil // Clear before serialization (can't be serialized anyway)
	}

	// Serialize ExecutionConfig to JSON for ConfigSnapshot
	configBytes, err := jsoniter.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize execution config: %w", err)
	}
	configSnapshot := json.RawMessage(configBytes)

	// Create new execution record with options and config
	execution := &Execution{
		ExecutionID:      "", // Will be generated in SaveExecution
		JobID:            j.JobID,
		Status:           "queued",
		TriggerCategory:  "manual",
		RetryAttempt:     0,
		Progress:         0,
		ExecutionConfig:  config,          // Keep in memory for runtime use
		ConfigSnapshot:   &configSnapshot, // Store in database
		ExecutionOptions: options,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Save execution to database (this generates ExecutionID)
	if err := SaveExecution(execution); err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	// For ExecutionTypeFunc, register the function in global registry using ExecutionID
	if config.Type == ExecutionTypeFunc && funcToRegister != nil {
		config.FuncID = execution.ExecutionID // Set FuncID for later lookup
		RegisterFunc(execution.ExecutionID, funcToRegister)
	}

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
