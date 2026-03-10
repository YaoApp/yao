package sandbox

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	sandboxv2 "github.com/yaoapp/yao/sandbox/v2"
)

// AttachManage registers sandbox management CRUD routes on the given group.
// oauth.Guard is already applied by the parent Attach call on the same group.
//   - GET    /              — list sandboxes (filtered by owner)
//   - POST   /              — create sandbox (owner from token)
//   - GET    /:id           — get sandbox (owner check)
//   - DELETE /:id           — remove sandbox (owner check)
//   - POST   /:id/exec     — execute command (owner check)
//   - POST   /:id/heartbeat — heartbeat (owner check)
func AttachManage(group *gin.RouterGroup) {
	group.GET("", handleList)
	group.POST("", handleCreate)
	group.GET("/:id", handleGet)
	group.DELETE("/:id", handleRemove)
	group.POST("/:id/exec", handleExec)
	group.POST("/:id/heartbeat", handleHeartbeat)
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

// --- request / response types ---

type createSandboxRequest struct {
	ID          string            `json:"id,omitempty"`
	NodeID      string            `json:"node_id"`
	Image       string            `json:"image"`
	WorkDir     string            `json:"work_dir,omitempty"`
	User        string            `json:"user,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Memory      int64             `json:"memory,omitempty"`
	CPUs        float64           `json:"cpus,omitempty"`
	VNC         bool              `json:"vnc,omitempty"`
	Policy      string            `json:"policy,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	MountMode   string            `json:"mount_mode,omitempty"`
	MountPath   string            `json:"mount_path,omitempty"`
}

type execRequest struct {
	Cmd     []string          `json:"cmd" binding:"required"`
	WorkDir string            `json:"work_dir,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

type heartbeatRequest struct {
	Active       bool `json:"active"`
	ProcessCount int  `json:"process_count"`
}

type sandboxSystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	NumCPU   int    `json:"num_cpu"`
	TotalMem int64  `json:"total_mem,omitempty"`
	Shell    string `json:"shell,omitempty"`
	TempDir  string `json:"temp_dir,omitempty"`
}

type sandboxResponse struct {
	ID           string            `json:"id"`
	ContainerID  string            `json:"container_id"`
	NodeID       string            `json:"node_id"`
	Owner        string            `json:"owner"`
	Status       string            `json:"status"`
	Policy       string            `json:"policy"`
	Labels       map[string]string `json:"labels,omitempty"`
	Image        string            `json:"image"`
	VNC          bool              `json:"vnc"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActive   time.Time         `json:"last_active"`
	ProcessCount int               `json:"process_count"`
	System       sandboxSystemInfo `json:"system"`
	WorkspaceID  string            `json:"workspace_id,omitempty"`
}

func boxToResponse(b *sandboxv2.Box) sandboxResponse {
	snap := b.Snapshot()
	info := b.ComputerInfo()
	return sandboxResponse{
		ID:           snap.ID,
		ContainerID:  snap.ContainerID,
		NodeID:       snap.NodeID,
		Owner:        snap.Owner,
		Status:       snap.Status,
		Policy:       string(snap.Policy),
		Labels:       snap.Labels,
		Image:        snap.Image,
		VNC:          snap.VNC,
		CreatedAt:    snap.CreatedAt,
		LastActive:   snap.LastActive,
		ProcessCount: snap.ProcessCount,
		WorkspaceID:  b.WorkspaceID(),
		System: sandboxSystemInfo{
			OS:       info.System.OS,
			Arch:     info.System.Arch,
			Hostname: info.System.Hostname,
			NumCPU:   info.System.NumCPU,
			TotalMem: info.System.TotalMem,
			Shell:    info.System.Shell,
			TempDir:  info.System.TempDir,
		},
	}
}

func getManager(c *gin.Context) *sandboxv2.Manager {
	defer func() { recover() }()
	return sandboxv2.M()
}

// checkBoxOwner verifies the caller owns the sandbox.
func checkBoxOwner(c *gin.Context, box *sandboxv2.Box, owner string) bool {
	if owner == "" {
		return true
	}
	info := box.ComputerInfo()
	if info.Owner != "" && info.Owner != owner {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to access this sandbox"})
		return false
	}
	return true
}

// --- handlers ---

func handleList(c *gin.Context) {
	mgr := getManager(c)
	if mgr == nil {
		response.RespondWithSuccess(c, http.StatusOK, []sandboxResponse{})
		return
	}

	authInfo := authorized.GetInfo(c)
	owner := resolveOwner(authInfo)

	boxes, err := mgr.List(context.Background(), sandboxv2.ListOptions{
		Owner:  owner,
		NodeID: c.Query("node_id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]sandboxResponse, 0, len(boxes))
	for _, b := range boxes {
		result = append(result, boxToResponse(b))
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleCreate(c *gin.Context) {
	mgr := getManager(c)
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sandbox service not available"})
		return
	}

	var req createSandboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image is required"})
		return
	}

	authInfo := authorized.GetInfo(c)
	owner := resolveOwner(authInfo)

	opts := sandboxv2.CreateOptions{
		ID:          req.ID,
		Owner:       owner,
		NodeID:      req.NodeID,
		Image:       req.Image,
		WorkDir:     req.WorkDir,
		User:        req.User,
		Env:         req.Env,
		Memory:      req.Memory,
		CPUs:        req.CPUs,
		VNC:         req.VNC,
		Policy:      sandboxv2.LifecyclePolicy(req.Policy),
		Labels:      req.Labels,
		WorkspaceID: req.WorkspaceID,
		MountMode:   req.MountMode,
		MountPath:   req.MountPath,
	}

	box, err := mgr.Create(context.Background(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.RespondWithSuccess(c, http.StatusCreated, boxToResponse(box))
}

func handleGet(c *gin.Context) {
	mgr := getManager(c)
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sandbox service not available"})
		return
	}

	id := c.Param("id")
	box, err := mgr.Get(context.Background(), id)
	if err != nil {
		if err == sandboxv2.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "sandbox not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	authInfo := authorized.GetInfo(c)
	if !checkBoxOwner(c, box, resolveOwner(authInfo)) {
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, boxToResponse(box))
}

func handleRemove(c *gin.Context) {
	mgr := getManager(c)
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sandbox service not available"})
		return
	}

	id := c.Param("id")
	box, err := mgr.Get(context.Background(), id)
	if err != nil {
		if err == sandboxv2.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "sandbox not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	authInfo := authorized.GetInfo(c)
	if !checkBoxOwner(c, box, resolveOwner(authInfo)) {
		return
	}

	if err := mgr.Remove(context.Background(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func handleExec(c *gin.Context) {
	mgr := getManager(c)
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sandbox service not available"})
		return
	}

	id := c.Param("id")
	box, err := mgr.Get(context.Background(), id)
	if err != nil {
		if err == sandboxv2.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "sandbox not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	authInfo := authorized.GetInfo(c)
	if !checkBoxOwner(c, box, resolveOwner(authInfo)) {
		return
	}

	var req execRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var opts []sandboxv2.ExecOption
	if req.WorkDir != "" {
		opts = append(opts, sandboxv2.WithWorkDir(req.WorkDir))
	}
	if len(req.Env) > 0 {
		opts = append(opts, sandboxv2.WithEnv(req.Env))
	}
	if req.Timeout > 0 {
		opts = append(opts, sandboxv2.WithTimeout(time.Duration(req.Timeout)*time.Second))
	}

	result, err := box.Exec(context.Background(), req.Cmd, opts...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleHeartbeat(c *gin.Context) {
	mgr := getManager(c)
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sandbox service not available"})
		return
	}

	id := c.Param("id")

	box, err := mgr.Get(context.Background(), id)
	if err != nil {
		if err == sandboxv2.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "sandbox not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	authInfo := authorized.GetInfo(c)
	if !checkBoxOwner(c, box, resolveOwner(authInfo)) {
		return
	}

	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mgr.Heartbeat(id, req.Active, req.ProcessCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
