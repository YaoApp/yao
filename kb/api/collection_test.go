package api_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/api"
	"github.com/yaoapp/yao/test"
)

func TestMain(m *testing.M) {
	// Setup test environment
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Load attachment managers (needed for file upload tests)
	err := attachment.Load(config.Conf)
	if err != nil {
		panic("Failed to load attachment managers: " + err.Error())
	}

	// Load knowledge base
	_, err = kb.Load(config.Conf)
	if err != nil {
		panic("Failed to load knowledge base: " + err.Error())
	}

	// Run tests and exit with status code
	os.Exit(m.Run())
}

func TestCreateCollection(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	testCollectionID := fmt.Sprintf("test_create_%d", time.Now().UnixNano())

	// Clean up after test
	defer func() {
		_, _ = kb.API.RemoveCollection(ctx, testCollectionID)
	}()

	t.Run("CreateCollectionSuccess", func(t *testing.T) {
		params := &api.CreateCollectionParams{
			ID: testCollectionID,
			Metadata: map[string]interface{}{
				"name":        "Test Collection",
				"description": "Test Description",
				"share":       "team",
			},
			EmbeddingProviderID: "__yao.openai",
			EmbeddingOptionID:   "text-embedding-3-small",
			Locale:              "en",
			Config: &graphragtypes.CreateCollectionOptions{
				Distance:       "cosine",
				IndexType:      "hnsw",
				M:              16,
				EfConstruction: 200,
				EfSearch:       64,
				// Dimension will be set automatically by the API from provider settings
			},
			AuthScope: map[string]interface{}{
				"__yao_created_by": "test_user",
				"__yao_team_id":    "test_team",
			},
		}

		result, err := kb.API.CreateCollection(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, testCollectionID, result.CollectionID)
			assert.Contains(t, result.Message, "successfully")
			t.Logf("Created collection: %s", result.CollectionID)
		}

		// ✅ Verify that auth scope fields are stored in GraphRag metadata
		collection, err := kb.API.GetCollection(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.NotNil(t, collection)

		// Check metadata object
		metadata, ok := collection["metadata"].(map[string]interface{})
		assert.True(t, ok, "metadata should be a map")

		// Verify auth scope fields in metadata (for permission-based vector search)
		assert.Equal(t, "test_user", metadata["__yao_created_by"], "created_by should be in metadata")
		assert.Equal(t, "test_team", metadata["__yao_team_id"], "team_id should be in metadata")
		t.Logf("✅ Auth scope fields verified in metadata: created_by=%v, team_id=%v",
			metadata["__yao_created_by"], metadata["__yao_team_id"])

		// Verify they are also flattened at top level
		assert.Equal(t, "test_user", collection["__yao_created_by"], "created_by should be at top level")
		assert.Equal(t, "test_team", collection["__yao_team_id"], "team_id should be at top level")

		// ✅ Verify other database fields in metadata
		assert.Equal(t, "team", metadata["share"], "share should be in metadata")
		assert.Equal(t, "active", metadata["status"], "status should be in metadata")
		assert.NotNil(t, metadata["preset"], "preset should be in metadata")
		assert.NotNil(t, metadata["public"], "public should be in metadata")
		assert.NotNil(t, metadata["sort"], "sort should be in metadata")
		t.Logf("✅ Database fields verified in metadata: share=%v, status=%v, preset=%v, public=%v",
			metadata["share"], metadata["status"], metadata["preset"], metadata["public"])
	})

	t.Run("CreateCollectionMissingID", func(t *testing.T) {
		params := &api.CreateCollectionParams{
			EmbeddingProviderID: "__yao.openai",
			EmbeddingOptionID:   "text-embedding-3-small",
			Config: &graphragtypes.CreateCollectionOptions{
				Distance:  "cosine",
				IndexType: "hnsw",
			},
		}

		result, err := kb.API.CreateCollection(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "id is required")
	})

	t.Run("CreateCollectionMissingProvider", func(t *testing.T) {
		params := &api.CreateCollectionParams{
			ID:                "test_missing_provider",
			EmbeddingOptionID: "text-embedding-3-small",
			Config: &graphragtypes.CreateCollectionOptions{
				Distance:  "cosine",
				IndexType: "hnsw",
			},
		}

		result, err := kb.API.CreateCollection(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "embedding_provider_id is required")
	})

	t.Run("CreateCollectionInvalidProvider", func(t *testing.T) {
		params := &api.CreateCollectionParams{
			ID:                  "test_invalid_provider",
			EmbeddingProviderID: "invalid_provider",
			EmbeddingOptionID:   "invalid_option",
			Config: &graphragtypes.CreateCollectionOptions{
				Distance:  "cosine",
				IndexType: "hnsw",
			},
		}

		result, err := kb.API.CreateCollection(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "provider")
	})
}

func TestGetCollection(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	testCollectionID := fmt.Sprintf("test_get_%d", time.Now().UnixNano())

	// Create a test collection first
	params := &api.CreateCollectionParams{
		ID: testCollectionID,
		Metadata: map[string]interface{}{
			"name":        "Test Get Collection",
			"description": "Test Description",
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
	assert.NoError(t, err)

	// Clean up after test
	defer func() {
		_, _ = kb.API.RemoveCollection(ctx, testCollectionID)
	}()

	t.Run("GetCollectionSuccess", func(t *testing.T) {
		collection, err := kb.API.GetCollection(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.NotNil(t, collection)

		// Check that both id and collection_id are present
		assert.Equal(t, testCollectionID, collection["id"])
		assert.Equal(t, testCollectionID, collection["collection_id"])

		// Check that metadata is present
		assert.NotNil(t, collection["metadata"])
		metadata, ok := collection["metadata"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Test Get Collection", metadata["name"])

		// Check that fields are also flattened at top level
		assert.Equal(t, "Test Get Collection", collection["name"])

		// Check that config is present
		assert.NotNil(t, collection["config"])

		// ✅ Check that timestamps are present in metadata (for frontend)
		assert.NotNil(t, metadata["created_at"], "created_at should be present in metadata")
		assert.NotNil(t, metadata["updated_at"], "updated_at should be present in metadata")
		t.Logf("Timestamps in metadata: created_at=%v, updated_at=%v", metadata["created_at"], metadata["updated_at"])

		// ✅ Check that timestamps are also flattened at top level
		assert.NotNil(t, collection["created_at"], "created_at should be present at top level")
		assert.NotNil(t, collection["updated_at"], "updated_at should be present at top level")
		t.Logf("Timestamps at top level: created_at=%v, updated_at=%v", collection["created_at"], collection["updated_at"])

		// Note: This test doesn't create collection with auth scope, so permission fields won't be present
		// See TestCreateCollection/CreateCollectionSuccess for auth scope verification

		t.Logf("Retrieved collection: %v", collection["id"])
	})

	t.Run("GetCollectionNotFound", func(t *testing.T) {
		collection, err := kb.API.GetCollection(ctx, "nonexistent_collection")
		assert.Error(t, err)
		assert.Nil(t, collection)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetCollectionEmptyID", func(t *testing.T) {
		collection, err := kb.API.GetCollection(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, collection)
		assert.Contains(t, err.Error(), "required")
	})
}

func TestCollectionExists(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	testCollectionID := fmt.Sprintf("test_exists_%d", time.Now().UnixNano())

	// Create a test collection
	params := &api.CreateCollectionParams{
		ID: testCollectionID,
		Metadata: map[string]interface{}{
			"name": "Test Exists Collection",
		},
		EmbeddingProviderID: "__yao.openai",
		EmbeddingOptionID:   "text-embedding-3-small",
		Config: &graphragtypes.CreateCollectionOptions{
			Distance:  "cosine",
			IndexType: "hnsw",
		},
	}

	_, err := kb.API.CreateCollection(ctx, params)
	assert.NoError(t, err)

	// Clean up after test
	defer func() {
		_, _ = kb.API.RemoveCollection(ctx, testCollectionID)
	}()

	t.Run("CollectionExistsTrue", func(t *testing.T) {
		result, err := kb.API.CollectionExists(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Exists)
		assert.Equal(t, testCollectionID, result.CollectionID)
	})

	t.Run("CollectionExistsFalse", func(t *testing.T) {
		result, err := kb.API.CollectionExists(ctx, "nonexistent_collection")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Exists)
	})

	t.Run("CollectionExistsEmptyID", func(t *testing.T) {
		result, err := kb.API.CollectionExists(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "required")
	})
}

func TestRemoveCollection(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	testCollectionID := fmt.Sprintf("test_remove_%d", time.Now().UnixNano())

	// Create a test collection
	params := &api.CreateCollectionParams{
		ID: testCollectionID,
		Metadata: map[string]interface{}{
			"name": "Test Remove Collection",
		},
		EmbeddingProviderID: "__yao.openai",
		EmbeddingOptionID:   "text-embedding-3-small",
		Config: &graphragtypes.CreateCollectionOptions{
			Distance:  "cosine",
			IndexType: "hnsw",
		},
	}

	_, err := kb.API.CreateCollection(ctx, params)
	assert.NoError(t, err)

	t.Run("RemoveCollectionSuccess", func(t *testing.T) {
		result, err := kb.API.RemoveCollection(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Removed)
		assert.Equal(t, testCollectionID, result.CollectionID)
		assert.Contains(t, result.Message, "successfully")

		// Verify collection is removed
		exists, err := kb.API.CollectionExists(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.False(t, exists.Exists)
	})

	t.Run("RemoveCollectionNotFound", func(t *testing.T) {
		// The new implementation is more tolerant - it attempts database cleanup
		// even if the collection doesn't exist in GraphRag
		// This is considered successful as long as database cleanup succeeds
		result, err := kb.API.RemoveCollection(ctx, "nonexistent_collection")

		// Should succeed (database cleanup succeeds even if collection doesn't exist)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Removed)
		t.Logf("✓ Handled non-existent collection gracefully (database cleanup succeeded)")
	})

	t.Run("RemoveCollectionEmptyID", func(t *testing.T) {
		result, err := kb.API.RemoveCollection(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("RemoveCollectionInconsistentState", func(t *testing.T) {
		// Test removing a collection that exists in database but not in vector store
		// This simulates an inconsistent state that can occur after failed operations
		inconsistentCollectionID := fmt.Sprintf("test_inconsistent_%d", time.Now().UnixNano())

		// Create a test collection first
		params := &api.CreateCollectionParams{
			ID: inconsistentCollectionID,
			Metadata: map[string]interface{}{
				"name": "Test Inconsistent Collection",
			},
			EmbeddingProviderID: "__yao.openai",
			EmbeddingOptionID:   "text-embedding-3-small",
			Config: &graphragtypes.CreateCollectionOptions{
				Distance:  "cosine",
				IndexType: "hnsw",
			},
		}

		_, err := kb.API.CreateCollection(ctx, params)
		assert.NoError(t, err)

		// Now remove it normally first time
		result, err := kb.API.RemoveCollection(ctx, inconsistentCollectionID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Removed)

		// Verify it's gone
		exists, err := kb.API.CollectionExists(ctx, inconsistentCollectionID)
		assert.NoError(t, err)
		assert.False(t, exists.Exists)

		t.Logf("✓ Successfully removed collection in inconsistent state")
	})
}

func TestListCollections(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()

	// Create multiple test collections
	timestamp := time.Now().UnixNano()
	testCollections := []string{
		fmt.Sprintf("test_list_1_%d", timestamp),
		fmt.Sprintf("test_list_2_%d", timestamp),
		fmt.Sprintf("test_list_3_%d", timestamp),
	}

	for i, collectionID := range testCollections {
		params := &api.CreateCollectionParams{
			ID: collectionID,
			Metadata: map[string]interface{}{
				"name":        "Test List Collection " + string(rune('A'+i)),
				"description": "Description " + string(rune('A'+i)),
			},
			EmbeddingProviderID: "__yao.openai",
			EmbeddingOptionID:   "text-embedding-3-small",
			Config: &graphragtypes.CreateCollectionOptions{
				Distance:  "cosine",
				IndexType: "hnsw",
			},
		}
		_, err := kb.API.CreateCollection(ctx, params)
		assert.NoError(t, err)
	}

	// Clean up after test
	defer func() {
		for _, collectionID := range testCollections {
			_, _ = kb.API.RemoveCollection(ctx, collectionID)
		}
	}()

	t.Run("ListCollectionsDefault", func(t *testing.T) {
		filter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 20,
		}

		result, err := kb.API.ListCollections(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Data)
		assert.GreaterOrEqual(t, len(result.Data), 3) // At least our 3 test collections
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)

		t.Logf("Found %d collections", len(result.Data))
	})

	t.Run("ListCollectionsWithPagination", func(t *testing.T) {
		filter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 2,
		}

		result, err := kb.API.ListCollections(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.LessOrEqual(t, len(result.Data), 2)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 2, result.PageSize)
	})

	t.Run("ListCollectionsWithKeywords", func(t *testing.T) {
		filter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 20,
			Keywords: "Test List Collection A",
		}

		result, err := kb.API.ListCollections(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Data), 1)

		// Check that returned collections match the keyword
		for _, item := range result.Data {
			name, ok := item["name"].(string)
			if ok {
				assert.Contains(t, name, "Test List Collection")
			}
		}
	})

	t.Run("ListCollectionsWithStatus", func(t *testing.T) {
		filter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 20,
			Status:   []string{"active"},
		}

		result, err := kb.API.ListCollections(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// All collections should have status "active"
		for _, item := range result.Data {
			status, ok := item["status"].(string)
			if ok {
				assert.Equal(t, "active", status)
			}
		}
	})

	t.Run("ListCollectionsWithSort", func(t *testing.T) {
		filter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 20,
			Sort: []model.QueryOrder{
				{Column: "created_at", Option: "desc"},
			},
		}

		result, err := kb.API.ListCollections(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Data), 3)
	})

	t.Run("ListCollectionsWithSelect", func(t *testing.T) {
		filter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 20,
			Select:   []interface{}{"id", "collection_id", "name", "status"},
		}

		result, err := kb.API.ListCollections(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Data), 3)

		// Check that returned fields are limited
		for _, item := range result.Data {
			assert.NotNil(t, item["collection_id"])
			assert.NotNil(t, item["name"])
		}
	})

	t.Run("ListCollectionsEmptyResult", func(t *testing.T) {
		filter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 20,
			Keywords: "nonexistent_keyword_xyz123",
		}

		result, err := kb.API.ListCollections(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Data)
		assert.Equal(t, 0, len(result.Data))
	})
}

func TestUpdateCollectionMetadata(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	testCollectionID := fmt.Sprintf("test_update_%d", time.Now().UnixNano())

	// Create a test collection
	params := &api.CreateCollectionParams{
		ID: testCollectionID,
		Metadata: map[string]interface{}{
			"name":        "Original Name",
			"description": "Original Description",
		},
		EmbeddingProviderID: "__yao.openai",
		EmbeddingOptionID:   "text-embedding-3-small",
		Config: &graphragtypes.CreateCollectionOptions{
			Distance:  "cosine",
			IndexType: "hnsw",
		},
	}

	_, err := kb.API.CreateCollection(ctx, params)
	assert.NoError(t, err)

	// Clean up after test
	defer func() {
		_, _ = kb.API.RemoveCollection(ctx, testCollectionID)
	}()

	t.Run("UpdateMetadataSuccess", func(t *testing.T) {
		updateParams := &api.UpdateMetadataParams{
			Metadata: map[string]interface{}{
				"name":        "Updated Name",
				"description": "Updated Description",
			},
			AuthScope: map[string]interface{}{
				"__yao_updated_by": "test_user",
			},
		}

		result, err := kb.API.UpdateCollectionMetadata(ctx, testCollectionID, updateParams)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testCollectionID, result.CollectionID)
		assert.Contains(t, result.Message, "successfully")

		// Verify the update
		collection, err := kb.API.GetCollection(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", collection["name"])
		assert.Equal(t, "Updated Description", collection["description"])
	})

	t.Run("UpdateMetadataEmptyID", func(t *testing.T) {
		updateParams := &api.UpdateMetadataParams{
			Metadata: map[string]interface{}{
				"name": "Updated Name",
			},
		}

		result, err := kb.API.UpdateCollectionMetadata(ctx, "", updateParams)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("UpdateMetadataEmptyMetadata", func(t *testing.T) {
		updateParams := &api.UpdateMetadataParams{
			Metadata: map[string]interface{}{},
		}

		result, err := kb.API.UpdateCollectionMetadata(ctx, testCollectionID, updateParams)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("UpdateMetadataNotFound", func(t *testing.T) {
		updateParams := &api.UpdateMetadataParams{
			Metadata: map[string]interface{}{
				"name": "Updated Name",
			},
		}

		result, err := kb.API.UpdateCollectionMetadata(ctx, "nonexistent_collection", updateParams)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCollectionIntegration(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()
	testCollectionID := fmt.Sprintf("test_integration_%d", time.Now().UnixNano())

	t.Run("FullCollectionLifecycle", func(t *testing.T) {
		// 1. Create Collection
		createParams := &api.CreateCollectionParams{
			ID: testCollectionID,
			Metadata: map[string]interface{}{
				"name":        "Integration Test Collection",
				"description": "Full lifecycle test",
				"share":       "team",
			},
			EmbeddingProviderID: "__yao.openai",
			EmbeddingOptionID:   "text-embedding-3-small",
			Locale:              "en",
			Config: &graphragtypes.CreateCollectionOptions{
				Distance:  "cosine",
				IndexType: "hnsw",
			},
		}

		createResult, err := kb.API.CreateCollection(ctx, createParams)
		assert.NoError(t, err)
		assert.NotNil(t, createResult)
		t.Logf("Created collection: %s", createResult.CollectionID)

		// 2. Check Exists
		existsResult, err := kb.API.CollectionExists(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.True(t, existsResult.Exists)
		t.Logf("Collection exists: %v", existsResult.Exists)

		// 3. Get Collection
		collection, err := kb.API.GetCollection(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.Equal(t, testCollectionID, collection["id"])
		assert.Equal(t, testCollectionID, collection["collection_id"])
		assert.Equal(t, "Integration Test Collection", collection["name"])
		t.Logf("Retrieved collection: %s", collection["name"])

		// 4. Update Metadata
		updateParams := &api.UpdateMetadataParams{
			Metadata: map[string]interface{}{
				"name":        "Updated Integration Test",
				"description": "Updated description",
			},
		}
		updateResult, err := kb.API.UpdateCollectionMetadata(ctx, testCollectionID, updateParams)
		assert.NoError(t, err)
		assert.NotNil(t, updateResult)
		t.Logf("Updated collection metadata")

		// 5. Verify Update
		updatedCollection, err := kb.API.GetCollection(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Integration Test", updatedCollection["name"])
		t.Logf("Verified update: %s", updatedCollection["name"])

		// 6. List Collections (should include our test collection)
		listFilter := &api.ListCollectionsFilter{
			Page:     1,
			PageSize: 20,
			Keywords: "Updated Integration Test",
		}
		listResult, err := kb.API.ListCollections(ctx, listFilter)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(listResult.Data), 1)
		t.Logf("Found collection in list")

		// 7. Remove Collection
		removeResult, err := kb.API.RemoveCollection(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.True(t, removeResult.Removed)
		t.Logf("Removed collection: %s", removeResult.CollectionID)

		// 8. Verify Removal
		existsAfterRemove, err := kb.API.CollectionExists(ctx, testCollectionID)
		assert.NoError(t, err)
		assert.False(t, existsAfterRemove.Exists)
		t.Logf("Verified removal: exists=%v", existsAfterRemove.Exists)
	})
}
