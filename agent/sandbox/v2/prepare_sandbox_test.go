package sandboxv2_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func TestRunPrepareSteps_Exec(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	box := sandboxtest.CreateBox(t, m, nodeID)

	steps := []types.PrepareStep{
		{Action: "exec", Cmd: "echo prepare-test"},
	}
	err := sandboxv2.RunPrepareSteps(context.Background(), steps, box, "test-ast", "hash1", "")
	if err != nil {
		t.Fatalf("RunPrepareSteps exec: %v", err)
	}
}

func TestRunPrepareSteps_File(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	box := sandboxtest.CreateBox(t, m, nodeID)

	// runFileStep writes to the workspace volume (via Tai), NOT to the
	// container's local filesystem. Verify the round-trip through the same
	// workspace API rather than via `cat` (which only sees the container fs).
	// Use a relative path because workspace.FS.ReadFile enforces fs.ValidPath
	// (no leading slash, no "..").
	const filePath = "tmp/prepare-file.txt"
	const content = "hello prepare"

	steps := []types.PrepareStep{
		{Action: "file", Path: filePath, Content: []byte(content)},
	}
	if err := sandboxv2.RunPrepareSteps(context.Background(), steps, box, "test-ast", "hash1", ""); err != nil {
		t.Fatalf("RunPrepareSteps file: %v", err)
	}

	ws := box.Workspace()
	if ws == nil {
		t.Fatal("box.Workspace() returned nil")
	}
	got, err := ws.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ws.ReadFile(%s): %v", filePath, err)
	}
	if string(got) != content {
		t.Fatalf("file content mismatch: got %q, want %q", string(got), content)
	}
}

func TestRunPrepareSteps_Once(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	box := sandboxtest.CreateBox(t, m, nodeID)

	marker := fmt.Sprintf("once-%d", time.Now().UnixNano())
	steps := []types.PrepareStep{
		{Action: "exec", Cmd: fmt.Sprintf("echo %s >> /tmp/once-log.txt", marker), Once: true},
	}

	err := sandboxv2.RunPrepareSteps(context.Background(), steps, box, "test-ast", "hash-once", "")
	if err != nil {
		t.Fatalf("RunPrepareSteps first: %v", err)
	}

	err = sandboxv2.RunPrepareSteps(context.Background(), steps, box, "test-ast", "hash-once", "")
	if err != nil {
		t.Fatalf("RunPrepareSteps second: %v", err)
	}
}

func TestRunPrepareSteps_Empty(t *testing.T) {
	testprepare.PrepareSandbox(t)

	err := sandboxv2.RunPrepareSteps(context.Background(), nil, nil, "", "", "")
	if err != nil {
		t.Fatalf("expected no error for empty steps, got %v", err)
	}
}

func TestRunPrepareSteps_HostExec(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	hostTaiID := sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr())
	host := sandboxtest.CreateHost(t, m, hostTaiID)

	steps := []types.PrepareStep{
		{Action: "exec", Cmd: "echo host-prepare"},
	}
	err := sandboxv2.RunPrepareSteps(context.Background(), steps, host, "test-ast", "hash2", "")
	if err != nil {
		t.Fatalf("RunPrepareSteps on host: %v", err)
	}
}

func TestRunPrepareSteps_Error(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	box := sandboxtest.CreateBox(t, m, nodeID)

	steps := []types.PrepareStep{
		{Action: "exec", Cmd: "nonexistent-command-xyz"},
	}
	err := sandboxv2.RunPrepareSteps(context.Background(), steps, box, "test-ast", "hash3", "")
	if err == nil {
		t.Fatal("expected error for invalid command")
	}
}
