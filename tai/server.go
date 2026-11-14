// Package tai integrates the tai-golang transport server with Yao.
package tai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yaoapp/tai-golang/node"
	"github.com/yaoapp/tai-golang/pb"
	"github.com/yaoapp/tai-golang/transport"
	taitypes "github.com/yaoapp/tai-golang/types"
	"github.com/yaoapp/tai-golang/workspace"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth"
)

// Config controls the embedded Tai transport server.
type Config struct {
	TransportPort int
	OAuth         OAuthTokens
	IssueNode     func(context.Context, string) (*oauth.NodeToken, error)
	Observer      NodeObserver
}

// OAuthTokens is the subset of OAuth node-token operations used by Tai.
type OAuthTokens interface {
	ValidateNodeToken(context.Context, string) (*oauth.NodeInfo, error)
	RefreshNodeToken(context.Context, string) (*oauth.NodeToken, error)
}

// NodeObserver receives node registration/disconnection events.
type NodeObserver interface {
	NodeRegistered(context.Context, Node) error
	NodeDisconnected(context.Context, string)
}

// Node is a neutral snapshot of a registered Tai node.
type Node struct {
	ID                string
	Version           string
	OS                string
	Arch              string
	Capabilities      []string
	ContainerRuntimes []Runtime
	Online            bool
}

// Runtime is one container runtime advertised by Tai.
type Runtime struct {
	Name    string
	Version string
	Socket  string
}

// Server owns the embedded tai-golang transport.
type Server struct {
	tr     *transport.Server
	reg    *node.Registry
	auth   *AuthProvider
	port   int
	mu     sync.Mutex
	closed bool
}

var active atomic.Pointer[Server]

// NewServer creates a Tai transport server. Yao's config.Conf.Port is the
// SDK HTTP port: its discovery and transport handlers are mounted on Yao's
// existing HTTP server by openapi, so the SDK's standalone HTTP listener is
// intentionally not started.
func NewServer(cfg Config) (*Server, error) {
	if cfg.OAuth == nil {
		return nil, errors.New("tai: oauth service is required")
	}
	if cfg.TransportPort == 0 {
		cfg.TransportPort = 6099
	}
	auth := &AuthProvider{OAuth: cfg.OAuth, IssueNode: cfg.IssueNode}
	codec, err := transport.NewFrameCodec(0)
	if err != nil {
		return nil, err
	}
	pool := defaultPoolConfig()
	reg := node.NewRegistry(pool, workspace.DefaultPolicy())
	reg.SetAuth(auth)
	tr := transport.NewServer(auth, reg, codec)
	transport.SetDefault(tr)
	reg.SetOnRegister(func(rn taitypes.RemoteNode) error {
		if cfg.Observer == nil {
			return nil
		}
		return cfg.Observer.NodeRegistered(context.Background(), snapshot(rn))
	})
	reg.SetOnDisconnect(func(nodeID string) {
		if cfg.Observer != nil {
			cfg.Observer.NodeDisconnected(context.Background(), nodeID)
		}
	})
	tr.InitLightPool(pool)
	tr.InitHeavyPool(pool)
	return &Server{tr: tr, reg: reg, auth: auth, port: cfg.TransportPort}, nil
}

func defaultPoolConfig() *pb.PoolConfig {
	return &pb.PoolConfig{
		LightMinConns: 2, LightMaxConns: 8, LightMaxStreams: 256,
		LightIdleTimeoutSec: 300, HeavyMinIdle: 2, HeavyMaxActive: 20,
		HeavyMaxIdle: 6, HeavyIdleTimeoutSec: 300, HeavyHealthCheckSec: 30,
	}
}

// Start binds the TCP transport. HTTP handlers are already mounted on Yao's
// HTTP server and become usable as soon as the active server is set.
func (s *Server) Start(ctx context.Context) error {
	if s == nil || s.tr == nil {
		return errors.New("tai: server is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	addr := fmt.Sprintf(":%d", s.port)
	if err := s.tr.StartTCP(addr, nil); err != nil {
		return err
	}
	active.Store(s)
	return nil
}

// Close stops the transport.
func (s *Server) Close() error {
	if s == nil || s.tr == nil {
		return nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()
	if active.Load() == s {
		active.Store(nil)
	}
	s.tr.CloseHeavyPool()
	s.tr.CloseLightPool()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.tr.Close(ctx)
}

// TransportPort returns the bound Tai transport port, or 0 when not started.
func (s *Server) TransportPort() int {
	if s == nil || s.tr == nil {
		return 0
	}
	_, port, err := net.SplitHostPort(s.tr.TCPAddr())
	if err != nil {
		return 0
	}
	value, _ := strconv.Atoi(port)
	return value
}

// HTTPHandler exposes the SDK transport HTTP handlers for mounting on Yao's
// existing router: /v1/tai/connect, /v1/tai/control, /v1/tai/light, ...
func HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server := active.Load()
		if server == nil || server.tr == nil || server.auth == nil {
			http.Error(w, "tai transport is not running", http.StatusServiceUnavailable)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tai/token/refresh":
			refreshNodeToken(server.auth, w, r)
			return
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tai/heartbeat":
			heartbeatNode(server.auth, w, r)
			return
		}
		server.tr.Handler().ServeHTTP(w, r)
	})
}

func refreshNodeToken(auth *AuthProvider, w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeToken string `json:"node_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	token, err := auth.RefreshNodeToken(req.NodeToken)
	if err != nil {
		http.Error(w, "refresh failed", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"node_token": token, "expires_in": 7 * 24 * 3600})
}

func heartbeatNode(auth *AuthProvider, w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeToken string `json:"node_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if _, err := auth.ValidateNodeToken(req.NodeToken); err != nil {
		http.Error(w, "invalid node_token", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// WellKnownInfo returns the transport discovery payload for the active server.
func WellKnownInfo() *taitypes.WellKnownResponse {
	server := active.Load()
	if server == nil {
		return nil
	}
	port := server.TransportPort()
	if port == 0 {
		return nil
	}
	return &taitypes.WellKnownResponse{
		TransportPort: port,
		Protocols:     []string{"HTTP/2", "HTTP/3"},
		Version:       "yao",
	}
}

func snapshot(rn taitypes.RemoteNode) Node {
	info := rn.Info()
	n := Node{Online: rn.Online()}
	if info == nil {
		return n
	}
	n.ID = info.ID
	n.Version = info.Version
	n.OS = info.OS
	n.Arch = info.Arch
	n.Capabilities = append([]string(nil), info.Capabilities...)
	for _, rt := range info.ContainerRuntimes {
		n.ContainerRuntimes = append(n.ContainerRuntimes, Runtime{Name: rt.Name, Version: rt.Version, Socket: rt.Socket})
	}
	return n
}

// Provider reads registered Tai nodes from the SDK's default registry.
type Provider struct{}

// Node returns a registered node snapshot.
func (Provider) Node(nodeID string) (*Node, bool) {
	if node.Default() == nil {
		return nil, false
	}
	rn := node.Get(nodeID)
	if rn == nil {
		return nil, false
	}
	n := snapshot(rn)
	return &n, true
}

// ListWorkspaces returns the node's workspace registry entries.
func (Provider) ListWorkspaces(ctx context.Context, nodeID string) ([]taitypes.RegistryEntry, error) {
	if node.Default() == nil {
		return nil, fmt.Errorf("tai node registry is not initialized")
	}
	rn := node.Get(nodeID)
	if rn == nil {
		return nil, fmt.Errorf("tai node %s is not registered", nodeID)
	}
	return rn.ListWorkspaces(ctx, taitypes.ListOptions{NodeID: nodeID})
}

// AuthProvider adapts Yao OAuth node tokens to the Tai SDK AuthProvider.
type AuthProvider struct {
	OAuth     OAuthTokens
	IssueNode func(context.Context, string) (*oauth.NodeToken, error)
}

// ValidateNodeToken implements taitypes.AuthProvider.
func (a *AuthProvider) ValidateNodeToken(raw string) (*taitypes.NodeInfo, error) {
	if a == nil || a.OAuth == nil {
		return nil, errors.New("tai: oauth service is not configured")
	}
	info, err := a.OAuth.ValidateNodeToken(context.Background(), raw)
	if err != nil {
		return nil, err
	}
	return &taitypes.NodeInfo{NodeID: info.NodeID, UserID: info.UserID}, nil
}

// IssueNodeToken implements taitypes.AuthProvider.
func (a *AuthProvider) IssueNodeToken(nodeID string) (string, error) {
	if a == nil || a.IssueNode == nil {
		return "", errors.New("tai: node token issuer is not configured")
	}
	token, err := a.IssueNode(context.Background(), nodeID)
	if err != nil {
		return "", err
	}
	return token.Token, nil
}

// RefreshNodeToken implements taitypes.AuthProvider.
func (a *AuthProvider) RefreshNodeToken(oldToken string) (string, error) {
	if a == nil || a.OAuth == nil {
		return "", errors.New("tai: oauth service is not configured")
	}
	token, err := a.OAuth.RefreshNodeToken(context.Background(), oldToken)
	if err != nil {
		return "", err
	}
	return token.Token, nil
}

// IssueUserToken is not implemented in the first computer phase.
func (a *AuthProvider) IssueUserToken(string, taitypes.Scope) (string, *taitypes.UserIdentity, error) {
	return "", nil, errors.New("tai: user tokens are not implemented")
}

// RefreshUserToken is not implemented in the first computer phase.
func (a *AuthProvider) RefreshUserToken(string) (string, error) {
	return "", errors.New("tai: user tokens are not implemented")
}

// ValidateUserToken is not implemented in the first computer phase.
func (a *AuthProvider) ValidateUserToken(string) (*taitypes.UserIdentity, error) {
	return nil, errors.New("tai: user tokens are not implemented")
}

// WellKnownInfo implements taitypes.AuthProvider.
func (a *AuthProvider) WellKnownInfo() *taitypes.WellKnownResponse {
	return WellKnownInfo()
}

var _ taitypes.AuthProvider = (*AuthProvider)(nil)

// ServerURL returns the cloud-reachable Tai transport HTTP URL. Cloud hosts
// must be able to reach this address for node registration.
func ServerURL() string {
	if value := strings.TrimRight(os.Getenv("YAO_SERVER_URL"), "/"); value != "" {
		return value
	}
	host := config.Conf.Host
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	port := config.Conf.Port
	if port == 0 {
		port = 5099
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}
