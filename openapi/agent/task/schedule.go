package task

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/share"
)

// handleTaskScheduleGet returns the schedule config from agent_task table.
func handleTaskScheduleGet(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	task, err := tasksvc.Get(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}

	result := map[string]interface{}{
		"schedule": task.Schedule,
		"next_run": task.NextRun,
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleTaskScheduleUpdate writes schedule config to agent_task.schedule column.
func handleTaskScheduleUpdate(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	var schedule tasksvc.ScheduleConfig
	if err := c.ShouldBindJSON(&schedule); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	_, err := tasksvc.Update(c.Request.Context(), auth, chatID, &tasksvc.UpdateReq{
		Schedule: &schedule,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{"ok": true})
}

// handleTaskScheduleLogsGet returns paginated schedule trigger logs.
func handleTaskScheduleLogsGet(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")
	if _, err := tasksvc.Get(c.Request.Context(), auth, chatID); err != nil {
		respondError(c, http.StatusNotFound, err)
		return
	}

	page := queryIntDefault(c, "page", 1)
	if page < 1 {
		page = 1
	}
	pageSize := queryIntDefault(c, "page_size", 20)
	if pageSize < 1 {
		pageSize = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	tbl := scheduleLogTable()
	total, countErr := capsule.Global.Query().Table(tbl).
		Where("chat_id", "=", chatID).
		Count()
	if countErr != nil {
		log.Warn("[Schedule Logs] count error table=%s chatID=%s: %v", tbl, chatID, countErr)
	}

	rows, err := capsule.Global.Query().Table(tbl).
		Select("triggered_at").
		Where("chat_id", "=", chatID).
		OrderByDesc("triggered_at").
		Offset(offset).
		Limit(pageSize).
		Get()
	if err != nil {
		log.Warn("[Schedule Logs] query error table=%s chatID=%s: %v", tbl, chatID, err)
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	log.Trace("[Schedule Logs] table=%s chatID=%s total=%d rows=%d", tbl, chatID, total, len(rows))
	response.RespondWithSuccess(c, http.StatusOK, gin.H{
		"logs":      rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func queryIntDefault(c *gin.Context, key string, def int) int {
	s := c.Query(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func scheduleLogTable() string {
	if m, err := model.Get("__yao.agent.schedule_log"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_schedule_log"
}
