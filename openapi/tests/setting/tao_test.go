package setting_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/tests/testutils"
	"github.com/yaoapp/yao/setting"
)

// ----------- Functional tests (system:root token) -----------

func TestTaoGetConfig_Unconfigured(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	req, err := http.NewRequest("GET", serverURL+baseURL()+"/setting/tao/config", nil)
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

	assert.Equal(t, "unconfigured", body["status"])
	assert.Equal(t, "", body["key"])
	assert.Equal(t, "", body["base_url"])
	assert.Equal(t, false, body["balance_available"])

	svc, ok := body["services"].(map[string]interface{})
	assert.True(t, ok, "services should be a map")
	assert.Equal(t, false, svc["llm"])
	assert.Equal(t, false, svc["search"])
}

func TestTaoGetConfig_Unauthenticated(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	req, err := http.NewRequest("GET", serverURL+baseURL()+"/setting/tao/config", nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestTaoVerify_MissingKey(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	payload := map[string]interface{}{}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", serverURL+baseURL()+"/setting/tao/verify?locale=en-us", bytes.NewReader(raw))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "verify without key should return 400")
}

func TestTaoBalance_Unconfigured(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)

	// Clean any existing tao config
	if setting.Global != nil {
		setting.Global.Delete(setting.ScopeID{Scope: setting.ScopeUser, UserID: "test"}, "tao")
	}

	token := obtainToken(t, serverURL)

	req, err := http.NewRequest("POST", serverURL+baseURL()+"/setting/tao/balance", bytes.NewReader([]byte("{}")))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "balance when not configured should return 400")
}

func TestTaoSignupGift(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)
	token := obtainToken(t, serverURL)

	req, err := http.NewRequest("GET", serverURL+baseURL()+"/setting/tao/signup-gift?locale=en-us", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// This calls the real Tao upstream — may fail in CI without network.
	// Accept 200 (success) or 502 (network blocked).
	if resp.StatusCode == http.StatusBadGateway {
		t.Skip("Tao upstream unreachable (network blocked), skipping signup-gift test")
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Contains(t, body, "signup_gift")
}

// ----------- ACL permission tests -----------

func TestTaoACL_ReadOnlyScopeCannotWrite(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)

	readToken := obtainRestrictedToken(t, serverURL, "setting:tao:read:all")

	// GET config should work
	req, _ := http.NewRequest("GET", serverURL+baseURL()+"/setting/tao/config", nil)
	req.Header.Set("Authorization", "Bearer "+readToken)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "read-only scope should allow GET config")

	// POST verify should be denied
	payload := map[string]interface{}{"key": "sk-test"}
	raw, _ := json.Marshal(payload)
	req2, _ := http.NewRequest("POST", serverURL+baseURL()+"/setting/tao/verify?locale=en-us", bytes.NewReader(raw))
	req2.Header.Set("Authorization", "Bearer "+readToken)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	assert.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp2.StatusCode, "read-only scope should deny POST verify")
}

func TestTaoACL_NoScopeCannotAccess(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	initSettingRegistry(t)

	noSettingToken := obtainRestrictedToken(t, serverURL, "kb:collections:read:all")

	req, _ := http.NewRequest("GET", serverURL+baseURL()+"/setting/tao/config", nil)
	req.Header.Set("Authorization", "Bearer "+noSettingToken)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "token without setting scope should be denied")
}
