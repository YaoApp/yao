package weixin

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/yaoapp/yao/attachment"
	weixinapi "github.com/yaoapp/yao/integrations/weixin"
)

type resolvedMedia struct {
	Wrapper  string
	MimeType string
	FileName string
}

func convertMessage(ctx context.Context, bot *weixinapi.Bot, items []weixinapi.MsgItem, groups []string) (string, []resolvedMedia) {
	var textBuf strings.Builder
	var media []resolvedMedia

	for _, item := range items {
		switch item.Type {
		case weixinapi.ItemTypeText:
			if item.TextItem != nil && item.TextItem.Text != "" {
				text := item.TextItem.Text
				if item.RefMsg != nil {
					text = formatRefMessage(item.RefMsg, text)
				}
				textBuf.WriteString(text)
			}
		case weixinapi.ItemTypeVoice:
			if item.VoiceItem != nil {
				if item.VoiceItem.Text != "" {
					textBuf.WriteString(item.VoiceItem.Text)
				} else if item.VoiceItem.Media != nil && item.VoiceItem.Media.EncryptQueryParam != "" {
					m := resolveVoice(ctx, bot, item.VoiceItem, groups)
					if m != nil {
						media = append(media, *m)
					}
				}
			}
		case weixinapi.ItemTypeImage:
			if item.ImageItem != nil {
				m := resolveImage(ctx, bot, item.ImageItem, groups)
				if m != nil {
					media = append(media, *m)
				}
			}
		case weixinapi.ItemTypeFile:
			if item.FileItem != nil && item.FileItem.Media != nil && item.FileItem.Media.EncryptQueryParam != "" {
				m := resolveFile(ctx, bot, item.FileItem, groups)
				if m != nil {
					media = append(media, *m)
				}
			}
		case weixinapi.ItemTypeVideo:
			if item.VideoItem != nil && item.VideoItem.Media != nil && item.VideoItem.Media.EncryptQueryParam != "" {
				m := resolveVideo(ctx, bot, item.VideoItem, groups)
				if m != nil {
					media = append(media, *m)
				}
			}
		}
	}

	return textBuf.String(), media
}

func formatRefMessage(ref *weixinapi.RefMessage, text string) string {
	if ref == nil || ref.MessageItem == nil {
		return text
	}
	var refBody string
	if ref.MessageItem.TextItem != nil {
		refBody = ref.MessageItem.TextItem.Text
	}
	title := ref.Title
	if title == "" && refBody == "" {
		return text
	}
	return fmt.Sprintf("[引用: %s | %s]\n%s", title, refBody, text)
}

func resolveImage(ctx context.Context, bot *weixinapi.Bot, img *weixinapi.ImageItem, groups []string) *resolvedMedia {
	if img.AesKey != "" && img.Media != nil && img.Media.EncryptQueryParam != "" {
		rawKey, err := hex.DecodeString(img.AesKey)
		if err == nil {
			data, err := weixinapi.DecryptFromRaw(bot.CDNBaseURL(), img.Media.EncryptQueryParam, rawKey)
			if err == nil {
				return storeMedia(ctx, data, "image/jpeg", "image.jpg", groups)
			}
		}
	}
	if img.Media != nil && img.Media.EncryptQueryParam != "" && img.Media.AesKey != "" {
		data, err := weixinapi.DownloadAndDecrypt(bot.CDNBaseURL(), img.Media.EncryptQueryParam, img.Media.AesKey)
		if err == nil {
			return storeMedia(ctx, data, "image/jpeg", "image.jpg", groups)
		}
	}
	return nil
}

func resolveVoice(ctx context.Context, bot *weixinapi.Bot, voice *weixinapi.VoiceItem, groups []string) *resolvedMedia {
	data, err := weixinapi.DownloadAndDecrypt(bot.CDNBaseURL(), voice.Media.EncryptQueryParam, voice.Media.AesKey)
	if err != nil {
		log.Error("weixin: voice decrypt failed: %v", err)
		return nil
	}
	mime := "audio/mpeg"
	ext := "mp3"
	if voice.EncodeType == 6 {
		mime = "audio/silk"
		ext = "silk"
	}
	return storeMedia(ctx, data, mime, "voice."+ext, groups)
}

func resolveFile(ctx context.Context, bot *weixinapi.Bot, file *weixinapi.FileItem, groups []string) *resolvedMedia {
	data, err := weixinapi.DownloadAndDecrypt(bot.CDNBaseURL(), file.Media.EncryptQueryParam, file.Media.AesKey)
	if err != nil {
		log.Error("weixin: file decrypt failed: %v", err)
		return nil
	}
	filename := file.FileName
	if filename == "" {
		filename = "file.bin"
	}
	mime := weixinapi.MimeFromFilename(filename)
	return storeMedia(ctx, data, mime, filename, groups)
}

func resolveVideo(ctx context.Context, bot *weixinapi.Bot, video *weixinapi.VideoItem, groups []string) *resolvedMedia {
	data, err := weixinapi.DownloadAndDecrypt(bot.CDNBaseURL(), video.Media.EncryptQueryParam, video.Media.AesKey)
	if err != nil {
		log.Error("weixin: video decrypt failed: %v", err)
		return nil
	}
	return storeMedia(ctx, data, "video/mp4", "video.mp4", groups)
}

func storeMedia(ctx context.Context, data []byte, mimeType, filename string, groups []string) *resolvedMedia {
	manager, exists := attachment.Managers["__yao.attachment"]
	if !exists {
		log.Error("weixin: __yao.attachment manager not found")
		return nil
	}

	fh := makeFileHeader(filename, mimeType, int64(len(data)))
	reader := bytes.NewReader(data)
	file, err := manager.Upload(ctx, fh, reader, attachment.UploadOption{Groups: groups})
	if err != nil {
		log.Error("weixin: attachment upload failed: %v", err)
		return nil
	}

	return &resolvedMedia{
		Wrapper:  fmt.Sprintf("__yao.attachment://%s", file.ID),
		MimeType: mimeType,
		FileName: filename,
	}
}

func makeFileHeader(filename, contentType string, size int64) *attachment.FileHeader {
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	hdr.Set("Content-Type", contentType)
	return &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: filename,
			Header:   hdr,
			Size:     size,
		},
	}
}
