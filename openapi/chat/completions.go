package chat

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent"
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

	ctx := context.NewOpenAPI(c, cache)

	fmt.Println("-----------------------------------------------")
	fmt.Println("Chat ID: ", ctx.ChatID)
	fmt.Println("Assistant ID: ", ctx.AssistantID)
	fmt.Println("----")
	// utils.Dump(ctx)
	// Print request body
	fmt.Println("\n--- Request Body ---")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Printf("Error reading body: %v\n", err)
	} else {
		fmt.Printf("%s\n", string(body))
		// Restore the body for further processing
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	fmt.Println("-----------------------------------------------")

	c.JSON(response.StatusOK, gin.H{"message": "Create Completions", "chat_id": ctx.ChatID})

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
