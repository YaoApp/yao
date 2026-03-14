package registry

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/yao/tai/types"
)

// TaiNode represents a registered Tai instance (direct or tunnel).
// Internal use only; external callers receive types.NodeMeta via Get()/List().
type TaiNode struct {
	TaiID        string
	MachineID    string
	Version      string
	Auth         types.AuthInfo
	System       types.SystemInfo
	Mode         string // "direct" | "tunnel"
	Addr         string // direct mode: "tai-host"; tunnel mode: empty
	YaoBase      string // Yao server base URL reported by Tai (tunnel mode)
	Ports        types.Ports
	Capabilities types.Capabilities

	registerStream any // taipb.TaiTunnel_RegisterServer (stored as any to avoid import cycle)

	Status      string // "online" | "offline" | "connecting"
	ConnectedAt time.Time
	LastPing    time.Time
	DisplayName string // optional human-readable name for UI

	resources any // *tai.ConnResources; stored as any to avoid import cycle

	localListeners map[int]*tunnelListener
}

func (n *TaiNode) meta() types.NodeMeta {
	return types.NodeMeta{
		TaiID: n.TaiID, MachineID: n.MachineID, Version: n.Version,
		Auth: n.Auth, System: n.System,
		Mode: n.Mode, Addr: n.Addr, YaoBase: n.YaoBase,
		Ports: n.Ports, Capabilities: n.Capabilities,
		Status: n.Status, ConnectedAt: n.ConnectedAt, LastPing: n.LastPing,
		DisplayName: n.DisplayName,
	}
}

// tunnelListener wraps a TCP listener that bridges each accepted connection
// through the tunnel to a specific Tai port.
type tunnelListener struct {
	listener net.Listener
	taiID    string
	port     int
	cancel   func()
}

var (
	global *Registry
	once   sync.Once
)

// BridgeFunc bridges a local TCP connection to a target port on a tunnel node.
// Set via SetBridgeFunc once the gRPC tunnel handler is ready.
type BridgeFunc func(taiID string, targetPort int, localConn net.Conn)

// Registry manages all Tai nodes (direct and tunnel).
type Registry struct {
	mu       sync.RWMutex
	nodes    map[string]*TaiNode
	logger   *slog.Logger
	bridgeFn BridgeFunc
	bridgeMu sync.RWMutex
}

// Init initializes the global registry singleton.
func Init(logger *slog.Logger) {
	once.Do(func() {
		if logger == nil {
			logger = slog.Default()
		}
		global = &Registry{
			nodes:  make(map[string]*TaiNode),
			logger: logger,
		}
	})
}

// InitWithWriter initializes the global registry using the given io.Writer
// and log format ("JSON" or "TEXT"). If w is nil it falls back to stderr.
// This is the preferred way to integrate with the application log system.
func InitWithWriter(w io.Writer, logMode string) {
	if w == nil {
		w = os.Stderr
	}
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	var handler slog.Handler
	if strings.EqualFold(logMode, "JSON") {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	Init(slog.New(handler))
}

// Global returns the global registry instance.
func Global() *Registry {
	return global
}

// Register adds or updates a Tai node in the registry.
func (r *Registry) Register(node *TaiNode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	node.Status = "online"
	node.ConnectedAt = time.Now()
	node.LastPing = time.Now()
	if node.localListeners == nil {
		node.localListeners = make(map[int]*tunnelListener)
	}
	r.nodes[node.TaiID] = node

	r.logger.Info("tai node registered",
		"tai_id", node.TaiID, "mode", node.Mode, "version", node.Version)
}

// Unregister removes a Tai node, closes its local listeners,
// and any held ConnResources.
func (r *Registry) Unregister(taiID string) {
	r.mu.Lock()
	node, ok := r.nodes[taiID]
	if ok {
		for _, tl := range node.localListeners {
			tl.cancel()
			tl.listener.Close()
		}
		delete(r.nodes, taiID)
	}
	r.mu.Unlock()

	if ok {
		if node.resources != nil {
			if closer, ok := node.resources.(ResourceCloser); ok {
				closer.Close()
			}
		}
		r.logger.Info("tai node unregistered", "tai_id", taiID)
	}
}

// Get returns the metadata of a Tai node by ID. Returns nil, false if not found.
func (r *Registry) Get(taiID string) (*types.NodeMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[taiID]
	if !ok {
		return nil, false
	}
	m := n.meta()
	return &m, true
}

// List returns metadata of all registered Tai nodes.
func (r *Registry) List() []types.NodeMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]types.NodeMeta, 0, len(r.nodes))
	for _, n := range r.nodes {
		result = append(result, n.meta())
	}
	return result
}

// UpdatePing records a heartbeat timestamp.
func (r *Registry) UpdatePing(taiID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[taiID]; ok {
		n.LastPing = time.Now()
	}
}

// ResourceCloser is implemented by *tai.ConnResources to allow the registry
// to close resources without importing the tai package (avoids import cycle).
type ResourceCloser interface {
	Close() error
}

// SetResources binds connection resources to a registered node.
// If the node already has resources, the old ones are closed asynchronously.
// The node status is set to "online".
func (r *Registry) SetResources(taiID string, res any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.nodes[taiID]
	if !ok {
		return
	}
	if n.resources != nil {
		if closer, ok := n.resources.(ResourceCloser); ok {
			go closer.Close()
		}
	}
	n.resources = res
	n.Status = "online"
}

// GetResources returns the *tai.ConnResources for a node (as any).
// Callers should type-assert to *tai.ConnResources.
func (r *Registry) GetResources(taiID string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[taiID]
	if !ok || n.resources == nil {
		return nil, false
	}
	return n.resources, true
}

// SetBridgeFunc sets the function used by OpenLocalListener to bridge
// TCP connections through the gRPC tunnel (Forward stream).
func (r *Registry) SetBridgeFunc(fn BridgeFunc) {
	r.bridgeMu.Lock()
	defer r.bridgeMu.Unlock()
	r.bridgeFn = fn
}

// SetRegisterStream stores the gRPC Register stream for a tunnel node.
func (r *Registry) SetRegisterStream(taiID string, stream any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[taiID]; ok {
		n.registerStream = stream
	}
}

// GetRegisterStream returns the gRPC Register stream for a tunnel node.
func (r *Registry) GetRegisterStream(taiID string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if n, ok := r.nodes[taiID]; ok {
		return n.registerStream
	}
	return nil
}

// GenerateChannelID creates a random channel ID for Forward stream matching.
func GenerateChannelID() (string, error) {
	return generateChannelID()
}

// FindTaiIDByAuthClient returns the TaiID of the first node whose
// Auth.ClientID matches the given OAuth client ID. Returns "" if not found.
func (r *Registry) FindTaiIDByAuthClient(clientID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, n := range r.nodes {
		if n.Auth.ClientID == clientID {
			return n.TaiID
		}
	}
	return ""
}

// ListByTeam returns metadata of all nodes belonging to the given team.
func (r *Registry) ListByTeam(teamID string) []types.NodeMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []types.NodeMeta
	for _, n := range r.nodes {
		if n.Auth.TeamID == teamID {
			result = append(result, n.meta())
		}
	}
	return result
}

// ListByUser returns metadata of all nodes registered by the given user
// that are NOT associated with any team.
func (r *Registry) ListByUser(userID string) []types.NodeMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []types.NodeMeta
	for _, n := range r.nodes {
		if n.Auth.TeamID == "" && n.Auth.UserID == userID {
			result = append(result, n.meta())
		}
	}
	return result
}

// StartHealthCheck runs a background goroutine that periodically checks
// direct-mode nodes for heartbeat timeout. Nodes whose LastPing exceeds
// timeout are marked offline. Nodes that remain offline longer than
// cleanupAfter are automatically unregistered.
// The goroutine stops when ctx.Done() is closed.
func (r *Registry) StartHealthCheck(done <-chan struct{}, interval, timeout, cleanupAfter time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				r.checkHealth(timeout, cleanupAfter)
			}
		}
	}()
}

func (r *Registry) checkHealth(timeout, cleanupAfter time.Duration) {
	now := time.Now()
	var toRemove []string

	r.mu.Lock()
	for id, n := range r.nodes {
		if n.Mode != "direct" {
			continue
		}
		elapsed := now.Sub(n.LastPing)
		if n.Status == "online" && elapsed > timeout {
			n.Status = "offline"
			r.logger.Warn("tai node offline (heartbeat timeout)",
				"tai_id", id, "last_ping", n.LastPing)
		}
		if n.Status == "offline" && elapsed > timeout+cleanupAfter {
			toRemove = append(toRemove, id)
		}
	}
	r.mu.Unlock()

	for _, id := range toRemove {
		r.logger.Info("tai node auto-unregistered (offline too long)", "tai_id", id)
		r.Unregister(id)
	}
}

// OpenLocalListener creates a localhost TCP listener that tunnels every
// accepted connection to the specified port on the given Tai node.
// Returns the listener address (e.g. "127.0.0.1:54321").
func (r *Registry) OpenLocalListener(taiID string, targetPort int) (net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	ctx, cancel := newContext()
	tl := &tunnelListener{listener: ln, taiID: taiID, port: targetPort, cancel: cancel}

	r.mu.Lock()
	node := r.nodes[taiID]
	if node == nil {
		r.mu.Unlock()
		cancel()
		ln.Close()
		return nil, fmt.Errorf("tai node %s not found", taiID)
	}
	node.localListeners[targetPort] = tl
	r.mu.Unlock()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					r.logger.Debug("tunnel listener accept error", "err", err)
					return
				}
			}
			go r.bridgeTunnelConn(taiID, targetPort, conn)
		}
	}()

	r.logger.Info("tunnel local listener started",
		"tai_id", taiID, "target_port", targetPort, "local_addr", ln.Addr().String())
	return ln, nil
}

func (r *Registry) bridgeTunnelConn(taiID string, targetPort int, localConn net.Conn) {
	r.bridgeMu.RLock()
	fn := r.bridgeFn
	r.bridgeMu.RUnlock()

	if fn != nil {
		fn(taiID, targetPort, localConn)
		return
	}

	localConn.Close()
	r.logger.Error("no bridge function configured", "tai_id", taiID, "port", targetPort)
}

// ChannelIDBytes is the number of random bytes used to generate a channel ID.
// The resulting hex string is 2× this value (64 characters).
const ChannelIDBytes = 32

// ChannelIDShortLen is the max characters shown in log messages.
const ChannelIDShortLen = 16

func generateChannelID() (string, error) {
	b := make([]byte, ChannelIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ShortChannelID truncates a channel ID for log display.
func ShortChannelID(id string) string {
	if len(id) <= ChannelIDShortLen {
		return id
	}
	return id[:ChannelIDShortLen]
}

type contextCancel struct {
	done chan struct{}
}

func newContext() (*contextCancel, func()) {
	cc := &contextCancel{done: make(chan struct{})}
	return cc, func() { close(cc.done) }
}

func (c *contextCancel) Done() <-chan struct{} {
	return c.done
}
