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

func TestListByTeam_IncludesPublic(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-cloud", Mode: "cloud"})
	r.Register(&TaiNode{TaiID: "tai-tunnel", Mode: "tunnel", Auth: types.AuthInfo{TeamID: "team-a"}})

	teamA := r.ListByTeam("team-a")
	if len(teamA) != 2 {
		t.Errorf("ListByTeam(team-a) = %d nodes, want 2", len(teamA))
	}

	ids := map[string]bool{}
	for _, snap := range teamA {
		ids[snap.TaiID] = true
	}
	for _, id := range []string{"tai-cloud", "tai-tunnel"} {
		if !ids[id] {
			t.Errorf("missing node %q in ListByTeam(team-a)", id)
		}
	}

	teamB := r.ListByTeam("team-b")
	if len(teamB) != 1 {
		t.Errorf("ListByTeam(team-b) = %d nodes, want 1", len(teamB))
	}
	if teamB[0].TaiID != "tai-cloud" {
		t.Errorf("ListByTeam(team-b) node = %q, want tai-cloud", teamB[0].TaiID)
	}
}

func TestListByUser_IncludesPublic(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-cloud", Mode: "cloud"})
	r.Register(&TaiNode{TaiID: "tai-tunnel", Mode: "tunnel", Auth: types.AuthInfo{UserID: "user-1"}})

	user1 := r.ListByUser("user-1")
	if len(user1) != 2 {
		t.Errorf("ListByUser(user-1) = %d nodes, want 2", len(user1))
	}

	ids := map[string]bool{}
	for _, snap := range user1 {
		ids[snap.TaiID] = true
	}
	for _, id := range []string{"tai-cloud", "tai-tunnel"} {
		if !ids[id] {
			t.Errorf("missing node %q in ListByUser(user-1)", id)
		}
	}

	user2 := r.ListByUser("user-2")
	if len(user2) != 1 {
		t.Errorf("ListByUser(user-2) = %d nodes, want 1", len(user2))
	}
	if user2[0].TaiID != "tai-cloud" {
		t.Errorf("ListByUser(user-2) node = %q, want tai-cloud", user2[0].TaiID)
	}
}

func TestSetResources_FiresOnResourcesReady(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001", Mode: "tunnel"})

	called := make(chan string, 1)
	r.SetOnResourcesReady(func(taiID string, resources any) {
		called <- taiID
	})

	r.SetResources("tai-001", "fake-resources")

	select {
	case id := <-called:
		if id != "tai-001" {
			t.Errorf("callback taiID = %q, want tai-001", id)
		}
	case <-time.After(time.Second):
		t.Fatal("OnResourcesReady callback was not called within 1s")
	}
}

func TestSetResources_NoCallbackForMissingNode(t *testing.T) {
	r := newTestRegistry()

	called := make(chan string, 1)
	r.SetOnResourcesReady(func(taiID string, resources any) {
		called <- taiID
	})

	r.SetResources("ghost", "fake-resources")

	select {
	case id := <-called:
		t.Fatalf("callback should not fire for missing node, got taiID=%q", id)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSetResources_NoCallbackWhenNotSet(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001", Mode: "tunnel"})
	r.SetResources("tai-001", "fake-resources")

	res, ok := r.GetResources("tai-001")
	if !ok {
		t.Fatal("expected resources to be stored")
	}
	if res != "fake-resources" {
		t.Errorf("resources = %v, want fake-resources", res)
	}
}
