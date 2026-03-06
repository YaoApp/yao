package tunnel

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	oauth "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/tai/registry"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleControl handles the Tai control channel WebSocket: GET /ws/tai.
// Authenticates via Bearer token, reads register + ping messages,
// and maintains the Tai node in the global registry.
func HandleControl(c *gin.Context) {
	logger := slog.Default()
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registry not initialized"})
		return
	}

	bearer := extractBearer(c.Request)
	if bearer == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
		return
	}

	authInfo, err := authenticateBearerFunc(bearer)
	if err != nil {
		logger.Warn("tunnel auth failed", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("ws upgrade failed", "err", err)
		return
	}

	// Read the register message
	var regMsg registerMessage
	if err := conn.ReadJSON(&regMsg); err != nil {
		logger.Error("read register message", "err", err)
		conn.Close()
		return
	}
	if regMsg.Type != "register" {
		logger.Error("expected register message", "got", regMsg.Type)
		conn.Close()
		return
	}
	if regMsg.TaiID == "" {
		logger.Error("register message missing tai_id")
		conn.Close()
		return
	}

	node := &registry.TaiNode{
		TaiID:        regMsg.TaiID,
		MachineID:    regMsg.MachineID,
		Version:      regMsg.Version,
		Auth:         authInfo,
		Mode:         "tunnel",
		YaoBase:      regMsg.Server,
		Ports:        regMsg.Ports,
		Capabilities: regMsg.Capabilities,
		ControlConn:  conn,
	}
	reg.Register(node)
	defer func() {
		reg.Unregister(regMsg.TaiID)
		logger.Info("tai tunnel disconnected", "tai_id", regMsg.TaiID)
	}()

	if err := reg.WriteControlJSON(regMsg.TaiID, map[string]string{"type": "registered", "tai_id": regMsg.TaiID}); err != nil {
		logger.Error("write registered response", "err", err)
		return
	}

	logger.Info("tai tunnel connected", "tai_id", regMsg.TaiID, "version", regMsg.Version)

	for {
		var msg controlMsg
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Debug("control channel read error", "err", err)
			}
			return
		}

		switch msg.Type {
		case "ping":
			reg.UpdatePing(regMsg.TaiID)
			if err := reg.WriteControlJSON(regMsg.TaiID, map[string]string{"type": "pong"}); err != nil {
				logger.Debug("pong write failed", "err", err)
				return
			}
		default:
			logger.Debug("unknown control message", "type", msg.Type)
		}
	}
}

// HandleData handles a Tai data channel WebSocket: GET /ws/tai/data/:channel_id.
// Authenticates via Bearer token, verifies the caller matches the pending
// channel's owner, then wraps the WS as a net.Conn for bidirectional bridging.
func HandleData(c *gin.Context) {
	logger := slog.Default()
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registry not initialized"})
		return
	}

	bearer := extractBearer(c.Request)
	if bearer == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
		return
	}
	authInfo, err := authenticateBearerFunc(bearer)
	if err != nil {
		logger.Warn("data channel auth failed", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	channelID := c.Param("channel_id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing channel_id"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("ws data upgrade failed", "err", err)
		return
	}

	wsConn := newWSConn(conn)
	if err := reg.AcceptDataChannel(channelID, authInfo.ClientID, wsConn); err != nil {
		logger.Debug("accept data channel failed", "channel_id", channelID, "err", err)
		conn.Close()
		return
	}
}

// registerMessage is the JSON structure for Tai's register message.
type registerMessage struct {
	Type         string          `json:"type"`
	TaiID        string          `json:"tai_id"`
	MachineID    string          `json:"machine_id"`
	Version      string          `json:"version"`
	Server       string          `json:"server"`
	Ports        map[string]int  `json:"ports"`
	Capabilities map[string]bool `json:"capabilities"`
}

// controlMsg is a generic control channel message.
type controlMsg struct {
	Type string `json:"type"`
}

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return auth[7:]
	}
	return ""
}

var authenticateBearerFunc = authenticateBearerDefault

func authenticateBearerDefault(token string) (registry.AuthInfo, error) {
	svc := oauth.OAuth
	if svc == nil {
		return registry.AuthInfo{}, fmt.Errorf("oauth service not initialized")
	}

	result, err := svc.AuthenticateToken(oauth.AuthInput{
		AccessToken: token,
	})
	if err != nil {
		return registry.AuthInfo{}, err
	}

	info := registry.AuthInfo{}
	if result.Info != nil {
		info.Subject = result.Info.Subject
		info.UserID = result.Info.UserID
		info.ClientID = result.Info.ClientID
		info.Scope = result.Info.Scope
		info.TeamID = result.Info.TeamID
		info.TenantID = result.Info.TenantID
	}
	return info, nil
}

// wsConn wraps a gorilla/websocket.Conn to implement net.Conn for raw byte bridging.
type wsConn struct {
	ws     *websocket.Conn
	reader io.Reader
	mu     sync.Mutex
}

func newWSConn(ws *websocket.Conn) *wsConn {
	return &wsConn{ws: ws}
}

func (c *wsConn) Read(p []byte) (int, error) {
	for {
		if c.reader != nil {
			n, err := c.reader.Read(p)
			if n > 0 {
				return n, nil
			}
			c.reader = nil
			if err != nil && err != io.EOF {
				return 0, err
			}
		}
		_, reader, err := c.ws.NextReader()
		if err != nil {
			return 0, err
		}
		c.reader = reader
	}
}

func (c *wsConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.ws.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *wsConn) Close() error {
	return c.ws.Close()
}

func (c *wsConn) LocalAddr() net.Addr  { return c.ws.LocalAddr() }
func (c *wsConn) RemoteAddr() net.Addr { return c.ws.RemoteAddr() }

func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.ws.SetReadDeadline(t); err != nil {
		return err
	}
	return c.ws.SetWriteDeadline(t)
}

func (c *wsConn) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *wsConn) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }
