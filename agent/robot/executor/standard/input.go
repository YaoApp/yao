package standard

import (
	"encoding/json"
	"fmt"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// InputFormatter provides methods to format input data for assistant prompts
// Each phase has specific input requirements:
// - P0 (Inspiration): ClockContext + Robot identity + Available resources
// - P1 (Goals): InspirationReport/TriggerInput + Robot identity + Available resources
// - P2 (Tasks): Goals + Available resources
// - P3 (Run): Tasks
// - P4 (Delivery): Task results
// - P5 (Learning): Execution summary
type InputFormatter struct{}

// NewInputFormatter creates a new InputFormatter
func NewInputFormatter() *InputFormatter {
	return &InputFormatter{}
}

// FormatClockContext formats ClockContext as user message content
// Used by P0 (Inspiration) phase
func (f *InputFormatter) FormatClockContext(clock *robottypes.ClockContext, robot *robottypes.Robot) string {
	if clock == nil {
		return ""
	}

	var sb strings.Builder

	// Time context section
	sb.WriteString("## Current Time Context\n\n")
	sb.WriteString(fmt.Sprintf("- **Now**: %s\n", clock.Now.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **Day**: %s\n", clock.DayOfWeek))
	sb.WriteString(fmt.Sprintf("- **Date**: %d/%d/%d\n", clock.Year, clock.Month, clock.DayOfMonth))
	sb.WriteString(fmt.Sprintf("- **Week**: %d of year\n", clock.WeekOfYear))
	sb.WriteString(fmt.Sprintf("- **Timezone**: %s\n", clock.TZ))

	// Time markers
	sb.WriteString("\n### Time Markers\n")
	if clock.IsWeekend {
		sb.WriteString("- ✓ Weekend\n")
	}
	if clock.IsMonthStart {
		sb.WriteString("- ✓ Month Start (1st-3rd)\n")
	}
	if clock.IsMonthEnd {
		sb.WriteString("- ✓ Month End (last 3 days)\n")
	}
	if clock.IsQuarterEnd {
		sb.WriteString("- ✓ Quarter End\n")
	}
	if clock.IsYearEnd {
		sb.WriteString("- ✓ Year End\n")
	}

	// Robot identity section (if available)
	if robot != nil && robot.Config != nil && robot.Config.Identity != nil {
		sb.WriteString("\n## Robot Identity\n\n")
		sb.WriteString(fmt.Sprintf("- **Role**: %s\n", robot.Config.Identity.Role))
		if len(robot.Config.Identity.Duties) > 0 {
			sb.WriteString("- **Duties**:\n")
			for _, duty := range robot.Config.Identity.Duties {
				sb.WriteString(fmt.Sprintf("  - %s\n", duty))
			}
		}
		if len(robot.Config.Identity.Rules) > 0 {
			sb.WriteString("- **Rules**:\n")
			for _, rule := range robot.Config.Identity.Rules {
				sb.WriteString(fmt.Sprintf("  - %s\n", rule))
			}
		}
	}

	return sb.String()
}

// FormatRobotIdentity formats robot identity as user message content
// Used to provide context about the robot's role and duties
func (f *InputFormatter) FormatRobotIdentity(robot *robottypes.Robot) string {
	if robot == nil || robot.Config == nil || robot.Config.Identity == nil {
		return ""
	}

	var sb strings.Builder
	identity := robot.Config.Identity

	sb.WriteString("## Robot Identity\n\n")
	sb.WriteString(fmt.Sprintf("- **Role**: %s\n", identity.Role))

	if len(identity.Duties) > 0 {
		sb.WriteString("- **Duties**:\n")
		for _, duty := range identity.Duties {
			sb.WriteString(fmt.Sprintf("  - %s\n", duty))
		}
	}

	if len(identity.Rules) > 0 {
		sb.WriteString("- **Rules**:\n")
		for _, rule := range identity.Rules {
			sb.WriteString(fmt.Sprintf("  - %s\n", rule))
		}
	}

	return sb.String()
}

// FormatAvailableResources formats available resources (agents, MCP tools, KB, DB) as user message content
// Used by P0 (Inspiration) and P1 (Goals) to inform the agent what tools are available
// This is critical for generating achievable goals - without knowing available tools,
// the agent might generate goals that cannot be accomplished
func (f *InputFormatter) FormatAvailableResources(robot *robottypes.Robot) string {
	if robot == nil || robot.Config == nil {
		return ""
	}

	var sb strings.Builder
	hasContent := false

	// Available Agents
	if robot.Config.Resources != nil && len(robot.Config.Resources.Agents) > 0 {
		if !hasContent {
			sb.WriteString("## Available Resources\n\n")
			hasContent = true
		}
		sb.WriteString("### Agents\n")
		sb.WriteString("These are the AI assistants you can delegate tasks to:\n")
		for _, agent := range robot.Config.Resources.Agents {
			sb.WriteString(fmt.Sprintf("- **%s**\n", agent))
		}
		sb.WriteString("\n")
	}

	// Available MCP Tools
	if robot.Config.Resources != nil && len(robot.Config.Resources.MCP) > 0 {
		if !hasContent {
			sb.WriteString("## Available Resources\n\n")
			hasContent = true
		}
		sb.WriteString("### MCP Tools\n")
		sb.WriteString("These are the external tools and services you can use:\n")
		for _, mcp := range robot.Config.Resources.MCP {
			if len(mcp.Tools) > 0 {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", mcp.ID, strings.Join(mcp.Tools, ", ")))
			} else {
				sb.WriteString(fmt.Sprintf("- **%s**: all tools available\n", mcp.ID))
			}
		}
		sb.WriteString("\n")
	}

	// Available Knowledge Base
	if robot.Config.KB != nil && len(robot.Config.KB.Collections) > 0 {
		if !hasContent {
			sb.WriteString("## Available Resources\n\n")
			hasContent = true
		}
		sb.WriteString("### Knowledge Base\n")
		sb.WriteString("You have access to these knowledge collections:\n")
		for _, collection := range robot.Config.KB.Collections {
			sb.WriteString(fmt.Sprintf("- %s\n", collection))
		}
		sb.WriteString("\n")
	}

	// Available Database Models
	if robot.Config.DB != nil && len(robot.Config.DB.Models) > 0 {
		if !hasContent {
			sb.WriteString("## Available Resources\n\n")
			hasContent = true
		}
		sb.WriteString("### Database\n")
		sb.WriteString("You can query these database models:\n")
		for _, model := range robot.Config.DB.Models {
			sb.WriteString(fmt.Sprintf("- %s\n", model))
		}
		sb.WriteString("\n")
	}

	if !hasContent {
		return ""
	}

	sb.WriteString("**Important**: Only plan goals and tasks that can be accomplished with the above resources.\n")
	return sb.String()
}

// FormatInspirationReport formats InspirationReport as user message content
// Used by P1 (Goals) phase when trigger is Clock
func (f *InputFormatter) FormatInspirationReport(report *robottypes.InspirationReport) string {
	if report == nil {
		return ""
	}

	var sb strings.Builder

	// Clock context summary (if available)
	if report.Clock != nil {
		sb.WriteString("## Time Context\n\n")
		sb.WriteString(fmt.Sprintf("- **Time**: %s %s\n", report.Clock.DayOfWeek, report.Clock.Now.Format("15:04")))
		sb.WriteString(fmt.Sprintf("- **Date**: %d/%d/%d\n", report.Clock.Year, report.Clock.Month, report.Clock.DayOfMonth))

		// Add relevant time markers
		var markers []string
		if report.Clock.IsWeekend {
			markers = append(markers, "Weekend")
		}
		if report.Clock.IsMonthStart {
			markers = append(markers, "Month Start")
		}
		if report.Clock.IsMonthEnd {
			markers = append(markers, "Month End")
		}
		if report.Clock.IsQuarterEnd {
			markers = append(markers, "Quarter End")
		}
		if len(markers) > 0 {
			sb.WriteString(fmt.Sprintf("- **Markers**: %s\n", strings.Join(markers, ", ")))
		}
		sb.WriteString("\n")
	}

	// Inspiration content
	if report.Content != "" {
		sb.WriteString("## Inspiration Report\n\n")
		sb.WriteString(report.Content)
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatTriggerInput formats TriggerInput as user message content
// Used by P1 (Goals) phase when trigger is Human or Event
func (f *InputFormatter) FormatTriggerInput(input *robottypes.TriggerInput) string {
	if input == nil {
		return ""
	}

	var sb strings.Builder

	// Human intervention
	if input.Action != "" {
		sb.WriteString("## Human Intervention\n\n")
		sb.WriteString(fmt.Sprintf("- **Action**: %s\n", input.Action))
		if input.UserID != "" {
			sb.WriteString(fmt.Sprintf("- **User**: %s\n", input.UserID))
		}

		// Messages
		if len(input.Messages) > 0 {
			sb.WriteString("\n### User Input\n\n")
			for _, msg := range input.Messages {
				if content, ok := msg.Content.(string); ok {
					sb.WriteString(content)
					sb.WriteString("\n")
				}
			}
		}
		return sb.String()
	}

	// Event trigger
	if input.Source != "" {
		sb.WriteString("## Event Trigger\n\n")
		sb.WriteString(fmt.Sprintf("- **Source**: %s\n", input.Source))
		sb.WriteString(fmt.Sprintf("- **Event Type**: %s\n", input.EventType))

		// Event data
		if input.Data != nil {
			sb.WriteString("\n### Event Data\n\n")
			sb.WriteString("```json\n")
			if data, err := json.MarshalIndent(input.Data, "", "  "); err == nil {
				sb.WriteString(string(data))
			}
			sb.WriteString("\n```\n")
		}
		return sb.String()
	}

	return ""
}

// FormatGoals formats Goals as user message content
// Used by P2 (Tasks) phase
func (f *InputFormatter) FormatGoals(goals *robottypes.Goals, robot *robottypes.Robot) string {
	if goals == nil {
		return ""
	}

	var sb strings.Builder

	// Goals content
	sb.WriteString("## Goals\n\n")
	sb.WriteString(goals.Content)
	sb.WriteString("\n")

	// Delivery target (from P1) - important for task planning
	// Tasks should be designed to produce output suitable for the delivery method
	if goals.Delivery != nil {
		sb.WriteString("\n## Delivery Target\n\n")
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", goals.Delivery.Type))
		if len(goals.Delivery.Recipients) > 0 {
			sb.WriteString(fmt.Sprintf("- **Recipients**: %s\n", strings.Join(goals.Delivery.Recipients, ", ")))
		}
		if goals.Delivery.Format != "" {
			sb.WriteString(fmt.Sprintf("- **Format**: %s\n", goals.Delivery.Format))
		}
		if goals.Delivery.Template != "" {
			sb.WriteString(fmt.Sprintf("- **Template**: %s\n", goals.Delivery.Template))
		}
		sb.WriteString("\n**Note**: Design tasks to produce output suitable for this delivery method.\n")
	}

	// Available resources - reuse FormatAvailableResources for consistency
	resourcesContent := f.FormatAvailableResources(robot)
	if resourcesContent != "" {
		sb.WriteString("\n")
		sb.WriteString(resourcesContent)
	}

	return sb.String()
}

// FormatTasks formats Tasks as user message content
// Used by P3 (Run) phase
func (f *InputFormatter) FormatTasks(tasks []robottypes.Task) string {
	if len(tasks) == 0 {
		return "No tasks to execute."
	}

	var sb strings.Builder

	sb.WriteString("## Tasks to Execute\n\n")
	for i, task := range tasks {
		sb.WriteString(fmt.Sprintf("### Task %d: %s\n\n", i+1, task.ID))
		sb.WriteString(fmt.Sprintf("- **Goal Reference**: %s\n", task.GoalRef))
		sb.WriteString(fmt.Sprintf("- **Source**: %s\n", task.Source))
		sb.WriteString(fmt.Sprintf("- **Executor**: %s (%s)\n", task.ExecutorID, task.ExecutorType))

		// Task content
		if len(task.Messages) > 0 {
			sb.WriteString("\n**Instructions**:\n")
			for _, msg := range task.Messages {
				if content, ok := msg.Content.(string); ok {
					sb.WriteString(content)
					sb.WriteString("\n")
				}
			}
		}

		// Arguments
		if len(task.Args) > 0 {
			sb.WriteString("\n**Arguments**:\n")
			if args, err := json.MarshalIndent(task.Args, "", "  "); err == nil {
				sb.WriteString("```json\n")
				sb.WriteString(string(args))
				sb.WriteString("\n```\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatTaskResults formats TaskResults as user message content
// Used by P4 (Delivery) and P5 (Learning) phases
func (f *InputFormatter) FormatTaskResults(results []robottypes.TaskResult) string {
	if len(results) == 0 {
		return "No task results."
	}

	var sb strings.Builder

	sb.WriteString("## Task Results\n\n")

	successCount := 0
	failCount := 0
	validatedPassedCount := 0
	validatedTotalCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failCount++
		}
		if result.Validation != nil {
			validatedTotalCount++
			if result.Validation.Passed {
				validatedPassedCount++
			}
		}

		sb.WriteString(fmt.Sprintf("### Task: %s\n\n", result.TaskID))
		if result.Success {
			sb.WriteString("- **Status**: ✓ Success\n")
		} else {
			sb.WriteString("- **Status**: ✗ Failed\n")
		}
		sb.WriteString(fmt.Sprintf("- **Duration**: %dms\n", result.Duration))

		// Validation result (P3)
		if result.Validation != nil {
			if result.Validation.Passed {
				sb.WriteString(fmt.Sprintf("- **Validation**: ✓ Passed (score: %.2f)\n", result.Validation.Score))
			} else {
				sb.WriteString("- **Validation**: ✗ Failed\n")
				if len(result.Validation.Issues) > 0 {
					sb.WriteString("  - Issues:\n")
					for _, issue := range result.Validation.Issues {
						sb.WriteString(fmt.Sprintf("    - %s\n", issue))
					}
				}
			}
		}

		// Output
		if result.Output != nil {
			sb.WriteString("\n**Output**:\n")
			if output, err := json.MarshalIndent(result.Output, "", "  "); err == nil {
				sb.WriteString("```json\n")
				sb.WriteString(string(output))
				sb.WriteString("\n```\n")
			} else {
				sb.WriteString(fmt.Sprintf("%v\n", result.Output))
			}
		}

		// Error
		if result.Error != "" {
			sb.WriteString(fmt.Sprintf("\n**Error**: %s\n", result.Error))
		}
		sb.WriteString("\n")
	}

	// Summary
	sb.WriteString(fmt.Sprintf("## Summary\n\n- Total: %d tasks\n- Success: %d\n- Failed: %d\n- Validated: %d/%d\n",
		len(results), successCount, failCount, validatedPassedCount, validatedTotalCount))

	return sb.String()
}

// FormatExecutionSummary formats the entire execution for P5 (Learning) phase
func (f *InputFormatter) FormatExecutionSummary(exec *robottypes.Execution) string {
	if exec == nil {
		return ""
	}

	var sb strings.Builder

	// Execution metadata
	sb.WriteString("## Execution Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", exec.ID))
	sb.WriteString(fmt.Sprintf("- **Trigger**: %s\n", exec.TriggerType))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", exec.Status))
	sb.WriteString(fmt.Sprintf("- **Start Time**: %s\n", exec.StartTime.Format("2006-01-02 15:04:05")))
	if exec.EndTime != nil {
		sb.WriteString(fmt.Sprintf("- **End Time**: %s\n", exec.EndTime.Format("2006-01-02 15:04:05")))
		duration := exec.EndTime.Sub(exec.StartTime)
		sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", duration.String()))
	}
	if exec.Error != "" {
		sb.WriteString(fmt.Sprintf("- **Error**: %s\n", exec.Error))
	}
	sb.WriteString("\n")

	// Inspiration (P0)
	if exec.Inspiration != nil && exec.Inspiration.Content != "" {
		sb.WriteString("## Inspiration (P0)\n\n")
		sb.WriteString(exec.Inspiration.Content)
		sb.WriteString("\n\n")
	}

	// Goals (P1)
	if exec.Goals != nil && exec.Goals.Content != "" {
		sb.WriteString("## Goals (P1)\n\n")
		sb.WriteString(exec.Goals.Content)
		sb.WriteString("\n\n")
	}

	// Tasks (P2)
	if len(exec.Tasks) > 0 {
		sb.WriteString("## Tasks (P2)\n\n")
		for i, task := range exec.Tasks {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s (executor: %s)\n",
				i+1, task.Status, task.ID, task.ExecutorID))
		}
		sb.WriteString("\n")
	}

	// Results (P3)
	if len(exec.Results) > 0 {
		sb.WriteString("## Results (P3)\n\n")
		for _, result := range exec.Results {
			status := "✓"
			if !result.Success {
				status = "✗"
			}
			sb.WriteString(fmt.Sprintf("- %s %s (%dms)\n", status, result.TaskID, result.Duration))
		}
		sb.WriteString("\n")
	}

	// Delivery (P4)
	if exec.Delivery != nil {
		sb.WriteString("## Delivery (P4)\n\n")
		if exec.Delivery.Content != nil {
			sb.WriteString(fmt.Sprintf("- **Summary**: %s\n", exec.Delivery.Content.Summary))
		}
		if exec.Delivery.Success {
			sb.WriteString("- **Status**: ✓ Success\n")
		} else {
			sb.WriteString(fmt.Sprintf("- **Status**: ✗ Failed (%s)\n", exec.Delivery.Error))
		}
		if len(exec.Delivery.Results) > 0 {
			sb.WriteString(fmt.Sprintf("- **Channels**: %d\n", len(exec.Delivery.Results)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildMessages is a convenience method to build messages array from content
func (f *InputFormatter) BuildMessages(userContent string) []agentcontext.Message {
	return []agentcontext.Message{
		{
			Role:    agentcontext.RoleUser,
			Content: userContent,
		},
	}
}

// BuildMessagesWithSystem builds messages array with system and user content
func (f *InputFormatter) BuildMessagesWithSystem(systemContent, userContent string) []agentcontext.Message {
	return []agentcontext.Message{
		{
			Role:    agentcontext.RoleSystem,
			Content: systemContent,
		},
		{
			Role:    agentcontext.RoleUser,
			Content: userContent,
		},
	}
}
