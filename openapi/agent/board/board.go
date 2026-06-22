package board

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	boardsvc "github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach registers board CRUD routes on the given group (scoped to /agent/boards)
func Attach(group *gin.RouterGroup, oauth oauthtypes.OAuth) {
	group.Use(oauth.Guard)

	group.GET("", handleList)
	group.POST("", handleCreate)
	group.GET("/templates", handleTemplates)
	group.POST("/from-template", handleFromTemplate)

	group.GET("/:board_id", handleGet)
	group.PUT("/:board_id", handleUpdate)
	group.DELETE("/:board_id", handleDelete)
	group.GET("/:board_id/tasks", handleBoardTasks)

	group.POST("/:board_id/columns", handleColumnCreate)
	group.PUT("/:board_id/columns/reorder", handleColumnReorder)
	group.PUT("/:board_id/columns/:column_id", handleColumnUpdate)
	group.DELETE("/:board_id/columns/:column_id", handleColumnDelete)
}

func handleList(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	result, err := boardsvc.List(c.Request.Context(), auth, &boardsvc.ListQuery{})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleCreate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	var req boardsvc.CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	result, err := boardsvc.Create(c.Request.Context(), auth, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusCreated, result)
}

func handleTemplates(c *gin.Context) {
	locale := c.Query("locale")
	result, err := boardsvc.Templates(c.Request.Context(), locale)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleFromTemplate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	var req boardsvc.FromTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	result, err := boardsvc.FromTemplate(c.Request.Context(), auth, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusCreated, result)
}

func handleGet(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	result, err := boardsvc.Get(c.Request.Context(), auth, boardID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleUpdate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	var req boardsvc.UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	result, err := boardsvc.Update(c.Request.Context(), auth, boardID, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleDelete(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	err := boardsvc.Delete(c.Request.Context(), auth, boardID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusNoContent, nil)
}

func handleBoardTasks(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	result, err := boardsvc.Tasks(c.Request.Context(), auth, boardID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleColumnCreate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	var req boardsvc.ColumnReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	result, err := boardsvc.CreateColumn(c.Request.Context(), auth, boardID, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusCreated, result)
}

func handleColumnReorder(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	var req struct {
		ColumnIDs []string `json:"column_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	err := boardsvc.ReorderColumns(c.Request.Context(), auth, boardID, req.ColumnIDs)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}

func handleColumnUpdate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	colID := c.Param("column_id")
	var req boardsvc.ColumnReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	result, err := boardsvc.UpdateColumn(c.Request.Context(), auth, boardID, colID, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleColumnDelete(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	boardID := c.Param("board_id")
	colID := c.Param("column_id")
	err := boardsvc.DeleteColumn(c.Request.Context(), auth, boardID, colID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusNoContent, nil)
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
