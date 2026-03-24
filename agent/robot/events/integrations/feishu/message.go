package feishu

import (
	"context"
	"fmt"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/event"
	fsapi "github.com/yaoapp/yao/integrations/feishu"
)

// handleMessages processes a batch of Feishu messages for one chat.
func (a *Adapter) handleMessages(ctx context.Context, entry *botEntry, cms []*fsapi.ConvertedMessage) {
	if len(cms) == 0 {
		return
	}

	var allParts []interface{}
	var lastCM *fsapi.ConvertedMessage

	for _, cm := range cms {
		if cm == nil {
			continue
		}

		dedupKey := fmt.Sprintf("fs:%s:%s", entry.robotID, cm.MessageID)
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
			Channel:    "feishu",
			MessageID:  lastCM.MessageID,
			AppID:      entry.appID,
			ChatID:     lastCM.ChatID,
			SenderID:   lastCM.SenderID,
			SenderName: lastCM.SenderName,
			Locale:     events.NormalizeLocale(lastCM.LanguageCode),
			Extra: map[string]any{
				"feishu_message_id": lastCM.MessageID,
				"sender_id":         lastCM.SenderID,
				"app_id":            entry.appID,
			},
		},
	}

	if _, err := event.Push(ctx, events.Message, msgPayload); err != nil {
		log.Error("feishu adapter: event.Push robot.message failed robot=%s: %v", entry.robotID, err)
	}
}

func buildContentParts(cm *fsapi.ConvertedMessage) []interface{} {
	var parts []interface{}

	if cm.HasText() {
		parts = append(parts, map[string]interface{}{
			"type": "text",
			"text": cm.Text,
		})
	}

	for _, mi := range cm.MediaItems {
		if mi.Wrapper == "" {
			continue
		}
		parts = append(parts, map[string]interface{}{
			"type":      "file",
			"file_url":  mi.Wrapper,
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
