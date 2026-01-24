# Robot OpenAPI - Design Document

> Based on: `yao/agent/robot/` (Backend), `cui/packages/cui/pages/mission-control/` (Frontend)
> Gap Analysis: `yao/openapi/agent/robot/GAPS.md`

## 1. Overview

### 1.1 Purpose

Provide HTTP REST API endpoints for Robot Agent management, designed to support the Mission Control frontend UI.

### 1.2 Implementation Strategy

> **Low-risk phases first. Medium-risk features (Chat API, SSE Event Bus) can be deferred.**

| Phase | Risk | Features | Frontend Fallback |
|-------|------|----------|-------------------|
| 1. Core CRUD | 🟢 Low | List, Get, Create, Update, Delete | - |
| 2. Execution Management | 🟢 Low | List, Get, Control executions | - |
| 3. Results & Activities | 🟢 Low | Deliverables, Activity feed | - |
| 4. i18n | 🟢 Low | Locale parameter support | - |
| 5. Chat API | 🟡 Medium (Deferred) | Multi-turn conversation | Single-submit mode |
| 6. SSE Event Bus | 🟡 Medium (Deferred) | Real-time status streams | Polling every 3-5s |

### 1.3 Route Decision: `/v1/agent/robots`

**Analysis of existing `openapi/` route structure:**

| Package | Route | Description |
|---------|-------|-------------|
| `agent/` | `/v1/agent/assistants` | Assistant CRUD, info |
| `chat/` | `/v1/chat/completions` | Chat completions |
| `kb/` | `/v1/kb/collections` | Knowledge base |
| `job/` | `/v1/job/jobs` | Job management |
| `file/` | `/v1/file/*` | File operations |
| `user/` | `/v1/user/*` | User management |
| `team/` | `/v1/team/*` | Team management |

**Decision:** Put Robot routes under `/v1/agent/robots` because:

1. **Semantic Alignment**: Robot is a type of Agent (Autonomous Robot Agent), just like Assistant is a type of Agent
2. **Existing Pattern**: `openapi/agent/` already handles `/v1/agent/assistants`
3. **Logical Grouping**: Agent-related APIs grouped together
4. **Consistent Hierarchy**: `/v1/agent/{type}` pattern

**Route Comparison:**

| Option | Path | Verdict |
|--------|------|---------|
| ❌ `/v1/robots` | New top-level namespace | Inconsistent with agent grouping |
| ✅ `/v1/agent/robots` | Under agent namespace | Follows existing pattern |
| ❌ `/v1/members?type=robot` | Reuse members | Less intuitive for operations |

### 1.4 Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Frontend (Mission Control)                          │
│  cui/packages/cui/pages/mission-control/                                │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ HTTP REST / SSE
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      OpenAPI Layer                                       │
│  yao/openapi/agent/                                                      │
│  - Routes: /v1/agent/assistants/* (existing)                             │
│  - Routes: /v1/agent/robots/* (NEW)                                      │
│  - Auth: OAuth2 via Guard middleware                                     │
│  - SSE: Real-time updates                                               │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Robot API Layer                                     │
│  yao/agent/robot/api/                                                    │
│  - Go functions: Get(), List(), Trigger(), etc.                          │
│  - Business logic                                                        │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Robot Core                                          │
│  yao/agent/robot/                                                        │
│  - Manager, Executor, Cache, Pool, Store                                 │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.5 Design Principles

1. **Layered Architecture**: OpenAPI layer only handles HTTP concerns (routing, request parsing, response formatting). Business logic stays in `robot/api/`.
2. **Consistent with Existing Patterns**: Follow `yao/openapi/agent/` conventions, extend existing agent package
3. **Incremental Implementation**: Start with core CRUD, then add real-time features
4. **Frontend-Backend Balance**: API design considers both frontend needs and backend capabilities

---

## 2. Differences Analysis

### 2.1 Frontend Expectations vs Backend Reality

| Feature | Frontend (API.md) | Backend (robot/api/) | Gap | Solution |
|---------|-------------------|----------------------|-----|----------|
| Robot List | `GET /v1/robots` with `name`, `description` | `List()` returns `types.Robot` | Field mapping needed | Map in OpenAPI layer |
| Robot Detail | `GET /v1/robots/:id` with full `config` | `Get()` returns Robot + Config | Need format conversion | Map to frontend format |
| Create Robot | POST with `work_mode` | Not implemented | New feature | Add `Create()` |
| Update Robot | PUT with partial update | Not implemented | New feature | Add `Update()` |
| Delete Robot | DELETE | Not implemented | New feature | Add `Remove()` |
| Trigger | Immediate execution | `Trigger()` returns sync result | Works | Wrap with SSE events |
| Intervene | Immediate intervention | `Intervene()` returns sync result | Works | Wrap with SSE events |
| Multi-turn Chat | Chat before execute | Not implemented | **Deferred** | Frontend uses single-submit |
| Results List | `/results` endpoint | No separate results API | New feature | Derive from executions |
| Activities | `/activities` endpoint | No activities tracking | New feature | Derive from executions |
| Real-time Stream | SSE `/stream` endpoints | No SSE support | **Deferred** | Frontend uses polling |
| i18n | `?locale=` query param | No i18n support | New feature | Add locale handling |

### 2.2 Field Mapping (Backend → Frontend API)

The `__yao.member` model already has the necessary fields, with different names:

| Frontend API | Backend DB (`__yao.member`) | Backend Go (`types.Robot`) | Mapping |
|--------------|----------------------------|---------------------------|---------|
| `member_id` | `member_id` | `MemberID` | Direct |
| `name` | `member_id` | `MemberID` | **Reuse** (slug-like identifier) |
| `display_name` | `display_name` | `DisplayName` | Direct |
| `description` | `bio` | Need to add `Bio` field | Map in OpenAPI layer |
| `email` | `robot_email` | `RobotEmail` | Direct |

**Required Backend Changes:**
1. Add `Bio` field to `types.Robot` struct
2. Add `bio` to `cache/load.go` memberFields

### 2.3 Type Differences

| Frontend Type | Backend Type | Solution |
|---------------|--------------|----------|
| `RobotState.name` | `Robot.MemberID` | Map `member_id` to `name` |
| `RobotState.description` | `Robot.Bio` (new) | Add field, map to `description` |
| `Execution.name` | Not in `types.Execution` | Derive from goals or input in OpenAPI layer |
| `Execution.current_task_name` | Not in `types.Execution` | Derive from current task in OpenAPI layer |
| `ResultFile` | No equivalent | New type in OpenAPI layer (derive from delivery) |
| `Activity` | No equivalent | New type in OpenAPI layer (derive from executions) |

---

## 3. API Endpoints

> **Base Path:** `/v1/agent/robots`

### 3.1 Robot Management

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | /v1/agent/robots | `ListRobots` | List all robots |
| GET | /v1/agent/robots/:id | `GetRobot` | Get robot details |
| POST | /v1/agent/robots | `CreateRobot` | Create robot |
| PUT | /v1/agent/robots/:id | `UpdateRobot` | Update robot |
| DELETE | /v1/agent/robots/:id | `DeleteRobot` | Delete robot |

### 3.2 Execution Management

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | /v1/agent/robots/:id/executions | `ListExecutions` | List executions |
| GET | /v1/agent/robots/:id/executions/:exec_id | `GetExecution` | Get execution detail |
| POST | /v1/agent/robots/:id/trigger | `TriggerRobot` | Trigger execution (SSE) |
| POST | /v1/agent/robots/:id/intervene | `InterveneRobot` | Intervene execution (SSE) |
| POST | /v1/agent/robots/:id/executions/:exec_id/pause | `PauseExecution` | Pause execution |
| POST | /v1/agent/robots/:id/executions/:exec_id/resume | `ResumeExecution` | Resume execution |
| POST | /v1/agent/robots/:id/executions/:exec_id/cancel | `CancelExecution` | Cancel execution |
| POST | /v1/agent/robots/:id/executions/:exec_id/retry | `RetryExecution` | Retry execution |

### 3.3 Results Management

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | /v1/agent/robots/:id/results | `ListResults` | List deliverables |
| GET | /v1/agent/robots/:id/results/:result_id | `GetResult` | Get deliverable detail |

### 3.4 Activities & Real-time

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | /v1/agent/robots/activities | `ListActivities` | List recent activities |
| GET | /v1/agent/robots/stream | `StreamRobots` | Robot status SSE |
| GET | /v1/agent/robots/:id/executions/:exec_id/stream | `StreamExecution` | Execution progress SSE |

---

## 4. Response Types

### 4.1 RobotResponse (for list and detail)

```go
// RobotResponse - formatted robot for API response
// Maps backend fields to frontend expected format
type RobotResponse struct {
    MemberID    string          `json:"member_id"`
    TeamID      string          `json:"team_id"`
    Name        string          `json:"name"`         // From Robot.MemberID (slug-like identifier)
    DisplayName string          `json:"display_name"` // From Robot.DisplayName
    Description string          `json:"description,omitempty"` // From Robot.Bio
    Status      string          `json:"status"`       // idle | working | paused | error | maintenance
    Running     int             `json:"running"`      // Current running count
    MaxRunning  int             `json:"max_running"`  // From Config.Quota.Max
    LastRun     *string         `json:"last_run,omitempty"`     // ISO timestamp
    NextRun     *string         `json:"next_run,omitempty"`     // ISO timestamp
    RunningIDs  []string        `json:"running_ids,omitempty"`  // Execution IDs
    Config      *ConfigResponse `json:"config,omitempty"`       // Full config (for detail)
}

// NewRobotResponse converts backend Robot to API response
func NewRobotResponse(robot *types.Robot) *RobotResponse {
    return &RobotResponse{
        MemberID:    robot.MemberID,
        TeamID:      robot.TeamID,
        Name:        robot.MemberID,    // Use MemberID as unique identifier
        DisplayName: robot.DisplayName,
        Description: robot.Bio,          // Map Bio to Description
        Status:      string(robot.Status),
        // ... other fields
    }
}
```

### 4.2 ConfigResponse (robot config)

```go
// ConfigResponse - formatted config for API response
type ConfigResponse struct {
    Identity  *IdentityConfig  `json:"identity,omitempty"`
    Clock     *ClockConfig     `json:"clock,omitempty"`
    Events    []EventConfig    `json:"events,omitempty"`
    Quota     *QuotaConfig     `json:"quota,omitempty"`
    Resources *ResourcesConfig `json:"resources,omitempty"`
    Delivery  *DeliveryConfig  `json:"delivery,omitempty"`
    Triggers  *TriggersConfig  `json:"triggers,omitempty"`
    Learn     *LearnConfig     `json:"learn,omitempty"`
    Executor  *ExecutorConfig  `json:"executor,omitempty"`
}
```

### 4.3 ExecutionResponse

```go
// ExecutionResponse - formatted execution for API response
type ExecutionResponse struct {
    ID              string           `json:"id"`
    MemberID        string           `json:"member_id"`
    TeamID          string           `json:"team_id"`
    TriggerType     string           `json:"trigger_type"`
    StartTime       string           `json:"start_time"`
    EndTime         *string          `json:"end_time,omitempty"`
    Status          string           `json:"status"`
    Phase           string           `json:"phase"`
    Error           *string          `json:"error,omitempty"`

    // UI display fields (from backend Execution)
    // These are updated by executor at each phase for frontend display
    Name            string           `json:"name,omitempty"`             // Execution title
    CurrentTaskName string           `json:"current_task_name,omitempty"` // Current task description

    // Phase outputs (for detail view)
    Goals           *GoalsResponse   `json:"goals,omitempty"`
    Tasks           []TaskResponse   `json:"tasks,omitempty"`
    Current         *CurrentState    `json:"current,omitempty"`
    Delivery        *DeliveryResult  `json:"delivery,omitempty"`
}
```

**UI Display Fields Update Timeline:**

| Phase | `Name` | `CurrentTaskName` |
|-------|--------|-------------------|
| Created | Human: from `input.messages[0]`<br>Clock/Event: "Preparing..." | "Starting..." |
| `inspiration` | - | "Analyzing context..." |
| `goals` complete | Extracted from first goal in `goals.content` | "Planning goals..." |
| `tasks` | - | "Breaking down tasks..." |
| `run` (each task) | - | Current task description |
| Completed/Failed | - | "Completed" / "Failed: {error}" |

### 4.4 ResultResponse

```go
// ResultResponse - deliverable file for Results tab
type ResultResponse struct {
    ID            string `json:"id"`
    MemberID      string `json:"member_id"`
    ExecutionID   string `json:"execution_id"`
    Name          string `json:"name"`
    Type          string `json:"type"`  // pdf, xlsx, csv, json, md
    Size          int64  `json:"size"`  // bytes
    CreatedAt     string `json:"created_at"`
    TriggerType   string `json:"trigger_type,omitempty"`
    ExecutionName string `json:"execution_name,omitempty"`
}
```

### 4.5 ActivityResponse

```go
// ActivityResponse - activity item
type ActivityResponse struct {
    ID          string `json:"id"`
    Type        string `json:"type"`  // completed | file | error | started | paused
    MemberID    string `json:"member_id"`
    RobotName   string `json:"robot_name"`   // Localized
    Title       string `json:"title"`        // Localized
    Description string `json:"description,omitempty"`  // Localized
    FileID      string `json:"file_id,omitempty"`
    Timestamp   string `json:"timestamp"`
}
```

---

## 5. Request Types

### 5.1 CreateRobotRequest

```go
// CreateRobotRequest - create robot request
type CreateRobotRequest struct {
    Locale      string          `json:"locale,omitempty"` // zh-CN | en-US
    Name        string          `json:"name"`             // Unique identifier
    DisplayName string          `json:"display_name"`     // Display name
    Email       string          `json:"email,omitempty"`  // Robot email
    ManagerID   string          `json:"manager_id,omitempty"`  // Manager user ID
    WorkMode    string          `json:"work_mode"`        // autonomous | on-demand
    Identity    *IdentityConfig `json:"identity"`
    Resources   *ResourcesConfig `json:"resources,omitempty"`
}
```

### 5.2 UpdateRobotRequest

```go
// UpdateRobotRequest - update robot request
type UpdateRobotRequest struct {
    Locale      string          `json:"locale,omitempty"`
    DisplayName *string         `json:"display_name,omitempty"`
    Config      *ConfigResponse `json:"config,omitempty"` // Partial update supported
}
```

### 5.3 TriggerRequest (SSE)

```go
// TriggerRequest - trigger robot execution
type TriggerRequest struct {
    Locale      string    `json:"locale,omitempty"`
    Messages    []Message `json:"messages"`
    Attachments []Attachment `json:"attachments,omitempty"`
}

// Message - chat message
type Message struct {
    Role    string `json:"role"`    // user | assistant
    Content string `json:"content"`
}

// Attachment - file attachment
type Attachment struct {
    File string `json:"file"` // __yao.attachment://fileID
    Name string `json:"name,omitempty"`
}
```

### 5.4 InterveneRequest (SSE)

```go
// InterveneRequest - intervene during execution
type InterveneRequest struct {
    Locale      string    `json:"locale,omitempty"`
    ExecutionID string    `json:"execution_id"`
    Action      string    `json:"action"`   // task.add | goal.adjust | instruct
    Messages    []Message `json:"messages"`
    Priority    string    `json:"priority,omitempty"` // high | normal | low
    Position    string    `json:"position,omitempty"` // first | last | next | at
}
```

---

## 6. Deferred Features

### 6.1 Multi-turn Chat API (Phase 5 - Deferred)

> **Risk Level:** 🟡 Medium - Requires new stateful component
> **Frontend Fallback:** Single-submit mode (user input → immediate execution)

The frontend `ChatDrawer` component expects multi-turn conversation before execution:

```
User: "Help me analyze competitor pricing"
       ↓
Robot: "Got it. Which competitors?"
       ↓
User: "Focus on Company A and B"
       ↓
Robot: "Understood. Ready to start?"
       ↓
User clicks [Confirm] → Execution starts
```

**Current backend behavior:** `Trigger()` immediately submits to execution pool.

**Deferred implementation:**
```
POST /v1/agent/robots/:id/chat
{
  "conversation_id": "conv_001",  // For continuing conversation
  "messages": [{ "role": "user", "content": "..." }]
}

Response (SSE):
event: message
data: {"role": "assistant", "content": "..."}

event: state
data: {"conversation_id": "conv_001", "ready_to_execute": false}
```

**For now:** Frontend can skip chat flow, directly call `/trigger` with user message.

### 6.2 SSE Event Bus (Phase 6 - Deferred)

> **Risk Level:** 🟡 Medium - Requires modification of executor/manager
> **Frontend Fallback:** Polling (GET /executions every 3-5 seconds)

Real-time status updates via SSE require an event bus integrated with:
- Manager (robot status changes)
- Executor (execution progress)

**For now:** Frontend uses polling to refresh status.

---

## 7. SSE Events

### 7.1 Trigger/Intervene SSE Events

```
event: received
data: {"message": "Task received, creating execution..."}

event: execution
data: {"execution_id": "exec_002", "status": "pending"}

event: message
data: {"role": "assistant", "content": "好的，我开始处理..."}

event: phase
data: {"phase": "goals", "message": "正在生成目标..."}

event: complete
data: {"execution_id": "exec_002", "status": "running"}

event: error
data: {"error": "Something went wrong"}
```

### 7.2 Robot Stream SSE Events (Phase 6 - Deferred)

```
event: robot_status
data: {"member_id": "robot_001", "status": "working", "running": 1}

event: execution_start
data: {"member_id": "robot_001", "execution_id": "exec_001", "name": "每日报表生成"}

event: execution_complete
data: {"member_id": "robot_001", "execution_id": "exec_001", "status": "completed"}

event: activity
data: {"id": "act_001", "type": "completed", "member_id": "robot_001", ...}
```

### 7.3 Execution Stream SSE Events (Phase 6 - Deferred)

```
event: phase
data: {"phase": "tasks", "progress": "2/5 tasks"}

event: task_start
data: {"task_id": "task_002", "order": 2}

event: task_complete
data: {"task_id": "task_002", "status": "completed"}

event: message
data: {"role": "assistant", "content": "正在分析数据..."}

event: delivery
data: {"summary": "...", "attachments": [...]}

event: complete
data: {"status": "completed"}

event: error
data: {"error": "Something went wrong", "phase": "run"}
```

---

## 8. i18n Support

### 8.1 Locale Detection

**For API requests (Human trigger):**

Priority order:
1. Request body field: `locale: "zh-CN"` (in TriggerRequest)
2. Query parameter: `?locale=zh-CN`
3. Accept-Language header
4. Robot's `default_locale` config
5. System default: `en-US`

**For Clock/Event triggers (no user context):**

Priority order:
1. Robot's `default_locale` config (from `robot_config.default_locale`)
2. System default: `en-US`

### 8.2 Robot Default Locale

Robots can configure a default language for clock/event triggered executions:

```go
// In RobotConfig (robot_config field in __yao.member)
type Config struct {
    // ... other fields ...
    DefaultLocale string `json:"default_locale,omitempty"` // "en-US", "zh-CN"
}
```

**Language resolution:**
```go
func getLocale(robot *Robot, input *TriggerInput) string {
    // 1. Human trigger with explicit locale
    if input != nil && input.Locale != "" {
        return input.Locale
    }
    // 2. Robot configured default
    if robot.Config != nil && robot.Config.DefaultLocale != "" {
        return robot.Config.DefaultLocale
    }
    // 3. System default
    return "en-US"
}
```

### 8.3 Localized Fields

| Response Type | Localized Fields |
|---------------|------------------|
| RobotResponse | display_name, description |
| ExecutionResponse | name, current_task_name |
| TaskResponse | (none - tasks use executor_id) |
| ResultResponse | name, execution_name |
| ActivityResponse | robot_name, title, description |

---

## 9. Authentication & Authorization

### 9.1 Guard Middleware

All endpoints require OAuth2 authentication via `oauth.Guard` middleware.

```go
// In router registration
router.Use(oauth.Guard())
```

### 9.2 Permission Checks

| Endpoint | Required Scope |
|----------|----------------|
| GET /robots | `robots:read` |
| POST /robots | `robots:write` |
| PUT/DELETE /robots/:id | `robots:write` + ownership check |
| Trigger/Intervene | `robots:execute` |
| Stream endpoints | `robots:read` |

### 9.3 Team Isolation

Robots are team-scoped. Users can only access robots in their team.

```go
func checkTeamAccess(ctx context.Context, memberID string) error {
    auth := oauth.GetAuthorized(ctx)
    robot, _ := robotapi.Get(memberID)
    if robot.TeamID != auth.TeamID {
        return errors.New("access denied")
    }
    return nil
}
```

---

## 10. File Structure

### 10.1 Backend Store + API Layers

```
yao/agent/robot/
├── store/                      # Store Layer (Core CRUD)
│   ├── store.go               # Common interfaces
│   ├── execution.go           # ExecutionStore (EXISTS)
│   └── robot.go               # RobotStore (NEW)
│
├── api/                        # API Layer (Thin wrappers)
│   ├── robot.go               # Get, List, Create, Update, Remove
│   ├── execution.go           # Execution management
│   ├── trigger.go             # Trigger, Intervene
│   ├── results.go             # ListResults, GetResult (NEW)
│   └── activities.go          # ListActivities (NEW)
│
├── types/                      # Type definitions
│   └── robot.go               # Add Bio field
│
└── cache/                      # Cache Layer
    └── load.go                # Add bio to memberFields
```

### 10.2 OpenAPI Layer

**Decision: Sub-package under `openapi/agent/`**

Robot logic is complex enough to warrant its own package. This keeps code organized and follows the pattern used by other complex modules.

```
yao/openapi/agent/
├── agent.go            # Main route registration (MODIFY: add robot.Attach)
├── assistant.go        # Assistant handlers (existing)
├── filter.go           # Query filtering (existing)
├── models.go           # LLM models (existing)
├── types.go            # Types (existing)
│
└── robot/              # Robot sub-package (NEW)
    ├── DESIGN.md       # This document ✅
    ├── TODO.md         # Implementation plan ✅
    ├── GAPS.md         # Gap analysis ✅
    │
    ├── robot.go        # Route registration (Attach function)
    ├── types.go        # Request/Response types
    │
    ├── list.go         # GET /v1/agent/robots
    ├── detail.go       # GET/POST/PUT/DELETE /v1/agent/robots/:id
    │
    ├── execution.go    # Execution list/detail/control handlers
    ├── trigger.go      # POST /trigger, POST /intervene (SSE)
    │
    ├── results.go      # GET /results, GET /results/:id
    ├── activities.go   # GET /activities
    │
    ├── stream.go       # GET /stream, GET /executions/:id/stream (SSE)
    │
    ├── filter.go       # Query param parsing helpers
    └── utils.go        # Locale, time formatting utilities
```

**Route Registration (in `openapi/agent/agent.go`):**

```go
import "github.com/yaoapp/yao/openapi/agent/robot"

func Attach(group *gin.RouterGroup, oauth types.OAuth) {
    group.Use(oauth.Guard)
    
    // Assistant routes (existing)
    group.GET("/assistants", ListAssistants)
    group.POST("/assistants", CreateAssistant)
    // ...
    
    // Robot routes (NEW)
    robot.Attach(group.Group("/robots"), oauth)
}
```

**Robot Route Registration (`robot/robot.go`):**

```go
package robot

func Attach(group *gin.RouterGroup, oauth types.OAuth) {
    // Robot CRUD
    group.GET("", ListRobots)
    group.POST("", CreateRobot)
    group.GET("/:id", GetRobot)
    group.PUT("/:id", UpdateRobot)
    group.DELETE("/:id", DeleteRobot)
    
    // Activities (before :id to avoid conflict)
    group.GET("/activities", ListActivities)
    group.GET("/stream", StreamRobots)
    
    // Execution management
    group.GET("/:id/executions", ListExecutions)
    group.GET("/:id/executions/:exec_id", GetExecution)
    group.GET("/:id/executions/:exec_id/stream", StreamExecution)
    group.POST("/:id/executions/:exec_id/pause", PauseExecution)
    group.POST("/:id/executions/:exec_id/resume", ResumeExecution)
    group.POST("/:id/executions/:exec_id/cancel", CancelExecution)
    group.POST("/:id/executions/:exec_id/retry", RetryExecution)
    
    // Trigger & Intervene (SSE)
    group.POST("/:id/trigger", TriggerRobot)
    group.POST("/:id/intervene", InterveneRobot)
    
    // Results
    group.GET("/:id/results", ListResults)
    group.GET("/:id/results/:result_id", GetResult)
}
```

---

## 11. Error Handling

### 11.1 Error Response Format

```json
{
  "error": {
    "code": "ROBOT_NOT_FOUND",
    "message": "Robot not found",
    "details": {
      "member_id": "robot_001"
    }
  }
}
```

### 11.2 Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ROBOT_NOT_FOUND | 404 | Robot does not exist |
| EXECUTION_NOT_FOUND | 404 | Execution does not exist |
| ROBOT_BUSY | 409 | Robot at max capacity |
| TRIGGER_DISABLED | 403 | Trigger type disabled |
| EXECUTION_NOT_RUNNING | 400 | Cannot pause/resume non-running execution |
| INVALID_REQUEST | 400 | Request validation failed |
| UNAUTHORIZED | 401 | Not authenticated |
| FORBIDDEN | 403 | No permission |

---

## 12. Implementation Notes

### 12.1 Backend Architecture: Store + API Layers

> **Principle:** Store layer handles database CRUD, API layer handles business logic.
> This enables reuse across Golang API, JSAPI, and Yao Process.

```
Consumers (Golang API / JSAPI / Yao Process)
              │
              ▼
┌─────────────────────────────────────────┐
│         API Layer (robot/api/)          │
│  Thin wrappers: validation, cache ops   │
└─────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│       Store Layer (robot/store/)        │
│  Core CRUD: RobotStore, ExecutionStore  │
└─────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│       Model Layer (__yao.member)        │
└─────────────────────────────────────────┘
```

### 12.2 Store Layer Extensions

**File: `store/robot.go` (NEW)** - Core Robot CRUD

```go
type RobotStore struct {
    modelID string  // "__yao.member"
}

func (s *RobotStore) Save(ctx context.Context, record *RobotRecord) error
func (s *RobotStore) Get(ctx context.Context, memberID string) (*RobotRecord, error)
func (s *RobotStore) List(ctx context.Context, opts *ListOptions) ([]*RobotRecord, error)
func (s *RobotStore) Delete(ctx context.Context, memberID string) error
func (s *RobotStore) UpdateConfig(ctx context.Context, memberID string, config map[string]interface{}) error
```

**File: `store/execution.go` (extend)**

```go
func (s *ExecutionStore) ListResults(ctx context.Context, memberID string, opts *ResultsQuery) ([]*ResultRecord, error)
func (s *ExecutionStore) GetResult(ctx context.Context, resultID string) (*ResultRecord, error)
func (s *ExecutionStore) ListActivities(ctx context.Context, opts *ActivityQuery) ([]*ActivityRecord, error)
```

### 12.3 API Layer Extensions

**File: `api/robot.go` (extend)** - Thin wrappers

```go
// Create - calls store.RobotStore.Save() + cache refresh
func Create(ctx *types.Context, teamID string, req *CreateRobotRequest) (*types.Robot, error)

// Update - calls store.RobotStore.UpdateConfig() + cache refresh
func Update(ctx *types.Context, memberID string, req *UpdateRobotRequest) (*types.Robot, error)

// Remove - calls store.RobotStore.Delete() + cache invalidate
func Remove(ctx *types.Context, memberID string) error
```

**File: `api/results.go` (NEW)**

```go
func ListResults(ctx *types.Context, memberID string, query *ResultQuery) (*ResultsResult, error)
func GetResult(ctx *types.Context, resultID string) (*ResultFile, error)
```

**File: `api/activities.go` (NEW)**

```go
func ListActivities(ctx *types.Context, query *ActivityQuery) (*ActivitiesResult, error)
```

### 12.4 Localization

Add `Locale` parameter support for localized responses.

### 12.5 SSE Implementation

Use standard Go SSE pattern:

```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, _ := w.(http.Flusher)
    
    for event := range events {
        fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
        flusher.Flush()
    }
}
```

### 12.6 Localization Strategy

- Store display names in `__yao.member.display_name` (single language) initially
- Future: Add `display_name_cn`, `display_name_en` or use JSON `{"en": "...", "cn": "..."}`
- Execution names derived from goals or input message
- Activities derive titles from execution data

---

## 13. API Base Path Decision

Based on analysis of existing `openapi/` structure:

| Option | Path | Pros | Cons |
|--------|------|------|------|
| ❌ A | `/v1/robots` | Shorter path | New namespace, inconsistent |
| ✅ B | `/v1/agent/robots` | Groups with agent APIs, consistent | Longer path |
| ❌ C | `/v1/members?type=robot` | Uses existing members | Less intuitive |

**Decision**: Use `/v1/agent/robots` as base path.

**Rationale:**
1. `openapi/agent/` already exists with `/v1/agent/assistants`
2. Robot is conceptually an Agent type (Autonomous Robot Agent)
3. Follows the established pattern: `/v1/agent/{agent-type}`
4. Keeps agent-related APIs logically grouped

**Frontend Impact:**
- Update `cui/packages/cui/pages/mission-control/API.md` base path from `/v1/robots` to `/v1/agent/robots`
- Minimal code change (just update base URL constant)

---

## 14. References

- Frontend API Requirements: `cui/packages/cui/pages/mission-control/API.md`
- Backend Robot Design: `yao/agent/robot/DESIGN.md`
- Backend Technical Spec: `yao/agent/robot/TECHNICAL.md`
- Existing OpenAPI Patterns: `yao/openapi/kb/`, `yao/openapi/chat/`
