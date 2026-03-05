# Workspace — Test Specification

Design: [DESIGN.md](./DESIGN.md)

## Principles

- **Black-box testing**: all `*_test.go` files use `package workspace_test` — tests only access exported API
- **No Docker required**: workspace unit tests use `volume.NewLocal(t.TempDir())` via `tai.WithVolume` — no Docker daemon needed
- **Skip when unavailable**: `skipIfNoTai(t)` for remote-mode tests
- **Tests follow implementation**: `*_test.go` lives next to the code it tests
- **Coverage > 80%**: per file and overall

## Prerequisites

No external services required for unit tests. Tests create a temp directory for storage.

### Remote mode (optional)

For remote-mode tests via Tai gRPC:

```bash
SANDBOX_TEST_REMOTE_ADDR=tai://127.0.0.1 go test -v ./workspace/
```

## Directory Structure

```
workspace/
├── workspace.go         # Types (Workspace, CreateOptions, MountMode, etc.)
├── errors.go            # Error definitions
├── manager.go           # Manager (CRUD, file I/O, Nodes)
├── workspace_test.go    # CRUD tests
├── fileio_test.go       # File I/O + FS tests
├── testutils_test.go    # Shared test helpers
├── DESIGN.md            # Design document
├── TEST.md              # This file
└── Makefile             # Test runner
```

## testutils (internal to workspace_test)

```go
// testutils_test.go
package workspace_test

func setupManager(t *testing.T) *workspace.Manager
func setupManagerMultiNode(t *testing.T) *workspace.Manager
func localClient(t *testing.T, dataDir string) *tai.Client
func createTestWorkspace(t *testing.T, m *workspace.Manager, opts ...func(*workspace.CreateOptions)) *workspace.Workspace
func skipIfNoTai(t *testing.T)
```

## Required Test Cases

| File | Required Cases |
|------|---------------|
| `workspace_test.go` | Create / Create auto ID / Create explicit ID / Create with labels / Create invalid node / Create node not found / Get / Get not found / List / List filter owner / List filter node / Update name / Update labels / Update not found / Delete / Delete not found / Nodes / NodeForWorkspace / NodeForWorkspace not found |
| `fileio_test.go` | ReadWriteFile / WriteFile nested path / ListDir / Remove file / FS ReadFile / FS WriteFile / FS MkdirAll / FS Rename / FS WalkDir / FS Remove / FS not found |

## Running Tests

```bash
# All workspace tests (no Docker needed)
make -C workspace test

# Single test
go test -v ./workspace/ -run TestCreate

# With race detector
go test -race -v ./workspace/

# With coverage
go test -v -coverprofile=coverage.out ./workspace/
```
