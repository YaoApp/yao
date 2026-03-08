package tunnel

import (
	"bufio"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yaoapp/yao/tai/registry"
)

// HandleProxy handles HTTP reverse proxy requests for a tunnel-connected Tai:
// ANY /tai/:taiID/proxy/*path
// Opens a data channel to Tai's HTTP port, forwards the HTTP request,
// and streams the response back.
func HandleProxy(c *gin.Context) {
	logger := slog.Default()
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registry not initialized"})
		return
	}

	taiID := c.Param("taiID")
	node, ok := reg.Get(taiID)
	if !ok || node.Status != "online" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "tai node not available"})
		return
	}

	httpPort := node.Ports["http"]
	if httpPort == 0 {
		httpPort = 8099
	}

	channelID, resultCh, err := reg.RequestChannel(taiID, httpPort)
	if err != nil {
		logger.Error("request channel failed", "tai_id", taiID, "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "tunnel channel failed"})
		return
	}

	remoteConn, ok := <-resultCh
	if !ok || remoteConn == nil {
		logger.Error("data channel timeout", "tai_id", taiID, "channel_id", channelID)
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "data channel timeout"})
		return
	}
	defer remoteConn.Close()

	path := c.Param("path")
	outReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, "http://tai-tunnel"+path, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "build request failed"})
		return
	}
	outReq.Header = c.Request.Header.Clone()
	outReq.Host = c.Request.Host

	if err := outReq.Write(remoteConn); err != nil {
		logger.Error("write request to tunnel", "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "write to tunnel failed"})
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(remoteConn), outReq)
	if err != nil {
		logger.Error("read response from tunnel", "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "read from tunnel failed"})
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// HandleVNC handles VNC WebSocket proxying for a tunnel-connected Tai:
// GET /tai/:taiID/vnc/*path
// Upgrades the client connection to WebSocket, opens a data channel to
// Tai's VNC port, and bridges the two WebSocket connections.
func HandleVNC(c *gin.Context) {
	logger := slog.Default()
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registry not initialized"})
		return
	}

	taiID := c.Param("taiID")
	node, ok := reg.Get(taiID)
	if !ok || node.Status != "online" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "tai node not available"})
		return
	}

	vncPort := node.Ports["vnc"]
	if vncPort == 0 {
		vncPort = 16080
	}

	channelID, resultCh, err := reg.RequestChannel(taiID, vncPort)
	if err != nil {
		logger.Error("request vnc channel failed", "tai_id", taiID, "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "tunnel channel failed"})
		return
	}

	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("ws upgrade client failed", "err", err)
		return
	}

	taiConn, ok := <-resultCh
	if !ok || taiConn == nil {
		logger.Error("vnc data channel timeout", "tai_id", taiID, "channel_id", channelID)
		clientConn.Close()
		return
	}

	bridgeWSToConn(clientConn, taiConn)
}

// bridgeWSToConn bridges a client WebSocket to a net.Conn (tunnel data channel).
func bridgeWSToConn(clientWS *websocket.Conn, taiConn net.Conn) {
	done := make(chan struct{}, 2)

	// client WS -> tai conn
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, data, err := clientWS.ReadMessage()
			if err != nil {
				return
			}
			if _, err := taiConn.Write(data); err != nil {
				return
			}
		}
	}()

	// tai conn -> client WS
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, err := taiConn.Read(buf)
			if n > 0 {
				if wErr := clientWS.WriteMessage(websocket.BinaryMessage, buf[:n]); wErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	<-done
	clientWS.Close()
	taiConn.Close()
	<-done
}
