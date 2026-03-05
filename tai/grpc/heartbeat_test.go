package grpc

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yaoapp/yao/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestCountUserProcesses(t *testing.T) {
	n := countUserProcesses()
	if n < 0 {
		t.Errorf("countUserProcesses() = %d, want >= 0", n)
	}
}

func TestSampleResources(t *testing.T) {
	cpu, mem := sampleResources()
	if cpu < 0 || mem < 0 {
		t.Errorf("sampleResources() = (%d, %d), want non-negative", cpu, mem)
	}
}

// ── HeartbeatLoop tests with mock gRPC server ───────────────────────────────

type mockYaoServer struct {
	pb.UnimplementedYaoServer
	calls  atomic.Int32
	action string
}

func (m *mockYaoServer) Heartbeat(_ context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	m.calls.Add(1)
	return &pb.HeartbeatResponse{Action: m.action}, nil
}

func startMockServer(t *testing.T, srv *mockYaoServer) (addr string, stop func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterYaoServer(s, srv)
	go s.Serve(lis)
	return lis.Addr().String(), s.Stop
}

func dialClient(t *testing.T, addr string) *Client {
	t.Helper()
	c, err := Dial(addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHeartbeatLoop_SendsHeartbeats(t *testing.T) {
	mock := &mockYaoServer{action: "ok"}
	addr, stop := startMockServer(t, mock)
	defer stop()

	client := dialClient(t, addr)
	defer client.Close()

	t.Setenv("YAO_HEARTBEAT_INTERVAL", "50ms")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	HeartbeatLoop(ctx, client, "sb-test")

	calls := mock.calls.Load()
	if calls < 2 {
		t.Errorf("expected at least 2 heartbeat calls, got %d", calls)
	}
}

func TestHeartbeatLoop_ShutdownAction(t *testing.T) {
	mock := &mockYaoServer{action: "shutdown"}
	addr, stop := startMockServer(t, mock)
	defer stop()

	client := dialClient(t, addr)
	defer client.Close()

	action, err := client.Heartbeat(context.Background(), "sb-shutdown", 0, 0, 0)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if action != "shutdown" {
		t.Errorf("action = %q, want %q", action, "shutdown")
	}
	if mock.calls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls.Load())
	}
}

func TestHeartbeatLoop_ContextCancelStops(t *testing.T) {
	mock := &mockYaoServer{action: "ok"}
	addr, stop := startMockServer(t, mock)
	defer stop()

	client := dialClient(t, addr)
	defer client.Close()

	t.Setenv("YAO_HEARTBEAT_INTERVAL", "5s")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		HeartbeatLoop(ctx, client, "sb-cancel")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("HeartbeatLoop did not stop after context cancel")
	}
}

func TestHeartbeatLoop_IntervalParsing(t *testing.T) {
	mock := &mockYaoServer{action: "ok"}
	addr, stop := startMockServer(t, mock)
	defer stop()

	client := dialClient(t, addr)
	defer client.Close()

	t.Setenv("YAO_HEARTBEAT_INTERVAL", "30ms")

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	HeartbeatLoop(ctx, client, "sb-interval")

	calls := mock.calls.Load()
	if calls < 3 {
		t.Errorf("with 30ms interval over 150ms, expected >= 3 calls, got %d", calls)
	}
}

func TestHeartbeatLoop_InvalidIntervalUsesDefault(t *testing.T) {
	mock := &mockYaoServer{action: "ok"}
	addr, stop := startMockServer(t, mock)
	defer stop()

	client := dialClient(t, addr)
	defer client.Close()

	t.Setenv("YAO_HEARTBEAT_INTERVAL", "not-a-duration")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	HeartbeatLoop(ctx, client, "sb-invalid")

	if mock.calls.Load() > 0 {
		t.Error("with default 10s interval and 100ms timeout, expected 0 calls")
	}
}

func TestClientHeartbeat_ReturnsAction(t *testing.T) {
	mock := &mockYaoServer{action: "ok"}
	addr, stop := startMockServer(t, mock)
	defer stop()

	conn, err := grpc.NewClient("passthrough:///"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	c := &Client{conn: conn, svc: pb.NewYaoClient(conn)}
	action, err := c.Heartbeat(context.Background(), "sb-1", 50, 2048, 5)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if action != "ok" {
		t.Errorf("action = %q, want %q", action, "ok")
	}
}
