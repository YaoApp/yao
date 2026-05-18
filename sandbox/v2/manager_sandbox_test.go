package sandbox_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func TestManager_Create(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))
	if box.ID() == "" {
		t.Fatal("expected non-empty box ID")
	}
}

func TestManager_CreateWithLabels(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	labels := map[string]string{"env": "test", "owner": "ci"}
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()), func(co *sandbox.CreateOptions) {
		co.Labels = labels
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := box.Info(ctx)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil BoxInfo")
	}
}

func TestManager_CreateNoImage(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := m.Create(ctx, sandbox.CreateOptions{
		Owner:  "test-user",
		NodeID: sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()),
	})
	if err == nil {
		t.Fatal("expected error when Image is empty")
	}
}

func TestManager_Get(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := m.Get(ctx, box.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID() != box.ID() {
		t.Fatalf("ID mismatch: got %s, want %s", got.ID(), box.ID())
	}
}

func TestManager_GetNotFound(t *testing.T) {
	testprepare.PrepareSandbox(t)

	m := sandbox.M()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.Get(ctx, "nonexistent-id-12345")
	if err != sandbox.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_GetOrCreate(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	id := fmt.Sprintf("sb-goc-%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	box1, err := m.GetOrCreate(ctx, sandbox.CreateOptions{
		ID: id, Image: sandboxtest.TestImage(), Owner: "test-user", NodeID: nodeID,
	})
	if err != nil {
		t.Fatalf("GetOrCreate first: %v", err)
	}
	t.Cleanup(func() {
		cCtx, cCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cCancel()
		m.Remove(cCtx, box1.ID())
	})

	box2, err := m.GetOrCreate(ctx, sandbox.CreateOptions{
		ID: id, Image: sandboxtest.TestImage(), Owner: "test-user", NodeID: nodeID,
	})
	if err != nil {
		t.Fatalf("GetOrCreate second: %v", err)
	}
	if box2.ID() != box1.ID() {
		t.Fatalf("GetOrCreate not idempotent: %s != %s", box2.ID(), box1.ID())
	}
}

func TestManager_List(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())

	box1 := sandboxtest.CreateBox(t, m, nodeID, func(co *sandbox.CreateOptions) {
		co.Owner = "list-test-owner"
	})
	box2 := sandboxtest.CreateBox(t, m, nodeID, func(co *sandbox.CreateOptions) {
		co.Owner = "list-test-owner"
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	boxes, err := m.List(ctx, sandbox.ListOptions{Owner: "list-test-owner"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ids := map[string]bool{}
	for _, b := range boxes {
		ids[b.ID()] = true
	}
	if !ids[box1.ID()] || !ids[box2.ID()] {
		t.Fatalf("List missing boxes: got %v, want %s and %s", ids, box1.ID(), box2.ID())
	}
}

func TestManager_ListByOwner(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())

	sandboxtest.CreateBox(t, m, nodeID, func(co *sandbox.CreateOptions) {
		co.Owner = "owner-a"
	})
	sandboxtest.CreateBox(t, m, nodeID, func(co *sandbox.CreateOptions) {
		co.Owner = "owner-b"
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	boxes, err := m.List(ctx, sandbox.ListOptions{Owner: "owner-a"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, b := range boxes {
		if b.Owner() != "owner-a" {
			t.Fatalf("unexpected owner %q in filtered list", b.Owner())
		}
	}
}

func TestManager_Remove(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	box, err := m.Create(ctx, sandbox.CreateOptions{
		ID:     fmt.Sprintf("sb-rm-%d", time.Now().UnixNano()),
		Image:  sandboxtest.TestImage(),
		Owner:  "test-user",
		NodeID: nodeID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := m.Remove(ctx, box.ID()); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, err = m.Get(ctx, box.ID())
	if err != sandbox.ErrNotFound {
		t.Fatalf("expected ErrNotFound after Remove, got %v", err)
	}
}

func TestManager_Heartbeat(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	if err := m.Heartbeat(box.ID(), true, 1); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	if err := m.Heartbeat("nonexistent-heartbeat", true, 0); err != sandbox.ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown sandbox, got %v", err)
	}
}

func TestManager_StartBox(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := box.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := m.StartBox(ctx, box.ID()); err != nil {
		t.Fatalf("StartBox: %v", err)
	}

	res, err := box.Exec(ctx, []string{"echo", "alive"})
	if err != nil {
		t.Fatalf("Exec after StartBox: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d", res.ExitCode)
	}
}

func TestManager_Nodes(t *testing.T) {
	testprepare.PrepareSandbox(t)

	m := sandbox.M()
	nodes := m.Nodes()
	if len(nodes) == 0 {
		t.Fatal("expected at least one registered node")
	}
}
