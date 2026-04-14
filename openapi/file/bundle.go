package file

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	ws "github.com/yaoapp/yao/workspace"
)

const maxBundleFiles = 50

type bundleRequest struct {
	Files       []bundleFileItem `json:"files" binding:"required,min=1"`
	ArchiveName string           `json:"archive_name"`
}

type bundleFileItem struct {
	File     string `json:"file" binding:"required"`
	Filename string `json:"filename" binding:"required"`
}

// bundle streams a ZIP archive containing the requested files.
//
// Supported URI schemes in each item's `file` field:
//   - {uploaderID}://{fileID}   — reads from attachment manager
//   - workspace://{wsId}/{path} — reads from workspace file system
//   - http(s)://...             — fetches external URL
func bundle(c *gin.Context) {
	var req bundleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Files) > maxBundleFiles {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Too many files (%d), max %d", len(req.Files), maxBundleFiles),
		})
		return
	}

	archiveName := req.ArchiveName
	if archiveName == "" {
		archiveName = "attachments.zip"
	}

	authInfo := authorized.GetInfo(c)

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		archiveName, url.PathEscape(archiveName)))
	c.Status(http.StatusOK)

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	seen := map[string]int{}

	for _, item := range req.Files {
		name := dedup(item.Filename, seen)
		data, err := resolveFile(c, authInfo, item.File)
		if err != nil {
			continue
		}
		w, err := zw.Create(name)
		if err != nil {
			if rc, ok := data.(io.ReadCloser); ok {
				rc.Close()
			}
			continue
		}
		io.Copy(w, data)
		if rc, ok := data.(io.ReadCloser); ok {
			rc.Close()
		}
	}
}

func resolveFile(c *gin.Context, authInfo *types.AuthorizedInfo, fileURI string) (io.Reader, error) {
	if strings.HasPrefix(fileURI, "workspace://") {
		return resolveWorkspaceFile(c, fileURI)
	}
	if strings.HasPrefix(fileURI, "http://") || strings.HasPrefix(fileURI, "https://") {
		return resolveHTTPFile(fileURI)
	}
	return resolveWrapperFile(c, authInfo, fileURI)
}

func resolveWrapperFile(c *gin.Context, authInfo *types.AuthorizedInfo, fileURI string) (io.Reader, error) {
	parts := strings.SplitN(fileURI, "://", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid wrapper URI: %s", fileURI)
	}

	uploaderID := parts[0]
	fileID := parts[1]

	manager, ok := attachment.Managers[uploaderID]
	if !ok {
		return nil, fmt.Errorf("uploader not found: %s", uploaderID)
	}

	fileInfo, err := manager.Info(c.Request.Context(), fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if authInfo != nil {
		allowed, err := checkFilePermission(authInfo, fileInfo, true)
		if err != nil || !allowed {
			return nil, fmt.Errorf("permission denied for file %s", fileID)
		}
	}

	data, err := manager.Read(c.Request.Context(), fileID)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	return bytes.NewReader(data), nil
}

func resolveWorkspaceFile(c *gin.Context, fileURI string) (io.Reader, error) {
	rest := strings.TrimPrefix(fileURI, "workspace://")
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return nil, fmt.Errorf("invalid workspace URI: %s", fileURI)
	}
	wsID := rest[:idx]
	filePath := rest[idx+1:]

	m := ws.M()
	if m == nil {
		return nil, fmt.Errorf("workspace service not available")
	}

	data, err := m.ReadFile(context.Background(), wsID, filePath)
	if err != nil {
		return nil, fmt.Errorf("workspace file read failed: %w", err)
	}

	return bytes.NewReader(data), nil
}

func resolveHTTPFile(fileURL string) (io.Reader, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch returned %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func dedup(name string, seen map[string]int) string {
	lower := strings.ToLower(name)
	n, exists := seen[lower]
	if !exists {
		seen[lower] = 1
		return name
	}
	seen[lower] = n + 1
	ext := ""
	base := name
	if dot := strings.LastIndex(name, "."); dot > 0 {
		ext = name[dot:]
		base = name[:dot]
	}
	return fmt.Sprintf("%s (%d)%s", base, n, ext)
}
