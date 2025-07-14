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
	ID          string
	Type        Type
	Sort        int
	Path        string
	Label       string
	Description string
	Tags        []string
	Status      Status
	Store       StoreType
	Mtime       time.Time
	Ctime       time.Time
}

// ListOptions for DSL list
type ListOptions struct {
	Sort  string
	Order string
	Tags  []string
}

// CreateOptions for DSL upsert
type CreateOptions struct {
	ID          string      // ID is the id of the DSL, if not provided, a new id will be generated, required
	Source      string      // Source is the source of the DSL, if not provided, the DSL will be loaded from the file system
	Store       StoreType   // Store is the store type of the DSL, if not provided, the DSL will be loaded from the file system
	LoadOptions interface{} // LoadOptions is the options for the DSL, if not provided, the DSL will be loaded from the file system
}

// UpdateOptions for DSL upsert
type UpdateOptions struct {
	ID            string      // ID is the id of the DSL, if not provided, a new id will be generated, required
	Info          *Info       // Info is the info of the DSL, if not provided, the DSL will be loaded from the file system, one of info or source must be provided
	Source        string      // Source is the source of the DSL, if not provided, the DSL will be loaded from the file system, one of info or source must be provided
	ReloadOptions interface{} // ReloadOptions is the options for the DSL, if not provided, the DSL will be loaded from the file system
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
