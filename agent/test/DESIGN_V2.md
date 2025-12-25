# Agent Test Framework V2 Design

## Overview

This document describes the design for Agent Test Framework V2, which extends the existing testing capabilities with:

- **Multi-turn conversations** - Test agents across multiple interaction rounds
- **Agent-driven testing** - Use agents to generate test cases and simulate user responses
- **Interactive testing** - Human-in-the-loop testing mode

## Quick Reference: Format Rules

| Context               | Format                   | Example                                                 |
| --------------------- | ------------------------ | ------------------------------------------------------- |
| `-i` flag (CLI)       | Prefix required          | `agents:workers.test.gen`, `scripts:tests.gen`          |
| JSONL assertion `use` | Prefix required          | `"use": "agents:workers.test.validator"`                |
| JSONL `simulator.use` | No prefix (agent only)   | `"use": "workers.test.user-sim"`                        |
| `--simulator` flag    | No prefix (agent only)   | `--simulator workers.test.user-sim`                     |
| `t.assert.Agent()`    | No prefix (method-bound) | `t.assert.Agent(resp, "workers.test.validator", {...})` |

## Problem Statement

Current single-turn testing cannot adequately test:

1. **Conversational flows** - Agents that guide users through multi-step processes
2. **Confirmation dialogs** - Agents that ask for user confirmation before actions
3. **Clarification requests** - Agents that ask follow-up questions when input is ambiguous
4. **Stateful interactions** - Agents that maintain context across multiple turns

## Design Goals

1. **Unified format** - Single test case format that supports all modes
2. **Flexible execution** - Static, dynamic (simulator), and interactive modes
3. **Graceful degradation** - Skip tests when required input is unavailable
4. **CI/CD compatible** - Non-interactive mode for automated pipelines
5. **Agent-driven** - Both input generation and user simulation can be agent-powered

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         yao agent test                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  INPUT SOURCES (-i flag)                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ JSONL File  │  │   Message   │  │ Generator   │  │ Interactive │     │
│  │ ./test.jsonl│  │ "Hello..."  │  │ agents:xxx  │  │ (stdin)     │     │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│         │                │                │                │            │
│         └────────────────┴────────────────┴────────────────┘            │
│                                   │                                      │
│                                   ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                        Test Case Parser                          │    │
│  │  - Single-turn: {input, assertions}                              │    │
│  │  - Multi-turn:  {turns: [{input, assertions}, ...]}              │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                   │                                      │
│                                   ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      Multi-Turn Executor                         │    │
│  │                                                                   │    │
│  │   ┌─────────────────────────────────────────────────────────┐    │    │
│  │   │ MODE SELECTION (based on test case fields)              │    │    │
│  │   │                                                          │    │    │
│  │   │   Has `turns`? ─────────────────▶ STATIC MODE            │    │    │
│  │   │   Has `simulator` + `checkpoints`? ──▶ DYNAMIC MODE      │    │    │
│  │   │   Neither? ─────────────────────▶ SINGLE-TURN (legacy)   │    │    │
│  │   └─────────────────────────────────────────────────────────┘    │    │
│  │                              │                                   │    │
│  │              ┌───────────────┴───────────────┐                   │    │
│  │              ▼                               ▼                   │    │
│  │   ┌─────────────────────┐       ┌─────────────────────────┐      │    │
│  │   │    STATIC MODE      │       │     DYNAMIC MODE        │      │    │
│  │   │                     │       │                         │      │    │
│  │   │ FOR each turn:      │       │ LOOP until terminated:  │      │    │
│  │   │  1. Send input      │       │  1. Simulator → input   │      │    │
│  │   │  2. Get response    │       │  2. Send to Agent       │      │    │
│  │   │  3. Run assertions  │       │  3. Check checkpoints   │      │    │
│  │   │  4. Continue/Fail   │       │  4. Check termination   │      │    │
│  │   │                     │       │                         │      │    │
│  │   │ All passed → PASS   │       │ All checkpoints → PASS  │      │    │
│  │   │ Any failed → FAIL   │       │ Timeout/Missing → FAIL  │      │    │
│  │   └─────────────────────┘       └─────────────────────────┘      │    │
│  │                                                                   │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                   │                                      │
│                                   ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                         Assertions                               │    │
│  │  - Static: Per-turn assertions                                   │    │
│  │  - Dynamic: Checkpoint assertions (order-independent)            │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                   │                                      │
│                                   ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                          Reporter                                │    │
│  │  - Console output                                                │    │
│  │  - JSONL output                                                  │    │
│  │  - Custom reporter agent                                         │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Core Challenge: Detecting "Awaiting Input" State

The key challenge is determining when an agent is waiting for user input vs. when it has completed its task.

### Detection Strategies

#### Strategy 1: Explicit Declaration (Recommended)

Agent explicitly declares its state in the response:

```json
{
  "content": "What is the expense amount?",
  "awaiting_input": true,
  "input_hint": "Enter amount, e.g., $3500"
}
```

#### Strategy 2: Finish Reason Analysis

Use the LLM's `finish_reason` to determine state:

- `stop` - Agent completed normally (may or may not need input)
- `tool_calls` - Agent is executing tools (not awaiting input)
- `length` - Response truncated (not awaiting input)

#### Strategy 3: Content Heuristics

Analyze response content for question patterns:

```go
func looksLikeQuestion(content string) bool {
    patterns := []string{
        `\?$`,                    // Ends with question mark
        `(?i)^(what|how|when|where|which|who|please|could you)`,
        `(?i)(confirm|verify|proceed|continue)\?`,
    }
    for _, pattern := range patterns {
        if regexp.MustCompile(pattern).MatchString(content) {
            return true
        }
    }
    return false
}
```

#### Strategy 4: Tool-Based Detection

Certain tools indicate awaiting input:

```go
func toolRequiresInput(toolCall ToolCall) bool {
    confirmationTools := []string{
        "request_confirmation",
        "ask_user",
        "get_user_input",
    }
    return contains(confirmationTools, toolCall.Name)
}
```

### Recommended Approach: Hybrid Detection

Combine multiple strategies with priority:

```go
func IsAwaitingInput(result *TurnResult) (awaiting bool, reason string) {
    // Priority 1: Explicit declaration
    if result.AwaitingInput {
        return true, "agent_declared"
    }

    // Priority 2: Tool-based detection
    for _, tc := range result.ToolCalls {
        if toolRequiresInput(tc) {
            return true, "tool_requires_confirmation"
        }
    }

    // Priority 3: Content heuristics
    if looksLikeQuestion(result.Content) {
        return true, "content_is_question"
    }

    return false, "completed"
}
```

## Standard Agent Interface

All agent-driven features (generator, simulator, validator) use standard Yao Agent interfaces.

### Using `context.Options`

The test framework uses `context.Options` to pass parameters, aligned with `Assistant.Stream()`:

```go
// context.Options - standard Yao Agent options
type Options struct {
    Skip      *Skip          `json:"skip,omitempty"`      // Skip history, trace, etc.
    Connector string         `json:"connector,omitempty"` // LLM connector to use
    Search    any            `json:"search,omitempty"`    // Search behavior control
    Mode      string         `json:"mode,omitempty"`      // Agent mode (chat, etc.)
    Metadata  map[string]any `json:"metadata,omitempty"`  // Custom metadata
}
```

### Test Framework Usage

```go
// Prepare options for agent invocation
options := &context.Options{
    Skip: &context.Skip{
        History: true,  // Don't save test messages to history
        Trace:   false, // Keep trace for debugging
    },
    Metadata: map[string]any{
        "test_mode": "validator",  // "generator" | "simulator" | "validator"
        "test_id":   "T001",
        "criteria":  "Response should be helpful",
        // ... other custom params from test config
    },
}

// Prepare context
ctx := context.New(parent, authorized, chatID)
ctx.Referer = "agent-test"

// Call agent with standard Stream API
assistant := agent.Get(agentID)
response, err := assistant.Stream(ctx, messages, options)
```

### Test Case Options Field

Test cases can specify options to pass to the target agent or helper agents:

```jsonl
{
  "id": "T001",
  "input": "Hello",
  "options": {
    "connector": "openai-gpt4",
    "skip": {
      "history": true
    },
    "metadata": {
      "scenario": "edge-case"
    }
  }
}
```

### Generator Mode

Used when `-i agents:xxx` is specified to generate test cases:

```go
// Framework calls generator agent
options := &context.Options{
    Skip: &context.Skip{History: true},
    Metadata: map[string]any{
        "test_mode":          "generator",
        "target_agent":       "assistants.expense",
        "target_description": "Expense reimbursement assistant",
        "target_tools":       []string{"create_expense", "get_policy"},
        // From query params: agents:xxx?count=10&focus=edge-cases
        "count":      10,
        "focus":      "edge-cases",
        "complexity": "medium",
    },
}
```

Expected structured output:

```json
{
  "cases": [
    {"id": "G001", "input": "...", "assertions": [...]},
    {"id": "G002", "input": "...", "assertions": [...]}
  ]
}
```

### Simulator Mode

Used when test case has `simulator` config:

```go
// Framework calls simulator agent for next user input
options := &context.Options{
    Skip: &context.Skip{History: true},
    Metadata: map[string]any{
        "test_mode":    "simulator",
        "test_id":      "T001",
        // From simulator.options.metadata in test case
        "persona":      "New employee",
        "goal":         "Submit expense report",
        // Runtime context
        "turn_number":  3,
        "max_turns":    10,
        "tool_results": map[string]any{...},
    },
}

// Messages include full conversation history
messages := conversationHistory
```

Expected structured output:

```json
{
  "input": "The amount is $500",
  "goal_achieved": false,
  "reasoning": "Providing requested amount info"
}
```

### Validator Mode

Used when assertion has `type: "agent"`:

```go
// Framework calls validator agent
options := &context.Options{
    Skip: &context.Skip{History: true},
    Metadata: map[string]any{
        "test_mode": "validator",
        "test_id":   "T001",
        // From assertion.metadata
        "criteria":        "Response should be helpful",
        "expected_intent": "answer question",
        "tone":            "professional",
    },
}

// Messages include agent response to validate
messages := []context.Message{
    {Role: "user", Content: "Original user question"},
    {Role: "assistant", Content: "Agent's response to validate"},
}
```

Expected structured output:

```json
{
  "passed": true,
  "score": 0.95,
  "reason": "Response is helpful and addresses the question",
  "suggestions": []
}
```

## Assertion Types

### Static Assertions (Existing)

| Type          | Description            | Example                                                    |
| ------------- | ---------------------- | ---------------------------------------------------------- |
| `contains`    | Response contains text | `{"type": "contains", "value": "success"}`                 |
| `equals`      | Exact match            | `{"type": "equals", "value": "OK"}`                        |
| `regex`       | Regex pattern match    | `{"type": "regex", "pattern": "order-\\d+"}`               |
| `json_path`   | JSONPath value check   | `{"type": "json_path", "path": "$.status", "value": "ok"}` |
| `tool_called` | Tool was invoked       | `{"type": "tool_called", "name": "create_expense"}`        |
| `type`        | Value type check       | `{"type": "type", "path": "$.count", "value": "number"}`   |

### Agent-Driven Assertions (New)

For fuzzy, semantic, or context-aware validation. Uses `options` aligned with `context.Options`:

```jsonl
{
  "type": "agent",
  "use": "agents:workers.test.validator",
  "options": {
    "connector": "openai-gpt4",
    "metadata": {
      "criteria": "Response should be helpful and answer the user's question",
      "expected_intent": "provide expense submission guidance",
      "tone": "professional and friendly"
    }
  }
}
```

### Script Assertions (in JSONL)

For custom validation logic in JSONL test cases:

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

### Combined Assertions

Mix static and agent-driven assertions:

```jsonl
{
  "assertions": [
    {
      "type": "tool_called",
      "name": "create_expense"
    },
    {
      "type": "agent",
      "use": "agents:workers.test.validator",
      "options": {
        "metadata": {
          "criteria": "Confirmation message should include expense amount and be polite"
        }
      }
    }
  ]
}
```

## Script Testing with Agent Assertions

Script tests can also use Agent-driven assertions via the `t.assert.Agent()` method.

### API

```typescript
// t.assert.Agent(response, agentID, options?) -> ValidatorResult
// agentID: Direct agent ID without prefix, e.g., "workers.test.validator"
interface ValidatorResult {
  passed: boolean;
  score?: number;
  reason: string;
  suggestions?: string[];
}
```

### Usage in Script Tests

```typescript
// tests/expense_test.ts
export function TestExpenseResponse(t: TestingT, ctx: Context) {
  // Call the agent being tested
  const response = Process("agents.expense.Stream", ctx, [
    { role: "user", content: "How do I submit an expense?" },
  ]);

  // Static assertions
  t.assert.NotNil(response);
  t.assert.Contains(response.content, "expense");

  // Agent-driven assertion - automatically fails test if validation fails
  t.assert.Agent(response.content, "workers.test.validator", {
    metadata: {
      criteria:
        "Response should explain the expense submission process clearly",
      expected_topics: ["receipt", "approval", "deadline"],
      tone: "helpful",
    },
  });
}
```

### With Conversation Context

```typescript
export function TestMultiTurnExpense(t: TestingT, ctx: Context) {
  const messages = [
    { role: "user", content: "I need to submit a travel expense" },
    { role: "assistant", content: "I'd be happy to help..." },
    { role: "user", content: "It's for a flight to Beijing, $2000" },
  ];

  const response = Process("agents.expense.Stream", ctx, messages);

  // Agent assertion with conversation context
  // Automatically fails test and logs suggestions if validation fails
  t.assert.Agent(response.content, "workers.test.validator", {
    metadata: {
      criteria:
        "Response should confirm the expense details and ask for receipt",
      conversation: messages,
    },
  });
}
```

### Implementation

The `t.assert.Agent()` method internally:

1. Prepares `context.Options` with `test_mode: "validator"`
2. Calls the validator agent via `Assistant.Stream()`
3. Parses structured output (JSON)
4. Returns `ValidatorResult`

```go
// In script_assert.go
func assertAgentMethod(iso *v8go.Isolate, t *TestingT, agentCtx *context.Context) *v8go.FunctionTemplate {
    return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
        // Parse arguments: response, agentID, options
        response := args[0].String()
        agentID := args[1].String()  // Direct agent ID, e.g., "workers.test.validator"
        options := parseOptions(args[2])

        // Prepare validator options
        validatorOpts := &context.Options{
            Skip: &context.Skip{History: true},
            Metadata: map[string]any{
                "test_mode": "validator",
                ...options.Metadata,
            },
        }

        // Call validator agent
        assistant, _ := agent.Get(agentID)
        result, _ := assistant.Stream(agentCtx, messages, validatorOpts)

        // Parse and return result
        return toJsValue(parseValidatorResult(result))
    })
}
```

## Test Case Format

### Single-Turn (Existing)

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

### Multi-Turn: Static Mode

For **deterministic flows** where you know the exact conversation sequence:

```jsonl
{
  "id": "T001",
  "name": "Expense Reimbursement - Happy Path",
  "mode": "static",
  "options": {
    "connector": "openai-gpt4",
    "skip": {
      "history": true
    }
  },
  "turns": [
    {
      "input": "I want to submit an expense report",
      "assertions": [
        {
          "type": "contains",
          "value": "type of expense"
        }
      ]
    },
    {
      "input": "Business travel to Beijing, $3500",
      "assertions": [
        {
          "type": "tool_called",
          "name": "create_expense"
        }
      ]
    },
    {
      "input": "Yes, confirm",
      "assertions": [
        {
          "type": "contains",
          "value": "submitted"
        }
      ]
    }
  ]
}
```

**Characteristics:**

- Fixed number of turns
- Each turn has specific input and assertions
- Test fails if any turn assertion fails
- Best for regression testing known flows

### Multi-Turn: Dynamic Mode (Checkpoints)

For **coverage testing** where you care about functionality, not exact sequence:

```jsonl
{
  "id": "T002",
  "name": "Expense Submission Coverage",
  "mode": "dynamic",
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
      "description": "Agent calls create_expense tool",
      "assertion": {
        "type": "tool_called",
        "name": "create_expense"
      }
    },
    {
      "id": "confirm_submit",
      "description": "Agent confirms submission",
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

**Characteristics:**

- Simulator drives the conversation
- Checkpoints are verified across all turns (order-independent by default)
- Test passes when ALL checkpoints are reached
- Test fails if max_turns/timeout reached before all checkpoints
- Best for functional coverage testing

### Checkpoints with Order Constraints

When checkpoints must occur in a specific order:

```jsonl
{
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
      "id": "confirm_submit",
      "description": "Agent confirms submission",
      "after": [
        "call_create"
      ],
      "assertion": {
        "type": "contains",
        "value": "submitted"
      }
    }
  ]
}
```

### Checkpoints with Agent Validation

Use Agent-driven assertions for semantic validation:

```jsonl
{
  "checkpoints": [
    {
      "id": "helpful_guidance",
      "description": "Agent provides helpful expense guidance",
      "assertion": {
        "type": "agent",
        "use": "agents:workers.test.validator",
        "options": {
          "metadata": {
            "criteria": "Response explains expense process clearly and professionally"
          }
        }
      }
    },
    {
      "id": "tool_called",
      "description": "Agent creates expense record",
      "assertion": {
        "type": "tool_called",
        "name": "create_expense"
      }
    }
  ]
}
```

### Dynamic Mode Termination

| Condition                            | Result     | Description                |
| ------------------------------------ | ---------- | -------------------------- |
| All checkpoints reached              | ✅ PASSED  | All functionality verified |
| Agent completes, checkpoints missing | ❌ FAILED  | Missing coverage           |
| max_turns exceeded                   | ❌ FAILED  | Timeout - flow too long    |
| timeout exceeded                     | ❌ FAILED  | Time limit reached         |
| Checkpoint assertion fails           | ❌ FAILED  | Functionality broken       |
| Simulator error                      | ⚠️ SKIPPED | Cannot continue            |

### Field Descriptions

| Field                       | Type   | Required      | Description                                |
| --------------------------- | ------ | ------------- | ------------------------------------------ |
| `id`                        | string | Yes           | Unique test identifier                     |
| `name`                      | string | No            | Human-readable test name                   |
| `mode`                      | string | No            | `"static"` (default) or `"dynamic"`        |
| `options`                   | object | No            | `context.Options` passed to target agent   |
| **Static Mode Fields**      |
| `turns`                     | array  | Yes (static)  | Static turn definitions                    |
| `turns[].input`             | string | Yes           | User input for this turn                   |
| `turns[].assertions`        | array  | No            | Assertions for this turn's response        |
| `turns[].options`           | object | No            | Per-turn options override                  |
| **Dynamic Mode Fields**     |
| `simulator`                 | object | Yes (dynamic) | User simulator configuration               |
| `simulator.use`             | string | Yes           | Simulator agent ID (no prefix)             |
| `simulator.options`         | object | No            | `context.Options` passed to simulator      |
| `checkpoints`               | array  | Yes (dynamic) | Functionality checkpoints to verify        |
| `checkpoints[].id`          | string | Yes           | Unique checkpoint identifier               |
| `checkpoints[].description` | string | No            | Human-readable description                 |
| `checkpoints[].assertion`   | object | Yes           | Assertion to verify                        |
| `checkpoints[].after`       | array  | No            | Checkpoint IDs that must occur first       |
| `max_turns`                 | int    | No            | Maximum turns before timeout (default: 20) |
| `timeout`                   | string | No            | Maximum time (default: "5m")               |
| **Shared Fields**           |
| `interactive`               | object | No            | Interactive mode configuration             |
| `interactive.enabled`       | bool   | No            | Enable human input (default: false)        |
| `interactive.timeout`       | string | No            | Timeout for human input (default: "5m")    |

## Execution Modes

### Static Mode

Uses predefined `turns` array. Best for **regression testing** known flows.

```
┌─────────────────────────────────────────────────────────┐
│                    Static Mode Flow                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  FOR each turn in turns[]:                               │
│    1. Send turn.input to Agent                           │
│    2. Get Agent response                                 │
│    3. Run turn.assertions                                │
│       ├─ PASS → Continue to next turn                    │
│       └─ FAIL → Test FAILED, stop                        │
│                                                          │
│  All turns completed → Test PASSED                       │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Dynamic Mode (Checkpoints)

Uses simulator + checkpoints. Best for **coverage testing** functionality.

```
┌─────────────────────────────────────────────────────────┐
│                   Dynamic Mode Flow                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Initialize: pending_checkpoints = all checkpoints       │
│                                                          │
│  LOOP (until terminated):                                │
│    1. Simulator generates user input                     │
│    2. Send input to Agent                                │
│    3. Get Agent response                                 │
│    4. Check response against pending_checkpoints         │
│       └─ If matched → Move to reached_checkpoints        │
│    5. Check termination conditions:                      │
│       ├─ All checkpoints reached → PASSED                │
│       ├─ Agent completed, missing checkpoints → FAILED   │
│       ├─ max_turns exceeded → FAILED                     │
│       └─ timeout exceeded → FAILED                       │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Interactive Mode

For debugging, human can provide input when agent awaits:

```bash
# Enable with --interactive flag
yao agent test -i ./tests.jsonl --interactive
```

In interactive mode:

- Static mode: Human can override any turn input
- Dynamic mode: Human can replace simulator for specific turns

### Mode Selection

| Has `turns`? | Has `simulator` + `checkpoints`? | Mode                    |
| ------------ | -------------------------------- | ----------------------- |
| Yes          | No                               | Static                  |
| No           | Yes                              | Dynamic                 |
| Yes          | Yes                              | ❌ Invalid (choose one) |
| No           | No                               | Single-turn (legacy)    |

### Example: Static vs Dynamic

**Same feature, different testing approaches:**

```jsonl
// Static Mode - Exact sequence testing
{
  "id": "expense-static",
  "mode": "static",
  "turns": [
    {"input": "Submit expense", "assertions": [{"type": "contains", "value": "type"}]},
    {"input": "Travel, $500", "assertions": [{"type": "tool_called", "name": "create_expense"}]},
    {"input": "Confirm", "assertions": [{"type": "contains", "value": "submitted"}]}
  ]
}

// Dynamic Mode - Coverage testing
{
  "id": "expense-dynamic",
  "mode": "dynamic",
  "simulator": {"use": "workers.test.user-sim", "options": {"metadata": {"goal": "Submit $500 expense"}}},
  "checkpoints": [
    {"id": "ask", "assertion": {"type": "contains", "value": "type"}},
    {"id": "create", "assertion": {"type": "tool_called", "name": "create_expense"}},
    {"id": "done", "assertion": {"type": "contains", "value": "submitted"}}
  ],
  "max_turns": 10
}
```

## Execution Flow

### Static Mode Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Static Mode Execution                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  INITIALIZE:                                                     │
│    - Load turns[] from test case                                 │
│    - Set current_turn = 0                                        │
│                              ↓                                   │
│  FOR each turn in turns[]:                                       │
│    ┌─────────────────────────────────────────────────────────┐   │
│    │ 1. Get input from turns[current_turn].input             │   │
│    │                         ↓                               │   │
│    │ 2. Send input to Agent                                  │   │
│    │                         ↓                               │   │
│    │ 3. Get Agent response                                   │   │
│    │                         ↓                               │   │
│    │ 4. Run turns[current_turn].assertions                   │   │
│    │           │                                             │   │
│    │           ├─ PASS → Continue to next turn               │   │
│    │           └─ FAIL → Test FAILED, stop                   │   │
│    └─────────────────────────────────────────────────────────┘   │
│                              ↓                                   │
│  All turns completed → Test PASSED                               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Dynamic Mode Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Dynamic Mode Execution                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  INITIALIZE:                                                     │
│    - pending_checkpoints = all checkpoints                       │
│    - reached_checkpoints = []                                    │
│    - turn_count = 0                                              │
│    - start_time = now()                                          │
│                              ↓                                   │
│  LOOP:                                                           │
│    ┌─────────────────────────────────────────────────────────┐   │
│    │ 1. Call Simulator Agent → Get user input                │   │
│    │    (pass: persona, goal, conversation history)          │   │
│    │                         ↓                               │   │
│    │ 2. Send input to Target Agent                           │   │
│    │                         ↓                               │   │
│    │ 3. Get Agent response                                   │   │
│    │                         ↓                               │   │
│    │ 4. Check response against pending_checkpoints           │   │
│    │    FOR each pending checkpoint:                         │   │
│    │      - Run checkpoint.assertion                         │   │
│    │      - If PASS and `after` satisfied → move to reached  │   │
│    │                         ↓                               │   │
│    │ 5. Check termination conditions:                        │   │
│    │    ├─ pending_checkpoints empty?                        │   │
│    │    │   → Test PASSED ✅                                 │   │
│    │    │                                                    │   │
│    │    ├─ Agent completed (not awaiting)?                   │   │
│    │    │   → Test FAILED ❌ (missing checkpoints)           │   │
│    │    │                                                    │   │
│    │    ├─ turn_count >= max_turns?                          │   │
│    │    │   → Test FAILED ❌ (turn limit)                    │   │
│    │    │                                                    │   │
│    │    ├─ now() - start_time > timeout?                     │   │
│    │    │   → Test FAILED ❌ (timeout)                       │   │
│    │    │                                                    │   │
│    │    └─ Otherwise → Continue loop                         │   │
│    └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Command Line Interface

### Flags Reference

| Flag | Long            | Description                                                |
| ---- | --------------- | ---------------------------------------------------------- |
| `-i` | `--input`       | Input source: file path, message, or `type:id` reference   |
| `-n` | `--name`        | Target agent ID (the agent being tested)                   |
| `-o` | `--output`      | Output file path for results                               |
| `-c` | `--connector`   | Override connector for the target agent                    |
| `-v` | `--verbose`     | Verbose output showing all turns                           |
|      | `--interactive` | Enable human input when agent awaits                       |
|      | `--simulator`   | Default simulator agent ID (e.g., `workers.test.user-sim`) |
|      | `--timeout`     | Timeout per test case (default: 5m)                        |
|      | `--parallel`    | Number of parallel test cases                              |
|      | `--fail-fast`   | Stop on first failure                                      |
|      | `--dry-run`     | Generate/parse tests without running                       |

### Input Sources (`-i` flag)

The `-i` flag supports multiple input sources with unified `type:id` format:

```bash
# 1. File path (default, no prefix needed)
yao agent test -i ./tests/multi-turn.jsonl

# 2. Direct message (no prefix, auto-detected as non-file)
yao agent test -i "Hello, how are you?" -n assistants.chat

# 3. Agent-generated test cases
yao agent test -i agents:workers.test.generator -n assistants.expense

# 4. Script-generated test cases
yao agent test -i scripts:tests.generate -n assistants.expense

# 5. With parameters (query string style)
yao agent test -i "agents:workers.test.generator?count=10&focus=edge-cases" -n assistants.expense
```

### Input Type Prefixes

| Prefix     | Description                 | Example                    |
| ---------- | --------------------------- | -------------------------- |
| (none)     | File path or direct message | `./tests.jsonl`, `"Hello"` |
| `agents:`  | Agent generates test cases  | `agents:workers.test.gen`  |
| `scripts:` | Script generates test cases | `scripts:tests.generate`   |

### Input Format

```
[prefix:]<id>[?param1=value1&param2=value2]
```

- `prefix` - Input type: `agents:` or `scripts:` (optional, default is file/message)
- `id` - Agent ID or script ID
- `?params` - Query parameters passed to generator

#### Generator Agent Interface

```typescript
// Input to generator agent
interface GeneratorInput {
  target_agent: string; // Agent being tested (from -n flag)
  target_description?: string; // Agent's description/purpose
  target_tools?: Tool[]; // Agent's available tools
  count?: number; // Number of test cases to generate
  focus?: string; // Focus area: "happy-path", "edge-cases", "errors"
  complexity?: string; // "simple", "medium", "complex"
}

// Output from generator agent
interface GeneratorOutput {
  cases: TestCase[]; // Generated test cases
}
```

#### Example Generator Prompt

```
You are a test case generator for AI agents.

Target Agent: {{target_agent}}
Description: {{target_description}}
Available Tools: {{target_tools}}

Generate {{count}} test cases with focus on: {{focus}}

For each test case, provide:
- id: Unique identifier
- name: Descriptive name
- input: User message or turns array for multi-turn
- assertions: Expected behaviors to verify

Output as JSON array of test cases.
```

### Complete Examples

```bash
# Basic: Run tests from file
yao agent test -i ./tests/expense.jsonl

# With target agent specified (required for message/agent input)
yao agent test -i "Help me file an expense" -n assistants.expense

# Agent generates tests, then runs them
yao agent test \
  -i "agents:workers.test.generator?count=20&focus=edge-cases" \
  -n assistants.expense

# Script generates tests
yao agent test \
  -i "scripts:tests.expense.generate?scenario=approval-flow" \
  -n assistants.expense

# Fully dynamic: Agent generates tests + Agent simulates user responses
yao agent test \
  -i "agents:workers.test.generator?count=10" \
  -n assistants.expense \
  --simulator workers.test.user-simulator

# Generate tests only, save to file (dry-run)
yao agent test \
  -i "agents:workers.test.generator?count=50" \
  -n assistants.expense \
  -o ./tests/generated.jsonl \
  --dry-run

# Interactive mode: human provides input when agent awaits
yao agent test -i ./tests/multi-turn.jsonl --interactive

# CI/CD mode: skip tests requiring human input
yao agent test -i ./tests/multi-turn.jsonl --skip-interactive

# Fail instead of skip when input unavailable
yao agent test -i ./tests/multi-turn.jsonl --on-missing-input=fail

# Verbose output showing all turns
yao agent test -i ./tests/multi-turn.jsonl -v
```

## Output Format

### Console Output

```
═══════════════════════════════════════════════════════════════
  Agent Test (Multi-Turn)
═══════════════════════════════════════════════════════════════
ℹ Agent: assistants.expense
ℹ Input: ./tests/expense-flow.jsonl (5 test cases)

───────────────────────────────────────────────────────────────
  Running Tests
───────────────────────────────────────────────────────────────

► [T001] Expense Reimbursement Flow (3 turns)
  ├─ Turn 1: "I want to submit an expense" → PASSED (2.1s)
  │          Agent: "What type of expense would you like to submit?"
  │          ✓ contains "type of expense"
  │
  ├─ Turn 2: "Business travel, $3500" → PASSED (3.2s)
  │          Agent: [tool: create_expense({amount: 3500, type: "travel"})]
  │          ✓ tool_called "create_expense"
  │
  ├─ Turn 3: "Yes, confirm" → PASSED (1.8s)
  │          Agent: "Expense submitted. Reference: EXP-2025-001"
  │          ✓ contains "submitted"
  │
  └─ Final Assertions: PASSED
     ✓ $.expense.status = "submitted"

► [T002] Large Expense Approval
  ├─ Turn 1: "Submit $100,000 equipment purchase" → PASSED (2.0s)
  │          Agent: "This requires manager approval. Please provide PO number."
  │
  ├─ Turn 2: SKIPPED
  │          Reason: Agent awaiting input, no next turn defined
  │          Agent asked: "Please provide PO number"
  │          Hint: Add more turns, use --interactive, or configure simulator
  │
  └─ Result: SKIPPED

► [T003] Dynamic Expense Flow (simulator: workers.test.user-sim)
  ├─ Turn 1: [Initial] "Help me file an expense" → PASSED (2.1s)
  ├─ Turn 2: [Simulated] "It's for client dinner, $250" → PASSED (2.8s)
  ├─ Turn 3: [Simulated] "Yesterday evening" → PASSED (2.2s)
  ├─ Turn 4: [Simulated] "Confirm" → PASSED (1.9s)
  │          Goal achieved: Expense submitted
  │
  └─ Final Assertions: PASSED

───────────────────────────────────────────────────────────────
  Summary
───────────────────────────────────────────────────────────────
  Total:   3 tests
  Passed:  2
  Failed:  0
  Skipped: 1

  Total turns: 10
  Avg turns/test: 3.3
  Total time: 18.1s
```

### JSONL Output

```jsonl
{
  "id": "T001",
  "name": "Expense Reimbursement Flow",
  "status": "passed",
  "turns": [
    {
      "turn": 1,
      "input": "I want to submit an expense",
      "input_source": "static",
      "output": "What type of expense would you like to submit?",
      "awaiting_input": true,
      "assertions": [
        {
          "type": "contains",
          "value": "type of expense",
          "passed": true
        }
      ],
      "duration_ms": 2100
    },
    {
      "turn": 2,
      "input": "Business travel, $3500",
      "input_source": "static",
      "output": "",
      "tool_calls": [
        {
          "name": "create_expense",
          "args": {
            "amount": 3500
          }
        }
      ],
      "awaiting_input": true,
      "assertions": [
        {
          "type": "tool_called",
          "name": "create_expense",
          "passed": true
        }
      ],
      "duration_ms": 3200
    },
    {
      "turn": 3,
      "input": "Yes, confirm",
      "input_source": "static",
      "output": "Expense submitted. Reference: EXP-2025-001",
      "awaiting_input": false,
      "assertions": [
        {
          "type": "contains",
          "value": "submitted",
          "passed": true
        }
      ],
      "duration_ms": 1800
    }
  ],
  "final_assertions": [
    {
      "type": "json_path",
      "path": "$.expense.status",
      "value": "submitted",
      "passed": true
    }
  ],
  "total_turns": 3,
  "duration_ms": 7100
}
```

## User Simulator Agent

### Interface

The simulator agent receives conversation context and generates the next user input:

```typescript
// Input to simulator
interface SimulatorInput {
  persona: string; // User persona description
  goal: string; // What user wants to achieve
  conversation: Message[]; // Conversation history
  last_response: string; // Agent's last response
  turn_number: number; // Current turn (1-based)
  max_turns: number; // Maximum allowed turns
}

// Output from simulator
interface SimulatorOutput {
  input: string; // Generated user input
  goal_achieved: boolean; // Whether goal is complete
  reasoning?: string; // Why this input was chosen
}
```

### Example Simulator Prompt

```
You are simulating a user with the following characteristics:

Persona: {{persona}}
Goal: {{goal}}

Current conversation:
{{conversation}}

The agent just responded:
"{{last_response}}"

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

Existing single-turn tests continue to work unchanged:

```jsonl
// This still works (single-turn, legacy format)
{"id": "T001", "input": "Hello", "assertions": [...]}

// Static mode with one turn (equivalent)
{"id": "T001", "mode": "static", "turns": [{"input": "Hello", "assertions": [...]}]}
```

## Error Handling

### Static Mode Errors

| Error Type       | Behavior            | Output                       |
| ---------------- | ------------------- | ---------------------------- |
| Agent timeout    | Mark turn as FAILED | `error: "timeout after 30s"` |
| Agent error      | Mark turn as FAILED | `error: "agent error: ..."`  |
| Assertion failed | Mark turn as FAILED | `assertion_errors: [...]`    |
| All turns passed | Test PASSED         | `status: "passed"`           |
| Any turn failed  | Test FAILED         | `status: "failed"`           |

### Dynamic Mode Errors

| Error Type                  | Behavior    | Output                              |
| --------------------------- | ----------- | ----------------------------------- |
| All checkpoints reached     | Test PASSED | `status: "passed"`                  |
| Checkpoints missing         | Test FAILED | `error: "missing checkpoints: ..."` |
| Max turns exceeded          | Test FAILED | `error: "max turns (20) exceeded"`  |
| Timeout exceeded            | Test FAILED | `error: "timeout after 5m"`         |
| Simulator error             | Test FAILED | `error: "simulator error: ..."`     |
| Checkpoint assertion failed | Test FAILED | `error: "checkpoint X failed"`      |

## Context and State

### Conversation Context

Each multi-turn test maintains a conversation context:

```go
type ConversationContext struct {
    SessionID    string            // Unique session for this test
    Messages     []Message         // Full conversation history
    ToolResults  map[string]any    // Results from tool calls
    Variables    map[string]any    // Custom variables set during test
    TurnCount    int               // Current turn number
}
```

### Context Passing to Simulator

The simulator receives context via standard Yao Agent Context metadata:

```go
// Framework prepares context for simulator
ctx.Metadata = map[string]any{
    "test_mode":     "simulator",
    "test_id":       "T001",
    // From simulator config
    "persona":       "New employee",
    "goal":          "Submit expense report",
    // Runtime context
    "session_id":    "test-session-001",
    "turn_count":    3,
    "tool_results":  map[string]any{...},
}

// Messages include conversation history
messages := []Message{
    {Role: "user", Content: "I want to submit an expense"},
    {Role: "assistant", Content: "What type of expense?"},
    {Role: "user", Content: "Travel expense"},
    {Role: "assistant", Content: "Please confirm the details..."},
}
```

## Attachments in Multi-Turn

Multi-turn tests support attachments at the turn level:

```jsonl
{
  "id": "T001",
  "name": "Receipt Upload Flow",
  "turns": [
    {
      "input": "I want to submit an expense with receipt",
      "attachments": [
        {
          "type": "image",
          "source": "file://./tests/fixtures/receipt.jpg"
        }
      ]
    },
    {
      "input": "The amount is $150"
    }
  ]
}
```

## Open Questions

1. **Session Management**: How to handle session state across turns? Use existing session or create new per-test?

2. **Timeout Strategy**: Per-turn timeout vs. total test timeout?

3. **Parallel Execution**: Can multi-turn tests run in parallel, or must they be sequential?

4. **Retry Logic**: If a turn fails, retry just that turn or restart entire conversation?

5. **Snapshot Testing**: Should we support "golden file" comparison for conversation flows?
