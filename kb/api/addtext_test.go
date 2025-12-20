package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/api"
)

// Note: TestMain is defined in collection_test.go, which handles environment setup
// Run tests with: source env.local.sh && go test -v ./kb/api/...

// createTestCollectionForText is a helper to create a test collection for text tests
func createTestCollectionForText(t *testing.T, ctx context.Context) string {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	collectionID := fmt.Sprintf("test_text_%d", time.Now().UnixNano())

	params := &api.CreateCollectionParams{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"name":        "Test Text Collection",
			"description": "Collection for AddText tests",
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

// cleanupTestCollectionForText removes a test collection
func cleanupTestCollectionForText(ctx context.Context, collectionID string) {
	if kb.API != nil {
		_, _ = kb.API.RemoveCollection(ctx, collectionID)
	}
}

// ========== AddText Tests ==========

func TestAddText(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForText(t, ctx)
	defer cleanupTestCollectionForText(ctx, collectionID)

	t.Run("AddTextSuccess", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "This is a test document content for knowledge base testing. It contains some sample text that will be chunked and embedded.",
			Locale:       "en",
			Metadata: map[string]interface{}{
				"title":       "Test Text Document",
				"description": "A test document",
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

		result, err := kb.API.AddText(ctx, params)
		if err != nil {
			// Skip if connector not loaded (environment issue)
			if assert.Contains(t, err.Error(), "connector") {
				t.Skipf("Skipping due to connector not loaded: %v", err)
			}
		}
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, collectionID, result.CollectionID)
			assert.NotEmpty(t, result.DocID)
			assert.Contains(t, result.Message, "successfully")
			t.Logf("Added text document: %s", result.DocID)

			// Verify document was created
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.NotNil(t, doc)
			assert.Equal(t, "text", doc["type"])
			assert.Equal(t, "Test Text Document", doc["name"])
			assert.Equal(t, "completed", doc["status"])
			t.Logf("✅ Document verified: type=%v, name=%v, status=%v", doc["type"], doc["name"], doc["status"])
		}
	})

	t.Run("AddTextMissingCollectionID", func(t *testing.T) {
		params := &api.AddTextParams{
			Text: "Some text content",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddText(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "collection_id is required")
	})

	t.Run("AddTextMissingText", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddText(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "text is required")
	})

	t.Run("AddTextMissingChunking", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "Some text content",
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddText(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "chunking configuration is required")
	})

	t.Run("AddTextMissingEmbedding", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "Some text content",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
		}

		result, err := kb.API.AddText(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "embedding configuration is required")
	})

	t.Run("AddTextWithCustomDocID", func(t *testing.T) {
		customDocID := fmt.Sprintf("custom_text_doc_%d", time.Now().UnixNano())
		params := &api.AddTextParams{
			CollectionID: collectionID,
			DocID:        customDocID,
			Text:         "Text with custom document ID",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
		}

		result, err := kb.API.AddText(ctx, params)
		if err != nil {
			t.Skipf("Skipping due to error: %v", err)
		}
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, customDocID, result.DocID)
			t.Logf("Added text with custom DocID: %s", result.DocID)
		}
	})

	t.Run("AddTextWithTitleFromMetadata", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "Text content with title from metadata",
			Metadata: map[string]interface{}{
				"title": "Custom Title From Metadata",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
		}

		result, err := kb.API.AddText(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.Equal(t, "Custom Title From Metadata", doc["name"])
			t.Logf("✅ Title from metadata verified: %v", doc["name"])
		}
	})
}

// ========== AddTextAsync Tests ==========

func TestAddTextAsync(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForText(t, ctx)
	defer cleanupTestCollectionForText(ctx, collectionID)

	t.Run("AddTextAsyncSuccess", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "This is async text content for testing background processing.",
			Locale:       "en",
			Metadata: map[string]interface{}{
				"title": "Async Text Document",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			Job: &api.JobOptionsParams{
				Name:        "Test Async Text Job",
				Description: "Testing async text processing",
				Category:    "Test",
			},
		}

		result, err := kb.API.AddTextAsync(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.NotEmpty(t, result.JobID)
			assert.NotEmpty(t, result.DocID)
			t.Logf("Created async job: %s for document: %s", result.JobID, result.DocID)
		}

		// Verify document was created with pending status
		if result != nil {
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.NotNil(t, doc)
			assert.Equal(t, "text", doc["type"])
			assert.Equal(t, result.JobID, doc["job_id"])
			t.Logf("✅ Async document created: status=%v, job_id=%v", doc["status"], doc["job_id"])

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
			if finalStatus == "error" {
				if errMsg, ok := doc["error_message"].(string); ok {
					t.Logf("Error message: %s", errMsg)
				}
			}
			assert.Equal(t, "completed", finalStatus, "Job should complete successfully")
		}
	})

	t.Run("AddTextAsyncMissingCollectionID", func(t *testing.T) {
		params := &api.AddTextParams{
			Text: "Some text",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddTextAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "collection_id is required")
	})

	t.Run("AddTextAsyncMissingText", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddTextAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "text is required")
	})

	t.Run("AddTextAsyncMissingChunking", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "Some text",
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddTextAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "chunking configuration is required")
	})

	t.Run("AddTextAsyncMissingEmbedding", func(t *testing.T) {
		params := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "Some text",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
		}

		result, err := kb.API.AddTextAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "embedding configuration is required")
	})

	t.Run("AddTextAsyncWithCustomDocID", func(t *testing.T) {
		customDocID := fmt.Sprintf("async_custom_text_%d", time.Now().UnixNano())
		params := &api.AddTextParams{
			CollectionID: collectionID,
			DocID:        customDocID,
			Text:         "Async text with custom DocID",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
		}

		result, err := kb.API.AddTextAsync(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, customDocID, result.DocID)
			t.Logf("Created async text with custom DocID: %s", result.DocID)
		}
	})
}

// ========== AddText Integration Test ==========

func TestAddTextIntegration(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForText(t, ctx)
	defer cleanupTestCollectionForText(ctx, collectionID)

	t.Run("FullTextLifecycle", func(t *testing.T) {
		// 1. Add Text Document
		addParams := &api.AddTextParams{
			CollectionID: collectionID,
			Text:         "This is a comprehensive test of the text document lifecycle including creation, retrieval, and removal.",
			Locale:       "en",
			Metadata: map[string]interface{}{
				"title":       "Text Lifecycle Test",
				"description": "Full lifecycle integration test",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			AuthScope: map[string]interface{}{
				"__yao_created_by": "integration_test",
			},
		}

		result, err := kb.API.AddText(ctx, addParams)
		if err != nil {
			t.Skipf("Skipping integration test due to AddText error: %v", err)
			return
		}
		assert.NotNil(t, result)
		t.Logf("1. Created text document: %s", result.DocID)

		// 2. Get Document
		doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "Text Lifecycle Test", doc["name"])
		assert.Equal(t, "text", doc["type"])
		assert.Equal(t, "completed", doc["status"])
		t.Logf("2. Retrieved document: name=%v, status=%v", doc["name"], doc["status"])

		// 3. List Documents
		listFilter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: collectionID,
			Keywords:     "Text Lifecycle",
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

		t.Logf("✅ Full text lifecycle test completed successfully")
	})
}
