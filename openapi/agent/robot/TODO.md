# Robot OpenAPI - Implementation TODO

> Based on: `openapi/agent/robot/DESIGN.md`, `openapi/agent/robot/GAPS.md`
> Depends on: `yao/agent/robot/api/` (Go API layer)
> Base Path: `/v1/agent/robots`

---

## Field Alignment Review Summary

> Last reviewed: 2026-01-23

### Robot Fields ✅ Fully Aligned

| Backend (`types.go`) | Frontend (`types.ts`) | Status |
|---------------------|----------------------|--------|
| `member_id` | `member_id` | ✅ |
| `team_id` | `team_id` | ✅ |
| `display_name` | `display_name` | ✅ |
| `bio` | `bio` / `description` | ✅ |
| `name` (← member_id) | `name` | ✅ |
| `description` (← bio) | `description` | ✅ |
| `robot_status` | `robot_status` | ✅ |
| `autonomous_mode` | `autonomous_mode` | ✅ |
| `robot_config` | `robot_config` | ✅ |
| `robot_email` | `robot_email` | ✅ |
| All other fields | Same | ✅ |

### Execution Fields ✅ Aligned

| Backend (`types.go`) | Frontend (`types.ts`) | Status |
|---------------------|----------------------|--------|
| `id` | `id` | ✅ Aligned |
| `member_id` | `member_id` | ✅ |
| `team_id` | `team_id` | ✅ |
| `trigger_type` | `trigger_type` | ✅ |
| `status` | `status` | ✅ |
| `phase` | `phase` | ✅ |
| `start_time` | `start_time` | ✅ |
| `end_time` | `end_time` | ✅ |
| `error` | `error` | ✅ |
| `input` | `input` | ✅ Optional |
| Phase outputs | Same | ✅ Detail view |
| `name` | `name` | ✅ Added |
| `current_task_name` | `current_task_name` | ✅ Added |
| - | `job_id` | 🗑️ **Dead field, to be removed** |

### Task Fields ✅ Aligned

| Backend (`types.go`) | Frontend (`types.ts`) | Status |
|---------------------|----------------------|--------|
| `id` | `id` | ✅ |
| `description` | `description` | ✅ Added |
| `goal_ref` | `goal_ref` | ✅ |
| `source` | `source` | ✅ |
| `executor_type` | `executor_type` | ✅ |
| `executor_id` | `executor_id` | ✅ |
| `status` | `status` | ✅ |
| `order` | `order` | ✅ |
| `start_time` | `start_time` | ✅ |
| `end_time` | `end_time` | ✅ |

**Action Items:**
- [x] **Backend**: `Execution` struct - add `Name`, `CurrentTaskName` fields (see Improvement Plan below)
- [x] **Backend**: `RobotConfig` struct - add `DefaultLocale` field (see Improvement Plan below)
- [x] **Backend**: `TriggerInput` struct - add `Locale` field (see Improvement Plan below)
- [x] **Backend**: Database model `execution.mod.yao` - add `name`, `current_task_name` columns
- [x] **Backend**: Executor - update `Name`, `CurrentTaskName` at each phase
- [x] **Backend**: Store layer - add `UpdateUIFields()` method
- [x] **Backend**: Unit tests for UI fields and i18n (executor/standard/ui_fields_test.go, store/execution_test.go)
- [x] **Backend**: `Task` struct - add `Description` field for human-readable task description
- [x] **Backend**: `ParseTask()` - save description from LLM output to `Task.Description`
- [x] **Frontend**: `Task` type - add `description` field
- [x] **Frontend**: Task list display - use `description` as primary title, fallback to `executor_id`
- [ ] **Frontend**: Remove `job_id` field from `types.ts`
- [ ] **Frontend**: Remove `job_id` mock data from `mock/data.ts`
- [ ] **Frontend**: Use `name` and `current_task_name` directly from API response

---

### Improvement Plan: Execution UI Display Fields ✅ Implemented

> **Problem:** Frontend needs to display "execution title" and "current task", which must be dynamically updated at different phases
> **Solution:** Backend manages these fields centrally; `Execution` struct gets new fields, executor updates them at each phase

**1. Execution struct fields (`agent/robot/types/robot.go`):** ✅
```go
type Execution struct {
    // ... existing fields ...
    
    // UI display fields (updated by executor at each phase)
    Name            string `json:"name,omitempty"`             // Execution title
    CurrentTaskName string `json:"current_task_name,omitempty"` // Current task description
}
```

**2. Update timeline:** ✅

| Phase | `Name` | `CurrentTaskName` |
|-------|--------|-------------------|
| Created | Human: extract from `input.messages[0]`<br>Clock/Event: "Preparing..." (localized) | "Starting..." (localized) |
| `inspiration` | - | "Analyzing context..." (localized) |
| `goals` complete | Extract first line from `goals.content` | "Planning goals..." (localized) |
| `tasks` | - | "Breaking down tasks..." (localized) |
| `run` (each task) | - | Current `task` description (e.g., "Task 1/3: ...") |
| Completed/Failed | - | "Completed" / "Failed: {error}" (localized) |

**3. Implementation files:**
- `agent/robot/types/robot.go` - Execution struct fields ✅
- `agent/robot/store/execution.go` - UpdateUIFields() method ✅
- `agent/robot/executor/standard/executor.go` - initUIFields(), updateUIFields(), i18n messages ✅
- `agent/robot/executor/standard/inspiration.go` - Update CurrentTaskName ✅
- `agent/robot/executor/standard/goals.go` - Update Name and CurrentTaskName ✅
- `agent/robot/executor/standard/tasks.go` - Update CurrentTaskName ✅
- `agent/robot/executor/standard/run.go` - Update CurrentTaskName for each task ✅
- `yao/models/agent/execution.mod.yao` - Database columns ✅

---

### Improvement Plan: i18n Default Locale ✅ Implemented

> **Problem:** Clock/Event triggers have no user context, unknown which language to use for generated content
> **Solution:** `RobotConfig` gets a default locale configuration field

**1. RobotConfig struct field (`agent/robot/types/config.go`):** ✅
```go
type Config struct {
    // ... existing fields ...
    DefaultLocale string `json:"default_locale,omitempty"` // "en" | "zh", default "en"
}

// GetDefaultLocale returns the default locale (default: "en")
func (c *Config) GetDefaultLocale() string {
    if c == nil || c.DefaultLocale == "" {
        return "en"
    }
    return c.DefaultLocale
}
```

**2. TriggerInput struct field (`agent/robot/types/robot.go`):** ✅
```go
type TriggerInput struct {
    // ... existing fields ...
    Locale string `json:"locale,omitempty"` // Language from human trigger
}
```

**3. Locale determination logic (`agent/robot/executor/standard/executor.go`):** ✅
```go
func getEffectiveLocale(robot *Robot, input *TriggerInput) string {
    // 1. Human trigger: use locale from request
    if input != nil && input.Locale != "" {
        return input.Locale
    }
    // 2. Clock/Event trigger: use Robot config
    if robot != nil && robot.Config != nil {
        return robot.Config.GetDefaultLocale()
    }
    // 3. System default
    return "en"
}
```

**4. Locale source priority:** ✅

| Trigger Type | Locale Source |
|--------------|---------------|
| Human | Request `locale` → Robot `default_locale` → "en" |
| Event | Robot `default_locale` → "en" |
| Clock | Robot `default_locale` → "en" |

**5. Localized messages (`executor.go`):** ✅
```go
var uiMessages = map[string]map[string]string{
    "en": {
        "preparing":           "Preparing...",
        "starting":            "Starting...",
        "scheduled_execution": "Scheduled execution",
        "event_prefix":        "Event: ",
        "event_triggered":     "Event triggered",
        "analyzing_context":   "Analyzing context...",
        "planning_goals":      "Planning goals...",
        "breaking_down_tasks": "Breaking down tasks...",
        "completed":           "Completed",
        "failed_prefix":       "Failed: ",
        "task_prefix":         "Task",
    },
    "zh": {
        "preparing":           "准备中...",
        "starting":            "启动中...",
        "scheduled_execution": "定时执行",
        // ... more Chinese messages
    },
}
```

> **Note:** User preference locale fallback deferred to future version

### Deferred Features (Phase 5/6)

| Feature | Current Status | Future Plan |
|---------|---------------|-------------|
| Trigger/Intervene UI | Backend done, frontend deferred | Phase 5 (requires SSE) |
| Real-time refresh | Polling 60s | Phase 6 (SSE streams) |
| Multi-turn chat | Not started | Phase 5 |

---

## Implementation Strategy

> **Integrate frontend immediately after each phase to validate deliverables.**
> Frontend has fallback mechanisms (polling, single-submit mode).

```
🟢 Phase 1: Core CRUD ✅
  Backend → SDK → Page Integration
  └─ List, Get, Create, Update, Delete robots

✅ Phase 1-FE: Frontend Integration ✅ [Completed]
  └─ SDK (openapi/robot.ts) ✅
  └─ Page Integration (Robot list, detail, create, edit, delete) ✅
  └─ UI/UX (CreatureLoading, bubble animations) ✅

✅ Phase 1.5: Robot Manager Lifecycle ✅ [Completed]
  └─ Auto-start Manager on Yao startup (async)
  └─ Auto-reload cache on robot update
  └─ Auto-remove from cache on robot delete
  └─ Graceful shutdown on Yao unload
  └─ Lazy-load for non-autonomous robots (load on trigger, unload after execution)
  └─ Unit tests: TestManagerLazyLoadNonAutonomous (6 test cases)

🟢 Phase 2: Execution Management
  Backend → SDK → Page Integration
  └─ List, Get, Control executions, Trigger/Intervene

🟢 Phase 3: Results & Activities
  Backend → SDK → Page Integration
  └─ List deliverables, Activity feed

🟢 Phase 4: i18n
  Backend → SDK → Page Integration
  └─ Locale parameter support

🟡 Medium Risk (Deferred):
  Phase 5: Multi-turn Chat API + Trigger/Intervene UI
  Phase 6: Real-time SSE Streams (replace polling)
```

---

## 🟢 Phase 1: Core CRUD ✅ [Low Risk]

**Goal:** Basic robot management endpoints
**Risk:** 🟢 Low - All new code, no changes to existing logic
**Status:** ✅ Backend Complete → Proceed to Phase 1.5 Frontend Integration

### 1.1 Backend Prerequisites ✅

#### Types & Cache
- [x] Add `Bio` field to `types.Robot` struct in `yao/agent/robot/types/robot.go`
- [x] Add `bio` to `memberFields` in `yao/agent/robot/cache/load.go`

#### Store Layer (Core CRUD - implement first)
- [x] Create `store/robot.go` with `RobotStore` struct
- [x] Implement `RobotStore.Save()` - create/update robot member
- [x] Implement `RobotStore.Get()` - get by member_id  
- [x] Implement `RobotStore.List()` - list with filters
- [x] Implement `RobotStore.Delete()` - delete robot member
- [x] Implement `RobotStore.UpdateConfig()` - update config only
- [x] Implement `RobotStore.UpdateStatus()` - update status only
- [x] Add Yao permission fields support (`__yao_created_by`, `__yao_team_id`, etc.)
- [x] Add tests: `store/robot_test.go`

#### API Layer (Thin wrappers calling store)
- [x] Implement `api.CreateRobot()` - call `store.RobotStore.Save()` + cache refresh
  - [x] Auto-generate `member_id` if not provided (12-digit numeric, matches existing pattern)
- [x] Implement `api.UpdateRobot()` - partial update + cache refresh
- [x] Implement `api.RemoveRobot()` - call `store.RobotStore.Delete()` + cache invalidate
- [x] Implement `api.GetRobotResponse()` - get robot as API response
- [x] Add `AuthScope` for Yao permission fields
- [x] Add request/response types in `api/types.go`
- [x] Add tests: `api/robot_test.go`

#### Utils Layer
- [x] Create `utils/convert.go` with unified type conversion functions
- [x] Implement `To<Type>` functions (ToBool, ToInt, ToFloat64, ToTimestamp, ToJSONValue)
- [x] Implement `Get<Type>` functions for map value extraction
- [x] Add tests: `utils/convert_test.go`

### 1.2 OpenAPI Setup ✅

- [x] Create `openapi/agent/robot/` directory (sub-package under agent)
- [x] Create `robot.go` - route registration with `Attach()` function
- [x] Register routes in `openapi/agent/agent.go` via `robot.Attach(group.Group("/robots"), oauth)`
- [x] Add OAuth guard middleware

### 1.3 OpenAPI Types ✅

> Note: Core types already exist in `agent/robot/api/types.go`. OpenAPI layer needs HTTP-specific types.

- [x] `types.go` - HTTP request/response types
  - [x] `RobotResponse` struct (with field mapping: `name` ← `member_id`, `description` ← `bio`)
  - [x] `RobotStatusResponse` struct
  - [x] `ListRobotsResponse` struct
  - [x] `CreateRobotRequest` struct (HTTP binding)
  - [x] `UpdateRobotRequest` struct (HTTP binding)
  - [x] `NewRobotResponse()` - conversion from `api.RobotResponse`
  - [x] `NewRobotStatusResponse()` - conversion from `api.RobotState`

### 1.4 List Robots ✅

- [x] `list.go` - GET /v1/agent/robots
- [x] Parse query params: `status`, `keywords`, `page`, `pagesize`, `team_id`
- [x] Call `robot/api.ListRobots()`
- [x] Team constraint from auth info
- [x] Test: `tests/agent/robot_test.go#TestListRobots`

### 1.5 Get Robot ✅

- [x] `detail.go` - GET /v1/agent/robots/:id
- [x] Parse path param
- [x] Call `robot/api.GetRobotResponse()`
- [x] Team access check
- [x] Test: `tests/agent/robot_test.go#TestGetRobot`

### 1.6 Create Robot ✅

- [x] POST /v1/agent/robots handler
- [x] Parse HTTP request to `CreateRobotRequest`
- [x] Auto-generate `member_id` if not provided (12-digit numeric, consistent with existing API)
- [x] Apply `AuthScope` with permission fields (CreatedBy, TeamID, TenantID)
- [x] Call `robot/api.CreateRobot()`
- [x] Return created robot (201 Created)
- [x] Handle duplicate (409 Conflict)
- [x] Test: `tests/agent/robot_test.go#TestCreateRobot`

### 1.7 Update Robot ✅

- [x] PUT /v1/agent/robots/:id handler
- [x] Parse HTTP request to `UpdateRobotRequest`
- [x] Team permission check
- [x] Apply `AuthScope` with UpdatedBy
- [x] Call `robot/api.UpdateRobot()`
- [x] Return updated robot
- [x] Test: `tests/agent/robot_test.go#TestUpdateRobot`

### 1.8 Delete Robot ✅

- [x] DELETE /v1/agent/robots/:id handler
- [x] Team permission check
- [x] Call `robot/api.RemoveRobot()`
- [x] Handle running executions (409 Conflict)
- [x] Return success response
- [x] Test: `tests/agent/robot_test.go#TestDeleteRobot`

### 1.9 Status Endpoint ✅

- [x] GET /v1/agent/robots/:id/status handler
- [x] Call `robot/api.GetRobotStatus()`
- [x] Return runtime status (running count, max, last/next run)
- [x] Test: `tests/agent/robot_test.go#TestGetRobotStatus`

### 1.10 Utilities ✅

- [x] `utils.go` - helper functions
  - [x] `GetLocale(c *gin.Context)` - extract locale from query/header
  - [x] `ParseBoolValue()` - parse bool from string

### 1.11 Permission Logic ✅

- [x] `permission.go` - permission check functions
  - [x] `CanRead()` - read permission check (creator or team member)
  - [x] `CanWrite()` - write permission check (creator only)
  - [x] `GetEffectiveTeamID()` - get effective team_id (user_id for personal users)
  - [x] `BuildListFilter()` - build list filter based on permissions
- [x] Apply permission checks in handlers:
  - [x] `GetRobot` - check `CanRead()` with `YaoTeamID` and `YaoCreatedBy`
  - [x] `GetRobotStatus` - check `CanRead()`
  - [x] `UpdateRobot` - check `CanWrite()`
  - [x] `DeleteRobot` - check `CanWrite()`
  - [x] `ListRobots` - use `BuildListFilter()` for team filtering
  - [x] `CreateRobot` - auto-set `__yao_team_id` to `user_id` for personal users
- [x] Add Yao permission fields to API layer:
  - [x] `api/types.go` - add `YaoCreatedBy`, `YaoTeamID` to `RobotResponse` and `RobotState`
  - [x] `api/robot.go` - populate permission fields in `recordToResponse()` and `GetRobotStatus()`
  - [x] `store/robot.go` - add `__yao_*` fields to `robotFields`
- [x] Permission tests in `tests/agent/robot_test.go#TestRobotPermissions`

---

## ✅ Phase 1-FE: Frontend Integration ✅ [Completed]

**Goal:** Implement frontend SDK and integrate pages to validate Phase 1 deliverables
**Status:** ✅ Completed

### 1-FE.1 SDK Implementation ✅

> Location: `cui/packages/cui/openapi/agent/robot/`

- [x] Create `robot/types.ts` - TypeScript types for Robot API
  - [x] `RobotFilter` - filter options for listing (including `autonomous_mode`)
  - [x] `Robot` - robot data structure
  - [x] `RobotStatusResponse` - runtime status
  - [x] `RobotCreateRequest` / `RobotUpdateRequest` - CRUD requests
  - [x] `RobotDeleteResponse` - delete response
- [x] Create `robot/robots.ts` - Robot API SDK class (`AgentRobots`)
  - [x] `List(filter)` - GET /v1/agent/robots
  - [x] `Get(id)` - GET /v1/agent/robots/:id
  - [x] `GetStatus(id)` - GET /v1/agent/robots/:id/status
  - [x] `Create(data)` - POST /v1/agent/robots
  - [x] `Update(id, data)` - PUT /v1/agent/robots/:id
  - [x] `Delete(id)` - DELETE /v1/agent/robots/:id
- [x] Create `robot/index.ts` - exports
- [x] Update `agent/api.ts` - add `robots` property to Agent class
- [x] Update `agent/index.ts` - export robot module
- [x] Linter check passed

### 1-FE.2 Page Integration ✅

> Location: `cui/packages/cui/pages/mission-control/`

- [x] Create `useRobots` hook for API calls
  - [x] `listRobots(filter)` - list robots with pagination
  - [x] `getRobot(id)` - get single robot
  - [x] `getRobotStatus(id)` - get runtime status
  - [x] `createRobot(data)` - create robot
  - [x] `updateRobot(id, data)` - update robot
  - [x] `deleteRobot(id)` - delete robot
  - [x] Error handling and loading state
- [x] Robot List Page (`mission-control/index.tsx`)
  - [x] Replace mock data with `listRobots()` API (fallback to mock)
  - [x] Fetch status for each robot via `getRobotStatus()`
  - [x] Refresh list after robot created/updated/deleted
  - [x] Empty state with "Create Agent" button (with bubble animation)
  - [ ] Implement pagination (TODO: Phase 2)
  - [ ] Implement filters (status, keywords, team) (TODO: Phase 2)
- [x] Robot Detail Modal (`AgentModal`)
  - [x] Real-time status refresh via `getRobotStatus(id)`
  - [x] Auto-refresh every 10 seconds while modal open
  - [x] Merge real-time status with robot data
- [x] Create Robot (`AddAgentModal`)
  - [x] Call `createRobot()` API
  - [x] Handle success/error messages
  - [x] Form validation (existing)
  - [x] Load email domains, managers, agents, MCP servers from API
- [x] Edit Robot (`ConfigTab` in `AgentModal`)
  - [x] Load robot data from API (`getRobot()`)
  - [x] Load email domains, managers, roles from Team API
  - [x] Load agents and MCP servers from API
  - [x] Pre-populate form with existing data
  - [x] Call `updateRobot()` API with `robot_config.clock` for schedule
  - [x] Handle success/error messages
  - [x] Work Schedule panel saves correctly
- [x] Delete Robot (`AdvancedPanel` in `ConfigTab`)
  - [x] Confirmation dialog with name input
  - [x] Call `deleteRobot()` API
  - [x] Handle running execution conflict (409)
  - [x] Refresh list after deletion

### 1-FE.3 UI/UX Enhancements ✅

- [x] `CreatureLoading` component with organic animations
  - [x] Breathing aura, floating creature, orbit ring, particles
  - [x] Three sizes: small, medium, large
  - [x] Used in ConfigTab, ResultsTab, HistoryTab
- [x] Empty state "Create Agent" button with bubble animation
  - [x] Cyan, purple, pink glowing bubbles rising
- [x] CSS variable compliance (`--color_mission_button_text`)
- [x] Consistent loading animations across all tabs

### 1-FE.4 Verification ✅

- [x] Manual test: Create → List → Get → Update → Delete
- [ ] E2E automated test (TODO: Phase 3)
- [x] Permission test: Personal user vs Team user (manual tested)
- [x] Error handling: 400, 403, 404, 409, 500

---

## 🟢 Phase 2: Execution Management [Backend ✅ | Frontend ⬜]

> **Backend:** Steps 1-4 ✅ Complete (including UI fields and i18n)
> **Frontend:** Step 5 ⬜ Pending
> **Deferred:** Trigger/Intervene UI → Phase 5 (requires SSE)

**Goal:** Execution listing, details, control, and trigger/intervene (single-submit mode)
**Risk:** 🟢 Low - Wraps existing `robot/api` functions
**Workflow:** 1. Implement All Endpoints → 2. Linter Check → 3. Code Review → 4. Unit Tests → 5. Frontend Integration

---

### Step 1: Implement All OpenAPI Endpoints ✅

> Location: `yao/openapi/agent/robot/`
> Calls: `yao/agent/robot/api/` (existing functions)

#### 2.1.1 Types (`types.go`) ✅

- [x] `ExecutionFilter` - query params for listing
- [x] `ExecutionResponse` - single execution response
- [x] `ExecutionListResponse` - paginated list response
- [x] `ExecutionControlResponse` - pause/resume/cancel response
- [x] `TriggerRequest` - trigger execution request
- [x] `TriggerResponse` - trigger result response
- [x] `InterveneRequest` - human intervention request
- [x] `InterveneResponse` - intervention result response

#### 2.1.2 Execution Handlers (`execution.go`) ✅

> **Permission Note:** Execution permissions are inherited from the parent robot.
> Check robot's `__yao_team_id` and `__yao_created_by` for access control.

- [x] `ListExecutions` - GET /v1/agent/robots/:id/executions
  - Parse query: `status`, `trigger_type`, `keyword`, `page`, `pagesize`
  - Call `robot/api.ListExecutions()`
  - Permission: Check robot CanRead (via robot ID)
- [x] `GetExecution` - GET /v1/agent/robots/:id/executions/:exec_id
  - Call `robot/api.GetExecution()`
  - Permission: Check robot CanRead (via robot ID)
- [x] `PauseExecution` - POST /v1/agent/robots/:id/executions/:exec_id/pause
  - Call `robot/api.PauseExecution()`
  - Permission: Check robot CanWrite (via robot ID)
- [x] `ResumeExecution` - POST /v1/agent/robots/:id/executions/:exec_id/resume
  - Call `robot/api.ResumeExecution()`
  - Permission: Check robot CanWrite (via robot ID)
- [x] `CancelExecution` - POST /v1/agent/robots/:id/executions/:exec_id/cancel
  - Call `robot/api.StopExecution()`
  - Permission: Check robot CanWrite (via robot ID)

#### 2.1.3 Trigger Handlers (`trigger.go`) ✅

> **Permission Note:** Same as execution - check robot's permission.

- [x] `TriggerRobot` - POST /v1/agent/robots/:id/trigger
  - Parse `TriggerRequest` (messages, trigger_type)
  - Call `robot/api.Trigger()`
  - Return execution ID and status
  - Permission: Check robot CanWrite (via robot ID)
- [x] `InterveneRobot` - POST /v1/agent/robots/:id/intervene
  - Parse `InterveneRequest` (action, messages)
  - Call `robot/api.Intervene()`
  - Return result
  - Permission: Check robot CanWrite (via robot ID)

#### 2.1.4 Route Registration (`robot.go`) ✅

- [x] Add execution routes to `Attach()`:
  - `GET /:id/executions`
  - `GET /:id/executions/:exec_id`
  - `POST /:id/executions/:exec_id/pause`
  - `POST /:id/executions/:exec_id/resume`
  - `POST /:id/executions/:exec_id/cancel`
  - `POST /:id/trigger`
  - `POST /:id/intervene`

---

### Step 2: Linter Check ✅

- [x] Run `ReadLints` on all modified files
- [x] Fix any linter errors
- [x] Verify imports are correct
- [x] Build verification passed

---

### Step 3: Code Review ✅

- [x] Review type definitions (`types.go`)
  - `ExecutionFilter`, `ExecutionResponse`, `ExecutionListResponse`, `ExecutionControlResponse`
  - `TriggerRequest`, `TriggerResponse`, `InterveneRequest`, `InterveneResponse`
  - Conversion functions: `NewExecutionListResponse`, `NewExecutionResponseFromExecution`, `NewExecutionResponseBrief`
- [x] Review permission handling
  - All execution/trigger handlers check robot permission first
  - Read permission for listing and getting executions
  - Write permission for control (pause/resume/cancel), trigger, and intervene
  - Permission inherited from parent robot (check via `YaoTeamID` and `YaoCreatedBy`)
- [x] Review error handling
  - Fixed: Use `errors.Is()` instead of `==` for error comparison
  - Proper HTTP status codes (400, 404, 403, 500)
  - Consistent error response format
- [x] Review response formats
  - Brief format for list view (omits phase outputs)
  - Full format for detail view (includes all fields)
  - Consistent with existing robot responses

---

### Step 4: Unit Tests ✅

> Location: `yao/openapi/tests/agent/`
> Uses `testing.Short()` to skip AI/manager-dependent tests

- [x] Create `robot_execution_test.go`
  - [x] `TestListExecutions` - list executions with pagination/filters
  - [x] `TestGetExecution` - get execution details, not found cases
  - [x] `TestExecutionControl` - pause/resume/cancel endpoints
  - [x] `TestExecutionPermissions` - permission inheritance from robot
- [x] Create `robot_trigger_test.go`
  - [x] `TestTriggerRobot` - trigger with messages, action, invalid body
  - [x] `TestInterveneRobot` - intervene with action, missing action validation
  - [x] `TestTriggerPermissions` - permission inheritance from robot
- [x] All tests use `testing.Short()` to skip AI-dependent tests
- [x] Tests compile successfully
- [x] All tests pass (with manager not started gracefully handled)

---

### Step 5: Frontend Integration ⬜

> Location: `cui/packages/cui/openapi/agent/robot/`
> **Note:** Trigger/Intervene API deferred to Phase 5 (waiting for SSE support)
> **Note:** Use 1-minute polling for execution list refresh (will switch to SSE in Phase 6)

#### 5.1 Prerequisites ✅

> **Dependency:** Backend improvement plans completed (see "Improvement Plan" sections above)

**Backend (Completed):**
- [x] `Execution` struct - add `Name`, `CurrentTaskName` fields
- [x] `RobotConfig` struct - add `DefaultLocale` field
- [x] `TriggerInput` struct - add `Locale` field
- [x] Executor - update `Name`, `CurrentTaskName` at each phase
- [x] Store - add `UpdateUIFields()` method
- [x] Unit tests for UI fields and i18n

**Frontend Cleanup (Completed):**
- [x] Components already use `exec.id` (no changes needed)
- [x] Remove `job_id` field from `types.ts`
- [x] ~~Remove `job_id` from `mock/data.ts`~~ (mock kept for reference, not used)
- [x] Use `name`/`current_task_name` directly from API response (string, not `{en, cn}`)

#### 5.2 SDK Types (`types.ts`) ✅

- [x] `ExecutionFilter` interface
- [x] `ExecutionResponse` interface (align with backend)
- [x] `ExecutionListResponse` interface
- [x] `ExecutionControlResponse` interface
- [x] `ExecStatus`, `TriggerType`, `Phase` type aliases

**Deferred to Phase 5 (SSE):**
- [ ] ~~`TriggerRequest` / `TriggerResponse` interfaces~~
- [ ] ~~`InterveneRequest` / `InterveneResponse` interfaces~~

#### 5.3 SDK Methods (`robots.ts`) ✅

- [x] `ListExecutions(robotId, filter)`
- [x] `GetExecution(robotId, execId)`
- [x] `PauseExecution(robotId, execId)`
- [x] `ResumeExecution(robotId, execId)`
- [x] `CancelExecution(robotId, execId)`

**Deferred to Phase 5 (SSE):**
- [ ] ~~`Trigger(robotId, data)`~~
- [ ] ~~`Intervene(robotId, data)`~~

#### 5.4 Page Integration ✅

- [x] ActiveTab: Replace mock with `ListExecutions()` API
  - [x] Filter: `status=running|pending`
  - [x] Polling: 1-minute interval (60000ms) - will switch to SSE in Phase 6
- [x] HistoryTab: Replace mock with `ListExecutions()` API
  - [x] Filter: `status` filter, `keyword` search
  - [x] Pagination: page/pagesize
  - [x] Polling: 1-minute interval for list refresh
- [x] Execution Detail: Call `GetExecution()` API
  - [x] Display execution phases and outputs
  - [x] Display `name` and `current_task_name` from API
  - [x] Display `error` field for failed executions
  - [x] Execution controls: Pause/Resume/Cancel buttons (call control APIs)
  - [x] Auto-refresh while execution is running (5s for running)
- [x] useRobots hook extended with execution methods

**Deferred to Phase 5 (SSE):**
- [ ] ~~Assign Task Modal: Call `Trigger()` API~~
- [ ] ~~GuideExecution: Call `Intervene()` API~~

#### 5.5 Polling vs SSE Strategy

**Current (Phase 2):** Polling
- Refresh execution list every 60 seconds
- Manual refresh button for immediate update
- Acceptable latency for status display

**Future (Phase 6):** SSE Real-time
- `GET /robots/:id/executions/stream` - real-time execution updates
- `GET /robots/stream` - robot status changes
- Instant updates, no polling delay

---

## 🟢 Phase 3: Results & Activities ✅ [Completed]

**Goal:** Deliverables listing and activity feed
**Risk:** 🟢 Low - Read-only queries, derived from existing data
**Status:** ✅ Completed 2026-01-22

> **Implementation Pattern:** Follow Phase 2 approach - Store → API → OpenAPI → Frontend SDK → UI

---

### Step 1: Store Layer ✅

> Location: `yao/agent/robot/store/execution.go`
> Add methods to existing `ExecutionStore` - query from `delivery` field

- [x] `ListResults()` - query completed executions with delivery content
  - Filter by: `member_id`, `team_id`, `trigger_type`, `keyword` (search in name)
  - Only return executions where `delivery.content` is not null
  - Return: `*ResultListResponse` with pagination info
  - Order by: `end_time desc` (newest first)
- [x] `CountResults()` - count total results for pagination
- [x] `GetResult()` - get single execution by ID (reuse existing `Get()`)
- [x] `ListActivities()` - derive activities from execution status changes
  - Query recent executions across all robots (for team)
  - Transform to activity format: `{type, robot_id, execution_id, message, timestamp}`
  - Activity types: `execution.started`, `execution.completed`, `execution.failed`, `execution.cancelled`
  - Filter by: `team_id`, `since` (timestamp), `limit`

**Unit Tests:** `store/execution_test.go` ✅
- [x] `TestListResults` - verify filtering and pagination
- [x] `TestCountResults` - verify count accuracy
- [x] `TestListActivities` - verify activity derivation

---

### Step 2: API Layer ✅

> Location: `yao/agent/robot/api/`
> Thin wrappers calling store methods

**File: `api/results.go`** ✅
- [x] `ResultQuery` struct - query parameters
- [x] `ResultItem` struct - result list item (subset of execution)
- [x] `ResultDetail` struct - full result with delivery content
- [x] `ResultListResponse` struct - paginated response
- [x] `ListResults(ctx, robotID, query)` - call store, transform to response
- [x] `GetResult(ctx, resultID)` - call store, return detail

**File: `api/activities.go`** ✅
- [x] `ActivityQuery` struct - query parameters
- [x] `Activity` struct - activity item
- [x] `ActivityListResponse` struct - response with activities
- [x] `ListActivities(ctx, query)` - call store, transform to response

---

### Step 3: OpenAPI Handlers ✅

> Location: `yao/openapi/agent/robot/`

**File: `results.go`** ✅
- [x] `ListResults` handler - GET /v1/agent/robots/:id/results
  - Parse query params: `trigger_type`, `keyword`, `page`, `pagesize`
  - Check robot permission (read)
  - Call `robotapi.ListResults()`
  - Return `ResultListResponse`
- [x] `GetResult` handler - GET /v1/agent/robots/:id/results/:result_id
  - Check robot permission (read)
  - Call `robotapi.GetResult()`
  - Return `ResultDetailResponse`

**File: `activities.go`** ✅
- [x] `ListActivities` handler - GET /v1/agent/robots/activities
  - Parse query params: `limit`, `since`
  - Use team_id from auth
  - Call `robotapi.ListActivities()`
  - Return `ActivityListResponse`

**Types in `types.go`:** ✅
- [x] `ResultFilter` struct - query params
- [x] `ResultResponse` struct - list item
- [x] `ResultDetailResponse` struct - full detail
- [x] `ResultListResponse` struct - paginated list
- [x] `ActivityResponse` struct - activity item
- [x] `ActivityListResponse` struct - activity list
- [x] Conversion functions: `NewResultResponse()`, `NewResultDetailResponse()`, `NewActivityResponse()`

**Routes in `robot.go`:** ✅
- [x] Register `GET /v1/agent/robots/:id/results` → `ListResults`
- [x] Register `GET /v1/agent/robots/:id/results/:result_id` → `GetResult`
- [x] Register `GET /v1/agent/robots/activities` → `ListActivities`

**OpenAPI Integration Tests:** `openapi/tests/agent/robot_results_activities_test.go` ✅
- [x] `TestListResults` - test with filters, pagination, keyword search
- [x] `TestGetResult` - test single result detail
- [x] `TestListActivities` - test activity feed with `since` and `type` parameters
- [x] `TestResultsPermissions` - test permission checks

**Store Layer Unit Tests:** `agent/robot/store/execution_test.go` ✅
- [x] `filters_by_type_completed` - test filtering by completed type
- [x] `filters_by_type_failed` - test filtering by failed type
- [x] `filters_by_type_invalid_returns_empty` - test invalid type returns empty

**Permissions:** ✅
- [x] Added to `yaobots/openapi/scopes/agent/robots.yml`
- [x] Added to `yaobots/openapi/scopes/alias.yml`
- [x] Added to `yao-dev-app/openapi/scopes/agent/robots.yml`
- [x] Added to `yao-dev-app/openapi/scopes/alias.yml`

---

### Step 4: Frontend SDK ✅

> Location: `cui/packages/cui/openapi/agent/robot/`

**Types in `types.ts`:** ✅
- [x] `ResultFilter` interface
- [x] `Result` interface
- [x] `ResultDetail` interface
- [x] `ResultListResponse` interface
- [x] `Activity` interface
- [x] `ActivityListResponse` interface

**Methods in `robots.ts`:** ✅
- [x] `ListResults(robotId: string, filter?: ResultFilter): Promise<ResultListResponse>`
- [x] `GetResult(robotId: string, resultId: string): Promise<ResultDetail>`
- [x] `ListActivities(params?: { limit?: number, since?: string }): Promise<ActivityListResponse>`

**Hook in `hooks/useRobots.ts`:** ✅
- [x] `listResults` - wrapper for API
- [x] `getResult` - wrapper for API
- [x] `listActivities` - wrapper for API

---

### Step 5: Frontend UI Integration ✅

> Location: `cui/packages/cui/pages/mission-control/`

**Results Tab (`ResultsTab.tsx`):** ✅
- [x] Replace mock data with `listResults()` API
- [x] Implement result detail modal/drawer with `getResult()` API
- [x] Add filtering (trigger type, keyword search)
- [x] Add pagination (infinite scroll)

**Result Detail Modal (`ResultDetailModal/index.tsx`):** ✅
- [x] Updated to use `ResultDetail` type from API
- [x] Displays delivery content (summary, body, attachments)

**Activity Feed:** ✅
- [x] Replace mock data with `listActivities()` API
- [x] Added `loadActivities()` function to fetch from API
- [x] Periodic refresh (30s polling, same as robots)
- [x] Updated Activity Banner to use API data format
- [x] Updated Activity Modal to use API data format
- [x] Added loading and empty states
- [x] Added `type` filter parameter to API (full stack: store → API → OpenAPI → SDK → UI)
- [x] Filter to show only `execution.completed` via API `type` param (not client-side)
- [x] Reset carousel index on data refresh (show latest activity first)
- [x] Click activity item to open result detail modal (overlays activity list)

**Error Handling UI:** ✅
- [x] Error state displays centered in content area (not in toolbar)
- [x] Error state hides empty placeholder
- [x] Retry button for reloading
- [x] Uses CSS variable `--color_danger` (no hardcoded colors)

**Verify:**
- [x] Results display correctly with delivery content
- [x] Attachments show properly
- [x] Error state displays properly with retry option
- [x] Activity feed displays from API (30s polling refresh)
- [x] Activity item click opens result detail

---

### Future Enhancements (Not in current scope)

- [ ] Activity feed real-time updates via SSE/WebSocket
- [ ] Push notifications for new results

---

### API Reference

**GET /v1/agent/robots/:id/results**
```
Query Params:
  - trigger_type: string (clock|human|event)
  - keyword: string (search in summary)
  - page: number (default: 1)
  - pagesize: number (default: 20, max: 100)

Response:
{
  "data": [
    {
      "id": "exec-id",
      "member_id": "robot-id",
      "trigger_type": "clock",
      "status": "completed",
      "name": "Execution title",
      "summary": "Delivery summary...",
      "start_time": "2026-01-24T10:00:00Z",
      "end_time": "2026-01-24T10:05:00Z",
      "has_attachments": true
    }
  ],
  "total": 50,
  "page": 1,
  "pagesize": 20
}
```

**GET /v1/agent/robots/:id/results/:result_id**
```
Response:
{
  "id": "exec-id",
  "member_id": "robot-id",
  "trigger_type": "clock",
  "status": "completed",
  "name": "Execution title",
  "delivery": {
    "content": {
      "summary": "...",
      "body": "...",
      "attachments": [...]
    },
    "success": true,
    "sent_at": "2026-01-24T10:05:00Z"
  },
  "start_time": "2026-01-24T10:00:00Z",
  "end_time": "2026-01-24T10:05:00Z"
}
```

**GET /v1/agent/robots/activities**
```
Query Params:
  - limit: number (default: 20, max: 100)
  - since: string (ISO timestamp, optional)

Response:
{
  "data": [
    {
      "type": "execution.completed",
      "robot_id": "robot-id",
      "robot_name": "Sales Robot",
      "execution_id": "exec-id",
      "message": "Completed: Weekly report generation",
      "timestamp": "2026-01-24T10:05:00Z"
    }
  ]
}
```

---

## 🟢 Phase 4: i18n ⬜ [Low Risk]

**Goal:** Locale parameter support
**Risk:** 🟢 Low - Additive, optional parameter

### 4.1 Locale Handling ⬜

- [ ] Add `getLocale(r *http.Request)` to utils.go
- [ ] Parse locale from query param, body, or header
- [ ] Add `Locale` field to context if needed

### 4.2 Localized Responses ⬜

- [ ] Localize `display_name` in RobotResponse
- [ ] Localize `description` in RobotResponse
- [ ] Localize `name` in ExecutionResponse (derive from goals/input)
- [ ] Localize `current_task_name` in ExecutionResponse

### 4.3 Frontend Integration ⬜

> Integrate immediately after backend completion

- [ ] SDK: Add `locale` parameter support to all API calls
- [ ] Page: Use current language setting when calling APIs
- [ ] Verify: Data correctly localized after language switch

---

## 🟡 Phase 5: Multi-turn Chat API + Trigger/Intervene UI ⬜ [Medium Risk - Deferred]

> **Frontend Fallback:** Single-submit mode (user input → immediate execution)
> **Risk:** 🟡 Medium - New stateful component
> **Dependency:** Requires SSE infrastructure (partially)

**Goal:** Multi-turn conversation before execution + Human trigger/intervene UI

### 5.1 Backend Prerequisites ⬜

- [ ] Create `store/conversation.go` - temporary conversation storage (redis/memory)
- [ ] Create `types/conversation.go` - Conversation, ChatRequest, ChatResponse types
- [ ] Create `api/chat.go` - Chat() handler with LLM call
- [ ] Extend `api/trigger.go` - support `conversation_id` parameter

### 5.2 Chat Endpoint ⬜

- [ ] POST /v1/robots/:id/chat (SSE)
- [ ] Parse ChatRequest (conversation_id, messages, attachments)
- [ ] Create or continue conversation
- [ ] Call LLM for response
- [ ] Store updated conversation
- [ ] Return assistant message + conversation_id
- [ ] Test: `tests/robot/chat_test.go`

### 5.3 Trigger with Conversation ⬜

- [ ] Extend POST /v1/robots/:id/trigger
- [ ] Accept `conversation_id` parameter
- [ ] Use conversation history as execution input
- [ ] Auto-cleanup conversation after execution starts

### 5.4 Frontend Trigger/Intervene Integration (Deferred from Phase 2) ⬜

> **Note:** These features require SSE for proper UX (streaming response)
> Currently backend `/trigger` and `/intervene` endpoints exist but return immediately
> Frontend needs streaming response to show assistant's reaction before confirming

**SDK Types:**
- [ ] `TriggerRequest` / `TriggerResponse` interfaces
- [ ] `InterveneRequest` / `InterveneResponse` interfaces
- [ ] `ChatMessage` interface for multi-turn

**SDK Methods:**
- [ ] `Trigger(robotId, data)` - with SSE support
- [ ] `Intervene(robotId, data)` - with SSE support
- [ ] `Chat(robotId, data)` - multi-turn conversation SSE

**Page Integration:**
- [ ] AssignTaskDrawer: Multi-turn chat before trigger
- [ ] GuideExecutionDrawer: Multi-turn intervention
- [ ] Real-time streaming response display

---

## 🟡 Phase 6: Real-time SSE Streams ⬜ [Medium Risk - Deferred]

> **Frontend Current:** Polling every 60 seconds (1 minute)
> **Frontend Future:** SSE streams for instant updates
> **Risk:** 🟡 Medium - Requires modification of executor/manager

**Goal:** SSE streams for real-time status updates, replacing polling

### 6.1 Backend Event System ⬜

Need to add in `robot/`:

- [ ] Create `events/bus.go` - Event bus for pub/sub
- [ ] Integrate event publishing in `manager/manager.go`
- [ ] Integrate event publishing in `executor/standard/executor.go`
- [ ] Publish: robot_status, execution_start, execution_complete, phase, task events

### 6.2 Robot Status Stream ⬜

- [ ] `stream.go` - stream handlers
- [ ] GET /v1/robots/stream
  - [ ] Subscribe to manager status updates
  - [ ] Stream `robot_status` events
  - [ ] Stream `execution_start` events
  - [ ] Stream `execution_complete` events
  - [ ] Stream `activity` events
- [ ] Test: `tests/robot/stream_test.go`

### 6.3 Execution Progress Stream ⬜

- [ ] GET /v1/robots/:id/executions/:exec_id/stream
  - [ ] Subscribe to execution updates
  - [ ] Stream `phase` events
  - [ ] Stream `task_start` / `task_complete` events
  - [ ] Stream `message` events
  - [ ] Stream `delivery` event
  - [ ] Stream `complete` / `error` events
- [ ] Test: `tests/robot/execution_stream_test.go`

---

## Backend Extensions Required

> **Architecture:** Store layer handles CRUD, API layer handles business logic.
> This enables reuse across Golang API, JSAPI, and Yao Process.

### robot/store/ Extensions (Core CRUD)

| Function | Phase | Risk | Status | Description |
|----------|-------|------|--------|-------------|
| `RobotStore.Save()` | 1 | 🟢 Low | ✅ | Create/update robot member |
| `RobotStore.Get()` | 1 | 🟢 Low | ✅ | Get robot by member_id |
| `RobotStore.List()` | 1 | 🟢 Low | ✅ | List robots with filters |
| `RobotStore.Delete()` | 1 | 🟢 Low | ✅ | Delete robot member |
| `RobotStore.UpdateConfig()` | 1 | 🟢 Low | ✅ | Update config only |
| `RobotStore.UpdateStatus()` | 1 | 🟢 Low | ✅ | Update status only |
| `ExecutionStore.ListResults()` | 3 | 🟢 Low | ⬜ | Query deliverables from executions |
| `ExecutionStore.GetResult()` | 3 | 🟢 Low | ⬜ | Get single deliverable |
| `ExecutionStore.ListActivities()` | 3 | 🟢 Low | ⬜ | Derive activities from history |
| Conversation store | 5 | 🟡 Medium | ⬜ | Temporary chat history (Deferred) |

### robot/types/ Extensions

| Type/Field | Phase | Risk | Status | Description |
|------------|-------|------|--------|-------------|
| `Robot.Bio` | 1 | 🟢 Low | ✅ | Add field, maps to `__yao.member.bio` |
| Execution name derivation | 2 | 🟢 Low | ⬜ | Derive in OpenAPI layer from goals or input |

> **Note:** `Robot.Name` is NOT needed. Frontend `name` maps to existing `Robot.MemberID`.

### robot/cache/ Extensions

| File | Phase | Risk | Status | Description |
|------|-------|------|--------|-------------|
| `load.go` | 1 | 🟢 Low | ✅ | Add `bio` to `memberFields` slice |

### robot/utils/ Extensions

| File | Phase | Risk | Status | Description |
|------|-------|------|--------|-------------|
| `convert.go` | 1 | 🟢 Low | ✅ | Unified type conversion utilities |
| `convert_test.go` | 1 | 🟢 Low | ✅ | Tests for conversion utilities |

### robot/api/ Extensions (Thin wrappers calling store)

| Function | Phase | Risk | Status | Description |
|----------|-------|------|--------|-------------|
| `CreateRobot()` | 1 | 🟢 Low | ✅ | Call `store.RobotStore.Save()` + cache refresh |
| `UpdateRobot()` | 1 | 🟢 Low | ✅ | Partial update + cache refresh |
| `RemoveRobot()` | 1 | 🟢 Low | ✅ | Call `store.RobotStore.Delete()` + cache invalidate |
| `GetRobotResponse()` | 1 | 🟢 Low | ✅ | Get robot as API response |
| `ListResults()` | 3 | 🟢 Low | ⬜ | Call `store.ExecutionStore.ListResults()` |
| `GetResult()` | 3 | 🟢 Low | ⬜ | Call `store.ExecutionStore.GetResult()` |
| `ListActivities()` | 3 | 🟢 Low | ⬜ | Call `store.ExecutionStore.ListActivities()` |
| `RetryExecution()` | 2 | 🟢 Low | ⬜ | Re-trigger with same input |
| `Chat()` | 5 | 🟡 Medium | ⬜ | Multi-turn conversation (Deferred) |

### Event System (Phase 6 - Deferred)

| Component | Phase | Risk | Description |
|-----------|-------|------|-------------|
| Event bus | 6 | 🟡 Medium | Pub/sub for real-time updates |
| Manager events | 6 | 🟡 Medium | Publish robot status changes |
| Executor events | 6 | 🟡 Medium | Publish execution progress |

---

## Testing Strategy

### Test Files Structure

```
yao/openapi/tests/robot/
├── list_test.go
├── get_test.go
├── create_test.go
├── update_test.go
├── delete_test.go
├── execution_list_test.go
├── execution_get_test.go
├── execution_control_test.go
├── trigger_test.go
├── intervene_test.go
├── results_test.go
├── activities_test.go
├── stream_test.go
└── execution_stream_test.go
```

### Test Utilities

- [ ] Create test robot helper
- [ ] Create test execution helper
- [ ] SSE client for streaming tests
- [ ] Mock data generators

---

## Progress Tracking

| Phase | Risk | Backend | Frontend | Description |
|-------|------|---------|----------|-------------|
| 1. Core CRUD | 🟢 | ✅ | ✅ | Robot CRUD endpoints |
| 1-FE Frontend Integration | 🟢 | - | ✅ | SDK ✅, Page Integration ✅, UI/UX ✅ |
| 1.5 Manager Lifecycle | 🟢 | ✅ | - | Auto-start, auto-reload, graceful shutdown |
| 2. Execution | 🟢 | ✅ | ⬜ | Execution listing, control, trigger (backend complete with UI fields & i18n) |
| 3. Results/Activities | 🟢 | ⬜ | ⬜ | Deliverables and activity feed |
| 4. i18n | 🟢 | ✅ | ⬜ | Locale parameter support (backend executor i18n complete) |
| 5. Chat API | 🟡 | ⬜ | ⬜ | Multi-turn conversation (Deferred) |
| 6. SSE Streams | 🟡 | ⬜ | ⬜ | Real-time status updates (Deferred) |

Legend: ⬜ Not started | 🟡 In progress | ✅ Complete

### Phase 1 Detailed Status

| Component | Status | Notes |
|-----------|--------|-------|
| `types.Robot.Bio` | ✅ | Field added |
| `cache/load.go` | ✅ | `bio` in memberFields |
| `store/robot.go` | ✅ | Full CRUD with permission fields |
| `store/robot_test.go` | ✅ | Integration tests |
| `api/robot.go` | ✅ | Create/Update/Remove/GetResponse |
| `api/types.go` | ✅ | Request/Response types, AuthScope |
| `api/robot_test.go` | ✅ | API tests |
| `utils/convert.go` | ✅ | Type conversion utilities |
| `utils/convert_test.go` | ✅ | Unit tests |
| `openapi/agent/robot/robot.go` | ✅ | Route registration with Attach() |
| `openapi/agent/robot/types.go` | ✅ | HTTP request/response types |
| `openapi/agent/robot/list.go` | ✅ | List robots handler with permission filter |
| `openapi/agent/robot/detail.go` | ✅ | CRUD handlers with permission checks |
| `openapi/agent/robot/permission.go` | ✅ | Permission check functions (CanRead/CanWrite) |
| `openapi/agent/robot/utils.go` | ✅ | Helper functions |
| `openapi/agent/agent.go` | ✅ | Robot routes registered |
| `openapi/tests/agent/robot_test.go` | ✅ | Integration tests + Permission tests |

---

## Quick Reference

### Current Location

```
yao/openapi/agent/robot/           # This directory (sub-package under agent)
├── DESIGN.md       # Design document ✅
├── TODO.md         # This file ✅
├── robot.go        # Route registration (Attach function) ✅
├── types.go        # All request/response types ✅
├── list.go         # GET /v1/agent/robots ✅
├── detail.go       # GET/POST/PUT/DELETE /v1/agent/robots/:id ✅
├── permission.go   # Permission check functions (CanRead/CanWrite) ✅
├── utils.go        # Utilities ✅
├── execution.go    # Execution endpoints (Phase 2)
├── trigger.go      # Trigger/Intervene SSE (Phase 2)
├── results.go      # Results endpoints (Phase 3)
├── activities.go   # Activities endpoint (Phase 3)
├── stream.go       # Real-time streams (Phase 6 - Deferred)
└── filter.go       # Query filtering (optional)
```

### Parent Directory

```
yao/openapi/agent/
├── agent.go        # MODIFY: add robot.Attach() call
├── assistant.go    # Existing
├── filter.go       # Existing
├── models.go       # Existing
├── types.go        # Existing
│
└── robot/          # NEW sub-package (this directory)
    └── ...
```

### Route Registration (in agent/agent.go)

```go
import "github.com/yaoapp/yao/openapi/agent/robot"

func Attach(group *gin.RouterGroup, oauth types.OAuth) {
    group.Use(oauth.Guard)
    
    // Existing assistant routes
    group.GET("/assistants", ListAssistants)
    group.POST("/assistants", CreateAssistant)
    group.GET("/assistants/tags", ListAssistantTags)
    group.GET("/assistants/:id", GetAssistant)
    group.GET("/assistants/:id/info", GetAssistantInfo)
    group.PUT("/assistants/:id", UpdateAssistant)
    
    // Robot routes (NEW)
    robot.Attach(group.Group("/robots"), oauth)
}
```

### Dependencies

| Package | Usage |
|---------|-------|
| `yao/agent/robot/api` | Go API functions (Get, List, Trigger, etc.) |
| `yao/agent/robot/types` | Robot types (Robot, Execution, etc.) |
| `yao/openapi/oauth` | Authentication, Guard middleware |
| `yao/openapi/oauth/types` | OAuth types (AuthorizedInfo) |
| `yao/openapi/response` | Response helpers |

### Import Path

```go
package robot

import (
    "github.com/gin-gonic/gin"
    robotapi "github.com/yaoapp/yao/agent/robot/api"
    robottypes "github.com/yaoapp/yao/agent/robot/types"
    "github.com/yaoapp/yao/openapi/oauth/types"
)
```

---

## Notes

### Priority

| Priority | Phase | Required For | Risk |
|----------|-------|--------------|------|
| 1 | Phase 1 (CRUD) | Basic UI functionality | 🟢 Low |
| 2 | Phase 2 (Execution) | Active/History tabs, Assign Task | 🟢 Low |
| 3 | Phase 3 (Results) | Results tab | 🟢 Low |
| 4 | Phase 4 (i18n) | Multi-language support | 🟢 Low |
| 5 | Phase 5 (Chat) | Enhanced UX (deferred) | 🟡 Medium |
| 6 | Phase 6 (SSE) | Real-time updates (deferred) | 🟡 Medium |

### Frontend Fallbacks

| Feature | Full Implementation | Fallback |
|---------|---------------------|----------|
| Assign Task | Multi-turn chat → Confirm → Execute | Single-submit → Execute |
| Real-time Status | SSE push | Polling every 3-5s |

### Frontend Integration

**Execute immediately after each phase backend completion:**

1. **SDK Implementation** - `cui/packages/cui/openapi/agent/robot/`
2. **Type Definitions** - TypeScript request/response types
3. **Hook Implementation** - `cui/packages/cui/hooks/useRobots.ts`
4. **Page Integration** - Replace mock data, call real APIs
5. **E2E Verification** - Full flow testing

**File Locations:**
```
cui/packages/cui/
├── openapi/
│   └── agent/
│       └── robot/
│           ├── types.ts      # TypeScript types
│           ├── robots.ts     # AgentRobots SDK class
│           └── index.ts      # Exports
├── hooks/
│   └── useRobots.ts          # React hook for robot API calls
├── styles/
│   └── preset/
│       └── vars.less         # CSS variables (--color_mission_button_text)
└── pages/
    └── mission-control/
        ├── index.tsx         # Robot list (grid) page
        ├── index.less        # Styles with bubble animations
        └── components/
            ├── AgentModal/           # Robot detail modal
            ├── AddAgentModal/        # Create robot modal
            └── CreatureLoading/      # Branded loading component
                ├── index.tsx
                └── index.less
```

### Incremental Deployment

Each phase independently deliverable:

| Phase | Backend | Frontend | Verifiable Features |
|-------|---------|----------|---------------------|
| 1 | ✅ | ✅ | Robot CRUD basic management |
| 2 | ✅ | ⬜ | Execution list/control/trigger (backend with UI fields & i18n) |
| 3 | ⬜ | ⬜ | Results/Activities viewing |
| 4 | ✅ | ⬜ | Multi-language support (backend executor i18n) |
| 5 | ⬜ | ⬜ | Multi-turn chat UX (optional) |
| 6 | ⬜ | ⬜ | Real-time push (optional) |
