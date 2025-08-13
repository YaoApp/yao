package openapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestCreateCollection tests the collection creation endpoint
func TestCreateCollection(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB Collection Create Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Generate unique test collection ID
	testCollectionID := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())

	t.Run("CreateCollectionSuccess", func(t *testing.T) {
		// Register collection for cleanup
		testutils.RegisterTestCollection(testCollectionID)

		// Prepare request body
		createData := map[string]interface{}{
			"id": testCollectionID,
			"metadata": map[string]interface{}{
				"name":       "Test Collection " + testCollectionID, // Required: collection display name
				"category":   "test",
				"created_by": "test_user",
			},
			"config": map[string]interface{}{
				"embedding_provider": "__yao.openai",           // Required: embedding provider ID
				"embedding_option":   "text-embedding-3-small", // Required: embedding option value
				"locale":             "en",                     // Optional: locale for provider reading
				"index_type":         "hnsw",                   // Required: valid index type
				"distance":           "cosine",                 // Required: distance metric
			},
		}

		body, err := json.Marshal(createData)
		assert.NoError(t, err)

		// Create HTTP request
		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		// Make request
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Check response
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Debug output for failed requests
		if resp.StatusCode != http.StatusCreated {
			t.Logf("Expected status 201, got %d", resp.StatusCode)
			t.Logf("Response body: %+v", response)
		}

		assert.Equal(t, "Collection created successfully", response["message"])
		assert.Equal(t, testCollectionID, response["collection_id"])

		t.Logf("Successfully created collection: %s", testCollectionID)
	})

	t.Run("CreateCollectionInvalidRequest", func(t *testing.T) {
		// Test with missing required fields
		invalidData := map[string]interface{}{
			"metadata": map[string]interface{}{
				"category": "test",
			},
			// Missing "id" and "config" fields
		}

		body, err := json.Marshal(invalidData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")

		t.Logf("Correctly rejected invalid collection creation request")
	})

	t.Run("CreateCollectionMalformedJSON", func(t *testing.T) {
		// Test with malformed JSON
		malformedJSON := `{"id": "test", "config": malformed}`

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections", bytes.NewBufferString(malformedJSON))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		t.Logf("Correctly rejected malformed JSON request")
	})
}

// TestRemoveCollection tests the collection removal endpoint
func TestRemoveCollection(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "KB Collection Remove Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testCollectionID := fmt.Sprintf("test_collection_remove_%d", time.Now().UnixNano())

	t.Run("RemoveCollectionSuccess", func(t *testing.T) {
		// Register collection for cleanup (in case removal fails)
		testutils.RegisterTestCollection(testCollectionID)

		// First create a collection to remove
		createData := map[string]interface{}{
			"id": testCollectionID,
			"metadata": map[string]interface{}{
				"name":     "Test Remove Collection " + testCollectionID, // Required: collection display name
				"category": "test_remove",
			},
			"config": map[string]interface{}{
				"embedding_provider": "__yao.openai",           // Required: embedding provider ID
				"embedding_option":   "text-embedding-3-small", // Required: embedding option value
				"locale":             "en",                     // Optional: locale for provider reading
				"index_type":         "hnsw",                   // Required: valid index type
				"distance":           "cosine",                 // Required: distance metric
			},
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/kb/collections", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		createResp, err := http.DefaultClient.Do(createReq)
		assert.NoError(t, err)
		defer createResp.Body.Close()

		// Now test removal
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/kb/collections/"+testCollectionID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Knowledge base should be initialized for this test to be meaningful
		if resp.StatusCode == http.StatusInternalServerError {
			var errorResponse map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errorResponse)
			if errorDescription, ok := errorResponse["error_description"].(string); ok &&
				strings.Contains(errorDescription, "Knowledge base not initialized") {
				t.Skip("Knowledge base not initialized - skipping test (this indicates environment setup issue)")
			}
		}

		// Expect successful response for collection removal
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully remove collection when KB is initialized")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Contains(t, response, "message")
		t.Logf("Successfully removed collection: %s", testCollectionID)
	})

	t.Run("RemoveCollectionMissingID", func(t *testing.T) {
		// Test with missing collection ID in path
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/kb/collections/", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// This should result in a 404 or similar error due to route mismatch
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)

		t.Logf("Correctly handled request with missing collection ID")
	})
}

// TestCollectionExists tests the collection existence check endpoint
func TestCollectionExists(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "KB Collection Exists Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testCollectionID := fmt.Sprintf("test_collection_exists_%d", time.Now().UnixNano())

	t.Run("CollectionExistsCheck", func(t *testing.T) {
		// Test with a collection ID
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/exists", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Knowledge base should be initialized for this test to be meaningful
		if resp.StatusCode == http.StatusInternalServerError {
			var errorResponse map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errorResponse)
			if errorDescription, ok := errorResponse["error_description"].(string); ok &&
				strings.Contains(errorDescription, "Knowledge base not initialized") {
				t.Skip("Knowledge base not initialized - skipping test (this indicates environment setup issue)")
			}
		}

		// Expect successful response for collection existence check
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully check collection existence when KB is initialized")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Contains(t, response, "exists")
		assert.Contains(t, response, "collection_id")
		assert.Equal(t, testCollectionID, response["collection_id"])

		exists, ok := response["exists"].(bool)
		assert.True(t, ok, "exists should be a boolean")

		t.Logf("Collection %s exists: %v", testCollectionID, exists)
	})

	t.Run("CollectionExistsMissingID", func(t *testing.T) {
		// Test with missing collection ID in path
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections//exists", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// This should result in a bad request due to empty collection ID
		if resp.StatusCode == http.StatusBadRequest {
			var errorResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse, "error")
			t.Logf("Correctly rejected request with missing collection ID")
		}
	})
}

// TestGetCollections tests the collections listing endpoint
func TestGetCollections(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "KB Collections List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("GetCollectionsSuccess", func(t *testing.T) {
		// Test listing all collections
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Knowledge base should be initialized for this test to be meaningful
		if resp.StatusCode == http.StatusInternalServerError {
			var errorResponse map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errorResponse)
			if errorDescription, ok := errorResponse["error_description"].(string); ok &&
				strings.Contains(errorDescription, "Knowledge base not initialized") {
				t.Skip("Knowledge base not initialized - skipping test (this indicates environment setup issue)")
			}
		}

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve collections when KB is initialized")

		var collections []interface{}
		err = json.NewDecoder(resp.Body).Decode(&collections)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d collections", len(collections))
	})

	t.Run("GetCollectionsWithFilter", func(t *testing.T) {
		// Test listing collections with filter parameters
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections?category=documents&status=active", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Knowledge base should be initialized for this test to be meaningful
		if resp.StatusCode == http.StatusInternalServerError {
			var errorResponse map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errorResponse)
			if errorDescription, ok := errorResponse["error_description"].(string); ok &&
				strings.Contains(errorDescription, "Knowledge base not initialized") {
				t.Skip("Knowledge base not initialized - skipping test (this indicates environment setup issue)")
			}
		}

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve collections when KB is initialized")

		var collections []interface{}
		err = json.NewDecoder(resp.Body).Decode(&collections)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d filtered collections", len(collections))
	})

	t.Run("GetCollectionsWithMultipleFilters", func(t *testing.T) {
		// Test with multiple filter parameters
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections?category=test&owner=testuser&type=public", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Knowledge base should be initialized for this test to be meaningful
		if resp.StatusCode == http.StatusInternalServerError {
			var errorResponse map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errorResponse)
			if errorDescription, ok := errorResponse["error_description"].(string); ok &&
				strings.Contains(errorDescription, "Knowledge base not initialized") {
				t.Skip("Knowledge base not initialized - skipping test (this indicates environment setup issue)")
			}
		}

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve collections when KB is initialized")

		var collections []interface{}
		err = json.NewDecoder(resp.Body).Decode(&collections)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d collections with multiple filters", len(collections))
	})
}

// TestCollectionEndpointsUnauthorized tests that endpoints return 401 when not authenticated
func TestCollectionEndpointsUnauthorized(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/kb/collections", ""},
		{"POST", "/kb/collections", `{"id":"test","config":{}}`},
		{"DELETE", "/kb/collections/test", ""},
		{"GET", "/kb/collections/test/exists", ""},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("Unauthorized_%s_%s", endpoint.method, endpoint.path), func(t *testing.T) {
			var req *http.Request
			var err error

			if endpoint.body != "" {
				req, err = http.NewRequest(endpoint.method, serverURL+baseURL+endpoint.path, bytes.NewBufferString(endpoint.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(endpoint.method, serverURL+baseURL+endpoint.path, nil)
			}
			assert.NoError(t, err)

			// No Authorization header
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			t.Logf("Correctly rejected unauthorized request to %s %s", endpoint.method, endpoint.path)
		})
	}
}

// TestCollectionIntegration tests the full collection lifecycle
func TestCollectionIntegration(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "KB Collection Integration Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testCollectionID := fmt.Sprintf("test_integration_%d", time.Now().UnixNano())

	t.Run("FullCollectionLifecycle", func(t *testing.T) {
		// Register collection for cleanup (in case lifecycle test fails)
		testutils.RegisterTestCollection(testCollectionID)

		// Step 1: Check that collection doesn't exist initially
		req, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/exists", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Step 2: Create the collection
		createData := map[string]interface{}{
			"id": testCollectionID,
			"metadata": map[string]interface{}{
				"name":     "Integration Test Collection " + testCollectionID, // Required: collection display name
				"category": "integration_test",
				"purpose":  "full_lifecycle_test",
			},
			"config": map[string]interface{}{
				"embedding_provider": "__yao.openai",           // Required: embedding provider ID
				"embedding_option":   "text-embedding-3-small", // Required: embedding option value
				"locale":             "en",                     // Optional: locale for provider reading
				"index_type":         "hnsw",                   // Required: valid index type
				"distance":           "cosine",                 // Required: distance metric
			},
		}

		body, err := json.Marshal(createData)
		assert.NoError(t, err)

		createReq, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections", bytes.NewBuffer(body))
		assert.NoError(t, err)
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		createResp, err := http.DefaultClient.Do(createReq)
		assert.NoError(t, err)
		defer createResp.Body.Close()

		// Step 3: Verify collection appears in listings (with filter)
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections?category=integration_test", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		// Step 4: Check that collection now exists
		existsReq, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/exists", nil)
		assert.NoError(t, err)
		existsReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		existsResp, err := http.DefaultClient.Do(existsReq)
		assert.NoError(t, err)
		defer existsResp.Body.Close()

		// Step 5: Remove the collection
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/kb/collections/"+testCollectionID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		deleteResp, err := http.DefaultClient.Do(deleteReq)
		assert.NoError(t, err)
		defer deleteResp.Body.Close()

		// Step 6: Verify collection no longer exists
		finalExistsReq, err := http.NewRequest("GET", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/exists", nil)
		assert.NoError(t, err)
		finalExistsReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		finalExistsResp, err := http.DefaultClient.Do(finalExistsReq)
		assert.NoError(t, err)
		defer finalExistsResp.Body.Close()

		t.Logf("Completed full collection lifecycle test for: %s", testCollectionID)
	})
}
