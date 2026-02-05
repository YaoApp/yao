package sandbox

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/sandbox/vncproxy"
)

var vncProxy *vncproxy.Proxy

// Attach attaches sandbox handlers to the router group
// Routes:
//   - GET /sandbox/:id/vnc - Get VNC status
//   - GET /sandbox/:id/vnc/client - Get noVNC client page
//   - GET /sandbox/:id/vnc/ws - WebSocket proxy to container VNC
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	// Initialize VNC proxy lazily on first request
	// This avoids startup errors if Docker is not available

	// VNC status endpoint
	group.GET("/:id/vnc", oauth.Guard, handleVNCStatus)

	// VNC client page
	group.GET("/:id/vnc/client", oauth.Guard, handleVNCClient)

	// VNC WebSocket proxy
	group.GET("/:id/vnc/ws", oauth.Guard, handleVNCWebSocket)
}

// ensureProxy ensures the VNC proxy is initialized
func ensureProxy() error {
	if vncProxy != nil {
		return nil
	}

	var err error
	vncProxy, err = vncproxy.NewProxy(nil)
	return err
}

// handleVNCStatus returns VNC status for a sandbox container
func handleVNCStatus(c *gin.Context) {
	if err := ensureProxy(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "VNC service not available",
		})
		return
	}

	// Rewrite path to match vncproxy expected format
	sandboxID := c.Param("id")
	c.Request.URL.Path = "/v1/sandbox/" + sandboxID + "/vnc"

	vncProxy.HandleVNCStatus(c.Writer, c.Request)
}

// handleVNCClient serves the noVNC client page
func handleVNCClient(c *gin.Context) {
	if err := ensureProxy(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "VNC service not available",
		})
		return
	}

	// Rewrite path to match vncproxy expected format
	sandboxID := c.Param("id")
	c.Request.URL.Path = "/v1/sandbox/" + sandboxID + "/vnc/client"

	vncProxy.HandleVNCClient(c.Writer, c.Request)
}

// handleVNCWebSocket proxies WebSocket to container VNC
func handleVNCWebSocket(c *gin.Context) {
	if err := ensureProxy(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "VNC service not available",
		})
		return
	}

	// Rewrite path to match vncproxy expected format
	sandboxID := c.Param("id")
	c.Request.URL.Path = "/v1/sandbox/" + sandboxID + "/vnc/ws"

	vncProxy.HandleVNCWebSocket(c.Writer, c.Request)
}

// Close closes the VNC proxy and releases resources
func Close() error {
	if vncProxy != nil {
		return vncProxy.Close()
	}
	return nil
}

// pathPrefix stores the router path prefix for sandbox endpoints
var pathPrefix string = "/v1/sandbox"

// SetPathPrefix sets the path prefix for sandbox URLs
// Called during router setup with the actual OpenAPI base URL
func SetPathPrefix(prefix string) {
	pathPrefix = strings.TrimSuffix(prefix, "/") + "/sandbox"
}

// GetVNCClientURL returns the API VNC client page URL
// sandboxID is the sandbox identifier (userID-chatID)
// Returns the URL path like "/v1/sandbox/{id}/vnc/client"
// Note: For CUI navigation, use "$dashboard/sandbox/{id}" directly with sandbox_id
func GetVNCClientURL(sandboxID string) string {
	if sandboxID == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/vnc/client", pathPrefix, sandboxID)
}
