package dingtalk

import (
	"context"
	"fmt"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/event"
	dtapi "github.com/yaoapp/yao/integrations/dingtalk"
)

// handleMessages processes a batch of DingTalk messages.
func (a *Adapter) handleMessages(ctx context.Context, entry *botEntry, cms []*dtapi.ConvertedMessage) {
	if len(cms) == 0 {
		return
	}

	var allParts []interface{}
	var lastCM *dtapi.ConvertedMessage

	for _, cm := range cms {
		if cm == nil {
			continue
		}

		dedupKey := fmt.Sprintf("dt:%s:%s", entry.robotID, cm.MessageID)
		if !a.dedup.markSeen(dedupKey) {
			continue
		}

		parts := buildContentParts(cm)
		if len(parts) == 0 {
			continue
		}

		allParts = append(allParts, parts...)
		lastCM = cm
	}

	if len(allParts) == 0 || lastCM == nil {
		return
	}

	content := mergeContentParts(allParts)

	msgPayload := events.MessagePayload{
		RobotID: entry.robotID,
		Messages: []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: content},
		},
		Metadata: &events.MessageMetadata{
			Channel:    "dingtalk",
			MessageID:  lastCM.MessageID,
			AppID:      entry.clientID,
			ChatID:     lastCM.ConversationID,
			SenderID:   lastCM.SenderID,
			SenderName: lastCM.SenderNick,
			Locale:     "zh-cn",
			Extra: map[string]any{
				"session_webhook":   lastCM.SessionWebhook,
				"conversation_type": lastCM.ConversationType,
				"dt_message_id":     lastCM.MessageID,
				"sender_id":         lastCM.SenderID,
				"app_id":            entry.clientID,
			},
		},
	}

	if _, err := event.Push(ctx, events.Message, msgPayload); err != nil {
		log.Error("dingtalk adapter: event.Push robot.message failed robot=%s: %v", entry.robotID, err)
	}
}

func buildContentParts(cm *dtapi.ConvertedMessage) []interface{} {
	var parts []interface{}

	if cm.HasText() {
		parts = append(parts, map[string]interface{}{
			"type": "text",
			"text": cm.Text,
		})
	}

	for _, mi := range cm.MediaItems {
		if mi.Wrapper == "" && mi.URL == "" {
			continue
		}
		url := mi.Wrapper
		if url == "" {
			url = mi.URL
		}
		parts = append(parts, map[string]interface{}{
			"type":      "file",
			"file_url":  url,
			"mime_type": mi.MimeType,
			"file_name": mi.FileName,
		})
	}

	return parts
}

func mergeContentParts(parts []interface{}) interface{} {
	allText := true
	for _, p := range parts {
		m, ok := p.(map[string]interface{})
		if !ok || m["type"] != "text" {
			allText = false
			break
		}
	}

	if allText {
		var buf strings.Builder
		for i, p := range parts {
			if i > 0 {
				buf.WriteString("\n")
			}
			m := p.(map[string]interface{})
			buf.WriteString(m["text"].(string))
		}
		return buf.String()
	}

	return parts
}
