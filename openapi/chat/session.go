package chat

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/assistant"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// =============================================================================
// Chat Session Handlers
// =============================================================================

// ListChats lists chat sessions with pagination and filtering
// GET /v1/chat/sessions
func ListChats(c *gin.Context) {
	// Get chat store
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Chat storage not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Build filter from query parameters
	filter := buildChatFilter(c, authInfo)

	// Call store to list chats
	result, err := chatStore.ListChats(filter)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Build response based on grouping mode
	// When group_by is set, data should be nil to avoid duplication
	resp := gin.H{
		"page":      result.Page,
		"pagesize":  result.PageSize,
		"pagecount": result.PageCount,
		"total":     result.Total,
	}

	if len(result.Groups) > 0 {
		// Grouped response: only include groups
		resp["groups"] = result.Groups
	} else {
		// Flat response: only include data
		resp["data"] = result.Data
	}

	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// GetChat retrieves a single chat session by ID
// GET /v1/chat/sessions/:chat_id
func GetChat(c *gin.Context) {
	// Get chat store
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Chat storage not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get chat ID from URL parameter
	chatID := c.Param("chat_id")
	if chatID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Chat ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Check permission
	hasPermission, err := checkChatPermission(chatStore, authInfo, chatID, true)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this chat",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Get chat
	chat, err := chatStore.GetChat(chatID)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Chat not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, chat)
}

// UpdateChat updates a chat session
// PUT /v1/chat/sessions/:chat_id
func UpdateChat(c *gin.Context) {
	// Get chat store
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Chat storage not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get chat ID from URL parameter
	chatID := c.Param("chat_id")
	if chatID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Chat ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse request body
	var req UpdateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Check permission (write access)
	hasPermission, err := checkChatPermission(chatStore, authInfo, chatID, false)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to update this chat",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	// Add update scope
	if authInfo != nil {
		updates["__yao_updated_by"] = authInfo.UserID
	}

	if len(updates) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No fields to update",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Update chat
	if err := chatStore.UpdateChat(chatID, updates); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"message": "Chat updated successfully",
		"chat_id": chatID,
	})
}

// DeleteChat deletes a chat session
// DELETE /v1/chat/sessions/:chat_id
func DeleteChat(c *gin.Context) {
	// Get chat store
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Chat storage not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get chat ID from URL parameter
	chatID := c.Param("chat_id")
	if chatID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Chat ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Check permission (write access)
	hasPermission, err := checkChatPermission(chatStore, authInfo, chatID, false)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to delete this chat",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Delete chat
	if err := chatStore.DeleteChat(chatID); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"message": "Chat deleted successfully",
		"chat_id": chatID,
	})
}

// =============================================================================
// Message Handlers
// =============================================================================

// GetMessages retrieves messages for a chat session
// GET /v1/chat/sessions/:chat_id/messages
func GetMessages(c *gin.Context) {
	// Get chat store
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Chat storage not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get chat ID from URL parameter
	chatID := c.Param("chat_id")
	if chatID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Chat ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Check permission (read access)
	hasPermission, err := checkChatPermission(chatStore, authInfo, chatID, true)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this chat",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Build message filter
	filter := buildMessageFilter(c)

	// Get messages
	messages, err := chatStore.GetMessages(chatID, filter)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get locale from query parameter or Accept-Language header
	locale := getLocale(c)

	// Collect unique assistant IDs from messages and fetch their info
	assistantIDs := collectAssistantIDs(messages)
	assistants := assistant.GetInfoByIDs(assistantIDs, locale)

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"chat_id":    chatID,
		"messages":   messages,
		"count":      len(messages),
		"assistants": assistants,
	})
}

// getLocale extracts locale from request
// Priority: 1. Query param "locale", 2. Accept-Language header
func getLocale(c *gin.Context) string {
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

// collectAssistantIDs extracts unique assistant IDs from messages
func collectAssistantIDs(messages []*storetypes.Message) []string {
	seen := make(map[string]bool)
	var ids []string

	for _, msg := range messages {
		if msg.AssistantID != "" && !seen[msg.AssistantID] {
			seen[msg.AssistantID] = true
			ids = append(ids, msg.AssistantID)
		}
	}

	return ids
}

// =============================================================================
// Helper Functions
// =============================================================================

// buildChatFilter builds ChatFilter from query parameters
func buildChatFilter(c *gin.Context, authInfo *oauthtypes.AuthorizedInfo) storetypes.ChatFilter {
	filter := storetypes.ChatFilter{}

	// Pagination
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if filter.Page == 0 {
		filter.Page = 1
	}

	if pagesizeStr := c.Query("pagesize"); pagesizeStr != "" {
		if ps, err := strconv.Atoi(pagesizeStr); err == nil && ps > 0 && ps <= 100 {
			filter.PageSize = ps
		}
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}

	// Business filters
	filter.AssistantID = strings.TrimSpace(c.Query("assistant_id"))
	filter.Status = strings.TrimSpace(c.Query("status"))
	filter.Keywords = strings.TrimSpace(c.Query("keywords"))

	// Time range filter
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &t
		}
	}
	filter.TimeField = strings.TrimSpace(c.Query("time_field"))
	if filter.TimeField == "" {
		filter.TimeField = "last_message_at"
	}

	// Sorting
	filter.OrderBy = strings.TrimSpace(c.Query("order_by"))
	if filter.OrderBy == "" {
		filter.OrderBy = "last_message_at"
	}
	filter.Order = strings.TrimSpace(c.Query("order"))
	if filter.Order == "" {
		filter.Order = "desc"
	}

	// Grouping
	filter.GroupBy = strings.TrimSpace(c.Query("group_by"))

	// Permission filters based on auth constraints
	if authInfo != nil {
		// Direct permission filters (AND logic)
		if authInfo.Constraints.OwnerOnly {
			filter.UserID = authInfo.UserID
		}
		if authInfo.Constraints.TeamOnly {
			filter.TeamID = authInfo.TeamID
		}

		// For complex permission logic (OR conditions), use QueryFilter
		// Example: user can see their own chats OR team shared chats
		if authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
			// Team member can see: own chats OR team shared chats
			filter.QueryFilter = func(qb query.Query) {
				qb.Where(func(sub query.Query) {
					sub.Where("__yao_created_by", authInfo.UserID).
						OrWhere(func(inner query.Query) {
							inner.Where("__yao_team_id", authInfo.TeamID).
								Where("share", "team")
						})
				})
			}
			// Clear direct filters since we're using QueryFilter
			filter.UserID = ""
			filter.TeamID = ""
		}
	}

	return filter
}

// buildMessageFilter builds MessageFilter from query parameters
func buildMessageFilter(c *gin.Context) storetypes.MessageFilter {
	filter := storetypes.MessageFilter{}

	// Filter parameters
	filter.RequestID = strings.TrimSpace(c.Query("request_id"))
	filter.Role = strings.TrimSpace(c.Query("role"))
	filter.BlockID = strings.TrimSpace(c.Query("block_id"))
	filter.ThreadID = strings.TrimSpace(c.Query("thread_id"))
	filter.Type = strings.TrimSpace(c.Query("type"))

	// Pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			filter.Limit = l
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	return filter
}

// checkChatPermission checks if the user has permission to access the chat
// readable: true for read access, false for write access
func checkChatPermission(chatStore storetypes.ChatStore, authInfo *oauthtypes.AuthorizedInfo, chatID string, readable bool) (bool, error) {
	// No auth info means no constraints (for internal calls)
	if authInfo == nil {
		return true, nil
	}

	// No constraints means full access
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true, nil
	}

	// Get chat to check permissions
	chat, err := chatStore.GetChat(chatID)
	if err != nil {
		return false, err
	}

	// For read access, check if chat is public or shared with team
	if readable {
		if chat.Public {
			return true, nil
		}
		if chat.Share == "team" && authInfo.Constraints.TeamOnly && chat.TeamID == authInfo.TeamID {
			return true, nil
		}
	}

	// Combined Team and Owner permission validation
	if authInfo.Constraints.TeamOnly && authInfo.Constraints.OwnerOnly {
		if chat.CreatedBy == authInfo.UserID && chat.TeamID == authInfo.TeamID {
			return true, nil
		}
		return false, nil
	}

	// Owner only permission validation
	if authInfo.Constraints.OwnerOnly && chat.CreatedBy == authInfo.UserID {
		return true, nil
	}

	// Team only permission validation
	if authInfo.Constraints.TeamOnly && chat.TeamID == authInfo.TeamID {
		return true, nil
	}

	return false, nil
}
