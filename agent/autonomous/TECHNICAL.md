# Autonomous Agent - Technical Design

## 1. Code Structure

```
yao/agent/autonomous/
├── DESIGN.md                 # Product design doc
├── TECHNICAL.md              # This file
│
├── autonomous.go             # Package entry, Init(), Shutdown()
│
├── api/                      # All API forms
│   ├── api.go                # Go API (facade)
│   ├── process.go            # Yao Process: autonomous.*
│   └── jsapi.go              # JS API for scripts
│
├── types/                    # Types only (no logic, no external deps)
│   ├── enums.go              # Phase, ClockMode, TriggerType, etc.
│   ├── config.go             # Config, Clock, Identity, Quota, etc.
│   ├── robot.go              # Robot, Execution
│   ├── task.go               # Goal, Task, TaskResult
│   ├── request.go            # InterveneRequest, EventRequest, etc.
│   ├── inspiration.go        # ClockContext, InspirationReport
│   ├── interfaces.go         # All interfaces (Manager, Trigger, etc.)
│   └── errors.go             # Error definitions
│
├── manager/                  # Manager package (orchestration)
│   ├── manager.go            # Manager struct, Start/Stop, ticker loop
│   └── lifecycle.go          # OnRobotCreate/Delete/Update
│
├── pool/                     # Worker pool & task dispatch
│   ├── pool.go               # Pool struct, Submit
│   ├── queue.go              # Priority queue
│   └── worker.go             # Worker goroutines
│
├── executor/                 # Executor package
│   ├── executor.go           # Executor struct, Execute
│   ├── phase.go              # RunPhase dispatcher
│   ├── inspiration.go        # P0: Inspiration (clock only)
│   ├── goals.go              # P1: Goal generation
│   ├── tasks.go              # P2: Task planning
│   ├── run.go                # P3: Task execution
│   ├── delivery.go           # P4: Delivery
│   ├── learning.go           # P5: Learning
│   ├── agent.go              # Call assistant/agent unified method
│   └── prompt.go             # Prompt building helpers
│
├── utils/                    # Utility functions
│   ├── convert.go            # Type conversions (JSON, map, struct)
│   ├── time.go               # Time parsing, formatting, timezone
│   ├── id.go                 # ID generation (nanoid, uuid)
│   └── validate.go           # Validation helpers
│
├── trigger/                  # All trigger sources
│   ├── trigger.go            # Trigger interface & dispatcher
│   ├── clock.go              # Clock trigger (tick, schedule matching)
│   ├── intervene.go          # Human intervention trigger
│   ├── event.go              # Event trigger (webhook, db change)
│   └── control.go            # Pause/Resume/Cancel
│
├── cache/                    # Cache package
│   ├── cache.go              # Cache struct, Get/List
│   ├── load.go               # LoadAll, LoadOne
│   └── refresh.go            # Refresh logic
│
├── dedup/                    # Deduplication package
│   ├── dedup.go              # Dedup struct
│   ├── fast.go               # Fast in-memory check
│   └── semantic.go           # Semantic check via agent
│
├── store/                    # Data store package (KB, FS, DB access)
│   ├── store.go              # Store struct, interface
│   ├── kb.go                 # Knowledge base operations
│   ├── fs.go                 # File system operations
│   ├── db.go                 # Database queries
│   └── learning.go           # Learning entry save (to KB)
│
├── job/                      # Job system integration
│   ├── job.go                # Create/Get job for robot
│   ├── execution.go          # Create/Update execution
│   └── log.go                # Write execution logs
│
└── plan/                     # Plan queue (deferred tasks)
    ├── plan.go               # Plan queue struct
    └── schedule.go           # Schedule for later
```

### Dependency Graph (No Cycles)

```
                              ┌──────────┐
                              │  types/  │  (pure types, no deps)
                              └────┬─────┘
                                   │
    ┌───────┬───────┬───────┬──────┼──────┬───────┬───────┬───────┐
    │       │       │       │      │      │       │       │       │
    ▼       ▼       ▼       ▼      ▼      ▼       ▼       ▼       ▼
┌───────┐┌───────┐┌───────┐┌──────┐┌────┐┌──────┐┌───────┐
│ cache ││ dedup ││ store ││ pool ││job ││ plan ││ utils │
└───┬───┘└───┬───┘└───┬───┘└──┬───┘└──┬─┘└──────┘└───────┘
    │        │        │       │       │
    └────────┴────────┴───────┴───────┘
                      │
       ┌──────────────┴──────────────┐
       │                             │
       ▼                             ▼
┌────────────┐                ┌────────────┐
│  trigger/  │                │  executor/ │
└──────┬─────┘                └──────┬─────┘
       │                             │
       └──────────────┬──────────────┘
                      │
                      ▼
               ┌────────────┐
               │  manager/  │
               └──────┬─────┘
                      │
       ┌──────────────┴──────────────┐
       │                             │
       ▼                             ▼
┌─────────────┐              ┌─────────────┐
│autonomous.go│              │    api/     │
└─────────────┘              └─────────────┘
```

### Package Dependencies

| Package     | Imports                                                 |
| ----------- | ------------------------------------------------------- |
| `types/`    | stdlib only                                             |
| `utils/`    | stdlib only                                             |
| `cache/`    | `types/`                                                |
| `dedup/`    | `types/`                                                |
| `store/`    | `types/`                                                |
| `pool/`     | `types/`                                                |
| `trigger/`  | `types/`, `cache/`                                      |
| `job/`      | `types/`, `yao/job`                                     |
| `plan/`     | `types/`                                                |
| `executor/` | `types/`, `cache/`, `dedup/`, `store/`, `pool/`, `job/` |
| `manager/`  | `types/`, `cache/`, `pool/`, `trigger/`, `executor/`    |
| `api/`      | `types/`, `manager/`, `trigger/`                        |
| root        | all packages                                            |

### Public API (`api/`)

三种 API 形态，统一放在 `api/` 目录下：

#### Go API (`api/api.go`)

```go
package api

// Robot Management
func GetRobot(memberID string) (*types.Robot, error)
func ListRobots(teamID string) ([]*types.Robot, error)
func RefreshRobot(memberID string) error

// Triggers
func Intervene(req *types.InterveneRequest) (*types.ExecutionResult, error)
func HandleEvent(req *types.EventRequest) (*types.ExecutionResult, error)

// Control
func GetStatus(memberID string) (*types.RobotState, error)
func Pause(memberID string) error
func Resume(memberID string) error
func Cancel(memberID, executionID string) error

// Query
func GetExecution(executionID string) (*types.Execution, error)
func ListExecutions(memberID string, limit int) ([]*types.Execution, error)
```

#### Yao Process (`api/process.go`)

```go
// autonomous.GetRobot(memberID) -> Robot
// autonomous.ListRobots(teamID) -> []Robot
// autonomous.Intervene(teamID, memberID, action, description, priority) -> ExecutionResult
// autonomous.HandleEvent(memberID, source, eventType, data) -> ExecutionResult
// autonomous.GetStatus(memberID) -> RobotState
// autonomous.Pause(memberID) -> bool
// autonomous.Resume(memberID) -> bool
// autonomous.Cancel(memberID, executionID) -> bool
// autonomous.GetExecution(executionID) -> Execution
// autonomous.ListExecutions(memberID, limit) -> []Execution
```

#### JS API (`api/jsapi.go`)

```js
// In Yao scripts
const robot = $autonomous.GetRobot("member_123");
const result = $autonomous.Intervene({
  member_id: "member_123",
  action: "add_task",
  description: "Prepare report",
});
const status = $autonomous.GetStatus("member_123");
```

---

## 2. Type Definitions

> All types are in `autonomous/types/` package. Other files import as:
>
> ```go
> import "github.com/yaoapp/yao/agent/autonomous/types"
> ```

### 2.1 Enums

```go
// types/enums.go
package types

// Phase - execution phase
type Phase string

const (
    PhaseInspiration Phase = "inspiration" // P0: Clock only
    PhaseGoals       Phase = "goals"       // P1
    PhaseTasks       Phase = "tasks"       // P2
    PhaseRun         Phase = "run"         // P3
    PhaseDelivery    Phase = "delivery"    // P4
    PhaseLearning    Phase = "learning"    // P5
)

// AllPhases for iteration
var AllPhases = []Phase{
    PhaseInspiration, PhaseGoals, PhaseTasks,
    PhaseRun, PhaseDelivery, PhaseLearning,
}

// ClockMode - clock trigger mode
type ClockMode string

const (
    ClockTimes    ClockMode = "times"    // run at specific times
    ClockInterval ClockMode = "interval" // run every X duration
    ClockDaemon   ClockMode = "daemon"   // run continuously
)

// TriggerType - trigger source
type TriggerType string

const (
    TriggerClock TriggerType = "clock"
    TriggerHuman TriggerType = "human"
    TriggerEvent TriggerType = "event"
)

// ExecStatus - execution status
type ExecStatus string

const (
    ExecPending   ExecStatus = "pending"
    ExecRunning   ExecStatus = "running"
    ExecCompleted ExecStatus = "completed"
    ExecFailed    ExecStatus = "failed"
    ExecCancelled ExecStatus = "cancelled"
)

// RobotStatus - matches __yao.member.robot_status
type RobotStatus string

const (
    RobotIdle        RobotStatus = "idle"
    RobotWorking     RobotStatus = "working"
    RobotPaused      RobotStatus = "paused"
    RobotError       RobotStatus = "error"
    RobotMaintenance RobotStatus = "maintenance"
)

// DeliveryType - output delivery type
type DeliveryType string

const (
    DeliveryEmail   DeliveryType = "email"
    DeliveryFile    DeliveryType = "file"
    DeliveryWebhook DeliveryType = "webhook"
    DeliveryNotify  DeliveryType = "notify"
)

// DedupResult - deduplication result
type DedupResult string

const (
    DedupSkip    DedupResult = "skip"    // skip execution
    DedupMerge   DedupResult = "merge"   // merge with existing
    DedupProceed DedupResult = "proceed" // proceed normally
)

// InterventionAction - human intervention actions
type InterventionAction string

const (
    ActionAddTask    InterventionAction = "add_task"
    ActionAdjustGoal InterventionAction = "adjust_goal"
    ActionCancelTask InterventionAction = "cancel_task"
    ActionPause      InterventionAction = "pause"
    ActionResume     InterventionAction = "resume"
    ActionAbort      InterventionAction = "abort"
    ActionPlan       InterventionAction = "plan"
)

// Priority levels
type Priority string

const (
    PriorityHigh   Priority = "high"
    PriorityNormal Priority = "normal"
    PriorityLow    Priority = "low"
)
```

### 2.2 Config Types

```go
// types/config.go
package types

import "time"

// Config - robot_config in __yao.member
type Config struct {
    Triggers  *Triggers  `json:"triggers,omitempty"`
    Clock     *Clock     `json:"clock,omitempty"`
    Identity  *Identity  `json:"identity"`
    Quota     *Quota     `json:"quota,omitempty"`
    PrivateKB *KBConfig  `json:"private_kb,omitempty"`
    SharedKB  *KBConfig  `json:"shared_kb,omitempty"`
    Resources *Resources `json:"resources,omitempty"`
    Delivery  *Delivery  `json:"delivery,omitempty"`
    Events    []Event    `json:"events,omitempty"`
    Monitor   *Monitor   `json:"monitor,omitempty"`
}

// Validate validates the config
func (c *Config) Validate() error {
    if c.Identity == nil || c.Identity.Role == "" {
        return ErrMissingIdentity
    }
    if c.Clock != nil {
        if err := c.Clock.Validate(); err != nil {
            return err
        }
    }
    return nil
}

// Triggers - trigger enable/disable
type Triggers struct {
    Clock     *TriggerSwitch `json:"clock,omitempty"`
    Intervene *TriggerSwitch `json:"intervene,omitempty"`
    Event     *TriggerSwitch `json:"event,omitempty"`
}

type TriggerSwitch struct {
    Enabled bool     `json:"enabled"`
    Actions []string `json:"actions,omitempty"` // for intervene
}

// IsEnabled checks if trigger is enabled (default: true)
func (t *Triggers) IsEnabled(typ TriggerType) bool {
    if t == nil {
        return true
    }
    switch typ {
    case TriggerClock:
        return t.Clock == nil || t.Clock.Enabled
    case TriggerHuman:
        return t.Intervene == nil || t.Intervene.Enabled
    case TriggerEvent:
        return t.Event == nil || t.Event.Enabled
    }
    return false
}

// Clock - when to wake up
type Clock struct {
    Mode    ClockMode `json:"mode"`              // times | interval | daemon
    Times   []string  `json:"times,omitempty"`   // ["09:00", "14:00"]
    Days    []string  `json:"days,omitempty"`    // ["Mon", "Tue"] or ["*"]
    Every   string    `json:"every,omitempty"`   // "30m", "1h"
    TZ      string    `json:"tz,omitempty"`      // "Asia/Shanghai"
    Timeout string    `json:"timeout,omitempty"` // "30m"
}

// Validate validates clock config
func (c *Clock) Validate() error {
    switch c.Mode {
    case ClockTimes:
        if len(c.Times) == 0 {
            return ErrClockTimesEmpty
        }
    case ClockInterval:
        if c.Every == "" {
            return ErrClockIntervalEmpty
        }
    case ClockDaemon:
        // no extra validation
    default:
        return ErrClockModeInvalid
    }
    return nil
}

// GetTimeout returns parsed timeout duration
func (c *Clock) GetTimeout() time.Duration {
    if c.Timeout == "" {
        return 30 * time.Minute // default
    }
    d, err := time.ParseDuration(c.Timeout)
    if err != nil {
        return 30 * time.Minute
    }
    return d
}

// GetLocation returns timezone location
func (c *Clock) GetLocation() *time.Location {
    if c.TZ == "" {
        return time.Local
    }
    loc, err := time.LoadLocation(c.TZ)
    if err != nil {
        return time.Local
    }
    return loc
}

// Identity - who is this robot
type Identity struct {
    Role   string   `json:"role"`
    Duties []string `json:"duties,omitempty"`
    Rules  []string `json:"rules,omitempty"`
}

// Quota - concurrency limits
type Quota struct {
    Max      int `json:"max"`      // max running (default: 2)
    Queue    int `json:"queue"`    // queue size (default: 10)
    Priority int `json:"priority"` // 1-10 (default: 5)
}

// GetMax returns max with default
func (q *Quota) GetMax() int {
    if q == nil || q.Max <= 0 {
        return 2
    }
    return q.Max
}

// GetQueue returns queue size with default
func (q *Quota) GetQueue() int {
    if q == nil || q.Queue <= 0 {
        return 10
    }
    return q.Queue
}

// GetPriority returns priority with default
func (q *Quota) GetPriority() int {
    if q == nil || q.Priority <= 0 {
        return 5
    }
    return q.Priority
}

// KBConfig - knowledge base config
type KBConfig struct {
    ID    string   `json:"id,omitempty"`
    Refs  []string `json:"refs,omitempty"`
    Learn *Learn   `json:"learn,omitempty"`
}

type Learn struct {
    On    bool     `json:"on"`
    Types []string `json:"types,omitempty"` // execution, feedback, insight
    Keep  int      `json:"keep,omitempty"`  // days, 0 = forever
}

// Resources - available agents and tools
type Resources struct {
    Phases map[Phase]string `json:"phases,omitempty"` // phase -> agent ID
    Agents []string         `json:"agents,omitempty"`
    MCP    []MCPConfig      `json:"mcp,omitempty"`
}

// GetPhaseAgent returns agent ID for phase (default: __yao.{phase})
func (r *Resources) GetPhaseAgent(phase Phase) string {
    if r != nil && r.Phases != nil {
        if id, ok := r.Phases[phase]; ok && id != "" {
            return id
        }
    }
    return "__yao." + string(phase)
}

type MCPConfig struct {
    ID    string   `json:"id"`
    Tools []string `json:"tools,omitempty"` // empty = all
}

// Delivery - output delivery
type Delivery struct {
    Type DeliveryType           `json:"type"`
    Opts map[string]interface{} `json:"opts,omitempty"`
}

// Event - event trigger config
type Event struct {
    Type   string                 `json:"type"`   // webhook | database
    Source string                 `json:"source"` // path or table
    Filter map[string]interface{} `json:"filter,omitempty"`
}

// Monitor - monitoring config
type Monitor struct {
    On     bool    `json:"on"`
    Alerts []Alert `json:"alerts,omitempty"`
}

type Alert struct {
    Name     string   `json:"name"`
    When     string   `json:"when"`  // failed | timeout | error_rate
    Value    float64  `json:"value,omitempty"`
    Window   string   `json:"window,omitempty"` // 1h | 24h
    Do       []Action `json:"do"`
    Cooldown string   `json:"cooldown,omitempty"`
}

type Action struct {
    Type string                 `json:"type"` // email | webhook | notify
    Opts map[string]interface{} `json:"opts,omitempty"`
}
```

### 2.3 Core Types

```go
// types/robot.go
package types

import (
    "context"
    "sync"
    "time"
)

// Robot - runtime representation of an autonomous robot
type Robot struct {
    // From __yao.member
    MemberID       string      `json:"member_id"`
    TeamID         string      `json:"team_id"`
    DisplayName    string      `json:"display_name"`
    SystemPrompt   string      `json:"system_prompt"`
    Status         RobotStatus `json:"robot_status"`
    AutonomousMode bool        `json:"autonomous_mode"`

    // Parsed config
    Config *Config `json:"-"`

    // Runtime state (job.Job stored as interface{} to avoid import cycle)
    Job           interface{} `json:"-"` // *job.Job, set by manager
    JobID         string      `json:"-"` // job_id for quick access
    LastExecution time.Time   `json:"-"`
    NextExecution time.Time   `json:"-"`

    // Concurrency control
    running   int
    runningMu sync.Mutex
}

// CanRun checks if robot can accept new execution
func (r *Robot) CanRun() bool {
    r.runningMu.Lock()
    defer r.runningMu.Unlock()
    return r.running < r.Config.Quota.GetMax()
}

// IncrRunning increments running count
func (r *Robot) IncrRunning() {
    r.runningMu.Lock()
    defer r.runningMu.Unlock()
    r.running++
}

// DecrRunning decrements running count
func (r *Robot) DecrRunning() {
    r.runningMu.Lock()
    defer r.runningMu.Unlock()
    if r.running > 0 {
        r.running--
    }
}

// Execution - single execution context
type Execution struct {
    ID          string      `json:"id"`
    MemberID    string      `json:"member_id"`
    TeamID      string      `json:"team_id"`
    TriggerType TriggerType `json:"trigger_type"`
    TriggerData interface{} `json:"trigger_data,omitempty"`
    StartTime   time.Time   `json:"start_time"`
    EndTime     *time.Time  `json:"end_time,omitempty"`
    Status      ExecStatus  `json:"status"`
    Phase       Phase       `json:"phase"`
    Error       string      `json:"error,omitempty"`

    // Phase outputs
    Inspiration *InspirationReport `json:"inspiration,omitempty"`
    Goals       []Goal             `json:"goals,omitempty"`
    Tasks       []Task             `json:"tasks,omitempty"`
    Results     []TaskResult       `json:"results,omitempty"`
    Delivery    *DeliveryResult    `json:"delivery,omitempty"`
    Learning    []LearningEntry    `json:"learning,omitempty"`

    // Context
    ctx    context.Context
    cancel context.CancelFunc
    robot  *Robot
}

// Goal - generated goal
type Goal struct {
    ID          string   `json:"id"`
    Description string   `json:"description"`
    Priority    Priority `json:"priority"`
    Rationale   string   `json:"rationale,omitempty"`
    Tags        []string `json:"tags,omitempty"`
}

// Task - planned task
type Task struct {
    ID           string     `json:"id"`
    GoalID       string     `json:"goal_id"`
    Description  string     `json:"description"`
    ExecutorType string     `json:"executor_type"` // "assistant" | "mcp"
    ExecutorID   string     `json:"executor_id"`
    Args         []any      `json:"args,omitempty"`
    Status       ExecStatus `json:"status"`
    Order        int        `json:"order"`
}

// TaskResult - task execution result
type TaskResult struct {
    TaskID    string      `json:"task_id"`
    Success   bool        `json:"success"`
    Output    interface{} `json:"output,omitempty"`
    Error     string      `json:"error,omitempty"`
    Duration  int64       `json:"duration_ms"`
    Validated bool        `json:"validated"`
}

// DeliveryResult - delivery output
type DeliveryResult struct {
    Type    DeliveryType `json:"type"`
    Success bool         `json:"success"`
    Details interface{}  `json:"details,omitempty"`
    Error   string       `json:"error,omitempty"`
}

// LearningEntry - knowledge to save
type LearningEntry struct {
    Type    string      `json:"type"` // execution | feedback | insight
    Content string      `json:"content"`
    Tags    []string    `json:"tags,omitempty"`
    Meta    interface{} `json:"meta,omitempty"`
}
```

### 2.4 Clock Context

```go
// types/clock.go
package types

import "time"

// ClockContext - time context for P0 inspiration
type ClockContext struct {
    Now          time.Time `json:"now"`
    Hour         int       `json:"hour"`          // 0-23
    DayOfWeek    string    `json:"day_of_week"`   // Monday, Tuesday...
    DayOfMonth   int       `json:"day_of_month"`  // 1-31
    WeekOfYear   int       `json:"week_of_year"`  // 1-52
    Month        int       `json:"month"`         // 1-12
    Year         int       `json:"year"`
    IsWeekend    bool      `json:"is_weekend"`
    IsMonthStart bool      `json:"is_month_start"` // 1st-3rd
    IsMonthEnd   bool      `json:"is_month_end"`   // last 3 days
    IsQuarterEnd bool      `json:"is_quarter_end"`
    IsYearEnd    bool      `json:"is_year_end"`
    TZ           string    `json:"tz"`
}

// NewClockContext creates clock context from time
func NewClockContext(t time.Time, tz string) *ClockContext {
    loc := time.Local
    if tz != "" {
        if l, err := time.LoadLocation(tz); err == nil {
            loc = l
        }
    }
    t = t.In(loc)

    _, week := t.ISOWeek()
    dayOfMonth := t.Day()
    lastDay := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, loc).Day()

    return &ClockContext{
        Now:          t,
        Hour:         t.Hour(),
        DayOfWeek:    t.Weekday().String(),
        DayOfMonth:   dayOfMonth,
        WeekOfYear:   week,
        Month:        int(t.Month()),
        Year:         t.Year(),
        IsWeekend:    t.Weekday() == time.Saturday || t.Weekday() == time.Sunday,
        IsMonthStart: dayOfMonth <= 3,
        IsMonthEnd:   dayOfMonth >= lastDay-2,
        IsQuarterEnd: (t.Month()%3 == 0) && dayOfMonth >= lastDay-2,
        IsYearEnd:    t.Month() == 12 && dayOfMonth >= 29,
        TZ:           loc.String(),
    }
}
```

### 2.5 Inspiration Report

```go
// types/inspiration.go
package types

// InspirationReport - P0 output
type InspirationReport struct {
    Clock         *ClockContext `json:"clock"`
    Summary       string                   `json:"summary"`
    Highlights    []Highlight              `json:"highlights,omitempty"`
    Opportunities []Opportunity            `json:"opportunities,omitempty"`
    Risks         []Risk                   `json:"risks,omitempty"`
    WorldInsights []WorldInsight           `json:"world_insights,omitempty"`
    Suggestions   []string                 `json:"suggestions,omitempty"`
    PendingItems  []PendingItem            `json:"pending_items,omitempty"`
}

type Highlight struct {
    Source   string `json:"source"`   // data | event | feedback
    Priority string `json:"priority"` // high | medium | low
    Content  string `json:"content"`
    Change   string `json:"change,omitempty"` // +50%, -20%, etc.
}

type Opportunity struct {
    Description string `json:"description"`
    Impact      string `json:"impact"` // high | medium | low
    Urgency     string `json:"urgency"`
}

type Risk struct {
    Description string `json:"description"`
    Severity    string `json:"severity"` // high | medium | low
    Mitigation  string `json:"mitigation,omitempty"`
}

type WorldInsight struct {
    Source  string `json:"source"` // news | competitor | industry
    Title   string `json:"title"`
    Summary string `json:"summary"`
    Impact  string `json:"impact,omitempty"`
    URL     string `json:"url,omitempty"`
}

type PendingItem struct {
    Type        string `json:"type"` // goal | task | plan
    ID          string `json:"id"`
    Description string `json:"description"`
    DueDate     string `json:"due_date,omitempty"`
}
```

### 2.6 Request/Response Types

```go
// types/request.go
package types

import (
    "context"
    "time"
)

// InterveneRequest - human intervention request
type InterveneRequest struct {
    TeamID      string             `json:"team_id"`
    MemberID    string             `json:"member_id"`
    Action      InterventionAction `json:"action"`
    Description string             `json:"description"`
    Priority    Priority           `json:"priority,omitempty"`
    PlanTime    *time.Time         `json:"plan_time,omitempty"` // for action=plan
}

// EventRequest - event trigger request
type EventRequest struct {
    MemberID  string                 `json:"member_id"`
    Source    string                 `json:"source"`     // webhook path or table name
    EventType string                 `json:"event_type"` // lead.created, etc.
    Data      map[string]interface{} `json:"data"`
}

// ExecutionResult - trigger result
type ExecutionResult struct {
    ExecutionID string     `json:"execution_id"`
    Status      ExecStatus `json:"status"`
    Message     string     `json:"message,omitempty"`
}

// RobotState - robot status query result
type RobotState struct {
    MemberID    string      `json:"member_id"`
    TeamID      string      `json:"team_id"`
    DisplayName string      `json:"display_name"`
    Status      RobotStatus `json:"status"`
    LastRun     *time.Time  `json:"last_run,omitempty"`
    NextRun     *time.Time  `json:"next_run,omitempty"`
    RunningID   string      `json:"running_id,omitempty"` // current execution ID
    RunningCnt  int         `json:"running_cnt"`          // current running count
}
```

---

## 3. Interfaces

> Interfaces are also in `types/` package to avoid cycles.

### 3.1 Manager Interface

```go
// types/interfaces.go
package types

import (
    "context"
    "time"
)

// Manager - manages all robots
type Manager interface {
    // Lifecycle
    Start() error
    Stop() error

    // Cache operations
    LoadActiveRobots(ctx context.Context) error
    GetRobot(teamID, memberID string) *Robot
    ListRobots(teamID string) []*Robot
    RefreshRobot(teamID, memberID string) error

    // Clock trigger (called by internal ticker)
    Tick(ctx context.Context, now time.Time) error

    // Robot lifecycle (called when member created/deleted)
    OnRobotCreate(ctx context.Context, teamID, memberID string) error
    OnRobotDelete(ctx context.Context, teamID, memberID string) error
    OnRobotUpdate(ctx context.Context, teamID, memberID string) error
}
```

### 3.2 Trigger Interface

```go
// types/interfaces.go (continued)
package types

import "context"

// Trigger - called by openapi layer
type Trigger interface {
    // Human intervention
    Intervene(ctx context.Context, req *InterveneRequest) (*ExecutionResult, error)

    // Event trigger
    HandleEvent(ctx context.Context, req *EventRequest) (*ExecutionResult, error)

    // Query & control
    GetStatus(ctx context.Context, teamID, memberID string) (*RobotState, error)
    Pause(ctx context.Context, teamID, memberID string) error
    Resume(ctx context.Context, teamID, memberID string) error
    Cancel(ctx context.Context, teamID, memberID, executionID string) error
}
```

### 3.3 Executor Interface

```go
// types/interfaces.go (continued)
package types

import "context"

// Executor - executes robot phases
type Executor interface {
    // Execute runs all phases for a trigger
    Execute(ctx context.Context, robot *Robot, triggerType TriggerType, triggerData interface{}) (*Execution, error)

    // Individual phase execution (for testing/debugging)
    RunPhase(ctx context.Context, exec *Execution, phase Phase) error
}
```

### 3.4 Phase Interface

```go
// types/interfaces.go (continued)
package types

// PhaseExecutor - phase executor interface
type PhaseExecutor interface {
    // Name returns phase name
    Name() Phase

    // Execute runs the phase
    Execute(ctx context.Context, exec *Execution) error
}
```

### 3.5 Cache Interface

```go
// types/interfaces.go (continued)
package types

import "context"

// Cache - robot cache interface
type Cache interface {
    // Load all active robots
    LoadAll(ctx context.Context) error

    // Get robot by ID
    Get(teamID, memberID string) *Robot

    // List robots by team
    List(teamID string) []*Robot

    // Add/Update/Remove
    Add(robot *Robot)
    Update(robot *Robot)
    Remove(teamID, memberID string)

    // Stats
    Count() int
    CountByTeam(teamID string) int
}
```

### 3.6 Scheduler Interface

```go
// types/interfaces.go (continued)
package types

import "context"

// Scheduler - worker pool and queue
type Scheduler interface {
    // Start/Stop
    Start() error
    Stop() error

    // Submit execution
    Submit(ctx context.Context, robot *Robot, triggerType TriggerType, triggerData interface{}) error

    // Queue status
    QueueSize() int
    WorkerCount() int
    ActiveCount() int
}

// SchedulerConfig - scheduler configuration
type SchedulerConfig struct {
    Workers    int // global worker count (default: 10)
    QueueSize  int // global queue size (default: 1000)
    MaxPerTeam int // max concurrent per team (default: 20)
}
```

### 3.7 Dedup Interface

```go
// types/interfaces.go (continued)
package types

import (
    "context"
    "time"
)

// Dedup - deduplication service
type Dedup interface {
    // CheckExecution - fast check for duplicate execution
    CheckExecution(ctx context.Context, memberID string, triggerType TriggerType) (DedupResult, error)

    // CheckGoal - semantic check for duplicate goal
    CheckGoal(ctx context.Context, memberID string, goal *Goal) (DedupResult, error)

    // CheckTask - semantic check for duplicate task
    CheckTask(ctx context.Context, memberID string, task *Task) (DedupResult, error)

    // MarkExecuted - mark execution as done
    MarkExecuted(ctx context.Context, memberID string, triggerType TriggerType, window time.Duration)
}
```

---

## 4. Key Implementations

### 4.1 Manager Implementation

```go
// manager.go
package autonomous

import (
    "context"
    "sync"
    "time"

    "github.com/yaoapp/kun/log"
    "github.com/yaoapp/yao/agent/autonomous/types"
)

// Ensure manager implements types.Manager
var _ types.Manager = (*manager)(nil)

type manager struct {
    cache     types.Cache
    scheduler types.Scheduler
    dedup     types.Dedup
    executor  types.Executor

    ticker   *time.Ticker
    tickerMu sync.Mutex
    stopCh   chan struct{}
    wg       sync.WaitGroup
}

// NewManager creates a new manager
func NewManager(cfg *types.SchedulerConfig) types.Manager {
    return &manager{
        cache:     newCache(),
        scheduler: newScheduler(cfg),
        dedup:     newDedup(),
        executor:  newExecutor(),
        stopCh:    make(chan struct{}),
    }
}

func (m *manager) Start() error {
    // Load active robots
    if err := m.cache.LoadAll(context.Background()); err != nil {
        return err
    }

    // Start scheduler
    if err := m.scheduler.Start(); err != nil {
        return err
    }

    // Start ticker (every minute)
    m.ticker = time.NewTicker(time.Minute)
    m.wg.Add(1)
    go m.tickLoop()

    log.Info("Autonomous manager started with %d robots", m.cache.Count())
    return nil
}

func (m *manager) Stop() error {
    close(m.stopCh)
    m.ticker.Stop()
    m.wg.Wait()
    return m.scheduler.Stop()
}

func (m *manager) tickLoop() {
    defer m.wg.Done()
    for {
        select {
        case <-m.stopCh:
            return
        case t := <-m.ticker.C:
            if err := m.Tick(context.Background(), t); err != nil {
                log.Error("Tick error: %v", err)
            }
        }
    }
}

func (m *manager) Tick(ctx context.Context, now time.Time) error {
    robots := m.cache.List("") // all teams
    for _, robot := range robots {
        // Skip if not autonomous or paused
        if !robot.AutonomousMode || robot.Status == types.RobotPaused {
            continue
        }

        // Check if clock trigger is enabled
        if !robot.Config.Triggers.IsEnabled(types.TriggerClock) {
            continue
        }

        // Check if should run now
        if !m.shouldRun(robot, now) {
            continue
        }

        // Check dedup
        result, err := m.dedup.CheckExecution(ctx, robot.MemberID, types.TriggerClock)
        if err != nil {
            log.Warn("Dedup check error for %s: %v", robot.MemberID, err)
            continue
        }
        if result == types.DedupSkip {
            continue
        }

        // Submit to scheduler
        if err := m.scheduler.Submit(ctx, robot, types.TriggerClock, nil); err != nil {
            log.Warn("Submit error for %s: %v", robot.MemberID, err)
        }
    }
    return nil
}

func (m *manager) shouldRun(robot *types.Robot, now time.Time) bool {
    cfg := robot.Config.Clock
    if cfg == nil {
        return false
    }

    loc := cfg.GetLocation()
    now = now.In(loc)

    switch cfg.Mode {
    case types.ClockTimes:
        return m.shouldRunTimes(cfg, now)
    case types.ClockInterval:
        return m.shouldRunInterval(robot, cfg, now)
    case types.ClockDaemon:
        return robot.CanRun() // always run if can
    }
    return false
}

func (m *manager) shouldRunTimes(cfg *types.Clock, now time.Time) bool {
    // Check day
    if len(cfg.Days) > 0 && cfg.Days[0] != "*" {
        dayMatch := false
        for _, d := range cfg.Days {
            if d == now.Weekday().String()[:3] {
                dayMatch = true
                break
            }
        }
        if !dayMatch {
            return false
        }
    }

    // Check time (within 1 minute window)
    nowTime := now.Format("15:04")
    for _, t := range cfg.Times {
        if t == nowTime {
            return true
        }
    }
    return false
}

func (m *manager) shouldRunInterval(robot *types.Robot, cfg *types.Clock, now time.Time) bool {
    every, err := time.ParseDuration(cfg.Every)
    if err != nil {
        return false
    }
    return now.Sub(robot.LastExecution) >= every
}
```

### 4.2 Executor Implementation

```go
// executor.go
package autonomous

import (
    "context"
    "fmt"
    "time"

    gonanoid "github.com/matoous/go-nanoid/v2"
    "github.com/yaoapp/kun/log"
    "github.com/yaoapp/yao/agent/autonomous/types"
    "github.com/yaoapp/yao/job"
)

// Ensure executor implements types.Executor
var _ types.Executor = (*executor)(nil)

type executor struct{}

func newExecutor() types.Executor {
    return &executor{}
}

func (e *executor) Execute(ctx context.Context, robot *types.Robot, triggerType types.TriggerType, triggerData interface{}) (*types.Execution, error) {
    // Create execution
    exec := &types.Execution{
        ID:          gonanoid.Must(),
        MemberID:    robot.MemberID,
        TeamID:      robot.TeamID,
        TriggerType: triggerType,
        TriggerData: triggerData,
        StartTime:   time.Now(),
        Status:      types.ExecRunning,
    }

    // Create context with timeout
    timeout := robot.Config.Clock.GetTimeout()
    exec.ctx, exec.cancel = context.WithTimeout(ctx, timeout)
    defer exec.cancel()

    // Update robot status
    robot.IncrRunning()
    defer robot.DecrRunning()

    // Determine phases to run
    phases := e.getPhasesToRun(triggerType)

    // Run phases
    for _, phase := range phases {
        exec.Phase = phase
        if err := e.RunPhase(exec.ctx, exec, phase); err != nil {
            exec.Status = ExecFailed
            exec.Error = err.Error()
            e.saveExecution(exec)
            return exec, err
        }
    }

    // Mark completed
    now := time.Now()
    exec.EndTime = &now
    exec.Status = types.ExecCompleted
    e.saveExecution(exec)

    return exec, nil
}

func (e *executor) getPhasesToRun(triggerType types.TriggerType) []types.Phase {
    if triggerType == types.TriggerClock {
        return types.AllPhases // P0 -> P5
    }
    // Human/Event: skip P0
    return []types.Phase{types.PhaseGoals, types.PhaseTasks, types.PhaseRun, types.PhaseDelivery, types.PhaseLearning}
}

func (e *executor) RunPhase(ctx context.Context, exec *types.Execution, phase types.Phase) error {
    log.Debug("Running phase %s for %s", phase, exec.MemberID)

    switch phase {
    case types.PhaseInspiration:
        return e.runInspiration(ctx, exec)
    case types.PhaseGoals:
        return e.runGoals(ctx, exec)
    case types.PhaseTasks:
        return e.runTasks(ctx, exec)
    case types.PhaseRun:
        return e.runExecution(ctx, exec)
    case types.PhaseDelivery:
        return e.runDelivery(ctx, exec)
    case types.PhaseLearning:
        return e.runLearning(ctx, exec)
    default:
        return fmt.Errorf("unknown phase: %s", phase)
    }
}

func (e *executor) saveExecution(exec *types.Execution) {
    // Save to job system
    jobExec := &job.Execution{
        ExecutionID:     exec.ID,
        JobID:           "robot_" + exec.MemberID,
        Status:          string(exec.Status),
        TriggerCategory: string(exec.TriggerType),
    }
    if exec.StartTime.IsZero() == false {
        jobExec.StartedAt = &exec.StartTime
    }
    if exec.EndTime != nil {
        jobExec.EndedAt = exec.EndTime
    }
    job.SaveExecution(jobExec)
}
```

### 4.3 Yao Process

```go
// process.go
package autonomous

import (
    "context"

    "github.com/yaoapp/gou/process"
    "github.com/yaoapp/kun/log"
    "github.com/yaoapp/yao/agent/autonomous/types"
)

func init() {
    process.Register("autonomous.Execute", processExecute)
    process.Register("autonomous.Intervene", processIntervene)
    process.Register("autonomous.HandleEvent", processHandleEvent)
    process.Register("autonomous.GetStatus", processGetStatus)
    process.Register("autonomous.Pause", processPause)
    process.Register("autonomous.Resume", processResume)
}

// processExecute - autonomous.Execute(memberID, triggerType, triggerData)
func processExecute(p *process.Process) interface{} {
    memberID := p.ArgsString(0)
    triggerType := types.TriggerType(p.ArgsString(1, "clock"))
    triggerData := p.Args[2]

    mgr := GetManager()
    robot := mgr.GetRobot("", memberID) // teamID not needed for lookup
    if robot == nil {
        return map[string]interface{}{"error": "robot not found"}
    }

    exec, err := GetExecutor().Execute(context.Background(), robot, triggerType, triggerData)
    if err != nil {
        log.Error("Execute error: %v", err)
        return map[string]interface{}{"error": err.Error()}
    }

    return map[string]interface{}{
        "execution_id": exec.ID,
        "status":       exec.Status,
    }
}

// processIntervene - autonomous.Intervene(teamID, memberID, action, description, priority)
func processIntervene(p *process.Process) interface{} {
    req := &types.InterveneRequest{
        TeamID:      p.ArgsString(0),
        MemberID:    p.ArgsString(1),
        Action:      types.InterventionAction(p.ArgsString(2)),
        Description: p.ArgsString(3),
        Priority:    types.Priority(p.ArgsString(4, "normal")),
    }

    result, err := GetTrigger().Intervene(context.Background(), req)
    if err != nil {
        return map[string]interface{}{"error": err.Error()}
    }
    return result
}

// processHandleEvent - autonomous.HandleEvent(memberID, source, eventType, data)
func processHandleEvent(p *process.Process) interface{} {
    req := &types.EventRequest{
        MemberID:  p.ArgsString(0),
        Source:    p.ArgsString(1),
        EventType: p.ArgsString(2),
        Data:      p.ArgsMap(3),
    }

    result, err := GetTrigger().HandleEvent(context.Background(), req)
    if err != nil {
        return map[string]interface{}{"error": err.Error()}
    }
    return result
}

// processGetStatus - autonomous.GetStatus(teamID, memberID)
func processGetStatus(p *process.Process) interface{} {
    state, err := GetTrigger().GetStatus(
        context.Background(),
        p.ArgsString(0),
        p.ArgsString(1),
    )
    if err != nil {
        return map[string]interface{}{"error": err.Error()}
    }
    return state
}

// processPause - autonomous.Pause(teamID, memberID)
func processPause(p *process.Process) interface{} {
    err := GetTrigger().Pause(
        context.Background(),
        p.ArgsString(0),
        p.ArgsString(1),
    )
    if err != nil {
        return map[string]interface{}{"error": err.Error()}
    }
    return map[string]interface{}{"success": true}
}

// processResume - autonomous.Resume(teamID, memberID)
func processResume(p *process.Process) interface{} {
    err := GetTrigger().Resume(
        context.Background(),
        p.ArgsString(0),
        p.ArgsString(1),
    )
    if err != nil {
        return map[string]interface{}{"error": err.Error()}
    }
    return map[string]interface{}{"success": true}
}
```

---

## 5. Errors

```go
// types/errors.go
package types

import "errors"

var (
    // Config errors
    ErrMissingIdentity    = errors.New("identity.role is required")
    ErrClockTimesEmpty    = errors.New("clock.times is required for times mode")
    ErrClockIntervalEmpty = errors.New("clock.every is required for interval mode")
    ErrClockModeInvalid   = errors.New("clock.mode must be times, interval, or daemon")

    // Runtime errors
    ErrRobotNotFound      = errors.New("robot not found")
    ErrRobotPaused        = errors.New("robot is paused")
    ErrRobotBusy          = errors.New("robot has reached max concurrent executions")
    ErrTriggerDisabled    = errors.New("trigger type is disabled for this robot")
    ErrExecutionCancelled = errors.New("execution was cancelled")
    ErrExecutionTimeout   = errors.New("execution timed out")

    // Phase errors
    ErrPhaseAgentNotFound = errors.New("phase agent not found")
    ErrGoalGenFailed      = errors.New("goal generation failed")
    ErrTaskPlanFailed     = errors.New("task planning failed")
    ErrDeliveryFailed     = errors.New("delivery failed")
)
```

---

## 6. Global Singletons

```go
// global.go
package autonomous

import (
    "sync"

    "github.com/yaoapp/yao/agent/autonomous/types"
)

var (
    globalManager  types.Manager
    globalTrigger  types.Trigger
    globalExecutor types.Executor
    globalOnce     sync.Once
)

// Init initializes the autonomous system
func Init(cfg *types.SchedulerConfig) error {
    var initErr error
    globalOnce.Do(func() {
        mgr := NewManager(cfg)
        if err := mgr.Start(); err != nil {
            initErr = err
            return
        }
        globalManager = mgr
        globalTrigger = newTrigger(mgr)
        globalExecutor = newExecutor()
    })
    return initErr
}

// GetManager returns the global manager
func GetManager() types.Manager {
    return globalManager
}

// GetTrigger returns the global trigger
func GetTrigger() types.Trigger {
    return globalTrigger
}

// GetExecutor returns the global executor
func GetExecutor() types.Executor {
    return globalExecutor
}

// Shutdown stops the autonomous system
func Shutdown() error {
    if globalManager != nil {
        return globalManager.Stop()
    }
    return nil
}
```

---

## 7. Integration Points

### 7.1 With Job System

```go
// Job creation on robot create
func createRobotJob(robot *types.Robot) error {
    j, err := job.Once(job.GOROUTINE, map[string]interface{}{
        "job_id":      "robot_" + robot.MemberID,
        "category_id": "autonomous_robot",
        "name":        robot.DisplayName,
    })
    if err != nil {
        return err
    }
    return job.SaveJob(j)
}
```

### 7.2 With Assistant

```go
// Call phase agent
func callPhaseAgent(ctx context.Context, agentID string, prompt string) (string, error) {
    ast, err := assistant.Get(agentID)
    if err != nil {
        return "", err
    }

    messages := []chatctx.Message{
        {Role: "user", Content: prompt},
    }

    resp, err := ast.Stream(chatctx.New(ctx), messages)
    if err != nil {
        return "", err
    }

    return resp.Content, nil
}
```

### 7.3 With Member Model

```go
// Load robot from __yao.member
func loadRobotFromMember(memberID string) (*types.Robot, error) {
    mod := model.Select("__yao.member")
    data, err := mod.Find(memberID, model.QueryParam{})
    if err != nil {
        return nil, err
    }

    robot := &types.Robot{
        MemberID:       data.Get("member_id").(string),
        TeamID:         data.Get("team_id").(string),
        DisplayName:    data.Get("display_name").(string),
        SystemPrompt:   data.Get("system_prompt").(string),
        Status:         types.RobotStatus(data.Get("robot_status").(string)),
        AutonomousMode: data.Get("autonomous_mode").(bool),
    }

    // Parse robot_config
    if cfgData := data.Get("robot_config"); cfgData != nil {
        var cfg types.Config
        if err := jsoniter.Unmarshal(cfgData.([]byte), &cfg); err != nil {
            return nil, err
        }
        robot.Config = &cfg
    }

    return robot, nil
}
```

---

## 8. Testing

```go
// manager_test.go
package autonomous

import (
    "context"
    "testing"
    "time"

    "github.com/yaoapp/yao/agent/autonomous/types"
)

func TestManagerTick(t *testing.T) {
    mgr := NewManager(&types.SchedulerConfig{Workers: 2})
    defer mgr.Stop()

    // Add test robot
    robot := &types.Robot{
        MemberID:       "test_robot",
        TeamID:         "test_team",
        AutonomousMode: true,
        Status:         types.RobotIdle,
        Config: &types.Config{
            Clock: &types.Clock{
                Mode:  types.ClockTimes,
                Times: []string{"09:00"},
                Days:  []string{"*"},
            },
            Identity: &types.Identity{Role: "Test"},
        },
    }

    mgr.(*manager).cache.Add(robot)

    // Tick at 09:00
    now := time.Date(2024, 1, 1, 9, 0, 0, 0, time.Local)
    err := mgr.Tick(context.Background(), now)
    if err != nil {
        t.Fatalf("Tick error: %v", err)
    }

    // Check execution was submitted
    // ...
}
```
