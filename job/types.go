package job

import (
	"context"
	"encoding/json"
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

// HandlerFunc the job handler function
type HandlerFunc func(ctx context.Context, execution *Execution) error

// Job represents the main job entity
type Job struct {
	ID                 uint                   `json:"id"`
	JobID              string                 `json:"job_id"`
	Name               string                 `json:"name"`
	Icon               *string                `json:"icon,omitempty"`        // nullable: true
	Description        *string                `json:"description,omitempty"` // nullable: true
	CategoryID         string                 `json:"category_id"`
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

	// Relationships
	Category   *Category   `json:"category,omitempty"`
	Executions []Execution `json:"executions,omitempty"`
	Logs       []Log       `json:"logs,omitempty"`

	ctx           context.Context
	cancel        context.CancelFunc
	workerManager *WorkerManager // For testing: allows using custom worker manager
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
	ID                uint             `json:"id"`
	ExecutionID       string           `json:"execution_id"`
	JobID             string           `json:"job_id"`
	Status            string           `json:"status"` // default: "queued"
	TriggerCategory   string           `json:"trigger_category"`
	TriggerSource     *string          `json:"trigger_source,omitempty"`      // nullable: true
	TriggerContext    *json.RawMessage `json:"trigger_context,omitempty"`     // nullable: true
	ScheduledAt       *time.Time       `json:"scheduled_at,omitempty"`        // nullable: true
	WorkerID          *string          `json:"worker_id,omitempty"`           // nullable: true
	ProcessID         *string          `json:"process_id,omitempty"`          // nullable: true
	RetryAttempt      int              `json:"retry_attempt"`                 // default: 0
	ParentExecutionID *string          `json:"parent_execution_id,omitempty"` // nullable: true
	StartedAt         *time.Time       `json:"started_at,omitempty"`          // nullable: true
	EndedAt           *time.Time       `json:"ended_at,omitempty"`            // nullable: true
	TimeoutSeconds    *int             `json:"timeout_seconds,omitempty"`     // nullable: true
	Duration          *int             `json:"duration,omitempty"`            // nullable: true
	Progress          int              `json:"progress"`                      // default: 0
	ConfigSnapshot    *json.RawMessage `json:"config_snapshot,omitempty"`     // nullable: true
	Result            *json.RawMessage `json:"result,omitempty"`              // nullable: true
	ErrorInfo         *json.RawMessage `json:"error_info,omitempty"`          // nullable: true
	StackTrace        *string          `json:"stack_trace,omitempty"`         // nullable: true
	Metrics           *json.RawMessage `json:"metrics,omitempty"`             // nullable: true
	Context           *json.RawMessage `json:"context,omitempty"`             // nullable: true
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`

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
