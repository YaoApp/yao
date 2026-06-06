package tai

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	sandboxTypes "github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/webproxy"
)

// bindRequest is the JSON body for creating bindings.
type bindRequest struct {
	TargetID string         `json:"target_id" binding:"required"`
	Port     int            `json:"port,omitempty"`
	Label    string         `json:"label,omitempty"`
	Services []serviceEntry `json:"services,omitempty"`
}

type serviceEntry struct {
	Label string `json:"label"`
	Port  int    `json:"port"`
}

// --- New grouped response types ---

type bindingsGroupedResponse struct {
	Domain  string        `json:"domain"`
	Prefix  string        `json:"prefix"`
	Targets []targetGroup `json:"targets"`
}

type targetGroup struct {
	TargetID   string            `json:"target_id"`
	Kind       string            `json:"kind"`
	Assistants []assistantGroup  `json:"assistants"`
	Temporary  []temporaryStatus `json:"temporary"`
}

type assistantGroup struct {
	AssistantID string          `json:"assistant_id"`
	Name        string          `json:"name"`
	Services    []serviceStatus `json:"services"`
}

type serviceStatus struct {
	Label    string `json:"label"`
	Port     int    `json:"port"`
	Bound    bool   `json:"bound"`
	HostPort int    `json:"host_port,omitempty"`
	Status   string `json:"status,omitempty"`
}

type temporaryStatus struct {
	HostPort   int    `json:"host_port"`
	TargetPort int    `json:"target_port"`
	Label      string `json:"label"`
	Status     string `json:"status"`
}

// handleListBindings returns grouped services + bindings.
// With target_id: returns data for a single target.
// Without target_id: query-all, returns all targets grouped by assistant.
func handleListBindings(c *gin.Context) {
	if webproxy.WP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webproxy service not enabled"})
		return
	}
	targetID := c.Query("target_id")
	if targetID != "" {
		c.JSON(http.StatusOK, buildGroupedResponse(c, targetID))
	} else {
		c.JSON(http.StatusOK, buildAllResponse(c))
	}
}

// handleCreateBindings creates one or more bindings, returns full updated state.
func handleCreateBindings(c *gin.Context) {
	if webproxy.WP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webproxy service not enabled"})
		return
	}
	taiID := c.Param("taiID")

	var req bindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	containerID, err := resolveContainerID(c.Request.Context(), req.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Port > 0 {
		_, err := webproxy.WP().Bind(webproxy.BindOptions{
			TaiID:       taiID,
			TargetID:    req.TargetID,
			ContainerID: containerID,
			TargetPort:  req.Port,
			Label:       req.Label,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if len(req.Services) > 0 {
		for _, svc := range req.Services {
			webproxy.WP().Bind(webproxy.BindOptions{
				TaiID:       taiID,
				TargetID:    req.TargetID,
				ContainerID: containerID,
				TargetPort:  svc.Port,
				Label:       svc.Label,
			})
		}
	} else if req.TargetID != webproxy.HostID {
		// Auto-resolve only for Box mode (single agent per box).
		// Host mode must provide explicit services.
		services := resolveBoxServices(c, req.TargetID)
		for _, svc := range services {
			webproxy.WP().Bind(webproxy.BindOptions{
				TaiID:       taiID,
				TargetID:    req.TargetID,
				ContainerID: containerID,
				TargetPort:  svc.Port,
				Label:       svc.Label,
			})
		}
	}

	c.JSON(http.StatusOK, buildGroupedResponse(c, req.TargetID))
}

// handleDeleteBinding removes a binding and returns full updated state.
func handleDeleteBinding(c *gin.Context) {
	if webproxy.WP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webproxy service not enabled"})
		return
	}
	portStr := c.Param("hostPort")
	hostPort, err := strconv.Atoi(portStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host port"})
		return
	}

	if err := webproxy.WP().Unbind(hostPort); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	targetID := c.Query("target_id")
	if targetID != "" {
		c.JSON(http.StatusOK, buildGroupedResponse(c, targetID))
	} else {
		c.JSON(http.StatusOK, buildAllResponse(c))
	}
}

// buildGroupedResponse returns the grouped response for a single target.
func buildGroupedResponse(c *gin.Context, targetID string) *bindingsGroupedResponse {
	domain, prefix := webproxy.WP().GetConfig()
	tg := buildTargetGroup(c, targetID)
	return &bindingsGroupedResponse{
		Domain:  domain,
		Prefix:  prefix,
		Targets: []targetGroup{tg},
	}
}

// buildAllResponse returns the grouped response for all targets.
// Logic 1 (what services exist): from assistant cache, filter IsSandbox.
// Logic 2 (what is connected): from webproxy active bindings.
func buildAllResponse(c *gin.Context) *bindingsGroupedResponse {
	domain, prefix := webproxy.WP().GetConfig()

	allBindings := webproxy.WP().List("")
	targetIDs := collectTargetIDs(allBindings)

	// Always include host target
	hasHost := false
	for _, tid := range targetIDs {
		if tid == webproxy.HostID {
			hasHost = true
			break
		}
	}
	if !hasHost {
		targetIDs = append([]string{webproxy.HostID}, targetIDs...)
	}

	targets := make([]targetGroup, 0, len(targetIDs))
	for _, tid := range targetIDs {
		tg := buildTargetGroup(c, tid)
		if len(tg.Assistants) > 0 || len(tg.Temporary) > 0 {
			targets = append(targets, tg)
		}
	}

	return &bindingsGroupedResponse{
		Domain:  domain,
		Prefix:  prefix,
		Targets: targets,
	}
}

// buildTargetGroup builds the full grouped data for a single target.
func buildTargetGroup(c *gin.Context, targetID string) targetGroup {
	bindings := webproxy.WP().List(targetID)
	bindingByPort := indexByTargetPort(bindings)

	var assistants []assistantGroup
	var kind string

	if targetID == webproxy.HostID {
		kind = "host"
		assistants = resolveHostAssistants(c, bindingByPort)
	} else {
		kind = "box"
		assistants = resolveBoxAssistants(c, targetID, bindingByPort)
	}

	// Temporary: bindings whose port is NOT in any configured service
	configuredPorts := collectConfiguredPorts(assistants)
	temporary := make([]temporaryStatus, 0)
	for i := range bindings {
		if !configuredPorts[bindings[i].TargetPort] {
			temporary = append(temporary, temporaryStatus{
				HostPort:   bindings[i].HostPort,
				TargetPort: bindings[i].TargetPort,
				Label:      bindings[i].Label,
				Status:     bindings[i].Status,
			})
		}
	}

	return targetGroup{
		TargetID:   targetID,
		Kind:       kind,
		Assistants: assistants,
		Temporary:  temporary,
	}
}

// resolveHostAssistants returns grouped services for all sandbox agents (Host mode).
func resolveHostAssistants(c *gin.Context, bindingByPort map[int]*webproxy.BindingInfo) []assistantGroup {
	cache := assistant.GetCache()
	if cache == nil {
		return nil
	}

	allAssistants := cache.All()
	var ids []string
	var sandboxAgents []*assistant.Assistant
	for _, ast := range allAssistants {
		if !ast.IsSandbox {
			continue
		}
		ids = append(ids, ast.ID)
		sandboxAgents = append(sandboxAgents, ast)
	}
	if len(ids) == 0 {
		return nil
	}

	info := authorized.GetInfo(c)
	userID, teamID := "", ""
	if info != nil {
		userID, teamID = info.UserID, info.TeamID
	}
	allServices := assistant.ResolveServicesBatch(ids, userID, teamID)

	locale := getLocale(c)
	var groups []assistantGroup
	for _, ast := range sandboxAgents {
		svcs := allServices[ast.ID]
		if len(svcs) == 0 {
			continue
		}
		groups = append(groups, assistantGroup{
			AssistantID: ast.ID,
			Name:        ast.GetName(locale),
			Services:    matchBindings(svcs, bindingByPort),
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	return groups
}

// resolveBoxAssistants returns grouped services for a single Box target.
func resolveBoxAssistants(c *gin.Context, targetID string, bindingByPort map[int]*webproxy.BindingInfo) []assistantGroup {
	assistantID := ""
	if mgr := sandbox.M(); mgr != nil {
		if box, err := mgr.Get(c.Request.Context(), targetID); err == nil && box != nil {
			assistantID = box.Label("sandbox-assistant")
		}
	}
	if assistantID == "" {
		return nil
	}

	info := authorized.GetInfo(c)
	userID, teamID := "", ""
	if info != nil {
		userID, teamID = info.UserID, info.TeamID
	}
	allServices := assistant.ResolveServicesBatch([]string{assistantID}, userID, teamID)
	svcs := allServices[assistantID]
	if len(svcs) == 0 {
		return nil
	}

	locale := getLocale(c)
	name := assistantID
	if cache := assistant.GetCache(); cache != nil {
		if ast, ok := cache.Get(assistantID); ok {
			name = ast.GetName(locale)
		}
	}

	return []assistantGroup{{
		AssistantID: assistantID,
		Name:        name,
		Services:    matchBindings(svcs, bindingByPort),
	}}
}

// resolveBoxServices resolves services for a Box target (used in auto-connect).
func resolveBoxServices(c *gin.Context, targetID string) []serviceEntry {
	assistantID := ""
	if mgr := sandbox.M(); mgr != nil {
		if box, err := mgr.Get(c.Request.Context(), targetID); err == nil && box != nil {
			assistantID = box.Label("sandbox-assistant")
		}
	}
	if assistantID == "" {
		return nil
	}

	info := authorized.GetInfo(c)
	userID, teamID := "", ""
	if info != nil {
		userID, teamID = info.UserID, info.TeamID
	}
	allServices := assistant.ResolveServicesBatch([]string{assistantID}, userID, teamID)
	svcs := allServices[assistantID]

	entries := make([]serviceEntry, 0, len(svcs))
	for _, svc := range svcs {
		if svc.Port > 0 {
			entries = append(entries, serviceEntry{Label: svc.Label, Port: svc.Port})
		}
	}
	return entries
}

// resolveContainerID converts a target_id (sandbox ID or __host__) to a Docker container ID.
func resolveContainerID(ctx context.Context, targetID string) (string, error) {
	if targetID == webproxy.HostID {
		return "", nil
	}
	box, err := sandbox.M().Get(ctx, targetID)
	if err != nil {
		return "", err
	}
	return box.ContainerID(), nil
}

// --- helpers ---

func matchBindings(svcs []sandboxTypes.ServiceConfig, bindingByPort map[int]*webproxy.BindingInfo) []serviceStatus {
	result := make([]serviceStatus, 0, len(svcs))
	for _, svc := range svcs {
		ss := serviceStatus{Label: svc.Label, Port: svc.Port}
		if b, ok := bindingByPort[svc.Port]; ok {
			ss.Bound = true
			ss.HostPort = b.HostPort
			ss.Status = b.Status
		}
		result = append(result, ss)
	}
	return result
}

func indexByTargetPort(bindings []webproxy.BindingInfo) map[int]*webproxy.BindingInfo {
	m := make(map[int]*webproxy.BindingInfo, len(bindings))
	for i := range bindings {
		m[bindings[i].TargetPort] = &bindings[i]
	}
	return m
}

func collectTargetIDs(bindings []webproxy.BindingInfo) []string {
	seen := make(map[string]bool)
	var ids []string
	for i := range bindings {
		tid := bindings[i].TargetID
		if !seen[tid] {
			seen[tid] = true
			ids = append(ids, tid)
		}
	}
	return ids
}

func collectConfiguredPorts(groups []assistantGroup) map[int]bool {
	ports := make(map[int]bool)
	for _, g := range groups {
		for _, s := range g.Services {
			ports[s.Port] = true
		}
	}
	return ports
}

func getLocale(c *gin.Context) string {
	if l := c.Query("locale"); l != "" {
		return strings.ToLower(l)
	}
	if al := c.GetHeader("Accept-Language"); al != "" {
		parts := strings.Split(al, ",")
		if len(parts) > 0 {
			return strings.ToLower(strings.TrimSpace(strings.Split(parts[0], ";")[0]))
		}
	}
	return "en-us"
}
