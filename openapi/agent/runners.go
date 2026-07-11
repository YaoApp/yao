package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"
)

var runnerDescriptions = map[string]string{
	"yaocode":  "Local execution, zero dependencies",
	"tai":      "Remote execution (Yao Code SDK)",
	"claude":   "Remote execution (Claude CLI)",
	"opencode": "Remote execution (OpenCode CLI)",
}

// RunnerInfo describes a supported runner for the API response.
type RunnerInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Available   bool     `json:"available"`
	Nodes       []string `json:"nodes"`
}

// ListRunners returns the list of supported runners and their availability.
// GET /api/v1/agent/runners
func ListRunners(c *gin.Context) {
	authInfo := authorized.GetInfo(c)
	nodesByRunner := runnerNodes(authInfo)

	var runners []RunnerInfo
	for _, name := range sandboxv2.SupportedRunners {
		nodes := nodesByRunner[name]
		if nodes == nil {
			nodes = []string{}
		}
		runners = append(runners, RunnerInfo{
			Name:        name,
			Description: runnerDescriptions[name],
			Available:   len(nodes) > 0,
			Nodes:       nodes,
		})
	}

	c.JSON(http.StatusOK, gin.H{"runners": runners})
}

func runnerNodes(authInfo *oauthTypes.AuthorizedInfo) map[string][]string {
	result := map[string][]string{}
	reg := registry.Global()
	if reg == nil {
		return result
	}

	for _, node := range reg.List() {
		if node.Status != "online" && node.Status != "" {
			continue
		}
		if !taitypes.IsPublicNode(node.Mode) && !runnerNodeOwnedBy(&node, authInfo) {
			continue
		}
		runners := node.Capabilities.Runners
		if len(runners) == 0 {
			runners = sandboxv2.InferRunners(node, "")
		}
		for _, r := range runners {
			result[r] = append(result[r], node.TaiID)
		}
	}
	return result
}

func runnerNodeOwnedBy(snap *taitypes.NodeMeta, authInfo *oauthTypes.AuthorizedInfo) bool {
	if authInfo == nil {
		return true
	}
	if authInfo.TeamID != "" {
		return snap.Auth.TeamID == authInfo.TeamID
	}
	if authInfo.UserID != "" {
		return snap.Auth.TeamID == "" && snap.Auth.UserID == authInfo.UserID
	}
	return true
}
