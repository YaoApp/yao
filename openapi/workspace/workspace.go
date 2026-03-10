package workspace

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
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
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	group.Use(oauth.Guard)

	group.GET("", handleList)
	group.POST("", handleCreate)
	group.GET("/:id", handleGet)
	group.PUT("/:id", handleUpdate)
	group.DELETE("/:id", handleDelete)

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
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Owner     string            `json:"owner"`
	Node      string            `json:"node"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
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
	response.RespondWithSuccess(c, http.StatusOK, result)
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

	fmt.Printf("[workspace] handleReadFile id=%s path=%q\n", c.Param("id"), path)

	data, err := mgr().ReadFile(context.Background(), c.Param("id"), path)
	if err != nil {
		fmt.Printf("[workspace] ReadFile error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[workspace] ReadFile ok, size=%d, encoding=%q\n", len(data), c.Query("encoding"))

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
	fmt.Printf("[workspace] serving ext=%q mime=%q size=%d\n", ext, mimeType, len(data))
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
