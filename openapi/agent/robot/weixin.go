package robot

import (
	"os"

	"github.com/gin-gonic/gin"
	api "github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/openapi/response"
)

type createWeixinQRCodeRequest struct {
	APIHost string `json:"api_host"`
}

// CreateWeixinQRCode handles POST /robots/integrations/weixin/qrcode
func CreateWeixinQRCode(c *gin.Context) {
	var req createWeixinQRCodeRequest
	_ = c.ShouldBindJSON(&req)

	apiHost := req.APIHost
	if apiHost == "" {
		apiHost = os.Getenv("YAO_WEIXIN_API_HOST")
	}
	if apiHost == "" {
		apiHost = "https://ilinkai.weixin.qq.com"
	}

	sessionKey, qrcodeURL, qrcodeImg, err := api.WeixinQRCodeCreate(apiHost)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"session_key": sessionKey,
		"qrcode_url":  qrcodeURL,
		"qrcode_img":  qrcodeImg,
	})
}

// PollWeixinQRCode handles GET /robots/integrations/weixin/qrcode/:session_key
func PollWeixinQRCode(c *gin.Context) {
	sessionKey := c.Param("session_key")
	if sessionKey == "" {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "session_key is required",
		})
		return
	}

	status, botToken, accountID, baseURL, _, err := api.WeixinQRCodePoll(sessionKey)
	if err != nil {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	result := gin.H{"status": status}
	if status == "confirmed" {
		result["bot_token"] = botToken
		result["account_id"] = accountID
		result["base_url"] = baseURL
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
