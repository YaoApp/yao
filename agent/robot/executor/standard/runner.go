package standard

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/process"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// Runner handles execution of individual tasks
type Runner struct {
	ctx       *robottypes.Context
	robot     *robottypes.Robot
	config    *RunConfig
	validator *Validator // reusable validator instance
}

// NewRunner creates a new task runner
func NewRunner(ctx *robottypes.Context, robot *robottypes.Robot, config *RunConfig) *Runner {
	return &Runner{
		ctx:       ctx,
		robot:     robot,
		config:    config,
		validator: NewValidator(ctx, robot, config),
	}
}

// RunnerContext provides context for task execution
type RunnerContext struct {
	// PreviousResults contains results from previously executed tasks
	PreviousResults []robottypes.TaskResult

	// Goals contains the goals from P1 (for context)
	Goals *robottypes.Goals

	// SystemPrompt is the robot's system prompt
	SystemPrompt string
}

// BuildTaskContext builds context for a task including previous results
func (r *Runner) BuildTaskContext(exec *robottypes.Execution, taskIndex int) *RunnerContext {
	ctx := &RunnerContext{
		Goals:        exec.Goals,
		SystemPrompt: r.robot.SystemPrompt,
	}

	// Include results from previous tasks (with bounds check)
	if taskIndex > 0 && len(exec.Results) > 0 {
		endIndex := taskIndex
		if endIndex > len(exec.Results) {
			endIndex = len(exec.Results)
		}
		ctx.PreviousResults = exec.Results[:endIndex]
	}

	return ctx
}

// ExecuteWithRetry executes a task with the new multi-turn conversation flow:
// 1. Call assistant and get result
// 2. Validate result (determines: passed, complete, needReply, replyContent)
// 3. If needReply, continue conversation with replyContent
// 4. Repeat until complete or max turns exceeded
func (r *Runner) ExecuteWithRetry(task *robottypes.Task, taskCtx *RunnerContext) *robottypes.TaskResult {
	startTime := time.Now()

	result := &robottypes.TaskResult{
		TaskID: task.ID,
	}

	// For non-assistant tasks (MCP, Process), use simple single-call execution
	if task.ExecutorType != robottypes.ExecutorAssistant {
		output, err := r.executeNonAssistantTask(task, taskCtx)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("execution failed: %s", err.Error())
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		result.Output = output

		// For MCP tasks: only validate structure (no semantic validation needed)
		// MCP tools return structured data - if execution succeeded, the result is valid
		if task.ExecutorType == robottypes.ExecutorMCP {
			validation := r.validateMCPOutput(task, output)
			result.Validation = validation
			result.Success = validation.Passed
			result.Duration = time.Since(startTime).Milliseconds()
			if !result.Success && validation != nil {
				result.Error = fmt.Sprintf("validation failed: %v", validation.Issues)
			}
			return result
		}

		// For Process tasks: use full validation (semantic validation may still be useful)
		validation := r.validator.ValidateWithContext(task, output, nil)
		result.Validation = validation
		// For Process tasks:
		// - No multi-turn conversation, so Complete is determined by validation alone
		// - Success if passed OR score meets threshold (for partial success scenarios)
		result.Success = validation.Complete || (validation.Passed && validation.Score >= r.config.ValidationThreshold)
		result.Duration = time.Since(startTime).Milliseconds()

		if !result.Success && validation != nil {
			result.Error = fmt.Sprintf("validation failed: %v", validation.Issues)
		}
		return result
	}

	// For assistant tasks, use multi-turn conversation flow
	output, validation, err := r.executeAssistantWithMultiTurn(task, taskCtx)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Output = output         // Preserve partial output for debugging
		result.Validation = validation // Preserve validation result for debugging
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	result.Output = output
	result.Validation = validation
	result.Success = validation.Complete && validation.Passed
	result.Duration = time.Since(startTime).Milliseconds()

	if !result.Success && validation != nil {
		result.Error = fmt.Sprintf("task incomplete: %v", validation.Issues)
	}

	return result
}

// executeNonAssistantTask executes MCP or Process tasks (single-call, no multi-turn)
func (r *Runner) executeNonAssistantTask(task *robottypes.Task, taskCtx *RunnerContext) (interface{}, error) {
	switch task.ExecutorType {
	case robottypes.ExecutorMCP:
		return r.ExecuteMCPTask(task, taskCtx)
	case robottypes.ExecutorProcess:
		return r.ExecuteProcessTask(task, taskCtx)
	default:
		return nil, fmt.Errorf("unsupported executor type: %s (expected mcp or process)", task.ExecutorType)
	}
}

// executeAssistantWithMultiTurn executes an assistant task with multi-turn conversation support
// This is the main execution flow for assistant tasks:
// 1. Call assistant and get result
// 2. Validate result (determines: passed, complete, needReply, replyContent)
// 3. If needReply, continue conversation with replyContent
// 4. Repeat until complete or max turns exceeded
func (r *Runner) executeAssistantWithMultiTurn(task *robottypes.Task, taskCtx *RunnerContext) (interface{}, *robottypes.ValidationResult, error) {
	// Create conversation for the entire task execution (shared across all turns)
	chatID := fmt.Sprintf("robot-%s-task-%s", r.robot.MemberID, task.ID)
	conv := NewConversation(task.ExecutorID, chatID, r.config.MaxTurnsPerTask)

	// Add system prompt if available
	if taskCtx.SystemPrompt != "" {
		conv.WithSystemPrompt(taskCtx.SystemPrompt)
	}

	// Build initial messages
	messages := r.BuildAssistantMessages(task, taskCtx)
	input := r.FormatMessagesAsText(messages)

	// Ensure we have valid input for the first turn
	if strings.TrimSpace(input) == "" {
		return nil, nil, fmt.Errorf("no valid input messages for task %s", task.ID)
	}

	var lastOutput interface{}
	var lastValidation *robottypes.ValidationResult
	var lastCallResult *CallResult

	for turn := 1; turn <= r.config.MaxTurnsPerTask; turn++ {
		// Phase 1: Call assistant
		turnResult, err := conv.Turn(r.ctx, input)
		if err != nil {
			return lastOutput, lastValidation, fmt.Errorf("turn %d failed: %w", turn, err)
		}

		lastCallResult = turnResult.Result
		lastOutput = r.extractOutput(lastCallResult)

		// Phase 2: Validate result
		lastValidation = r.validator.ValidateWithContext(task, lastOutput, lastCallResult)

		// Check if complete
		if lastValidation.Complete && lastValidation.Passed {
			return lastOutput, lastValidation, nil // Success!
		}

		// Phase 3: Check if we should continue conversation
		if !lastValidation.NeedReply {
			// No need to continue, but not complete either
			// This could be a validation failure that can't be fixed by conversation
			if lastValidation.Passed {
				// Passed but not complete (e.g., empty output)
				return lastOutput, lastValidation, nil
			}
			// Failed and can't retry
			return lastOutput, lastValidation, fmt.Errorf("validation failed: %v", lastValidation.Issues)
		}

		// Prepare next turn input
		input = lastValidation.ReplyContent
		if input == "" {
			// Fallback: generate default reply
			input = r.generateDefaultReply(lastValidation, task)
		}
	}

	// Max turns exceeded
	if lastValidation == nil {
		lastValidation = &robottypes.ValidationResult{
			Passed:   false,
			Complete: false,
			Issues:   []string{fmt.Sprintf("max turns (%d) exceeded without completion", r.config.MaxTurnsPerTask)},
		}
	} else {
		lastValidation.Issues = append(lastValidation.Issues,
			fmt.Sprintf("max turns (%d) exceeded without completion", r.config.MaxTurnsPerTask))
	}

	return lastOutput, lastValidation, fmt.Errorf("max turns (%d) exceeded without completion", r.config.MaxTurnsPerTask)
}

// extractOutput extracts the output from a CallResult
func (r *Runner) extractOutput(result *CallResult) interface{} {
	if result == nil {
		return nil
	}

	// Try to extract structured JSON output
	if data, err := result.GetJSON(); err == nil {
		return data
	}

	// Fall back to text content
	return result.GetText()
}

// generateDefaultReply generates a default reply when validation doesn't provide one
func (r *Runner) generateDefaultReply(validation *robottypes.ValidationResult, task *robottypes.Task) string {
	var sb strings.Builder

	if len(validation.Issues) > 0 {
		sb.WriteString("Please address the following issues:\n")
		for _, issue := range validation.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString("\n")
	}

	if task.ExpectedOutput != "" {
		sb.WriteString(fmt.Sprintf("Expected output: %s\n", task.ExpectedOutput))
	}

	sb.WriteString("\nPlease provide an improved response.")

	return sb.String()
}

// ExecuteMCPTask executes a task using an MCP tool
// Requires task.MCPServer and task.MCPTool fields to be set
// executor_id is the combined form: "mcp_server.mcp_tool" (e.g., "ark.image.text2img.generate")
func (r *Runner) ExecuteMCPTask(task *robottypes.Task, taskCtx *RunnerContext) (interface{}, error) {
	// Validate MCP-specific fields
	if task.MCPServer == "" || task.MCPTool == "" {
		return nil, fmt.Errorf("MCP task requires mcp_server and mcp_tool fields (executor_id: %s)", task.ExecutorID)
	}

	// Get MCP client
	client, err := mcp.Select(task.MCPServer)
	if err != nil {
		return nil, fmt.Errorf("MCP server not found: %s: %w", task.MCPServer, err)
	}

	// Build arguments map from task.Args
	args := make(map[string]interface{})
	if len(task.Args) > 0 {
		// First argument should be a map of tool arguments
		if argsMap, ok := task.Args[0].(map[string]interface{}); ok {
			args = argsMap
		} else {
			// If not a map, try to convert single argument
			args["input"] = task.Args[0]
		}
	}

	// Call MCP tool
	result, err := client.CallTool(r.ctx.Context, task.MCPTool, args)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed (%s.%s): %w", task.MCPServer, task.MCPTool, err)
	}

	return result, nil
}

// ExecuteProcessTask executes a task using a Yao process
// ExecutorID is the process name (e.g., "models.user.Find", "scripts.myScript.Run")
func (r *Runner) ExecuteProcessTask(task *robottypes.Task, taskCtx *RunnerContext) (interface{}, error) {
	// Create process with task arguments
	proc, err := process.Of(task.ExecutorID, task.Args...)
	if err != nil {
		return nil, fmt.Errorf("process creation failed: %w", err)
	}

	// Set context for timeout and cancellation
	proc.Context = r.ctx.Context

	// Execute the process
	if err := proc.Execute(); err != nil {
		return nil, fmt.Errorf("process execution failed: %w", err)
	}
	defer proc.Release()

	// Return the result
	return proc.Value(), nil
}

// BuildAssistantMessages builds messages for an assistant task
// Note: In the new multi-turn flow, validation feedback is handled via ValidateWithContext.ReplyContent
func (r *Runner) BuildAssistantMessages(task *robottypes.Task, taskCtx *RunnerContext) []agentcontext.Message {
	messages := make([]agentcontext.Message, 0)

	// Add context from previous tasks if available
	if len(taskCtx.PreviousResults) > 0 {
		contextMsg := r.FormatPreviousResultsAsContext(taskCtx.PreviousResults)
		if contextMsg != "" {
			messages = append(messages, agentcontext.Message{
				Role:    agentcontext.RoleUser,
				Content: contextMsg,
			})
		}
	}

	// Add task messages
	messages = append(messages, task.Messages...)

	return messages
}

// FormatMessagesAsText converts messages to a single text string
func (r *Runner) FormatMessagesAsText(messages []agentcontext.Message) string {
	var result string
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case string:
			result += content + "\n\n"
		case []interface{}:
			// Handle multi-part content (e.g., text + images)
			for _, part := range content {
				if textPart, ok := part.(map[string]interface{}); ok {
					if text, ok := textPart["text"].(string); ok {
						result += text + "\n\n"
					}
				}
			}
		default:
			// Try JSON marshaling as fallback
			if content != nil {
				if jsonBytes, err := json.Marshal(content); err == nil {
					result += string(jsonBytes) + "\n\n"
				}
			}
		}
	}
	return result
}

// FormatPreviousResultsAsContext formats previous task results as context
func (r *Runner) FormatPreviousResultsAsContext(results []robottypes.TaskResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Previous Task Results\n\n")
	sb.WriteString("The following tasks have been completed. Use their results as needed:\n\n")

	for _, result := range results {
		sb.WriteString(fmt.Sprintf("### Task: %s\n", result.TaskID))
		if result.Success {
			sb.WriteString("- Status: ✓ Success\n")
		} else {
			sb.WriteString("- Status: ✗ Failed\n")
		}

		if result.Output != nil {
			outputJSON, err := json.MarshalIndent(result.Output, "", "  ")
			if err == nil {
				sb.WriteString(fmt.Sprintf("- Output:\n```json\n%s\n```\n", string(outputJSON)))
			} else {
				sb.WriteString(fmt.Sprintf("- Output: %v\n", result.Output))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// validateMCPOutput performs simple structure validation for MCP task output
// MCP tools return structured data - if execution succeeded, the result is valid
// Only validates that output is non-empty and has expected structure
// Does NOT perform semantic validation (that's for Agent tasks only)
func (r *Runner) validateMCPOutput(task *robottypes.Task, output interface{}) *robottypes.ValidationResult {
	result := &robottypes.ValidationResult{
		Passed:   true,
		Score:    1.0,
		Complete: true,
	}

	// Check if output is nil or empty
	if output == nil {
		result.Passed = false
		result.Score = 0
		result.Complete = false
		result.Issues = append(result.Issues, "MCP tool returned nil output")
		return result
	}

	// Check for empty output based on type
	switch o := output.(type) {
	case string:
		if strings.TrimSpace(o) == "" {
			result.Passed = false
			result.Score = 0
			result.Complete = false
			result.Issues = append(result.Issues, "MCP tool returned empty string")
			return result
		}
	case map[string]interface{}:
		if len(o) == 0 {
			result.Passed = false
			result.Score = 0
			result.Complete = false
			result.Issues = append(result.Issues, "MCP tool returned empty object")
			return result
		}
	case []interface{}:
		if len(o) == 0 {
			result.Passed = false
			result.Score = 0
			result.Complete = false
			result.Issues = append(result.Issues, "MCP tool returned empty array")
			return result
		}
	}

	// MCP execution succeeded with non-empty output - validation passed
	// No semantic validation needed for MCP tools
	return result
}
