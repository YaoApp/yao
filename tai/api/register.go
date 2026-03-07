package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/tai/registry"
)

// authenticateBearer validates a Bearer token and returns the caller's identity.
// Package-level var so tests can inject a mock without an OAuth service.
var authenticateBearer = authenticateBearerDefault

func authenticateBearerDefault(token string) (registry.AuthInfo, error) {
	svc := oauth.OAuth
	if svc == nil {
		return registry.AuthInfo{}, fmt.Errorf("oauth service not initialized")
	}
	result, err := svc.AuthenticateToken(oauth.AuthInput{AccessToken: token})
	if err != nil {
		return registry.AuthInfo{}, err
	}
	info := registry.AuthInfo{}
	if result.Info != nil {
		info.Subject = result.Info.Subject
		info.UserID = result.Info.UserID
		info.ClientID = result.Info.ClientID
		info.Scope = result.Info.Scope
		info.TeamID = result.Info.TeamID
		info.TenantID = result.Info.TenantID
	}
	return info, nil
}

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return auth[7:]
	}
	return ""
}

// registerRequest is the JSON body for POST /tai-nodes/register.
type registerRequest struct {
	TaiID        string              `json:"tai_id"`
	MachineID    string              `json:"machine_id"`
	Version      string              `json:"version"`
	Addr         string              `json:"addr"`
	Ports        map[string]int      `json:"ports"`
	Capabilities map[string]bool     `json:"capabilities"`
	System       registry.SystemInfo `json:"system"`
}

// heartbeatRequest is the JSON body for POST /tai-nodes/heartbeat.
type heartbeatRequest struct {
	TaiID string `json:"tai_id"`
}

// HandleRegister handles POST /tai-nodes/register.
// Validates Bearer token, extracts AuthInfo, and writes the node to the Registry.
func HandleRegister(c *gin.Context) {
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registry not initialized"})
		return
	}

	bearer := extractBearer(c.Request)
	if bearer == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
		return
	}

	authInfo, err := authenticateBearer(bearer)
	if err != nil {
		slog.Warn("tai register auth failed", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.TaiID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tai_id is required"})
		return
	}

	node := &registry.TaiNode{
		TaiID:        req.TaiID,
		MachineID:    req.MachineID,
		Version:      req.Version,
		Auth:         authInfo,
		System:       req.System,
		Mode:         "direct",
		Addr:         req.Addr,
		Ports:        req.Ports,
		Capabilities: req.Capabilities,
	}
	reg.Register(node)

	remoteIP := c.ClientIP()
	slog.Info("tai node registered via API",
		"tai_id", req.TaiID, "remote_ip", remoteIP, "user_id", authInfo.UserID)

	c.JSON(http.StatusOK, gin.H{
		"status":    "registered",
		"tai_id":    req.TaiID,
		"remote_ip": remoteIP,
	})
}

// HandleHeartbeat handles POST /tai-nodes/heartbeat.
// Validates Bearer token and updates the node's last ping timestamp.
func HandleHeartbeat(c *gin.Context) {
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registry not initialized"})
		return
	}

	bearer := extractBearer(c.Request)
	if bearer == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
		return
	}

	authInfo, err := authenticateBearer(bearer)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.TaiID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tai_id is required"})
		return
	}

	snap, ok := reg.Get(req.TaiID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "tai node not found"})
		return
	}
	if snap.Auth.ClientID != authInfo.ClientID {
		c.JSON(http.StatusForbidden, gin.H{"error": "tai_id does not belong to this client"})
		return
	}

	reg.UpdatePing(req.TaiID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleUnregister handles DELETE /tai-nodes/register/:tai_id.
// Validates Bearer token, checks ownership, and removes the node.
func HandleUnregister(c *gin.Context) {
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registry not initialized"})
		return
	}

	bearer := extractBearer(c.Request)
	if bearer == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
		return
	}

	authInfo, err := authenticateBearer(bearer)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	taiID := c.Param("tai_id")
	if taiID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tai_id is required"})
		return
	}

	snap, ok := reg.Get(taiID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "tai node not found"})
		return
	}
	if snap.Auth.ClientID != authInfo.ClientID {
		c.JSON(http.StatusForbidden, gin.H{"error": "tai_id does not belong to this client"})
		return
	}

	reg.Unregister(taiID)
	slog.Info("tai node unregistered via API", "tai_id", taiID, "user_id", authInfo.UserID)

	c.JSON(http.StatusOK, gin.H{"status": "unregistered"})
}
