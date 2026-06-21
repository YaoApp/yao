package task

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach registers task CRUD routes on the given group (scoped to /agent/tasks)
func Attach(group *gin.RouterGroup, oauth oauthtypes.OAuth) {
	group.Use(oauth.Guard)
	group.GET("", handleList)
	group.POST("", handleCreate)
	group.GET("/:chat_id", handleGet)
	group.PUT("/:chat_id", handleUpdate)
	group.DELETE("/:chat_id", handleDelete)
	group.PUT("/:chat_id/move", handleMove)
}

func handleList(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))

	q := &tasksvc.ListQuery{
		RunStatus:   c.Query("run_status"),
		AssistantID: c.Query("assistant_id"),
		BoardID:     c.Query("board_id"),
	}
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		q.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 {
		q.PageSize = ps
	}

	result, err := tasksvc.List(c.Request.Context(), auth, q)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleCreate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))

	var req tasksvc.CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	result, err := tasksvc.Create(c.Request.Context(), auth, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusCreated, result)
}

func handleGet(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	result, err := tasksvc.Get(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleUpdate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	var req tasksvc.UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	result, err := tasksvc.Update(c.Request.Context(), auth, chatID, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleDelete(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	err := tasksvc.Delete(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusNoContent, nil)
}

func handleMove(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	var req tasksvc.MoveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	err := tasksvc.Move(c.Request.Context(), auth, chatID, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}

func toProcessAuth(info *oauthtypes.AuthorizedInfo) *process.AuthorizedInfo {
	if info == nil {
		return &process.AuthorizedInfo{}
	}
	return &process.AuthorizedInfo{
		Subject:   info.Subject,
		ClientID:  info.ClientID,
		Scope:     info.Scope,
		SessionID: info.SessionID,
		UserID:    info.UserID,
		TeamID:    info.TeamID,
		TenantID:  info.TenantID,
		Constraints: process.DataConstraints{
			OwnerOnly:   info.Constraints.OwnerOnly,
			CreatorOnly: info.Constraints.CreatorOnly,
			TeamOnly:    info.Constraints.TeamOnly,
		},
	}
}

func respondError(c *gin.Context, status int, err error) {
	response.RespondWithError(c, status, &response.ErrorResponse{
		Code:             "error",
		ErrorDescription: err.Error(),
	})
}
