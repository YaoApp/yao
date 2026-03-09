package tai

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	hepb "github.com/yaoapp/yao/tai/hostexec/pb"
	"github.com/yaoapp/yao/tai/proxy"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/sandbox"
	sipb "github.com/yaoapp/yao/tai/serverinfo/pb"
	"github.com/yaoapp/yao/tai/vnc"
	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/tai/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Runtime selects which container runtime to use via Tai.
type Runtime int

const (
	Docker Runtime = iota
	K8s
)

func (r Runtime) apply(c *config) { c.runtime = r }

// Option configures a Client.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// Ports configures service ports for Tai server.
type Ports struct {
	GRPC   int // default 19100
	HTTP   int // default 8099
	VNC    int // default 16080
	Docker int // default 12375
	K8s    int // default 16443
}

// WithPorts overrides default Tai service ports.
// Ports set here take precedence over server-reported values from ServerInfo.
func WithPorts(p Ports) Option {
	return optionFunc(func(c *config) {
		c.ports = p
		c.userPorts = p
	})
}

// WithHTTPClient sets a custom HTTP client for proxy and VNC health checks.
func WithHTTPClient(hc *http.Client) Option {
	return optionFunc(func(c *config) { c.httpClient = hc })
}

// WithDataDir sets the workspace root directory for Local mode.
func WithDataDir(dir string) Option {
	return optionFunc(func(c *config) { c.dataDir = dir })
}

// WithKubeConfig sets the kubeconfig file path for K8s runtime.
// Supports both absolute and relative paths (relative paths are resolved to absolute).
func WithKubeConfig(path string) Option {
	return optionFunc(func(c *config) { c.kubeConfig = path })
}

// WithNamespace sets the namespace for K8s runtime. Default is "default".
func WithNamespace(ns string) Option {
	return optionFunc(func(c *config) { c.namespace = ns })
}

// WithVolume injects a custom Volume implementation.
// Useful for testing workspace operations without Docker.
func WithVolume(vol volume.Volume) Option {
	return optionFunc(func(c *config) { c.volume = vol })
}

type config struct {
	runtime    Runtime
	ports      Ports
	userPorts  Ports // tracks explicitly set ports (zero = not set by user)
	httpClient *http.Client
	dataDir    string
	kubeConfig string
	namespace  string
	volume     volume.Volume // override volume (for testing without Docker)
}

func defaultPorts() Ports {
	return Ports{
		GRPC: 19100,
		HTTP: 8099,
		VNC:  16080,
	}
}

func mergedPorts(p Ports) Ports {
	d := defaultPorts()
	if p.GRPC != 0 {
		d.GRPC = p.GRPC
	}
	if p.HTTP != 0 {
		d.HTTP = p.HTTP
	}
	if p.VNC != 0 {
		d.VNC = p.VNC
	}
	if p.Docker != 0 {
		d.Docker = p.Docker
	}
	if p.K8s != 0 {
		d.K8s = p.K8s
	}
	return d
}

// Client provides unified access to all Tai SDK sub-packages.
type Client struct {
	scheme   string // "tai", "docker", or "tunnel"
	host     string
	addr     string
	taiID    string // registry key — set by initLocal/initRemote/initTunnel
	ports    Ports
	dataDir  string // host-side data directory for local volume
	vol      volume.Volume
	sb       sandbox.Sandbox
	img      sandbox.Image
	prx      proxy.Proxy
	vc       vnc.VNC
	he       hepb.HostExecClient
	grpcConn *grpc.ClientConn

	// tunnel mode: local listeners that bridge to Tai via WS
	tunnelListeners []net.Listener
}

// New creates a Client based on the address protocol:
//
//	"local"          → Local mode, platform default Docker socket
//	"docker://addr"  → Local mode, specified Docker daemon
//	"tai://host"     → Remote mode via Tai Server
//
// Empty string is not allowed — use "local" for default local Docker.
func New(addr string, opts ...Option) (*Client, error) {
	cfg := &config{ports: defaultPorts()}
	for _, o := range opts {
		o.apply(cfg)
	}
	cfg.ports = mergedPorts(cfg.ports)

	scheme, host, dockerAddr, grpcPort, err := parseAddr(addr)
	if err != nil {
		return nil, err
	}

	if grpcPort > 0 {
		cfg.ports.GRPC = grpcPort
	}

	c := &Client{
		scheme: scheme,
		host:   host,
		addr:   dockerAddr,
		ports:  cfg.ports,
	}

	switch scheme {
	case "docker":
		return c.initLocal(cfg)
	case "tai":
		return c.initRemote(cfg)
	case "tunnel":
		return c.initTunnel(cfg)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}
}

func (c *Client) initLocal(cfg *config) (*Client, error) {
	sb, err := sandbox.NewLocal(c.addr)
	if err != nil && cfg.volume == nil {
		return nil, err
	}
	if sb != nil {
		c.sb = sb
		c.img = sandbox.NewDockerImage(sandbox.DockerCli(sb))
		c.prx = proxy.NewLocal(sb)
		c.vc = vnc.NewLocal(sb)
	}

	if cfg.volume != nil {
		c.vol = cfg.volume
		c.dataDir = cfg.dataDir
	} else {
		dataDir := cfg.dataDir
		if dataDir == "" {
			dataDir = "/tmp/tai-volumes"
		}
		c.dataDir = dataDir
		c.vol = volume.NewLocal(dataDir)
	}

	if reg := registry.Global(); reg != nil {
		id := c.host
		if id == "" {
			id = c.addr
		}
		if id == "" {
			id = "local"
		}
		c.taiID = id
		reg.Register(&registry.TaiNode{
			TaiID: id,
			Mode:  "local",
			Addr:  c.addr,
		})
		reg.SetClient(id, c)
	}
	return c, nil
}

func (c *Client) initRemote(cfg *config) (*Client, error) {
	grpcAddr := fmt.Sprintf("%s:%d", c.host, c.ports.GRPC)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", grpcAddr, err)
	}
	c.grpcConn = conn
	c.he = hepb.NewHostExecClient(conn)

	info, err := c.discoverServerInfo(conn, cfg)
	if err != nil {
		info = &discoveredInfo{Capabilities: map[string]bool{"docker": true}}
	}

	hasDocker := info.Capabilities["docker"]
	hasK8s := info.Capabilities["k8s"]
	hasHostExec := info.Capabilities["host_exec"]

	if !hasDocker && !hasK8s && !hasHostExec {
		conn.Close()
		return nil, fmt.Errorf("tai %s: no capabilities available (docker/k8s/host_exec all false)", c.host)
	}

	c.vol = volume.NewRemote(conn)

	if cfg.runtime == K8s {
		if cfg.kubeConfig == "" {
			conn.Close()
			return nil, fmt.Errorf("tai %s: K8s runtime requested but no kubeconfig provided", c.host)
		}
		k8sPort := c.ports.K8s
		if k8sPort == 0 {
			k8sPort = 16443
		}
		sbAddr := fmt.Sprintf("%s:%d", c.host, k8sPort)
		sb, err := sandbox.NewK8s(sbAddr, sandbox.K8sOption{
			Namespace:  cfg.namespace,
			KubeConfig: cfg.kubeConfig,
		})
		if err == nil {
			c.sb = sb
			c.img = sandbox.NewK8sImage()
		}
	} else if hasDocker {
		dockerPort := c.ports.Docker
		if dockerPort == 0 {
			dockerPort = 12375
		}
		sbAddr := fmt.Sprintf("tcp://%s:%d", c.host, dockerPort)
		sb, err := sandbox.NewDocker(sbAddr)
		if err == nil {
			c.sb = sb
			c.img = sandbox.NewDockerImage(sandbox.DockerCli(sb))
		}
	}

	if c.sb != nil {
		hc := cfg.httpClient
		c.prx = proxy.NewRemote(c.host, c.ports.HTTP, hc)
		c.vc = vnc.NewRemote(c.host, c.ports.VNC, hc)
	}

	if reg := registry.Global(); reg != nil {
		id := fmt.Sprintf("%s-%d", c.host, c.ports.GRPC)
		c.taiID = id
		reg.Register(&registry.TaiNode{
			TaiID:        id,
			Mode:         "direct",
			Version:      info.Version,
			System:       info.System,
			Capabilities: info.Capabilities,
			Addr:         fmt.Sprintf("tai://%s:%d", c.host, c.ports.GRPC),
			Ports: map[string]int{
				"grpc":   c.ports.GRPC,
				"http":   c.ports.HTTP,
				"vnc":    c.ports.VNC,
				"docker": c.ports.Docker,
				"k8s":    c.ports.K8s,
			},
		})
		reg.SetClient(id, c)
	}

	return c, nil
}

func (c *Client) initTunnel(cfg *config) (*Client, error) {
	reg := registry.Global()
	if reg == nil {
		return nil, fmt.Errorf("tai registry not initialized")
	}

	taiID := c.host // for tunnel:// scheme, host stores the taiID
	c.taiID = taiID
	node, ok := reg.Get(taiID)
	if !ok || node.Status != "online" {
		return nil, fmt.Errorf("tai node %s not online", taiID)
	}

	c.ports = Ports{
		GRPC:   nodePort(node.Ports, "grpc", 19100),
		HTTP:   nodePort(node.Ports, "http", 8099),
		VNC:    nodePort(node.Ports, "vnc", 16080),
		Docker: nodePort(node.Ports, "docker", 12375),
	}

	grpcLn, err := reg.OpenLocalListener(taiID, c.ports.GRPC)
	if err != nil {
		return nil, fmt.Errorf("open grpc tunnel listener: %w", err)
	}
	c.tunnelListeners = append(c.tunnelListeners, grpcLn)

	grpcAddr := grpcLn.Addr().String()
	conn, err := grpc.NewClient("passthrough:///"+grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		grpcLn.Close()
		return nil, fmt.Errorf("grpc dial tunnel %s: %w", grpcAddr, err)
	}
	c.grpcConn = conn
	c.he = hepb.NewHostExecClient(conn)
	c.vol = volume.NewRemote(conn)

	info, err := c.discoverServerInfo(conn, cfg)
	if err != nil {
		info = &discoveredInfo{Capabilities: map[string]bool{"docker": true}}
	}

	hasDocker := info.Capabilities["docker"]
	hasHostExec := info.Capabilities["host_exec"]

	if !hasDocker && !hasHostExec {
		c.closeTunnelListeners()
		conn.Close()
		return nil, fmt.Errorf("tai %s: no capabilities available via tunnel", taiID)
	}

	if hasDocker && c.ports.Docker > 0 {
		dockerLn, err := reg.OpenLocalListener(taiID, c.ports.Docker)
		if err == nil {
			c.tunnelListeners = append(c.tunnelListeners, dockerLn)
			sbAddr := fmt.Sprintf("tcp://%s", dockerLn.Addr().String())
			sb, err := sandbox.NewDocker(sbAddr)
			if err == nil {
				c.sb = sb
				c.img = sandbox.NewDockerImage(sandbox.DockerCli(sb))
			}
		}
	}

	if c.sb != nil {
		c.prx = proxy.NewTunnel(taiID, node.YaoBase)
		c.vc = vnc.NewTunnel(taiID, node.YaoBase)
	}
	reg.SetClient(taiID, c)
	return c, nil
}

func (c *Client) closeTunnelListeners() {
	for _, ln := range c.tunnelListeners {
		ln.Close()
	}
	c.tunnelListeners = nil
}

func nodePort(ports map[string]int, key string, fallback int) int {
	if p, ok := ports[key]; ok && p > 0 {
		return p
	}
	return fallback
}

// Close releases all resources.
func (c *Client) Close() error {
	var errs []error
	if c.sb != nil {
		if err := c.sb.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.vol != nil {
		if err := c.vol.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.grpcConn != nil {
		if err := c.grpcConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	c.closeTunnelListeners()
	if c.taiID != "" {
		if reg := registry.Global(); reg != nil {
			reg.Unregister(c.taiID)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close: %v", errs)
	}
	return nil
}

// Volume returns the Volume IO layer. Never nil.
func (c *Client) Volume() volume.Volume { return c.vol }

// DataDir returns the host-side data directory used by the local volume.
// Empty for remote (Tai gRPC) connections — the Tai server manages paths.
func (c *Client) DataDir() string { return c.dataDir }

// Host returns the raw host parsed from the address (IP or hostname).
func (c *Client) Host() string { return c.host }

// TaiID returns the registry key for this client.
func (c *Client) TaiID() string { return c.taiID }

// Workspace returns an fs.FS-compatible filesystem for the given session.
func (c *Client) Workspace(sessionID string) workspace.FS {
	return workspace.New(c.vol, sessionID)
}

// Sandbox returns the container lifecycle manager.
// Nil when the Tai server has no container runtime (host-exec-only mode).
func (c *Client) Sandbox() sandbox.Sandbox { return c.sb }

// Image returns the container image manager.
// Nil when the Tai server has no container runtime.
func (c *Client) Image() sandbox.Image { return c.img }

// Proxy returns the HTTP reverse proxy helper.
// Nil when the Tai server has no container runtime.
func (c *Client) Proxy() proxy.Proxy { return c.prx }

// VNC returns the VNC WebSocket helper.
// Nil when the Tai server has no container runtime.
func (c *Client) VNC() vnc.VNC { return c.vc }

// HostExec returns the HostExec gRPC client for executing commands on the Tai
// host machine. Returns nil in local mode (no Tai server).
func (c *Client) HostExec() hepb.HostExecClient { return c.he }

// IsLocal returns true if the client connects directly to a Docker daemon.
func (c *Client) IsLocal() bool { return c.scheme == "docker" }

func parseAddr(addr string) (scheme, host, dockerAddr string, grpcPort int, err error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", "", "", 0, fmt.Errorf("empty address: use \"local\" for default Docker daemon")
	}

	if addr == "local" {
		return "docker", "", "", 0, nil
	}

	// Bare IP or host(:port) without scheme → normalise before url.Parse,
	// which misparses bare addresses (treats them as path, not host).
	if !strings.Contains(addr, "://") {
		if isLocalHost(addr) {
			return "docker", "", "", 0, nil
		}
		// host:port — split carefully (IPv6 like [::1]:19100 is already handled above)
		h := addr
		if idx := strings.LastIndex(addr, ":"); idx > 0 {
			h = addr[:idx]
		}
		if isLocalHost(h) {
			return "docker", "", "", 0, nil
		}
		addr = "tai://" + addr
	}

	u, parseErr := url.Parse(addr)
	if parseErr != nil {
		return "", "", "", 0, fmt.Errorf("parse addr %q: %w", addr, parseErr)
	}

	switch u.Scheme {
	case "tai":
		hostname := u.Hostname()
		if hostname == "" {
			return "", "", "", 0, fmt.Errorf("tai:// requires a host")
		}
		if portStr := u.Port(); portStr != "" {
			if p, convErr := strconv.Atoi(portStr); convErr == nil && p > 0 {
				grpcPort = p
			}
		}
		return "tai", hostname, "", grpcPort, nil

	case "tunnel":
		taiID := u.Host
		if taiID == "" {
			return "", "", "", 0, fmt.Errorf("tunnel:// requires a tai ID")
		}
		return "tunnel", taiID, "", 0, nil

	case "docker":
		return "docker", "", addr, 0, nil

	case "unix":
		return "docker", "", addr, 0, nil

	case "tcp":
		return "docker", "", addr, 0, nil

	case "npipe":
		return "docker", "", addr, 0, nil

	default:
		return "", "", "", 0, fmt.Errorf("unsupported scheme %q in addr %q", u.Scheme, addr)
	}
}

func isLocalHost(h string) bool {
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}

type discoveredInfo struct {
	Capabilities map[string]bool
	System       registry.SystemInfo
	Version      string
}

// discoverServerInfo calls ServerInfo.GetInfo on the remote Tai server, merges
// discovered ports into c.ports, and returns capabilities + system info.
// Ports explicitly set via WithPorts take precedence over server-reported values.
func (c *Client) discoverServerInfo(conn *grpc.ClientConn, cfg *config) (*discoveredInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := sipb.NewServerInfoClient(conn)
	resp, err := client.GetInfo(ctx, &sipb.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	up := cfg.userPorts

	if p := int(resp.Ports["http"]); p > 0 && up.HTTP == 0 {
		c.ports.HTTP = p
	}
	if p := int(resp.Ports["docker"]); p > 0 && up.Docker == 0 {
		c.ports.Docker = p
	}
	if p := int(resp.Ports["vnc"]); p > 0 && up.VNC == 0 {
		c.ports.VNC = p
	}
	if p := int(resp.Ports["k8s"]); p > 0 && up.K8s == 0 {
		c.ports.K8s = p
	}

	caps := resp.Capabilities
	if caps == nil {
		caps = make(map[string]bool)
	}

	var sys registry.SystemInfo
	if s := resp.System; s != nil {
		sys = registry.SystemInfo{
			OS:       s.Os,
			Arch:     s.Arch,
			Hostname: s.Hostname,
			NumCPU:   int(s.NumCpu),
			TotalMem: s.TotalMem,
			Shell:    s.Shell,
			TempDir:  s.TempDir,
		}
	}

	return &discoveredInfo{
		Capabilities: caps,
		System:       sys,
		Version:      resp.Version,
	}, nil
}

// RegisterLocal probes the local Docker environment and, if reachable,
// creates a Client and registers it as the "local" node in the registry.
// Returns true if a local node was successfully registered.
// Silently returns false if Docker is not available — this is not an error.
func RegisterLocal(opts ...Option) bool {
	reg := registry.Global()
	if reg == nil {
		return false
	}
	if _, ok := reg.Get("local"); ok {
		return true
	}

	c, err := New("local", opts...)
	if err != nil {
		return false
	}
	_ = c // registered by initLocal → reg.Register + reg.SetClient
	return true
}

// GetClient returns a registered *Client by taiID from the global registry.
func GetClient(taiID string) (*Client, bool) {
	reg := registry.Global()
	if reg == nil {
		return nil, false
	}
	snap, ok := reg.Get(taiID)
	if !ok {
		return nil, false
	}
	c, ok := snap.Client().(*Client)
	if !ok || c == nil {
		return nil, false
	}
	return c, true
}

// GetNodeSnapshot returns the registry snapshot for a Tai node by ID.
// Callers can inspect System, Capabilities, Mode and other registry-level fields.
func GetNodeSnapshot(taiID string) (*registry.NodeSnapshot, bool) {
	reg := registry.Global()
	if reg == nil {
		return nil, false
	}
	return reg.Get(taiID)
}
