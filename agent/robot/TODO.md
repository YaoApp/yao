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

## Phase 3: Complete Scheduling System ✅

**Goal:** Implement complete scheduling system. Executor is stub (simulates success).

**Status:** Complete - All 7 sub-tasks done, 80+ integration tests passing

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

### ✅ 3.3 Manager Implementation (COMPLETE)

> **Note:** Manager is the scheduling core, depends on completed Cache and Pool.

- [x] `manager/manager.go` - Manager struct
  - [x] `Start()` - load cache, start pool, start ticker goroutine
  - [x] `Stop()` - graceful shutdown (wait for running, drain queue)
  - [x] `Tick()` - main loop:
    1. Get all cached robots
    2. For each robot with clock trigger enabled
    3. Check if should execute (times/interval/daemon modes)
    4. Submit to pool
  - [x] `TriggerManual()` - manual trigger for testing/API
  - [x] Clock modes: times, interval, daemon
  - [x] Day matching for times mode
  - [x] Timezone handling
  - [x] Skip paused/error/maintenance robots
- [x] Test: manager start/stop, tick cycle, manual trigger, clock modes, goroutine leak

### ✅ 3.4 Trigger Implementation (COMPLETE)

- [x] `trigger/trigger.go` - validation and helper functions
  - [x] `ValidateIntervention()` - validate human intervention requests
  - [x] `ValidateEvent()` - validate event trigger requests
  - [x] `BuildEventInput()` - build TriggerInput from event request
  - [x] `GetActionCategory()` / `GetActionDescription()` - action helpers
- [x] `trigger/clock.go` - ClockMatcher for clock trigger matching
  - [x] `times` mode: match specific times (09:00, 14:00)
  - [x] `interval` mode: run every X duration (30m, 1h)
  - [x] `daemon` mode: restart immediately after completion
  - [x] Timezone handling
  - [x] Day-of-week filtering
- [x] `trigger/control.go` - ExecutionController for pause/resume/stop
  - [x] Track/Untrack executions
  - [x] Pause/Resume execution
  - [x] Stop execution (cancel context)
  - [x] WaitIfPaused() for executor integration
- [x] `manager/manager.go` - integrated trigger handling
  - [x] `Intervene()` - human intervention handler
  - [x] `HandleEvent()` - event trigger handler
  - [x] `PauseExecution()` / `ResumeExecution()` / `StopExecution()`
  - [x] `ListExecutions()` / `ListExecutionsByMember()`
- [x] Tests: `trigger/trigger_test.go`, `trigger/clock_test.go`, `trigger/control_test.go`
  - [x] Validation tests for intervention and event requests
  - [x] Clock matching tests for all modes
  - [x] ExecutionController lifecycle tests
  - [x] Manager integration tests for Intervene/HandleEvent

### ✅ 3.5 Job Integration (COMPLETE)

- [x] `job/job.go` - create job
  - [x] `job_id`: `robot_exec_{execID}`
  - [x] `category_name`: `Autonomous Robot` / `自主机器人` (localized)
  - [x] Metadata: member_id, team_id, trigger_type, exec_id, display_name
  - [x] `Options` struct for extensibility (Priority, MaxRetryCount, DefaultTimeout, Metadata)
  - [x] `Create()`, `Get()`, `Update()`, `Complete()`, `Fail()`, `Cancel()`
  - [x] Status mapping: ExecPending→queued, ExecRunning→running, etc.
  - [x] Localization support (en-US, zh-CN)
- [x] `job/execution.go` - execution lifecycle
  - [x] `CreateOptions` struct for extensibility
  - [x] `CreateExecution()` - create both robot Execution and job.Execution
  - [x] `UpdatePhase()` - update phase with progress tracking (10%→25%→40%→60%→80%→95%)
  - [x] `UpdateStatus()` - update execution status
  - [x] `CompleteExecution()` / `FailExecution()` / `CancelExecution()`
  - [x] TriggerType → TriggerCategory mapping (clock→scheduled, human→manual, event→event)
  - [x] Duration calculation on completion/failure/cancellation
- [x] `job/log.go` - write phase logs
  - [x] `Log()` - base log function with context
  - [x] `LogPhaseStart()` / `LogPhaseEnd()` / `LogPhaseError()`
  - [x] `LogError()` / `LogInfo()` / `LogDebug()` / `LogWarn()`
  - [x] `LogTaskStart()` / `LogTaskEnd()`
  - [x] `LogDelivery()` / `LogLearning()`
  - [x] Localization support for all log messages
- [x] Test: job creation, execution tracking, log writing
  - [x] `job/job_test.go` - 17 test cases
  - [x] `job/execution_test.go` - 26 test cases
  - [x] `job/log_test.go` - 24 test cases
  - [x] All tests passing with real database

### ✅ 3.6 Executor Architecture (COMPLETE)

Pluggable executor architecture with multiple execution modes:

```
executor/
├── types/
│   ├── types.go      # Executor interface, Config types
│   └── helpers.go    # Shared helper functions
├── standard/
│   ├── executor.go   # Real Agent execution (production)
│   ├── agent.go      # AgentCaller for LLM calls
│   ├── input.go      # InputFormatter for prompts
│   ├── inspiration.go # P0: Inspiration phase
│   ├── goals.go      # P1: Goals phase
│   ├── tasks.go      # P2: Tasks phase
│   ├── run.go        # P3: Run phase
│   ├── delivery.go   # P4: Delivery phase
│   └── learning.go   # P5: Learning phase
├── dryrun/
│   └── executor.go   # Simulated execution (testing/demo)
├── sandbox/
│   └── executor.go   # Container-isolated (NOT IMPLEMENTED)
└── executor.go       # Factory functions
```

**Execution Modes:**

| Mode     | Use Case                         | Status             |
| -------- | -------------------------------- | ------------------ |
| Standard | Production with real Agent calls | ✅ Implemented     |
| DryRun   | Tests, demos, scheduling tests   | ✅ Implemented     |
| Sandbox  | Container-isolated execution     | ⬜ Not Implemented |

> **⚠️ Sandbox Mode:** Requires container-level isolation (Docker/gVisor/Firecracker)
> for true security. Current placeholder behaves like DryRun. Future feature.

- [x] `executor/types/types.go` - `Executor` interface, `PhaseExecutor` interface
- [x] `executor/types/helpers.go` - `BuildTriggerInput()` shared helper
- [x] `executor/executor.go` - Factory functions (`New`, `NewDryRun`, `NewWithMode`)
- [x] `executor/standard/executor.go` - Real execution with Job integration
- [x] `executor/standard/phases.go` - Phase implementations (P0-P5)
- [x] `executor/dryrun/executor.go` - Simulated execution with callbacks
- [x] `executor/sandbox/executor.go` - Placeholder (NOT IMPLEMENTED)
- [x] Manager integration - accepts `Executor` interface via config
- [x] Tests use DryRun mode for scheduling/concurrency tests

### 3.7 Integration Test (End-to-End Scheduling) ✅

- [x] Create test robot in `__yao.member` with clock config
- [x] Start manager
- [x] Wait for clock trigger
- [x] Verify:
  - [x] Robot loaded to cache
  - [x] Clock trigger matched
  - [x] Job submitted to pool
  - [x] Worker picked up job
  - [x] Executor stub called
  - [x] Job execution recorded
  - [x] Logs written
- [x] Test human intervention trigger
- [x] Test event trigger
- [x] Test concurrent executions (multiple robots)
- [x] Test quota enforcement (per-robot limit)
- [x] Test pause/resume/stop

**Test Files Created:**

- `manager/integration_test.go` - Core scheduling flow (Cache→Pool→Executor)
- `manager/integration_clock_test.go` - Clock trigger modes (times/interval/daemon)
- `manager/integration_human_test.go` - Human intervention trigger tests
- `manager/integration_event_test.go` - Event trigger tests
- `manager/integration_concurrent_test.go` - Concurrent execution & quota tests
- `manager/integration_control_test.go` - Pause/Resume/Stop tests

**Test Coverage:**

- 27 top-level test functions
- 80+ sub-tests covering all verification points
- 3x run stability verified

---

## Phase 4: Agent Call Infrastructure ✅

**Goal:** Implement unified Agent/Assistant calling mechanism. This is the foundation for all phase implementations (P0-P5).

**Architecture Note:**

- **Prompt construction is handled by Assistant layer** (`prompts.yml` in each assistant)
- **Executor only prepares input data** (ClockContext, InspirationReport, etc.) and calls Assistant
- **Assistant framework handles** prompt rendering, LLM API calls, streaming

**Implemented:**

1. A unified way to call assistants with streaming support
2. Input data formatting for each phase
3. Response parsing (markdown and structured data via `gou/text`)
4. Multi-turn conversation support

### 4.1 Agent Caller Implementation ✅

- [x] `executor/agent.go` - `AgentCaller` struct with `SkipOutput`, `SkipHistory`, `SkipSearch`, `ChatID`
- [x] `executor/agent.go` - `Call(ctx, assistantID, messages)` - basic call with full response
- [x] `executor/agent.go` - `CallWithMessages(ctx, assistantID, userContent)` - convenience method
- [x] `executor/agent.go` - `CallWithSystemAndUser(ctx, assistantID, systemContent, userContent)`
- [x] `executor/agent.go` - handle assistant not found error
- [x] `executor/agent.go` - handle LLM API errors gracefully
- [x] `executor/agent.go` - `CallResult.GetJSON()` / `GetJSONArray()` - parse JSON response using `gou/text`
- [x] `executor/agent.go` - `Conversation` struct for multi-turn dialogues
- [x] `executor/agent.go` - `Conversation.Turn()`, `RunUntil()`, `Reset()`, `WithSystemPrompt()`
- [x] `executor/agent.go` - Use `agentcontext.Noop()` logger to suppress debug output

### 4.2 Input Formatters ✅

- [x] `executor/input.go` - `FormatClockContext(clockCtx, robot)` - format clock context as message content
- [x] `executor/input.go` - `FormatInspirationReport(report)` - format P0 output for P1 input
- [x] `executor/input.go` - `FormatTriggerInput(input)` - format Human/Event trigger for P1 input
- [x] `executor/input.go` - `FormatGoals(goals, robot)` - format P1 output for P2 input
- [x] `executor/input.go` - `FormatTasks(tasks)` - format P2 output for P3 input
- [x] `executor/input.go` - `FormatTaskResults(results)` - format P3 output for P4/P5 input
- [x] `executor/input.go` - `FormatExecutionSummary(exec)` - format full execution for P5 input
- [x] `executor/input.go` - `BuildMessages()`, `BuildMessagesWithSystem()` - helper methods

### 4.3 Test Assistants ✅

- [x] `yao-dev-app/assistants/tests/robot-single/` - Single-turn test assistant
- [x] `yao-dev-app/assistants/tests/robot-conversation/` - Multi-turn conversation test assistant

### 4.4 Tests ✅

- [x] `executor/agent_test.go` - 22 test cases for AgentCaller and Conversation
- [x] `executor/input_test.go` - 20 test cases for InputFormatter
- [x] Verify: assistant can be called and returns response
- [x] Verify: multi-turn conversation maintains state
- [x] Verify: input data is well-formatted for assistant prompts
- [x] Verify: JSON/YAML extraction from LLM output works correctly

---

## Phase 5: Test Scenario & Assistants Setup ✅

**Goal:** Create realistic test scenarios with all required assistants.

**Architecture:**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      6 Generic Phase Agents (P0-P5)                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  inspiration  │  goals  │  tasks  │  validation  │  delivery  │  learning   │
│     (P0)      │  (P1)   │  (P2)   │     (P3)     │    (P4)    │    (P5)     │
└───────────────┴─────────┴─────────┴──────────────┴────────────┴─────────────┘
                                    ↓ P2 assigns tasks to
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Expert Agents (Task Executors)                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  text-writer   │  web-reader  │  data-analyst  │  summarizer  │  ...        │
│  (Generate)    │  (Fetch URL) │  (Analyze)     │  (Summarize) │             │
└───────────────┴──────────────┴────────────────┴──────────────┴─────────────┘
```

**Test Strategy:**

- Phase Agents (P0-P5) are **generic** and reusable across all robot types
- Expert Agents are **specialized** for specific tasks (text, web, data, etc.)
- Each P0-P5 test uses **different expert combinations** to cover real scenarios
- Tests use `interval: 1s` or `TriggerManual()` for easy triggering (no time dependency)

### 5.1 Directory Structure

```
yao-dev-app/assistants/
├── robot/                    # Generic Phase Agents
│   ├── inspiration/          # P0: Analyze clock context, generate insights
│   │   ├── package.yao
│   │   └── prompts.yml
│   ├── goals/                # P1: Generate prioritized goals
│   │   ├── package.yao
│   │   └── prompts.yml
│   ├── tasks/                # P2: Split goals into executable tasks
│   │   ├── package.yao
│   │   └── prompts.yml
│   ├── validation/           # P3: Validate task results
│   │   ├── package.yao
│   │   └── prompts.yml
│   ├── delivery/             # P4: Format and deliver results
│   │   ├── package.yao
│   │   └── prompts.yml
│   └── learning/             # P5: Summarize execution, extract insights
│       ├── package.yao
│       └── prompts.yml
│
└── experts/                  # Expert Agents (Task Executors)
    ├── text-writer/          # Generate text content (reports, emails, summaries)
    │   ├── package.yao
    │   └── prompts.yml
    ├── web-reader/           # Fetch and parse web page content
    │   ├── package.yao
    │   └── prompts.yml
    ├── data-analyst/         # Analyze data, generate insights
    │   ├── package.yao
    │   └── prompts.yml
    └── summarizer/           # Summarize long text into key points
        ├── package.yao
        └── prompts.yml
```

### 5.2 Generic Phase Agents

#### 5.2.1 Inspiration Agent (P0)

- [x] `robot/inspiration/package.yao` - config with model, temperature
- [x] `robot/inspiration/prompts.yml` - system prompt:
  - Input: Clock context (time, day, markers), robot identity
  - Output: Markdown report with Summary, Highlights, Opportunities, Risks
  - Style: Analytical, context-aware

#### 5.2.2 Goals Agent (P1)

- [x] `robot/goals/package.yao` - config
- [x] `robot/goals/prompts.yml` - system prompt:
  - Input: Inspiration report OR trigger input (human/event)
  - Output: Prioritized goals in markdown (High/Normal/Low)
  - Style: Strategic, actionable

#### 5.2.3 Tasks Agent (P2)

- [x] `robot/tasks/package.yao` - config
- [x] `robot/tasks/prompts.yml` - system prompt:
  - Input: Goals, available expert agents list
  - Output: Structured task list (JSON) with executor assignments
  - Style: Detailed, executable

#### 5.2.4 Validation Agent (P3)

- [x] `robot/validation/package.yao` - config
- [x] `robot/validation/prompts.yml` - system prompt:
  - Input: Task result, expected outcome
  - Output: Validation result (pass/fail, issues, suggestions)
  - Style: Critical, thorough

#### 5.2.5 Delivery Agent (P4)

- [x] `robot/delivery/package.yao` - config
- [x] `robot/delivery/prompts.yml` - system prompt:
  - Input: Task results, delivery target (email, report, notification)
  - Output: Formatted delivery content
  - Style: Clear, professional

#### 5.2.6 Learning Agent (P5)

- [x] `robot/learning/package.yao` - config
- [x] `robot/learning/prompts.yml` - system prompt:
  - Input: Full execution summary
  - Output: Insights, patterns, improvement suggestions
  - Style: Reflective, insightful

### 5.3 Expert Agents (Task Executors)

#### 5.3.1 Text Writer

- [x] `experts/text-writer/package.yao` - config
- [x] `experts/text-writer/prompts.yml` - system prompt:
  - Input: Topic, key points, style (formal/casual), length
  - Output: Generated text content
  - Use cases: Weekly reports, email drafts, summaries

#### 5.3.2 Web Reader

- [x] `experts/web-reader/package.yao` - config with hooks
- [x] `experts/web-reader/prompts.yml` - system prompt:
  - Input: URL or topic to search
  - Output: Extracted content, key information
  - Use cases: News fetching, competitor monitoring, research
- [x] `experts/web-reader/src/fetch.ts` - HTTP fetching utilities
- [x] `experts/web-reader/src/fetch_test.ts` - 19 test cases (100% pass)
- [x] `experts/web-reader/src/index.ts` - Create/Next hooks

#### 5.3.3 Data Analyst

- [x] `experts/data-analyst/package.yao` - config
- [x] `experts/data-analyst/prompts.yml` - system prompt:
  - Input: Data description, analysis goal
  - Output: Analysis report, trends, insights
  - Use cases: Sales analysis, performance review

#### 5.3.4 Summarizer

- [x] `experts/summarizer/package.yao` - config
- [x] `experts/summarizer/prompts.yml` - system prompt:
  - Input: Long text content
  - Output: Concise summary with key points
  - Use cases: Document summarization, meeting notes

### 5.4 Test Scenarios

Each phase test uses different expert combinations:

| Test | Phase | Trigger          | Expert Agents Used       | Verification                  |
| ---- | ----- | ---------------- | ------------------------ | ----------------------------- |
| T1   | P0    | Clock (interval) | -                        | Clock → Inspiration report    |
| T2   | P1    | Clock            | -                        | Inspiration → Goals           |
| T3   | P1    | Human            | -                        | User input → Goals            |
| T4   | P2    | Clock            | text-writer, web-reader  | Goals → Tasks with executors  |
| T5   | P3    | Clock            | text-writer              | Task exec → Result validation |
| T6   | P3    | Human            | summarizer               | Task exec → Result validation |
| T7   | P4    | Clock            | -                        | Results → Delivery format     |
| T8   | P5    | Clock            | -                        | Full execution → Insights     |
| T9   | E2E   | Clock            | text-writer, summarizer  | Full P0→P5 flow               |
| T10  | E2E   | Human            | web-reader, data-analyst | Full P1→P5 flow               |

### 5.5 Verification

- [x] All 6 Phase Agents load correctly (`robot.inspiration`, `robot.goals`, etc.)
- [x] All 4 Expert Agents load correctly (`experts.text-writer`, `experts.web-reader`, etc.)
- [x] Web Reader `fetch.ts` utilities tested (19 tests, 100% pass)

---

## Phase 6: P0 Inspiration Implementation ✅

**Goal:** Implement P0 (Inspiration Agent). Clock trigger → P0 → stub P1-P5.

**Depends on:** Phase 4 (Agent Call Infrastructure), Phase 5 (Assistants Setup)

**Status:** COMPLETED

### 6.1 P0 Implementation

- [x] `executor/inspiration.go` - `RunInspiration(ctx, exec, data)` - real implementation
- [x] `executor/inspiration.go` - build prompt using `InputFormatter.FormatClockContext()`
- [x] `executor/inspiration.go` - call Inspiration Agent using `AgentCaller`
- [x] `executor/inspiration.go` - parse response to `InspirationReport` (markdown content)
- [x] `types/robot.go` - added `GetRobot()`/`SetRobot()` methods for Execution
- [x] `executor/executor.go` - set robot reference on execution creation

### 6.2 Tests

- [x] `executor/inspiration_test.go` - P0 with real LLM call (8 test cases)
- [x] Test: clock context correctly formatted in prompt
- [x] Test: robot identity included in prompt
- [x] Test: markdown report generated with expected sections
- [x] Test: handles LLM errors gracefully (robot nil, agent not found)
- [x] Test: uses clock from trigger input or creates new one
- [x] `InputFormatter.FormatClockContext()` unit tests (4 test cases)

### 6.3 Notes

- `executor_test.go` temporarily moved to `.bak` - will restore when all phases implemented
- P0 uses `robot.inspiration` test agent from `yao-dev-app/assistants/robot/inspiration/`

---

## Phase 7: P1 Goals Implementation

**Goal:** Implement P1 (Goal Generation Agent). P0 → P1 → stub P2-P5.

**Depends on:** Phase 6 (P0 Inspiration)

### 7.1 P1 Implementation

- [ ] `executor/goals.go` - `RunGoals(ctx, exec, data)` - real implementation
- [ ] `executor/goals.go` - build prompt with inspiration report
- [ ] `executor/goals.go` - call Goals Agent
- [ ] `executor/goals.go` - parse response to `Goals` struct
- [ ] `executor/goals.go` - handle Human/Event trigger (skip P0, use input directly)

### 7.2 Tests

- [ ] `executor/goals_test.go` - P1 with real LLM call
- [ ] Test: inspiration report in prompt (Clock trigger)
- [ ] Test: user input in prompt (Human trigger)
- [ ] Test: goals markdown generated with priorities
- [ ] Test: goals are actionable and measurable

---

## Phase 8: P2 Tasks Implementation

**Goal:** Implement P2 (Task Planning Agent). P1 → P2 → stub P3-P5.

**Depends on:** Phase 7 (P1 Goals)

### 8.1 P2 Implementation

- [ ] `executor/tasks.go` - `RunTasks(ctx, exec, data)` - real implementation
- [ ] `executor/tasks.go` - build prompt with goals
- [ ] `executor/tasks.go` - include available tools/agents in prompt
- [ ] `executor/tasks.go` - call Tasks Agent
- [ ] `executor/tasks.go` - parse response to `[]Task` (structured JSON)
- [ ] `executor/tasks.go` - validate task structure

### 8.2 Tests

- [ ] `executor/tasks_test.go` - P2 with real LLM call
- [ ] Test: goals included in prompt
- [ ] Test: available tools listed in prompt
- [ ] Test: structured tasks generated (2-3 tasks per goal)
- [ ] Test: each task has valid executor type and ID

---

## Phase 9: P3 Run Implementation

**Goal:** Implement P3 (Task Execution). P2 → P3 → stub P4-P5.

**Depends on:** Phase 8 (P2 Tasks)

### 9.1 Implementation

- [ ] `executor/run.go` - `RunExecution(ctx, exec, data)` - real implementation
- [ ] `executor/run.go` - iterate tasks in order
- [ ] `executor/run.go` - dispatch to correct executor (assistant/mcp/process)
- [ ] `executor/run.go` - collect results with timing
- [ ] `executor/run.go` - handle task failures gracefully
- [ ] `executor/run.go` - support pause/resume during execution

### 9.2 Validation Agent Setup

- [ ] `robot/validation/package.yao` - Validation Agent config
- [ ] `robot/validation/prompts.yml` - validation prompts

### 9.3 Tests

- [ ] `executor/run_test.go` - P3 with real agent calls
- [ ] Test: tasks executed in order
- [ ] Test: results collected with correct structure
- [ ] Test: task failure doesn't stop entire execution
- [ ] Test: pause/resume works during task execution

---

## Phase 10: P4 Delivery Implementation

**Goal:** Implement P4 (Delivery). P3 → P4 → stub P5.

**Depends on:** Phase 9 (P3 Run)

### 10.1 Delivery Agent Setup

- [ ] `robot/delivery/package.yao` - Delivery Agent config
- [ ] `robot/delivery/prompts.yml` - delivery prompts

### 10.2 Implementation

- [ ] `executor/delivery.go` - `RunDelivery(ctx, exec, data)` - real implementation
- [ ] `executor/delivery.go` - build delivery content from results
- [ ] `executor/delivery.go` - support email delivery
- [ ] `executor/delivery.go` - support file delivery
- [ ] `executor/delivery.go` - support webhook delivery
- [ ] `executor/delivery.go` - support notify delivery

### 10.3 Tests

- [ ] `executor/delivery_test.go` - P4 delivery
- [ ] Test: delivery content generated correctly
- [ ] Test: email delivery (mock or real)
- [ ] Test: file delivery to configured path

---

## Phase 11: P5 Learning Implementation

**Goal:** Implement P5 (Learning). Full execution flow complete.

**Depends on:** Phase 10 (P4 Delivery)

### 11.1 Learning Agent Setup

- [ ] `robot/learning/package.yao` - Learning Agent config
- [ ] `robot/learning/prompts.yml` - learning prompts

### 11.2 Store Implementation

- [ ] `store/store.go` - Store interface and struct
- [ ] `store/kb.go` - KB operations (create, save, search)
- [ ] `store/learning.go` - save learning entries to private KB

### 11.3 Implementation

- [ ] `executor/learning.go` - `RunLearning(ctx, exec, data)` - real implementation
- [ ] `executor/learning.go` - extract learnings from execution
- [ ] `executor/learning.go` - call Learning Agent
- [ ] `executor/learning.go` - save to private KB

### 11.4 Tests

- [ ] `executor/learning_test.go` - P5 learning
- [ ] Test: learnings extracted from execution
- [ ] Test: learnings saved to KB
- [ ] Test: KB can be queried for past learnings

---

## Phase 12: API & Integration

**Goal:** Complete API implementation, end-to-end tests.

### 12.1 API Implementation

- [ ] `api/api.go` - implement all Go API functions
- [ ] `api/process.go` - implement all Process handlers
- [ ] `api/jsapi.go` - implement JSAPI

### 12.2 End-to-End Tests

- [ ] Full clock trigger flow (P0 → P5)
- [ ] Human intervention flow (P1 → P5)
- [ ] Event trigger flow (P1 → P5)
- [ ] Concurrent execution test
- [ ] Pause/Resume/Stop test

### 12.3 Integration with OpenAPI

- [ ] HTTP endpoints for human intervention
- [ ] Webhook endpoints for events

---

## Phase 13: Advanced Features

**Goal:** Implement dedup, semantic dedup, plan queue.

### 13.1 Fast Dedup (Time-Window)

> **Note:** Manager has `// TODO: dedup check` comment placeholder. Integrate after implementation.

- [ ] `dedup/dedup.go` - Dedup struct
- [ ] `dedup/fast.go` - fast in-memory time-window dedup
  - [ ] Key: `memberID:triggerType:window`
  - [ ] Check before submit
  - [ ] Mark after submit
- [ ] Integrate into Manager.Tick()
- [ ] Test: dedup check/mark, window expiry

### 13.2 Semantic Dedup

- [ ] `dedup/semantic.go` - call Dedup Agent for goal/task level dedup
- [ ] Dedup Agent setup (`assistants/robot/dedup/`)
- [ ] Test: semantic dedup with real LLM

### 13.3 Plan Queue

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

| Phase                 | Status | Description                                                                  |
| --------------------- | ------ | ---------------------------------------------------------------------------- |
| 1. Types & Interfaces | ✅     | All types, enums, interfaces                                                 |
| 2. Skeleton           | ✅     | Empty stubs, code compiles                                                   |
| 3. Scheduling System  | ✅     | Cache + Pool + Trigger + Job + Executor architecture                         |
| 4. Agent Infra        | ✅     | AgentCaller, InputFormatter, test assistants                                 |
| 5. Test Scenarios     | ✅     | Phase agents (P0-P5), expert agents                                          |
| 6. P0 Inspiration     | ✅     | Inspiration Agent integration                                                |
| 7. P1 Goals           | ⬜     | Goal Generation Agent integration                                            |
| 8. P2 Tasks           | ⬜     | Task Planning Agent integration                                              |
| 9. P3 Run             | ⬜     | Task execution (assistant/mcp/process)                                       |
| 10. P4 Delivery       | ⬜     | Output delivery (email/file/webhook/notify)                                  |
| 11. P5 Learning       | ⬜     | Learning Agent + KB save                                                     |
| 12. API & Integration | ⬜     | Complete API, end-to-end tests                                               |
| 13. Advanced          | ⬜     | Semantic dedup, plan queue, Sandbox mode (requires container infrastructure) |

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
