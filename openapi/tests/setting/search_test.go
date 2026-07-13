package setting_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// ---------------------------------------------------------------------------
// Functional tests (system:root token)
// ---------------------------------------------------------------------------

func TestSearchGet(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	req, err := http.NewRequest("GET", serverURL+baseURL()+"/setting/search", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if !assert.NoError(t, err) || !assert.NotNil(t, resp) {
		return
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)

	assert.Contains(t, body, "presets")
	assert.Contains(t, body, "providers")
	assert.Contains(t, body, "tool_assignment")

	presets, ok := body["presets"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 5, len(presets), "should have 5 presets: cloud, tavily, serper, brightdata, direct")

	providers, ok := body["providers"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 5, len(providers), "should have 5 provider configs")

	// Cloud provider should be first
	first, _ := providers[0].(map[string]interface{})
	assert.Equal(t, "cloud", first["preset_key"])
}

func TestSearchGetUnauthenticated(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	req, err := http.NewRequest("GET", serverURL+baseURL()+"/setting/search", nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestSearchProviderUpdate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	payload := map[string]interface{}{
		"field_values": map[string]string{
			"api_key": "tvly-test-key-12345",
		},
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/tavily", bytes.NewReader(raw))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "tavily", body["preset_key"])

	// api_key should be masked in response
	fv, _ := body["field_values"].(map[string]interface{})
	maskedKey, _ := fv["api_key"].(string)
	assert.True(t, strings.Contains(maskedKey, "..."), "api_key should be masked")

	// GET should also return masked key
	req2, _ := http.NewRequest("GET", serverURL+baseURL()+"/setting/search", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := http.DefaultClient.Do(req2)
	assert.NoError(t, err)
	defer resp2.Body.Close()

	var getData map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&getData)
	providers, _ := getData["providers"].([]interface{})
	for _, p := range providers {
		pm, _ := p.(map[string]interface{})
		if pm["preset_key"] == "tavily" {
			tfv, _ := pm["field_values"].(map[string]interface{})
			assert.True(t, strings.Contains(tfv["api_key"].(string), "..."), "GET should return masked key")
		}
	}
}

func TestSearchProviderUpdateCloud(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	payload := map[string]interface{}{
		"field_values": map[string]string{},
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/cloud", bytes.NewReader(raw))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "cloud provider should be rejected")
}

func TestSearchProviderToggle(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	// Save tavily first
	savePayload := map[string]interface{}{
		"field_values": map[string]string{"api_key": "tvly-toggle-key"},
	}
	raw, _ := json.Marshal(savePayload)
	req, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/tavily", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	resp.Body.Close()

	// Enable tavily
	enablePayload := map[string]interface{}{"enabled": true}
	raw, _ = json.Marshal(enablePayload)
	req2, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/tavily/toggle", bytes.NewReader(raw))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	assert.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&body)
	assert.Equal(t, true, body["enabled"])

	// Assign tavily to web_search
	assignPayload := map[string]interface{}{"web_search": "tavily"}
	raw, _ = json.Marshal(assignPayload)
	req3, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/tool-assignment", bytes.NewReader(raw))
	req3.Header.Set("Authorization", "Bearer "+token)
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := http.DefaultClient.Do(req3)
	assert.NoError(t, err)
	resp3.Body.Close()

	// Disable tavily -- should clear tool_assignment
	disablePayload := map[string]interface{}{"enabled": false}
	raw, _ = json.Marshal(disablePayload)
	req4, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/tavily/toggle", bytes.NewReader(raw))
	req4.Header.Set("Authorization", "Bearer "+token)
	req4.Header.Set("Content-Type", "application/json")
	resp4, err := http.DefaultClient.Do(req4)
	assert.NoError(t, err)
	resp4.Body.Close()

	// Verify tool_assignment cleared
	req5, _ := http.NewRequest("GET", serverURL+baseURL()+"/setting/search", nil)
	req5.Header.Set("Authorization", "Bearer "+token)
	resp5, err := http.DefaultClient.Do(req5)
	assert.NoError(t, err)
	defer resp5.Body.Close()

	var getData map[string]interface{}
	json.NewDecoder(resp5.Body).Decode(&getData)
	ta, _ := getData["tool_assignment"].(map[string]interface{})
	assert.Nil(t, ta["web_search"], "web_search should be cleared after disabling tavily")
}

func TestSearchToolAssignment(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	// Save and enable tavily
	savePayload := map[string]interface{}{"field_values": map[string]string{"api_key": "tvly-assign-key"}}
	raw, _ := json.Marshal(savePayload)
	req, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/tavily", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	enablePayload := map[string]interface{}{"enabled": true}
	raw, _ = json.Marshal(enablePayload)
	req2, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/tavily/toggle", bytes.NewReader(raw))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := http.DefaultClient.Do(req2)
	resp2.Body.Close()

	// Assign tavily to web_search
	assignPayload := map[string]interface{}{"web_search": "tavily"}
	raw, _ = json.Marshal(assignPayload)
	req3, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/tool-assignment", bytes.NewReader(raw))
	req3.Header.Set("Authorization", "Bearer "+token)
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := http.DefaultClient.Do(req3)
	assert.NoError(t, err)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp3.Body).Decode(&body)
	assert.Equal(t, "tavily", body["web_search"])
}

func TestSearchToolAssignmentDisabledProvider(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	// Try to assign a provider that isn't enabled
	assignPayload := map[string]interface{}{"web_search": "serper"}
	raw, _ := json.Marshal(assignPayload)
	req, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/tool-assignment", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "should reject assignment to disabled provider")
}

func TestSearchProviderTest(t *testing.T) {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		t.Skip("TAVILY_API_KEY not set, skipping search provider test")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	payload := map[string]interface{}{
		"field_values": map[string]string{"api_key": apiKey},
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", serverURL+baseURL()+"/setting/search/providers/tavily/test", bytes.NewReader(raw))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, true, body["success"])
}

func TestSearchProviderTestCloud(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	req, _ := http.NewRequest("POST", serverURL+baseURL()+"/setting/search/providers/cloud/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// ACL permission tests
// ---------------------------------------------------------------------------

func TestSearchACL_ReadOnlyScopeCannotWrite(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)

	readToken := obtainRestrictedToken(t, serverURL, "setting:search:read:all")

	// GET should work
	req, _ := http.NewRequest("GET", serverURL+baseURL()+"/setting/search", nil)
	req.Header.Set("Authorization", "Bearer "+readToken)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "read-only scope should allow GET")

	// PUT should be denied
	payload := map[string]interface{}{
		"field_values": map[string]string{"api_key": "tvly-acl-test"},
	}
	raw, _ := json.Marshal(payload)
	req2, _ := http.NewRequest("PUT", serverURL+baseURL()+"/setting/search/providers/tavily", bytes.NewReader(raw))
	req2.Header.Set("Authorization", "Bearer "+readToken)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	assert.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp2.StatusCode, "read-only scope should deny PUT")
}
