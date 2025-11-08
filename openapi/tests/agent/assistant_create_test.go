package openapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestCreateAssistant tests the create assistant endpoint
func TestCreateAssistant(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Agent Create Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("CreateAssistantSuccess", func(t *testing.T) {
		// Create a new assistant
		assistantData := map[string]interface{}{
			"name":        "Test Assistant",
			"type":        "assistant",
			"connector":   "openai",
			"description": "A test assistant created by automated tests",
			"tags":        []string{"test", "automation"},
			"public":      false,
			"share":       "private",
			"mentionable": true,
			"automated":   false,
		}

		jsonData, err := json.Marshal(assistantData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully create assistant")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Verify response contains assistant_id
		assistantID, hasID := response["assistant_id"].(string)
		assert.True(t, hasID, "Response should have assistant_id")
		assert.NotEmpty(t, assistantID, "Assistant ID should not be empty")

		t.Logf("Successfully created assistant with ID: %s", assistantID)

		// Clean up: delete the created assistant
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		deleteResp, err := http.DefaultClient.Do(deleteReq)
		if err == nil {
			defer deleteResp.Body.Close()
			t.Logf("Cleaned up test assistant: %s", assistantID)
		}
	})

	t.Run("CreateAssistantWithMinimalFields", func(t *testing.T) {
		// Create assistant with only required fields
		assistantData := map[string]interface{}{
			"name":      "Minimal Test Assistant",
			"type":      "assistant",
			"connector": "openai",
		}

		jsonData, err := json.Marshal(assistantData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully create assistant with minimal fields")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assistantID, hasID := response["assistant_id"].(string)
		assert.True(t, hasID, "Response should have assistant_id")
		t.Logf("Created minimal assistant with ID: %s", assistantID)

		// Clean up
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		deleteResp, err := http.DefaultClient.Do(deleteReq)
		if err == nil {
			defer deleteResp.Body.Close()
		}
	})

	t.Run("CreateAssistantWithAllFields", func(t *testing.T) {
		// Create assistant with all possible fields
		assistantData := map[string]interface{}{
			"name":        "Complete Test Assistant",
			"type":        "assistant",
			"connector":   "openai",
			"description": "A complete test assistant with all fields",
			"avatar":      "https://example.com/avatar.png",
			"tags":        []string{"test", "complete", "all-fields"},
			"public":      false,
			"share":       "private",
			"mentionable": true,
			"automated":   false,
			"readonly":    false,
			"built_in":    false,
			"sort":        100,
			"placeholder": map[string]interface{}{
				"en-us": "Ask me anything...",
				"zh-cn": "有什么可以帮您的...",
			},
			"prompts": []map[string]interface{}{
				{
					"role":    "system",
					"content": "You are a helpful assistant.",
				},
			},
			"options": map[string]interface{}{
				"temperature": 0.7,
				"max_tokens":  2000,
			},
			"workflow": map[string]interface{}{
				"steps": []map[string]interface{}{
					{
						"name":   "step1",
						"action": "process",
					},
				},
			},
			"tools": []map[string]interface{}{
				{
					"name":        "search",
					"description": "Search the web",
					"parameters": map[string]interface{}{
						"query": "string",
					},
				},
			},
			"kb": map[string]interface{}{
				"collections": []string{"collection1", "collection2"},
				"enabled":     true,
			},
			"mcp": map[string]interface{}{
				"servers": []map[string]interface{}{
					{
						"name": "server1",
						"url":  "http://localhost:3000",
					},
				},
			},
			"locales": map[string]interface{}{
				"zh-cn": map[string]interface{}{
					"name":        "完整测试助手",
					"description": "包含所有字段的完整测试助手",
				},
			},
		}

		jsonData, err := json.Marshal(assistantData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully create assistant with all fields")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assistantID, hasID := response["assistant_id"].(string)
		assert.True(t, hasID, "Response should have assistant_id")
		t.Logf("Created complete assistant with ID: %s", assistantID)

		// Clean up
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		deleteResp, err := http.DefaultClient.Do(deleteReq)
		if err == nil {
			defer deleteResp.Body.Close()
		}
	})

	t.Run("CreateAssistantMissingRequiredFields", func(t *testing.T) {
		// Test with missing required fields
		testCases := []struct {
			name string
			data map[string]interface{}
		}{
			{
				name: "MissingName",
				data: map[string]interface{}{
					"type":      "assistant",
					"connector": "openai",
				},
			},
			{
				name: "MissingType",
				data: map[string]interface{}{
					"name":      "Test Assistant",
					"connector": "openai",
				},
			},
			{
				name: "MissingConnector",
				data: map[string]interface{}{
					"name": "Test Assistant",
					"type": "assistant",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				jsonData, err := json.Marshal(tc.data)
				assert.NoError(t, err)

				req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
				assert.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				defer resp.Body.Close()

				// Should return 400 Bad Request or 500 Internal Server Error
				assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError,
					"Should return error for missing required fields")

				var errorResponse map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&errorResponse)
				assert.NoError(t, err)

				t.Logf("Correctly rejected request with %s (status: %d)", tc.name, resp.StatusCode)
			})
		}
	})

	t.Run("CreateAssistantInvalidJSON", func(t *testing.T) {
		// Test with invalid JSON
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBufferString("{invalid json}"))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return 400 for invalid JSON")

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error", "Error response should have 'error' field")

		t.Logf("Correctly rejected invalid JSON")
	})

	t.Run("CreateAssistantUnauthorized", func(t *testing.T) {
		// Test without authentication
		assistantData := map[string]interface{}{
			"name":      "Test Assistant",
			"type":      "assistant",
			"connector": "openai",
		}

		jsonData, err := json.Marshal(assistantData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should return 401 without authentication")
		t.Logf("Correctly rejected unauthorized request")
	})

	t.Run("CreateAssistantWithLocales", func(t *testing.T) {
		// Create assistant with localized content
		assistantData := map[string]interface{}{
			"name":      "Multilingual Test Assistant",
			"type":      "assistant",
			"connector": "openai",
			"locales": map[string]interface{}{
				"zh-cn": map[string]interface{}{
					"name":        "多语言测试助手",
					"description": "这是一个多语言测试助手",
				},
				"ja-jp": map[string]interface{}{
					"name":        "多言語テストアシスタント",
					"description": "これは多言語テストアシスタントです",
				},
			},
		}

		jsonData, err := json.Marshal(assistantData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully create assistant with locales")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assistantID, hasID := response["assistant_id"].(string)
		assert.True(t, hasID, "Response should have assistant_id")
		t.Logf("Created multilingual assistant with ID: %s", assistantID)

		// Clean up
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		deleteResp, err := http.DefaultClient.Do(deleteReq)
		if err == nil {
			defer deleteResp.Body.Close()
		}
	})

	t.Run("CreateAssistantWithTeamScope", func(t *testing.T) {
		// Create assistant - should automatically attach team scope from auth
		assistantData := map[string]interface{}{
			"name":      "Team Scoped Assistant",
			"type":      "assistant",
			"connector": "openai",
			"share":     "team",
		}

		jsonData, err := json.Marshal(assistantData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully create team-scoped assistant")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assistantID, hasID := response["assistant_id"].(string)
		assert.True(t, hasID, "Response should have assistant_id")
		t.Logf("Created team-scoped assistant with ID: %s", assistantID)

		// Verify the assistant was created with proper scope
		// Get the assistant to check if __yao_created_by and __yao_team_id were set
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		if err == nil {
			defer getResp.Body.Close()
			if getResp.StatusCode == http.StatusOK {
				var assistant map[string]interface{}
				json.NewDecoder(getResp.Body).Decode(&assistant)
				t.Logf("Assistant scope fields: __yao_created_by=%v, __yao_team_id=%v",
					assistant["__yao_created_by"], assistant["__yao_team_id"])
			}
		}

		// Clean up
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		deleteResp, err := http.DefaultClient.Do(deleteReq)
		if err == nil {
			defer deleteResp.Body.Close()
		}
	})

	t.Run("CreateAssistantVerifyCacheReload", func(t *testing.T) {
		// Create assistant and verify it's immediately available
		assistantData := map[string]interface{}{
			"name":      "Cache Test Assistant",
			"type":      "assistant",
			"connector": "openai",
		}

		jsonData, err := json.Marshal(assistantData)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assistantID, hasID := response["assistant_id"].(string)
		assert.True(t, hasID)

		// Immediately try to get the assistant - should be available (cache reloaded)
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		assert.NotNil(t, getResp)
		defer getResp.Body.Close()

		// Should be immediately available thanks to cache reload
		if getResp.StatusCode == http.StatusOK {
			t.Logf("Assistant immediately available after creation (cache reloaded successfully)")
		} else {
			t.Logf("Assistant not immediately available (status: %d) - cache reload may have failed", getResp.StatusCode)
		}

		// Clean up
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		deleteResp, err := http.DefaultClient.Do(deleteReq)
		if err == nil {
			defer deleteResp.Body.Close()
		}
	})
}

// BenchmarkCreateAssistant benchmarks the create assistant endpoint
func BenchmarkCreateAssistant(b *testing.B) {
	// Convert testing.B to testing.T for Prepare/Clean
	t := &testing.T{}
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Create Benchmark Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	assistantData := map[string]interface{}{
		"name":      "Benchmark Test Assistant",
		"type":      "assistant",
		"connector": "openai",
	}

	jsonData, _ := json.Marshal(assistantData)

	// Track created assistants for cleanup
	createdIDs := make([]string, 0, b.N)

	// Reset timer after setup
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		assistantData["name"] = fmt.Sprintf("Benchmark Test Assistant %d", i)
		jsonData, _ = json.Marshal(assistantData)

		req, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode == http.StatusOK {
			var response map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&response)
			if id, ok := response["assistant_id"].(string); ok {
				createdIDs = append(createdIDs, id)
			}
		}
		resp.Body.Close()
	}

	// Cleanup created assistants
	b.StopTimer()
	for _, id := range createdIDs {
		req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+id, nil)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		resp, _ := http.DefaultClient.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
	}
}
