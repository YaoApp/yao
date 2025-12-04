package chat

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/response"
)

// GinCreateCompletions handles POST /chat/:assistant_id/completions - Create a chat completion
func GinCreateCompletions(c *gin.Context) {

	agent := agent.GetAgent()
	cache, err := agent.GetCacheStore()
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get cache store: " + err.Error(),
		})
		return
	}

	completionReq, ctx, opts, err := context.GetCompletionRequest(c, cache)
	if err != nil {
		fmt.Println("-----------------------------------------------")
		fmt.Println("Error: ", err.Error())
		fmt.Println("-----------------------------------------------")

		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to parse request: " + err.Error(),
		})
		return
	}

	defer func() {
		log.Trace("[HTTP] Handler defer: calling ctx.Release()")
		ctx.Release()
	}()

	ast, err := assistant.Get(ctx.AssistantID)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get assistant: " + err.Error(),
		})
		return
	}

	// Set SSE headers for streaming response
	c.Header("Content-Type", "text/event-stream;charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable buffering in nginx

	// Stream the completion (uses default handler which sends to ctx.Writer)
	// The Stream method will automatically close the writer and send [DONE] marker
	log.Trace("[HTTP] Calling ast.Stream()")
	_, err = ast.Stream(ctx, completionReq.Messages, opts)
	log.Trace("[HTTP] ast.Stream() returned, err=%v", err)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to stream: " + err.Error(),
		})
		return
	}

	// c.JSON(response.StatusOK, gin.H{
	// 	"message":        "Create Completions",
	// 	"chat_id":        ctx.ChatID,
	// 	"assistant_id":   ctx.AssistantID,
	// 	"model":          completionReq.Model,
	// 	"messages_count": len(completionReq.Messages),
	// })

	// // Print headers
	// fmt.Println("\n--- Headers ---")
	// for key, values := range c.Request.Header {
	// 	for _, value := range values {
	// 		fmt.Printf("%s: %s\n", key, value)
	// 	}
	// }

	// // Print path parameters
	// fmt.Println("\n--- Path Parameters ---")
	// for _, param := range c.Params {
	// 	fmt.Printf("%s: %s\n", param.Key, param.Value)
	// }

	// // Print query parameters
	// fmt.Println("\n--- Query Parameters ---")
	// for key, values := range c.Request.URL.Query() {
	// 	for _, value := range values {
	// 		fmt.Printf("%s: %s\n", key, value)
	// 	}
	// }

	// // Print request body
	// fmt.Println("\n--- Request Body ---")
	// body, err = io.ReadAll(c.Request.Body)
	// if err != nil {
	// 	fmt.Printf("Error reading body: %v\n", err)
	// } else {
	// 	fmt.Printf("%s\n", string(body))
	// 	// Restore the body for further processing
	// 	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	// }
	// fmt.Println("===============================================")

	// // Handle Sid - try multiple methods for maximum compatibility
	// var sid string

	// // Method 1: Check if client sent X-Session-Id header
	// sid = c.GetHeader("X-Session-Id")

	// // Method 2: Try to read from cookie
	// if sid == "" {
	// 	sid, err = c.Cookie("Sid")
	// 	if err == nil && sid != "" {
	// 		fmt.Printf("Existing Sid from cookie: %s\n", sid)
	// 	}
	// } else {
	// 	fmt.Printf("Existing Sid from header: %s\n", sid)
	// }

	// // Method 3: For clients that can't store cookies/headers (like Electron cross-origin),
	// // generate a deterministic session ID based on client fingerprint
	// if sid == "" {
	// 	// Use Authorization token if available (most stable identifier)
	// 	authToken := c.GetHeader("Authorization")
	// 	userAgent := c.GetHeader("User-Agent")

	// 	if authToken != "" {
	// 		// Generate stable session ID from auth token
	// 		hash := md5.Sum([]byte(authToken))
	// 		sid = hex.EncodeToString(hash[:])
	// 		fmt.Printf("Generated deterministic Sid from auth token: %s\n", sid)
	// 	} else {
	// 		// Fallback: generate random UUID
	// 		sid = uuid.New().String()
	// 		fmt.Printf("Generated random Sid: %s\n", sid)
	// 	}

	// 	fmt.Printf("Client fingerprint - UserAgent: %s\n", userAgent)
	// }

	// // Try to set cookie (may not work for cross-origin, but doesn't hurt)
	// c.SetCookie("Sid", sid, 86400*30, "/", "", false, false)

	// // Return Sid in response header and body for client reference
	// c.Header("X-Session-Id", sid)

	// response.RespondWithSuccess(c, response.StatusOK, gin.H{"message": "Create Completions", "sid": sid})
}

// GinUpdateCompletions handles PUT /chat/:assistant_id/completions - Update a chat completion metadata
func GinUpdateCompletions(c *gin.Context) {}

// GinAppendMessages handles POST /chat/:assistant_id/completions/:context_id/append
// Appends messages to a running completion (for user pre-input while AI is still generating)
func GinAppendMessages(c *gin.Context) {
	// Get context_id from URL parameter
	contextID := c.Param("context_id")
	if contextID == "" {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "context_id is required",
		})
		return
	}

	// Parse request body
	var req AppendMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate interrupt type
	if req.Type != context.InterruptGraceful && req.Type != context.InterruptForce {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid interrupt type. Must be 'graceful' or 'force'",
		})
		return
	}

	// Validate messages
	// Allow empty messages for force interrupt (pure cancellation without appending)
	if len(req.Messages) == 0 && req.Type != context.InterruptForce {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "At least one message is required (unless force interrupt for cancellation)",
		})
		return
	}

	// Create interrupt signal
	signal := &context.InterruptSignal{
		Type:      req.Type,
		Messages:  req.Messages,
		Timestamp: time.Now().UnixMilli(),
		Metadata:  req.Metadata,
	}

	// Send interrupt signal to the context
	if err := context.SendInterrupt(contextID, signal); err != nil {
		log.Trace("[INTERRUPT] Failed to send interrupt signal: context_id=%s, error=%v", contextID, err)
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to send interrupt: " + err.Error(),
		})
		return
	}

	// Return success response
	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"message":    "Messages appended successfully",
		"context_id": contextID,
		"type":       req.Type,
		"timestamp":  signal.Timestamp,
	})
}
