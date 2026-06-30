package setting

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/assistant"
	agentconfig "github.com/yaoapp/yao/agent/config"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tools/secret"
)

func init() {
	secret.LoadPredefinedFn = func(assistantID string) map[string]secret.PredefinedMeta {
		ast, err := loadAssistant(assistantID)
		if err != nil || ast == nil || ast.SandboxV2 == nil || ast.SandboxV2.Secrets == nil {
			return nil
		}
		result := make(map[string]secret.PredefinedMeta, len(ast.SandboxV2.Secrets))
		for k, entry := range ast.SandboxV2.Secrets {
			if entry != nil {
				result[k] = secret.PredefinedMeta{
					Label:       entry.Label,
					Description: entry.Description,
				}
			}
		}
		return result
	}
}

func agentSettingScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

func agentSettingNS(id string) string {
	return "agent." + id
}

func loadAssistant(id string) (ast *assistant.Assistant, err error) {
	defer func() {
		if r := recover(); r != nil {
			ast = nil
			err = fmt.Errorf("loadAssistant panic: %v", r)
		}
	}()

	cache := assistant.GetCache()
	if cache != nil {
		if a, ok := cache.Get(id); ok && a != nil {
			return a, nil
		}
	}
	return assistant.Get(id)
}

// handleAgentSettingGet returns the combined data needed by the CUI Sandbox Tab.
// Core data (user setting) is read directly from setting.Registry.
// sandbox_config is an optional enhancement from loadAssistant.
// GET /setting/agent/:id
func handleAgentSettingGet(c *gin.Context) {
	id := c.Param("id")
	info := authorized.GetInfo(c)

	resolved, _ := agentconfig.Resolve(agentconfig.ResolveOptions{
		AssistantID: id,
		UserID:      info.UserID,
		TeamID:      info.TeamID,
	})
	if resolved == nil {
		resolved = &agentconfig.Resolved{}
	}

	runners := resolved.Runners
	if runners == nil {
		runners = []string{}
	}

	// Convert services to API format
	services := make([]map[string]interface{}, 0, len(resolved.Services))
	for _, svc := range resolved.Services {
		services = append(services, map[string]interface{}{
			"label": svc.Name,
			"port":  svc.Port,
		})
	}

	settingMap := map[string]interface{}{
		"runners":  runners,
		"image":    resolved.Image,
		"services": services,
		"secrets":  resolved.Secrets,
	}

	result := map[string]interface{}{
		"setting":           settingMap,
		"sandbox_config":    nil,
		"supported_runners": sandboxv2.SupportedRunners,
	}

	if ast, err := loadAssistant(id); err == nil && ast != nil && ast.SandboxV2 != nil {
		ports := make([]map[string]interface{}, 0, len(ast.SandboxV2.Computer.Ports))
		for _, p := range ast.SandboxV2.Computer.Ports {
			ports = append(ports, map[string]interface{}{
				"label": p.Label,
				"port":  p.Port,
			})
		}
		result["sandbox_config"] = map[string]interface{}{
			"runner": map[string]interface{}{
				"supports": ast.SandboxV2.Runner.Supports,
				"name":     ast.SandboxV2.Runner.Name,
			},
			"ports": ports,
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleAgentSettingUpdate updates the runners/image preferences (not secrets).
// PUT /setting/agent/:id
func handleAgentSettingUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	id := c.Param("id")
	info := authorized.GetInfo(c)
	scope := agentSettingScope(info)
	ns := agentSettingNS(id)

	var body struct {
		Runners  []string              `json:"runners"`
		Image    string                `json:"image"`
		Services []types.ServiceConfig `json:"services,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	for _, r := range body.Runners {
		if !containsRunner(sandboxv2.SupportedRunners, r) {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("unknown runner: %s", r))
			return
		}
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	// Read-modify-write to preserve secrets
	existing, _ := setting.Global.Get(scope, ns)
	if existing == nil {
		existing = make(map[string]interface{})
	}

	existing["runners"] = body.Runners
	existing["image"] = body.Image
	existing["services"] = body.Services

	if _, err := setting.Global.Set(scope, ns, existing); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"runners":  body.Runners,
		"image":    body.Image,
		"services": body.Services,
	})
}

// handleAgentSecretsGet returns secrets merged via config.Resolve (L1+L2).
// GET /setting/agent/:id/secrets
func handleAgentSecretsGet(c *gin.Context) {
	id := c.Param("id")
	info := authorized.GetInfo(c)

	resolved, err := agentconfig.Resolve(agentconfig.ResolveOptions{
		AssistantID: id,
		UserID:      info.UserID,
		TeamID:      info.TeamID,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	secrets := resolved.Secrets
	if secrets == nil {
		secrets = make(map[string]agentconfig.SecretInfo)
	}
	response.RespondWithSuccess(c, http.StatusOK, secrets)
}

// handleAgentSecretsUpdate creates or updates secret values.
// PUT /setting/agent/:id/secrets
func handleAgentSecretsUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	id := c.Param("id")
	info := authorized.GetInfo(c)

	var body map[string]SecretUpdateEntry
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := SecretsWriteContext{
		Scope:     agentSettingScope(info),
		Namespace: agentSettingNS(id),
	}
	keys, err := SecretsUpdate(ctx, body)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"updated": keys,
	})
}

// handleAgentSecretDelete removes a single secret key.
// DELETE /setting/agent/:id/secrets/:key
func handleAgentSecretDelete(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	id := c.Param("id")
	key := c.Param("key")
	info := authorized.GetInfo(c)

	ctx := SecretsWriteContext{
		Scope:     agentSettingScope(info),
		Namespace: agentSettingNS(id),
	}
	if err := SecretDelete(ctx, key); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{"success": true})
}

// handleAgentSkillsList returns the skills defined in the assistant's skills/ directory.
// GET /setting/agent/:id/skills
func handleAgentSkillsList(c *gin.Context) {
	id := c.Param("id")

	ast, err := loadAssistant(id)
	if err != nil || ast == nil {
		log.Warn("[agent-skills] loadAssistant(%s) failed: %v", id, err)
		response.RespondWithSuccess(c, http.StatusOK, make([]map[string]string, 0))
		return
	}

	skills := make([]map[string]string, 0)

	if ast.Path == "" {
		response.RespondWithSuccess(c, http.StatusOK, skills)
		return
	}

	skillsDir := filepath.Join(config.Conf.AppSource, ast.Path, "skills")
	dirInfo, statErr := os.Stat(skillsDir)
	if statErr != nil || !dirInfo.IsDir() {
		response.RespondWithSuccess(c, http.StatusOK, skills)
		return
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		response.RespondWithSuccess(c, http.StatusOK, skills)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}

		name, desc := parseSkillFrontmatter(skillFile)
		if name == "" {
			name = entry.Name()
		}
		skills = append(skills, map[string]string{
			"name":        name,
			"description": desc,
		})
	}

	response.RespondWithSuccess(c, http.StatusOK, skills)
}

// handleAgentSkillDetail returns the full content of a single SKILL.md.
// GET /setting/agent/:id/skills/:name
func handleAgentSkillDetail(c *gin.Context) {
	id := c.Param("id")
	skillName := c.Param("name")

	if skillName == "" {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{Code: "invalid_request", ErrorDescription: "skill name is required"})
		return
	}

	ast, err := loadAssistant(id)
	if err != nil || ast == nil {
		log.Warn("[agent-skill-detail] loadAssistant(%s) failed: %v", id, err)
		response.RespondWithError(c, http.StatusNotFound, &response.ErrorResponse{Code: "not_found", ErrorDescription: "assistant not found"})
		return
	}

	if ast.Path == "" {
		response.RespondWithError(c, http.StatusNotFound, &response.ErrorResponse{Code: "not_found", ErrorDescription: "skill not found"})
		return
	}

	skillFile := filepath.Join(config.Conf.AppSource, ast.Path, "skills", skillName, "SKILL.md")
	if _, statErr := os.Stat(skillFile); statErr != nil {
		response.RespondWithError(c, http.StatusNotFound, &response.ErrorResponse{Code: "not_found", ErrorDescription: "skill not found"})
		return
	}

	data, readErr := os.ReadFile(skillFile)
	if readErr != nil {
		log.Error("[agent-skill-detail] read %s: %v", skillFile, readErr)
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{Code: "server_error", ErrorDescription: "failed to read skill file"})
		return
	}

	name, desc := parseSkillFrontmatter(skillFile)
	if name == "" {
		name = skillName
	}

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"name":        name,
		"description": desc,
		"content":     string(data),
	})
}

// parseSkillFrontmatter extracts name and description from a SKILL.md YAML frontmatter.
func parseSkillFrontmatter(path string) (name, description string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return "", ""
	}

	end := strings.Index(content[3:], "---")
	if end < 0 {
		return "", ""
	}
	frontmatter := content[3 : end+3]

	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = strings.Trim(name, `"'`)
		}
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			description = strings.Trim(description, `"'`)
		}
	}
	return name, description
}

func containsRunner(runners []string, target string) bool {
	for _, r := range runners {
		if r == target {
			return true
		}
	}
	return false
}
