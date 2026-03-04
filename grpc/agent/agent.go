package agent

import (
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/grpc/auth"
	"github.com/yaoapp/yao/grpc/pb"
)

// Handler implements the AgentStream gRPC method.
type Handler struct{}

// AgentStream resolves an assistant by ID and streams agent output as AgentChunk messages.
// Mirrors openapi/chat/completions.go GinCreateCompletions flow via context.GetGRPCAgentRequest.
func (h *Handler) AgentStream(req *pb.AgentRequest, stream grpc.ServerStreamingServer[pb.AgentChunk]) error {
	ctx := stream.Context()

	if req.AssistantId == "" {
		return status.Error(codes.InvalidArgument, "assistant_id is required")
	}

	agentDSL := agent.GetAgent()
	if agentDSL == nil {
		return status.Error(codes.Internal, "agent DSL not initialized")
	}

	cache, err := agentDSL.GetCacheStore()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get cache store: %v", err)
	}

	messages, agentCtx, opts, err := agentContext.GetGRPCAgentRequest(ctx, agentContext.GRPCAgentInput{
		AssistantID: req.AssistantId,
		Messages:    req.Messages,
		Options:     req.Options,
		AuthInfo:    auth.GetAuthorizedInfo(ctx),
		Cache:       cache,
		Writer:      &grpcStreamWriter{stream: stream, header: make(http.Header)},
	})
	if err != nil {
		return toGRPCError(err)
	}
	defer agentCtx.Release()

	ast, err := assistant.Get(agentCtx.AssistantID)
	if err != nil {
		return status.Errorf(codes.NotFound, "assistant not found: %v", err)
	}

	_, err = ast.Stream(agentCtx, messages, opts)
	if err != nil {
		return status.Errorf(codes.Internal, "agent stream failed: %v", err)
	}

	return stream.Send(&pb.AgentChunk{Done: true})
}

func toGRPCError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "is required") ||
		strings.Contains(msg, "must not be empty") ||
		strings.Contains(msg, "invalid") {
		return status.Error(codes.InvalidArgument, msg)
	}
	return status.Error(codes.Internal, msg)
}

// grpcStreamWriter bridges agent/context.Writer (http.ResponseWriter) to gRPC stream.
type grpcStreamWriter struct {
	stream grpc.ServerStreamingServer[pb.AgentChunk]
	header http.Header
	code   int
}

func (w *grpcStreamWriter) Header() http.Header        { return w.header }
func (w *grpcStreamWriter) WriteHeader(statusCode int) { w.code = statusCode }
func (w *grpcStreamWriter) Write(data []byte) (int, error) {
	if err := w.stream.Send(&pb.AgentChunk{Data: data}); err != nil {
		return 0, err
	}
	return len(data), nil
}

// Flush implements http.Flusher for streaming compatibility.
func (w *grpcStreamWriter) Flush() {}
