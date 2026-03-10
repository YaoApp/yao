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
	tai "github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/taiid"
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
	if regMsg.NodeID == "" || regMsg.MachineID == "" {
		logger.Error("register message missing node_id or machine_id")
		conn.Close()
		return
	}
	resolvedTaiID, err := taiid.Generate(regMsg.MachineID, regMsg.NodeID)
	if err != nil {
		logger.Error("taiid generation failed", "err", err)
		conn.Close()
		return
	}

	addr := ""
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		addr = "tunnel://" + host
	}

	node := &registry.TaiNode{
		TaiID:        resolvedTaiID,
		MachineID:    regMsg.MachineID,
		Version:      regMsg.Version,
		DisplayName:  regMsg.DisplayName,
		Auth:         authInfo,
		System:       regMsg.System,
		Mode:         "tunnel",
		Addr:         addr,
		YaoBase:      regMsg.Server,
		Ports:        regMsg.Ports,
		Capabilities: regMsg.Capabilities,
		ControlConn:  conn,
	}
	reg.Register(node)
	defer func() {
		reg.Unregister(resolvedTaiID)
		logger.Info("tai tunnel disconnected", "tai_id", resolvedTaiID)
	}()

	if err := reg.WriteControlJSON(resolvedTaiID, map[string]string{"type": "registered", "tai_id": resolvedTaiID}); err != nil {
		logger.Error("write registered response", "err", err)
		return
	}

	logger.Info("tai tunnel connected", "tai_id", resolvedTaiID, "version", regMsg.Version)

	go connectTunnelNode(resolvedTaiID, reg, logger)

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
			reg.UpdatePing(resolvedTaiID)
			if err := reg.WriteControlJSON(resolvedTaiID, map[string]string{"type": "pong"}); err != nil {
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

	resolvedTaiID := reg.FindTaiIDByAuthClient(authInfo.ClientID)
	if resolvedTaiID == "" {
		resolvedTaiID = authInfo.ClientID
	}

	wsConn := newWSConn(conn)
	if err := reg.AcceptDataChannel(channelID, resolvedTaiID, wsConn); err != nil {
		logger.Debug("accept data channel failed", "channel_id", channelID, "err", err,
			"auth_client_id", authInfo.ClientID, "resolved_tai_id", resolvedTaiID)
		conn.Close()
		return
	}
}

// registerMessage is the JSON structure for Tai's register message.
type registerMessage struct {
	Type         string              `json:"type"`
	NodeID       string              `json:"node_id,omitempty"`
	ClientID     string              `json:"client_id,omitempty"`
	MachineID    string              `json:"machine_id"`
	DisplayName  string              `json:"display_name,omitempty"`
	Version      string              `json:"version"`
	Server       string              `json:"server"`
	Ports        map[string]int      `json:"ports"`
	Capabilities map[string]bool     `json:"capabilities"`
	System       registry.SystemInfo `json:"system"`
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

	slog.Info("[tunnel-auth] info from token",
		"subject", info.Subject, "user_id", info.UserID,
		"client_id", info.ClientID, "team_id", info.TeamID,
		"scope", info.Scope)

	if result.Claims != nil {
		slog.Info("[tunnel-auth] claims",
			"claims.TeamID", result.Claims.TeamID,
			"claims.ClientID", result.Claims.ClientID,
			"extra", fmt.Sprintf("%+v", result.Claims.Extra))

		if info.TeamID == "" && result.Claims.TeamID != "" {
			info.TeamID = result.Claims.TeamID
		}
		if info.TeamID == "" {
			switch v := result.Claims.Extra["team_id"].(type) {
			case string:
				info.TeamID = v
			case float64:
				info.TeamID = fmt.Sprintf("%.0f", v)
			}
		}
		if info.TenantID == "" {
			if v, ok := result.Claims.Extra["tenant_id"].(string); ok {
				info.TenantID = v
			}
		}
	}

	slog.Info("[tunnel-auth] final", "team_id", info.TeamID, "client_id", info.ClientID)
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

// connectTunnelNode creates a tai.Client through the tunnel and binds it to the taiID.
func connectTunnelNode(taiID string, reg *registry.Registry, logger *slog.Logger) {
	client, err := tai.New("tunnel://" + taiID)
	if err != nil {
		logger.Warn("failed to connect tunnel node",
			"tai_id", taiID, "err", err)
		return
	}
	_ = client // initTunnel already calls reg.SetClient(taiID, c)
	logger.Info("tai client created for tunnel node", "tai_id", taiID)
}
