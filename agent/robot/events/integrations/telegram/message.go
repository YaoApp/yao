package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/event"
	tgapi "github.com/yaoapp/yao/integrations/telegram"
)

// handleMessages builds a single event payload from a batch of ConvertedMessages
// belonging to the same chat. Consecutive user messages are merged into one to
// keep the messages array clean for the LLM.
func (a *Adapter) handleMessages(ctx context.Context, entry *botEntry, cms []*tgapi.ConvertedMessage) {
	if len(cms) == 0 {
		return
	}

	var allParts []interface{}
	var lastCM *tgapi.ConvertedMessage

	for _, cm := range cms {
		if cm == nil {
			continue
		}

		if isBotCommand(cm) {
			continue
		}

		dedupKey := fmt.Sprintf("tg:%s:%d", entry.robotID, cm.UpdateID)
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
			Channel:    "telegram",
			MessageID:  strconv.FormatInt(lastCM.UpdateID, 10),
			AppID:      entry.appID,
			ChatID:     strconv.FormatInt(lastCM.ChatID, 10),
			SenderID:   strconv.FormatInt(lastCM.SenderID, 10),
			SenderName: lastCM.SenderName,
			Locale:     events.NormalizeLocale(lastCM.LanguageCode),
			Extra: map[string]any{
				"tg_message_id": lastCM.MessageID,
				"sender_id":     strconv.FormatInt(lastCM.SenderID, 10),
				"app_id":        entry.appID,
			},
		},
	}

	if _, err := event.Push(ctx, events.Message, msgPayload); err != nil {
		log.Error("telegram adapter: event.Push robot.message failed robot=%s: %v", entry.robotID, err)
	}
}

// buildContentParts extracts content parts from a single ConvertedMessage.
func buildContentParts(cm *tgapi.ConvertedMessage) []interface{} {
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

// mergeContentParts merges collected parts into a single content value.
// If all parts are text-only, they are joined with newlines into a plain string.
// Otherwise the full parts array is returned.
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

// isBotCommand returns true if the message is a Telegram bot command (text starting with "/").
func isBotCommand(cm *tgapi.ConvertedMessage) bool {
	return !cm.HasMedia() && strings.HasPrefix(strings.TrimSpace(cm.Text), "/")
}
