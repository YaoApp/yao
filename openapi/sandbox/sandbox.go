package sandbox

import (
	"net/http"

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

	// VNC status endpoint (requires auth)
	group.GET("/:id/vnc", oauth.Guard, handleVNCStatus)

	// VNC client page (requires auth)
	group.GET("/:id/vnc/client", oauth.Guard, handleVNCClient)

	// VNC WebSocket proxy (requires auth)
	// Note: WebSocket upgrade happens after auth middleware
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
