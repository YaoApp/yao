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

	"github.com/gorilla/websocket"
)

// SystemInfo describes the host machine running Tai.
type SystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	NumCPU   int    `json:"num_cpu"`
	TotalMem int64  `json:"total_mem,omitempty"`
	Shell    string `json:"shell,omitempty"`
	TempDir  string `json:"temp_dir,omitempty"`
}

// TaiNode represents a registered Tai instance (direct or tunnel).
// Internal use only; external callers receive NodeSnapshot via Get()/List().
type TaiNode struct {
	TaiID        string
	MachineID    string
	Version      string
	Auth         AuthInfo
	System       SystemInfo
	Mode         string         // "direct" | "tunnel"
	Addr         string         // direct mode: "tai-host"; tunnel mode: empty
	YaoBase      string         // Yao server base URL reported by Tai (tunnel mode)
	Ports        map[string]int // {"grpc":19100, "http":8099, "vnc":16080, "docker":12375}
	Capabilities map[string]bool

	ControlConn *websocket.Conn
	connMu      sync.Mutex // protects ControlConn writes

	Status      string // "online" | "offline" | "connecting"
	ConnectedAt time.Time
	LastPing    time.Time
	DisplayName string // optional human-readable name for UI

	client any // *tai.Client; stored as any to avoid import cycle

	localListeners map[int]*tunnelListener
}

// NodeSnapshot is a read-only copy of TaiNode fields safe to use outside locks.
type NodeSnapshot struct {
	TaiID        string
	MachineID    string
	Version      string
	Auth         AuthInfo
	System       SystemInfo
	Mode         string
	Addr         string
	YaoBase      string
	Ports        map[string]int
	Capabilities map[string]bool
	Status       string
	ConnectedAt  time.Time
	LastPing     time.Time
	DisplayName  string
	client       any
}

func (n *TaiNode) snapshot() NodeSnapshot {
	ports := make(map[string]int, len(n.Ports))
	for k, v := range n.Ports {
		ports[k] = v
	}
	caps := make(map[string]bool, len(n.Capabilities))
	for k, v := range n.Capabilities {
		caps[k] = v
	}
	return NodeSnapshot{
		TaiID: n.TaiID, MachineID: n.MachineID, Version: n.Version,
		Auth: n.Auth, System: n.System,
		Mode: n.Mode, Addr: n.Addr, YaoBase: n.YaoBase,
		Ports: ports, Capabilities: caps,
		Status: n.Status, ConnectedAt: n.ConnectedAt, LastPing: n.LastPing,
		DisplayName: n.DisplayName,
		client:      n.client,
	}
}

// Client returns the associated *tai.Client (as any to avoid import cycle).
// Callers should type-assert: snap.Client().(*tai.Client).
func (s *NodeSnapshot) Client() any { return s.client }

// AuthInfo holds Yao user authorization extracted from OAuth token.
type AuthInfo struct {
	Subject  string
	UserID   string
	ClientID string
	Scope    string
	TeamID   string
	TenantID string
}

// pendingChannel represents a channel awaiting Tai's data WS connection.
type pendingChannel struct {
	taiID  string
	result chan net.Conn
	timer  *time.Timer
}

// tunnelListener wraps a TCP listener that bridges each accepted connection
// through the WS tunnel to a specific Tai port.
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

// Registry manages all Tai nodes (direct and tunnel).
type Registry struct {
	mu      sync.RWMutex
	nodes   map[string]*TaiNode
	pending map[string]*pendingChannel
	logger  *slog.Logger
}

// Init initializes the global registry singleton.
func Init(logger *slog.Logger) {
	once.Do(func() {
		if logger == nil {
			logger = slog.Default()
		}
		global = &Registry{
			nodes:   make(map[string]*TaiNode),
			pending: make(map[string]*pendingChannel),
			logger:  logger,
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

// Unregister removes a Tai node and closes its local listeners and control connection.
func (r *Registry) Unregister(taiID string) {
	r.mu.Lock()
	node, ok := r.nodes[taiID]
	if ok {
		for _, tl := range node.localListeners {
			tl.cancel()
			tl.listener.Close()
		}
		node.connMu.Lock()
		if node.ControlConn != nil {
			node.ControlConn.Close()
			node.ControlConn = nil
		}
		node.connMu.Unlock()
		delete(r.nodes, taiID)
	}
	r.mu.Unlock()

	if ok {
		r.logger.Info("tai node unregistered", "tai_id", taiID)
	}
}

// Get returns a snapshot of a Tai node by ID. Returns nil, false if not found.
func (r *Registry) Get(taiID string) (*NodeSnapshot, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[taiID]
	if !ok {
		return nil, false
	}
	snap := n.snapshot()
	return &snap, true
}

// List returns snapshots of all registered Tai nodes.
func (r *Registry) List() []NodeSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]NodeSnapshot, 0, len(r.nodes))
	for _, n := range r.nodes {
		result = append(result, n.snapshot())
	}
	return result
}

// WriteControlJSON sends a JSON message on the node's control channel
// with proper serialization. Returns error if node not found or not tunnel.
func (r *Registry) WriteControlJSON(taiID string, v interface{}) error {
	r.mu.RLock()
	node := r.nodes[taiID]
	r.mu.RUnlock()

	if node == nil {
		return fmt.Errorf("tai node %s not found", taiID)
	}

	node.connMu.Lock()
	defer node.connMu.Unlock()
	if node.ControlConn == nil {
		return fmt.Errorf("tai node %s has no active control channel", taiID)
	}
	return node.ControlConn.WriteJSON(v)
}

// UpdatePing records a heartbeat timestamp.
func (r *Registry) UpdatePing(taiID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[taiID]; ok {
		n.LastPing = time.Now()
	}
}

// SetClient associates a *tai.Client with a registered node.
// Called by tai.New() after successful initialization.
func (r *Registry) SetClient(taiID string, c any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[taiID]; ok {
		n.client = c
	}
}

// ListByTeam returns snapshots of all nodes belonging to the given team.
func (r *Registry) ListByTeam(teamID string) []NodeSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []NodeSnapshot
	for _, n := range r.nodes {
		if n.Auth.TeamID == teamID {
			result = append(result, n.snapshot())
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

// RequestChannel sends an "open" command to a tunnel-connected Tai via its
// control channel. Returns a channel_id that Tai will use to connect back.
// Blocks until the data channel is established or timeout.
func (r *Registry) RequestChannel(taiID string, targetPort int) (string, chan net.Conn, error) {
	r.mu.RLock()
	node := r.nodes[taiID]
	r.mu.RUnlock()

	if node == nil {
		return "", nil, fmt.Errorf("tai node %s not found", taiID)
	}
	if node.Mode != "tunnel" {
		return "", nil, fmt.Errorf("tai node %s is not a tunnel node", taiID)
	}
	node.connMu.Lock()
	hasConn := node.ControlConn != nil
	node.connMu.Unlock()
	if !hasConn {
		return "", nil, fmt.Errorf("tai node %s has no active control channel", taiID)
	}

	channelID, err := generateChannelID()
	if err != nil {
		return "", nil, fmt.Errorf("generate channel_id: %w", err)
	}

	resultCh := make(chan net.Conn, 1)
	timer := time.AfterFunc(30*time.Second, func() {
		r.mu.Lock()
		if pc, ok := r.pending[channelID]; ok {
			close(pc.result)
			delete(r.pending, channelID)
		}
		r.mu.Unlock()
	})

	r.mu.Lock()
	r.pending[channelID] = &pendingChannel{taiID: taiID, result: resultCh, timer: timer}
	r.mu.Unlock()

	msg := map[string]interface{}{
		"type":        "open",
		"channel_id":  channelID,
		"target_port": targetPort,
	}
	if err := r.WriteControlJSON(taiID, msg); err != nil {
		r.mu.Lock()
		delete(r.pending, channelID)
		r.mu.Unlock()
		timer.Stop()
		return "", nil, fmt.Errorf("send open command: %w", err)
	}

	return channelID, resultCh, nil
}

// AcceptDataChannel resolves a pending channel when Tai connects its data WS.
// The taiID must match the node that requested the channel via RequestChannel.
func (r *Registry) AcceptDataChannel(channelID, taiID string, conn net.Conn) error {
	r.mu.Lock()
	pc, ok := r.pending[channelID]
	if ok {
		delete(r.pending, channelID)
	}
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("no pending channel for %s", channelID)
	}
	if pc.taiID != taiID {
		pc.timer.Stop()
		close(pc.result)
		return fmt.Errorf("channel %s: tai_id mismatch (expected %s, got %s)", channelID, pc.taiID, taiID)
	}
	pc.timer.Stop()
	pc.result <- conn
	return nil
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
	channelID, resultCh, err := r.RequestChannel(taiID, targetPort)
	if err != nil {
		localConn.Close()
		r.logger.Error("request channel failed", "tai_id", taiID, "port", targetPort, "err", err)
		return
	}

	remoteConn, ok := <-resultCh
	if !ok || remoteConn == nil {
		localConn.Close()
		r.logger.Error("data channel timeout", "tai_id", taiID, "channel_id", channelID)
		return
	}

	bridgeTCP(localConn, remoteConn)
}

// bridgeTCP copies bytes bidirectionally between two net.Conn, closing both when done.
func bridgeTCP(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	cp := func(dst, src net.Conn) {
		defer wg.Done()
		io.Copy(dst, src)
		dst.Close()
	}

	go cp(a, b)
	go cp(b, a)
	wg.Wait()
}

func generateChannelID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
