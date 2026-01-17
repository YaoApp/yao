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

// ExecuteWithRetry executes a task with retry mechanism
func (r *Runner) ExecuteWithRetry(task *robottypes.Task, taskCtx *RunnerContext) *robottypes.TaskResult {
	startTime := time.Now()

	result := &robottypes.TaskResult{
		TaskID: task.ID,
	}

	var lastOutput interface{}
	var lastValidation *robottypes.ValidationResult
	var allErrors []string // Collect all errors for debugging

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Execute the task
		output, err := r.ExecuteTask(task, taskCtx, lastValidation)
		if err != nil {
			// Execution error - don't retry, return immediately
			// (Retries are only for validation failures, not execution errors)
			errMsg := fmt.Sprintf("execution failed on attempt %d: %s", attempt+1, err.Error())
			allErrors = append(allErrors, errMsg)
			result.Success = false
			result.Error = strings.Join(allErrors, "; ")
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		lastOutput = output
		result.Output = output

		// Validate the result (reuse validator instance)
		validation := r.validator.Validate(task, output)
		lastValidation = validation
		result.Validation = validation

		// Check if validation passed (unified logic)
		validationPassed := validation.Passed || validation.Score >= r.config.ValidationThreshold
		if validationPassed {
			result.Success = true
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		// Validation failed - check if we should retry
		if !r.config.RetryOnValidationFailure || attempt >= r.config.MaxRetries {
			break
		}

		// Prepare for retry with validation feedback
		// The next iteration will include validation issues in the context
	}

	// All retries exhausted
	result.Success = false
	result.Output = lastOutput
	result.Validation = lastValidation
	result.Duration = time.Since(startTime).Milliseconds()

	if len(allErrors) > 0 {
		result.Error = strings.Join(allErrors, "; ")
	} else if lastValidation != nil {
		result.Error = fmt.Sprintf("validation failed after %d attempts: %v",
			r.config.MaxRetries+1, lastValidation.Issues)
	}

	return result
}

// ExecuteTask executes a single task based on its executor type
func (r *Runner) ExecuteTask(task *robottypes.Task, taskCtx *RunnerContext, prevValidation *robottypes.ValidationResult) (interface{}, error) {
	switch task.ExecutorType {
	case robottypes.ExecutorAssistant:
		return r.ExecuteAssistantTask(task, taskCtx, prevValidation)
	case robottypes.ExecutorMCP:
		return r.ExecuteMCPTask(task, taskCtx)
	case robottypes.ExecutorProcess:
		return r.ExecuteProcessTask(task, taskCtx)
	default:
		return nil, fmt.Errorf("unknown executor type: %s", task.ExecutorType)
	}
}

// ExecuteAssistantTask executes a task using an AI assistant
// Supports multi-turn conversation for complex tasks
func (r *Runner) ExecuteAssistantTask(task *robottypes.Task, taskCtx *RunnerContext, prevValidation *robottypes.ValidationResult) (interface{}, error) {
	// Build messages for the assistant
	messages := r.BuildAssistantMessages(task, taskCtx, prevValidation)

	// Create conversation for multi-turn support
	chatID := fmt.Sprintf("robot-%s-task-%s", r.robot.MemberID, task.ID)
	conv := NewConversation(task.ExecutorID, chatID, r.config.MaxTurnsPerTask)

	// Add system prompt if available
	if taskCtx.SystemPrompt != "" {
		conv.WithSystemPrompt(taskCtx.SystemPrompt)
	}

	// First turn: send the task
	firstInput := r.FormatMessagesAsText(messages)
	turnResult, err := conv.Turn(r.ctx, firstInput)
	if err != nil {
		return nil, fmt.Errorf("assistant call failed: %w", err)
	}

	// Check if the assistant needs more information (multi-turn)
	// We detect this by checking if the response indicates incompleteness
	// or if there are tool calls that need results
	response := turnResult.Result

	// For simple tasks, return the result directly
	if response.Response == nil || len(response.Response.Tools) == 0 {
		// Try to extract structured output
		if data, err := response.GetJSON(); err == nil {
			return data, nil
		}
		// Return text content
		return response.GetText(), nil
	}

	// Handle multi-turn conversation with auto-reply simulation
	// Similar to the test framework's dynamic runner
	for turn := 2; turn <= r.config.MaxTurnsPerTask; turn++ {
		// Check if we have a complete response
		if r.IsResponseComplete(response) {
			break
		}

		// Generate auto-reply based on tool results or context
		autoReply := r.GenerateAutoReply(response, task)
		if autoReply == "" {
			break // No more input needed
		}

		// Continue conversation
		turnResult, err = conv.Turn(r.ctx, autoReply)
		if err != nil {
			return nil, fmt.Errorf("assistant turn %d failed: %w", turn, err)
		}
		response = turnResult.Result
	}

	// Extract final output
	if data, err := response.GetJSON(); err == nil {
		return data, nil
	}
	return response.GetText(), nil
}

// ExecuteMCPTask executes a task using an MCP tool
// ExecutorID format: "mcpClientID.toolName" (e.g., "filesystem.read_file")
func (r *Runner) ExecuteMCPTask(task *robottypes.Task, taskCtx *RunnerContext) (interface{}, error) {
	// Parse MCP executor ID (format: clientID.toolName)
	parts := strings.SplitN(task.ExecutorID, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid MCP executor ID: %s (expected format: clientID.toolName)", task.ExecutorID)
	}

	clientID, toolName := parts[0], parts[1]

	// Get MCP client
	client, err := mcp.Select(clientID)
	if err != nil {
		return nil, fmt.Errorf("MCP client not found: %s: %w", clientID, err)
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
	result, err := client.CallTool(r.ctx.Context, toolName, args)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
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
func (r *Runner) BuildAssistantMessages(task *robottypes.Task, taskCtx *RunnerContext, prevValidation *robottypes.ValidationResult) []agentcontext.Message {
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

	// Add validation feedback if this is a retry
	if prevValidation != nil && !prevValidation.Passed {
		feedbackMsg := r.FormatValidationFeedback(prevValidation)
		messages = append(messages, agentcontext.Message{
			Role:    agentcontext.RoleUser,
			Content: feedbackMsg,
		})
	}

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

// FormatValidationFeedback formats validation feedback for retry
func (r *Runner) FormatValidationFeedback(validation *robottypes.ValidationResult) string {
	var sb strings.Builder
	sb.WriteString("## Validation Feedback\n\n")
	sb.WriteString("Your previous response did not pass validation. Please address the following issues:\n\n")

	if len(validation.Issues) > 0 {
		sb.WriteString("### Issues\n")
		for _, issue := range validation.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString("\n")
	}

	if len(validation.Suggestions) > 0 {
		sb.WriteString("### Suggestions\n")
		for _, suggestion := range validation.Suggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Please provide an improved response that addresses these issues.\n")

	return sb.String()
}

// IsResponseComplete checks if an assistant response is complete
// (no pending tool calls, has content)
func (r *Runner) IsResponseComplete(result *CallResult) bool {
	if result == nil || result.Response == nil {
		return true
	}

	// If there are tool calls, check if all have results
	if len(result.Response.Tools) > 0 {
		for _, tool := range result.Response.Tools {
			if tool.Result == nil {
				return false // Still waiting for tool results
			}
		}
		// All tools have results - response is complete
		return true
	}

	// No tools - check if there's content
	return result.Content != "" || result.Next != nil
}

// GenerateAutoReply generates an automatic reply for multi-turn conversation
// This simulates user responses when the assistant needs more information
func (r *Runner) GenerateAutoReply(result *CallResult, task *robottypes.Task) string {
	if result == nil || result.Response == nil {
		return ""
	}

	// If there are tool results, format them as the reply
	if len(result.Response.Tools) > 0 {
		var replies []string
		for _, tool := range result.Response.Tools {
			if tool.Result != nil {
				resultJSON, err := json.Marshal(tool.Result)
				if err == nil {
					replies = append(replies, fmt.Sprintf("Tool %s result: %s", tool.Tool, string(resultJSON)))
				}
			}
		}
		if len(replies) > 0 {
			return fmt.Sprintf("Tool execution results:\n%s\n\nPlease continue with the task.", strings.Join(replies, "\n"))
		}
	}

	// If the response asks for clarification, provide generic guidance
	// Use case-insensitive matching
	content := strings.ToLower(result.GetText())
	clarificationKeywords := []string{"need more", "clarify", "please provide", "what", "which"}
	for _, keyword := range clarificationKeywords {
		if strings.Contains(content, keyword) {
			return fmt.Sprintf("Please proceed with the task as best as you can based on the available information. "+
				"The expected output is: %s", task.ExpectedOutput)
		}
	}

	return ""
}
