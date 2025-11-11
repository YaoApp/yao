package context

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// NewOpenAPI create a new context from openapi context
func NewOpenAPI(c *gin.Context, cache store.Store) Context {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Extract assistant ID (route parameter takes priority, handled in GetAssistantID)
	assistantID, _ := GetAssistantID(c)

	// Extract chat ID (may generate from messages if not provided)
	// GetChatID internally calls GetChatIDByMessages which auto-caches
	chatID, _ := GetChatID(c, cache)

	// Parse client information from User-Agent header
	userAgent := c.GetHeader("User-Agent")
	clientType := getClientType(userAgent)
	clientIP := c.ClientIP()

	// Create context with extracted parameters
	ctx := Context{
		Context:     c.Request.Context(),
		Space:       plan.NewMemorySharedSpace(),
		Authorized:  authInfo,
		ChatID:      chatID,
		AssistantID: assistantID,
		Locale:      GetLocale(c),
		Theme:       GetTheme(c),
		Referer:     GetReferer(c),
		Accept:      GetAccept(c),
		Client: Client{
			Type:      clientType,
			UserAgent: userAgent,
			IP:        clientIP,
		},
	}

	return ctx
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

// getPayloadField reads a string field from request body
func getPayloadField(c *gin.Context, fieldName string) string {
	if c.Request.Body == nil {
		return ""
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}

	// Restore body for further use
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	if len(body) == 0 {
		return ""
	}

	// Parse JSON to extract field
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	if value, ok := payload[fieldName]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}

	return ""
}

// GetAssistantID extracts assistant ID from request with priority:
// 1. Query parameter "assistant_id"
// 2. Header "X-Yao-Assistant"
// 3. Query parameter "model" - splits by "-" takes last field, extracts ID from "yao_xxx" prefix
// 4. Payload "model" field - same parsing as query parameter
func GetAssistantID(c *gin.Context) (string, error) {
	// Priority 1: Query parameter assistant_id
	if assistantID := c.Query("assistant_id"); assistantID != "" {
		return assistantID, nil
	}

	// Priority 2: Header X-Yao-Assistant
	if assistantID := c.GetHeader("X-Yao-Assistant"); assistantID != "" {
		return assistantID, nil
	}

	// Priority 3 & 4: Extract from model parameter (Query or Payload)
	model := c.Query("model")
	if model == "" {
		model = getPayloadField(c, "model")
	}

	if model != "" {
		// Split by "-" and get the last field
		parts := strings.Split(model, "-")
		lastField := strings.TrimSpace(parts[len(parts)-1])

		// Check if it has yao_ prefix
		if strings.HasPrefix(lastField, "yao_") {
			assistantID := strings.TrimPrefix(lastField, "yao_")
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
// 2. Request body "messages" field
func GetMessages(c *gin.Context) ([]Message, error) {
	// Try query parameter first
	if messagesJSON := c.Query("messages"); messagesJSON != "" {
		var messages []Message
		if err := json.Unmarshal([]byte(messagesJSON), &messages); err == nil && len(messages) > 0 {
			return messages, nil
		}
	}

	// Check if request body exists
	if c.Request.Body == nil {
		return nil, fmt.Errorf("messages field is required")
	}

	// Try request body
	// Read body carefully to allow reuse
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Restore body for further processing
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// If body is empty, return error
	if len(body) == 0 {
		return nil, fmt.Errorf("messages field is required")
	}

	var requestBody struct {
		Messages []Message `json:"messages"`
	}

	if err := json.Unmarshal(body, &requestBody); err != nil {
		return nil, fmt.Errorf("failed to parse messages from request body: %w", err)
	}

	if len(requestBody.Messages) == 0 {
		return nil, fmt.Errorf("messages field is required and must not be empty")
	}

	return requestBody.Messages, nil
}

// GetLocale extracts locale from request with priority:
// 1. Query parameter "locale"
// 2. Header "Accept-Language"
func GetLocale(c *gin.Context) string {
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

	return ""
}

// GetTheme extracts theme from request with priority:
// 1. Query parameter "theme"
// 2. Header "X-Yao-Theme"
func GetTheme(c *gin.Context) string {
	// Priority 1: Query parameter
	if theme := c.Query("theme"); theme != "" {
		return strings.ToLower(theme)
	}

	// Priority 2: Header
	if theme := c.GetHeader("X-Yao-Theme"); theme != "" {
		return strings.ToLower(theme)
	}

	return ""
}

// GetReferer extracts referer from request with priority:
// 1. Query parameter "referer"
// 2. Header "X-Yao-Referer"
// 3. Default to "api"
func GetReferer(c *gin.Context) string {
	// Priority 1: Query parameter
	if referer := c.Query("referer"); referer != "" {
		return validateReferer(referer)
	}

	// Priority 2: Header
	if referer := c.GetHeader("X-Yao-Referer"); referer != "" {
		return validateReferer(referer)
	}

	// Priority 3: Default
	return RefererAPI
}

// GetAccept extracts accept type from request with priority:
// 1. Query parameter "accept"
// 2. Header "X-Yao-Accept"
// 3. Parse from client type (User-Agent)
func GetAccept(c *gin.Context) Accept {
	// Priority 1: Query parameter
	if accept := c.Query("accept"); accept != "" {
		return validateAccept(accept)
	}

	// Priority 2: Header
	if accept := c.GetHeader("X-Yao-Accept"); accept != "" {
		return validateAccept(accept)
	}

	// Priority 3: Parse from User-Agent
	userAgent := c.GetHeader("User-Agent")
	clientType := getClientType(userAgent)
	return parseAccept(clientType)
}

// GetChatID get the chat ID from the request
// Priority:
// 1. Query parameter "chat_id"
// 2. Header "X-Yao-Chat"
// 3. Generate from messages using GetChatIDByMessages
func GetChatID(c *gin.Context, cache store.Store) (string, error) {
	// Priority 1: Query parameter chat_id
	if chatID := c.Query("chat_id"); chatID != "" {
		return chatID, nil
	}

	// Priority 2: Header X-Yao-Chat
	if chatID := c.GetHeader("X-Yao-Chat"); chatID != "" {
		return chatID, nil
	}

	// Priority 3: Generate from messages
	messages, err := GetMessages(c)
	if err != nil {
		return "", fmt.Errorf("failed to get messages for chat ID generation: %w", err)
	}

	chatID, err := GetChatIDByMessages(cache, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate chat ID from messages: %w", err)
	}

	return chatID, nil
}
