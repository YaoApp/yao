package task

import (
	"context"
	"encoding/json"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/event"
	"github.com/yaoapp/yao/llmprovider"
)

// ExtractTaskMetadata asynchronously extracts title, tags, and priority from
// the user's first message using a lightweight LLM. Fire-and-forget goroutine.
// Called from API layer (WS/REST handlers) after first Run.
func ExtractTaskMetadata(chatID string, userMessage string, auth *process.AuthorizedInfo) {
	go func() {
		defer func() { recover() }()

		if llmprovider.Global == nil {
			return
		}

		lightConn, err := llmprovider.Global.GetRoleModelBy("light", auth)
		if err != nil || lightConn == nil {
			return
		}

		extractChatID := agentcontext.GenChatID()
		ctx := agentcontext.New(context.Background(), toOAuthInfo(auth), extractChatID)
		defer ctx.Release()

		llmInstance, err := llm.New(lightConn, &agentcontext.CompletionOptions{})
		if err != nil {
			return
		}

		resp, err := llmInstance.Post(ctx, []agentcontext.Message{
			{Role: "system", Content: extractPrompt},
			{Role: "user", Content: userMessage},
		}, &agentcontext.CompletionOptions{})
		if err != nil || resp == nil {
			return
		}

		contentStr, ok := resp.Content.(string)
		if !ok {
			return
		}
		contentStr = cleanMarkdownFences(contentStr)

		var meta struct {
			Title    string   `json:"title"`
			Tags     []string `json:"tags"`
			Priority string   `json:"priority"`
		}
		if err := json.Unmarshal([]byte(contentStr), &meta); err != nil {
			return
		}

		if meta.Title != "" && len([]rune(meta.Title)) <= 50 {
			capsule.Global.Query().Table(tableChat()).
				Where("chat_id", "=", chatID).
				Update(map[string]interface{}{"title": meta.Title})
		}

		taskUpdates := map[string]interface{}{}
		if len(meta.Tags) > 0 && len(meta.Tags) <= 5 {
			tagsJSON, _ := json.Marshal(meta.Tags)
			taskUpdates["tags"] = string(tagsJSON)
		}
		if meta.Priority != "" && isValidPriority(meta.Priority) {
			taskUpdates["priority"] = meta.Priority
		}
		if len(taskUpdates) > 0 {
			capsule.Global.Query().Table(tableTask()).
				Where("chat_id", "=", chatID).
				Update(taskUpdates)
		}

		eventData := map[string]any{
			"chat_id":       chatID,
			"__yao_team_id": auth.TeamID,
		}
		if meta.Title != "" {
			eventData["title"] = meta.Title
		}
		if len(meta.Tags) > 0 {
			eventData["tags"] = meta.Tags
		}
		if meta.Priority != "" {
			eventData["priority"] = meta.Priority
		}
		event.Push(context.Background(), "task.updated", eventData)

		if dc, ok := GetDaemon(chatID); ok {
			dc.Broadcast(&message.Message{
				Type: "event",
				Props: map[string]interface{}{
					"event": "task_updated",
					"data":  eventData,
				},
			})
		}
	}()
}

const extractPrompt = `根据用户消息,提取任务元数据,返回严格 JSON (不要 markdown 代码块):
{"title": "20字内任务标题", "tags": ["标签1","标签2"], "priority": "none|low|medium|high"}
规则:
- title: 概括用户意图,简洁有力,不超过20字
- tags: 最多3个相关分类标签,如无明确分类则返回空数组
- priority: 根据紧急程度判断,没有明确紧急信号则 none
仅返回 JSON,不要其他内容。`
