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

// TestAddText tests the add text endpoint (sync)
func TestAddText(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB AddText Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test collection first
	testCollectionID := fmt.Sprintf("test_addtext_collection_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	createData := map[string]interface{}{
		"id": testCollectionID,
		"metadata": map[string]interface{}{
			"name":     "Test Collection for AddText",
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

	t.Run("AddTextInvalidRequest", func(t *testing.T) {
		// Test with missing required fields
		invalidData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing text, chunking, embedding
		}

		body, err := json.Marshal(invalidData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("AddTextMissingText", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing text
			"chunking": map[string]interface{}{
				"provider_id": "__yao.structured",
				"option_id":   "standard",
			},
			"embedding": map[string]interface{}{
				"provider_id": "__yao.openai",
				"option_id":   "text-embedding-3-small",
			},
		}

		body, err := json.Marshal(addData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		// Error message contains Text (case insensitive check)
		assert.Contains(t, response["error_description"], "Text")
	})

	t.Run("AddTextNonExistentCollection", func(t *testing.T) {
		// Test with a non-existent collection
		addData := map[string]interface{}{
			"collection_id": "non_existent_collection_12345",
			"text":          "This is a test text content for the knowledge base.",
			"chunking": map[string]interface{}{
				"provider_id": "__yao.structured",
				"option_id":   "standard",
			},
			"embedding": map[string]interface{}{
				"provider_id": "__yao.openai",
				"option_id":   "text-embedding-3-small",
			},
		}

		body, err := json.Marshal(addData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/non_existent_collection_12345/documents/text", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 403 Forbidden, 404 Not Found, or 500 Internal Server Error for non-existent collection
		// (depends on whether permission check or collection lookup happens first)
		assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusInternalServerError,
			"Expected 403, 404, or 500, got %d", resp.StatusCode)
	})

	t.Run("AddTextMissingChunking", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			"text":          "This is a test text content.",
			// Missing chunking
			"embedding": map[string]interface{}{
				"provider_id": "__yao.openai",
				"option_id":   "text-embedding-3-small",
			},
		}

		body, err := json.Marshal(addData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		// Error message contains Chunking (case insensitive check)
		assert.Contains(t, response["error_description"], "Chunking")
	})

	t.Run("AddTextMissingEmbedding", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			"text":          "This is a test text content.",
			"chunking": map[string]interface{}{
				"provider_id": "__yao.structured",
				"option_id":   "standard",
			},
			// Missing embedding
		}

		body, err := json.Marshal(addData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		// Error message contains Embedding (case insensitive check)
		assert.Contains(t, response["error_description"], "Embedding")
	})

	t.Run("AddTextUnauthorized", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			"text":          "This is a test text content.",
			"chunking": map[string]interface{}{
				"provider_id": "__yao.structured",
				"option_id":   "standard",
			},
			"embedding": map[string]interface{}{
				"provider_id": "__yao.openai",
				"option_id":   "text-embedding-3-small",
			},
		}

		body, err := json.Marshal(addData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestAddTextAsync tests the add text async endpoint
func TestAddTextAsync(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB AddTextAsync Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test collection first
	testCollectionID := fmt.Sprintf("test_addtext_async_collection_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	createData := map[string]interface{}{
		"id": testCollectionID,
		"metadata": map[string]interface{}{
			"name":     "Test Collection for AddTextAsync",
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

	t.Run("AddTextAsyncInvalidRequest", func(t *testing.T) {
		// Test with missing required fields
		invalidData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing text, chunking, embedding
		}

		body, err := json.Marshal(invalidData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text/async", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("AddTextAsyncMissingText", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing text
			"chunking": map[string]interface{}{
				"provider_id": "__yao.structured",
				"option_id":   "standard",
			},
			"embedding": map[string]interface{}{
				"provider_id": "__yao.openai",
				"option_id":   "text-embedding-3-small",
			},
		}

		body, err := json.Marshal(addData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text/async", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		// Error message contains Text (case insensitive check)
		assert.Contains(t, response["error_description"], "Text")
	})

	t.Run("AddTextAsyncUnauthorized", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			"text":          "This is a test text content for async processing.",
			"chunking": map[string]interface{}{
				"provider_id": "__yao.structured",
				"option_id":   "standard",
			},
			"embedding": map[string]interface{}{
				"provider_id": "__yao.openai",
				"option_id":   "text-embedding-3-small",
			},
		}

		body, err := json.Marshal(addData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/text/async", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
