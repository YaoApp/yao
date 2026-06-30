package task

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	agentconfig "github.com/yaoapp/yao/agent/config"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	ws "github.com/yaoapp/yao/workspace"
)

// handleTaskSkillsGet returns skills grouped by bundled (agent) and extended (workspace).
// bundled: from the assistant's skills/ directory (agent-provided, immutable)
// extended: from the task's bound workspace .yao/skills/ directory (runtime, user-managed)
func handleTaskSkillsGet(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	task, err := tasksvc.Get(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}

	bundled := scanSkillsDir(getAssistantSkillsDir(task.AssistantID))

	var extended []skillEntry
	if task.LastWorkspace != nil && *task.LastWorkspace != "" {
		extended = scanSkillsDir(getWorkspaceSkillsDir(c.Request.Context(), *task.LastWorkspace))
	} else {
		extended = make([]skillEntry, 0)
	}

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"bundled":       bundled,
		"extended":      extended,
		"has_workspace": task.LastWorkspace != nil && *task.LastWorkspace != "",
	})
}

// handleTaskComputersGet returns available compute nodes.
func handleTaskComputersGet(c *gin.Context) {
	response.RespondWithSuccess(c, http.StatusOK, []interface{}{})
}

// handleTaskSandboxGet returns sandbox config from config.Resolve.
func handleTaskSandboxGet(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	task, err := tasksvc.Get(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}

	resolved, err := agentconfig.Resolve(agentconfig.ResolveOptions{
		AssistantID: task.AssistantID,
		ChatID:      chatID,
		UserID:      auth.UserID,
		TeamID:      auth.TeamID,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"runner":    resolved.Runner,
		"image":     resolved.Image,
		"services":  resolved.Services,
		"timeout":   resolved.Timeout,
		"max_turns": resolved.MaxTurns,
	})
}

// handleTaskSandboxPut is not yet implemented.
func handleTaskSandboxPut(c *gin.Context) {
	response.RespondWithError(c, http.StatusNotImplemented, &response.ErrorResponse{
		Code:             "not_implemented",
		ErrorDescription: "sandbox PUT is not yet implemented",
	})
}

func getAssistantSkillsDir(assistantID string) string {
	ast, err := assistant.Get(assistantID)
	if err != nil || ast == nil || ast.Path == "" {
		return ""
	}
	return filepath.Join(config.Conf.AppSource, ast.Path, "skills")
}

func getWorkspaceSkillsDir(ctx context.Context, workspaceID string) string {
	m := ws.M()
	if m == nil {
		return ""
	}
	rootDir, err := m.MountPath(ctx, workspaceID)
	if err != nil || rootDir == "" {
		return ""
	}
	return filepath.Join(rootDir, ".yao", "skills")
}

type skillEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func scanSkillsDir(dir string) []skillEntry {
	result := make([]skillEntry, 0)
	if dir == "" {
		return result
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return result
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		name, desc := parseSkillFrontmatter(skillFile)
		if name == "" {
			name = entry.Name()
		}
		result = append(result, skillEntry{Name: name, Description: desc})
	}
	return result
}

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
