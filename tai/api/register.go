package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth"
	tai "github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/taiid"
	"github.com/yaoapp/yao/tai/types"
)

// authenticateBearer validates a Bearer token and returns the caller's identity.
// Package-level var so tests can inject a mock without an OAuth service.
var authenticateBearer = authenticateBearerDefault

func authenticateBearerDefault(token string) (types.AuthInfo, error) {
	svc := oauth.OAuth
	if svc == nil {
		return types.AuthInfo{}, fmt.Errorf("oauth service not initialized")
	}
	result, err := svc.AuthenticateToken(oauth.AuthInput{AccessToken: token})
	if err != nil {
		return types.AuthInfo{}, err
	}
	info := types.AuthInfo{}
	if result.Info != nil {
		info.Subject = result.Info.Subject
		info.UserID = result.Info.UserID
		info.ClientID = result.Info.ClientID
		info.Scope = result.Info.Scope
		info.TeamID = result.Info.TeamID
		info.TenantID = result.Info.TenantID
	}

	slog.Info("[auth] buildAuthInfo result",
		"subject", info.Subject, "user_id", info.UserID,
		"client_id", info.ClientID, "team_id", info.TeamID,
		"scope", info.Scope)

	if result.Claims != nil {
		slog.Info("[auth] claims",
			"claims.TeamID", result.Claims.TeamID,
			"claims.TenantID", result.Claims.TenantID,
			"claims.ClientID", result.Claims.ClientID,
			"claims.Subject", result.Claims.Subject)
		if result.Claims.Extra != nil {
			slog.Info("[auth] claims.Extra", "extra", fmt.Sprintf("%+v", result.Claims.Extra))
		} else {
			slog.Info("[auth] claims.Extra is nil")
		}

		if info.TeamID == "" {
			switch v := result.Claims.Extra["team_id"].(type) {
			case string:
				info.TeamID = v
				slog.Info("[auth] team_id from Extra (string)", "team_id", v)
			case float64:
				info.TeamID = fmt.Sprintf("%.0f", v)
				slog.Info("[auth] team_id from Extra (float64)", "team_id", info.TeamID)
			default:
				slog.Info("[auth] team_id not found in Extra or unknown type",
					"type", fmt.Sprintf("%T", result.Claims.Extra["team_id"]),
					"value", fmt.Sprintf("%v", result.Claims.Extra["team_id"]))
			}
		}
		if info.TenantID == "" {
			if v, ok := result.Claims.Extra["tenant_id"].(string); ok {
				info.TenantID = v
			}
		}
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
	NodeID       string           `json:"node_id,omitempty"`
	ClientID     string           `json:"client_id,omitempty"`
	MachineID    string           `json:"machine_id"`
	DisplayName  string           `json:"display_name,omitempty"`
	Version      string           `json:"version"`
	Addr         string           `json:"addr"`
	Ports        map[string]int   `json:"ports"`
	Capabilities map[string]bool  `json:"capabilities"`
	System       types.SystemInfo `json:"system"`
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

	if req.NodeID == "" || req.MachineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id and machine_id are required"})
		return
	}

	resolvedTaiID, err := taiid.Generate(req.MachineID, req.NodeID)
	if err != nil {
		slog.Warn("taiid generation failed", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to generate tai_id"})
		return
	}

	remoteIP := c.ClientIP()
	addr := req.Addr
	if addr == "" && remoteIP != "" {
		grpcPort := req.Ports["grpc"]
		if grpcPort > 0 {
			addr = fmt.Sprintf("tai://%s:%d", remoteIP, grpcPort)
		} else {
			addr = remoteIP
		}
	}

	node := &registry.TaiNode{
		TaiID:        resolvedTaiID,
		MachineID:    req.MachineID,
		Version:      req.Version,
		DisplayName:  req.DisplayName,
		Auth:         authInfo,
		System:       req.System,
		Mode:         "direct",
		Addr:         addr,
		Ports:        portsFromMap(req.Ports),
		Capabilities: capsFromMap(req.Capabilities),
	}
	reg.Register(node)
	slog.Info("[register] node registered via API",
		"tai_id", resolvedTaiID, "addr", addr, "remote_ip", remoteIP,
		"user_id", authInfo.UserID, "team_id", authInfo.TeamID)

	allBefore := reg.List()
	slog.Info("[register] registry snapshot after Register",
		"total", len(allBefore))
	for _, s := range allBefore {
		slog.Info("[register]   node", "tai_id", s.TaiID, "mode", s.Mode, "addr", s.Addr)
	}

	if strings.HasPrefix(addr, "tai://") {
		slog.Info("[register] launching connectRegisteredNode goroutine",
			"tai_id", resolvedTaiID, "addr", addr)
		go connectRegisteredNode(resolvedTaiID, addr, portsFromMap(req.Ports), reg)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "registered",
		"tai_id":    resolvedTaiID,
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

func portsFromMap(m map[string]int) types.Ports {
	return types.Ports{
		GRPC:   m["grpc"],
		HTTP:   m["http"],
		VNC:    m["vnc"],
		Docker: m["docker"],
		K8s:    m["k8s"],
	}
}

func capsFromMap(m map[string]bool) types.Capabilities {
	return types.Capabilities{
		Docker:   m["docker"],
		K8s:      m["k8s"],
		HostExec: m["host_exec"],
		VNC:      m["vnc"],
	}
}

// connectRegisteredNode dials the Tai node via DialRemote and binds the
// returned ConnResources to the taiID in the registry. No double-registration.
func connectRegisteredNode(taiID, addr string, ports types.Ports, reg *registry.Registry) {
	slog.Info("[connect] start", "tai_id", taiID, "addr", addr)

	host := extractHost(addr)
	if host == "" {
		slog.Warn("[connect] failed to extract host from addr", "addr", addr)
		return
	}

	res, err := tai.DialRemote(host, ports)
	if err != nil {
		slog.Warn("[connect] DialRemote failed",
			"tai_id", taiID, "addr", addr, "err", err)
		return
	}

	reg.SetResources(taiID, res)
	slog.Info("[connect] done", "tai_id", taiID)
}

func extractHost(addr string) string {
	addr = strings.TrimPrefix(addr, "tai://")
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		return addr[:idx]
	}
	return addr
}
