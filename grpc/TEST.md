# Yao gRPC Server — Test Specification

Design: [DESIGN.md](./DESIGN.md) | Implementation: [IMPL.md](./IMPL.md)

## Principles

- **Black-box testing**: all `*_test.go` files use `package xxx_test` — tests only access exported API via gRPC client
- **Tests follow implementation**: `*_test.go` lives next to the code it tests (`grpc/auth/guard_test.go` beside `grpc/auth/guard.go`)
- **Real server**: every test starts a real gRPC server on a random TCP port, exercises the full interceptor → handler chain
- **Coverage > 80%**: per sub-package and overall

## Prerequisites

```bash
source $YAO_SOURCE_ROOT/env.local.sh
```

Required environment variables (same as existing Yao tests):

| Variable | Purpose |
|----------|---------|
| `YAO_TEST_APPLICATION` | Path to `yao-dev-app` |
| `YAO_DB_DRIVER` / `YAO_DB_PRIMARY` | Database connection |
| `YAO_JWT_SECRET` / `YAO_DB_AESKEY` | Crypto keys |
| `OPENAI_TEST_KEY` | LLM streaming tests |
| `ANTHROPIC_API_KEY` | LLM streaming tests (Anthropic) |

## Directory Structure

```
grpc/
├── grpc.go
├── tests/
│   └── testutils/
│       └── testutils.go        # shared test utilities
├── auth/
│   ├── guard.go
│   ├── guard_test.go           # package auth_test
│   ├── endpoint.go
│   ├── endpoint_test.go        # package auth_test
│   └── scope.go
├── run/
│   ├── run.go
│   └── run_test.go             # package run_test
├── shell/
│   ├── shell.go
│   └── shell_test.go           # package shell_test
├── api/
│   ├── api.go
│   └── api_test.go             # package api_test
├── mcp/
│   ├── mcp.go
│   └── mcp_test.go             # package mcp_test
├── llm/
│   ├── llm.go
│   └── llm_test.go             # package llm_test
├── agent/
│   ├── agent.go
│   └── agent_test.go           # package agent_test
└── health/
    ├── health.go
    └── health_test.go          # package health_test
```

Tests live beside the code they verify. `grpc/tests/testutils/` is shared infrastructure only.

## testutils API

`grpc/tests/testutils/testutils.go` provides the test harness used by all sub-packages.

```go
package testutils

// Prepare initializes the full Yao runtime (DB, V8, models, scripts, etc.)
// then starts a real gRPC server on :0 (random port).
// Returns a connected grpc.ClientConn ready to create service clients.
//
// Internally calls:
//   test.Prepare(t, config.Conf)   — Yao runtime
//   grpc.StartServer(cfg{Port:0})  — gRPC server
//   grpc.Dial("127.0.0.1:port")   — client connection
func Prepare(t *testing.T) *grpc.ClientConn

// Clean gracefully stops the gRPC server and tears down the Yao runtime.
// Always use with defer:
//   conn := testutils.Prepare(t)
//   defer testutils.Clean()
func Clean()

// Addr returns the gRPC server address "127.0.0.1:xxxxx".
func Addr() string

// ObtainAccessToken mints a token with the given scopes.
// Calls oauth.MakeAccessToken directly — no HTTP round-trip.
func ObtainAccessToken(t *testing.T, scopes ...string) string

// ObtainAccessTokenForUser mints a token for a specific user ID.
func ObtainAccessTokenForUser(t *testing.T, userID string, scopes ...string) string

// WithToken returns ctx with Bearer token in gRPC metadata.
func WithToken(ctx context.Context, token string) context.Context

// WithRefreshToken returns ctx with both Bearer and x-refresh-token metadata.
func WithRefreshToken(ctx context.Context, token, refreshToken string) context.Context

// WithSandboxMetadata returns ctx with x-sandbox-id and x-grpc-upstream metadata.
func WithSandboxMetadata(ctx context.Context, sandboxID, upstream string) context.Context

// NewClient creates a pb.YaoServiceClient from a connection.
func NewClient(conn *grpc.ClientConn) pb.YaoServiceClient
```

## How to Write a Test

### Standard pattern

Every test file follows this structure:

```go
// grpc/run/run_test.go
package run_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "github.com/yaoapp/yao/grpc/pb"
    "github.com/yaoapp/yao/grpc/tests/testutils"
)

func TestRun_ProcessExec(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()

    client := testutils.NewClient(conn)
    token := testutils.ObtainAccessToken(t, "grpc:run")
    ctx := testutils.WithToken(context.Background(), token)

    resp, err := client.Run(ctx, &pb.RunRequest{
        Process: "utils.app.Ping",
    })
    assert.NoError(t, err)
    assert.NotNil(t, resp.Data)
}

func TestRun_InvalidProcess(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()

    client := testutils.NewClient(conn)
    token := testutils.ObtainAccessToken(t, "grpc:run")
    ctx := testutils.WithToken(context.Background(), token)

    _, err := client.Run(ctx, &pb.RunRequest{Process: "nonexistent.process"})
    assert.Error(t, err)
}
```

### Auth tests

Auth tests verify the interceptor chain through the gRPC client:

```go
// grpc/auth/guard_test.go
package auth_test

func TestAuth_NoToken_Rejected(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()
    client := testutils.NewClient(conn)

    _, err := client.Run(context.Background(), &pb.RunRequest{Process: "utils.app.Ping"})
    st, _ := status.FromError(err)
    assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_WrongScope_Denied(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()
    client := testutils.NewClient(conn)

    token := testutils.ObtainAccessToken(t, "grpc:mcp")
    ctx := testutils.WithToken(context.Background(), token)

    _, err := client.Run(ctx, &pb.RunRequest{Process: "utils.app.Ping"})
    st, _ := status.FromError(err)
    assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestAuth_TokenRefresh(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()
    client := testutils.NewClient(conn)

    // Mint an expired token + valid refresh token,
    // send request with x-refresh-token metadata,
    // verify response header contains x-new-access-token.
}

func TestHealthz_Public(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()
    client := testutils.NewClient(conn)

    resp, err := client.Healthz(context.Background(), &pb.Empty{})
    assert.NoError(t, err)
    assert.Equal(t, "ok", resp.Status)
}
```

### Streaming tests

```go
// grpc/llm/llm_test.go
package llm_test

func TestChatCompletionsStream(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()
    client := testutils.NewClient(conn)

    token := testutils.ObtainAccessToken(t, "grpc:llm")
    ctx := testutils.WithToken(context.Background(), token)

    stream, err := client.ChatCompletionsStream(ctx, &pb.ChatRequest{
        // ... model, messages, etc.
    })
    assert.NoError(t, err)

    var chunks int
    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            break
        }
        assert.NoError(t, err)
        chunks++
        assert.NotEmpty(t, chunk.Data)
    }
    assert.Greater(t, chunks, 0)
}
```

```go
// grpc/agent/agent_test.go
package agent_test

func TestAgentStream(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()
    client := testutils.NewClient(conn)

    token := testutils.ObtainAccessToken(t, "grpc:agent")
    ctx := testutils.WithToken(context.Background(), token)

    stream, err := client.AgentStream(ctx, &pb.AgentRequest{
        RobotID: "test-robot",
        // ...
    })
    assert.NoError(t, err)

    var chunks int
    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            break
        }
        assert.NoError(t, err)
        chunks++
        // Each chunk carries JSON-serialized agent/output/message.Message
    }
    assert.Greater(t, chunks, 0)
}

func TestAgentStream_InvalidRobot(t *testing.T) {
    conn := testutils.Prepare(t)
    defer testutils.Clean()
    client := testutils.NewClient(conn)

    token := testutils.ObtainAccessToken(t, "grpc:agent")
    ctx := testutils.WithToken(context.Background(), token)

    stream, err := client.AgentStream(ctx, &pb.AgentRequest{
        RobotID: "nonexistent-robot",
    })
    // Either err on open or first Recv returns error
    if err == nil {
        _, err = stream.Recv()
    }
    assert.Error(t, err)
}
```

## Required Test Cases

Each sub-package must cover at minimum:

| Sub-package | Required cases |
|-------------|----------------|
| `auth` | valid token / no token (Unauthenticated) / expired token + refresh / wrong scope (PermissionDenied) / Healthz skips auth |
| `health` | Healthz returns ok without token |
| `run` | valid process / nonexistent process / bad arguments |
| `shell` | valid command / command not found / timeout |
| `api` | valid proxy / 404 endpoint |
| `mcp` | MCPListTools / MCPCallTool / MCPListResources / MCPReadResource |
| `llm` | ChatCompletions (unary) / ChatCompletionsStream (multiple chunks) / invalid model |
| `agent` | AgentStream (receives message chunks) / nonexistent robot ID |

## Makefile

Add to [Makefile](../Makefile):

```makefile
TESTFOLDER_GRPC := $(shell $(GO) list ./grpc/...)

.PHONY: unit-test-grpc
unit-test-grpc:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_GRPC); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=10m \
			-covermode=count -coverprofile=profile.out \
			-coverpkg=$$(echo $$d | sed "s/\/test$$//g") \
			-skip='TestMemoryLeak|TestIsolateDisposal|TestLeak_|TestScenario_' \
			$$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done
```

Also add `|grpc` to the `TESTFOLDER_CORE` exclude pattern so core-test does not duplicate gRPC tests.

## CI Integration

Add `grpc-test` job to `unit-test.yml` and `pr-test.yml`:

```yaml
grpc-test:
  runs-on: ubuntu-latest
  services:
    mongodb:
      image: mongo:6.0
      ports:
        - 27017:27017
      env:
        MONGO_INITDB_ROOT_USERNAME: root
        MONGO_INITDB_ROOT_PASSWORD: "123456"
        MONGO_INITDB_DATABASE: test
  strategy:
    matrix:
      go: ["1.25"]
  steps:
    # ... standard checkout + setup (same as core-test) ...

    - name: Setup ENV (SQLite)
      run: |
        mkdir -p ${{ github.WORKSPACE }}/../app/db
        echo "YAO_DB_DRIVER=sqlite3" >> $GITHUB_ENV
        echo "YAO_DB_PRIMARY=${{ github.WORKSPACE }}/../app/db/yao.db" >> $GITHUB_ENV

    - name: Run gRPC Tests
      run: make unit-test-grpc

    - name: Codecov Report
      uses: codecov/codecov-action@v4
      with:
        token: ${{ secrets.CODECOV_TOKEN }}
```

Key decisions:
- SQLite only — gRPC is a transport layer, no need for MySQL matrix
- No Qdrant/Neo4j/MCP-everything services needed
- LLM/Agent streaming uses real `OPENAI_TEST_KEY` + `ANTHROPIC_API_KEY` (same secrets as agent-test job)

## Coverage

- Target: >80% per sub-package, >80% overall
- `grpc.go` (server lifecycle) covered indirectly via testutils.Prepare/Clean
- Coverage collected via `-coverprofile`, reported to Codecov

## Phase Test Schedule

Tests are written alongside implementation, not after:

| Phase | Test files | Repo |
|-------|------------|------|
| Phase 1 (auth + server) | `auth/guard_test.go`, `health/health_test.go` | yao |
| Phase 2 (handlers) | `run/run_test.go`, `shell/shell_test.go`, `api/api_test.go`, `mcp/mcp_test.go` | yao |
| Phase 3 (LLM + Agent) | `llm/llm_test.go`, `agent/agent_test.go` | yao |
| Phase 4 (Tai gateway) | Tai repo tests — gateway forwards `x-grpc-upstream`, conn cache reuse, missing metadata rejected | tai |
| Phase 5 (yao-grpc client) | `tai/grpc/grpc_test.go` — dial, method wrappers, token refresh via response metadata, `x-grpc-upstream` attachment | yao |
| Phase 6 (Device Flow) | `openapi/oauth/*_test.go` — DeviceAuthorization, device_code grant, poll pending/approved/expired | yao |

Each Phase PR must include tests for all new code. Coverage must meet threshold before merge.

## Running Tests

```bash
# All gRPC tests
make unit-test-grpc

# Single sub-package
go test -v ./grpc/auth/

# Single test
go test -v -run TestAuth_NoToken_Rejected ./grpc/auth/
```
