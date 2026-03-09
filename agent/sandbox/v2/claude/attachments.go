package claude

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/attachment"
	workspace "github.com/yaoapp/yao/tai/workspace"
)

// prepareAttachments resolves __yao.attachment:// URLs in messages,
// copies actual files into the workspace .attachments/{chatID}/ directory via ws.Copy,
// and replaces multimodal content parts with text references.
func prepareAttachments(ctx context.Context, messages []agentContext.Message, chatID string, ws workspace.FS) ([]agentContext.Message, error) {
	usedNames := make(map[string]int)
	attachDir := ".attachments/" + chatID

	result := make([]agentContext.Message, len(messages))
	copy(result, messages)

	for i, msg := range result {
		if msg.Role != "user" {
			continue
		}

		parts, ok := msg.Content.([]interface{})
		if !ok {
			if typedParts, ok := msg.Content.([]agentContext.ContentPart); ok {
				iparts := make([]interface{}, len(typedParts))
				for j, p := range typedParts {
					m := map[string]interface{}{"type": string(p.Type)}
					if p.Text != "" {
						m["text"] = p.Text
					}
					if p.ImageURL != nil {
						m["image_url"] = map[string]interface{}{
							"url":    p.ImageURL.URL,
							"detail": string(p.ImageURL.Detail),
						}
					}
					if p.File != nil {
						m["file"] = map[string]interface{}{
							"url":      p.File.URL,
							"filename": p.File.Filename,
						}
					}
					iparts[j] = m
				}
				parts = iparts
			} else {
				continue
			}
		}

		if len(parts) == 0 {
			continue
		}

		var textParts []string

		for _, item := range parts {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			partType, _ := m["type"].(string)

			switch partType {
			case "text":
				if text, ok := m["text"].(string); ok && text != "" {
					textParts = append(textParts, text)
				}

			case "image_url":
				imgData, _ := m["image_url"].(map[string]interface{})
				if imgData == nil {
					continue
				}
				url, _ := imgData["url"].(string)
				if url == "" {
					continue
				}
				uploaderName, fileID, isWrapper := attachment.Parse(url)
				if !isWrapper {
					textParts = append(textParts, fmt.Sprintf("[Image: %s]", url))
					continue
				}
				ref, err := resolveAttachment(ctx, uploaderName, fileID, "", attachDir, usedNames, ws)
				if err != nil {
					textParts = append(textParts, "[Attached image: failed to load]")
					continue
				}
				textParts = append(textParts, ref)

			case "file":
				fileData, _ := m["file"].(map[string]interface{})
				if fileData == nil {
					continue
				}
				url, _ := fileData["url"].(string)
				hintName, _ := fileData["filename"].(string)
				if url == "" {
					continue
				}
				uploaderName, fileID, isWrapper := attachment.Parse(url)
				if !isWrapper {
					textParts = append(textParts, fmt.Sprintf("[File: %s]", url))
					continue
				}
				ref, err := resolveAttachment(ctx, uploaderName, fileID, hintName, attachDir, usedNames, ws)
				if err != nil {
					textParts = append(textParts, "[Attached file: failed to load]")
					continue
				}
				textParts = append(textParts, ref)
			}
		}

		if len(textParts) > 0 {
			newMsg := result[i]
			newMsg.Content = strings.Join(textParts, "\n\n")
			result[i] = newMsg
		}
	}

	return result, nil
}

// resolveAttachment gets the local path of an attachment and copies it into
// the workspace via ws.Copy("local:///abs/path", ".attachments/{chatID}/filename").
func resolveAttachment(
	ctx context.Context,
	uploaderName, fileID, hintName, attachDir string,
	usedNames map[string]int,
	ws workspace.FS,
) (string, error) {
	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return "", fmt.Errorf("attachment manager not found: %s", uploaderName)
	}

	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	absPath, _, err := manager.LocalPath(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get local path: %w", err)
	}

	filename := fileInfo.Filename
	if filename == "" && hintName != "" {
		filename = hintName
	}
	if filename == "" {
		ext := extensionFromContentType(fileInfo.ContentType)
		filename = fileID + ext
	}

	baseName := filename
	if count, exists := usedNames[baseName]; exists {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		filename = fmt.Sprintf("%s_%d%s", name, count+1, ext)
		usedNames[baseName] = count + 1
	} else {
		usedNames[baseName] = 0
	}

	dstPath := attachDir + "/" + filename
	src := "local:///" + absPath

	if _, err := ws.Copy(src, dstPath); err != nil {
		return "", fmt.Errorf("failed to copy attachment to workspace: %w", err)
	}

	sizeStr := formatFileSize(fileInfo.Bytes)
	return fmt.Sprintf("[Attached file: %s (%s, %s)]", dstPath, fileInfo.ContentType, sizeStr), nil
}

func extensionFromContentType(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "text/css":
		return ".css"
	case "text/javascript", "application/javascript":
		return ".js"
	case "application/json":
		return ".json"
	case "application/zip":
		return ".zip"
	default:
		return ""
	}
}

func formatFileSize(bytes int) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
