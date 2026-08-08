package resolve

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/tools/webfetch"
	ws "github.com/yaoapp/yao/workspace"
)

var noop = func() {}

// ToLocalPath resolves various URI schemes to a local file path.
// Returns (localPath, cleanupFunc, error). Callers should defer cleanup().
func ToLocalPath(ctx context.Context, src string) (string, func(), error) {
	switch {
	case strings.HasPrefix(src, "workspace://"):
		wsID, relPath := ParseWorkspaceURI(src)
		if wsID == "" || relPath == "" {
			return "", noop, fmt.Errorf("invalid workspace URI: %s", src)
		}
		data, err := ws.M().ReadFile(ctx, wsID, relPath)
		if err != nil {
			return "", noop, err
		}
		tmpFile, err := writeTempFile(data, filepath.Ext(relPath))
		if err != nil {
			return "", noop, err
		}
		return tmpFile, func() { os.Remove(tmpFile) }, nil

	case strings.HasPrefix(src, "attach://"):
		uploaderID, fileID := parseAttachURI(src)
		if uploaderID == "" || fileID == "" {
			return "", noop, fmt.Errorf("invalid attach URI: %s", src)
		}
		mgr, ok := attachment.Managers[uploaderID]
		if !ok {
			return "", noop, fmt.Errorf("unknown attachment manager: %s", uploaderID)
		}
		path, _, err := mgr.LocalPath(ctx, fileID)
		if err != nil {
			return "", noop, err
		}
		return path, noop, nil

	case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"):
		data, err := webfetch.DownloadBytes(src, 100<<20)
		if err != nil {
			return "", noop, fmt.Errorf("download %s: %w", src, err)
		}
		ext := extFromURL(src)
		tmpFile, err := writeTempFile(data, ext)
		if err != nil {
			return "", noop, err
		}
		return tmpFile, func() { os.Remove(tmpFile) }, nil

	case strings.HasPrefix(src, "data:"):
		tmpFile, err := decodeDataURI(src)
		if err != nil {
			return "", noop, fmt.Errorf("decode data URI: %w", err)
		}
		return tmpFile, func() { os.Remove(tmpFile) }, nil

	default:
		if filepath.IsAbs(src) {
			if _, err := os.Stat(src); err == nil {
				return src, noop, nil
			}
		}
		return "", noop, fmt.Errorf("unsupported URI or file not found: %s", src)
	}
}

// ParseWorkspaceURI parses workspace://wsID/rel/path into workspace ID and relative path.
func ParseWorkspaceURI(uri string) (wsID string, relPath string) {
	rest := strings.TrimPrefix(uri, "workspace://")
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return "", ""
	}
	return rest[:idx], rest[idx+1:]
}

func parseAttachURI(uri string) (uploaderID, fileID string) {
	rest := strings.TrimPrefix(uri, "attach://")
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return "", ""
	}
	return rest[:idx], rest[idx+1:]
}

func decodeDataURI(uri string) (string, error) {
	parts := strings.SplitN(uri, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid data URI format")
	}

	header := parts[0]
	payload := parts[1]

	var data []byte
	var err error
	if strings.Contains(header, ";base64") {
		data, err = base64.StdEncoding.DecodeString(payload)
	} else {
		data = []byte(payload)
	}
	if err != nil {
		return "", err
	}

	ext := extFromMediaType(header)
	return writeTempFile(data, ext)
}

func writeTempFile(data []byte, ext string) (string, error) {
	if ext == "" {
		ext = ".bin"
	}
	tmpFile, err := os.CreateTemp("", "resolve-*"+ext)
	if err != nil {
		return "", err
	}
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}
	return tmpFile.Name(), nil
}

func extFromURL(rawURL string) string {
	path := rawURL
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return ".bin"
	}
	return ext
}

func extFromMediaType(header string) string {
	// data:[mediatype][;base64]
	meta := strings.TrimPrefix(header, "data:")
	if idx := strings.Index(meta, ";"); idx >= 0 {
		meta = meta[:idx]
	}
	switch meta {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/mpeg":
		return ".mp3"
	default:
		return ".bin"
	}
}
