package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	tgapi "github.com/yaoapp/yao/integrations/telegram"
)

// Reply sends the assistant message back to the originating Telegram chat.
// Content may be a plain string or []ContentPart (text, image_url, file, etc.).
// Each adapter is responsible for interpreting the standard message format.
func (a *Adapter) Reply(ctx context.Context, msg *agentcontext.Message, metadata *events.MessageMetadata) error {
	if msg == nil || metadata == nil {
		return fmt.Errorf("nil message or metadata")
	}

	chatID, err := strconv.ParseInt(metadata.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id %q: %w", metadata.ChatID, err)
	}

	var replyTo int64
	if metadata.Extra != nil {
		if v, ok := metadata.Extra["tg_message_id"]; ok {
			switch id := v.(type) {
			case int64:
				replyTo = id
			case float64:
				replyTo = int64(id)
			}
		}
	}

	entry := a.resolveByChat(metadata)
	if entry == nil {
		return fmt.Errorf("no bot registered for channel metadata (appID=%s)", metadata.AppID)
	}

	if err := entry.bot.SendTyping(ctx, chatID); err != nil {
		log.Debug("telegram reply: send typing failed: %v", err)
	}

	return a.sendContent(ctx, entry.bot, chatID, replyTo, msg.Content)
}

// sendContent dispatches based on the Content type.
func (a *Adapter) sendContent(ctx context.Context, bot *tgapi.Bot, chatID, replyTo int64, content interface{}) error {
	switch c := content.(type) {
	case string:
		if strings.TrimSpace(c) == "" {
			return nil
		}
		return bot.SendMessage(ctx, chatID, c, replyTo)

	case []interface{}:
		return a.sendParts(ctx, bot, chatID, replyTo, c)

	default:
		parts, ok := toContentParts(content)
		if ok {
			return a.sendPartsTyped(ctx, bot, chatID, replyTo, parts)
		}
		return bot.SendMessage(ctx, chatID, fmt.Sprintf("%v", content), replyTo)
	}
}

// sendParts handles []interface{} content parts (common from JSON unmarshalling).
func (a *Adapter) sendParts(ctx context.Context, bot *tgapi.Bot, chatID, replyTo int64, parts []interface{}) error {
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
			if err := a.flushText(ctx, bot, chatID, replyTo, &textBuf); err != nil {
				return err
			}
			if imgMap, ok := m["image_url"].(map[string]interface{}); ok {
				if url, ok := imgMap["url"].(string); ok {
					if err := sendFileOrWrapper(ctx, bot, chatID, replyTo, url, ""); err != nil {
						log.Error("telegram reply: send image: %v", err)
					}
				}
			}
		case "file":
			if err := a.flushText(ctx, bot, chatID, replyTo, &textBuf); err != nil {
				return err
			}
			if fileMap, ok := m["file"].(map[string]interface{}); ok {
				url, _ := fileMap["url"].(string)
				filename, _ := fileMap["filename"].(string)
				if url != "" {
					if err := sendFileOrWrapper(ctx, bot, chatID, replyTo, url, filename); err != nil {
						log.Error("telegram reply: send file: %v", err)
					}
				}
			}
		}
	}
	return a.flushText(ctx, bot, chatID, replyTo, &textBuf)
}

// sendPartsTyped handles typed []agentcontext.ContentPart slices.
func (a *Adapter) sendPartsTyped(ctx context.Context, bot *tgapi.Bot, chatID, replyTo int64, parts []agentcontext.ContentPart) error {
	var textBuf strings.Builder
	for _, part := range parts {
		switch part.Type {
		case agentcontext.ContentText:
			textBuf.WriteString(part.Text)
		case agentcontext.ContentImageURL:
			if err := a.flushText(ctx, bot, chatID, replyTo, &textBuf); err != nil {
				return err
			}
			if part.ImageURL != nil {
				if err := sendFileOrWrapper(ctx, bot, chatID, replyTo, part.ImageURL.URL, ""); err != nil {
					log.Error("telegram reply: send image: %v", err)
				}
			}
		case agentcontext.ContentFile:
			if err := a.flushText(ctx, bot, chatID, replyTo, &textBuf); err != nil {
				return err
			}
			if part.File != nil {
				if err := sendFileOrWrapper(ctx, bot, chatID, replyTo, part.File.URL, part.File.Filename); err != nil {
					log.Error("telegram reply: send file: %v", err)
				}
			}
		}
	}
	return a.flushText(ctx, bot, chatID, replyTo, &textBuf)
}

func (a *Adapter) flushText(ctx context.Context, bot *tgapi.Bot, chatID, replyTo int64, buf *strings.Builder) error {
	if buf.Len() == 0 {
		return nil
	}
	err := bot.SendMessage(ctx, chatID, buf.String(), replyTo)
	buf.Reset()
	return err
}

// sendFileOrWrapper sends a file from a wrapper (__yao.attachment://xxx) or URL.
func sendFileOrWrapper(ctx context.Context, bot *tgapi.Bot, chatID, replyTo int64, url, caption string) error {
	if strings.Contains(url, "://") && !strings.HasPrefix(url, "http") {
		return bot.SendMedia(ctx, chatID, url, caption, replyTo)
	}
	if strings.HasPrefix(url, "http") {
		mediaType := tgapi.DetectMediaType("")
		return bot.SendMediaByURL(ctx, chatID, mediaType, url, caption, replyTo)
	}
	return fmt.Errorf("unsupported file URL scheme: %s", url)
}

// toContentParts tries to type-assert content to []agentcontext.ContentPart.
func toContentParts(content interface{}) ([]agentcontext.ContentPart, bool) {
	parts, ok := content.([]agentcontext.ContentPart)
	return parts, ok
}

// resolveByChat finds the bot entry matching the metadata.
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
