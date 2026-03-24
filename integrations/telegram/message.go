package telegram

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/yaoapp/yao/attachment"
)

// SendTyping sends a "typing" chat action to indicate the bot is preparing a response.
func (b *Bot) SendTyping(ctx context.Context, chatID int64) error {
	sdk, err := b.sdk()
	if err != nil {
		return err
	}
	_, err = sdk.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: chatID,
		Action: models.ChatActionTyping,
	})
	return err
}

// SendMessage sends a message to a chat. If the text contains Markdown formatting,
// it is automatically converted to Telegram-compatible HTML.
func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string, replyTo int64) error {
	sdk, err := b.sdk()
	if err != nil {
		return err
	}

	formatted := FormatTelegramHTML(text)
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      formatted,
		ParseMode: models.ParseModeHTML,
	}
	if replyTo > 0 {
		params.ReplyParameters = &models.ReplyParameters{MessageID: int(replyTo)}
	}
	_, err = sdk.SendMessage(ctx, params)
	return err
}

// MediaType indicates which Telegram send method to use.
type MediaType string

const (
	MediaPhoto     MediaType = "photo"
	MediaDocument  MediaType = "document"
	MediaAudio     MediaType = "audio"
	MediaVideo     MediaType = "video"
	MediaVoice     MediaType = "voice"
	MediaAnimation MediaType = "animation"
	MediaSticker   MediaType = "sticker"
)

// SendMedia sends a media message from a Yao attachment wrapper
// (e.g. "__yao.attachment://ccd472d11feb96e03a3fc468f494045c").
// It reads the file from the attachment manager, detects the media type from
// the stored content type, and uploads it to the Telegram chat.
func (b *Bot) SendMedia(ctx context.Context, chatID int64, wrapper string, caption string, replyTo int64) error {
	managerName, fileID, err := parseWrapper(wrapper)
	if err != nil {
		return err
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

	mediaType := DetectMediaType(resp.ContentType)
	filename := fileID + resp.Extension

	file := &models.InputFileUpload{Filename: filename, Data: resp.Reader}
	return b.sendMedia(ctx, chatID, mediaType, file, caption, replyTo)
}

// SendMediaByURL sends a media message from a public URL.
// Telegram downloads the file directly from the URL.
func (b *Bot) SendMediaByURL(ctx context.Context, chatID int64, mediaType MediaType, url string, caption string, replyTo int64) error {
	file := &models.InputFileString{Data: url}
	return b.sendMedia(ctx, chatID, mediaType, file, caption, replyTo)
}

// SendMediaByReader sends a media message by uploading raw bytes.
func (b *Bot) SendMediaByReader(ctx context.Context, chatID int64, mediaType MediaType, filename string, data io.Reader, caption string, replyTo int64) error {
	file := &models.InputFileUpload{Filename: filename, Data: data}
	return b.sendMedia(ctx, chatID, mediaType, file, caption, replyTo)
}

// DetectMediaType guesses the MediaType from a MIME string.
// Falls back to MediaDocument for unknown types.
func DetectMediaType(mimeType string) MediaType {
	lower := strings.ToLower(mimeType)
	switch {
	case strings.HasPrefix(lower, "image/webp"):
		return MediaSticker
	case strings.HasPrefix(lower, "image/gif"):
		return MediaAnimation
	case strings.HasPrefix(lower, "image/"):
		return MediaPhoto
	case strings.HasPrefix(lower, "video/"):
		return MediaVideo
	case strings.HasPrefix(lower, "audio/ogg"):
		return MediaVoice
	case strings.HasPrefix(lower, "audio/"):
		return MediaAudio
	default:
		return MediaDocument
	}
}

// parseWrapper splits "__yao.attachment://fileID" into manager name and file ID.
func parseWrapper(wrapper string) (managerName string, fileID string, err error) {
	idx := strings.Index(wrapper, "://")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid attachment wrapper: %s", wrapper)
	}
	return wrapper[:idx], wrapper[idx+3:], nil
}

func (b *Bot) sendMedia(ctx context.Context, chatID int64, mediaType MediaType, file models.InputFile, caption string, replyTo int64) error {
	sdk, err := b.sdk()
	if err != nil {
		return err
	}

	var replyParams *models.ReplyParameters
	if replyTo > 0 {
		replyParams = &models.ReplyParameters{MessageID: int(replyTo)}
	}

	htmlCaption := FormatTelegramHTML(caption)

	switch mediaType {
	case MediaPhoto:
		_, err = sdk.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:          chatID,
			Photo:           file,
			Caption:         htmlCaption,
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: replyParams,
		})

	case MediaDocument:
		_, err = sdk.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:          chatID,
			Document:        file,
			Caption:         htmlCaption,
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: replyParams,
		})

	case MediaAudio:
		_, err = sdk.SendAudio(ctx, &bot.SendAudioParams{
			ChatID:          chatID,
			Audio:           file,
			Caption:         htmlCaption,
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: replyParams,
		})

	case MediaVideo:
		_, err = sdk.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:          chatID,
			Video:           file,
			Caption:         htmlCaption,
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: replyParams,
		})

	case MediaVoice:
		_, err = sdk.SendVoice(ctx, &bot.SendVoiceParams{
			ChatID:          chatID,
			Voice:           file,
			Caption:         htmlCaption,
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: replyParams,
		})

	case MediaAnimation:
		_, err = sdk.SendAnimation(ctx, &bot.SendAnimationParams{
			ChatID:          chatID,
			Animation:       file,
			Caption:         htmlCaption,
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: replyParams,
		})

	case MediaSticker:
		_, err = sdk.SendSticker(ctx, &bot.SendStickerParams{
			ChatID:          chatID,
			Sticker:         file,
			ReplyParameters: replyParams,
		})

	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}

	return err
}
