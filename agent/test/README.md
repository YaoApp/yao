# Agent Test Framework

A comprehensive testing framework for Yao AI agents with support for standard testing, dynamic (simulator-driven) testing, agent-driven assertions, and CI integration.

## Quick Start

### Standard Tests

```bash
# Test with direct message (auto-detect agent from current directory)
cd assistants/keyword
yao agent test -i "Extract keywords from: AI and machine learning"

# Test with direct message (specify agent explicitly)
yao agent test -i "Hello world" -n workers.system.keyword

# Test with JSONL file (auto-detect agent from path)
yao agent test -i assistants/keyword/tests/inputs.jsonl

# Generate HTML report
yao agent test -i tests/inputs.jsonl -o report.html

# Stability analysis (run each test 5 times)
yao agent test -i tests/inputs.jsonl --runs 5
```

### Agent-Driven Input

```bash
# Generate test cases using an agent
yao agent test -i "agents:tests.generator-agent?count=10" -n assistants.expense

# Preview generated tests without running (dry-run)
yao agent test -i "agents:tests.generator-agent?count=5" -n assistants.expense --dry-run
```

### Dynamic Mode (Simulator)

```bash
# Run dynamic tests with simulator
yao agent test -i tests/dynamic.jsonl --simulator tests.simulator-agent

# See detailed turn-by-turn output
yao agent test -i tests/dynamic.jsonl -v
```

### Script Tests

```bash
# Test agent handler scripts (hooks, tools, setup functions)
yao agent test -i scripts.expense.setup -v

# Run specific tests with regex filter
yao agent test -i scripts.expense.setup --run "TestSystemReady" -v

# Run with custom context (authorization, metadata)
yao agent test -i scripts.expense.setup --ctx tests/context.json -v
```

## Input Modes

The `-i` flag supports multiple input modes:

### 1. JSONL File Mode

Load test cases from a file:

```bash
yao agent test -i tests/inputs.jsonl
```

Agent is auto-detected by traversing up from the input file to find `package.yao`.

### 2. Direct Message Mode

Test with a single message:

```bash
# Auto-detect agent from current working directory
cd assistants/keyword
yao agent test -i "Extract keywords from this text"

# Or specify agent explicitly
yao agent test -i "Hello" -n workers.system.keyword
```

### 3. Agent-Driven Input Mode

Generate test cases using a generator agent:

```bash
# Basic usage (-n specifies the target agent to test)
yao agent test -i "agents:tests.generator-agent" -n assistants.expense

# With parameters
yao agent test -i "agents:tests.generator-agent?count=10&focus=edge-cases" -n assistants.expense

# Dry-run to preview generated tests
yao agent test -i "agents:tests.generator-agent?count=5" -n assistants.expense --dry-run
```

**Note**: The `-n` flag is **required** for agent-driven input mode to specify which agent to test. The generator agent creates test cases for the target agent.

### 4. Script Test Mode

Test agent handler scripts:

```bash
yao agent test -i scripts.expense.setup -v
```

Script test input format: `scripts.<assistant>.<module>` (e.g., `scripts.expense.setup` → `assistants/expense/src/setup_test.ts`).

### 5. Script-Generated Input Mode

Generate test cases using a script:

```bash
yao agent test -i "scripts:tests.gen.Generate" -n assistants.expense
```

**Note**: `scripts.xxx` (with dot) runs script tests, while `scripts:xxx` (with colon) generates test cases from a script.

## Test Modes

### Standard Mode

Single call to agent with optional message history. Each test is independent and stateless.

```jsonl
{
  "id": "T001",
  "input": "Hello",
  "assert": {
    "type": "contains",
    "value": "Hi"
  }
}
```

### Dynamic Mode

Simulator-driven testing with checkpoint validation. A simulator agent generates user messages while checkpoints verify agent behavior.

```jsonl
{
  "id": "T001",
  "input": "I want to order coffee",
  "simulator": {
    "use": "tests.simulator-agent",
    "options": {
      "metadata": {
        "persona": "Customer",
        "goal": "Order a latte"
      }
    }
  },
  "checkpoints": [
    {
      "id": "greeting",
      "assert": {
        "type": "regex",
        "value": "(?i)hello"
      }
    },
    {
      "id": "ask_size",
      "after": [
        "greeting"
      ],
      "assert": {
        "type": "regex",
        "value": "(?i)size"
      }
    }
  ],
  "max_turns": 10
}
```

## Command Line Options

| Flag          | Description                                              | Default                    |
| ------------- | -------------------------------------------------------- | -------------------------- |
| `-i`          | Input: JSONL file, message, `agents:xxx`, or `scripts:x` | (required)                 |
| `-o`          | Output file path                                         | `output-{timestamp}.jsonl` |
| `-n`          | Agent ID (optional, auto-detected)                       | auto-detect                |
| `-a`          | Application directory                                    | auto-detect                |
| `-e`          | Environment file                                         | -                          |
| `-c`          | Override connector                                       | agent default              |
| `-u`          | Test user ID                                             | `test-user`                |
| `-t`          | Test team ID                                             | `test-team`                |
| `-r`          | Reporter agent ID for custom report                      | built-in                   |
| `-v`          | Verbose output                                           | false                      |
| `--ctx`       | Path to context JSON file for custom authorization       | -                          |
| `--simulator` | Default simulator agent ID for dynamic mode              | -                          |
| `--before`    | Global BeforeAll hook (e.g., `env_test.BeforeAll`)       | -                          |
| `--after`     | Global AfterAll hook (e.g., `env_test.AfterAll`)         | -                          |
| `--runs`      | Runs per test (stability analysis)                       | 1                          |
| `--run`       | Regex pattern to filter which tests to run               | -                          |
| `--timeout`   | Timeout per test                                         | 2m                         |
| `--parallel`  | Parallel test cases                                      | 1                          |
| `--fail-fast` | Stop on first failure                                    | false                      |
| `--dry-run`   | Generate test cases without running them                 | false                      |

## Custom Context File

Create a JSON file for custom authorization:

```json
{
  "chat_id": "test-chat-001",
  "authorized": {
    "user_id": "test-user-123",
    "team_id": "test-team-456",
    "constraints": {
      "owner_only": true,
      "extra": { "department": "engineering" }
    }
  },
  "metadata": {
    "mode": "test"
  }
}
```

Use with `--ctx`:

```bash
yao agent test -i scripts.expense.setup --ctx tests/context.json -v
```

## Input Format (JSONL)

Each line is a JSON object. Below are examples organized by scenario.

### Scenario 1: Simple Text Input

Basic test with string input:

```jsonl
{"id": "greeting-basic", "input": "Hello, how are you?"}
{"id": "greeting-chinese", "input": "你好，请问有什么可以帮助你的？"}
```

### Scenario 2: With Assertions

Validate response content:

```jsonl
{"id": "keyword-extract", "input": "Extract keywords from: AI and machine learning", "assert": {"type": "contains", "value": "AI"}}
{"id": "json-response", "input": "What's the weather?", "assert": {"type": "json_path", "path": "need_search", "value": true}}
{"id": "no-error", "input": "Help me", "assert": {"type": "not_contains", "value": "error"}}
```

### Scenario 3: Multiple Assertions

All assertions must pass:

```jsonl
{
  "id": "expense-submit",
  "input": "Submit $500 travel expense",
  "assert": [
    {
      "type": "contains",
      "value": "expense"
    },
    {
      "type": "not_contains",
      "value": "error"
    },
    {
      "type": "regex",
      "value": "(?i)(submitted|created|confirmed)"
    }
  ]
}
```

### Scenario 4: Conversation History

Test with multi-turn context:

```jsonl
{
  "id": "expense-confirm",
  "input": [
    {
      "role": "user",
      "content": "Submit an expense"
    },
    {
      "role": "assistant",
      "content": "What type of expense?"
    },
    {
      "role": "user",
      "content": "Travel, $500"
    },
    {
      "role": "assistant",
      "content": "Please confirm: $500 travel expense"
    },
    {
      "role": "user",
      "content": "Yes, confirm"
    }
  ],
  "assert": {
    "type": "regex",
    "value": "(?i)(submitted|created)"
  }
}
```

### Scenario 5: With File Attachments

Test with images or documents:

```jsonl
{
  "id": "receipt-analyze",
  "input": {
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "Analyze this receipt"
      },
      {
        "type": "image",
        "source": "file://fixtures/receipt.jpg"
      }
    ]
  },
  "assert": {
    "type": "contains",
    "value": "amount"
  }
}
```

### Scenario 6: Agent-Driven Assertion

Use LLM to validate response semantics:

```jsonl
{
  "id": "helpful-response",
  "input": "How do I reset my password?",
  "assert": {
    "type": "agent",
    "use": "agents:tests.validator-agent",
    "value": "Response should provide clear step-by-step instructions"
  }
}
```

### Scenario 7: With Options

Override connector or skip features:

```jsonl
{"id": "fast-model", "input": "Quick question", "options": {"connector": "deepseek.v3", "skip": {"history": true, "trace": true}}}
{"id": "scenario-test", "input": "Query users", "options": {"metadata": {"scenario": "filter"}}, "assert": {"type": "json_path", "path": "from", "value": "users"}}
```

### Scenario 8: With Before/After Hooks

Setup and teardown for each test:

```jsonl
{
  "id": "with-user-data",
  "input": "Show my expenses",
  "before": "env_test.Before",
  "after": "env_test.After",
  "assert": {
    "type": "contains",
    "value": "expense"
  }
}
```

### Scenario 9: Skip Test

Temporarily disable a test:

```jsonl
{
  "id": "wip-feature",
  "input": "New feature test",
  "skip": true
}
```

### Scenario 10: Dynamic Mode (Simulator)

Multi-turn testing with user simulator:

```jsonl
{
  "id": "coffee-order",
  "input": "I want to order coffee",
  "simulator": {
    "use": "tests.simulator-agent",
    "options": {
      "metadata": {
        "persona": "Regular customer",
        "goal": "Order a medium latte"
      }
    }
  },
  "checkpoints": [
    {
      "id": "greeting",
      "assert": {
        "type": "regex",
        "value": "(?i)(hello|hi|help)"
      }
    },
    {
      "id": "ask-size",
      "after": [
        "greeting"
      ],
      "assert": {
        "type": "regex",
        "value": "(?i)size"
      }
    },
    {
      "id": "confirm",
      "after": [
        "ask-size"
      ],
      "assert": {
        "type": "regex",
        "value": "(?i)confirm"
      }
    }
  ],
  "max_turns": 10
}
```

### Scenario 11: Dynamic Mode with Optional Checkpoint

Some checkpoints are optional:

```jsonl
{
  "id": "expense-flow",
  "input": "Submit expense",
  "simulator": {
    "use": "tests.simulator-agent",
    "options": {
      "metadata": {
        "persona": "New employee",
        "goal": "Submit $500 travel expense"
      }
    }
  },
  "checkpoints": [
    {
      "id": "ask-type",
      "assert": {
        "type": "regex",
        "value": "(?i)type"
      }
    },
    {
      "id": "suggest-category",
      "required": false,
      "assert": {
        "type": "contains",
        "value": "category"
      }
    },
    {
      "id": "confirm",
      "after": [
        "ask-type"
      ],
      "assert": {
        "type": "regex",
        "value": "(?i)confirm"
      }
    }
  ],
  "max_turns": 15
}
```

### Standard Mode Fields

| Field      | Type                           | Required | Description                           |
| ---------- | ------------------------------ | -------- | ------------------------------------- |
| `id`       | string                         | Yes      | Test case ID                          |
| `input`    | string \| Message \| []Message | Yes      | Test input                            |
| `assert`   | Assertion \| []Assertion       | No       | Assertion rules                       |
| `expected` | any                            | No       | Expected output (exact match)         |
| `user`     | string                         | No       | Override user ID for this test        |
| `team`     | string                         | No       | Override team ID for this test        |
| `metadata` | map                            | No       | Additional metadata for hooks         |
| `options`  | Options                        | No       | Context options                       |
| `timeout`  | string                         | No       | Override timeout (e.g., "30s")        |
| `skip`     | bool                           | No       | Skip this test                        |
| `before`   | string                         | No       | Before hook (e.g., `env_test.Before`) |
| `after`    | string                         | No       | After hook (e.g., `env_test.After`)   |

### Dynamic Mode Fields

| Field                         | Type   | Required | Description                            |
| ----------------------------- | ------ | -------- | -------------------------------------- |
| `id`                          | string | Yes      | Test case ID                           |
| `input`                       | string | Yes      | Initial user message                   |
| `simulator`                   | object | Yes      | Simulator configuration                |
| `simulator.use`               | string | Yes      | Simulator agent ID (no prefix)         |
| `simulator.options`           | object | No       | Simulator options                      |
| `simulator.options.metadata`  | map    | No       | Metadata (persona, goal, etc.)         |
| `simulator.options.connector` | string | No       | Override simulator connector           |
| `checkpoints`                 | array  | Yes      | Checkpoints to verify                  |
| `checkpoints[].id`            | string | Yes      | Checkpoint identifier                  |
| `checkpoints[].description`   | string | No       | Human-readable description             |
| `checkpoints[].assert`        | object | Yes      | Assertion to validate                  |
| `checkpoints[].after`         | array  | No       | Checkpoint IDs that must occur first   |
| `checkpoints[].required`      | bool   | No       | Is checkpoint required (default: true) |
| `max_turns`                   | int    | No       | Maximum turns (default: 20)            |
| `timeout`                     | string | No       | Override timeout (e.g., "2m")          |

### Options

The `options` field allows per-test-case configuration:

| Field                    | Type   | Description                                |
| ------------------------ | ------ | ------------------------------------------ |
| `connector`              | string | Override connector (e.g., `"deepseek.v3"`) |
| `mode`                   | string | Agent mode (default: `"chat"`)             |
| `search`                 | bool   | Enable/disable search mode                 |
| `disable_global_prompts` | bool   | Temporarily disable global prompts         |
| `metadata`               | map    | Custom data passed to hooks                |
| `skip`                   | object | Skip configuration (see below)             |

### Options.skip

| Field     | Type | Description             |
| --------- | ---- | ----------------------- |
| `history` | bool | Skip history loading    |
| `trace`   | bool | Skip trace logging      |
| `output`  | bool | Skip output to client   |
| `keyword` | bool | Skip keyword extraction |
| `search`  | bool | Skip auto search        |

### Input Types

| Type        | Description          | Example                                               |
| ----------- | -------------------- | ----------------------------------------------------- |
| `string`    | Simple text          | `"Hello world"`                                       |
| `Message`   | Single message       | `{"role": "user", "content": "..."}`                  |
| `[]Message` | Conversation history | `[{"role": "user", ...}, {"role": "assistant", ...}]` |

## Assertions

Use `assert` for flexible validation. If `assert` is defined, it takes precedence over `expected`.

### Static Assertions

| Type           | Description                   | Example                                                   |
| -------------- | ----------------------------- | --------------------------------------------------------- |
| `equals`       | Exact match                   | `{"type": "equals", "value": {"key": "val"}}`             |
| `contains`     | Output contains value         | `{"type": "contains", "value": "keyword"}`                |
| `not_contains` | Output does not contain value | `{"type": "not_contains", "value": "error"}`              |
| `json_path`    | Extract JSON path and compare | `{"type": "json_path", "path": "$.field", "value": true}` |
| `regex`        | Match regex pattern           | `{"type": "regex", "value": "\\d+"}`                      |
| `type`         | Check output type             | `{"type": "type", "value": "object"}`                     |
| `tool_called`  | Check if a tool was called    | `{"type": "tool_called", "value": "setup"}`               |
| `tool_result`  | Check tool execution result   | `{"type": "tool_result", "value": {"tool": "setup", "result": {"success": true}}}` |

### Assertion Fields

| Field     | Type   | Description                                              |
| --------- | ------ | -------------------------------------------------------- |
| `type`    | string | Assertion type (required)                                |
| `value`   | any    | Expected value or pattern                                |
| `path`    | string | JSON path for `json_path` type                           |
| `script`  | string | Script name for `script` type                            |
| `use`     | string | Agent/script ID for `agent` type (with `agents:` prefix) |
| `options` | object | Options for agent assertions                             |
| `message` | string | Custom failure message                                   |
| `negate`  | bool   | Invert the assertion result                              |

### Agent-Driven Assertions

For semantic or fuzzy validation using an LLM:

```jsonl
{
  "id": "T001",
  "input": "Hello",
  "assert": {
    "type": "agent",
    "use": "agents:tests.validator-agent",
    "value": "Response should be friendly and helpful"
  }
}
```

The validator agent receives the output and criteria, then returns `{"passed": true/false, "reason": "..."}`.

**How it works:**

1. The framework builds a validation request with the agent's response (including tool result messages)
2. The validator agent evaluates the response against the criteria
3. The validator returns a JSON response with `passed` and `reason`

**Output in test report (for checkpoints):**

```json
{
  "agent_validation": {
    "passed": true,
    "reason": "Response explicitly confirms setup completion",
    "criteria": "Response should be friendly and helpful",
    "input": "Hello! How can I help you today?",
    "response": {
      "passed": true,
      "reason": "Response explicitly confirms setup completion"
    }
  }
}
```

- `input`: The content sent to the validator (agent response + tool result messages)
- `response`: The raw JSON response from the validator agent
- `criteria`: The validation criteria from the test case

### Tool Assertions

For validating that specific tools were called and their results:

#### tool_called

Check if a specific tool was called:

```jsonl
{
  "id": "T001",
  "input": "Set up my expense system",
  "assert": {
    "type": "tool_called",
    "value": "setup"
  }
}
```

**Value formats:**

- **String**: Tool name (supports suffix matching, e.g., `"setup"` matches `"agents_expense_tools__setup"`)
- **Array**: Any of the specified tools must be called
- **Object**: Match tool name and optionally arguments

```jsonl
// Match any of these tools
{"type": "tool_called", "value": ["setup", "init"]}

// Match tool with specific arguments
{"type": "tool_called", "value": {"name": "setup", "arguments": {"action": "init"}}}
```

#### tool_result

Check the result of a tool execution:

```jsonl
{
  "id": "T001",
  "input": "Set up my expense system",
  "assert": {
    "type": "tool_result",
    "value": {
      "tool": "setup",
      "result": {
        "success": true
      }
    }
  }
}
```

**Result matching:**

- If `result` is omitted, only checks that the tool executed without error
- Supports partial matching (only specified fields are checked)
- Supports regex patterns with `regex:` prefix for string values

```jsonl
// Just check tool executed without error
{"type": "tool_result", "value": {"tool": "setup"}}

// Check specific result fields
{"type": "tool_result", "value": {"tool": "setup", "result": {"success": true}}}

// Use regex for message matching
{"type": "tool_result", "value": {"tool": "setup", "result": {"message": "regex:(?i)setup.*complete"}}}
```

### Script Assertions

For custom validation logic:

```jsonl
{
  "id": "T001",
  "input": "Test",
  "assert": {
    "type": "script",
    "script": "scripts.test.Validate"
  }
}
```

### Multiple Assertions

All assertions must pass:

```jsonl
{
  "id": "T001",
  "input": "Hello",
  "assert": [
    {
      "type": "contains",
      "value": "Hi"
    },
    {
      "type": "not_contains",
      "value": "error"
    },
    {
      "type": "json_path",
      "path": "status",
      "value": "ok"
    }
  ]
}
```

## File Attachments

Test inputs support file attachments using the `file://` protocol:

```jsonl
{
  "id": "T001",
  "input": {
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "Analyze this image"
      },
      {
        "type": "image",
        "source": "file://fixtures/receipt.jpg"
      }
    ]
  }
}
```

Supported types: images (jpg, png, gif, webp), audio (wav, mp3), documents (pdf, doc, txt).

## Before/After Hooks

Hooks allow you to run setup and teardown code before and after tests. Hook scripts must be placed in the agent's `src/` directory with `_test.ts` suffix.

### Hook Types

| Hook        | Scope    | When Called           | Use Case                        |
| ----------- | -------- | --------------------- | ------------------------------- |
| `Before`    | Per-test | Before each test case | Create test data, setup context |
| `After`     | Per-test | After each test case  | Cleanup test data, log results  |
| `BeforeAll` | Global   | Once before all tests | Database migration, init        |
| `AfterAll`  | Global   | Once after all tests  | Global cleanup, report          |

### Execution Order

```
BeforeAll (global)
  ├─ Before (test 1)
  │    └─ Test 1 execution
  │    └─ After (test 1)
  ├─ Before (test 2)
  │    └─ Test 2 execution
  │    └─ After (test 2)
  └─ ...
AfterAll (global)
```

### Per-Test Hooks

Defined in JSONL, scripts located in agent's `src/` directory:

```jsonl
{
  "id": "T001",
  "input": "Test",
  "before": "env_test.Before",
  "after": "env_test.After"
}
```

### Global Hooks

Via CLI flags:

```bash
yao agent test -i tests/inputs.jsonl --before env_test.BeforeAll --after env_test.AfterAll
```

### Hook Function Signatures

```typescript
// assistants/expense/src/env_test.ts

/**
 * Before - Called before each test case
 * @param ctx - Agent context with user/team info
 * @param testCase - The test case about to run
 * @returns any - Data passed to After hook (optional)
 */
export function Before(ctx: Context, testCase: TestCase): any {
  const userId = Process("models.user.Create", { name: "Test User" });
  return { userId }; // This data is passed to After
}

/**
 * After - Called after each test case (pass or fail)
 * @param ctx - Agent context
 * @param testCase - The test case that ran
 * @param result - Test result with status, output, duration
 * @param beforeData - Data returned from Before hook
 */
export function After(
  ctx: Context,
  testCase: TestCase,
  result: TestResult,
  beforeData: any
) {
  if (beforeData?.userId) {
    Process("models.user.Delete", beforeData.userId);
  }
  if (result.status === "failed") {
    console.log(`Test ${testCase.id} failed: ${result.error}`);
  }
}

/**
 * BeforeAll - Called once before all tests
 * @param ctx - Agent context
 * @param testCases - Array of all test cases
 * @returns any - Data passed to AfterAll hook (optional)
 */
export function BeforeAll(ctx: Context, testCases: TestCase[]): any {
  Process("models.migrate");
  return { initialized: true, count: testCases.length };
}

/**
 * AfterAll - Called once after all tests complete
 * @param ctx - Agent context
 * @param results - Array of all test results
 * @param beforeData - Data returned from BeforeAll hook
 */
export function AfterAll(ctx: Context, results: TestResult[], beforeData: any) {
  const passed = results.filter((r) => r.status === "passed").length;
  console.log(`Tests completed: ${passed}/${results.length} passed`);
  Process("models.cleanup");
}
```

### Hook Parameters

**Context** - Agent execution context:

```typescript
interface Context {
  locale: string; // Locale (e.g., "en-us")
  authorized: {
    user_id: string; // Test user ID
    team_id: string; // Test team ID
    constraints?: object; // Access constraints
  };
  metadata: object; // Custom metadata from test case
}
```

**TestCase** - Test case definition:

```typescript
interface TestCase {
  id: string; // Test case ID
  input: any; // Test input (string, Message, or Message[])
  assert?: object; // Assertion rules
  expected?: any; // Expected output
  user?: string; // Override user ID
  team?: string; // Override team ID
  metadata?: object; // Custom metadata
  options?: object; // Context options
  timeout?: string; // Timeout (e.g., "30s")
  skip?: boolean; // Skip flag
  before?: string; // Before hook reference
  after?: string; // After hook reference
}
```

**TestResult** - Test execution result:

```typescript
interface TestResult {
  id: string; // Test case ID
  status: string; // "passed" | "failed" | "error" | "skipped" | "timeout"
  input: any; // Actual input sent
  output: any; // Agent response
  expected?: any; // Expected output (if defined)
  error?: string; // Error message (if failed)
  duration_ms: number; // Execution time in milliseconds
  assertions?: object[]; // Assertion results
}
```

### Common Use Cases

**Database Setup/Teardown**:

```typescript
export function Before(ctx: Context, testCase: TestCase): any {
  // Create test records
  const user = Process("models.user.Create", {
    name: "Test",
    email: "test@example.com",
  });
  const expense = Process("models.expense.Create", {
    user_id: user.id,
    amount: 100,
  });
  return { user, expense };
}

export function After(
  ctx: Context,
  testCase: TestCase,
  result: TestResult,
  data: any
) {
  // Clean up in reverse order
  if (data?.expense) Process("models.expense.Delete", data.expense.id);
  if (data?.user) Process("models.user.Delete", data.user.id);
}
```

**Conditional Setup Based on Metadata**:

```typescript
export function Before(ctx: Context, testCase: TestCase): any {
  const scenario = testCase.metadata?.scenario || "default";

  if (scenario === "empty_db") {
    Process("models.expense.DeleteAll");
  } else if (scenario === "with_data") {
    Process("scripts.tests.seed.LoadTestData");
  }

  return { scenario };
}
```

**Logging and Debugging**:

```typescript
export function After(
  ctx: Context,
  testCase: TestCase,
  result: TestResult,
  data: any
) {
  if (result.status === "failed") {
    console.log("=== Test Failed ===");
    console.log("Test ID:", testCase.id);
    console.log("Input:", JSON.stringify(testCase.input));
    console.log("Output:", JSON.stringify(result.output));
    console.log("Error:", result.error);
  }
}
```

## Script Testing

Test agent handler scripts with the `t.assert` API:

```typescript
// assistants/expense/src/setup_test.ts
import { SystemReady } from "./setup";

export function TestSystemReady(t: TestingT, ctx: Context) {
  const result = SystemReady(ctx);

  t.assert.True(result.success, "Should succeed");
  t.assert.Equal(result.status, "ready", "Status should be ready");
  t.assert.NotNil(result.data, "Data should not be nil");
}

export function TestWithAgentAssertion(t: TestingT, ctx: Context) {
  const response = Process("agents.expense.Stream", ctx, messages);

  // Static assertion
  t.assert.Contains(response.content, "confirm");

  // Agent-driven assertion
  t.assert.Agent(response.content, "tests.validator-agent", {
    criteria: "Response should ask for confirmation",
  });
}
```

### Available Assertions

| Method                           | Description                    |
| -------------------------------- | ------------------------------ |
| `t.assert.True(value, msg)`      | Assert value is true           |
| `t.assert.False(value, msg)`     | Assert value is false          |
| `t.assert.Equal(a, b, msg)`      | Assert a equals b              |
| `t.assert.NotEqual(a, b, msg)`   | Assert a not equals b          |
| `t.assert.Nil(value, msg)`       | Assert value is null/undefined |
| `t.assert.NotNil(value, msg)`    | Assert value is not nil        |
| `t.assert.Contains(s, sub, msg)` | Assert string contains substr  |
| `t.assert.Len(arr, n, msg)`      | Assert array/string length     |
| `t.assert.Agent(resp, id, opts)` | Agent-driven assertion         |

## Dynamic Mode

For testing complex conversation flows where the path is unpredictable:

```jsonl
{
  "id": "coffee-order",
  "input": "I want to order coffee",
  "simulator": {
    "use": "tests.simulator-agent",
    "options": {
      "metadata": {
        "persona": "Customer ordering a latte",
        "goal": "Complete the coffee order"
      }
    }
  },
  "checkpoints": [
    {
      "id": "greeting",
      "description": "Agent greets customer",
      "assert": {
        "type": "regex",
        "value": "(?i)(hello|hi|help)"
      }
    },
    {
      "id": "ask_size",
      "description": "Agent asks for size",
      "after": [
        "greeting"
      ],
      "assert": {
        "type": "regex",
        "value": "(?i)size"
      }
    },
    {
      "id": "confirm",
      "description": "Agent confirms order",
      "after": [
        "ask_size"
      ],
      "assert": {
        "type": "regex",
        "value": "(?i)confirm"
      }
    }
  ],
  "max_turns": 10
}
```

### Console Output (Dynamic Mode)

```
► [coffee-order] (dynamic, 3 checkpoints)
ℹ Dynamic test: coffee-order (max 10 turns)
ℹ   Turn 1: User: I want to order coffee
ℹ   Turn 1: Agent: Hello! What can I get for you?
ℹ     ✓ checkpoint: greeting
ℹ   Turn 2: User: A medium latte please
ℹ   Turn 2: Agent: What size would you like?
ℹ     ✓ checkpoint: ask_size
ℹ   Turn 3: User: Medium
ℹ   Turn 3: Agent: Let me confirm: Medium latte. Correct?
ℹ     ✓ checkpoint: confirm
  └─ PASSED (3 turns, 3 checkpoints, 8.5s)
```

### Dynamic Mode Output Structure

Each turn in the output includes:

```typescript
interface TurnResult {
  turn: number; // Turn number (1-based)
  input: string; // User message
  output: any; // Agent response summary (for display)
  response: {
    // Full agent response (for detailed analysis)
    content: string; // LLM text content
    tool_calls: [
      {
        // Tool calls made
        tool: string; // Tool name
        arguments: any; // Call arguments
        result: any; // Execution result
      }
    ];
    next: any; // Next hook data
  };
  checkpoints_reached: string[]; // Checkpoint IDs reached
  duration_ms: number; // Execution time
}
```

### Checkpoint Result Structure

Each checkpoint in the output includes:

```typescript
interface CheckpointResult {
  id: string; // Checkpoint identifier
  reached: boolean; // Whether checkpoint was reached
  reached_at_turn?: number; // Turn number when reached (if reached)
  required: boolean; // Whether checkpoint is required
  passed: boolean; // Whether assertion passed
  message?: string; // Assertion result message
  agent_validation?: {
    // Agent assertion details (for type: "agent")
    passed: boolean; // Validator's determination
    reason: string; // Explanation from validator
    criteria: string; // Validation criteria checked
    input: any; // Content sent to validator
    response: {
      // Raw validator response
      passed: boolean;
      reason: string;
    };
  };
}
```

**Note**: For agent-based assertions (`type: "agent"`), the `agent_validation` field provides full transparency into the validation process. The `input` field contains the combined output (agent text response + tool result messages) that was validated.

## Output Formats

Determined by `-o` file extension:

| Extension | Format   | Description            |
| --------- | -------- | ---------------------- |
| `.jsonl`  | JSONL    | Streaming (default)    |
| `.json`   | JSON     | Complete structured    |
| `.md`     | Markdown | Human-readable         |
| `.html`   | HTML     | Interactive web report |

## Stability Analysis

Run each test multiple times to measure consistency:

```bash
yao agent test -i tests/inputs.jsonl --runs 5 -o stability.json
```

| Pass Rate | Classification  |
| --------- | --------------- |
| 100%      | Stable          |
| 80-99%    | Mostly Stable   |
| 50-79%    | Unstable        |
| < 50%     | Highly Unstable |

## CI Integration

```bash
# Exit code: 0 = all passed, 1 = failures
yao agent test -i tests/inputs.jsonl --fail-fast

# Run with parallel execution
yao agent test -i tests/inputs.jsonl --parallel 4
```

### GitHub Actions Example

```yaml
- name: Run Agent Tests
  run: |
    yao agent test -i assistants/expense/tests/inputs.jsonl \
      -u ci-user -t ci-team \
      --runs 3 \
      -o report.json

- name: Run Dynamic Tests
  run: |
    yao agent test -i assistants/expense/tests/dynamic.jsonl \
      --simulator tests.simulator-agent \
      -v

- name: Run Script Tests
  run: |
    yao agent test -i scripts.expense.setup -v
```

## Format Rules Reference

| Context                | Format                   | Example                                   |
| ---------------------- | ------------------------ | ----------------------------------------- |
| `-i agents:xxx` (CLI)  | Colon prefix             | `agents:tests.generator`                  |
| `-i scripts:xxx` (CLI) | Colon prefix             | `scripts:tests.gen.Generate`              |
| `-i scripts.xxx` (CLI) | Dot prefix (test mode)   | `scripts.expense.setup`                   |
| JSONL assertion `use`  | Prefix required          | `"use": "agents:tests.validator"`         |
| JSONL `simulator.use`  | No prefix (agent only)   | `"use": "tests.simulator-agent"`          |
| `--simulator` flag     | No prefix (agent only)   | `--simulator tests.simulator-agent`       |
| `t.assert.Agent()`     | No prefix (method-bound) | `t.assert.Agent(resp, "tests.validator")` |
| JSONL `before/after`   | No prefix (in src/)      | `"before": "env_test.Before"`             |
| `--before/--after`     | No prefix (in src/)      | `--before env_test.BeforeAll`             |

**Script input modes**:

- `scripts.xxx` (dot) - Run script tests (`*_test.ts` functions)
- `scripts:xxx` (colon) - Generate test cases from a script

## Built-in Test Agents

The framework provides three specialized agents for testing:

### Generator Agent (`tests.generator-agent`)

Generates test cases based on target agent description.

**package.yao**:

```json
{
  "name": "Test Case Generator",
  "connector": "gpt-4o",
  "description": "Generates test cases for agent testing",
  "options": { "temperature": 0.7 },
  "automated": true
}
```

**prompts.yml**:

```yaml
- role: system
  content: |
    You are a test case generator. Generate test cases based on the target agent.

    ## Input Format
    - `target_agent`: Agent info (id, description, tools)
    - `count`: Number of test cases (default: 5)
    - `focus`: Focus area (e.g., "edge-cases", "happy-path")

    ## Output Format
    JSON array of test cases:
    [
      {
        "id": "test-id",
        "input": "User message",
        "assert": [{"type": "contains", "value": "expected"}]
      }
    ]
```

**Usage**:

```bash
yao agent test -i "agents:tests.generator-agent?count=10" -n assistants.expense
```

### Validator Agent (`tests.validator-agent`)

Validates agent responses for agent-driven assertions.

**package.yao**:

```json
{
  "name": "Response Validator",
  "connector": "gpt-4o",
  "description": "Validates responses against criteria",
  "options": { "temperature": 0 },
  "automated": true
}
```

**prompts.yml**:

```yaml
- role: system
  content: |
    You are a response validator. Evaluate whether the response meets the criteria.

    ## Input Format
    - `output`: The response to validate
    - `criteria`: The validation rules
    - `input`: Original input (optional)

    ## Output Format
    JSON object (no markdown):
    {"passed": true/false, "reason": "explanation"}

    ## Examples
    Input: {"output": "Paris is the capital", "criteria": "factually accurate"}
    Output: {"passed": true, "reason": "Statement is correct"}
```

**Usage in JSONL**:

```jsonl
{
  "id": "T001",
  "input": "Hello",
  "assert": {
    "type": "agent",
    "use": "agents:tests.validator-agent",
    "value": "Response should be friendly"
  }
}
```

**Usage in script tests**:

```typescript
t.assert.Agent(response, "tests.validator-agent", {
  criteria: "Response should be helpful",
});
```

### Simulator Agent (`tests.simulator-agent`)

Simulates user behavior for dynamic mode testing.

**package.yao**:

```json
{
  "name": "User Simulator",
  "connector": "gpt-4o",
  "description": "Simulates user behavior for dynamic testing",
  "options": { "temperature": 0.7 },
  "automated": true
}
```

**prompts.yml**:

```yaml
- role: system
  content: |
    You are a user simulator. Generate realistic user messages based on persona and goal.

    ## Input Format
    - `persona`: User description (e.g., "New employee")
    - `goal`: What user wants to achieve
    - `conversation`: Previous messages
    - `turn_number`: Current turn
    - `max_turns`: Maximum turns

    ## Output Format
    JSON object:
    {
      "message": "User response",
      "goal_achieved": false,
      "reasoning": "Strategy explanation"
    }

    ## Guidelines
    1. Stay in character
    2. Work toward the goal
    3. Be realistic (include natural variations)
    4. Set goal_achieved: true when done
```

**Usage in JSONL**:

```jsonl
{
  "id": "dynamic-test",
  "input": "I need help",
  "simulator": {
    "use": "tests.simulator-agent",
    "options": {
      "metadata": {
        "persona": "New employee",
        "goal": "Submit expense report"
      }
    }
  },
  "checkpoints": [
    {
      "id": "greeting",
      "assert": {
        "type": "regex",
        "value": "(?i)hello"
      }
    }
  ],
  "max_turns": 10
}
```

**Usage via CLI**:

```bash
yao agent test -i tests/dynamic.jsonl --simulator tests.simulator-agent
```

## Exit Codes

| Code | Description                                         |
| ---- | --------------------------------------------------- |
| 0    | All tests passed                                    |
| 1    | Tests failed, configuration error, or runtime error |
