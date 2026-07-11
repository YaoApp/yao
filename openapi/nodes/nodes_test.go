package nodes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/nodes"
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

func registerCloudAndTunnelNodes(reg *registry.Registry) {
	reg.Register(&registry.TaiNode{
		TaiID: "tai-cloud",
		Mode:  "cloud",
	})
	reg.Register(&registry.TaiNode{
		TaiID: "tai-tunnel",
		Mode:  "tunnel",
		Auth:  taitypes.AuthInfo{UserID: "user-owner"},
	})
}

func newNodesRouter() *gin.Engine {
	router := gin.New()
	group := router.Group("/nodes")
	nodes.Attach(group, authOAuth{})
	return router
}

func listNodes(t *testing.T, router *gin.Engine, userID, teamID string) []map[string]interface{} {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	if userID != "" {
		req.Header.Set("X-Test-User-ID", userID)
	}
	if teamID != "" {
		req.Header.Set("X-Test-Team-ID", teamID)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	return result
}

func nodeIDs(nodes []map[string]interface{}) []string {
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if id, ok := n["tai_id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func TestListNodes_CloudVisibleToAll(t *testing.T) {
	teardown := setupMockRegistry(t)
	defer teardown()

	reg := registry.Global()
	registerCloudAndTunnelNodes(reg)

	router := newNodesRouter()

	ownerNodes := listNodes(t, router, "user-owner", "")
	assert.ElementsMatch(t, []string{"tai-cloud", "tai-tunnel"}, nodeIDs(ownerNodes))

	otherUserNodes := listNodes(t, router, "user-other", "")
	assert.ElementsMatch(t, []string{"tai-cloud"}, nodeIDs(otherUserNodes))

	thirdUserNodes := listNodes(t, router, "user-third", "")
	assert.ElementsMatch(t, []string{"tai-cloud"}, nodeIDs(thirdUserNodes))
}

func TestListNodes_TunnelOnlyOwner(t *testing.T) {
	teardown := setupMockRegistry(t)
	defer teardown()

	reg := registry.Global()
	reg.Register(&registry.TaiNode{
		TaiID: "tai-cloud",
		Mode:  "cloud",
	})
	reg.Register(&registry.TaiNode{
		TaiID: "tai-tunnel-user",
		Mode:  "tunnel",
		Auth:  taitypes.AuthInfo{UserID: "alice"},
	})
	reg.Register(&registry.TaiNode{
		TaiID: "tai-tunnel-team",
		Mode:  "tunnel",
		Auth:  taitypes.AuthInfo{TeamID: "team-a"},
	})

	router := newNodesRouter()

	aliceNodes := listNodes(t, router, "alice", "")
	assert.Contains(t, nodeIDs(aliceNodes), "tai-tunnel-user")
	assert.NotContains(t, nodeIDs(aliceNodes), "tai-tunnel-team")

	bobNodes := listNodes(t, router, "bob", "")
	assert.Contains(t, nodeIDs(bobNodes), "tai-cloud")
	assert.NotContains(t, nodeIDs(bobNodes), "tai-tunnel-user")
	assert.NotContains(t, nodeIDs(bobNodes), "tai-tunnel-team")

	teamANodes := listNodes(t, router, "member-a", "team-a")
	assert.Contains(t, nodeIDs(teamANodes), "tai-tunnel-team")
	assert.NotContains(t, nodeIDs(teamANodes), "tai-tunnel-user")

	teamBNodes := listNodes(t, router, "member-b", "team-b")
	assert.Contains(t, nodeIDs(teamBNodes), "tai-cloud")
	assert.NotContains(t, nodeIDs(teamBNodes), "tai-tunnel-team")
}
