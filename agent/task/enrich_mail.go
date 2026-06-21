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
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/event"
	"github.com/yaoapp/yao/llmprovider"
)

// enrichMailContent asynchronously enriches a mail's content using a light LLM.
// Triggered from setStatus after inbox.OnStatusChange returns a mailID.
func enrichMailContent(mailID string, chatID string, mailType string, auth *process.AuthorizedInfo) {
	go func() {
		defer func() { recover() }()

		if llmprovider.Global == nil {
			return
		}

		var recentMessages []string
		if dc, ok := GetDaemon(chatID); ok {
			recentMessages = extractRecentText(dc, 10)
		} else {
			recentMessages = loadRecentMessagesText(chatID, 10)
		}
		if len(recentMessages) == 0 {
			return
		}

		lightConn, err := llmprovider.Global.GetRoleModelBy("light", auth)
		if err != nil || lightConn == nil {
			return
		}

		enrichChatID := agentcontext.GenChatID()
		ctx := agentcontext.New(context.Background(), toOAuthInfo(auth), enrichChatID)
		defer ctx.Release()

		llmInstance, err := llm.New(lightConn, &agentcontext.CompletionOptions{})
		if err != nil {
			return
		}

		prompt := buildEnrichPrompt(mailType, recentMessages)
		resp, err := llmInstance.Post(ctx, []agentcontext.Message{
			{Role: "system", Content: prompt},
		}, &agentcontext.CompletionOptions{})
		if err != nil || resp == nil {
			return
		}

		contentStr, ok := resp.Content.(string)
		if !ok {
			return
		}
		contentStr = cleanMarkdownFences(contentStr)

		var enriched struct {
			Title    string         `json:"title"`
			Body     string         `json:"body"`
			Priority string         `json:"priority"`
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(contentStr), &enriched); err != nil {
			return
		}

		updates := map[string]interface{}{"updated_at": time.Now()}
		if enriched.Title != "" {
			updates["title"] = enriched.Title
		}
		if enriched.Body != "" {
			updates["body"] = enriched.Body
		}
		if enriched.Priority != "" && isValidMailPriority(enriched.Priority) {
			updates["priority"] = enriched.Priority
		}
		if enriched.Metadata != nil {
			metaJSON, _ := json.Marshal(enriched.Metadata)
			updates["metadata"] = string(metaJSON)
		}

		capsule.Global.Query().Table(tableMail()).
			Where("mail_id", "=", mailID).
			Update(updates)

		event.Push(context.Background(), "mail.updated", map[string]any{
			"mail_id":          mailID,
			"title":            enriched.Title,
			"body":             enriched.Body,
			"priority":         enriched.Priority,
			"metadata":         enriched.Metadata,
			"__yao_created_by": auth.UserID,
		})
	}()
}

// extractRecentText extracts text content from the last N messages in ringBuffer
func extractRecentText(dc *DaemonContext, n int) []string {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	start := len(dc.ringBuffer) - n
	if start < 0 {
		start = 0
	}

	var texts []string
	for _, msg := range dc.ringBuffer[start:] {
		switch msg.Type {
		case "text", "error", "execute":
			if content, ok := msg.Props["content"].(string); ok && content != "" {
				role := "assistant"
				if r, ok := msg.Props["role"].(string); ok {
					role = r
				}
				texts = append(texts, fmt.Sprintf("[%s] %s", role, content))
			}
		}
	}
	return texts
}

// loadRecentMessagesText loads recent messages from DB when daemon is not alive
func loadRecentMessagesText(chatID string, n int) []string {
	rows, err := capsule.Global.Query().Table(tableMessage()).
		Select("role", "content", "type").
		Where("chat_id", "=", chatID).
		OrderBy("sequence", "desc").
		Limit(n).
		Get()
	if err != nil {
		return nil
	}

	var texts []string
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		role := getString(row, "role")
		content := getString(row, "content")
		if content != "" {
			texts = append(texts, fmt.Sprintf("[%s] %s", role, content))
		}
	}
	return texts
}

// buildEnrichPrompt builds the system prompt for mail enrichment based on type
func buildEnrichPrompt(mailType string, recentMessages []string) string {
	msgContext := strings.Join(recentMessages, "\n---\n")

	switch mailType {
	case "input":
		return fmt.Sprintf(`根据以下 AI 助手的最近对话,生成收件箱通知内容。
助手正在等待用户输入。分析对话判断需要什么输入。

对话上下文:
%s

返回严格 JSON:
{"title":"简洁通知标题(30字内)","body":"详细说明(100字内)","priority":"high","metadata":{"input_hint":"提示语","input_fields":[]}}
仅返回 JSON`, msgContext)

	case "completed":
		return fmt.Sprintf(`根据以下 AI 助手的最近对话,生成收件箱通知内容。
任务已完成。总结关键成果。

对话上下文:
%s

返回严格 JSON:
{"title":"简洁通知标题(30字内)","body":"关键成果总结(100字内)","priority":"low","metadata":{"summary":"一句话总结"}}
仅返回 JSON`, msgContext)

	case "failed":
		return fmt.Sprintf(`根据以下 AI 助手的最近对话,生成收件箱通知内容。
任务执行失败。分析失败原因。

对话上下文:
%s

返回严格 JSON:
{"title":"简洁通知标题(30字内)","body":"失败原因+建议(100字内)","priority":"high","metadata":{"error_type":"timeout|crash|tool_error|llm_error|unknown","suggestion":"修复建议"}}
仅返回 JSON`, msgContext)
	}
	return ""
}
