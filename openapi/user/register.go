package user

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// getRegisterConfig is the handler for get register configuration
func getRegisterConfig(c *gin.Context) {
	// Get locale from query parameter (optional)
	locale := c.Query("locale")

	// Get register configuration for the specified locale (already includes third_party from signin config)
	config := GetRegisterConfig(locale)

	// Set session id if not exists
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
		response.SendSessionCookie(c, sid)
	}

	// If no configuration found, return error
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No register configuration found for the requested locale",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Return the register configuration
	response.RespondWithSuccess(c, response.StatusOK, config)
}

// register is the handler for user registration
func register(c *gin.Context) {
	// This is a placeholder - you may need to implement the actual registration logic here
}
