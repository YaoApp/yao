# Agent Test Package Design

## Overview

Agent Test Package provides a framework for testing AI agents with structured test cases.
It supports batch testing, report generation, stability analysis, and CI integration.

Additionally, it supports **Script Testing** for testing Agent handler scripts (hooks, tools, etc.) with a Go-like testing interface.

### Quick Start

```bash
# Quick test with a single message (auto-detect agent from current directory)
cd assistants/keyword
yao agent test -i "hello world"

# Or specify agent explicitly
yao agent test -i "hello world" -n keyword.agent

# Run tests from JSONL file (auto-detect agent from path)
yao agent test -i assistants/keyword/tests/inputs.jsonl

# Run with stability analysis (5 runs per test case)
yao agent test -i assistants/keyword/tests/inputs.jsonl --runs 5

# Generate HTML report
yao agent test -i assistants/keyword/tests/inputs.jsonl -r report.html -o report.html

# Run script tests (test agent handler scripts)
yao agent test -i scripts.expense.setup -v

# Run script tests with specific user/team context
yao agent test -i scripts.expense.tools -u admin -t ops-team -v
```

## Usage

```bash
# Basic usage - auto-detect agent, output to same directory as input
# Output: tests/output-20241217100000.jsonl
yao agent test -i tests/inputs.jsonl

# Override connector
yao agent test -i tests/inputs.jsonl -c openai.gpt4

# Specify agent explicitly
yao agent test -i tests/inputs.jsonl -n my.agent

# Specify test environment (user and team)
yao agent test -i tests/inputs.jsonl -u test-user -t test-team

# Run multiple times for stability analysis
yao agent test -i tests/inputs.jsonl --runs 5

# Custom timeout per test case (default: 5m)
yao agent test -i tests/inputs.jsonl --timeout 10m

# Run tests in parallel (4 concurrent test cases)
yao agent test -i tests/inputs.jsonl --parallel 4

# Combine parallel and timeout for faster execution
yao agent test -i tests/inputs.jsonl --parallel 8 --timeout 2m

# Custom output file path
yao agent test -i tests/inputs.jsonl -o /path/to/results.jsonl

# Use custom reporter agent for personalized report (HTML)
yao agent test -i tests/inputs.jsonl -r report.html -o report.html

# Use custom reporter agent for personalized report (Markdown)
yao agent test -i tests/inputs.jsonl -r report.markdown -o report.md

# Full example with all options
yao agent test -i tests/inputs.jsonl \
  -n keyword.agent \
  -c deepseek.v3 \
  -u test-user \
  -t test-team \
  --runs 3 \
  --timeout 10m \
  --parallel 4 \
  -r report.html \
  -o report.html
```

### Input Modes

The `-i` flag supports three input modes:

**1. JSONL File Mode** - Load test cases from a file:

```bash
yao agent test -i tests/inputs.jsonl
```

**2. Direct Message Mode** - Test with a single message:

```bash
# Auto-detect agent from current working directory
cd assistants/keyword
yao agent test -i "Extract keywords from this text"

# Or specify agent explicitly
yao agent test -i "Extract keywords from this text" -n keyword.agent
yao agent test -i "你好世界" -n keyword.agent -c deepseek.v3
```

When using direct message mode:

- Agent is resolved from current working directory (looks for `package.yao` upward)
- If not found, use `-n` flag to specify agent explicitly
- Output is printed to stdout (or saved to `-o` if specified)
- Useful for quick testing and debugging

**3. Script Test Mode** - Test agent handler scripts:

```bash
# Run all tests in a script module
yao agent test -i scripts.expense.setup -v

# Run with specific user/team context
yao agent test -i scripts.expense.tools -u admin -t ops-team

# Run with timeout
yao agent test -i scripts.expense.setup --timeout 30s

# Run specific tests by pattern (like go test -run)
yao agent test -i scripts.expense.setup -run TestSystemReady

# Run tests matching a regex pattern
yao agent test -i scripts.expense.setup -run "TestSystem.*"
```

When using script test mode:

- Input starts with `scripts.` prefix to indicate script testing
- Maps to the script file (e.g., `scripts.expense.setup` → `expense/src/setup_test.ts`)
- Automatically discovers and runs all `Test*` functions in the script
- Uses Go-like testing interface with assertions
- See [Script Testing](#script-testing) section for details

### Default Output Path

When `-o` is not specified and using JSONL file mode, the output file is automatically generated in the same directory as the input file:

```
{input_directory}/output-{timestamp}.jsonl
```

Example:

- Input: `/app/assistants/keyword/tests/inputs.jsonl`
- Output: `/app/assistants/keyword/tests/output-20241217100000.jsonl`

The timestamp format is `YYYYMMDDHHMMSS` (e.g., `20241217100000` for 2024-12-17 10:00:00).

When using direct message mode without `-o`, output is printed to stdout.

## Command Line Options

| Flag | Long Flag     | Description                           | Default                    | Example                                 |
| ---- | ------------- | ------------------------------------- | -------------------------- | --------------------------------------- |
| `-i` | `--input`     | Input: JSONL file path or message     | -                          | `-i tests/inputs.jsonl` or `-i "hello"` |
| `-o` | `--output`    | Path to output file (format by ext)   | `output-{timestamp}.jsonl` | `-o report.html`                        |
| `-n` | `--name`      | Explicit agent ID                     | auto-detect                | `-n keyword.agent`                      |
| `-c` | `--connector` | Override connector                    | agent default              | `-c openai.gpt4`                        |
| `-u` | `--user`      | Test user ID (global override)        | "test-user"                | `-u admin`                              |
| `-t` | `--team`      | Test team ID (global override)        | "test-team"                | `-t ops-team`                           |
|      | `--ctx`       | Path to context JSON file             | -                          | `--ctx tests/context.json`              |
| `-r` | `--reporter`  | Custom reporter agent ID              | - (use built-in)           | `-r report.beautiful`                   |
|      | `--runs`      | Number of runs for stability analysis | 1                          | `--runs 5`                              |
|      | `--run`       | Regex pattern to filter tests         | -                          | `--run "TestSystem.*"`                  |
|      | `--timeout`   | Default timeout per test case         | 5m                         | `--timeout 10m`                         |
|      | `--parallel`  | Number of parallel test cases         | 1                          | `--parallel 4`                          |
| `-v` | `--verbose`   | Verbose output                        | false                      | `-v`                                    |
|      | `--fail-fast` | Stop on first failure                 | false                      | `--fail-fast`                           |

**Notes**:

- Without `-o` flag, output is saved to `{input_dir}/output-{timestamp}.jsonl`
- Output format is determined by `-o` file extension: `.jsonl`, `.json`, `.md`, `.html`
- Use `-r` to specify a custom reporter agent for personalized report generation

## Agent Resolution

The agent is resolved in the following order:

1. **Explicit specification** (`-n` flag): Use the specified agent ID
2. **Path-based detection**: Traverse up from `tests/inputs.jsonl` to find `package.yao`

### Path-based Detection Example

```
/app/assistants/workers/system/keyword/
├── package.yao          <- Agent definition
├── prompts.yml
├── src/
│   └── index.ts
└── tests/
    └── inputs.jsonl     <- Test input file
```

Given input path `/app/assistants/workers/system/keyword/tests/inputs.jsonl`:

1. Check `/app/assistants/workers/system/keyword/tests/package.yao` - not found
2. Check `/app/assistants/workers/system/keyword/package.yao` - **found!**
3. Load agent from `/app/assistants/workers/system/keyword/`

## Test Environment

Agent calls require a `Context` with user and tenant information. The test framework creates a test context with configurable environment:

```go
// TestEnvironment configures the test execution context
type TestEnvironment struct {
    UserID     string // User ID for authorized info (-u flag)
    TeamID     string // Team ID for authorized info (-t flag)
    Locale     string // Locale (default: "en-us")
    ClientType string // Client type (default: "test")
    ClientIP   string // Client IP (default: "127.0.0.1")
    Referer    string // Request referer (default: "test")
    Accept     string // Accept format (default: "standard")
}
```

Example context creation (similar to `agent_next_test.go`):

```go
func newTestContext(env *TestEnvironment, chatID, assistantID string) *context.Context {
    authorized := &types.AuthorizedInfo{
        Subject: env.UserID,
        UserID:  env.UserID,
        TeamID:  env.TeamID,
    }
    ctx := context.New(stdContext.Background(), authorized, chatID)
    ctx.AssistantID = assistantID
    ctx.Locale = env.Locale
    ctx.Client = context.Client{
        Type: env.ClientType,
        IP:   env.ClientIP,
    }
    ctx.Referer = env.Referer
    ctx.Accept = env.Accept
    return ctx
}
```

## Stability Analysis (Multiple Runs)

When `--runs N` is specified (N > 1), the framework runs each test case N times and collects stability metrics:

### Stability Metrics

| Metric             | Description                                |
| ------------------ | ------------------------------------------ |
| `pass_rate`        | Percentage of runs that passed (0-100%)    |
| `consistency`      | How consistent the outputs are across runs |
| `avg_duration_ms`  | Average execution time                     |
| `min_duration_ms`  | Minimum execution time                     |
| `max_duration_ms`  | Maximum execution time                     |
| `std_deviation_ms` | Standard deviation of execution time       |

### Stability Report Structure

```json
{
  "summary": {
    "total_cases": 42,
    "total_runs": 126,
    "runs_per_case": 3,
    "overall_pass_rate": 95.2,
    "stable_cases": 38,
    "unstable_cases": 4,
    "duration_ms": 45678
  },
  "results": [
    {
      "id": "T001",
      "runs": 3,
      "passed": 3,
      "failed": 0,
      "pass_rate": 100.0,
      "consistency": 1.0,
      "stable": true,
      "avg_duration_ms": 234,
      "min_duration_ms": 210,
      "max_duration_ms": 256,
      "std_deviation_ms": 18.5,
      "run_details": [
        {"run": 1, "status": "passed", "duration_ms": 234, "output": {...}},
        {"run": 2, "status": "passed", "duration_ms": 210, "output": {...}},
        {"run": 3, "status": "passed", "duration_ms": 256, "output": {...}}
      ]
    },
    {
      "id": "T002",
      "runs": 3,
      "passed": 2,
      "failed": 1,
      "pass_rate": 66.7,
      "consistency": 0.67,
      "stable": false,
      "run_details": [...]
    }
  ]
}
```

### Stability Classification

| Pass Rate | Classification  |
| --------- | --------------- |
| 100%      | Stable          |
| 80-99%    | Mostly Stable   |
| 50-79%    | Unstable        |
| < 50%     | Highly Unstable |

## Custom Reporter Agent

By default, the framework outputs JSONL format. You can specify a reporter agent (`-r` flag) for personalized report generation:

### Reporter Agent Interface

The reporter agent receives the test results and generates a custom report:

```json
// Input to reporter agent
{
  "report": {
    "summary": {...},
    "results": [...],
    "metadata": {...}
  },
  "format": "html",  // or "markdown", "text"
  "options": {
    "verbose": true,
    "include_outputs": true
  }
}
```

### Built-in Reporter Agents

| Agent ID          | Description                            |
| ----------------- | -------------------------------------- |
| `report.json`     | JSON format (default, no agent needed) |
| `report.markdown` | Markdown format with tables            |
| `report.html`     | Interactive HTML report                |
| `report.summary`  | Brief text summary                     |

### Custom Reporter Example

Create a custom reporter agent at `assistants/reporters/my-reporter/`:

```yaml
# prompts.yml
- role: system
  content: |
    You are a test report generator. Generate a beautiful report from test results.

    Output format: HTML with embedded CSS

    Requirements:
    - Show summary statistics prominently
    - Use color coding (green=pass, red=fail)
    - Include charts for stability metrics
    - Make it printable
```

## Script Testing

Script Testing allows you to test Agent handler scripts (hooks, tools, setup functions, etc.) using a Go-like testing interface. This is useful for unit testing individual functions in your agent's TypeScript/JavaScript code.

### Quick Start

```bash
# Run script tests
yao agent test -i scripts.expense.setup -v

# With user/team context
yao agent test -i scripts.expense.setup -u admin -t ops-team -v

# With timeout
yao agent test -i scripts.expense.setup --timeout 30s -v
```

### Script Resolution

The `scripts.` prefix indicates script test mode. The script is resolved as follows:

| Input                   | Script Path            | Test File                   |
| ----------------------- | ---------------------- | --------------------------- |
| `scripts.expense.setup` | `expense/src/setup.ts` | `expense/src/setup_test.ts` |
| `scripts.expense.tools` | `expense/src/tools.ts` | `expense/src/tools_test.ts` |
| `scripts.keyword.index` | `keyword/src/index.ts` | `keyword/src/index_test.ts` |

The test file naming convention is `{module}_test.ts` (similar to Go's `_test.go` convention).

### Test Function Signature

Test functions must follow this signature:

```typescript
function TestFunctionName(t: testing.T, ctx: agent.Context) {
  // Test logic here
}
```

**Requirements:**

- Function name must start with `Test` (case-sensitive)
- First parameter `t` is the testing object with assertions
- Second parameter `ctx` is the agent context (same as used in hooks/tools)
- Functions not starting with `Test` are ignored (can be used as helpers)

### Example Test File

```typescript
// setup_test.ts
// @ts-nocheck

// Test the SystemReady function
function TestSystemReady(t: testing.T, ctx: agent.Context) {
  const { assert } = t;

  // Call the function being tested
  const result = SystemReady(ctx);

  // Assert the result
  assert.True(result, "SystemReady should return true");
}

// Test error case
function TestSystemReadyWithInvalidContext(t: testing.T, ctx: agent.Context) {
  const { assert } = t;

  // Modify context to simulate error condition
  ctx.User = null;

  const result = SystemReady(ctx);
  assert.False(result, "SystemReady should return false when user is null");
}

// Helper function (not a test - doesn't start with "Test")
function createMockData() {
  return { id: 1, name: "test" };
}

// Test with helper
function TestSetupWithMockData(t: testing.T, ctx: agent.Context) {
  const { assert } = t;
  const mockData = createMockData();

  const result = Setup(ctx, mockData);
  assert.NotNil(result, "Setup should return a result");
  assert.Equal(result.id, 1, "Result ID should match");
}
```

### Testing Object (`t`)

The `t` parameter provides the testing interface:

```typescript
interface testing.T {
  // Assertions object
  assert: testing.Assert;

  // Test metadata
  name: string;        // Current test function name
  failed: boolean;     // Whether the test has failed

  // Logging (output appears in test report)
  log(...args: any[]): void;      // Log info message
  error(...args: any[]): void;    // Log error message

  // Control flow
  skip(reason?: string): void;    // Skip this test
  fail(reason?: string): void;    // Mark test as failed
  fatal(reason?: string): void;   // Mark as failed and stop execution
}
```

### Assertions (`t.assert`)

The `assert` object provides assertion methods:

| Method                              | Description                        |
| ----------------------------------- | ---------------------------------- |
| `True(value, message?)`             | Assert value is true               |
| `False(value, message?)`            | Assert value is false              |
| `Equal(actual, expected, message?)` | Assert deep equality               |
| `NotEqual(actual, expected, msg?)`  | Assert not equal                   |
| `Nil(value, message?)`              | Assert value is null/undefined     |
| `NotNil(value, message?)`           | Assert value is not null/undefined |
| `Contains(str, substr, message?)`   | Assert string contains substring   |
| `NotContains(str, substr, msg?)`    | Assert string does not contain     |
| `Len(value, length, message?)`      | Assert array/string length         |
| `Greater(a, b, message?)`           | Assert a > b                       |
| `GreaterOrEqual(a, b, message?)`    | Assert a >= b                      |
| `Less(a, b, message?)`              | Assert a < b                       |
| `LessOrEqual(a, b, message?)`       | Assert a <= b                      |
| `Error(err, message?)`              | Assert err is an error             |
| `NoError(err, message?)`            | Assert err is null/undefined       |
| `Panic(fn, message?)`               | Assert function throws             |
| `NoPanic(fn, message?)`             | Assert function does not throw     |
| `Match(value, pattern, message?)`   | Assert value matches regex         |
| `NotMatch(value, pattern, msg?)`    | Assert value does not match regex  |
| `JSONPath(obj, path, expected, m?)` | Assert JSON path value             |
| `Type(value, typeName, message?)`   | Assert value type                  |

### Agent Context (`ctx`)

The `ctx` parameter is the same `agent.Context` used in agent hooks and tools:

```typescript
interface agent.Context {
  // User information (from -u flag or default)
  User: {
    ID: string;
    Name?: string;
  };

  // Team information (from -t flag or default)
  Team: {
    ID: string;
    Name?: string;
  };

  // Locale (default: "en-us")
  Locale: string;

  // Client information
  Client: {
    Type: string;    // "test"
    IP: string;      // "127.0.0.1"
  };

  // Metadata (can be set via test case)
  Metadata: Record<string, any>;

  // Chat/Session ID
  ChatID: string;

  // Assistant ID (resolved from script path)
  AssistantID: string;
}
```

### Script Test Output

Script test results are reported in the same format as agent tests:

```
═══════════════════════════════════════════════════════════════════════════════
  Script Test: scripts.expense.setup
═══════════════════════════════════════════════════════════════════════════════
  Script: expense/src/setup_test.ts
  Tests: 3 functions
  User: test-user
  Team: test-team
───────────────────────────────────────────────────────────────────────────────
  Running Tests
───────────────────────────────────────────────────────────────────────────────
► [TestSystemReady] ...
  ✓ PASSED (12ms)

► [TestSystemReadyWithInvalidContext] ...
  ✓ PASSED (8ms)

► [TestSetupWithMockData] ...
  ✗ FAILED (15ms)
    └─ assertion failed: Result ID should match
       expected: 1
       actual: 2

═══════════════════════════════════════════════════════════════════════════════
  Summary: 2 passed, 1 failed, 0 skipped (35ms)
═══════════════════════════════════════════════════════════════════════════════
```

### Script Test Options

Script tests support the following command line options:

| Flag          | Description                      | Default     | Example              |
| ------------- | -------------------------------- | ----------- | -------------------- |
| `-u`          | User ID for context              | "test-user" | `-u admin`           |
| `-t`          | Team ID for context              | "test-team" | `-t ops-team`        |
| `--ctx`       | Path to context JSON file        | -           | `--ctx context.json` |
| `-v`          | Verbose output                   | false       | `-v`                 |
| `--run`       | Regex to filter tests            | -           | `--run "TestSystem"` |
| `--timeout`   | Timeout per test function        | 30s         | `--timeout 1m`       |
| `--fail-fast` | Stop on first failure            | false       | `--fail-fast`        |
| `-o`          | Output file for report           | stdout      | `-o report.json`     |
| `-r`          | Reporter agent for custom report | -           | `-r report.html`     |

The `--run` flag accepts a Go-style regex pattern to filter which tests to run:

```bash
# Run only TestSystemReady
yao agent test -i scripts.expense.setup --run TestSystemReady

# Run all tests starting with "TestSystem"
yao agent test -i scripts.expense.setup --run "TestSystem.*"

# Run tests containing "Error"
yao agent test -i scripts.expense.setup --run ".*Error.*"
```

### Custom Context Configuration

The `--ctx` flag allows you to provide a JSON file with custom context configuration, giving full control over authorization data, metadata, and client information:

```bash
# Use custom context file
yao agent test -i scripts.expense.setup --ctx tests/context.json -v
```

**Context JSON Format:**

```json
{
  "authorized": {
    "sub": "user-12345",
    "client_id": "my-app",
    "scope": "read write",
    "session_id": "sess-abc123",
    "user_id": "admin",
    "team_id": "team-001",
    "tenant_id": "acme-corp",
    "remember_me": false,
    "constraints": {
      "owner_only": false,
      "creator_only": false,
      "editor_only": false,
      "team_only": true,
      "extra": {
        "department": "engineering",
        "region": "us-west"
      }
    }
  },
  "metadata": {
    "request_id": "req-123",
    "trace_id": "trace-456",
    "custom_field": "custom_value"
  },
  "client": {
    "type": "web",
    "user_agent": "Mozilla/5.0",
    "ip": "192.168.1.100"
  },
  "locale": "zh-cn",
  "referer": "https://example.com/dashboard"
}
```

**Field Descriptions:**

| Field                      | Description                                         |
| -------------------------- | --------------------------------------------------- |
| `authorized.sub`           | Subject identifier (JWT sub claim)                  |
| `authorized.client_id`     | OAuth client ID                                     |
| `authorized.scope`         | Access scope                                        |
| `authorized.session_id`    | Session identifier                                  |
| `authorized.user_id`       | User identifier (overrides -u flag)                 |
| `authorized.team_id`       | Team identifier (overrides -t flag)                 |
| `authorized.tenant_id`     | Tenant identifier                                   |
| `authorized.remember_me`   | Remember me flag                                    |
| `authorized.constraints`   | Data access constraints (set by ACL enforcement)    |
| `constraints.owner_only`   | Only access owner's data                            |
| `constraints.creator_only` | Only access creator's data                          |
| `constraints.editor_only`  | Only access editor's data                           |
| `constraints.team_only`    | Only access team's data (filter by team_id)         |
| `constraints.extra`        | User-defined constraints (department, region, etc.) |
| `metadata`                 | Custom metadata passed to context                   |
| `client.type`              | Client type (web, mobile, test, etc.)               |
| `client.user_agent`        | Client user agent string                            |
| `client.ip`                | Client IP address                                   |
| `locale`                   | Locale setting (e.g., "en-us", "zh-cn")             |
| `referer`                  | Request referer URL                                 |

**Priority:** When both `-u`/`-t` flags and `--ctx` file are provided, the context file values take precedence.

### Script Test Report Format

When using `-o` to save results:

```json
{
  "type": "script_test",
  "script": "scripts.expense.setup",
  "script_path": "expense/src/setup_test.ts",
  "summary": {
    "total": 3,
    "passed": 2,
    "failed": 1,
    "skipped": 0,
    "duration_ms": 35
  },
  "environment": {
    "user_id": "test-user",
    "team_id": "test-team",
    "locale": "en-us"
  },
  "results": [
    {
      "name": "TestSystemReady",
      "status": "passed",
      "duration_ms": 12,
      "logs": []
    },
    {
      "name": "TestSystemReadyWithInvalidContext",
      "status": "passed",
      "duration_ms": 8,
      "logs": []
    },
    {
      "name": "TestSetupWithMockData",
      "status": "failed",
      "duration_ms": 15,
      "error": "assertion failed: Result ID should match",
      "assertion": {
        "type": "Equal",
        "expected": 1,
        "actual": 2,
        "message": "Result ID should match"
      },
      "logs": []
    }
  ],
  "metadata": {
    "started_at": "2024-12-17T10:00:00Z",
    "completed_at": "2024-12-17T10:00:00Z",
    "version": "0.10.5"
  }
}
```

### Best Practices

1. **Naming Convention**: Use descriptive test names that explain what's being tested

   - Good: `TestSystemReadyWithValidUser`, `TestSetupReturnsErrorOnMissingConfig`
   - Bad: `Test1`, `TestIt`

2. **One Assertion Per Concept**: Each test should verify one behavior

   ```typescript
   // Good: Focused tests
   function TestSetupCreatesDatabase(t, ctx) { ... }
   function TestSetupInitializesCache(t, ctx) { ... }

   // Bad: Testing too many things
   function TestSetup(t, ctx) {
     // tests database, cache, config, etc.
   }
   ```

3. **Use Helper Functions**: Extract common setup logic

   ```typescript
   function setupTestContext(ctx) {
     ctx.Metadata.testMode = true;
     return ctx;
   }

   function TestFeatureA(t, ctx) {
     ctx = setupTestContext(ctx);
     // ...
   }
   ```

4. **Test Error Cases**: Don't just test happy paths

   ```typescript
   function TestSetupWithMissingConfig(t, ctx) {
     const { assert } = t;
     ctx.Metadata.config = null;

     const result = Setup(ctx);
     assert.Error(result.error, "Should return error for missing config");
   }
   ```

5. **Clean Up**: If your test modifies global state, clean up after
   ```typescript
   function TestWithGlobalState(t, ctx) {
     const originalValue = GlobalConfig.value;
     try {
       GlobalConfig.value = "test";
       // ... test logic
     } finally {
       GlobalConfig.value = originalValue;
     }
   }
   ```

## Input Format (JSONL)

Each line in the input file is a JSON object with the following structure:

```jsonl
{"id": "T001", "input": "Simple text input"}
{"id": "T002", "input": {"role": "user", "content": "Message with role"}}
{"id": "T003", "input": {"role": "user", "content": [{"type": "text", "text": "ContentPart array"}]}}
{"id": "T004", "input": [{"role": "user", "content": "First message"}, {"role": "assistant", "content": "Response"}, {"role": "user", "content": "Follow-up"}]}
{"id": "T005", "input": "Text input", "expected": {"keywords": ["keyword1", "keyword2"]}}
{"id": "T006", "input": "Test with specific user", "user": "admin", "team": "ops-team"}
```

### Input Types

| Type        | Description              | Example                                               |
| ----------- | ------------------------ | ----------------------------------------------------- |
| `string`    | Simple text input        | `"Hello world"`                                       |
| `Message`   | Single message with role | `{"role": "user", "content": "..."}`                  |
| `[]Message` | Conversation history     | `[{"role": "user", ...}, {"role": "assistant", ...}]` |

### Fields

| Field      | Type                           | Required | Description                                          |
| ---------- | ------------------------------ | -------- | ---------------------------------------------------- |
| `id`       | string                         | Yes      | Unique test case identifier (e.g., "T001")           |
| `input`    | string \| Message \| []Message | Yes      | Test input                                           |
| `expected` | any                            | No       | Expected output for exact match validation           |
| `assert`   | Assertion \| []Assertion       | No       | Custom assertion rules (see Assertions section)      |
| `user`     | string                         | No       | User ID for this test case (overridden by `-u` flag) |
| `team`     | string                         | No       | Team ID for this test case (overridden by `-t` flag) |
| `metadata` | map                            | No       | Additional metadata for the test case                |
| `skip`     | bool                           | No       | Skip this test case                                  |
| `timeout`  | string                         | No       | Override timeout (e.g., "30s", "1m")                 |

### Assertions

The `assert` field allows flexible validation of agent output. If `assert` is defined, it takes precedence over `expected`.

#### Assertion Types

| Type           | Description                                     | Example                                                          |
| -------------- | ----------------------------------------------- | ---------------------------------------------------------------- |
| `equals`       | Exact match (default if only `expected` is set) | `{"type": "equals", "value": {"need_search": false}}`            |
| `contains`     | Output contains the expected string/value       | `{"type": "contains", "value": "keyword"}`                       |
| `not_contains` | Output does not contain the string/value        | `{"type": "not_contains", "value": "error"}`                     |
| `json_path`    | Extract value using JSON path and compare       | `{"type": "json_path", "path": "$.need_search", "value": false}` |
| `regex`        | Match output against regex pattern              | `{"type": "regex", "value": "\\d{3}-\\d{4}"}`                    |
| `type`         | Check output type (string, object, array, etc.) | `{"type": "type", "value": "object"}`                            |
| `script`       | Run a custom assertion script                   | `{"type": "script", "script": "scripts.test.Assert"}`            |

#### Assertion Structure

```typescript
interface Assertion {
  type: string; // Assertion type (required)
  value?: any; // Expected value or pattern
  path?: string; // JSON path for json_path assertions
  script?: string; // Script name for script assertions
  message?: string; // Custom failure message
  negate?: boolean; // Invert the assertion result
}
```

#### Examples

**Simple contains check:**

```jsonl
{
  "id": "T001",
  "input": "Hello",
  "assert": {
    "type": "contains",
    "value": "need_search"
  }
}
```

**JSON path validation (for agents returning JSON):**

```jsonl
{
  "id": "T002",
  "input": "What's the weather?",
  "assert": {
    "type": "json_path",
    "path": "$.need_search",
    "value": true
  }
}
```

**Multiple assertions (all must pass):**

```jsonl
{
  "id": "T003",
  "input": "Calculate 2+2",
  "assert": [
    {
      "type": "json_path",
      "path": "$.need_search",
      "value": false
    },
    {
      "type": "json_path",
      "path": "$.confidence",
      "value": 0.99
    },
    {
      "type": "not_contains",
      "value": "error"
    }
  ]
}
```

**Custom script assertion:**

```jsonl
{
  "id": "T004",
  "input": "Complex test",
  "assert": {
    "type": "script",
    "script": "scripts.test.ValidateOutput"
  }
}
```

The script receives `(output, input, expected)` and should return:

```typescript
// Simple boolean
return true; // or false

// Or detailed result
return {
  pass: true,
  message: "Validation passed: output contains expected keywords",
};
```

**Negated assertion:**

```jsonl
{
  "id": "T005",
  "input": "Hello",
  "assert": {
    "type": "contains",
    "value": "error",
    "negate": true
  }
}
```

#### JSON Path Notes

- Supports simple dot-notation paths: `$.field.subfield` or `field.subfield`
- Automatically extracts JSON from markdown code blocks (e.g., ` ```json ... ``` `)
- Works with both string output and structured objects

### Environment Override Priority

The test environment (user/team) is determined by the following priority (highest first):

1. **Command line flags** (`-u`, `-t`): Global override for all test cases
2. **Test case fields** (`user`, `team`): Per-test case configuration
3. **Default values**: "test-user", "test-team"

Example:

```bash
# All tests run as "admin" user in "prod-team", regardless of test case settings
yao agent test -i tests/inputs.jsonl -u admin -t prod-team -o report.json
```

```jsonl
# T001 uses default user/team
{"id": "T001", "input": "Hello"}

# T002 uses specific user/team (unless overridden by -u/-t flags)
{"id": "T002", "input": "Admin action", "user": "admin", "team": "admin-team"}

# T003 uses specific user only, team uses default
{"id": "T003", "input": "User specific test", "user": "special-user"}
```

## Output Format

### Default: JSONL (without `-r` flag)

By default (without `-r` flag), the output is JSONL format - one JSON object per line, suitable for streaming and CI integration:

```jsonl
{"type": "start", "timestamp": "2024-12-17T10:00:00Z", "agent_id": "keyword", "total_cases": 42}
{"type": "result", "id": "T001", "status": "passed", "duration_ms": 234, "output": {"keywords": ["AI", "ML"]}}
{"type": "result", "id": "T002", "status": "passed", "duration_ms": 189, "output": {"keywords": ["cloud"]}}
{"type": "result", "id": "T003", "status": "failed", "duration_ms": 0, "error": "timeout after 30s"}
{"type": "summary", "total": 42, "passed": 40, "failed": 2, "duration_ms": 12345}
```

This format is:

- **Streamable**: Results are output as they complete
- **Parseable**: Each line is valid JSON, easy to process with `jq` or scripts
- **CI-friendly**: Exit code indicates pass/fail status

### Custom Report (with `-r` flag)

```json
{
  "summary": {
    "total": 42,
    "passed": 40,
    "failed": 2,
    "skipped": 0,
    "duration_ms": 12345,
    "agent_id": "keyword",
    "connector": "deepseek.v3",
    "runs_per_case": 1,
    "overall_pass_rate": 95.2
  },
  "environment": {
    "user_id": "test-user",
    "team_id": "test-team",
    "locale": "en-us"
  },
  "results": [
    {
      "id": "T001",
      "status": "passed",
      "input": "...",
      "output": { "keywords": ["AI", "machine learning"] },
      "expected": null,
      "duration_ms": 234,
      "error": null
    }
  ],
  "metadata": {
    "started_at": "2024-12-17T10:00:00Z",
    "completed_at": "2024-12-17T10:00:12Z",
    "version": "0.10.5"
  }
}
```

### HTML Report

Beautiful, interactive HTML report with:

- Summary statistics (pass/fail/skip counts, duration)
- Stability charts (when runs > 1)
- Filterable test results table
- Expandable input/output details
- Error highlighting
- Export options

### Markdown Report

```markdown
# Agent Test Report

## Summary

| Metric    | Value       |
| --------- | ----------- |
| Agent     | keyword     |
| Connector | deepseek.v3 |
| Total     | 42          |
| Passed    | 40          |
| Failed    | 2           |
| Pass Rate | 95.2%       |
| Duration  | 12.3s       |

## Environment

| Setting | Value     |
| ------- | --------- |
| User    | test-user |
| Team    | test-team |
| Locale  | en-us     |

## Results

### ✅ T001 - Passed (234ms)

...
```

## Architecture

```
agent/test/
├── DESIGN.md           # This file
├── types.go            # Core types and interfaces
├── interfaces.go       # Runner and Reporter interfaces
├── runner.go           # Test runner implementation
├── loader.go           # Test case loader
├── resolver.go         # Agent resolver
├── context.go          # Test context creation
├── assert.go           # Assertion implementation
├── input.go            # Input parsing
├── output.go           # Output formatting
├── script.go           # Script test runner (NEW)
├── script_types.go     # Script test types (NEW)
├── script_assert.go    # Script assertion bindings (NEW)
└── reporter/
    ├── json.go         # JSON reporter
    ├── html.go         # HTML reporter
    ├── markdown.go     # Markdown reporter
    └── agent.go        # Agent-based custom reporter
```

## Core Components

### 1. TestCase

Represents a single test case loaded from JSONL.

### 2. TestResult

Represents the result of running a single test case.

### 3. TestReport

Represents the complete test report with summary and results.

### 4. Runner

Executes test cases against an agent:

- Loads test cases from JSONL
- Resolves agent from path or explicit ID
- Creates test context with environment
- Executes each test case (optionally multiple runs)
- Collects results and stability metrics

### 5. ScriptRunner (NEW)

Executes script tests for agent handler scripts:

- Resolves script path from `scripts.` prefix
- Discovers `Test*` functions in the script
- Creates test context with environment
- Executes each test function with testing object and context
- Collects results and generates report

### 6. ScriptTestCase (NEW)

Represents a single script test function:

```go
type ScriptTestCase struct {
    Name     string // Function name (e.g., "TestSystemReady")
    Function string // Full function reference
}
```

### 7. ScriptTestResult (NEW)

Represents the result of running a script test function:

```go
type ScriptTestResult struct {
    Name       string        `json:"name"`
    Status     Status        `json:"status"`
    DurationMs int64         `json:"duration_ms"`
    Error      string        `json:"error,omitempty"`
    Assertion  *AssertionInfo `json:"assertion,omitempty"`
    Logs       []string      `json:"logs,omitempty"`
}

type AssertionInfo struct {
    Type     string      `json:"type"`
    Expected interface{} `json:"expected,omitempty"`
    Actual   interface{} `json:"actual,omitempty"`
    Message  string      `json:"message,omitempty"`
}
```

### 8. ScriptTestReport (NEW)

Represents the complete script test report:

```go
type ScriptTestReport struct {
    Type        string              `json:"type"` // "script_test"
    Script      string              `json:"script"`
    ScriptPath  string              `json:"script_path"`
    Summary     *ScriptTestSummary  `json:"summary"`
    Environment *Environment        `json:"environment"`
    Results     []*ScriptTestResult `json:"results"`
    Metadata    *ReportMetadata     `json:"metadata"`
}

type ScriptTestSummary struct {
    Total      int   `json:"total"`
    Passed     int   `json:"passed"`
    Failed     int   `json:"failed"`
    Skipped    int   `json:"skipped"`
    DurationMs int64 `json:"duration_ms"`
}
```

### 9. Reporter

Generates reports in various formats. The format is determined by the `-o` file extension:

| Extension | Format   | Description                |
| --------- | -------- | -------------------------- |
| `.jsonl`  | JSONL    | Streaming, line-by-line    |
| `.json`   | JSON     | Full structured report     |
| `.md`     | Markdown | Human-readable with tables |
| `.html`   | HTML     | Interactive web report     |

#### Custom Reporter Agent (`-r` flag)

When `-r <agent-id>` is specified, the framework calls the specified agent to generate the report:

1. Test execution completes, `TestReport` is generated
2. Framework calls the reporter agent with input:
   ```json
   {
     "report": {
       /* TestReport object */
     },
     "format": "html",
     "options": { "verbose": true }
   }
   ```
3. Agent processes the report and returns formatted content
4. Framework writes the returned content to the output file

Example usage:

```bash
# Use custom reporter agent to generate a beautiful HTML report
yao agent test -i tests/inputs.jsonl -r report.beautiful -o report.html

# Use custom reporter agent to generate Slack-formatted summary
yao agent test -i tests/inputs.jsonl -r report.slack -o summary.txt
```

This allows for fully customizable report generation using AI agents

## Configuration

### Test Options

```go
type Options struct {
    // Input/Output
    Input       string        // Input source: file path, message, or scripts.xxx
    InputMode   InputMode     // Auto-detected: file, message, or script
    OutputFile  string        // Path to output report

    // Agent Selection
    AgentID     string        // Explicit agent ID (optional)
    Connector   string        // Override connector (optional)

    // Test Environment
    UserID      string        // Test user ID (-u flag)
    TeamID      string        // Test team ID (-t flag)
    Locale      string        // Locale (default: "en-us")

    // Execution
    Timeout     time.Duration // Default timeout per test
    Parallel    int           // Number of parallel tests (default: 1)
    Runs        int           // Number of runs per test case (default: 1)

    // Reporting
    ReporterID  string        // Reporter agent ID for custom report

    // Behavior
    Verbose     bool          // Verbose output
    FailFast    bool          // Stop on first failure
}

// InputMode represents the input mode for test cases
type InputMode string

const (
    InputModeFile    InputMode = "file"    // JSONL file input
    InputModeMessage InputMode = "message" // Direct message input
    InputModeScript  InputMode = "script"  // Script test mode (NEW)
)
```

### Input Mode Detection

The input mode is automatically detected based on the input value:

| Input Pattern     | Mode      | Description                |
| ----------------- | --------- | -------------------------- |
| `scripts.xxx.yyy` | `script`  | Script test mode           |
| `*.jsonl`         | `file`    | JSONL file mode            |
| `path/to/file`    | `file`    | File path (if file exists) |
| `"any text"`      | `message` | Direct message mode        |

```go
func DetectInputMode(input string) InputMode {
    // Check for script test prefix
    if strings.HasPrefix(input, "scripts.") {
        return InputModeScript
    }

    // Check if it's a file path
    if strings.HasSuffix(input, ".jsonl") || fileExists(input) {
        return InputModeFile
    }

    // Default to message mode
    return InputModeMessage
}
```

## Script Testing Implementation

### Script Resolution

```go
// ResolveScript resolves the script path from scripts.xxx.yyy format
func ResolveScript(input string) (*ScriptInfo, error) {
    // Remove "scripts." prefix
    path := strings.TrimPrefix(input, "scripts.")

    // Split into parts: "expense.setup" -> ["expense", "setup"]
    parts := strings.Split(path, ".")
    if len(parts) < 2 {
        return nil, fmt.Errorf("invalid script path: %s", input)
    }

    // Build paths
    // assistantDir: expense
    // moduleName: setup
    // scriptPath: expense/src/setup.ts
    // testPath: expense/src/setup_test.ts
    assistantDir := parts[0]
    moduleName := parts[1]

    return &ScriptInfo{
        ID:         input,
        Assistant:  assistantDir,
        Module:     moduleName,
        ScriptPath: filepath.Join(assistantDir, "src", moduleName+".ts"),
        TestPath:   filepath.Join(assistantDir, "src", moduleName+"_test.ts"),
    }, nil
}
```

### Test Function Discovery

Test functions are discovered by scanning the script for functions starting with `Test`:

```go
// DiscoverTests finds all Test* functions in the script
func DiscoverTests(scriptPath string) ([]*ScriptTestCase, error) {
    // Use the JavaScript runtime to list exported functions
    // Filter for functions starting with "Test"
    // Return list of test cases
}
```

### Testing Object Binding

The `testing.T` object is provided to test functions via JavaScript runtime binding:

```go
// TestingT represents the testing object passed to test functions
type TestingT struct {
    name    string
    failed  bool
    skipped bool
    logs    []string
    assert  *AssertObject
}

// AssertObject provides assertion methods
type AssertObject struct {
    t *TestingT
}

func (a *AssertObject) True(value bool, message ...string) {
    if !value {
        a.t.fail(formatMessage("expected true, got false", message))
    }
}

func (a *AssertObject) Equal(actual, expected interface{}, message ...string) {
    if !reflect.DeepEqual(actual, expected) {
        a.t.fail(formatMessage(
            fmt.Sprintf("expected %v, got %v", expected, actual),
            message,
        ))
    }
}

// ... other assertion methods
```

### Script Execution Flow

```
1. Parse input: "scripts.expense.setup"
2. Resolve script info:
   - TestPath: expense/src/setup_test.ts
   - ScriptPath: expense/src/setup.ts
3. Discover test functions: [TestSystemReady, TestSetupWithMockData, ...]
4. For each test function:
   a. Create testing.T object
   b. Create agent.Context with environment
   c. Execute: TestFunction(t, ctx)
   d. Collect result (passed/failed/skipped)
5. Generate report
```

### Integration with Existing Runner

```go
func (r *Executor) Run() (*Report, error) {
    switch r.opts.InputMode {
    case InputModeScript:
        return r.RunScriptTests()
    case InputModeMessage:
        return r.RunDirect()
    default:
        return r.RunTests()
    }
}

func (r *Executor) RunScriptTests() (*Report, error) {
    // 1. Resolve script
    scriptInfo, err := ResolveScript(r.opts.Input)
    if err != nil {
        return nil, err
    }

    // 2. Discover tests
    tests, err := DiscoverTests(scriptInfo.TestPath)
    if err != nil {
        return nil, err
    }

    // 3. Run each test
    results := make([]*ScriptTestResult, 0, len(tests))
    for _, tc := range tests {
        result := r.runScriptTest(tc, scriptInfo)
        results = append(results, result)

        if r.opts.FailFast && result.Status == StatusFailed {
            break
        }
    }

    // 4. Generate report
    return r.buildScriptReport(scriptInfo, results), nil
}
```

## Exit Codes

| Code | Description         |
| ---- | ------------------- |
| 0    | All tests passed    |
| 1    | Some tests failed   |
| 2    | Configuration error |
| 3    | Runtime error       |

## CI Integration

### GitHub Actions Example

```yaml
- name: Run Agent Tests
  run: |
    yao agent test -i assistants/keyword/tests/inputs.jsonl \
      -u ci-user -t ci-team \
      --runs 3 \
      -o report.json

- name: Check Stability
  run: |
    # Fail if any test has pass rate below 80%
    jq -e '.results | all(.pass_rate >= 80)' report.json

- name: Upload Test Report
  uses: actions/upload-artifact@v3
  with:
    name: agent-test-report
    path: report.json
```

### Exit Code Handling

The command exits with code 1 if any tests fail, making it easy to integrate with CI pipelines.

## Future Enhancements

1. **Snapshot Testing**: Compare outputs against saved snapshots
2. **Fuzzing**: Generate random inputs for robustness testing
3. **Coverage**: Track which agent code paths are exercised
4. **Benchmarking**: Performance metrics and regression detection
5. **Diff Reports**: Compare results between runs
6. **Flaky Test Detection**: Automatic identification of unstable tests
7. **Test Prioritization**: Run most important/failing tests first
8. **Script Test Enhancements**:
   - Parallel script test execution
   - Setup/Teardown hooks (`TestMain`, `BeforeEach`, `AfterEach`)
   - Mocking utilities for external dependencies
   - Code coverage for TypeScript/JavaScript scripts
