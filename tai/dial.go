package tai

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	yaoconfig "github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/tai/hostexec"
	hepb "github.com/yaoapp/yao/tai/hostexec/pb"
	"github.com/yaoapp/yao/tai/proxy"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/runtime"
	sipb "github.com/yaoapp/yao/tai/serverinfo/pb"
	"github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/tai/vnc"
	"github.com/yaoapp/yao/tai/volume"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// DialRemote establishes connections to a remote Tai node via gRPC (direct mode).
// Does NOT interact with the registry. Caller must call ConnResources.Close().
func DialRemote(host string, ports types.Ports, opts ...DialOption) (*ConnResources, error) {
	cfg := &dialConfig{ports: mergedPorts(ports)}
	for _, o := range opts {
		o.applyDial(cfg)
	}

	grpcAddr := fmt.Sprintf("%s:%d", host, cfg.ports.GRPC)
	conn, err := dialGRPC(grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", grpcAddr, err)
	}

	return buildResources(conn, cfg, &remoteEnv{host: host, httpClient: cfg.httpClient})
}

// DialTunnel establishes connections to a Tai node through the WebSocket tunnel.
// Requires the node to already be registered in the registry (online).
// Does NOT call registry.SetResources. Caller must call ConnResources.Close().
func DialTunnel(taiID string, reg *registry.Registry, opts ...DialOption) (*ConnResources, error) {
	node, ok := reg.Get(taiID)
	if !ok || node.Status != "online" {
		return nil, fmt.Errorf("tai node %s not online", taiID)
	}

	cfg := &dialConfig{
		ports: types.Ports{
			GRPC:   intOr(node.Ports.GRPC, 19100),
			HTTP:   intOr(node.Ports.HTTP, 8099),
			VNC:    intOr(node.Ports.VNC, 16080),
			Docker: intOr(node.Ports.Docker, 12375),
			K8s:    intOr(node.Ports.K8s, 16443),
		},
	}
	for _, o := range opts {
		o.applyDial(cfg)
	}

	grpcLn, err := reg.OpenLocalListener(taiID, cfg.ports.GRPC)
	if err != nil {
		return nil, fmt.Errorf("open grpc tunnel listener: %w", err)
	}

	conn, err := dialGRPC("passthrough:///" + grpcLn.Addr().String())
	if err != nil {
		grpcLn.Close()
		return nil, fmt.Errorf("grpc dial tunnel %s: %w", grpcLn.Addr(), err)
	}

	env := &tunnelEnv{
		taiID:     taiID,
		yaoBase:   node.YaoBase,
		reg:       reg,
		regCaps:   node.Capabilities,
		listeners: []net.Listener{grpcLn},
	}

	res, err := buildResources(conn, cfg, env)
	if err != nil {
		grpcLn.Close()
		conn.Close()
		return nil, err
	}
	res.Listeners = env.listeners
	return res, nil
}

// DialLocal establishes connections to the local host as a Tai node.
// Docker is probed but not required — when unavailable the node still
// provides Volume (and optionally HostExec) capabilities.
// Does NOT interact with the registry. Caller must call ConnResources.Close().
func DialLocal(addr string, dataDir string, vol volume.Volume) (*ConnResources, error) {
	sb, _ := runtime.NewLocal(addr) // Docker failure is non-fatal

	res := &ConnResources{
		DataDir: dataDir,
		System:  CollectSystemInfo(),
	}

	if sb != nil {
		res.Runtime = sb
		res.Image = runtime.NewDockerImage(runtime.DockerCli(sb))
		res.Proxy = proxy.NewLocal(sb)
		res.VNC = vnc.NewLocal(sb)
	}

	if yaoconfig.Conf.HostExec.Enabled {
		res.HostExec = hostexec.NewLocalClient(dataDir, hostexec.Policy{
			FullAccess:      yaoconfig.Conf.HostExec.FullAccess,
			AllowedCommands: yaoconfig.Conf.HostExec.AllowedCommands,
			AllowedDirs:     yaoconfig.Conf.HostExec.AllowedDirs,
			DeniedDirs:      yaoconfig.Conf.HostExec.DeniedDirs,
		})
	}

	if vol != nil {
		res.Volume = vol
	} else {
		if dataDir == "" {
			dataDir = "/tmp/tai-volumes"
		}
		res.DataDir = dataDir
		res.Volume = volume.NewLocal(dataDir)
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// Shared build logic
// ---------------------------------------------------------------------------

// dialEnv abstracts the mode-specific differences (remote vs tunnel) that
// buildResources needs.
type dialEnv interface {
	fallbackCaps() map[string]bool
	mergeCaps(discovered map[string]bool) types.Capabilities
	// listenAddr opens or formats a host:port address for the given port.
	// Tunnel mode opens a local listener; remote mode formats host:port.
	listenAddr(port int) (string, error)
	newProxy(ports types.Ports) proxy.Proxy
	newVNC(ports types.Ports) vnc.VNC
}

// buildResources constructs a ConnResources from an established gRPC
// connection. Shared by DialRemote and DialTunnel.
func buildResources(conn *grpc.ClientConn, cfg *dialConfig, env dialEnv) (*ConnResources, error) {
	info, err := discoverInfo(conn, cfg)
	if err != nil {
		info = &discoveredInfo{Capabilities: env.fallbackCaps()}
	}

	caps := env.mergeCaps(info.Capabilities)

	res := &ConnResources{
		GRPCConn: conn,
		HostExec: hepb.NewHostExecClient(conn),
		Volume:   volume.NewRemote(conn),
		Caps:     caps,
		System:   info.System,
		Ports:    cfg.ports,
		Version:  info.Version,
	}

	if cfg.runtime == types.K8s || (!caps.Docker && caps.K8s) {
		if cfg.kubeConfig != "" {
			k8sPort := cfg.ports.K8s
			if k8sPort == 0 {
				k8sPort = 16443
			}
			addr, err := env.listenAddr(k8sPort)
			if err == nil {
				sb, err := runtime.NewK8s(addr, runtime.K8sOption{
					Namespace:  cfg.namespace,
					KubeConfig: cfg.kubeConfig,
				})
				if err == nil {
					res.Runtime = sb
					res.Image = runtime.NewK8sImage()
				}
			}
		}
	} else if caps.Docker {
		dockerPort := cfg.ports.Docker
		if dockerPort == 0 {
			dockerPort = 12375
		}
		addr, err := env.listenAddr(dockerPort)
		if err == nil {
			sb, err := runtime.NewDocker("tcp://" + addr)
			if err == nil {
				res.Runtime = sb
				res.Image = runtime.NewDockerImage(runtime.DockerCli(sb))
			}
		}
	}

	if res.Runtime != nil {
		res.Proxy = env.newProxy(cfg.ports)
	}
	res.VNC = env.newVNC(cfg.ports)

	return res, nil
}

// ---------------------------------------------------------------------------
// remoteEnv — direct TCP connections
// ---------------------------------------------------------------------------

type remoteEnv struct {
	host       string
	httpClient *http.Client
}

func (e *remoteEnv) fallbackCaps() map[string]bool {
	return map[string]bool{"docker": true}
}

func (e *remoteEnv) mergeCaps(discovered map[string]bool) types.Capabilities {
	return types.Capabilities{
		Docker:   discovered["docker"],
		K8s:      discovered["k8s"],
		HostExec: discovered["host_exec"],
	}
}

func (e *remoteEnv) listenAddr(port int) (string, error) {
	return fmt.Sprintf("%s:%d", e.host, port), nil
}

func (e *remoteEnv) newProxy(ports types.Ports) proxy.Proxy {
	return proxy.NewRemote(e.host, ports.HTTP, e.httpClient)
}

func (e *remoteEnv) newVNC(ports types.Ports) vnc.VNC {
	return vnc.NewRemote(e.host, ports.VNC, e.httpClient)
}

// ---------------------------------------------------------------------------
// tunnelEnv — connections via WebSocket tunnel
// ---------------------------------------------------------------------------

type tunnelEnv struct {
	taiID     string
	yaoBase   string
	reg       *registry.Registry
	regCaps   types.Capabilities
	listeners []net.Listener
}

func (e *tunnelEnv) fallbackCaps() map[string]bool {
	return make(map[string]bool)
}

func (e *tunnelEnv) mergeCaps(discovered map[string]bool) types.Capabilities {
	return types.Capabilities{
		Docker:   discovered["docker"] || e.regCaps.Docker,
		K8s:      discovered["k8s"] || e.regCaps.K8s,
		HostExec: discovered["host_exec"] || e.regCaps.HostExec,
	}
}

func (e *tunnelEnv) listenAddr(port int) (string, error) {
	ln, err := e.reg.OpenLocalListener(e.taiID, port)
	if err != nil {
		return "", err
	}
	e.listeners = append(e.listeners, ln)
	return ln.Addr().String(), nil
}

func (e *tunnelEnv) newProxy(_ types.Ports) proxy.Proxy {
	return proxy.NewTunnel(e.taiID, e.yaoBase)
}

func (e *tunnelEnv) newVNC(_ types.Ports) vnc.VNC {
	return vnc.NewTunnel(e.taiID, e.yaoBase)
}

// ---------------------------------------------------------------------------
// Dial options
// ---------------------------------------------------------------------------

// DialOption configures a Dial* call.
type DialOption interface {
	applyDial(*dialConfig)
}

type dialOptionFunc func(*dialConfig)

func (f dialOptionFunc) applyDial(c *dialConfig) { f(c) }

// WithDialRuntime selects the container runtime for the dial call.
func WithDialRuntime(rt types.Runtime) DialOption {
	return dialOptionFunc(func(c *dialConfig) { c.runtime = rt })
}

// WithDialKubeConfig sets the kubeconfig for K8s runtime.
func WithDialKubeConfig(path string) DialOption {
	return dialOptionFunc(func(c *dialConfig) { c.kubeConfig = path })
}

// WithDialNamespace sets the K8s namespace.
func WithDialNamespace(ns string) DialOption {
	return dialOptionFunc(func(c *dialConfig) { c.namespace = ns })
}

// WithDialHTTPClient sets a custom HTTP client for proxy/VNC.
func WithDialHTTPClient(hc *http.Client) DialOption {
	return dialOptionFunc(func(c *dialConfig) { c.httpClient = hc })
}

type dialConfig struct {
	runtime    types.Runtime
	ports      types.Ports
	kubeConfig string
	namespace  string
	httpClient *http.Client
	userPorts  types.Ports
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func dialGRPC(target string) (*grpc.ClientConn, error) {
	return grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
}

// ---------------------------------------------------------------------------
// ServerInfo discovery (shared by DialRemote / DialTunnel)
// ---------------------------------------------------------------------------

type discoveredInfo struct {
	Capabilities map[string]bool
	System       types.SystemInfo
	Version      string
}

func discoverInfo(conn *grpc.ClientConn, cfg *dialConfig) (*discoveredInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := sipb.NewServerInfoClient(conn)
	resp, err := client.GetInfo(ctx, &sipb.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	up := cfg.userPorts

	if p := int(resp.Ports["http"]); p > 0 && up.HTTP == 0 {
		cfg.ports.HTTP = p
	}
	if p := int(resp.Ports["docker"]); p > 0 && up.Docker == 0 {
		cfg.ports.Docker = p
	}
	if p := int(resp.Ports["vnc"]); p > 0 && up.VNC == 0 {
		cfg.ports.VNC = p
	}
	if p := int(resp.Ports["k8s"]); p > 0 && up.K8s == 0 {
		cfg.ports.K8s = p
	}

	caps := resp.Capabilities
	if caps == nil {
		caps = make(map[string]bool)
	}

	var sys types.SystemInfo
	if s := resp.System; s != nil {
		sys = types.SystemInfo{
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
