# Agent Testing

A comprehensive testing framework for Yao AI agents with support for standard testing, dynamic (simulator-driven) testing, agent-driven assertions, and CI integration.

## Quick Start

```bash
# Test with direct message (auto-detect agent from current directory)
cd assistants/my-assistant
yao agent test -i "Hello, how are you?"

# Test with JSONL file
yao agent test -i tests/inputs.jsonl

# Generate HTML report
yao agent test -i tests/inputs.jsonl -o report.html

# Stability analysis (run each test 5 times)
yao agent test -i tests/inputs.jsonl --runs 5
```

## Input Modes

The `-i` flag supports multiple input modes:

| Mode | Example | Description |
|------|---------|-------------|
| Direct message | `-i "Hello"` | Single message test |
| JSONL file | `-i tests/inputs.jsonl` | Multiple test cases |
| Agent-driven | `-i "agents:tests.generator?count=10"` | Generate tests with agent |
| Script test | `-i scripts.expense.setup` | Test handler scripts |
| Script-generated | `-i "scripts:tests.gen.Generate"` | Generate tests from script |

## Test Case Format (JSONL)

### Basic Test

```jsonl
{"id": "greeting", "input": "Hello", "assert": {"type": "contains", "value": "Hi"}}
```

### With Conversation History

```jsonl
{
  "id": "multi-turn",
  "input": [
    {"role": "user", "content": "What's 2+2?"},
    {"role": "assistant", "content": "4"},
    {"role": "user", "content": "Multiply by 3"}
  ],
  "assert": {"type": "contains", "value": "12"}
}
```

### With File Attachments

```jsonl
{
  "id": "image-test",
  "input": {
    "role": "user",
    "content": [
      {"type": "text", "text": "Describe this image"},
      {"type": "image", "source": "file://fixtures/test.jpg"}
    ]
  }
}
```

## Assertions

### Static Assertions

| Type | Description | Example |
|------|-------------|---------|
| `equals` | Exact match | `{"type": "equals", "value": {"key": "val"}}` |
| `contains` | Output contains value | `{"type": "contains", "value": "keyword"}` |
| `not_contains` | Output does not contain | `{"type": "not_contains", "value": "error"}` |
| `regex` | Match regex pattern | `{"type": "regex", "value": "\\d+"}` |
| `json_path` | Extract and compare | `{"type": "json_path", "path": "$.field", "value": true}` |
| `type` | Check output type | `{"type": "type", "value": "object"}` |
| `tool_called` | Check tool was called | `{"type": "tool_called", "value": "setup"}` |
| `tool_result` | Check tool result | `{"type": "tool_result", "value": {"tool": "setup", "result": {"success": true}}}` |

### Agent-Driven Assertions

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

### Multiple Assertions

All assertions must pass:

```jsonl
{
  "id": "complete-check",
  "input": "Submit expense",
  "assert": [
    {"type": "contains", "value": "expense"},
    {"type": "not_contains", "value": "error"},
    {"type": "regex", "value": "(?i)(submitted|created)"}
  ]
}
```

## Dynamic Mode (Simulator)

For testing complex conversation flows with a user simulator:

```jsonl
{
  "id": "order-flow",
  "input": "I want to order coffee",
  "simulator": {
    "use": "tests.simulator-agent",
    "options": {
      "metadata": {
        "persona": "Customer",
        "goal": "Order a medium latte"
      }
    }
  },
  "checkpoints": [
    {
      "id": "greeting",
      "assert": {"type": "regex", "value": "(?i)(hello|hi)"}
    },
    {
      "id": "ask-size",
      "after": ["greeting"],
      "assert": {"type": "regex", "value": "(?i)size"}
    },
    {
      "id": "confirm",
      "after": ["ask-size"],
      "assert": {"type": "regex", "value": "(?i)confirm"}
    }
  ],
  "max_turns": 10
}
```

Run with:

```bash
yao agent test -i tests/dynamic.jsonl --simulator tests.simulator-agent -v
```

## Script Testing

Test agent handler scripts with the `t.assert` API:

```typescript
// assistants/my-assistant/src/setup_test.ts
import { SystemReady } from "./setup";

export function TestSystemReady(t: TestingT, ctx: Context) {
  const result = SystemReady(ctx);
  
  t.assert.True(result.success, "Should succeed");
  t.assert.Equal(result.status, "ready", "Status should be ready");
  t.assert.NotNil(result.data, "Data should not be nil");
}

export function TestWithAgentAssertion(t: TestingT, ctx: Context) {
  const response = Process("agents.my-assistant.Stream", ctx, messages);
  
  // Static assertion
  t.assert.Contains(response.content, "confirm");
  
  // Agent-driven assertion
  t.assert.Agent(response.content, "tests.validator-agent", {
    criteria: "Response should ask for confirmation"
  });
}
```

Run with:

```bash
yao agent test -i scripts.my-assistant.setup -v
```

### Available Assertions

| Method | Description |
|--------|-------------|
| `t.assert.True(value, msg)` | Assert value is true |
| `t.assert.False(value, msg)` | Assert value is false |
| `t.assert.Equal(a, b, msg)` | Assert a equals b |
| `t.assert.NotEqual(a, b, msg)` | Assert a not equals b |
| `t.assert.Nil(value, msg)` | Assert value is null/undefined |
| `t.assert.NotNil(value, msg)` | Assert value is not nil |
| `t.assert.Contains(s, sub, msg)` | Assert string contains substr |
| `t.assert.Len(arr, n, msg)` | Assert array/string length |
| `t.assert.Agent(resp, id, opts)` | Agent-driven assertion |

## Before/After Hooks

### Per-Test Hooks

```jsonl
{
  "id": "with-setup",
  "input": "Show my data",
  "before": "env_test.Before",
  "after": "env_test.After"
}
```

### Global Hooks

```bash
yao agent test -i tests/inputs.jsonl --before env_test.BeforeAll --after env_test.AfterAll
```

### Hook Implementation

```typescript
// assistants/my-assistant/src/env_test.ts

export function Before(ctx: Context, testCase: TestCase): any {
  const userId = Process("models.user.Create", { name: "Test User" });
  return { userId }; // Passed to After
}

export function After(ctx: Context, testCase: TestCase, result: TestResult, beforeData: any) {
  if (beforeData?.userId) {
    Process("models.user.Delete", beforeData.userId);
  }
}

export function BeforeAll(ctx: Context, testCases: TestCase[]): any {
  Process("models.migrate");
  return { initialized: true };
}

export function AfterAll(ctx: Context, results: TestResult[], beforeData: any) {
  const passed = results.filter(r => r.status === "passed").length;
  console.log(`Tests completed: ${passed}/${results.length} passed`);
}
```

## Custom Context

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
  }
}
```

Use with `--ctx`:

```bash
yao agent test -i scripts.my-assistant.setup --ctx tests/context.json -v
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-i` | Input: JSONL file, message, `agents:xxx`, or `scripts:xxx` | (required) |
| `-o` | Output file path | `output-{timestamp}.jsonl` |
| `-n` | Agent ID (optional, auto-detected) | auto-detect |
| `-a` | Application directory | auto-detect |
| `-e` | Environment file | - |
| `-c` | Override connector | agent default |
| `-u` | Test user ID | `test-user` |
| `-t` | Test team ID | `test-team` |
| `-r` | Reporter agent ID | built-in |
| `-v` | Verbose output | false |
| `--ctx` | Path to context JSON file | - |
| `--simulator` | Default simulator agent ID | - |
| `--before` | Global BeforeAll hook | - |
| `--after` | Global AfterAll hook | - |
| `--runs` | Runs per test (stability analysis) | 1 |
| `--run` | Regex pattern to filter tests | - |
| `--timeout` | Timeout per test | 2m |
| `--parallel` | Parallel test cases | 1 |
| `--fail-fast` | Stop on first failure | false |
| `--dry-run` | Generate tests without running | false |

## Output Formats

Determined by `-o` file extension:

| Extension | Format | Description |
|-----------|--------|-------------|
| `.jsonl` | JSONL | Streaming (default) |
| `.json` | JSON | Complete structured |
| `.md` | Markdown | Human-readable |
| `.html` | HTML | Interactive web report |

## Stability Analysis

Run each test multiple times to measure consistency:

```bash
yao agent test -i tests/inputs.jsonl --runs 5 -o stability.json
```

| Pass Rate | Classification |
|-----------|----------------|
| 100% | Stable |
| 80-99% | Mostly Stable |
| 50-79% | Unstable |
| < 50% | Highly Unstable |

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
    yao agent test -i assistants/my-assistant/tests/inputs.jsonl \
      -u ci-user -t ci-team \
      --runs 3 \
      -o report.json

- name: Run Dynamic Tests
  run: |
    yao agent test -i assistants/my-assistant/tests/dynamic.jsonl \
      --simulator tests.simulator-agent \
      -v

- name: Run Script Tests
  run: |
    yao agent test -i scripts.my-assistant.setup -v
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | All tests passed |
| 1 | Tests failed, configuration error, or runtime error |
