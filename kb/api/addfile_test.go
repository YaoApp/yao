package api_test

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/api"
)

// Note: TestMain is defined in collection_test.go, which handles environment setup
// Run tests with: source env.local.sh && go test -v ./kb/api/...

// createTestCollectionForFile is a helper to create a test collection for file tests
func createTestCollectionForFile(t *testing.T, ctx context.Context) string {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	collectionID := fmt.Sprintf("test_file_%d", time.Now().UnixNano())

	params := &api.CreateCollectionParams{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"name":        "Test File Collection",
			"description": "Collection for AddFile tests",
		},
		EmbeddingProviderID: "__yao.openai",
		EmbeddingOptionID:   "text-embedding-3-small",
		Locale:              "en",
		Config: &graphragtypes.CreateCollectionOptions{
			Distance:  "cosine",
			IndexType: "hnsw",
		},
	}

	_, err := kb.API.CreateCollection(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}

	return collectionID
}

// cleanupTestCollectionForFile removes a test collection
func cleanupTestCollectionForFile(ctx context.Context, collectionID string) {
	if kb.API != nil {
		_, _ = kb.API.RemoveCollection(ctx, collectionID)
	}
}

// ========== AddFile Tests ==========
// Note: Full AddFile tests require actual files to be uploaded via attachment manager
// These tests verify parameter validation and error handling

func TestAddFile(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForFile(t, ctx)
	defer cleanupTestCollectionForFile(ctx, collectionID)

	t.Run("AddFileMissingCollectionID", func(t *testing.T) {
		params := &api.AddFileParams{
			FileID: "some_file_id",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFile(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "collection_id is required")
	})

	t.Run("AddFileMissingFileID", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFile(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "file_id is required")
	})

	t.Run("AddFileMissingChunking", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "some_file_id",
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFile(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "chunking configuration is required")
	})

	t.Run("AddFileMissingEmbedding", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "some_file_id",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
		}

		result, err := kb.API.AddFile(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "embedding configuration is required")
	})

	t.Run("AddFileInvalidUploader", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "some_file_id",
			Uploader:     "invalid_uploader",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFile(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid uploader")
	})

	t.Run("AddFileNotFound", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "nonexistent_file_id",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFile(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		// Error could be "file not found" or "invalid uploader" depending on environment
		assert.True(t, err != nil, "Expected an error")
	})
}

// ========== AddFileAsync Tests ==========

func TestAddFileAsync(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForFile(t, ctx)
	defer cleanupTestCollectionForFile(ctx, collectionID)

	t.Run("AddFileAsyncMissingCollectionID", func(t *testing.T) {
		params := &api.AddFileParams{
			FileID: "some_file_id",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFileAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "collection_id is required")
	})

	t.Run("AddFileAsyncMissingFileID", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFileAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "file_id is required")
	})

	t.Run("AddFileAsyncMissingChunking", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "some_file_id",
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFileAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "chunking configuration is required")
	})

	t.Run("AddFileAsyncMissingEmbedding", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "some_file_id",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
		}

		result, err := kb.API.AddFileAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "embedding configuration is required")
	})

	t.Run("AddFileAsyncInvalidUploader", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "some_file_id",
			Uploader:     "invalid_uploader",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFileAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid uploader")
	})

	t.Run("AddFileAsyncNotFound", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       "nonexistent_file_id",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddFileAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		// Error could be "file not found" or "invalid uploader" depending on environment
		assert.True(t, err != nil, "Expected an error")
	})
}

// ========== AddFile with Real File Tests ==========

// getTestUploader returns the uploader name and manager for testing
func getTestUploader(t *testing.T) (string, *attachment.Manager) {
	// Try __yao.attachment first (system uploader)
	if manager, ok := attachment.Managers["__yao.attachment"]; ok {
		return "__yao.attachment", manager
	}
	// Try local manager
	if manager, ok := attachment.Managers["local"]; ok {
		return "local", manager
	}
	// List available managers for debugging
	var available []string
	for name := range attachment.Managers {
		available = append(available, name)
	}
	t.Fatalf("No attachment manager available. Available managers: %v", available)
	return "", nil
}

// uploadTestFile uploads a test file using the attachment manager and returns the file ID
func uploadTestFile(t *testing.T, ctx context.Context, filename, content string) string {
	_, manager := getTestUploader(t)

	// Create file header
	fileHeader := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: filename,
			Size:     int64(len(content)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	// Upload the file
	reader := strings.NewReader(content)
	file, err := manager.Upload(ctx, fileHeader, reader, attachment.UploadOption{})
	if err != nil {
		t.Fatalf("Failed to upload test file: %v", err)
	}

	t.Logf("Uploaded test file: %s (ID: %s)", filename, file.ID)
	return file.ID
}

// cleanupTestFile removes a test file
func cleanupTestFile(ctx context.Context, t *testing.T, fileID string) {
	_, manager := getTestUploader(t)
	_ = manager.Delete(ctx, fileID)
}

func TestAddFileWithRealFile(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForFile(t, ctx)
	defer cleanupTestCollectionForFile(ctx, collectionID)

	// Get the uploader name
	uploaderName, _ := getTestUploader(t)

	// Upload a test file
	testContent := `This is a test document for the knowledge base.
It contains content to test the file processing functionality.`

	fileID := uploadTestFile(t, ctx, "test_document.txt", testContent)
	defer cleanupTestFile(ctx, t, fileID)

	t.Run("AddFileSuccess", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       fileID,
			Uploader:     uploaderName,
			Locale:       "en",
			Metadata: map[string]interface{}{
				"description": "A test file document",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			AuthScope: map[string]interface{}{
				"__yao_created_by": "test_user",
			},
		}

		result, err := kb.API.AddFile(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, collectionID, result.CollectionID)
			assert.NotEmpty(t, result.DocID)
			assert.Equal(t, fileID, result.FileID)
			assert.Contains(t, result.Message, "successfully")
			t.Logf("Added file document: %s", result.DocID)
		}

		// Verify document was created
		if result != nil {
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.NotNil(t, doc)
			assert.Equal(t, "file", doc["type"])
			assert.Equal(t, "completed", doc["status"])
			t.Logf("✅ File Document verified: type=%v, status=%v", doc["type"], doc["status"])
		}
	})
}

func TestAddFileAsyncWithRealFile(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForFile(t, ctx)
	defer cleanupTestCollectionForFile(ctx, collectionID)

	// Get the uploader name
	uploaderName, _ := getTestUploader(t)

	// Upload a test file for async processing
	testContent := `Async test document content.

This document will be processed asynchronously.

The job system should handle the processing in the background.`

	fileID := uploadTestFile(t, ctx, "async_test_document.txt", testContent)
	defer cleanupTestFile(ctx, t, fileID)

	t.Run("AddFileAsyncSuccess", func(t *testing.T) {
		params := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       fileID,
			Uploader:     uploaderName,
			Locale:       "en",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			Job: &api.JobOptionsParams{
				Name:        "Test Async File Job",
				Description: "Testing async file processing",
				Category:    "Test",
			},
		}

		result, err := kb.API.AddFileAsync(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.NotEmpty(t, result.JobID)
			assert.NotEmpty(t, result.DocID)
			t.Logf("Created async file job: %s for document: %s", result.JobID, result.DocID)
		}

		// Verify document was created with pending status
		if result != nil {
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.NotNil(t, doc)
			assert.Equal(t, "file", doc["type"])
			assert.Equal(t, result.JobID, doc["job_id"])
			t.Logf("✅ Async file document created: status=%v, job_id=%v", doc["status"], doc["job_id"])

			// Wait for job to complete (max 30 seconds)
			maxWait := 30 * time.Second
			pollInterval := 500 * time.Millisecond
			startTime := time.Now()
			var finalStatus string

			for time.Since(startTime) < maxWait {
				doc, err = kb.API.GetDocument(ctx, result.DocID, nil)
				if err != nil {
					t.Logf("Error getting document: %v", err)
					break
				}
				finalStatus, _ = doc["status"].(string)
				if finalStatus == "completed" || finalStatus == "error" {
					break
				}
				time.Sleep(pollInterval)
			}

			t.Logf("✅ Job completed: final status=%s, elapsed=%v", finalStatus, time.Since(startTime))
			assert.Equal(t, "completed", finalStatus, "Job should complete successfully")
		}
	})
}

func TestAddFileIntegration(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForFile(t, ctx)
	defer cleanupTestCollectionForFile(ctx, collectionID)

	// Get the uploader name
	uploaderName, _ := getTestUploader(t)

	t.Run("FullFileLifecycle", func(t *testing.T) {
		// Upload a test file
		testContent := `Integration test document.

This document tests the full lifecycle of file processing:
1. Upload file
2. Add to knowledge base
3. Verify document creation
4. List documents
5. Remove document

End of test content.`

		fileID := uploadTestFile(t, ctx, "lifecycle_test.txt", testContent)
		defer cleanupTestFile(ctx, t, fileID)

		// 1. Add File Document
		addParams := &api.AddFileParams{
			CollectionID: collectionID,
			FileID:       fileID,
			Uploader:     uploaderName,
			Locale:       "en",
			Metadata: map[string]interface{}{
				"title":       "File Lifecycle Test",
				"description": "Full lifecycle integration test",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			AuthScope: map[string]interface{}{
				"__yao_created_by": "integration_test",
			},
		}

		result, err := kb.API.AddFile(ctx, addParams)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result == nil {
			t.Fatalf("Failed to create file document: result is nil")
		}
		t.Logf("1. Created file document: %s", result.DocID)

		// 2. Get Document
		doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "file", doc["type"])
		assert.Equal(t, "completed", doc["status"])
		t.Logf("2. Retrieved document: name=%v, type=%v, status=%v", doc["name"], doc["type"], doc["status"])

		// 3. List Documents
		listFilter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: collectionID,
		}
		listResult, err := kb.API.ListDocuments(ctx, listFilter)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(listResult.Data), 1)
		t.Logf("3. Found document in list: %d documents", len(listResult.Data))

		// 4. Remove Document
		removeParams := &api.RemoveDocumentsParams{
			DocumentIDs: []string{result.DocID},
		}
		removeResult, err := kb.API.RemoveDocuments(ctx, removeParams)
		assert.NoError(t, err)
		assert.NotNil(t, removeResult)
		t.Logf("4. Removed document: %d deleted", removeResult.DeletedCount)

		// 5. Verify Removal
		_, err = kb.API.GetDocument(ctx, result.DocID, nil)
		assert.Error(t, err)
		t.Logf("5. Verified document removal")

		t.Logf("✅ Full file lifecycle test completed successfully")
	})
}
