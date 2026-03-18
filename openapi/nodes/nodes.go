package nodes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"
)

// Attach registers Tai node endpoints on the given group.
//   - GET /        — list nodes (filtered by team/user from token)
//   - GET /:id     — get single node (owner check)
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	group.Use(oauth.Guard)
	group.GET("", handleList)
	group.GET("/:id", handleGet)
}

type nodeResponse struct {
	TaiID        string          `json:"tai_id"`
	MachineID    string          `json:"machine_id,omitempty"`
	Version      string          `json:"version,omitempty"`
	DisplayName  string          `json:"display_name,omitempty"`
	Mode         string          `json:"mode"`
	Addr         string          `json:"addr,omitempty"`
	Status       string          `json:"status"`
	System       systemResponse  `json:"system"`
	Capabilities map[string]bool `json:"capabilities,omitempty"`
	Ports        map[string]int  `json:"ports,omitempty"`
	ConnectedAt  *time.Time      `json:"connected_at,omitempty"`
	LastPing     *time.Time      `json:"last_ping,omitempty"`
}

type systemResponse struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	NumCPU   int    `json:"num_cpu"`
	TotalMem int64  `json:"total_mem,omitempty"`
	Shell    string `json:"shell,omitempty"`
}

func snapToResponse(s taitypes.NodeMeta) nodeResponse {
	r := nodeResponse{
		TaiID:        s.TaiID,
		MachineID:    s.MachineID,
		Version:      s.Version,
		DisplayName:  s.DisplayName,
		Mode:         s.Mode,
		Addr:         s.Addr,
		Status:       s.Status,
		Capabilities: map[string]bool{"docker": s.Capabilities.Docker, "k8s": s.Capabilities.K8s, "host_exec": s.Capabilities.HostExec},
		Ports:        map[string]int{"grpc": s.Ports.GRPC, "http": s.Ports.HTTP, "vnc": s.Ports.VNC, "docker": s.Ports.Docker, "k8s": s.Ports.K8s},
		System: systemResponse{
			OS:       s.System.OS,
			Arch:     s.System.Arch,
			Hostname: s.System.Hostname,
			NumCPU:   s.System.NumCPU,
			TotalMem: s.System.TotalMem,
			Shell:    s.System.Shell,
		},
	}
	if !s.ConnectedAt.IsZero() {
		r.ConnectedAt = &s.ConnectedAt
	}
	if !s.LastPing.IsZero() {
		r.LastPing = &s.LastPing
	}
	return r
}

// nodeOwnedBy checks whether a node belongs to the caller.
// TeamID match → true; no team and UserID match → true.
func nodeOwnedBy(snap *taitypes.NodeMeta, authInfo *types.AuthorizedInfo) bool {
	if authInfo == nil {
		return true
	}

	if authInfo.TeamID != "" {
		return snap.Auth.TeamID == authInfo.TeamID
	}
	if authInfo.UserID != "" {
		return snap.Auth.TeamID == "" && snap.Auth.UserID == authInfo.UserID
	}
	return true
}

func handleList(c *gin.Context) {
	reg := registry.Global()
	if reg == nil {
		response.RespondWithSuccess(c, http.StatusOK, []nodeResponse{})
		return
	}

	authInfo := authorized.GetInfo(c)
	snaps := reg.List()

	result := make([]nodeResponse, 0, len(snaps))
	for i := range snaps {
		s := &snaps[i]
		if s.Mode != "local" && !nodeOwnedBy(s, authInfo) {
			continue
		}
		result = append(result, snapToResponse(*s))
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func handleGet(c *gin.Context) {
	reg := registry.Global()
	if reg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "node registry not available"})
		return
	}

	id := c.Param("id")
	snap, ok := reg.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	authInfo := authorized.GetInfo(c)
	if snap.Mode != "local" && !nodeOwnedBy(snap, authInfo) {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to access this node"})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, snapToResponse(*snap))
}
