package workspace

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/tai/registry"
	ws "github.com/yaoapp/yao/workspace"
)

const maxFileSize = 50 << 20 // 50MB

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
	group.GET("/:id/preview/*path", handlePreview)

	// Git operations
	group.GET("/:id/git/repos", handleGitListRepos)
	group.GET("/:id/git/status", handleGitStatus)
	group.GET("/:id/git/diff", handleGitFileDiff)
	group.POST("/:id/git/add", handleGitAdd)
	group.POST("/:id/git/reset", handleGitReset)
	group.POST("/:id/git/commit", handleGitCommit)
	group.POST("/:id/git/discard", handleGitDiscardChanges)

	// Git config & credential management
	group.GET("/:id/git/config", handleGitConfigGet)
	group.POST("/:id/git/config", handleGitConfigSet)
	group.POST("/:id/git/credential", handleGitCredentialSet)
	group.GET("/:id/git/credentials", handleGitCredentialList)
	group.DELETE("/:id/git/credential", handleGitCredentialDelete)
	group.POST("/:id/git/ssh-key", handleGitSSHKeyImport)
	group.GET("/:id/git/ssh-keys", handleGitSSHKeyList)
	group.DELETE("/:id/git/ssh-key", handleGitSSHKeyDelete)

	// Git remote sync
	group.POST("/:id/git/fetch", handleGitFetch)
	group.POST("/:id/git/pull", handleGitPull)
	group.POST("/:id/git/push", handleGitPush)
	group.POST("/:id/git/sync", handleGitSync)
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
		c.JSON(http.StatusForbidden, gin.H{"error": "identity required to access workspace"})
		return false
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
	id := c.Param("id")
	if err := ws.ValidateID(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}

	m := mgr()
	if m == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace service not available"})
		return nil, false
	}

	w, err := m.Get(context.Background(), id)
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
	if owner == "" {
		response.RespondWithSuccess(c, http.StatusOK, []workspaceResponse{})
		return
	}

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
		switch n.Mode {
		case "local":
			kind = "host"
		case "cloud":
			kind = "cloud"
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

	if err := ws.ValidateID(req.ID); err != nil {
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

	data, info, err := mgr().ReadFile(context.Background(), c.Param("id"), path)
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

	modTime := time.Time{}
	if info != nil {
		modTime = info.Mtime
	}
	c.Header("Content-Type", mimeType)
	c.Header("Cache-Control", "no-cache")
	http.ServeContent(c.Writer, c.Request, path, modTime, bytes.NewReader(data))
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

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large, max 50MB"})
			return
		}
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

// --- Git request types ---

type gitAddRequest struct {
	RepoPath string   `json:"repo_path" binding:"required"`
	Files    []string `json:"files"`
}

type gitResetRequest struct {
	RepoPath string   `json:"repo_path" binding:"required"`
	Files    []string `json:"files"`
}

type gitCommitRequest struct {
	RepoPath    string `json:"repo_path" binding:"required"`
	Message     string `json:"message" binding:"required"`
	AuthorName  string `json:"author_name,omitempty"`
	AuthorEmail string `json:"author_email,omitempty"`
	AllowEmpty  bool   `json:"allow_empty,omitempty"`
}

type gitDiscardRequest struct {
	RepoPath string   `json:"repo_path" binding:"required"`
	Files    []string `json:"files"`
}

// --- Git handlers ---

func handleGitListRepos(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	refresh := c.Query("refresh") == "true"
	repos, err := mgr().GitListRepos(context.Background(), c.Param("id"), refresh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, repos)
}

func handleGitStatus(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	repoPath := c.Query("repo_path")
	if repoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo_path required"})
		return
	}
	result, err := mgr().GitStatus(context.Background(), c.Param("id"), repoPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleGitFileDiff(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	repoPath := c.Query("repo_path")
	filePath := c.Query("file_path")
	if repoPath == "" || filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo_path and file_path required"})
		return
	}
	staged := c.Query("staged") == "true"
	result, err := mgr().GitFileDiff(context.Background(), c.Param("id"), repoPath, filePath, staged)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleGitAdd(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitAdd(context.Background(), c.Param("id"), req.RepoPath, req.Files); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitReset(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitReset(context.Background(), c.Param("id"), req.RepoPath, req.Files); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitCommit(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := mgr().GitCommit(context.Background(), c.Param("id"), req.RepoPath, req.Message, req.AuthorName, req.AuthorEmail, req.AllowEmpty)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleGitDiscardChanges(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitDiscardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitDiscardChanges(context.Background(), c.Param("id"), req.RepoPath, req.Files); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

// --- Git Config request types ---

type gitConfigSetRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

type gitCredentialSetRequest struct {
	Host     string `json:"host" binding:"required"`
	Username string `json:"username,omitempty"`
	Token    string `json:"token" binding:"required"`
}

type gitSSHKeyImportRequest struct {
	Name       string `json:"name" binding:"required"`
	PrivateKey string `json:"private_key" binding:"required"`
	PublicKey  string `json:"public_key,omitempty"`
	Host       string `json:"host,omitempty"`
}

type gitFetchRequest struct {
	RepoPath string `json:"repo_path" binding:"required"`
	Remote   string `json:"remote,omitempty"`
}

type gitPullRequest struct {
	RepoPath string `json:"repo_path" binding:"required"`
	Remote   string `json:"remote,omitempty"`
	Rebase   bool   `json:"rebase,omitempty"`
}

type gitPushRequest struct {
	RepoPath    string `json:"repo_path" binding:"required"`
	Remote      string `json:"remote,omitempty"`
	Force       bool   `json:"force,omitempty"`
	SetUpstream bool   `json:"set_upstream,omitempty"`
}

type gitSyncRequest struct {
	RepoPath    string `json:"repo_path" binding:"required"`
	Remote      string `json:"remote,omitempty"`
	SetUpstream bool   `json:"set_upstream,omitempty"`
}

// --- Git Config handlers ---

func handleGitConfigGet(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	key := c.Query("key")
	values, err := mgr().GitConfigGet(context.Background(), c.Param("id"), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"values": values})
}

func handleGitConfigSet(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitConfigSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitConfigSet(context.Background(), c.Param("id"), req.Key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitCredentialSet(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitCredentialSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitCredentialSet(context.Background(), c.Param("id"), req.Host, req.Username, req.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitCredentialList(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	entries, err := mgr().GitCredentialList(context.Background(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, entries)
}

func handleGitCredentialDelete(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	host := c.Query("host")
	if host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "host required"})
		return
	}
	if err := mgr().GitCredentialDelete(context.Background(), c.Param("id"), host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitSSHKeyImport(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitSSHKeyImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitSSHKeyImport(context.Background(), c.Param("id"), req.Name, req.PrivateKey, req.PublicKey, req.Host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitSSHKeyList(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	keys, err := mgr().GitSSHKeyList(context.Background(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, keys)
}

func handleGitSSHKeyDelete(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	if err := mgr().GitSSHKeyDelete(context.Background(), c.Param("id"), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

// --- Git Remote Sync handlers ---

func handleGitFetch(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitFetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitFetch(context.Background(), c.Param("id"), req.RepoPath, req.Remote); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitPull(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitPullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := mgr().GitPull(context.Background(), c.Param("id"), req.RepoPath, req.Remote, req.Rebase)
	if err != nil {
		status := http.StatusInternalServerError
		resp := gin.H{"error": err.Error()}
		if result != nil {
			resp["has_conflicts"] = result.HasConflicts
			if result.HasConflicts {
				status = http.StatusConflict
			}
		}
		c.JSON(status, resp)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitPush(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mgr().GitPush(context.Background(), c.Param("id"), req.RepoPath, req.Remote, req.Force, req.SetUpstream); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"success": true})
}

func handleGitSync(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}
	var req gitSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	id := c.Param("id")
	result := gin.H{"fetched": false, "pulled": false, "pushed": false, "has_conflicts": false}

	fetchErr := mgr().GitFetch(ctx, id, req.RepoPath, req.Remote)
	if fetchErr != nil {
		errMsg := fetchErr.Error()
		isNonFatal := strings.Contains(errMsg, "empty") ||
			strings.Contains(errMsg, "up-to-date") ||
			strings.Contains(errMsg, "already up to date")
		if !isNonFatal {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch: " + errMsg})
			return
		}
	}
	result["fetched"] = fetchErr == nil

	status, err := mgr().GitStatus(ctx, id, req.RepoPath)
	if err != nil {
		response.RespondWithSuccess(c, http.StatusOK, result)
		return
	}

	if status.Behind > 0 {
		pullResult, pullErr := mgr().GitPull(ctx, id, req.RepoPath, req.Remote, false)
		if pullErr != nil {
			result["error"] = "pull: " + pullErr.Error()
			if pullResult != nil && pullResult.HasConflicts {
				result["has_conflicts"] = true
			}
			c.JSON(http.StatusConflict, result)
			return
		}
		result["pulled"] = true
	}

	setUpstream := req.SetUpstream || !status.HasUpstream
	if status.Ahead > 0 || setUpstream {
		if err := mgr().GitPush(ctx, id, req.RepoPath, req.Remote, false, setUpstream); err != nil {
			result["error"] = "push: " + err.Error()
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		result["pushed"] = true
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handlePreview serves workspace files with correct MIME types for browser rendering.
// Relative paths in HTML/CSS/JS resolve naturally since the URL structure preserves
// the file hierarchy: GET /:id/preview/app/index.html referencing ./style.css
// becomes GET /:id/preview/app/style.css.
func handlePreview(c *gin.Context) {
	_, ok := resolveAndCheckWS(c)
	if !ok {
		return
	}

	path := c.Param("path")
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	data, info, err := mgr().ReadFile(context.Background(), c.Param("id"), path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found: " + path})
		return
	}

	if int64(len(data)) > maxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large for preview, max 50MB"})
		return
	}

	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	modTime := time.Time{}
	if info != nil {
		modTime = info.Mtime
	}
	c.Header("Content-Type", mimeType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-cache")
	http.ServeContent(c.Writer, c.Request, path, modTime, bytes.NewReader(data))
}
