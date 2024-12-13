package neo

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/helper"
)

// API registers the Neo API endpoints
func (neo *DSL) API(router *gin.Engine, path string) error {

	// Get the guards
	middlewares, err := neo.getGuardHandlers()
	if err != nil {
		return err
	}

	// Cross-Domain handlers
	cors, err := neo.getCorsHandlers(router, path)
	if err != nil {
		return err
	}

	// Append cors handlers
	middlewares = append(middlewares, cors...)

	// Register chat endpoint
	router.GET(path, append(middlewares, neo.handleChat)...)
	router.POST(path, append(middlewares, neo.handleChat)...)

	// Register chat list endpoint
	router.GET(path+"/chats", append(middlewares, neo.handleChatList)...)

	// Register chat history endpoint
	router.GET(path+"/history", append(middlewares, neo.handleChatHistory)...)

	return nil
}

// handleChat handles the chat request
func (neo *DSL) handleChat(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		sid = uuid.New().String()
	}

	content := c.Query("content")
	if content == "" {
		c.JSON(400, gin.H{"message": "content is required", "code": 400})
		return
	}

	// Set the context
	ctx, cancel := NewContextWithCancel(sid, c.Query("chat_id"), c.Query("context"))
	defer cancel()

	err := neo.Answer(ctx, content, c)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
	}
}

// handleChatList handles the chat list request
func (neo *DSL) handleChatList(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	list, err := neo.Conversation.GetChats(sid)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, map[string]interface{}{"data": list})
	c.Done()
}

// handleChatHistory handles the chat history request
func (neo *DSL) handleChatHistory(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	cid := c.Query("chat_id")
	history, err := neo.Conversation.GetHistory(sid, cid)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, map[string]interface{}{"data": history})
	c.Done()
}

// getCorsHandlers returns CORS middleware handlers
func (neo *DSL) getCorsHandlers(router *gin.Engine, path string) ([]gin.HandlerFunc, error) {
	if len(neo.Allows) == 0 {
		return []gin.HandlerFunc{}, nil
	}

	allowsMap := map[string]bool{}
	for _, allow := range neo.Allows {
		allow = strings.TrimPrefix(allow, "http://")
		allow = strings.TrimPrefix(allow, "https://")
		allowsMap[allow] = true
	}

	router.OPTIONS(path+"/history", neo.optionsHandler)
	router.OPTIONS(path+"/commands", neo.optionsHandler)
	return []gin.HandlerFunc{neo.corsMiddleware(allowsMap)}, nil
}

// corsMiddleware handles CORS requests
func (neo *DSL) corsMiddleware(allowsMap map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		referer := neo.getOrigin(c)
		if referer != "" {
			if !api.IsAllowed(c, allowsMap) {
				c.JSON(403, gin.H{"message": referer + " not allowed", "code": 403})
				c.Abort()
				return
			}
			url, _ := url.Parse(referer)
			referer = fmt.Sprintf("%s://%s", url.Scheme, url.Host)
			c.Writer.Header().Set("Access-Control-Allow-Origin", referer)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
			c.Next()
		}
	}
}

// optionsHandler handles OPTIONS requests
func (neo *DSL) optionsHandler(c *gin.Context) {
	origin := neo.getOrigin(c)
	c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.AbortWithStatus(204)
}

// getOrigin returns the request origin
func (neo *DSL) getOrigin(c *gin.Context) string {
	referer := c.Request.Referer()
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		origin = referer
	}
	return origin
}

// getGuardHandlers returns authentication middleware handlers
func (neo *DSL) getGuardHandlers() ([]gin.HandlerFunc, error) {
	if neo.Guard == "" {
		return []gin.HandlerFunc{neo.defaultGuard}, nil
	}

	// Validate the custom guard
	_, err := process.Of(neo.Guard)
	if err != nil {
		return nil, err
	}

	// Return custom guard
	return []gin.HandlerFunc{api.ProcessGuard(neo.Guard)}, nil
}

// defaultGuard is the default authentication handler
func (neo *DSL) defaultGuard(c *gin.Context) {
	token := strings.TrimSpace(strings.TrimPrefix(c.Query("token"), "Bearer "))
	if token == "" {
		c.JSON(403, gin.H{"message": "token is required", "code": 403})
		c.Abort()
		return
	}

	user := helper.JwtValidate(token)
	c.Set("__sid", user.SID)
	c.Next()
}
