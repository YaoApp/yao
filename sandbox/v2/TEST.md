# Sandbox V2 — Test Specification

Design: [DESIGN.md](./DESIGN.md) | Implementation: [IMPL.md](./IMPL.md)

## Principles

- **Black-box testing**: all `*_test.go` files use `package sandbox_test` — tests only access exported API
- **Real containers**: tests create real Docker containers via tai SDK, no mocking
- **Skip when unavailable**: `skipIfNoDocker(t)` / `skipIfNoTai(t)` — CI has Docker and Tai; local dev may not
- **Tests follow implementation**: `*_test.go` lives next to the code it tests
- **Coverage > 80%**: per file and overall

## Prerequisites

```bash
source $YAO_SOURCE_ROOT/env.local.sh
```

### Docker (required for all container tests)

Docker daemon must be running. Tests connect via default socket.

### Tai (required for remote-mode tests only)

```bash
docker run -d --name tai \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -p 2375:2375 -p 9100:9100 -p 8080:8080 -p 6080:6080 \
  yaoapp/tai:latest
```

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `YAO_TEST_APPLICATION` | Path to `yao-dev-app` | — (required) |
| `YAO_DB_DRIVER` / `YAO_DB_PRIMARY` | Database connection | — (required) |
| `YAO_JWT_SECRET` / `YAO_DB_AESKEY` | Crypto keys (for OAuth token creation) | — (required) |
| `SANDBOX_TEST_IMAGE` | Container image for tests | `yaoapp/sandbox-v2-test:latest` |
| `SANDBOX_TEST_REMOTE_ADDR` | Tai remote address, e.g. `tai://127.0.0.1` | — (skip remote tests if empty) |
| `TAI_TEST_HOST` | Tai HTTP proxy host | `127.0.0.1` |

## Directory Structure

```
sandbox/v2/
├── sandbox.go
├── sandbox_test.go             # Init/M singleton tests
├── manager.go
├── manager_test.go             # Create/Get/GetOrCreate/List/Remove, pool management
├── manager_lifecycle_test.go   # Start recovery, Cleanup, idle tracking, heartbeat
├── box.go
├── box_test.go                 # Exec/Stream/Workspace/Proxy/VNC, lifecycle
├── box_attach_test.go          # Attach WS/SSE (needs service in container)
├── config.go
├── types.go
├── errors.go
├── grpc.go
├── grpc_test.go                # OAuth token creation/revocation, env var building
├── testutils_test.go           # shared test helpers (unexported, package sandbox_test)
└── DESIGN.md
```

## testutils (internal to sandbox_test)

Shared helpers in `testutils_test.go` — not a separate package, lives inside `package sandbox_test`.

```go
// testutils_test.go
package sandbox_test

// skipIfNoDocker skips the test if Docker is not available.
func skipIfNoDocker(t *testing.T)

// skipIfNoTai skips the test if SANDBOX_TEST_REMOTE_ADDR is empty.
func skipIfNoTai(t *testing.T)

// testImage returns SANDBOX_TEST_IMAGE or "yaoapp/sandbox-v2-test:latest".
func testImage() string

// setupManager initializes sandbox with a local pool, returns cleanup func.
// Calls sandbox.Init + sandbox.M().Start.
func setupManager(t *testing.T) func()

// setupManagerWithRemote initializes sandbox with local + remote pools.
func setupManagerWithRemote(t *testing.T) func()

// createTestBox creates a box with defaults and returns it. Registers t.Cleanup for removal.
func createTestBox(t *testing.T, opts ...sandbox.CreateOption) *sandbox.Box
```

## How to Write a Test

### Standard pattern

```go
// manager_test.go
package sandbox_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestCreate(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()

    box, err := sandbox.M().Create(context.Background(), sandbox.CreateOptions{
        Image: testImage(),
        Owner: "test-user",
    })
    require.NoError(t, err)
    defer box.Remove(context.Background())

    assert.NotEmpty(t, box.ID())
    assert.Equal(t, "test-user", box.Owner())
}

func TestCreate_NoImage(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()

    _, err := sandbox.M().Create(context.Background(), sandbox.CreateOptions{})
    assert.Error(t, err) // Image is required
}
```

### Container execution tests

```go
// box_test.go
package sandbox_test

func TestExec(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    result, err := box.Exec(context.Background(), []string{"echo", "hello"})
    require.NoError(t, err)
    assert.Equal(t, 0, result.ExitCode)
    assert.Equal(t, "hello\n", result.Stdout)
    assert.Empty(t, result.Stderr)
}

func TestExec_NonZeroExit(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    result, err := box.Exec(context.Background(), []string{"sh", "-c", "exit 42"})
    require.NoError(t, err)
    assert.Equal(t, 42, result.ExitCode)
}
```

### Streaming tests

```go
// box_test.go
package sandbox_test

func TestStream(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    s, err := box.Stream(context.Background(), []string{"sh", "-c", "echo a; sleep 0.1; echo b"})
    require.NoError(t, err)

    out, _ := io.ReadAll(s.Stdout)
    code, err := s.Wait()
    assert.NoError(t, err)
    assert.Equal(t, 0, code)
    assert.Contains(t, string(out), "a\n")
    assert.Contains(t, string(out), "b\n")
}

func TestStream_Cancel(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    s, err := box.Stream(context.Background(), []string{"sleep", "60"})
    require.NoError(t, err)

    s.Cancel()
    code, _ := s.Wait()
    assert.NotEqual(t, 0, code) // killed
}

func TestStream_Stdin(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    s, err := box.Stream(context.Background(), []string{"cat"})
    require.NoError(t, err)

    s.Stdin.Write([]byte("hello\n"))
    s.Stdin.Close()

    out, _ := io.ReadAll(s.Stdout)
    code, _ := s.Wait()
    assert.Equal(t, 0, code)
    assert.Equal(t, "hello\n", string(out))
}
```

### Workspace tests

```go
// box_test.go
package sandbox_test

func TestWorkspace_ReadWrite(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    ws := box.Workspace()
    err := ws.WriteFile("test.txt", []byte("hello"), 0644)
    require.NoError(t, err)

    data, err := fs.ReadFile(ws, "test.txt")
    require.NoError(t, err)
    assert.Equal(t, "hello", string(data))
}

func TestWorkspace_MkdirAll(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    ws := box.Workspace()
    err := ws.MkdirAll("a/b/c", 0755)
    require.NoError(t, err)

    info, err := fs.Stat(ws, "a/b/c")
    require.NoError(t, err)
    assert.True(t, info.IsDir())
}

func TestWorkspace_WalkDir(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    ws := box.Workspace()
    ws.MkdirAll("src", 0755)
    ws.WriteFile("src/main.go", []byte("package main"), 0644)
    ws.WriteFile("src/util.go", []byte("package main"), 0644)

    var files []string
    fs.WalkDir(ws, "src", func(path string, d fs.DirEntry, err error) error {
        if !d.IsDir() { files = append(files, path) }
        return nil
    })
    assert.Len(t, files, 2)
}
```

### Lifecycle tests

```go
// manager_lifecycle_test.go
package sandbox_test

func TestIdleCleanup_Session(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()

    box, err := sandbox.M().Create(context.Background(), sandbox.CreateOptions{
        Image:       testImage(),
        Policy:      sandbox.Session,
        IdleTimeout: 2 * time.Second,
    })
    require.NoError(t, err)

    // Box exists
    _, err = sandbox.M().Get(context.Background(), box.ID())
    assert.NoError(t, err)

    // Wait for idle + cleanup cycle
    time.Sleep(4 * time.Second)
    sandbox.M().Cleanup(context.Background())

    // Box should be gone
    _, err = sandbox.M().Get(context.Background(), box.ID())
    assert.ErrorIs(t, err, sandbox.ErrNotFound)
}

func TestStartRecovery(t *testing.T) {
    skipIfNoDocker(t)

    // Phase 1: create a box, then shut down Manager
    cleanup1 := setupManager(t)
    box, err := sandbox.M().Create(context.Background(), sandbox.CreateOptions{
        Image: testImage(),
        Owner: "recovery-test",
    })
    require.NoError(t, err)
    boxID := box.ID()
    cleanup1() // closes Manager, but container stays

    // Phase 2: new Manager, Start should discover the container
    cleanup2 := setupManager(t)
    defer cleanup2()

    recovered, err := sandbox.M().Get(context.Background(), boxID)
    require.NoError(t, err)
    assert.Equal(t, boxID, recovered.ID())
    assert.Equal(t, "recovery-test", recovered.Owner())

    // Clean up
    recovered.Remove(context.Background())
}

func TestHeartbeat(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()
    box := createTestBox(t)

    // Simulate heartbeat
    err := sandbox.M().Heartbeat(box.ID(), true, 3)
    assert.NoError(t, err)

    info, _ := box.Info(context.Background())
    assert.Equal(t, 3, info.ProcessCount)
}

func TestHeartbeat_NotFound(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()

    err := sandbox.M().Heartbeat("nonexistent", true, 1)
    assert.ErrorIs(t, err, sandbox.ErrNotFound)
}
```

### Pool management tests

```go
// manager_test.go
package sandbox_test

func TestPoolLimits_MaxTotal(t *testing.T) {
    skipIfNoDocker(t)

    // Init with MaxTotal=1
    err := sandbox.Init(sandbox.Config{
        Pool: []sandbox.Pool{{
            Name:     "limited",
            Addr:     "local",
            MaxTotal: 1,
        }},
    })
    require.NoError(t, err)
    sandbox.M().Start(context.Background())
    defer sandbox.M().Close()

    box1, err := sandbox.M().Create(context.Background(), sandbox.CreateOptions{
        Image: testImage(),
    })
    require.NoError(t, err)
    defer box1.Remove(context.Background())

    _, err = sandbox.M().Create(context.Background(), sandbox.CreateOptions{
        Image: testImage(),
    })
    assert.ErrorIs(t, err, sandbox.ErrLimitExceeded)
}

func TestAddPool(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()

    err := sandbox.M().AddPool(context.Background(), sandbox.Pool{
        Name: "new-pool",
        Addr: "local",
    })
    assert.NoError(t, err)

    pools := sandbox.M().Pools()
    names := make([]string, len(pools))
    for i, p := range pools { names[i] = p.Name }
    assert.Contains(t, names, "new-pool")
}

func TestRemovePool_InUse(t *testing.T) {
    skipIfNoDocker(t)
    cleanup := setupManager(t)
    defer cleanup()

    box := createTestBox(t)
    _ = box

    err := sandbox.M().RemovePool(context.Background(), "local", false)
    assert.ErrorIs(t, err, sandbox.ErrPoolInUse)
}
```

### Multi-pool tests

```go
// manager_test.go
package sandbox_test

func TestMultiNode(t *testing.T) {
    skipIfNoDocker(t)
    skipIfNoTai(t)
    cleanup := setupManagerWithRemote(t)
    defer cleanup()

    // Create on local
    local, err := sandbox.M().Create(context.Background(), sandbox.CreateOptions{
        Image: testImage(),
        Pool:  "local",
    })
    require.NoError(t, err)
    defer local.Remove(context.Background())

    // Create on remote
    remote, err := sandbox.M().Create(context.Background(), sandbox.CreateOptions{
        Image: testImage(),
        Pool:  "remote",
    })
    require.NoError(t, err)
    defer remote.Remove(context.Background())

    // Both should exec
    r1, _ := local.Exec(context.Background(), []string{"echo", "local"})
    r2, _ := remote.Exec(context.Background(), []string{"echo", "remote"})
    assert.Equal(t, "local\n", r1.Stdout)
    assert.Equal(t, "remote\n", r2.Stdout)
}
```

### OAuth / gRPC env injection tests

```go
// grpc_test.go
package sandbox_test

func TestBuildGRPCEnv_Local(t *testing.T) {
    env := sandbox.BuildGRPCEnv(&sandbox.Pool{Addr: "local"}, "sb-001", "tok", "ref")
    assert.Equal(t, "sb-001", env["YAO_SANDBOX_ID"])
    assert.Equal(t, "tok", env["YAO_TOKEN"])
    assert.Equal(t, "ref", env["YAO_REFRESH_TOKEN"])
    assert.NotEmpty(t, env["YAO_GRPC_ADDR"])
}

func TestBuildGRPCEnv_Remote(t *testing.T) {
    env := sandbox.BuildGRPCEnv(&sandbox.Pool{Addr: "tai://gpu.internal"}, "sb-002", "tok", "ref")
    assert.NotEmpty(t, env["YAO_GRPC_ADDR"])
}

func TestCreateContainerTokens(t *testing.T) {
    // Requires Yao runtime for OAuth
    cleanup := setupManager(t)
    defer cleanup()

    access, refresh, err := sandbox.CreateContainerTokens("sb-test", "user-1")
    require.NoError(t, err)
    assert.NotEmpty(t, access)
    assert.NotEmpty(t, refresh)
}
```

## Required Test Cases

| File | Required Cases |
|------|---------------|
| `sandbox_test.go` | `Init` succeeds / `M()` panics before Init / double Init is safe |
| `manager_test.go` | Create / Create with explicit ID / Create no image (error) / Get / Get not found / GetOrCreate / List / List with owner filter / Remove / pool limits MaxTotal / pool limits MaxPerUser / AddPool / RemovePool / RemovePool in use / Pools |
| `manager_lifecycle_test.go` | Start recovery from labels / Cleanup Session idle / Cleanup LongRunning stop then remove / Persistent never cleaned / Heartbeat updates / Heartbeat not found / OneShot removed after Exec |
| `box_test.go` | Exec success / Exec non-zero exit / Exec with WorkDir / Exec with Env / Exec with Timeout / Stream read / Stream cancel / Stream stdin / Workspace ReadFile+WriteFile / Workspace MkdirAll / Workspace Remove / Workspace Rename / Workspace WalkDir / VNC (skip if no VNC image) / Proxy URL / Start+Stop+Start / Info |
| `box_attach_test.go` | Attach WS (skip if no WS server image) / Attach SSE (skip if no SSE server image) |
| `grpc_test.go` | BuildGRPCEnv local / BuildGRPCEnv remote / CreateContainerTokens / RevokeContainerTokens |

## Makefile

Add to [Makefile](../../Makefile):

```makefile
TESTFOLDER_SANDBOX_V2 := $(shell $(GO) list ./sandbox/v2/...)

.PHONY: unit-test-sandbox-v2
unit-test-sandbox-v2:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER_SANDBOX_V2); do \
		$(GO) test -tags $(TESTTAGS) -v -timeout=10m \
			-covermode=count -coverprofile=profile.out \
			-coverpkg=$$d \
			$$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done
```

## CI Integration

Add `sandbox-v2-test` job to `unit-test.yml` and `pr-test.yml`:

```yaml
sandbox-v2-test:
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
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go }}

    - name: Start Tai container
      run: |
        docker run -d --name tai \
          -v /var/run/docker.sock:/var/run/docker.sock \
          -p 2375:2375 -p 9100:9100 -p 8080:8080 -p 6080:6080 \
          yaoapp/tai:latest
        sleep 3

    - name: Build V2 test image
      run: |
        cd sandbox/docker
        bash build.sh v2

    - name: Setup ENV
      run: |
        echo "YAO_DB_DRIVER=sqlite3" >> $GITHUB_ENV
        echo "YAO_DB_PRIMARY=${{ github.WORKSPACE }}/../app/db/yao.db" >> $GITHUB_ENV
        echo "SANDBOX_TEST_IMAGE=yaoapp/sandbox-v2-test:latest" >> $GITHUB_ENV
        echo "SANDBOX_TEST_REMOTE_ADDR=tai://127.0.0.1" >> $GITHUB_ENV
        echo "TAI_TEST_HOST=127.0.0.1" >> $GITHUB_ENV
        mkdir -p ${{ github.WORKSPACE }}/../app/db

    - name: Run Sandbox V2 Tests
      run: make unit-test-sandbox-v2

    - name: Codecov Report
      uses: codecov/codecov-action@v4
      with:
        token: ${{ secrets.CODECOV_TOKEN }}
```

Key decisions:
- SQLite only — sandbox is infrastructure, not data-model dependent
- Tai container provides remote mode — exercises the full proxy path
- `sandbox-v2-test` as default test image — includes `tai` (heartbeat), `openai-proxy`, Nginx, WS echo + SSE test services
- CI builds test image from source (Step 4.5) — ensures binary compatibility with latest tai SDK changes
- Attach tests (WS/SSE) use `sandbox-v2-test` image's built-in test services

## Coverage

- Target: >80% per file, >80% overall
- `sandbox.go` (singleton) covered via `sandbox_test.go`
- `manager.go` is the heaviest file — must have dedicated `manager_test.go` + `manager_lifecycle_test.go`
- `box.go` exercises all tai SDK integration points
- `grpc.go` tested with pure unit tests (token generation, env building)

## Running Tests

```bash
# All sandbox v2 tests (local Docker only)
make unit-test-sandbox-v2

# With remote mode (start Tai first)
SANDBOX_TEST_REMOTE_ADDR=tai://127.0.0.1 make unit-test-sandbox-v2

# Single file
go test -v ./sandbox/v2/ -run TestCreate

# Single test
go test -v ./sandbox/v2/ -run TestExec_NonZeroExit

# With race detector
go test -race -v ./sandbox/v2/
```
