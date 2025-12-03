package agent

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	agenttypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// ListAssistants lists assistants with pagination and filtering
func ListAssistants(c *gin.Context) {

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get Agent instance from global variable
	agentInstance := agent.GetAgent()
	if agentInstance == nil || agentInstance.Store == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Agent store not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pagesize := 20
	if pagesizeStr := c.Query("pagesize"); pagesizeStr != "" {
		if ps, err := strconv.Atoi(pagesizeStr); err == nil && ps > 0 && ps <= 100 {
			pagesize = ps
		}
	}

	// Validate pagination
	if err := ValidatePagination(page, pagesize); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse select parameter
	var selectFields []string
	if selectParam := strings.TrimSpace(c.Query("select")); selectParam != "" {
		requestedFields := strings.Split(selectParam, ",")
		for _, field := range requestedFields {
			field = strings.TrimSpace(field)
			if field != "" && availableAssistantFields[field] {
				selectFields = append(selectFields, field)
			}
		}
		// If no valid fields found, use default
		if len(selectFields) == 0 {
			selectFields = defaultAssistantFields
		}
	} else {
		selectFields = defaultAssistantFields
	}

	// Parse filter parameters
	keywords := strings.TrimSpace(c.Query("keywords"))
	typeParam := strings.TrimSpace(c.Query("type"))
	if typeParam == "" {
		typeParam = "assistant" // Default type
	}
	connector := strings.TrimSpace(c.Query("connector"))
	assistantID := strings.TrimSpace(c.Query("assistant_id"))

	// Parse assistant IDs (multiple)
	var assistantIDs []string
	if assistantIDsParam := c.Query("assistant_ids"); assistantIDsParam != "" {
		assistantIDs = strings.Split(assistantIDsParam, ",")
		// Trim spaces
		for i, id := range assistantIDs {
			assistantIDs[i] = strings.TrimSpace(id)
		}
	}

	// Parse tags
	var tags []string
	if tagsParam := c.Query("tags"); tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
		// Trim spaces
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	// Parse boolean filters
	var builtIn, mentionable, automated *bool
	if builtInParam := c.Query("built_in"); builtInParam != "" {
		builtIn = parseBoolValue(builtInParam)
	}
	if mentionableParam := c.Query("mentionable"); mentionableParam != "" {
		mentionable = parseBoolValue(mentionableParam)
	}
	if automatedParam := c.Query("automated"); automatedParam != "" {
		automated = parseBoolValue(automatedParam)
	}

	// Note: public and share filters are not yet supported in AssistantFilter
	// They would need to be added to the store layer for proper filtering

	// Parse locale
	locale := "en-us" // Default locale
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	// Build filter using the existing AssistantFilter structure
	filter := BuildAssistantFilter(AssistantFilterParams{
		Page:         page,
		PageSize:     pagesize,
		Keywords:     keywords,
		Type:         typeParam,
		Connector:    connector,
		AssistantID:  assistantID,
		AssistantIDs: assistantIDs,
		Tags:         tags,
		SelectFields: selectFields,
		BuiltIn:      builtIn,
		Mentionable:  mentionable,
		Automated:    automated,
	})

	// Apply permission-based filtering (Scope filtering)
	filter.QueryFilter = AuthQueryFilter(c, authInfo)

	// Use the existing GetAssistants method from agent.Store
	result, err := agentInstance.Store.GetAssistants(filter, locale)
	if err != nil {
		log.Error("Failed to list assistants: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to list assistants: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Filter sensitive fields for built-in assistants
	// For built-in assistants, clear code-level fields (prompts, workflow, tools, kb, mcp, options)
	FilterBuiltInFields(result.Data)

	// Return the result with standard response format
	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetAssistant retrieves a single assistant by ID with permission verification
func GetAssistant(c *gin.Context) {

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get Agent instance from global variable
	agentInstance := agent.GetAgent()
	if agentInstance == nil || agentInstance.Store == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Agent store not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get assistant ID from URL parameter
	assistantID := c.Param("id")
	if assistantID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "assistant_id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse select fields (optional - if not provided, returns default fields)
	// Query parameter: ?select=field1,field2,field3
	var fields []string
	if selectParam := c.Query("select"); selectParam != "" {
		fields = strings.Split(selectParam, ",")
		// Trim whitespace from each field
		for i, field := range fields {
			fields[i] = strings.TrimSpace(field)
		}
	}

	// Parse locale (optional - if not provided, returns raw data without i18n translation)
	// This is useful for form editing scenarios where you need the original values
	var assistant *agenttypes.AssistantModel
	var err error

	if loc := c.Query("locale"); loc != "" {
		// If locale is specified, get assistant with translation
		locale := strings.ToLower(strings.TrimSpace(loc))
		assistant, err = agentInstance.Store.GetAssistant(assistantID, fields, locale)
	} else {
		// If no locale specified, get raw data without translation
		assistant, err = agentInstance.Store.GetAssistant(assistantID, fields)
	}
	if err != nil {
		log.Error("Failed to get assistant %s: %v", assistantID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Assistant not found: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Check read permission
	hasPermission, err := checkAssistantPermission(authInfo, assistantID, true)
	if err != nil {
		log.Error("Failed to check permission for assistant %s: %v", assistantID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to check permission: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this assistant",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Filter sensitive fields for built-in assistants
	FilterBuiltInAssistant(assistant)

	// Return the result with standard response format
	response.RespondWithSuccess(c, response.StatusOK, assistant)
}

// ListAssistantTags lists assistant tags with permission-based filtering
func ListAssistantTags(c *gin.Context) {

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get Agent instance from global variable
	agentInstance := agent.GetAgent()
	if agentInstance == nil || agentInstance.Store == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Agent store not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Parse locale
	locale := "en-us" // Default locale
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	// Parse filter parameters
	typeParam := strings.TrimSpace(c.Query("type"))
	if typeParam == "" {
		typeParam = "assistant" // Default type
	}
	connector := strings.TrimSpace(c.Query("connector"))
	keywords := strings.TrimSpace(c.Query("keywords"))

	// Parse boolean filters
	var builtIn, mentionable, automated *bool
	if builtInParam := c.Query("built_in"); builtInParam != "" {
		builtIn = parseBoolValue(builtInParam)
	}
	if mentionableParam := c.Query("mentionable"); mentionableParam != "" {
		mentionable = parseBoolValue(mentionableParam)
	}
	if automatedParam := c.Query("automated"); automatedParam != "" {
		automated = parseBoolValue(automatedParam)
	}

	// Build filter
	filter := BuildAssistantFilter(AssistantFilterParams{
		Type:        typeParam,
		Connector:   connector,
		Keywords:    keywords,
		BuiltIn:     builtIn,
		Mentionable: mentionable,
		Automated:   automated,
	})

	// Apply permission-based filtering (Scope filtering)
	filter.QueryFilter = AuthQueryFilter(c, authInfo)

	// Get tags with filter
	tags, err := agentInstance.Store.GetAssistantTags(filter, locale)
	if err != nil {
		log.Error("Failed to get assistant tags: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get assistant tags: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return the result with standard response format
	response.RespondWithSuccess(c, response.StatusOK, tags)
}

// CreateAssistant creates a new assistant
func CreateAssistant(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get Agent instance from global variable
	agentInstance := agent.GetAgent()
	if agentInstance == nil || agentInstance.Store == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Agent store not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Parse request body
	var assistantData map[string]interface{}
	if err := c.ShouldBindJSON(&assistantData); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Convert to AssistantModel
	model, err := agenttypes.ToAssistantModel(assistantData)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid assistant data: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Attach create scope to the assistant data
	if authInfo != nil {
		scope := authInfo.AccessScope()
		model.YaoCreatedBy = scope.CreatedBy
		model.YaoUpdatedBy = scope.UpdatedBy
		model.YaoTeamID = scope.TeamID
		model.YaoTenantID = scope.TenantID
	}

	// Save assistant using Store
	id, err := agentInstance.Store.SaveAssistant(model)
	if err != nil {
		log.Error("Failed to create assistant: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create assistant: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Update the assistant map with the returned ID
	assistantData["assistant_id"] = id

	// Clear cache and reload assistant to make it effective
	cache := assistant.GetCache()
	if cache != nil {
		cache.Remove(id)
	}

	// Reload the assistant to ensure it's available in cache with updated data
	_, err = assistant.Get(id)
	if err != nil {
		// Just log the error, don't fail the request
		log.Error("Error reloading assistant %s: %v", id, err)
	}

	// Return success response with only assistant_id
	response.RespondWithSuccess(c, response.StatusOK, map[string]interface{}{
		"assistant_id": id,
	})
}

// UpdateAssistant updates an existing assistant
func UpdateAssistant(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get Agent instance from global variable
	agentInstance := agent.GetAgent()
	if agentInstance == nil || agentInstance.Store == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Agent store not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get assistant ID from URL parameter
	assistantID := c.Param("id")
	if assistantID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "assistant_id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Check update permission
	hasPermission, err := checkAssistantPermission(authInfo, assistantID, false)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// 403 Forbidden
	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to update this assistant",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Parse request body with update data
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Add update metadata
	if authInfo != nil {
		scope := authInfo.AccessScope()
		updateData["__yao_updated_by"] = scope.UpdatedBy
	}

	// Update assistant using Store
	err = agentInstance.Store.UpdateAssistant(assistantID, updateData)
	if err != nil {
		log.Error("Failed to update assistant %s: %v", assistantID, err)
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Assistant not found: " + assistantID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to update assistant: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Clear cache and reload assistant to make it effective
	cache := assistant.GetCache()
	if cache != nil {
		cache.Remove(assistantID)
	}

	// Reload the assistant to ensure it's available in cache with updated data
	_, err = assistant.Get(assistantID)
	if err != nil {
		// Just log the error, don't fail the request
		log.Error("Error reloading assistant %s: %v", assistantID, err)
	}

	// Return success response with only assistant_id
	response.RespondWithSuccess(c, response.StatusOK, map[string]interface{}{
		"assistant_id": assistantID,
	})
}

// GetAssistantInfo retrieves essential assistant information for InputArea component
// Returns only the fields needed for UI display: id, name, avatar, description, connector, connector_options, modes, default_mode
func GetAssistantInfo(c *gin.Context) {

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get Agent instance from global variable
	agentInstance := agent.GetAgent()
	if agentInstance == nil || agentInstance.Store == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Agent store not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get assistant ID from URL parameter
	assistantID := c.Param("id")
	if assistantID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "assistant_id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse locale (optional - defaults to "en-us")
	locale := "en-us"
	if loc := c.Query("locale"); loc != "" {
		locale = strings.ToLower(strings.TrimSpace(loc))
	}

	// Define fields needed for InputArea
	infoFields := []string{
		"assistant_id",
		"name",
		"avatar",
		"description",
		"connector",
		"connector_options",
		"modes",
		"default_mode",
	}

	// Get assistant with specific fields and locale
	assistant, err := agentInstance.Store.GetAssistant(assistantID, infoFields, locale)
	if err != nil {
		log.Error("Failed to get assistant info %s: %v", assistantID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Assistant not found: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Check read permission (same as GetAssistant)
	hasPermission, err := checkAssistantPermission(authInfo, assistantID, true)
	if err != nil {
		log.Error("Failed to check permission for assistant %s: %v", assistantID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to check permission: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this assistant",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Build response with only the required fields
	infoResponse := map[string]interface{}{
		"assistant_id": assistant.ID,
		"name":         assistant.Name,
		"avatar":       assistant.Avatar,
		"description":  assistant.Description,
		"connector":    assistant.Connector,
	}

	// Add optional fields if they exist
	if assistant.ConnectorOptions != nil {
		infoResponse["connector_options"] = assistant.ConnectorOptions
	}
	if len(assistant.Modes) > 0 {
		infoResponse["modes"] = assistant.Modes
	}
	if assistant.DefaultMode != "" {
		infoResponse["default_mode"] = assistant.DefaultMode
	}

	// Return the result with standard response format
	response.RespondWithSuccess(c, response.StatusOK, infoResponse)
}

// checkAssistantPermission checks if the user has permission to access the assistant
// Similar logic to checkCollectionPermission in openapi/kb/collection.go
// readable: true for read permission, false for write permission
func checkAssistantPermission(authInfo *types.AuthorizedInfo, assistantID string, readable ...bool) (bool, error) {
	// No auth info, allow access
	if authInfo == nil {
		return true, nil
	}

	// No constraints, allow access
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true, nil
	}

	// Get Agent instance
	agentInstance := agent.GetAgent()
	if agentInstance == nil || agentInstance.Store == nil {
		return false, fmt.Errorf("agent store not initialized")
	}

	// Get assistant from store - only need default fields for permission check
	assistant, err := agentInstance.Store.GetAssistant(assistantID, nil)
	if err != nil {
		return false, fmt.Errorf("assistant not found: %s", assistantID)
	}

	// If readable mode, check if the assistant is accessible for reading
	if len(readable) > 0 && readable[0] {
		// If assistant is public, allow read access
		if assistant.Public {
			return true, nil
		}

		// Team only permission validation for read
		if assistant.Share == "team" && authInfo.Constraints.TeamOnly {
			return true, nil
		}
	}

	// Check if user is the creator - always allow creator to access their own assistant
	if assistant.YaoCreatedBy != "" && assistant.YaoCreatedBy == authInfo.UserID {
		return true, nil
	}

	// Combined Team and Owner permission validation
	if authInfo.Constraints.TeamOnly && authInfo.Constraints.OwnerOnly {
		if assistant.YaoTeamID != "" && assistant.YaoTeamID == authInfo.TeamID {
			return true, nil
		}
		return false, nil
	}

	// Team only permission validation
	if authInfo.Constraints.TeamOnly && assistant.YaoTeamID != "" && assistant.YaoTeamID == authInfo.TeamID {
		return true, nil
	}

	// Owner only permission validation (already handled above by creator check)
	if authInfo.Constraints.OwnerOnly {
		return false, nil
	}

	return false, fmt.Errorf("no permission to access assistant: %s", assistantID)
}
