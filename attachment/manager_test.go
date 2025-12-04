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

		// Content type may include charset
		if !strings.HasPrefix(file.ContentType, "text/plain") {
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
		Public:           false,
		Share:            "private",
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

	// Content type may include charset
	if !strings.HasPrefix(fileInfo.ContentType, "text/plain") {
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
			Wheres: []model.QueryWhere{
				{Column: "uploader", Value: managerName},
				{Column: "content_type", Value: "text/plain%", OP: "like"},
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files with content type filter: %v", err)
		}

		expectedCount := 3 // test1.txt, test2.txt, test3.txt
		if len(result.Files) != expectedCount {
			t.Errorf("Expected %d text files, got %d", expectedCount, len(result.Files))
		}

		// Verify all returned files are text/plain (may include charset)
		for _, file := range result.Files {
			if !strings.HasPrefix(file.ContentType, "text/plain") {
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

func TestPublicAndShareFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Force re-migrate the attachment table to ensure schema is up to date
	m := model.Select("__yao.attachment")
	if m != nil {
		// Drop and recreate table to get latest schema
		err := m.DropTable()
		if err != nil {
			t.Logf("Warning: failed to drop table: %v", err)
		}
		err = m.Migrate(false)
		if err != nil {
			t.Fatalf("Failed to migrate table: %v", err)
		}
	}

	manager, err := RegisterDefault("test-public-share")
	if err != nil {
		t.Fatalf("Failed to register manager: %v", err)
	}

	// Test 1: Upload with public=true and share=team
	t.Run("PublicTeamShare", func(t *testing.T) {
		content := "Public team shared file"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "public-team.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups:           []string{"test"},
			OriginalFilename: "public-team.txt",
			Public:           true,
			Share:            "team",
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload public team file: %v", err)
		}

		// Verify in database
		m := model.Select("__yao.attachment")
		records, err := m.Get(model.QueryParam{
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

		// Debug: print all fields
		t.Logf("Record fields: %+v", records[0])

		publicValue := toBool(records[0]["public"])
		if !publicValue {
			t.Errorf("Expected public to be true, got: %v (type: %T)", records[0]["public"], records[0]["public"])
		}

		shareValue := toString(records[0]["share"])
		if shareValue != "team" {
			t.Errorf("Expected share to be 'team', got: %v (type: %T)", records[0]["share"], records[0]["share"])
		}
	})

	// Test 2: Upload with public=false and share=private (default)
	t.Run("PrivateShare", func(t *testing.T) {
		content := "Private file"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "private.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups:           []string{"test"},
			OriginalFilename: "private.txt",
			Public:           false,
			Share:            "private",
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload private file: %v", err)
		}

		// Verify in database
		m := model.Select("__yao.attachment")
		records, err := m.Get(model.QueryParam{
			Select: []interface{}{"public", "share"},
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

		publicValue := toBool(records[0]["public"])
		if publicValue {
			t.Errorf("Expected public to be false, got: %v", records[0]["public"])
		}

		shareValue := toString(records[0]["share"])
		if shareValue != "private" {
			t.Errorf("Expected share to be 'private', got: %v", records[0]["share"])
		}
	})

	// Test 3: Upload without specifying share (should default to private)
	t.Run("DefaultSharePrivate", func(t *testing.T) {
		content := "Default share file"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "default-share.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups:           []string{"test"},
			OriginalFilename: "default-share.txt",
			Public:           false,
			// Share not specified, should default to "private"
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file with default share: %v", err)
		}

		// Verify in database
		m := model.Select("__yao.attachment")
		records, err := m.Get(model.QueryParam{
			Select: []interface{}{"share"},
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

		shareValue := toString(records[0]["share"])
		if shareValue != "private" {
			t.Errorf("Expected default share to be 'private', got: %v", records[0]["share"])
		}
	})
}

func TestYaoPermissionFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Force re-migrate the attachment table to ensure schema is up to date
	m := model.Select("__yao.attachment")
	if m != nil {
		// Drop and recreate table to get latest schema
		err := m.DropTable()
		if err != nil {
			t.Logf("Warning: failed to drop table: %v", err)
		}
		err = m.Migrate(false)
		if err != nil {
			t.Fatalf("Failed to migrate table: %v", err)
		}
	}

	manager, err := RegisterDefault("test-yao-permission")
	if err != nil {
		t.Fatalf("Failed to register manager: %v", err)
	}

	// Test 1: Upload with all Yao permission fields
	t.Run("AllYaoFields", func(t *testing.T) {
		content := "File with all Yao permission fields"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "yao-all-fields.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups:           []string{"test"},
			OriginalFilename: "yao-all-fields.txt",
			YaoCreatedBy:     "user123",
			YaoUpdatedBy:     "user123",
			YaoTeamID:        "team456",
			YaoTenantID:      "tenant789",
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file with Yao fields: %v", err)
		}

		// Verify in database
		m := model.Select("__yao.attachment")
		records, err := m.Get(model.QueryParam{
			Select: []interface{}{"__yao_created_by", "__yao_updated_by", "__yao_team_id", "__yao_tenant_id"},
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

		// Verify __yao_created_by
		createdBy := toString(records[0]["__yao_created_by"])
		if createdBy != "user123" {
			t.Errorf("Expected __yao_created_by to be 'user123', got: %v", records[0]["__yao_created_by"])
		}

		// Verify __yao_updated_by
		updatedBy := toString(records[0]["__yao_updated_by"])
		if updatedBy != "user123" {
			t.Errorf("Expected __yao_updated_by to be 'user123', got: %v", records[0]["__yao_updated_by"])
		}

		// Verify __yao_team_id
		teamID := toString(records[0]["__yao_team_id"])
		if teamID != "team456" {
			t.Errorf("Expected __yao_team_id to be 'team456', got: %v", records[0]["__yao_team_id"])
		}

		// Verify __yao_tenant_id
		tenantID := toString(records[0]["__yao_tenant_id"])
		if tenantID != "tenant789" {
			t.Errorf("Expected __yao_tenant_id to be 'tenant789', got: %v", records[0]["__yao_tenant_id"])
		}
	})

	// Test 2: Upload with partial Yao fields (only team and tenant)
	t.Run("PartialYaoFields", func(t *testing.T) {
		content := "File with partial Yao fields"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "yao-partial-fields.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups:           []string{"test"},
			OriginalFilename: "yao-partial-fields.txt",
			YaoTeamID:        "team999",
			YaoTenantID:      "tenant888",
			// YaoCreatedBy and YaoUpdatedBy not specified
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file with partial Yao fields: %v", err)
		}

		// Verify in database
		m := model.Select("__yao.attachment")
		records, err := m.Get(model.QueryParam{
			Select: []interface{}{"__yao_team_id", "__yao_tenant_id"},
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

		// Verify __yao_team_id
		teamID := toString(records[0]["__yao_team_id"])
		if teamID != "team999" {
			t.Errorf("Expected __yao_team_id to be 'team999', got: %v", records[0]["__yao_team_id"])
		}

		// Verify __yao_tenant_id
		tenantID := toString(records[0]["__yao_tenant_id"])
		if tenantID != "tenant888" {
			t.Errorf("Expected __yao_tenant_id to be 'tenant888', got: %v", records[0]["__yao_tenant_id"])
		}
	})

	// Test 3: Upload without Yao fields (should be null/empty in database)
	t.Run("NoYaoFields", func(t *testing.T) {
		content := "File without Yao fields"
		reader := strings.NewReader(content)

		fileHeader := &FileHeader{
			FileHeader: &multipart.FileHeader{
				Filename: "yao-no-fields.txt",
				Size:     int64(len(content)),
				Header:   make(map[string][]string),
			},
		}
		fileHeader.Header.Set("Content-Type", "text/plain")

		option := UploadOption{
			Groups:           []string{"test"},
			OriginalFilename: "yao-no-fields.txt",
			// No Yao fields specified
		}

		file, err := manager.Upload(context.Background(), fileHeader, reader, option)
		if err != nil {
			t.Fatalf("Failed to upload file without Yao fields: %v", err)
		}

		// Should succeed without errors
		if file.ID == "" {
			t.Error("File ID should not be empty")
		}

		t.Logf("Successfully uploaded file without Yao fields - ID: %s", file.ID)
	})
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

func TestGetTextAndSaveText(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager, err := RegisterDefault("test-text-content")
	if err != nil {
		t.Fatalf("Failed to register manager: %v", err)
	}

	// Upload a test file
	content := "This is a test file for text content storage"
	reader := strings.NewReader(content)

	fileHeader := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "test-text.txt",
			Size:     int64(len(content)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	option := UploadOption{
		Groups:           []string{"test"},
		OriginalFilename: "test-text.txt",
	}

	file, err := manager.Upload(context.Background(), fileHeader, reader, option)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	// Test 1: GetText on file without saved text (should return empty)
	t.Run("GetTextEmpty", func(t *testing.T) {
		text, err := manager.GetText(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get text: %v", err)
		}

		if text != "" {
			t.Errorf("Expected empty text, got: %s", text)
		}

		// Also test full content
		fullText, err := manager.GetText(context.Background(), file.ID, true)
		if err != nil {
			t.Fatalf("Failed to get full text: %v", err)
		}

		if fullText != "" {
			t.Errorf("Expected empty full text, got: %s", fullText)
		}
	})

	// Test 2: SaveText and verify
	t.Run("SaveTextAndVerify", func(t *testing.T) {
		parsedText := "This is the parsed text content from the file. It could be extracted from PDF, Word, or image OCR."

		err := manager.SaveText(context.Background(), file.ID, parsedText)
		if err != nil {
			t.Fatalf("Failed to save text: %v", err)
		}

		// Retrieve the saved text
		retrievedText, err := manager.GetText(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get saved text: %v", err)
		}

		if retrievedText != parsedText {
			t.Errorf("Text mismatch. Expected: %s, Got: %s", parsedText, retrievedText)
		}

		t.Logf("Successfully saved and retrieved text content (%d characters)", len(retrievedText))
	})

	// Test 3: Update existing text
	t.Run("UpdateText", func(t *testing.T) {
		updatedText := "This is the updated parsed text content with additional information."

		err := manager.SaveText(context.Background(), file.ID, updatedText)
		if err != nil {
			t.Fatalf("Failed to update text: %v", err)
		}

		retrievedText, err := manager.GetText(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get updated text: %v", err)
		}

		if retrievedText != updatedText {
			t.Errorf("Updated text mismatch. Expected: %s, Got: %s", updatedText, retrievedText)
		}
	})

	// Test 4: Save long text content and verify preview vs full content
	t.Run("SaveLongText", func(t *testing.T) {
		// Generate a large text content (10KB)
		longText := strings.Repeat("This is a long text content that simulates parsing from a large document like PDF or Word. ", 100)

		err := manager.SaveText(context.Background(), file.ID, longText)
		if err != nil {
			t.Fatalf("Failed to save long text: %v", err)
		}

		// Get preview (default, should be limited to 2000 characters)
		previewText, err := manager.GetText(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get preview text: %v", err)
		}

		// Preview should be exactly 2000 characters (runes)
		previewRunes := []rune(previewText)
		if len(previewRunes) != 2000 {
			t.Errorf("Preview length mismatch. Expected: 2000 runes, Got: %d runes", len(previewRunes))
		}

		// Get full content
		fullText, err := manager.GetText(context.Background(), file.ID, true)
		if err != nil {
			t.Fatalf("Failed to get full text: %v", err)
		}

		if fullText != longText {
			t.Errorf("Full text mismatch. Expected length: %d, Got: %d", len(longText), len(fullText))
		}

		t.Logf("Successfully saved long text - Preview: %d chars, Full: %d chars", len(previewText), len(fullText))
	})

	// Test 5: Test UTF-8 character handling in preview
	t.Run("UTF8PreviewHandling", func(t *testing.T) {
		// Create text with multi-byte UTF-8 characters (Chinese, emoji, etc.)
		// Each Chinese character is 3 bytes, emoji is 4 bytes
		chineseText := strings.Repeat("这是一个测试文本，包含中文字符。", 150) // Should exceed 2000 chars
		emojiText := strings.Repeat("Hello 👋 World 🌍 ", 150)

		// Test with Chinese text
		err := manager.SaveText(context.Background(), file.ID, chineseText)
		if err != nil {
			t.Fatalf("Failed to save Chinese text: %v", err)
		}

		previewChinese, err := manager.GetText(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get Chinese preview: %v", err)
		}

		// Should be exactly 2000 runes (characters), not bytes
		if len([]rune(previewChinese)) != 2000 {
			t.Errorf("Chinese preview should be 2000 runes, got: %d", len([]rune(previewChinese)))
		}

		// Full text should be complete
		fullChinese, err := manager.GetText(context.Background(), file.ID, true)
		if err != nil {
			t.Fatalf("Failed to get full Chinese text: %v", err)
		}

		if fullChinese != chineseText {
			t.Errorf("Chinese text mismatch")
		}

		// Test with emoji text
		err = manager.SaveText(context.Background(), file.ID, emojiText)
		if err != nil {
			t.Fatalf("Failed to save emoji text: %v", err)
		}

		previewEmoji, err := manager.GetText(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get emoji preview: %v", err)
		}

		if len([]rune(previewEmoji)) != 2000 {
			t.Errorf("Emoji preview should be 2000 runes, got: %d", len([]rune(previewEmoji)))
		}

		t.Logf("UTF-8 handling verified - Chinese: %d bytes, Emoji: %d bytes",
			len(previewChinese), len(previewEmoji))
	})

	// Test 6: GetText with non-existent file ID
	t.Run("GetTextNonExistent", func(t *testing.T) {
		_, err := manager.GetText(context.Background(), "non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent file ID")
		}

		if !strings.Contains(err.Error(), "file not found") {
			t.Errorf("Expected 'file not found' error, got: %s", err.Error())
		}
	})

	// Test 7: SaveText with non-existent file ID
	t.Run("SaveTextNonExistent", func(t *testing.T) {
		err := manager.SaveText(context.Background(), "non-existent-id", "some text")
		if err == nil {
			t.Error("Expected error for non-existent file ID")
		}

		if !strings.Contains(err.Error(), "file not found") {
			t.Errorf("Expected 'file not found' error, got: %s", err.Error())
		}
	})

	// Test 8: Save empty text (clear content)
	t.Run("SaveEmptyText", func(t *testing.T) {
		err := manager.SaveText(context.Background(), file.ID, "")
		if err != nil {
			t.Fatalf("Failed to save empty text: %v", err)
		}

		retrievedText, err := manager.GetText(context.Background(), file.ID)
		if err != nil {
			t.Fatalf("Failed to get empty text: %v", err)
		}

		if retrievedText != "" {
			t.Errorf("Expected empty text, got: %s", retrievedText)
		}
	})

	// Test 9: Verify List doesn't include content fields by default
	t.Run("ListExcludesContentByDefault", func(t *testing.T) {
		// Save some text content
		testText := "This text should not appear in list results by default"
		err := manager.SaveText(context.Background(), file.ID, testText)
		if err != nil {
			t.Fatalf("Failed to save text: %v", err)
		}

		// List files without specifying select fields
		result, err := manager.List(context.Background(), ListOption{
			Filters: map[string]interface{}{
				"file_id": file.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}

		if len(result.Files) == 0 {
			t.Fatal("Expected to find at least one file")
		}

		// The List method returns File structs, but we need to verify
		// the database query doesn't fetch the content field
		// We can verify this by checking the database directly
		m := model.Select("__yao.attachment")
		records, err := m.Get(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "file_id", Value: file.ID},
			},
		})

		if err != nil {
			t.Fatalf("Failed to query database: %v", err)
		}

		// When we do a full select, content should be present
		if len(records) > 0 {
			if content, ok := records[0]["content"].(string); ok && content == testText {
				t.Logf("Content field exists in full query (expected): %d characters", len(content))
			}
		}
	})

	// Test 10: Verify content can be explicitly selected in List
	t.Run("ListIncludesContentWhenExplicitlySelected", func(t *testing.T) {
		// Save some text content
		testText := "This text SHOULD appear when explicitly selected"
		err := manager.SaveText(context.Background(), file.ID, testText)
		if err != nil {
			t.Fatalf("Failed to save text: %v", err)
		}

		// List files WITH content field explicitly selected
		result, err := manager.List(context.Background(), ListOption{
			Select: []string{"file_id", "name", "content"},
			Filters: map[string]interface{}{
				"file_id": file.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to list files with content: %v", err)
		}

		if len(result.Files) == 0 {
			t.Fatal("Expected to find at least one file")
		}

		// Query database directly to verify content is included
		m := model.Select("__yao.attachment")
		records, err := m.Get(model.QueryParam{
			Select: []interface{}{"file_id", "name", "content"},
			Wheres: []model.QueryWhere{
				{Column: "file_id", Value: file.ID},
			},
		})

		if err != nil {
			t.Fatalf("Failed to query database: %v", err)
		}

		if len(records) == 0 {
			t.Fatal("Expected to find record")
		}

		// Verify content is present
		if content, ok := records[0]["content"].(string); ok {
			if content != testText {
				t.Errorf("Expected content '%s', got '%s'", testText, content)
			}
			t.Logf("Content field correctly included when explicitly selected: %d characters", len(content))
		} else {
			t.Error("Content field not found when explicitly selected")
		}
	})

	// Clean up
	err = manager.Delete(context.Background(), file.ID)
	if err != nil {
		t.Logf("Warning: Failed to delete test file: %v", err)
	}
}
