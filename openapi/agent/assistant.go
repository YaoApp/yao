package agent

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
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
	response.RespondWithSuccess(c, response.StatusOK, map[string]interface{}{
		"data": tags,
	})
}
