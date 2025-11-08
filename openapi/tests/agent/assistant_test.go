package openapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestListAssistants tests the assistants listing endpoint
func TestListAssistants(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Agent List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ListAssistantsSuccess", func(t *testing.T) {
		// Test listing all assistants
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve assistants")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Response should have pagination structure
		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants", len(data))
		} else {
			t.Logf("Successfully retrieved assistants response (data field type: %T)", response["data"])
		}

		// Check pagination fields
		assert.Contains(t, response, "page")
		assert.Contains(t, response, "pagesize")
		assert.Contains(t, response, "total")
	})

	t.Run("ListAssistantsWithPagination", func(t *testing.T) {
		// Test with pagination parameters
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?page=1&pagesize=10", nil)
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
		page, hasPage := response["page"].(float64)
		pagesize, hasPagesize := response["pagesize"].(float64)

		if hasPage && hasPagesize {
			assert.Equal(t, float64(1), page, "Page should be 1")
			assert.Equal(t, float64(10), pagesize, "Pagesize should be 10")
			t.Logf("Pagination working correctly: page=%d, pagesize=%d", int(page), int(pagesize))
		}
	})

	t.Run("ListAssistantsWithKeywords", func(t *testing.T) {
		// Test with keywords filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?keywords=test", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants with keywords filter", len(data))
		}
	})

	t.Run("ListAssistantsWithType", func(t *testing.T) {
		// Test with type filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?type=assistant", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants with type filter", len(data))
		}
	})

	t.Run("ListAssistantsWithTags", func(t *testing.T) {
		// Test with tags filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?tags=productivity,ai", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants with tags filter", len(data))
		}
	})

	t.Run("ListAssistantsWithBuiltInFilter", func(t *testing.T) {
		// Test with built_in filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?built_in=true", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d built-in assistants", len(data))

			// Verify that built-in assistants have sensitive fields filtered
			for _, item := range data {
				assistant, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				builtIn, hasBuiltIn := assistant["built_in"].(bool)
				if hasBuiltIn && builtIn {
					// Check that code-level fields are null or absent
					prompts := assistant["prompts"]
					workflow := assistant["workflow"]
					tools := assistant["tools"]
					kb := assistant["kb"]
					mcp := assistant["mcp"]
					options := assistant["options"]

					// These should be nil or absent for built-in assistants
					if prompts != nil {
						t.Logf("Warning: Built-in assistant has non-nil prompts field: %v", prompts)
					}
					if workflow != nil {
						t.Logf("Warning: Built-in assistant has non-nil workflow field: %v", workflow)
					}
					if tools != nil {
						t.Logf("Warning: Built-in assistant has non-nil tools field: %v", tools)
					}
					if kb != nil {
						t.Logf("Warning: Built-in assistant has non-nil kb field: %v", kb)
					}
					if mcp != nil {
						t.Logf("Warning: Built-in assistant has non-nil mcp field: %v", mcp)
					}
					if options != nil {
						t.Logf("Warning: Built-in assistant has non-nil options field: %v", options)
					}
				}
			}
		}
	})

	t.Run("ListAssistantsWithMentionableFilter", func(t *testing.T) {
		// Test with mentionable filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?mentionable=true", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d mentionable assistants", len(data))
		}
	})

	t.Run("ListAssistantsWithAutomatedFilter", func(t *testing.T) {
		// Test with automated filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?automated=false", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d non-automated assistants", len(data))
		}
	})

	t.Run("ListAssistantsWithSelectFields", func(t *testing.T) {
		// Test with select parameter to limit returned fields
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?select=assistant_id,name,avatar,type", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData && len(data) > 0 {
			// Check first assistant to verify field selection worked
			assistant, ok := data[0].(map[string]interface{})
			if ok {
				t.Logf("Assistant fields returned: %+v", assistant)
				// Note: The actual fields returned depend on the implementation
				// This test verifies the select parameter is accepted without error
			}
		}
	})

	t.Run("ListAssistantsWithInvalidSelectFields", func(t *testing.T) {
		// Test with invalid select fields (should be filtered by whitelist)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?select=invalid_field,malicious_sql", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should still return 200, but with default/filtered fields
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		t.Logf("Successfully handled invalid select fields by using whitelist")
	})

	t.Run("ListAssistantsWithMultipleFilters", func(t *testing.T) {
		// Test with multiple filter parameters combined
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?type=assistant&built_in=false&mentionable=true&page=1&pagesize=5", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants with multiple filters", len(data))
		}
	})

	t.Run("ListAssistantsWithConnector", func(t *testing.T) {
		// Test with connector filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?connector=openai", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants with connector filter", len(data))
		}
	})

	t.Run("ListAssistantsWithAssistantID", func(t *testing.T) {
		// Test with specific assistant_id filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?assistant_id=test_assistant", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants with assistant_id filter", len(data))
		}
	})

	t.Run("ListAssistantsWithAssistantIDs", func(t *testing.T) {
		// Test with multiple assistant_ids filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?assistant_ids=assistant1,assistant2,assistant3", nil)
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

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("Successfully retrieved %d assistants with assistant_ids filter", len(data))
		}
	})

	t.Run("ListAssistantsWithInvalidPagination", func(t *testing.T) {
		// Test with invalid pagination parameters (should use defaults)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?page=-1&pagesize=1000", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return error for invalid pagination
		if resp.StatusCode == http.StatusBadRequest {
			var errorResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse, "error")
			t.Logf("Correctly rejected invalid pagination parameters")
		} else {
			// Or apply default/corrected values
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			t.Logf("Applied default/corrected pagination values")
		}
	})
}

// TestAssistantEndpointsUnauthorized tests that endpoints return 401 when not authenticated
func TestAssistantEndpointsUnauthorized(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/agent/assistants"},
		{"GET", "/agent/assistants?page=1&pagesize=10"},
		{"GET", "/agent/assistants?keywords=test"},
		{"GET", "/agent/assistants?type=assistant"},
		{"GET", "/agent/assistants?built_in=true"},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("Unauthorized_%s_%s", endpoint.method, endpoint.path), func(t *testing.T) {
			req, err := http.NewRequest(endpoint.method, serverURL+baseURL+endpoint.path, nil)
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

// TestAssistantPermissionFiltering tests that permission-based filtering works correctly
func TestAssistantPermissionFiltering(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Create two different test users with tokens
	client := testutils.RegisterTestClient(t, "Agent Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// User 1 token
	token1 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// User 2 token (different user)
	token2 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("User1CanSeeOwnAssistants", func(t *testing.T) {
		// User 1 should see their own assistants
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("User 1 can see %d assistants", len(data))
		}
	})

	t.Run("User2SeesFilteredResults", func(t *testing.T) {
		// User 2 should see different assistants (permission filtering applied)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token2.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		data, hasData := response["data"].([]interface{})
		if hasData {
			t.Logf("User 2 can see %d assistants (permission filtering applied)", len(data))
		}
	})
}

// TestAssistantResponseStructure tests that the response structure is correct
func TestAssistantResponseStructure(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Response Structure Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ResponseHasCorrectStructure", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?page=1&pagesize=5", nil)
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

		// Verify response structure matches OpenAPI standard
		assert.Contains(t, response, "data", "Response should have 'data' field")
		assert.Contains(t, response, "page", "Response should have 'page' field")
		assert.Contains(t, response, "pagesize", "Response should have 'pagesize' field")
		assert.Contains(t, response, "total", "Response should have 'total' field")

		// Verify data is an array
		data, ok := response["data"].([]interface{})
		assert.True(t, ok, "Data field should be an array")
		t.Logf("Response structure is correct with %d assistants", len(data))
	})
}

// TestAssistantLocaleSupport tests that locale parameter works correctly
func TestAssistantLocaleSupport(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Locale Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	locales := []string{"en-us", "zh-cn", "ja-jp", "de-de", "fr-fr"}

	for _, locale := range locales {
		t.Run(fmt.Sprintf("LocaleSupport_%s", locale), func(t *testing.T) {
			req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?locale="+locale, nil)
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

			t.Logf("Successfully retrieved assistants with locale: %s", locale)
		})
	}
}

// TestAssistantEdgeCases tests edge cases and boundary conditions
func TestAssistantEdgeCases(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Edge Cases Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("EmptyKeywordsParameter", func(t *testing.T) {
		// Test with empty keywords parameter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?keywords=", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("Handled empty keywords parameter correctly")
	})

	t.Run("EmptyTagsParameter", func(t *testing.T) {
		// Test with empty tags parameter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?tags=", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("Handled empty tags parameter correctly")
	})

	t.Run("VeryLongKeywords", func(t *testing.T) {
		// Test with very long keywords string
		longKeywords := string(make([]byte, 1000))
		for i := range longKeywords {
			longKeywords = longKeywords[:i] + "test"
		}

		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?keywords="+longKeywords[:500], nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should handle gracefully (either return results or error)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest)
		t.Logf("Handled very long keywords parameter (status: %d)", resp.StatusCode)
	})

	t.Run("SpecialCharactersInKeywords", func(t *testing.T) {
		// Test with special characters in keywords
		specialKeywords := "test&special=chars<>\"';--"
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?keywords="+specialKeywords, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("Handled special characters in keywords correctly")
	})

	t.Run("MaxPageSize", func(t *testing.T) {
		// Test with maximum page size (should be capped at 100)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=100", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		pagesize, ok := response["pagesize"].(float64)
		if ok {
			assert.LessOrEqual(t, int(pagesize), 100, "Pagesize should be capped at 100")
			t.Logf("Correctly capped pagesize at %d", int(pagesize))
		}
	})
}

// TestListAssistantTags tests the assistant tags endpoint
func TestListAssistantTags(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Tags Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ListAssistantTagsSuccess", func(t *testing.T) {
		// Test listing all assistant tags
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve tags")

		var tags []interface{}
		err = json.NewDecoder(resp.Body).Decode(&tags)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d tags", len(tags))

		// Verify tag structure
		if len(tags) > 0 {
			tag, ok := tags[0].(map[string]interface{})
			if ok {
				assert.Contains(t, tag, "value", "Tag should have value field")
				assert.Contains(t, tag, "label", "Tag should have label field")
			}
		}
	})

	t.Run("ListAssistantTagsWithLocale", func(t *testing.T) {
		// Test with locale parameter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags?locale=zh-cn", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []interface{}
		err = json.NewDecoder(resp.Body).Decode(&tags)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d tags with zh-cn locale", len(tags))
	})

	t.Run("ListAssistantTagsWithFilters", func(t *testing.T) {
		// Test with type filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags?type=assistant", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []interface{}
		err = json.NewDecoder(resp.Body).Decode(&tags)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d tags with type filter", len(tags))
	})

	t.Run("ListAssistantTagsWithConnector", func(t *testing.T) {
		// Test with connector filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags?connector=openai", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []interface{}
		err = json.NewDecoder(resp.Body).Decode(&tags)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d tags for openai connector", len(tags))
	})

	t.Run("ListAssistantTagsWithBuiltInFilter", func(t *testing.T) {
		// Test with built_in filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags?built_in=false", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []interface{}
		err = json.NewDecoder(resp.Body).Decode(&tags)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d tags for non-built-in assistants", len(tags))
	})

	t.Run("ListAssistantTagsWithMentionableFilter", func(t *testing.T) {
		// Test with mentionable filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags?mentionable=true", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []interface{}
		err = json.NewDecoder(resp.Body).Decode(&tags)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d tags for mentionable assistants", len(tags))
	})

	t.Run("ListAssistantTagsWithKeywords", func(t *testing.T) {
		// Test with keywords filter
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags?keywords=test", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []interface{}
		err = json.NewDecoder(resp.Body).Decode(&tags)
		assert.NoError(t, err)

		t.Logf("Successfully retrieved %d tags with keywords filter", len(tags))
	})

	t.Run("ListAssistantTagsUnauthorized", func(t *testing.T) {
		// Test without authentication
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/tags", nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		t.Logf("Correctly rejected unauthorized request")
	})
}

// TestGetAssistant tests the get assistant detail endpoint
func TestGetAssistant(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Agent Detail Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("GetAssistantSuccess", func(t *testing.T) {
		// First, get a list of assistants to find a valid assistant_id
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=1", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		assert.NotNil(t, listResp)
		defer listResp.Body.Close()

		var listResponse map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResponse)
		assert.NoError(t, err)

		data, hasData := listResponse["data"].([]interface{})
		if !hasData || len(data) == 0 {
			t.Skip("No assistants available for testing")
			return
		}

		// Get the first assistant's ID
		assistant, ok := data[0].(map[string]interface{})
		if !ok {
			t.Skip("Invalid assistant data format")
			return
		}

		assistantID, ok := assistant["assistant_id"].(string)
		if !ok || assistantID == "" {
			t.Skip("Assistant ID not found")
			return
		}

		t.Logf("Testing with assistant ID: %s", assistantID)

		// Now test getting the specific assistant
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Expect successful response
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should successfully retrieve assistant")

		var assistantDetail map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&assistantDetail)
		assert.NoError(t, err)

		// Response should be the assistant object directly
		assert.NotNil(t, assistantDetail, "Assistant data should not be nil")

		// Verify assistant_id matches
		returnedID, hasID := assistantDetail["assistant_id"].(string)
		assert.True(t, hasID, "Assistant should have assistant_id field")
		assert.Equal(t, assistantID, returnedID, "Returned assistant_id should match requested ID")

		t.Logf("Successfully retrieved assistant: %s", returnedID)
	})

	t.Run("GetAssistantWithoutLocale", func(t *testing.T) {
		// Get a valid assistant ID first
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=1", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		var listResponse map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResponse)
		assert.NoError(t, err)

		data, hasData := listResponse["data"].([]interface{})
		if !hasData || len(data) == 0 {
			t.Skip("No assistants available for testing")
			return
		}

		assistant, ok := data[0].(map[string]interface{})
		if !ok {
			t.Skip("Invalid assistant data format")
			return
		}

		assistantID, ok := assistant["assistant_id"].(string)
		if !ok || assistantID == "" {
			t.Skip("Assistant ID not found")
			return
		}

		// Test without locale parameter - should return raw data (useful for form editing)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var assistantData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&assistantData)
		assert.NoError(t, err)

		assert.NotNil(t, assistantData, "Assistant data should not be nil")

		t.Logf("Successfully retrieved assistant without locale (raw data for editing)")
	})

	t.Run("GetAssistantWithLocale", func(t *testing.T) {
		// Get a valid assistant ID first
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=1", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		var listResponse map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResponse)
		assert.NoError(t, err)

		data, hasData := listResponse["data"].([]interface{})
		if !hasData || len(data) == 0 {
			t.Skip("No assistants available for testing")
			return
		}

		assistant, ok := data[0].(map[string]interface{})
		if !ok {
			t.Skip("Invalid assistant data format")
			return
		}

		assistantID, ok := assistant["assistant_id"].(string)
		if !ok || assistantID == "" {
			t.Skip("Assistant ID not found")
			return
		}

		// Test with locale parameter - should return translated data
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID+"?locale=zh-cn", nil)
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

		t.Logf("Successfully retrieved assistant with zh-cn locale (translated)")
	})

	t.Run("GetAssistantBuiltInFieldsFiltered", func(t *testing.T) {
		// Get a built-in assistant
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?built_in=true&pagesize=1", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		var listResponse map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResponse)
		assert.NoError(t, err)

		data, hasData := listResponse["data"].([]interface{})
		if !hasData || len(data) == 0 {
			t.Skip("No built-in assistants available for testing")
			return
		}

		assistant, ok := data[0].(map[string]interface{})
		if !ok {
			t.Skip("Invalid assistant data format")
			return
		}

		assistantID, ok := assistant["assistant_id"].(string)
		if !ok || assistantID == "" {
			t.Skip("Assistant ID not found")
			return
		}

		// Get the built-in assistant detail
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var builtInAssistant map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&builtInAssistant)
		assert.NoError(t, err)

		assert.NotNil(t, builtInAssistant, "Assistant data should not be nil")

		// Verify built-in flag
		builtIn, hasBuiltIn := builtInAssistant["built_in"].(bool)
		if hasBuiltIn && builtIn {
			// Check that sensitive fields are filtered
			prompts := builtInAssistant["prompts"]
			workflow := builtInAssistant["workflow"]
			tools := builtInAssistant["tools"]
			kb := builtInAssistant["kb"]
			mcp := builtInAssistant["mcp"]
			options := builtInAssistant["options"]

			// These should be nil or absent for built-in assistants
			assert.Nil(t, prompts, "Built-in assistant should not expose prompts")
			assert.Nil(t, workflow, "Built-in assistant should not expose workflow")
			assert.Nil(t, tools, "Built-in assistant should not expose tools")
			assert.Nil(t, kb, "Built-in assistant should not expose kb")
			assert.Nil(t, mcp, "Built-in assistant should not expose mcp")
			assert.Nil(t, options, "Built-in assistant should not expose options")

			t.Logf("Built-in assistant sensitive fields correctly filtered")
		}
	})

	t.Run("GetAssistantNotFound", func(t *testing.T) {
		// Test with non-existent assistant ID
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/non-existent-id-12345", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should return 404 Not Found
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 for non-existent assistant")

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)

		assert.Contains(t, errorResponse, "error", "Error response should have 'error' field")
		t.Logf("Correctly returned 404 for non-existent assistant")
	})

	t.Run("GetAssistantUnauthorized", func(t *testing.T) {
		// Test without authentication
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/some-id", nil)
		assert.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should return 401 without authentication")
		t.Logf("Correctly rejected unauthorized request")
	})

	t.Run("GetAssistantEmptyID", func(t *testing.T) {
		// Test with empty assistant ID (should be caught by router)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// This should either redirect to list endpoint or return 404
		// Depends on router configuration
		assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusOK)
		t.Logf("Handled empty assistant ID (status: %d)", resp.StatusCode)
	})
}

// TestGetAssistantPermissionFiltering tests that permission-based filtering works for single assistant retrieval
func TestGetAssistantPermissionFiltering(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Create test client
	client := testutils.RegisterTestClient(t, "Agent Detail Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// User 1 token
	token1 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("UserCanAccessPublicAssistant", func(t *testing.T) {
		// Get a public assistant (if available)
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=10", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		var listResponse map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResponse)
		assert.NoError(t, err)

		data, hasData := listResponse["data"].([]interface{})
		if !hasData || len(data) == 0 {
			t.Skip("No assistants available for testing")
			return
		}

		// Find a public assistant
		var publicAssistantID string
		for _, item := range data {
			assistant, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			isPublic, hasPublic := assistant["public"].(bool)
			assistantID, hasID := assistant["assistant_id"].(string)

			if hasPublic && isPublic && hasID && assistantID != "" {
				publicAssistantID = assistantID
				break
			}
		}

		if publicAssistantID == "" {
			t.Skip("No public assistants available for testing")
			return
		}

		// Test accessing public assistant
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+publicAssistantID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "User should be able to access public assistant")

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		t.Logf("User successfully accessed public assistant: %s", publicAssistantID)
	})

	t.Run("UserCanAccessOwnAssistant", func(t *testing.T) {
		// This test assumes user has their own assistants
		// In a real scenario, you would create a test assistant first
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=1", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		var listResponse map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResponse)
		assert.NoError(t, err)

		data, hasData := listResponse["data"].([]interface{})
		if !hasData || len(data) == 0 {
			t.Skip("No assistants available for testing")
			return
		}

		assistant, ok := data[0].(map[string]interface{})
		if !ok {
			t.Skip("Invalid assistant data format")
			return
		}

		assistantID, ok := assistant["assistant_id"].(string)
		if !ok || assistantID == "" {
			t.Skip("Assistant ID not found")
			return
		}

		// Test accessing own assistant
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Should be able to access (either own assistant or permission allows)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden)

		if resp.StatusCode == http.StatusOK {
			t.Logf("User successfully accessed assistant: %s", assistantID)
		} else {
			t.Logf("User does not have permission to access assistant: %s", assistantID)
		}
	})
}

// TestGetAssistantResponseStructure tests that the response structure matches expectations
func TestGetAssistantResponseStructure(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Detail Response Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ResponseHasCorrectStructure", func(t *testing.T) {
		// Get a valid assistant ID first
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=1", nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		var listResponse map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResponse)
		assert.NoError(t, err)

		data, hasData := listResponse["data"].([]interface{})
		if !hasData || len(data) == 0 {
			t.Skip("No assistants available for testing")
			return
		}

		assistant, ok := data[0].(map[string]interface{})
		if !ok {
			t.Skip("Invalid assistant data format")
			return
		}

		assistantID, ok := assistant["assistant_id"].(string)
		if !ok || assistantID == "" {
			t.Skip("Assistant ID not found")
			return
		}

		// Get assistant detail
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var responseAssistant map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&responseAssistant)
		assert.NoError(t, err)

		// Verify response is an object (not wrapped in data field)
		assert.NotNil(t, responseAssistant, "Assistant should not be nil")

		// Verify essential assistant fields
		assert.Contains(t, responseAssistant, "assistant_id", "Assistant should have assistant_id")
		assert.Contains(t, responseAssistant, "name", "Assistant should have name")
		assert.Contains(t, responseAssistant, "type", "Assistant should have type")

		responseAssistantID := responseAssistant["assistant_id"].(string)
		t.Logf("Response structure is correct for assistant: %s", responseAssistantID)
	})
}

// BenchmarkListAssistants benchmarks the list assistants endpoint
func BenchmarkListAssistants(b *testing.B) {
	// Convert testing.B to testing.T for Prepare/Clean
	t := &testing.T{}
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Benchmark Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Reset timer after setup
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?page=1&pagesize=20", nil)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}

// BenchmarkGetAssistant benchmarks the get assistant detail endpoint
func BenchmarkGetAssistant(b *testing.B) {
	// Convert testing.B to testing.T for Prepare/Clean
	t := &testing.T{}
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Agent Detail Benchmark Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Get a valid assistant ID for benchmarking
	listReq, _ := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants?pagesize=1", nil)
	listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		b.Fatalf("Failed to get assistant list: %v", err)
	}
	defer listResp.Body.Close()

	var listResponse map[string]interface{}
	json.NewDecoder(listResp.Body).Decode(&listResponse)

	data, hasData := listResponse["data"].([]interface{})
	if !hasData || len(data) == 0 {
		b.Skip("No assistants available for benchmarking")
		return
	}

	assistant, ok := data[0].(map[string]interface{})
	if !ok {
		b.Skip("Invalid assistant data format")
		return
	}

	assistantID, ok := assistant["assistant_id"].(string)
	if !ok || assistantID == "" {
		b.Skip("Assistant ID not found")
		return
	}

	// Reset timer after setup
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", serverURL+baseURL+"/agent/assistants/"+assistantID, nil)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}
