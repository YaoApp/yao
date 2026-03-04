package llm

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/gou/connector"
	agentContext "github.com/yaoapp/yao/agent/context"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/grpc/auth"
	"github.com/yaoapp/yao/grpc/pb"
)

// Handler implements the LLM gRPC methods.
type Handler struct{}

// ChatCompletions sends messages to an LLM connector and returns the full response (unary).
func (h *Handler) ChatCompletions(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	if req.Connector == "" {
		return nil, status.Error(codes.InvalidArgument, "connector is required")
	}

	llmInstance, completionOpts, ctxMessages, agentCtx, err := prepareLLMCall(ctx, req)
	if err != nil {
		return nil, err
	}
	defer agentCtx.Release()

	noopHandler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }
	response, err := llmInstance.Stream(agentCtx, ctxMessages, completionOpts, noopHandler)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "LLM call failed: %v", err)
	}

	data, err := json.Marshal(toOpenAIFormat(response))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal LLM response: %v", err)
	}

	return &pb.ChatResponse{Data: data}, nil
}

// ChatCompletionsStream sends messages to an LLM connector and streams response chunks.
func (h *Handler) ChatCompletionsStream(req *pb.ChatRequest, stream grpc.ServerStreamingServer[pb.ChatChunk]) error {
	ctx := stream.Context()

	if req.Connector == "" {
		return status.Error(codes.InvalidArgument, "connector is required")
	}

	llmInstance, completionOpts, ctxMessages, agentCtx, err := prepareLLMCall(ctx, req)
	if err != nil {
		return err
	}
	defer agentCtx.Release()

	streamHandler := func(chunkType message.StreamChunkType, data []byte) int {
		if ctx.Err() != nil {
			return 1
		}
		if chunkType == message.ChunkText || chunkType == message.ChunkThinking {
			if sendErr := stream.Send(&pb.ChatChunk{Data: data}); sendErr != nil {
				return 1
			}
		}
		return 0
	}

	_, err = llmInstance.Stream(agentCtx, ctxMessages, completionOpts, streamHandler)
	if err != nil {
		return status.Errorf(codes.Internal, "LLM stream failed: %v", err)
	}

	return stream.Send(&pb.ChatChunk{Done: true})
}

// prepareLLMCall builds the LLM instance, messages, and agent context from the gRPC request.
// Mirrors agent/llm/process.go ProcessChatCompletions logic without the process wrapper.
func prepareLLMCall(ctx context.Context, req *pb.ChatRequest) (agentLLM.LLM, *agentContext.CompletionOptions, []agentContext.Message, *agentContext.Context, error) {
	ctxMessages, err := parseMessagesToContext(req.Messages)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var opts map[string]interface{}
	if len(req.Options) > 0 {
		if err := json.Unmarshal(req.Options, &opts); err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.InvalidArgument, "invalid options JSON: %v", err)
		}
	}

	conn, err := connector.Select(req.Connector)
	if err != nil {
		return nil, nil, nil, nil, status.Errorf(codes.NotFound, "connector %s not found: %v", req.Connector, err)
	}

	completionOpts := agentLLM.BuildCompletionOptions(conn, opts)

	llmInstance, err := agentLLM.New(conn, completionOpts)
	if err != nil {
		return nil, nil, nil, nil, status.Errorf(codes.Internal, "failed to create LLM: %v", err)
	}

	authInfo := auth.GetAuthorizedInfo(ctx)
	chatID := agentContext.GenChatID()
	agentCtx := agentContext.New(ctx, authInfo, chatID)

	return llmInstance, completionOpts, ctxMessages, agentCtx, nil
}

// parseMessagesToContext converts raw JSON message bytes to []agentContext.Message via JSON round-trip.
func parseMessagesToContext(raw []byte) ([]agentContext.Message, error) {
	if len(raw) == 0 {
		return nil, status.Error(codes.InvalidArgument, "messages are required")
	}

	var messages []agentContext.Message
	if err := json.Unmarshal(raw, &messages); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid messages JSON: %v", err)
	}
	if len(messages) == 0 {
		return nil, status.Error(codes.InvalidArgument, "messages must not be empty")
	}

	return messages, nil
}

// toOpenAIFormat converts CompletionResponse to OpenAI chat.completions format.
func toOpenAIFormat(resp *agentContext.CompletionResponse) map[string]interface{} {
	if resp == nil {
		return map[string]interface{}{"choices": []interface{}{}}
	}

	msgMap := map[string]interface{}{
		"role":    resp.Role,
		"content": resp.Content,
	}
	if len(resp.ToolCalls) > 0 {
		msgMap["tool_calls"] = resp.ToolCalls
	}

	choice := map[string]interface{}{
		"index":         0,
		"message":       msgMap,
		"finish_reason": "stop",
	}

	result := map[string]interface{}{
		"id":      resp.ID,
		"object":  "chat.completion",
		"created": resp.Created,
		"model":   resp.Model,
		"choices": []interface{}{choice},
	}
	if resp.Usage != nil {
		result["usage"] = resp.Usage
	}

	return result
}
