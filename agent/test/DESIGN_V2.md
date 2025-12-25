# Agent Test Framework V2 Design

## Overview

This document describes the design for Agent Test Framework V2, which extends the existing testing capabilities with:

- **Multi-turn conversations** - Test agents across multiple interaction rounds
- **Agent-driven testing** - Use agents to generate test cases and simulate user responses
- **Interactive testing** - Human-in-the-loop testing mode

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
│  │ ./test.jsonl│  │ "Hello..."  │  │ agent:xxx   │  │ (stdin)     │     │
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
│  │   ┌─────────┐    ┌─────────────────┐    ┌─────────────────┐      │    │
│  │   │  Turn   │───▶│  Target Agent   │───▶│    Response     │      │    │
│  │   │  Input  │    │  (being tested) │    │    + State      │      │    │
│  │   └─────────┘    └─────────────────┘    └────────┬────────┘      │    │
│  │        ▲                                         │               │    │
│  │        │                                         ▼               │    │
│  │        │         ┌─────────────────────────────────────┐         │    │
│  │        │         │      Awaiting Input Detection       │         │    │
│  │        │         │  - Explicit declaration             │         │    │
│  │        │         │  - Tool-based detection             │         │    │
│  │        │         │  - Content heuristics               │         │    │
│  │        │         └──────────────┬──────────────────────┘         │    │
│  │        │                        │                                │    │
│  │        │              ┌─────────┴─────────┐                      │    │
│  │        │              ▼                   ▼                      │    │
│  │        │         Awaiting=YES        Awaiting=NO                 │    │
│  │        │              │                   │                      │    │
│  │        │              ▼                   ▼                      │    │
│  │   NEXT INPUT     ┌─────────┐         ┌─────────┐                 │    │
│  │   SOURCES:       │ Get Next│         │Complete │                 │    │
│  │                  │  Input  │         │  Test   │                 │    │
│  │   ┌──────────┐   └────┬────┘         └─────────┘                 │    │
│  │   │  Static  │◀───────┤                                          │    │
│  │   │  turns[] │        │                                          │    │
│  │   └──────────┘        │                                          │    │
│  │   ┌──────────┐        │                                          │    │
│  │   │Simulator │◀───────┤                                          │    │
│  │   │  Agent   │        │                                          │    │
│  │   └──────────┘        │                                          │    │
│  │   ┌──────────┐        │                                          │    │
│  │   │  Human   │◀───────┤                                          │    │
│  │   │  Input   │        │                                          │    │
│  │   └──────────┘        │                                          │    │
│  │   ┌──────────┐        │                                          │    │
│  │   │   SKIP   │◀───────┘                                          │    │
│  │   │ (no src) │                                                   │    │
│  │   └──────────┘                                                   │    │
│  │                                                                   │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                   │                                      │
│                                   ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                         Assertions                               │    │
│  │  - Per-turn assertions                                           │    │
│  │  - Final assertions                                              │    │
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
        // From simulator.metadata in test case
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

### Multi-Turn (New)

```jsonl
{
  "id": "T001",
  "name": "Expense Reimbursement Flow",
  "type": "multi_turn",
  "options": {
    "connector": "openai-gpt4",
    "skip": {
      "history": true
    },
    "metadata": {
      "test_scenario": "happy-path"
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
      "input": "Business travel to Beijing, flight $2000, hotel $1500",
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
  ],
  "simulator": {
    "use": "agents:workers.test.user-simulator",
    "options": {
      "metadata": {
        "persona": "New employee unfamiliar with expense process",
        "goal": "Submit a $3500 travel expense",
        "max_turns": 10
      }
    }
  },
  "interactive": {
    "enabled": false,
    "timeout": "5m"
  },
  "on_missing_input": "skip",
  "final_assertions": [
    {
      "type": "json_path",
      "path": "$.expense.status",
      "value": "submitted"
    }
  ]
}
```

### Field Descriptions

| Field                 | Type   | Required | Description                                      |
| --------------------- | ------ | -------- | ------------------------------------------------ |
| `id`                  | string | Yes      | Unique test identifier                           |
| `name`                | string | No       | Human-readable test name                         |
| `type`                | string | No       | `"single_turn"` (default) or `"multi_turn"`      |
| `options`             | object | No       | `context.Options` passed to target agent         |
| `options.connector`   | string | No       | LLM connector to use                             |
| `options.skip`        | object | No       | Skip config (history, trace, etc.)               |
| `options.search`      | any    | No       | Search behavior control                          |
| `options.mode`        | string | No       | Agent mode                                       |
| `options.metadata`    | object | No       | Custom metadata passed to agent                  |
| `turns`               | array  | No       | Static turn definitions                          |
| `turns[].input`       | string | Yes      | User input for this turn                         |
| `turns[].assertions`  | array  | No       | Assertions for this turn's response              |
| `turns[].options`     | object | No       | Per-turn options override                        |
| `simulator`           | object | No       | Dynamic input generator configuration            |
| `simulator.use`       | string | Yes      | Simulator reference: `agents:id` or `scripts:id` |
| `simulator.options`   | object | No       | `context.Options` passed to simulator agent      |
| `interactive`         | object | No       | Interactive mode configuration                   |
| `interactive.enabled` | bool   | No       | Enable human input (default: false)              |
| `interactive.timeout` | string | No       | Timeout for human input (default: "5m")          |
| `on_missing_input`    | string | No       | `"skip"`, `"fail"`, or `"end"` (default: "skip") |
| `final_assertions`    | array  | No       | Assertions after conversation completes          |

## Execution Modes

### Mode 1: Static Turns

Uses predefined `turns` array. Best for deterministic flows.

```
Turn 1: Send turns[0].input → Assert turns[0].assertions
Turn 2: Send turns[1].input → Assert turns[1].assertions
...
```

### Mode 2: Dynamic Simulator

Uses an agent to simulate user responses. Best for complex/variable flows.

```
Turn 1: Send initial input → Get response
Turn 2: Simulator generates input based on response → Get response
...
Until: Goal achieved OR max_turns reached
```

### Mode 3: Interactive

Prompts human for input when agent awaits. Best for debugging/exploration.

```
Turn 1: Send input → Get response
Turn 2: [Agent awaiting] → Prompt human → Get response
...
```

### Mode 4: Skip (Default Fallback)

When agent awaits input but no source available, skip with explanation.

### Mode Priority

When multiple input sources are configured, they are used in this order:

1. **Static turns** - If `turns[n+1]` exists, use it
2. **Simulator** - If no more static turns but simulator configured, use it
3. **Interactive** - If `--interactive` flag and no simulator, prompt human
4. **Skip/Fail/End** - Based on `on_missing_input` setting

This allows hybrid testing: define some turns statically, then let simulator handle the rest.

```jsonl
{
  "turns": [
    {
      "input": "Start expense report"
    },
    {
      "input": "Travel expense, $500"
    }
  ],
  "simulator": {
    "use": "agents:workers.test.user-sim",
    "metadata": {
      "goal": "Complete the expense submission"
    }
  }
}
```

In this example:

- Turn 1-2: Use static inputs
- Turn 3+: Simulator generates inputs until goal achieved

## Execution Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Multi-Turn Test Execution                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  START: Get initial input                                        │
│         ├─ From turns[0].input if defined                        │
│         └─ From test.input (single-turn compat)                  │
│                              ↓                                   │
│  LOOP:                                                           │
│    ┌─────────────────────────────────────────────────────────┐   │
│    │ 1. Send input to Agent                                  │   │
│    │                         ↓                               │   │
│    │ 2. Get Agent response                                   │   │
│    │                         ↓                               │   │
│    │ 3. Execute turn assertions (if defined)                 │   │
│    │                         ↓                               │   │
│    │ 4. Check: Is Agent awaiting input?                      │   │
│    │           │                                             │   │
│    │           ├─ NO → Exit loop (conversation complete)     │   │
│    │           │                                             │   │
│    │           └─ YES → Get next input:                      │   │
│    │                    │                                    │   │
│    │                    ├─ turns[n+1] exists?                │   │
│    │                    │   → Use static input               │   │
│    │                    │                                    │   │
│    │                    ├─ simulator configured?             │   │
│    │                    │   → Call simulator agent           │   │
│    │                    │                                    │   │
│    │                    ├─ interactive enabled?              │   │
│    │                    │   → Prompt for human input         │   │
│    │                    │                                    │   │
│    │                    └─ None available?                   │   │
│    │                        → Handle per on_missing_input:   │   │
│    │                           skip: SKIP test               │   │
│    │                           fail: FAIL test               │   │
│    │                           end:  Exit loop normally      │   │
│    └─────────────────────────────────────────────────────────┘   │
│                              ↓                                   │
│  END: Execute final_assertions                                   │
│       Report result                                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Command Line Interface

### Flags Reference

| Flag | Long            | Description                                              |
| ---- | --------------- | -------------------------------------------------------- |
| `-i` | `--input`       | Input source: file path, message, or `type:id` reference |
| `-n` | `--name`        | Target agent ID (the agent being tested)                 |
| `-o` | `--output`      | Output file path for results                             |
| `-c` | `--connector`   | Override connector for the target agent                  |
| `-v` | `--verbose`     | Verbose output showing all turns                         |
|      | `--interactive` | Enable human input when agent awaits                     |
|      | `--simulator`   | Default simulator: `agents:id` or `scripts:id`           |
|      | `--timeout`     | Timeout per test case (default: 5m)                      |
|      | `--parallel`    | Number of parallel test cases                            |
|      | `--fail-fast`   | Stop on first failure                                    |
|      | `--dry-run`     | Generate/parse tests without running                     |

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
  --simulator agents:workers.test.user-simulator

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
// This still works
{"id": "T001", "input": "Hello", "assertions": [...]}

// Equivalent to
{"id": "T001", "type": "single_turn", "turns": [{"input": "Hello", "assertions": [...]}]}
```

## Error Handling

### Turn-Level Errors

| Error Type       | Behavior             | Output                           |
| ---------------- | -------------------- | -------------------------------- |
| Agent timeout    | Mark turn as FAILED  | `error: "timeout after 30s"`     |
| Agent error      | Mark turn as FAILED  | `error: "agent error: ..."`      |
| Assertion failed | Mark turn as FAILED  | `assertion_errors: [...]`        |
| Simulator error  | Mark turn as SKIPPED | `skip_reason: "simulator error"` |

### Test-Level Errors

| Error Type                   | Behavior             | Output                             |
| ---------------------------- | -------------------- | ---------------------------------- |
| No initial input             | Mark test as FAILED  | `error: "no initial input"`        |
| Max turns exceeded           | Mark test as FAILED  | `error: "max turns (20) exceeded"` |
| All turns passed             | Mark test as PASSED  | `status: "passed"`                 |
| Any turn failed              | Mark test as FAILED  | `status: "failed"`                 |
| Skipped due to missing input | Mark test as SKIPPED | `status: "skipped"`                |

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
