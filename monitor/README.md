# Yao Monitor

A process-level inspection service for Yao. Monitor schedules periodic health checks (watchers) and records anomalies. It knows nothing about business logic — the business layer defines what to check, how to judge, and what action to take.

## Quick Start

### 1. Implement a Watcher

```go
package sandbox

import (
    "context"
    "time"

    "github.com/yaoapp/yao/monitor"
)

type sandboxWatcher struct{}

func (w *sandboxWatcher) Name() string           { return "sandbox" }
func (w *sandboxWatcher) Interval() time.Duration { return 30 * time.Second }

func (w *sandboxWatcher) Check(ctx context.Context) []monitor.Alert {
    // Inspect containers, compare states, detect idle timeouts, etc.
    // Return an empty slice if everything is normal.
    return nil
}
```

### 2. Register via init()

```go
func init() {
    monitor.Register(&sandboxWatcher{})
}
```

Registration happens before `monitor.Start()` is called. The watcher will be picked up automatically when the engine boots.

### 3. That's It

The engine calls `monitor.Start()` / `monitor.Stop()` during load/unload. Your watcher's `Check()` will be called at the interval you specified, in its own goroutine.

## Alert Levels

| Level | Constant | Use Case |
|-------|----------|----------|
| Trace | `monitor.Trace` | Heartbeat, periodic status sync, routine checks |
| Info  | `monitor.Info` | Notable events: state changes, service registrations |
| Warn  | `monitor.Warn` | Needs attention: idle timeout, degraded state |
| Error | `monitor.Error` | Needs immediate action: crash, unreachable |

Which level to use is entirely up to the business watcher — the monitor just records what it's told.

All alert levels are delivered to subscribers via `Subscribe()`.

### Log Level by Mode

The minimum level written to `monitor.log` depends on Yao's run mode (`YAO_ENV`):

| Mode | Min Level | Effect |
|------|-----------|--------|
| `production` | Info | Trace alerts are **not** written to the log file |
| `development` | Trace | **Everything** is written |

This keeps production logs lean while giving full visibility during development.

## Alert Actions

An alert can carry an `Action` — a function the monitor executes synchronously within the tick:

```go
monitor.Alert{
    Level:   monitor.Warn,
    Target:  "box:abc123",
    Message: "idle timeout exceeded, stopping",
    Action:  func(ctx context.Context) { box.Stop(ctx) },
}
```

- Actions run synchronously in the watcher's goroutine.
- A panicking action is recovered and logged; subsequent alerts in the same tick continue.
- A long-running action blocks the next tick of *this* watcher only, not others.

## API

```go
// Register a watcher (call before Start, typically in init).
monitor.Register(w Watcher)

// Start the monitor (called by engine).
monitor.Start(ctx context.Context) error

// Stop the monitor (called by engine).
monitor.Stop() error

// Subscribe to alert notifications. Returns a subscription ID.
// Non-blocking: full channels are skipped.
monitor.Subscribe(ch chan<- *monitor.Alert) string

// Unsubscribe by ID.
monitor.Unsubscribe(id string)

// Health returns runtime status of the monitor and all watchers.
monitor.Health() HealthStatus
```

## Health Check

```go
status := monitor.Health()
// status.Running   — is the monitor running?
// status.Watchers  — per-watcher stats:
//   .Name        — watcher name
//   .Interval    — check frequency
//   .LastTick    — when the last tick completed
//   .LastAlerts  — alert count from the last tick
//   .TotalTicks  — total ticks since start
//   .Panics      — total panics caught
```

A watcher is considered healthy if `LastTick` is within `Interval × 3` of the current time.

## Logging

Monitor writes to `logs/monitor.log` (independent from `application.log`):

- **Lifecycle events** (Info): monitor started/stopped, watcher started/stopped
- **Warn/Error alerts**: always written with watcher name, target, and message
- **Info alerts**: written in both production and development
- **Trace alerts**: written only in development mode (skipped in production)
- **Panics**: always written at Error level

Log rotation uses lumberjack (50 MB, 3 backups, 7 days). Format follows `YAO_LOG_MODE` (TEXT or JSON).

## Panic Safety

- If `Check()` panics, the watcher recovers and continues on the next tick.
- If `Action()` panics, the watcher recovers and processes remaining alerts.
- Panic counts are tracked in `Health().Watchers[].Panics`.

## File Structure

```
monitor/
├── DESIGN.md         — Architecture and design decisions
├── README.md         — This file
├── types.go          — Level, Alert, Watcher interface
├── logger.go         — Independent slog.Logger → monitor.log
├── service.go        — Register, Start, Stop, Subscribe, Health
└── service_test.go
```
