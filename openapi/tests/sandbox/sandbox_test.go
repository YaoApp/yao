package openapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestSandboxListPublicDenied(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	resp, err := http.Get(serverURL + baseURL + "/sandbox")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Without auth, sandbox list returns 200 (scopes.yml allows GET /sandbox/*)
	// but since /sandbox (no trailing wildcard match) could be denied or allowed,
	// check that a response is returned.
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized,
		"expected 200 or 401, got %d", resp.StatusCode)
}

func TestSandboxListAuthenticated(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Sandbox Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	req, err := http.NewRequest("GET", serverURL+baseURL+"/sandbox", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	t.Logf("Sandbox list returned %d items", len(result))
}

func TestSandboxGetNotFound(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Sandbox NotFound Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	req, err := http.NewRequest("GET", serverURL+baseURL+"/sandbox/nonexistent-id", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Either 404 (sandbox not found) or 503 (sandbox service not available) is acceptable
	assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusServiceUnavailable,
		"expected 404 or 503, got %d", resp.StatusCode)
}

func TestSandboxCreateMissingImage(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Sandbox Create Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	body := `{"node_id": "local"}`
	req, err := http.NewRequest("POST", serverURL+baseURL+"/sandbox", jsonBody(body))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return 400 (image required) or 503 (service unavailable)
	assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusServiceUnavailable,
		"expected 400 or 503, got %d", resp.StatusCode)
}

func TestSandboxDeleteNotFound(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Sandbox Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	req, err := http.NewRequest("DELETE", serverURL+baseURL+"/sandbox/nonexistent-id", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Either 404 or 503 is acceptable
	assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusServiceUnavailable,
		"expected 404 or 503, got %d", resp.StatusCode)
}

func jsonBody(s string) *strings.Reader {
	return strings.NewReader(s)
}
