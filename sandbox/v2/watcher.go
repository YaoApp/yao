package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/yao/monitor"
)

func init() {
	monitor.Register(&sandboxWatcher{})
}

type sandboxWatcher struct{}

func (w *sandboxWatcher) Name() string            { return "sandbox" }
func (w *sandboxWatcher) Interval() time.Duration { return 30 * time.Second }

func (w *sandboxWatcher) Check(ctx context.Context) []monitor.Alert {
	if mgr == nil {
		return nil
	}

	var alerts []monitor.Alert
	mgr.boxes.Range(func(_, v any) bool {
		b := v.(*Box)

		status := b.inspectStatus(ctx)
		old, _ := b.status.Swap(status).(string)
		if old != "" && old != status {
			alerts = append(alerts, monitor.Alert{
				Level:   monitor.Info,
				Target:  "box:" + b.id,
				Message: fmt.Sprintf("status %s → %s", old, status),
			})
			if status == "running" && old != "running" {
				b.touch()
				alerts = append(alerts, monitor.Alert{
					Level:  monitor.Trace,
					Target: "box:" + b.id,
					Message: fmt.Sprintf("touch on resume, lastCall reset to %s",
						time.UnixMilli(b.lastCall.Load()).Format(time.RFC3339)),
				})
			}
		}

		if status != "running" {
			return true
		}

		// maxLifetime: independent of idle — prevents indefinitely running containers
		if b.policy == LongRunning {
			if lifetime := b.maxLifetime(); lifetime > 0 {
				age := time.Since(b.createdAt)
				if age > lifetime {
					alerts = append(alerts, monitor.Alert{
						Level:   monitor.Warn,
						Target:  "box:" + b.id,
						Message: fmt.Sprintf("lifetime expired (%s), removing", lifetime),
						Action:  func(ctx context.Context) { mgr.Remove(ctx, b.id) },
					})
				} else {
					alerts = append(alerts, monitor.Alert{
						Level:  monitor.Trace,
						Target: "box:" + b.id,
						Message: fmt.Sprintf("lifetime remaining %s (max=%s)",
							(lifetime - age).Round(time.Second), lifetime),
					})
				}
			}
		}

		idle := time.Since(b.idleSince())
		timeout := b.idleTimeout()

		alerts = append(alerts, monitor.Alert{
			Level:  monitor.Trace,
			Target: "box:" + b.id,
			Message: fmt.Sprintf("heartbeat status=%s policy=%s idle=%s timeout=%s",
				status, b.policy, idle.Round(time.Second), timeout),
		})

		if timeout <= 0 || idle <= timeout {
			return true
		}

		switch b.policy {
		case Session:
			alerts = append(alerts, monitor.Alert{
				Level:   monitor.Warn,
				Target:  "box:" + b.id,
				Message: fmt.Sprintf("session idle expired (idle=%s, timeout=%s), removing", idle.Round(time.Second), timeout),
				Action: func(ctx context.Context) {
					mgr.Remove(ctx, b.id)
				},
			})

		case LongRunning:
			alerts = append(alerts, monitor.Alert{
				Level:   monitor.Warn,
				Target:  "box:" + b.id,
				Message: fmt.Sprintf("longrunning idle expired (idle=%s, timeout=%s), stopping", idle.Round(time.Second), timeout),
				Action: func(ctx context.Context) {
					b.Stop(ctx)
				},
			})
		}

		return true
	})
	return alerts
}
