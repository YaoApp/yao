package openapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/assistant"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// =============================================================================
// Test Setup Helpers
// =============================================================================

// createTestSearch creates a test search record in the database
func createTestSearch(t *testing.T, requestID, chatID, query, source string, refs []storetypes.Reference) {
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not initialized")
	}

	search := &storetypes.Search{
		RequestID:  requestID,
		ChatID:     chatID,
		Query:      query,
		Source:     source,
		Duration:   100,
		References: refs,
		CreatedAt:  time.Now(),
	}

	err := chatStore.SaveSearch(search)
	if err != nil {
		t.Fatalf("Failed to create test search: %v", err)
	}

	t.Logf("Created test search: request_id=%s, query=%s", requestID, query)
}

// cleanupTestSearches deletes test search records
func cleanupTestSearches(t *testing.T, chatID string) {
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		return
	}

	err := chatStore.DeleteSearches(chatID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test searches for chat %s: %v", chatID, err)
	} else {
		t.Logf("Cleaned up test searches for chat: %s", chatID)
	}
}

// =============================================================================
// Get References Tests
// =============================================================================

// TestGetReferences tests the get all references endpoint
func TestGetReferences(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Reference Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create test chat
	chatID := createTestChat(t, "Reference Test Chat", "test-assistant")
	defer cleanupTestChat(t, chatID)

	requestID := fmt.Sprintf("req_%s", uuid.New().String())

	// Create test search with references
	refs := []storetypes.Reference{
		{Index: 1, Type: "web", Title: "Go Documentation", URL: "https://golang.org/doc/", Snippet: "Go is an open source programming language", Content: "Full content 1"},
		{Index: 2, Type: "web", Title: "Go by Example", URL: "https://gobyexample.com/", Snippet: "Go by Example is a hands-on introduction", Content: "Full content 2"},
	}
	createTestSearch(t, requestID, chatID, "golang documentation", "web", refs)
	defer cleanupTestSearches(t, chatID)

	// Create second search with more references
	refs2 := []storetypes.Reference{
		{Index: 3, Type: "kb", Title: "Internal Doc", Snippet: "Internal documentation snippet", Content: "Full content 3"},
	}
	createTestSearch(t, requestID, chatID, "internal docs", "kb", refs2)

	t.Run("GetAllReferences", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.Equal(t, requestID, result["request_id"])
		assert.Equal(t, float64(3), result["total"])

		references := result["references"].([]interface{})
		assert.Len(t, references, 3)

		// Check first reference
		ref1 := references[0].(map[string]interface{})
		assert.Equal(t, float64(1), ref1["index"])
		assert.Equal(t, "web", ref1["type"])
		assert.Equal(t, "Go Documentation", ref1["title"])
		assert.Equal(t, "https://golang.org/doc/", ref1["url"])

		// Check third reference (from second search)
		ref3 := references[2].(map[string]interface{})
		assert.Equal(t, float64(3), ref3["index"])
		assert.Equal(t, "kb", ref3["type"])
		assert.Equal(t, "Internal Doc", ref3["title"])

		t.Logf("Successfully retrieved %d references", len(references))
	})

	t.Run("GetReferences_NotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/non_existent_request_id", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 200 with empty references
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.Equal(t, float64(0), result["total"])
		t.Log("Non-existent request returns empty references as expected")
	})

	t.Run("GetReferences_Unauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID, nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		t.Log("Unauthorized request rejected as expected")
	})
}

// TestGetReference tests the get single reference endpoint
func TestGetReference(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Single Reference Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create test chat
	chatID := createTestChat(t, "Single Reference Test Chat", "test-assistant")
	defer cleanupTestChat(t, chatID)

	requestID := fmt.Sprintf("req_%s", uuid.New().String())

	// Create test search with references
	refs := []storetypes.Reference{
		{Index: 1, Type: "web", Title: "First Reference", URL: "https://example.com/1", Snippet: "First snippet", Content: "First content"},
		{Index: 2, Type: "kb", Title: "Second Reference", Snippet: "Second snippet", Content: "Second content"},
		{Index: 3, Type: "db", Title: "Third Reference", Snippet: "Third snippet", Content: "Third content"},
	}
	createTestSearch(t, requestID, chatID, "test query", "web", refs)
	defer cleanupTestSearches(t, chatID)

	t.Run("GetSingleReference", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/2", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ref map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&ref)
		assert.NoError(t, err)

		assert.Equal(t, float64(2), ref["index"])
		assert.Equal(t, "kb", ref["type"])
		assert.Equal(t, "Second Reference", ref["title"])
		assert.Equal(t, "Second snippet", ref["snippet"])
		assert.Equal(t, "Second content", ref["content"])

		t.Logf("Successfully retrieved reference at index 2")
	})

	t.Run("GetReference_FirstIndex", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/1", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ref map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&ref)
		assert.NoError(t, err)

		assert.Equal(t, float64(1), ref["index"])
		assert.Equal(t, "web", ref["type"])
		assert.Equal(t, "First Reference", ref["title"])

		t.Log("Successfully retrieved first reference")
	})

	t.Run("GetReference_NotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/999", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		t.Log("Non-existent reference returns 404 as expected")
	})

	t.Run("GetReference_InvalidIndex", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/invalid", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		t.Log("Invalid index returns 400 as expected")
	})

	t.Run("GetReference_ZeroIndex", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/0", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		t.Log("Zero index returns 400 as expected")
	})

	t.Run("GetReference_NegativeIndex", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/-1", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		t.Log("Negative index returns 400 as expected")
	})

	t.Run("GetReference_Unauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/1", nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		t.Log("Unauthorized request rejected as expected")
	})
}

// TestGetReferences_MultipleSearches tests references aggregation from multiple searches
func TestGetReferences_MultipleSearches(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Multiple Searches Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create test chat
	chatID := createTestChat(t, "Multiple Searches Test Chat", "test-assistant")
	defer cleanupTestChat(t, chatID)

	requestID := fmt.Sprintf("req_%s", uuid.New().String())

	// Create first search (web)
	refs1 := []storetypes.Reference{
		{Index: 1, Type: "web", Title: "Web Result 1", URL: "https://example.com/1"},
		{Index: 2, Type: "web", Title: "Web Result 2", URL: "https://example.com/2"},
	}
	createTestSearch(t, requestID, chatID, "web search query", "web", refs1)

	// Create second search (kb)
	refs2 := []storetypes.Reference{
		{Index: 3, Type: "kb", Title: "KB Result 1"},
		{Index: 4, Type: "kb", Title: "KB Result 2"},
	}
	createTestSearch(t, requestID, chatID, "kb search query", "kb", refs2)

	// Create third search (db)
	refs3 := []storetypes.Reference{
		{Index: 5, Type: "db", Title: "DB Result 1"},
	}
	createTestSearch(t, requestID, chatID, "db search query", "db", refs3)

	defer cleanupTestSearches(t, chatID)

	t.Run("AggregatedReferences", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.Equal(t, float64(5), result["total"])

		references := result["references"].([]interface{})
		assert.Len(t, references, 5)

		// Verify all types are present
		types := make(map[string]int)
		for _, r := range references {
			ref := r.(map[string]interface{})
			refType := ref["type"].(string)
			types[refType]++
		}

		assert.Equal(t, 2, types["web"])
		assert.Equal(t, 2, types["kb"])
		assert.Equal(t, 1, types["db"])

		t.Logf("Successfully aggregated references: web=%d, kb=%d, db=%d", types["web"], types["kb"], types["db"])
	})

	t.Run("GetSpecificReference", func(t *testing.T) {
		// Get reference from second search
		req, err := http.NewRequest("GET", serverURL+baseURL+"/chat/references/"+requestID+"/4", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ref map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&ref)
		assert.NoError(t, err)

		assert.Equal(t, float64(4), ref["index"])
		assert.Equal(t, "kb", ref["type"])
		assert.Equal(t, "KB Result 2", ref["title"])

		t.Log("Successfully retrieved specific reference from aggregated searches")
	})
}
