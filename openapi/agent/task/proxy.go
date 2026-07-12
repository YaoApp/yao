package task

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	taiapi "github.com/yaoapp/yao/openapi/tai"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/webproxy"
)

// handleTaskProxyBind creates a port binding (idempotent).
// This is a shortcut that resolves the computer from chatid, then delegates
// to the shared BindAndRespond logic.
func handleTaskProxyBind(c *gin.Context) {
	if webproxy.WP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webproxy service not enabled"})
		return
	}

	var req struct {
		Port  int    `json:"port" binding:"required"`
		Label string `json:"label"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	computer, err := resolveComputer(c)
	if err != nil {
		return
	}
	if computer == nil {
		c.JSON(http.StatusOK, gin.H{"error": "sandbox_not_running"})
		return
	}

	var targetID, containerID, taiID string
	if box, ok := computer.(*sandbox.Box); ok {
		targetID = box.ID()
		containerID = box.ContainerID()
		taiID = box.NodeID()
	} else if host, ok := computer.(*sandbox.Host); ok {
		targetID = webproxy.HostID
		taiID = host.NodeID()
	} else {
		targetID = webproxy.HostID
	}

	useTunnel := false
	if taiID != "" {
		if reg := registry.Global(); reg != nil {
			if node, ok := reg.Get(taiID); ok && node.Mode != "local" {
				useTunnel = true
			}
		}
	}

	taiapi.BindAndRespond(c, webproxy.BindOptions{
		TaiID:       taiID,
		TargetID:    targetID,
		ContainerID: containerID,
		TargetPort:  req.Port,
		Label:       req.Label,
		UseTunnel:   useTunnel,
	})
}

// handleTaskProxyList lists active bindings for the task's sandbox.
func handleTaskProxyList(c *gin.Context) {
	if webproxy.WP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webproxy service not enabled"})
		return
	}

	computer, err := resolveComputer(c)
	if err != nil {
		return
	}
	if computer == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var targetID string
	if box, ok := computer.(*sandbox.Box); ok {
		targetID = box.ID()
	} else {
		targetID = webproxy.HostID
	}

	bindings := webproxy.WP().List(targetID)
	domain, prefix := webproxy.WP().GetConfig()

	result := make([]gin.H, 0, len(bindings))
	for _, b := range bindings {
		result = append(result, gin.H{
			"host_port":   b.HostPort,
			"target_port": b.TargetPort,
			"label":       b.Label,
			"status":      b.Status,
			"url":         taiapi.BuildProxyURL(b.HostPort, domain, prefix),
		})
	}
	c.JSON(http.StatusOK, result)
}

// handleTaskProxyUnbind removes a port binding.
func handleTaskProxyUnbind(c *gin.Context) {
	if webproxy.WP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webproxy service not enabled"})
		return
	}

	hostPort, err := strconv.Atoi(c.Param("hostPort"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host port"})
		return
	}

	if err := webproxy.WP().Unbind(hostPort); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
