package registry

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newTestRegistry creates a standalone registry for testing (bypasses global singleton).
func newTestRegistry() *Registry {
	return &Registry{
		nodes:   make(map[string]*TaiNode),
		pending: make(map[string]*pendingChannel),
		logger:  slog.Default(),
	}
}

func TestRegister_SetsFieldsAndOnline(t *testing.T) {
	r := newTestRegistry()
	node := &TaiNode{
		TaiID:     "tai-001",
		MachineID: "m-abc",
		Version:   "1.0.0",
		Mode:      "tunnel",
		Ports:     map[string]int{"grpc": 19100},
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
		Ports: map[string]int{"grpc": 19100, "http": 8099},
	})

	snap, _ := r.Get("tai-001")
	snap.Ports["grpc"] = 0

	snap2, _ := r.Get("tai-001")
	if snap2.Ports["grpc"] != 19100 {
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

func TestWriteControlJSON_NoNode(t *testing.T) {
	r := newTestRegistry()
	err := r.WriteControlJSON("missing", map[string]string{"type": "test"})
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestWriteControlJSON_NilConn(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001"})
	err := r.WriteControlJSON("tai-001", map[string]string{"type": "test"})
	if err == nil {
		t.Fatal("expected error for nil ControlConn")
	}
}

func TestRequestChannel_NotFound(t *testing.T) {
	r := newTestRegistry()
	_, _, err := r.RequestChannel("ghost", 19100)
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestRequestChannel_DirectMode(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001", Mode: "direct"})
	_, _, err := r.RequestChannel("tai-001", 19100)
	if err == nil {
		t.Fatal("expected error for direct-mode node")
	}
}

func TestAcceptDataChannel_NotPending(t *testing.T) {
	r := newTestRegistry()
	pipe1, pipe2 := net.Pipe()
	defer pipe1.Close()
	defer pipe2.Close()

	err := r.AcceptDataChannel("unknown-channel", "tai-001", pipe1)
	if err == nil {
		t.Fatal("expected error for non-pending channel")
	}
}

func TestAcceptDataChannel_TaiIDMismatch(t *testing.T) {
	r := newTestRegistry()

	resultCh := make(chan net.Conn, 1)
	timer := time.AfterFunc(5*time.Second, func() {})
	r.mu.Lock()
	r.pending["ch-001"] = &pendingChannel{taiID: "tai-owner", result: resultCh, timer: timer}
	r.mu.Unlock()

	pipe1, pipe2 := net.Pipe()
	defer pipe1.Close()
	defer pipe2.Close()

	err := r.AcceptDataChannel("ch-001", "tai-intruder", pipe1)
	if err == nil {
		t.Fatal("expected error for tai_id mismatch")
	}
}

func TestAcceptDataChannel_Success(t *testing.T) {
	r := newTestRegistry()

	resultCh := make(chan net.Conn, 1)
	timer := time.AfterFunc(5*time.Second, func() {})
	r.mu.Lock()
	r.pending["ch-002"] = &pendingChannel{taiID: "tai-001", result: resultCh, timer: timer}
	r.mu.Unlock()

	pipe1, pipe2 := net.Pipe()
	defer pipe2.Close()

	if err := r.AcceptDataChannel("ch-002", "tai-001", pipe1); err != nil {
		t.Fatalf("AcceptDataChannel: %v", err)
	}

	select {
	case conn := <-resultCh:
		if conn == nil {
			t.Fatal("expected non-nil conn")
		}
		conn.Close()
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for conn on resultCh")
	}
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

func TestBridgeTCP(t *testing.T) {
	a1, a2 := net.Pipe()
	b1, b2 := net.Pipe()

	go bridgeTCP(a2, b1)

	msg := []byte("hello tunnel")
	go func() {
		a1.Write(msg)
		a1.Close()
	}()

	buf := make([]byte, 64)
	n, _ := b2.Read(buf)
	if string(buf[:n]) != "hello tunnel" {
		t.Errorf("got %q, want %q", buf[:n], "hello tunnel")
	}
	b2.Close()
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

func TestWriteControlJSON_Success(t *testing.T) {
	done := make(chan map[string]string, 1)

	srv := newWSServer(func(conn *websocket.Conn) {
		var msg map[string]string
		conn.ReadJSON(&msg)
		done <- msg
		conn.Close()
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001", Mode: "tunnel", ControlConn: wsConn})

	payload := map[string]string{"type": "test", "data": "hello"}
	if err := r.WriteControlJSON("tai-001", payload); err != nil {
		t.Fatalf("WriteControlJSON: %v", err)
	}

	select {
	case got := <-done:
		if got["type"] != "test" {
			t.Errorf("type = %q, want test", got["type"])
		}
		if got["data"] != "hello" {
			t.Errorf("data = %q, want hello", got["data"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive message")
	}
}

func TestRequestChannel_Success(t *testing.T) {
	openCh := make(chan map[string]interface{}, 1)

	srv := newWSServer(func(conn *websocket.Conn) {
		var msg map[string]interface{}
		conn.ReadJSON(&msg)
		openCh <- msg
		time.Sleep(time.Second)
		conn.Close()
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001", Mode: "tunnel", ControlConn: wsConn})

	channelID, resultCh, err := r.RequestChannel("tai-001", 19100)
	if err != nil {
		t.Fatalf("RequestChannel: %v", err)
	}
	if channelID == "" {
		t.Fatal("channelID should not be empty")
	}
	if len(channelID) != 64 {
		t.Errorf("channelID len = %d, want 64", len(channelID))
	}
	if resultCh == nil {
		t.Fatal("resultCh should not be nil")
	}

	select {
	case cmd := <-openCh:
		if cmd["type"] != "open" {
			t.Errorf("cmd type = %v, want open", cmd["type"])
		}
		if cmd["channel_id"] != channelID {
			t.Errorf("cmd channel_id = %v, want %s", cmd["channel_id"], channelID)
		}
		if tp, ok := cmd["target_port"].(float64); !ok || int(tp) != 19100 {
			t.Errorf("cmd target_port = %v, want 19100", cmd["target_port"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for open command")
	}
}

func TestRequestChannel_NoControlConn(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{TaiID: "tai-001", Mode: "tunnel"})

	_, _, err := r.RequestChannel("tai-001", 19100)
	if err == nil {
		t.Fatal("expected error for nil ControlConn")
	}
}

func TestOpenLocalListener_Success(t *testing.T) {
	r := newTestRegistry()

	controlCh := make(chan map[string]interface{}, 1)
	srv := newWSServer(func(conn *websocket.Conn) {
		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			controlCh <- msg
		}
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	r.Register(&TaiNode{TaiID: "tai-001", Mode: "tunnel", ControlConn: wsConn})

	ln, err := r.OpenLocalListener("tai-001", 19100)
	if err != nil {
		t.Fatalf("OpenLocalListener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	if addr == "" {
		t.Fatal("listener address should not be empty")
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("addr = %q, want 127.0.0.1:*", addr)
	}

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("connect to local listener: %v", err)
	}
	defer conn.Close()

	select {
	case cmd := <-controlCh:
		if cmd["type"] != "open" {
			t.Errorf("open cmd type = %v, want open", cmd["type"])
		}
		if _, ok := cmd["channel_id"].(string); !ok {
			t.Error("open cmd missing channel_id")
		}
		if tp, ok := cmd["target_port"].(float64); !ok || int(tp) != 19100 {
			t.Errorf("target_port = %v, want 19100", cmd["target_port"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for open command from local listener")
	}
}

func TestOpenLocalListener_NodeNotFound(t *testing.T) {
	r := newTestRegistry()
	_, err := r.OpenLocalListener("ghost", 19100)
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

func newWSServer(handler func(*websocket.Conn)) *httptest.Server {
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handler(conn)
	}))
}

func TestRegister_SystemInfo(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{
		TaiID: "tai-001",
		System: SystemInfo{
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
	r.Register(&TaiNode{TaiID: "tai-a", Auth: AuthInfo{TeamID: "team-dev"}})
	r.Register(&TaiNode{TaiID: "tai-b", Auth: AuthInfo{TeamID: "team-dev"}})
	r.Register(&TaiNode{TaiID: "tai-c", Auth: AuthInfo{TeamID: "team-ops"}})

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

func TestNodeSnapshot_AuthInfo(t *testing.T) {
	r := newTestRegistry()
	r.Register(&TaiNode{
		TaiID: "tai-001",
		Auth: AuthInfo{
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
