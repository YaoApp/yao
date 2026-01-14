package types

import "time"

// ==================== Internal Interfaces ====================
// These are internal implementation interfaces, not exposed via API.
// External API is defined in api/api.go
// All interfaces use *Context (not context.Context) for consistency.

// Manager - robot lifecycle and clock trigger management
type Manager interface {
	Start() error
	Stop() error
	Tick(ctx *Context, now time.Time) error
}

// Executor - executes robot phases
type Executor interface {
	Execute(ctx *Context, robot *Robot, trigger TriggerType, data interface{}) (*Execution, error)
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
