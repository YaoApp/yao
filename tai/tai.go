package tai

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yaoapp/yao/tai/proxy"
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
	GRPC   int // default 9100
	HTTP   int // default 8080
	VNC    int // default 6080
	Docker int // default 2375
	K8s    int // default 6443
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
		GRPC: 9100,
		HTTP: 8080,
		VNC:  6080,
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
	scheme   string // "tai" or "docker"
	host     string
	addr     string
	ports    Ports
	dataDir  string // host-side data directory for local volume
	vol      volume.Volume
	sb       sandbox.Sandbox
	img      sandbox.Image
	prx      proxy.Proxy
	vc       vnc.VNC
	grpcConn *grpc.ClientConn
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
	return c, nil
}

func (c *Client) initRemote(cfg *config) (*Client, error) {
	grpcAddr := fmt.Sprintf("%s:%d", c.host, c.ports.GRPC)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", grpcAddr, err)
	}
	c.grpcConn = conn

	// Auto-discover server ports via ServerInfo RPC.
	// Only overwrite ports that were NOT explicitly set by WithPorts.
	if err := c.discoverPorts(conn, cfg); err != nil {
		// Non-fatal: fall back to defaults / WithPorts values.
		// Old Tai servers without ServerInfo will hit this path.
		_ = err
	}

	c.vol = volume.NewRemote(conn)

	switch cfg.runtime {
	case K8s:
		k8sPort := c.ports.K8s
		if k8sPort == 0 {
			k8sPort = 6443
		}
		sbAddr := fmt.Sprintf("%s:%d", c.host, k8sPort)
		sb, err := sandbox.NewK8s(sbAddr, sandbox.K8sOption{
			Namespace:  cfg.namespace,
			KubeConfig: cfg.kubeConfig,
		})
		if err != nil {
			conn.Close()
			return nil, err
		}
		c.sb = sb
		c.img = sandbox.NewK8sImage()
	default:
		dockerPort := c.ports.Docker
		if dockerPort == 0 {
			dockerPort = 2375
		}
		sbAddr := fmt.Sprintf("tcp://%s:%d", c.host, dockerPort)
		sb, err := sandbox.NewDocker(sbAddr)
		if err != nil {
			conn.Close()
			return nil, err
		}
		c.sb = sb
		c.img = sandbox.NewDockerImage(sandbox.DockerCli(sb))
	}

	hc := cfg.httpClient
	c.prx = proxy.NewRemote(c.host, c.ports.HTTP, hc)
	c.vc = vnc.NewRemote(c.host, c.ports.VNC, hc)
	return c, nil
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

// Workspace returns an fs.FS-compatible filesystem for the given session.
func (c *Client) Workspace(sessionID string) workspace.FS {
	return workspace.New(c.vol, sessionID)
}

// Sandbox returns the container lifecycle manager. Never nil.
func (c *Client) Sandbox() sandbox.Sandbox { return c.sb }

// Image returns the container image manager. Never nil.
func (c *Client) Image() sandbox.Image { return c.img }

// Proxy returns the HTTP reverse proxy helper. Never nil.
func (c *Client) Proxy() proxy.Proxy { return c.prx }

// VNC returns the VNC WebSocket helper. Never nil.
func (c *Client) VNC() vnc.VNC { return c.vc }

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
		// host:port — split carefully (IPv6 like [::1]:9100 is already handled above)
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

// discoverPorts calls ServerInfo.GetInfo on the remote Tai server and merges
// discovered ports into c.ports. Ports explicitly set via WithPorts (non-zero
// in the original config before merging defaults) take precedence.
func (c *Client) discoverPorts(conn *grpc.ClientConn, cfg *config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := sipb.NewServerInfoClient(conn)
	resp, err := client.GetInfo(ctx, &sipb.GetInfoRequest{})
	if err != nil {
		return err
	}

	// cfg.userPorts tracks what the caller explicitly passed to WithPorts.
	// Only overwrite ports that the caller did NOT explicitly set.
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
	return nil
}
