package task

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	agentconfig "github.com/yaoapp/yao/agent/config"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
)

// handleTaskScheduleGet returns the resolved schedule config + runtime status.
func handleTaskScheduleGet(c *gin.Context) {
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

	result := map[string]interface{}{
		"schedule": resolved.Schedule,
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleTaskScheduleUpdate writes schedule config at task level.
func handleTaskScheduleUpdate(c *gin.Context) {
	info := authorized.GetInfo(c)
	chatID := c.Param("chat_id")

	var schedule agentconfig.ScheduleConfig
	if err := c.ShouldBindJSON(&schedule); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	reg := setting.Global
	if reg == nil {
		respondError(c, http.StatusInternalServerError, fmt.Errorf("setting registry not initialized"))
		return
	}

	scope := setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
	ns := "task-config.task." + chatID

	existing, _ := reg.Get(scope, ns)
	if existing == nil {
		existing = make(map[string]interface{})
	}

	data, _ := json.Marshal(schedule)
	var schedMap map[string]interface{}
	json.Unmarshal(data, &schedMap)
	existing["schedule"] = schedMap

	if _, err := reg.Set(scope, ns, existing); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	tasksvc.GlobalScheduleEngine.Update(chatID, tasksvc.ScheduleConfig{
		Enabled:       schedule.Enabled,
		Mode:          schedule.Mode,
		Times:         schedule.Times,
		Days:          schedule.Days,
		IntervalValue: schedule.IntervalValue,
		IntervalUnit:  schedule.IntervalUnit,
		Timezone:      schedule.Timezone,
		StartDate:     schedule.StartDate,
		EndDate:       schedule.EndDate,
	})

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{"ok": true})
}
