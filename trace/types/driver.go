package types

import "context"

// TraceLog represents a log entry
type TraceLog struct {
	Timestamp int64  // Log timestamp (milliseconds since epoch)
	Level     string // Log level (info, debug, error, warn)
	Message   string // Log message
	Data      []any  // Additional data arguments
	NodeID    string // Node ID this log belongs to
}

// Driver defines the storage driver interface that providers must implement
// Driver is only responsible for persistence operations, not business logic
type Driver interface {
	// SaveNode persists a node to storage
	SaveNode(ctx context.Context, traceID string, node *TraceNode) error

	// LoadNode loads a node from storage
	LoadNode(ctx context.Context, traceID string, nodeID string) (*TraceNode, error)

	// LoadTrace loads the entire trace tree from storage
	LoadTrace(ctx context.Context, traceID string) (*TraceNode, error)

	// SaveSpace persists a space to storage
	SaveSpace(ctx context.Context, traceID string, space *TraceSpace) error

	// LoadSpace loads a space from storage
	LoadSpace(ctx context.Context, traceID string, spaceID string) (*TraceSpace, error)

	// DeleteSpace removes a space from storage
	DeleteSpace(ctx context.Context, traceID string, spaceID string) error

	// ListSpaces lists all space IDs for a trace
	ListSpaces(ctx context.Context, traceID string) ([]string, error)

	// Space KV Operations
	// SetSpaceKey stores a value by key in a space
	SetSpaceKey(ctx context.Context, traceID, spaceID, key string, value any) error

	// GetSpaceKey retrieves a value by key from a space
	GetSpaceKey(ctx context.Context, traceID, spaceID, key string) (any, error)

	// HasSpaceKey checks if a key exists in a space
	HasSpaceKey(ctx context.Context, traceID, spaceID, key string) bool

	// DeleteSpaceKey removes a key-value pair from a space
	DeleteSpaceKey(ctx context.Context, traceID, spaceID, key string) error

	// ClearSpaceKeys removes all key-value pairs from a space
	ClearSpaceKeys(ctx context.Context, traceID, spaceID string) error

	// ListSpaceKeys returns all keys in a space
	ListSpaceKeys(ctx context.Context, traceID, spaceID string) ([]string, error)

	// SaveLog appends a log entry to storage
	SaveLog(ctx context.Context, traceID string, log *TraceLog) error

	// LoadLogs loads all logs for a trace or specific node
	LoadLogs(ctx context.Context, traceID string, nodeID string) ([]*TraceLog, error)

	// SaveTraceInfo persists trace metadata to storage
	SaveTraceInfo(ctx context.Context, info *TraceInfo) error

	// LoadTraceInfo loads trace metadata from storage
	LoadTraceInfo(ctx context.Context, traceID string) (*TraceInfo, error)

	// DeleteTrace removes entire trace and all its data
	DeleteTrace(ctx context.Context, traceID string) error

	// SaveUpdate persists a trace update event to storage
	SaveUpdate(ctx context.Context, traceID string, update *TraceUpdate) error

	// LoadUpdates loads trace update events from storage (filtering by timestamp in milliseconds)
	LoadUpdates(ctx context.Context, traceID string, since int64) ([]*TraceUpdate, error)

	// Archive archives a trace (compress and make read-only)
	Archive(ctx context.Context, traceID string) error

	// IsArchived checks if a trace is archived
	IsArchived(ctx context.Context, traceID string) (bool, error)

	// Close closes the driver and releases resources
	Close() error
}
