package task

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/inbox"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/event"
	"github.com/yaoapp/yao/llmprovider"
	"gopkg.in/yaml.v3"
)

//go:embed enrich_result_prompt.yml
var enrichPromptYAML []byte

var (
	enrichSystemTpl *template.Template
	enrichUserTpl   *template.Template
)

func init() {
	var raw struct {
		System string `yaml:"system"`
		User   string `yaml:"user"`
	}
	if err := yaml.Unmarshal(enrichPromptYAML, &raw); err != nil {
		panic("enrich_result_prompt.yml parse error: " + err.Error())
	}
	enrichSystemTpl = template.Must(template.New("enrich_system").Parse(raw.System))
	enrichUserTpl = template.Must(template.New("enrich_user").Parse(raw.User))
}

// enrichTaskResult performs a single LLM call after daemon exits to extract all task metadata:
// title, run_status, summary, instruction, outputs, mail content, tags, priority.
// recentMessages is passed directly from the execution context (req.Messages + dc.ringBuffer).
func enrichTaskResult(chatID string, auth *process.AuthorizedInfo, isFirstRun bool, execErr error, recentMessages []string, locale string) {
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
	systemPrompt, userContent := buildEnrichResultPrompt(recentMessages, isFirstRun, execErr, locale)
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

	applyEnrichResult(chatID, auth, &result, isFirstRun, execErr, locale)
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

func applyEnrichResult(chatID string, auth *process.AuthorizedInfo, result *enrichResult, isFirstRun bool, execErr error, locale string) {
	taskUpdates := map[string]interface{}{"updated_at": time.Now()}
	chatUpdates := map[string]interface{}{"updated_at": time.Now()}
	eventData := map[string]any{"chat_id": chatID, "__yao_team_id": auth.TeamID}

	// Determine final run_status
	finalStatus := result.RunStatus
	if execErr != nil && execErr != errUserCancelled {
		finalStatus = "failed"
	}
	if execErr == errUserCancelled && finalStatus != "cancelled" {
		finalStatus = "cancelled"
	}
	if finalStatus == "" {
		finalStatus = "completed"
	}
	if finalStatus != "completed" && finalStatus != "waiting" && finalStatus != "failed" && finalStatus != "cancelled" {
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

	// Instruction — build ScheduledInstruction JSON
	if result.Instruction != "" {
		si := ScheduledInstruction{
			Prompt:        result.Instruction,
			Locale:        strings.ToLower(locale),
			FirstQuestion: GetOriginalPromptAsString(chatID),
			FirstAnswer:   GetFirstAssistantMessage(chatID),
			UpdatedAt:     time.Now().Format(time.RFC3339),
		}
		siJSON, _ := json.Marshal(si)
		taskUpdates["instruction"] = string(siJSON)
		eventData["instruction"] = si
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
	event.Push(context.Background(), "mail.updated", mailEventData)
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

func buildEnrichResultPrompt(recentMessages []string, isFirstRun bool, execErr error, locale string) (systemPrompt, userContent string) {
	lang := localeToLanguage(locale)

	var sysBuf bytes.Buffer
	enrichSystemTpl.Execute(&sysBuf, map[string]any{
		"IsFirstRun": isFirstRun,
		"Language":   lang,
	})
	systemPrompt = sysBuf.String()

	execStatus := "completed normally"
	if execErr != nil {
		if execErr.Error() == "user_cancelled" {
			execStatus = "user cancelled (task was stopped mid-execution by user)"
		} else {
			execStatus = fmt.Sprintf("execution error: %s", execErr.Error())
		}
	}

	var userBuf bytes.Buffer
	enrichUserTpl.Execute(&userBuf, map[string]any{
		"ExecStatus": execStatus,
		"Messages":   strings.Join(recentMessages, "\n---\n"),
	})
	userContent = userBuf.String()
	return
}

func localeToLanguage(locale string) string {
	switch strings.ToLower(locale) {
	case "zh-cn", "zh-hans":
		return "Chinese (Simplified)"
	case "zh-tw", "zh-hant":
		return "Chinese (Traditional)"
	case "ja", "ja-jp":
		return "Japanese"
	case "ko", "ko-kr":
		return "Korean"
	default:
		return "English"
	}
}
