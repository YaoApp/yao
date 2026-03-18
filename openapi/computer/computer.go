package computer

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	sandboxv2 "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"

	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach registers computer option routes on the given group.
//   - GET /options — list available computers (filtered by ComputerFilter query params)
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {
	group.Use(oauth.Guard)
	group.GET("/options", handleOptions)
}

type computerSystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	NumCPU   int    `json:"num_cpu"`
	TotalMem int64  `json:"total_mem,omitempty"`
}

type computerOption struct {
	Kind        string             `json:"kind"`
	ID          string             `json:"id"`
	DisplayName string             `json:"display_name"`
	ContainerID string             `json:"container_id,omitempty"`
	NodeID      string             `json:"node_id"`
	Status      string             `json:"status"`
	Mode        string             `json:"mode,omitempty"`
	Addr        string             `json:"addr,omitempty"`
	Image       string             `json:"image,omitempty"`
	Policy      string             `json:"policy,omitempty"`
	VNC         bool               `json:"vnc"`
	System      computerSystemInfo `json:"system"`
}

func handleOptions(c *gin.Context) {
	authInfo := authorized.GetInfo(c)

	kindFilter := c.Query("kind")
	imageFilter := c.Query("image")
	osFilter := c.Query("os")
	archFilter := c.Query("arch")

	var vncFilter *bool
	if v := c.Query("vnc"); v != "" {
		b, _ := strconv.ParseBool(v)
		vncFilter = &b
	}

	var minCPUs float64
	if v := c.Query("min_cpus"); v != "" {
		minCPUs, _ = strconv.ParseFloat(v, 64)
	}

	var minMem int64
	if v := c.Query("min_mem"); v != "" {
		minMem = parseMemString(v)
	}

	var result []computerOption

	reg := registry.Global()
	if reg == nil {
		response.RespondWithSuccess(c, http.StatusOK, []computerOption{})
		return
	}

	snaps := reg.List()
	sort.Slice(snaps, func(i, j int) bool {
		return strings.ToLower(nodeDisplayName(snaps[i])) < strings.ToLower(nodeDisplayName(snaps[j]))
	})

	// Host entries: nodes with host_exec capability
	if kindFilter == "" || kindFilter == "host" {
		for i := range snaps {
			s := &snaps[i]
			if s.Mode != "local" && !nodeOwnedBy(s, authInfo) {
				continue
			}
			if !s.Capabilities.HostExec {
				continue
			}
			if !matchNodeFilter(s, osFilter, archFilter, minCPUs, minMem) {
				continue
			}
			result = append(result, nodeToHostOption(*s))
		}
	}

	// Node entries: nodes with container runtime capability
	if kindFilter == "" || kindFilter == "node" {
		for i := range snaps {
			s := &snaps[i]
			if s.Mode != "local" && !nodeOwnedBy(s, authInfo) {
				continue
			}
			hasRuntime := s.Capabilities.Docker || s.Capabilities.K8s
			if !hasRuntime {
				continue
			}
			if !matchNodeFilter(s, osFilter, archFilter, minCPUs, minMem) {
				continue
			}
			result = append(result, nodeToNodeOption(*s))
		}
	}

	// Box entries: persistent/longrunning boxes only
	if kindFilter == "" || kindFilter == "box" {
		if mgr := getManager(); mgr != nil {
			owner := resolveOwner(authInfo)
			boxes, err := mgr.List(context.Background(), sandboxv2.ListOptions{})
			if err == nil {
				for _, b := range boxes {
					snap := b.Snapshot()
					if snap.Owner != owner {
						continue
					}
					if snap.Policy != sandboxv2.Persistent && snap.Policy != sandboxv2.LongRunning {
						continue
					}
					if imageFilter != "" && snap.Image != imageFilter {
						continue
					}
					if vncFilter != nil && snap.VNC != *vncFilter {
						continue
					}
					result = append(result, boxToOption(b))
				}
			}
		}
	}

	if result == nil {
		result = []computerOption{}
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func matchNodeFilter(s *taitypes.NodeMeta, osFilter, archFilter string, minCPUs float64, minMem int64) bool {
	if osFilter != "" && !strings.EqualFold(s.System.OS, osFilter) {
		return false
	}
	if archFilter != "" && !strings.EqualFold(s.System.Arch, archFilter) {
		return false
	}
	if minCPUs > 0 && float64(s.System.NumCPU) < minCPUs {
		return false
	}
	if minMem > 0 && s.System.TotalMem < minMem {
		return false
	}
	return true
}

func nodeDisplayName(s taitypes.NodeMeta) string {
	if s.DisplayName != "" {
		return s.DisplayName
	}
	if s.System.Hostname != "" {
		return s.System.Hostname
	}
	return s.TaiID
}

func nodeToHostOption(s taitypes.NodeMeta) computerOption {
	displayName := nodeDisplayName(s)

	status := "stopped"
	if s.Status == "online" {
		status = "running"
	}

	addr := s.Addr
	if addr == "" {
		scheme := s.Mode
		if scheme == "" {
			scheme = "tai"
		}
		addr = scheme + "://" + s.TaiID
	}

	return computerOption{
		Kind:        "host",
		ID:          s.TaiID,
		DisplayName: displayName,
		NodeID:      s.TaiID,
		Status:      status,
		Mode:        s.Mode,
		Addr:        addr,
		VNC:         s.Capabilities.VNC,
		System: computerSystemInfo{
			OS:       s.System.OS,
			Arch:     s.System.Arch,
			Hostname: s.System.Hostname,
			NumCPU:   s.System.NumCPU,
			TotalMem: s.System.TotalMem,
		},
	}
}

func nodeToNodeOption(s taitypes.NodeMeta) computerOption {
	displayName := nodeDisplayName(s)

	status := "stopped"
	if s.Status == "online" {
		status = "running"
	}

	addr := s.Addr
	if addr == "" {
		scheme := s.Mode
		if scheme == "" {
			scheme = "tai"
		}
		addr = scheme + "://" + s.TaiID
	}

	return computerOption{
		Kind:        "node",
		ID:          s.TaiID,
		DisplayName: displayName,
		NodeID:      s.TaiID,
		Status:      status,
		Mode:        s.Mode,
		Addr:        addr,
		VNC:         s.Capabilities.VNC,
		System: computerSystemInfo{
			OS:       s.System.OS,
			Arch:     s.System.Arch,
			Hostname: s.System.Hostname,
			NumCPU:   s.System.NumCPU,
			TotalMem: s.System.TotalMem,
		},
	}
}

func boxToOption(b *sandboxv2.Box) computerOption {
	snap := b.Snapshot()
	info := b.ComputerInfo()

	displayName := info.DisplayName
	if displayName == "" {
		displayName = info.System.Hostname
	}
	if displayName == "" {
		displayName = snap.ID
	}

	var mode, addr string
	if ns, ok := tai.GetNodeMeta(snap.NodeID); ok {
		mode = ns.Mode
		addr = ns.Addr
	}
	if addr == "" && snap.NodeID != "" {
		scheme := mode
		if scheme == "" {
			scheme = "local"
		}
		addr = scheme + "://" + snap.NodeID
	}

	return computerOption{
		Kind:        "box",
		ID:          snap.ID,
		DisplayName: displayName,
		ContainerID: snap.ContainerID,
		NodeID:      snap.NodeID,
		Status:      snap.Status,
		Mode:        mode,
		Addr:        addr,
		Image:       snap.Image,
		Policy:      string(snap.Policy),
		VNC:         snap.VNC,
		System: computerSystemInfo{
			OS:       info.System.OS,
			Arch:     info.System.Arch,
			Hostname: info.System.Hostname,
			NumCPU:   info.System.NumCPU,
			TotalMem: info.System.TotalMem,
		},
	}
}

func nodeOwnedBy(snap *taitypes.NodeMeta, authInfo *oauthTypes.AuthorizedInfo) bool {
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

func resolveOwner(authInfo *oauthTypes.AuthorizedInfo) string {
	if authInfo != nil && authInfo.TeamID != "" {
		return authInfo.TeamID
	}
	if authInfo != nil {
		return authInfo.UserID
	}
	return ""
}

func getManager() *sandboxv2.Manager {
	defer func() { recover() }()
	return sandboxv2.M()
}

func parseMemString(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}

	multiplier := int64(1)
	switch {
	case strings.HasSuffix(s, "g"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "m"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "k"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "k")
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(val * float64(multiplier))
}
