package task

import (
	"net/http"

	"github.com/gin-gonic/gin"
	agentconfig "github.com/yaoapp/yao/agent/config"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	settingapi "github.com/yaoapp/yao/openapi/setting"
	"github.com/yaoapp/yao/setting"
)

// handleTaskSecretsGet returns merged secrets for a task via config.Resolve (L1+L2+L3).
func handleTaskSecretsGet(c *gin.Context) {
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

	secrets := resolved.Secrets
	if secrets == nil {
		secrets = make(map[string]agentconfig.SecretInfo)
	}
	response.RespondWithSuccess(c, http.StatusOK, secrets)
}

// handleTaskSecretsUpdate creates or updates secrets at task level.
func handleTaskSecretsUpdate(c *gin.Context) {
	info := authorized.GetInfo(c)
	chatID := c.Param("chat_id")

	var body map[string]settingapi.SecretUpdateEntry
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	ctx := settingapi.SecretsWriteContext{
		Scope:     setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID},
		Namespace: "task-config.task." + chatID,
	}
	keys, err := settingapi.SecretsUpdate(ctx, body)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"updated": keys,
	})
}

// handleTaskSecretDelete removes a single secret key at task level.
func handleTaskSecretDelete(c *gin.Context) {
	info := authorized.GetInfo(c)
	chatID := c.Param("chat_id")
	key := c.Param("key")

	ctx := settingapi.SecretsWriteContext{
		Scope:     setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID},
		Namespace: "task-config.task." + chatID,
	}
	if err := settingapi.SecretDelete(ctx, key); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{"success": true})
}
