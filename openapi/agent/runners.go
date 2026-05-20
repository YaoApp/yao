package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/tai/registry"
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
	nodesByRunner := runnerNodes()

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

func runnerNodes() map[string][]string {
	result := map[string][]string{}
	reg := registry.Global()
	if reg == nil {
		return result
	}

	for _, node := range reg.List() {
		if node.Status != "online" && node.Status != "" {
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
