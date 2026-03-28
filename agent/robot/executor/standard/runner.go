package standard

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/process"
	kunlog "github.com/yaoapp/kun/log"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// Runner handles execution of individual tasks
type Runner struct {
	ctx    *robottypes.Context
	robot  *robottypes.Robot
	config *RunConfig
	chatID string // execution-level chatID for conversation persistence (§8.4)
	log    *execLogger
}

// NewRunner creates a new task runner
func NewRunner(ctx *robottypes.Context, robot *robottypes.Robot, config *RunConfig, chatID string, execID string) *Runner {
	return &Runner{
		ctx:    ctx,
		robot:  robot,
		config: config,
		chatID: chatID,
		log:    newExecLogger(robot, execID),
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

// ExecuteTask executes a single task (V2 simplified: single call, no validation loop).
// Success is determined purely by whether the call itself succeeds without error.
// Quality evaluation is deferred to the Delivery Agent (P4) using ExpectedOutput.
func (r *Runner) ExecuteTask(task *robottypes.Task, taskCtx *RunnerContext) *robottypes.TaskResult {
	startTime := time.Now()

	result := &robottypes.TaskResult{
		TaskID: task.ID,
	}

	// For non-assistant tasks (MCP, Process), single-call execution
	if task.ExecutorType != robottypes.ExecutorAssistant {
		output, err := r.executeNonAssistantTask(task, taskCtx)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("execution failed: %s", err.Error())
			result.Duration = time.Since(startTime).Milliseconds()
			r.log.logTaskOutput(task, result)
			return result
		}

		result.Output = output
		result.Success = true
		result.Duration = time.Since(startTime).Milliseconds()
		r.log.logTaskOutput(task, result)
		return result
	}

	// For assistant tasks, single call via conversation
	output, callResult, err := r.executeAssistantTask(task, taskCtx)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).Milliseconds()
		r.log.logTaskOutput(task, result)
		return result
	}

	result.Output = output
	result.Success = true
	result.Duration = time.Since(startTime).Milliseconds()

	// Check if assistant signals it needs human input (V2 suspend protocol)
	if needInput, question := detectNeedMoreInfo(callResult); needInput {
		result.NeedInput = true
		result.InputQuestion = question
	}

	r.log.logTaskOutput(task, result)
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

// executeAssistantTask executes an assistant task with a single conversation turn.
// Returns the extracted output, the raw CallResult (for need_input detection), and any error.
func (r *Runner) executeAssistantTask(task *robottypes.Task, taskCtx *RunnerContext) (interface{}, *CallResult, error) {
	caller := NewAgentCaller()
	caller.log = r.log
	caller.Connector = r.robot.LanguageModel
	caller.Workspace = r.robot.Workspace
	caller.ChatID = r.chatID

	messages := r.BuildAssistantMessages(task, taskCtx)
	input := r.FormatMessagesAsText(messages)

	if strings.TrimSpace(input) == "" {
		return nil, nil, fmt.Errorf("no valid input messages for task %s", task.ID)
	}

	if taskCtx.SystemPrompt != "" {
		input = "## Context\n\n" + taskCtx.SystemPrompt + "\n\n## Task\n\n" + input
	}

	kunlog.Trace("[robot-runner] executeAssistantTask: task=%s assistant=%s promptLen=%d prevResults=%d",
		task.ID, task.ExecutorID, len(input), len(taskCtx.PreviousResults))

	r.log.logTaskInput(task, input)

	result, err := caller.CallWithMessages(r.ctx, task.ExecutorID, input)
	if err != nil {
		return nil, nil, fmt.Errorf("assistant call failed: %w", err)
	}

	output := r.extractOutput(result)
	return output, result, nil
}

// detectNeedMoreInfo checks if the assistant's response signals it needs human input.
// The protocol: Next hook returns {data: {status: "need_input", question: "..."}}.
// Also handles the unwrapped form {status: "need_input", question: "..."} for robustness.
func detectNeedMoreInfo(result *CallResult) (bool, string) {
	if result == nil || result.Next == nil {
		return false, ""
	}
	m, ok := result.Next.(map[string]interface{})
	if !ok {
		return false, ""
	}

	// Unwrap "data" envelope if present (Next hook standard: {data: {status: ...}})
	if data, ok := m["data"].(map[string]interface{}); ok {
		m = data
	}

	status, _ := m["status"].(string)
	if status != "need_input" {
		return false, ""
	}
	question, _ := m["question"].(string)
	if question == "" {
		question = result.GetText()
	}
	return true, question
}

// extractOutput extracts the output from a CallResult
// Priority: Next hook data > LLM Completion content
// Next is the agent's formal A2A output (could be string, map, array, number, etc.)
// Content is the raw LLM completion text (fallback only when Next is absent)
func (r *Runner) extractOutput(result *CallResult) interface{} {
	if result == nil {
		return nil
	}
	if result.Next != nil {
		return result.Next
	}
	if result.Content != "" {
		return result.Content
	}
	return nil
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

	contextLen := sb.Len()
	kunlog.Trace("[robot-runner] FormatPreviousResultsAsContext: results=%d totalLen=%d", len(results), contextLen)
	return sb.String()
}
