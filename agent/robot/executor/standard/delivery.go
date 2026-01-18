package standard

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunDelivery executes P4: Delivery phase
// Calls the Delivery Agent to generate content, then routes to Delivery Center
//
// Input:
//   - Full execution context (P0-P3)
//   - Robot config
//
// Output:
//   - DeliveryResult with content and channel results
//
// Process:
//  1. Call Delivery Agent with full execution context
//  2. Agent generates DeliveryContent (summary, body, attachments)
//  3. Route content to Delivery Center for actual delivery
func (e *Executor) RunDelivery(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	// Get robot for identity and resources
	robot := exec.GetRobot()
	if robot == nil {
		return fmt.Errorf("robot not found in execution")
	}

	// Get agent ID for delivery phase
	agentID := "__yao.delivery" // default
	if robot.Config != nil && robot.Config.Resources != nil {
		agentID = robot.Config.Resources.GetPhaseAgent(robottypes.PhaseDelivery)
	}

	// Build input for Delivery Agent
	formatter := NewInputFormatter()
	userContent := formatter.FormatDeliveryInput(exec, robot)

	if userContent == "" {
		return fmt.Errorf("no content available for delivery generation")
	}

	// Call Delivery Agent
	caller := NewAgentCaller()
	result, err := caller.CallWithMessages(ctx, agentID, userContent)
	if err != nil {
		return fmt.Errorf("delivery agent (%s) call failed: %w", agentID, err)
	}

	// Parse response as JSON
	// Delivery Agent returns: { "content": { "summary": "...", "body": "...", "attachments": [...] } }
	data, err := result.GetJSON()
	if err != nil {
		// Fallback: if not JSON, create minimal content from raw text
		content := result.GetText()
		if content == "" {
			return fmt.Errorf("delivery agent returned empty response")
		}
		exec.Delivery = &robottypes.DeliveryResult{
			RequestID: generateRequestID(exec.ID),
			Content: &robottypes.DeliveryContent{
				Summary: truncateSummary(content, 200),
				Body:    content,
			},
			Success: true,
		}
		return e.routeToDeliveryCenter(ctx, exec, robot)
	}

	// Build DeliveryContent from JSON
	content := parseDeliveryContent(data)
	if content == nil {
		return fmt.Errorf("delivery agent (%s) returned invalid content", agentID)
	}

	// Build DeliveryResult
	exec.Delivery = &robottypes.DeliveryResult{
		RequestID: generateRequestID(exec.ID),
		Content:   content,
		Success:   true,
	}

	// Route to Delivery Center for actual delivery
	return e.routeToDeliveryCenter(ctx, exec, robot)
}

// routeToDeliveryCenter sends content to the Delivery Center for actual delivery
// The Delivery Center decides which channels to use based on robot/user preferences
func (e *Executor) routeToDeliveryCenter(ctx *robottypes.Context, exec *robottypes.Execution, robot *robottypes.Robot) error {
	if exec.Delivery == nil || exec.Delivery.Content == nil {
		return fmt.Errorf("no delivery content to route")
	}

	// Get delivery preferences from robot config
	var prefs *robottypes.DeliveryPreferences
	if robot.Config != nil {
		prefs = robot.Config.Delivery
	}

	// If no preferences configured, skip delivery (content is still saved in exec.Delivery)
	if prefs == nil || !hasActiveChannels(prefs) {
		// No channels configured - mark as success but with no results
		exec.Delivery.Success = true
		return nil
	}

	// Create Delivery Center and execute
	center := NewDeliveryCenter()
	results, err := center.Deliver(ctx, exec.Delivery.Content, &robottypes.DeliveryContext{
		MemberID:    exec.MemberID,
		ExecutionID: exec.ID,
		TriggerType: exec.TriggerType,
		TeamID:      exec.TeamID,
	}, prefs, robot)

	// Update delivery result
	exec.Delivery.Results = results
	now := time.Now()
	exec.Delivery.SentAt = &now

	if err != nil {
		exec.Delivery.Success = false
		exec.Delivery.Error = err.Error()
		return err
	}

	// Check if all channels succeeded
	allSuccess := true
	for _, r := range results {
		if !r.Success {
			allSuccess = false
			break
		}
	}
	exec.Delivery.Success = allSuccess

	return nil
}

// parseDeliveryContent parses the Delivery Agent response into DeliveryContent
func parseDeliveryContent(data map[string]interface{}) *robottypes.DeliveryContent {
	if data == nil {
		return nil
	}

	// Try to get content object
	contentData, ok := data["content"].(map[string]interface{})
	if !ok {
		// Fallback: maybe the data itself is the content
		contentData = data
	}

	content := &robottypes.DeliveryContent{}

	// Parse summary
	if summary, ok := contentData["summary"].(string); ok {
		content.Summary = summary
	}

	// Parse body
	if body, ok := contentData["body"].(string); ok {
		content.Body = body
	}

	// Parse attachments
	if attachments, ok := contentData["attachments"].([]interface{}); ok {
		for _, att := range attachments {
			if attMap, ok := att.(map[string]interface{}); ok {
				attachment := parseDeliveryAttachment(attMap)
				if attachment != nil {
					content.Attachments = append(content.Attachments, *attachment)
				}
			}
		}
	}

	// Validate: at least summary or body should be present
	if content.Summary == "" && content.Body == "" {
		return nil
	}

	return content
}

// parseDeliveryAttachment parses a single attachment from the agent response
func parseDeliveryAttachment(data map[string]interface{}) *robottypes.DeliveryAttachment {
	if data == nil {
		return nil
	}

	att := &robottypes.DeliveryAttachment{}

	if title, ok := data["title"].(string); ok {
		att.Title = title
	}
	if desc, ok := data["description"].(string); ok {
		att.Description = desc
	}
	if taskID, ok := data["task_id"].(string); ok {
		att.TaskID = taskID
	}
	if file, ok := data["file"].(string); ok {
		att.File = file
	}

	// At minimum, need title and file
	if att.Title == "" || att.File == "" {
		return nil
	}

	return att
}

// generateRequestID generates a unique request ID for delivery tracking
func generateRequestID(execID string) string {
	return fmt.Sprintf("dlv-%s-%d", execID, time.Now().UnixNano()%1000000)
}

// getTaskDescription extracts a description from task messages
func getTaskDescription(task robottypes.Task) string {
	if len(task.Messages) == 0 {
		return task.GoalRef
	}

	// Try to get text from first message
	for _, msg := range task.Messages {
		if content, ok := msg.Content.(string); ok && content != "" {
			// Truncate if too long
			if len(content) > 100 {
				return content[:97] + "..."
			}
			return content
		}
	}

	// Fallback to goal reference
	if task.GoalRef != "" {
		return task.GoalRef
	}

	return "Task " + task.ID
}

// truncateSummary truncates text to maxLen characters
func truncateSummary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	// Find last space before maxLen to avoid cutting words
	truncated := text[:maxLen]
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		return truncated[:idx] + "..."
	}
	return truncated + "..."
}

// hasActiveChannels checks if any delivery channel is configured
func hasActiveChannels(prefs *robottypes.DeliveryPreferences) bool {
	if prefs == nil {
		return false
	}
	if prefs.Email != nil && prefs.Email.Enabled && len(prefs.Email.Targets) > 0 {
		return true
	}
	if prefs.Webhook != nil && prefs.Webhook.Enabled && len(prefs.Webhook.Targets) > 0 {
		return true
	}
	if prefs.Process != nil && prefs.Process.Enabled && len(prefs.Process.Targets) > 0 {
		return true
	}
	return false
}

// FormatDeliveryInput formats the full execution context for the Delivery Agent
func (f *InputFormatter) FormatDeliveryInput(exec *robottypes.Execution, robot *robottypes.Robot) string {
	if exec == nil {
		return ""
	}

	var sb strings.Builder

	// Robot identity
	if robot != nil && robot.Config != nil && robot.Config.Identity != nil {
		sb.WriteString("## Robot Identity\n\n")
		sb.WriteString(fmt.Sprintf("- **Role**: %s\n", robot.Config.Identity.Role))
		if len(robot.Config.Identity.Duties) > 0 {
			sb.WriteString("- **Duties**: ")
			sb.WriteString(strings.Join(robot.Config.Identity.Duties, ", "))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Trigger type
	sb.WriteString("## Execution Context\n\n")
	sb.WriteString(fmt.Sprintf("- **Trigger**: %s\n", exec.TriggerType))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", exec.Status))
	sb.WriteString(fmt.Sprintf("- **Start Time**: %s\n", exec.StartTime.Format("2006-01-02 15:04:05")))
	if exec.EndTime != nil {
		duration := exec.EndTime.Sub(exec.StartTime)
		sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", duration.String()))
	}
	sb.WriteString("\n")

	// Inspiration (P0) - for clock trigger
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
			// Extract task description from messages if available
			taskDesc := getTaskDescription(task)
			sb.WriteString(fmt.Sprintf("%d. **%s** - %s\n", i+1, task.ID, taskDesc))
			sb.WriteString(fmt.Sprintf("   - Executor: %s (%s)\n", task.ExecutorID, task.ExecutorType))
			sb.WriteString(fmt.Sprintf("   - Status: %s\n", task.Status))
			if task.ExpectedOutput != "" {
				sb.WriteString(fmt.Sprintf("   - Expected: %s\n", task.ExpectedOutput))
			}
		}
		sb.WriteString("\n")
	}

	// Results (P3) - detailed
	if len(exec.Results) > 0 {
		sb.WriteString("## Results (P3)\n\n")

		successCount := 0
		failCount := 0

		for _, result := range exec.Results {
			if result.Success {
				successCount++
				sb.WriteString(fmt.Sprintf("### ✓ Task: %s\n\n", result.TaskID))
			} else {
				failCount++
				sb.WriteString(fmt.Sprintf("### ✗ Task: %s\n\n", result.TaskID))
			}

			sb.WriteString(fmt.Sprintf("- **Duration**: %dms\n", result.Duration))

			// Validation
			if result.Validation != nil {
				if result.Validation.Passed {
					sb.WriteString(fmt.Sprintf("- **Validation**: ✓ Passed (score: %.2f)\n", result.Validation.Score))
				} else {
					sb.WriteString("- **Validation**: ✗ Failed\n")
					if len(result.Validation.Issues) > 0 {
						for _, issue := range result.Validation.Issues {
							sb.WriteString(fmt.Sprintf("  - %s\n", issue))
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
		sb.WriteString(fmt.Sprintf("### Summary\n\n- **Total Tasks**: %d\n- **Succeeded**: %d\n- **Failed**: %d\n\n",
			len(exec.Results), successCount, failCount))
	}

	return sb.String()
}
