package registry

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/tai/types"
)

// newTestRegistry creates a standalone registry for testing (bypasses global singleton).
func newTestRegistry() *Registry {
	return &Registry{
		nodes:  make(map[string]*TaiNode),
		logger: slog.Default(),
	}
}

func TestRegister_SetsFieldsAndOnline(t *testing.T) {
	r := newTestRegistry()
	node := &TaiNode{
		TaiID:     "tai-001",
		MachineID: "m-abc",
		Version:   "1.0.0",
		Mode:      "tunnel",
		Ports:     types.Ports{GRPC: 19100},
	}
	r.Register(node)

	snap, ok := r.Get("tai-001")
	if !ok {
		t.Fatal("expected node to exist after Register")
	}
	if snap.Status != "online" {
		t.Errorf("Status = %q, want online", snap.Status)
	}
	if snap.MachineID != "m-abc" {
		t.Errorf("MachineID = %q, want m-abc", snap.MachineID)
	}
	if snap.ConnectedAt.IsZero() {
		t.Error("ConnectedAt should be set")
	}
}

func TestRegister_Overwrite(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001", Version: "1.0"})
	r.Register(&TaiNode{TaiID: "tai-001", Version: "2.0"})

	snap, ok := r.Get("tai-001")
	if !ok {
		t.Fatal("node should exist")
	}
	if snap.Version != "2.0" {
		t.Errorf("Version = %q, want 2.0 after re-register", snap.Version)
	}
}

func TestUnregister_RemovesNode(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001"})
	r.Unregister("tai-001")

	if _, ok := r.Get("tai-001"); ok {
		t.Error("expected node to be removed after Unregister")
	}
}

func TestUnregister_Nonexistent(t *testing.T) {
	r := newTestRegistry()
	r.Unregister("ghost")
}

func TestGet_NotFound(t *testing.T) {
	r := newTestRegistry()
	if _, ok := r.Get("missing"); ok {
		t.Error("expected false for missing node")
	}
}

func TestList_Empty(t *testing.T) {
	r := newTestRegistry()
	if got := r.List(); len(got) != 0 {
		t.Errorf("List() = %d items, want 0", len(got))
	}
}

func TestList_MultipleNodes(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "a"})
	r.Register(&TaiNode{TaiID: "b"})
	r.Register(&TaiNode{TaiID: "c"})

	list := r.List()
	if len(list) != 3 {
		t.Errorf("List() = %d items, want 3", len(list))
	}

	ids := map[string]bool{}
	for _, snap := range list {
		ids[snap.TaiID] = true
	}
	for _, id := range []string{"a", "b", "c"} {
		if !ids[id] {
			t.Errorf("missing node %q in List()", id)
		}
	}
}

func TestSnapshot_DeepCopy(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{
		TaiID: "tai-001",
		Ports: types.Ports{GRPC: 19100, HTTP: 8099},
	})

	snap, _ := r.Get("tai-001")
	snap.Ports.GRPC = 0

	snap2, _ := r.Get("tai-001")
	if snap2.Ports.GRPC != 19100 {
		t.Error("snapshot modification leaked into registry node")
	}
}

func TestUpdatePing(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001"})
	time.Sleep(10 * time.Millisecond)

	r.UpdatePing("tai-001")
	snap, _ := r.Get("tai-001")
	if snap.LastPing.Before(snap.ConnectedAt) {
		t.Error("LastPing should be after ConnectedAt")
	}
}

func TestUpdatePing_NonexistentNode(t *testing.T) {
	r := newTestRegistry()
	r.UpdatePing("ghost")
}

func TestGenerateChannelID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateChannelID()
		if err != nil {
			t.Fatalf("generateChannelID: %v", err)
		}
		if len(id) != 64 {
			t.Errorf("len = %d, want 64 hex chars", len(id))
		}
		if seen[id] {
			t.Fatalf("duplicate channel ID: %s", id)
		}
		seen[id] = true
	}
}

func TestConcurrentRegisterGet(t *testing.T) {
	r := newTestRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		id := "tai-" + string(rune('A'+i%26))

		go func() {
			defer wg.Done()
			r.Register(&TaiNode{TaiID: id, Mode: "tunnel"})
		}()

		go func() {
			defer wg.Done()
			r.Get(id)
			r.List()
		}()
	}

	wg.Wait()
}

func TestOpenLocalListener_NodeNotFound(t *testing.T) {
	r := newTestRegistry()
	_, err := r.OpenLocalListener("ghost", 19100)
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestRegister_SystemInfo(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{
		TaiID: "tai-001",
		System: types.SystemInfo{
			OS:       "linux",
			Arch:     "amd64",
			Hostname: "docker-host-01",
			NumCPU:   16,
		},
	})

	snap, ok := r.Get("tai-001")
	if !ok {
		t.Fatal("node not found")
	}
	if snap.System.OS != "linux" {
		t.Errorf("System.OS = %q, want linux", snap.System.OS)
	}
	if snap.System.Arch != "amd64" {
		t.Errorf("System.Arch = %q, want amd64", snap.System.Arch)
	}
	if snap.System.Hostname != "docker-host-01" {
		t.Errorf("System.Hostname = %q, want docker-host-01", snap.System.Hostname)
	}
	if snap.System.NumCPU != 16 {
		t.Errorf("System.NumCPU = %d, want 16", snap.System.NumCPU)
	}
}

func TestListByTeam(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-a", Auth: types.AuthInfo{TeamID: "team-dev"}})
	r.Register(&TaiNode{TaiID: "tai-b", Auth: types.AuthInfo{TeamID: "team-dev"}})
	r.Register(&TaiNode{TaiID: "tai-c", Auth: types.AuthInfo{TeamID: "team-ops"}})

	devNodes := r.ListByTeam("team-dev")
	if len(devNodes) != 2 {
		t.Errorf("ListByTeam(team-dev) = %d nodes, want 2", len(devNodes))
	}

	opsNodes := r.ListByTeam("team-ops")
	if len(opsNodes) != 1 {
		t.Errorf("ListByTeam(team-ops) = %d nodes, want 1", len(opsNodes))
	}

	empty := r.ListByTeam("team-ghost")
	if len(empty) != 0 {
		t.Errorf("ListByTeam(team-ghost) = %d nodes, want 0", len(empty))
	}
}

func TestStartHealthCheck_MarkOffline(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-direct", Mode: "direct"})
	r.Register(&TaiNode{TaiID: "tai-tunnel", Mode: "tunnel"})

	// Manually set LastPing to the past for the direct node.
	r.mu.Lock()
	r.nodes["tai-direct"].LastPing = time.Now().Add(-5 * time.Second)
	r.mu.Unlock()

	done := make(chan struct{})
	r.StartHealthCheck(done, 50*time.Millisecond, 2*time.Second, 10*time.Minute)
	defer close(done)

	time.Sleep(200 * time.Millisecond)

	snap, ok := r.Get("tai-direct")
	if !ok {
		t.Fatal("direct node should still exist")
	}
	if snap.Status != "offline" {
		t.Errorf("direct node Status = %q, want offline", snap.Status)
	}

	// Tunnel nodes should not be affected.
	snap2, ok := r.Get("tai-tunnel")
	if !ok {
		t.Fatal("tunnel node should still exist")
	}
	if snap2.Status != "online" {
		t.Errorf("tunnel node Status = %q, want online", snap2.Status)
	}
}

func TestStartHealthCheck_AutoCleanup(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-stale", Mode: "direct"})

	// Set LastPing far in the past so it exceeds both timeout and cleanupAfter.
	r.mu.Lock()
	r.nodes["tai-stale"].LastPing = time.Now().Add(-1 * time.Hour)
	r.mu.Unlock()

	done := make(chan struct{})
	r.StartHealthCheck(done, 50*time.Millisecond, 1*time.Second, 1*time.Second)
	defer close(done)

	time.Sleep(200 * time.Millisecond)

	if _, ok := r.Get("tai-stale"); ok {
		t.Error("stale node should have been auto-unregistered")
	}
}

func TestStartHealthCheck_PingKeepsAlive(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-alive", Mode: "direct"})

	done := make(chan struct{})
	r.StartHealthCheck(done, 50*time.Millisecond, 2*time.Second, 10*time.Minute)
	defer close(done)

	// Continuously ping to keep the node alive.
	for i := 0; i < 4; i++ {
		time.Sleep(30 * time.Millisecond)
		r.UpdatePing("tai-alive")
	}

	snap, ok := r.Get("tai-alive")
	if !ok {
		t.Fatal("node should still exist")
	}
	if snap.Status != "online" {
		t.Errorf("Status = %q, want online", snap.Status)
	}
}

func TestNodeMeta_AuthInfo(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{
		TaiID: "tai-001",
		Auth: types.AuthInfo{
			Subject:  "user123",
			ClientID: "tai-001",
			Scope:    "tai:tunnel",
		},
	})

	snap, ok := r.Get("tai-001")
	if !ok {
		t.Fatal("node not found")
	}
	if snap.Auth.Subject != "user123" {
		t.Errorf("Auth.Subject = %q, want user123", snap.Auth.Subject)
	}
	if snap.Auth.Scope != "tai:tunnel" {
		t.Errorf("Auth.Scope = %q, want tai:tunnel", snap.Auth.Scope)
	}
}
