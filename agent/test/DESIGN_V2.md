# Agent Test Framework V2 Design

## Overview

This document describes the design for Agent Test Framework V2, which extends the existing testing capabilities with:

- **Message history support** - Test agents with conversation context via `messages[]`
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
│  │  Standard Mode:     {input: "...", messages: [...], assertions}   │  │
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
│  │  - JSONL output                                                    │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Test Modes

### Standard Mode (Default)

Single call to agent with optional message history. **No multi-turn state management needed.**

| Field        | Type   | Description                                      |
| ------------ | ------ | ------------------------------------------------ |
| `input`      | string | Simple text input (shorthand for single message) |
| `messages`   | array  | Full message history (overrides `input`)         |
| `assertions` | array  | Assertions to validate response                  |
| `options`    | object | `context.Options` passed to agent                |

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

### With Message History (New)

Test agent with conversation context - simulates multi-turn without complex state:

```jsonl
{
  "id": "T002",
  "name": "Expense submission - final confirmation",
  "messages": [
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
  "messages": [
    {"role": "user", "content": "I want to submit an expense"}
  ],
  "assertions": [{"type": "contains", "value": "type"}]
}

// Test 2: Second turn - agent should create expense
{
  "id": "expense-turn2",
  "messages": [
    {"role": "user", "content": "I want to submit an expense"},
    {"role": "assistant", "content": "What type of expense would you like to submit?"},
    {"role": "user", "content": "Business travel, $3500"}
  ],
  "assertions": [{"type": "tool_called", "name": "create_expense"}]
}

// Test 3: Final turn - agent should confirm submission
{
  "id": "expense-turn3",
  "messages": [
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
  "messages": [
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

| Field        | Type   | Required | Description                       |
| ------------ | ------ | -------- | --------------------------------- |
| `id`         | string | Yes      | Unique test identifier            |
| `name`       | string | No       | Human-readable test name          |
| `input`      | string | No\*     | Simple text input                 |
| `messages`   | array  | No\*     | Full message history              |
| `assertions` | array  | No       | Assertions to validate response   |
| `options`    | object | No       | `context.Options` passed to agent |

\*Either `input` or `messages` required

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

## Execution Flow

### Standard Mode

```
┌─────────────────────────────────────────────────────────────────┐
│                    Standard Mode Execution                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Parse test case                                              │
│     ├─ Has `messages`? → Use as-is                               │
│     └─ Has `input`? → Convert to [{role: "user", content: input}]│
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

| Flag | Long          | Description                                              |
| ---- | ------------- | -------------------------------------------------------- |
| `-i` | `--input`     | Input source: file path, message, or `type:id` reference |
| `-n` | `--name`      | Target agent ID (the agent being tested)                 |
| `-o` | `--output`    | Output file path for results                             |
| `-c` | `--connector` | Override connector for the target agent                  |
| `-v` | `--verbose`   | Verbose output                                           |
|      | `--simulator` | Default simulator agent ID                               |
|      | `--timeout`   | Timeout per test case (default: 5m)                      |
|      | `--parallel`  | Number of parallel test cases                            |
|      | `--fail-fast` | Stop on first failure                                    |
|      | `--dry-run`   | Generate/parse tests without running                     |

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

### Console Output

```
═══════════════════════════════════════════════════════════════
  Agent Test
═══════════════════════════════════════════════════════════════
ℹ Agent: assistants.expense
ℹ Input: ./tests/expense.jsonl (3 test cases)

───────────────────────────────────────────────────────────────
  Running Tests
───────────────────────────────────────────────────────────────

✓ [expense-turn1] First turn - ask type (1.2s)
  Messages: 1, Assertions: 1/1 passed

✓ [expense-turn2] Second turn - create expense (2.1s)
  Messages: 3, Assertions: 1/1 passed

✓ [expense-turn3] Final turn - confirm (1.8s)
  Messages: 5, Assertions: 1/1 passed

───────────────────────────────────────────────────────────────
  Summary
───────────────────────────────────────────────────────────────
  Total:   3 tests
  Passed:  3
  Failed:  0
  Time:    5.1s
```

### JSONL Output

```jsonl
{
  "id": "expense-turn3",
  "name": "Final turn - confirm",
  "status": "passed",
  "messages_count": 5,
  "response": "Expense submitted. Reference: EXP-2025-001",
  "assertions": [
    {
      "type": "contains",
      "value": "submitted",
      "passed": true
    }
  ],
  "duration_ms": 1800
}
```

## Dynamic Mode Output

```jsonl
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
    {
      "turn": 2,
      "input": "Client dinner, $250",
      "output": "Confirm?"
    },
    {
      "turn": 3,
      "input": "Yes",
      "output": "Submitted!"
    }
  ],
  "checkpoints": [
    {
      "id": "ask_type",
      "reached_at_turn": 1,
      "passed": true
    },
    {
      "id": "call_create",
      "reached_at_turn": 2,
      "passed": true
    },
    {
      "id": "confirm",
      "reached_at_turn": 3,
      "passed": true
    }
  ],
  "total_turns": 3,
  "duration_ms": 6800
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
// This still works
{"id": "T001", "input": "Hello", "assertions": [...]}

// Equivalent to
{"id": "T001", "messages": [{"role": "user", "content": "Hello"}], "assertions": [...]}
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

## Comparison: Old vs New Design

| Aspect             | Old (Static Mode)         | New (Messages)              |
| ------------------ | ------------------------- | --------------------------- |
| Multi-turn testing | Sequential turn execution | Pass message history        |
| State management   | Session state per test    | Stateless                   |
| Parallelization    | Sequential within test    | Fully parallel              |
| Implementation     | Complex turn loop         | Single agent call           |
| Debugging          | Need to trace turns       | Clear input/output per test |
| Flexibility        | Coupled turns             | Independent tests           |

## Open Questions

1. **Message Generation**: Should we provide a helper to generate message history from a script?

2. **Snapshot Testing**: Should we support "golden file" comparison for responses?

3. **Retry Logic**: If a test fails, should we support automatic retry?
