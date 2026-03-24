package robot

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/monitor"
)

func init() {
	monitor.Register(&robotTasksWatcher{})
}

// WatcherConfig holds tuning knobs for the robot-tasks watcher.
type WatcherConfig struct {
	Interval       time.Duration
	MaxRunDuration time.Duration
	WaitingTimeout time.Duration
	ConfirmTimeout time.Duration
}

var defaultConfig = WatcherConfig{
	Interval:       5 * time.Minute,
	MaxRunDuration: 4 * time.Hour,
	WaitingTimeout: 24 * time.Hour,
	ConfirmTimeout: 1 * time.Hour,
}

type robotTasksWatcher struct {
	config WatcherConfig
}

func (w *robotTasksWatcher) Name() string { return "robot-tasks" }

func (w *robotTasksWatcher) Interval() time.Duration {
	if w.config.Interval > 0 {
		return w.config.Interval
	}
	return defaultConfig.Interval
}

func (w *robotTasksWatcher) Check(ctx context.Context) []monitor.Alert {
	mgr := api.GetManager()
	if mgr == nil || !mgr.IsStarted() {
		return nil
	}

	var alerts []monitor.Alert
	execStore := store.NewExecutionStore()
	now := time.Now()

	alerts = append(alerts, w.checkZombieRunning(ctx, execStore, now)...)
	alerts = append(alerts, w.checkWaitingTimeout(ctx, execStore, now)...)
	alerts = append(alerts, w.checkConfirmingTimeout(ctx, execStore, now)...)

	return alerts
}

func (w *robotTasksWatcher) maxRunDuration() time.Duration {
	if w.config.MaxRunDuration > 0 {
		return w.config.MaxRunDuration
	}
	return defaultConfig.MaxRunDuration
}

func (w *robotTasksWatcher) waitingTimeout() time.Duration {
	if w.config.WaitingTimeout > 0 {
		return w.config.WaitingTimeout
	}
	return defaultConfig.WaitingTimeout
}

func (w *robotTasksWatcher) confirmTimeout() time.Duration {
	if w.config.ConfirmTimeout > 0 {
		return w.config.ConfirmTimeout
	}
	return defaultConfig.ConfirmTimeout
}

// checkZombieRunning finds running executions that exceeded maxRunDuration
// and are not tracked by the in-memory execController.
func (w *robotTasksWatcher) checkZombieRunning(ctx context.Context, execStore *store.ExecutionStore, now time.Time) []monitor.Alert {
	var alerts []monitor.Alert
	maxDur := w.maxRunDuration()

	result, err := execStore.List(ctx, &store.ListOptions{
		Status:   types.ExecRunning,
		PageSize: 100,
	})
	if err != nil {
		return nil
	}

	mgr := api.GetManager()

	for _, rec := range result.Data {
		if rec.StartTime == nil {
			continue
		}
		deadline := rec.StartTime.Add(maxDur)
		if now.Before(deadline) {
			continue
		}

		// Skip if still tracked by execController (genuinely running)
		if mgr != nil {
			if _, err := mgr.GetExecutionStatus(rec.ExecutionID); err == nil {
				continue
			}
		}

		execID := rec.ExecutionID
		alerts = append(alerts, monitor.Alert{
			Level:   monitor.Warn,
			Target:  fmt.Sprintf("execution:%s", execID),
			Message: fmt.Sprintf("zombie running execution %s (started %s, exceeded %v)", execID, rec.StartTime.Format(time.RFC3339), maxDur),
			Action: func(ctx context.Context) {
				mod := model.Select("__yao.agent.execution")
				if mod == nil {
					return
				}
				// CAS: only update if still running
				mod.UpdateWhere(
					model.QueryParam{
						Wheres: []model.QueryWhere{
							{Column: "execution_id", Value: execID},
							{Column: "status", Value: string(types.ExecRunning)},
						},
					},
					map[string]interface{}{
						"status":   string(types.ExecFailed),
						"error":    "killed by watcher: exceeded max run duration",
						"end_time": time.Now(),
					},
				)
			},
		})
	}

	return alerts
}

// checkWaitingTimeout finds waiting executions past the waiting timeout.
func (w *robotTasksWatcher) checkWaitingTimeout(ctx context.Context, execStore *store.ExecutionStore, now time.Time) []monitor.Alert {
	var alerts []monitor.Alert
	timeout := w.waitingTimeout()

	result, err := execStore.List(ctx, &store.ListOptions{
		Status:   types.ExecWaiting,
		PageSize: 100,
	})
	if err != nil {
		return nil
	}

	for _, rec := range result.Data {
		if rec.UpdatedAt == nil {
			continue
		}
		if now.Before(rec.UpdatedAt.Add(timeout)) {
			continue
		}

		execID := rec.ExecutionID
		alerts = append(alerts, monitor.Alert{
			Level:   monitor.Warn,
			Target:  fmt.Sprintf("execution:%s", execID),
			Message: fmt.Sprintf("waiting execution %s timed out (last updated %s, timeout %v)", execID, rec.UpdatedAt.Format(time.RFC3339), timeout),
			Action: func(ctx context.Context) {
				mod := model.Select("__yao.agent.execution")
				if mod == nil {
					return
				}
				mod.UpdateWhere(
					model.QueryParam{
						Wheres: []model.QueryWhere{
							{Column: "execution_id", Value: execID},
							{Column: "status", Value: string(types.ExecWaiting)},
						},
					},
					map[string]interface{}{
						"status":   string(types.ExecCancelled),
						"error":    "cancelled by watcher: waiting timeout exceeded",
						"end_time": time.Now(),
					},
				)
			},
		})
	}

	return alerts
}

// checkConfirmingTimeout finds confirming executions past the confirm timeout.
func (w *robotTasksWatcher) checkConfirmingTimeout(ctx context.Context, execStore *store.ExecutionStore, now time.Time) []monitor.Alert {
	var alerts []monitor.Alert
	timeout := w.confirmTimeout()

	result, err := execStore.List(ctx, &store.ListOptions{
		Status:   types.ExecConfirming,
		PageSize: 100,
	})
	if err != nil {
		return nil
	}

	for _, rec := range result.Data {
		if rec.UpdatedAt == nil {
			continue
		}
		if now.Before(rec.UpdatedAt.Add(timeout)) {
			continue
		}

		execID := rec.ExecutionID
		alerts = append(alerts, monitor.Alert{
			Level:   monitor.Info,
			Target:  fmt.Sprintf("execution:%s", execID),
			Message: fmt.Sprintf("confirming execution %s timed out (last updated %s, timeout %v)", execID, rec.UpdatedAt.Format(time.RFC3339), timeout),
			Action: func(ctx context.Context) {
				mod := model.Select("__yao.agent.execution")
				if mod == nil {
					return
				}
				mod.UpdateWhere(
					model.QueryParam{
						Wheres: []model.QueryWhere{
							{Column: "execution_id", Value: execID},
							{Column: "status", Value: string(types.ExecConfirming)},
						},
					},
					map[string]interface{}{
						"status":   string(types.ExecCancelled),
						"error":    "cancelled by watcher: confirmation timeout exceeded",
						"end_time": time.Now(),
					},
				)
			},
		})
	}

	return alerts
}
