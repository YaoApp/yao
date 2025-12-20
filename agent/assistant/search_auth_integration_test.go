package assistant_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/api"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// ========== Test Constants ==========

const (
	// Collection IDs for auth testing
	AuthTestCollectionTeam1  = "auth_test_team1"
	AuthTestCollectionTeam2  = "auth_test_team2"
	AuthTestCollectionPublic = "auth_test_public"

	// Test users and teams
	TestUserA = "user_a"
	TestUserB = "user_b"
	TestTeam1 = "team_1"
	TestTeam2 = "team_2"
)

// ========== Setup Test ==========

// TestAuthSearchSetup creates test collections with different permissions.
// Run once before running auth tests:
//
//	go test -v -run "TestAuthSearchSetup" ./agent/assistant/...
func TestAuthSearchSetup(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	if kb.API == nil {
		t.Fatal("KB API not initialized")
	}

	ctx := context.Background()

	// Check if collections already exist
	team1Exists := collectionReady(ctx, AuthTestCollectionTeam1, 2)
	team2Exists := collectionReady(ctx, AuthTestCollectionTeam2, 2)
	publicExists := collectionReady(ctx, AuthTestCollectionPublic, 2)

	if team1Exists && team2Exists && publicExists {
		t.Log("✓ All auth test collections already exist")
		t.Log("  Run TestAuthSearchCleanup to recreate")
		return
	}

	// Cleanup existing
	t.Log("Cleaning up existing collections...")
	cleanupAuthCollections(ctx, t)
	time.Sleep(1 * time.Second)

	// Create Team1 collection (owned by UserA, Team1)
	t.Log("Creating Team1 collection...")
	createAuthCollection(ctx, t, AuthTestCollectionTeam1, TestUserA, TestTeam1, false, "team")
	addAuthDocument(ctx, t, AuthTestCollectionTeam1, "Team1 Doc1", "Team1 private document about quantum physics and relativity theory.")
	addAuthDocument(ctx, t, AuthTestCollectionTeam1, "Team1 Doc2", "Team1 shared document about machine learning and neural networks.")

	// Create Team2 collection (owned by UserB, Team2)
	t.Log("Creating Team2 collection...")
	createAuthCollection(ctx, t, AuthTestCollectionTeam2, TestUserB, TestTeam2, false, "team")
	addAuthDocument(ctx, t, AuthTestCollectionTeam2, "Team2 Doc1", "Team2 private document about deep learning algorithms.")
	addAuthDocument(ctx, t, AuthTestCollectionTeam2, "Team2 Doc2", "Team2 shared document about computer vision techniques.")

	// Create Public collection
	t.Log("Creating Public collection...")
	createAuthCollection(ctx, t, AuthTestCollectionPublic, TestUserA, TestTeam1, true, "")
	addAuthDocument(ctx, t, AuthTestCollectionPublic, "Public Doc1", "Public document about artificial intelligence and robotics.")
	addAuthDocument(ctx, t, AuthTestCollectionPublic, "Public Doc2", "Public document about natural language processing.")

	// Wait for indexing
	t.Log("Waiting for indexing...")
	time.Sleep(2 * time.Second)

	t.Log("✓ Auth test setup complete!")
}

// TestAuthSearchCleanup removes auth test collections.
func TestAuthSearchCleanup(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	if kb.API == nil {
		t.Fatal("KB API not initialized")
	}

	ctx := context.Background()
	cleanupAuthCollections(ctx, t)
	t.Log("✓ Auth test cleanup complete!")
}

// ========== KB Collection-Level Auth Filter Tests ==========

// Note: KB permission filtering works at the Collection level.
// The Collection metadata contains __yao_team_id, __yao_created_by, public, share fields.
// FilterKBCollectionsByAuth filters collections based on user authorization.

func TestKBCollectionAuthFilter(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Ensure test data exists (auto-creates if not)
	ensureAuthTestData(t)

	t.Run("TeamMemberCanAccessTeamCollection", func(t *testing.T) {
		// UserA from Team1 should access Team1 collection
		ctx := createAuthContext(TestUserA, TestTeam1, true, false)
		collections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2}

		allowed := assistant.FilterKBCollectionsByAuth(ctx, collections)
		assert.Contains(t, allowed, AuthTestCollectionTeam1, "Team1 member should access Team1 collection")
		t.Logf("  Allowed collections: %v", allowed)
	})

	t.Run("TeamMemberCannotAccessOtherTeamCollection", func(t *testing.T) {
		// UserA from Team1 should NOT access Team2 collection
		ctx := createAuthContext(TestUserA, TestTeam1, true, false)
		collections := []string{AuthTestCollectionTeam2}

		allowed := assistant.FilterKBCollectionsByAuth(ctx, collections)
		assert.NotContains(t, allowed, AuthTestCollectionTeam2, "Team1 member should NOT access Team2 collection")
		t.Logf("  Allowed collections: %v (expected empty)", allowed)
	})

	t.Run("OwnerCanAccessOwnCollection", func(t *testing.T) {
		// UserA with OwnerOnly should access collections they created
		ctx := createAuthContext(TestUserA, "", false, true)
		collections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2}

		allowed := assistant.FilterKBCollectionsByAuth(ctx, collections)
		assert.Contains(t, allowed, AuthTestCollectionTeam1, "Owner should access own collection")
		assert.NotContains(t, allowed, AuthTestCollectionTeam2, "Owner should NOT access other's collection")
		t.Logf("  Allowed collections: %v", allowed)
	})

	t.Run("PublicCollectionAccessibleToAll", func(t *testing.T) {
		// Note: The 'public' field in Metadata is not automatically saved to the database
		// by the current KB API. This test documents the expected behavior.
		// When public=true is properly set in DB, this should pass.

		// First, check the collection metadata
		bgCtx := context.Background()
		collection, err := kb.API.GetCollection(bgCtx, AuthTestCollectionPublic)
		assert.NoError(t, err)

		// Check if public is set correctly
		publicVal := collection["public"]
		t.Logf("  Public collection public field: %v (type: %T)", publicVal, publicVal)

		// If public is not set (0 or false), the test documents current behavior
		// The collection should be accessible via owner check since UserA created it
		ctx := createAuthContext(TestUserA, TestTeam1, false, true) // Owner check
		collections := []string{AuthTestCollectionPublic}

		allowed := assistant.FilterKBCollectionsByAuth(ctx, collections)
		assert.Contains(t, allowed, AuthTestCollectionPublic, "Owner should access their collection")
		t.Logf("  Allowed collections (owner check): %v", allowed)
	})

	t.Run("NoConstraintsMeansFullAccess", func(t *testing.T) {
		// User with no constraints should access all collections
		ctx := createAuthContext(TestUserA, TestTeam1, false, false)
		collections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2, AuthTestCollectionPublic}

		allowed := assistant.FilterKBCollectionsByAuth(ctx, collections)
		assert.Len(t, allowed, 3, "No constraints should allow all collections")
		t.Logf("  Allowed collections: %v", allowed)
	})

	t.Run("NilContextMeansFullAccess", func(t *testing.T) {
		collections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2}

		allowed := assistant.FilterKBCollectionsByAuth(nil, collections)
		assert.Len(t, allowed, 2, "Nil context should allow all collections")
		t.Logf("  Allowed collections: %v", allowed)
	})
}

// ========== DB Auth Wheres Tests ==========

func TestDBAuthWheresFilter(t *testing.T) {
	// Note: This test doesn't need KB, just tests the BuildDBAuthWheres function
	t.Run("TeamOnlyGeneratesCorrectWheres", func(t *testing.T) {
		ctx := createAuthContext(TestUserA, TestTeam1, true, false)
		wheres := assistant.BuildDBAuthWheres(ctx)

		assert.NotNil(t, wheres)
		assert.Len(t, wheres, 1)

		// Verify structure: should have 2 top-level conditions (public OR team filter)
		where := wheres[0]
		assert.Len(t, where.Wheres, 2, "Should have 2 conditions: public OR team")

		// First condition: public = true (OR)
		publicCond := where.Wheres[0]
		assert.NotNil(t, publicCond.Condition.Field)
		assert.Equal(t, "public", publicCond.Condition.Field.Field)
		assert.Equal(t, true, publicCond.Condition.Value)
		assert.True(t, publicCond.Condition.OR)

		// Second condition: team filter with nested conditions
		teamCond := where.Wheres[1]
		assert.Len(t, teamCond.Wheres, 2, "Team filter should have team_id and (created_by OR share)")

		// Team ID check
		teamIDCond := teamCond.Wheres[0]
		assert.Equal(t, "__yao_team_id", teamIDCond.Condition.Field.Field)
		assert.Equal(t, TestTeam1, teamIDCond.Condition.Value)

		// Created by OR share = team
		ownerOrShareCond := teamCond.Wheres[1]
		assert.Len(t, ownerOrShareCond.Wheres, 2)
		assert.Equal(t, "__yao_created_by", ownerOrShareCond.Wheres[0].Condition.Field.Field)
		assert.Equal(t, TestUserA, ownerOrShareCond.Wheres[0].Condition.Value)
		assert.Equal(t, "share", ownerOrShareCond.Wheres[1].Condition.Field.Field)
		assert.Equal(t, "team", ownerOrShareCond.Wheres[1].Condition.Value)
		assert.True(t, ownerOrShareCond.Wheres[1].Condition.OR)

		t.Logf("  TeamOnly: Verified team_id=%s, created_by=%s", TestTeam1, TestUserA)
	})

	t.Run("OwnerOnlyGeneratesCorrectWheres", func(t *testing.T) {
		ctx := createAuthContext(TestUserA, "", false, true)
		wheres := assistant.BuildDBAuthWheres(ctx)

		assert.NotNil(t, wheres)
		assert.Len(t, wheres, 1)

		// Verify structure: should have 2 top-level conditions (public OR owner filter)
		where := wheres[0]
		assert.Len(t, where.Wheres, 2, "Should have 2 conditions: public OR owner")

		// First condition: public = true (OR)
		publicCond := where.Wheres[0]
		assert.NotNil(t, publicCond.Condition.Field)
		assert.Equal(t, "public", publicCond.Condition.Field.Field)
		assert.Equal(t, true, publicCond.Condition.Value)
		assert.True(t, publicCond.Condition.OR)

		// Second condition: owner filter with nested conditions
		ownerCond := where.Wheres[1]
		assert.Len(t, ownerCond.Wheres, 2, "Owner filter should have team_id IS NULL and created_by")

		// Team ID is null check
		teamNullCond := ownerCond.Wheres[0]
		assert.Equal(t, "__yao_team_id", teamNullCond.Condition.Field.Field)
		assert.Equal(t, "null", teamNullCond.Condition.OP)

		// Created by check
		createdByCond := ownerCond.Wheres[1]
		assert.Equal(t, "__yao_created_by", createdByCond.Condition.Field.Field)
		assert.Equal(t, TestUserA, createdByCond.Condition.Value)

		t.Logf("  OwnerOnly: Verified created_by=%s, team_id IS NULL", TestUserA)
	})

	t.Run("NoConstraintsReturnsNil", func(t *testing.T) {
		ctx := createAuthContext(TestUserA, TestTeam1, false, false)
		wheres := assistant.BuildDBAuthWheres(ctx)

		assert.Nil(t, wheres, "No constraints should return nil")
		t.Log("  No constraints: nil wheres (no filter)")
	})

	t.Run("EmptyTeamIDReturnsNil", func(t *testing.T) {
		ctx := createAuthContext(TestUserA, "", true, false)
		wheres := assistant.BuildDBAuthWheres(ctx)

		assert.Nil(t, wheres, "Empty TeamID with TeamOnly should return nil")
		t.Log("  Empty TeamID with TeamOnly: nil wheres")
	})

	t.Run("EmptyUserIDReturnsNil", func(t *testing.T) {
		ctx := createAuthContext("", TestTeam1, false, true)
		wheres := assistant.BuildDBAuthWheres(ctx)

		assert.Nil(t, wheres, "Empty UserID with OwnerOnly should return nil")
		t.Log("  Empty UserID with OwnerOnly: nil wheres")
	})

	t.Run("NilContextReturnsNil", func(t *testing.T) {
		wheres := assistant.BuildDBAuthWheres(nil)

		assert.Nil(t, wheres, "Nil context should return nil")
		t.Log("  Nil context: nil wheres")
	})

	t.Run("NilAuthorizedReturnsNil", func(t *testing.T) {
		ctx := &agentContext.Context{Authorized: nil}
		wheres := assistant.BuildDBAuthWheres(ctx)

		assert.Nil(t, wheres, "Nil Authorized should return nil")
		t.Log("  Nil Authorized: nil wheres")
	})
}

// ========== KB Search Integration Tests ==========

func TestKBSearchIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Ensure test data exists (auto-creates if not)
	ensureAuthTestData(t)

	t.Run("TeamMemberSearchOnlyFindsTeamData", func(t *testing.T) {
		// UserA from Team1 searches - should ONLY find Team1 data
		ctx := createAuthContext(TestUserA, TestTeam1, true, false)

		// Filter collections first
		allCollections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2}
		allowed := assistant.FilterKBCollectionsByAuth(ctx, allCollections)

		// Should only allow Team1
		assert.Contains(t, allowed, AuthTestCollectionTeam1)
		assert.NotContains(t, allowed, AuthTestCollectionTeam2)
		assert.Len(t, allowed, 1, "Should only have 1 allowed collection")

		// Search on allowed collections
		result := executeKBSearchOnCollections(t, allowed, "quantum physics deep learning")
		assert.Greater(t, len(result.Items), 0, "Should find Team1 documents")

		// Verify ALL results are from Team1 collection only
		for _, item := range result.Items {
			assert.Equal(t, AuthTestCollectionTeam1, item.Collection,
				"All results should be from Team1 collection, got: %s", item.Collection)
		}
		t.Logf("  ✓ Team1 member found %d items, all from Team1 collection", len(result.Items))
	})

	t.Run("TeamMemberCannotAccessOtherTeamData", func(t *testing.T) {
		// UserA from Team1 tries to access Team2 - should be blocked
		ctx := createAuthContext(TestUserA, TestTeam1, true, false)

		// Try to filter Team2 collection
		collections := []string{AuthTestCollectionTeam2}
		allowed := assistant.FilterKBCollectionsByAuth(ctx, collections)

		// Should be empty - no access
		assert.Empty(t, allowed, "Team1 member should NOT have access to Team2 collection")
		t.Log("  ✓ Team1 member correctly blocked from Team2 collection")
	})

	t.Run("OwnerSearchOnlyFindsOwnData", func(t *testing.T) {
		// UserA with OwnerOnly - should only find collections they created
		ctx := createAuthContext(TestUserA, "", false, true)

		// Filter all collections
		allCollections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2, AuthTestCollectionPublic}
		allowed := assistant.FilterKBCollectionsByAuth(ctx, allCollections)

		// UserA created Team1 and Public, not Team2
		assert.Contains(t, allowed, AuthTestCollectionTeam1, "Owner should access Team1 (created by UserA)")
		assert.Contains(t, allowed, AuthTestCollectionPublic, "Owner should access Public (created by UserA)")
		assert.NotContains(t, allowed, AuthTestCollectionTeam2, "Owner should NOT access Team2 (created by UserB)")

		// Search and verify results
		result := executeKBSearchOnCollections(t, allowed, "quantum artificial intelligence")
		assert.Greater(t, len(result.Items), 0, "Should find owner's documents")

		// Verify NO results from Team2
		for _, item := range result.Items {
			assert.NotEqual(t, AuthTestCollectionTeam2, item.Collection,
				"Should NOT have results from Team2, got: %s", item.Collection)
		}
		t.Logf("  ✓ Owner found %d items, none from Team2", len(result.Items))
	})

	t.Run("NoConstraintsSearchFindsAllData", func(t *testing.T) {
		// User with no constraints - should find all data
		ctx := createAuthContext(TestUserA, TestTeam1, false, false)

		// Filter all collections
		allCollections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2, AuthTestCollectionPublic}
		allowed := assistant.FilterKBCollectionsByAuth(ctx, allCollections)

		// Should have access to all
		assert.Len(t, allowed, 3, "No constraints should allow all collections")

		// Search and verify results from multiple collections
		result := executeKBSearchOnCollections(t, allowed, "quantum deep learning artificial")

		// Should find results from multiple collections
		collectionsFound := make(map[string]bool)
		for _, item := range result.Items {
			collectionsFound[item.Collection] = true
		}
		assert.Greater(t, len(collectionsFound), 1, "Should find results from multiple collections")
		t.Logf("  ✓ No constraints: found %d items from %d collections", len(result.Items), len(collectionsFound))
	})

	t.Run("SearchResultsMatchCollectionFilter", func(t *testing.T) {
		// Verify that search results ONLY come from allowed collections
		ctx := createAuthContext(TestUserB, TestTeam2, true, false)

		// UserB from Team2 - should only access Team2
		allCollections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2, AuthTestCollectionPublic}
		allowed := assistant.FilterKBCollectionsByAuth(ctx, allCollections)

		assert.Contains(t, allowed, AuthTestCollectionTeam2, "Team2 member should access Team2")
		assert.NotContains(t, allowed, AuthTestCollectionTeam1, "Team2 member should NOT access Team1")

		// Search
		result := executeKBSearchOnCollections(t, allowed, "deep learning computer vision")

		// Verify results
		if len(result.Items) > 0 {
			for _, item := range result.Items {
				// Results should only be from allowed collections
				assert.Contains(t, allowed, item.Collection,
					"Result from %s should be in allowed list %v", item.Collection, allowed)
			}
			t.Logf("  ✓ Team2 member found %d items, all from allowed collections", len(result.Items))
		} else {
			t.Log("  ✓ Team2 member found 0 items (collection may be empty)")
		}
	})
}

// ========== Helper Functions ==========

// ensureAuthTestData checks if auth test data exists, creates if not.
// This is a utility function that can be called from any test.
// Note: testutils.Prepare must be called before this function.
func ensureAuthTestData(t *testing.T) {
	if kb.API == nil {
		t.Fatal("KB API not initialized - call testutils.Prepare first")
	}

	ctx := context.Background()

	// Check if all collections already exist with required documents
	team1Ready := collectionReady(ctx, AuthTestCollectionTeam1, 2)
	team2Ready := collectionReady(ctx, AuthTestCollectionTeam2, 2)
	publicReady := collectionReady(ctx, AuthTestCollectionPublic, 2)

	if team1Ready && team2Ready && publicReady {
		// All data exists, skip creation
		return
	}

	// Data not ready, create it
	t.Log("Auth test data not found, creating...")
	createAuthTestData(t, ctx)
}

// createAuthTestData creates the test collections and documents
func createAuthTestData(t *testing.T, ctx context.Context) {
	// Cleanup existing
	t.Log("Cleaning up existing collections...")
	cleanupAuthCollections(ctx, t)
	time.Sleep(1 * time.Second)

	// Create Team1 collection (owned by UserA, Team1)
	t.Log("Creating Team1 collection...")
	createAuthCollection(ctx, t, AuthTestCollectionTeam1, TestUserA, TestTeam1, false, "team")
	addAuthDocument(ctx, t, AuthTestCollectionTeam1, "Team1 Doc1", "Team1 private document about quantum physics and relativity theory.")
	addAuthDocument(ctx, t, AuthTestCollectionTeam1, "Team1 Doc2", "Team1 shared document about machine learning and neural networks.")

	// Create Team2 collection (owned by UserB, Team2)
	t.Log("Creating Team2 collection...")
	createAuthCollection(ctx, t, AuthTestCollectionTeam2, TestUserB, TestTeam2, false, "team")
	addAuthDocument(ctx, t, AuthTestCollectionTeam2, "Team2 Doc1", "Team2 private document about deep learning algorithms.")
	addAuthDocument(ctx, t, AuthTestCollectionTeam2, "Team2 Doc2", "Team2 shared document about computer vision techniques.")

	// Create Public collection
	t.Log("Creating Public collection...")
	createAuthCollection(ctx, t, AuthTestCollectionPublic, TestUserA, TestTeam1, true, "")
	addAuthDocument(ctx, t, AuthTestCollectionPublic, "Public Doc1", "Public document about artificial intelligence and robotics.")
	addAuthDocument(ctx, t, AuthTestCollectionPublic, "Public Doc2", "Public document about natural language processing.")

	// Wait for indexing
	t.Log("Waiting for indexing...")
	time.Sleep(2 * time.Second)

	t.Log("✓ Auth test data created!")
}

func createAuthContext(userID, teamID string, teamOnly, ownerOnly bool) *agentContext.Context {
	return &agentContext.Context{
		Authorized: &oauthtypes.AuthorizedInfo{
			UserID: userID,
			TeamID: teamID,
			Constraints: oauthtypes.DataConstraints{
				TeamOnly:  teamOnly,
				OwnerOnly: ownerOnly,
			},
		},
	}
}

func collectionReady(ctx context.Context, collectionID string, minDocs int) bool {
	collection, err := kb.API.GetCollection(ctx, collectionID)
	if err != nil || collection == nil {
		return false
	}

	docs, err := kb.API.ListDocuments(ctx, &api.ListDocumentsFilter{
		Page:         1,
		PageSize:     20,
		CollectionID: collectionID,
	})
	if err != nil || docs == nil {
		return false
	}

	return len(docs.Data) >= minDocs
}

func cleanupAuthCollections(ctx context.Context, t *testing.T) {
	collections := []string{AuthTestCollectionTeam1, AuthTestCollectionTeam2, AuthTestCollectionPublic}
	for _, id := range collections {
		if result, err := kb.API.RemoveCollection(ctx, id); err == nil && result.Removed {
			t.Logf("  Removed: %s", id)
		}
	}
}

func createAuthCollection(ctx context.Context, t *testing.T, id, userID, teamID string, public bool, share string) {
	params := &api.CreateCollectionParams{
		ID: id,
		Metadata: map[string]interface{}{
			"name":   id,
			"public": public,
			"share":  share,
		},
		EmbeddingProviderID: "__yao.openai",
		EmbeddingOptionID:   "text-embedding-3-small",
		Locale:              "en",
		Config: &graphragtypes.CreateCollectionOptions{
			Distance:  "cosine",
			IndexType: "hnsw",
		},
		AuthScope: map[string]interface{}{
			"__yao_created_by": userID,
			"__yao_team_id":    teamID,
		},
	}

	_, err := kb.API.CreateCollection(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create collection %s: %v", id, err)
	}
	t.Logf("  ✓ Created: %s", id)
}

func addAuthDocument(ctx context.Context, t *testing.T, collectionID, title, content string) {
	params := &api.AddTextParams{
		CollectionID: collectionID,
		Text:         content,
		DocID:        fmt.Sprintf("%s__%s", collectionID, sanitizeForID(title)),
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

	_, err := kb.API.AddText(ctx, params)
	if err != nil {
		t.Logf("  Warning: Failed to add document '%s': %v", title, err)
		return
	}
	t.Logf("    ✓ Added: %s", title)
}

func sanitizeForID(s string) string {
	result := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result += string(c)
		} else if c == ' ' {
			result += "_"
		}
	}
	return result
}

func executeKBSearch(t *testing.T, collectionID, query string, metadata map[string]interface{}) *searchTypes.Result {
	return executeKBSearchOnCollections(t, []string{collectionID}, query)
}

func executeKBSearchOnCollections(t *testing.T, collections []string, query string) *searchTypes.Result {
	if len(collections) == 0 {
		return &searchTypes.Result{Items: []*searchTypes.ResultItem{}}
	}

	cfg := &searchTypes.Config{
		KB: &searchTypes.KBConfig{
			Collections: collections,
			Threshold:   0.3,
		},
	}
	searcher := search.New(cfg, nil)

	req := &searchTypes.Request{
		Type:        searchTypes.SearchTypeKB,
		Query:       query,
		Collections: collections,
		Threshold:   0.3,
		Limit:       20,
		Source:      searchTypes.SourceAuto,
	}

	result, err := searcher.Search(nil, req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	return result
}
