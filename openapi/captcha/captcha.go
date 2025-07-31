package captcha

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach attaches the hello world handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Health check
	group.GET("/image", image)

	// OAuth Protected Resource
	group.GET("/audio", audio)
}

// image captcha
func image(c *gin.Context) {
	var option helper.CaptchaOption = helper.NewCaptchaOption()

	err := c.ShouldBindQuery(&option)
	if err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	// Set the type to image
	option.Type = "image"
	id, content := helper.CaptchaMake(option)
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"id": id, "data": content})
}

// audio captcha
func audio(c *gin.Context) {
	var option helper.CaptchaOption = helper.NewCaptchaOption()

	err := c.ShouldBindQuery(&option)
	if err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
	}

	// Set the type to audio
	option.Type = "audio"
	id, content := helper.CaptchaMake(option)
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"id": id, "data": content})
}
