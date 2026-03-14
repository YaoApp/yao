package tai

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	yaoTai "github.com/yaoapp/yao/tai"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin:  func(r *http.Request) bool { return true },
	Subprotocols: []string{"binary"},
}

// handleLocalVNC resolves the container's VNC address via Docker socket
// and proxies the WebSocket connection.
func handleLocalVNC(c *gin.Context, taiID string) {
	containerID := extractContainerID(c.Param("path"))
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing container ID in path"})
		return
	}

	res, ok := yaoTai.GetResources(taiID)
	if !ok || res.VNC == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "VNC not available for node " + taiID})
		return
	}

	targetURL, err := res.VNC.URL(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "resolve VNC target: " + err.Error()})
		return
	}

	clientConn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	dialer := websocket.Dialer{
		Subprotocols:     []string{"binary"},
		HandshakeTimeout: 5 * time.Second,
	}
	targetConn, _, err := dialer.Dial(targetURL, nil)
	if err != nil {
		clientConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "VNC connection failed"))
		return
	}
	defer targetConn.Close()

	bridgeWebSocket(clientConn, targetConn)
}
