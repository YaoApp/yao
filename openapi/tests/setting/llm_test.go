package setting_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func requireOpenAIKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("OPENAI_TEST_KEY")
	if key == "" {
		t.Skip("OPENAI_TEST_KEY not set")
	}
	return key
}

func initLLMRegistry(t *testing.T) {
	t.Helper()
	if err := llmprovider.Init(); err != nil {
		t.Fatalf("llmprovider.Init: %v", err)
	}
	if config.Conf.DB.AESKey != "" {
		llmprovider.Global.SetEncryptionKey(config.Conf.DB.AESKey)
	}
}

func llmURL(serverURL, path string) string {
	return serverURL + baseURL() + "/setting/llm" + path
}

func llmGet(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	return resp
}

func llmPost(t *testing.T, url, token string, payload interface{}) *http.Response {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, _ := json.Marshal(payload)
		body = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	return resp
}

func llmPut(t *testing.T, url, token string, payload interface{}) *http.Response {
	t.Helper()
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	return resp
}

func llmDelete(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	return resp
}

func llmBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	return body
}

func createTestOpenAI(t *testing.T, serverURL, token string) string {
	t.Helper()
	apiKey := requireOpenAIKey(t)
	payload := map[string]interface{}{
		"preset_key": "openai",
		"api_key":    apiKey,
		"model_ids":  []string{"gpt-4o", "gpt-4o-mini"},
	}
	resp := llmPost(t, llmURL(serverURL, "/providers"), token, payload)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "createTestOpenAI should succeed")
	body := llmBody(t, resp)
	scopedKey, _ := body["key"].(string)
	t.Cleanup(func() { llmprovider.Global.Delete(scopedKey) })
	return scopedKey
}

// ----------- Functional tests -----------

func TestLLMGetPageData(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	createTestOpenAI(t, serverURL, token)

	resp := llmGet(t, llmURL(serverURL, ""), token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := llmBody(t, resp)
	assert.Contains(t, body, "providers")
	assert.Contains(t, body, "roles")
	assert.Contains(t, body, "preset_providers")

	providers, ok := body["providers"].([]interface{})
	assert.True(t, ok, "providers should be an array")
	assert.GreaterOrEqual(t, len(providers), 1)

	if len(providers) > 0 {
		p := providers[0].(map[string]interface{})
		assert.Contains(t, p, "key")
		assert.Contains(t, p, "name")
		assert.Contains(t, p, "models")
		assert.NotContains(t, p, "connector_id", "internal field should be stripped")
		assert.NotContains(t, p, "source", "internal field should be stripped")
		assert.NotContains(t, p, "owner", "internal field should be stripped")
	}

	presets, ok := body["preset_providers"].([]interface{})
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(presets), 5, "should have at least 5 presets")
}

func TestLLMGetUnauthenticated(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	req, _ := http.NewRequest("GET", llmURL(serverURL, ""), nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLLMProviderCreate(t *testing.T) {
	realKey := requireOpenAIKey(t)
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	payload := map[string]interface{}{
		"preset_key": "openai",
		"api_key":    realKey,
		"model_ids":  []string{"gpt-4o"},
	}
	resp := llmPost(t, llmURL(serverURL, "/providers"), token, payload)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body := llmBody(t, resp)
	scopedKey, _ := body["key"].(string)
	t.Cleanup(func() { llmprovider.Global.Delete(scopedKey) })

	assert.Contains(t, scopedKey, ".openai", "scoped key should end with .openai")
	assert.Equal(t, "OpenAI", body["name"])
	assert.Equal(t, "openai", body["type"])

	apiKey, _ := body["api_key"].(string)
	assert.NotEqual(t, realKey, apiKey, "API key should be masked")
	assert.NotEmpty(t, apiKey)

	models, _ := body["models"].([]interface{})
	assert.Equal(t, 1, len(models))
}

func TestLLMProviderCreateCustom(t *testing.T) {
	realKey := requireOpenAIKey(t)
	mirror := os.Getenv("TEST_MOAPI_MIRROR")
	if mirror == "" {
		mirror = "https://api.openai.com"
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	payload := map[string]interface{}{
		"key":     "my-custom-llm",
		"name":    "My Custom LLM",
		"type":    "openai",
		"api_url": mirror,
		"api_key": realKey,
		"models": []map[string]interface{}{
			{"id": "custom-model", "name": "Custom Model", "capabilities": []string{"streaming"}, "enabled": true},
		},
		"require_key": true,
	}
	resp := llmPost(t, llmURL(serverURL, "/providers"), token, payload)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body := llmBody(t, resp)
	scopedKey, _ := body["key"].(string)
	t.Cleanup(func() { llmprovider.Global.Delete(scopedKey) })

	assert.Contains(t, scopedKey, ".my-custom-llm", "scoped key should end with .my-custom-llm")
	assert.Equal(t, "My Custom LLM", body["name"])
	assert.Equal(t, true, body["is_custom"])

	models, _ := body["models"].([]interface{})
	assert.Equal(t, 1, len(models))
}

func TestLLMProviderUpdate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	scopedKey := createTestOpenAI(t, serverURL, token)

	updatePayload := map[string]interface{}{
		"name":    "Updated OpenAI",
		"api_url": "https://api.openai.com/v2",
		"models": []map[string]interface{}{
			{"id": "gpt-4o", "name": "GPT-4o Updated", "capabilities": []string{"vision", "tool_calls"}, "enabled": true},
		},
	}
	resp := llmPut(t, llmURL(serverURL, "/providers/"+scopedKey), token, updatePayload)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := llmBody(t, resp)
	assert.Equal(t, "Updated OpenAI", body["name"])
	assert.Equal(t, "https://api.openai.com/v2", body["api_url"])

	apiKey, _ := body["api_key"].(string)
	assert.NotEmpty(t, apiKey, "API key should be preserved when not sent")
}

func TestLLMProviderDelete(t *testing.T) {
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	createPayload := map[string]interface{}{
		"preset_key": "anthropic",
		"api_key":    anthropicKey,
	}
	createResp := llmPost(t, llmURL(serverURL, "/providers"), token, createPayload)
	createBody := llmBody(t, createResp)
	createResp.Body.Close()
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)
	scopedKey, _ := createBody["key"].(string)

	rolesPayload := map[string]interface{}{
		"default": map[string]interface{}{
			"provider": scopedKey,
			"model":    "claude-sonnet-4-6",
		},
	}
	rolesResp := llmPut(t, llmURL(serverURL, "/roles"), token, rolesPayload)
	rolesResp.Body.Close()
	assert.Equal(t, http.StatusOK, rolesResp.StatusCode)

	deleteResp := llmDelete(t, llmURL(serverURL, "/providers/"+scopedKey), token)
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)

	body := llmBody(t, deleteResp)
	assert.Equal(t, true, body["success"])
	assert.NotEmpty(t, body["warning"], "should warn about cleared roles")

	getResp := llmGet(t, llmURL(serverURL, ""), token)
	defer getResp.Body.Close()
	getBody := llmBody(t, getResp)
	roles, _ := getBody["roles"].(map[string]interface{})
	assert.NotContains(t, roles, "default", "role referencing deleted provider should be cleared")
}

func TestLLMProviderDeleteForbidden(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	otherOwner := llmprovider.ProviderOwner{Type: "user", UserID: "some-other-user-999"}
	scopedKey := llmprovider.ScopedKey(&otherOwner, "other-team-provider")
	otherProvider := &llmprovider.Provider{
		Key:     scopedKey,
		Name:    "Other Team's Provider",
		Type:    "openai",
		APIURL:  "https://api.example.com",
		Models:  []llmprovider.ModelInfo{},
		Enabled: true,
		Source:  llmprovider.ProviderSourceDynamic,
		Owner:   otherOwner,
	}
	llmprovider.Global.Create(otherProvider)
	t.Cleanup(func() { llmprovider.Global.Delete(scopedKey) })

	resp := llmDelete(t, llmURL(serverURL, "/providers/"+scopedKey), token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "should not be able to delete another user's provider")
}

func TestLLMProviderTest(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	scopedKey := createTestOpenAI(t, serverURL, token)

	resp := llmPost(t, llmURL(serverURL, "/providers/"+scopedKey+"/test"), token, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := llmBody(t, resp)
	assert.Equal(t, true, body["success"])
	assert.NotEmpty(t, body["message"])
	assert.NotNil(t, body["latency_ms"])
}

func TestLLMRoles(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)
	scopedKey := createTestOpenAI(t, serverURL, token)

	rolesPayload := map[string]interface{}{
		"default": map[string]interface{}{
			"provider": scopedKey,
			"model":    "gpt-4o",
		},
		"vision": map[string]interface{}{
			"provider": scopedKey,
			"model":    "gpt-4o",
		},
	}
	resp := llmPut(t, llmURL(serverURL, "/roles"), token, rolesPayload)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := llmBody(t, resp)
	assert.Contains(t, body, "default")
	assert.Contains(t, body, "vision")

	getResp := llmGet(t, llmURL(serverURL, ""), token)
	defer getResp.Body.Close()
	getBody := llmBody(t, getResp)
	roles, _ := getBody["roles"].(map[string]interface{})
	assert.Contains(t, roles, "default")
	assert.Contains(t, roles, "vision")
}

func TestLLMRolesValidation(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)
	token := obtainToken(t, serverURL)

	resp1 := llmPut(t, llmURL(serverURL, "/roles"), token, map[string]interface{}{
		"vision": map[string]interface{}{
			"provider": "openai",
			"model":    "gpt-4o",
		},
	})
	defer resp1.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp1.StatusCode, "should require 'default' role")

	resp2 := llmPut(t, llmURL(serverURL, "/roles"), token, map[string]interface{}{
		"default": map[string]interface{}{
			"provider": "nonexistent-provider",
			"model":    "some-model",
		},
	})
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode, "should reject non-existent provider")

	scopedKey := createTestOpenAI(t, serverURL, token)

	resp3 := llmPut(t, llmURL(serverURL, "/roles"), token, map[string]interface{}{
		"default": map[string]interface{}{
			"provider": scopedKey,
			"model":    "nonexistent-model",
		},
	})
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp3.StatusCode, "should reject non-existent model")
}

// ----------- ACL permission tests -----------

func TestLLMACL_ReadOnlyScopeCannotWrite(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)

	readToken := obtainRestrictedToken(t, serverURL, "setting:llm:read:all")

	resp := llmGet(t, llmURL(serverURL, ""), readToken)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "read-only scope should allow GET")

	createPayload := map[string]interface{}{
		"preset_key": "openai",
		"api_key":    "sk-test",
	}
	resp2 := llmPost(t, llmURL(serverURL, "/providers"), readToken, createPayload)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp2.StatusCode, "read-only scope should deny POST")
}

func TestLLMACL_NoScopeCannotAccess(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	initLLMRegistry(t)

	noSettingToken := obtainRestrictedToken(t, serverURL, "kb:collections:read:all")

	resp := llmGet(t, llmURL(serverURL, ""), noSettingToken)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "token without setting scope should be denied")
}
