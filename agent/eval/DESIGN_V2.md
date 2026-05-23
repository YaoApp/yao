# Agent Test Framework V2 Design

## Overview

This document describes the design for Agent Test Framework V2, which extends the existing testing capabilities with:

- **Message history support** - Test agents with conversation context via `input` array (already implemented)
- **Agent-driven testing** - Use agents to generate test cases and validate responses
- **Dynamic testing** - Simulator-driven testing with checkpoint validation

## Quick Reference: Format Rules

| Context               | Format                   | Example                                                 |
| --------------------- | ------------------------ | ------------------------------------------------------- |
| `-i` flag (CLI)       | Prefix required          | `agents:workers.test.gen`, `scripts:tests.gen`          |
| JSONL assertion `use` | Prefix required          | `"use": "agents:workers.test.validator"`                |
| JSONL `simulator.use` | No prefix (agent only)   | `"use": "workers.test.user-simulator"`                  |
| `--simulator` flag    | No prefix (agent only)   | `--simulator workers.test.user-simulator`               |
| `t.assert.Agent()`    | No prefix (method-bound) | `t.assert.Agent(resp, "workers.test.validator", {...})` |
| JSONL `before/after`  | No prefix (in src/)      | `"before": "env_test.Before"`                           |
| `--before/--after`    | No prefix (in src/)      | `--before env_test.BeforeAll`                           |

## Design Goals

1. **Simple** - Single-turn with optional message history, no complex multi-turn state
2. **Stateless** - Each test is independent, no session management needed
3. **Parallel** - Tests can run in parallel since they don't share state
4. **Flexible** - Support both static (messages) and dynamic (simulator) testing
5. **Agent-driven** - Input generation, simulation, and validation can all be agent-powered

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         yao agent test                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  INPUT SOURCES (-i flag)                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                      │
│  │ JSONL File  │  │   Message   │  │ Generator   │                      │
│  │ ./test.jsonl│  │ "Hello..."  │  │ agents:xxx  │                      │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                      │
│         │                │                │                              │
│         └────────────────┴────────────────┘                              │
│                          │                                               │
│                          ▼                                               │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                      Test Case Parser                              │  │
│  │                                                                    │  │
│  │  Standard Mode:     {input: "..." | [...], assertions}            │  │
│  │  Dynamic Mode:      {simulator: {...}, checkpoints: [...]}        │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                          │                                               │
│          ┌───────────────┴───────────────┐                              │
│          ▼                               ▼                              │
│  ┌───────────────────┐       ┌───────────────────────┐                  │
│  │   STANDARD MODE   │       │    DYNAMIC MODE       │                  │
│  │                   │       │                       │                  │
│  │ 1. Build messages │       │ LOOP:                 │                  │
│  │ 2. Call Agent     │       │  1. Simulator→input   │                  │
│  │ 3. Run assertions │       │  2. Call Agent        │                  │
│  │                   │       │  3. Check checkpoints │                  │
│  │ → PASS/FAIL       │       │  4. Until done        │                  │
│  └───────────────────┘       └───────────────────────┘                  │
│                          │                                               │
│                          ▼                                               │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                          Reporter                                  │  │
│  │  - Console output                                                  │  │
│  │  - JSON file output                                                │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Test Modes

### Standard Mode (Default)

Single call to agent with optional message history. **No multi-turn state management needed.**

| Field        | Type                           | Description                                   |
| ------------ | ------------------------------ | --------------------------------------------- |
| `input`      | string \| Message \| Message[] | Text, single message, or conversation history |
| `assertions` | array                          | Assertions to validate response               |
| `options`    | object                         | `context.Options` passed to agent             |

### Dynamic Mode

Simulator-driven testing with checkpoint validation.

| Field         | Type   | Description                      |
| ------------- | ------ | -------------------------------- |
| `simulator`   | object | Simulator agent configuration    |
| `checkpoints` | array  | Functional checkpoints to verify |
| `max_turns`   | int    | Maximum turns before timeout     |
| `timeout`     | string | Maximum time (e.g., "5m")        |

## Test Case Format

### Simple Input (Existing)

```jsonl
{
  "id": "T001",
  "input": "Hello",
  "assertions": [
    {
      "type": "contains",
      "value": "Hi"
    }
  ]
}
```

### With Message History (Existing)

The `input` field already supports message arrays for conversation context:

```jsonl
{
  "id": "T002",
  "name": "Expense submission - final confirmation",
  "input": [
    {
      "role": "user",
      "content": "I want to submit an expense"
    },
    {
      "role": "assistant",
      "content": "What type of expense would you like to submit?"
    },
    {
      "role": "user",
      "content": "Business travel to Beijing, $3500"
    },
    {
      "role": "assistant",
      "content": "I'll create an expense for business travel, $3500. Please confirm."
    },
    {
      "role": "user",
      "content": "Yes, confirm"
    }
  ],
  "assertions": [
    {
      "type": "contains",
      "value": "submitted"
    },
    {
      "type": "tool_called",
      "name": "create_expense"
    }
  ]
}
```

**Key insight**: Instead of executing 3 turns sequentially, we pass the full conversation history. The agent sees the context and responds to the last message. This is:

- **Simpler** - No turn-by-turn execution, no session state
- **Faster** - Single API call instead of multiple
- **Parallelizable** - Each test is independent
- **Debuggable** - Clear input/output for each test

### Testing Different Points in a Conversation

To test agent behavior at different conversation stages, create separate test cases:

```jsonl
// Test 1: First turn - agent should ask for expense type
{
  "id": "expense-turn1",
  "input": [{"role": "user", "content": "I want to submit an expense"}],
  "assertions": [{"type": "contains", "value": "type"}]
}

// Test 2: Second turn - agent should create expense
{
  "id": "expense-turn2",
  "input": [
    {"role": "user", "content": "I want to submit an expense"},
    {"role": "assistant", "content": "What type of expense would you like to submit?"},
    {"role": "user", "content": "Business travel, $3500"}
  ],
  "assertions": [{"type": "tool_called", "name": "create_expense"}]
}

// Test 3: Final turn - agent should confirm submission
{
  "id": "expense-turn3",
  "input": [
    {"role": "user", "content": "I want to submit an expense"},
    {"role": "assistant", "content": "What type of expense?"},
    {"role": "user", "content": "Business travel, $3500"},
    {"role": "assistant", "content": "Confirm $3500 expense?"},
    {"role": "user", "content": "Yes"}
  ],
  "assertions": [{"type": "contains", "value": "submitted"}]
}
```

### With Attachments

```jsonl
{
  "id": "T003",
  "input": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "What's in this receipt?"
        },
        {
          "type": "image",
          "source": "file://./fixtures/receipt.jpg"
        }
      ]
    }
  ],
  "assertions": [
    {
      "type": "contains",
      "value": "amount"
    }
  ]
}
```

### Dynamic Mode (Simulator + Checkpoints)

For coverage testing where conversation flow is unpredictable:

```jsonl
{
  "id": "T004",
  "name": "Expense Submission Coverage",
  "simulator": {
    "use": "workers.test.user-simulator",
    "options": {
      "metadata": {
        "persona": "New employee unfamiliar with expense process",
        "goal": "Submit a $3500 travel expense"
      }
    }
  },
  "checkpoints": [
    {
      "id": "ask_type",
      "description": "Agent asks for expense type",
      "assertion": {
        "type": "contains",
        "value": "type"
      }
    },
    {
      "id": "call_create",
      "description": "Agent calls create_expense",
      "after": [
        "ask_type"
      ],
      "assertion": {
        "type": "tool_called",
        "name": "create_expense"
      }
    },
    {
      "id": "confirm",
      "description": "Agent confirms submission",
      "after": [
        "call_create"
      ],
      "assertion": {
        "type": "contains",
        "value": "submitted"
      }
    }
  ],
  "max_turns": 10,
  "timeout": "2m"
}
```

## Field Descriptions

### Standard Mode Fields

| Field        | Type                           | Required | Description                                       |
| ------------ | ------------------------------ | -------- | ------------------------------------------------- |
| `id`         | string                         | Yes      | Unique test identifier                            |
| `name`       | string                         | No       | Human-readable test name                          |
| `input`      | string \| Message \| Message[] | Yes      | Input: text, single message, or message array     |
| `assertions` | array                          | No       | Assertions to validate response (alias: `assert`) |
| `options`    | object                         | No       | `context.Options` passed to agent                 |
| `before`     | string                         | No       | Before script (e.g., `env_test.Before`)           |
| `after`      | string                         | No       | After script (e.g., `env_test.After`)             |

**Note**: The `input` field supports three formats:

- `string`: Simple text (converted to `[{role: "user", content: "..."}]`)
- `object`: Single message `{role: "...", content: "..."}`
- `array`: Message history `[{role: "user", ...}, {role: "assistant", ...}, ...]`

### Dynamic Mode Fields

| Field                       | Type   | Required | Description                                |
| --------------------------- | ------ | -------- | ------------------------------------------ |
| `id`                        | string | Yes      | Unique test identifier                     |
| `name`                      | string | No       | Human-readable test name                   |
| `simulator`                 | object | Yes      | User simulator configuration               |
| `simulator.use`             | string | Yes      | Simulator agent ID (no prefix)             |
| `simulator.options`         | object | No       | `context.Options` passed to simulator      |
| `checkpoints`               | array  | Yes      | Functionality checkpoints to verify        |
| `checkpoints[].id`          | string | Yes      | Unique checkpoint identifier               |
| `checkpoints[].description` | string | No       | Human-readable description                 |
| `checkpoints[].assertion`   | object | Yes      | Assertion to verify                        |
| `checkpoints[].after`       | array  | No       | Checkpoint IDs that must occur first       |
| `max_turns`                 | int    | No       | Maximum turns before timeout (default: 20) |
| `timeout`                   | string | No       | Maximum time (default: "5m")               |
| `options`                   | object | No       | `context.Options` passed to target agent   |
| `before`                    | string | No       | Before script function                     |
| `after`                     | string | No       | After script function                      |

## Before and After Scripts

JSONL test cases can reference `*_test.ts` scripts for environment preparation:

### Script Location

Scripts are located in the agent's `src/` directory (as `*_test.ts` files):

```
assistants/expense/
├── package.yao
├── prompts.yml
├── src/
│   ├── index.ts          # Main agent script
│   └── env_test.ts       # Before/after functions
└── tests/
    ├── inputs.jsonl      # Test cases
    └── fixtures/
        └── receipt.jpg
```

### Script Interface

```typescript
// src/env_test.ts

// Before function - called before test case runs
// Returns context data that will be passed to After
export function Before(ctx: Context, testCase: TestCase): BeforeResult {
  // Prepare database
  const userId = Process("models.user.Create", {
    name: "Test User",
    email: "test@example.com",
  });

  // Prepare knowledge base
  Process("knowledge.expense.Index", {
    documents: [{ title: "Policy", content: "Max expense $5000" }],
  });

  return {
    data: { userId, testId: testCase.id },
  };
}

// After function - called after test case completes (pass or fail)
export function After(
  ctx: Context,
  testCase: TestCase,
  result: TestResult,
  beforeData: any
) {
  // Clean up database
  if (beforeData?.userId) {
    Process("models.user.Delete", beforeData.userId);
  }

  // Clean up knowledge base
  Process("knowledge.expense.Clear");
}

// Global before - called once before all test cases
export function BeforeAll(ctx: Context, testCases: TestCase[]): BeforeResult {
  // One-time initialization
  Process("models.migrate");
  return { data: { initialized: true } };
}

// Global after - called once after all test cases
export function AfterAll(ctx: Context, results: TestResult[], beforeData: any) {
  // Final cleanup
  Process("models.cleanup");
}
```

### Test Case with Before/After

```jsonl
{
  "id": "T001",
  "name": "Submit expense with user context",
  "before": "env_test.Before",
  "after": "env_test.After",
  "input": "Submit a $500 travel expense",
  "assertions": [
    {
      "type": "tool_called",
      "name": "create_expense"
    }
  ]
}
```

### Global Before/After via CLI

```bash
# Run with global before/after
yao agent test -i ./tests/inputs.jsonl \
  --before env_test.BeforeAll \
  --after env_test.AfterAll
```

### Execution Order

```
┌─────────────────────────────────────────────────────────────────┐
│                    Test Execution with Before/After              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. BeforeAll() - Global initialization (once)                   │
│                              ↓                                   │
│  FOR EACH test case:                                             │
│    2. Before() - Per-test initialization                         │
│                              ↓                                   │
│    3. Run test (call agent, check assertions)                    │
│                              ↓                                   │
│    4. After() - Per-test cleanup (always runs)                   │
│                              ↓                                   │
│  5. AfterAll() - Global cleanup (once)                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Note**: Script tests (`*_test.ts`) don't need before/after fields since they can call functions directly within the test.

## Execution Flow

### Standard Mode

```
┌─────────────────────────────────────────────────────────────────┐
│                    Standard Mode Execution                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Parse test case                                              │
│     ├─ `input` is array? → Use as messages                       │
│     └─ `input` is string? → Convert to [{role: "user", content}] │
│                              ↓                                   │
│  2. Call Agent.Stream(ctx, messages, options)                    │
│                              ↓                                   │
│  3. Run assertions against response                              │
│     ├─ All PASS → Test PASSED ✅                                 │
│     └─ Any FAIL → Test FAILED ❌                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Dynamic Mode

```
┌─────────────────────────────────────────────────────────────────┐
│                    Dynamic Mode Execution                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Initialize:                                                     │
│    - pending_checkpoints = all checkpoints                       │
│    - messages = []                                               │
│    - turn_count = 0                                              │
│                              ↓                                   │
│  LOOP:                                                           │
│    1. Call Simulator → get user input                            │
│    2. Append user message to messages                            │
│    3. Call Agent.Stream(ctx, messages, options)                  │
│    4. Append assistant response to messages                      │
│    5. Check response against pending_checkpoints                 │
│       └─ If matched (and `after` satisfied) → move to reached    │
│    6. Check termination:                                         │
│       ├─ All checkpoints reached → PASSED ✅                     │
│       ├─ Simulator signals goal_achieved → FAILED ❌             │
│       ├─ turn_count >= max_turns → FAILED ❌                     │
│       └─ timeout exceeded → FAILED ❌                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Assertion Types

### Static Assertions

| Type          | Description            | Example                                                    |
| ------------- | ---------------------- | ---------------------------------------------------------- |
| `contains`    | Response contains text | `{"type": "contains", "value": "success"}`                 |
| `equals`      | Exact match            | `{"type": "equals", "value": "OK"}`                        |
| `regex`       | Regex pattern match    | `{"type": "regex", "pattern": "order-\\d+"}`               |
| `json_path`   | JSONPath value check   | `{"type": "json_path", "path": "$.status", "value": "ok"}` |
| `tool_called` | Tool was invoked       | `{"type": "tool_called", "name": "create_expense"}`        |
| `type`        | Value type check       | `{"type": "type", "path": "$.count", "value": "number"}`   |

### Agent-Driven Assertions

For semantic or fuzzy validation:

```jsonl
{
  "type": "agent",
  "use": "agents:workers.test.validator",
  "options": {
    "metadata": {
      "criteria": "Response should be helpful and answer the user's question",
      "tone": "professional and friendly"
    }
  }
}
```

### Script Assertions

For custom validation logic:

```jsonl
{
  "type": "script",
  "use": "scripts:tests.validate-expense",
  "options": {
    "metadata": {
      "min_amount": 100,
      "max_amount": 10000
    }
  }
}
```

## Script Testing with Agent Assertions

Script tests can use Agent-driven assertions via `t.assert.Agent()`:

```typescript
export function TestExpenseResponse(t: TestingT, ctx: Context) {
  const messages = [
    { role: "user", content: "I want to submit an expense" },
    { role: "assistant", content: "What type of expense?" },
    { role: "user", content: "Travel, $500" },
  ];

  const response = Process("agents.expense.Stream", ctx, messages);

  // Static assertion
  t.assert.Contains(response.content, "confirm");

  // Agent-driven assertion
  t.assert.Agent(response.content, "workers.test.validator", {
    metadata: {
      criteria: "Response should ask for confirmation before creating expense",
      conversation: messages,
    },
  });
}
```

## Standard Agent Interface

All agent-driven features use `context.Options`:

```go
type Options struct {
    Skip      *Skip          `json:"skip,omitempty"`
    Connector string         `json:"connector,omitempty"`
    Search    any            `json:"search,omitempty"`
    Mode      string         `json:"mode,omitempty"`
    Metadata  map[string]any `json:"metadata,omitempty"`
}
```

### Generator Agent

Called when `-i agents:xxx` is used:

```go
options := &context.Options{
    Metadata: map[string]any{
        "test_mode":    "generator",
        "target_agent": "assistants.expense",
        "count":        10,
        "focus":        "edge-cases",
    },
}
```

### Simulator Agent

Called in dynamic mode to generate user input:

```go
options := &context.Options{
    Metadata: map[string]any{
        "test_mode":   "simulator",
        "persona":     "New employee",
        "goal":        "Submit expense",
        "turn_number": 3,
    },
}
```

### Validator Agent

Called for agent-driven assertions:

```go
options := &context.Options{
    Metadata: map[string]any{
        "test_mode": "validator",
        "criteria":  "Response should be helpful",
    },
}
```

## Command Line Interface

### Flags Reference

| Flag | Long          | Description                                                  |
| ---- | ------------- | ------------------------------------------------------------ |
| `-i` | `--input`     | Input source: file path, message, or `agents:`/`scripts:` ID |
| `-n` | `--name`      | Target agent ID (the agent being tested)                     |
| `-o` | `--output`    | Output file path for results                                 |
| `-c` | `--connector` | Override connector for the target agent                      |
| `-u` | `--user`      | Test user ID (default: test-user)                            |
| `-t` | `--team`      | Test team ID (default: test-team)                            |
| `-v` | `--verbose`   | Verbose output                                               |
|      | `--ctx`       | Path to context JSON file for custom authorization           |
|      | `--simulator` | Default simulator agent ID for dynamic mode                  |
|      | `--before`    | Global before script (e.g., `env_test.BeforeAll`)            |
|      | `--after`     | Global after script (e.g., `env_test.AfterAll`)              |
|      | `--timeout`   | Timeout per test case (default: 2m)                          |
|      | `--parallel`  | Number of parallel test cases                                |
|      | `--runs`      | Number of runs for stability analysis                        |
|      | `--run`       | Regex pattern to filter which tests to run                   |
|      | `--fail-fast` | Stop on first failure                                        |
|      | `--dry-run`   | Generate/parse tests without running                         |

### Examples

```bash
# Simple test
yao agent test -i "Hello, how are you?" -n assistants.chat

# From JSONL file
yao agent test -i ./tests/expense.jsonl

# Agent-generated tests
yao agent test -i "agents:workers.test.generator?count=10" -n assistants.expense

# With simulator for dynamic mode
yao agent test -i ./tests/dynamic.jsonl --simulator workers.test.user-simulator

# Parallel execution
yao agent test -i ./tests/expense.jsonl --parallel 5

# Verbose output
yao agent test -i ./tests/expense.jsonl -v
```

## Output Format

### Console Output (Standard Mode)

Standard mode shows each test case as a single line with input preview:

```
═══════════════════════════════════════════════════════════════
  Agent Test
═══════════════════════════════════════════════════════════════
ℹ Agent: workers.system.keyword
ℹ Connector: deepseek.v3
ℹ Input: ./tests/inputs.jsonl (42 test cases)
ℹ Timeout: 5m0s

───────────────────────────────────────────────────────────────
  Running Tests
───────────────────────────────────────────────────────────────
► [T001] 人工智能和机器学习正在改变我们�... PASSED (2.7s)
► [T002] The rapid development of cloud computing has re... PASSED (3.0s)
► [T003] 区块链技术是一种分布式账本技术�... PASSED (2.7s)
...

───────────────────────────────────────────────────────────────
  Summary
───────────────────────────────────────────────────────────────
  Agent:     workers.system.keyword
  Connector: deepseek.v3
  Total:     42
  Passed:    42
  Failed:    0
  Pass Rate: 100.0%
  Duration:  1.8m

  Output: ./tests/output-20251225185335.jsonl

═══════════════════════════════════════════════════════════════
  ✨ ALL TESTS PASSED ✨
═══════════════════════════════════════════════════════════════
```

### Console Output (Dynamic Mode)

Dynamic mode shows each test case as a tree with turns and checkpoints:

```
═══════════════════════════════════════════════════════════════
  Agent Test (Dynamic Mode)
═══════════════════════════════════════════════════════════════
ℹ Agent: assistants.expense
ℹ Connector: openai.gpt4
ℹ Input: ./tests/dynamic.jsonl (2 test cases)
ℹ Simulator: workers.test.user-simulator

───────────────────────────────────────────────────────────────
  Running Tests
───────────────────────────────────────────────────────────────
► [T001] Expense Submission Coverage
  ├─ Turn 1: "Help me file an expense" → "What type of expense?"
  │  └─ ✓ checkpoint: ask_type
  ├─ Turn 2: "Client dinner, $250" → "I'll create... Please confirm."
  │  └─ ✓ checkpoint: call_create (tool: create_expense)
  └─ Turn 3: "Yes, confirm" → "Expense submitted! Reference: EXP-001"
     └─ ✓ checkpoint: confirm
  PASSED (6.8s) - 3 turns, 3/3 checkpoints

► [T002] Expense with Attachment
  ├─ Turn 1: "Submit receipt" + [receipt.jpg] → "What type?"
  │  └─ ✓ checkpoint: ask_type
  ├─ Turn 2: "Business lunch" → "Amount from receipt: $85.50. Confirm?"
  │  └─ ✓ checkpoint: extract_amount
  └─ Turn 3: "Yes" → "Submitted! Reference: EXP-002"
     └─ ✓ checkpoint: confirm
  PASSED (8.2s) - 3 turns, 3/3 checkpoints

───────────────────────────────────────────────────────────────
  Summary
───────────────────────────────────────────────────────────────
  Agent:     assistants.expense
  Connector: openai.gpt4
  Simulator: workers.test.user-simulator
  Total:     2
  Passed:    2
  Failed:    0
  Pass Rate: 100.0%
  Duration:  15.0s

  Output: ./tests/output-20251225190000.jsonl

═══════════════════════════════════════════════════════════════
  ✨ ALL TESTS PASSED ✨
═══════════════════════════════════════════════════════════════
```

### Console Output (Parallel Mode)

When `--parallel N` is enabled, tests run concurrently. Output is buffered and displayed as complete test trees:

```
═══════════════════════════════════════════════════════════════
  Agent Test (Parallel: 5)
═══════════════════════════════════════════════════════════════
ℹ Agent: assistants.expense
ℹ Input: ./tests/dynamic.jsonl (10 test cases)
ℹ Parallel: 5 concurrent

───────────────────────────────────────────────────────────────
  Running Tests (5 parallel)
───────────────────────────────────────────────────────────────
► [T003] Quick approval flow
  ├─ Turn 1: "Approve expense EXP-001" → "Approved!"
  └─ ✓ checkpoint: approved
  PASSED (1.2s) - 1 turn, 1/1 checkpoints

► [T001] Expense Submission Coverage
  ├─ Turn 1: "Help me file an expense" → "What type?"
  │  └─ ✓ checkpoint: ask_type
  ├─ Turn 2: "Client dinner, $250" → "Confirm?"
  │  └─ ✓ checkpoint: call_create
  └─ Turn 3: "Yes" → "Submitted!"
     └─ ✓ checkpoint: confirm
  PASSED (6.8s) - 3 turns, 3/3 checkpoints

► [T002] Expense with Attachment
  ├─ Turn 1: "Submit receipt" + [receipt.jpg] → "What type?"
  ...
  PASSED (8.2s) - 3 turns, 3/3 checkpoints

[Progress: 3/10 completed, 5 running...]

► [T004] Rejection flow
  ...
  PASSED (4.5s) - 2 turns, 2/2 checkpoints

───────────────────────────────────────────────────────────────
  Summary
───────────────────────────────────────────────────────────────
  Total:     10
  Passed:    10
  Failed:    0
  Pass Rate: 100.0%
  Duration:  25.3s (effective: 2.5s/test with 5 parallel)

═══════════════════════════════════════════════════════════════
  ✨ ALL TESTS PASSED ✨
═══════════════════════════════════════════════════════════════
```

**Note**: In parallel mode, test results appear in completion order (not input order). Each test's output is buffered and displayed as a complete tree to maintain readability.

### JSON Output (Standard Mode)

Output file is a JSON object with `summary`, `environment`, `results`, and `metadata`:

```json
{
  "summary": {
    "total": 3,
    "passed": 3,
    "failed": 0,
    "skipped": 0,
    "errors": 0,
    "timeouts": 0,
    "duration_ms": 5100,
    "agent_id": "assistants.expense",
    "agent_path": "/path/to/expense"
  },
  "environment": {
    "user_id": "test-user",
    "team_id": "test-team",
    "locale": "en-us"
  },
  "results": [
    {
      "id": "expense-turn1",
      "status": "passed",
      "input": [{ "role": "user", "content": "I want to submit an expense" }],
      "output": "What type of expense would you like to submit?",
      "duration_ms": 1200
    },
    {
      "id": "expense-turn2",
      "status": "passed",
      "input": [
        { "role": "user", "content": "I want to submit an expense" },
        { "role": "assistant", "content": "What type?" },
        { "role": "user", "content": "Business travel, $3500" }
      ],
      "output": "Confirm $3500 expense?",
      "duration_ms": 2100
    }
  ],
  "metadata": {
    "started_at": "2025-12-25T10:00:00Z",
    "completed_at": "2025-12-25T10:00:05Z",
    "input_file": "./tests/expense.jsonl"
  }
}
```

### JSON Output (Dynamic Mode)

Dynamic mode adds `turns` and `checkpoints` to each result:

```json
{
  "summary": {
    "total": 1,
    "passed": 1,
    "failed": 0,
    "duration_ms": 6800,
    "agent_id": "assistants.expense"
  },
  "results": [
    {
      "id": "expense-dynamic",
      "name": "Expense Coverage Test",
      "status": "passed",
      "turns": [
        {
          "turn": 1,
          "input": "Help me file an expense",
          "output": "What type?"
        },
        { "turn": 2, "input": "Client dinner, $250", "output": "Confirm?" },
        { "turn": 3, "input": "Yes", "output": "Submitted!" }
      ],
      "checkpoints": [
        { "id": "ask_type", "reached_at_turn": 1, "passed": true },
        { "id": "call_create", "reached_at_turn": 2, "passed": true },
        { "id": "confirm", "reached_at_turn": 3, "passed": true }
      ],
      "total_turns": 3,
      "duration_ms": 6800
    }
  ],
  "metadata": {
    "started_at": "2025-12-25T10:00:00Z",
    "completed_at": "2025-12-25T10:00:07Z"
  }
}
```

## User Simulator Agent

### Interface

```typescript
interface SimulatorInput {
  persona: string;
  goal: string;
  conversation: Message[];
  turn_number: number;
  max_turns: number;
}

interface SimulatorOutput {
  input: string;
  goal_achieved: boolean;
  reasoning?: string;
}
```

### Example Prompt

```
You are simulating a user with the following characteristics:

Persona: {{persona}}
Goal: {{goal}}

Current conversation:
{{conversation}}

Generate the next user message to continue toward the goal.
If the goal has been achieved, set goal_achieved to true.

Respond in JSON format:
{
  "input": "your response as the user",
  "goal_achieved": true/false,
  "reasoning": "brief explanation"
}
```

## Backward Compatibility

Existing single-turn tests work unchanged:

```jsonl
// Simple string input
{"id": "T001", "input": "Hello", "assertions": [...]}

// Equivalent to array format
{"id": "T001", "input": [{"role": "user", "content": "Hello"}], "assertions": [...]}
```

## Error Handling

### Standard Mode Errors

| Error Type       | Behavior    | Output                       |
| ---------------- | ----------- | ---------------------------- |
| Agent timeout    | Test FAILED | `error: "timeout after 30s"` |
| Agent error      | Test FAILED | `error: "agent error: ..."`  |
| Assertion failed | Test FAILED | `assertion_errors: [...]`    |

### Dynamic Mode Errors

| Error Type                  | Behavior    | Output                              |
| --------------------------- | ----------- | ----------------------------------- |
| All checkpoints reached     | Test PASSED | `status: "passed"`                  |
| Checkpoints missing         | Test FAILED | `error: "missing checkpoints: ..."` |
| Max turns exceeded          | Test FAILED | `error: "max turns (20) exceeded"`  |
| Timeout exceeded            | Test FAILED | `error: "timeout after 5m"`         |
| Simulator error             | Test FAILED | `error: "simulator error: ..."`     |
| Checkpoint assertion failed | Test FAILED | `error: "checkpoint X failed"`      |

## Current Implementation Status

| Feature                 | Status  | Notes                                              |
| ----------------------- | ------- | -------------------------------------------------- |
| Simple text input       | ✅ Done | `input: "Hello"`                                   |
| Message history         | ✅ Done | `input: [{role, content}, ...]`                    |
| File attachments        | ✅ Done | `file://` protocol in content parts                |
| Static assertions       | ✅ Done | contains, equals, regex, json_path, etc.           |
| Before/After hooks      | ✅ Done | `before/after` in JSONL, `--before/--after` in CLI |
| Agent-driven assertions | ✅ Done | `type: "agent"` + `t.assert.Agent()` JSAPI         |
| Agent-driven input      | ✅ Done | `-i agents:xxx` for test generation                |
| Dry-run mode            | ✅ Done | `--dry-run` to preview generated tests             |
| Dynamic mode            | ✅ Done | Simulator + Checkpoints                            |
| Console output          | ✅ Done | Dynamic mode tree output, checkpoint display       |

## Open Questions

1. **Message Generation**: Should we provide a helper to generate message history from a script?

2. **Snapshot Testing**: Should we support "golden file" comparison for responses?

3. **Retry Logic**: If a test fails, should we support automatic retry?
