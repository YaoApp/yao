package task

import (
	"context"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/event"
)

const orphanThreshold = 10 * time.Minute

func healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healOrphanedTasks()
		}
	}
}

func healOrphanedTasks() {
	if capsule.Global == nil {
		return
	}
	cutoff := time.Now().Add(-orphanThreshold)
	rows, err := capsule.Global.Query().Table(tableTask()).
		Select("chat_id", "__yao_team_id").
		WhereIn("run_status", []interface{}{"running", "queued"}).
		Where("updated_at", "<", cutoff).
		WhereNull("deleted_at").
		Get()
	if err != nil || len(rows) == 0 {
		return
	}

	type orphanInfo struct {
		chatID string
		teamID string
	}
	var orphans []orphanInfo
	for _, row := range rows {
		chatID, _ := row["chat_id"].(string)
		teamID, _ := row["__yao_team_id"].(string)
		if chatID == "" {
			continue
		}
		if _, exists := daemonRegistry.Load(chatID); !exists {
			orphans = append(orphans, orphanInfo{chatID: chatID, teamID: teamID})
		}
	}

	if len(orphans) == 0 {
		return
	}

	chatIDs := make([]interface{}, len(orphans))
	for i, o := range orphans {
		chatIDs[i] = o.chatID
	}

	now := time.Now()
	_, err = capsule.Global.Query().Table(tableTask()).
		WhereIn("chat_id", chatIDs).
		WhereIn("run_status", []interface{}{"running", "queued"}).
		Update(map[string]interface{}{
			"run_status":    "failed",
			"error_message": "health check: no active daemon found",
			"completed_at":  now,
			"updated_at":    now,
		})
	if err != nil {
		log.Warn("[HealthCheck] heal error: %v", err)
		return
	}

	for _, o := range orphans {
		event.Push(context.Background(), "task.updated", map[string]any{
			"chat_id":       o.chatID,
			"run_status":    "failed",
			"__yao_team_id": o.teamID,
		})
	}
	log.Info("[HealthCheck] healed %d orphaned tasks", len(orphans))
}
