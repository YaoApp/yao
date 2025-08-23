package chat

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/yao/neo"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the agent handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Protect all endpoints with OAuth
	group.Use(oauth.Guard)

	// Chat Completion
	group.GET("/completions", chatCompletion)
	group.POST("/completions", chatCompletion)

}

// Chat Completion (SSE)
// Note: This is a temporary implementation for full-process testing,
// and the interface may undergo significant global changes in the future.
func chatCompletion(c *gin.Context) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream;charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	sid := c.GetString("__sid")
	if sid == "" {
		sid = uuid.New().String()
	}

	content := c.Query("content")
	if content == "" {
		msg := message.New().Error("content is required").Done()
		msg.Write(c.Writer)
		return
	}

	chatID := c.Query("chat_id")
	if chatID == "" {
		// Only generate new chat_id if not provided
		chatID = fmt.Sprintf("chat_%d", time.Now().UnixNano())
	}

	// Set the context with validated chat_id
	ctx, cancel := chatctx.NewWithCancel(sid, chatID, c.Query("context"))
	defer cancel()
	defer ctx.Release() // Release the context after the request is done

	// Set the assistant ID
	assistantID := c.Query("assistant_id")
	if assistantID != "" {
		ctx = chatctx.WithAssistantID(ctx, assistantID)
	}

	// Set the silent mode
	silent := c.Query("silent")
	if silent == "true" || silent == "1" {
		ctx = chatctx.WithSilent(ctx, true)
	}

	// Set the history visible
	historyVisible := c.Query("history_visible")
	if historyVisible != "" {
		ctx = chatctx.WithHistoryVisible(ctx, historyVisible == "true" || historyVisible == "1")
	}

	// Set the client type
	clientType := c.Query("client_type")
	if clientType != "" {
		ctx = chatctx.WithClientType(ctx, clientType)
	}

	// Get neo instance and call Answer
	neoInstance := neo.GetNeo()
	err := neoInstance.Answer(ctx, content, c)

	// Error handling
	if err != nil {
		message.New().Done().Error(err).Write(c.Writer)
		c.Done()
		return
	}
}
