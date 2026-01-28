package types

import "time"

// ==================== Internal Interfaces ====================
// These are internal implementation interfaces, not exposed via API.
// External API is defined in api/api.go
// All interfaces use *Context (not context.Context) for consistency.

// ExecutionControl provides pause/resume/stop control for running executions
// This interface is implemented by trigger.ControlledExecution
type ExecutionControl interface {
	// IsPaused returns true if execution is paused
	IsPaused() bool
	// IsCancelled returns true if execution is cancelled
	IsCancelled() bool
	// WaitIfPaused blocks until resumed or cancelled, returns error if cancelled
	WaitIfPaused() error
	// CheckCancelled returns ErrExecutionCancelled if cancelled
	CheckCancelled() error
}

// Manager - robot lifecycle and clock trigger management
type Manager interface {
	Start() error
	Stop() error
	Tick(ctx *Context, now time.Time) error
}

// Executor - executes robot phases
type Executor interface {
	// ExecuteWithControl runs execution with pre-generated ID and execution control (used by pool)
	// control: optional, allows pause/resume functionality
	ExecuteWithControl(ctx *Context, robot *Robot, trigger TriggerType, data interface{}, execID string, control ExecutionControl) (*Execution, error)

	// ExecuteWithID runs execution with a pre-generated ID but no control (for backward compatibility)
	ExecuteWithID(ctx *Context, robot *Robot, trigger TriggerType, data interface{}, execID string) (*Execution, error)

	// Execute runs execution with auto-generated ID (for direct calls)
	Execute(ctx *Context, robot *Robot, trigger TriggerType, data interface{}) (*Execution, error)

	// Metrics and control (for monitoring and testing)
	ExecCount() int    // total execution count
	CurrentCount() int // currently running count
	Reset()            // reset counters
}

// Pool - worker pool for concurrent execution
type Pool interface {
	Start() error
	Stop() error
	Submit(ctx *Context, robot *Robot, trigger TriggerType, data interface{}) (string, error)
	Running() int
	Queued() int
}

// Cache - in-memory robot cache
type Cache interface {
	Load(ctx *Context) error
	Get(memberID string) *Robot
	List(teamID string) []*Robot
	Refresh(ctx *Context, memberID string) error
	Add(robot *Robot)
	Remove(memberID string)
}

// Dedup - deduplication check
type Dedup interface {
	Check(ctx *Context, memberID string, trigger TriggerType) (DedupResult, error)
	Mark(memberID string, trigger TriggerType, window time.Duration)
}

// Store - data storage operations (KB, DB)
type Store interface {
	SaveLearning(ctx *Context, memberID string, entries []LearningEntry) error
	GetHistory(ctx *Context, memberID string, limit int) ([]LearningEntry, error)
	SearchKB(ctx *Context, collections []string, query string) ([]interface{}, error)
	QueryDB(ctx *Context, models []string, query interface{}) ([]interface{}, error)
}
