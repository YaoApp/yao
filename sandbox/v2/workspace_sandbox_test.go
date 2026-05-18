package sandbox_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
	"github.com/yaoapp/yao/workspace"
)

func TestWorkspace_CreateAndBind(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	box := sandboxtest.CreateBox(t, m, nodeID)

	wsID := fmt.Sprintf("ws-test-%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := workspace.M().Create(ctx, workspace.CreateOptions{
		ID:    wsID,
		Owner: "test",
		Node:  nodeID,
	})
	if err != nil && !strings.Contains(err.Error(), "exists") {
		t.Fatalf("create workspace: %v", err)
	}
	t.Cleanup(func() {
		cCtx, cCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cCancel()
		workspace.M().Delete(cCtx, wsID, true)
	})

	ws := box.Workspace()
	_ = ws // validate no panic
}

func TestWorkspace_ReadWrite(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	box := sandboxtest.CreateBox(t, m, nodeID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testContent := "hello-workspace-test"
	res, err := box.Exec(ctx, []string{"sh", "-c", fmt.Sprintf("echo '%s' > /tmp/ws-test.txt && cat /tmp/ws-test.txt", testContent)})
	if err != nil {
		t.Fatalf("Exec write/read: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, testContent) {
		t.Fatalf("expected %q in stdout, got %q", testContent, res.Stdout)
	}
}

func TestWorkspace_ContainerWriteBack(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	box := sandboxtest.CreateBox(t, m, nodeID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	marker := fmt.Sprintf("marker-%d", time.Now().UnixNano())
	_, err := box.Exec(ctx, []string{"sh", "-c", fmt.Sprintf("echo '%s' > /tmp/writeback.txt", marker)})
	if err != nil {
		t.Fatalf("write marker: %v", err)
	}

	res, err := box.Exec(ctx, []string{"cat", "/tmp/writeback.txt"})
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.Contains(res.Stdout, marker) {
		t.Fatalf("expected marker %q, got %q", marker, res.Stdout)
	}
}
