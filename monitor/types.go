package monitor

import (
	"context"
	"time"
)

// Level represents alert severity.
type Level int

const (
	Trace Level = iota // Heartbeat, periodic status sync — not logged
	Info               // Notable events: state changes, registrations
	Warn               // Needs attention: idle timeout, degraded state
	Error              // Needs immediate action: crash, unreachable
)

func (l Level) String() string {
	switch l {
	case Trace:
		return "trace"
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// Alert represents a single finding from a watcher check.
type Alert struct {
	Watcher string                    // Source watcher name (set by monitor)
	Level   Level                     // Severity
	Target  string                    // Target identifier, e.g. "box:abc123", "robot:member-456"
	Message string                    // Human-readable description
	Action  func(ctx context.Context) // Business-layer action; nil means notification only
}

// Watcher is the interface that business modules implement and register
// with the monitor service.
type Watcher interface {
	// Name returns a globally unique watcher name, used for logging and dedup.
	Name() string

	// Interval returns the check frequency.
	Interval() time.Duration

	// Check performs a single inspection and returns any alerts found.
	// An empty slice means everything is normal.
	// ctx is cancelled when the monitor stops.
	Check(ctx context.Context) []Alert
}
