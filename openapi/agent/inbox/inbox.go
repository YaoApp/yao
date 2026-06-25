package inbox

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	inboxsvc "github.com/yaoapp/yao/agent/inbox"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach registers inbox routes on the given group (scoped to /agent/inbox)
func Attach(group *gin.RouterGroup, oauth oauthtypes.OAuth) {
	group.Use(oauth.Guard)

	group.GET("", handleList)
	group.GET("/stats", handleStats)
	group.GET("/unread-count", handleUnreadCount)
	group.PUT("/read-all", handleReadAll)

	group.PUT("/:mail_id/read", handleRead)
	group.PUT("/:mail_id/star", handleStar)
	group.PUT("/:mail_id/unstar", handleUnstar)
	group.PUT("/:mail_id/pin", handlePin)
	group.PUT("/:mail_id/unpin", handleUnpin)
}

func handleList(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	q := &inboxsvc.ListQuery{
		Filter:  c.Query("filter"),
		Keyword: c.Query("keyword"),
		ChatID:  c.Query("chat_id"),
	}
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		q.Page = p
	}
	if s, err := strconv.Atoi(c.Query("size")); err == nil && s > 0 {
		q.Size = s
	}
	result, err := inboxsvc.List(c.Request.Context(), auth, q)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleStats(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	result, err := inboxsvc.Stats(c.Request.Context(), auth)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleUnreadCount(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	result, err := inboxsvc.UnreadCount(c.Request.Context(), auth)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleReadAll(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	var req struct {
		Type string `json:"type"`
	}
	c.ShouldBindJSON(&req)
	count, err := inboxsvc.ReadAll(c.Request.Context(), auth, req.Type)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"count": count})
}

func handleRead(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	mailID := c.Param("mail_id")
	err := inboxsvc.Read(c.Request.Context(), auth, mailID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}

func handleStar(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	mailID := c.Param("mail_id")
	err := inboxsvc.Star(c.Request.Context(), auth, mailID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}

func handleUnstar(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	mailID := c.Param("mail_id")
	err := inboxsvc.Unstar(c.Request.Context(), auth, mailID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}

func handlePin(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	mailID := c.Param("mail_id")
	err := inboxsvc.Pin(c.Request.Context(), auth, mailID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}

func handleUnpin(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	mailID := c.Param("mail_id")
	err := inboxsvc.Unpin(c.Request.Context(), auth, mailID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
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
