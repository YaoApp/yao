package openapi

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
)

// For compatibility with all platform clients, providing standard OpenAPI model interfaces

// GinGetModels handles GET /chat/models - Get all chat models
func GinGetModels(c *gin.Context) {
	mockResponse := ModelsResponse{
		Object: "list",
		Data: []Model{
			{
				ID:      "gpt-4o-1024",
				Object:  "model",
				Created: 1686935002,
				OwnedBy: "organization-owner",
			},
			{
				ID:      "model-id-1",
				Object:  "model",
				Created: 1686935002,
				OwnedBy: "organization-owner",
			},
			{
				ID:      "model-id-2",
				Object:  "model",
				Created: 1686935002,
				OwnedBy: "openai",
			},
		},
	}

	response.RespondWithSuccess(c, response.StatusOK, mockResponse)
}

// GinGetModelDetails handles GET /chat/models/:model_name - Get model details
func GinGetModelDetails(c *gin.Context) {
	response.RespondWithSuccess(c, response.StatusOK, gin.H{"message": "placeholder"})
}
