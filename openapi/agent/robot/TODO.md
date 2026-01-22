# Robot OpenAPI - Implementation TODO

> Based on: `openapi/agent/robot/DESIGN.md`, `openapi/agent/robot/GAPS.md`
> Depends on: `yao/agent/robot/api/` (Go API layer)
> Base Path: `/v1/agent/robots`

---

## Implementation Strategy

> **Low-risk phases first. Medium-risk features can be deferred.**
> Frontend has fallback mechanisms (polling, single-submit mode).

```
🟢 Low Risk (Do First):
  Phase 1: Core CRUD (MVP)
    └─ List, Get, Create, Update, Delete robots
  
  Phase 2: Execution Management
    └─ List, Get, Control executions, Trigger/Intervene (single-submit)
  
  Phase 3: Results & Activities
    └─ List deliverables, Activity feed
  
  Phase 4: i18n
    └─ Locale parameter support

🟡 Medium Risk (Deferred):
  Phase 5: Multi-turn Chat API
    └─ Conversation before execution (Frontend fallback: single-submit)
  
  Phase 6: Real-time SSE Streams
    └─ Robot status stream, Execution progress (Frontend fallback: polling)
```

---

## 🟢 Phase 1: Core CRUD ⬜ [Low Risk]

**Goal:** Basic robot management endpoints
**Risk:** 🟢 Low - All new code, no changes to existing logic

### 1.1 Backend Prerequisites ⬜

- [ ] Add `Bio` field to `types.Robot` struct in `yao/agent/robot/types/robot.go`
- [ ] Add `bio` to `memberFields` in `yao/agent/robot/cache/load.go`
- [ ] Implement `api.Create()` in `yao/agent/robot/api/robot.go`
- [ ] Implement `api.Update()` in `yao/agent/robot/api/robot.go`
- [ ] Implement `api.Remove()` in `yao/agent/robot/api/robot.go`

### 1.2 Setup ⬜

- [ ] Create `openapi/agent/robot/` directory (sub-package under agent)
- [ ] Create `robot.go` - route registration with `Attach()` function
- [ ] Register routes in `openapi/agent/agent.go` via `robot.Attach(group.Group("/robots"), oauth)`
- [ ] Add OAuth guard middleware

### 1.3 Types ⬜

- [ ] `types.go` - request/response types
  - [ ] `RobotResponse` struct (with field mapping: `name` ← `member_id`, `description` ← `bio`)
  - [ ] `ConfigResponse` struct (and sub-types)
  - [ ] `ListRobotsResponse` struct
  - [ ] `CreateRobotRequest` struct
  - [ ] `UpdateRobotRequest` struct
  - [ ] `NewRobotResponse()` - conversion function
  - [ ] Error response types

### 1.4 List Robots ⬜

- [ ] `list.go` - GET /v1/robots
- [ ] Parse query params: `locale`, `status`, `keywords`, `page`, `pagesize`
- [ ] Call `robot/api.List()`
- [ ] Format response with localization
- [ ] Test: `tests/robot/list_test.go`

### 1.5 Get Robot ⬜

- [ ] `detail.go` - GET /v1/robots/:id
- [ ] Parse path param and `locale` query
- [ ] Call `robot/api.Get()` and `robot/api.Status()`
- [ ] Format response with full config
- [ ] Team access check
- [ ] Test: `tests/robot/get_test.go`

### 1.6 Create Robot ⬜

- [ ] POST /v1/robots handler
- [ ] Parse `CreateRobotRequest`
- [ ] Validate required fields
- [ ] Call `robot/api.Create()`
- [ ] Return created robot
- [ ] Test: `tests/robot/create_test.go`

### 1.7 Update Robot ⬜

- [ ] PUT /v1/robots/:id handler
- [ ] Parse `UpdateRobotRequest`
- [ ] Ownership/permission check
- [ ] Call `robot/api.Update()`
- [ ] Return updated robot
- [ ] Test: `tests/robot/update_test.go`

### 1.8 Delete Robot ⬜

- [ ] DELETE /v1/robots/:id handler
- [ ] Ownership/permission check
- [ ] Call `robot/api.Remove()`
- [ ] Return success response
- [ ] Test: `tests/robot/delete_test.go`

### 1.9 Utilities ⬜

- [ ] `utils.go` - helper functions
  - [ ] `getLocale(r *http.Request)` - extract locale
  - [ ] `formatTime(t *time.Time)` - format to ISO string
  - [ ] `localizeString(value, locale)` - localization helper
- [ ] `filter.go` - query filtering
  - [ ] Parse query params to `ListQuery`
  - [ ] Parse query params to `ExecutionQuery`

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

---

## 🟢 Phase 3: Results & Activities ⬜ [Low Risk]

**Goal:** Deliverables listing and activity feed
**Risk:** 🟢 Low - Read-only queries, derived from existing data

### 3.1 Backend Prerequisites ⬜

Need to add in `robot/api/`:

- [ ] `ListResults(memberID, query)` function
- [ ] `GetResult(resultID)` function
- [ ] `ListActivities(query)` function

Need to add in `robot/store/`:

- [ ] Results store (query from execution delivery data)
- [ ] Activities store (or derive from job logs)

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

### robot/types/ Extensions

| Type/Field | Phase | Risk | Description |
|------------|-------|------|-------------|
| `Robot.Bio` | 1 | 🟢 Low | Add field, maps to `__yao.member.bio` |
| Execution name derivation | 2 | 🟢 Low | Derive in OpenAPI layer from goals or input |

> **Note:** `Robot.Name` is NOT needed. Frontend `name` maps to existing `Robot.MemberID`.

### robot/cache/ Extensions

| File | Phase | Risk | Description |
|------|-------|------|-------------|
| `load.go` | 1 | 🟢 Low | Add `bio` to `memberFields` slice |

### robot/api/ Extensions

| Function | Phase | Risk | Description |
|----------|-------|------|-------------|
| `Create()` | 1 | 🟢 Low | Create robot member via model |
| `Update()` | 1 | 🟢 Low | Update robot config via model |
| `Remove()` | 1 | 🟢 Low | Delete robot member via model |
| `ListResults()` | 3 | 🟢 Low | Query from execution delivery data |
| `GetResult()` | 3 | 🟢 Low | Get deliverable detail |
| `ListActivities()` | 3 | 🟢 Low | Derive from execution history |
| `RetryExecution()` | 2 | 🟢 Low | Re-trigger with same input |
| `Chat()` | 5 | 🟡 Medium | Multi-turn conversation handler |

### robot/store/ Extensions

| Store | Phase | Risk | Description |
|-------|-------|------|-------------|
| Results query | 3 | 🟢 Low | Query from execution delivery data |
| Activities query | 3 | 🟢 Low | Derive from execution history |
| Conversation store | 5 | 🟡 Medium | Temporary chat history (redis/memory) |

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

| Phase | Risk | Status | Description |
|-------|------|--------|-------------|
| 1. Core CRUD | 🟢 | ⬜ | Basic robot management |
| 2. Execution | 🟢 | ⬜ | Execution listing, control, trigger/intervene |
| 3. Results/Activities | 🟢 | ⬜ | Deliverables and activity feed |
| 4. i18n | 🟢 | ⬜ | Locale parameter support |
| 5. Chat API | 🟡 | ⬜ | Multi-turn conversation (Deferred) |
| 6. SSE Streams | 🟡 | ⬜ | Real-time status updates (Deferred) |

Legend: ⬜ Not started | 🟡 In progress | ✅ Complete | 🟢 Low Risk | 🟡 Medium Risk

---

## Quick Reference

### Current Location

```
yao/openapi/agent/robot/           # This directory (sub-package under agent)
├── DESIGN.md       # Design document ✅
├── TODO.md         # This file ✅
├── robot.go        # Route registration (Attach function)
├── types.go        # All request/response types
├── list.go         # GET /v1/agent/robots
├── detail.go       # GET/POST/PUT/DELETE /v1/agent/robots/:id
├── execution.go    # Execution endpoints
├── trigger.go      # Trigger/Intervene SSE
├── results.go      # Results endpoints
├── activities.go   # Activities endpoint
├── stream.go       # Real-time streams
├── filter.go       # Query filtering
└── utils.go        # Utilities
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

After each phase:
1. Test endpoints manually
2. Update frontend `openapi/robot.ts` to use real API
3. Remove mock data usage
4. Test end-to-end flow

### Incremental Deployment

Each phase can be deployed independently:
- Phase 1: Basic management works
- Phase 2: Execution history + trigger works
- Phase 3: Results listing works
- Phase 4: Multi-language works
- Phase 5: Enhanced chat UX (optional)
- Phase 6: Real-time updates (optional)
