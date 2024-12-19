package base

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/neo/assistant"
)

// AllowedFileTypes the allowed file types
var AllowedFileTypes = map[string]string{
	"application/json":   "json",
	"application/pdf":    "pdf",
	"application/msword": "doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "docx",
	"application/vnd.oasis.opendocument.text":                                   "odt",
	"application/vnd.ms-excel":                                                  "xls",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
	"application/vnd.ms-powerpoint":                                             "ppt",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
}

// MaxSize 20M max file size
var MaxSize int64 = 20 * 1024 * 1024

// Upload the file
func (ast *Base) Upload(ctx context.Context, file *multipart.FileHeader, reader io.Reader, option map[string]interface{}) (*assistant.File, error) {

	// check file size
	if file.Size > MaxSize {
		return nil, fmt.Errorf("file size %d exceeds the maximum size of %d", file.Size, MaxSize)
	}

	contentType := file.Header.Get("Content-Type")
	if !ast.allowed(contentType) {
		return nil, fmt.Errorf("file type %s not allowed", contentType)
	}

	data, err := fs.Get("data")
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(file.Filename)
	id, err := ast.id(file.Filename, ext)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%s%s", id, ext)
	_, err = data.Write(filename, reader, 0644)
	if err != nil {
		return nil, err
	}

	return &assistant.File{
		ID:          filename,
		Filename:    filename,
		ContentType: contentType,
		Bytes:       int(file.Size),
		CreatedAt:   int(time.Now().Unix()),
	}, nil
}

func (ast *Base) id(temp string, ext string) (string, error) {
	date := time.Now().Format("20060102")
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(temp)))[:8]
	return fmt.Sprintf("/__assistants/%s/%s/%s%s", ast.ID, date, hash, ext), nil
}

func (ast *Base) allowed(contentType string) bool {
	if _, ok := AllowedFileTypes[contentType]; ok {
		return true
	}
	// text/* // image/* // audio/* // video/*
	if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "audio/") || strings.HasPrefix(contentType, "video/") {
		return true
	}
	return false
}

// Download downloads a file
func (ast *Base) Download(ctx context.Context, fileID string) (*assistant.FileResponse, error) {

	// Get the data filesystem
	data, err := fs.Get("data")
	if err != nil {
		return nil, fmt.Errorf("get filesystem error: %s", err.Error())
	}

	// Check if file exists
	exists, err := data.Exists(fileID)
	if err != nil {
		return nil, fmt.Errorf("check file error: %s", err.Error())
	}
	if !exists {
		return nil, fmt.Errorf("file %s not found", fileID)
	}

	// Open the file
	reader, err := data.ReadCloser(fileID)
	if err != nil {
		return nil, err
	}

	// Get content type and extension
	ext := filepath.Ext(fileID)

	// Get content type from mime type
	contentType := "application/octet-stream"
	if v, err := data.MimeType(fileID); err == nil {
		contentType = v
	}

	for mimeType, extension := range AllowedFileTypes {
		if "."+extension == ext {
			contentType = mimeType
			break
		}
	}

	return &assistant.FileResponse{
		Reader:      reader,
		ContentType: contentType,
		Extension:   ext,
	}, nil
}
