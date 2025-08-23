package openapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/dsl/types"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestDSLCreate tests the DSL creation endpoint
func TestDSLCreate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "DSL Create Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Generate unique test ID
	testID := fmt.Sprintf("test_model_%d", time.Now().UnixNano())

	// Model test data
	modelSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Test User Model" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "name", "type": "string", "length": 80, "comment": "User Name", "index": true },
    { "name": "status", "type": "enum", "option": ["active", "disabled"], "default": "active", "comment": "Status", "index": true }
  ],
  "tags": ["test_%s"],
  "label": "Test Model",
  "description": "Test Model Description",
  "option": { "timestamps": true, "soft_deletes": true }
}`, testID, testID, testID)

	// Test creation with different stores
	stores := []string{"db", "file"}

	for _, store := range stores {
		t.Run(fmt.Sprintf("CreateModel_%s", store), func(t *testing.T) {
			// testutils.Prepare request body
			createData := map[string]interface{}{
				"id":     testID + "_" + store,
				"source": modelSource,
				"store":  store,
			}

			body, err := json.Marshal(createData)
			assert.NoError(t, err)

			// Create HTTP request
			req, err := http.NewRequest("POST", serverURL+baseURL+"/dsl/create/model", bytes.NewBuffer(body))
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
			assert.Equal(t, "DSL created successfully", response["message"])

			t.Logf("Successfully created model DSL with store: %s", store)
		})
	}
}

// TestDSLInspect tests the DSL inspection endpoint
func TestDSLInspect(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "DSL Inspect Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testID := fmt.Sprintf("test_inspect_%d", time.Now().UnixNano())
	modelSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Test Inspect Model" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "name", "type": "string", "length": 80, "comment": "User Name", "index": true }
  ],
  "tags": ["test_inspect"],
  "label": "Test Inspect Model",
  "description": "Test Model for Inspection",
  "option": { "timestamps": true }
}`, testID, testID)

	// First create a model
	createData := map[string]interface{}{
		"id":     testID,
		"source": modelSource,
		"store":  "db",
	}

	body, _ := json.Marshal(createData)
	req, _ := http.NewRequest("POST", serverURL+baseURL+"/dsl/create/model", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	createResp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer createResp.Body.Close()
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Now test inspection
	req, err = http.NewRequest("GET", serverURL+baseURL+"/dsl/inspect/model/"+testID, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var info types.Info
	err = json.NewDecoder(resp.Body).Decode(&info)
	assert.NoError(t, err)

	// Verify the inspection results
	assert.Equal(t, testID, info.ID)
	assert.Equal(t, types.TypeModel, info.Type)
	assert.Equal(t, "Test Inspect Model", info.Label)
	assert.Equal(t, "Test Model for Inspection", info.Description)
	assert.Contains(t, info.Tags, "test_inspect")
	assert.False(t, info.Readonly)
	assert.False(t, info.Builtin)

	t.Logf("Successfully inspected model DSL: %+v", info)
}

// TestDSLSource tests the DSL source retrieval endpoint
func TestDSLSource(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "DSL Source Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testID := fmt.Sprintf("test_source_%d", time.Now().UnixNano())
	modelSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Test Source Model" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "email", "type": "string", "length": 100, "comment": "Email", "index": true }
  ],
  "tags": ["test_source"],
  "label": "Test Source Model",
  "description": "Test Model for Source Retrieval"
}`, testID, testID)

	// Create model first
	createData := map[string]interface{}{
		"id":     testID,
		"source": modelSource,
		"store":  "db",
	}

	body, err := json.Marshal(createData)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", serverURL+baseURL+"/dsl/create/model", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	createResp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)
	defer createResp.Body.Close()

	// Test source retrieval
	req, err = http.NewRequest("GET", serverURL+baseURL+"/dsl/source/model/"+testID, nil)
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

	sourceReturned := response["source"].(string)
	assert.Equal(t, modelSource, sourceReturned)

	t.Logf("Successfully retrieved model DSL source")
}

// TestDSLList tests the DSL listing endpoint
func TestDSLList(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "DSL List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create multiple test models
	testTag := fmt.Sprintf("test_list_%d", time.Now().UnixNano())

	for i := 0; i < 3; i++ {
		testID := fmt.Sprintf("test_list_model_%d_%d", time.Now().UnixNano(), i)
		modelSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Test List Model %d" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "title", "type": "string", "length": 100 }
  ],
  "tags": ["%s"],
  "label": "Test List Model %d"
}`, testID, testID, i, testTag, i)

		createData := map[string]interface{}{
			"id":     testID,
			"source": modelSource,
			"store":  "db",
		}

		body, err := json.Marshal(createData)
		assert.NoError(t, err)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/dsl/create/model", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		createResp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, createResp)
		createResp.Body.Close()
	}

	// Test listing all models
	req, err := http.NewRequest("GET", serverURL+baseURL+"/dsl/list/model", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var data []interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, len(data), 3, "Should have at least 3 models")

	// Test listing with tags filter
	req, err = http.NewRequest("GET", serverURL+baseURL+"/dsl/list/model?tags="+testTag, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var taggedData []interface{}
	err = json.NewDecoder(resp.Body).Decode(&taggedData)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, len(taggedData), 3, "Should find the tagged models")

	t.Logf("Successfully listed %d model DSLs", len(data))
}

// TestDSLUpdate tests the DSL update endpoint
func TestDSLUpdate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "DSL Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testID := fmt.Sprintf("test_update_%d", time.Now().UnixNano())

	// Original model
	originalSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Original Model" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "name", "type": "string", "length": 80 }
  ],
  "tags": ["test_update"],
  "label": "Original Model"
}`, testID, testID)

	// Updated model
	updatedSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Updated Model" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "name", "type": "string", "length": 80 },
    { "name": "email", "type": "string", "length": 100 }
  ],
  "tags": ["test_update", "updated"],
  "label": "Updated Model",
  "description": "Updated model description"
}`, testID, testID)

	// Create original model
	createData := map[string]interface{}{
		"id":     testID,
		"source": originalSource,
		"store":  "db",
	}

	body, err := json.Marshal(createData)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", serverURL+baseURL+"/dsl/create/model", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	createResp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)
	createResp.Body.Close()

	// Update model
	updateData := map[string]interface{}{
		"id":     testID,
		"source": updatedSource,
	}

	body, err = json.Marshal(updateData)
	assert.NoError(t, err)
	req, err = http.NewRequest("PUT", serverURL+baseURL+"/dsl/update/model", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "DSL updated successfully", response["message"])

	// Verify the update by inspecting the model
	req, err = http.NewRequest("GET", serverURL+baseURL+"/dsl/inspect/model/"+testID, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	inspectResp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, inspectResp)
	defer inspectResp.Body.Close()

	var info types.Info
	err = json.NewDecoder(inspectResp.Body).Decode(&info)
	assert.NoError(t, err)

	assert.Equal(t, "Updated Model", info.Label)
	assert.Equal(t, "Updated model description", info.Description)
	assert.Contains(t, info.Tags, "updated")

	t.Logf("Successfully updated model DSL")
}

// TestDSLExists tests the DSL existence check endpoint
func TestDSLExists(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "DSL Exists Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testID := fmt.Sprintf("test_exists_%d", time.Now().UnixNano())
	nonExistentID := fmt.Sprintf("non_existent_%d", time.Now().UnixNano())

	// Test non-existent model first
	req, err := http.NewRequest("GET", serverURL+baseURL+"/dsl/exists/model/"+nonExistentID, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response["exists"].(bool))

	// Create a model
	modelSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s" },
  "columns": [{"name": "id", "type": "ID"}]
}`, testID, testID)

	createData := map[string]interface{}{
		"id":     testID,
		"source": modelSource,
		"store":  "db",
	}

	body, err := json.Marshal(createData)
	assert.NoError(t, err)
	createReq, err := http.NewRequest("POST", serverURL+baseURL+"/dsl/create/model", bytes.NewBuffer(body))
	assert.NoError(t, err)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	createResp, err := http.DefaultClient.Do(createReq)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)
	createResp.Body.Close()

	// Test existing model
	req, err = http.NewRequest("GET", serverURL+baseURL+"/dsl/exists/model/"+testID, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["exists"].(bool))

	t.Logf("Successfully tested model DSL existence")
}

// TestDSLDelete tests the DSL deletion endpoint
func TestDSLDelete(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "DSL Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	testID := fmt.Sprintf("test_delete_%d", time.Now().UnixNano())

	// Create a model first
	modelSource := fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s" },
  "columns": [{"name": "id", "type": "ID"}]
}`, testID, testID)

	createData := map[string]interface{}{
		"id":     testID,
		"source": modelSource,
		"store":  "db",
	}

	body, err := json.Marshal(createData)
	assert.NoError(t, err)
	createReq, err := http.NewRequest("POST", serverURL+baseURL+"/dsl/create/model", bytes.NewBuffer(body))
	assert.NoError(t, err)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	createResp, err := http.DefaultClient.Do(createReq)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)
	createResp.Body.Close()

	// Delete the model
	req, err := http.NewRequest("DELETE", serverURL+baseURL+"/dsl/delete/model/"+testID, nil)
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
	assert.Equal(t, "DSL deleted successfully", response["message"])

	// Verify deletion by checking existence
	existsReq, err := http.NewRequest("GET", serverURL+baseURL+"/dsl/exists/model/"+testID, nil)
	assert.NoError(t, err)
	existsReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	existsResp, err := http.DefaultClient.Do(existsReq)
	assert.NoError(t, err)
	assert.NotNil(t, existsResp)
	defer existsResp.Body.Close()

	var existsResponse map[string]interface{}
	err = json.NewDecoder(existsResp.Body).Decode(&existsResponse)
	assert.NoError(t, err)
	assert.False(t, existsResponse["exists"].(bool))

	t.Logf("Successfully deleted model DSL")
}

// TestDSLValidate tests the DSL validation endpoint
func TestDSLValidate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "DSL Validate Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	tests := []struct {
		name        string
		source      string
		description string
	}{
		{
			name: "ValidModel",
			source: `{
  "name": "valid_model",
  "table": { "name": "valid_model" },
  "columns": [
    {"name": "id", "type": "ID"},
    {"name": "name", "type": "string", "length": 80}
  ]
}`,
			description: "Valid model definition",
		},
		{
			name: "AnotherModel",
			source: `{
  "name": "another_model",
  "table": { "name": "another_model" },
  "columns": [
    {"name": "id", "type": "ID"},
    {"name": "title", "type": "string", "length": 100}
  ]
}`,
			description: "Another model definition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := map[string]string{
				"source": tt.source,
			}

			body, err := json.Marshal(requestBody)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", serverURL+baseURL+"/dsl/validate/model", bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			// Just verify the response has the expected structure
			assert.Contains(t, response, "valid")
			assert.Contains(t, response, "messages")

			valid, ok := response["valid"].(bool)
			assert.True(t, ok, "valid should be a boolean")

			if messages, ok := response["messages"]; ok {
				t.Logf("Validation messages for %s: %v", tt.description, messages)
			}

			t.Logf("Successfully validated %s: valid=%v", tt.description, valid)
		})
	}
}

// TestDSLUnauthorized tests that endpoints return 401 when not authenticated
func TestDSLUnauthorized(t *testing.T) {
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
		{"GET", "/dsl/inspect/model/test", ""},
		{"GET", "/dsl/source/model/test", ""},
		{"GET", "/dsl/list/model", ""},
		{"GET", "/dsl/exists/model/test", ""},
		{"POST", "/dsl/create/model", `{"id":"test","source":"{}"}`},
		{"PUT", "/dsl/update/model", `{"id":"test","source":"{}"}`},
		{"DELETE", "/dsl/delete/model/test", ""},
		{"POST", "/dsl/validate/model", `{"source":"{}"}`},
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
