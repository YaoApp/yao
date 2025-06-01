package attachment

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"strings"
	"testing"
)

func TestManagerUpload(t *testing.T) {
	// Create a local storage manager
	manager, err := New(ManagerOption{
		Driver:       "local",
		MaxSize:      "10M",
		ChunkSize:    "2M",
		AllowedTypes: []string{"text/*", "image/*", ".txt", ".jpg", ".png"},
		Options: map[string]interface{}{
			"path": "/tmp/test_attachments",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test simple text file upload
	t.Run("SimpleTextUpload", func(t *testing.T) {
		content := "Hello, World!"
		reader := strings.NewReader(content)

		// Create a mock file header
		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "test.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			UserID:      "user123",
			ChatID:      "chat456",
			AssistantID: "assistant789",
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file: %v", err)
		}

		if file.Filename != "test.txt" {
			t.Errorf("Expected filename 'test.txt', got '%s'", file.Filename)
		}

		if file.ContentType != "text/plain" {
			t.Errorf("Expected content type 'text/plain', got '%s'", file.ContentType)
		}

		// Test download
		response, err := manager.Download(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to download file: %v", err)
		}
		defer response.Reader.Close()

		downloadedContent, err := manager.Read(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(downloadedContent) != content {
			t.Errorf("Expected content '%s', got '%s'", content, string(downloadedContent))
		}

		// Test ReadBase64
		base64Content, err := manager.ReadBase64(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to read file as base64: %v", err)
		}

		expectedBase64 := base64.StdEncoding.EncodeToString([]byte(content))
		if base64Content != expectedBase64 {
			t.Errorf("Expected base64 '%s', got '%s'", expectedBase64, base64Content)
		}
	})

	// Test gzip compression
	t.Run("GzipUpload", func(t *testing.T) {
		content := "This is a test file that will be compressed with gzip."
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "test_gzip.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Gzip:   true,
			UserID: "user123",
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload gzipped file: %v", err)
		}

		// The stored file should be compressed, but when we read it back,
		// we should get the original content (if the storage handles decompression)
		downloadedContent, err := manager.Read(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to read gzipped file: %v", err)
		}

		if string(downloadedContent) != content {
			t.Errorf("Expected content '%s', got '%s'", content, string(downloadedContent))
		}
	})

	// Test chunked upload
	t.Run("ChunkedUpload", func(t *testing.T) {
		content := "This is a large file that will be uploaded in chunks. " +
			strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 100)

		chunkSize := 1024
		totalSize := len(content)

		var lastFile *File
		for start := 0; start < totalSize; start += chunkSize {
			end := start + chunkSize - 1
			if end >= totalSize {
				end = totalSize - 1
			}

			chunk := []byte(content[start : end+1])

			fileHeader := &FileHeader{
				FileHeader: &multipart.FileHeader{
					Filename: "large_file.txt",
					Size:     int64(len(chunk)),
					Header:   make(map[string][]string),
				},
			}
			fileHeader.Header.Set("Content-Type", "text/plain")
			fileHeader.Header.Set("Content-Range",
				fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
			fileHeader.Header.Set("Content-Uid", "unique-file-id-123")

			option := UploadOption{UserID: "user123"}
			file, err := manager.Upload(context.Background(), fileHeader, bytes.NewReader(chunk), option)
			if err != nil {
				t.Fatalf("Failed to upload chunk starting at %d: %v", start, err)
			}

			lastFile = file
		}

		// After uploading all chunks, read the complete file
		if lastFile != nil {
			downloadedContent, err := manager.Read(context.Background(), lastFile.ID)
			if err != nil {
				t.Fatalf("Failed to read chunked file: %v", err)
			}

			if string(downloadedContent) != content {
				t.Errorf("Chunked upload content mismatch. Expected length %d, got %d",
					len(content), len(downloadedContent))
			}
		}
	})
}

func TestManagerValidation(t *testing.T) {
	manager, err := New(ManagerOption{
		Driver:       "local",
		MaxSize:      "1K", // Very small max size for testing
		AllowedTypes: []string{"text/plain"},
		Options: map[string]interface{}{
			"path": "/tmp/test_attachments",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test file size validation
	t.Run("FileSizeValidation", func(t *testing.T) {
		content := strings.Repeat("a", 2048) // 2KB, exceeds 1KB limit
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "large.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{}

		_, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err == nil {
			t.Error("Expected error for file size exceeding limit")
		}
	})

	// Test file type validation
	t.Run("FileTypeValidation", func(t *testing.T) {
		content := "test"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "test.jpg",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "image/jpeg") // Not allowed

		option := UploadOption{}

		_, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err == nil {
			t.Error("Expected error for disallowed file type")
		}
	})
}
