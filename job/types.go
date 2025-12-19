package job

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// ScheduleType the schedule type
type ScheduleType string

// ScheduleType constants
const (
	ScheduleTypeOnce   ScheduleType = "once"
	ScheduleTypeCron   ScheduleType = "cron"
	ScheduleTypeDaemon ScheduleType = "daemon"
)

// ModeType tye execution mode
type ModeType string

// ModeType constants
const (
	GOROUTINE ModeType = "GOROUTINE" // Execute using Go goroutine (lightweight, fast)
	PROCESS   ModeType = "PROCESS"   // Independent process isolated
)

// LogLevel the log level
type LogLevel uint8

// These are the different logging levels. You can set the logging level to log
// on your instance of logger, obtained with `logrus.New()`.
const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	Panic LogLevel = iota
	// FatalLevel level. Logs and then calls `logger.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	Fatal
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	Error
	// WarnLevel level. Non-critical entries that deserve eyes.
	Warn
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	Info
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	Debug
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	Trace
)

// ExecutionType represents different execution methods
type ExecutionType string

// Execution type constants
const (
	ExecutionTypeProcess ExecutionType = "process" // Yao process (default)
	ExecutionTypeCommand ExecutionType = "command" // System command
	ExecutionTypeFunc    ExecutionType = "func"    // Go function (internal use)
)

// ExecutionOptions holds common execution options
type ExecutionOptions struct {
	Priority   int                    `json:"priority"`    // Execution priority (higher = more important)
	SharedData map[string]interface{} `json:"shared_data"` // Shared data (session, context, etc.)
}

// NewExecutionOptions creates a new ExecutionOptions with default values
func NewExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		Priority:   0,
		SharedData: make(map[string]interface{}),
	}
}

// WithPriority sets the priority and returns the options for chaining
func (o *ExecutionOptions) WithPriority(priority int) *ExecutionOptions {
	o.Priority = priority
	return o
}

// WithSharedData sets shared data and returns the options for chaining
func (o *ExecutionOptions) WithSharedData(data map[string]interface{}) *ExecutionOptions {
	o.SharedData = data
	return o
}

// AddSharedData adds a key-value pair to shared data and returns the options for chaining
func (o *ExecutionOptions) AddSharedData(key string, value interface{}) *ExecutionOptions {
	if o.SharedData == nil {
		o.SharedData = make(map[string]interface{})
	}
	o.SharedData[key] = value
	return o
}

// ExecutionFunc is the function signature for ExecutionTypeFunc
// The function receives the execution context and returns an error if failed
type ExecutionFunc func(ctx *ExecutionContext) error

// ExecutionContext provides context for ExecutionFunc
type ExecutionContext struct {
	Ctx       context.Context        // Go context
	Execution *Execution             // Current execution
	Args      map[string]interface{} // Function arguments
}

// funcRegistry is a global registry for ExecutionFunc
// Key is the funcID (execution_id), value is the function
var funcRegistry = make(map[string]ExecutionFunc)
var funcRegistryMutex sync.RWMutex

// RegisterFunc registers a function in the global registry
func RegisterFunc(funcID string, fn ExecutionFunc) {
	funcRegistryMutex.Lock()
	defer funcRegistryMutex.Unlock()
	funcRegistry[funcID] = fn
}

// GetFunc retrieves a function from the global registry
func GetFunc(funcID string) (ExecutionFunc, bool) {
	funcRegistryMutex.RLock()
	defer funcRegistryMutex.RUnlock()
	fn, ok := funcRegistry[funcID]
	return fn, ok
}

// UnregisterFunc removes a function from the global registry
func UnregisterFunc(funcID string) {
	funcRegistryMutex.Lock()
	defer funcRegistryMutex.Unlock()
	delete(funcRegistry, funcID)
}

// ExecutionConfig holds execution configuration based on type
type ExecutionConfig struct {
	Type        ExecutionType          `json:"type"`
	ProcessName string                 `json:"process_name,omitempty"` // Yao process name
	ProcessArgs []interface{}          `json:"process_args,omitempty"` // Yao process arguments
	Command     string                 `json:"command,omitempty"`      // System command
	CommandArgs []string               `json:"command_args,omitempty"` // Command arguments
	Environment map[string]string      `json:"environment,omitempty"`  // Environment variables
	Func        ExecutionFunc          `json:"-"`                      // Go function (not serialized, use FuncID instead)
	FuncID      string                 `json:"func_id,omitempty"`      // Function ID for registry lookup
	FuncName    string                 `json:"func_name,omitempty"`    // Function name for logging
	FuncArgs    map[string]interface{} `json:"func_args,omitempty"`    // Function arguments
}

// Job represents the main job entity
type Job struct {
	ID                 uint                   `json:"id"`
	JobID              string                 `json:"job_id"`
	Name               string                 `json:"name"`
	Icon               *string                `json:"icon,omitempty"`        // nullable: true
	Description        *string                `json:"description,omitempty"` // nullable: true
	CategoryID         string                 `json:"category_id"`
	CategoryName       string                 `json:"category_name,omitempty"`
	MaxWorkerNums      int                    `json:"max_worker_nums"`               // default: 1
	Status             string                 `json:"status"`                        // default: "draft"
	Mode               ModeType               `json:"mode"`                          // default: "goroutine"
	ScheduleType       string                 `json:"schedule_type"`                 // default: "once"
	ScheduleExpression *string                `json:"schedule_expression,omitempty"` // nullable: true
	MaxRetryCount      int                    `json:"max_retry_count"`               // default: 0
	DefaultTimeout     *int                   `json:"default_timeout,omitempty"`     // nullable: true
	Priority           int                    `json:"priority"`                      // default: 0
	CreatedBy          string                 `json:"created_by"`
	NextRunAt          *time.Time             `json:"next_run_at,omitempty"`          // nullable: true
	LastRunAt          *time.Time             `json:"last_run_at,omitempty"`          // nullable: true
	CurrentExecutionID *string                `json:"current_execution_id,omitempty"` // nullable: true
	Config             map[string]interface{} `json:"config,omitempty"`               // nullable: true
	Sort               int                    `json:"sort"`                           // default: 0
	Enabled            bool                   `json:"enabled"`                        // default: true
	System             bool                   `json:"system"`                         // default: false
	Readonly           bool                   `json:"readonly"`                       // default: false
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`

	// Yao custom fields
	YaoCreatedBy string `json:"__yao_created_by,omitempty"` // nullable: true
	YaoUpdatedBy string `json:"__yao_updated_by,omitempty"` // nullable: true
	YaoTeamID    string `json:"__yao_team_id,omitempty"`    // nullable: true
	YaoTenantID  string `json:"__yao_tenant_id,omitempty"`

	// Relationships
	Category   *Category   `json:"category,omitempty"`
	Executions []Execution `json:"executions,omitempty"`
	Logs       []Log       `json:"logs,omitempty"`

	ctx    context.Context
	cancel context.CancelFunc

	// Job-level cancellation for running executions
	executionContexts map[string]context.CancelFunc // executionID -> cancel function
	executionMutex    sync.RWMutex
}

// Category represents job categories for organization
type Category struct {
	ID          uint      `json:"id"`
	CategoryID  string    `json:"category_id"`
	Name        string    `json:"name"`
	Icon        *string   `json:"icon,omitempty"`        // nullable: true
	Description *string   `json:"description,omitempty"` // nullable: true
	Sort        int       `json:"sort"`                  // default: 0
	System      bool      `json:"system"`                // default: false
	Enabled     bool      `json:"enabled"`               // default: true
	Readonly    bool      `json:"readonly"`              // default: false
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Jobs []Job `json:"jobs,omitempty"`
}

// Execution represents individual job execution instances
type Execution struct {
	ID                uint              `json:"id"`
	ExecutionID       string            `json:"execution_id"`
	JobID             string            `json:"job_id"`
	Status            string            `json:"status"` // default: "queued"
	TriggerCategory   string            `json:"trigger_category"`
	TriggerSource     *string           `json:"trigger_source,omitempty"`      // nullable: true
	TriggerContext    *json.RawMessage  `json:"trigger_context,omitempty"`     // nullable: true
	ScheduledAt       *time.Time        `json:"scheduled_at,omitempty"`        // nullable: true
	WorkerID          *string           `json:"worker_id,omitempty"`           // nullable: true
	ProcessID         *string           `json:"process_id,omitempty"`          // nullable: true
	RetryAttempt      int               `json:"retry_attempt"`                 // default: 0
	ParentExecutionID *string           `json:"parent_execution_id,omitempty"` // nullable: true
	StartedAt         *time.Time        `json:"started_at,omitempty"`          // nullable: true
	EndedAt           *time.Time        `json:"ended_at,omitempty"`            // nullable: true
	TimeoutSeconds    *int              `json:"timeout_seconds,omitempty"`     // nullable: true
	Duration          *int              `json:"duration,omitempty"`            // nullable: true
	Progress          int               `json:"progress"`                      // default: 0
	ExecutionConfig   *ExecutionConfig  `json:"execution_config,omitempty"`    // Execution configuration
	ExecutionOptions  *ExecutionOptions `json:"execution_options,omitempty"`   // Execution options (priority, shared data)
	ConfigSnapshot    *json.RawMessage  `json:"config_snapshot,omitempty"`     // nullable: true
	Result            *json.RawMessage  `json:"result,omitempty"`              // nullable: true
	ErrorInfo         *json.RawMessage  `json:"error_info,omitempty"`          // nullable: true
	StackTrace        *string           `json:"stack_trace,omitempty"`         // nullable: true
	Metrics           *json.RawMessage  `json:"metrics,omitempty"`             // nullable: true
	Context           *json.RawMessage  `json:"context,omitempty"`             // nullable: true
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`

	// Relationships
	Job             *Job        `json:"job,omitempty"`
	ParentExecution *Execution  `json:"parent_execution,omitempty"`
	ChildExecutions []Execution `json:"child_executions,omitempty"`
	Logs            []Log       `json:"logs,omitempty"`
}

// Log represents job execution logs and events
type Log struct {
	ID          uint             `json:"id"`
	JobID       string           `json:"job_id"`
	Level       string           `json:"level"` // default: "info"
	Message     string           `json:"message"`
	Context     *json.RawMessage `json:"context,omitempty"`      // nullable: true
	Source      *string          `json:"source,omitempty"`       // nullable: true
	ExecutionID *string          `json:"execution_id,omitempty"` // nullable: true
	Step        *string          `json:"step,omitempty"`         // nullable: true
	Progress    *int             `json:"progress,omitempty"`     // nullable: true
	Duration    *int             `json:"duration,omitempty"`     // nullable: true
	ErrorCode   *string          `json:"error_code,omitempty"`   // nullable: true
	StackTrace  *string          `json:"stack_trace,omitempty"`  // nullable: true
	WorkerID    *string          `json:"worker_id,omitempty"`    // nullable: true
	ProcessID   *string          `json:"process_id,omitempty"`   // nullable: true
	Timestamp   time.Time        `json:"timestamp"`              // default: "now()"
	Sequence    int              `json:"sequence"`               // default: 0
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`

	// Relationships
	Job       *Job       `json:"job,omitempty"`
	Execution *Execution `json:"execution,omitempty"`
}
