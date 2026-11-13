// Package account manages a user's Tao API key binding.
package account

import (
	"net/http"

	"github.com/gin-gonic/gin"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Handler implements /v1/tao/account.
type Handler struct{}

// Attach registers Tao account routes.
func Attach(group *gin.RouterGroup, oauthSvc oauthtypes.OAuth) {
	h := &Handler{}
	group.GET("", oauthSvc.Guard, h.Get)
	group.PUT("", oauthSvc.Guard, h.Bind)
	group.DELETE("", oauthSvc.Guard, h.Delete)
}

func (h *Handler) Get(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	bound, err := taocomputer.HasKey(c.Request.Context(), info.UserID)
	if err != nil {
		serverError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"bound": bound})
}

func (h *Handler) Bind(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	var req struct {
		APIKey string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	if err := taocomputer.BindKey(c.Request.Context(), info.UserID, req.APIKey); err != nil {
		badRequest(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"bound": true})
}

func (h *Handler) Delete(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	if err := taocomputer.DeleteKey(c.Request.Context(), info.UserID); err != nil {
		serverError(c, err)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func badRequest(c *gin.Context, err error) {
	response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{Code: response.ErrInvalidRequest.Code, ErrorDescription: err.Error()})
}

func forbidden(c *gin.Context) {
	response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{Code: "forbidden", ErrorDescription: "access denied"})
}

func serverError(c *gin.Context, err error) {
	response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{Code: "server_error", ErrorDescription: err.Error()})
}
