package signin

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach attaches the signin handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	group.GET("/signin", getConfig)
	group.POST("/signin", signin)
	group.GET("/signin/authback/:id", authback)
}

// getConfig is the handler for get signin configuration
func getConfig(c *gin.Context) {
	// Get locale from query parameter (optional)
	locale := c.Query("locale")

	// Get public configuration for the specified locale
	config := GetPublicConfig(locale)

	// If no configuration found, return error
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No signin configuration found for the requested locale",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Return the public configuration
	response.RespondWithSuccess(c, response.StatusOK, config)
}

// signin is the handler for signin (password login)
func signin(c *gin.Context) {}

// authback is the handler for authback
func authback(c *gin.Context) {}
