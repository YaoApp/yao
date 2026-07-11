package setting

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"
)

// handleSetupStatus aggregates all system-level configuration checkpoints.
// GET /setting/setup-status
func handleSetupStatus(c *gin.Context) {
	info := authorized.GetInfo(c)
	locale := strings.ToLower(c.DefaultQuery("locale", "en-us"))
	isCN := strings.HasPrefix(locale, "zh")

	checkpoints := make(map[string]Checkpoint, 6)

	checkpoints["llm_default"] = checkLLMDefault(info, isCN)
	checkpoints["llm_vision"] = checkLLMVision(info, isCN)
	checkpoints["sandbox_node"] = checkSandboxNode(info, isCN)
	checkpoints["sandbox_image"] = checkSandboxImage(info, locale, isCN)
	checkpoints["search"] = checkSearch(info, isCN)
	checkpoints["smtp"] = checkSMTP(info, isCN)

	completed := true
	for _, cp := range checkpoints {
		if cp.Required && cp.Status == "fail" {
			completed = false
			break
		}
	}

	bannerDismissed := false
	onboardingCompleted := false
	if setting.Global != nil {
		// Read user-scope only: these are personal preferences that must not
		// inherit from system/team scopes.
		prefs, _ := setting.Global.Get(preferenceScope(info), preferenceNS)
		if prefs != nil {
			if v, ok := prefs["banner_dismissed"].(bool); ok {
				bannerDismissed = v
			}
			if v, ok := prefs["onboarding_completed"].(bool); ok {
				onboardingCompleted = v
			}
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, SetupStatus{
		Completed:           completed,
		Checkpoints:         checkpoints,
		OnboardingCompleted: onboardingCompleted,
		BannerDismissed:     bannerDismissed,
	})
}

// handleAssistantSetupStatus checks configuration readiness for a specific assistant.
// GET /setting/setup-status/assistant/:id
func handleAssistantSetupStatus(c *gin.Context) {
	id := c.Param("id")
	info := authorized.GetInfo(c)
	locale := strings.ToLower(c.DefaultQuery("locale", "en-us"))
	isCN := strings.HasPrefix(locale, "zh")

	cache := assistant.GetCache()
	var ast *assistant.Assistant
	if cache != nil {
		ast, _ = cache.Get(id)
	}
	if ast == nil {
		var err error
		ast, err = assistant.Get(id)
		if err != nil || ast == nil {
			respondError(c, http.StatusNotFound, fmt.Sprintf("assistant %q not found", id))
			return
		}
	}

	checkpoints := make(map[string]Checkpoint)
	allReady := true

	// connector check
	cp := checkAssistantConnector(ast, info, isCN)
	checkpoints["connector"] = cp
	if cp.Status == "fail" {
		allReady = false
	}

	// sandbox check (only if V2 sandbox configured)
	if ast.HasSandboxV2() {
		cp := checkAssistantSandbox(ast, info, locale, isCN)
		checkpoints["sandbox_ready"] = cp
		if cp.Status == "fail" {
			allReady = false
		}
	}

	// search check (only if uses.search is configured and not disabled)
	if ast.Uses != nil && ast.Uses.Search != "" && ast.Uses.Search != "disabled" {
		cp := checkAssistantSearch(ast, info, isCN)
		checkpoints["search"] = cp
		if cp.Status == "fail" {
			allReady = false
		}
	}

	name := ast.GetName(locale)
	if name == "" {
		name = ast.ID
	}

	response.RespondWithSuccess(c, http.StatusOK, AssistantSetupStatus{
		AssistantID:   id,
		AssistantName: name,
		Ready:         allReady,
		Checkpoints:   checkpoints,
	})
}

// ---------------------------------------------------------------------------
// System-level checkpoint helpers
// ---------------------------------------------------------------------------

// parseRoleTarget extracts provider key and model ID from a role value.
// Supports both map format {"provider":"x","model":"y"} and legacy string "provider::model".
func parseRoleTarget(val interface{}) (providerKey, modelID string) {
	switch v := val.(type) {
	case map[string]interface{}:
		providerKey, _ = v["provider"].(string)
		modelID, _ = v["model"].(string)
	case string:
		if v == "" {
			return
		}
		parts := strings.SplitN(v, "::", 2)
		providerKey = parts[0]
		if len(parts) == 2 {
			modelID = parts[1]
		}
	}
	return
}

func checkLLMDefault(info *oauthTypes.AuthorizedInfo, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: true,
		Label:    "Default Model",
		Path:     "/settings/models",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "默认模型"
	}

	if setting.Global == nil || llmprovider.Global == nil {
		return cp
	}

	roles, _ := setting.Global.GetMerged(info.UserID, info.TeamID, llmprovider.RolesNamespace)
	if roles == nil {
		return cp
	}

	providerKey, _ := parseRoleTarget(roles["default"])
	if providerKey == "" {
		return cp
	}

	p, err := llmprovider.Global.Get(providerKey)
	if err != nil || p == nil || !p.Enabled {
		return cp
	}

	cp.Status = "pass"
	return cp
}

func checkLLMVision(info *oauthTypes.AuthorizedInfo, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: false,
		Label:    "Vision Model",
		Path:     "/settings/models",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "视觉模型"
	}

	if setting.Global == nil || llmprovider.Global == nil {
		return cp
	}

	roles, _ := setting.Global.GetMerged(info.UserID, info.TeamID, llmprovider.RolesNamespace)
	if roles == nil {
		return cp
	}

	// Check dedicated vision role first
	if providerKey, modelID := parseRoleTarget(roles["vision"]); providerKey != "" {
		if p, err := llmprovider.Global.Get(providerKey); err == nil && p != nil && p.Enabled {
			for _, m := range p.Models {
				if m.Enabled && hasCapability(m.Capabilities, "vision") {
					if modelID == "" || m.ID == modelID {
						cp.Status = "pass"
						return cp
					}
				}
			}
		}
	}

	// Fallback: check if default role has vision capability
	if providerKey, _ := parseRoleTarget(roles["default"]); providerKey != "" {
		if p, err := llmprovider.Global.Get(providerKey); err == nil && p != nil && p.Enabled {
			for _, m := range p.Models {
				if m.Enabled && hasCapability(m.Capabilities, "vision") {
					cp.Status = "pass"
					return cp
				}
			}
		}
	}

	return cp
}

func checkSandboxNode(info *oauthTypes.AuthorizedInfo, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: true,
		Label:    "Sandbox Node",
		Path:     "/settings/sandbox",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "沙箱节点"
	}

	reg := registry.Global()
	if reg == nil {
		return cp
	}

	for _, snap := range reg.List() {
		if !taitypes.IsPublicNode(snap.Mode) && !sandboxNodeOwnedBy(&snap, info) {
			continue
		}
		if snap.Status == "online" && (snap.Capabilities.Docker || snap.Capabilities.HostExec) {
			cp.Status = "pass"
			return cp
		}
	}

	return cp
}

func checkSandboxImage(info *oauthTypes.AuthorizedInfo, locale string, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: true,
		Label:    "Sandbox Images",
		Path:     "/settings/sandbox",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "沙箱镜像"
	}

	needed := collectAssistantImages(locale)
	if len(needed) == 0 {
		cp.Status = "pass"
		if isCN {
			cp.Detail = "无需镜像"
		} else {
			cp.Detail = "No images needed"
		}
		return cp
	}

	reg := registry.Global()
	if reg == nil {
		cp.Detail = fmt.Sprintf("0/%d", len(needed))
		return cp
	}

	downloaded := 0
	for _, snap := range reg.List() {
		if !taitypes.IsPublicNode(snap.Mode) && !sandboxNodeOwnedBy(&snap, info) {
			continue
		}
		if snap.Status != "online" || !snap.Capabilities.Docker {
			continue
		}

		res, ok := tai.GetResources(snap.TaiID)
		if !ok || res.Image == nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		images, err := res.Image.List(ctx)
		cancel()
		if err != nil {
			continue
		}

		tagIndex := make(map[string]bool)
		for _, img := range images {
			for _, tag := range img.Tags {
				tagIndex[tag] = true
			}
		}

		for imageRef := range needed {
			if tagIndex[imageRef] {
				downloaded++
			}
		}
		break // only check the first usable node
	}

	if isCN {
		cp.Detail = fmt.Sprintf("%d/%d 镜像已下载", downloaded, len(needed))
	} else {
		cp.Detail = fmt.Sprintf("%d/%d images downloaded", downloaded, len(needed))
	}
	if downloaded > 0 {
		cp.Status = "pass"
	}
	return cp
}

func checkSearch(info *oauthTypes.AuthorizedInfo, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: false,
		Label:    "Search Provider",
		Path:     "/settings/search",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "搜索服务"
	}

	if setting.Global == nil {
		return cp
	}

	for _, preset := range searchPresets {
		if preset.IsCloud {
			saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, cloudNS)
			if saved != nil {
				if v, ok := saved["status"].(string); ok && v == "connected" {
					cp.Status = "pass"
					return cp
				}
			}
		} else {
			saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, searchProviderNS(preset.Key))
			if saved != nil {
				if v, ok := saved["status"].(string); ok && v == "connected" {
					cp.Status = "pass"
					return cp
				}
			}
		}
	}

	return cp
}

func checkSMTP(info *oauthTypes.AuthorizedInfo, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: false,
		Label:    "SMTP Email",
		Path:     "/settings/smtp",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "邮件服务"
	}

	if setting.Global == nil {
		return cp
	}

	saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, smtpNS)
	if saved == nil {
		return cp
	}

	status, _ := saved["status"].(string)
	if status == "connected" {
		if enabled, ok := saved["enabled"].(bool); ok && !enabled {
			return cp
		}
		cp.Status = "pass"
	}

	return cp
}

// ---------------------------------------------------------------------------
// Assistant-level checkpoint helpers
// ---------------------------------------------------------------------------

func checkAssistantConnector(ast *assistant.Assistant, info *oauthTypes.AuthorizedInfo, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: true,
		Label:    "Connector",
		Path:     "/settings/models",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "模型连接"
	}

	connID := ast.Connector
	if connID == "" {
		connID = "default"
	}

	// Role-based connector: "use::vision", "use::heavy", etc.
	if strings.HasPrefix(connID, "use::") {
		roleName := strings.TrimPrefix(connID, "use::")
		if setting.Global == nil {
			return cp
		}
		roles, _ := setting.Global.GetMerged(info.UserID, info.TeamID, llmprovider.RolesNamespace)
		if roles == nil {
			return cp
		}
		pk, mid := parseRoleTarget(roles[roleName])
		if pk == "" {
			return cp
		}
		connID = pk
		if mid != "" {
			connID = pk + "::" + mid
		}
	}

	// "default" means use the default role
	if connID == "default" {
		if setting.Global == nil {
			return cp
		}
		roles, _ := setting.Global.GetMerged(info.UserID, info.TeamID, llmprovider.RolesNamespace)
		if roles == nil {
			return cp
		}
		pk, mid := parseRoleTarget(roles["default"])
		if pk == "" {
			return cp
		}
		connID = pk
		if mid != "" {
			connID = pk + "::" + mid
		}
	}

	parts := strings.SplitN(connID, "::", 2)
	if llmprovider.Global == nil {
		return cp
	}
	p, err := llmprovider.Global.Get(parts[0])
	if err != nil || p == nil || !p.Enabled {
		return cp
	}

	cp.Status = "pass"
	return cp
}

func checkAssistantSandbox(ast *assistant.Assistant, info *oauthTypes.AuthorizedInfo, locale string, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: true,
		Label:    "Sandbox Ready",
		Path:     "/settings/sandbox",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "沙箱就绪"
	}

	// Use the unified availability check: if the agent can be selected to a
	// node (host-mode or box-mode), the sandbox is considered ready.
	if ast.SandboxV2 != nil {
		globalRunner := ""
		if sandboxv2.GlobalRunnerFunc != nil {
			globalRunner = sandboxv2.GlobalRunnerFunc()
		}
		preferred, allowed := sandboxv2.ResolveRunnerSet(nil, &ast.SandboxV2.Runner, globalRunner)
		avail := sandboxv2.CheckAvailability(nil, allowed, preferred, ast.SandboxV2.Computer.Image, ast.SandboxV2.Filter)
		if avail.Runnable {
			cp.Status = "pass"
			return cp
		}
	}

	imageRef := ""
	if ast.SandboxV2 != nil && ast.SandboxV2.Computer.Image != "" {
		imageRef = ast.SandboxV2.Computer.Image
	}
	if imageRef == "" {
		cp.Status = "pass"
		return cp
	}

	// Fallback: detailed Docker image check for better error messages.
	reg := registry.Global()
	if reg == nil {
		if isCN {
			cp.Detail = "沙箱节点未配置"
		} else {
			cp.Detail = "No sandbox node configured"
		}
		return cp
	}

	dockerNodeFound := false
	for _, snap := range reg.List() {
		if !taitypes.IsPublicNode(snap.Mode) && !sandboxNodeOwnedBy(&snap, info) {
			continue
		}
		if snap.Status != "online" || !snap.Capabilities.Docker {
			continue
		}
		dockerNodeFound = true

		res, ok := tai.GetResources(snap.TaiID)
		if !ok || res.Image == nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		images, err := res.Image.List(ctx)
		cancel()
		if err != nil {
			continue
		}

		for _, img := range images {
			for _, tag := range img.Tags {
				if tag == imageRef {
					cp.Status = "pass"
					return cp
				}
			}
		}
	}

	if !dockerNodeFound {
		if isCN {
			cp.Detail = "Docker 未安装或节点离线"
		} else {
			cp.Detail = "Docker not installed or node offline"
		}
	} else {
		if isCN {
			cp.Detail = "镜像未下载"
		} else {
			cp.Detail = "Image not downloaded"
		}
	}
	return cp
}

func checkAssistantSearch(ast *assistant.Assistant, info *oauthTypes.AuthorizedInfo, isCN bool) Checkpoint {
	cp := Checkpoint{
		Required: false,
		Label:    "Search",
		Path:     "/settings/search",
		Status:   "fail",
	}
	if isCN {
		cp.Label = "搜索"
	}

	if setting.Global == nil {
		return cp
	}

	// Check cloud search
	cloudSaved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, cloudNS)
	if cloudSaved != nil {
		if v, ok := cloudSaved["status"].(string); ok && v == "connected" {
			cp.Status = "pass"
			return cp
		}
	}

	// Check any standalone search provider
	for _, preset := range searchPresets {
		if preset.IsCloud {
			continue
		}
		saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, searchProviderNS(preset.Key))
		if saved != nil {
			if v, ok := saved["status"].(string); ok && v == "connected" {
				cp.Status = "pass"
				return cp
			}
		}
	}

	return cp
}

func hasCapability(caps []string, target string) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}
