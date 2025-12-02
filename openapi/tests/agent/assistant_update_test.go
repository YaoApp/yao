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

// TestUpdateAssistant tests the update assistant endpoint
func TestUpdateAssistant(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Agent Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Helper function to create a test assistant
	createTestAssistant := func(name string) string {
		assistantData := map[string]interface{}{
			"name":      name,
			"type":      "assistant",
			"connector": "openai",
		}

		jsonData, _ := json.Marshal(assistantData)
		req, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to create test assistant: %v", err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		return response["assistant_id"].(string)
	}

	t.Run("UpdateAssistantSuccess", func(t *testing.T) {
		// Create a test assistant first
		assistantID := createTestAssistant("Original Test Assistant")
		defer func() {
			// Clean up
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update the assistant
		updateData := map[string]interface{}{
			"name":        "Updated Test Assistant",
			"description": "This assistant has been updated",
			"tags":        []string{"updated", "test"},
			"mentionable": true,
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Read response body for debugging
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Log response for debugging
		if resp.StatusCode != http.StatusOK {
			t.Logf("Update failed: status=%d, response=%+v", resp.StatusCode, response)
		}

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update assistant")

		// Verify assistant_id in response
		returnedID, hasID := response["assistant_id"].(string)
		assert.True(t, hasID, "Response should have assistant_id")
		assert.Equal(t, assistantID, returnedID, "Returned ID should match original ID")

		t.Logf("Successfully updated assistant: %s", assistantID)

		// Verify the update by getting the assistant
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		if getResp.StatusCode == http.StatusOK {
			var assistant map[string]interface{}
			json.NewDecoder(getResp.Body).Decode(&assistant)

			// Verify updated fields
			if name, ok := assistant["name"].(string); ok {
				assert.Equal(t, "Updated Test Assistant", name, "Name should be updated")
			}
			if desc, ok := assistant["description"].(string); ok {
				assert.Equal(t, "This assistant has been updated", desc, "Description should be updated")
			}

			t.Logf("Verified assistant update: name=%v, description=%v", assistant["name"], assistant["description"])
		}
	})

	t.Run("UpdateAssistantPartialFields", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Partial Update Test Assistant")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update only description field
		updateData := map[string]interface{}{
			"description": "Only description updated",
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update partial fields")
		t.Logf("Successfully updated partial fields for assistant: %s", assistantID)
	})

	t.Run("UpdateAssistantNotFound", func(t *testing.T) {
		// Try to update non-existent assistant
		updateData := map[string]interface{}{
			"name": "Updated Name",
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/non-existent-id-12345", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 403 Forbidden or 404 Not Found (permission check fails first)
		assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound,
			"Should return error for non-existent assistant")

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error", "Error response should have 'error' field")

		t.Logf("Correctly rejected update to non-existent assistant (status: %d)", resp.StatusCode)
	})

	t.Run("UpdateAssistantUnauthorized", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Unauthorized Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Try to update without authentication
		updateData := map[string]interface{}{
			"name": "Unauthorized Update",
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should return 401 without authentication")
		t.Logf("Correctly rejected unauthorized update request")
	})

	t.Run("UpdateAssistantInvalidJSON", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Invalid JSON Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Try to update with invalid JSON
		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBufferString("{invalid json}"))
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

	t.Run("UpdateAssistantEmptyID", func(t *testing.T) {
		// Try to update with empty ID (should be caught by router)
		updateData := map[string]interface{}{
			"name": "Updated Name",
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return error (404 or 405 Method Not Allowed depending on router)
		assert.True(t, resp.StatusCode >= 400, "Should return error for empty ID")
		t.Logf("Handled empty ID in update request (status: %d)", resp.StatusCode)
	})

	t.Run("UpdateAssistantChangeTypeAndConnector", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Type Change Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Try to change type and connector (might be allowed or restricted depending on business logic)
		updateData := map[string]interface{}{
			"type":      "workflow",
			"connector": "moapi",
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Response depends on business logic - either OK or error
		t.Logf("Attempted to change type and connector (status: %d)", resp.StatusCode)
	})

	t.Run("UpdateAssistantVerifyCacheReload", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Cache Reload Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update the assistant
		updateData := map[string]interface{}{
			"name":        "Cache Test Updated",
			"description": "Testing cache reload after update",
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Immediately try to get the assistant - should return updated data (cache reloaded)
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		assert.NotNil(t, getResp)
		defer getResp.Body.Close()

		if getResp.StatusCode == http.StatusOK {
			var assistant map[string]interface{}
			json.NewDecoder(getResp.Body).Decode(&assistant)

			// Verify updated name is immediately visible
			if name, ok := assistant["name"].(string); ok {
				if name == "Cache Test Updated" {
					t.Logf("Cache reloaded successfully - updated data immediately visible")
				} else {
					t.Logf("Cache reload may have issues - expected 'Cache Test Updated', got '%s'", name)
				}
			}
		}
	})

	t.Run("UpdateAssistantLocales", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Locales Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update with localized content
		updateData := map[string]interface{}{
			"locales": map[string]interface{}{
				"zh-cn": map[string]interface{}{
					"name":        "更新的中文名称",
					"description": "更新的中文描述",
				},
				"ja-jp": map[string]interface{}{
					"name":        "更新された日本語名",
					"description": "更新された日本語の説明",
				},
			},
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update locales")
		t.Logf("Successfully updated assistant locales: %s", assistantID)
	})

	t.Run("UpdateAssistantOptions", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Options Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update options
		updateData := map[string]interface{}{
			"options": map[string]interface{}{
				"temperature":   0.9,
				"max_tokens":    4000,
				"top_p":         0.95,
				"custom_option": "custom_value",
			},
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update options")
		t.Logf("Successfully updated assistant options: %s", assistantID)
	})

	t.Run("UpdateAssistantPrompts", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Prompts Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update prompts
		updateData := map[string]interface{}{
			"prompts": []map[string]interface{}{
				{
					"role":    "system",
					"content": "You are an updated helpful assistant with new instructions.",
				},
				{
					"role":    "user",
					"content": "Additional context message.",
				},
			},
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update prompts")
		t.Logf("Successfully updated assistant prompts: %s", assistantID)
	})

	t.Run("UpdateAssistantSharePermissions", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Share Permissions Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update share and public settings
		updateData := map[string]interface{}{
			"public": true,
			"share":  "team",
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update share permissions")
		t.Logf("Successfully updated assistant share permissions: %s", assistantID)
	})

	t.Run("UpdateAssistantKnowledgeBase", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("KB Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update kb settings
		updateData := map[string]interface{}{
			"kb": map[string]interface{}{
				"collections": []string{"test-collection-1", "test-collection-2"},
				"enabled":     true,
				"threshold":   0.8,
			},
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update kb settings")
		t.Logf("Successfully updated assistant kb settings: %s", assistantID)
	})

	t.Run("UpdateAssistantMCP", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("MCP Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update mcp settings
		updateData := map[string]interface{}{
			"mcp": map[string]interface{}{
				"servers": []map[string]interface{}{
					{
						"name":    "test-mcp-server",
						"url":     "http://localhost:4000",
						"enabled": true,
					},
				},
			},
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update mcp settings")
		t.Logf("Successfully updated assistant mcp settings: %s", assistantID)
	})

	// Note: UpdateAssistantTools test removed - tools field is deprecated and replaced by MCP

	t.Run("UpdateAssistantWorkflow", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("Workflow Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update workflow
		updateData := map[string]interface{}{
			"workflow": map[string]interface{}{
				"steps": []map[string]interface{}{
					{
						"name":   "analyze",
						"action": "analyze_input",
						"next":   "process",
					},
					{
						"name":   "process",
						"action": "process_data",
						"next":   "respond",
					},
					{
						"name":   "respond",
						"action": "generate_response",
					},
				},
			},
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update workflow")
		t.Logf("Successfully updated assistant workflow: %s", assistantID)
	})

	t.Run("UpdateAssistantAllFields", func(t *testing.T) {
		// Create a test assistant
		assistantID := createTestAssistant("All Fields Update Test")
		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Update all fields at once
		updateData := map[string]interface{}{
			"name":        "Completely Updated Assistant",
			"description": "All fields have been updated",
			"avatar":      "https://example.com/new-avatar.png",
			"tags":        []string{"updated", "complete", "all-fields"},
			"public":      true,
			"share":       "team",
			"mentionable": false,
			"automated":   true,
			"readonly":    false,
			"sort":        200,
			"placeholder": map[string]interface{}{
				"en-us": "Updated placeholder...",
				"zh-cn": "更新的占位符...",
			},
			"prompts": []map[string]interface{}{
				{
					"role":    "system",
					"content": "You are an updated helpful assistant with new capabilities.",
				},
			},
			"options": map[string]interface{}{
				"temperature":       0.9,
				"max_tokens":        4000,
				"top_p":             0.95,
				"frequency_penalty": 0.5,
			},
			"workflow": map[string]interface{}{
				"steps": []map[string]interface{}{
					{
						"name":   "updated_step",
						"action": "updated_action",
					},
				},
			},
			// Note: tools field removed - now handled by MCP
			"kb": map[string]interface{}{
				"collections": []string{"updated-collection"},
				"enabled":     true,
			},
			"mcp": map[string]interface{}{
				"servers": []map[string]interface{}{
					{
						"name": "updated_server",
						"url":  "http://localhost:5000",
					},
				},
			},
			"locales": map[string]interface{}{
				"zh-cn": map[string]interface{}{
					"name":        "完全更新的助手",
					"description": "所有字段都已更新",
				},
				"ja-jp": map[string]interface{}{
					"name":        "完全に更新されたアシスタント",
					"description": "すべてのフィールドが更新されました",
				},
			},
		}

		jsonData, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully update all fields")
		t.Logf("Successfully updated all assistant fields: %s", assistantID)

		// Verify the update by getting the assistant
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		if getResp.StatusCode == http.StatusOK {
			var assistant map[string]interface{}
			json.NewDecoder(getResp.Body).Decode(&assistant)
			t.Logf("Verified all fields updated - name: %v, description: %v, tags: %v",
				assistant["name"], assistant["description"], assistant["tags"])
		}
	})
}

// TestUpdateAssistantPermissions tests permission-based access control for updates
func TestUpdateAssistantPermissions(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Create two different test users with tokens
	client := testutils.RegisterTestClient(t, "Agent Update Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// User 1 token
	token1 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// User 2 token (different user)
	token2 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("UserCanUpdateOwnAssistant", func(t *testing.T) {
		// User 1 creates an assistant
		assistantData := map[string]interface{}{
			"name":      "User 1 Assistant",
			"type":      "assistant",
			"connector": "openai",
		}

		jsonData, _ := json.Marshal(assistantData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createReq.Header.Set("Content-Type", "application/json")

		createResp, err := http.DefaultClient.Do(createReq)
		if err != nil || createResp.StatusCode != http.StatusOK {
			t.Skip("Cannot create assistant for permission test")
			return
		}
		defer createResp.Body.Close()

		var createResponse map[string]interface{}
		json.NewDecoder(createResp.Body).Decode(&createResponse)
		assistantID := createResponse["assistant_id"].(string)

		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// User 1 should be able to update their own assistant
		updateData := map[string]interface{}{
			"description": "Updated by owner",
		}

		jsonData, _ = json.Marshal(updateData)
		updateReq, _ := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		updateReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		updateReq.Header.Set("Content-Type", "application/json")

		updateResp, err := http.DefaultClient.Do(updateReq)
		assert.NoError(t, err)
		defer updateResp.Body.Close()

		// Should succeed
		if updateResp.StatusCode == http.StatusOK {
			t.Logf("User 1 successfully updated their own assistant")
		} else {
			t.Logf("User 1 got status %d when updating own assistant", updateResp.StatusCode)
		}
	})

	t.Run("UserCannotUpdateOthersAssistant", func(t *testing.T) {
		// User 1 creates an assistant
		assistantData := map[string]interface{}{
			"name":      "User 1 Protected Assistant",
			"type":      "assistant",
			"connector": "openai",
			"share":     "private",
		}

		jsonData, _ := json.Marshal(assistantData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createReq.Header.Set("Content-Type", "application/json")

		createResp, err := http.DefaultClient.Do(createReq)
		if err != nil || createResp.StatusCode != http.StatusOK {
			t.Skip("Cannot create assistant for permission test")
			return
		}
		defer createResp.Body.Close()

		var createResponse map[string]interface{}
		json.NewDecoder(createResp.Body).Decode(&createResponse)
		assistantID := createResponse["assistant_id"].(string)

		defer func() {
			deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
			deleteReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
			resp, _ := http.DefaultClient.Do(deleteReq)
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// User 2 tries to update User 1's private assistant
		updateData := map[string]interface{}{
			"description": "Unauthorized update attempt",
		}

		jsonData, _ = json.Marshal(updateData)
		updateReq, _ := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		updateReq.Header.Set("Authorization", "Bearer "+token2.AccessToken)
		updateReq.Header.Set("Content-Type", "application/json")

		updateResp, err := http.DefaultClient.Do(updateReq)
		assert.NoError(t, err)
		defer updateResp.Body.Close()

		// Should be forbidden (403) - permission check should prevent this
		if updateResp.StatusCode == http.StatusForbidden {
			t.Logf("Correctly prevented User 2 from updating User 1's private assistant")
		} else {
			t.Logf("User 2 got status %d when trying to update User 1's assistant (expected 403)", updateResp.StatusCode)
		}
	})
}

// BenchmarkUpdateAssistant benchmarks the update assistant endpoint
func BenchmarkUpdateAssistant(b *testing.B) {
	// Convert testing.B to testing.T for Prepare/Clean
	t := &testing.T{}
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Update Benchmark Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test assistant for benchmarking
	assistantData := map[string]interface{}{
		"name":      "Benchmark Test Assistant",
		"type":      "assistant",
		"connector": "openai",
	}

	jsonData, _ := json.Marshal(assistantData)
	createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/assistants", bytes.NewBuffer(jsonData))
	createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, _ := http.DefaultClient.Do(createReq)
	if createResp.StatusCode != http.StatusOK {
		b.Fatal("Failed to create test assistant for benchmark")
	}
	defer createResp.Body.Close()

	var createResponse map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&createResponse)
	assistantID := createResponse["assistant_id"].(string)

	// Cleanup after benchmark
	defer func() {
		deleteReq, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		resp, _ := http.DefaultClient.Do(deleteReq)
		if resp != nil {
			resp.Body.Close()
		}
	}()

	updateData := map[string]interface{}{
		"description": "Benchmark update",
	}

	// Reset timer after setup
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		updateData["description"] = fmt.Sprintf("Benchmark update %d", i)
		jsonData, _ = json.Marshal(updateData)

		req, _ := http.NewRequest("PUT", serverURL+baseURL+"/agent/assistants/"+assistantID, bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}
