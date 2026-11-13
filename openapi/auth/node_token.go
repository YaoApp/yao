package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// NodeTokenService is the subset of OAuth used by node-token HTTP handlers.
type NodeTokenService interface {
	RefreshNodeToken(ctx context.Context, raw string) (*oauth.NodeToken, error)
	RotateNodeToken(ctx context.Context, computerID string) (*oauth.NodeToken, error)
	RevokeNodeToken(ctx context.Context, jti string) error
	GetNodeTokenInfo(ctx context.Context, jti string) (*oauth.NodeInfo, error)
}

// NodeTokenHandler owns the HTTP-facing node-token lifecycle.
// nodeTokenValidator is an optional capability used to mark a node online.
type nodeTokenValidator interface {
	ValidateNodeToken(context.Context, string) (*oauth.NodeInfo, error)
}

type NodeTokenHandler struct {
	Service    NodeTokenService
	Authorizer ComputerAuthorizer
}

// Attach registers node-token routes under /auth.
func Attach(group *gin.RouterGroup, oauthSvc types.OAuth) {
	service, ok := oauthSvc.(NodeTokenService)
	if !ok {
		return
	}
	handler := &NodeTokenHandler{Service: service, Authorizer: ModelComputerAuthorizer{}}
	authGroup := group.Group("/auth")
	authGroup.POST("/node-tokens/refresh", handler.Refresh)
	authGroup.POST("/node-tokens/rotate", oauthSvc.Guard, handler.Rotate)
	authGroup.DELETE("/node-tokens/:jti", oauthSvc.Guard, handler.Revoke)
}

// Refresh handles POST /auth/node-tokens/refresh.
// It intentionally does not use oauth.Guard: the bearer is a node_token.
func (h *NodeTokenHandler) Refresh(c *gin.Context) {
	raw := bearerToken(c)
	if raw == "" {
		var body struct {
			NodeToken string `json:"node_token"`
		}
		_ = c.ShouldBindJSON(&body)
		raw = body.NodeToken
	}
	if raw == "" {
		respondNodeError(c, http.StatusUnauthorized, "invalid_token", "node_token is required")
		return
	}
	if validator, ok := h.Service.(nodeTokenValidator); ok {
		if info, err := validator.ValidateNodeToken(c.Request.Context(), raw); err == nil && info.ComputerID != "" {
			markComputerOnline(c.Request.Context(), info.ComputerID)
		}
	}
	token, err := h.Service.RefreshNodeToken(c.Request.Context(), raw)
	if err != nil {
		respondNodeError(c, nodeErrorStatus(err), "invalid_token", "invalid node_token")
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, RefreshNodeTokenResponse{NodeToken: token.Token, ExpiresAt: token.ExpiresAt})
}

// Rotate handles POST /auth/node-tokens/rotate.
func (h *NodeTokenHandler) Rotate(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		respondNodeError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req RotateNodeTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ComputerID == "" {
		respondNodeError(c, http.StatusBadRequest, "invalid_request", "computer_id is required")
		return
	}
	if err := h.Authorizer.CanManage(c.Request.Context(), info, req.ComputerID); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, ErrComputerNotFound) {
			status = http.StatusNotFound
		}
		respondNodeError(c, status, "forbidden", "cannot manage computer")
		return
	}
	token, err := h.Service.RotateNodeToken(c.Request.Context(), req.ComputerID)
	if err != nil {
		respondNodeError(c, nodeErrorStatus(err), "conflict", "cannot rotate node_token")
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, RefreshNodeTokenResponse{NodeToken: token.Token, ExpiresAt: token.ExpiresAt})
}

// Revoke handles DELETE /auth/node-tokens/:jti.
func (h *NodeTokenHandler) Revoke(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		respondNodeError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	jti := c.Param("jti")
	if jti == "" {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	nodeInfo, err := h.Service.GetNodeTokenInfo(c.Request.Context(), jti)
	if err != nil {
		// Unknown jti is intentionally idempotent to avoid enumeration.
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	if err := h.Authorizer.CanManage(c.Request.Context(), info, nodeInfo.ComputerID); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, ErrComputerNotFound) {
			status = http.StatusNotFound
		}
		respondNodeError(c, status, "forbidden", "cannot manage computer")
		return
	}
	if err := h.Service.RevokeNodeToken(c.Request.Context(), jti); err != nil {
		respondNodeError(c, http.StatusInternalServerError, "server_error", "cannot revoke node_token")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func bearerToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func nodeErrorStatus(err error) int {
	switch {
	case errors.Is(err, oauth.ErrNodeTokenConflict):
		return http.StatusConflict
	case errors.Is(err, oauth.ErrNodeTokenBusy):
		return http.StatusTooManyRequests
	case errors.Is(err, oauth.ErrNodeTokenMissing):
		return http.StatusNotFound
	default:
		return http.StatusUnauthorized
	}
}

func respondNodeError(c *gin.Context, status int, code, description string) {
	response.RespondWithError(c, status, &response.ErrorResponse{Code: code, ErrorDescription: description})
}

// markComputerOnline records a successful node authentication.
func markComputerOnline(ctx context.Context, computerID string) {
	m := model.Select("__yao.computers")
	if m == nil {
		return
	}
	_ = m.Update(computerID, map[string]interface{}{
		"status": "online", "last_seen_at": time.Now().UTC(),
	})
}
