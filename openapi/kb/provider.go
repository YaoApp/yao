package kb

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/providers/factory"
	"github.com/yaoapp/yao/openapi/response"
)

// GetProviders get all providers
func GetProviders(c *gin.Context) {
	providerType := c.Param("providerType")
	locale := strings.ToLower(c.Query("locale"))
	if locale == "" {
		locale = "en"
	}

	if providerType == "" {
		// Create a custom error with the same structure but specific message
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: providerType is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Filter providers by ids
	ids := strings.Split(c.Query("ids"), ",")
	providers, err := kb.GetProviders(providerType, ids, locale)
	if err != nil {
		// Create a custom error with the same structure but specific message
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get providers: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// response with success
	response.RespondWithSuccess(c, response.StatusOK, providers)
}

// GetProviderSchema get provider schema
func GetProviderSchema(c *gin.Context) {
	providerType := c.Param("providerType")
	providerID := c.Param("providerID")
	locale := strings.ToLower(c.Query("locale"))
	if locale == "" {
		locale = "en"
	}

	if providerType == "" || providerID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: providerType and providerID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	provider, err := kb.GetProviderWithLanguage(providerType, providerID, locale)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get provider: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	schema, err := factory.GetSchema(factory.ProviderType(providerType), provider, locale)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get provider schema: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, schema)
}
