package workspace

import (
	"context"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/tai/registry"
	ws "github.com/yaoapp/yao/workspace"
)

// Attach registers workspace management routes on the given group.
//   - GET    /               — list workspaces (filtered by owner from token)
//   - POST   /               — create workspace (owner from token)
//   - GET    /:id            — get workspace (owner check)
//   - PUT    /:id            — update workspace (owner check)
//   - DELETE /:id            — delete workspace (owner check)
//   - GET    /:id/files      — list files
//   - GET    /:id/files/*path — read file
//   - PUT    /:id/files/*path — write file
//   - DELETE /:id/files/*path — delete file
//   - POST   /:id/mkdir      — create directory
//   - POST   /:id/rename     — rename file/directory
//   - GET    /:id/rootdir   — get workspace root directory absolute path
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	group.Use(oauth.Guard)

	group.GET("", handleList)
	group.GET("/options", handleOptions)
	group.POST("", handleCreate)
	group.GET("/:id", handleGet)
	group.PUT("/:id", handleUpdate)
	group.DELETE("/:id", handleDelete)

	group.GET("/:id/rootdir", handleRootDir)
	group.GET("/:id/files", handleListFiles)
	group.GET("/:id/files/*path", handleReadFile)
	group.PUT("/:id/files/*path", handleWriteFile)
	group.DELETE("/:id/files/*path", handleDeleteFile)
	group.POST("/:id/mkdir", handleMkdir)
	group.POST("/:id/rename", handleRename)
}

// resolveOwner returns TeamID if present, otherwise UserID.
func resolveOwner(authInfo *types.AuthorizedInfo) string {
	if authInfo != nil && authInfo.TeamID != "" {
		return authInfo.TeamID
	}
	if authInfo != nil {
		return authInfo.UserID
	}
	return ""
}

// checkWSOwner verifies the caller owns the workspace.
func checkWSOwner(c *gin.Context, w *ws.Workspace, owner string) bool {
	if owner == "" {
		return true
	}
	if w.Owner != "" && w.Owner != owner {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to access this workspace"})
		return false
	}
	return true
}

// --- request / response types ---

type createRequest struct {
	ID     string            `json:"id,omitempty"`
	Name   string            `json:"name" binding:"required"`
	Node   string            `json:"node" binding:"required"`
	Labels map[string]string `json:"labels,omitempty"`
}

type updateRequest struct {
	Name   *string           `json:"name,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

type mkdirRequest struct {
	Path string `json:"path" binding:"required"`
}

type renameRequest struct {
	OldPath string `json:"old_path" binding:"required"`
	NewPath string `json:"new_path" binding:"required"`
}

type workspaceResponse struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Owner            string            `json:"owner"`
	Node             string            `json:"node"`
	NodeName         string            `json:"node_name,omitempty"`
	NodeOS           string            `json:"node_os,omitempty"`
	NodeArch         string            `json:"node_arch,omitempty"`
	NodeKind         string            `json:"node_kind,omitempty"`
	NodeOnline       bool              `json:"node_online"`
	NodeCapabilities map[string]bool   `json:"node_capabilities,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
}

type optionsResponse struct {
	Data           []workspaceResponse `json:"data"`
	HasOnlineNodes bool                `json:"has_online_nodes"`
}

func toResponse(w *ws.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:        w.ID,
		Name:      w.Name,
		Owner:     w.Owner,
		Node:      w.Node,
		Labels:    w.Labels,
		CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: w.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func mgr() *ws.Manager {
	return ws.M()
}

// resolveAndCheckWS fetches the workspace and verifies owner permission.
func resolveAndCheckWS(c *gin.Context) (*ws.Workspace, bool) {
	m := mgr()
	if m == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace service not available"})
		return nil, false
	}

	w, err := m.Get(context.Background(), c.Param("id"))
	if err != nil {
		if err == ws.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}

	authInfo := authorized.GetInfo(c)
	if !checkWSOwner(c, w, resolveOwner(authInfo)) {
		return nil, false
	}
	return w, true
}

// --- handlers ---

func handleList(c *gin.Context) {
	m := mgr()
	if m == nil {
		response.RespondWithSuccess(c, http.StatusOK, []workspaceResponse{})
		return
	}

	authInfo := authorized.GetInfo(c)
	owner := resolveOwner(authInfo)

	list, err := m.List(context.Background(), ws.ListOptions{
		Owner: owner,
		Node:  c.Query("node"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]workspaceResponse, 0, len(list))
	for _, w := range list {
		result = append(result, toResponse(w))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt > result[j].CreatedAt
	})
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleOptions returns workspace options for the InputArea selector.
// Each workspace is enriched with its node's display info (name, OS, arch, kind, online).
// The response also includes has_online_nodes so the frontend can determine sendBlocked
// even when the workspace list is empty.
func handleOptions(c *gin.Context) {
	m := mgr()
	if m == nil {
		response.RespondWithSuccess(c, http.StatusOK, optionsResponse{Data: []workspaceResponse{}})
		return
	}

	authInfo := authorized.GetInfo(c)
	owner := resolveOwner(authInfo)

	list, err := m.List(context.Background(), ws.ListOptions{
		Owner: owner,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	nodeMap := buildNodeMap()
	hasOnline := false
	for _, n := range nodeMap {
		if n.online {
			hasOnline = true
			break
		}
	}

	result := make([]workspaceResponse, 0, len(list))
	for _, w := range list {
		r := toResponse(w)
		if info, ok := nodeMap[w.Node]; ok {
			r.NodeName = info.displayName
			r.NodeOS = info.os
			r.NodeArch = info.arch
			r.NodeKind = info.kind
			r.NodeOnline = info.online
			r.NodeCapabilities = info.capabilities
		}
		result = append(result, r)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt > result[j].CreatedAt
	})

	response.RespondWithSuccess(c, http.StatusOK, optionsResponse{
		Data:           result,
		HasOnlineNodes: hasOnline,
	})
}

type nodeInfo struct {
	displayName  string
	os           string
	arch         string
	kind         string
	online       bool
	capabilities map[string]bool
}

func buildNodeMap() map[string]nodeInfo {
	reg := registry.Global()
	if reg == nil {
		return nil
	}
	nodes := reg.List()
	m := make(map[string]nodeInfo, len(nodes))
	for _, n := range nodes {
		kind := "node"
		if n.Mode == "local" {
			kind = "host"
		}
		name := n.DisplayName
		if name == "" {
			name = n.System.Hostname
		}
		if name == "" {
			name = n.TaiID
		}
		caps := map[string]bool{}
		if n.Capabilities.HostExec {
			caps["host_exec"] = true
		}
		if n.Capabilities.Docker {
			caps["docker"] = true
		}
		if n.Capabilities.K8s {
			caps["k8s"] = true
		}
		if n.Capabilities.VNC {
			caps["vnc"] = true
		}
		m[n.TaiID] = nodeInfo{
			displayName:  name,
			os:           strings.ToLower(n.System.OS),
			arch:         strings.ToLower(n.System.Arch),
			kind:         kind,
			online:       n.Status == "online" || n.Status == "",
			capabilities: caps,
		}
	}
	return m
}

func handleCreate(c *gin.Context) {
	m := mgr()
	if m == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace service not available"})
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authInfo := authorized.GetInfo(c)
	owner := resolveOwner(authInfo)

	w, err := m.Create(context.Background(), ws.CreateOptions{
		ID:     req.ID,
		Name:   req.Name,
		Owner:  owner,
		Node:   req.Node,
		Labels: req.Labels,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.RespondWithSuccess(c, http.StatusCreated, toResponse(w))
}

func handleGet(c *gin.Context) {
	w, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, toResponse(w))
}

func handleUpdate(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	w, err := mgr().Update(context.Background(), c.Param("id"), ws.UpdateOptions{
		Name:   req.Name,
		Labels: req.Labels,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, toResponse(w))
}

func handleDelete(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	force := c.Query("force") == "true"
	if err := mgr().Delete(context.Background(), c.Param("id"), force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func handleRootDir(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	rootDir, err := mgr().MountPath(context.Background(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"root_dir": rootDir})
}

func handleListFiles(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	dir := c.DefaultQuery("path", ".")
	entries, err := mgr().ListDir(context.Background(), c.Param("id"), dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, entries)
}

func handleReadFile(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	path := c.Param("path")
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	log.Trace("[workspace] handleReadFile id=%s path=%q", c.Param("id"), path)

	data, err := mgr().ReadFile(context.Background(), c.Param("id"), path)
	if err != nil {
		log.Trace("[workspace] ReadFile error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Trace("[workspace] ReadFile ok, size=%d, encoding=%q", len(data), c.Query("encoding"))

	if c.Query("encoding") == "base64" {
		response.RespondWithSuccess(c, http.StatusOK, gin.H{
			"content":  base64.StdEncoding.EncodeToString(data),
			"encoding": "base64",
		})
		return
	}

	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	log.Trace("[workspace] serving ext=%q mime=%q size=%d", ext, mimeType, len(data))
	c.Data(http.StatusOK, mimeType, data)
}

func handleWriteFile(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	path := c.Param("path")
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	if err := mgr().WriteFile(context.Background(), c.Param("id"), path, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func handleDeleteFile(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	path := c.Param("path")
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if err := mgr().Remove(context.Background(), c.Param("id"), path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func handleMkdir(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	var req mkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mgr().MkdirAll(context.Background(), c.Param("id"), req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func handleRename(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	var req renameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mgr().Rename(context.Background(), c.Param("id"), req.OldPath, req.NewPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
