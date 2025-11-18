package types

// Manager the trace manager interface
// Manager automatically tracks current node(s) state, users don't need to manage nodes manually
// Context is bound to Manager at creation time
type Manager interface {

	// Node Tree Operations - work on current node(s)
	// Add creates next sequential node - auto-joins if currently in parallel state
	Add(input TraceInput, option TraceNodeOption) (Node, error)
	// Parallel creates multiple concurrent child nodes, returns Node interfaces for direct control
	Parallel(parallelInputs []TraceParallelInput) ([]Node, error)

	// Log Operations - log to current node(s) with chainable interface
	Info(format string, args ...any) Manager
	Debug(format string, args ...any) Manager
	Error(format string, args ...any) Manager
	Warn(format string, args ...any) Manager

	// Node Status Operations - operate on current node(s)
	SetOutput(output TraceOutput) error
	SetMetadata(key string, value any) error
	Complete(output ...TraceOutput) error // Optional output parameter
	Fail(err error) error

	// Query Operations
	GetRootNode() (*TraceNode, error)
	GetNode(id string) (*TraceNode, error)
	GetCurrentNodes() ([]*TraceNode, error)

	// Memory Space Operations
	CreateSpace(option TraceSpaceOption) (*TraceSpace, error)
	GetSpace(id string) (*TraceSpace, error)
	HasSpace(id string) bool
	DeleteSpace(id string) error
	ListSpaces() []*TraceSpace

	// Space Key-Value Operations (with automatic event broadcasting)
	SetSpaceValue(spaceID, key string, value any) error
	GetSpaceValue(spaceID, key string) (any, error)
	HasSpaceValue(spaceID, key string) bool
	DeleteSpaceValue(spaceID, key string) error
	ClearSpaceValues(spaceID string) error
	ListSpaceKeys(spaceID string) []string

	// Trace Control Operations
	// MarkComplete marks the entire trace as completed (sends trace_complete event)
	MarkComplete() error

	// Subscription Operations
	// Subscribe subscribes to trace updates (replay history + real-time)
	Subscribe() (<-chan *TraceUpdate, error)
	// SubscribeFrom subscribes from a specific timestamp (for resume)
	SubscribeFrom(since int64) (<-chan *TraceUpdate, error)
	// IsComplete checks if the trace is completed
	IsComplete() bool
}

// Node represents a trace node with operations for tree building and logging
// Context is bound to Node at creation time
type Node interface {
	// Log Operations - chainable interface
	Info(format string, args ...any) Node
	Debug(format string, args ...any) Node
	Error(format string, args ...any) Node
	Warn(format string, args ...any) Node

	// Node Tree Operations
	Add(input TraceInput, option TraceNodeOption) (Node, error)
	Parallel(parallelInputs []TraceParallelInput) ([]Node, error)
	Join(nodes []*TraceNode, input TraceInput, option TraceNodeOption) (Node, error)

	// Node Data Operations
	ID() string
	SetOutput(output TraceOutput) error
	SetMetadata(key string, value any) error

	// Node Status Operations
	SetStatus(status string) error
	Complete(output ...TraceOutput) error // Optional output parameter
	Fail(err error) error
}

// Space represents a key-value storage space
type Space interface {
	// ID returns the space identifier
	ID() string

	// Set stores a value by key
	Set(key string, value any) error

	// Get retrieves a value by key
	Get(key string) (any, error)

	// Has checks if a key exists
	Has(key string) bool

	// Delete removes a key-value pair
	Delete(key string) error

	// Clear removes all key-value pairs
	Clear() error

	// Keys returns all keys in the space
	Keys() []string
}
