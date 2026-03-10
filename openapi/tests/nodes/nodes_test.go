package openapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestNodesListAuthenticated(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Nodes Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	req, err := http.NewRequest("GET", serverURL+baseURL+"/nodes", nil)
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
	t.Logf("Nodes list returned %d items", len(result))

	for _, node := range result {
		assert.NotEmpty(t, node["tai_id"], "node should have tai_id")
		assert.NotEmpty(t, node["mode"], "node should have mode")
		assert.NotEmpty(t, node["status"], "node should have status")
		t.Logf("Node: tai_id=%s, mode=%s, status=%s", node["tai_id"], node["mode"], node["status"])
	}
}

func TestNodesListUnauthorized(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	resp, err := http.Get(serverURL + baseURL + "/nodes")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNodesGetNotFound(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Nodes Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	req, err := http.NewRequest("GET", serverURL+baseURL+"/nodes/nonexistent-node", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusServiceUnavailable,
		"expected 404 or 503, got %d", resp.StatusCode)
}
