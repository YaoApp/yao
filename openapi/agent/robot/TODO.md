# Robot OpenAPI - Implementation TODO

> Based on: `openapi/agent/robot/DESIGN.md`, `openapi/agent/robot/GAPS.md`
> Depends on: `yao/agent/robot/api/` (Go API layer)
> Base Path: `/v1/agent/robots`

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
  Phase 5: Multi-turn Chat API
  Phase 6: Real-time SSE Streams
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

## 🟢 Phase 2: Execution Management ⬜ [Low Risk]

**Goal:** Execution listing, details, control, and trigger/intervene (single-submit mode)
**Risk:** 🟢 Low - Wraps existing API functions

### 2.1 List Executions ⬜

- [ ] `execution.go` - GET /v1/robots/:id/executions
- [ ] Parse query params: `status`, `trigger_type`, `keyword`, `page`, `pagesize`
- [ ] Call `robot/api.GetExecutions()`
- [ ] Add derived fields: `name`, `current_task_name`
- [ ] Format response
- [ ] Test: `tests/robot/execution_list_test.go`

### 2.2 Get Execution ⬜

- [ ] GET /v1/robots/:id/executions/:exec_id
- [ ] Call `robot/api.GetExecution()`
- [ ] Full task details with localization
- [ ] Test: `tests/robot/execution_get_test.go`

### 2.3 Execution Control ⬜

- [ ] POST /v1/robots/:id/executions/:exec_id/pause
  - [ ] Call `robot/api.Pause()`
- [ ] POST /v1/robots/:id/executions/:exec_id/resume
  - [ ] Call `robot/api.Resume()`
- [ ] POST /v1/robots/:id/executions/:exec_id/cancel
  - [ ] Call `robot/api.Stop()`
- [ ] POST /v1/robots/:id/executions/:exec_id/retry
  - [ ] Re-trigger with same input
- [ ] Test: `tests/robot/execution_control_test.go`

### 2.4 Execution Types ⬜

- [ ] Add to `types.go`:
  - [ ] `ExecutionResponse` struct
  - [ ] `TaskResponse` struct
  - [ ] `CurrentStateResponse` struct
  - [ ] `GoalsResponse` struct
  - [ ] `DeliveryResultResponse` struct

### 2.5 Trigger & Intervene (Single-Submit Mode) ⬜

> **Note:** This is single-submit mode. Multi-turn chat is deferred to Phase 5.

- [ ] `trigger.go` - POST /v1/robots/:id/trigger
- [ ] Parse `TriggerRequest` (messages, attachments)
- [ ] Call `robot/api.Trigger()` 
- [ ] Return execution ID and status
- [ ] Optional: Return SSE stream for progress
- [ ] Test: `tests/robot/trigger_test.go`

- [ ] POST /v1/robots/:id/intervene
- [ ] Parse `InterveneRequest`
- [ ] Call `robot/api.Intervene()`
- [ ] Return result
- [ ] Test: `tests/robot/intervene_test.go`

### 2.6 Trigger Types ⬜

- [ ] Add to `types.go`:
  - [ ] `TriggerRequest` struct
  - [ ] `TriggerResponse` struct
  - [ ] `InterveneRequest` struct
  - [ ] `InterveneResponse` struct
  - [ ] `Message` struct
  - [ ] `Attachment` struct

### 2.7 Frontend Integration ⬜

> Integrate immediately after backend completion

- [ ] SDK: Add execution methods to `robot.ts`
  - [ ] `listExecutions(robotId, params)`
  - [ ] `getExecution(robotId, execId)`
  - [ ] `pauseExecution()`, `resumeExecution()`, `cancelExecution()`
  - [ ] `triggerRobot(robotId, data)`
  - [ ] `intervene(robotId, data)`
- [ ] Page: Execution list/detail page integration
- [ ] Page: Assign Task (trigger execution) integration
- [ ] Verify: E2E testing

---

## 🟢 Phase 3: Results & Activities ⬜ [Low Risk]

**Goal:** Deliverables listing and activity feed
**Risk:** 🟢 Low - Read-only queries, derived from existing data

### 3.1 Backend Prerequisites ⬜

#### Store Layer (Core implementation)
- [ ] Add `ExecutionStore.ListResults()` - query deliverables from execution delivery data
- [ ] Add `ExecutionStore.GetResult()` - get single deliverable detail
- [ ] Add `ExecutionStore.ListActivities()` - derive activities from execution history

#### API Layer (Thin wrappers)
- [ ] Create `api/results.go` with `ListResults()`, `GetResult()` - call store
- [ ] Create `api/activities.go` with `ListActivities()` - call store

### 3.2 Results Endpoints ⬜

- [ ] `results.go` - results handlers
- [ ] GET /v1/robots/:id/results
  - [ ] Parse filters: `trigger_type`, `keyword`, `page`, `pagesize`
  - [ ] Call `robot/api.ListResults()`
  - [ ] Format response
- [ ] GET /v1/robots/:id/results/:result_id
  - [ ] Call `robot/api.GetResult()`
  - [ ] Return full delivery content
- [ ] Test: `tests/robot/results_test.go`

### 3.3 Results Types ⬜

- [ ] Add to `types.go`:
  - [ ] `ResultResponse` struct
  - [ ] `ResultDetailResponse` struct
  - [ ] `DeliveryContentResponse` struct
  - [ ] `DeliveryAttachmentResponse` struct

### 3.4 Activities Endpoints ⬜

- [ ] `activities.go` - activities handlers
- [ ] GET /v1/robots/activities
  - [ ] Parse: `limit`, `since`
  - [ ] Call `robot/api.ListActivities()`
  - [ ] Format response
- [ ] Test: `tests/robot/activities_test.go`

### 3.5 Activity Types ⬜

- [ ] Add to `types.go`:
  - [ ] `ActivityResponse` struct
  - [ ] `ActivityType` constants

### 3.6 Frontend Integration ⬜

> Integrate immediately after backend completion

- [ ] SDK: Add results/activities methods to `robot.ts`
  - [ ] `listResults(robotId, params)`
  - [ ] `getResult(robotId, resultId)`
  - [ ] `listActivities(params)`
- [ ] Page: Results Tab integration
- [ ] Page: Activity Feed integration
- [ ] Verify: E2E testing

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

## 🟡 Phase 5: Multi-turn Chat API ⬜ [Medium Risk - Deferred]

> **Frontend Fallback:** Single-submit mode (user input → immediate execution)
> **Risk:** 🟡 Medium - New stateful component

**Goal:** Multi-turn conversation before execution

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

---

## 🟡 Phase 6: Real-time SSE Streams ⬜ [Medium Risk - Deferred]

> **Frontend Fallback:** Polling (GET /executions every 3-5 seconds)
> **Risk:** 🟡 Medium - Requires modification of executor/manager

**Goal:** SSE streams for real-time status updates

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
| 2. Execution | 🟢 | ⬜ | ⬜ | Execution listing, control, trigger |
| 3. Results/Activities | 🟢 | ⬜ | ⬜ | Deliverables and activity feed |
| 4. i18n | 🟢 | ⬜ | ⬜ | Locale parameter support |
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
| 2 | ⬜ | ⬜ | Execution list/control/trigger |
| 3 | ⬜ | ⬜ | Results/Activities viewing |
| 4 | ⬜ | ⬜ | Multi-language support |
| 5 | ⬜ | ⬜ | Multi-turn chat UX (optional) |
| 6 | ⬜ | ⬜ | Real-time push (optional) |
