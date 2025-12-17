# Agent Test Framework

A testing framework for Yao AI agents with support for assertions, stability analysis, and CI integration.

## Quick Start

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

## Input Modes

The `-i` flag supports two input modes:

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

Output is printed to stdout (or saved to `-o` if specified).

## Command Line Options

| Flag          | Description                              | Default                    |
| ------------- | ---------------------------------------- | -------------------------- |
| `-i`          | Input: JSONL file path or direct message | (required)                 |
| `-o`          | Output file path                         | `output-{timestamp}.jsonl` |
| `-n`          | Agent ID (optional, auto-detected)       | auto-detect                |
| `-c`          | Override connector                       | agent default              |
| `-u`          | Test user ID                             | `test-user`                |
| `-t`          | Test team ID                             | `test-team`                |
| `-r`          | Reporter agent ID                        | built-in                   |
| `--runs`      | Runs per test (stability analysis)       | 1                          |
| `--timeout`   | Timeout per test                         | 5m                         |
| `--parallel`  | Parallel test cases                      | 1                          |
| `-v`          | Verbose output                           | false                      |
| `--fail-fast` | Stop on first failure                    | false                      |

## Agent Resolution

The agent is resolved in the following priority order:

1. **Explicit `-n` flag**: `yao agent test -i "msg" -n my.agent`
2. **Path-based detection**: Traverse up from input file to find `package.yao`
3. **Current directory**: For direct message mode, look for `package.yao` in cwd

Example directory structure:

```
assistants/workers/system/keyword/
├── package.yao          <- Agent definition (auto-detected)
├── prompts.yml
├── src/
│   └── index.ts
└── tests/
    └── inputs.jsonl     <- Input file
```

## Input Format (JSONL)

Each line is a JSON object:

```jsonl
{"id": "T001", "input": "Simple text"}
{"id": "T002", "input": {"role": "user", "content": "Message with role"}}
{"id": "T003", "input": [{"role": "user", "content": "Hi"}, {"role": "assistant", "content": "Hello"}, {"role": "user", "content": "Follow-up"}]}
{"id": "T004", "input": "Test", "assert": {"type": "json_path", "path": "field", "value": true}}
{"id": "T005", "input": "Skip this", "skip": true}
```

### Fields

| Field      | Type                           | Required | Description                    |
| ---------- | ------------------------------ | -------- | ------------------------------ |
| `id`       | string                         | Yes      | Test case ID                   |
| `input`    | string \| Message \| []Message | Yes      | Test input                     |
| `assert`   | Assertion \| []Assertion       | No       | Assertion rules                |
| `expected` | any                            | No       | Expected output (exact match)  |
| `user`     | string                         | No       | Override user ID               |
| `team`     | string                         | No       | Override team ID               |
| `timeout`  | string                         | No       | Override timeout (e.g., "30s") |
| `skip`     | bool                           | No       | Skip this test                 |
| `metadata` | map                            | No       | Additional metadata            |

### Input Types

| Type        | Description          | Example                                               |
| ----------- | -------------------- | ----------------------------------------------------- |
| `string`    | Simple text          | `"Hello world"`                                       |
| `Message`   | Single message       | `{"role": "user", "content": "..."}`                  |
| `[]Message` | Conversation history | `[{"role": "user", ...}, {"role": "assistant", ...}]` |

## Assertions

Use `assert` for flexible validation. If `assert` is defined, it takes precedence over `expected`.

### Assertion Types

| Type           | Description                   | Example                                                   |
| -------------- | ----------------------------- | --------------------------------------------------------- |
| `equals`       | Exact match                   | `{"type": "equals", "value": {"key": "val"}}`             |
| `contains`     | Output contains value         | `{"type": "contains", "value": "keyword"}`                |
| `not_contains` | Output does not contain value | `{"type": "not_contains", "value": "error"}`              |
| `json_path`    | Extract JSON path and compare | `{"type": "json_path", "path": "$.field", "value": true}` |
| `regex`        | Match regex pattern           | `{"type": "regex", "value": "\\d+"}`                      |
| `type`         | Check output type             | `{"type": "type", "value": "object"}`                     |
| `script`       | Run custom assertion script   | `{"type": "script", "script": "scripts.test.Check"}`      |

### Assertion Options

| Field     | Type   | Description                 |
| --------- | ------ | --------------------------- |
| `type`    | string | Assertion type (required)   |
| `value`   | any    | Expected value or pattern   |
| `path`    | string | JSON path (for `json_path`) |
| `script`  | string | Script name (for `script`)  |
| `message` | string | Custom failure message      |
| `negate`  | bool   | Invert the result           |

### Examples

**JSON path validation:**

```jsonl
{
  "id": "T001",
  "input": "What's the weather?",
  "assert": {
    "type": "json_path",
    "path": "need_search",
    "value": true
  }
}
```

**Multiple assertions (all must pass):**

```jsonl
{
  "id": "T002",
  "input": "Hello",
  "assert": [
    {
      "type": "json_path",
      "path": "need_search",
      "value": false
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
  "id": "T003",
  "input": "Test",
  "assert": {
    "type": "script",
    "script": "scripts.test.Validate"
  }
}
```

Script receives `(output, input, expected)` and returns:

```javascript
// Boolean
return true;

// Or detailed result
return { pass: true, message: "Validation passed" };
```

**Negated assertion:**

```jsonl
{
  "id": "T004",
  "input": "Hello",
  "assert": {
    "type": "contains",
    "value": "error",
    "negate": true
  }
}
```

### JSON Path Notes

- Supports dot-notation: `$.field.subfield` or `field.subfield`
- Auto-extracts JSON from markdown code blocks (` ```json ... ``` `)
- Works with both string output and structured objects

## Output Formats

Determined by `-o` file extension:

| Extension | Format   | Description            |
| --------- | -------- | ---------------------- |
| `.jsonl`  | JSONL    | Streaming (default)    |
| `.json`   | JSON     | Complete structured    |
| `.md`     | Markdown | Human-readable         |
| `.html`   | HTML     | Interactive web report |

### Default Output Path

When `-o` is not specified in file mode:

```
{input_directory}/output-{timestamp}.jsonl
```

Example: `tests/output-20241217100000.jsonl`

In direct message mode without `-o`, output is printed to stdout.

## Stability Analysis

Run each test multiple times to measure consistency:

```bash
yao agent test -i tests/inputs.jsonl --runs 5 -o stability.json
```

Output includes:

- Pass rate per test
- Stability classification (stable, mostly_stable, unstable, highly_unstable)
- Average/min/max duration
- Standard deviation

### Stability Classification

| Pass Rate | Classification  |
| --------- | --------------- |
| 100%      | Stable          |
| 80-99%    | Mostly Stable   |
| 50-79%    | Unstable        |
| < 50%     | Highly Unstable |

## Test Environment

The test framework creates a context with configurable environment:

| Setting    | Flag | Default     |
| ---------- | ---- | ----------- |
| User ID    | `-u` | `test-user` |
| Team ID    | `-t` | `test-team` |
| Locale     | -    | `en-us`     |
| ClientType | -    | `test`      |
| ClientIP   | -    | `127.0.0.1` |

Priority: Command line flags > Test case fields > Defaults

## Custom Reporter Agent

Use `-r` to specify a custom agent for report generation:

```bash
yao agent test -i tests/inputs.jsonl -r report.beautiful -o report.html
```

The reporter agent receives:

```json
{
  "report": { "summary": {...}, "results": [...] },
  "format": "html",
  "options": { "verbose": true }
}
```

## CI Integration

```bash
# Exit code: 0 = all passed, 1 = failures
yao agent test -i tests/inputs.jsonl -o results.jsonl --fail-fast

# Parse JSONL results
cat results.jsonl | jq 'select(.type == "summary")'
```

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
    jq -e '.results | all(.pass_rate >= 80)' report.json
```

## Examples

```bash
# Quick development test (auto-detect agent)
cd assistants/keyword
yao agent test -i "Extract keywords: AI and ML"

# Quick development test (specify agent)
yao agent test -i "Hello" -n workers.system.keyword

# Full test suite with HTML report
yao agent test -i tests/inputs.jsonl -o report.html -v

# Override connector
yao agent test -i tests/inputs.jsonl -c openai.gpt4

# Stability analysis
yao agent test -i tests/inputs.jsonl --runs 10 -o stability.json

# Parallel execution with timeout
yao agent test -i tests/inputs.jsonl --parallel 4 --timeout 2m

# Custom test environment
yao agent test -i tests/inputs.jsonl -u admin -t prod-team

# Custom reporter agent
yao agent test -i tests/inputs.jsonl -r report.beautiful -o custom-report.md

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

## Exit Codes

| Code | Description                                         |
| ---- | --------------------------------------------------- |
| 0    | All tests passed                                    |
| 1    | Tests failed, configuration error, or runtime error |
