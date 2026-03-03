package tai

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/yaoapp/yao/tai/proxy"
	"github.com/yaoapp/yao/tai/sandbox"
	"github.com/yaoapp/yao/tai/vnc"
	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/tai/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Runtime selects which container runtime to use via Tai.
type Runtime int

const (
	Docker     Runtime = iota // default
	Containerd                // Phase 2
	K8s                       // Phase 2
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
	GRPC       int // default 9100
	HTTP       int // default 8080
	VNC        int // default 6080
	Docker     int // default 2375
	Containerd int // default 2376
	K8s        int // default 6443
}

// WithPorts overrides default Tai service ports.
func WithPorts(p Ports) Option {
	return optionFunc(func(c *config) { c.ports = p })
}

// WithHTTPClient sets a custom HTTP client for proxy and VNC health checks.
func WithHTTPClient(hc *http.Client) Option {
	return optionFunc(func(c *config) { c.httpClient = hc })
}

// WithDataDir sets the workspace root directory for Local mode.
func WithDataDir(dir string) Option {
	return optionFunc(func(c *config) { c.dataDir = dir })
}

type config struct {
	runtime    Runtime
	ports      Ports
	httpClient *http.Client
	dataDir    string
}

func defaultPorts() Ports {
	return Ports{
		GRPC:       9100,
		HTTP:       8080,
		VNC:        6080,
		Docker:     2375,
		Containerd: 2376,
		K8s:        6443,
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
	if p.Containerd != 0 {
		d.Containerd = p.Containerd
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
	vol      volume.Volume
	sb       sandbox.Sandbox
	prx      proxy.Proxy
	vc       vnc.VNC
	grpcConn *grpc.ClientConn
}

// New creates a Client based on the address protocol:
//
//	""               → Local mode, platform default Docker socket
//	"docker://addr"  → Local mode, specified Docker daemon
//	"tai://host"     → Remote mode via Tai Server
func New(addr string, opts ...Option) (*Client, error) {
	cfg := &config{ports: defaultPorts()}
	for _, o := range opts {
		o.apply(cfg)
	}
	cfg.ports = mergedPorts(cfg.ports)

	scheme, host, dockerAddr, err := parseAddr(addr)
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}
	c.sb = sb
	c.prx = proxy.NewLocal(sb)
	c.vc = vnc.NewLocal(sb)

	dataDir := cfg.dataDir
	if dataDir == "" {
		dataDir = "/tmp/tai-volumes"
	}
	c.vol = volume.NewLocal(dataDir)
	return c, nil
}

func (c *Client) initRemote(cfg *config) (*Client, error) {
	// gRPC connection
	grpcAddr := fmt.Sprintf("%s:%d", c.host, c.ports.GRPC)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", grpcAddr, err)
	}
	c.grpcConn = conn
	c.vol = volume.NewRemote(conn)

	// Sandbox (default Docker, Phase 2: containerd/k8s)
	var sbAddr string
	switch cfg.runtime {
	case Containerd:
		sbAddr = fmt.Sprintf("tcp://%s:%d", c.host, c.ports.Containerd)
		// Phase 2: return sandbox.NewContainerd(sbAddr)
		return nil, fmt.Errorf("containerd runtime not yet implemented")
	case K8s:
		sbAddr = fmt.Sprintf("tcp://%s:%d", c.host, c.ports.K8s)
		// Phase 2: return sandbox.NewK8s(sbAddr)
		return nil, fmt.Errorf("k8s runtime not yet implemented")
	default:
		sbAddr = fmt.Sprintf("tcp://%s:%d", c.host, c.ports.Docker)
		sb, err := sandbox.NewDocker(sbAddr)
		if err != nil {
			conn.Close()
			return nil, err
		}
		c.sb = sb
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

// Workspace returns an fs.FS-compatible filesystem for the given session.
func (c *Client) Workspace(sessionID string) workspace.FS {
	return workspace.New(c.vol, sessionID)
}

// Sandbox returns the container lifecycle manager. Never nil.
func (c *Client) Sandbox() sandbox.Sandbox { return c.sb }

// Proxy returns the HTTP reverse proxy helper. Never nil.
func (c *Client) Proxy() proxy.Proxy { return c.prx }

// VNC returns the VNC WebSocket helper. Never nil.
func (c *Client) VNC() vnc.VNC { return c.vc }

// IsLocal returns true if the client connects directly to a Docker daemon.
func (c *Client) IsLocal() bool { return c.scheme == "docker" }

func parseAddr(addr string) (scheme, host, dockerAddr string, err error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "docker", "", "", nil
	}

	u, parseErr := url.Parse(addr)
	if parseErr != nil {
		return "", "", "", fmt.Errorf("parse addr %q: %w", addr, parseErr)
	}

	switch u.Scheme {
	case "tai":
		host = u.Host
		if host == "" {
			return "", "", "", fmt.Errorf("tai:// requires a host")
		}
		if idx := strings.Index(host, ":"); idx >= 0 {
			host = host[:idx]
		}
		return "tai", host, "", nil

	case "docker":
		return "docker", "", addr, nil

	case "unix":
		return "docker", "", addr, nil

	case "tcp":
		return "docker", "", addr, nil

	case "npipe":
		return "docker", "", addr, nil

	default:
		return "", "", "", fmt.Errorf("unsupported scheme %q in addr %q", u.Scheme, addr)
	}
}
