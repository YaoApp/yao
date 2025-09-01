package job

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/config"
)

// Process the process mode
type Process struct{}

// ExecuteYaoProcess executes a Yao process using independent process mode (yao run command)
func (p *Process) ExecuteYaoProcess(ctx context.Context, work *WorkRequest, progress *Progress) error {
	execConfig := work.Execution.ExecutionConfig

	work.Execution.Info("Executing Yao process: %s (process mode)", execConfig.ProcessName)

	// Prepare yao run command arguments
	args := []string{"run", execConfig.ProcessName}

	// Convert and add process arguments using the proper conversion function
	convertedArgs := convertArgsForYaoRun(execConfig.ProcessArgs)
	args = append(args, convertedArgs...)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, "yao", args...)

	// Set working directory to Yao application root
	if config.Conf.Root != "" {
		cmd.Dir = config.Conf.Root
	} else {
		// Fallback to current directory if config is not available
		cmd.Dir, _ = os.Getwd()
	}

	// Set environment variables
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("YAO_JOB_ID=%s", work.Job.JobID),
		fmt.Sprintf("YAO_EXECUTION_ID=%s", work.Execution.ExecutionID),
	)

	// Add shared data as environment variables
	if work.Execution.ExecutionOptions != nil && work.Execution.ExecutionOptions.SharedData != nil {
		for key, value := range work.Execution.ExecutionOptions.SharedData {
			if valueBytes, err := jsoniter.Marshal(value); err == nil {
				env = append(env, fmt.Sprintf("YAO_JOB_SHARED_%s=%s", key, string(valueBytes)))
			} else {
				env = append(env, fmt.Sprintf("YAO_JOB_SHARED_%s=%v", key, value))
			}
		}
	}
	cmd.Env = env

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it was cancelled
		if ctx.Err() != nil {
			work.Execution.Warn("Yao process cancelled: %s", ctx.Err().Error())
			work.Execution.Status = "cancelled"
		} else {
			work.Execution.Error("Yao process failed: %s, output: %s", err.Error(), string(output))
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
		return fmt.Errorf("yao process execution failed: %v", err)
	}

	work.Execution.Info("Yao process completed successfully, output: %s", string(output))

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

// ExecuteSystemCommand executes a system command using independent process mode
func (p *Process) ExecuteSystemCommand(ctx context.Context, work *WorkRequest, progress *Progress) error {
	execConfig := work.Execution.ExecutionConfig

	work.Execution.Info("Executing command: %s (process mode)", execConfig.Command)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, execConfig.Command, execConfig.CommandArgs...)

	// Set working directory to Yao application root directory
	if config.Conf.Root != "" {
		cmd.Dir = config.Conf.Root
	} else {
		// Fallback to current directory
		cmd.Dir, _ = os.Getwd()
	}

	// Set environment variables
	env := os.Environ()
	if len(execConfig.Environment) > 0 {
		for key, value := range execConfig.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Add job context
	env = append(env,
		fmt.Sprintf("YAO_JOB_ID=%s", work.Job.JobID),
		fmt.Sprintf("YAO_EXECUTION_ID=%s", work.Execution.ExecutionID),
	)

	// Add shared data as environment variables
	if work.Execution.ExecutionOptions != nil && work.Execution.ExecutionOptions.SharedData != nil {
		for key, value := range work.Execution.ExecutionOptions.SharedData {
			if valueBytes, err := jsoniter.Marshal(value); err == nil {
				env = append(env, fmt.Sprintf("YAO_JOB_SHARED_%s=%s", key, string(valueBytes)))
			} else {
				env = append(env, fmt.Sprintf("YAO_JOB_SHARED_%s=%v", key, value))
			}
		}
	}
	cmd.Env = env

	// Execute command
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

// convertArgsForYaoRun converts arguments to proper format for yao run command
func convertArgsForYaoRun(args []interface{}) []string {
	result := make([]string, 0, len(args))

	for _, arg := range args {
		if arg == nil {
			result = append(result, "")
			continue
		}

		// Use type assertion for basic types - direct conversion
		switch v := arg.(type) {
		case string:
			result = append(result, v)
		case bool:
			result = append(result, fmt.Sprintf("%t", v))
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			result = append(result, fmt.Sprintf("%d", v))
		case float32, float64:
			result = append(result, fmt.Sprintf("%g", v))
		default:
			// Complex types (slice, map, struct, etc.) need JSON serialization with :: prefix
			if argBytes, err := jsoniter.Marshal(arg); err == nil {
				result = append(result, "::"+string(argBytes))
			} else {
				// Fallback to string representation if JSON marshaling fails
				result = append(result, fmt.Sprintf("%v", arg))
			}
		}
	}

	return result
}
