package context

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// GetCompletionRequest parse completion request and create context from openapi request
// Returns: *CompletionRequest, *Context, *Options, error
func GetCompletionRequest(c *gin.Context, cache store.Store) (*CompletionRequest, *Context, *Options, error) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Parse completion request from payload or query first
	completionReq, err := parseCompletionRequestData(c)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse completion request: %w", err)
	}

	// Extract assistant ID using completionReq (can extract from model field)
	assistantID, err := GetAssistantID(c, completionReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get assistant ID: %w", err)
	}

	// Extract chat ID (may generate from messages if not provided)
	chatID, err := GetChatID(c, cache, completionReq)
	if err != nil {
		// Fallback: Generate a new chat ID if extraction fails
		chatID = GenChatID()
	}

	// Parse client information from User-Agent header
	userAgent := c.GetHeader("User-Agent")
	clientType := getClientType(userAgent)
	clientIP := c.ClientIP()

	// Create context with unique ID using New() to ensure proper initialization
	ctx := New(c.Request.Context(), authInfo, chatID)

	// Set context fields (session-level state)
	ctx.Cache = cache
	ctx.Writer = c.Writer
	ctx.AssistantID = assistantID
	ctx.Locale = GetLocale(c, completionReq)
	ctx.Theme = GetTheme(c, completionReq)
	ctx.Referer = GetReferer(c, completionReq)
	ctx.Accept = GetAccept(c, completionReq)
	ctx.Client = Client{
		Type:      clientType,
		UserAgent: userAgent,
		IP:        clientIP,
	}
	ctx.Route = GetRoute(c, completionReq)
	ctx.Metadata = GetMetadata(c, completionReq)

	// Create Options (call-level parameters)
	opts := &Options{
		Context: c.Request.Context(),
		Skip:    GetSkip(c, completionReq),
		Mode:    GetMode(c, completionReq),
	}

	// Try to extract custom connector from model field
	// If model is a valid connector ID, set it to opts.Connector
	// Otherwise, keep the standard OpenAI-compatible behavior (model as assistant ID)
	if completionReq != nil && completionReq.Model != "" {
		// Check if model is a valid connector (not containing "-yao_" which indicates assistant ID format)
		if !strings.Contains(completionReq.Model, "-yao_") {
			// Try to validate if it's a real connector
			if _, err := connector.Select(completionReq.Model); err == nil {
				// It's a valid connector, use it
				opts.Connector = completionReq.Model
			}
			// If not a valid connector, ignore it (keep opts.Connector empty to use assistant's default)
		}
	}

	// Initialize interrupt controller
	ctx.Interrupt = NewInterruptController()

	// Register context to global registry first (required for interrupt handler callback)
	if err := Register(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to register context: %w", err)
	}

	// Start interrupt listener after registration
	// Only monitors interrupt signals (user stop button for appending messages)
	// HTTP context cancellation is handled by LLM/Agent layers naturally
	ctx.Interrupt.Start(ctx.ID)

	return completionReq, ctx, opts, nil
}

// getClientType parses the client type from User-Agent header
func getClientType(userAgent string) string {
	if userAgent == "" {
		return "web" // Default to web
	}

	ua := strings.ToLower(userAgent)

	// Check for specific client types
	switch {
	case strings.Contains(ua, "yao-agent") || strings.Contains(ua, "agent"):
		return "agent"
	case strings.Contains(ua, "yao-jssdk") || strings.Contains(ua, "jssdk"):
		return "jssdk"
	case strings.Contains(ua, "android"):
		return "android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod"):
		return "ios"
	case strings.Contains(ua, "windows"):
		return "windows"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		return "macos"
	case strings.Contains(ua, "linux"):
		return "linux"
	default:
		return "web"
	}
}

// GetAssistantID extracts assistant ID from request with priority:
// 1. Query parameter "assistant_id"
// 2. Header "X-Yao-Assistant"
// 3. Extract from model field (from CompletionRequest or Query) - splits by "-" takes last field, extracts ID from "yao_xxx" prefix
func GetAssistantID(c *gin.Context, req *CompletionRequest) (string, error) {
	// Priority 1: Query parameter assistant_id
	if assistantID := c.Query("assistant_id"); assistantID != "" {
		return assistantID, nil
	}

	// Priority 2: Header X-Yao-Assistant
	if assistantID := c.GetHeader("X-Yao-Assistant"); assistantID != "" {
		return assistantID, nil
	}

	// Priority 3: Extract from model field (from CompletionRequest or Query)
	model := ""
	if req != nil && req.Model != "" {
		model = req.Model
	} else {
		model = c.Query("model")
	}

	if model != "" {
		// Parse model ID using the same logic as ParseModelID
		// Expected format: [prefix-]assistantName-model-yao_assistantID
		// Find the last occurrence of "-yao_"
		parts := strings.Split(model, "-yao_")
		if len(parts) >= 2 {
			assistantID := parts[len(parts)-1]
			if assistantID != "" {
				return assistantID, nil
			}
		}
	}

	// If no assistant ID found, return error
	return "", fmt.Errorf("assistant_id is required")
}

// GetMessages extracts messages from the request
// Priority:
// 1. Query parameter "messages" (JSON string)
// 2. CompletionRequest.Messages (from payload)
func GetMessages(c *gin.Context, req *CompletionRequest) ([]Message, error) {
	// Priority 1: Query parameter messages
	if messagesJSON := c.Query("messages"); messagesJSON != "" {
		var messages []Message
		if err := json.Unmarshal([]byte(messagesJSON), &messages); err == nil && len(messages) > 0 {
			return messages, nil
		}
	}

	// Priority 2: From CompletionRequest (payload)
	if req != nil && len(req.Messages) > 0 {
		return req.Messages, nil
	}

	return nil, fmt.Errorf("messages field is required")
}

// GetLocale extracts locale from request with priority:
// 1. Query parameter "locale"
// 2. Header "Accept-Language"
// 3. CompletionRequest metadata "locale" (from payload)
func GetLocale(c *gin.Context, req *CompletionRequest) string {
	// Priority 1: Query parameter
	if locale := c.Query("locale"); locale != "" {
		return strings.ToLower(locale)
	}

	// Priority 2: Header Accept-Language
	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		// Parse Accept-Language header (e.g., "en-US,en;q=0.9,zh;q=0.8")
		// Take the first language
		parts := strings.Split(acceptLang, ",")
		if len(parts) > 0 {
			// Remove quality value if present
			lang := strings.Split(parts[0], ";")[0]
			return strings.ToLower(strings.TrimSpace(lang))
		}
	}

	// Priority 3: From CompletionRequest metadata
	if req != nil && req.Metadata != nil {
		if locale, ok := req.Metadata["locale"]; ok {
			if localeStr, ok := locale.(string); ok && localeStr != "" {
				return strings.ToLower(localeStr)
			}
		}
	}

	return ""
}

// GetTheme extracts theme from request with priority:
// 1. Query parameter "theme"
// 2. Header "X-Yao-Theme"
// 3. CompletionRequest metadata "theme" (from payload)
func GetTheme(c *gin.Context, req *CompletionRequest) string {
	// Priority 1: Query parameter
	if theme := c.Query("theme"); theme != "" {
		return strings.ToLower(theme)
	}

	// Priority 2: Header
	if theme := c.GetHeader("X-Yao-Theme"); theme != "" {
		return strings.ToLower(theme)
	}

	// Priority 3: From CompletionRequest metadata
	if req != nil && req.Metadata != nil {
		if theme, ok := req.Metadata["theme"]; ok {
			if themeStr, ok := theme.(string); ok && themeStr != "" {
				return strings.ToLower(themeStr)
			}
		}
	}

	return ""
}

// GetReferer extracts referer from request with priority:
// 1. Query parameter "referer"
// 2. Header "X-Yao-Referer"
// 3. CompletionRequest metadata "referer" (from payload)
// 4. Default to "api"
func GetReferer(c *gin.Context, req *CompletionRequest) string {
	// Priority 1: Query parameter
	if referer := c.Query("referer"); referer != "" {
		return validateReferer(referer)
	}

	// Priority 2: Header
	if referer := c.GetHeader("X-Yao-Referer"); referer != "" {
		return validateReferer(referer)
	}

	// Priority 3: From CompletionRequest metadata
	if req != nil && req.Metadata != nil {
		if referer, ok := req.Metadata["referer"]; ok {
			if refererStr, ok := referer.(string); ok && refererStr != "" {
				return validateReferer(refererStr)
			}
		}
	}

	// Priority 4: Default
	return RefererAPI
}

// GetAccept extracts accept type from request with priority:
// 1. Query parameter "accept"
// 2. Header "X-Yao-Accept"
// 3. CompletionRequest metadata "accept" (from payload)
// 4. Default to "standard" (OpenAI-compatible format)
func GetAccept(c *gin.Context, req *CompletionRequest) Accept {
	// Priority 1: Query parameter
	if accept := c.Query("accept"); accept != "" {
		return validateAccept(accept)
	}

	// Priority 2: Header
	if accept := c.GetHeader("X-Yao-Accept"); accept != "" {
		return validateAccept(accept)
	}

	// Priority 3: From CompletionRequest metadata
	if req != nil && req.Metadata != nil {
		if accept, ok := req.Metadata["accept"]; ok {
			if acceptStr, ok := accept.(string); ok && acceptStr != "" {
				return validateAccept(acceptStr)
			}
		}
	}

	// Priority 4: Default to "standard" (OpenAI-compatible format)
	return AcceptStandard

	// // Future: Parse from User-Agent if needed
	// userAgent := c.GetHeader("User-Agent")
	// clientType := getClientType(userAgent)
	// return parseAccept(clientType)
}

// GetChatID get the chat ID from the request
// Priority:
// 1. Query parameter "chat_id"
// 2. Header "X-Yao-Chat"
// 3. CompletionRequest metadata "chat_id" (from payload)
// 4. Generate from messages using GetChatIDByMessages
func GetChatID(c *gin.Context, cache store.Store, req *CompletionRequest) (string, error) {
	// Priority 1: Query parameter chat_id
	if chatID := c.Query("chat_id"); chatID != "" {
		return chatID, nil
	}

	// Priority 2: Header X-Yao-Chat
	if chatID := c.GetHeader("X-Yao-Chat"); chatID != "" {
		return chatID, nil
	}

	// Priority 3: From CompletionRequest metadata
	if req != nil && req.Metadata != nil {
		if chatID, ok := req.Metadata["chat_id"]; ok {
			if chatIDStr, ok := chatID.(string); ok && chatIDStr != "" {
				return chatIDStr, nil
			}
		}
	}

	// Priority 4: Generate from messages
	messages, err := GetMessages(c, req)
	if err != nil {
		return "", fmt.Errorf("failed to get messages for chat ID generation: %w", err)
	}

	chatID, err := GetChatIDByMessages(cache, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate chat ID from messages: %w", err)
	}

	return chatID, nil
}

// GetRoute extracts route from request with priority:
// 1. Query parameter "route"
// 2. Header "X-Yao-Route"
// 3. CompletionRequest.Route (from payload)
func GetRoute(c *gin.Context, req *CompletionRequest) string {
	// Priority 1: Query parameter
	if route := c.Query("route"); route != "" {
		return route
	}

	// Priority 2: Header
	if route := c.GetHeader("X-Yao-Route"); route != "" {
		return route
	}

	// Priority 3: From CompletionRequest
	if req != nil && req.Route != "" {
		return req.Route
	}

	return ""
}

// GetMode extracts mode from request with priority:
// 1. Query parameter "mode"
// 2. Header "X-Yao-Mode"
// 3. CompletionRequest metadata "mode" (from payload)
func GetMode(c *gin.Context, req *CompletionRequest) string {
	// Priority 1: Query parameter
	if mode := c.Query("mode"); mode != "" {
		return mode
	}

	// Priority 2: Header
	if mode := c.GetHeader("X-Yao-Mode"); mode != "" {
		return mode
	}

	// Priority 3: From CompletionRequest metadata
	if req != nil && req.Metadata != nil {
		if mode, ok := req.Metadata["mode"]; ok {
			if modeStr, ok := mode.(string); ok && modeStr != "" {
				return modeStr
			}
		}
	}

	return ""
}

// GetSkip extracts skip configuration from request with priority:
// 1. CompletionRequest.Skip (from payload body) - Priority
// 2. Individual query parameters: "skip_history", "skip_trace"
func GetSkip(c *gin.Context, req *CompletionRequest) *Skip {
	// Priority 1: From CompletionRequest body (most direct)
	if req != nil && req.Skip != nil {
		return req.Skip
	}

	// Priority 2: Individual query parameters (recommended for query usage)
	skipHistory := c.Query("skip_history") == "true" || c.Query("skip_history") == "1"
	skipTrace := c.Query("skip_trace") == "true" || c.Query("skip_trace") == "1"

	// Check if any skip parameter is set
	if c.Query("skip_history") != "" || c.Query("skip_trace") != "" {
		return &Skip{
			History: skipHistory,
			Trace:   skipTrace,
		}
	}

	return nil
}

// GetMetadata extracts metadata from request with priority:
// 1. Query parameter "metadata" (JSON string)
// 2. Header "X-Yao-Metadata" (Base64 encoded JSON string)
// 3. CompletionRequest.Metadata (from payload)
func GetMetadata(c *gin.Context, req *CompletionRequest) map[string]interface{} {
	// Priority 1: Query parameter (JSON string)
	if metadataJSON := c.Query("metadata"); metadataJSON != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err == nil {
			return metadata
		}
	}

	// Priority 2: Header (Base64 encoded JSON string)
	if metadataBase64 := c.GetHeader("X-Yao-Metadata"); metadataBase64 != "" {
		// Try to decode Base64
		if decoded, err := base64.StdEncoding.DecodeString(metadataBase64); err == nil {
			var metadata map[string]interface{}
			if err := json.Unmarshal(decoded, &metadata); err == nil {
				return metadata
			}
		}
		// Fallback: try to parse as plain JSON
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataBase64), &metadata); err == nil {
			return metadata
		}
	}

	// Priority 3: From CompletionRequest
	if req != nil && req.Metadata != nil {
		return req.Metadata
	}

	return nil
}

// parseCompletionRequestData extracts CompletionRequest from the request
// Data can be passed via:
// 1. Request body (JSON payload) - Priority
// 2. Query parameters
func parseCompletionRequestData(c *gin.Context) (*CompletionRequest, error) {
	var req CompletionRequest

	// Try to parse from request body first
	if c.Request.Body != nil {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

		// Restore body for further processing
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// If body is not empty, try to parse it
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, fmt.Errorf("failed to parse completion request from body: %w", err)
			}

			// If we got valid data from body, validate and return
			// Model is optional if assistant_id can be extracted later
			if len(req.Messages) > 0 {
				return &req, nil
			}
		}
	}

	// Fallback: Try to parse from query parameters
	// Model is optional (can be extracted from assistant_id)
	model := c.Query("model")
	req.Model = model

	// Messages (required, must be JSON string in query)
	messagesJSON := c.Query("messages")
	if messagesJSON == "" {
		return nil, fmt.Errorf("messages field is required")
	}

	var messages []Message
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return nil, fmt.Errorf("failed to parse messages from query: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages field must not be empty")
	}
	req.Messages = messages

	// Optional fields from query
	if tempStr := c.Query("temperature"); tempStr != "" {
		var temp float64
		if _, err := fmt.Sscanf(tempStr, "%f", &temp); err == nil {
			req.Temperature = &temp
		}
	}

	if maxTokensStr := c.Query("max_tokens"); maxTokensStr != "" {
		var maxTokens int
		if _, err := fmt.Sscanf(maxTokensStr, "%d", &maxTokens); err == nil {
			req.MaxTokens = &maxTokens
		}
	}

	if maxCompletionTokensStr := c.Query("max_completion_tokens"); maxCompletionTokensStr != "" {
		var maxCompletionTokens int
		if _, err := fmt.Sscanf(maxCompletionTokensStr, "%d", &maxCompletionTokens); err == nil {
			req.MaxCompletionTokens = &maxCompletionTokens
		}
	}

	if streamStr := c.Query("stream"); streamStr != "" {
		stream := streamStr == "true" || streamStr == "1"
		req.Stream = &stream
	}

	// Audio config from query (JSON string)
	if audioJSON := c.Query("audio"); audioJSON != "" {
		var audio AudioConfig
		if err := json.Unmarshal([]byte(audioJSON), &audio); err == nil {
			req.Audio = &audio
		}
	}

	// Stream options from query (JSON string)
	if streamOptionsJSON := c.Query("stream_options"); streamOptionsJSON != "" {
		var streamOptions StreamOptions
		if err := json.Unmarshal([]byte(streamOptionsJSON), &streamOptions); err == nil {
			req.StreamOptions = &streamOptions
		}
	}

	// Metadata from query (JSON string)
	if metadataJSON := c.Query("metadata"); metadataJSON != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err == nil {
			req.Metadata = metadata
		}
	}

	return &req, nil
}
