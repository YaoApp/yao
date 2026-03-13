package grpc

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	agenthandler "github.com/yaoapp/yao/grpc/agent"
	apihandler "github.com/yaoapp/yao/grpc/api"
	"github.com/yaoapp/yao/grpc/auth"
	"github.com/yaoapp/yao/grpc/health"
	llmhandler "github.com/yaoapp/yao/grpc/llm"
	mcphandler "github.com/yaoapp/yao/grpc/mcp"
	"github.com/yaoapp/yao/grpc/pb"
	runhandler "github.com/yaoapp/yao/grpc/run"
	sandboxhandler "github.com/yaoapp/yao/grpc/sandbox"
	shellhandler "github.com/yaoapp/yao/grpc/shell"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/tunnel"
	"github.com/yaoapp/yao/tai/tunnel/taipb"
)

var (
	mu        sync.Mutex
	server    *grpc.Server
	listeners []net.Listener
	addrs     []string
)

type yaoServer struct {
	pb.UnimplementedYaoServer
	health  health.Handler
	run     runhandler.Handler
	shell   shellhandler.Handler
	api     apihandler.Handler
	mcp     mcphandler.Handler
	llm     llmhandler.Handler
	agent   agenthandler.Handler
	sandbox *sandboxhandler.Handler
}

// ── Health ───────────────────────────────────────────────────────────────────

func (s *yaoServer) Healthz(ctx context.Context, req *pb.Empty) (*pb.HealthzResponse, error) {
	return s.health.Healthz(ctx, req)
}

// ── Base ─────────────────────────────────────────────────────────────────────

func (s *yaoServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
	return s.run.Run(ctx, req)
}

func (s *yaoServer) Shell(ctx context.Context, req *pb.ShellRequest) (*pb.ShellResponse, error) {
	return s.shell.Shell(ctx, req)
}

// V2 stubs — Stream and ShellStream depend on gou/stream package.
func (s *yaoServer) Stream(req *pb.RunRequest, stream grpc.ServerStreamingServer[pb.Chunk]) error {
	return status.Error(codes.Unimplemented, "Stream not implemented (V2)")
}

func (s *yaoServer) ShellStream(req *pb.ShellRequest, stream grpc.ServerStreamingServer[pb.Chunk]) error {
	return status.Error(codes.Unimplemented, "ShellStream not implemented (V2)")
}

// ── API ──────────────────────────────────────────────────────────────────────

func (s *yaoServer) API(ctx context.Context, req *pb.APIRequest) (*pb.APIResponse, error) {
	return s.api.API(ctx, req)
}

// ── MCP ──────────────────────────────────────────────────────────────────────

func (s *yaoServer) MCPListTools(ctx context.Context, req *pb.MCPListRequest) (*pb.MCPListResponse, error) {
	return s.mcp.MCPListTools(ctx, req)
}

func (s *yaoServer) MCPCallTool(ctx context.Context, req *pb.MCPCallRequest) (*pb.MCPCallResponse, error) {
	return s.mcp.MCPCallTool(ctx, req)
}

func (s *yaoServer) MCPListResources(ctx context.Context, req *pb.MCPListRequest) (*pb.MCPResourcesResponse, error) {
	return s.mcp.MCPListResources(ctx, req)
}

func (s *yaoServer) MCPReadResource(ctx context.Context, req *pb.MCPResourceRequest) (*pb.MCPResourceResponse, error) {
	return s.mcp.MCPReadResource(ctx, req)
}

// ── LLM ──────────────────────────────────────────────────────────────────────

func (s *yaoServer) ChatCompletions(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	return s.llm.ChatCompletions(ctx, req)
}

func (s *yaoServer) ChatCompletionsStream(req *pb.ChatRequest, stream grpc.ServerStreamingServer[pb.ChatChunk]) error {
	return s.llm.ChatCompletionsStream(req, stream)
}

// ── Agent ────────────────────────────────────────────────────────────────────

func (s *yaoServer) AgentStream(req *pb.AgentRequest, stream grpc.ServerStreamingServer[pb.AgentChunk]) error {
	return s.agent.AgentStream(req, stream)
}

// ── Sandbox ──────────────────────────────────────────────────────────────────

func (s *yaoServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if s.sandbox == nil {
		return &pb.HeartbeatResponse{Action: "ok"}, nil
	}
	return s.sandbox.Heartbeat(ctx, req)
}

// SandboxHandler returns the sandbox handler for external access (e.g., Manager integration).
func SandboxHandler() *sandboxhandler.Handler {
	mu.Lock()
	defer mu.Unlock()
	return sandboxH
}

var sandboxH *sandboxhandler.Handler
var tunnelH *tunnel.TunnelHandler

// SetSandboxOnBeat sets the heartbeat callback for the sandbox handler.
// Must be called before StartServer.
func SetSandboxOnBeat(fn func(data *sandboxhandler.HeartbeatData) string) {
	sandboxH = sandboxhandler.NewHandler(fn)
}

// ── Server lifecycle ─────────────────────────────────────────────────────────

// StartServer initializes and starts the gRPC server based on config.
// It supports multiple bind addresses and returns immediately (listeners run in goroutines).
func StartServer(cfg config.Config) error {
	if strings.ToLower(cfg.GRPC.Enabled) == "off" {
		log.Info("gRPC server disabled (YAO_GRPC=off)")
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	server = grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             15 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.ChainUnaryInterceptor(auth.UnaryInterceptor),
		grpc.ChainStreamInterceptor(auth.StreamInterceptor),
	)
	if sandboxH == nil {
		sandboxH = sandboxhandler.NewHandler(nil)
	}
	pb.RegisterYaoServer(server, &yaoServer{sandbox: sandboxH})

	if reg := registry.Global(); reg != nil {
		tunnelH = tunnel.NewTunnelHandler(reg)
		taipb.RegisterTaiTunnelServer(server, tunnelH)
	}

	hosts := ExpandHosts(cfg.GRPC.Host)
	port := strconv.Itoa(cfg.GRPC.Port)

	for _, h := range hosts {
		addr := net.JoinHostPort(h, port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			stopLocked()
			return err
		}
		listeners = append(listeners, lis)
		addrs = append(addrs, lis.Addr().String())
		log.Info("gRPC server listening on %s", lis.Addr().String())

		go func(l net.Listener) {
			if err := server.Serve(l); err != nil {
				log.Error("gRPC server error on %s: %s", l.Addr().String(), err.Error())
			}
		}(lis)
	}

	return nil
}

// stopLocked performs cleanup while the caller already holds mu.
func stopLocked() {
	s := server
	server = nil
	listeners = nil
	addrs = nil

	if s == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Info("gRPC server stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Warn("gRPC server graceful stop timed out, forcing stop")
		s.Stop()
	}
}

// Stop gracefully stops the gRPC server with a 5-second timeout.
// If GracefulStop doesn't complete in time (e.g. active streams), it forces Stop.
// Safe to call if server was never started.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	stopLocked()
}

// GRPCServer returns the active gRPC server instance.
// Used by the Tai tunnel server to serve data channel connections
// on the existing gRPC server.
func GRPCServer() *grpc.Server {
	mu.Lock()
	defer mu.Unlock()
	return server
}

// TunnelHandler returns the gRPC tunnel handler for forward requests.
func TunnelHandler() *tunnel.TunnelHandler {
	mu.Lock()
	defer mu.Unlock()
	return tunnelH
}

// Addr returns all addresses the gRPC server is listening on.
func Addr() []string {
	mu.Lock()
	defer mu.Unlock()
	result := make([]string, len(addrs))
	copy(result, addrs)
	return result
}

// expandHosts parses comma-separated host entries, expanding special values:
//   - "internal" → 127.0.0.1 + all private-network IPv4 addresses (10.x, 172.16-31.x, 192.168.x)
//   - "localhost" → 127.0.0.1
//
// Duplicates are removed.
func ExpandHosts(raw string) []string {
	seen := map[string]bool{}
	var result []string
	for _, h := range strings.Split(raw, ",") {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}

		switch strings.ToLower(h) {
		case "localhost":
			h = "127.0.0.1"
			if !seen[h] {
				seen[h] = true
				result = append(result, h)
			}
		case "internal":
			if !seen["127.0.0.1"] {
				seen["127.0.0.1"] = true
				result = append(result, "127.0.0.1")
			}
			for _, ip := range InternalIPs() {
				if !seen[ip] {
					seen[ip] = true
					result = append(result, ip)
				}
			}
		default:
			if !seen[h] {
				seen[h] = true
				result = append(result, h)
			}
		}
	}
	return result
}

// InternalIPs returns all IPv4 addresses on private-network interfaces
// (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16).
func InternalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			if isPrivateIP(ip) {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}

func isPrivateIP(ip net.IP) bool {
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}
