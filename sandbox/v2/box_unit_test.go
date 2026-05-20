package sandbox_test

import (
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func newTestBox(opts ...func(*testBoxOpts)) *sandbox.Box {
	o := testBoxOpts{
		id: "sb-test", owner: "user-1", containerID: "ctr-abc",
		nodeID: "node-1", workDir: "/app", image: "alpine:latest",
		policy: sandbox.OneShot, displayName: "Test Box",
		labels: map[string]string{"env": "test"},
		sys:    sandbox.SystemInfo{OS: "linux", Arch: "amd64", Shell: "bash"},
	}
	for _, fn := range opts {
		fn(&o)
	}
	return sandbox.ExportNewBoxForTest(
		o.id, o.owner, o.containerID, o.nodeID, o.workDir, o.image,
		o.policy, o.labels, o.displayName, o.sys,
		o.idleTimeout, o.maxLifetime, o.stopTimeout,
	)
}

type testBoxOpts struct {
	id, owner, containerID, nodeID, workDir, image, displayName string
	policy                                                      sandbox.LifecyclePolicy
	labels                                                      map[string]string
	sys                                                         sandbox.SystemInfo
	idleTimeout, maxLifetime, stopTimeout                       time.Duration
}

func TestBoxAccessors(t *testing.T) {
	b := newTestBox()
	if b.ID() != "sb-test" {
		t.Errorf("ID: got %q", b.ID())
	}
	if b.Owner() != "user-1" {
		t.Errorf("Owner: got %q", b.Owner())
	}
	if b.ContainerID() != "ctr-abc" {
		t.Errorf("ContainerID: got %q", b.ContainerID())
	}
	if b.NodeID() != "node-1" {
		t.Errorf("NodeID: got %q", b.NodeID())
	}
}

func TestBoxComputerInfo(t *testing.T) {
	b := newTestBox()
	info := b.ComputerInfo()

	if info.Kind != "box" {
		t.Errorf("Kind: got %q", info.Kind)
	}
	if info.NodeID != "node-1" {
		t.Errorf("NodeID: got %q", info.NodeID)
	}
	if info.BoxID != "sb-test" {
		t.Errorf("BoxID: got %q", info.BoxID)
	}
	if info.ContainerID != "ctr-abc" {
		t.Errorf("ContainerID: got %q", info.ContainerID)
	}
	if info.Owner != "user-1" {
		t.Errorf("Owner: got %q", info.Owner)
	}
	if info.Image != "alpine:latest" {
		t.Errorf("Image: got %q", info.Image)
	}
	if info.Policy != sandbox.OneShot {
		t.Errorf("Policy: got %q", info.Policy)
	}
	if info.DisplayName != "Test Box" {
		t.Errorf("DisplayName: got %q", info.DisplayName)
	}
	if info.System.OS != "linux" {
		t.Errorf("System.OS: got %q", info.System.OS)
	}
	if info.System.Shell != "bash" {
		t.Errorf("System.Shell: got %q", info.System.Shell)
	}
	if info.Status != "online" {
		t.Errorf("Status: got %q", info.Status)
	}
}

func TestBoxGetWorkDir_Set(t *testing.T) {
	b := newTestBox()
	if b.GetWorkDir() != "/app" {
		t.Errorf("got %q, want %q", b.GetWorkDir(), "/app")
	}
}

func TestBoxGetWorkDir_Default(t *testing.T) {
	b := newTestBox(func(o *testBoxOpts) { o.workDir = "" })
	if b.GetWorkDir() != "/workspace" {
		t.Errorf("got %q, want %q", b.GetWorkDir(), "/workspace")
	}
}

func TestBoxBindWorkplace(t *testing.T) {
	b := newTestBox()
	if b.WorkspaceID() != "" {
		t.Errorf("initial WorkspaceID should be empty, got %q", b.WorkspaceID())
	}
	b.BindWorkplace("ws-123")
	if b.WorkspaceID() != "ws-123" {
		t.Errorf("after BindWorkplace: got %q, want %q", b.WorkspaceID(), "ws-123")
	}
}

func TestBoxSnapshot(t *testing.T) {
	b := newTestBox()
	snap := b.Snapshot()

	if snap.ID != "sb-test" {
		t.Errorf("ID: got %q", snap.ID)
	}
	if snap.ContainerID != "ctr-abc" {
		t.Errorf("ContainerID: got %q", snap.ContainerID)
	}
	if snap.NodeID != "node-1" {
		t.Errorf("NodeID: got %q", snap.NodeID)
	}
	if snap.Owner != "user-1" {
		t.Errorf("Owner: got %q", snap.Owner)
	}
	if snap.Policy != sandbox.OneShot {
		t.Errorf("Policy: got %q", snap.Policy)
	}
	if snap.Image != "alpine:latest" {
		t.Errorf("Image: got %q", snap.Image)
	}
	if snap.Status != "unknown" {
		t.Errorf("Status: got %q (expected unknown for new box)", snap.Status)
	}
	if snap.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestBoxIsStopped(t *testing.T) {
	b := newTestBox()
	if b.IsStopped() {
		t.Error("new box should not be stopped")
	}
}

func TestBoxComputerInfo_WindowsSystem(t *testing.T) {
	b := newTestBox(func(o *testBoxOpts) {
		o.sys = sandbox.SystemInfo{OS: "windows", Arch: "amd64", Shell: "pwsh"}
	})
	info := b.ComputerInfo()
	if info.System.OS != "windows" {
		t.Errorf("OS: got %q", info.System.OS)
	}
	if info.System.Shell != "pwsh" {
		t.Errorf("Shell: got %q", info.System.Shell)
	}
}
