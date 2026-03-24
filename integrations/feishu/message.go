package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/yaoapp/yao/attachment"
)

// SendTyping is a placeholder for typing indicator support.
// Feishu does not provide a public typing status API; this is a no-op
// so callers can use a uniform interface across all adapters.
func (b *Bot) SendTyping(ctx context.Context, chatID string) error {
	return nil
}

// SendTextMessage sends a text message to a chat.
func (b *Bot) SendTextMessage(ctx context.Context, chatID, text string) (string, error) {
	content, _ := json.Marshal(map[string]string{"text": text})
	return b.sendMessage(ctx, "chat_id", chatID, "text", string(content))
}

// SendTextToUser sends a text message to a user by open_id.
func (b *Bot) SendTextToUser(ctx context.Context, openID, text string) (string, error) {
	content, _ := json.Marshal(map[string]string{"text": text})
	return b.sendMessage(ctx, "open_id", openID, "text", string(content))
}

// SendCardMessage sends a Markdown-rendered interactive card message to a chat.
// Feishu's text type doesn't render Markdown; the interactive card type does.
func (b *Bot) SendCardMessage(ctx context.Context, chatID, markdown string) (string, error) {
	card := buildMarkdownCard(markdown)
	content, _ := json.Marshal(card)
	return b.sendMessage(ctx, "chat_id", chatID, "interactive", string(content))
}

// ReplyCardMessage replies with a Markdown-rendered interactive card.
func (b *Bot) ReplyCardMessage(ctx context.Context, messageID, markdown string) (string, error) {
	card := buildMarkdownCard(markdown)
	content, _ := json.Marshal(card)
	return b.replyMessage(ctx, messageID, "interactive", string(content))
}

// buildMarkdownCard constructs a Feishu interactive card with lark_md content.
// Uses the non-template card structure: config + elements (div with lark_md).
func buildMarkdownCard(markdown string) map[string]interface{} {
	return map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": markdown,
				},
			},
		},
	}
}

// SendImageMessage sends an image by image_key to a chat.
func (b *Bot) SendImageMessage(ctx context.Context, chatID, imageKey string) (string, error) {
	content, _ := json.Marshal(map[string]string{"image_key": imageKey})
	return b.sendMessage(ctx, "chat_id", chatID, "image", string(content))
}

// SendFileMessage sends a file by file_key to a chat.
func (b *Bot) SendFileMessage(ctx context.Context, chatID, fileKey string) (string, error) {
	content, _ := json.Marshal(map[string]string{"file_key": fileKey})
	return b.sendMessage(ctx, "chat_id", chatID, "file", string(content))
}

// ReplyTextMessage replies to a message with text.
func (b *Bot) ReplyTextMessage(ctx context.Context, messageID, text string) (string, error) {
	content, _ := json.Marshal(map[string]string{"text": text})
	return b.replyMessage(ctx, messageID, "text", string(content))
}

func (b *Bot) sendMessage(ctx context.Context, receiveIDType, receiveID, msgType, content string) (string, error) {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType(msgType).
			Content(content).
			Build()).
		Build()

	resp, err := b.client.Im.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu send message: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu send message: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data != nil && resp.Data.MessageId != nil {
		return *resp.Data.MessageId, nil
	}
	return "", nil
}

// UploadImage uploads an image to Feishu and returns the image_key.
func (b *Bot) UploadImage(ctx context.Context, filename string, reader io.Reader) (string, error) {
	req := larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType("message").
			Image(reader).
			Build()).
		Build()

	resp, err := b.client.Im.Image.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu upload image: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu upload image: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.ImageKey == nil {
		return "", fmt.Errorf("feishu upload image: empty image_key in response")
	}
	return *resp.Data.ImageKey, nil
}

// UploadFile uploads a file to Feishu and returns the file_key.
// fileType must be one of: opus, mp4, pdf, doc, xls, ppt, stream.
func (b *Bot) UploadFile(ctx context.Context, filename, fileType string, reader io.Reader) (string, error) {
	req := larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileType(fileType).
			FileName(filename).
			File(reader).
			Build()).
		Build()

	resp, err := b.client.Im.File.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu upload file: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu upload file: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.FileKey == nil {
		return "", fmt.Errorf("feishu upload file: empty file_key in response")
	}
	return *resp.Data.FileKey, nil
}

// SendImageFromWrapper sends an image from a Yao attachment wrapper (e.g. "__yao.attachment://xxx").
func (b *Bot) SendImageFromWrapper(ctx context.Context, chatID, wrapper, caption string) error {
	managerName, fileID, ok := attachment.Parse(wrapper)
	if !ok {
		return fmt.Errorf("invalid attachment wrapper: %s", wrapper)
	}

	manager, exists := attachment.Managers[managerName]
	if !exists {
		return fmt.Errorf("attachment manager %s not found", managerName)
	}

	resp, err := manager.Download(ctx, fileID)
	if err != nil {
		return fmt.Errorf("attachment download %s: %w", fileID, err)
	}
	defer resp.Reader.Close()

	filename := fileID + resp.Extension
	imageKey, err := b.UploadImage(ctx, filename, resp.Reader)
	if err != nil {
		return err
	}

	if caption != "" {
		if _, err := b.SendTextMessage(ctx, chatID, caption); err != nil {
			return err
		}
	}
	_, err = b.SendImageMessage(ctx, chatID, imageKey)
	return err
}

// SendFileFromWrapper sends a file from a Yao attachment wrapper (e.g. "__yao.attachment://xxx").
func (b *Bot) SendFileFromWrapper(ctx context.Context, chatID, wrapper, caption string) error {
	managerName, fileID, ok := attachment.Parse(wrapper)
	if !ok {
		return fmt.Errorf("invalid attachment wrapper: %s", wrapper)
	}

	manager, exists := attachment.Managers[managerName]
	if !exists {
		return fmt.Errorf("attachment manager %s not found", managerName)
	}

	resp, err := manager.Download(ctx, fileID)
	if err != nil {
		return fmt.Errorf("attachment download %s: %w", fileID, err)
	}
	defer resp.Reader.Close()

	filename := fileID + resp.Extension
	fileType := detectFeishuFileType(resp.ContentType, resp.Extension)

	fileKey, err := b.UploadFile(ctx, filename, fileType, resp.Reader)
	if err != nil {
		return err
	}

	if caption != "" {
		if _, err := b.SendTextMessage(ctx, chatID, caption); err != nil {
			return err
		}
	}
	_, err = b.SendFileMessage(ctx, chatID, fileKey)
	return err
}

// detectFeishuFileType maps a MIME type / extension to a Feishu file type.
func detectFeishuFileType(contentType, ext string) string {
	lower := strings.ToLower(contentType)
	switch {
	case strings.Contains(lower, "audio/ogg"), strings.Contains(lower, "audio/opus"):
		return "opus"
	case strings.HasPrefix(lower, "video/"):
		return "mp4"
	case strings.Contains(lower, "pdf"):
		return "pdf"
	case strings.Contains(lower, "msword"),
		strings.Contains(lower, "wordprocessingml"),
		strings.Contains(lower, "opendocument.text"):
		return "doc"
	case strings.Contains(lower, "ms-excel"),
		strings.Contains(lower, "spreadsheetml"),
		strings.Contains(lower, "opendocument.spreadsheet"):
		return "xls"
	case strings.Contains(lower, "ms-powerpoint"),
		strings.Contains(lower, "presentationml"),
		strings.Contains(lower, "opendocument.presentation"):
		return "ppt"
	}

	switch strings.ToLower(filepath.Ext(ext)) {
	case ".pdf":
		return "pdf"
	case ".doc", ".docx":
		return "doc"
	case ".xls", ".xlsx":
		return "xls"
	case ".ppt", ".pptx":
		return "ppt"
	case ".mp4", ".mov", ".avi":
		return "mp4"
	case ".opus", ".ogg":
		return "opus"
	}
	return "stream"
}

func (b *Bot) replyMessage(ctx context.Context, messageID, msgType, content string) (string, error) {
	req := larkim.NewReplyMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(msgType).
			Content(content).
			Build()).
		Build()

	resp, err := b.client.Im.Message.Reply(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu reply message: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu reply message: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data != nil && resp.Data.MessageId != nil {
		return *resp.Data.MessageId, nil
	}
	return "", nil
}
