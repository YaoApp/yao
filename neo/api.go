package neo

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/neo/conversation"
	"github.com/yaoapp/yao/neo/message"
)

// API registers the Neo API endpoints
func (neo *DSL) API(router *gin.Engine, path string) error {

	// Get the guards
	middlewares, err := neo.getGuardHandlers()
	if err != nil {
		return err
	}

	// Register OPTIONS handlers for all endpoints
	router.OPTIONS(path, neo.optionsHandler)
	router.OPTIONS(path+"/status", neo.optionsHandler)
	router.OPTIONS(path+"/chats", neo.optionsHandler)
	router.OPTIONS(path+"/chats/:id", neo.optionsHandler)
	router.OPTIONS(path+"/history", neo.optionsHandler)
	router.OPTIONS(path+"/upload", neo.optionsHandler)
	router.OPTIONS(path+"/download", neo.optionsHandler)
	router.OPTIONS(path+"/mentions", neo.optionsHandler)
	router.OPTIONS(path+"/dangerous/clear_chats", neo.optionsHandler)

	// Register endpoints with middlewares
	router.GET(path, append(middlewares, neo.handleChat)...)
	router.POST(path, append(middlewares, neo.handleChat)...)

	// Status check
	router.GET(path+"/status", append(middlewares, neo.handleStatus)...)

	// Chat api
	router.GET(path+"/chats", append(middlewares, neo.handleChatList)...)
	router.GET(path+"/chats/:id", append(middlewares, neo.handleChatDetail)...)
	router.POST(path+"/chats/:id", append(middlewares, neo.handleChatUpdate)...)
	router.DELETE(path+"/chats/:id", append(middlewares, neo.handleChatDelete)...)

	// History api
	router.GET(path+"/history", append(middlewares, neo.handleChatHistory)...)

	// File api
	router.POST(path+"/upload", append(middlewares, neo.handleUpload)...)
	router.GET(path+"/download", append(middlewares, neo.handleDownload)...)

	// Mention api
	router.GET(path+"/mentions", append(middlewares, neo.handleMentions)...)

	// Dangerous operations
	router.DELETE(path+"/dangerous/clear_chats", append(middlewares, neo.handleChatsDeleteAll)...)

	return nil
}

// handleStatus handles the status request
func (neo *DSL) handleStatus(c *gin.Context) {
	c.Status(200)
	c.Done()
}

// handleUpload handles the upload request
func (neo *DSL) handleUpload(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		sid = uuid.New().String()
	}

	// Set the context
	ctx, cancel := NewContextWithCancel(sid, c.Query("chat_id"), "")
	defer cancel()

	// Upload the file
	file, err := neo.Upload(ctx, c)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, file)
	c.Done()
}

// handleChat handles the chat request
func (neo *DSL) handleChat(c *gin.Context) {
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
	ctx, cancel := NewContextWithCancel(sid, chatID, c.Query("context"))
	defer cancel()

	neo.Answer(ctx, content, c)
}

// handleChatList handles the chat list request
func (neo *DSL) handleChatList(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	// Create filter from query parameters
	filter := conversation.ChatFilter{
		Keywords: c.Query("keywords"),
		Order:    c.Query("order"),
	}

	// Parse page and pagesize
	if page := c.Query("page"); page != "" {
		if n, err := strconv.Atoi(page); err == nil {
			filter.Page = n
		}
	}

	if pageSize := c.Query("pagesize"); pageSize != "" {
		if n, err := strconv.Atoi(pageSize); err == nil {
			filter.PageSize = n
		}
	}

	response, err := neo.Conversation.GetChats(sid, filter)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, map[string]interface{}{"data": response})
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

// handleDownload handles the download request
func (neo *DSL) handleDownload(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	fileID := c.Query("file_id")
	if fileID == "" {
		c.JSON(400, gin.H{"message": "file_id is required", "code": 400})
		c.Done()
		return
	}

	// Set the context
	ctx, cancel := NewContextWithCancel(sid, c.Query("chat_id"), "")
	defer cancel()

	// Download the file
	fileResponse, err := neo.Download(ctx, c)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}
	defer fileResponse.Reader.Close()

	// Set response headers
	c.Header("Content-Type", fileResponse.ContentType)
	if disposition := c.Query("disposition"); disposition == "attachment" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fileID)+fileResponse.Extension))
	}

	// Copy the file content to response
	_, err = io.Copy(c.Writer, fileResponse.Reader)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		return
	}
}

// getCorsHandlers returns CORS middleware handlers
func (neo *DSL) getCorsHandlers() ([]gin.HandlerFunc, error) {
	if len(neo.Allows) == 0 {
		return []gin.HandlerFunc{}, nil
	}

	allowsMap := map[string]bool{}
	for _, allow := range neo.Allows {
		allow = strings.TrimPrefix(allow, "http://")
		allow = strings.TrimPrefix(allow, "https://")
		allowsMap[allow] = true
	}

	return []gin.HandlerFunc{neo.corsMiddleware(allowsMap)}, nil
}

// corsMiddleware handles CORS requests
func (neo *DSL) corsMiddleware(allowsMap map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := neo.getOrigin(c)
		if origin == "" {
			c.Next()
			return
		}

		// Check if origin is allowed
		if !api.IsAllowed(c, allowsMap) {
			c.AbortWithStatusJSON(403, gin.H{
				"message": origin + " not allowed",
				"code":    403,
			})
			return
		}

		// Set CORS headers
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// optionsHandler handles OPTIONS requests
func (neo *DSL) optionsHandler(c *gin.Context) {
	origin := neo.getOrigin(c)
	if origin != "" {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours
	}
	c.AbortWithStatus(204)
}

// getOrigin returns the request origin
func (neo *DSL) getOrigin(c *gin.Context) string {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		origin = c.Request.Referer()
		if origin != "" {
			if u, err := url.Parse(origin); err == nil {
				origin = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			}
		}
	}
	return origin
}

// getGuardHandlers returns authentication middleware handlers
func (neo *DSL) getGuardHandlers() ([]gin.HandlerFunc, error) {

	// Cross-Domain handlers
	cors, err := neo.getCorsHandlers()
	if err != nil {
		return nil, err
	}

	if neo.Guard == "" {
		middlewares := append(cors, neo.defaultGuard)
		return middlewares, nil
	}

	// Validate the custom guard
	_, err = process.Of(neo.Guard)
	if err != nil {
		return nil, err
	}

	middlewares := append(cors, api.ProcessGuard(neo.Guard, cors...))
	return middlewares, nil
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

// handleChatDetail handles getting a single chat's details
func (neo *DSL) handleChatDetail(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	chatID := c.Param("id")
	if chatID == "" {
		c.JSON(400, gin.H{"message": "chat id is required", "code": 400})
		c.Done()
		return
	}

	chat, err := neo.Conversation.GetChat(sid, chatID)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, map[string]interface{}{"data": chat})
	c.Done()
}

// handleMentions handles getting mentions for a chat
func (neo *DSL) handleMentions(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	// Get keywords from query parameter
	keywords := strings.ToLower(c.Query("keywords"))
	mentions, err := neo.GetMentions(keywords)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// Add test data
	testMentions := []Mention{
		{
			ID:     "assistant_1",
			Name:   "Alice AI",
			Type:   "assistant",
			Avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Alice",
		},
		{
			ID:     "assistant_2",
			Name:   "Bob Bot",
			Type:   "assistant",
			Avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Bob",
		},
		{
			ID:     "assistant_3",
			Name:   "Carol AI",
			Type:   "assistant",
			Avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Carol",
		},
	}

	// Filter mentions by keywords
	if keywords != "" {
		filtered := []Mention{}
		for _, m := range testMentions {
			if strings.Contains(strings.ToLower(m.Name), keywords) {
				filtered = append(filtered, m)
			}
		}
		testMentions = filtered
	}

	// Append test data to actual mentions
	mentions = append(mentions, testMentions...)

	c.JSON(200, map[string]interface{}{"data": mentions})
	c.Done()
}

// handleChatUpdate handles updating a chat's details
func (neo *DSL) handleChatUpdate(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	chatID := c.Param("id")
	if chatID == "" {
		c.JSON(400, gin.H{"message": "chat id is required", "code": 400})
		c.Done()
		return
	}

	// Get title from request body
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, gin.H{"message": "invalid request body", "code": 400})
		c.Done()
		return
	}

	// If content is not empty, Generate the chat title
	if body.Content != "" {
		ctx, cancel := NewContextWithCancel(sid, c.Query("chat_id"), "")
		defer cancel()

		title, err := neo.GenerateChatTitle(ctx, body.Content, c)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
			return
		}
		body.Title = title
	}

	if body.Title == "" {
		c.JSON(400, gin.H{"message": "title is required", "code": 400})
		c.Done()
		return
	}

	err := neo.Conversation.UpdateChatTitle(sid, chatID, body.Title)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, gin.H{"message": "ok", "title": body.Title, "chat_id": chatID})
	c.Done()
}

// handleChatDelete handles deleting a single chat
func (neo *DSL) handleChatDelete(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	chatID := c.Param("id")
	if chatID == "" {
		c.JSON(400, gin.H{"message": "chat id is required", "code": 400})
		c.Done()
		return
	}

	err := neo.Conversation.DeleteChat(sid, chatID)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, gin.H{"message": "ok"})
	c.Done()
}

// handleChatsDeleteAll handles deleting all chats for a user
func (neo *DSL) handleChatsDeleteAll(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	err := neo.Conversation.DeleteAllChats(sid)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, gin.H{"message": "ok"})
	c.Done()
}
