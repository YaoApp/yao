package sandbox_test

import (
	"context"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestCreateAndExec(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := box.Exec(ctx, []string{"echo", "hello"})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if result.ExitCode != 0 {
				t.Errorf("exit code = %d, want 0", result.ExitCode)
			}
			if result.Stdout != "hello\n" {
				t.Errorf("stdout = %q, want %q", result.Stdout, "hello\n")
			}
		})
	}
}

func TestCreateWithLabels(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc, func(co *sandbox.CreateOptions) {
				co.Labels = map[string]string{"app": "test-app"}
			})

			ctx := context.Background()
			info, err := box.Info(ctx)
			if err != nil {
				t.Fatalf("Info: %v", err)
			}
			if info.Labels["app"] != "test-app" {
				t.Errorf("label app = %q, want %q", info.Labels["app"], "test-app")
			}
		})
	}
}

func TestGet(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			got, err := m.Get(context.Background(), box.ID())
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.ID() != box.ID() {
				t.Errorf("ID = %q, want %q", got.ID(), box.ID())
			}
		})
	}
}

func TestGetNotFound(t *testing.T) {
	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			_, err := m.Get(context.Background(), "nonexistent")
			if err != sandbox.ErrNotFound {
				t.Errorf("err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestList(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc, func(co *sandbox.CreateOptions) {
				co.Owner = "user-list"
			})

			boxes, err := m.List(context.Background(), sandbox.ListOptions{Owner: "user-list"})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			found := false
			for _, b := range boxes {
				if b.ID() == box.ID() {
					found = true
				}
			}
			if !found {
				t.Error("created box not found in list")
			}

			empty, err := m.List(context.Background(), sandbox.ListOptions{Owner: "nobody"})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(empty) != 0 {
				t.Errorf("expected 0 results for unknown owner, got %d", len(empty))
			}
		})
	}
}

func TestRemove(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			ensureTestImage(t, m, pc.TaiID)
			ctx := context.Background()
			box, err := m.Create(ctx, sandbox.CreateOptions{
				Image:  testImage(),
				Owner:  "test-user",
				NodeID: pc.TaiID,
			})
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			if err := m.Remove(ctx, box.ID()); err != nil {
				t.Fatalf("Remove: %v", err)
			}

			_, err = m.Get(ctx, box.ID())
			if err != sandbox.ErrNotFound {
				t.Errorf("after Remove, Get err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestCreateNoImage(t *testing.T) {
	m, nodes := setupManager(t, nodeConfig{Name: "local", Addr: testLocalAddr()})

	_, err := m.Create(context.Background(), sandbox.CreateOptions{
		Owner:  "test",
		NodeID: nodes[0].TaiID,
	})
	if err == nil {
		t.Error("expected error for missing image")
	}
}

func TestCreateNoNodeID(t *testing.T) {
	m, _ := setupManager(t, nodeConfig{Name: "local", Addr: testLocalAddr()})

	_, err := m.Create(context.Background(), sandbox.CreateOptions{
		Image: testImage(),
	})
	if err != sandbox.ErrNodeMissing {
		t.Errorf("err = %v, want ErrNodeMissing", err)
	}
}

func TestMultiNode(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoTai(t)

	nodes := testNodes()
	if len(nodes) < 2 {
		t.Skip("need at least 2 nodes (local + remote) for multi-node test")
	}

	m, registered := setupManager(t, nodes...)

	for _, pc := range registered {
		ensureTestImage(t, m, pc.TaiID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	localBox, err := m.Create(ctx, sandbox.CreateOptions{
		Image:  testImage(),
		Owner:  "test-user",
		NodeID: registered[0].TaiID,
	})
	if err != nil {
		t.Fatalf("Create on local: %v", err)
	}
	defer m.Remove(ctx, localBox.ID())

	remoteBox, err := m.Create(ctx, sandbox.CreateOptions{
		Image:  testImage(),
		Owner:  "test-user",
		NodeID: registered[1].TaiID,
	})
	if err != nil {
		t.Fatalf("Create on remote: %v", err)
	}
	defer m.Remove(ctx, remoteBox.ID())

	r1, err := localBox.Exec(ctx, []string{"echo", "local"})
	if err != nil {
		t.Fatalf("Exec on local: %v", err)
	}
	if r1.Stdout != "local\n" {
		t.Errorf("local stdout = %q, want %q", r1.Stdout, "local\n")
	}

	r2, err := remoteBox.Exec(ctx, []string{"echo", "remote"})
	if err != nil {
		t.Fatalf("Exec on remote: %v", err)
	}
	if r2.Stdout != "remote\n" {
		t.Errorf("remote stdout = %q, want %q", r2.Stdout, "remote\n")
	}

	localInfo, err := localBox.Info(ctx)
	if err != nil {
		t.Fatalf("Info local: %v", err)
	}
	remoteInfo, err := remoteBox.Info(ctx)
	if err != nil {
		t.Fatalf("Info remote: %v", err)
	}
	if localInfo.ID == remoteInfo.ID {
		t.Error("local and remote boxes should have different IDs")
	}
}
