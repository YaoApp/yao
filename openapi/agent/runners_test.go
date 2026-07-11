package agent_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/agent"
	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupMockRegistry(t *testing.T) func() {
	t.Helper()
	reg := registry.NewForTest()
	registry.SetGlobalForTest(reg)
	return func() {
		registry.SetGlobalForTest(nil)
	}
}

func setAuthContext(c *gin.Context, userID, teamID string) {
	c.Set("__subject", "test-subject")
	c.Set("__client_id", "test-client")
	c.Set("__scope", "openid profile")
	if userID != "" {
		c.Set("__user_id", userID)
	}
	if teamID != "" {
		c.Set("__team_id", teamID)
	}
}

func listRunners(t *testing.T, userID, teamID string) map[string][]string {
	t.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/agent/runners", nil)
	setAuthContext(c, userID, teamID)

	agent.ListRunners(c)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp struct {
		Runners []struct {
			Name      string   `json:"name"`
			Available bool     `json:"available"`
			Nodes     []string `json:"nodes"`
		} `json:"runners"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	nodesByRunner := make(map[string][]string, len(resp.Runners))
	for _, r := range resp.Runners {
		nodesByRunner[r.Name] = r.Nodes
	}
	return nodesByRunner
}

func runnerNodes(nodesByRunner map[string][]string, runner string) []string {
	nodes, ok := nodesByRunner[runner]
	if !ok {
		return nil
	}
	return nodes
}

func TestRunnersACL(t *testing.T) {
	teardown := setupMockRegistry(t)
	defer teardown()

	reg := registry.Global()
	reg.Register(&registry.TaiNode{
		TaiID: "tai-cloud",
		Mode:  "cloud",
		Capabilities: taitypes.Capabilities{
			Runners: []string{"claude"},
		},
	})
	reg.Register(&registry.TaiNode{
		TaiID: "tai-tunnel",
		Mode:  "tunnel",
		Auth:  taitypes.AuthInfo{UserID: "user-owner"},
		Capabilities: taitypes.Capabilities{
			Runners: []string{"claude"},
		},
	})

	ownerRunners := listRunners(t, "user-owner", "")
	assert.Contains(t, runnerNodes(ownerRunners, "claude"), "tai-cloud")
	assert.Contains(t, runnerNodes(ownerRunners, "claude"), "tai-tunnel")

	otherRunners := listRunners(t, "user-other", "")
	assert.Contains(t, runnerNodes(otherRunners, "claude"), "tai-cloud")
	assert.NotContains(t, runnerNodes(otherRunners, "claude"), "tai-tunnel")
}
