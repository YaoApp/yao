# Robot Agent - Implementation TODO

> Based on DESIGN.md and TECHNICAL.md
> Test environment: `source yao/env.local.sh`
> Test assistants: `yao-dev-app/assistants/robot/`

---

## Workflow: Human-AI Collaboration

**Important:** Follow this workflow strictly for each sub-task.

```
┌─────────────────────────────────────────────────────────────────┐
│                     Implementation Workflow                      │
├─────────────────────────────────────────────────────────────────┤
│  1. AI: Implement code for current sub-task                     │
│  2. AI: Present code for review (DO NOT write tests yet)        │
│  3. Human: Review code, provide feedback                        │
│  4. AI: Iterate based on feedback                               │
│  5. Human: Confirm "LGTM" or "Approved"                         │
│  6. AI: Write tests for the approved code                       │
│  7. Human: Review tests                                         │
│  8. AI: Run tests, fix if needed                                │
│  9. Human: Confirm sub-task complete, move to next              │
└─────────────────────────────────────────────────────────────────┘
```

**Rules:**

| Rule                     | Description                                |
| ------------------------ | ------------------------------------------ |
| One sub-task at a time   | Focus only on current sub-task             |
| No tests before approval | Wait for human "LGTM" before writing tests |
| No jumping ahead         | Do not implement future phases             |
| Ask if unclear           | When in doubt, ask before proceeding       |

---

## Core Principle

- Phase 1-2: Types + Skeleton (code compiles)
- Phase 3: Complete scheduling system (Cache + Pool + Trigger + Dedup + Job), executor is stub
- Phase 4-9: Implement executor phases one by one (P0 → P5)
- Phase 10: API completion, end-to-end tests
- Monitoring: Provided by Job system, no separate implementation

---

## Phase 1: Types & Interfaces ✅

**Goal:** Define all types, enums, interfaces. No logic, no external deps.

**Status:** Complete - 88.4% test coverage, all tests passing

### 1.1 Enums (`types/enums.go`)

- [x] `Phase` - execution phases (inspiration, goals, tasks, run, delivery, learning)
- [x] `ClockMode` - clock trigger modes (times, interval, daemon)
- [x] `TriggerType` - trigger sources (clock, human, event)
- [x] `ExecStatus` - execution status (pending, running, completed, failed, cancelled)
- [x] `RobotStatus` - robot status (idle, working, paused, error, maintenance)
- [x] `InterventionAction` - human actions (task.add, goal.adjust, etc.)
- [x] `Priority` - priority levels (high, normal, low)
- [x] `DeliveryType` - delivery types (email, file, webhook, notify)
- [x] `DedupResult` - dedup results (skip, merge, proceed)
- [x] `EventSource` - event sources (webhook, database)
- [x] `LearningType` - learning types (execution, feedback, insight)
- [x] `TaskSource` - task sources (auto, human, event)
- [x] `ExecutorType` - executor types (assistant, mcp, process)
- [x] `TaskStatus` - task status (pending, running, completed, failed, skipped, cancelled)
- [x] `InsertPosition` - insert positions (first, last, next, at)

### 1.2 Context (`types/context.go`)

- [x] `Context` struct - robot execution context
- [x] `NewContext()` - constructor
- [x] `UserID()`, `TeamID()` - helper methods

### 1.3 Config Types (`types/config.go`)

- [x] `Config` - main config struct
- [x] `Triggers`, `TriggerSwitch` - trigger enable/disable
- [x] `Clock` - clock config with validation
- [x] `Identity` - role, duties, rules
- [x] `Quota` - concurrency limits with defaults
- [x] `KB`, `DB` - knowledge base and database config
- [x] `Learn` - learning config
- [x] `Resources`, `MCPConfig` - available agents and tools
- [x] `Delivery` - output delivery config
- [x] `Event` - event trigger config

### 1.4 Core Types (`types/robot.go`)

- [x] `Robot` struct - runtime robot representation
- [x] `Robot` methods - `CanRun()`, `RunningCount()`, `AddExecution()`, `RemoveExecution()`, `GetExecution()`, `GetExecutions()`
- [x] `Execution` struct - single execution instance
- [x] `TriggerInput` - stored trigger input
- [x] `CurrentState` - current executing state
- [x] `Goals` - P1 output (markdown)
- [x] `Task` - planned task (structured)
- [x] `TaskResult` - task execution result
- [x] `DeliveryResult` - delivery output
- [x] `LearningEntry` - knowledge to save

### 1.5 Clock Context (`types/clock.go`)

- [x] `ClockContext` struct - time context for P0
- [x] `NewClockContext()` - constructor

### 1.6 Inspiration (`types/inspiration.go`)

- [x] `InspirationReport` struct - P0 output

### 1.7 Request/Response (`types/request.go`)

- [x] `InterveneRequest` - human intervention request
- [x] `EventRequest` - event trigger request
- [x] `ExecutionResult` - trigger result
- [x] `RobotState` - robot status query result

### 1.8 Interfaces (`types/interfaces.go`)

- [x] `Manager` interface
- [x] `Executor` interface
- [x] `Pool` interface
- [x] `Cache` interface
- [x] `Dedup` interface
- [x] `Store` interface

### 1.9 Errors (`types/errors.go`)

- [x] Config errors
- [x] Runtime errors
- [x] Phase errors

### 1.10 Tests

- [x] `types/enums_test.go` - enum validation
- [x] `types/config_test.go` - config validation
- [x] `types/clock_test.go` - clock context creation
- [x] `types/robot_test.go` - robot methods

---

## Phase 2: Skeleton Implementation ✅

**Goal:** Create all packages with empty/stub implementations. Code compiles.

**Status:** Complete - All packages compile successfully, no circular dependencies

### 2.1 Utils (`utils/`) ✅

- [x] `utils/convert.go` - JSON, map, struct conversions (implement)
- [x] `utils/time.go` - time parsing, formatting, timezone (implement)
- [x] `utils/id.go` - ID generation (nanoid) (implement)
- [x] `utils/validate.go` - validation helpers (implement)
- [x] Test: `utils/utils_test.go`

### 2.2 Package Skeletons ✅ (stubs only, implemented in Phase 3)

Create empty structs and stub methods that return nil/empty/success:

- [x] `cache/cache.go` - Cache struct, stub methods
- [x] `dedup/dedup.go` - Dedup struct, stub methods
- [x] `store/store.go` - Store struct, stub methods
- [x] `pool/pool.go` - Pool struct, stub methods
- [x] `job/job.go` - job helper stubs
- [x] `plan/plan.go` - Plan struct, stub methods
- [x] `trigger/trigger.go` - trigger dispatcher stub
- [x] `executor/executor.go` - Executor struct, stub `Execute()`
- [x] `manager/manager.go` - Manager struct, stub methods

### 2.3 API Skeletons ✅

- [x] `api/api.go` - Go API facade (all function signatures, return errors)
- [x] `api/process.go` - Yao Process registration (all processes, return errors)
- [x] `api/jsapi.go` - JSAPI registration (all methods, return errors)

### 2.4 Root ✅

- [x] `robot.go` - package entry
  - [x] `Init()` - placeholder
  - [x] `Shutdown()` - placeholder

### 2.5 Compile Test ✅

- [x] All packages compile without errors
- [x] All imports resolve correctly
- [x] No circular dependencies

---

## Phase 3: Complete Scheduling System

**Goal:** Implement complete scheduling system. Executor is stub (simulates success).

This phase delivers a fully working scheduling pipeline:

```
Trigger → Manager → Cache → Dedup → Pool → Worker → Executor(stub) → Job
```

### ✅ 3.1 Cache Implementation (COMPLETE)

- [x] `cache/cache.go` - Cache struct with thread-safe map
- [x] `cache/load.go` - load robots from `__yao.member` where `member_type='robot'` and `autonomous_mode=true`
  - [x] Implemented pagination (100 robots per page)
  - [x] Configurable model name via `SetMemberModel()`
- [x] `cache/refresh.go` - refresh single robot, periodic full refresh (every hour)
- [x] Test: load/refresh with real DB
  - [x] Created comprehensive integration tests with real database
  - [x] Tests cover Load, LoadByID, Refresh, ListByTeam, GetByStatus
  - [x] All tests passing with proper cleanup

### ✅ 3.2 Pool Implementation (COMPLETE)

- [x] `pool/pool.go` - worker pool with configurable size (global limit)
  - [x] Default config: 10 workers, 100 queue size
  - [x] Configurable via `pool.NewWithConfig()`
- [x] `pool/queue.go` - priority queue (sorted by: robot priority, trigger type, wait time)
  - [x] Two-level limit: global queue + per-robot queue
  - [x] Priority: Robot Priority × 1000 + Trigger Priority × 100
- [x] `pool/worker.go` - worker goroutines, dispatch to executor
  - [x] Non-blocking quota check with re-enqueue
  - [x] Graceful shutdown support
- [x] Test: submit jobs, verify execution order, verify concurrency limits
  - [x] 15 test cases covering all edge cases
  - [x] All tests passing

### 3.3 Trigger Implementation

- [ ] `trigger/trigger.go` - trigger dispatcher (routes to clock/intervene/event)
- [ ] `trigger/clock.go` - clock trigger
  - [ ] `times` mode: match specific times (09:00, 14:00)
  - [ ] `interval` mode: run every X duration (30m, 1h)
  - [ ] `daemon` mode: restart immediately after completion
  - [ ] Timezone handling
- [ ] `trigger/intervene.go` - human intervention
  - [ ] Parse action (task.add, goal.adjust, etc.)
  - [ ] Build TriggerInput with Messages
- [ ] `trigger/event.go` - event handling
  - [ ] Webhook event dispatch
  - [ ] Database change event dispatch
- [ ] `trigger/control.go` - execution control
  - [ ] Pause execution
  - [ ] Resume execution
  - [ ] Cancel/Stop execution
- [ ] Test: clock matching (all modes), intervention handling, event dispatch

### 3.4 Dedup Implementation

- [ ] `dedup/dedup.go` - Dedup struct
- [ ] `dedup/fast.go` - fast in-memory time-window dedup
  - [ ] Key: `memberID:triggerType:window`
  - [ ] Check before submit
  - [ ] Mark after submit
- [ ] Test: dedup check/mark, window expiry

### 3.5 Job Integration

- [ ] `job/job.go` - create job
  - [ ] `job_id`: `robot_exec_{execID}`
  - [ ] `category_id`: `autonomous_robot`
  - [ ] Metadata: member_id, team_id, trigger_type, exec_id
- [ ] `job/execution.go` - execution lifecycle
  - [ ] Create execution on trigger
  - [ ] Update status on phase change
  - [ ] Complete/fail on finish
- [ ] `job/log.go` - write phase logs
  - [ ] Log phase start/end
  - [ ] Log errors
- [ ] Test: job creation, execution tracking, log writing

### 3.6 Manager Implementation

- [ ] `manager/manager.go` - Manager struct
  - [ ] `Start()` - start ticker goroutine, start pool
  - [ ] `Stop()` - graceful shutdown (wait for running, drain queue)
  - [ ] `Tick()` - main loop:
    1. Get all cached robots
    2. For each robot with clock trigger enabled
    3. Check if should execute (schedule match + dedup)
    4. Submit to pool
- [ ] Test: manager start/stop, tick cycle

### 3.7 Executor Stub

- [ ] `executor/executor.go` - stub implementation
  - [ ] `Execute()` - simulate full execution
    1. Create Execution record
    2. Update phase: P0 → P1 → P2 → P3 → P4 → P5
    3. Sleep briefly between phases (simulate work)
    4. Return success with mock data
- [ ] Test: verify stub called, verify phase progression

### 3.8 Integration Test (End-to-End Scheduling)

- [ ] Create test robot in `__yao.member` with clock config
- [ ] Start manager
- [ ] Wait for clock trigger
- [ ] Verify:
  - [ ] Robot loaded to cache
  - [ ] Clock trigger matched
  - [ ] Dedup checked
  - [ ] Job submitted to pool
  - [ ] Worker picked up job
  - [ ] Executor stub called
  - [ ] Job execution recorded
  - [ ] Logs written
- [ ] Test human intervention trigger
- [ ] Test event trigger
- [ ] Test concurrent executions (multiple robots)
- [ ] Test quota enforcement (per-robot limit)
- [ ] Test pause/resume/stop

---

## Phase 4: Executor - P0 Inspiration

**Goal:** Implement P0 (Inspiration Agent). Clock trigger → P0 → stub P1-P5.

### 4.1 Test Assistant Setup

Create `yao-dev-app/assistants/robot/` directory:

- [ ] `inspiration/package.yao` - Inspiration Agent config
- [ ] `inspiration/prompts.yml` - P0 prompts
- [ ] `inspiration/src/index.ts` - hooks if needed

### 4.2 P0 Implementation

- [ ] `executor/inspiration.go` - build prompt with `ClockContext`
- [ ] `executor/inspiration.go` - call Inspiration Agent
- [ ] `executor/inspiration.go` - parse response to `InspirationReport`
- [ ] `executor/prompt.go` - `BuildInspirationPrompt()`

### 4.3 Tests

- [ ] `executor/inspiration_test.go` - P0 with real LLM call
- [ ] Verify: clock context in prompt
- [ ] Verify: markdown report generated

---

## Phase 5: Executor - P1 Goals

**Goal:** Implement P1 (Goal Generation Agent). P0 → P1 → stub P2-P5.

### 5.1 Test Assistant Setup

- [ ] `goals/package.yao` - Goal Generation Agent config
- [ ] `goals/prompts.yml` - P1 prompts

### 5.2 P1 Implementation

- [ ] `executor/goals.go` - build prompt with inspiration report
- [ ] `executor/goals.go` - call Goal Agent
- [ ] `executor/goals.go` - parse response to `Goals` (markdown)
- [ ] `executor/prompt.go` - `BuildGoalsPrompt()`

### 5.3 Tests

- [ ] `executor/goals_test.go` - P1 with real LLM call
- [ ] Verify: inspiration report in prompt
- [ ] Verify: goals markdown generated

---

## Phase 6: Executor - P2 Tasks

**Goal:** Implement P2 (Task Planning Agent). P1 → P2 → stub P3-P5.

### 6.1 Test Assistant Setup

- [ ] `tasks/package.yao` - Task Planning Agent config
- [ ] `tasks/prompts.yml` - P2 prompts

### 6.2 P2 Implementation

- [ ] `executor/tasks.go` - build prompt with goals
- [ ] `executor/tasks.go` - call Task Agent
- [ ] `executor/tasks.go` - parse response to `[]Task` (structured)
- [ ] `executor/prompt.go` - `BuildTasksPrompt()`

### 6.3 Tests

- [ ] `executor/tasks_test.go` - P2 with real LLM call
- [ ] Verify: goals in prompt
- [ ] Verify: structured tasks generated

---

## Phase 7: Executor - P3 Run

**Goal:** Implement P3 (Task Execution). P2 → P3 → stub P4-P5.

### 7.1 Implementation

- [ ] `executor/run.go` - iterate tasks
- [ ] `executor/run.go` - call executor (assistant/mcp/process)
- [ ] `executor/run.go` - collect results
- [ ] `executor/agent.go` - unified agent call method

### 7.2 Validation Agent Setup

- [ ] `validation/package.yao` - Validation Agent config
- [ ] `validation/prompts.yml` - validation prompts

### 7.3 Tests

- [ ] `executor/run_test.go` - P3 with real agent calls
- [ ] Verify: tasks executed in order
- [ ] Verify: results collected

---

## Phase 8: Executor - P4 Delivery

**Goal:** Implement P4 (Delivery). P3 → P4 → stub P5.

### 8.1 Test Assistant Setup

- [ ] `delivery/package.yao` - Delivery Agent config
- [ ] `delivery/prompts.yml` - delivery prompts

### 8.2 Implementation

- [ ] `executor/delivery.go` - build delivery content
- [ ] `executor/delivery.go` - send via configured channel (email/file/webhook/notify)

### 8.3 Tests

- [ ] `executor/delivery_test.go` - P4 delivery
- [ ] Verify: delivery sent (mock or real)

---

## Phase 9: Executor - P5 Learning

**Goal:** Implement P5 (Learning). Full execution flow complete.

### 9.1 Test Assistant Setup

- [ ] `learning/package.yao` - Learning Agent config
- [ ] `learning/prompts.yml` - learning prompts

### 9.2 Store Implementation

- [ ] `store/kb.go` - KB operations (create, save, search)
- [ ] `store/learning.go` - save learning entries to private KB

### 9.3 Implementation

- [ ] `executor/learning.go` - extract learnings from execution
- [ ] `executor/learning.go` - call Learning Agent
- [ ] `executor/learning.go` - save to private KB

### 9.4 Tests

- [ ] `executor/learning_test.go` - P5 learning
- [ ] Verify: learnings saved to KB

---

## Phase 10: API & Integration

**Goal:** Complete API implementation, end-to-end tests.

### 10.1 API Implementation

- [ ] `api/api.go` - implement all Go API functions
- [ ] `api/process.go` - implement all Process handlers
- [ ] `api/jsapi.go` - implement JSAPI

### 10.2 End-to-End Tests

- [ ] Full clock trigger flow (P0 → P5)
- [ ] Human intervention flow (P1 → P5)
- [ ] Event trigger flow (P1 → P5)
- [ ] Concurrent execution test
- [ ] Pause/Resume/Stop test

### 10.3 Integration with OpenAPI

- [ ] HTTP endpoints for human intervention
- [ ] Webhook endpoints for events

---

## Phase 11: Advanced Features

**Goal:** Implement semantic dedup, plan queue.

### 11.1 Semantic Dedup

- [ ] `dedup/semantic.go` - call Dedup Agent for goal/task level dedup
- [ ] Dedup Agent setup (`assistants/robot/dedup/`)
- [ ] Test: semantic dedup with real LLM

### 11.2 Plan Queue

- [ ] `plan/plan.go` - plan queue implementation
  - [ ] Store planned tasks/goals
  - [ ] Execute at next cycle or specified time
- [ ] `plan/schedule.go` - schedule for later
- [ ] Test: plan queue operations

> **Note:** Monitoring is provided by Job system (Activity Monitor UI). No separate implementation needed.

---

## Test Assistants Structure

```
yao-dev-app/assistants/robot/
├── inspiration/           # P0: Inspiration Agent
│   ├── package.yao
│   └── prompts.yml
├── goals/                 # P1: Goal Generation Agent
│   ├── package.yao
│   └── prompts.yml
├── tasks/                 # P2: Task Planning Agent
│   ├── package.yao
│   └── prompts.yml
├── validation/            # P3: Validation Agent
│   ├── package.yao
│   └── prompts.yml
├── delivery/              # P4: Delivery Agent
│   ├── package.yao
│   └── prompts.yml
├── learning/              # P5: Learning Agent
│   ├── package.yao
│   └── prompts.yml
└── dedup/                 # Deduplication Agent
    ├── package.yao
    └── prompts.yml
```

---

## Notes

### Test Environment Setup

1. **Environment Variables:** Run `source yao/env.local.sh` before tests
2. **Test Preparation:** Use `testutils.Prepare(t)` to load config, KB, and agents

```go
package robot_test

import (
    "testing"
    "github.com/yaoapp/yao/agent/testutils"
)

func TestExample(t *testing.T) {
    // Load environment config (from YAO_TEST_APPLICATION)
    // This loads: config, connectors, KB, agents, models, etc.
    testutils.Prepare(t)
    defer testutils.Clean(t)

    // Your test code here
}
```

### Test Conventions

1. **Black-box Tests:** All tests in `*_test` package (external package)
2. **Real LLM Calls:** Use `gpt-4o` or `deepseek` connectors for agent tests
3. **Incremental:** Each phase builds on previous, all tests must pass before next phase
4. **No Skip:** Do NOT use `t.Skip()` except for `testing.Short()` (CI mode)
5. **Must Assert:** Every test MUST have result validation assertions

```go
func TestWithLLM(t *testing.T) {
    // Only allowed Skip: testing.Short() for CI
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    testutils.Prepare(t)
    defer testutils.Clean(t)

    // Your test code...
    result, err := SomeFunction()

    // MUST have assertions - no empty tests!
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, expected, result.Field)
}
```

### Test Rules

| Rule              | Description                               |
| ----------------- | ----------------------------------------- |
| No arbitrary Skip | Only `testing.Short()` skip allowed       |
| Must assert       | Every test must validate results          |
| No empty tests    | Tests without assertions will fail review |
| Real calls        | LLM tests use real API calls, not mocks   |

### Key Environment Variables

| Variable               | Description                     |
| ---------------------- | ------------------------------- |
| `YAO_TEST_APPLICATION` | Test app path (`yao-dev-app`)   |
| `OPENAI_TEST_KEY`      | OpenAI API key                  |
| `DEEPSEEK_API_KEY`     | DeepSeek API key                |
| `YAO_DB_DRIVER`        | Database driver (mysql/sqlite3) |
| `YAO_DB_PRIMARY`       | Database connection string      |

---

## Progress Tracking

| Phase                 | Status | Description                                          |
| --------------------- | ------ | ---------------------------------------------------- |
| 1. Types & Interfaces | ✅     | All types, enums, interfaces                         |
| 2. Skeleton           | ✅     | Empty stubs, code compiles                           |
| 3. Scheduling System  | ⬜     | Cache + Pool + Trigger + Dedup + Job (executor stub) |
| 4. P0 Inspiration     | ⬜     | Inspiration Agent integration                        |
| 5. P1 Goals           | ⬜     | Goal Generation Agent integration                    |
| 6. P2 Tasks           | ⬜     | Task Planning Agent integration                      |
| 7. P3 Run             | ⬜     | Task execution (assistant/mcp/process)               |
| 8. P4 Delivery        | ⬜     | Output delivery (email/file/webhook/notify)          |
| 9. P5 Learning        | ⬜     | Learning Agent + KB save                             |
| 10. API & Integration | ⬜     | Complete API, end-to-end tests                       |
| 11. Advanced          | ⬜     | Semantic dedup, plan queue                           |

Legend: ⬜ Not started | 🟡 In progress | ✅ Complete

---

## Quick Commands

```bash
# Setup environment
source yao/env.local.sh

# Run all robot tests
go test -v ./agent/robot/...

# Run specific phase tests
go test -v ./agent/robot/types/...
go test -v ./agent/robot/cache/...
go test -v ./agent/robot/pool/...
go test -v ./agent/robot/executor/...

# Run with coverage
go test -cover ./agent/robot/...
```
