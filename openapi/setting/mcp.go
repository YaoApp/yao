package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/yao/mcpclient"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

const mcpMaskPrefixLen = 7

func mcpOwner(info *oauthTypes.AuthorizedInfo) mcpclient.ClientOwner {
	if info.TeamID != "" {
		return mcpclient.ClientOwner{Type: "team", ID: info.TeamID}
	}
	return mcpclient.ClientOwner{Type: "user", ID: info.UserID}
}

func mcpCheckOwnership(c *mcpclient.Client, info *oauthTypes.AuthorizedInfo) error {
	owner := mcpOwner(info)
	if c.Owner.Type != owner.Type || c.Owner.ID != owner.ID {
		return fmt.Errorf("server not found")
	}
	return nil
}

func mcpMaskToken(token string) string {
	if token == "" {
		return ""
	}
	plain := decryptValue(token)
	if len(plain) <= mcpMaskPrefixLen {
		return strings.Repeat("*", len(plain))
	}
	suffix := plain[len(plain)-4:]
	prefix := plain[:mcpMaskPrefixLen]
	return prefix + "..." + suffix
}

func mcpClientToResponse(c *mcpclient.Client) map[string]interface{} {
	resp := map[string]interface{}{
		"id":        c.ID,
		"name":      c.Name,
		"label":     c.Label,
		"transport": string(c.Transport),
		"url":       c.URL,
		"enabled":   c.Enabled,
		"status":    c.Status,
	}
	if c.Description != "" {
		resp["description"] = c.Description
	}
	if c.AuthorizationToken != "" {
		resp["authorization_token"] = mcpMaskToken(c.AuthorizationToken)
	}
	if c.Timeout != "" {
		resp["timeout"] = c.Timeout
	}
	if len(c.Tags) > 0 {
		resp["tags"] = c.Tags
	}
	return resp
}

// handleMCPList returns MCP servers for the current user/team.
// Only http and sse transports are returned.
// GET /setting/mcp/servers
func handleMCPList(c *gin.Context) {
	info := authorized.GetInfo(c)
	owner := mcpOwner(info)

	if mcpclient.Global == nil {
		respondError(c, http.StatusInternalServerError, "MCP client registry not initialized")
		return
	}

	all, err := mcpclient.Global.List(&mcpclient.ClientFilter{
		Owner:  &owner,
		Source: mcpclient.ClientSourceAll,
	})
	if err != nil {
		all = []mcpclient.Client{}
	}

	servers := make([]map[string]interface{}, 0, len(all))
	for i := range all {
		t := all[i].Transport
		if t != mcpTypes.TransportHTTP && t != mcpTypes.TransportSSE {
			continue
		}
		servers = append(servers, mcpClientToResponse(&all[i]))
	}

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"servers": servers,
	})
}

// handleMCPCreate creates a new MCP server.
// POST /setting/mcp/servers
func handleMCPCreate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)

	if mcpclient.Global == nil {
		respondError(c, http.StatusInternalServerError, "MCP client registry not initialized")
		return
	}

	var body struct {
		Name               string   `json:"name"`
		Label              string   `json:"label"`
		Description        string   `json:"description"`
		Transport          string   `json:"transport"`
		URL                string   `json:"url"`
		AuthorizationToken string   `json:"authorization_token"`
		Timeout            string   `json:"timeout"`
		Tags               []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Name == "" {
		respondError(c, http.StatusBadRequest, "name is required")
		return
	}
	if body.URL == "" {
		respondError(c, http.StatusBadRequest, "url is required")
		return
	}
	if _, err := url.ParseRequestURI(body.URL); err != nil {
		respondError(c, http.StatusBadRequest, "invalid url format")
		return
	}

	transport := mcpTypes.TransportHTTP
	if body.Transport == "sse" {
		transport = mcpTypes.TransportSSE
	}

	owner := mcpOwner(info)

	existing, _ := mcpclient.Global.List(&mcpclient.ClientFilter{
		Owner:  &owner,
		Source: mcpclient.ClientSourceAll,
	})
	for _, ex := range existing {
		if strings.EqualFold(ex.Name, body.Name) {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("server with name \"%s\" already exists", body.Name))
			return
		}
	}

	clientID := owner.Type + "." + owner.ID + "." + body.Name
	client := &mcpclient.Client{
		ClientDSL: mcpTypes.ClientDSL{
			ID:        clientID,
			Name:      body.Name,
			Transport: transport,
			URL:       body.URL,
			Timeout:   body.Timeout,
			MetaInfo: gouTypes.MetaInfo{
				Label:       body.Label,
				Description: body.Description,
				Tags:        body.Tags,
			},
		},
		Enabled: true,
		Status:  "unconfigured",
		Source:  mcpclient.ClientSourceDynamic,
		Owner:   owner,
	}

	if body.AuthorizationToken != "" {
		client.AuthorizationToken = encryptValue(body.AuthorizationToken)
	}
	if body.Timeout == "" {
		client.Timeout = "30s"
	}

	token := body.AuthorizationToken
	status, _, errMsg := mcpProbeRaw(transport, body.URL, token, client.Timeout)
	if status != "connected" {
		respondError(c, http.StatusBadRequest, errMsg)
		return
	}

	client.Status = "connected"
	created, err := mcpclient.Global.Create(client)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, mcpClientToResponse(created))
}

// handleMCPUpdate updates an existing MCP server.
// PUT /setting/mcp/servers/:id
func handleMCPUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	id := c.Param("id")

	if mcpclient.Global == nil {
		respondError(c, http.StatusInternalServerError, "MCP client registry not initialized")
		return
	}

	existing, err := mcpclient.Global.Get(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "server not found")
		return
	}
	if err := mcpCheckOwnership(existing, info); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	var body struct {
		Name               string   `json:"name"`
		Label              string   `json:"label"`
		Description        string   `json:"description"`
		Transport          string   `json:"transport"`
		URL                string   `json:"url"`
		AuthorizationToken string   `json:"authorization_token"`
		Timeout            string   `json:"timeout"`
		Tags               []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.URL != "" {
		if _, err := url.ParseRequestURI(body.URL); err != nil {
			respondError(c, http.StatusBadRequest, "invalid url format")
			return
		}
	}

	updated := &mcpclient.Client{
		ClientDSL: mcpTypes.ClientDSL{
			ID:   id,
			Name: existing.Name,
			MetaInfo: gouTypes.MetaInfo{
				Label:       existing.Label,
				Description: existing.Description,
				Tags:        existing.Tags,
			},
			Transport:          existing.Transport,
			URL:                existing.URL,
			AuthorizationToken: existing.AuthorizationToken,
			Timeout:            existing.Timeout,
		},
		Enabled: existing.Enabled,
		Status:  existing.Status,
		Source:  existing.Source,
		Owner:   existing.Owner,
	}

	if body.Name != "" {
		updated.Name = body.Name
	}
	if body.Label != "" {
		updated.Label = body.Label
	}
	if body.Description != "" {
		updated.Description = body.Description
	}
	if body.Transport != "" {
		if body.Transport == "sse" {
			updated.Transport = mcpTypes.TransportSSE
		} else {
			updated.Transport = mcpTypes.TransportHTTP
		}
	}
	if body.URL != "" {
		updated.URL = body.URL
	}
	if body.AuthorizationToken != "" {
		updated.AuthorizationToken = encryptValue(body.AuthorizationToken)
	}
	if body.Timeout != "" {
		updated.Timeout = body.Timeout
	}
	if body.Tags != nil {
		updated.Tags = body.Tags
	}

	token := body.AuthorizationToken
	if token == "" && updated.AuthorizationToken != "" {
		token = decryptValue(updated.AuthorizationToken)
	}
	probeTransport := updated.Transport
	probeURL := updated.URL
	status, _, errMsg := mcpProbeRaw(probeTransport, probeURL, token, updated.Timeout)
	if status != "connected" {
		respondError(c, http.StatusBadRequest, errMsg)
		return
	}

	updated.Status = "connected"
	result, err := mcpclient.Global.Update(id, updated)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, mcpClientToResponse(result))
}

// handleMCPDelete removes an MCP server.
// DELETE /setting/mcp/servers/:id
func handleMCPDelete(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	id := c.Param("id")

	if mcpclient.Global == nil {
		respondError(c, http.StatusInternalServerError, "MCP client registry not initialized")
		return
	}

	existing, err := mcpclient.Global.Get(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "server not found")
		return
	}
	if err := mcpCheckOwnership(existing, info); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	if err := mcpclient.Global.Delete(id); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// mcpProbeRaw creates a temporary MCP client from raw config, tests Connect+Initialize+ListTools.
func mcpProbeRaw(transport mcpTypes.TransportType, urlStr, token, timeout string) (status string, latencyMs int64, errMsg string) {
	if timeout == "" {
		timeout = "30s"
	}
	tempID := fmt.Sprintf("__probe_%d", time.Now().UnixNano())
	dsl := mcpTypes.ClientDSL{
		ID:                 tempID,
		Name:               tempID,
		Transport:          transport,
		URL:                urlStr,
		AuthorizationToken: token,
		Timeout:            timeout,
	}
	dslJSON, err := json.Marshal(dsl)
	if err != nil {
		return "disconnected", 0, fmt.Sprintf("marshal: %s", err)
	}

	start := time.Now()
	mcpClient, err := mcp.LoadClientSourceWithType(string(dslJSON), tempID, "")
	if err != nil {
		return "disconnected", 0, fmt.Sprintf("load: %s", err)
	}
	defer mcp.UnloadClient(tempID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := mcpClient.Connect(ctx); err != nil {
		return "disconnected", time.Since(start).Milliseconds(), fmt.Sprintf("connect: %s", err)
	}
	defer mcpClient.Disconnect(context.Background())

	if _, err := mcpClient.Initialize(ctx); err != nil {
		return "disconnected", time.Since(start).Milliseconds(), fmt.Sprintf("initialize: %s", err)
	}

	_, err = mcpClient.ListTools(ctx, "")
	latencyMs = time.Since(start).Milliseconds()
	if err != nil {
		return "disconnected", latencyMs, fmt.Sprintf("listTools: %s", err)
	}
	return "connected", latencyMs, ""
}

// handleMCPTest tests connectivity using raw config (for add/edit before save).
// Creates a temporary runtime client, tests ListTools, then cleans up.
// POST /setting/mcp/test
func handleMCPTest(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	var body struct {
		Transport          string `json:"transport"`
		URL                string `json:"url"`
		AuthorizationToken string `json:"authorization_token"`
		Timeout            string `json:"timeout"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.URL == "" {
		respondError(c, http.StatusBadRequest, "url is required")
		return
	}

	transport := mcpTypes.TransportHTTP
	if body.Transport == "sse" {
		transport = mcpTypes.TransportSSE
	}
	timeout := body.Timeout
	if timeout == "" {
		timeout = "30s"
	}

	tempID := fmt.Sprintf("__test_%d", time.Now().UnixNano())
	dsl := mcpTypes.ClientDSL{
		ID:                 tempID,
		Name:               tempID,
		Transport:          transport,
		URL:                body.URL,
		AuthorizationToken: body.AuthorizationToken,
		Timeout:            timeout,
	}

	dslJSON, err := json.Marshal(dsl)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to marshal config")
		return
	}

	start := time.Now()
	mcpClient, err := mcp.LoadClientSourceWithType(string(dslJSON), tempID, "")
	if err != nil {
		response.RespondWithSuccess(c, http.StatusOK, mcpclient.ClientTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to load client: %s", err.Error()),
		})
		return
	}
	defer mcp.UnloadClient(tempID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := mcpClient.Connect(ctx); err != nil {
		latencyMs := time.Since(start).Milliseconds()
		response.RespondWithSuccess(c, http.StatusOK, mcpclient.ClientTestResult{
			Success:   false,
			Message:   fmt.Sprintf("Connection failed: %s", err.Error()),
			LatencyMs: latencyMs,
		})
		return
	}
	defer mcpClient.Disconnect(context.Background())

	if _, err := mcpClient.Initialize(ctx); err != nil {
		latencyMs := time.Since(start).Milliseconds()
		response.RespondWithSuccess(c, http.StatusOK, mcpclient.ClientTestResult{
			Success:   false,
			Message:   fmt.Sprintf("Initialization failed: %s", err.Error()),
			LatencyMs: latencyMs,
		})
		return
	}

	_, err = mcpClient.ListTools(ctx, "")
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		response.RespondWithSuccess(c, http.StatusOK, mcpclient.ClientTestResult{
			Success:   false,
			Message:   fmt.Sprintf("Connection failed: %s", err.Error()),
			LatencyMs: latencyMs,
		})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, mcpclient.ClientTestResult{
		Success:   true,
		Message:   "Connection successful",
		LatencyMs: latencyMs,
	})
}
