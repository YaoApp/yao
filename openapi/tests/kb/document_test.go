package openapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestListDocuments tests the document listing endpoint
func TestListDocuments(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB Document List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test collection first
	testCollectionID := fmt.Sprintf("test_doc_list_collection_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	createData := map[string]interface{}{
		"id": testCollectionID,
		"metadata": map[string]interface{}{
			"name":     "Test Collection for Document List",
			"category": "test",
		},
		"config": map[string]interface{}{
			"embedding_provider_id": "__yao.openai",
			"embedding_option_id":   "text-embedding-3-small",
			"locale":                "en",
			"index_type":            "hnsw",
			"distance":              "cosine",
		},
	}

	body, _ := json.Marshal(createData)
	req, _ := http.NewRequest("POST", serverURL+baseURL+"/kb/collections", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	resp.Body.Close()

	t.Run("ListDocumentsSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents?collection_id="+testCollectionID, nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Verify pagination fields exist
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "page")
		assert.Contains(t, response, "pagesize")
		assert.Contains(t, response, "total")
	})

	t.Run("ListDocumentsWithPagination", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents?collection_id="+testCollectionID+"&page=1&pagesize=10", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Verify pagination values
		assert.Equal(t, float64(1), response["page"])
		assert.Equal(t, float64(10), response["pagesize"])
	})

	t.Run("ListDocumentsWithStatusFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents?collection_id="+testCollectionID+"&status=completed", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ListDocumentsWithSort", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents?collection_id="+testCollectionID+"&sort=created_at+desc", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ListDocumentsUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents?collection_id="+testCollectionID, nil)
		assert.NoError(t, err)

		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestGetDocument tests the get document endpoint
func TestGetDocument(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB Document Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("GetDocumentNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents/non_existent_doc_id", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 404 Not Found
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetDocumentWithSelectFields", func(t *testing.T) {
		// This test verifies the select parameter works
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents/test_doc_id?select=id,name,status", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 404 since doc doesn't exist, but the request format is valid
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetDocumentUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/documents/test_doc_id", nil)
		assert.NoError(t, err)

		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestRemoveDocuments tests the remove documents endpoint
func TestRemoveDocuments(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB Document Remove Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("RemoveDocumentsMissingIDs", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/kb/documents", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request for missing document_ids
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Contains(t, response["error_description"], "document_ids")
	})

	t.Run("RemoveDocumentsEmptyIDs", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/kb/documents?document_ids=", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request for empty document_ids
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("RemoveDocumentsNonExistent", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/kb/documents?document_ids=non_existent_doc_1,non_existent_doc_2", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// The behavior depends on implementation - could be 200 with 0 removed or 404
		// Accept either as valid
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden)
	})

	t.Run("RemoveDocumentsUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/kb/documents?document_ids=doc1,doc2", nil)
		assert.NoError(t, err)

		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
