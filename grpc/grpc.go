package grpc

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		grpc.ChainUnaryInterceptor(auth.UnaryInterceptor),
		grpc.ChainStreamInterceptor(auth.StreamInterceptor),
	)
	if sandboxH == nil {
		sandboxH = sandboxhandler.NewHandler(nil)
	}
	pb.RegisterYaoServer(server, &yaoServer{sandbox: sandboxH})

	hosts := strings.Split(cfg.GRPC.Host, ",")
	port := strconv.Itoa(cfg.GRPC.Port)

	for _, h := range hosts {
		addr := net.JoinHostPort(strings.TrimSpace(h), port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			Stop()
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

// Stop gracefully stops the gRPC server. Safe to call if server was never started.
func Stop() {
	mu.Lock()
	defer mu.Unlock()

	if server != nil {
		server.GracefulStop()
		server = nil
	}
	listeners = nil
	addrs = nil
}

// GRPCServer returns the active gRPC server instance.
// Used by the Tai tunnel server to serve data channel connections
// on the existing gRPC server.
func GRPCServer() *grpc.Server {
	mu.Lock()
	defer mu.Unlock()
	return server
}

// Addr returns all addresses the gRPC server is listening on.
func Addr() []string {
	mu.Lock()
	defer mu.Unlock()
	result := make([]string, len(addrs))
	copy(result, addrs)
	return result
}
