package chat

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// =============================================================================
// Search Reference Handlers
// =============================================================================

// GetReferences retrieves all search references for a request
// GET /v1/chat/references/:request_id
func GetReferences(c *gin.Context) {
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

	// Get request ID from URL parameter
	requestID := c.Param("request_id")
	if requestID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Request ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get all search records for this request
	searches, err := chatStore.GetSearches(requestID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// If no searches found, return empty result
	if len(searches) == 0 {
		response.RespondWithSuccess(c, response.StatusOK, gin.H{
			"request_id": requestID,
			"references": []storetypes.Reference{},
			"total":      0,
		})
		return
	}

	// Get authorized information and check permission using chat_id from first search
	authInfo := authorized.GetInfo(c)
	chatID := searches[0].ChatID
	if chatID != "" {
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
				ErrorDescription: "Forbidden: No permission to access these references",
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
			return
		}
	}

	// Collect all references from all searches
	var allRefs []storetypes.Reference
	for _, search := range searches {
		allRefs = append(allRefs, search.References...)
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"request_id": requestID,
		"references": allRefs,
		"total":      len(allRefs),
	})
}

// GetReference retrieves a single reference by request ID and index
// GET /v1/chat/references/:request_id/:index
func GetReference(c *gin.Context) {
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

	// Get request ID from URL parameter
	requestID := c.Param("request_id")
	if requestID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Request ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get index from URL parameter
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 1 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid reference index, must be a positive integer",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get all search records to check permission first
	searches, err := chatStore.GetSearches(requestID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Check permission using chat_id from first search
	if len(searches) > 0 {
		authInfo := authorized.GetInfo(c)
		chatID := searches[0].ChatID
		if chatID != "" {
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
					ErrorDescription: "Forbidden: No permission to access this reference",
				}
				response.RespondWithError(c, response.StatusForbidden, errorResp)
				return
			}
		}
	}

	// Get the specific reference
	ref, err := chatStore.GetReference(requestID, index)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, ref)
}
