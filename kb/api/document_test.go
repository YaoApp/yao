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

// createTestCollectionForDoc is a helper to create a test collection for document tests
func createTestCollectionForDoc(t *testing.T, ctx context.Context) string {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	collectionID := fmt.Sprintf("test_doc_%d", time.Now().UnixNano())

	params := &api.CreateCollectionParams{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"name":        "Test Document Collection",
			"description": "Collection for document tests",
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

// cleanupTestCollectionForDoc removes a test collection
func cleanupTestCollectionForDoc(ctx context.Context, collectionID string) {
	if kb.API != nil {
		_, _ = kb.API.RemoveCollection(ctx, collectionID)
	}
}

// addTestDocument adds a test document and returns its ID
func addTestDocument(t *testing.T, ctx context.Context, collectionID, title string) string {
	params := &api.AddTextParams{
		CollectionID: collectionID,
		Text:         fmt.Sprintf("Test document content for %s", title),
		Metadata: map[string]interface{}{
			"title": title,
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
	if err != nil {
		t.Fatalf("Failed to add test document: %v", err)
	}
	return result.DocID
}

// ========== ListDocuments Tests ==========

func TestListDocuments(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForDoc(t, ctx)
	defer cleanupTestCollectionForDoc(ctx, collectionID)

	// Add some test documents
	for i := 0; i < 3; i++ {
		addTestDocument(t, ctx, collectionID, fmt.Sprintf("Test Document %d", i+1))
	}

	t.Run("ListDocumentsDefault", func(t *testing.T) {
		filter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: collectionID,
		}

		result, err := kb.API.ListDocuments(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Data), 3)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
		t.Logf("Found %d documents in collection", len(result.Data))
	})

	t.Run("ListDocumentsWithPagination", func(t *testing.T) {
		filter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     2,
			CollectionID: collectionID,
		}

		result, err := kb.API.ListDocuments(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.LessOrEqual(t, len(result.Data), 2)
	})

	t.Run("ListDocumentsWithKeywords", func(t *testing.T) {
		filter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: collectionID,
			Keywords:     "Test Document 1",
		}

		result, err := kb.API.ListDocuments(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Data), 1)
	})

	t.Run("ListDocumentsWithStatus", func(t *testing.T) {
		filter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: collectionID,
			Status:       []string{"completed"},
		}

		result, err := kb.API.ListDocuments(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		for _, doc := range result.Data {
			status, ok := doc["status"].(string)
			if ok {
				assert.Equal(t, "completed", status)
			}
		}
	})

	t.Run("ListDocumentsEmptyResult", func(t *testing.T) {
		filter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: collectionID,
			Keywords:     "nonexistent_keyword_xyz123",
		}

		result, err := kb.API.ListDocuments(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result.Data))
	})
}

// ========== GetDocument Tests ==========

func TestGetDocument(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForDoc(t, ctx)
	defer cleanupTestCollectionForDoc(ctx, collectionID)

	// Add a test document
	docID := addTestDocument(t, ctx, collectionID, "GetDocument Test")

	t.Run("GetDocumentSuccess", func(t *testing.T) {
		doc, err := kb.API.GetDocument(ctx, docID, nil)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, docID, doc["document_id"])
		assert.Equal(t, collectionID, doc["collection_id"])
		assert.Equal(t, "GetDocument Test", doc["name"])
		assert.Equal(t, "text", doc["type"])
		t.Logf("Retrieved document: %v", doc["name"])
	})

	t.Run("GetDocumentWithSelect", func(t *testing.T) {
		params := &api.GetDocumentParams{
			Select: []interface{}{"document_id", "name", "type", "status"},
		}

		doc, err := kb.API.GetDocument(ctx, docID, params)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.NotNil(t, doc["document_id"])
		assert.NotNil(t, doc["name"])
	})

	t.Run("GetDocumentNotFound", func(t *testing.T) {
		doc, err := kb.API.GetDocument(ctx, "nonexistent_doc_id", nil)
		assert.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetDocumentEmptyID", func(t *testing.T) {
		doc, err := kb.API.GetDocument(ctx, "", nil)
		assert.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "required")
	})
}

// ========== RemoveDocuments Tests ==========

func TestRemoveDocuments(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	collectionID := createTestCollectionForDoc(t, ctx)
	defer cleanupTestCollectionForDoc(ctx, collectionID)

	// Add test documents
	var docIDs []string
	for i := 0; i < 3; i++ {
		docID := addTestDocument(t, ctx, collectionID, fmt.Sprintf("Remove Test %d", i+1))
		docIDs = append(docIDs, docID)
	}

	t.Run("RemoveDocumentsSuccess", func(t *testing.T) {
		params := &api.RemoveDocumentsParams{
			DocumentIDs: docIDs[:2], // Remove first 2 documents
		}

		result, err := kb.API.RemoveDocuments(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.RequestedCount)
		assert.GreaterOrEqual(t, result.DeletedCount, 0)
		t.Logf("Removed documents: requested=%d, deleted=%d", result.RequestedCount, result.DeletedCount)

		// Verify documents are removed
		for _, docID := range docIDs[:2] {
			doc, err := kb.API.GetDocument(ctx, docID, nil)
			assert.Error(t, err)
			assert.Nil(t, doc)
		}

		// Verify remaining document still exists
		doc, err := kb.API.GetDocument(ctx, docIDs[2], nil)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
	})

	t.Run("RemoveDocumentsEmptyList", func(t *testing.T) {
		params := &api.RemoveDocumentsParams{
			DocumentIDs: []string{},
		}

		result, err := kb.API.RemoveDocuments(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("RemoveDocumentsNonexistent", func(t *testing.T) {
		params := &api.RemoveDocumentsParams{
			DocumentIDs: []string{"nonexistent_doc_1", "nonexistent_doc_2"},
		}

		result, err := kb.API.RemoveDocuments(ctx, params)
		// Should succeed but with 0 deleted
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.RequestedCount)
	})
}
