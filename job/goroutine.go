package job

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
)

// Goroutine the goroutine mode
type Goroutine struct{}

// ExecuteYaoProcess executes a Yao process using goroutine mode (process API)
func (g *Goroutine) ExecuteYaoProcess(ctx context.Context, work *WorkRequest, progress *Progress) error {
	config := work.Execution.ExecutionConfig

	work.Execution.Info("Executing Yao process: %s (goroutine mode)", config.ProcessName)

	// Create process with context
	proc := process.NewWithContext(ctx, config.ProcessName, config.ProcessArgs...)

	// Set shared data from ExecutionOptions to process context
	if work.Execution.ExecutionOptions != nil && work.Execution.ExecutionOptions.SharedData != nil {
		// SharedData itself is the Global context
		proc.WithGlobal(work.Execution.ExecutionOptions.SharedData)

		// Check if there's a 'sid' field in SharedData for session context
		if sidValue, exists := work.Execution.ExecutionOptions.SharedData["sid"]; exists {
			if sid, ok := sidValue.(string); ok {
				proc.WithSID(sid)
			}
		}
	}

	// Set callback function to handle real-time progress updates
	proc.WithCallback(func(process *process.Process, data map[string]interface{}) error {
		if data == nil {
			return nil
		}

		// Check if this is a progress update
		if dataType, ok := data["type"].(string); ok && dataType == "progress" {
			// Extract progress and message using helper function
			progressInt, message := extractProgressData(data)

			// Update execution progress
			if progressInt >= 0 {
				work.Execution.Progress = progressInt
			}

			// Log progress message
			if message != "" {
				work.Execution.Info("Progress update: %s (%.1f%%)", message, float64(work.Execution.Progress))
			}

			// Update progress tracker using Set method
			if progress != nil && (progressInt >= 0 || message != "") {
				progress.Set(progressInt, message)
			}

			// Save progress update to database
			if saveErr := SaveExecution(work.Execution); saveErr != nil {
				work.Execution.Error("Failed to save progress update: %s", saveErr.Error())
				return saveErr
			}
		}

		return nil
	})

	// Execute the process
	err := proc.Execute()

	// Always get result and release resources, even if there was an error
	result := proc.Value()
	proc.Release()

	if err != nil {
		work.Execution.Error("Yao process failed: %s", err.Error())

		// Update execution with error info
		work.Execution.Status = "failed"
		if saveErr := SaveExecution(work.Execution); saveErr != nil {
			work.Execution.Error("Failed to save execution error: %s", saveErr.Error())
		}

		return fmt.Errorf("yao process execution failed: %w", err)
	}

	work.Execution.Info("Yao process completed successfully, result: %v", result)

	// Update execution with success result
	work.Execution.Status = "completed"
	work.Execution.Progress = 100
	if result != nil {
		if resultBytes, err := jsoniter.Marshal(result); err == nil {
			work.Execution.Result = (*json.RawMessage)(&resultBytes)
		}
	}

	if saveErr := SaveExecution(work.Execution); saveErr != nil {
		work.Execution.Error("Failed to save execution result: %s", saveErr.Error())
		return fmt.Errorf("failed to save execution result: %w", saveErr)
	}

	return nil
}

// ExecuteSystemCommand executes a system command using goroutine mode
func (g *Goroutine) ExecuteSystemCommand(ctx context.Context, work *WorkRequest, progress *Progress) error {
	config := work.Execution.ExecutionConfig

	work.Execution.Info("Executing command: %s (goroutine mode)", config.Command)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, config.Command, config.CommandArgs...)

	// Set environment variables if provided
	if len(config.Environment) > 0 {
		env := os.Environ()
		for key, value := range config.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Add shared data as environment variables
	if work.Execution.ExecutionOptions != nil && work.Execution.ExecutionOptions.SharedData != nil {
		env := cmd.Env
		if env == nil {
			env = os.Environ()
		}
		for key, value := range work.Execution.ExecutionOptions.SharedData {
			env = append(env, fmt.Sprintf("YAO_JOB_SHARED_%s=%v", key, value))
		}
		cmd.Env = env
	}

	// Execute command with context cancellation support
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it was cancelled
		if ctx.Err() != nil {
			work.Execution.Warn("Command cancelled: %s", ctx.Err().Error())
			work.Execution.Status = "cancelled"
		} else {
			work.Execution.Error("Command failed: %s, output: %s", err.Error(), string(output))
			work.Execution.Status = "failed"

			// Store error output
			if len(output) > 0 {
				errorInfo := map[string]interface{}{
					"error":  err.Error(),
					"output": string(output),
				}
				if errorBytes, jsonErr := jsoniter.Marshal(errorInfo); jsonErr == nil {
					work.Execution.ErrorInfo = (*json.RawMessage)(&errorBytes)
				}
			}
		}

		// Save execution status
		if saveErr := SaveExecution(work.Execution); saveErr != nil {
			work.Execution.Error("Failed to save execution error: %s", saveErr.Error())
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("command execution failed: %v", err)
	}

	work.Execution.Info("Command completed successfully, output: %s", string(output))

	// Update execution with success result
	work.Execution.Status = "completed"
	work.Execution.Progress = 100
	if len(output) > 0 {
		result := map[string]interface{}{
			"output": string(output),
		}
		if resultBytes, err := jsoniter.Marshal(result); err == nil {
			work.Execution.Result = (*json.RawMessage)(&resultBytes)
		}
	}

	if saveErr := SaveExecution(work.Execution); saveErr != nil {
		work.Execution.Error("Failed to save execution result: %s", saveErr.Error())
		return fmt.Errorf("failed to save execution result: %w", saveErr)
	}

	return nil
}

// UpdateExecutionProgress updates execution progress from external callback
// This function can be called by system commands via HTTP API to report progress
func UpdateExecutionProgress(executionID string, progressData map[string]interface{}) error {
	// Load execution from database
	execution, err := GetExecution(executionID, model.QueryParam{})
	if err != nil {
		return fmt.Errorf("failed to load execution: %w", err)
	}

	if execution == nil {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	// Extract progress and message from callback data
	if progressVal, exists := progressData["progress"]; exists {
		if progress, ok := progressVal.(float64); ok {
			execution.Progress = int(progress)
		} else if progress, ok := progressVal.(int); ok {
			execution.Progress = progress
		}
	}

	if messageVal, exists := progressData["message"]; exists {
		if message, ok := messageVal.(string); ok {
			execution.Info("Progress update: %s (%.1f%%)", message, float64(execution.Progress))
		}
	}

	// Save updated execution to database
	if saveErr := SaveExecution(execution); saveErr != nil {
		return fmt.Errorf("failed to save progress update: %w", saveErr)
	}

	return nil
}

// ExecuteFunc executes a Go function using goroutine mode
func (g *Goroutine) ExecuteFunc(ctx context.Context, work *WorkRequest, progress *Progress) error {
	config := work.Execution.ExecutionConfig

	// Get function from global registry using FuncID (ExecutionID)
	funcID := config.FuncID
	if funcID == "" {
		funcID = work.Execution.ExecutionID // Fallback to ExecutionID
	}

	fn, ok := GetFunc(funcID)
	if !ok || fn == nil {
		return fmt.Errorf("execution function not found in registry (funcID: %s)", funcID)
	}

	// Ensure cleanup after execution (success or failure)
	defer UnregisterFunc(funcID)

	funcName := config.FuncName
	if funcName == "" {
		funcName = "anonymous"
	}

	work.Execution.Info("Executing function: %s (goroutine mode, funcID: %s)", funcName, funcID)

	// Create execution context
	execCtx := &ExecutionContext{
		Ctx:       ctx,
		Execution: work.Execution,
		Args:      config.FuncArgs,
	}

	// Execute the function
	err := fn(execCtx)
	if err != nil {
		// Check if it was cancelled
		if ctx.Err() != nil {
			work.Execution.Warn("Function cancelled: %s", ctx.Err().Error())
			work.Execution.Status = "cancelled"
		} else {
			work.Execution.Error("Function failed: %s", err.Error())
			work.Execution.Status = "failed"

			// Store error info
			errorInfo := map[string]interface{}{
				"error":     err.Error(),
				"func_name": funcName,
			}
			if errorBytes, jsonErr := jsoniter.Marshal(errorInfo); jsonErr == nil {
				work.Execution.ErrorInfo = (*json.RawMessage)(&errorBytes)
			}
		}

		// Save execution status
		if saveErr := SaveExecution(work.Execution); saveErr != nil {
			work.Execution.Error("Failed to save execution error: %s", saveErr.Error())
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("function execution failed: %w", err)
	}

	work.Execution.Info("Function completed successfully (funcID: %s)", funcID)

	// Update execution with success result
	work.Execution.Status = "completed"
	work.Execution.Progress = 100

	if saveErr := SaveExecution(work.Execution); saveErr != nil {
		work.Execution.Error("Failed to save execution result: %s", saveErr.Error())
		return fmt.Errorf("failed to save execution result: %w", saveErr)
	}

	return nil
}

// extractProgressData extracts progress and message from callback data
func extractProgressData(data map[string]interface{}) (int, string) {
	var progressInt int = -1 // Default to -1 to indicate no progress value
	var message string

	// Extract progress value with type assertion
	if progressVal, exists := data["progress"]; exists {
		switch v := progressVal.(type) {
		case float64:
			progressInt = int(v)
		case int:
			progressInt = v
		case int32:
			progressInt = int(v)
		case int64:
			progressInt = int(v)
		}
	}

	// Extract message with type assertion
	if messageVal, exists := data["message"]; exists {
		if msg, ok := messageVal.(string); ok {
			message = msg
		}
	}

	return progressInt, message
}
