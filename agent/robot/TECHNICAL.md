# Robot Agent - Technical Design

## 1. Code Structure

```
yao/agent/robot/
├── DESIGN.md                 # Product design doc
├── TECHNICAL.md              # This file
│
├── robot.go                  # Package entry, Init(), Shutdown()
│
├── api/                      # All API forms
│   ├── api.go                # Go API (facade)
│   ├── process.go            # Yao Process: robot.*
│   └── jsapi.go              # JS API: $robot.*
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
│  robot.go   │              │    api/     │
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

Three API forms, all in `api/` directory.

#### Go API (`api/api.go`)

```go
package api

import (
    "github.com/yaoapp/yao/agent/robot/types"
)

// ==================== CRUD ====================

// Get returns a robot by member ID
func Get(ctx *types.Context, memberID string) (*types.Robot, error)

// List returns robots with pagination and filtering
func List(ctx *types.Context, query *ListQuery) (*ListResult, error)

// Create creates a new robot member
func Create(ctx *types.Context, teamID string, req *CreateRequest) (*types.Robot, error)

// Update updates robot config
func Update(ctx *types.Context, memberID string, req *UpdateRequest) (*types.Robot, error)

// Remove deletes a robot member
func Remove(ctx *types.Context, memberID string) error

// ==================== Status ====================

// Status returns current robot runtime state
func Status(ctx *types.Context, memberID string) (*RobotState, error)

// UpdateStatus updates robot status (idle, paused, etc.)
func UpdateStatus(ctx *types.Context, memberID string, status types.RobotStatus) error

// ==================== Trigger ====================

// Trigger starts execution with specified trigger type and request
func Trigger(ctx *types.Context, memberID string, req *TriggerRequest) (*TriggerResult, error)

// ==================== Execution ====================

// GetExecutions returns execution history
func GetExecutions(ctx *types.Context, memberID string, query *ExecutionQuery) (*ExecutionResult, error)

// GetExecution returns a specific execution by ID
func GetExecution(ctx *types.Context, execID string) (*types.Execution, error)

// Pause pauses a running execution
func Pause(ctx *types.Context, execID string) error

// Resume resumes a paused execution
func Resume(ctx *types.Context, execID string) error

// Stop stops a running execution
func Stop(ctx *types.Context, execID string) error

```

#### API Types

```go
// ==================== CRUD Types ====================

// CreateRequest - request for Create()
type CreateRequest struct {
    DisplayName  string        `json:"display_name"`
    SystemPrompt string        `json:"system_prompt,omitempty"`
    Config       *types.Config `json:"robot_config"`
}

// UpdateRequest - request for Update()
type UpdateRequest struct {
    DisplayName  *string       `json:"display_name,omitempty"`
    SystemPrompt *string       `json:"system_prompt,omitempty"`
    Config       *types.Config `json:"robot_config,omitempty"`
}

// ListQuery - query options for List()
type ListQuery struct {
    TeamID    string `json:"team_id,omitempty"`    // filter by team
    Status    string `json:"status,omitempty"`     // idle | working | paused | error
    Keywords  string `json:"keywords,omitempty"`   // search display_name, role
    ClockMode string `json:"clock_mode,omitempty"` // times | interval | daemon
    Page      int    `json:"page,omitempty"`       // default 1
    PageSize  int    `json:"pagesize,omitempty"`   // default 20, max 100
    Order     string `json:"order,omitempty"`      // e.g. "created_at desc"
}

// ListResult - result of List()
type ListResult struct {
    Data     []*types.Robot `json:"data"`
    Total    int            `json:"total"`
    Page     int            `json:"page"`
    PageSize int            `json:"pagesize"`
}

// RobotState - runtime state from Status()
type RobotState struct {
    MemberID    string     `json:"member_id"`
    TeamID      string     `json:"team_id"`
    DisplayName string     `json:"display_name"`
    Status      string     `json:"status"`                 // idle | working | paused | error
    Running     int        `json:"running"`                // current running count
    MaxRunning  int        `json:"max_running"`            // max concurrent allowed
    LastRun     *time.Time `json:"last_run,omitempty"`
    NextRun     *time.Time `json:"next_run,omitempty"`
    CurrentExec string     `json:"current_exec,omitempty"` // current execution ID
}

// ==================== Trigger Types ====================

// TriggerRequest - request for Trigger()
type TriggerRequest struct {
    Type types.TriggerType `json:"type"` // human | event

    // Human intervention fields (when Type = human)
    Action      types.InterventionAction `json:"action,omitempty"`      // add_task | adjust_goal | cancel_task | plan
    Description string                   `json:"description,omitempty"` // task/goal description
    Priority    types.Priority           `json:"priority,omitempty"`    // high | normal | low
    PlanAt      *time.Time               `json:"plan_at,omitempty"`     // for action=plan

    // Event fields (when Type = event)
    Source    string                 `json:"source,omitempty"`     // webhook | database
    EventType string                 `json:"event_type,omitempty"` // lead.created, order.paid, etc.
    Data      map[string]interface{} `json:"data,omitempty"`       // event payload
}

// TriggerResult - result of Trigger()
type TriggerResult struct {
    Accepted  bool             `json:"accepted"`            // whether trigger was accepted
    Queued    bool             `json:"queued"`              // true if queued (quota full)
    Execution *types.Execution `json:"execution,omitempty"` // execution info if started
    JobID     string           `json:"job_id,omitempty"`    // job ID for tracking
    Message   string           `json:"message,omitempty"`   // status message
}

// ==================== Execution Types ====================

// ExecutionQuery - query options for GetExecutions()
type ExecutionQuery struct {
    Status   string `json:"status,omitempty"`   // pending | running | completed | failed
    Trigger  string `json:"trigger,omitempty"`  // clock | human | event
    Page     int    `json:"page,omitempty"`     // default 1
    PageSize int    `json:"pagesize,omitempty"` // default 20
}

// ExecutionResult - result of GetExecutions()
type ExecutionResult struct {
    Data     []*types.Execution `json:"data"`
    Total    int                `json:"total"`
    Page     int                `json:"page"`
    PageSize int                `json:"pagesize"`
}
```

#### Process API (`api/process.go`)

Yao Process registration. Naming convention: `robot.<Action>`

```go
// Process registration
func init() {
    process.Register("robot.Get", processGet)
    process.Register("robot.List", processList)
    process.Register("robot.Create", processCreate)
    process.Register("robot.Update", processUpdate)
    process.Register("robot.Remove", processRemove)
    process.Register("robot.Status", processStatus)
    process.Register("robot.UpdateStatus", processUpdateStatus)
    process.Register("robot.Trigger", processTrigger)
    process.Register("robot.Executions", processExecutions)
    process.Register("robot.Execution", processExecution)
    process.Register("robot.Pause", processPause)
    process.Register("robot.Resume", processResume)
    process.Register("robot.Stop", processStop)
}
```

| Process              | Args                                    | Returns           | Description        |
| -------------------- | --------------------------------------- | ----------------- | ------------------ |
| `robot.Get`          | `memberID`                              | `Robot`           | Get robot by ID    |
| `robot.List`         | `query`                                 | `ListResult`      | List robots        |
| `robot.Create`       | `teamID`, `data`                        | `Robot`           | Create robot       |
| `robot.Update`       | `memberID`, `data`                      | `Robot`           | Update robot       |
| `robot.Remove`       | `memberID`                              | `null`            | Delete robot       |
| `robot.Status`       | `memberID`                              | `RobotState`      | Get runtime status |
| `robot.UpdateStatus` | `memberID`, `status`                    | `null`            | Update status      |
| `robot.Trigger`      | `memberID`, `type`, `action`, `payload` | `TriggerResult`   | Trigger execution  |
| `robot.Executions`   | `memberID`, `query`                     | `ExecutionResult` | List executions    |
| `robot.Execution`    | `execID`                                | `Execution`       | Get execution      |
| `robot.Pause`        | `execID`                                | `null`            | Pause execution    |
| `robot.Resume`       | `execID`                                | `null`            | Resume execution   |
| `robot.Stop`         | `execID`                                | `null`            | Stop execution     |

**Usage:**

```javascript
// In Yao scripts
const robot = Process("robot.Get", "mem_abc123");

const list = Process("robot.List", {
  team_id: "team_xyz",
  status: "idle",
  page: 1,
  pagesize: 20,
});

const result = Process("robot.Trigger", "mem_abc123", "human", "task.add", {
  description: "Prepare meeting materials for BigCorp",
  priority: "high",
});

const execs = Process("robot.Executions", "mem_abc123", {
  status: "completed",
  page: 1,
});
```

#### JSAPI (`api/jsapi.go`)

Register to V8 Runtime using constructor pattern, similar to `new FS()`, `new Store()`, `new Query()`.

```go
func init() {
    // Register Robot constructor
    v8.RegisterFunction("Robot", ExportFunction)
}

// ExportFunction exports the Robot constructor
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
    return v8go.NewFunctionTemplate(iso, robotConstructor)
}

// robotConstructor: new Robot(memberID)
func robotConstructor(info *v8go.FunctionCallbackInfo) *v8go.Value {
    ctx := info.Context()
    args := info.Args()

    if len(args) < 1 {
        return bridge.JsException(ctx, "Robot requires member ID")
    }

    memberID := args[0].String()
    robotObj, err := RobotNew(ctx, memberID)
    if err != nil {
        return bridge.JsException(ctx, err.Error())
    }

    return robotObj
}

// RobotNew creates a Robot JS object with methods
func RobotNew(ctx *v8go.Context, memberID string) (*v8go.Value, error) {
    iso := ctx.Isolate()
    obj := v8go.NewObjectTemplate(iso)

    // Instance methods (operate on this robot)
    obj.Set("Status", v8go.NewFunctionTemplate(iso, jsStatus))
    obj.Set("UpdateStatus", v8go.NewFunctionTemplate(iso, jsUpdateStatus))
    obj.Set("Trigger", v8go.NewFunctionTemplate(iso, jsTrigger))
    obj.Set("Executions", v8go.NewFunctionTemplate(iso, jsExecutions))
    obj.Set("Pause", v8go.NewFunctionTemplate(iso, jsPause))
    obj.Set("Resume", v8go.NewFunctionTemplate(iso, jsResume))
    obj.Set("Stop", v8go.NewFunctionTemplate(iso, jsStop))

    // ... create instance with memberID stored
    return obj.NewInstance(ctx)
}
```

**Static methods (Robot.List, Robot.Create):**

```go
// Register static methods on Robot constructor
func RegisterStaticMethods(iso *v8go.Isolate, robotFn *v8go.FunctionTemplate) {
    robotFn.Set("List", v8go.NewFunctionTemplate(iso, jsListRobots))
    robotFn.Set("Create", v8go.NewFunctionTemplate(iso, jsCreateRobot))
    robotFn.Set("Get", v8go.NewFunctionTemplate(iso, jsGetRobot))
    robotFn.Set("Execution", v8go.NewFunctionTemplate(iso, jsGetExecution))
}
```

**TypeScript Interface:**

```typescript
interface RobotData {
  member_id: string;
  team_id: string;
  display_name: string;
  robot_status: "idle" | "working" | "paused" | "error" | "maintenance";
  robot_config: RobotConfig;
}

interface RobotState {
  member_id: string;
  status: string;
  running: number;
  max_running: number;
  last_run?: string;
  next_run?: string;
  current_exec?: string;
}

interface TriggerResult {
  accepted: boolean;
  queued: boolean;
  execution?: Execution;
  job_id?: string;
  message?: string;
}

// Robot instance (created via new Robot(memberID))
declare class Robot {
  constructor(memberID: string);

  // Instance methods
  Status(): RobotState;
  UpdateStatus(status: string): void;
  Trigger(request: TriggerRequest): TriggerResult;
  Executions(query?: ExecutionQuery): ExecutionResult;
  Pause(execID: string): void;
  Resume(execID: string): void;
  Stop(execID: string): void;

  // Static methods
  static List(query?: ListQuery): ListResult;
  static Create(teamID: string, data: CreateRequest): RobotData;
  static Get(memberID: string): RobotData;
  static Execution(execID: string): Execution;
}
```

**Usage:**

```javascript
// Create robot instance
const robot = new Robot("mem_abc123");

// Instance methods
const state = robot.Status();
if (state.status === "idle") {
  const result = robot.Trigger({
    type: "human",
    action: "task.add",
    description: "Analyze sales data",
    priority: "high",
  });
  console.log("Triggered:", result.accepted);
}

// Get execution history
const execs = robot.Executions({ status: "completed", page: 1 });

// Control execution
robot.Pause("exec_123");
robot.Resume("exec_123");
robot.Stop("exec_123");

// Static methods
const list = Robot.List({ team_id: "team_xyz", status: "idle" });
const data = Robot.Get("mem_abc123");
const newRobot = Robot.Create("team_xyz", {
  display_name: "Sales Bot",
  robot_config: { ... }
});
const exec = Robot.Execution("exec_456");
```

**Usage in Agent Hooks:**

```javascript
function Create(ctx, messages) {
  const robot = new Robot("mem_abc123");
  const state = robot.Status();

  if (state.status === "working") {
    ctx.Send({ type: "text", props: { content: "Robot is busy" } });
    return null;
  }

  const result = robot.Trigger({
    type: "human",
    action: "task.add",
    description: "Analyze this data",
    priority: "high",
  });

  if (result.accepted) {
    ctx.memory.context.Set("robot_exec_id", result.execution.id);
  }

  return { messages };
}

function Next(ctx, payload) {
  const execID = ctx.memory.context.Get("robot_exec_id");
  if (execID) {
    const exec = Robot.Execution(execID);
    if (exec.status === "completed") {
      ctx.Send({
        type: "text",
        props: { content: `Robot completed: ${exec.delivery?.summary}` },
      });
    }
  }
  return null;
}
```

---

## 2. Type Definitions

> All types are in `robot/types/` package. Other files import as:
>
> ```go
> import "github.com/yaoapp/yao/agent/robot/types"
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

// InterventionAction - human intervention action
// Format: category.action (e.g., "task.add", "goal.adjust")
type InterventionAction string

const (
    // Task operations
    ActionTaskAdd    InterventionAction = "task.add"    // add a new task
    ActionTaskCancel InterventionAction = "task.cancel" // cancel a task
    ActionTaskUpdate InterventionAction = "task.update" // update task details

    // Goal operations
    ActionGoalAdjust   InterventionAction = "goal.adjust"   // modify current goal
    ActionGoalAdd      InterventionAction = "goal.add"      // add a new goal
    ActionGoalComplete InterventionAction = "goal.complete" // mark goal as complete
    ActionGoalCancel   InterventionAction = "goal.cancel"   // cancel a goal

    // Plan operations (schedule for later)
    ActionPlanAdd    InterventionAction = "plan.add"    // add to plan queue
    ActionPlanRemove InterventionAction = "plan.remove" // remove from plan queue
    ActionPlanUpdate InterventionAction = "plan.update" // update planned item

    // Instruction (direct command)
    ActionInstruct InterventionAction = "instruct" // direct instruction to robot
)

// Priority - task/goal priority
type Priority string

const (
    PriorityHigh   Priority = "high"
    PriorityNormal Priority = "normal"
    PriorityLow    Priority = "low"
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
```

### 2.2 Context

```go
// types/context.go
package types

import (
    "context"
    "github.com/yaoapp/yao/openapi/oauth/types"
)

// Context - robot execution context (lightweight)
type Context struct {
    context.Context                          // embed standard context
    Auth      *types.AuthorizedInfo          `json:"auth,omitempty"`       // reuse oauth AuthorizedInfo
    MemberID  string                         `json:"member_id,omitempty"`  // current robot member ID
    RequestID string                         `json:"request_id,omitempty"` // request trace ID
    Locale    string                         `json:"locale,omitempty"`     // locale (e.g., "en-US")
}

// NewContext creates a new robot context
func NewContext(parent context.Context, auth *types.AuthorizedInfo) *Context {
    if parent == nil {
        parent = context.Background()
    }
    return &Context{
        Context: parent,
        Auth:    auth,
    }
}

// UserID returns user ID from auth
func (c *Context) UserID() string {
    if c.Auth == nil {
        return ""
    }
    return c.Auth.UserID
}

// TeamID returns team ID from auth
func (c *Context) TeamID() string {
    if c.Auth == nil {
        return ""
    }
    return c.Auth.TeamID
}
```

### 2.3 Config Types

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
    Goals       []Goal             `json:"goals,omitempty"` // all goals
    Tasks       []Task             `json:"tasks,omitempty"` // all tasks
    Current     *CurrentState      `json:"current,omitempty"` // current executing state
    Results     []TaskResult       `json:"results,omitempty"`
    Delivery    *DeliveryResult    `json:"delivery,omitempty"`
    Learning    []LearningEntry    `json:"learning,omitempty"`

    // Context
    ctx    context.Context
    cancel context.CancelFunc
    robot  *Robot
}

// CurrentState - current executing goal and task
type CurrentState struct {
    Goal      *Goal  `json:"goal,omitempty"`       // current goal being executed
    GoalIndex int    `json:"goal_index"`           // index in Goals slice
    Task      *Task  `json:"task,omitempty"`       // current task being executed
    TaskIndex int    `json:"task_index"`           // index in Tasks slice
    Progress  string `json:"progress,omitempty"`   // human-readable progress (e.g., "2/5 tasks")
}

// Goal - generated goal
type Goal struct {
    ID          string     `json:"id"`
    Description string     `json:"description"`
    Priority    Priority   `json:"priority"`
    Status      GoalStatus `json:"status"`
    Rationale   string     `json:"rationale,omitempty"`
    Tags        []string   `json:"tags,omitempty"`
    StartTime   *time.Time `json:"start_time,omitempty"`
    EndTime     *time.Time `json:"end_time,omitempty"`
}

// GoalStatus - goal execution status
type GoalStatus string

const (
    GoalPending    GoalStatus = "pending"
    GoalInProgress GoalStatus = "in_progress"
    GoalCompleted  GoalStatus = "completed"
    GoalFailed     GoalStatus = "failed"
    GoalSkipped    GoalStatus = "skipped"
)

// Task - planned task
type Task struct {
    ID           string       `json:"id"`
    GoalID       string       `json:"goal_id"`
    Description  string       `json:"description"`
    ExecutorType ExecutorType `json:"executor_type"`
    ExecutorID   string       `json:"executor_id"`
    Args         []any        `json:"args,omitempty"`
    Status       TaskStatus   `json:"status"`
    Order        int          `json:"order"`
    StartTime    *time.Time   `json:"start_time,omitempty"`
    EndTime      *time.Time   `json:"end_time,omitempty"`
}

// ExecutorType - task executor type
type ExecutorType string

const (
    ExecutorAssistant ExecutorType = "assistant"
    ExecutorMCP       ExecutorType = "mcp"
    ExecutorProcess   ExecutorType = "process"
)

// TaskStatus - task execution status
type TaskStatus string

const (
    TaskPending    TaskStatus = "pending"
    TaskRunning    TaskStatus = "running"
    TaskCompleted  TaskStatus = "completed"
    TaskFailed     TaskStatus = "failed"
    TaskSkipped    TaskStatus = "skipped"
    TaskCancelled  TaskStatus = "cancelled"
)

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

## 4. Errors

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
