package standard

import (
	"fmt"

	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunTasks executes P2: Tasks phase
// Calls the Tasks Agent to break down goals into executable tasks
//
// Input:
//   - Goals (from P1) with markdown content
//   - Available resources (Agents, MCP tools, KB, DB)
//
// Output:
//   - List of Task objects with executor assignments, expected outputs, and validation rules
func (e *Executor) RunTasks(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	// Get robot for resources
	robot := exec.GetRobot()
	if robot == nil {
		return fmt.Errorf("robot not found in execution")
	}

	// Validate: Goals must exist (from P1)
	if exec.Goals == nil || exec.Goals.Content == "" {
		return fmt.Errorf("goals not available for task planning")
	}

	// Get agent ID for tasks phase
	agentID := "__yao.tasks" // default
	if robot.Config != nil && robot.Config.Resources != nil {
		agentID = robot.Config.Resources.GetPhaseAgent(robottypes.PhaseTasks)
	}

	// Build prompt with goals and available resources
	formatter := NewInputFormatter()
	userContent := formatter.FormatGoals(exec.Goals, robot)

	if userContent == "" {
		return fmt.Errorf("tasks agent (%s) received empty input for task planning", agentID)
	}

	// Call agent
	caller := NewAgentCaller()
	result, err := caller.CallWithMessages(ctx, agentID, userContent)
	if err != nil {
		return fmt.Errorf("tasks agent (%s) call failed: %w", agentID, err)
	}

	// Parse response as JSON
	// Tasks Agent returns: { "tasks": [...] }
	data, err := result.GetJSON()
	if err != nil {
		return fmt.Errorf("tasks agent (%s) returned invalid JSON: %w", agentID, err)
	}

	// Extract tasks array
	tasksData, ok := data["tasks"].([]interface{})
	if !ok || len(tasksData) == 0 {
		return fmt.Errorf("tasks agent (%s) returned no tasks", agentID)
	}

	// Parse tasks
	tasks, err := ParseTasks(tasksData)
	if err != nil {
		return fmt.Errorf("tasks agent (%s) returned invalid task structure: %w", agentID, err)
	}

	// Validate tasks
	if err := ValidateTasks(tasks); err != nil {
		return fmt.Errorf("tasks validation failed: %w", err)
	}

	exec.Tasks = tasks
	return nil
}

// ParseTasks converts raw JSON array to []Task
// Tasks are sorted by Order field after parsing
func ParseTasks(data []interface{}) ([]robottypes.Task, error) {
	tasks := make([]robottypes.Task, 0, len(data))

	for i, item := range data {
		taskMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("task %d is not a valid object", i)
		}

		task, err := ParseTask(taskMap, i)
		if err != nil {
			return nil, fmt.Errorf("task %d: %w", i, err)
		}

		tasks = append(tasks, *task)
	}

	// Sort tasks by Order field to ensure correct execution sequence
	SortTasksByOrder(tasks)

	return tasks, nil
}

// ParseTask converts a map to Task struct
func ParseTask(data map[string]interface{}, index int) (*robottypes.Task, error) {
	task := &robottypes.Task{
		Status: robottypes.TaskPending,
		Order:  index,
	}

	// Required: id
	if id, ok := data["id"].(string); ok && id != "" {
		task.ID = id
	} else {
		task.ID = fmt.Sprintf("task-%03d", index+1)
	}

	// Required: executor_type
	if execType, ok := data["executor_type"].(string); ok {
		task.ExecutorType = ParseExecutorType(execType)
	} else {
		return nil, fmt.Errorf("missing executor_type")
	}

	// Required: executor_id
	if execID, ok := data["executor_id"].(string); ok && execID != "" {
		task.ExecutorID = execID
	} else {
		return nil, fmt.Errorf("missing executor_id")
	}

	// Optional: goal_ref
	if goalRef, ok := data["goal_ref"].(string); ok {
		task.GoalRef = goalRef
	}

	// Optional: source (default to auto)
	if source, ok := data["source"].(string); ok {
		task.Source = robottypes.TaskSource(source)
	} else {
		task.Source = robottypes.TaskSourceAuto
	}

	// Optional: order (override default)
	if order, ok := data["order"].(float64); ok {
		task.Order = int(order)
	}

	// Optional: messages (task instructions)
	if messages, ok := data["messages"].([]interface{}); ok {
		task.Messages = ParseMessages(messages)
	}

	// Optional: description -> convert to message if no messages
	if len(task.Messages) == 0 {
		if desc, ok := data["description"].(string); ok && desc != "" {
			task.Messages = []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: desc},
			}
		}
	}

	// Optional: args
	if args, ok := data["args"].([]interface{}); ok {
		task.Args = make([]any, len(args))
		copy(task.Args, args)
	}

	// Optional: expected_output (for P3 validation)
	if expectedOutput, ok := data["expected_output"].(string); ok {
		task.ExpectedOutput = expectedOutput
	}

	// Optional: validation_rules (for P3 validation)
	if rules, ok := data["validation_rules"].([]interface{}); ok {
		task.ValidationRules = make([]string, 0, len(rules))
		for _, r := range rules {
			if s, ok := r.(string); ok {
				task.ValidationRules = append(task.ValidationRules, s)
			}
		}
	}

	return task, nil
}

// ParseMessages converts raw message array to []Message
func ParseMessages(data []interface{}) []agentcontext.Message {
	messages := make([]agentcontext.Message, 0, len(data))

	for _, item := range data {
		msgMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		msg := agentcontext.Message{}

		// Role
		if role, ok := msgMap["role"].(string); ok {
			msg.Role = agentcontext.MessageRole(role)
		} else {
			msg.Role = agentcontext.RoleUser
		}

		// Content
		if content, ok := msgMap["content"].(string); ok {
			msg.Content = content
		} else if content, ok := msgMap["content"]; ok {
			// Handle non-string content (multimodal)
			msg.Content = content
		}

		if msg.Content != nil {
			messages = append(messages, msg)
		}
	}

	return messages
}

// ParseExecutorType converts string to ExecutorType
func ParseExecutorType(s string) robottypes.ExecutorType {
	switch s {
	case "agent", "assistant":
		return robottypes.ExecutorAssistant
	case "mcp":
		return robottypes.ExecutorMCP
	case "process":
		return robottypes.ExecutorProcess
	default:
		return robottypes.ExecutorAssistant // default to assistant
	}
}

// ValidateTasks validates the task list
func ValidateTasks(tasks []robottypes.Task) error {
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks generated")
	}

	seenIDs := make(map[string]bool)

	for i, task := range tasks {
		// Check unique ID
		if seenIDs[task.ID] {
			return fmt.Errorf("task %d: duplicate task ID '%s'", i, task.ID)
		}
		seenIDs[task.ID] = true

		// Check executor
		if task.ExecutorID == "" {
			return fmt.Errorf("task %d (%s): missing executor_id", i, task.ID)
		}

		// Check messages or description
		if len(task.Messages) == 0 {
			return fmt.Errorf("task %d (%s): missing messages or description", i, task.ID)
		}

		// Note: Executor existence is NOT validated here
		// - ValidateExecutorExists() can be called separately if needed
		// - Unknown executors will fail at P3 runtime with clear error message
		// - This allows flexibility for dynamically registered executors

		// Note: Validation rules are optional
		// - P3 can still do basic validation without explicit rules
	}

	return nil
}

// ValidateTasksWithResources validates tasks and checks executor existence
// Returns a list of warnings for unknown executors (does not fail)
func ValidateTasksWithResources(tasks []robottypes.Task, robot *robottypes.Robot) (warnings []string, err error) {
	// First do basic validation
	if err := ValidateTasks(tasks); err != nil {
		return nil, err
	}

	// Then check executor existence (warnings only)
	for _, task := range tasks {
		if !ValidateExecutorExists(task.ExecutorID, task.ExecutorType, robot) {
			warnings = append(warnings, fmt.Sprintf(
				"task %s: executor '%s' (%s) not found in available resources",
				task.ID, task.ExecutorID, task.ExecutorType,
			))
		}
	}

	return warnings, nil
}

// IsValidExecutorType checks if the executor type is valid
func IsValidExecutorType(t robottypes.ExecutorType) bool {
	switch t {
	case robottypes.ExecutorAssistant, robottypes.ExecutorMCP, robottypes.ExecutorProcess:
		return true
	default:
		return false
	}
}

// SortTasksByOrder sorts tasks by their Order field (ascending)
// This ensures tasks are executed in the correct sequence regardless of
// the order they appear in the LLM response
func SortTasksByOrder(tasks []robottypes.Task) {
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].Order < tasks[i].Order {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

// ValidateExecutorExists checks if the executor ID exists in available resources
// This is an optional validation - tasks with unknown executors will still be created
// but may fail during P3 execution
func ValidateExecutorExists(executorID string, executorType robottypes.ExecutorType, robot *robottypes.Robot) bool {
	if robot == nil || robot.Config == nil || robot.Config.Resources == nil {
		return true // Skip validation if no resources configured
	}

	switch executorType {
	case robottypes.ExecutorAssistant:
		for _, agent := range robot.Config.Resources.Agents {
			if agent == executorID {
				return true
			}
		}
		return false

	case robottypes.ExecutorMCP:
		for _, mcp := range robot.Config.Resources.MCP {
			if mcp.ID == executorID {
				return true
			}
		}
		return false

	case robottypes.ExecutorProcess:
		// Process executors are not validated against resources
		// They are validated at runtime by the Yao process system
		return true
	}

	return false
}
