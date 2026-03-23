package feishu

import (
	"context"
	"fmt"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	fsapi "github.com/yaoapp/yao/integrations/feishu"
)

// Reply sends the assistant message back to the originating Feishu chat.
func (a *Adapter) Reply(ctx context.Context, msg *agentcontext.Message, metadata *events.MessageMetadata) error {
	if msg == nil || metadata == nil {
		return fmt.Errorf("nil message or metadata")
	}

	entry := a.resolveByChat(metadata)
	if entry == nil {
		return fmt.Errorf("no bot registered for feishu metadata (appID=%s)", metadata.AppID)
	}

	var replyToMsgID string
	if metadata.Extra != nil {
		if v, ok := metadata.Extra["feishu_message_id"]; ok {
			if s, ok := v.(string); ok {
				replyToMsgID = s
			}
		}
	}

	if err := entry.bot.SendTyping(ctx, metadata.ChatID); err != nil {
		log.Debug("feishu reply: send typing failed: %v", err)
	}

	return a.sendContent(ctx, entry, metadata.ChatID, replyToMsgID, msg.Content)
}

func (a *Adapter) sendContent(ctx context.Context, entry *botEntry, chatID, replyToMsgID string, content interface{}) error {
	switch c := content.(type) {
	case string:
		if strings.TrimSpace(c) == "" {
			return nil
		}
		return a.sendMarkdown(ctx, entry, chatID, replyToMsgID, c)

	case []interface{}:
		return a.sendParts(ctx, entry, chatID, replyToMsgID, c)

	default:
		parts, ok := toContentParts(content)
		if ok {
			return a.sendPartsTyped(ctx, entry, chatID, replyToMsgID, parts)
		}
		return a.sendMarkdown(ctx, entry, chatID, replyToMsgID, fmt.Sprintf("%v", content))
	}
}

// sendMarkdown converts standard Markdown to Feishu lark_md and sends as an interactive card.
func (a *Adapter) sendMarkdown(ctx context.Context, entry *botEntry, chatID, replyToMsgID, text string) error {
	formatted := fsapi.FormatFeishuMarkdown(text)
	if replyToMsgID != "" {
		_, err := entry.bot.ReplyCardMessage(ctx, replyToMsgID, formatted)
		return err
	}
	_, err := entry.bot.SendCardMessage(ctx, chatID, formatted)
	return err
}

func (a *Adapter) sendParts(ctx context.Context, entry *botEntry, chatID, replyToMsgID string, parts []interface{}) error {
	var textBuf strings.Builder
	for _, part := range parts {
		m, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		partType, _ := m["type"].(string)
		switch partType {
		case "text":
			if text, ok := m["text"].(string); ok {
				textBuf.WriteString(text)
			}
		case "image_url":
			if err := a.flushText(ctx, entry, chatID, replyToMsgID, &textBuf); err != nil {
				return err
			}
			if imgMap, ok := m["image_url"].(map[string]interface{}); ok {
				if url, ok := imgMap["url"].(string); ok {
					if err := sendImageOrWrapper(ctx, entry, chatID, url, ""); err != nil {
						log.Error("feishu reply: send image: %v", err)
					}
				}
			}
		case "file":
			if err := a.flushText(ctx, entry, chatID, replyToMsgID, &textBuf); err != nil {
				return err
			}
			fileURL, _ := m["file_url"].(string)
			if fileURL == "" {
				if fileMap, ok := m["file"].(map[string]interface{}); ok {
					fileURL, _ = fileMap["url"].(string)
				}
			}
			if fileURL != "" {
				if err := sendFileOrWrapper(ctx, entry, chatID, fileURL, ""); err != nil {
					log.Error("feishu reply: send file: %v", err)
				}
			}
		}
	}
	return a.flushText(ctx, entry, chatID, replyToMsgID, &textBuf)
}

func (a *Adapter) sendPartsTyped(ctx context.Context, entry *botEntry, chatID, replyToMsgID string, parts []agentcontext.ContentPart) error {
	var textBuf strings.Builder
	for _, part := range parts {
		switch part.Type {
		case agentcontext.ContentText:
			textBuf.WriteString(part.Text)
		case agentcontext.ContentImageURL:
			if err := a.flushText(ctx, entry, chatID, replyToMsgID, &textBuf); err != nil {
				return err
			}
			if part.ImageURL != nil {
				if err := sendImageOrWrapper(ctx, entry, chatID, part.ImageURL.URL, ""); err != nil {
					log.Error("feishu reply: send image: %v", err)
				}
			}
		case agentcontext.ContentFile:
			if err := a.flushText(ctx, entry, chatID, replyToMsgID, &textBuf); err != nil {
				return err
			}
			if part.File != nil {
				if err := sendFileOrWrapper(ctx, entry, chatID, part.File.URL, part.File.Filename); err != nil {
					log.Error("feishu reply: send file: %v", err)
				}
			}
		}
	}
	return a.flushText(ctx, entry, chatID, replyToMsgID, &textBuf)
}

func (a *Adapter) flushText(ctx context.Context, entry *botEntry, chatID, replyToMsgID string, buf *strings.Builder) error {
	if buf.Len() == 0 {
		return nil
	}
	text := buf.String()
	buf.Reset()
	return a.sendMarkdown(ctx, entry, chatID, replyToMsgID, text)
}

func sendImageOrWrapper(ctx context.Context, entry *botEntry, chatID, url, caption string) error {
	if isWrapper(url) {
		return entry.bot.SendImageFromWrapper(ctx, chatID, url, caption)
	}
	if strings.HasPrefix(url, "http") {
		text := url
		if caption != "" {
			text = caption + "\n" + url
		}
		_, err := entry.bot.SendTextMessage(ctx, chatID, text)
		return err
	}
	return fmt.Errorf("unsupported image URL scheme: %s", url)
}

func sendFileOrWrapper(ctx context.Context, entry *botEntry, chatID, url, caption string) error {
	if isWrapper(url) {
		return entry.bot.SendFileFromWrapper(ctx, chatID, url, caption)
	}
	if strings.HasPrefix(url, "http") {
		text := url
		if caption != "" {
			text = caption + "\n" + url
		}
		_, err := entry.bot.SendTextMessage(ctx, chatID, text)
		return err
	}
	return fmt.Errorf("unsupported file URL scheme: %s", url)
}

func isWrapper(url string) bool {
	return strings.Contains(url, "://") && !strings.HasPrefix(url, "http")
}

func toContentParts(content interface{}) ([]agentcontext.ContentPart, bool) {
	parts, ok := content.([]agentcontext.ContentPart)
	return parts, ok
}

func (a *Adapter) resolveByChat(metadata *events.MessageMetadata) *botEntry {
	if metadata.AppID != "" {
		if entry, ok := a.resolveByAppID(metadata.AppID); ok {
			return entry
		}
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, entry := range a.bots {
		return entry
	}
	return nil
}
