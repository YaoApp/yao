package attachment

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestManagerUpload(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

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

		option := UploadOption{Groups: []string{"user123"}}

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
			Groups: []string{"user123"},
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

			option := UploadOption{Groups: []string{"user123"}}
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

func TestManagerMultiLevelGroups(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a local storage manager
	manager, err := New(ManagerOption{
		Driver:       "local",
		MaxSize:      "10M",
		AllowedTypes: []string{"text/*", "image/*"},
		Options: map[string]interface{}{
			"path": "/tmp/test_attachments",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test multi-level groups
	t.Run("MultiLevelGroups", func(t *testing.T) {
		content := "Test content for multi-level groups"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "multilevel.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		// Test with multi-level groups
		option := UploadOption{
			Groups: []string{"users", "user123", "chats", "chat456", "documents"},
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file with multi-level groups: %v", err)
		}

		// File ID should be 32 character hex (MD5 hash)
		if len(file.ID) != 32 {
			t.Errorf("File ID should be 32 characters: %s (length %d)", file.ID, len(file.ID))
		}

		// Check that it's all lowercase hex
		for _, r := range file.ID {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Errorf("File ID contains non-hex character: %c", r)
			}
		}

		// Test download
		downloadedContent, err := manager.Read(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to read file with multi-level groups: %v", err)
		}

		if string(downloadedContent) != content {
			t.Errorf("Content mismatch for multi-level groups file")
		}
	})

	// Test single group (backward compatibility)
	t.Run("SingleGroup", func(t *testing.T) {
		content := "Test content for single group"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "single.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups: []string{"knowledge"},
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file with single group: %v", err)
		}

		// File ID should be 32 character hex (MD5 hash)
		if len(file.ID) != 32 {
			t.Errorf("File ID should be 32 characters: %s (length %d)", file.ID, len(file.ID))
		}

		// Check that it's all lowercase hex
		for _, r := range file.ID {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Errorf("File ID contains non-hex character: %c", r)
			}
		}
	})

	// Test empty groups (no grouping)
	t.Run("EmptyGroups", func(t *testing.T) {
		content := "Test content without groups"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "nogroup.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups: []string{}, // Empty groups
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file without groups: %v", err)
		}

		// Should still work and create valid file ID
		if file.ID == "" {
			t.Error("File ID should not be empty")
		}
	})
}

func TestManagerValidation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

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

func TestManagerName(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a manager with a specific name
	managerName := "test-manager"
	manager, err := RegisterDefault(managerName)
	if err != nil {
		t.Fatalf("Failed to register manager: %v", err)
	}

	// Verify the manager name is set correctly
	if manager.Name != managerName {
		t.Errorf("Expected manager name '%s', got '%s'", managerName, manager.Name)
	}

	// Upload a file to verify the manager name is saved to database
	content := "Test file content"
	reader := strings.NewReader(content)

	fileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "test-manager-name.txt",
			Size:     int64(len(content)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	option := UploadOption{
		Groups: []string{"test"},
	}

	file, err := manager.Upload(context.Background(), fileHeader, reader, option)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	// Query database directly to verify manager name is stored
	m := model.Select("__yao.attachment")
	records, err := m.Get(model.QueryParam{
		Select: []interface{}{"uploader"},
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: file.ID},
		},
	})

	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if len(records) == 0 {
		t.Fatal("No record found in database")
	}

	storedManagerName, ok := records[0]["uploader"].(string)
	if !ok {
		t.Fatal("Uploader field is not a string")
	}

	if storedManagerName != managerName {
		t.Errorf("Expected stored uploader name '%s', got '%s'", managerName, storedManagerName)
	}
}

func TestUniqueFilenameGeneration(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager, err := RegisterDefault("test")
	if err != nil {
		t.Fatalf("Failed to register manager: %v", err)
	}

	// Upload two files with the same filename
	content1 := "First file content"
	content2 := "Second file content"

	// First file
	reader1 := strings.NewReader(content1)
	fileHeader1 := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "duplicate.txt", // Same filename
			Size:     int64(len(content1)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader1.Header.Set("Content-Type", "text/plain")

	// Second file
	reader2 := strings.NewReader(content2)
	fileHeader2 := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "duplicate.txt", // Same filename
			Size:     int64(len(content2)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader2.Header.Set("Content-Type", "text/plain")

	option := UploadOption{
		Groups: []string{"test"},
	}

	// Upload first file
	file1, err := manager.Upload(context.Background(), fileHeader1, reader1, option)
	if err != nil {
		t.Fatalf("Failed to upload first file: %v", err)
	}

	// Sleep a bit to ensure different timestamps
	time.Sleep(time.Millisecond)

	// Upload second file
	file2, err := manager.Upload(context.Background(), fileHeader2, reader2, option)
	if err != nil {
		t.Fatalf("Failed to upload second file: %v", err)
	}

	// Verify files have different IDs
	if file1.ID == file2.ID {
		t.Error("Files with same original name should have different IDs")
	}

	// Verify files have different storage paths
	if file1.Path == file2.Path {
		t.Error("Files with same original name should have different storage paths")
	}

	// Verify both files can be read independently
	data1, err := manager.Read(context.Background(), file1.ID)
	if err != nil {
		t.Fatalf("Failed to read first file: %v", err)
	}

	data2, err := manager.Read(context.Background(), file2.ID)
	if err != nil {
		t.Fatalf("Failed to read second file: %v", err)
	}

	if string(data1) != content1 {
		t.Errorf("First file content mismatch. Expected: %s, Got: %s", content1, string(data1))
	}

	if string(data2) != content2 {
		t.Errorf("Second file content mismatch. Expected: %s, Got: %s", content2, string(data2))
	}

	t.Logf("File 1 - ID: %s, Path: %s", file1.ID, file1.Path)
	t.Logf("File 2 - ID: %s, Path: %s", file2.ID, file2.Path)
}

func TestInfo(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager, err := RegisterDefault("test")
	if err != nil {
		t.Fatalf("Failed to register manager: %v", err)
	}

	// Upload a test file
	content := "Test file for info retrieval"
	reader := strings.NewReader(content)

	fileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "info-test.txt",
			Size:     int64(len(content)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	option := UploadOption{
		Groups:           []string{"info", "test"},
		OriginalFilename: "original-info-test.txt",
		ClientID:         "test-client-123",
		OpenID:           "test-openid-456",
		Gzip:             false,
	}

	uploadedFile, err := manager.Upload(context.Background(), fileHeader, reader, option)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	// Test the Info method
	fileInfo, err := manager.Info(context.Background(), uploadedFile.ID)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	// Verify file information
	if fileInfo.ID != uploadedFile.ID {
		t.Errorf("Expected file ID %s, got %s", uploadedFile.ID, fileInfo.ID)
	}

	if fileInfo.Filename != uploadedFile.Filename {
		t.Errorf("Expected filename %s, got %s", uploadedFile.Filename, fileInfo.Filename)
	}

	if fileInfo.ContentType != "text/plain" {
		t.Errorf("Expected content type 'text/plain', got %s", fileInfo.ContentType)
	}

	if fileInfo.Status != "uploaded" {
		t.Errorf("Expected status 'uploaded', got %s", fileInfo.Status)
	}

	if fileInfo.UserPath != option.OriginalFilename {
		t.Errorf("Expected user path %s, got %s", option.OriginalFilename, fileInfo.UserPath)
	}

	if fileInfo.Path != uploadedFile.Path {
		t.Errorf("Expected path %s, got %s", uploadedFile.Path, fileInfo.Path)
	}

	// Test with non-existent file ID
	_, err = manager.Info(context.Background(), "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent file ID, got nil")
	}

	t.Logf("Retrieved file info - ID: %s, Path: %s, UserPath: %s",
		fileInfo.ID, fileInfo.Path, fileInfo.UserPath)
}

func TestList(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Use unique manager name for test isolation
	managerName := fmt.Sprintf("test-list-%d", time.Now().UnixNano())
	manager, err := RegisterDefault(managerName)
	if err != nil {
		t.Fatalf("Failed to register manager: %v", err)
	}

	// Clean up existing records first
	m := model.Select("__yao.attachment")
	_, err = m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "uploader", Value: managerName},
		},
	})
	if err != nil {
		t.Logf("Warning: Failed to clean up existing records: %v", err)
	}

	// Upload multiple test files
	testFiles := []struct {
		filename    string
		content     string
		contentType string
		groups      []string
	}{
		{"test1.txt", "Content of test file 1", "text/plain", []string{"group1"}},
		{"test2.txt", "Content of test file 2", "text/plain", []string{"group1"}},
		{"image1.jpg", "Image content 1", "image/jpeg", []string{"group2", "images"}},
		{"doc1.pdf", "PDF content", "application/pdf", []string{"group2", "docs"}},
		{"test3.txt", "Content of test file 3", "text/plain", []string{"group1"}},
	}

	uploadedFiles := make([]*File, 0, len(testFiles))
	for _, tf := range testFiles {
		reader := strings.NewReader(tf.content)
		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: tf.filename,
				Size:     int64(len(tf.content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", tf.contentType)

		option := UploadOption{
			Groups: tf.groups,
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file %s: %v", tf.filename, err)
		}
		uploadedFiles = append(uploadedFiles, file)
	}

	// Test basic listing (no filters, default pagination)
	t.Run("BasicList", func(t *testing.T) {
		result, err := manager.List(context.Background(), ListOption{
			Filters: map[string]interface{}{
				"uploader": managerName,
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}

		if len(result.Files) != len(testFiles) {
			t.Errorf("Expected %d files, got %d", len(testFiles), len(result.Files))
		}

		if result.Total != int64(len(testFiles)) {
			t.Errorf("Expected total %d, got %d", len(testFiles), result.Total)
		}

		if result.Page != 1 {
			t.Errorf("Expected page 1, got %d", result.Page)
		}

		if result.PageSize != 20 {
			t.Errorf("Expected page size 20, got %d", result.PageSize)
		}
	})

	// Test pagination
	t.Run("Pagination", func(t *testing.T) {
		result, err := manager.List(context.Background(), ListOption{
			Page:     1,
			PageSize: 2,
			Filters: map[string]interface{}{
				"uploader": managerName,
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files with pagination: %v", err)
		}

		if len(result.Files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(result.Files))
		}

		if result.Total != int64(len(testFiles)) {
			t.Errorf("Expected total %d, got %d", len(testFiles), result.Total)
		}

		if result.Page != 1 {
			t.Errorf("Expected page 1, got %d", result.Page)
		}

		if result.PageSize != 2 {
			t.Errorf("Expected page size 2, got %d", result.PageSize)
		}

		if result.TotalPages != 3 { // 5 files / 2 per page = 3 pages
			t.Errorf("Expected 3 total pages, got %d", result.TotalPages)
		}
	})

	// Test filtering by content type
	t.Run("FilterByContentType", func(t *testing.T) {
		result, err := manager.List(context.Background(), ListOption{
			Filters: map[string]interface{}{
				"uploader":     managerName,
				"content_type": "text/plain",
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files with content type filter: %v", err)
		}

		expectedCount := 3 // test1.txt, test2.txt, test3.txt
		if len(result.Files) != expectedCount {
			t.Errorf("Expected %d text files, got %d", expectedCount, len(result.Files))
		}

		// Verify all returned files are text/plain
		for _, file := range result.Files {
			if file.ContentType != "text/plain" {
				t.Errorf("Expected content type 'text/plain', got '%s'", file.ContentType)
			}
		}
	})

	// Test wildcard filtering
	t.Run("WildcardFilter", func(t *testing.T) {
		result, err := manager.List(context.Background(), ListOption{
			Filters: map[string]interface{}{
				"uploader":     managerName,
				"content_type": "image/*",
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files with wildcard filter: %v", err)
		}

		expectedCount := 1 // image1.jpg
		if len(result.Files) != expectedCount {
			t.Errorf("Expected %d image files, got %d", expectedCount, len(result.Files))
		}
	})

	// Test ordering
	t.Run("OrderBy", func(t *testing.T) {
		result, err := manager.List(context.Background(), ListOption{
			OrderBy: "name asc",
			Filters: map[string]interface{}{
				"uploader": managerName,
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files with ordering: %v", err)
		}

		if len(result.Files) != len(testFiles) {
			t.Errorf("Expected %d files, got %d", len(testFiles), len(result.Files))
		}

		// Files should be ordered by name ascending
		// Note: The actual filenames are generated, so we just check that they're sorted
		for i := 1; i < len(result.Files); i++ {
			if result.Files[i-1].Filename > result.Files[i].Filename {
				t.Errorf("Files are not sorted by name ascending")
				break
			}
		}
	})

	// Test field selection
	t.Run("SelectFields", func(t *testing.T) {
		result, err := manager.List(context.Background(), ListOption{
			Select: []string{"file_id", "name", "content_type"},
			Filters: map[string]interface{}{
				"uploader": managerName,
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files with field selection: %v", err)
		}

		if len(result.Files) != len(testFiles) {
			t.Errorf("Expected %d files, got %d", len(testFiles), len(result.Files))
		}

		// Verify selected fields are populated
		for _, file := range result.Files {
			if file.ID == "" {
				t.Error("Expected file_id to be populated")
			}
			if file.Filename == "" {
				t.Error("Expected filename to be populated")
			}
			if file.ContentType == "" {
				t.Error("Expected content_type to be populated")
			}
		}
	})

	t.Logf("Successfully tested list functionality with %d files", len(uploadedFiles))
}

func TestManagerLocalPath(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Test with local storage
	t.Run("LocalStorage", func(t *testing.T) {
		// Create a local storage manager
		manager, err := New(ManagerOption{
			Driver:       "local",
			MaxSize:      "10M",
			AllowedTypes: []string{"text/*", "image/*", "application/*", ".txt", ".json", ".html", ".csv", ".yao"},
			Options: map[string]interface{}{
				"path": "/tmp/test_localpath_attachments",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create local manager: %v", err)
		}
		manager.Name = "localpath-test"

		// Test different file types
		testFiles := []struct {
			filename    string
			content     string
			contentType string
			expectedCT  string
		}{
			{"test.txt", "Hello LocalPath", "text/plain", "text/plain"},
			{"test.json", `{"localpath": "test"}`, "application/json", "application/json"},
			{"test.html", "<html><body>LocalPath Test</body></html>", "text/html", "text/html"},
			{"test.csv", "col1,col2\nlocalpath,test", "text/csv", "text/csv"},
			{"test.yao", "localpath yao content", "application/yao", "application/yao"},
		}

		for _, tf := range testFiles {
			// Upload file
			reader := strings.NewReader(tf.content)
			fileHeader := &FileHeader{
				FileHeader: &multipart.FileHeader{
					Filename: tf.filename,
					Size:     int64(len(tf.content)),
					Header:   make(map[string][]string),
				},
			}
			fileHeader.Header.Set("Content-Type", tf.contentType)

			option := UploadOption{
				Groups:           []string{"localpath", "test"},
				OriginalFilename: tf.filename,
			}

			file, err := manager.Upload(context.Background(), fileHeader, reader, option)
			if err != nil {
				t.Fatalf("Failed to upload file %s: %v", tf.filename, err)
			}

			// Test LocalPath
			localPath, detectedCT, err := manager.LocalPath(context.Background(), file.ID)
			if err != nil {
				t.Fatalf("Failed to get local path for %s: %v", tf.filename, err)
			}

			// Verify path is absolute
			if !filepath.IsAbs(localPath) {
				t.Errorf("Expected absolute path for %s, got: %s", tf.filename, localPath)
			}

			// Verify content type
			if detectedCT != tf.expectedCT {
				t.Errorf("Expected content type %s for %s, got: %s", tf.expectedCT, tf.filename, detectedCT)
			}

			// Verify file exists
			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				t.Errorf("File should exist at local path %s for %s", localPath, tf.filename)
			}

			// Verify file content
			fileContent, err := os.ReadFile(localPath)
			if err != nil {
				t.Fatalf("Failed to read file at local path for %s: %v", tf.filename, err)
			}

			if string(fileContent) != tf.content {
				t.Errorf("File content mismatch for %s. Expected: %s, Got: %s", tf.filename, tf.content, string(fileContent))
			}

			t.Logf("File %s - ID: %s, LocalPath: %s, ContentType: %s", tf.filename, file.ID, localPath, detectedCT)
		}
	})

	// Test with gzipped files in local storage
	t.Run("LocalStorage_Gzipped", func(t *testing.T) {
		manager, err := New(ManagerOption{
			Driver:       "local",
			MaxSize:      "10M",
			AllowedTypes: []string{"text/*"},
			Options: map[string]interface{}{
				"path": "/tmp/test_localpath_gzip_attachments",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create local manager: %v", err)
		}
		manager.Name = "localpath-gzip-test"

		content := "This content will be gzipped"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "gzipped.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups:           []string{"gzip", "test"},
			OriginalFilename: "gzipped.txt",
			Gzip:             true, // Enable gzip compression
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload gzipped file: %v", err)
		}

		// Test LocalPath - should get decompressed content
		localPath, contentType, err := manager.LocalPath(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get local path for gzipped file: %v", err)
		}

		// Verify content type
		if contentType != "text/plain" {
			t.Errorf("Expected content type text/plain, got: %s", contentType)
		}

		// For gzipped files in local storage, the storage path ends with .gz
		// but the content should be accessible normally through Read methods
		fileContent, err := manager.Read(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to read gzipped file: %v", err)
		}

		if string(fileContent) != content {
			t.Errorf("Gzipped file content mismatch. Expected: %s, Got: %s", content, string(fileContent))
		}

		t.Logf("Gzipped file - ID: %s, LocalPath: %s, ContentType: %s", file.ID, localPath, contentType)
	})
}

func TestManagerLocalPath_NonExistentFile(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager, err := New(ManagerOption{
		Driver:       "local",
		AllowedTypes: []string{"text/*"},
		Options: map[string]interface{}{
			"path": "/tmp/test_localpath_nonexistent",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	manager.Name = "nonexistent-test"

	// Test with non-existent file ID
	_, _, err = manager.LocalPath(context.Background(), "non-existent-file-id")
	if err == nil {
		t.Error("Expected error for non-existent file ID")
	}

	// Should contain "file not found" in the error chain
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Expected 'file not found' in error message, got: %s", err.Error())
	}
}

func TestManagerLocalPath_ValidationFlow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager, err := New(ManagerOption{
		Driver:       "local",
		AllowedTypes: []string{"text/*"},
		Options: map[string]interface{}{
			"path": "/tmp/test_localpath_validation",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	manager.Name = "validation-test"

	// Upload a file
	content := "Validation flow test content"
	reader := strings.NewReader(content)

	fileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "validation.txt",
			Size:     int64(len(content)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	option := UploadOption{
		Groups:           []string{"validation"},
		OriginalFilename: "original-validation.txt",
	}

	file, err := manager.Upload(context.Background(), fileHeader, reader, option)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	// Test complete flow: Upload -> LocalPath -> Verify -> Delete
	t.Run("CompleteFlow", func(t *testing.T) {
		// Get local path
		localPath, contentType, err := manager.LocalPath(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get local path: %v", err)
		}

		// Verify all properties
		if !filepath.IsAbs(localPath) {
			t.Error("Path should be absolute")
		}

		if contentType != "text/plain" {
			t.Errorf("Expected content type text/plain, got: %s", contentType)
		}

		// Verify file exists
		stat, err := os.Stat(localPath)
		if err != nil {
			t.Fatalf("File should exist at local path: %v", err)
		}

		if stat.Size() != int64(len(content)) {
			t.Errorf("File size mismatch. Expected: %d, Got: %d", len(content), stat.Size())
		}

		// Verify file content matches
		fileContent, err := os.ReadFile(localPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(fileContent) != content {
			t.Errorf("Content mismatch. Expected: %s, Got: %s", content, string(fileContent))
		}

		// Verify through manager's Read method as well
		managerContent, err := manager.Read(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to read through manager: %v", err)
		}

		if string(managerContent) != content {
			t.Errorf("Manager read content mismatch. Expected: %s, Got: %s", content, string(managerContent))
		}

		t.Logf("Validation complete - LocalPath: %s, Size: %d bytes, ContentType: %s", localPath, stat.Size(), contentType)
	})

	// Clean up
	err = manager.Delete(context.Background(), file.ID)
	if err != nil {
		t.Logf("Warning: Failed to delete test file: %v", err)
	}
}
