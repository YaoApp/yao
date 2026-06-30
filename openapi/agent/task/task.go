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

// Attach registers task CRUD and execution routes on the given group (scoped to /agent/tasks)
func Attach(group *gin.RouterGroup, oauth oauthtypes.OAuth) {
	group.Use(oauth.Guard)
	group.GET("", handleList)
	group.POST("", handleCreate)
	group.GET("/quota", handleGetQuota)
	group.GET("/:chat_id", handleGet)
	group.PUT("/:chat_id", handleUpdate)
	group.DELETE("/:chat_id", handleDelete)
	group.PUT("/:chat_id/move", handleMove)
	group.PUT("/:chat_id/archive", handleArchive)
	group.PUT("/:chat_id/unarchive", handleUnarchive)

	// Sub-resource endpoints
	group.GET("/:chat_id/secrets", handleTaskSecretsGet)
	group.PUT("/:chat_id/secrets", handleTaskSecretsUpdate)
	group.DELETE("/:chat_id/secrets/:key", handleTaskSecretDelete)
	group.GET("/:chat_id/schedule", handleTaskScheduleGet)
	group.PUT("/:chat_id/schedule", handleTaskScheduleUpdate)
	group.GET("/:chat_id/schedule/logs", handleTaskScheduleLogsGet)
	group.GET("/:chat_id/skills", handleTaskSkillsGet)
	group.GET("/:chat_id/computers", handleTaskComputersGet)
	group.GET("/:chat_id/sandbox", handleTaskSandboxGet)
	group.PUT("/:chat_id/sandbox", handleTaskSandboxPut)

	// Execution routes (Plan 3)
	group.GET("/:chat_id/ws", handleWS)
	group.GET("/:chat_id/stream", handleSSE)
	group.POST("/:chat_id/run", handleRun)
	group.POST("/:chat_id/stop", handleStop)
	group.POST("/:chat_id/input", handleInput)
	group.PUT("/:chat_id/priority", handleSetPriority)
}

func handleList(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))

	q := &tasksvc.ListQuery{
		RunStatus:   c.Query("run_status"),
		AssistantID: c.Query("assistant_id"),
		BoardID:     c.Query("board_id"),
		Locale:      c.Query("locale"),
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
	info := authorized.GetInfo(c)
	chatID := c.Param("chat_id")

	result, err := tasksvc.Get(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	if locale := c.Query("locale"); locale != "" {
		tasksvc.TranslateAssistantName(result, locale)
	}
	tasksvc.ResolveConnectorLabel(result, info)
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

func handleArchive(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	err := tasksvc.Archive(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}

func handleUnarchive(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	var req struct {
		ColumnID string `json:"column_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	err := tasksvc.Unarchive(c.Request.Context(), auth, chatID, req.ColumnID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
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

func handleRun(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")
	var req tasksvc.RunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	task, _ := tasksvc.Get(c.Request.Context(), auth, chatID)
	result, err := tasksvc.Run(c.Request.Context(), auth, chatID, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	if task != nil && task.RunCount == 0 {
		if firstMsg := tasksvc.ExtractFirstUserMessage(req.Messages); firstMsg != "" {
			tasksvc.ExtractTaskMetadata(chatID, firstMsg, auth)
		}
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleStop(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")
	force := c.Query("force") == "true"
	if err := tasksvc.Stop(c.Request.Context(), auth, chatID, force); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"ok": true})
}

func handleInput(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")
	var req tasksvc.InputReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if err := tasksvc.Input(c.Request.Context(), auth, chatID, &req); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"ok": true})
}

func handleSetPriority(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")
	var req struct {
		Priority int `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if err := tasksvc.SetPriority(c.Request.Context(), auth, chatID, req.Priority); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"ok": true})
}

func handleGetQuota(c *gin.Context) {
	info := authorized.GetInfo(c)
	status := tasksvc.GlobalQuota.GetStatus(info.TeamID)
	response.RespondWithSuccess(c, http.StatusOK, status)
}

func respondError(c *gin.Context, status int, err error) {
	response.RespondWithError(c, status, &response.ErrorResponse{
		Code:             "error",
		ErrorDescription: err.Error(),
	})
}
