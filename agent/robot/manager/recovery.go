package manager

import (
	"context"
	"log"

	"github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
)

var nonTerminalStatuses = []types.ExecStatus{
	types.ExecRunning, types.ExecPaused, types.ExecPending,
	types.ExecWaiting, types.ExecConfirming,
}

// recoverExecutions scans the DB for non-terminal executions left by a prior
// server crash. Running/paused/pending records are marked failed; waiting/confirming
// records are kept as-is and returned for notification.
func (m *Manager) recoverExecutions(ctx context.Context) []events.ExecPayload {
	execStore := store.NewExecutionStore()
	robotStore := store.NewRobotStore()

	var pendingNotifications []events.ExecPayload
	affectedMembers := map[string]bool{}

	pageSize := 100
	for page := 1; ; page++ {
		result, err := execStore.ListByStatuses(ctx, nonTerminalStatuses, &store.ListOptions{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			log.Printf("[recovery] failed to list non-terminal executions page %d: %v", page, err)
			break
		}
		if len(result.Data) == 0 {
			break
		}

		for _, record := range result.Data {
			affectedMembers[record.MemberID] = true

			switch record.Status {
			case types.ExecRunning, types.ExecPaused, types.ExecPending:
				if err := execStore.UpdateStatus(ctx, record.ExecutionID, types.ExecFailed,
					"execution interrupted by server restart"); err != nil {
					log.Printf("[recovery] failed to mark %s as failed: %v", record.ExecutionID, err)
				}
			case types.ExecWaiting, types.ExecConfirming:
				pendingNotifications = append(pendingNotifications, events.ExecPayload{
					ExecutionID: record.ExecutionID,
					MemberID:    record.MemberID,
					TeamID:      record.TeamID,
					Status:      string(record.Status),
				})
			}
		}

		if len(result.Data) < pageSize {
			break
		}
	}

	fixRobotStatuses(ctx, execStore, robotStore, affectedMembers)

	return pendingNotifications
}

// fixRobotStatuses sets robots to idle when they no longer have any non-terminal executions.
func fixRobotStatuses(ctx context.Context, execStore *store.ExecutionStore, robotStore *store.RobotStore, members map[string]bool) {
	for memberID := range members {
		result, err := execStore.ListByStatuses(ctx, nonTerminalStatuses, &store.ListOptions{
			MemberID: memberID,
			PageSize: 1,
		})
		if err != nil {
			log.Printf("[recovery] failed to check remaining executions for %s: %v", memberID, err)
			continue
		}
		if result.Total == 0 {
			if err := robotStore.UpdateStatus(ctx, memberID, types.RobotIdle); err != nil {
				log.Printf("[recovery] failed to set %s to idle: %v", memberID, err)
			}
		}
	}
}
