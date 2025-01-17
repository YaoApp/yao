package assistant

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/rag/driver"
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

// Upload implements file upload functionality
func (ast *Assistant) Upload(ctx context.Context, file *multipart.FileHeader, reader io.Reader, option map[string]interface{}) (*File, error) {
	// check file size
	if file.Size > MaxSize {
		return nil, fmt.Errorf("file size %d exceeds the maximum size of %d", file.Size, MaxSize)
	}

	contentType := file.Header.Get("Content-Type")
	if !ast.allowed(contentType) {
		return nil, fmt.Errorf("file type %s not allowed", contentType)
	}

	// Get chat ID and session ID from options
	chatID := ""
	sid := ""
	if v, ok := option["chat_id"].(string); ok {
		chatID = v
	}
	if v, ok := option["sid"].(string); ok {
		sid = v
	}

	// Generate file ID with namespace
	fileID, err := ast.generateFileID(file.Filename, sid, chatID)
	if err != nil {
		return nil, err
	}

	// Upload file to storage
	data, err := fs.Get("data")
	if err != nil {
		return nil, err
	}

	_, err = data.Write(fileID, reader, 0644)
	if err != nil {
		return nil, err
	}

	// Create file response
	fileResp := &File{
		ID:          fileID,
		Filename:    fileID,
		ContentType: contentType,
		Bytes:       int(file.Size),
		CreatedAt:   int(time.Now().Unix()),
	}

	// Handle RAG if available
	if err := ast.handleRAG(ctx, fileResp, reader, option); err != nil {
		return nil, fmt.Errorf("RAG handling error: %s", err.Error())
	}

	// Handle Vision if available
	if err := ast.handleVision(ctx, fileResp, option); err != nil {
		return nil, fmt.Errorf("Vision handling error: %s", err.Error())
	}

	return fileResp, nil
}

// generateFileID generates a file ID with proper namespace
func (ast *Assistant) generateFileID(filename string, sid string, chatID string) (string, error) {
	ext := filepath.Ext(filename)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filename)))[:8]
	date := time.Now().Format("20060102")

	// Build namespace
	namespace := fmt.Sprintf("__assistants/%s", ast.ID)
	if sid != "" {
		namespace = fmt.Sprintf("%s/%s", namespace, sid)
		if chatID != "" {
			namespace = fmt.Sprintf("%s/%s", namespace, chatID)
		}
	}

	return fmt.Sprintf("%s/%s/%s%s", namespace, date, hash, ext), nil
}

// handleRAG handles the file with RAG if available
func (ast *Assistant) handleRAG(ctx context.Context, file *File, reader io.Reader, option map[string]interface{}) error {
	if rag == nil {
		return nil
	}

	// Check if RAG processing is enabled
	if option, ok := option["rag"].(bool); !ok || !option {
		return nil
	}

	// Only handle text-based files
	if !strings.HasPrefix(file.ContentType, "text/") {
		return nil
	}

	// Reset reader to beginning
	if seeker, ok := reader.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	// Extract sid and chat_id from file path
	parts := strings.Split(file.ID, "/")
	indexName := fmt.Sprintf("%s%s", rag.Setting.IndexPrefix, ast.ID) // Default: prefix-assistant

	if len(parts) >= 4 { // Has sid
		sid := parts[2]
		indexName = fmt.Sprintf("%s%s-%s", rag.Setting.IndexPrefix, ast.ID, sid) // prefix-assistant-user

		if len(parts) >= 5 { // Has chat_id
			chatID := parts[3]
			indexName = fmt.Sprintf("%s%s-%s-%s", rag.Setting.IndexPrefix, ast.ID, sid, chatID) // prefix-assistant-user-chat
		}
	}

	// Check if index exists
	exists, err := rag.Engine.HasIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("check index error: %s", err.Error())
	}

	// Create index if not exists
	if !exists {
		err = rag.Engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
		if err != nil {
			return fmt.Errorf("create index error: %s", err.Error())
		}
	}

	// Reset reader again after checking index
	if seeker, ok := reader.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	// Upload and index the file
	result, err := rag.Uploader.Upload(ctx, reader, driver.FileUploadOptions{
		Async:        false,
		ChunkSize:    1024, // Default chunk size
		ChunkOverlap: 256,  // Default overlap
		IndexName:    indexName,
	})

	if err != nil {
		return fmt.Errorf("upload error: %s", err.Error())
	}

	if len(result.Documents) == 0 {
		return fmt.Errorf("no documents indexed")
	}

	// Store the document IDs
	docIDs := make([]string, len(result.Documents))
	for i, doc := range result.Documents {
		docIDs[i] = doc.DocID
	}
	file.DocIDs = docIDs

	return nil
}

// handleVision handles the file with Vision if available
func (ast *Assistant) handleVision(ctx context.Context, file *File, option map[string]interface{}) error {

	if vision == nil {
		return nil
	}

	handleVision := false
	if vv, has := option["vision"]; has {
		switch v := vv.(type) {
		case bool:
			handleVision = v
		case string:
			handleVision = v == "true" || v == "1" || v == "yes" || v == "on" || v == "enable"
		}
	}

	if !handleVision {
		return nil
	}

	// Check if file is an image
	if !strings.HasPrefix(file.ContentType, "image/") {
		return nil
	}

	// Reset reader for vision service
	data, err := fs.Get("data")
	if err != nil {
		return fmt.Errorf("get filesystem error: %s", err.Error())
	}

	exists, err := data.Exists(file.ID)
	if err != nil {
		return fmt.Errorf("check file error: %s", err.Error())
	}
	if !exists {
		return fmt.Errorf("file %s not found", file.ID)
	}

	// Read file content into memory
	imgData, err := data.ReadFile(file.ID)
	if err != nil {
		return fmt.Errorf("read file error: %s", err.Error())
	}

	// The model is vision capable
	if ast.vision {
		// For vision-capable models, upload to vision service to get URL
		resp, err := vision.Upload(ctx, file.Filename, bytes.NewReader(imgData), file.ContentType)
		if err != nil {
			return fmt.Errorf("vision upload error: %s", err.Error())
		}
		file.URL = resp.URL // Store the URL for vision-capable models to use
		return nil
	}

	// For non-vision models, get image description
	prompt := "Describe this image in detail."
	if v, ok := option["vision_prompt"].(string); ok {
		prompt = v
	}

	// Upload to vision service first Compress image
	resp, err := vision.Upload(ctx, file.Filename, bytes.NewReader(imgData), file.ContentType)
	if err != nil {
		return fmt.Errorf("vision upload error: %s", err.Error())
	}

	// Analyze using base64 data
	result, err := vision.Analyze(ctx, resp.FileID, prompt)
	if err != nil {
		return fmt.Errorf("vision analyze error: %s", err.Error())
	}

	// Extract description text from response
	if desc, ok := result.Description["description"].(string); ok {
		file.Description = desc
	} else if desc, ok := result.Description["text"].(string); ok {
		file.Description = desc
	} else {
		// Convert the entire description to JSON string as fallback
		bytes, err := jsoniter.Marshal(result.Description)
		if err == nil {
			file.Description = string(bytes)
		}
	}

	return nil
}

// Download implements file download functionality
func (ast *Assistant) Download(ctx context.Context, fileID string) (*FileResponse, error) {
	data, err := fs.Get("data")
	if err != nil {
		return nil, fmt.Errorf("get filesystem error: %s", err.Error())
	}

	exists, err := data.Exists(fileID)
	if err != nil {
		return nil, fmt.Errorf("check file error: %s", err.Error())
	}
	if !exists {
		return nil, fmt.Errorf("file %s not found", fileID)
	}

	reader, err := data.ReadCloser(fileID)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(fileID)
	contentType := "application/octet-stream"
	if v, err := data.MimeType(fileID); err == nil {
		contentType = v
	}

	return &FileResponse{
		Reader:      reader,
		ContentType: contentType,
		Extension:   ext,
	}, nil
}

func (ast *Assistant) allowed(contentType string) bool {
	if _, ok := AllowedFileTypes[contentType]; ok {
		return true
	}
	if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "audio/") || strings.HasPrefix(contentType, "video/") {
		return true
	}
	return false
}
