package neo

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/neo/assistant"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/openapi/oauth"
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
	router.OPTIONS(path+"/upload/:storage", neo.optionsHandler)
	router.OPTIONS(path+"/download", neo.optionsHandler)
	router.OPTIONS(path+"/mentions", neo.optionsHandler)
	router.OPTIONS(path+"/generate", neo.optionsHandler)
	router.OPTIONS(path+"/generate/title", neo.optionsHandler)
	router.OPTIONS(path+"/generate/prompts", neo.optionsHandler)
	router.OPTIONS(path+"/dangerous/clear_chats", neo.optionsHandler)
	router.OPTIONS(path+"/assistants", neo.optionsHandler)
	router.OPTIONS(path+"/assistants/:id", neo.optionsHandler)
	router.OPTIONS(path+"/assistants/:id/call", neo.optionsHandler)

	// Chat endpoint
	// Chat endpoint
	// Example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo?content=Hello&chat_id=chat_123&context=previous_context&token=xxx'
	// curl -X POST 'http://localhost:5099/api/__yao/neo' \
	//   -H 'Content-Type: application/json' \
	//   -d '{"content": "Hello", "chat_id": "chat_123", "context": "previous_context", "token": "xxx"}'
	router.GET(path, append(middlewares, neo.handleChat)...)
	router.POST(path, append(middlewares, neo.handleChat)...)

	// Status check endpoint
	// Example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/status?token=xxx'
	router.GET(path+"/status", append(middlewares, neo.handleStatus)...)

	// Assistant API endpoints
	// List assistants example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/assistants?page=1&pagesize=20&tags=tag1,tag2&token=xxx'
	router.GET(path+"/assistants", append(middlewares, neo.handleAssistantList)...)
	// Get all assistant tags example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/assistants/tags?token=xxx'
	router.GET(path+"/assistants/tags", append(middlewares, neo.handleAssistantTags)...)

	// Get assistant details example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/assistants/assistant_123?token=xxx'
	router.GET(path+"/assistants/:id", append(middlewares, neo.handleAssistantDetail)...)

	// Execute assistant API example:
	// curl -X POST 'http://localhost:5099/api/__yao/neo/assistants/assistant_123/api' \
	//   -H 'Content-Type: application/json' \
	//   -d '{"name": "Test", "payload": {"name": "yao", "age": 18}}'
	router.POST(path+"/assistants/:id/call", append(middlewares, neo.handleAssistantCall)...)

	// Create/Update assistant example:
	// curl -X POST 'http://localhost:5099/api/__yao/neo/assistants' \
	//   -H 'Content-Type: application/json' \
	//   -d '{"name": "My Assistant", "type": "chat", "tags": ["tag1", "tag2"], "mentionable": true, "avatar": "path/to/avatar.png", "token": "xxx"}'
	router.POST(path+"/assistants", append(middlewares, neo.handleAssistantSave)...)

	// Delete assistant example:
	// curl -X DELETE 'http://localhost:5099/api/__yao/neo/assistants/assistant_123?token=xxx'
	router.DELETE(path+"/assistants/:id", append(middlewares, neo.handleAssistantDelete)...)

	// Chat management endpoints
	// List chats example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/chats?page=1&pagesize=20&keywords=search+term&order=desc&token=xxx'
	router.GET(path+"/chats", append(middlewares, neo.handleChatList)...)

	// Get latest chat example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/chats/latest?assistant_id=assistant_123&token=xxx'
	router.GET(path+"/chats/latest", append(middlewares, neo.handleChatLatest)...)

	// Get chat details example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/chats/chat_123?token=xxx'
	router.GET(path+"/chats/:id", append(middlewares, neo.handleChatDetail)...)

	// Update chat example:
	// curl -X POST 'http://localhost:5099/api/__yao/neo/chats/chat_123' \
	//   -H 'Content-Type: application/json' \
	//   -d '{"title": "New Title", "content": "Chat content for title generation", "token": "xxx"}'
	router.POST(path+"/chats/:id", append(middlewares, neo.handleChatUpdate)...)

	// Delete chat example:
	// curl -X DELETE 'http://localhost:5099/api/__yao/neo/chats/chat_123?token=xxx'
	router.DELETE(path+"/chats/:id", append(middlewares, neo.handleChatDelete)...)

	// Chat history endpoint
	// Example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/history?chat_id=chat_123&token=xxx'
	router.GET(path+"/history", append(middlewares, neo.handleChatHistory)...)

	// File management endpoints
	// Upload file example:
	// curl -X POST 'http://localhost:5099/api/__yao/neo/upload?chat_id=chat_123&token=xxx' \
	//   -F 'file=@/path/to/file.txt'
	router.POST(path+"/upload/:storage", append(middlewares, neo.handleUpload)...)

	// Download file example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/download?file_id=file_123&disposition=attachment&token=xxx' \
	//   -o downloaded_file.txt
	router.GET(path+"/download", append(middlewares, neo.handleDownload)...)

	// Mentions endpoint
	// Example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/mentions?keywords=assistant&token=xxx'
	router.GET(path+"/mentions", append(middlewares, neo.handleMentions)...)

	// Generate title example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/generate/title?content=Chat+content&chat_id=chat_123&token=xxx'
	// curl -X POST 'http://localhost:5099/api/__yao/neo/generate/title' \
	//   -H 'Content-Type: application/json' \
	//   -d '{"content": "Chat content", "chat_id": "chat_123", "token": "xxx"}'
	router.GET(path+"/generate/title", append(middlewares, neo.handleGenerateTitle)...)
	router.POST(path+"/generate/title", append(middlewares, neo.handleGenerateTitle)...)

	// Generate prompts example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/generate/prompts?content=Generate+prompts&chat_id=chat_123&token=xxx'
	// curl -X POST 'http://localhost:5099/api/__yao/neo/generate/prompts' \
	//   -H 'Content-Type: application/json' \
	//   -d '{"content": "Generate prompts", "chat_id": "chat_123", "token": "xxx"}'
	router.GET(path+"/generate/prompts", append(middlewares, neo.handleGeneratePrompts)...)
	router.POST(path+"/generate/prompts", append(middlewares, neo.handleGeneratePrompts)...)

	// Utility endpoints
	// List connectors example:
	// curl -X GET 'http://localhost:5099/api/__yao/neo/utility/connectors?token=xxx'
	router.GET(path+"/utility/connectors", append(middlewares, neo.handleConnectors)...)

	// Dangerous operations
	// Dangerous operations
	// Clear all chats example:
	// curl -X DELETE 'http://localhost:5099/api/__yao/neo/dangerous/clear_chats?token=xxx'
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

	uid, isGuest, err := neo.UserOrGuestID(sid)
	if err != nil {
		c.JSON(401, gin.H{"message": fmt.Sprintf("Unauthorized, %s", err.Error()), "code": 401})
		c.Done()
		return
	}

	if uid == nil || uid == "" {
		c.JSON(401, gin.H{"message": "Unauthorized", "code": 401})
		c.Done()
		return
	}

	// Storage name must be chat, knowledge or assets
	storage := c.Param("storage")
	if storage != "chat" && storage != "knowledge" && storage != "assets" {
		c.JSON(400, gin.H{"message": "Invalid storage", "code": 400})
		c.Done()
		return
	}

	// Get the manager
	var manager, ok = attachment.Managers[storage]
	if !ok {
		c.JSON(400, gin.H{"message": "Invalid storage: " + storage, "code": 400})
		c.Done()
		return
	}

	// Get Option from form data
	var option UploadOption
	err = c.ShouldBind(&option)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": 400})
		c.Done()
		return
	}

	// Validate the option with the storage
	option.UserID = fmt.Sprintf("%v", uid)

	// Build multi-level groups based on storage type and IDs
	var groups []string
	switch storage {
	case "chat":
		if option.ChatID == "" {
			c.JSON(400, gin.H{"message": "chat_id is required", "code": 400})
			c.Done()
			return
		}
		// Build groups: ["users", "user123", "chats", "chat456"]
		groups = []string{"users", option.UserID, "chats", option.ChatID}
		if option.AssistantID != "" {
			// Add assistant level: ["users", "user123", "chats", "chat456", "assistants", "assistant789"]
			groups = append(groups, "assistants", option.AssistantID)
		}
	case "knowledge":
		if option.CollectionID == "" {
			c.JSON(400, gin.H{"message": "collection_id is required", "code": 400})
			c.Done()
			return
		}
		// Build groups: ["knowledge", "collection123", "users", "user456"]
		groups = []string{"knowledge", option.CollectionID, "users", option.UserID}
	case "assets":
		// Build groups: ["assets", "users", "user123"]
		groups = []string{"assets", "users", option.UserID}
	}

	// Set the groups in the attachment upload option
	option.UploadOption.Groups = groups

	// Get the file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": 400})
		c.Done()
		return
	}

	// Open the file
	reader, err := file.Open()
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}
	defer func() {
		reader.Close()
		os.Remove(file.Filename)
	}()

	// Upload the file
	header := attachment.GetHeader(c.Request.Header, file.Header, file.Size)
	res, err := manager.Upload(c.Request.Context(), header, reader, option.UploadOption)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// if storage is chat or knowledge, save the file to the store
	if storage == "chat" || storage == "knowledge" {

		attachment := map[string]interface{}{
			"file_id":      res.ID,
			"uid":          uid,
			"guest":        isGuest,
			"manager":      storage,
			"public":       option.Public,
			"name":         option.OriginalFilename,
			"content_type": res.ContentType,
			"bytes":        res.Bytes,
			"gzip":         option.Gzip,
			"status":       res.Status,
		}

		// Set the scope
		if option.Scope != nil {
			attachment["scope"] = option.Scope
		}

		// Set the collection_id
		if option.CollectionID != "" {
			attachment["collection_id"] = option.CollectionID
		}

		_, err = neo.Store.SaveAttachment(attachment)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
			return
		}
	}

	c.JSON(200, map[string]interface{}{"data": res})
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

	uid, _, err := neo.UserOrGuestID(sid)
	if err != nil {
		c.JSON(401, gin.H{"message": fmt.Sprintf("Unauthorized, %s", err.Error()), "code": 401})
		c.Done()
		return
	}

	if uid == nil || uid == "" {
		c.JSON(401, gin.H{"message": "Unauthorized", "code": 401})
		c.Done()
		return
	}

	fileID := c.Query("file_id")
	if fileID == "" {
		c.JSON(400, gin.H{"message": "file_id is required", "code": 400})
		c.Done()
		return
	}

	// Get the attachment
	attach, err := neo.Store.GetAttachment(fileID)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// Validate the permission ( Will be supported scope validation in the future )
	if (attach["public"] == 0 || attach["public"] == false) && attach["uid"] != uid {
		c.JSON(403, gin.H{"message": "Forbidden", "code": 403})
		c.Done()
		return
	}

	storage, ok := attach["manager"].(string)
	if !ok {
		c.JSON(400, gin.H{"message": "Invalid storage", "code": 400})
		c.Done()
		return
	}

	// Get the manager
	manager, ok := attachment.Managers[storage]
	if !ok {
		c.JSON(400, gin.H{"message": "Invalid storage", "code": 400})
		c.Done()
		return
	}

	name, ok := attach["name"].(string)
	if !ok {
		c.JSON(400, gin.H{"message": "Invalid name", "code": 400})
		c.Done()
		return
	}

	name = strings.TrimSuffix(name, ".gz")
	contentType, ok := attach["content_type"].(string)
	if !ok {
		c.JSON(400, gin.H{"message": "Invalid content type", "code": 400})
		c.Done()
		return
	}

	handle, err := manager.Download(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}
	defer handle.Reader.Close()

	// Set the response headers
	encoded := url.PathEscape(name)
	disposition := fmt.Sprintf(`attachment; filename="%s"`, encoded)
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", disposition)

	// Copy the file content to response
	_, err = io.Copy(c.Writer, handle.Reader)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		return
	}
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

	err := neo.Answer(ctx, content, c)

	// Error handling
	if err != nil {
		message.New().Done().Error(err).Write(c.Writer)
		c.Done()
		return
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

	// Create filter from query parameters
	filter := store.ChatFilter{
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

	locale := "en-us"
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	response, err := neo.Store.GetChats(sid, filter, locale)
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
	history, err := neo.Store.GetHistory(sid, cid)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, map[string]interface{}{"data": history})
	c.Done()
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
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Disposition, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, Content-Sync, Content-Fingerprint, Content-Uid, Content-Range")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Type, Content-Disposition, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, Content-Sync, Content-Fingerprint, Content-Uid, Content-Range")

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
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Disposition, Authorization, Accept, Content-Sync, Content-Fingerprint, Content-Uid, Content-Range")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours
		c.Header("Access-Control-Expose-Headers", "Content-Type, Content-Disposition, Authorization, Accept, Content-Sync, Content-Fingerprint, Content-Uid, Content-Range")
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

	// Check if the request is for OpenAPI OAuth
	if oauth.OAuth != nil {
		neo.guardOpenapiOauth(c)
		return
	}

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

// Openapi Oauth
func (neo *DSL) guardOpenapiOauth(c *gin.Context) {
	s := oauth.OAuth
	token := neo.getAccessToken(c)
	if token == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	// Validate the token
	_, err := s.VerifyToken(token)
	if err != nil {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	// Get the session ID
	sid := neo.getSessionID(c)
	if sid == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	c.Set("__sid", sid)
}

func (neo *DSL) getAccessToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if token == "" || token == "Bearer undefined" {
		cookie, err := c.Cookie("__Host-access_token")
		if err != nil {
			return ""
		}
		token = cookie
	}
	return strings.TrimPrefix(token, "Bearer ")
}

func (neo *DSL) getSessionID(c *gin.Context) string {
	sid, err := c.Cookie("__Host-session_id")
	if err != nil {
		return ""
	}
	return sid
}

// handleChatLatest handles getting the latest chat
func (neo *DSL) handleChatLatest(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	locale := "en-us"
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	// Get the chats
	chats, err := neo.Store.GetChats(sid, store.ChatFilter{Page: 1}, locale)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// Create a new chat
	if len(chats.Groups) == 0 || len(chats.Groups[0].Chats) == 0 {

		assistantID := neo.Use.Default
		queryAssistantID := c.Query("assistant_id")
		if queryAssistantID != "" {
			assistantID = queryAssistantID
		}

		// Get the assistant info
		ast, err := assistant.Get(assistantID)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
			return
		}

		c.JSON(200, map[string]interface{}{"data": map[string]interface{}{
			"placeholder":          ast.GetPlaceholder(locale),
			"assistant_id":         ast.ID,
			"assistant_name":       ast.GetName(locale),
			"assistant_avatar":     ast.Avatar,
			"assistant_deleteable": neo.Use.Default != ast.ID,
		}})
		c.Done()
		return
	}

	// Get the chat_id
	chatID, ok := chats.Groups[0].Chats[0]["chat_id"].(string)
	if !ok {
		c.JSON(404, gin.H{"message": "chat_id not found", "code": 404})
		c.Done()
		return
	}

	chat, err := neo.Store.GetChat(sid, chatID, locale)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// assistant_id is nil return the default assistant
	if chat.Chat["assistant_id"] == nil {
		chat.Chat["assistant_id"] = neo.Use.Default

		// Get the assistant info
		ast, err := assistant.Get(neo.Use.Default)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
			return
		}
		chat.Chat["assistant_name"] = ast.GetName(locale)
		chat.Chat["assistant_avatar"] = ast.Avatar
	}

	chat.Chat["assistant_deleteable"] = neo.Use.Default != chat.Chat["assistant_id"]
	c.JSON(200, map[string]interface{}{"data": chat})
	c.Done()
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

	locale := "en-us"
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	// Get the chat details
	chat, err := neo.Store.GetChat(sid, chatID, locale)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// assistant_id is nil return the default assistant
	if chat.Chat["assistant_id"] == nil {
		chat.Chat["assistant_id"] = neo.Use.Default

		// Get the assistant info
		ast, err := assistant.Get(neo.Use.Default)
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error(), "code": 500})
			c.Done()
			return
		}
		chat.Chat["assistant_name"] = ast.GetName(locale)
		chat.Chat["assistant_avatar"] = ast.Avatar
	}

	chat.Chat["assistant_deleteable"] = neo.Use.Default != chat.Chat["assistant_id"]
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

	locale := "en-us"
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	// Get keywords from query parameter
	keywords := strings.ToLower(c.Query("keywords"))
	mentionable := true

	// Query mentionable assistants
	filter := store.AssistantFilter{
		Keywords:    keywords,
		Mentionable: &mentionable,
		Page:        1,
		PageSize:    20,
	}

	response, err := neo.Store.GetAssistants(filter, locale)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// Convert assistants to mentions
	mentions := []Mention{}
	for _, item := range response.Data {
		mention := Mention{
			ID:     item["assistant_id"].(string),
			Name:   item["name"].(string),
			Type:   item["type"].(string),
			Avatar: item["avatar"].(string),
		}
		mentions = append(mentions, mention)
	}

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

	if body.Title == "" {
		c.JSON(400, gin.H{"message": "title is required", "code": 400})
		c.Done()
		return
	}

	err := neo.Store.UpdateChatTitle(sid, chatID, body.Title)
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

	err := neo.Store.DeleteChat(sid, chatID)
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

	err := neo.Store.DeleteAllChats(sid)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, gin.H{"message": "ok"})
	c.Done()
}

// handleGenerateTitle handles generating a chat title
func (neo *DSL) handleGenerateTitle(c *gin.Context) {
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

	chatID := fmt.Sprintf("generate_title_%d", time.Now().UnixNano())

	// Set the context with validated chat_id
	ctx, cancel := chatctx.NewWithCancel(sid, chatID, c.Query("context"))
	defer cancel()
	defer ctx.Release() // Release the context after the request is done

	// Set the assistant ID
	ctx = chatctx.WithHistoryVisible(ctx, false)
	ctx = chatctx.WithAssistantID(ctx, neo.Use.Title)

	err := neo.Answer(ctx, content, c)

	// Error handling
	if err != nil {
		message.New().Done().Error(err).Write(c.Writer)
		c.Done()
		return
	}
}

// handleGeneratePrompts handles generating prompts
func (neo *DSL) handleGeneratePrompts(c *gin.Context) {
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

	chatID := fmt.Sprintf("generate_prompts_%d", time.Now().UnixNano())

	// Set the context with validated chat_id
	ctx, cancel := chatctx.NewWithCancel(sid, chatID, c.Query("context"))
	defer cancel()
	defer ctx.Release() // Release the context after the request is done

	// Set the assistant ID
	ctx = chatctx.WithHistoryVisible(ctx, false)
	ctx = chatctx.WithAssistantID(ctx, neo.Use.Prompt)
	err := neo.Answer(ctx, content, c)

	// Error handling
	if err != nil {
		message.New().Done().Error(err).Write(c.Writer)
		c.Done()
		return
	}
}

// handleAssistantList handles listing assistants
func (neo *DSL) handleAssistantList(c *gin.Context) {
	// Parse filter parameters
	filter := store.AssistantFilter{
		Type:     "assistant",
		Page:     1,
		PageSize: 20,
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

	// Parse tags
	if tags := c.Query("tags"); tags != "" {
		filter.Tags = strings.Split(tags, ",")
	}

	// Parse keywords
	if keywords := c.Query("keywords"); keywords != "" {
		filter.Keywords = keywords
	}

	// Parse connector
	if connector := c.Query("connector"); connector != "" {
		filter.Connector = connector
	}

	// Parse select fields
	if selectFields := c.Query("select"); selectFields != "" {
		filter.Select = strings.Split(selectFields, ",")
	}

	// Parse built_in (support various boolean formats)
	if builtIn := c.Query("built_in"); builtIn != "" {
		val := parseBoolValue(builtIn)
		if val != nil {
			filter.BuiltIn = val
		}
	}

	// Parse mentionable (support various boolean formats)
	if mentionable := c.Query("mentionable"); mentionable != "" {
		val := parseBoolValue(mentionable)
		if val != nil {
			filter.Mentionable = val
		}
	}

	// Parse automated (support various boolean formats)
	if automated := c.Query("automated"); automated != "" {
		val := parseBoolValue(automated)
		if val != nil {
			filter.Automated = val
		}
	}

	// Parse assistant_id
	if assistantID := c.Query("assistant_id"); assistantID != "" {
		filter.AssistantID = assistantID
	}

	locale := "en-us" // Default locale
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	response, err := neo.Store.GetAssistants(filter, locale)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, response)
	c.Done()
}

// parseBoolValue parses various string formats into a boolean pointer
// Supports: 1, 0, "1", "0", "true", "false", etc.
func parseBoolValue(value string) *bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "1", "true", "yes", "on":
		v := true
		return &v
	case "0", "false", "no", "off":
		v := false
		return &v
	default:
		return nil
	}
}

// handleAssistantAPI handles the assistant API
func (neo *DSL) handleAssistantCall(c *gin.Context) {
	assistantID := c.Param("id")
	if assistantID == "" {
		c.JSON(400, gin.H{"message": "assistant id is required", "code": 400})
		c.Done()
		return
	}
	ast, err := assistant.Get(assistantID)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	payload := assistant.APIPayload{Sid: sid}
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(400, gin.H{"message": "invalid request body", "code": 400})
		c.Done()
		return
	}

	result, err := ast.Call(c, payload)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, result)
	c.Done()
}

// handleAssistantDetail handles getting a single assistant's details
func (neo *DSL) handleAssistantDetail(c *gin.Context) {
	assistantID := c.Param("id")
	if assistantID == "" {
		c.JSON(400, gin.H{"message": "assistant id is required", "code": 400})
		c.Done()
		return
	}

	filter := store.AssistantFilter{
		AssistantID: assistantID,
		Type:        "assistant",
		Page:        1,
		PageSize:    1,
	}

	locale := "en-us" // Default locale
	// Translate the response
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	response, err := neo.Store.GetAssistants(filter, locale)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	if len(response.Data) == 0 {
		c.JSON(404, gin.H{"message": "assistant not found", "code": 404})
		c.Done()
		return
	}

	c.JSON(200, map[string]interface{}{"data": response.Data[0]})
	c.Done()
}

// handleAssistantSave handles creating or updating an assistant
func (neo *DSL) handleAssistantSave(c *gin.Context) {
	var assistantData map[string]interface{}
	if err := c.BindJSON(&assistantData); err != nil {
		c.JSON(400, gin.H{"message": "invalid request body", "code": 400})
		c.Done()
		return
	}

	id, err := neo.Store.SaveAssistant(assistantData)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// Update the assistant map with the returned ID if it's not already set
	if _, ok := assistantData["assistant_id"]; !ok {
		assistantData["assistant_id"] = id
	}

	// Remove the assistant from cache to ensure fresh data on next load
	cache := assistant.GetCache()
	if cache != nil {
		cache.Remove(id.(string))
	}

	// Reload the assistant to ensure it's available in cache with updated data
	_, err = assistant.Get(id.(string))
	if err != nil {
		// Just log the error, don't fail the request
		fmt.Printf("Error reloading assistant %s: %v\n", id, err)
	}

	c.JSON(200, gin.H{"message": "ok", "data": assistantData})
	c.Done()
}

// handleAssistantDelete handles deleting an assistant
func (neo *DSL) handleAssistantDelete(c *gin.Context) {
	assistantID := c.Param("id")
	if assistantID == "" {
		c.JSON(400, gin.H{"message": "assistant id is required", "code": 400})
		c.Done()
		return
	}

	err := neo.Store.DeleteAssistant(assistantID)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	// Remove the assistant from cache to ensure it's fully deleted
	cache := assistant.GetCache()
	if cache != nil {
		cache.Remove(assistantID)
	}

	c.JSON(200, gin.H{"message": "ok"})
	c.Done()
}

// handleConnectors handles listing connectors
func (neo *DSL) handleConnectors(c *gin.Context) {
	options := []map[string]interface{}{}

	// Filter and format connectors
	for id, conn := range connector.Connectors {
		if conn.Is(connector.OPENAI) || conn.Is(connector.MOAPI) {
			setting := conn.Setting()
			label := setting["label"]
			if label == nil || label == "" {
				label = setting["name"]
			}
			if label == nil || label == "" {
				label = id
			}
			options = append(options, map[string]interface{}{
				"label": label,
				"value": id,
			})
		}
	}

	c.JSON(200, gin.H{"data": options})
	c.Done()
}

// handleAssistantTags handles getting all assistant tags
func (neo *DSL) handleAssistantTags(c *gin.Context) {
	sid := c.GetString("__sid")
	if sid == "" {
		c.JSON(400, gin.H{"message": "sid is required", "code": 400})
		c.Done()
		return
	}

	locale := "en-us" // Default locale
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	tags, err := neo.Store.GetAssistantTags(locale)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error(), "code": 500})
		c.Done()
		return
	}

	c.JSON(200, gin.H{"data": tags})
	c.Done()
}
