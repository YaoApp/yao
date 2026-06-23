package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/inbox"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/event"
	"github.com/yaoapp/yao/llmprovider"
)

// enrichTaskResult performs a single LLM call after daemon exits to extract all task metadata:
// title, run_status, summary, instruction, outputs, mail content, tags, priority.
// recentMessages is passed directly from the execution context (req.Messages + dc.ringBuffer).
func enrichTaskResult(chatID string, auth *process.AuthorizedInfo, isFirstRun bool, execErr error, recentMessages []string) {
	enrichChatID := agentcontext.GenChatID()
	ctx := agentcontext.New(context.Background(), toOAuthInfo(auth), enrichChatID)
	defer ctx.Release()
	ctx.Logger.SetAssistantID("task.enrich")
	ctx.Logger.Start()

	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Error("panic recovered: %v", r)
			ctx.Logger.End(false, fmt.Errorf("panic: %v", r))
		}
	}()

	if len(recentMessages) == 0 {
		ctx.Logger.End(false, fmt.Errorf("no conversation content"))
		markTaskFailed(chatID, auth, "execution produced no output")
		return
	}

	ctx.Logger.Phase("Get LLM Provider")
	if llmprovider.Global == nil {
		ctx.Logger.End(false, fmt.Errorf("LLM provider not configured"))
		markTaskFailed(chatID, auth, "enrichment failed: LLM provider not configured")
		return
	}

	lightConn, err := llmprovider.Global.GetRoleModelBy("light", auth)
	if err != nil || lightConn == nil {
		reason := "light model unavailable"
		if err != nil {
			reason = err.Error()
		}
		ctx.Logger.End(false, fmt.Errorf("%s", reason))
		markTaskFailed(chatID, auth, "enrichment failed: "+reason)
		return
	}

	opts := llm.BuildCompletionOptions(lightConn, nil)
	llmInstance, err := llm.New(lightConn, opts)
	if err != nil {
		ctx.Logger.End(false, fmt.Errorf("llm.New: %v", err))
		markTaskFailed(chatID, auth, "enrichment failed: "+err.Error())
		return
	}
	ctx.Logger.PhaseComplete("Get LLM Provider")

	ctx.Logger.Phase("LLM Call")
	systemPrompt, userContent := buildEnrichResultPrompt(recentMessages, isFirstRun, execErr)
	resp, err := llmInstance.Post(ctx, []agentcontext.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}, opts)
	if err != nil || resp == nil {
		reason := "LLM returned nil"
		if err != nil {
			reason = err.Error()
		}
		ctx.Logger.End(false, fmt.Errorf("LLM call: %s", reason))
		markTaskFailed(chatID, auth, "enrichment failed: "+reason)
		return
	}
	ctx.Logger.PhaseComplete("LLM Call")

	ctx.Logger.Phase("Parse & Apply")
	contentStr, ok := resp.Content.(string)
	if !ok {
		ctx.Logger.End(false, fmt.Errorf("invalid LLM response type"))
		markTaskFailed(chatID, auth, "enrichment failed: invalid LLM response type")
		return
	}
	contentStr = cleanMarkdownFences(contentStr)

	var result enrichResult
	if err := json.Unmarshal([]byte(contentStr), &result); err != nil {
		ctx.Logger.End(false, fmt.Errorf("JSON parse: %v", err))
		markTaskFailed(chatID, auth, "enrichment failed: "+err.Error())
		return
	}

	applyEnrichResult(chatID, auth, &result, isFirstRun, execErr)
	ctx.Logger.PhaseComplete("Parse & Apply")
	ctx.Logger.End(true, nil)
}

type enrichResult struct {
	Title       string `json:"title,omitempty"`
	RunStatus   string `json:"run_status"`
	Summary     string `json:"summary"`
	Instruction string `json:"instruction,omitempty"`
	Outputs     []any  `json:"outputs,omitempty"`
	Mail        *struct {
		Title    string `json:"title"`
		Body     string `json:"body"`
		Priority string `json:"priority"`
	} `json:"mail,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Priority string   `json:"priority,omitempty"`
}

func applyEnrichResult(chatID string, auth *process.AuthorizedInfo, result *enrichResult, isFirstRun bool, execErr error) {
	taskUpdates := map[string]interface{}{"updated_at": time.Now()}
	chatUpdates := map[string]interface{}{"updated_at": time.Now()}
	eventData := map[string]any{"chat_id": chatID, "__yao_team_id": auth.TeamID}

	// Determine final run_status
	finalStatus := result.RunStatus
	if execErr != nil {
		finalStatus = "failed"
	}
	if finalStatus == "" {
		finalStatus = "completed"
	}
	if finalStatus != "completed" && finalStatus != "waiting" && finalStatus != "failed" {
		finalStatus = "completed"
	}
	taskUpdates["run_status"] = finalStatus
	if finalStatus == "completed" || finalStatus == "failed" {
		taskUpdates["completed_at"] = time.Now()
	}
	eventData["run_status"] = finalStatus

	// Title (first run only)
	if isFirstRun && result.Title != "" && len([]rune(result.Title)) <= 50 {
		chatUpdates["title"] = result.Title
		eventData["title"] = result.Title
	}

	// Summary
	if result.Summary != "" {
		taskUpdates["summary"] = result.Summary
		eventData["summary"] = result.Summary
	}

	// Instruction
	if result.Instruction != "" {
		taskUpdates["instruction"] = result.Instruction
		eventData["instruction"] = result.Instruction
	}

	// Outputs
	if len(result.Outputs) > 0 {
		outputsJSON, _ := json.Marshal(result.Outputs)
		taskUpdates["outputs"] = string(outputsJSON)
		eventData["outputs"] = result.Outputs
	}

	// Tags (first run only)
	if isFirstRun && len(result.Tags) > 0 && len(result.Tags) <= 5 {
		tagsJSON, _ := json.Marshal(result.Tags)
		taskUpdates["tags"] = string(tagsJSON)
		eventData["tags"] = result.Tags
	}

	// Priority (first run only)
	if isFirstRun && result.Priority != "" && isValidPriority(result.Priority) {
		taskUpdates["priority"] = result.Priority
		eventData["priority"] = result.Priority
	}

	// Update DB
	capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Update(taskUpdates)

	if len(chatUpdates) > 1 {
		capsule.Global.Query().Table(tableChat()).
			Where("chat_id", "=", chatID).
			Update(chatUpdates)
	}

	// Push unified task.updated event
	event.Push(context.Background(), "task.updated", eventData)

	// Create mail + push mail.new
	createMailFromEnrich(chatID, auth, result, finalStatus)
}

func createMailFromEnrich(chatID string, auth *process.AuthorizedInfo, result *enrichResult, finalStatus string) {
	if finalStatus != "waiting" && finalStatus != "completed" && finalStatus != "failed" {
		return
	}

	inboxTask := &inbox.AgentTask{
		ChatID:    chatID,
		CreatedBy: auth.UserID,
		TeamID:    auth.TeamID,
	}
	if task, _ := getTaskBasicInfo(chatID); task != nil {
		inboxTask.AssistantID = task.AssistantID
		inboxTask.ColumnID = getStringVal(task.ColumnID)
	}

	mailID, _ := inbox.OnStatusChange(context.Background(), inboxTask, finalStatus)
	if mailID == "" {
		return
	}

	// Directly write mail content from LLM result (no second LLM call)
	mailUpdates := map[string]interface{}{"updated_at": time.Now()}
	if result.Mail != nil {
		if result.Mail.Title != "" {
			mailUpdates["title"] = result.Mail.Title
		}
		if result.Mail.Body != "" {
			mailUpdates["body"] = result.Mail.Body
		}
		if result.Mail.Priority != "" && isValidMailPriority(result.Mail.Priority) {
			mailUpdates["priority"] = result.Mail.Priority
		}
	}

	if len(mailUpdates) > 1 {
		capsule.Global.Query().Table(tableMail()).
			Where("mail_id", "=", mailID).
			Update(mailUpdates)
	}

	mailEventData := map[string]any{
		"mail_id":          mailID,
		"__yao_created_by": auth.UserID,
	}
	if result.Mail != nil {
		mailEventData["title"] = result.Mail.Title
		mailEventData["body"] = result.Mail.Body
		mailEventData["priority"] = result.Mail.Priority
	}
	event.Push(context.Background(), "mail.new", mailEventData)
}

// markTaskFailed marks a task as failed with a reason and pushes a task.updated event.
// Used when enrichment cannot proceed (no content, LLM unavailable, parse error, etc.).
func markTaskFailed(chatID string, auth *process.AuthorizedInfo, reason string) {
	capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Update(map[string]interface{}{
			"run_status":    "failed",
			"error_message": reason,
			"completed_at":  time.Now(),
			"updated_at":    time.Now(),
		})
	event.Push(context.Background(), "task.updated", map[string]any{
		"chat_id":       chatID,
		"run_status":    "failed",
		"error_message": reason,
		"__yao_team_id": auth.TeamID,
	})
}

func buildEnrichResultPrompt(recentMessages []string, isFirstRun bool, execErr error) (systemPrompt, userContent string) {
	firstRunFields := ""
	firstRunRules := ""
	if isFirstRun {
		firstRunFields = `  "title": "20字内任务标题",
  "tags": ["标签1","标签2"],
  "priority": "none|low|medium|high",`
		firstRunRules = "- title: 概括用户意图,简洁有力\n- tags: 最多3个分类标签\n- priority: 任务紧急程度\n"
	}

	systemPrompt = fmt.Sprintf(`你是任务元数据提取器。根据用户提供的 AI 助手执行对话和执行状态,提取所有元数据。

返回严格 JSON (不要 markdown 代码块):
{
%s
  "run_status": "completed|waiting|failed",
  "summary": "50字内卡片摘要 (waiting时:需要什么; completed时:做了什么; failed时:失败原因)",
  "instruction": "适合重复执行的完整指令摘要 (供自动化/定时执行使用,100字内)",
  "outputs": [{"type":"file|attachment|service|url","name":"名称","path":"路径或url"}],
  "mail": {"title":"30字内通知标题","body":"100字内通知正文","priority":"low|medium|high"}
}

规则:
- run_status: 如果助手明确要求用户回复/确认/提供信息 → "waiting"; 如果执行出错 → "failed"; 否则 → "completed"
- summary: 简洁有力,适合看板卡片展示
- instruction: 抽象出可重复执行的指令 (去掉临时性/一次性内容)
- outputs: 如果对话中提到生成了文件/服务/链接,提取出来;如果没有则返回空数组
- mail: 生成收件箱通知 (waiting 时 priority 高, completed 时 priority 低)
%s仅返回 JSON,不要其他内容。`, firstRunFields, firstRunRules)

	execStatus := "正常结束"
	if execErr != nil {
		execStatus = fmt.Sprintf("执行出错: %s", execErr.Error())
	}
	msgContext := strings.Join(recentMessages, "\n---\n")

	userContent = fmt.Sprintf("执行状态: %s\n\n对话上下文:\n%s", execStatus, msgContext)
	return
}
