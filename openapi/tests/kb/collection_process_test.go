package openapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestProcessCreateCollection tests the kb.collection.Create process
func TestProcessCreateCollection(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Ensure KB is initialized
	if kb.API == nil {
		t.Skip("Knowledge base not initialized - skipping test")
	}

	testCollectionID := fmt.Sprintf("test_process_create_%d", time.Now().UnixNano())

	t.Run("CreateCollectionWithAuth", func(t *testing.T) {
		// Register collection for cleanup
		testutils.RegisterTestCollection(testCollectionID)

		// Create process with authorized info
		p := process.New("kb.collection.Create").
			WithContext(context.Background()).
			WithAuthorized(&process.AuthorizedInfo{
				UserID:   "test_user_123",
				TeamID:   "test_team_456",
				Subject:  "user@example.com",
				ClientID: "test_client",
				Scope:    "openid profile",
				Constraints: process.DataConstraints{
					TeamOnly: true,
				},
			})

		// Prepare parameters - use map for Process API
		params := map[string]interface{}{
			"id": testCollectionID,
			"metadata": map[string]interface{}{
				"name":        "Process Test Collection",
				"description": "Created via Process API with auth",
			},
			"embedding_provider_id": "__yao.openai",
			"embedding_option_id":   "text-embedding-3-small",
			"locale":                "en",
			"config": map[string]interface{}{
				"index_type": "hnsw",
				"distance":   "cosine",
			},
		}

		p.Args = []interface{}{params}

		// Execute process
		result, err := p.Exec()
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify result
		resultMap, ok := result.(maps.MapStrAny)
		require.True(t, ok, "Result should be a maps.MapStrAny")
		assert.Equal(t, testCollectionID, resultMap["collection_id"])
		assert.Contains(t, resultMap, "message")

		t.Logf("✓ Successfully created collection via process: %s", testCollectionID)

		// Verify auth scope was applied by checking the collection
		collection, err := kb.API.GetCollection(context.Background(), testCollectionID)
		require.NoError(t, err)
		assert.NotNil(t, collection)

		// Check if auth fields were set
		if createdBy, ok := collection["__yao_created_by"]; ok {
			assert.Equal(t, "test_user_123", createdBy)
			t.Logf("✓ Auth scope applied: __yao_created_by = %v", createdBy)
		}
		if teamID, ok := collection["__yao_team_id"]; ok {
			assert.Equal(t, "test_team_456", teamID)
			t.Logf("✓ Auth scope applied: __yao_team_id = %v", teamID)
		}
	})

	t.Run("CreateCollectionWithoutAuth", func(t *testing.T) {
		testCollectionID2 := fmt.Sprintf("test_process_create_noauth_%d", time.Now().UnixNano())
		testutils.RegisterTestCollection(testCollectionID2)

		// Create process without authorized info
		p := process.New("kb.collection.Create").
			WithContext(context.Background())

		params := map[string]interface{}{
			"id": testCollectionID2,
			"metadata": map[string]interface{}{
				"name":        "Process Test Collection No Auth",
				"description": "Created via Process API without auth",
			},
			"embedding_provider_id": "__yao.openai",
			"embedding_option_id":   "text-embedding-3-small",
			"locale":                "en",
			"config": map[string]interface{}{
				"index_type": "hnsw",
				"distance":   "cosine",
			},
		}

		p.Args = []interface{}{params}

		// Execute process
		result, err := p.Exec()
		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap, ok := result.(maps.MapStrAny)
		require.True(t, ok)
		assert.Equal(t, testCollectionID2, resultMap["collection_id"])

		t.Logf("✓ Successfully created collection without auth: %s", testCollectionID2)
	})

	t.Run("CreateCollectionInvalidParams", func(t *testing.T) {
		// Create process with invalid parameters
		p := process.New("kb.collection.Create").
			WithContext(context.Background())

		// Missing required fields
		params := map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "Invalid Collection",
			},
			// Missing id, embedding_provider_id, etc.
		}

		p.Args = []interface{}{params}

		// Execute should throw exception or return error
		defer func() {
			if r := recover(); r != nil {
				t.Logf("✓ Correctly rejected invalid parameters via panic: %v", r)
				return
			}
		}()

		result, err := p.Exec()
		if err != nil {
			t.Logf("✓ Correctly rejected invalid parameters via error: %v", err)
			return
		}
		if result == nil {
			t.Log("✓ Correctly rejected invalid parameters (nil result)")
			return
		}

		t.Error("Should have thrown exception or returned error for invalid parameters")
	})
}

// TestProcessListCollections tests the kb.collection.List process
func TestProcessListCollections(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	if kb.API == nil {
		t.Skip("Knowledge base not initialized - skipping test")
	}

	t.Run("ListCollectionsWithAuth", func(t *testing.T) {
		// Test listing with auth filters
		p := process.New("kb.collection.List").
			WithContext(context.Background()).
			WithAuthorized(&process.AuthorizedInfo{
				UserID: "test_user_789",
				TeamID: "test_team_789",
				Constraints: process.DataConstraints{
					TeamOnly: true,
				},
			})

		filter := map[string]interface{}{
			"page":     1,
			"pagesize": 20,
		}

		p.Args = []interface{}{filter}

		// Execute process
		result, err := p.Exec()
		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap, ok := result.(maps.MapStrAny)
		require.True(t, ok, "Result should be a map")

		assert.Contains(t, resultMap, "data")
		assert.Contains(t, resultMap, "page")
		assert.Contains(t, resultMap, "pagesize")
		assert.Contains(t, resultMap, "total")

		t.Logf("✓ Retrieved collections with auth filters")
	})

	t.Run("ListCollectionsNoFilter", func(t *testing.T) {
		// Test listing without filter (should use defaults)
		p := process.New("kb.collection.List").
			WithContext(context.Background())

		// No arguments - should use default filter
		p.Args = []interface{}{}

		result, err := p.Exec()
		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap, ok := result.(maps.MapStrAny)
		require.True(t, ok)
		assert.Contains(t, resultMap, "data")
		assert.Equal(t, 1, resultMap["page"])
		assert.Equal(t, 20, resultMap["pagesize"])

		t.Logf("✓ Retrieved collections with default filter")
	})
}

// TestProcessGetCollection tests the kb.collection.Get process
func TestProcessGetCollection(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	if kb.API == nil {
		t.Skip("Knowledge base not initialized - skipping test")
	}

	testCollectionID := fmt.Sprintf("test_process_get_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	t.Run("GetCollectionNotFound", func(t *testing.T) {
		p := process.New("kb.collection.Get").
			WithContext(context.Background())

		p.Args = []interface{}{"nonexistent_collection_id"}

		// Execute should throw exception or return error
		defer func() {
			if r := recover(); r != nil {
				t.Logf("✓ Correctly rejected nonexistent collection via panic: %v", r)
				return
			}
		}()

		result, err := p.Exec()
		if err != nil {
			t.Logf("✓ Correctly rejected nonexistent collection via error: %v", err)
			return
		}
		if result == nil {
			t.Log("✓ Correctly rejected nonexistent collection (nil result)")
			return
		}

		t.Error("Should have thrown exception or returned error for nonexistent collection")
	})
}

// TestProcessCollectionExists tests the kb.collection.Exists process
func TestProcessCollectionExists(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	if kb.API == nil {
		t.Skip("Knowledge base not initialized - skipping test")
	}

	testCollectionID := fmt.Sprintf("test_process_exists_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	t.Run("CollectionExistsBeforeCreation", func(t *testing.T) {
		p := process.New("kb.collection.Exists").
			WithContext(context.Background())

		p.Args = []interface{}{testCollectionID}

		result, err := p.Exec()
		require.NoError(t, err)
		require.NotNil(t, result)

		resultMap, ok := result.(maps.MapStrAny)
		require.True(t, ok)

		assert.Equal(t, testCollectionID, resultMap["collection_id"])
		assert.Equal(t, false, resultMap["exists"])

		t.Logf("✓ Correctly reported collection does not exist")
	})
}

// TestProcessCollectionIntegration tests the full collection lifecycle via Process API
func TestProcessCollectionIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	if kb.API == nil {
		t.Skip("Knowledge base not initialized - skipping test")
	}

	testCollectionID := fmt.Sprintf("test_process_integration_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	ctx := context.Background()
	authInfo := &process.AuthorizedInfo{
		UserID:   "integration_user",
		TeamID:   "integration_team",
		Subject:  "integration@example.com",
		ClientID: "integration_client",
		Scope:    "openid profile",
		Constraints: process.DataConstraints{
			TeamOnly: true,
		},
	}

	t.Run("FullLifecycleViaProcess", func(t *testing.T) {
		// Step 1: Check collection doesn't exist
		p1 := process.New("kb.collection.Exists").WithContext(ctx)
		p1.Args = []interface{}{testCollectionID}
		result1, err := p1.Exec()
		require.NoError(t, err)
		existsResult := result1.(maps.MapStrAny)
		assert.Equal(t, false, existsResult["exists"])
		t.Logf("✓ Step 1: Confirmed collection doesn't exist")

		// Step 2: Create collection
		p2 := process.New("kb.collection.Create").WithContext(ctx).WithAuthorized(authInfo)
		p2.Args = []interface{}{
			map[string]interface{}{
				"id": testCollectionID,
				"metadata": map[string]interface{}{
					"name":        "Integration Test Collection",
					"description": "Full lifecycle test",
				},
				"embedding_provider_id": "__yao.openai",
				"embedding_option_id":   "text-embedding-3-small",
				"locale":                "en",
				"config": map[string]interface{}{
					"index_type": "hnsw",
					"distance":   "cosine",
				},
			},
		}
		result2, err := p2.Exec()
		require.NoError(t, err)
		createResult := result2.(maps.MapStrAny)
		assert.Equal(t, testCollectionID, createResult["collection_id"])
		t.Logf("✓ Step 2: Created collection")

		// Step 3: Verify collection exists
		p3 := process.New("kb.collection.Exists").WithContext(ctx)
		p3.Args = []interface{}{testCollectionID}
		result3, err := p3.Exec()
		require.NoError(t, err)
		existsResult2 := result3.(maps.MapStrAny)
		assert.Equal(t, true, existsResult2["exists"])
		t.Logf("✓ Step 3: Confirmed collection exists")

		// Step 4: Get collection
		p4 := process.New("kb.collection.Get").WithContext(ctx)
		p4.Args = []interface{}{testCollectionID}
		result4, err := p4.Exec()
		require.NoError(t, err)
		// GetCollection returns map[string]interface{}
		var getResult map[string]interface{}
		if mapStrAny, ok := result4.(maps.MapStrAny); ok {
			getResult = mapStrAny
		} else if m, ok := result4.(map[string]interface{}); ok {
			getResult = m
		} else {
			t.Fatalf("Unexpected result type: %T", result4)
		}
		assert.Equal(t, "Integration Test Collection", getResult["name"])
		t.Logf("✓ Step 4: Retrieved collection details")

		// Step 5: Update metadata
		p5 := process.New("kb.collection.UpdateMetadata").WithContext(ctx).WithAuthorized(authInfo)
		p5.Args = []interface{}{
			testCollectionID,
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":        "Updated Integration Collection",
					"description": "Updated via process",
				},
			},
		}
		result5, err := p5.Exec()
		require.NoError(t, err)
		updateResult := result5.(maps.MapStrAny)
		assert.Equal(t, testCollectionID, updateResult["collection_id"])
		t.Logf("✓ Step 5: Updated collection metadata")

		// Step 6: Verify update
		p6 := process.New("kb.collection.Get").WithContext(ctx)
		p6.Args = []interface{}{testCollectionID}
		result6, err := p6.Exec()
		require.NoError(t, err)
		// GetCollection returns map[string]interface{}
		var getResult2 map[string]interface{}
		if mapStrAny, ok := result6.(maps.MapStrAny); ok {
			getResult2 = mapStrAny
		} else if m, ok := result6.(map[string]interface{}); ok {
			getResult2 = m
		} else {
			t.Fatalf("Unexpected result type: %T", result6)
		}
		assert.Equal(t, "Updated Integration Collection", getResult2["name"])
		t.Logf("✓ Step 6: Verified metadata update")

		// Step 7: List collections (should include ours)
		p7 := process.New("kb.collection.List").WithContext(ctx).WithAuthorized(authInfo)
		p7.Args = []interface{}{
			map[string]interface{}{
				"page":     1,
				"pagesize": 100,
			},
		}
		result7, err := p7.Exec()
		require.NoError(t, err)
		listResult := result7.(maps.MapStrAny)
		assert.Contains(t, listResult, "data")
		t.Logf("✓ Step 7: Listed collections")

		// Step 8: Remove collection
		p8 := process.New("kb.collection.Remove").WithContext(ctx).WithAuthorized(authInfo)
		p8.Args = []interface{}{testCollectionID}
		result8, err := p8.Exec()
		require.NoError(t, err)
		removeResult := result8.(maps.MapStrAny)
		assert.Equal(t, true, removeResult["removed"])
		t.Logf("✓ Step 8: Removed collection")

		// Step 9: Verify collection no longer exists
		p9 := process.New("kb.collection.Exists").WithContext(ctx)
		p9.Args = []interface{}{testCollectionID}
		result9, err := p9.Exec()
		require.NoError(t, err)
		existsResult3 := result9.(maps.MapStrAny)
		assert.Equal(t, false, existsResult3["exists"])
		t.Logf("✓ Step 9: Confirmed collection no longer exists")

		t.Logf("✅ Full lifecycle completed successfully via Process API")
	})
}
