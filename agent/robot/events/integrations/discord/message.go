package discord

import (
	"context"
	"fmt"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/event"
	dcapi "github.com/yaoapp/yao/integrations/discord"
)

// handleMessages processes a batch of Discord messages.
func (a *Adapter) handleMessages(ctx context.Context, entry *botEntry, cms []*dcapi.ConvertedMessage) {
	if len(cms) == 0 {
		return
	}

	var allParts []interface{}
	var lastCM *dcapi.ConvertedMessage

	for _, cm := range cms {
		if cm == nil {
			continue
		}

		// Skip bot commands (messages starting with /)
		if strings.HasPrefix(strings.TrimSpace(cm.Text), "/") && !cm.HasMedia() {
			continue
		}

		dedupKey := fmt.Sprintf("dc:%s:%s", entry.robotID, cm.MessageID)
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
			Channel:    "discord",
			MessageID:  lastCM.MessageID,
			AppID:      entry.appID,
			ChatID:     lastCM.ChannelID,
			SenderID:   lastCM.AuthorID,
			SenderName: lastCM.AuthorName,
			Locale:     events.NormalizeLocale(discordLocale(lastCM.Locale)),
			Extra: map[string]any{
				"discord_message_id": lastCM.MessageID,
				"guild_id":           lastCM.GuildID,
				"is_dm":              lastCM.IsDM,
				"sender_id":          lastCM.AuthorID,
				"app_id":             entry.appID,
			},
		},
	}

	if _, err := event.Push(ctx, events.Message, msgPayload); err != nil {
		log.Error("discord adapter: event.Push robot.message failed robot=%s: %v", entry.robotID, err)
	}
}

func buildContentParts(cm *dcapi.ConvertedMessage) []interface{} {
	var parts []interface{}

	if cm.HasText() {
		parts = append(parts, map[string]interface{}{
			"type": "text",
			"text": cm.Text,
		})
	}

	for _, mi := range cm.MediaItems {
		url := mi.Wrapper
		if url == "" {
			url = mi.URL
		}
		if url == "" {
			continue
		}
		parts = append(parts, map[string]interface{}{
			"type":      "file",
			"file_url":  url,
			"mime_type": mi.ContentType,
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

func discordLocale(locale string) string {
	if locale == "" {
		return "en"
	}
	return locale
}
