package types

import (
	"time"
)

// Type for DSL
type Type string

// Status for DSL
type Status string

// StoreType for DSL store
type StoreType string

// LintSeverity for DSL linter
type LintSeverity string

// StoreType for DSL store
const (
	StoreTypeDB   StoreType = "db"
	StoreTypeFile StoreType = "file"
)

// Status for DSL
const (
	StatusLoading Status = "loading"
	StatusLoaded  Status = "loaded"
	StatusError   Status = "error"
)

// LintSeverity for DSL linter
const (
	LintSeverityError   LintSeverity = "error"
	LintSeverityWarning LintSeverity = "warning"
	LintSeverityInfo    LintSeverity = "info"
	LintSeverityHint    LintSeverity = "hint"
)

// Type for DSL
const (
	// TypeModel for model
	TypeModel Type = "model"
	// TypeAPI for api
	TypeAPI Type = "api"
	// TypeConnector for connector
	TypeConnector Type = "connector"
	// TypeMCPServer for MCP server
	TypeMCPServer Type = "mcp-server"
	// TypeMCPClient for MCP client
	TypeMCPClient Type = "mcp-client"
	// TypeStore for store
	TypeStore Type = "store"
	// TypeSchedule for schedule
	TypeSchedule Type = "schedule"

	// TypeTable for table
	TypeTable Type = "table"
	// TypeForm for form
	TypeForm Type = "form"
	// TypeList for list
	TypeList Type = "list"
	// TypeChart for chart
	TypeChart Type = "chart"
	// TypeDashboard for dashboard
	TypeDashboard Type = "dashboard"

	// TypeFlow for flow
	TypeFlow Type = "flow"
	// TypePipe for pipe
	TypePipe Type = "pipe"
	// TypeAIGC for aigc
	TypeAIGC Type = "aigc"

	// TypeUnknown for unknown
	TypeUnknown Type = "unknown"
)

// Info for DSL
type Info struct {
	ID string `json:"id" yaml:"id"` // Unique identifier for the DSL instance

	Type        Type     `json:"type" yaml:"type"`                                   // DSL type (model, api, table, form, list, chart, dashboard, etc.)
	Label       string   `json:"label,omitempty" yaml:"label,omitempty"`             // Display name for the DSL
	Description string   `json:"description,omitempty" yaml:"description,omitempty"` // Detailed description of the DSL
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`               // Tags for categorization and filtering

	Sort  int       `json:"sort,omitempty" yaml:"sort,omitempty"` // Sort order for display, default is 0
	Path  string    `json:"path" yaml:"path"`                     // File system path or identifier
	Store StoreType `json:"store" yaml:"store"`                   // Storage type (file or database)

	Readonly bool `json:"readonly,omitempty" yaml:"readonly,omitempty"` // Whether the DSL is readonly
	Builtin  bool `json:"built_in,omitempty" yaml:"built_in,omitempty"` // Whether this is a built-in DSL

	Status Status    `json:"status,omitempty" yaml:"status,omitempty"` // Current status (loading, loaded, error)
	Mtime  time.Time `json:"mtime" yaml:"mtime"`                       // Last modification timestamp
	Ctime  time.Time `json:"ctime" yaml:"ctime"`                       // Creation timestamp

	Source string `json:"source,omitempty" yaml:"source,omitempty"` // Source content, only available when explicitly requested
}

// ListOptions for DSL list
type ListOptions struct {
	Sort    string
	Order   string
	Store   StoreType
	Source  bool
	Tags    []string
	Pattern string // Pattern for file name matching, e.g. "test_*" for test files
}

// CreateOptions for DSL upsert
type CreateOptions struct {
	ID     string                 // ID is the id of the DSL, if not provided, a new id will be generated, required
	Source string                 // Source is the source of the DSL, if not provided, the DSL will be loaded from the file system
	Store  StoreType              // Store is the store type of the DSL, if not provided, the DSL will be loaded from the file system
	Load   map[string]interface{} // LoadOptions is the options for the DSL, if not provided, the DSL will be loaded from the file system
}

// UpdateOptions for DSL upsert
type UpdateOptions struct {
	ID     string                 // ID is the id of the DSL, if not provided, a new id will be generated, required
	Info   *Info                  // Info is the info of the DSL, if not provided, the DSL will be loaded from the file system, one of info or source must be provided
	Source string                 // Source is the source of the DSL, if not provided, the DSL will be loaded from the file system, one of info or source must be provided
	Reload map[string]interface{} // ReloadOptions is the options for the DSL, if not provided, the DSL will be loaded from the file system
}

// DeleteOptions for DSL delete options
type DeleteOptions struct {
	ID      string                 // ID is the id of the DSL, if not provided, a new id will be generated, required
	Path    string                 // Path is the path of the DSL, if not provided, the DSL will be loaded from the file system
	Options map[string]interface{} // Options is the options for the DSL, if not provided, the DSL will be loaded from the file system
}

// LoadOptions for DSL load options
type LoadOptions struct {
	ID      string
	Path    string
	Source  string
	Store   StoreType
	Options map[string]interface{}
}

// UnloadOptions for DSL unload options
type UnloadOptions struct {
	ID      string
	Path    string
	Store   StoreType
	Options map[string]interface{}
}

// ReloadOptions for DSL reload options
type ReloadOptions struct {
	ID      string
	Path    string
	Source  string
	Store   StoreType
	Options map[string]interface{}
}

// LintMessage for DSL linter
type LintMessage struct {
	File     string
	Line     int
	Column   int
	Message  string
	Severity LintSeverity
}

var lintMessages []LintMessage
