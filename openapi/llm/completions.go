package llm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	agentContext "github.com/yaoapp/yao/agent/context"
	agentllm "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// chatCompletionRequest mirrors the OpenAI ChatCompletion request body.
type chatCompletionRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	Stream      bool                     `json:"stream"`
	Temperature *float64                 `json:"temperature,omitempty"`
	MaxTokens   *int                     `json:"max_tokens,omitempty"`
	TopP        *float64                 `json:"top_p,omitempty"`
	Stop        interface{}              `json:"stop,omitempty"`
	Tools       interface{}              `json:"tools,omitempty"`
}

// openaiError is the standard OpenAI error envelope.
type openaiError struct {
	Error openaiErrorBody `json:"error"`
}

type openaiErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

func respondError(c *gin.Context, status int, errType, msg string) {
	c.JSON(status, openaiError{
		Error: openaiErrorBody{Message: msg, Type: errType},
	})
}

// handleChatCompletions implements POST /llm/chat/completions.
func handleChatCompletions(c *gin.Context) {
	var req chatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request_error", "Invalid request body: "+err.Error())
		return
	}

	if len(req.Messages) == 0 {
		respondError(c, http.StatusBadRequest, "invalid_request_error", "messages is required and must not be empty")
		return
	}
	if req.Model == "" {
		respondError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	info := authorized.GetInfo(c)

	conn, _, err := agentllm.ResolveConnector(req.Model, info)
	if err != nil {
		respondError(c, http.StatusNotFound, "not_found_error", "Connector not found: "+err.Error())
		return
	}

	resolvedModel := conn.ID()

	if err := checkConnectorAccess(resolvedModel, info); err != nil {
		respondError(c, http.StatusForbidden, "permission_error", err.Error())
		return
	}

	opts := buildOptsMap(req)
	completionOptions := agentllm.BuildCompletionOptions(conn, opts)

	llmInstance, err := agentllm.New(conn, completionOptions)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "server_error", "Failed to create LLM instance: "+err.Error())
		return
	}

	interfaceMessages := make([]interface{}, len(req.Messages))
	for i, m := range req.Messages {
		interfaceMessages[i] = m
	}
	ctxMessages, err := agentllm.ParseMessages(interfaceMessages)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request_error", "Invalid messages: "+err.Error())
		return
	}
	for i := range ctxMessages {
		if parts, ok := ctxMessages[i].Content.([]interface{}); ok {
			ctxMessages[i].Content = agentllm.NormalizeContentParts(parts)
		}
	}

	ctx := agentContext.New(c.Request.Context(), info, agentContext.GenChatID())
	defer ctx.Release()

	if req.Stream {
		handleStreamResponse(c, ctx, llmInstance, ctxMessages, completionOptions, resolvedModel)
	} else {
		handleNonStreamResponse(c, ctx, llmInstance, ctxMessages, completionOptions)
	}
}

func handleNonStreamResponse(
	c *gin.Context,
	ctx *agentContext.Context,
	llmInstance agentllm.LLM,
	messages []agentContext.Message,
	options *agentContext.CompletionOptions,
) {
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		return 0
	}

	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		respondError(c, http.StatusBadGateway, "upstream_error", "LLM call failed: "+sanitizeError(err))
		return
	}

	c.JSON(http.StatusOK, agentllm.ToOpenAIFormat(response))
}

func handleStreamResponse(
	c *gin.Context,
	ctx *agentContext.Context,
	llmInstance agentllm.LLM,
	messages []agentContext.Message,
	options *agentContext.CompletionOptions,
	model string,
) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		respondError(c, http.StatusInternalServerError, "server_error", "Streaming not supported")
		return
	}

	w := &openaiSSEWriter{
		w:       c.Writer,
		flusher: flusher,
		id:      "chatcmpl-" + uuid.New().String(),
		model:   model,
		created: time.Now().Unix(),
	}

	roleSent := false
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		select {
		case <-c.Request.Context().Done():
			return -1
		default:
		}

		switch chunkType {
		case message.ChunkMessageStart:
			if !roleSent {
				w.writeChunk(map[string]interface{}{"role": "assistant"}, nil)
				roleSent = true
			}
		case message.ChunkText:
			if !roleSent {
				w.writeChunk(map[string]interface{}{"role": "assistant"}, nil)
				roleSent = true
			}
			w.writeChunk(map[string]interface{}{"content": string(data)}, nil)
		case message.ChunkThinking:
			if !roleSent {
				w.writeChunk(map[string]interface{}{"role": "assistant"}, nil)
				roleSent = true
			}
			w.writeChunk(map[string]interface{}{"reasoning_content": string(data)}, nil)
		case message.ChunkToolCall:
			var tc interface{}
			if json.Unmarshal(data, &tc) == nil {
				w.writeChunk(map[string]interface{}{"tool_calls": tc}, nil)
			}
		case message.ChunkMessageEnd:
			finishReason := "stop"
			w.writeChunk(nil, &finishReason)
		}
		return 0
	}

	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
		errData, _ := json.Marshal(map[string]interface{}{
			"error": map[string]interface{}{
				"message": sanitizeError(err),
				"type":    "upstream_error",
			},
		})
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", errData)
		flusher.Flush()
		return
	}

	if response != nil && response.Usage != nil {
		w.writeChunk(nil, nil)
	}

	w.writeDone()
}

// openaiSSEWriter serializes OpenAI-compatible SSE chunks.
type openaiSSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	id      string
	model   string
	created int64
	mu      sync.Mutex
}

func (s *openaiSSEWriter) writeChunk(delta map[string]interface{}, finishReason *string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	choice := map[string]interface{}{
		"index": 0,
	}
	if delta != nil {
		choice["delta"] = delta
	} else {
		choice["delta"] = map[string]interface{}{}
	}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	} else {
		choice["finish_reason"] = nil
	}

	chunk := map[string]interface{}{
		"id":      s.id,
		"object":  "chat.completion.chunk",
		"created": s.created,
		"model":   s.model,
		"choices": []interface{}{choice},
	}

	data, err := json.Marshal(chunk)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(s.w, "data: %s\n\n", data)
	s.flusher.Flush()
}

func (s *openaiSSEWriter) writeDone() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = fmt.Fprint(s.w, "data: [DONE]\n\n")
	s.flusher.Flush()
}

// checkConnectorAccess verifies the caller has permission to use the resolved connector.
func checkConnectorAccess(connectorID string, identity llmprovider.Identity) error {
	if llmprovider.Global == nil {
		return nil
	}

	baseCID := connectorID
	if idx := strings.Index(baseCID, ":"); idx > 0 {
		baseCID = baseCID[:idx]
	}

	provider, err := llmprovider.Global.GetByConnectorID(baseCID)
	if err != nil {
		return nil
	}

	switch provider.Owner.Type {
	case "user":
		if identity == nil || provider.Owner.UserID != identity.GetUserID() {
			return fmt.Errorf("access denied: connector %q belongs to another user", connectorID)
		}
	case "team":
		if identity == nil || provider.Owner.TeamID != identity.GetTeamID() {
			return fmt.Errorf("access denied: connector %q belongs to another team", connectorID)
		}
	}
	return nil
}

func buildOptsMap(req chatCompletionRequest) map[string]interface{} {
	opts := map[string]interface{}{}
	if req.Temperature != nil {
		opts["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		opts["max_tokens"] = float64(*req.MaxTokens)
	}
	if req.TopP != nil {
		opts["top_p"] = *req.TopP
	}
	if req.Stop != nil {
		opts["stop"] = req.Stop
	}
	if req.Tools != nil {
		opts["tools"] = req.Tools
	}
	return opts
}

func sanitizeError(err error) string {
	msg := err.Error()
	for _, keyword := range []string{"Bearer ", "sk-", "key-"} {
		if idx := strings.Index(msg, keyword); idx >= 0 {
			msg = msg[:idx] + "[REDACTED]"
		}
	}
	return msg
}
