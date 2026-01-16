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
│   └── jsapi.go              # JS API: robot (global) + Robot (class)
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
│   └── manager.go            # Manager struct, Start/Stop, Tick
│
├── pool/                     # Worker pool & task dispatch
│   ├── pool.go               # Pool struct, Submit
│   ├── queue.go              # Priority queue
│   └── worker.go             # Worker goroutines
│
├── executor/                 # Executor package (pluggable architecture)
│   ├── executor.go           # Factory functions, unified entry
│   ├── types/
│   │   ├── types.go          # Executor interface, Config types
│   │   └── helpers.go        # Shared helper functions
│   ├── standard/
│   │   ├── executor.go       # Real Agent execution (production)
│   │   ├── agent.go          # AgentCaller for LLM calls
│   │   ├── input.go          # InputFormatter for prompts
│   │   ├── inspiration.go    # P0: Inspiration phase
│   │   ├── goals.go          # P1: Goals phase
│   │   ├── tasks.go          # P2: Tasks phase
│   │   ├── run.go            # P3: Run phase
│   │   ├── delivery.go       # P4: Delivery phase
│   │   └── learning.go       # P5: Learning phase
│   ├── dryrun/
│   │   └── executor.go       # Simulated execution (testing/demo)
│   └── sandbox/
│       └── executor.go       # Container-isolated (NOT IMPLEMENTED)
│
├── utils/                    # Utility functions
│   ├── convert.go            # Type conversions (JSON, map, struct)
│   ├── time.go               # Time parsing, formatting, timezone
│   ├── id.go                 # ID generation (nanoid, uuid)
│   └── validate.go           # Validation helpers
│
├── trigger/                  # Trigger utilities (logic in manager/)
│   ├── trigger.go            # Validation helpers, action utilities
│   ├── clock.go              # ClockMatcher (reusable clock matching logic)
│   └── control.go            # ExecutionController (pause/resume/stop)
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

> **Note:** `trigger/` is a utility package (validation, clock matching, execution control).
> All trigger logic flows through `manager/`.

```
                              ┌──────────┐
                              │  types/  │  (pure types, no deps)
                              └────┬─────┘
                                   │
    ┌───────┬───────┬───────┬──────┼──────┬───────┬───────┬───────┐
    │       │       │       │      │      │       │       │       │
    ▼       ▼       ▼       ▼      ▼      ▼       ▼       ▼       ▼
┌───────┐┌───────┐┌───────┐┌──────┐┌────┐┌──────┐┌───────┐┌─────────┐
│ cache ││ dedup ││ store ││ pool ││job ││ plan ││ utils ││ trigger │
└───┬───┘└───┬───┘└───┬───┘└──┬───┘└──┬─┘└──────┘└───────┘└────┬────┘
    │        │        │       │       │                        │
    └────────┴────────┴───────┴───────┴────────────────────────┘
                      │
                      ▼
               ┌────────────┐
               │  executor/ │
               └──────┬─────┘
                      │
                      ▼
               ┌────────────┐
               │  manager/  │  (imports trigger/ for utilities)
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

| Package     | Imports                                                     |
| ----------- | ----------------------------------------------------------- |
| `types/`    | stdlib only                                                 |
| `utils/`    | stdlib only                                                 |
| `cache/`    | `types/`                                                    |
| `dedup/`    | `types/`                                                    |
| `store/`    | `types/`                                                    |
| `pool/`     | `types/`                                                    |
| `trigger/`  | `types/`                                                    |
| `job/`      | `types/`, `yao/job`                                         |
| `plan/`     | `types/`                                                    |
| `executor/` | `types/`, `cache/`, `dedup/`, `store/`, `pool/`, `job/`     |
| `manager/`  | `types/`, `cache/`, `pool/`, `trigger/`, `executor/`        |
|             | Manager handles all trigger logic (clock, intervene, event) |
| `api/`      | `types/`, `manager/`                                        |
| root        | all packages                                                |

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
    TeamID    string            `json:"team_id,omitempty"`    // filter by team
    Status    types.RobotStatus `json:"status,omitempty"`     // idle | working | paused | error
    Keywords  string            `json:"keywords,omitempty"`   // search display_name, role
    ClockMode types.ClockMode   `json:"clock_mode,omitempty"` // times | interval | daemon
    Page      int               `json:"page,omitempty"`       // default 1
    PageSize  int               `json:"pagesize,omitempty"`   // default 20, max 100
    Order     string            `json:"order,omitempty"`      // e.g. "created_at desc"
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
    MemberID    string            `json:"member_id"`
    TeamID      string            `json:"team_id"`
    DisplayName string            `json:"display_name"`
    Status      types.RobotStatus `json:"status"`                // idle | working | paused | error
    Running     int               `json:"running"`               // current running execution count
    MaxRunning  int               `json:"max_running"`           // max concurrent allowed
    LastRun     *time.Time        `json:"last_run,omitempty"`
    NextRun     *time.Time        `json:"next_run,omitempty"`
    RunningIDs  []string          `json:"running_ids,omitempty"` // list of running execution IDs
}

// ==================== Trigger Types ====================

// TriggerRequest - request for Trigger()
// Input uses []context.Message to support rich content (text, images, files, audio)
type TriggerRequest struct {
    Type types.TriggerType `json:"type"` // human | event

    // Human intervention fields (when Type = human)
    Action   types.InterventionAction `json:"action,omitempty"`    // task.add | goal.adjust | task.cancel | plan.add
    Messages []context.Message        `json:"messages,omitempty"`  // user's input (supports text, images, files)
    PlanAt   *time.Time               `json:"plan_at,omitempty"`   // for action=plan.add
    InsertAt InsertPosition           `json:"insert_at,omitempty"` // where to insert: first | last | next | at
    AtIndex  int                      `json:"at_index,omitempty"`  // index when insert_at=at

    // Event fields (when Type = event)
    Source    types.EventSource      `json:"source,omitempty"`     // webhook | database
    EventType string                 `json:"event_type,omitempty"` // lead.created, order.paid, etc.
    Data      map[string]interface{} `json:"data,omitempty"`       // event payload

    // Executor mode (optional, overrides robot config)
    ExecutorMode types.ExecutorMode `json:"executor_mode,omitempty"` // standard | dryrun
}

// InsertPosition - where to insert task in queue
type InsertPosition string

const (
    InsertFirst InsertPosition = "first" // insert at beginning (highest priority)
    InsertLast  InsertPosition = "last"  // append at end (default)
    InsertNext  InsertPosition = "next"  // insert after current task
    InsertAt    InsertPosition = "at"    // insert at specific index (use AtIndex)
)

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
    Status   types.ExecStatus   `json:"status,omitempty"`  // pending | running | completed | failed
    Trigger  types.TriggerType  `json:"trigger,omitempty"` // clock | human | event
    Page     int                `json:"page,omitempty"`    // default 1
    PageSize int                `json:"pagesize,omitempty"`// default 20
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

| Process              | Args                  | Returns           | Description        |
| -------------------- | --------------------- | ----------------- | ------------------ |
| `robot.Get`          | `memberID`            | `Robot`           | Get robot by ID    |
| `robot.List`         | `query`               | `ListResult`      | List robots        |
| `robot.Create`       | `teamID`, `data`      | `Robot`           | Create robot       |
| `robot.Update`       | `memberID`, `data`    | `Robot`           | Update robot       |
| `robot.Remove`       | `memberID`            | `null`            | Delete robot       |
| `robot.Status`       | `memberID`            | `RobotState`      | Get runtime status |
| `robot.UpdateStatus` | `memberID`, `status`  | `null`            | Update status      |
| `robot.Trigger`      | `memberID`, `request` | `TriggerResult`   | Trigger execution  |
| `robot.Executions`   | `memberID`, `query`   | `ExecutionResult` | List executions    |
| `robot.Execution`    | `execID`              | `Execution`       | Get execution      |
| `robot.Pause`        | `execID`              | `null`            | Pause execution    |
| `robot.Resume`       | `execID`              | `null`            | Resume execution   |
| `robot.Stop`         | `execID`              | `null`            | Stop execution     |

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

// Trigger with text message
const result = Process("robot.Trigger", "mem_abc123", {
  type: "human",
  action: "task.add",
  messages: [
    { role: "user", content: "Prepare meeting materials for BigCorp" },
  ],
  insert_at: "first",
});

// Trigger with image (multimodal)
const imageResult = Process("robot.Trigger", "mem_abc123", {
  type: "human",
  action: "task.add",
  messages: [
    {
      role: "user",
      content: [
        { type: "text", text: "Analyze this chart and summarize key trends" },
        {
          type: "image_url",
          image_url: { url: "https://example.com/chart.png" },
        },
      ],
    },
  ],
  insert_at: "first",
});

// Trigger with event
const eventResult = Process("robot.Trigger", "mem_abc123", {
  type: "event",
  source: "webhook",
  event_type: "lead.created",
  data: { name: "John", email: "john@example.com" },
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

**Global object `robot` (static methods):**

```go
func init() {
    // Register global robot object (lowercase, for static methods)
    v8.RegisterObject("robot", ExportObject)
}

// ExportObject exports the robot global object
func ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
    obj := v8go.NewObjectTemplate(iso)
    obj.Set("List", v8go.NewFunctionTemplate(iso, jsList))
    obj.Set("Get", v8go.NewFunctionTemplate(iso, jsGet))
    obj.Set("Create", v8go.NewFunctionTemplate(iso, jsCreate))
    obj.Set("Update", v8go.NewFunctionTemplate(iso, jsUpdate))
    obj.Set("Remove", v8go.NewFunctionTemplate(iso, jsRemove))
    obj.Set("Execution", v8go.NewFunctionTemplate(iso, jsExecution))
    return obj
}
```

**TypeScript Interface:**

```typescript
// ==================== Types ====================

interface RobotData {
  member_id: string;
  team_id: string;
  display_name: string;
  robot_status: "idle" | "working" | "paused" | "error" | "maintenance";
  robot_config: RobotConfig;
}

interface RobotState {
  member_id: string;
  team_id: string;
  display_name: string;
  status: "idle" | "working" | "paused" | "error" | "maintenance";
  running: number; // current running execution count
  max_running: number; // max concurrent allowed
  last_run?: string;
  next_run?: string;
  running_ids?: string[]; // list of running execution IDs
}

interface TriggerResult {
  accepted: boolean;
  queued: boolean;
  execution?: Execution;
  job_id?: string;
  message?: string;
}

// Message - same as context.Message, supports rich content
interface Message {
  role: "user" | "assistant" | "system" | "tool";
  content: string | ContentPart[];
  name?: string;
  tool_call_id?: string;
  tool_calls?: ToolCall[];
}

interface ContentPart {
  type: "text" | "image_url" | "input_audio" | "file" | "data";
  text?: string;
  image_url?: { url: string; detail?: "auto" | "low" | "high" };
  input_audio?: { data: string; format: string };
  file?: { url: string; name?: string; mime_type?: string };
  data?: { data: string; mime_type: string };
}

interface TriggerRequest {
  type: "human" | "event";

  // Human intervention fields
  action?:
    | "task.add"
    | "task.cancel"
    | "task.update"
    | "goal.adjust"
    | "goal.add"
    | "goal.complete"
    | "goal.cancel"
    | "plan.add"
    | "plan.remove"
    | "plan.update"
    | "instruct";
  messages?: Message[]; // supports text, images, files, audio
  insert_at?: "first" | "last" | "next" | "at";
  at_index?: number;
  plan_at?: string; // ISO date for plan.add

  // Event fields
  source?: "webhook" | "database";
  event_type?: string; // lead.created, etc.
  data?: Record<string, any>;

  // Executor mode (optional, overrides robot config)
  executor_mode?: "standard" | "dryrun"; // sandbox not implemented
}

// ExecutorMode - executor mode type
type ExecutorMode = "standard" | "dryrun" | "sandbox";
// Note: "sandbox" requires container infrastructure, falls back to "dryrun"

interface ListQuery {
  team_id?: string;
  status?: "idle" | "working" | "paused" | "error" | "maintenance";
  keywords?: string;
  clock_mode?: "times" | "interval" | "daemon";
  page?: number;
  pagesize?: number;
}

interface ListResult {
  data: RobotData[];
  total: number;
  page: number;
  pagesize: number;
}

interface ExecutionQuery {
  status?: "pending" | "running" | "completed" | "failed" | "cancelled";
  trigger?: "clock" | "human" | "event";
  page?: number;
  pagesize?: number;
}

interface ExecutionResult {
  data: Execution[];
  total: number;
  page: number;
  pagesize: number;
}

interface CreateRequest {
  display_name: string;
  system_prompt?: string;
  robot_config: RobotConfig;
}

interface UpdateRequest {
  display_name?: string;
  system_prompt?: string;
  robot_config?: RobotConfig;
}

// ==================== Global object: robot ====================
// Static methods, no instance needed

interface RobotStatic {
  List(query?: ListQuery): ListResult;
  Get(memberID: string): RobotData;
  Create(teamID: string, data: CreateRequest): RobotData;
  Update(memberID: string, data: UpdateRequest): RobotData;
  Remove(memberID: string): void;
  Execution(execID: string): Execution;
}

declare const robot: RobotStatic;

// ==================== Constructor: Robot ====================
// Instance methods, operate on specific robot

declare class Robot {
  constructor(memberID: string);

  // Properties
  readonly memberID: string;

  // Instance methods
  Status(): RobotState;
  UpdateStatus(status: string): void;
  Trigger(request: TriggerRequest): TriggerResult;
  Executions(query?: ExecutionQuery): ExecutionResult;
  Pause(execID: string): void;
  Resume(execID: string): void;
  Stop(execID: string): void;
}
```

**Usage:**

```javascript
// ==================== Global object: robot ====================
// For CRUD and queries (no instance needed)

const list = robot.List({ team_id: "team_xyz", status: "idle" });
const data = robot.Get("mem_abc123");
const newRobot = robot.Create("team_xyz", {
  display_name: "Sales Bot",
  robot_config: { ... }
});
robot.Update("mem_abc123", { display_name: "Updated Bot" });
robot.Remove("mem_abc123");
const exec = robot.Execution("exec_456");

// ==================== Constructor: Robot ====================
// For operating on a specific robot instance

const bot = new Robot("mem_abc123");

// Instance methods
const state = bot.Status();
if (state.status === "idle") {
  const result = bot.Trigger({
    type: "human",
    action: "task.add",
    messages: [{ role: "user", content: "Analyze sales data" }],
    insert_at: "first",
  });
  console.log("Triggered:", result.accepted);
}

// Get execution history for this robot
const execs = bot.Executions({ status: "completed", page: 1 });

// Control execution
bot.Pause("exec_123");
bot.Resume("exec_123");
bot.Stop("exec_123");

// Update status
bot.UpdateStatus("paused");
```

**Usage in Agent Hooks:**

```javascript
function Create(ctx, messages) {
  const bot = new Robot("mem_abc123");
  const state = bot.Status();

  if (state.status === "working") {
    ctx.Send({ type: "text", props: { content: "Robot is busy" } });
    return null;
  }

  const result = bot.Trigger({
    type: "human",
    action: "task.add",
    messages: [{ role: "user", content: "Analyze this data" }],
    insert_at: "first",
  });

  if (result.accepted) {
    ctx.memory.context.Set("robot_exec_id", result.execution.id);
  }

  return { messages };
}

function Next(ctx, payload) {
  const execID = ctx.memory.context.Get("robot_exec_id");
  if (execID) {
    const exec = robot.Execution(execID); // use global object
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

// EventSource - event trigger source
type EventSource string

const (
    EventWebhook  EventSource = "webhook"  // HTTP webhook
    EventDatabase EventSource = "database" // DB change trigger
)

// LearningType - learning entry type
type LearningType string

const (
    LearnExecution LearningType = "execution" // execution record
    LearnFeedback  LearningType = "feedback"  // error/fix feedback
    LearnInsight   LearningType = "insight"   // pattern/tip insight
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
    KB        *KB        `json:"kb,omitempty"`        // shared knowledge base (same as assistant)
    DB        *DB        `json:"db,omitempty"`        // shared database (same as assistant)
    Learn     *Learn     `json:"learn,omitempty"`     // learning config for private KB
    Resources *Resources `json:"resources,omitempty"`
    Delivery  *Delivery  `json:"delivery,omitempty"`
    Events    []Event    `json:"events,omitempty"`
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

// KB - knowledge base config (same as assistant, from store/types)
// Shared KB collections accessible by this robot
type KB struct {
    Collections []string               `json:"collections,omitempty"` // KB collection IDs
    Options     map[string]interface{} `json:"options,omitempty"`
}

// DB - database config (same as assistant, from store/types)
// Shared database models accessible by this robot
type DB struct {
    Models  []string               `json:"models,omitempty"` // database model names
    Options map[string]interface{} `json:"options,omitempty"`
}

// Learn - learning config for robot's private KB
// Private KB is auto-created: robot_{team_id}_{member_id}_kb
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
    Type   EventSource            `json:"type"`   // webhook | database
    Source string                 `json:"source"` // webhook path or table name
    Filter map[string]interface{} `json:"filter,omitempty"`
}

// Monitor - monitoring config
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

// Robot - runtime representation of an autonomous robot (from __yao.member)
// Relationship: 1 Robot : N Executions (concurrent)
// Each trigger creates a new Execution (mapped to job.Job)
type Robot struct {
    // From __yao.member
    MemberID       string      `json:"member_id"`
    TeamID         string      `json:"team_id"`
    DisplayName    string      `json:"display_name"`
    SystemPrompt   string      `json:"system_prompt"`
    Status         RobotStatus `json:"robot_status"`
    AutonomousMode bool        `json:"autonomous_mode"`

    // Parsed config (from robot_config JSON field)
    Config *Config `json:"-"`

    // Runtime state
    LastRun   time.Time `json:"-"` // last execution start time
    NextRun   time.Time `json:"-"` // next scheduled execution (for clock trigger)

    // Concurrency control
    // Each Robot can run multiple Executions concurrently (up to Quota.Max)
    executions map[string]*Execution // execID -> Execution
    execMu     sync.RWMutex
}

// CanRun checks if robot can accept new execution
func (r *Robot) CanRun() bool {
    r.execMu.RLock()
    defer r.execMu.RUnlock()
    return len(r.executions) < r.Config.Quota.GetMax()
}

// RunningCount returns current running execution count
func (r *Robot) RunningCount() int {
    r.execMu.RLock()
    defer r.execMu.RUnlock()
    return len(r.executions)
}

// AddExecution adds an execution to tracking
func (r *Robot) AddExecution(exec *Execution) {
    r.execMu.Lock()
    defer r.execMu.Unlock()
    if r.executions == nil {
        r.executions = make(map[string]*Execution)
    }
    r.executions[exec.ID] = exec
}

// RemoveExecution removes an execution from tracking
func (r *Robot) RemoveExecution(execID string) {
    r.execMu.Lock()
    defer r.execMu.Unlock()
    delete(r.executions, execID)
}

// GetExecution returns an execution by ID
func (r *Robot) GetExecution(execID string) *Execution {
    r.execMu.RLock()
    defer r.execMu.RUnlock()
    return r.executions[execID]
}

// GetExecutions returns all running executions
func (r *Robot) GetExecutions() []*Execution {
    r.execMu.RLock()
    defer r.execMu.RUnlock()
    execs := make([]*Execution, 0, len(r.executions))
    for _, exec := range r.executions {
        execs = append(execs, exec)
    }
    return execs
}

// Execution - single execution instance
// Each trigger creates a new Execution, mapped to a job.Job for monitoring
// Relationship: 1 Execution = 1 job.Job
type Execution struct {
    ID          string      `json:"id"`           // unique execution ID
    MemberID    string      `json:"member_id"`    // robot member ID
    TeamID      string      `json:"team_id"`
    TriggerType TriggerType `json:"trigger_type"` // clock | human | event
    StartTime   time.Time   `json:"start_time"`
    EndTime     *time.Time  `json:"end_time,omitempty"`
    Status      ExecStatus  `json:"status"`
    Phase       Phase       `json:"phase"`
    Error       string      `json:"error,omitempty"`

    // Job integration (each Execution = 1 job.Job)
    JobID string `json:"job_id"` // corresponding job.Job ID

    // Trigger input (stored for traceability)
    Input *TriggerInput `json:"input,omitempty"` // original trigger input

    // Phase outputs
    Inspiration *InspirationReport `json:"inspiration,omitempty"` // P0: markdown
    Goals       *Goals             `json:"goals,omitempty"`       // P1: markdown
    Tasks       []Task             `json:"tasks,omitempty"`       // P2: structured tasks
    Current     *CurrentState      `json:"current,omitempty"`     // current executing state
    Results     []TaskResult       `json:"results,omitempty"`     // P3: task results
    Delivery    *DeliveryResult    `json:"delivery,omitempty"`
    Learning    []LearningEntry    `json:"learning,omitempty"`

    // Runtime (internal, not serialized)
    ctx    context.Context    `json:"-"`
    cancel context.CancelFunc `json:"-"`
    robot  *Robot             `json:"-"`
}

// TriggerInput - stored trigger input for traceability
type TriggerInput struct {
    // For human intervention
    Action   InterventionAction `json:"action,omitempty"`   // task.add, goal.adjust, etc.
    Messages []context.Message  `json:"messages,omitempty"` // user's input (text, images, files)
    UserID   string             `json:"user_id,omitempty"`  // who triggered

    // For event trigger
    Source    EventSource            `json:"source,omitempty"`     // webhook | database
    EventType string                 `json:"event_type,omitempty"` // lead.created, etc.
    Data      map[string]interface{} `json:"data,omitempty"`       // event payload

    // For clock trigger
    Clock *ClockContext `json:"clock,omitempty"` // time context when triggered
}

// CurrentState - current executing goal and task
type CurrentState struct {
    Task      *Task  `json:"task,omitempty"`     // current task being executed
    TaskIndex int    `json:"task_index"`         // index in Tasks slice
    Progress  string `json:"progress,omitempty"` // human-readable progress (e.g., "2/5 tasks")
}

// Goals - P1 output (markdown for LLM + structured metadata)
// P1 Agent reads InspirationReport and generates goals as markdown
// Example:
// ## Goals
// 1. [High] Analyze sales data and identify trends
//    - Reason: Sales up 50%, need to understand why
// 2. [Normal] Prepare weekly report for manager
//    - Reason: Friday 5pm, weekly report due
// 3. [Low] Update CRM with new leads
//    - Reason: 3 pending leads from yesterday
type Goals struct {
    Content  string          `json:"content"`            // markdown text
    Delivery *DeliveryTarget `json:"delivery,omitempty"` // where to send results (for P4)
}

// DeliveryTarget - where to deliver results (defined in P1, used in P4)
type DeliveryTarget struct {
    Type       DeliveryType           `json:"type"`                 // email | webhook | report | notification
    Recipients []string               `json:"recipients,omitempty"` // email addresses, webhook URLs, user IDs
    Format     string                 `json:"format,omitempty"`     // markdown | html | json | text
    Template   string                 `json:"template,omitempty"`   // template name or inline template
    Options    map[string]interface{} `json:"options,omitempty"`    // channel-specific options
}

// Task - planned task (structured, for execution)
type Task struct {
    ID       string            `json:"id"`
    Messages []context.Message `json:"messages"`           // original input (text, images, files)
    GoalRef  string            `json:"goal_ref,omitempty"` // reference to goal (e.g., "Goal 1")
    Source   TaskSource        `json:"source"`             // auto | human | event

    // Executor
    ExecutorType ExecutorType `json:"executor_type"`
    ExecutorID   string       `json:"executor_id"`
    Args         []any        `json:"args,omitempty"`

    // Validation (defined in P2, used in P3)
    ExpectedOutput  string   `json:"expected_output,omitempty"`  // what the task should produce
    ValidationRules []string `json:"validation_rules,omitempty"` // specific checks to perform

    // Runtime
    Status    TaskStatus `json:"status"`
    Order     int        `json:"order"` // execution order (0-based)
    StartTime *time.Time `json:"start_time,omitempty"`
    EndTime   *time.Time `json:"end_time,omitempty"`
}

// TaskSource - how task was created
type TaskSource string

const (
    TaskSourceAuto   TaskSource = "auto"   // generated by P2 (task planning)
    TaskSourceHuman  TaskSource = "human"  // added via human intervention
    TaskSourceEvent  TaskSource = "event"  // added via event trigger
)

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
    TaskID     string            `json:"task_id"`
    Success    bool              `json:"success"`
    Output     interface{}       `json:"output,omitempty"`
    Error      string            `json:"error,omitempty"`
    Duration   int64             `json:"duration_ms"`
    Validation *ValidationResult `json:"validation,omitempty"` // P3 validation result
}

// ValidationResult - P3 semantic validation result
type ValidationResult struct {
    Passed      bool     `json:"passed"`                // overall validation passed
    Score       float64  `json:"score,omitempty"`       // 0-1 confidence score
    Issues      []string `json:"issues,omitempty"`      // what failed
    Suggestions []string `json:"suggestions,omitempty"` // how to improve
    Details     string   `json:"details,omitempty"`     // detailed validation report
}

// DeliveryResult - P4 delivery output
type DeliveryResult struct {
    Type       DeliveryType `json:"type"`
    Success    bool         `json:"success"`
    Recipients []string     `json:"recipients,omitempty"` // who received
    Content    string       `json:"content,omitempty"`    // formatted content delivered
    Details    interface{}  `json:"details,omitempty"`    // channel-specific response
    Error      string       `json:"error,omitempty"`
    SentAt     *time.Time   `json:"sent_at,omitempty"`
}

// LearningEntry - knowledge to save
type LearningEntry struct {
    Type    LearningType `json:"type"` // execution | feedback | insight
    Content string       `json:"content"`
    Tags    []string     `json:"tags,omitempty"`
    Meta    interface{}  `json:"meta,omitempty"`
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

// InspirationReport - P0 output (simple markdown for LLM)
type InspirationReport struct {
    Clock   *ClockContext `json:"clock"`   // time context
    Content string        `json:"content"` // markdown text for LLM
}

// Content is markdown like:
// ## Summary
// ...
// ## Highlights
// - [High] Sales up 50%
// - [Medium] New lead from BigCorp
// ## Opportunities
// ...
// ## Risks
// ...
// ## World News
// ...
// ## Pending
// ...
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
// Processed by Manager.Intervene()
type InterveneRequest struct {
    TeamID       string                    `json:"team_id"`
    MemberID     string                    `json:"member_id"`
    Action       InterventionAction        `json:"action"`               // task.add, goal.adjust, etc.
    Messages     []agentcontext.Message    `json:"messages,omitempty"`   // user input (text, images, files)
    PlanTime     *time.Time                `json:"plan_time,omitempty"`  // for action=plan.add
    ExecutorMode ExecutorMode              `json:"executor_mode,omitempty"` // optional: standard | dryrun
}

// EventRequest - event trigger request
// Processed by Manager.HandleEvent()
type EventRequest struct {
    MemberID     string                 `json:"member_id"`
    Source       string                 `json:"source"`               // webhook path or table name
    EventType    string                 `json:"event_type"`           // lead.created, etc.
    Data         map[string]interface{} `json:"data,omitempty"`
    ExecutorMode ExecutorMode           `json:"executor_mode,omitempty"` // optional: standard | dryrun
}

// ExecutorMode - executor mode enum
type ExecutorMode string

const (
    ExecutorStandard ExecutorMode = "standard" // real Agent calls (default)
    ExecutorDryRun   ExecutorMode = "dryrun"   // simulated, no LLM calls
    ExecutorSandbox  ExecutorMode = "sandbox"  // container-isolated (NOT IMPLEMENTED)
)

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
    Running     int         `json:"running"`               // current running execution count
    MaxRunning  int         `json:"max_running"`           // max concurrent allowed
    LastRun     *time.Time  `json:"last_run,omitempty"`
    NextRun     *time.Time  `json:"next_run,omitempty"`
    RunningIDs  []string    `json:"running_ids,omitempty"` // list of running execution IDs
}
```

---

## 3. Interfaces

> Interfaces are also in `types/` package to avoid cycles.

### 3.1 Manager Interface

```go
// types/interfaces.go
package types

import "time"

// ==================== Internal Interfaces ====================
// These are internal implementation interfaces, not exposed via API.
// External API is defined in api/api.go
// All interfaces use *Context (not context.Context) for consistency.

// Manager - robot lifecycle, scheduling, and all trigger handling
// Manager is the central orchestrator, handling:
// - Clock triggers (via Tick)
// - Human intervention (via Intervene)
// - Event triggers (via HandleEvent)
// - Execution control (pause/resume/stop)
type Manager interface {
    // Lifecycle
    Start() error
    Stop() error

    // Clock trigger (called by internal ticker)
    Tick(ctx *Context, now time.Time) error

    // Manual trigger (for testing/API)
    TriggerManual(ctx *Context, memberID string, trigger TriggerType, data interface{}) (string, error)

    // Human intervention
    Intervene(ctx *Context, req *InterveneRequest) (*ExecutionResult, error)

    // Event trigger
    HandleEvent(ctx *Context, req *EventRequest) (*ExecutionResult, error)

    // Execution control
    PauseExecution(ctx *Context, execID string) error
    ResumeExecution(ctx *Context, execID string) error
    StopExecution(ctx *Context, execID string) error
}

// Executor - executes robot phases
type Executor interface {
    Execute(ctx *Context, robot *Robot, trigger TriggerType, data interface{}) (*Execution, error)
}

// Pool - worker pool for concurrent execution
type Pool interface {
    Start() error
    Stop() error
    Submit(ctx *Context, robot *Robot, trigger TriggerType, data interface{}) (string, error)
    Running() int
    Queued() int
}

// Cache - in-memory robot cache
type Cache interface {
    Load(ctx *Context) error
    Get(memberID string) *Robot
    List(teamID string) []*Robot
    Refresh(ctx *Context, memberID string) error
    Add(robot *Robot)
    Remove(memberID string)
}

// Dedup - deduplication check
type Dedup interface {
    Check(ctx *Context, memberID string, trigger TriggerType) (DedupResult, error)
    Mark(memberID string, trigger TriggerType, window time.Duration)
}

// Store - data storage operations (KB, DB)
type Store interface {
    SaveLearning(ctx *Context, memberID string, entries []LearningEntry) error
    GetHistory(ctx *Context, memberID string, limit int) ([]LearningEntry, error)
    SearchKB(ctx *Context, collections []string, query string) ([]interface{}, error)
    QueryDB(ctx *Context, models []string, query interface{}) ([]interface{}, error)
}
```

### 3.2 Trigger Utilities (`trigger/` package)

> **Note:** The `trigger/` package provides utilities, not the main trigger logic.
> All trigger handling is done by `Manager`.

```go
// trigger/trigger.go - Validation and helper functions

// ValidateIntervention validates a human intervention request
func ValidateIntervention(req *InterveneRequest) error

// ValidateEvent validates an event trigger request
func ValidateEvent(req *EventRequest) error

// BuildEventInput creates a TriggerInput from an event request
func BuildEventInput(req *EventRequest) *TriggerInput

// GetActionCategory returns the category of an intervention action
// e.g., "task.add" -> "task", "goal.adjust" -> "goal"
func GetActionCategory(action InterventionAction) string

// GetActionDescription returns a human-readable description of an action
func GetActionDescription(action InterventionAction) string
```

```go
// trigger/clock.go - Clock matching logic (reusable)

// ClockMatcher provides clock trigger matching logic
type ClockMatcher struct{}

// ShouldTrigger checks if a robot should be triggered based on its clock config
func (cm *ClockMatcher) ShouldTrigger(robot *Robot, now time.Time) bool

// ParseTime parses a time string in "HH:MM" format
func ParseTime(timeStr string) (hour, minute int, err error)

// FormatTime formats hour and minute to "HH:MM" string
func FormatTime(hour, minute int) string
```

```go
// trigger/control.go - Execution control (pause/resume/stop)

// ExecutionController manages execution lifecycle
type ExecutionController struct {
    executions map[string]*ControlledExecution
    mu         sync.RWMutex
}

// Track starts tracking an execution
func (c *ExecutionController) Track(execID, memberID, teamID string) *ControlledExecution

// Untrack stops tracking an execution
func (c *ExecutionController) Untrack(execID string)

// Pause pauses an execution
func (c *ExecutionController) Pause(execID string) error

// Resume resumes a paused execution
func (c *ExecutionController) Resume(execID string) error

// Stop stops an execution
func (c *ExecutionController) Stop(execID string) error

// ControlledExecution represents an execution that can be controlled
type ControlledExecution struct {
    ID        string
    MemberID  string
    TeamID    string
    Status    ExecStatus
    Phase     Phase
    StartTime time.Time
    PausedAt  *time.Time
    // ... internal fields for context and channels
}

// IsPaused returns true if the execution is paused
func (e *ControlledExecution) IsPaused() bool

// IsCancelled returns true if the execution is cancelled
func (e *ControlledExecution) IsCancelled() bool

// WaitIfPaused blocks until the execution is resumed or cancelled
func (e *ControlledExecution) WaitIfPaused() error

// CheckCancelled checks if the execution is cancelled and returns error if so
func (e *ControlledExecution) CheckCancelled() error
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
