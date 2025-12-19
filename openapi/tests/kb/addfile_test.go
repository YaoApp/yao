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

// TestAddFile tests the add file endpoint (sync)
func TestAddFile(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB AddFile Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test collection first
	testCollectionID := fmt.Sprintf("test_addfile_collection_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	createData := map[string]interface{}{
		"id": testCollectionID,
		"metadata": map[string]interface{}{
			"name":     "Test Collection for AddFile",
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

	t.Run("AddFileInvalidRequest", func(t *testing.T) {
		// Test with missing required fields
		invalidData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing file_id, chunking, embedding
		}

		body, err := json.Marshal(invalidData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/file", bytes.NewBuffer(body))
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

	t.Run("AddFileMissingFileID", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing file_id
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

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/file", bytes.NewBuffer(body))
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
		// Error message contains FileID (case insensitive check)
		assert.Contains(t, response["error_description"], "FileID")
	})

	t.Run("AddFileNonExistentCollection", func(t *testing.T) {
		// Test with a non-existent collection
		addData := map[string]interface{}{
			"collection_id": "non_existent_collection_12345",
			"file_id":       "test_file_123",
			"uploader":      "local",
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

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/non_existent_collection_12345/documents/file", bytes.NewBuffer(body))
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

	t.Run("AddFileUnauthorized", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			"file_id":       "test_file_123",
			"uploader":      "local",
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

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/file", bytes.NewBuffer(body))
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

// TestAddFileAsync tests the add file async endpoint
func TestAddFileAsync(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "KB AddFileAsync Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test collection first
	testCollectionID := fmt.Sprintf("test_addfile_async_collection_%d", time.Now().UnixNano())
	testutils.RegisterTestCollection(testCollectionID)

	createData := map[string]interface{}{
		"id": testCollectionID,
		"metadata": map[string]interface{}{
			"name":     "Test Collection for AddFileAsync",
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

	t.Run("AddFileAsyncInvalidRequest", func(t *testing.T) {
		// Test with missing required fields
		invalidData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing file_id, chunking, embedding
		}

		body, err := json.Marshal(invalidData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/file/async", bytes.NewBuffer(body))
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

	t.Run("AddFileAsyncMissingFileID", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			// Missing file_id
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

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/file/async", bytes.NewBuffer(body))
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
		// Error message contains FileID (case insensitive check)
		assert.Contains(t, response["error_description"], "FileID")
	})

	t.Run("AddFileAsyncUnauthorized", func(t *testing.T) {
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			"file_id":       "test_file_123",
			"uploader":      "local",
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

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/file/async", bytes.NewBuffer(body))
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

	t.Run("AddFileAsyncFileNotFound", func(t *testing.T) {
		// Test with a file_id that doesn't exist
		addData := map[string]interface{}{
			"collection_id": testCollectionID,
			"file_id":       "non_existent_file_12345",
			"uploader":      "local",
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

		req, err := http.NewRequest("POST", serverURL+baseURL+"/kb/collections/"+testCollectionID+"/documents/file/async", bytes.NewBuffer(body))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 404 Not Found for non-existent file
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
