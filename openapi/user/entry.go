package user

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// getEntryConfig is the handler for get unified auth entry configuration
func getEntryConfig(c *gin.Context) {
	// Get locale from query parameter (optional)
	locale := c.Query("locale")

	// Get entry configuration for the specified locale
	config := GetEntryConfig(locale)

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
			ErrorDescription: "No entry configuration found for the requested locale",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Create public config without sensitive data
	publicConfig := *config
	publicConfig.ClientSecret = "" // Remove sensitive data

	// Remove captcha secret from public config
	if publicConfig.Form != nil && publicConfig.Form.Captcha != nil && publicConfig.Form.Captcha.Options != nil {
		// Create a copy of captcha options without the secret
		captchaOptions := make(map[string]interface{})
		for k, v := range publicConfig.Form.Captcha.Options {
			if k != "secret" {
				captchaOptions[k] = v
			}
		}
		publicConfig.Form.Captcha.Options = captchaOptions
	}

	// Return the entry configuration
	response.RespondWithSuccess(c, response.StatusOK, publicConfig)
}

// entry is the handler for unified auth entry (login/register)
// The backend determines whether this is a login or registration based on email existence
func entry(c *gin.Context) {
	// This is a placeholder - you may need to implement the actual login/register logic here
	// The logic should:
	// 1. Check if the email exists in the database
	// 2. If exists: proceed with login flow
	// 3. If not exists: proceed with registration flow
}
