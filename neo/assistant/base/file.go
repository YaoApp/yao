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
	id, err := ast.id(file.Filename)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%s%s", id, ext)
	_, err = data.Write(filename, reader, 0644)
	if err != nil {
		return nil, err
	}

	return &assistant.File{
		ID:          strings.ReplaceAll(id, "/", "_"),
		Filename:    filename,
		ContentType: contentType,
		Bytes:       int(file.Size),
		CreatedAt:   int(time.Now().Unix()),
	}, nil
}

func (ast *Base) id(temp string) (string, error) {
	date := time.Now().Format("20060102")
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(temp)))[:8]
	return fmt.Sprintf("/__assistants/%s/%s/%s", ast.ID, date, hash), nil
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
