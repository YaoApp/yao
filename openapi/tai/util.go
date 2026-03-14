package tai

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// extractContainerID parses container ID from *path param.
// /{containerID}/ws → containerID
func extractContainerID(path string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/ws")
	path = strings.TrimSuffix(path, "/")
	if path == "" || path == "__host__" {
		return "__host__"
	}
	return path
}

// bridgeWebSocket copies messages bidirectionally between two WebSocket connections.
func bridgeWebSocket(client, target *websocket.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		defer func() { done <- struct{}{} }()
		for {
			mt, data, err := client.ReadMessage()
			if err != nil {
				return
			}
			if err := target.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()

	go func() {
		defer func() { done <- struct{}{} }()
		for {
			mt, data, err := target.ReadMessage()
			if err != nil {
				return
			}
			if err := client.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()

	<-done
}

// reverseProxy forwards an HTTP request to targetURL and streams the response back.
func reverseProxy(c *gin.Context, targetURL string) {
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create proxy request: " + err.Error()})
		return
	}
	for k, vv := range c.Request.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "proxy request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Flush()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			c.Writer.Flush()
		}
		if readErr != nil {
			return
		}
	}
}
