# Robot OpenAPI - Gap Analysis

> This document analyzes the gaps between existing backend implementation and frontend API requirements.
> Generated from reviewing: `yao/agent/robot/`, `yao/openapi/agent/`, `cui/packages/cui/pages/mission-control/`

---

## Summary

| Category | Risk | Status | Items to Implement |
|----------|------|--------|-------------------|
| Backend Types | 🟢 Low | 🟡 Partial | 1 field to add (`Bio`), 2 fields for Execution |
| Backend Cache | 🟢 Low | 🟡 Partial | Add `bio` to memberFields in `cache/load.go` |
| Backend API | 🟢 Low | 🟡 Partial | 7 functions missing (CRUD + Results + Activities) |
| OpenAPI Layer | 🟢 Low | ⬜ New | 19 endpoints, response type mapping |
| i18n | 🟢 Low | ⬜ New | Locale parameter support |
| **Chat API** | 🟡 Medium | ⬜ Deferred | Multi-turn conversation (frontend fallback: single-submit) |
| **SSE Infrastructure** | 🟡 Medium | ⬜ Deferred | Event bus + SSE handlers (frontend fallback: polling) |

### Key Field Mapping (Backend → Frontend)

| Frontend API | Backend DB (`__yao.member`) | Backend Go (`types.Robot`) |
|--------------|----------------------------|---------------------------|
| `name` | `member_id` | `MemberID` |
| `display_name` | `display_name` | `DisplayName` |
| `description` | `bio` | Need to add `Bio` field |
| `email` | `robot_email` | `RobotEmail` |

---

## 1. Backend Types Gaps (`yao/agent/robot/types/`)

### 1.1 Field Mapping (Backend → Frontend API)

The `__yao.member` model already has the necessary fields, but with different names:

| Frontend API Field | Backend DB Field | Status | Notes |
|-------------------|------------------|--------|-------|
| `member_id` | `member_id` | ✅ Exists | Global unique identifier |
| `name` | `member_id` | ✅ **Reuse** | Frontend expects a slug like `sales-analyst`, can use `member_id` |
| `display_name` | `display_name` | ✅ Exists | Localized display name |
| `description` | `bio` | ✅ Exists | `bio` field in `__yao.member` is the robot description |

**Backend Robot struct (`types/robot.go`):**
```go
type Robot struct {
    MemberID       string      `json:"member_id"`    // ✅ Exists
    TeamID         string      `json:"team_id"`      // ✅ Exists
    DisplayName    string      `json:"display_name"` // ✅ Exists
    SystemPrompt   string      `json:"system_prompt"`// ✅ Exists
    // ...
}
```

**Missing fields to add to Robot struct:**
```go
type Robot struct {
    // ... existing fields ...
    Bio            string      `json:"bio"`          // NEW: from __yao.member.bio (robot description)
}
```

**OpenAPI Response Mapping:**
```go
// In OpenAPI layer, map backend fields to frontend expected format
type RobotResponse struct {
    MemberID    string `json:"member_id"`
    Name        string `json:"name"`         // Use MemberID as unique slug
    DisplayName string `json:"display_name"`
    Description string `json:"description"`  // Map from Robot.Bio
    // ...
}
```

### 1.2 Cache/Load Update Needed

Update `cache/load.go` to fetch `bio` field:

```go
var memberFields = []interface{}{
    "id",
    "member_id",
    "team_id",
    "display_name",
    "bio",              // ADD THIS
    "system_prompt",
    "robot_status",
    "autonomous_mode",
    "robot_config",
    "robot_email",      // Already there
}
```

### 1.3 Missing Fields in `Execution` struct

| Field | Type | Location | Description |
|-------|------|----------|-------------|
| `Name` | `string` | `types/robot.go` | Derived from goals or human input, for UI display |
| `CurrentTaskName` | `string` | `types/robot.go` | What the agent is doing RIGHT NOW |

**Required (add to Execution struct):**
```go
type Execution struct {
    // ... existing fields ...
    Name            string `json:"name,omitempty"`              // NEW: execution name for UI
    CurrentTaskName string `json:"current_task_name,omitempty"` // NEW: current task description
}
```

> **Note:** These can be derived in the OpenAPI layer from existing fields:
> - `Name`: Derive from `Goals.Content` first line or `Input.Messages[0].Content`
> - `CurrentTaskName`: Derive from `Current.Task` executor info or progress

### 1.4 New Types Needed

#### Activity Type (for Activity API)

> **Note:** Activity can be derived from execution history without new storage.
> These types go in OpenAPI response layer, not core types.

```go
// openapi/agent/robot/types.go (API response types)

// ActivityType - activity type enum
type ActivityType string

const (
    ActivityCompleted ActivityType = "completed"
    ActivityFile      ActivityType = "file"
    ActivityError     ActivityType = "error"
    ActivityStarted   ActivityType = "started"
    ActivityPaused    ActivityType = "paused"
)

// ActivityResponse - activity item for UI
type ActivityResponse struct {
    ID          string       `json:"id"`
    Type        ActivityType `json:"type"`
    MemberID    string       `json:"member_id"`
    RobotName   string       `json:"robot_name"`   // Localized
    Title       string       `json:"title"`        // Localized
    Description string       `json:"description,omitempty"` // Localized
    FileID      string       `json:"file_id,omitempty"`
    Timestamp   string       `json:"timestamp"`    // ISO format
}
```

#### ResultFile Type (for Results API)

> **Note:** Results are derived from `execution.delivery.content.attachments`.
> No separate storage needed.

```go
// openapi/agent/robot/types.go (API response types)

// ResultFileResponse - deliverable file for Results Tab
type ResultFileResponse struct {
    ID            string `json:"id"`            // attachment index or file ID
    MemberID      string `json:"member_id"`
    ExecutionID   string `json:"execution_id"`
    Name          string `json:"name"`          // From attachment.Title
    Type          string `json:"type"`          // Derived from file extension
    Size          int64  `json:"size"`          // From file system
    CreatedAt     string `json:"created_at"`    // Execution end time
    TriggerType   string `json:"trigger_type,omitempty"`
    ExecutionName string `json:"execution_name,omitempty"` // Derived
}
```

---

## 2. Multi-turn Conversation Gap (Critical)

### 2.1 Frontend Expectation

The frontend `ChatDrawer` component expects **multi-turn conversation** before execution starts:

```
┌─────────────────────────────────────────────────────────────────┐
│  ASSIGN TASK DRAWER (ChatDrawer)                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  User: "Help me analyze competitor pricing"                      │
│                          ↓                                       │
│  Robot: "Got it. Which competitors? Any specific metrics?"       │
│                          ↓                                       │
│  User: "Focus on Company A and B, compare pricing tiers"         │
│                          ↓                                       │
│  Robot: "Understood. I'll analyze A and B pricing tiers.         │
│          Ready to start?"                                        │
│                          ↓                                       │
│  User clicks [Confirm] → Execution starts                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key Flow:**
1. User sends message → Backend returns assistant response
2. User can continue conversation (refine task)
3. User confirms → Execution actually starts

### 2.2 Current Backend Implementation

```go
// api/trigger.go - Current behavior
func Trigger(ctx *types.Context, memberID string, req *TriggerRequest) (*TriggerResult, error) {
    // Immediately submits to execution pool
    // No conversation state, no confirmation step
}
```

**Problem:** Backend triggers execution immediately on first message. No multi-turn conversation support.

### 2.3 Gap Analysis

| Feature | Frontend Expects | Backend Has |
|---------|------------------|-------------|
| Multi-turn chat | ✅ Yes | ❌ No |
| Conversation state | ✅ Yes | ❌ No |
| Confirm before execute | ✅ Yes | ❌ No |
| SSE for each message | ✅ Yes | ❌ No |

### 2.4 Required New API

**Option A: Chat API (Recommended)**

```
POST /v1/agent/robots/:id/chat
```

**Request:**
```json
{
  "conversation_id": "conv_001",  // Optional, for continuing conversation
  "messages": [
    { "role": "user", "content": "Help me analyze competitor pricing" }
  ],
  "attachments": []
}
```

**Response (SSE):**
```
event: message
data: {"role": "assistant", "content": "Got it. Which competitors?"}

event: state
data: {"conversation_id": "conv_001", "ready_to_execute": false}
```

**Then Trigger with conversation:**
```
POST /v1/agent/robots/:id/trigger
{
  "conversation_id": "conv_001",  // References chat history
  "confirm": true
}
```

**Option B: Extend Trigger API**

Add `confirm` parameter to trigger:
```json
{
  "messages": [...],
  "confirm": false  // false = chat mode, true = execute
}
```

### 2.5 Backend Implementation Needed

1. **Conversation Store** - Store chat history temporarily
   ```go
   // store/conversation.go (NEW)
   type ConversationStore interface {
       Create(memberID string, messages []Message) (conversationID string, error)
       Append(conversationID string, messages []Message) error
       Get(conversationID string) (*Conversation, error)
       Delete(conversationID string) error  // Auto-cleanup after execution
   }
   ```

2. **Chat Handler** - Process messages, return assistant response
   ```go
   // api/chat.go (NEW)
   func Chat(ctx *types.Context, memberID string, req *ChatRequest) (*ChatResponse, error) {
       // 1. Get or create conversation
       // 2. Call LLM for response (using robot's system prompt)
       // 3. Store updated conversation
       // 4. Return assistant message + conversation_id
   }
   ```

3. **Trigger Extension** - Support conversation_id
   ```go
   // api/trigger.go (MODIFY)
   type TriggerRequest struct {
       // ... existing fields ...
       ConversationID string `json:"conversation_id,omitempty"`  // NEW
   }
   ```

### 2.6 Same for Intervention

`GuideExecutionDrawer` also uses `ChatDrawer` and expects the same multi-turn behavior for intervention.

---

## 3. Backend Architecture: Store + API Layers

### 3.1 Architecture Decision

> **Principle:** Store layer handles database CRUD, API layer handles business logic.
> This enables reuse across Golang API, JSAPI, and Yao Process.

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Consumers                                    │
├──────────────────────────────────────────────────────────────────────┤
│  Golang API (robot/api)  │  JSAPI (JS Runtime)  │  Yao Process       │
└──────────────────────────────┴───────────────────────┴───────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      API Layer (robot/api/)                          │
│  Business logic, parameter validation, cache invalidation            │
│  - Thin wrappers that call store layer                               │
│  - Reusable across all consumers                                     │
└──────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      Store Layer (robot/store/)                      │
│  Pure database CRUD, no business logic                               │
│  - RobotStore: Robot member CRUD (NEW)                               │
│  - ExecutionStore: Execution records (EXISTS)                        │
└──────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      Model Layer (__yao.member, etc.)                │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.2 Store Layer: Missing Functions

**File: `store/robot.go` (NEW)** - Core CRUD implementation

| Function | Status | Description |
|----------|--------|-------------|
| `RobotStore.Save()` | ⬜ Missing | Create or update robot member |
| `RobotStore.Get()` | ⬜ Missing | Get robot by member_id |
| `RobotStore.List()` | ⬜ Missing | List robots with filters |
| `RobotStore.Delete()` | ⬜ Missing | Delete robot member |
| `RobotStore.UpdateConfig()` | ⬜ Missing | Update robot config only |

**File: `store/execution.go` (extend)**

| Function | Status | Description |
|----------|--------|-------------|
| `ExecutionStore.ListResults()` | ⬜ Missing | Query deliverables from executions |
| `ExecutionStore.GetResult()` | ⬜ Missing | Get single deliverable |
| `ExecutionStore.ListActivities()` | ⬜ Missing | Derive activities from history |

### 3.3 API Layer: Missing Functions

**File: `api/robot.go` (extend)** - Thin wrappers calling store

| Function | Status | Description |
|----------|--------|-------------|
| `Create()` | ⬜ Missing | Call `store.RobotStore.Save()` + cache refresh |
| `Update()` | ⬜ Missing | Call `store.RobotStore.UpdateConfig()` + cache refresh |
| `Remove()` | ⬜ Missing | Call `store.RobotStore.Delete()` + cache invalidate |

**File: `api/results.go` (NEW)** - Thin wrappers

| Function | Status | Description |
|----------|--------|-------------|
| `ListResults()` | ⬜ Missing | Call `store.ExecutionStore.ListResults()` |
| `GetResult()` | ⬜ Missing | Call `store.ExecutionStore.GetResult()` |

**File: `api/activities.go` (NEW)** - Thin wrappers

| Function | Status | Description |
|----------|--------|-------------|
| `ListActivities()` | ⬜ Missing | Call `store.ExecutionStore.ListActivities()` |

**File: `api/execution.go` (extend)**

| Function | Status | Description |
|----------|--------|-------------|
| `RetryExecution()` | ⬜ Missing | Re-trigger with same input |

### 3.4 Existing Functions (Already implemented)

**Store Layer (`store/`):**

| Function | File | Status |
|----------|------|--------|
| `ExecutionStore.Save()` | `execution.go` | ✅ Exists |
| `ExecutionStore.Get()` | `execution.go` | ✅ Exists |
| `ExecutionStore.List()` | `execution.go` | ✅ Exists |
| `ExecutionStore.Delete()` | `execution.go` | ✅ Exists |
| `ExecutionStore.UpdatePhase()` | `execution.go` | ✅ Exists |
| `ExecutionStore.UpdateStatus()` | `execution.go` | ✅ Exists |

**API Layer (`api/`):**

| Function | File | Status |
|----------|------|--------|
| `List()` | `robot.go` | ✅ Exists |
| `Get()` | `robot.go` | ✅ Exists |
| `GetStatus()` | `robot.go` | ✅ Exists |
| `Trigger()` | `trigger.go` | ✅ Exists |
| `Intervene()` | `trigger.go` | ✅ Exists |
| `GetExecutions()` | `execution.go` | ✅ Exists |
| `GetExecution()` | `execution.go` | ✅ Exists |
| `PauseExecution()` | `execution.go` | ✅ Exists |
| `ResumeExecution()` | `execution.go` | ✅ Exists |
| `StopExecution()` | `execution.go` | ✅ Exists |

### 3.5 Code Examples

**Store Layer (`store/robot.go`):**
```go
// RobotStore - persistent storage for robot members
type RobotStore struct {
    modelID string
}

func NewRobotStore() *RobotStore {
    return &RobotStore{modelID: "__yao.member"}
}

// Save creates or updates a robot member record
func (s *RobotStore) Save(ctx context.Context, record *RobotRecord) error

// Get retrieves a robot by member_id
func (s *RobotStore) Get(ctx context.Context, memberID string) (*RobotRecord, error)

// List retrieves robots with filters
func (s *RobotStore) List(ctx context.Context, opts *ListOptions) ([]*RobotRecord, error)

// Delete removes a robot member
func (s *RobotStore) Delete(ctx context.Context, memberID string) error
```

**API Layer (`api/robot.go`):**
```go
// Create creates a new robot member (thin wrapper)
func Create(ctx *types.Context, teamID string, req *CreateRobotRequest) (*types.Robot, error) {
    // 1. Validate request
    // 2. Call store.RobotStore.Save()
    // 3. Refresh cache
    // 4. Return robot
}

// Update updates robot config (thin wrapper)
func Update(ctx *types.Context, memberID string, req *UpdateRobotRequest) (*types.Robot, error) {
    // 1. Validate request
    // 2. Call store.RobotStore.UpdateConfig()
    // 3. Refresh cache
    // 4. Return updated robot
}

// Remove deletes a robot member (thin wrapper)
func Remove(ctx *types.Context, memberID string) error {
    // 1. Check permissions
    // 2. Call store.RobotStore.Delete()
    // 3. Invalidate cache
}
```

---

## 4. OpenAPI Layer (`yao/openapi/agent/robot/`)

### 4.1 Files to Create

```
yao/openapi/agent/robot/
├── DESIGN.md       # ✅ Exists
├── TODO.md         # ✅ Exists
├── GAPS.md         # ✅ This file
│
├── robot.go        # Route registration
├── types.go        # Request/Response types
├── list.go         # GET /v1/agent/robots
├── detail.go       # GET/POST/PUT/DELETE /v1/agent/robots/:id
├── execution.go    # Execution list/detail/control
├── trigger.go      # Trigger/Intervene (SSE)
├── results.go      # Results endpoints
├── activities.go   # Activities endpoint
├── stream.go       # Real-time SSE streams
├── filter.go       # Query param parsing
└── utils.go        # Locale, time formatting
```

### 4.2 Endpoints to Implement

#### Robot CRUD (5 endpoints)

| Endpoint | Handler | Backend API |
|----------|---------|-------------|
| `GET /robots` | `ListRobots` | `api.List()` ✅ |
| `GET /robots/:id` | `GetRobot` | `api.Get()` + `api.GetStatus()` ✅ |
| `POST /robots` | `CreateRobot` | `api.Create()` ⬜ |
| `PUT /robots/:id` | `UpdateRobot` | `api.Update()` ⬜ |
| `DELETE /robots/:id` | `DeleteRobot` | `api.Remove()` ⬜ |

#### Chat & Execution Management (9 endpoints)

| Endpoint | Handler | Backend API |
|----------|---------|-------------|
| `POST /robots/:id/chat` | `ChatWithRobot` | `api.Chat()` ⬜ **NEW - Multi-turn conversation** |
| `GET /robots/:id/executions` | `ListExecutions` | `api.GetExecutions()` ✅ |
| `GET /robots/:id/executions/:exec_id` | `GetExecution` | `api.GetExecution()` ✅ |
| `POST /robots/:id/trigger` | `TriggerRobot` | `api.Trigger()` ✅ (needs conversation_id support) |
| `POST /robots/:id/intervene` | `InterveneRobot` | `api.Intervene()` ✅ (needs conversation_id support) |
| `POST /robots/:id/executions/:exec_id/pause` | `PauseExecution` | `api.PauseExecution()` ✅ |
| `POST /robots/:id/executions/:exec_id/resume` | `ResumeExecution` | `api.ResumeExecution()` ✅ |
| `POST /robots/:id/executions/:exec_id/cancel` | `CancelExecution` | `api.StopExecution()` ✅ |
| `POST /robots/:id/executions/:exec_id/retry` | `RetryExecution` | `api.RetryExecution()` ⬜ |

#### Results (2 endpoints)

| Endpoint | Handler | Backend API |
|----------|---------|-------------|
| `GET /robots/:id/results` | `ListResults` | `api.ListResults()` ⬜ |
| `GET /robots/:id/results/:result_id` | `GetResult` | `api.GetResult()` ⬜ |

#### Activities (1 endpoint)

| Endpoint | Handler | Backend API |
|----------|---------|-------------|
| `GET /robots/activities` | `ListActivities` | `api.ListActivities()` ⬜ |

#### SSE Streams (3 endpoints)

| Endpoint | Handler | Backend Event Bus |
|----------|---------|-------------------|
| `GET /robots/stream` | `StreamRobots` | ⬜ New event bus needed |
| `GET /robots/:id/executions/:exec_id/stream` | `StreamExecution` | ⬜ New event bus needed |
| `POST /robots/:id/trigger` (SSE) | `TriggerRobot` | Wrap existing `api.Trigger()` |
| `POST /robots/:id/intervene` (SSE) | `InterveneRobot` | Wrap existing `api.Intervene()` |

---

## 5. SSE Infrastructure Gaps

### 5.1 Event Bus Needed

The backend needs an event bus to publish real-time events. Currently, the robot module doesn't have one.

**Required Components:**

```go
// robot/events/bus.go (NEW PACKAGE)

type EventBus struct {
    subscribers map[string][]chan Event
    mu          sync.RWMutex
}

type Event struct {
    Type    string      `json:"type"`    // robot_status, execution_start, etc.
    Payload interface{} `json:"payload"`
}

func (bus *EventBus) Publish(event Event)
func (bus *EventBus) Subscribe(topic string) <-chan Event
func (bus *EventBus) Unsubscribe(topic string, ch <-chan Event)
```

### 5.2 Event Publishers Needed

| Event | Source | When |
|-------|--------|------|
| `robot_status` | Manager | Robot status changes |
| `execution_start` | Executor | Execution begins |
| `execution_complete` | Executor | Execution ends |
| `phase` | Executor | Phase changes |
| `task_start` | Runner | Task begins |
| `task_complete` | Runner | Task ends |
| `activity` | Multiple | Any activity event |

### 5.3 Integration Points

**In `manager/manager.go`:**
```go
// Publish when robot status changes
eventBus.Publish(Event{Type: "robot_status", Payload: ...})
```

**In `executor/standard/executor.go`:**
```go
// Publish when execution starts/ends
eventBus.Publish(Event{Type: "execution_start", Payload: ...})
```

---

## 6. i18n Support Gaps

### 6.1 Current State

- No locale parameter in backend API
- No localization infrastructure

### 6.2 Required Changes

**Add locale to context:**
```go
// types/context.go
type Context struct {
    context.Context
    Auth     *types.AuthorizedInfo
    MemberID string
    Locale   string // NEW: "zh-CN" | "en-US"
}
```

**Add locale helper:**
```go
// utils/locale.go (NEW)
func GetLocale(r *http.Request) string
func Localize(key, locale string) string
```

**Localized fields:**
- `RobotState.display_name`
- `RobotState.description`
- `Execution.name`
- `Execution.current_task_name`
- `ResultFile.name`
- `ResultFile.execution_name`
- `Activity.robot_name`
- `Activity.title`
- `Activity.description`

---

## 7. Data Source Gaps

### 7.1 Results Data

Results are derived from execution delivery data. Need to:

1. **Query from `store/execution.go`** - executions with delivery attachments
2. **Extract attachment metadata** - file ID, name, type, size

**Implementation:**
```go
// store/results.go (NEW)
func (s *ExecutionStore) ListResults(ctx context.Context, memberID string, opts *ResultsQuery) ([]*ResultFile, int, error) {
    // Query executions with delivery.content.attachments
    // Extract and format as ResultFile
}
```

### 7.2 Activities Data

Activities can be derived from:
1. **Job system logs** - existing `job.ListLogs()`
2. **Execution state changes** - from `store/execution.go`

**Implementation Options:**

**Option A: Derive from execution history**
```go
func ListActivities(ctx context.Context, query *ActivityQuery) ([]*Activity, error) {
    // Query recent executions
    // Map to Activity based on status changes
}
```

**Option B: Separate activity log (recommended for real-time)**
```go
// New table: __yao.robot_activity
type ActivityRecord struct {
    ID          int64
    Type        ActivityType
    MemberID    string
    ExecutionID string
    Data        JSON
    Timestamp   time.Time
}
```

---

## 8. Implementation Priority

> **Strategy:** Low-risk phases first. Medium-risk features (Chat API, SSE) can be deferred.
> Frontend can use polling and single-submit mode as fallback.

---

### 🟢 Phase 1: Core CRUD [Low Risk]

1. ⬜ Add `Bio` field to `Robot` struct (`types/robot.go`)
2. ⬜ Add `bio` to `memberFields` in `cache/load.go`
3. ⬜ Implement `api.Create()`, `api.Update()`, `api.Remove()`
4. ⬜ Create OpenAPI handlers: list, detail, create, update, delete
5. ⬜ Add response type mapping (`name` ← `member_id`, `description` ← `bio`)

### 🟢 Phase 2: Execution Management [Low Risk]

1. ⬜ Add derived fields in OpenAPI layer (`name`, `current_task_name`)
2. ⬜ Implement `api.RetryExecution()`
3. ⬜ Create OpenAPI handlers: execution list, detail, control
4. ⬜ Wrap trigger/intervene (single-submit mode, no chat)

### 🟢 Phase 3: Results & Activities [Low Risk]

1. ⬜ Create `ActivityResponse` and `ResultFileResponse` types in OpenAPI layer
2. ⬜ Implement `api.ListResults()`, `api.GetResult()` (derive from executions)
3. ⬜ Implement `api.ListActivities()` (derive from execution history)
4. ⬜ Create OpenAPI handlers

### 🟢 Phase 4: i18n [Low Risk]

1. ⬜ Add `Locale` to context
2. ⬜ Add locale helper functions
3. ⬜ Implement localized response fields

---

### 🟡 Phase 5: Multi-turn Chat API [Medium Risk - Deferred]

> **Fallback:** Frontend uses single-submit mode (user input → immediate execution)

1. ⬜ Create `store/conversation.go` - temporary conversation storage
2. ⬜ Create `api/chat.go` - chat handler with LLM call
3. ⬜ Extend `api/trigger.go` - support `conversation_id`
4. ⬜ Create OpenAPI endpoint: `POST /robots/:id/chat` (SSE)
5. ⬜ Update `POST /robots/:id/trigger` to accept conversation reference
6. ⬜ Same for `POST /robots/:id/intervene`

### 🟡 Phase 6: Real-time SSE [Medium Risk - Deferred]

> **Fallback:** Frontend uses polling (GET /executions every 3-5s)

1. ⬜ Create event bus package
2. ⬜ Integrate event publishing in manager/executor
3. ⬜ Implement SSE stream handlers
4. ⬜ End-to-end testing

---

## 9. Testing Strategy

### Unit Tests

- `types/activity_test.go` - new types
- `types/result_test.go` - new types
- `api/robot_test.go` - CRUD functions
- `api/results_test.go` - results API
- `api/activities_test.go` - activities API

### Integration Tests

- `openapi/agent/robot/*_test.go` - HTTP endpoint tests
- `openapi/agent/robot/sse_test.go` - SSE stream tests

### E2E Tests

- Full flow: create robot → trigger → stream events → get results

---

## 10. Files to Modify Summary

### Backend (`yao/agent/robot/`)

#### Store Layer (Core CRUD - implement first)

| File | Action | Changes |
|------|--------|---------|
| `store/robot.go` | **Create** | `RobotStore` - Robot member CRUD (Save, Get, List, Delete, UpdateConfig) |
| `store/execution.go` | Modify | Add `ListResults()`, `GetResult()`, `ListActivities()` |
| `store/conversation.go` | Create | Temporary conversation storage (Phase 5 - Deferred) |

#### Types Layer

| File | Action | Changes |
|------|--------|---------|
| `types/robot.go` | Modify | Add `Bio` field |
| `types/conversation.go` | Create | `Conversation`, `ChatRequest`, `ChatResponse` types (Phase 5) |
| `types/context.go` | Modify | Add `Locale` field |

#### Cache Layer

| File | Action | Changes |
|------|--------|---------|
| `cache/load.go` | Modify | Add `bio` to `memberFields` slice |

#### API Layer (Thin wrappers calling store)

| File | Action | Changes |
|------|--------|---------|
| `api/robot.go` | Modify | Add `Create()`, `Update()`, `Remove()` - call store.RobotStore |
| `api/results.go` | Create | `ListResults()`, `GetResult()` - call store.ExecutionStore |
| `api/activities.go` | Create | `ListActivities()` - call store.ExecutionStore |
| `api/execution.go` | Modify | Add `RetryExecution()` |
| `api/chat.go` | Create | `Chat()` - multi-turn conversation (Phase 5 - Deferred) |
| `api/trigger.go` | Modify | Add `ConversationID` support (Phase 5 - Deferred) |

#### Events Layer (Phase 6 - Deferred)

| File | Action | Changes |
|------|--------|---------|
| `events/bus.go` | Create | Event bus for SSE |

### OpenAPI (`yao/openapi/agent/robot/`)

| File | Action | Description |
|------|--------|-------------|
| `robot.go` | Create | Route registration |
| `types.go` | Create | Request/Response types |
| `list.go` | Create | List robots handler |
| `detail.go` | Create | Robot CRUD handlers |
| `chat.go` | Create | Multi-turn chat SSE handler |
| `execution.go` | Create | Execution handlers |
| `trigger.go` | Create | Trigger/Intervene SSE (with conversation support) |
| `results.go` | Create | Results handlers |
| `activities.go` | Create | Activities handler |
| `stream.go` | Create | SSE streams |
| `filter.go` | Create | Query parsing |
| `utils.go` | Create | Utilities |

### Parent (`yao/openapi/agent/`)

| File | Action | Changes |
|------|--------|---------|
| `agent.go` | Modify | Add `robot.Attach(group.Group("/robots"), oauth)` |

---

## 11. References

- Frontend API Requirements: `cui/packages/cui/pages/mission-control/API.md`
- Backend Robot Types: `yao/agent/robot/types/`
- Backend Robot API: `yao/agent/robot/api/`
- OpenAPI Design: `yao/openapi/agent/robot/DESIGN.md`
- OpenAPI TODO: `yao/openapi/agent/robot/TODO.md`
