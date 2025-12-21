# Yao CLI Commands

The Yao CLI provides a set of commands for managing, running, and testing Yao applications.

## Installation

```bash
# Build from source
go build -o yao .

# Or install via go install
go install github.com/yaoapp/yao@latest
```

## Global Flags

| Flag     | Short | Description                     |
| -------- | ----- | ------------------------------- |
| `--app`  | `-a`  | Application directory path      |
| `--file` | `-f`  | Application package file (.yaz) |
| `--key`  | `-k`  | Application license key         |

## Environment Variables

| Variable   | Description                                  |
| ---------- | -------------------------------------------- |
| `YAO_ROOT` | Application root directory                   |
| `YAO_LANG` | Language setting (e.g., `zh-CN` for Chinese) |

## Commands

### `yao start`

Start the Yao application engine.

```bash
# Start in current directory
yao start

# Start with specific app directory
yao start -a /path/to/app

# Start in debug mode
yao start --debug
```

**Flags:**

| Flag                 | Description                   |
| -------------------- | ----------------------------- |
| `--debug`            | Enable development/debug mode |
| `--disable-watching` | Disable file watching         |

---

### `yao run`

Execute a Yao process.

```bash
# Run a process
yao run models.user.Find 1

# Run with JSON arguments
yao run models.user.Create '::[{"name":"John","age":30}]'

# Run in silent mode (JSON output only)
yao run -s models.user.Find 1
```

**Flags:**

| Flag       | Short | Description                              |
| ---------- | ----- | ---------------------------------------- |
| `--silent` | `-s`  | Silent mode - output result as JSON only |

**Argument Syntax:**

- Regular arguments: `arg1 arg2`
- JSON arguments: `'::[{"key":"value"}]'` (prefix with `::`)
- Escaped `::`: `'\::literal'`

---

### `yao migrate`

Update database schema based on model definitions.

```bash
# Migrate all models
yao migrate

# Migrate specific model
yao migrate -n user

# Force migrate in production mode
yao migrate --force

# Reset (drop and recreate) tables
yao migrate --reset
```

**Flags:**

| Flag      | Short | Description                      |
| --------- | ----- | -------------------------------- |
| `--name`  | `-n`  | Specific model name to migrate   |
| `--force` |       | Force migrate in production mode |
| `--reset` |       | Drop tables before migration     |

---

### `yao inspect`

Display application configuration.

```bash
yao inspect
```

---

### `yao version`

Show Yao version information.

```bash
# Show version
yao version

# Show all version details
yao version --all
```

**Flags:**

| Flag    | Description                                                          |
| ------- | -------------------------------------------------------------------- |
| `--all` | Print all version information (Go version, commit, build time, etc.) |

---

## Agent Commands

Commands for testing and managing AI agents.

### `yao agent test`

Test an agent with input cases from a JSONL file, direct message, or script tests.

```bash
# Test with direct message (development mode)
yao agent test -i "Extract keywords from: AI and machine learning" -n workers.system.keyword

# Test with JSONL file
yao agent test -i tests/inputs.jsonl

# Test with custom output file
yao agent test -i tests/inputs.jsonl -o report.html

# Test with specific connector
yao agent test -i tests/inputs.jsonl -c openai.gpt4

# Stability testing (multiple runs)
yao agent test -i tests/inputs.jsonl --runs 5

# Parallel execution
yao agent test -i tests/inputs.jsonl --parallel 4

# Verbose output
yao agent test -i tests/inputs.jsonl -v

# Script tests (test agent handler scripts)
yao agent test -i scripts.expense.setup -v

# Script tests with test filtering
yao agent test -i scripts.expense.setup --run "TestSystemReady"

# Script tests with custom context
yao agent test -i scripts.expense.setup --ctx tests/context.json -v
```

**Flags:**

| Flag          | Short | Description                                                      |
| ------------- | ----- | ---------------------------------------------------------------- |
| `--input`     | `-i`  | Input: JSONL file path, message, or script ID (required)         |
| `--output`    | `-o`  | Output file path (default: `output-{timestamp}.jsonl`)           |
| `--name`      | `-n`  | Agent ID (default: auto-detect from path)                        |
| `--connector` | `-c`  | Override default connector                                       |
| `--user`      | `-u`  | Test user ID (default: `test-user`)                              |
| `--team`      | `-t`  | Test team ID (default: `test-team`)                              |
| `--ctx`       |       | Path to context JSON file for custom authorization               |
| `--reporter`  | `-r`  | Reporter agent ID for custom report generation                   |
| `--runs`      |       | Number of runs per test case for stability analysis (default: 1) |
| `--run`       |       | Regex pattern to filter which tests to run                       |
| `--timeout`   |       | Timeout per test case (default: `5m`)                            |
| `--parallel`  |       | Number of parallel test cases (default: 1)                       |
| `--verbose`   | `-v`  | Enable verbose output                                            |
| `--fail-fast` |       | Stop on first failure                                            |
| `--app`       | `-a`  | Application directory                                            |
| `--env`       | `-e`  | Environment file                                                 |

**Input Modes:**

1. **Direct Message Mode**: For quick development/debugging

   ```bash
   yao agent test -i "Hello world" -n my.agent
   ```

   - Outputs result directly to stdout
   - No report file generated
   - Ideal for iterative development

2. **File Mode**: For comprehensive testing

   ```bash
   yao agent test -i tests/inputs.jsonl
   ```

   - Reads test cases from JSONL file
   - Generates detailed report
   - Supports stability analysis

3. **Script Test Mode**: For testing agent handler scripts
   ```bash
   yao agent test -i scripts.expense.setup -v
   ```
   - Tests TypeScript/JavaScript handler scripts (hooks, tools, setup functions)
   - Input format: `scripts.<assistant>.<module>` (e.g., `scripts.expense.setup`)
   - Automatically discovers and runs all `Test*` functions
   - Uses Go-like testing interface with assertions

**Script Test Function Signature:**

```typescript
// assistants/expense/src/setup_test.ts
import { SystemReady } from "./setup";

export function TestSystemReady(t: testing.T, ctx: agent.Context) {
  const result = SystemReady(ctx);
  t.assert.True(result.success, "SystemReady should succeed");
  t.assert.Equal(result.status, "ready", "Status should be ready");
}
```

**Context JSON Format (for `--ctx` flag):**

```json
{
  "authorized": {
    "sub": "user-12345",
    "client_id": "my-app",
    "user_id": "admin",
    "team_id": "team-001",
    "tenant_id": "acme-corp",
    "constraints": {
      "owner_only": true,
      "team_only": false,
      "extra": { "department": "engineering" }
    }
  },
  "metadata": { "request_id": "req-123" },
  "client": { "type": "web", "ip": "192.168.1.100" },
  "locale": "zh-cn"
}
```

**JSONL Input Format:**

```jsonl
{"id": "T001", "input": "Simple text input"}
{"id": "T002", "input": {"role": "user", "content": "Message with role"}}
{"id": "T003", "input": [{"role": "system", "content": "System prompt"}, {"role": "user", "content": "User message"}]}
{"id": "T004", "input": "Test with timeout", "timeout": "30s"}
{"id": "T005", "input": "Skip this test", "skip": true}
{"id": "T006", "input": "Test with specific user", "user": "alice", "team": "engineering"}
```

**Output Formats:**

| Extension | Format   | Description                |
| --------- | -------- | -------------------------- |
| `.jsonl`  | JSONL    | Streaming format (default) |
| `.json`   | JSON     | Complete structured report |
| `.md`     | Markdown | Human-readable with tables |
| `.html`   | HTML     | Interactive web report     |

**Agent Resolution:**

The agent is resolved in the following priority order:

1. Explicit `-n` flag: `yao agent test -i msg -n my.agent`
2. `YAO_ROOT` environment variable
3. Auto-detect from input file path (traverses up to find `package.yao`)
4. Auto-detect from current working directory

---

## SUI Commands

SUI (Serverless UI) template engine commands.

### `yao sui watch`

Auto-build templates when files change.

```bash
yao sui watch <sui-id> <template-name> [data]

# Example
yao sui watch default index '::{}'
```

### `yao sui build`

Build a template.

```bash
yao sui build <sui-id> <template-name> [data]

# Example
yao sui build default index '::{}'

# Debug mode
yao sui build default index '::{}' --debug
```

### `yao sui trans`

Translate template content.

```bash
yao sui trans <sui-id> <template-name>

# With specific locales
yao sui trans default index -l "en-US,zh-CN,ja-JP"
```

**SUI Flags:**

| Flag        | Short | Description                               |
| ----------- | ----- | ----------------------------------------- |
| `--data`    | `-d`  | Session data as JSON (prefix with `::`)   |
| `--debug`   | `-D`  | Enable debug mode                         |
| `--locales` | `-l`  | Locales for translation (comma-separated) |

---

## Examples

### Development Workflow

```bash
# Start development server
yao start --debug

# Run a process
yao run scripts.test.Hello "World"

# Test an agent interactively
yao agent test -i "What is the weather today?" -n assistant.weather

# Watch and auto-build templates
yao sui watch default home
```

### Testing Workflow

```bash
# Run comprehensive agent tests
yao agent test -i tests/inputs.jsonl -o report.html -v

# Run script tests for agent handlers
yao agent test -i scripts.expense.setup -v

# Run specific script tests with filtering
yao agent test -i scripts.expense.setup --run "TestSystem.*" -v

# Run script tests with custom context
yao agent test -i scripts.expense.setup --ctx tests/context.json -v

# Stability analysis (run each test 10 times)
yao agent test -i tests/inputs.jsonl --runs 10 -o stability-report.json

# Parallel testing with timeout
yao agent test -i tests/inputs.jsonl --parallel 4 --timeout 2m

# CI/CD integration
yao agent test -i tests/inputs.jsonl -o results.jsonl && echo "Tests passed"
```

### Database Migration

```bash
# Migrate all models
yao migrate

# Migrate specific model with reset
yao migrate -n user --reset --force
```

---

## Exit Codes

| Code | Description           |
| ---- | --------------------- |
| 0    | Success               |
| 1    | Error or test failure |

---

## Directory Structure

```
myapp/
├── app.yao              # Application configuration
├── .env                 # Environment variables
├── models/              # Data models
├── apis/                # API definitions
├── flows/               # Business flows
├── scripts/             # JavaScript/TypeScript scripts
├── assistants/          # AI agents
│   └── my-agent/
│       ├── package.yao  # Agent configuration
│       ├── prompts.yml  # Agent prompts
│       └── tests/
│           └── inputs.jsonl  # Test cases
└── public/              # Static files
```

---

## See Also

- [Yao Documentation](https://yaoapps.com/docs)
- [Agent Test Design](../agent/test/DESIGN.md)
- [SUI Documentation](https://yaoapps.com/docs/sui)
