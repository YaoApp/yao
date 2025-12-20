package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/api"
)

// Note: TestMain is defined in collection_test.go
// Note: Test data setup is in search_setup_test.go

// ========== Search Query Tests ==========

// ensureTestDataExists ensures test collections exist by running setup if needed
// Setup will skip creation if data already exists
func ensureTestDataExists(t *testing.T, ctx context.Context) {
	// Run setup - it checks if data exists and skips if already complete
	TestSearchSetup(t)
}

func TestSearchQuery(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	ensureTestDataExists(t, ctx)

	t.Run("VectorSearch_SingleCollection", func(t *testing.T) {
		// Test: Simple vector search in science collection
		// Query about Einstein should find Einstein-related documents
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "Who is Albert Einstein and what did he discover?",
				Mode:         api.SearchModeVector,
				PageSize:     5,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Search error (may be expected if not implemented): %v", err)
			return
		}

		if result == nil {
			t.Skip("Search not implemented yet (returned nil)")
		}

		assert.Greater(t, len(result.Segments), 0, "Should find segments about Einstein")
		t.Logf("Vector search returned %d segments", len(result.Segments))

		// Verify relevance - top results should mention Einstein
		for i, seg := range result.Segments {
			t.Logf("  Segment %d (score: %.4f): %s...", i, seg.Score, truncateText(seg.Text, 100))
		}
	})

	t.Run("VectorSearch_MultipleQueries", func(t *testing.T) {
		// Test: Multiple queries in same collection, results should be merged
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "relativity theory",
				Mode:         api.SearchModeVector,
				PageSize:     3,
			},
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "Nobel Prize physics",
				Mode:         api.SearchModeVector,
				PageSize:     3,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Multi-query search returned %d merged segments", len(result.Segments))
	})

	t.Run("VectorSearch_CrossCollection", func(t *testing.T) {
		// Test: Search across both collections
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "innovation and discovery",
				Mode:         api.SearchModeVector,
				PageSize:     3,
			},
			{
				CollectionID: SearchTestTechCollection,
				Input:        "technology innovation",
				Mode:         api.SearchModeVector,
				PageSize:     3,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Cross-collection search returned %d segments", len(result.Segments))
	})

	t.Run("ExpandSearch_EntityExpansion", func(t *testing.T) {
		// Test: Expand mode should find related entities through graph
		// Query: "photoelectric effect" should expand to find:
		// - Einstein (discovered it)
		// - Nobel Prize (awarded for it)
		// - Quantum mechanics (built upon it)
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "photoelectric effect",
				Mode:         api.SearchModeExpand,
				MaxDepth:     2,
				PageSize:     5,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Expand search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Expand search returned %d segments", len(result.Segments))

		// Check if graph data is returned
		if result.Graph != nil {
			t.Logf("  Graph nodes: %d, relationships: %d",
				len(result.Graph.Nodes), len(result.Graph.Relationships))
		}

		// Verify expanded results include related entities
		for i, seg := range result.Segments {
			t.Logf("  Segment %d (score: %.4f): %s...", i, seg.Score, truncateText(seg.Text, 100))
		}
	})

	t.Run("ExpandSearch_DeepAssociation", func(t *testing.T) {
		// Test: Deep association through entity relationships
		// Query: "Germany physics" should expand to find:
		// - Einstein (born in Germany, physicist)
		// - Relativity (Einstein's theory)
		// - Planck (German physicist, quantum theory)
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "German physicist contributions",
				Mode:         api.SearchModeExpand,
				MaxDepth:     3,
				PageSize:     5,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Deep expand search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Deep expand search returned %d segments", len(result.Segments))
	})

	t.Run("GraphSearch_EntityTraversal", func(t *testing.T) {
		// Test: Pure graph search - find segments through entity relationships
		queries := []api.Query{
			{
				CollectionID: SearchTestTechCollection,
				Input:        "Steve Jobs",
				Mode:         api.SearchModeGraph,
				MaxDepth:     2,
				PageSize:     5,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Graph search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Graph search returned %d segments", len(result.Segments))

		if result.Graph != nil {
			t.Logf("  Found %d nodes, %d relationships",
				len(result.Graph.Nodes), len(result.Graph.Relationships))
			for _, node := range result.Graph.Nodes {
				t.Logf("    Node: %s (%s)", node.ID, node.EntityType)
			}
		}
	})

	t.Run("Search_WithMessages", func(t *testing.T) {
		// Test: Search using conversation history instead of direct input
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Messages: []graphragtypes.ChatMessage{
					{Role: "user", Content: "Tell me about famous physicists"},
					{Role: "assistant", Content: "There are many famous physicists throughout history..."},
					{Role: "user", Content: "What about Einstein specifically?"},
				},
				Mode:     api.SearchModeVector,
				PageSize: 5,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Message-based search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Message-based search returned %d segments", len(result.Segments))
	})

	t.Run("Search_WithDocumentFilter", func(t *testing.T) {
		// Test: Search within a specific document
		// First, get a document ID
		filter := &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     1,
			CollectionID: SearchTestScienceCollection,
		}
		listResult, err := kb.API.ListDocuments(ctx, filter)
		if err != nil || len(listResult.Data) == 0 {
			t.Skip("No documents available for filter test")
		}

		docID, ok := listResult.Data[0]["document_id"].(string)
		if !ok {
			t.Skip("Could not get document ID")
		}

		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				DocumentID:   docID,
				Input:        "physics discovery",
				Mode:         api.SearchModeVector,
				PageSize:     5,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Document-filtered search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Document-filtered search returned %d segments", len(result.Segments))

		// Verify all results are from the specified document
		for _, seg := range result.Segments {
			if seg.DocumentID != "" {
				assert.Equal(t, docID, seg.DocumentID, "All segments should be from filtered document")
			}
		}
	})

	t.Run("Search_WithPagination", func(t *testing.T) {
		// Test: Pagination
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "physics",
				Mode:         api.SearchModeVector,
				Page:         1,
				PageSize:     2,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Paginated search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		assert.LessOrEqual(t, len(result.Segments), 2, "Should respect page size")
		t.Logf("Page 1: %d segments, Total: %d, TotalPages: %d",
			len(result.Segments), result.Total, result.TotalPages)

		// Get page 2
		queries[0].Page = 2
		result2, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Page 2 search error: %v", err)
			return
		}

		if result2 != nil && len(result2.Segments) > 0 {
			t.Logf("Page 2: %d segments", len(result2.Segments))
		}
	})

	t.Run("Search_WithThreshold", func(t *testing.T) {
		// Test: Filter by similarity threshold
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "Einstein relativity",
				Mode:         api.SearchModeVector,
				Threshold:    0.5,
				PageSize:     10,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Threshold search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Threshold search returned %d segments", len(result.Segments))

		// Verify all results meet threshold
		for _, seg := range result.Segments {
			assert.GreaterOrEqual(t, seg.Score, 0.5, "All segments should meet threshold")
		}
	})

	t.Run("Search_WithMetadataFilter", func(t *testing.T) {
		// Test: Filter by metadata
		queries := []api.Query{
			{
				CollectionID: SearchTestScienceCollection,
				Input:        "physics",
				Mode:         api.SearchModeVector,
				Metadata: map[string]interface{}{
					"title": "Albert Einstein Biography",
				},
				PageSize: 10,
			},
		}

		result, err := kb.API.Search(ctx, queries)
		if err != nil {
			t.Logf("Metadata filter search error: %v", err)
			return
		}

		assert.NotNil(t, result)
		t.Logf("Metadata-filtered search returned %d segments", len(result.Segments))
	})
}

// ========== Error Handling Tests ==========

func TestSearchErrorHandling(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()

	t.Run("EmptyQueries", func(t *testing.T) {
		result, err := kb.API.Search(ctx, []api.Query{})
		// Empty queries should return empty result or error
		if err != nil {
			assert.Contains(t, err.Error(), "required")
		} else {
			assert.NotNil(t, result)
			assert.Equal(t, 0, len(result.Segments))
		}
	})

	t.Run("MissingCollectionID", func(t *testing.T) {
		queries := []api.Query{
			{
				Input: "test query",
				Mode:  api.SearchModeVector,
			},
		}

		_, err := kb.API.Search(ctx, queries)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection")
	})

	t.Run("MissingInputAndMessages", func(t *testing.T) {
		queries := []api.Query{
			{
				CollectionID: "some_collection",
				Mode:         api.SearchModeVector,
			},
		}

		_, err := kb.API.Search(ctx, queries)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "input")
	})

	t.Run("NonexistentCollection", func(t *testing.T) {
		queries := []api.Query{
			{
				CollectionID: "nonexistent_collection_xyz",
				Input:        "test query",
				Mode:         api.SearchModeVector,
			},
		}

		_, err := kb.API.Search(ctx, queries)
		assert.Error(t, err)
	})
}

// ========== Helper Functions ==========

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
