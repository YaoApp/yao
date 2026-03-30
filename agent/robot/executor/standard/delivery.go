package standard

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/model"
	kunlog "github.com/yaoapp/kun/log"
	robotevents "github.com/yaoapp/yao/agent/robot/events"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/event"
)

// RunDelivery executes P4: Delivery phase
//
// Process:
//  1. Call Delivery Agent with full execution context
//  2. Agent generates DeliveryContent (summary, body, attachments)
//  3. Push delivery event for asynchronous routing via handlers
func (e *Executor) RunDelivery(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	robot := exec.GetRobot()
	if robot == nil {
		return fmt.Errorf("robot not found in execution")
	}

	locale := getEffectiveLocale(robot, exec.Input)
	e.updateUIFields(ctx, exec, "", getLocalizedMessage(locale, "generating_delivery"))

	// Get agent ID for delivery phase (per-robot config > global Uses > empty)
	agentID := robottypes.ResolvePhaseAgent(robot.Config, robottypes.PhaseDelivery)
	if agentID == "" {
		return fmt.Errorf("no Delivery Agent configured (set uses.delivery in agent.yml or resources.phases in robot config)")
	}

	formatter := NewInputFormatter()
	userContent := formatter.FormatDeliveryInput(exec, robot)

	if userContent == "" {
		return fmt.Errorf("no content available for delivery generation")
	}

	caller := NewAgentCaller()
	caller.Connector = robot.LanguageModel
	caller.Workspace = robot.Workspace
	result, err := caller.CallWithMessages(ctx, agentID, userContent)
	if err != nil {
		return fmt.Errorf("delivery agent (%s) call failed: %w", agentID, err)
	}

	data, err := result.GetJSON()
	if err != nil {
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
		return e.pushDeliveryEvent(ctx, exec, robot)
	}

	content := parseDeliveryContent(data)
	if content == nil {
		return fmt.Errorf("delivery agent (%s) returned invalid content", agentID)
	}

	exec.Delivery = &robottypes.DeliveryResult{
		RequestID: generateRequestID(exec.ID),
		Content:   content,
		Success:   true,
	}

	return e.pushDeliveryEvent(ctx, exec, robot)
}

// pushDeliveryEvent pushes a delivery event to the event bus.
// Registered handlers (see events/handlers.go) route to email/webhook/process channels.
func (e *Executor) pushDeliveryEvent(ctx *robottypes.Context, exec *robottypes.Execution, robot *robottypes.Robot) error {
	prefs := buildDeliveryPreferences(robot)

	chatID := exec.ChatID
	var extra map[string]any
	if exec.Input != nil && exec.Input.Data != nil {
		if sourceChatID, ok := exec.Input.Data["chat_id"].(string); ok && sourceChatID != "" {
			if channel, ok := exec.Input.Data["channel"].(string); ok && channel != "" {
				chatID = channel + ":" + sourceChatID
			}
		}
		if e, ok := exec.Input.Data["extra"].(map[string]any); ok {
			extra = e
		}
	}

	_, err := event.Push(ctx.Context, robotevents.Delivery, robotevents.DeliveryPayload{
		ExecutionID: exec.ID,
		MemberID:    exec.MemberID,
		TeamID:      exec.TeamID,
		ChatID:      chatID,
		Content:     exec.Delivery.Content,
		Preferences: prefs,
		Extra:       extra,
	})
	if err != nil {
		kunlog.Error("delivery event push failed: execution=%s error=%v", exec.ID, err)
	}
	return nil
}

// parseDeliveryContent parses the Delivery Agent response into DeliveryContent
func parseDeliveryContent(data map[string]interface{}) *robottypes.DeliveryContent {
	if data == nil {
		return nil
	}

	contentData, ok := data["content"].(map[string]interface{})
	if !ok {
		contentData = data
	}

	content := &robottypes.DeliveryContent{}

	if summary, ok := contentData["summary"].(string); ok {
		content.Summary = summary
	}
	if body, ok := contentData["body"].(string); ok {
		content.Body = body
	}

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

	if content.Summary == "" && content.Body == "" {
		return nil
	}

	return content
}

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

	if att.Title == "" || att.File == "" {
		return nil
	}

	return att
}

func generateRequestID(execID string) string {
	return fmt.Sprintf("dlv-%s-%d", execID, time.Now().UnixNano()%1000000)
}

func getTaskDescription(task robottypes.Task) string {
	if len(task.Messages) == 0 {
		return task.GoalRef
	}

	for _, msg := range task.Messages {
		if content, ok := msg.Content.(string); ok && content != "" {
			if len(content) > 100 {
				return content[:97] + "..."
			}
			return content
		}
	}

	if task.GoalRef != "" {
		return task.GoalRef
	}

	return "Task " + task.ID
}

func truncateSummary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	truncated := text[:maxLen]
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		return truncated[:idx] + "..."
	}
	return truncated + "..."
}

func buildDeliveryPreferences(robot *robottypes.Robot) *robottypes.DeliveryPreferences {
	if robot == nil {
		return nil
	}

	prefs := &robottypes.DeliveryPreferences{}

	managerEmail := robot.ManagerEmail
	if managerEmail == "" && robot.ManagerID != "" {
		managerEmail = getManagerEmail(robot.ManagerID)
		if managerEmail != "" {
			robot.ManagerEmail = managerEmail
		}
	}

	var emailTargets []robottypes.EmailTarget

	if managerEmail != "" {
		emailTargets = append(emailTargets, robottypes.EmailTarget{
			To: []string{managerEmail},
		})
	}

	if robot.Config != nil && robot.Config.Delivery != nil && robot.Config.Delivery.Email != nil {
		for _, target := range robot.Config.Delivery.Email.Targets {
			if len(target.To) > 0 {
				emailTargets = append(emailTargets, target)
			}
		}
	}

	if len(emailTargets) > 0 {
		prefs.Email = &robottypes.EmailPreference{
			Enabled: true,
			Targets: emailTargets,
		}
	}

	if robot.Config != nil && robot.Config.Delivery != nil && robot.Config.Delivery.Webhook != nil {
		if robot.Config.Delivery.Webhook.Enabled && len(robot.Config.Delivery.Webhook.Targets) > 0 {
			prefs.Webhook = robot.Config.Delivery.Webhook
		}
	}

	if robot.Config != nil && robot.Config.Delivery != nil && robot.Config.Delivery.Process != nil {
		if robot.Config.Delivery.Process.Enabled && len(robot.Config.Delivery.Process.Targets) > 0 {
			prefs.Process = robot.Config.Delivery.Process
		}
	}

	return prefs
}

func getManagerEmail(managerID string) string {
	if managerID == "" {
		return ""
	}

	m := model.Select("__yao.member")
	if m == nil {
		return ""
	}

	records, err := m.Get(model.QueryParam{
		Select: []interface{}{"email"},
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: managerID},
		},
		Limit: 1,
	})
	if err != nil || len(records) == 0 {
		return ""
	}

	if email, ok := records[0]["email"].(string); ok {
		return email
	}
	return ""
}

// FormatDeliveryInput formats the full execution context for the Delivery Agent
func (f *InputFormatter) FormatDeliveryInput(exec *robottypes.Execution, robot *robottypes.Robot) string {
	if exec == nil {
		return ""
	}

	var sb strings.Builder

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

	sb.WriteString("## Execution Context\n\n")
	sb.WriteString(fmt.Sprintf("- **Trigger**: %s\n", exec.TriggerType))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", exec.Status))
	sb.WriteString(fmt.Sprintf("- **Start Time**: %s\n", exec.StartTime.Format("2006-01-02 15:04:05")))
	if exec.EndTime != nil {
		duration := exec.EndTime.Sub(exec.StartTime)
		sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", duration.String()))
	}
	sb.WriteString("\n")

	if exec.Inspiration != nil && exec.Inspiration.Content != "" {
		sb.WriteString("## Inspiration (P0)\n\n")
		sb.WriteString(exec.Inspiration.Content)
		sb.WriteString("\n\n")
	}

	if exec.Goals != nil && exec.Goals.Content != "" {
		sb.WriteString("## Goals (P1)\n\n")
		sb.WriteString(exec.Goals.Content)
		sb.WriteString("\n\n")
	}

	if len(exec.Tasks) > 0 {
		sb.WriteString("## Tasks (P2)\n\n")
		for i, task := range exec.Tasks {
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

			if result.Error != "" {
				sb.WriteString(fmt.Sprintf("\n**Error**: %s\n", result.Error))
			}

			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("### Summary\n\n- **Total Tasks**: %d\n- **Succeeded**: %d\n- **Failed**: %d\n\n",
			len(exec.Results), successCount, failCount))
	}

	return sb.String()
}
