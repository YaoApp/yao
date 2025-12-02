package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/context"
	agenttypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// Model represents an OpenAI-compatible model object
type Model struct {
	ID      string `json:"id"`       // Model identifier (format: yao-agents-assistantName-model-yao_assistantID)
	Object  string `json:"object"`   // Always "model"
	Created int64  `json:"created"`  // Unix timestamp when the model was created
	OwnedBy string `json:"owned_by"` // Organization that owns the model
}

// ModelsListResponse represents the response for listing models (OpenAI compatible)
type ModelsListResponse struct {
	Object string  `json:"object"` // Always "list"
	Data   []Model `json:"data"`   // Array of model objects
}

// GetModels handles GET /models - List all available models
// Compatible with OpenAI API: https://platform.openai.com/docs/api-reference/models/list
func GetModels(c *gin.Context) {

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

	// Parse locale (optional - for assistant name translation)
	// Priority: 1. Query parameter "locale", 2. Header "Accept-Language", 3. Metadata
	locale := context.GetLocale(c, nil)

	// Build filter with permission-based filtering
	filter := agenttypes.AssistantFilter{
		Page:     1,
		PageSize: 1000, // Get all assistants
	}

	// Apply permission-based filtering (Scope filtering)
	filter.QueryFilter = AuthQueryFilter(c, authInfo)

	assistantsResponse, err := agentInstance.Store.GetAssistants(filter, locale)
	if err != nil {
		log.Error("Failed to get assistants: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve assistants: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert assistants to models
	models := make([]Model, 0, len(assistantsResponse.Data))
	for _, assistant := range assistantsResponse.Data {
		// Generate model ID: yao-agents-assistantName-model-yao_assistantID
		modelID := assistant.ModelID("yao-agents-")

		// Create model object
		model := Model{
			ID:      modelID,
			Object:  "model",
			Created: assistant.CreatedAt,
			OwnedBy: getOwner(*assistant),
		}

		models = append(models, model)
	}

	// Return OpenAI-compatible response
	response.RespondWithSuccess(c, response.StatusOK, ModelsListResponse{
		Object: "list",
		Data:   models,
	})
}

// GetModelDetails handles GET /models/:model_id - Retrieve a single model
// Compatible with OpenAI API: https://platform.openai.com/docs/api-reference/models/retrieve
func GetModelDetails(c *gin.Context) {

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get model ID from URL parameter
	modelID := c.Param("model_name")
	if modelID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "model_id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Extract assistant ID from model ID
	assistantID := agenttypes.ParseModelID(modelID)
	if assistantID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid model ID format",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

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

	// For model API, we only need minimal fields: assistant_id, name, connector, created_at, and permission fields
	modelFields := []string{
		"assistant_id",
		"name",
		"connector",
		"created_at",
		"built_in",
		"__yao_team_id",
		"__yao_created_by",
	}

	// Parse locale (optional - for assistant name translation)
	// Priority: 1. Query parameter "locale", 2. Header "Accept-Language", 3. Metadata
	locale := context.GetLocale(c, nil)

	var assistant *agenttypes.AssistantModel
	var err error

	if locale != "" {
		assistant, err = agentInstance.Store.GetAssistant(assistantID, modelFields, locale)
	} else {
		assistant, err = agentInstance.Store.GetAssistant(assistantID, modelFields)
	}

	if err != nil {
		log.Error("Failed to get assistant %s: %v", assistantID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Model not found: " + err.Error(),
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
			ErrorDescription: "Forbidden: No permission to access this model",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Generate model ID
	modelIDGenerated := assistant.ModelID("yao-agents-")

	// Return OpenAI-compatible model object
	model := Model{
		ID:      modelIDGenerated,
		Object:  "model",
		Created: assistant.CreatedAt,
		OwnedBy: getOwner(*assistant),
	}

	response.RespondWithSuccess(c, response.StatusOK, model)
}

// getOwner returns the owner of the assistant/model
func getOwner(assistant agenttypes.AssistantModel) string {
	// For built-in assistants
	if assistant.BuiltIn {
		return "system"
	}

	// If has team ID, return team
	if assistant.YaoTeamID != "" {
		return "team"
	}

	// If has creator ID, return user
	if assistant.YaoCreatedBy != "" {
		return "user"
	}

	// Default to system
	return "system"
}
