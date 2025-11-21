package types

// NodeStatus represents the status of a node
type NodeStatus string

// Node status constants
const (
	StatusPending   NodeStatus = "pending"   // Node created but not started
	StatusRunning   NodeStatus = "running"   // Node is currently executing
	StatusCompleted NodeStatus = "completed" // Node finished successfully
	StatusFailed    NodeStatus = "failed"    // Node failed with error
	StatusSkipped   NodeStatus = "skipped"   // Node was skipped
	StatusCancelled NodeStatus = "cancelled" // Node was cancelled
)

// TraceStatus represents the status of a trace
type TraceStatus string

// Trace status constants
const (
	TraceStatusPending   TraceStatus = "pending"   // Trace created but not started
	TraceStatusRunning   TraceStatus = "running"   // Trace is running
	TraceStatusCompleted TraceStatus = "completed" // Trace completed
	TraceStatusFailed    TraceStatus = "failed"    // Trace failed
	TraceStatusCancelled TraceStatus = "cancelled" // Trace was cancelled
)

// CompleteStatus represents the completion status in events
type CompleteStatus string

// Complete status constants (for event payloads)
const (
	CompleteStatusSuccess   CompleteStatus = "success"   // Operation succeeded
	CompleteStatusFailed    CompleteStatus = "failed"    // Operation failed
	CompleteStatusCancelled CompleteStatus = "cancelled" // Operation was cancelled
)

// TraceNodeOption defines options for creating a node
type TraceNodeOption struct {
	Label              string         `json:"label"`                          // Display label in UI
	Type               string         `json:"type"`                           // Node type identifier
	Icon               string         `json:"icon"`                           // Icon identifier
	Description        string         `json:"description"`                    // Node description
	Metadata           map[string]any `json:"metadata,omitempty"`             // Additional metadata
	AutoCompleteParent *bool          `json:"auto_complete_parent,omitempty"` // Auto-complete parent node(s) when this node is created (nil = default true)
}

// TraceSpaceOption defines options for creating a space
type TraceSpaceOption struct {
	Label       string         `json:"label"`              // Display label in UI
	Type        string         `json:"type"`               // Space type identifier
	Icon        string         `json:"icon"`               // Icon identifier
	Description string         `json:"description"`        // Space description
	TTL         int64          `json:"ttl"`                // Time to live in seconds (0 = no expiration) - for display/record only
	Metadata    map[string]any `json:"metadata,omitempty"` // Additional metadata
}

// TraceNode the trace node implementation
type TraceNode struct {
	ID              string           `json:"id"`         // Node ID
	ParentIDs       []string         `json:"parent_ids"` // Parent node IDs (supports multiple parents for implicit join)
	Children        []*TraceNode     `json:"children"`   // Child nodes (for tree structure)
	TraceNodeOption `json:",inline"` // Embedded option fields (Label, Icon, Description, Metadata)
	Status          NodeStatus       `json:"status"`           // Node status (pending, running, completed, failed, skipped)
	Input           TraceInput       `json:"input,omitempty"`  // Node input data
	Output          TraceOutput      `json:"output,omitempty"` // Node output data
	CreatedAt       int64            `json:"created_at"`       // Creation timestamp (milliseconds since epoch)
	StartTime       int64            `json:"start_time"`       // Start timestamp (milliseconds since epoch)
	EndTime         int64            `json:"end_time"`         // End timestamp (milliseconds since epoch)
	UpdatedAt       int64            `json:"updated_at"`       // Last update timestamp (milliseconds since epoch)
	// Other fields will be added during implementation
}

// TraceSpace the trace memory space implementation (can add methods for serialization)
type TraceSpace struct {
	ID               string           `json:"id"` // Space ID
	TraceSpaceOption `json:",inline"` // Embedded option fields (Label, Icon, Description, TTL, Metadata)
	CreatedAt        int64            `json:"created_at"` // Creation timestamp (milliseconds since epoch)
	UpdatedAt        int64            `json:"updated_at"` // Last update timestamp (milliseconds since epoch)
	// Internal data storage will be managed by implementation
}

// TraceSpaceData represents a space with all its key-value data (for API responses)
type TraceSpaceData struct {
	TraceSpace                // Embedded space metadata
	Data       map[string]any `json:"data"` // All key-value pairs in the space
}

// TraceParallelInput defines input and options for a parallel node
type TraceParallelInput struct {
	Input  TraceInput      // Input data for the node
	Option TraceNodeOption // Display options (label, icon, etc.)
}

// TraceInput the trace input (can add methods for validation)
type TraceInput = any

// TraceOutput the trace output (can add methods for formatting)
type TraceOutput = any

// Update event type constants (matching frontend SSE events)
const (
	// Trace lifecycle events
	UpdateTypeInit     = "init"     // Trace initialization
	UpdateTypeComplete = "complete" // Entire trace completed

	// Node lifecycle events
	UpdateTypeNodeStart    = "node_start"    // Node started (created)
	UpdateTypeNodeComplete = "node_complete" // Node completed successfully
	UpdateTypeNodeFailed   = "node_failed"   // Node failed with error
	UpdateTypeNodeUpdated  = "node_updated"  // Node data updated (output, metadata, status)

	// Log events
	UpdateTypeLogAdded = "log_added" // Log entry added to node

	// Memory/Space events
	UpdateTypeMemoryAdd    = "memory_add"    // Memory space item added (key-value added)
	UpdateTypeMemoryUpdate = "memory_update" // Memory space item updated
	UpdateTypeMemoryDelete = "memory_delete" // Memory space item deleted
	UpdateTypeSpaceCreated = "space_created" // Space was created
	UpdateTypeSpaceDeleted = "space_deleted" // Space was deleted
)

// TraceUpdate represents a trace update event for subscriptions
type TraceUpdate struct {
	Type      string `json:"type"`      // Update type (see UpdateType* constants)
	TraceID   string `json:"trace_id"`  // Trace ID
	NodeID    string `json:"node_id"`   // Node ID (optional, for node/log updates)
	SpaceID   string `json:"space_id"`  // Space ID (optional, for space updates)
	Timestamp int64  `json:"timestamp"` // Update timestamp (milliseconds since epoch)
	Data      any    `json:"data"`      // Update data (payload structures below)
}

// Event payload structures (matching frontend SSE format)

// TraceInitData payload for "init" event
type TraceInitData struct {
	TraceID   string     `json:"trace_id"`
	AgentName string     `json:"agent_name,omitempty"`
	RootNode  *TraceNode `json:"root_node,omitempty"`
}

// NodeStartData payload for "node_start" event
// Supports both single node and multiple nodes (for parallel operations)
type NodeStartData struct {
	Node  *TraceNode   `json:"node,omitempty"`  // Single node
	Nodes []*TraceNode `json:"nodes,omitempty"` // Multiple nodes (for parallel)
}

// NodeCompleteData payload for "node_complete" event
type NodeCompleteData struct {
	NodeID   string         `json:"node_id"`
	Status   CompleteStatus `json:"status"`   // "success" or "failed"
	EndTime  int64          `json:"end_time"` // milliseconds since epoch
	Duration int64          `json:"duration"` // duration in milliseconds
	Output   TraceOutput    `json:"output,omitempty"`
}

// NodeFailedData payload for "node_failed" event (same as NodeCompleteData but with error)
type NodeFailedData struct {
	NodeID   string         `json:"node_id"`
	Status   CompleteStatus `json:"status"`   // "failed"
	EndTime  int64          `json:"end_time"` // milliseconds since epoch
	Duration int64          `json:"duration"` // duration in milliseconds
	Error    string         `json:"error"`
}

// MemoryAddData payload for "memory_add" event
type MemoryAddData struct {
	Type string     `json:"type"` // Space type/ID (e.g., "context", "intent", "knowledge")
	Item MemoryItem `json:"item"`
}

// MemoryItem represents an item in memory space
type MemoryItem struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title,omitempty"`
	Content    any    `json:"content"`
	Timestamp  int64  `json:"timestamp"`            // milliseconds since epoch
	Importance string `json:"importance,omitempty"` // "high", "medium", "low"
}

// TraceCompleteData payload for "complete" event
type TraceCompleteData struct {
	TraceID       string      `json:"trace_id"`
	Status        TraceStatus `json:"status"`         // "completed"
	TotalDuration int64       `json:"total_duration"` // duration in milliseconds
}

// SpaceDeletedData payload for "space_deleted" event
type SpaceDeletedData struct {
	SpaceID string `json:"space_id"`
}

// MemoryDeleteData payload for "memory_delete" event
type MemoryDeleteData struct {
	SpaceID string `json:"space_id"`
	Key     string `json:"key,omitempty"`     // Empty when clearing all
	Cleared bool   `json:"cleared,omitempty"` // True when clearing all keys
}

// TraceInfo stores trace metadata and manager instance
type TraceInfo struct {
	ID         string         `json:"id"`
	Driver     string         `json:"driver"`
	Status     TraceStatus    `json:"status"` // Trace status
	Options    []any          `json:"options,omitempty"`
	Manager    Manager        `json:"-"`                     // Not persisted
	CreatedAt  int64          `json:"created_at"`            // milliseconds since epoch
	UpdatedAt  int64          `json:"updated_at"`            // milliseconds since epoch
	ArchivedAt *int64         `json:"archived_at,omitempty"` // milliseconds since epoch, nil if not archived
	Archived   bool           `json:"archived"`              // Whether this trace is archived (read-only)
	CreatedBy  string         `json:"__yao_created_by,omitempty"`
	UpdatedBy  string         `json:"__yao_updated_by,omitempty"`
	TeamID     string         `json:"__yao_team_id,omitempty"`
	TenantID   string         `json:"__yao_tenant_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// TraceOption defines options for creating a trace
type TraceOption struct {
	ID                   string         // Optional trace ID (if empty, generates new ID)
	CreatedBy            string         // User who created the trace
	TeamID               string         // Team ID
	TenantID             string         // Tenant ID
	Metadata             map[string]any // Additional metadata
	AutoArchive          bool           // Automatically archive when trace completes/fails
	ArchiveOnClose       bool           // Archive on explicit Close() call
	ArchiveCompressLevel int            // gzip compression level (0-9, default: gzip.DefaultCompression)
}
