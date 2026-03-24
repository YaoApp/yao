package weixin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/attachment"
	weixinapi "github.com/yaoapp/yao/integrations/weixin"
)

func (a *Adapter) Reply(ctx context.Context, msg *agentcontext.Message, metadata *events.MessageMetadata) error {
	if msg == nil || metadata == nil {
		return fmt.Errorf("weixin Reply: nil message or metadata")
	}

	entry := a.resolveByAccountID(metadata.AppID)
	if entry == nil {
		a.mu.RLock()
		for _, e := range a.bots {
			entry = e
			break
		}
		a.mu.RUnlock()
	}
	if entry == nil {
		return fmt.Errorf("weixin Reply: no bot registered (appID=%s)", metadata.AppID)
	}

	contextToken, _ := metadata.Extra["context_token"].(string)
	toUserID := metadata.SenderID
	if toUserID == "" {
		toUserID = metadata.ChatID
	}

	ticket := entry.ticketCache.Get(toUserID)
	if ticket == "" {
		if t, err := entry.bot.GetConfig(ctx, toUserID, contextToken); err == nil && t != "" {
			ticket = t
			entry.ticketCache.Set(toUserID, ticket)
		}
	}

	if ticket != "" {
		_ = entry.bot.SendTyping(ctx, toUserID, ticket, 1)
	}

	return a.sendContent(ctx, entry, toUserID, contextToken, msg.Content)
}

func (a *Adapter) sendContent(ctx context.Context, entry *botEntry, toUserID, contextToken string, content interface{}) error {
	switch c := content.(type) {
	case string:
		if strings.TrimSpace(c) == "" {
			return nil
		}
		return entry.bot.SendMessage(ctx, toUserID, contextToken, weixinapi.FormatWeixinText(c))

	case []interface{}:
		return a.sendParts(ctx, entry, toUserID, contextToken, c)

	default:
		parts, ok := toContentParts(content)
		if ok {
			return a.sendPartsTyped(ctx, entry, toUserID, contextToken, parts)
		}
		return entry.bot.SendMessage(ctx, toUserID, contextToken, weixinapi.FormatWeixinText(fmt.Sprintf("%v", content)))
	}
}

func (a *Adapter) sendParts(ctx context.Context, entry *botEntry, toUserID, contextToken string, parts []interface{}) error {
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
			if err := a.flushText(ctx, entry, toUserID, contextToken, &textBuf); err != nil {
				return err
			}
			if imgMap, ok := m["image_url"].(map[string]interface{}); ok {
				if url, ok := imgMap["url"].(string); ok {
					if err := a.sendMediaFromURL(ctx, entry, toUserID, contextToken, url, "", "image"); err != nil {
						log.Error("weixin reply: send image: %v", err)
					}
				}
			}
		case "file":
			if err := a.flushText(ctx, entry, toUserID, contextToken, &textBuf); err != nil {
				return err
			}
			fileURL, _ := m["file_url"].(string)
			fileName, _ := m["file_name"].(string)
			mimeType, _ := m["mime_type"].(string)
			if fileURL == "" {
				if fileMap, ok := m["file"].(map[string]interface{}); ok {
					fileURL, _ = fileMap["url"].(string)
					if fileName == "" {
						fileName, _ = fileMap["filename"].(string)
					}
				}
			}
			if fileURL != "" {
				mediaHint := detectMediaHint(mimeType, fileName)
				if err := a.sendMediaFromURL(ctx, entry, toUserID, contextToken, fileURL, fileName, mediaHint); err != nil {
					log.Error("weixin reply: send file: %v", err)
				}
			}
		}
	}
	return a.flushText(ctx, entry, toUserID, contextToken, &textBuf)
}

func (a *Adapter) sendPartsTyped(ctx context.Context, entry *botEntry, toUserID, contextToken string, parts []agentcontext.ContentPart) error {
	var textBuf strings.Builder
	for _, part := range parts {
		switch part.Type {
		case agentcontext.ContentText:
			textBuf.WriteString(part.Text)
		case agentcontext.ContentImageURL:
			if err := a.flushText(ctx, entry, toUserID, contextToken, &textBuf); err != nil {
				return err
			}
			if part.ImageURL != nil {
				if err := a.sendMediaFromURL(ctx, entry, toUserID, contextToken, part.ImageURL.URL, "", "image"); err != nil {
					log.Error("weixin reply: send image: %v", err)
				}
			}
		case agentcontext.ContentFile:
			if err := a.flushText(ctx, entry, toUserID, contextToken, &textBuf); err != nil {
				return err
			}
			if part.File != nil {
				mediaHint := detectMediaHint("", part.File.Filename)
				if err := a.sendMediaFromURL(ctx, entry, toUserID, contextToken, part.File.URL, part.File.Filename, mediaHint); err != nil {
					log.Error("weixin reply: send file: %v", err)
				}
			}
		}
	}
	return a.flushText(ctx, entry, toUserID, contextToken, &textBuf)
}

func (a *Adapter) flushText(ctx context.Context, entry *botEntry, toUserID, contextToken string, buf *strings.Builder) error {
	if buf.Len() == 0 {
		return nil
	}
	text := weixinapi.FormatWeixinText(buf.String())
	buf.Reset()
	return entry.bot.SendMessage(ctx, toUserID, contextToken, text)
}

func (a *Adapter) sendMediaFromURL(ctx context.Context, entry *botEntry, toUserID, contextToken, fileURL, fileName, mediaHint string) error {
	log.Info("weixin sendMedia: to=%s url=%s fileName=%q hint=%s contextToken_len=%d",
		toUserID, fileURL, fileName, mediaHint, len(contextToken))

	var plaintext []byte
	var contentType string

	if isWrapper(fileURL) {
		managerName, fileID, err := parseWrapper(fileURL)
		if err != nil {
			return err
		}
		log.Info("weixin sendMedia: wrapper manager=%s fileID=%s", managerName, fileID)
		manager, exists := attachment.Managers[managerName]
		if !exists {
			return fmt.Errorf("attachment manager %s not found", managerName)
		}
		resp, err := manager.Download(ctx, fileID)
		if err != nil {
			return fmt.Errorf("attachment download %s: %w", fileID, err)
		}
		defer resp.Reader.Close()
		plaintext, err = io.ReadAll(resp.Reader)
		if err != nil {
			return fmt.Errorf("read attachment %s: %w", fileID, err)
		}
		contentType = resp.ContentType
		if fileName == "" {
			fileName = fileID + resp.Extension
		}
		log.Info("weixin sendMedia: attachment downloaded bytes=%d contentType=%q fileName=%q", len(plaintext), contentType, fileName)
	} else if strings.HasPrefix(fileURL, "http") {
		resp, err := http.Get(fileURL) //nolint:gosec
		if err != nil {
			return fmt.Errorf("download %s: %w", fileURL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download %s: HTTP %d", fileURL, resp.StatusCode)
		}
		plaintext, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read %s: %w", fileURL, err)
		}
		contentType = resp.Header.Get("Content-Type")
		log.Info("weixin sendMedia: http downloaded bytes=%d contentType=%q", len(plaintext), contentType)
	} else {
		return fmt.Errorf("unsupported URL scheme: %s", fileURL)
	}

	if mediaHint == "" {
		mediaHint = detectMediaHint(contentType, fileName)
	}

	var mediaType int
	switch mediaHint {
	case "image":
		mediaType = weixinapi.UploadMediaImage
	case "video":
		mediaType = weixinapi.UploadMediaVideo
	default:
		mediaType = weixinapi.UploadMediaFile
	}

	log.Info("weixin sendMedia: uploading media_type=%d mediaHint=%s bytes=%d to=%s", mediaType, mediaHint, len(plaintext), toUserID)
	uploaded, err := entry.bot.UploadMedia(ctx, plaintext, toUserID, mediaType)
	if err != nil {
		log.Error("weixin UploadMedia failed: media_type=%d mediaHint=%s bytes=%d to=%s err=%v", mediaType, mediaHint, len(plaintext), toUserID, err)
		fallbackText := fileURL
		if fileName != "" {
			fallbackText = fileName + "\n" + fileURL
		}
		return entry.bot.SendMessage(ctx, toUserID, contextToken, fallbackText)
	}

	switch mediaHint {
	case "image":
		return entry.bot.SendImageMessage(ctx, toUserID, contextToken, uploaded)
	case "video":
		return entry.bot.SendVideoMessage(ctx, toUserID, contextToken, uploaded)
	default:
		if fileName == "" {
			fileName = "file.bin"
		}
		return entry.bot.SendFileMessage(ctx, toUserID, contextToken, fileName, uploaded)
	}
}

func detectMediaHint(mimeType, fileName string) string {
	lower := strings.ToLower(mimeType)
	if strings.HasPrefix(lower, "image/") {
		return "image"
	}
	if strings.HasPrefix(lower, "video/") {
		return "video"
	}
	// TODO(weixin-voice): audio/* detected as "file" because iLink Bot voice
	// playback is not yet functional. Switch to "voice" once supported.
	if fileName != "" {
		ext := strings.ToLower(fileName)
		if strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") ||
			strings.HasSuffix(ext, ".png") || strings.HasSuffix(ext, ".gif") ||
			strings.HasSuffix(ext, ".webp") || strings.HasSuffix(ext, ".bmp") {
			return "image"
		}
		if strings.HasSuffix(ext, ".mp4") || strings.HasSuffix(ext, ".mov") ||
			strings.HasSuffix(ext, ".avi") || strings.HasSuffix(ext, ".webm") {
			return "video"
		}
	}
	return "file"
}

func isWrapper(url string) bool {
	return strings.Contains(url, "://") && !strings.HasPrefix(url, "http")
}

func parseWrapper(wrapper string) (managerName, fileID string, err error) {
	idx := strings.Index(wrapper, "://")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid wrapper: %s", wrapper)
	}
	return wrapper[:idx], wrapper[idx+3:], nil
}

func toContentParts(content interface{}) ([]agentcontext.ContentPart, bool) {
	parts, ok := content.([]agentcontext.ContentPart)
	return parts, ok
}

func (a *Adapter) resolveByAccountID(accountID string) *botEntry {
	if accountID == "" {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	robotID, ok := a.accountIdx[accountID]
	if !ok {
		return nil
	}
	return a.bots[robotID]
}
