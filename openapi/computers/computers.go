// Package computers exposes the Computer module HTTP API.
package computers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/computer/types"
	oauth "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Handler implements /v1/computers.
type Handler struct {
	Service *computer.Service
}

// Attach registers Computer routes.
func Attach(group *gin.RouterGroup, oauthSvc oauthtypes.OAuth) {
	h := &Handler{Service: computer.New().WithTokens(oauth.OAuth)}
	group.GET("", oauthSvc.Guard, h.List)
	group.POST("/connect", oauthSvc.Guard, h.Connect)
	group.GET("/:computer_id", oauthSvc.Guard, h.Get)
	group.PATCH("/:computer_id", oauthSvc.Guard, h.Update)
	group.DELETE("/:computer_id", oauthSvc.Guard, h.Delete)
	group.POST("/:computer_id/disconnect", oauthSvc.Guard, h.Disconnect)
	group.POST("/:computer_id/reconnect", oauthSvc.Guard, h.Reconnect)
	group.GET("/:computer_id/workspaces", oauthSvc.Guard, h.Workspaces)
	group.GET("/:computer_id/runtimes", oauthSvc.Guard, h.Runtimes)
	group.GET("/:computer_id/containers", oauthSvc.Guard, h.Containers)
}

func (h *Handler) List(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	filter := types.ListFilter{Page: queryInt(c, "page"), PageSize: queryInt(c, "page_size")}
	if !isAdmin(c.Request.Context(), info.UserID) {
		filter.UserID = info.UserID
		filter.TeamID = info.TeamID
	}
	items, err := h.Service.List(c.Request.Context(), filter)
	if err != nil {
		serverError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"data": items})
}

func (h *Handler) Connect(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	var req types.ConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	if req.Visibility == types.VisibilityGlobal && !isAdmin(c.Request.Context(), info.UserID) {
		forbidden(c)
		return
	}
	req.Scope = types.ScopeUser
	req.UserID = info.UserID
	if info.TeamID != "" {
		req.Scope = types.ScopeTeam
		req.TeamID = info.TeamID
	}
	result, err := h.Service.Connect(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, computer.ErrInvalid) {
			badRequest(c, err)
			return
		}
		serverError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func (h *Handler) Get(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	item, err := h.Service.Get(c.Request.Context(), c.Param("computer_id"))
	if err != nil {
		notFoundOrError(c, err)
		return
	}
	if !canManage(c.Request.Context(), info, item) {
		forbidden(c)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, item)
}

func (h *Handler) Update(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	item, err := h.Service.Get(c.Request.Context(), c.Param("computer_id"))
	if err != nil {
		notFoundOrError(c, err)
		return
	}
	if !canManage(c.Request.Context(), info, item) {
		forbidden(c)
		return
	}
	var req types.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	if req.Visibility != nil && *req.Visibility == types.VisibilityGlobal && !isAdmin(c.Request.Context(), info.UserID) {
		forbidden(c)
		return
	}
	updated, err := h.Service.Update(c.Request.Context(), item.ID, req)
	if err != nil {
		if errors.Is(err, computer.ErrInvalid) {
			badRequest(c, err)
			return
		}
		serverError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, updated)
}

func (h *Handler) Delete(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	item, err := h.Service.Get(c.Request.Context(), c.Param("computer_id"))
	if err != nil {
		notFoundOrError(c, err)
		return
	}
	if !canManage(c.Request.Context(), info, item) {
		forbidden(c)
		return
	}
	if err := h.Service.Delete(c.Request.Context(), item.ID); err != nil {
		serverError(c, err)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (h *Handler) Disconnect(c *gin.Context) {
	h.changeStatus(c, types.StatusDisconnected)
}

func (h *Handler) Reconnect(c *gin.Context) {
	h.changeStatus(c, types.StatusPending)
}

func (h *Handler) changeStatus(c *gin.Context, status string) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	item, err := h.Service.Get(c.Request.Context(), c.Param("computer_id"))
	if err != nil {
		notFoundOrError(c, err)
		return
	}
	if !canManage(c.Request.Context(), info, item) {
		forbidden(c)
		return
	}
	var errNext error
	if status == types.StatusDisconnected {
		errNext = h.Service.Disconnect(c.Request.Context(), item.ID)
	} else {
		errNext = h.Service.Reconnect(c.Request.Context(), item.ID)
	}
	if errNext != nil {
		serverError(c, errNext)
		return
	}
	updated, _ := h.Service.Get(c.Request.Context(), item.ID)
	response.RespondWithSuccess(c, http.StatusOK, updated)
}

func (h *Handler) Workspaces(c *gin.Context) {
	h.queryList(c, func(id string) (interface{}, error) { return h.Service.ListWorkspaces(c.Request.Context(), id) })
}

func (h *Handler) Runtimes(c *gin.Context) {
	h.queryList(c, func(id string) (interface{}, error) { return h.Service.ListRuntimes(c.Request.Context(), id) })
}

func (h *Handler) Containers(c *gin.Context) {
	h.queryList(c, func(id string) (interface{}, error) { return h.Service.ListContainers(c.Request.Context(), id) })
}

func (h *Handler) queryList(c *gin.Context, load func(string) (interface{}, error)) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	item, err := h.Service.Get(c.Request.Context(), c.Param("computer_id"))
	if err != nil {
		notFoundOrError(c, err)
		return
	}
	if !canManage(c.Request.Context(), info, item) {
		forbidden(c)
		return
	}
	value, err := load(item.ID)
	if err != nil {
		serverError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, value)
}

func canManage(ctx context.Context, info *oauthtypes.AuthorizedInfo, item *types.Computer) bool {
	if info == nil || item == nil {
		return false
	}
	if item.UserID == info.UserID {
		return true
	}
	return isAdmin(ctx, info.UserID)
}

func isAdmin(ctx context.Context, userID string) bool {
	if role.RoleManager == nil || userID == "" {
		return false
	}
	roleID, err := role.RoleManager.GetUserRole(ctx, userID)
	return err == nil && (roleID == "system:root" || roleID == "system:admin")
}

func queryInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func badRequest(c *gin.Context, err error) {
	response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{Code: response.ErrInvalidRequest.Code, ErrorDescription: err.Error()})
}

func forbidden(c *gin.Context) {
	response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{Code: "forbidden", ErrorDescription: "access denied"})
}

func notFoundOrError(c *gin.Context, err error) {
	if errors.Is(err, computer.ErrNotFound) {
		response.RespondWithError(c, http.StatusNotFound, &response.ErrorResponse{Code: "not_found", ErrorDescription: "computer not found"})
		return
	}
	serverError(c, err)
}

func serverError(c *gin.Context, err error) {
	response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{Code: "server_error", ErrorDescription: err.Error()})
}
