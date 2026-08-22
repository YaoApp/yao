package shared

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	goullm "github.com/yaoapp/gou/llm"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/attachment"
	workspace "github.com/yaoapp/yao/tai/workspace"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	MaxSingleImageBase64 = 20 * 1024 * 1024
	MaxTotalImageBase64  = 48 * 1024 * 1024
	MaxImagesPerMessage  = 10
	DefaultMaxSize       = 1080
)

// ImageBlock holds one base64-encoded image ready for model input.
type ImageBlock struct {
	MediaType string
	Data      string
	Filename  string
}

// MessageParts holds extracted text and image content from the last user message.
type MessageParts struct {
	TextParts   []string
	ImageBlocks []ImageBlock
}

// HasVision checks whether the primary connector declares image input capability.
func HasVision(conn interface{}) bool {
	if conn == nil {
		return false
	}
	lc, ok := conn.(goullm.LLMConnector)
	if !ok {
		return false
	}
	caps := lc.GetCapabilities()
	return caps != nil && caps.HasVision()
}

// ExtractMessageParts wraps PrepareAttachments, then — when vision is true —
// scans image attachments from the original messages and produces resized base64
// ImageBlocks for inline model input. The high-res originals remain in workspace.
func ExtractMessageParts(
	ctx context.Context,
	messages []agentContext.Message,
	chatID string,
	ws workspace.FS,
	vision bool,
) ([]agentContext.Message, *MessageParts, error) {
	processed, _, err := PrepareAttachments(ctx, messages, chatID, ws)
	if err != nil {
		return nil, nil, err
	}

	parts := &MessageParts{}
	for i := len(processed) - 1; i >= 0; i-- {
		if processed[i].Role != "user" {
			continue
		}
		if s, ok := processed[i].Content.(string); ok && s != "" {
			parts.TextParts = []string{s}
		}
		break
	}

	if vision {
		infos := scanLastUserImageAttachments(ctx, messages)
		if len(infos) > 0 {
			totalBytes := 0
			for _, info := range infos {
				if len(parts.ImageBlocks) >= MaxImagesPerMessage {
					break
				}
				mgr, exists := attachment.Managers[info.uploaderName]
				if !exists {
					continue
				}
				data, err := mgr.Read(ctx, info.fileID)
				if err != nil {
					continue
				}
				resized, mime, resizeErr := ResizeForVision(data, DefaultMaxSize)
				if resizeErr != nil {
					continue
				}
				encoded := base64.StdEncoding.EncodeToString(resized)
				if len(encoded) > MaxSingleImageBase64 {
					continue
				}
				if totalBytes+len(encoded) > MaxTotalImageBase64 {
					break
				}
				totalBytes += len(encoded)
				parts.ImageBlocks = append(parts.ImageBlocks, ImageBlock{
					MediaType: mime,
					Data:      encoded,
					Filename:  info.filename,
				})
			}
		}
	}

	return processed, parts, nil
}

type imageAttachmentInfo struct {
	uploaderName string
	fileID       string
	filename     string
}

func scanLastUserImageAttachments(ctx context.Context, messages []agentContext.Message) []imageAttachmentInfo {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		parts, ok := messages[i].Content.([]interface{})
		if !ok {
			if typedParts, ok := messages[i].Content.([]agentContext.ContentPart); ok {
				iparts := make([]interface{}, len(typedParts))
				for j, p := range typedParts {
					m := map[string]interface{}{"type": string(p.Type)}
					if p.ImageURL != nil {
						m["image_url"] = map[string]interface{}{"url": p.ImageURL.URL}
					}
					if p.File != nil {
						m["file"] = map[string]interface{}{"url": p.File.URL, "filename": p.File.Filename}
					}
					iparts[j] = m
				}
				parts = iparts
			} else {
				return nil
			}
		}

		var infos []imageAttachmentInfo
		for _, item := range parts {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			partType, _ := m["type"].(string)
			var url, hintName string

			switch partType {
			case "image_url":
				imgData, _ := m["image_url"].(map[string]interface{})
				if imgData == nil {
					continue
				}
				url, _ = imgData["url"].(string)
			case "file":
				fileData, _ := m["file"].(map[string]interface{})
				if fileData == nil {
					continue
				}
				url, _ = fileData["url"].(string)
				hintName, _ = fileData["filename"].(string)
			default:
				continue
			}

			if url == "" {
				continue
			}
			uploaderName, fileID, isWrapper := attachment.Parse(url)
			if !isWrapper {
				continue
			}
			fileInfo, err := attachment.Managers[uploaderName].Info(ctx, fileID)
			if err != nil {
				continue
			}
			if !isImageMIME(fileInfo.ContentType) {
				continue
			}
			name := fileInfo.Filename
			if name == "" {
				name = hintName
			}
			infos = append(infos, imageAttachmentInfo{
				uploaderName: uploaderName,
				fileID:       fileID,
				filename:     name,
			})
		}
		return infos
	}
	return nil
}

func isImageMIME(ct string) bool {
	switch ct {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	}
	return false
}

// ResizeForVision decodes an image, resizes it to fit within maxSize on both
// dimensions (preserving aspect ratio), and re-encodes to the same format.
// Returns encoded bytes, MIME type, and error.
func ResizeForVision(data []byte, maxSize int) ([]byte, string, error) {
	reader := bytes.NewReader(data)
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, "", err
	}

	mime := "image/" + format
	if format == "jpeg" {
		mime = "image/jpeg"
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= maxSize && h <= maxSize {
		return data, mime, nil
	}

	newW, newH := w, h
	if w > h {
		newW = maxSize
		newH = int(float64(h) * float64(maxSize) / float64(w))
	} else {
		newH = maxSize
		newW = int(float64(w) * float64(maxSize) / float64(h))
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var buf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&buf, dst)
	case "gif":
		err = gif.Encode(&buf, dst, nil)
	default:
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85})
		mime = "image/jpeg"
	}
	if err != nil {
		return nil, "", err
	}

	return buf.Bytes(), mime, nil
}
