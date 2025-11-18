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
	Label       string         // Display label in UI
	Icon        string         // Icon identifier
	Description string         // Node description
	Metadata    map[string]any // Additional metadata
}

// TraceSpaceOption defines options for creating a space
type TraceSpaceOption struct {
	Label       string         // Display label in UI
	Icon        string         // Icon identifier
	Description string         // Space description
	TTL         int64          // Time to live in seconds (0 = no expiration) - for display/record only
	Metadata    map[string]any // Additional metadata
}

// TraceNode the trace node implementation
type TraceNode struct {
	ID              string       // Node ID
	ParentID        string       // Parent node ID
	Children        []*TraceNode // Child nodes (for tree structure)
	TraceNodeOption              // Embedded option fields (Label, Icon, Description, Metadata)
	Status          NodeStatus   // Node status (pending, running, completed, failed, skipped)
	Input           TraceInput   // Node input data
	Output          TraceOutput  // Node output data
	CreatedAt       int64        // Creation timestamp
	StartTime       int64        // Start timestamp
	EndTime         int64        // End timestamp
	UpdatedAt       int64        // Last update timestamp
	// Other fields will be added during implementation
}

// TraceSpace the trace memory space implementation (can add methods for serialization)
type TraceSpace struct {
	ID               string // Space ID
	TraceSpaceOption        // Embedded option fields (Label, Icon, Description, TTL, Metadata)
	CreatedAt        int64  // Creation timestamp
	UpdatedAt        int64  // Last update timestamp
	// Internal data storage will be managed by implementation
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
	Type      string // Update type (see UpdateType* constants)
	TraceID   string // Trace ID
	NodeID    string // Node ID (optional, for node/log updates)
	SpaceID   string // Space ID (optional, for space updates)
	Timestamp int64  // Update timestamp
	Data      any    // Update data (payload structures below)
}

// Event payload structures (matching frontend SSE format)

// TraceInitData payload for "init" event
type TraceInitData struct {
	TraceID   string     `json:"traceId"`
	AgentName string     `json:"agentName,omitempty"`
	RootNode  *TraceNode `json:"rootNode,omitempty"`
}

// NodeStartData payload for "node_start" event
// Supports both single node and multiple nodes (for parallel operations)
type NodeStartData struct {
	Node  *TraceNode   `json:"node,omitempty"`  // Single node
	Nodes []*TraceNode `json:"nodes,omitempty"` // Multiple nodes (for parallel)
}

// NodeCompleteData payload for "node_complete" event
type NodeCompleteData struct {
	NodeID   string         `json:"nodeId"`
	Status   CompleteStatus `json:"status"` // "success" or "failed"
	EndTime  int64          `json:"endTime"`
	Duration int64          `json:"duration"` // in milliseconds
	Output   TraceOutput    `json:"output,omitempty"`
}

// NodeFailedData payload for "node_failed" event (same as NodeCompleteData but with error)
type NodeFailedData struct {
	NodeID   string         `json:"nodeId"`
	Status   CompleteStatus `json:"status"` // "failed"
	EndTime  int64          `json:"endTime"`
	Duration int64          `json:"duration"`
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
	Timestamp  int64  `json:"timestamp"`
	Importance string `json:"importance,omitempty"` // "high", "medium", "low"
}

// TraceCompleteData payload for "complete" event
type TraceCompleteData struct {
	TraceID       string      `json:"traceId"`
	Status        TraceStatus `json:"status"` // "completed"
	TotalDuration int64       `json:"totalDuration"`
}

// SpaceDeletedData payload for "space_deleted" event
type SpaceDeletedData struct {
	SpaceID string `json:"spaceId"`
}

// MemoryDeleteData payload for "memory_delete" event
type MemoryDeleteData struct {
	SpaceID string `json:"spaceId"`
	Key     string `json:"key,omitempty"`     // Empty when clearing all
	Cleared bool   `json:"cleared,omitempty"` // True when clearing all keys
}

// TraceInfo stores trace metadata and manager instance
type TraceInfo struct {
	ID        string         `json:"id"`
	Driver    string         `json:"driver"`
	Status    TraceStatus    `json:"status"` // Trace status
	Options   []any          `json:"options,omitempty"`
	Manager   Manager        `json:"-"` // Not persisted
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
	CreatedBy string         `json:"__yao_created_by,omitempty"`
	UpdatedBy string         `json:"__yao_updated_by,omitempty"`
	TeamID    string         `json:"__yao_team_id,omitempty"`
	TenantID  string         `json:"__yao_tenant_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TraceOption defines options for creating a trace
type TraceOption struct {
	ID        string         // Optional trace ID (if empty, generates new ID)
	CreatedBy string         // User who created the trace
	TeamID    string         // Team ID
	TenantID  string         // Tenant ID
	Metadata  map[string]any // Additional metadata
}
