# Agent Testing Guide

## Test Tiers

| Tier | Build Tag | Dependencies | Timeout |
|------|-----------|-------------|---------|
| 0 | _(none)_ | Docker + Tai | 600s |
| 1 | `unit` | None | 120s |
| 2 | `integration` | App + DB + Mock LLM | 300s |
| 3 | `sandbox` | Docker + Tai + Mock LLM | 600s |
| 4 | `e2e` | Real LLM API | 900s |
| 5 | `stress` | App + DB + Mock LLM | 1800s |

## Running Tests Locally

### Prerequisites

1. Add mock LLM host entry:

```bash
echo "127.0.0.1 host.tai.internal" | sudo tee -a /etc/hosts
```

2. Build and start mock LLM:

```bash
cd unit-test/agent/mock-llm
go build -o /tmp/mock-llm .
/tmp/mock-llm -port 6920 -fixtures fixtures &
```

3. Copy env file (already gitignored, auto-loaded by `testprepare`):

```bash
cp unit-test/agent/env/agent-test.env.template unit-test/agent/env/agent-test.env
```

### Run Tests

```bash
# Tier 1: unit (no external deps)
go test -count=1 -tags unit ./agent/...

# Tier 2: integration (needs mock LLM)
go test -count=1 -tags integration ./agent/...

# Both tiers together
go test -count=1 -tags "unit,integration" ./agent/...

# PostgreSQL instead of SQLite
YAO_TEST_DB=postgres go test -count=1 -tags "unit,integration" ./agent/...

# Stress tests
go test -count=1 -timeout 1800s -tags stress ./agent/...
```

### Coverage

```bash
mkdir -p .build/coverage

PKGS=$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' -tags unit ./agent/... | tr '\n' ' ')
go test -count=1 -tags unit -coverprofile=.build/coverage/tier1.out -coverpkg=./agent/... $PKGS

PKGS=$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' -tags integration ./agent/... | tr '\n' ' ')
go test -count=1 -tags integration -coverprofile=.build/coverage/tier2.out -coverpkg=./agent/... $PKGS

head -1 .build/coverage/tier1.out > .build/coverage/all.out
tail -n +2 -q .build/coverage/tier*.out >> .build/coverage/all.out
go tool cover -func=.build/coverage/all.out | tail -1
```

> `go test -coverprofile` fails on packages without test files. Always filter with `go list` first.

## Writing Tests

### Rules

1. **Use `testprepare` only** -- never `testutils.PrepareAgent` or `test.Prepare`.
2. **Always set a build tag** -- `//go:build unit`, `integration`, `sandbox`, `e2e`, or `stress`.
3. **External test package** -- use `package foo_test`, not `package foo`.
4. **No `t.Skip`** -- tests must pass or `t.Fatal`. If a prerequisite is missing, fail explicitly.
5. **Test assistants** live in `unit-test/agent/app/assistants/tests/`. ID format: `tests.<name>`.
6. **Cleanup is mandatory** -- temporary resources (containers, files, DB rows) must be removed in `t.Cleanup`.
7. **Never modify app config to fix tests** -- fix the code, not `app.yao` or model definitions.

### Exposing Unexported Symbols

When an unexported function needs testing, create `export_test.go` in the implementation package:

```go
// agent/store/xun/export_test.go
package xun

var ExportBuildQuery = buildQuery
```

```go
// agent/store/xun/query_unit_test.go
package xun_test

func TestBuildQuery(t *testing.T) {
    q := xun.ExportBuildQuery("users", 10)
    // ...
}
```

### File Structure

Every package with tests needs a `main_test.go` (no build tag):

```go
package assistant_test

import (
    "os"
    "testing"
    "github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMain(m *testing.M) {
    testprepare.MustLoadEnv()
    os.Exit(m.Run())
}
```

Test files use the tier-appropriate prepare function:

```go
//go:build unit

package search_test

func TestCitationNext(t *testing.T) {
    // No prepare needed for pure unit tests
}
```

```go
//go:build integration

package assistant_test

func TestCreateHook_Echo(t *testing.T) {
    testprepare.PrepareSandbox(t)
    // ...
}
```

### Naming Convention

| Pattern | When |
|---------|------|
| `xxx_unit_test.go` | Tier 1 pure logic |
| `xxx_integration_test.go` | Tier 2 with app/db |
| `xxx_e2e_test.go` | Tier 4 real LLM |
| `stress_test.go` | Tier 5 perf/leak detection |
| `xxx_test.go` | Single-tier package (tag in header) |

### Prepare Functions

| Function | Tier | What it starts |
|----------|------|---------------|
| _(none)_ | 0 | sandbox/v2 has its own TestMain |
| `PrepareUnit(t)` | 1 | Loads env only |
| `PrepareSandbox(t)` | 2, 3 | App + DB + V8 + Mock LLM |
| `PrepareE2E(t)` | 4 | App + DB + V8 + Real LLM |
| `PrepareSandbox(t)` | 5 (stress) | Same as Tier 2, high-iteration perf/leak tests |

### Mock Infrastructure

| Component | How |
|-----------|-----|
| LLM | `openai.mock` connector -> `http://host.tai.internal:6920` |
| MCP | Built-in echo server (tools: `ping`, `echo`, `status`) |
| Search | Mock handler or `__yao.needsearch` via mock LLM |

## CI

Workflow: `.github/workflows/agent-unit-test.yml`

- **Platform**: Linux (ubuntu-latest). Windows is separate.
- **DB matrix**: SQLite3 + Postgres14.
- **Coverage**: Each tier generates a coverprofile, merged and uploaded to Codecov.
- **Mock LLM**: Built from `unit-test/agent/mock-llm/`, started before tests.
- **Env**: Generated from `agent-test.env.template`, secrets injected from GitHub repo secrets.
- **Host mapping**: `host.tai.internal` -> `127.0.0.1`.
