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

// createTestCollectionForURL is a helper to create a test collection for URL tests
func createTestCollectionForURL(t *testing.T, ctx context.Context) string {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	collectionID := fmt.Sprintf("test_url_%d", time.Now().UnixNano())

	params := &api.CreateCollectionParams{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"name":        "Test URL Collection",
			"description": "Collection for AddURL tests",
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

// cleanupTestCollectionForURL removes a test collection
func cleanupTestCollectionForURL(ctx context.Context, collectionID string) {
	if kb.API != nil {
		_, _ = kb.API.RemoveCollection(ctx, collectionID)
	}
}

// ========== AddURL Tests ==========

func TestAddURL(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForURL(t, ctx)
	defer cleanupTestCollectionForURL(ctx, collectionID)

	t.Run("AddURLSuccess", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://raw.githubusercontent.com/trheyi/yao/refs/heads/main/agent/caller/caller.go",
			Locale:       "en",
			Metadata: map[string]interface{}{
				"title":       "Yao Agent Caller",
				"description": "A test URL document",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			Fetcher: &api.ProviderConfigParams{
				ProviderID: "__yao.http",
				OptionID:   "http",
			},
			AuthScope: map[string]interface{}{
				"__yao_created_by": "test_user",
			},
		}

		result, err := kb.API.AddURL(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, collectionID, result.CollectionID)
			assert.NotEmpty(t, result.DocID)
			assert.Equal(t, "https://raw.githubusercontent.com/trheyi/yao/refs/heads/main/agent/caller/caller.go", result.URL)
			assert.Contains(t, result.Message, "successfully")
			t.Logf("Added URL document: %s", result.DocID)
		}

		// Verify document was created
		if result != nil {
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.NotNil(t, doc)
			assert.Equal(t, "url", doc["type"])
			assert.Equal(t, "https://raw.githubusercontent.com/trheyi/yao/refs/heads/main/agent/caller/caller.go", doc["url"])
			t.Logf("✅ URL Document verified: type=%v, url=%v, status=%v", doc["type"], doc["url"], doc["status"])
		}
	})

	t.Run("AddURLMissingCollectionID", func(t *testing.T) {
		params := &api.AddURLParams{
			URL: "https://example.com",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddURL(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "collection_id is required")
	})

	t.Run("AddURLMissingURL", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddURL(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "url is required")
	})

	t.Run("AddURLMissingChunking", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://example.com",
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddURL(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "chunking configuration is required")
	})

	t.Run("AddURLMissingEmbedding", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://example.com",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
		}

		result, err := kb.API.AddURL(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "embedding configuration is required")
	})

	t.Run("AddURLWithCustomDocID", func(t *testing.T) {
		customDocID := fmt.Sprintf("custom_url_doc_%d", time.Now().UnixNano())
		params := &api.AddURLParams{
			CollectionID: collectionID,
			DocID:        customDocID,
			URL:          "https://example.com", // Use root URL which always exists
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			Fetcher: &api.ProviderConfigParams{
				ProviderID: "__yao.http",
				OptionID:   "http",
			},
		}

		result, err := kb.API.AddURL(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, customDocID, result.DocID)
			t.Logf("Added URL with custom DocID: %s", result.DocID)
		}
	})

	t.Run("AddURLWithTitleFromMetadata", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://example.com", // Use root URL which always exists
			Metadata: map[string]interface{}{
				"title": "Custom URL Title From Metadata",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			Fetcher: &api.ProviderConfigParams{
				ProviderID: "__yao.http",
				OptionID:   "http",
			},
		}

		result, err := kb.API.AddURL(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.Equal(t, "Custom URL Title From Metadata", doc["name"])
			t.Logf("✅ Title from metadata verified: %v", doc["name"])
		}
	})
}

// ========== AddURLAsync Tests ==========

func TestAddURLAsync(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForURL(t, ctx)
	defer cleanupTestCollectionForURL(ctx, collectionID)

	t.Run("AddURLAsyncSuccess", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://example.com",
			Locale:       "en",
			Metadata: map[string]interface{}{
				"title": "Async URL Document",
			},
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			Fetcher: &api.ProviderConfigParams{
				ProviderID: "__yao.http",
				OptionID:   "http",
			},
			Job: &api.JobOptionsParams{
				Name:        "Test Async URL Job",
				Description: "Testing async URL processing",
				Category:    "Test",
			},
		}

		result, err := kb.API.AddURLAsync(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.NotEmpty(t, result.JobID)
			assert.NotEmpty(t, result.DocID)
			t.Logf("Created async URL job: %s for document: %s", result.JobID, result.DocID)
		}

		// Verify document was created
		if result != nil {
			doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
			assert.NoError(t, err)
			assert.NotNil(t, doc)
			assert.Equal(t, "url", doc["type"])
			assert.Equal(t, result.JobID, doc["job_id"])
			t.Logf("✅ Async URL document created: status=%v, job_id=%v", doc["status"], doc["job_id"])

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

	t.Run("AddURLAsyncMissingCollectionID", func(t *testing.T) {
		params := &api.AddURLParams{
			URL: "https://example.com",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddURLAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "collection_id is required")
	})

	t.Run("AddURLAsyncMissingURL", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddURLAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "url is required")
	})

	t.Run("AddURLAsyncMissingChunking", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://example.com",
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
			},
		}

		result, err := kb.API.AddURLAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "chunking configuration is required")
	})

	t.Run("AddURLAsyncMissingEmbedding", func(t *testing.T) {
		params := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://example.com",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
			},
		}

		result, err := kb.API.AddURLAsync(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "embedding configuration is required")
	})

	t.Run("AddURLAsyncWithCustomDocID", func(t *testing.T) {
		customDocID := fmt.Sprintf("async_custom_url_%d", time.Now().UnixNano())
		params := &api.AddURLParams{
			CollectionID: collectionID,
			DocID:        customDocID,
			URL:          "https://example.com/async-custom",
			Chunking: &api.ProviderConfigParams{
				ProviderID: "__yao.structured",
				OptionID:   "standard",
			},
			Embedding: &api.ProviderConfigParams{
				ProviderID: "__yao.openai",
				OptionID:   "text-embedding-3-small",
			},
			Fetcher: &api.ProviderConfigParams{
				ProviderID: "__yao.http",
				OptionID:   "http",
			},
		}

		result, err := kb.API.AddURLAsync(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, customDocID, result.DocID)
			t.Logf("Created async URL with custom DocID: %s", result.DocID)
		}
	})
}

// ========== AddURL Integration Test ==========

func TestAddURLIntegration(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForURL(t, ctx)
	defer cleanupTestCollectionForURL(ctx, collectionID)

	t.Run("FullURLLifecycle", func(t *testing.T) {
		// 1. Add URL Document
		addParams := &api.AddURLParams{
			CollectionID: collectionID,
			URL:          "https://example.com", // Use root URL which always exists
			Locale:       "en",
			Metadata: map[string]interface{}{
				"title":       "URL Lifecycle Test",
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
			Fetcher: &api.ProviderConfigParams{
				ProviderID: "__yao.http",
				OptionID:   "http",
			},
			AuthScope: map[string]interface{}{
				"__yao_created_by": "integration_test",
			},
		}

		result, err := kb.API.AddURL(ctx, addParams)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result == nil {
			t.Fatalf("Failed to create URL document: result is nil")
		}
		t.Logf("1. Created URL document: %s", result.DocID)

		// 2. Get Document
		doc, err := kb.API.GetDocument(ctx, result.DocID, nil)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "URL Lifecycle Test", doc["name"])
		assert.Equal(t, "url", doc["type"])
		assert.Equal(t, "https://example.com", doc["url"])
		t.Logf("2. Retrieved document: name=%v, url=%v, status=%v", doc["name"], doc["url"], doc["status"])

		// 3. List Documents
		listFilter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: collectionID,
			Keywords:     "URL Lifecycle",
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

		t.Logf("✅ Full URL lifecycle test completed successfully")
	})
}
