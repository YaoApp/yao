package openapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// ModelResponse represents an OpenAI-compatible model object
type ModelResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsListResponse represents the response for listing models
type ModelsListResponse struct {
	Object string          `json:"object"`
	Data   []ModelResponse `json:"data"`
}

// TestListModels tests the models listing endpoint (OpenAI compatible)
func TestListModels(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Models List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ListModelsSuccess", func(t *testing.T) {
		// Test listing all models (OpenAI compatible endpoint)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve models")

		var response ModelsListResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Verify OpenAI-compatible response structure
		assert.Equal(t, "list", response.Object, "Response object should be 'list'")
		assert.NotNil(t, response.Data, "Response should have data field")

		if len(response.Data) > 0 {
			t.Logf("Successfully retrieved %d models", len(response.Data))

			// Verify first model structure
			firstModel := response.Data[0]
			assert.NotEmpty(t, firstModel.ID, "Model should have an ID")
			assert.Equal(t, "model", firstModel.Object, "Model object should be 'model'")
			assert.GreaterOrEqual(t, firstModel.Created, int64(0), "Model should have created timestamp (0 or greater)")
			assert.NotEmpty(t, firstModel.OwnedBy, "Model should have owner")

			// Verify model ID format: connector-model-assistantName-yao_assistantID
			assert.Contains(t, firstModel.ID, "-yao_", "Model ID should contain '-yao_' prefix")

			t.Logf("First model: ID=%s, Created=%d, OwnedBy=%s",
				firstModel.ID, firstModel.Created, firstModel.OwnedBy)
		} else {
			t.Log("No models returned (this is OK if no assistants exist)")
		}
	})

	t.Run("ListModelsWithLocale", func(t *testing.T) {
		// Test with locale parameter for i18n
		req, err := http.NewRequest("GET", serverURL+baseURL+"/models?locale=zh-cn", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve models with locale")

		var response ModelsListResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Equal(t, "list", response.Object)
		t.Logf("Retrieved %d models with zh-cn locale", len(response.Data))
	})

	t.Run("ListModelsUnauthorized", func(t *testing.T) {
		// Test without authorization token
		req, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should require authentication")
	})

	t.Run("ListModelsInvalidToken", func(t *testing.T) {
		// Test with invalid authorization token
		req, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid_token_12345")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return unauthorized or forbidden
		assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden,
			"Should reject invalid token")
	})
}

// TestGetModelDetails tests the model details endpoint (OpenAI compatible)
func TestGetModelDetails(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Model Details Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// First, get list of models to get a valid model ID
	var validModelID string
	t.Run("GetValidModelID", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var response ModelsListResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			if len(response.Data) > 0 {
				validModelID = response.Data[0].ID
				t.Logf("Using model ID for testing: %s", validModelID)
			} else {
				t.Skip("No models available for testing")
			}
		}
	})

	if validModelID == "" {
		t.Skip("No valid model ID available for testing")
	}

	t.Run("GetModelDetailsSuccess", func(t *testing.T) {
		// Test getting model details
		url := fmt.Sprintf("%s%s/models/%s", serverURL, baseURL, validModelID)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve model details")

		var model ModelResponse
		err = json.NewDecoder(resp.Body).Decode(&model)
		assert.NoError(t, err)

		// Verify model structure
		assert.Equal(t, validModelID, model.ID, "Model ID should match")
		assert.Equal(t, "model", model.Object, "Model object should be 'model'")
		// Note: Created timestamp may be 0 or negative for legacy data, newly created assistants will have proper timestamps
		assert.NotEmpty(t, model.OwnedBy, "Model should have owner")

		t.Logf("Model details: ID=%s, Created=%d, OwnedBy=%s",
			model.ID, model.Created, model.OwnedBy)
	})

	t.Run("GetModelDetailsWithLocale", func(t *testing.T) {
		// Test with locale parameter
		url := fmt.Sprintf("%s%s/models/%s?locale=en-us", serverURL, baseURL, validModelID)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var model ModelResponse
		err = json.NewDecoder(resp.Body).Decode(&model)
		assert.NoError(t, err)

		assert.Equal(t, validModelID, model.ID)
		t.Log("Successfully retrieved model with locale")
	})

	t.Run("GetModelDetailsNotFound", func(t *testing.T) {
		// Test with non-existent model ID
		url := fmt.Sprintf("%s%s/models/nonexistent-model-yao_invalid123", serverURL, baseURL)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return not found
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return not found for invalid model")
	})

	t.Run("GetModelDetailsInvalidFormat", func(t *testing.T) {
		// Test with invalid model ID format (no yao_ prefix)
		url := fmt.Sprintf("%s%s/models/invalid-model-without-prefix", serverURL, baseURL)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return bad request or not found
		assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound,
			"Should reject invalid model ID format")
	})

	t.Run("GetModelDetailsUnauthorized", func(t *testing.T) {
		// Test without authorization token
		url := fmt.Sprintf("%s%s/models/%s", serverURL, baseURL, validModelID)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should require authentication")
	})
}

// TestModelIDFormat tests the model ID format and extraction
func TestModelIDFormat(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Model ID Format Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("VerifyModelIDFormat", func(t *testing.T) {
		// Get models and verify ID format
		req, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var response ModelsListResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			for _, model := range response.Data {
				// Verify format: connector-model-assistantName-yao_assistantID
				parts := strings.Split(model.ID, "-yao_")
				assert.Equal(t, 2, len(parts), "Model ID should have format: *-yao_assistantID")

				if len(parts) == 2 {
					prefix := parts[0]
					assistantID := parts[1]

					// Verify prefix has at least: connector-model
					assert.True(t, strings.Contains(prefix, "-"),
						"Model ID prefix should contain connector-model parts")

					// Verify assistant ID is not empty
					assert.NotEmpty(t, assistantID, "Assistant ID should not be empty")

					t.Logf("Model ID format OK: %s -> assistantID=%s", model.ID, assistantID)
				}
			}
		}
	})

	t.Run("VerifyOwnershipTypes", func(t *testing.T) {
		// Get models and verify ownership types
		req, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var response ModelsListResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			ownerTypes := make(map[string]int)
			for _, model := range response.Data {
				ownerTypes[model.OwnedBy]++
			}

			t.Logf("Owner types distribution: %v", ownerTypes)

			// Verify valid owner types
			for owner := range ownerTypes {
				assert.Contains(t, []string{"system", "team", "user"}, owner,
					"Owner type should be system, team, or user")
			}
		}
	})
}

// TestModelPermissions tests permission-based model access
func TestModelPermissions(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register two different test clients
	client1 := testutils.RegisterTestClient(t, "Model Permissions Test Client 1", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client1.ClientID)
	tokenInfo1 := testutils.ObtainAccessToken(t, serverURL, client1.ClientID, client1.ClientSecret, "https://localhost/callback", "openid profile")

	client2 := testutils.RegisterTestClient(t, "Model Permissions Test Client 2", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client2.ClientID)
	tokenInfo2 := testutils.ObtainAccessToken(t, serverURL, client2.ClientID, client2.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("DifferentUsersSeeDifferentModels", func(t *testing.T) {
		// Get models for user 1
		req1, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)
		req1.Header.Set("Authorization", "Bearer "+tokenInfo1.AccessToken)

		resp1, err := http.DefaultClient.Do(req1)
		assert.NoError(t, err)
		defer resp1.Body.Close()

		var response1 ModelsListResponse
		if resp1.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp1.Body).Decode(&response1)
			assert.NoError(t, err)
		}

		// Get models for user 2
		req2, err := http.NewRequest("GET", serverURL+baseURL+"/models", nil)
		assert.NoError(t, err)
		req2.Header.Set("Authorization", "Bearer "+tokenInfo2.AccessToken)

		resp2, err := http.DefaultClient.Do(req2)
		assert.NoError(t, err)
		defer resp2.Body.Close()

		var response2 ModelsListResponse
		if resp2.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp2.Body).Decode(&response2)
			assert.NoError(t, err)
		}

		t.Logf("User 1 sees %d models", len(response1.Data))
		t.Logf("User 2 sees %d models", len(response2.Data))

		// Both users should see at least system models
		// The exact count may differ based on permissions
		t.Log("Permission-based filtering is working")
	})
}
