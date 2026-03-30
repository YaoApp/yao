package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/yaoapp/yao/grpc/auth"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/tunnel/taipb"
)

const bufSize = 1024 * 1024

func startTestServer(t *testing.T) (taipb.TaiTunnelClient, *TunnelHandler, func()) {
	t.Helper()
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	taipb.RegisterTaiTunnelServer(srv, h)
	go srv.Serve(lis)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := taipb.NewTaiTunnelClient(conn)
	cleanup := func() {
		conn.Close()
		srv.Stop()
		lis.Close()
	}
	return client, h, cleanup
}

func TestRegister_HappyPath(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx := context.Background()
	stream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = stream.Send(&taipb.TunnelControl{
		Type:        "register",
		NodeId:      "test-node",
		MachineId:   "machine-001",
		Version:     "1.0.0",
		DisplayName: "Test Node",
		Ports:       &taipb.Ports{Grpc: 19100, Http: 8099, Vnc: 16080},
		Caps:        &taipb.Capabilities{Docker: true, HostExec: true},
		System:      &taipb.SystemInfo{Os: "linux", Arch: "amd64", Hostname: "test-host"},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Type != "registered" {
		t.Fatalf("expected type=registered, got %q", resp.Type)
	}
	if resp.TaiId == "" {
		t.Fatal("expected non-empty tai_id")
	}

	taiID := resp.TaiId
	node, ok := h.reg.Get(taiID)
	if !ok {
		t.Fatal("node not found in registry")
	}
	if node.Status != "online" {
		t.Errorf("expected status=online, got %q", node.Status)
	}
	if node.Mode != "tunnel" {
		t.Errorf("expected mode=tunnel, got %q", node.Mode)
	}
	if !node.Capabilities.Docker {
		t.Error("expected docker capability")
	}
	if !node.Capabilities.HostExec {
		t.Error("expected host_exec capability")
	}
	if node.Ports.GRPC != 19100 {
		t.Errorf("expected grpc port 19100, got %d", node.Ports.GRPC)
	}

	stream.CloseSend()
}

func TestRegister_MissingNodeID(t *testing.T) {
	client, _, cleanup := startTestServer(t)
	defer cleanup()

	stream, err := client.Register(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Send(&taipb.TunnelControl{
		Type:      "register",
		MachineId: "machine-001",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error for missing node_id")
	}
}

func TestRegister_WrongType(t *testing.T) {
	client, _, cleanup := startTestServer(t)
	defer cleanup()

	stream, err := client.Register(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Send(&taipb.TunnelControl{
		Type:      "ping",
		NodeId:    "test-node",
		MachineId: "machine-001",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error for wrong message type")
	}
}

func TestRegister_Ping(t *testing.T) {
	client, _, cleanup := startTestServer(t)
	defer cleanup()

	stream, err := client.Register(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Send(&taipb.TunnelControl{
		Type:      "register",
		NodeId:    "ping-node",
		MachineId: "machine-ping",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Type != "registered" {
		t.Fatalf("expected registered, got %q", resp.Type)
	}

	err = stream.Send(&taipb.TunnelControl{Type: "ping"})
	if err != nil {
		t.Fatal(err)
	}

	// Loop until we receive "pong"; skip "open" frames that may arrive from
	// the asynchronous connectTunnelNode goroutine if a real gRPC endpoint
	// happens to be reachable in the test environment.
	for {
		pong, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if pong.Type == "pong" {
			break
		}
		// skip unexpected frames (e.g. "open" from connectTunnelNode)
	}

	stream.CloseSend()
}

func TestForward_MissingMetadata(t *testing.T) {
	client, _, cleanup := startTestServer(t)
	defer cleanup()

	stream, err := client.Forward(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Server may close the stream before or after Send completes (race).
	// Either Send or Recv returning an error confirms the server rejected.
	sendErr := stream.Send(&taipb.ForwardData{Data: []byte("hello")})
	if sendErr != nil {
		return // server already closed stream — pass
	}

	_, recvErr := stream.Recv()
	if recvErr == nil {
		t.Fatal("expected error for missing channel_id metadata")
	}
}

func TestForward_NoPendingChannel(t *testing.T) {
	client, _, cleanup := startTestServer(t)
	defer cleanup()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "channel_id", "nonexistent-id")
	stream, err := client.Forward(ctx)
	if err != nil {
		t.Fatal(err)
	}

	sendErr := stream.Send(&taipb.ForwardData{Data: []byte("hello")})
	if sendErr != nil {
		return // server already closed stream — pass
	}

	_, recvErr := stream.Recv()
	if recvErr == nil {
		t.Fatal("expected error for non-existent channel_id")
	}
}

func TestRequestForward_NoRegisterStream(t *testing.T) {
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)

	reg.Register(&registry.TaiNode{TaiID: "no-stream", Mode: "tunnel"})

	_, err := h.requestForwardRaw("no-stream", 8099)
	if err == nil {
		t.Fatal("expected error when no register stream")
	}
}

func TestRequestForward_TypeMismatch(t *testing.T) {
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)

	reg.Register(&registry.TaiNode{TaiID: "bad-type", Mode: "tunnel"})
	reg.SetRegisterStream("bad-type", "not-a-stream")

	_, err := h.requestForwardRaw("bad-type", 8099)
	if err == nil {
		t.Fatal("expected error for type mismatch")
	}
}

// TestRegisterAndForward_FullRoundTrip simulates Tai's full lifecycle:
// 1. Tai opens Register stream and sends "register"
// 2. Yao responds with "registered"
// 3. Yao calls RequestForward which sends "open" via the Register stream
// 4. Tai opens a Forward stream with the matching channel_id
// 5. Yao's RequestForward returns the matched Forward stream
//
// connectTunnelNode (which calls DialTunnel) runs in the background but
// we race ahead to drive the matching manually; the DialTunnel will
// harmlessly fail or succeed without affecting the core matching test.
func TestRegisterAndForward_FullRoundTrip(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	regStream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = regStream.Send(&taipb.TunnelControl{
		Type:      "register",
		NodeId:    "fwd-node",
		MachineId: "fwd-machine",
		Ports:     &taipb.Ports{Http: 8099},
	})
	if err != nil {
		t.Fatal(err)
	}

	registered, err := regStream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if registered.Type != "registered" {
		t.Fatalf("expected registered, got %q", registered.Type)
	}
	taiID := registered.TaiId

	// The server's Register handler now runs the control-loop goroutine.
	// connectTunnelNode also fires in background (will fail in test — no real Tai gRPC).
	// We'll consume all "open" commands from the stream by acting as Tai.
	// First, launch our own RequestForward call that sends a fresh "open".
	// We need to drain any prior "open" commands from connectTunnelNode first.

	// Goroutine: consume messages from register stream, respond to "open" commands.
	type openInfo struct {
		channelID  string
		targetPort int32
	}
	openCh := make(chan openInfo, 10)
	go func() {
		for {
			msg, err := regStream.Recv()
			if err != nil {
				return
			}
			if msg.Type == "open" {
				openCh <- openInfo{channelID: msg.ChannelId, targetPort: msg.TargetPort}
			}
		}
	}()

	// Wait a bit for connectTunnelNode to try (and likely fail)
	time.Sleep(300 * time.Millisecond)

	// Drain any "open" commands from connectTunnelNode
drainLoop:
	for {
		select {
		case <-openCh:
		default:
			break drainLoop
		}
	}

	// Now call RequestForward ourselves — this sends a new "open" on the register stream.
	var requestErr error
	var requestResult taipb.TaiTunnel_ForwardServer
	var requestDone sync.WaitGroup
	requestDone.Add(1)
	go func() {
		defer requestDone.Done()
		requestResult, requestErr = h.requestForwardRaw(taiID, 8099)
	}()

	// Receive the "open" command
	var oi openInfo
	select {
	case oi = <-openCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for open command")
	}
	if oi.targetPort != 8099 {
		t.Errorf("expected target_port=8099, got %d", oi.targetPort)
	}
	if oi.channelID == "" {
		t.Fatal("expected non-empty channel_id")
	}

	// Tai opens a Forward stream with the matching channel_id
	fwdCtx := metadata.AppendToOutgoingContext(ctx, "channel_id", oi.channelID)
	fwdStream, err := client.Forward(fwdCtx)
	if err != nil {
		t.Fatal(err)
	}

	// Forward handler needs a first message to trigger stream delivery
	err = fwdStream.Send(&taipb.ForwardData{Data: []byte("hello from tai")})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for RequestForward to return
	requestDone.Wait()
	if requestErr != nil {
		t.Fatal("RequestForward failed:", requestErr)
	}
	if requestResult == nil {
		t.Fatal("expected non-nil forward stream from RequestForward")
	}

	regStream.CloseSend()
	fwdStream.CloseSend()
}

func TestRegister_Unregister_OnStreamClose(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx := context.Background()
	stream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = stream.Send(&taipb.TunnelControl{
		Type:      "register",
		NodeId:    "unreg-node",
		MachineId: "unreg-machine",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	taiID := resp.TaiId

	_, ok := h.reg.Get(taiID)
	if !ok {
		t.Fatal("node should exist after register")
	}

	stream.CloseSend()
	time.Sleep(200 * time.Millisecond)

	_, ok = h.reg.Get(taiID)
	if ok {
		t.Error("node should be unregistered after stream close")
	}
}

func TestNewTunnelHandler_SetsBridgeFunc(t *testing.T) {
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)
	if h.reg != reg {
		t.Error("expected handler to reference the same registry")
	}
	if GlobalHandler() != h {
		t.Error("expected global handler to be set")
	}
}

func TestBridgeConn_NoRegisterStream(t *testing.T) {
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)
	reg.Register(&registry.TaiNode{TaiID: "bridge-fail", Mode: "tunnel"})

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	h.bridgeConn("bridge-fail", 8099, serverConn)

	buf := make([]byte, 1)
	_, err := clientConn.Read(buf)
	if err == nil {
		t.Error("expected read error (conn should be closed by bridgeConn)")
	}
}

// ── forwardConn tests ──────────────────────────────────────────────────────

type mockForwardStream struct {
	taipb.TaiTunnel_ForwardServer
	recvData [][]byte
	recvIdx  int
	sent     [][]byte
	mu       sync.Mutex
}

func (m *mockForwardStream) Recv() (*taipb.ForwardData, error) {
	if m.recvIdx >= len(m.recvData) {
		return nil, io.EOF
	}
	data := m.recvData[m.recvIdx]
	m.recvIdx++
	return &taipb.ForwardData{Data: data}, nil
}

func (m *mockForwardStream) Send(msg *taipb.ForwardData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(msg.Data))
	copy(cp, msg.Data)
	m.sent = append(m.sent, cp)
	return nil
}

func TestForwardConn_Write(t *testing.T) {
	mock := &mockForwardStream{}
	fc := newForwardConn(mock)

	n, err := fc.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("expected write 5 bytes, got %d", n)
	}
	if len(mock.sent) != 1 || string(mock.sent[0]) != "hello" {
		t.Errorf("unexpected sent data: %v", mock.sent)
	}
}

func TestForwardConn_Read(t *testing.T) {
	mock := &mockForwardStream{
		recvData: [][]byte{[]byte("world")},
	}
	fc := newForwardConn(mock)

	buf := make([]byte, 10)
	n, err := fc.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "world" {
		t.Errorf("expected 'world', got %q", buf[:n])
	}
}

func TestForwardConn_Read_Buffered(t *testing.T) {
	mock := &mockForwardStream{
		recvData: [][]byte{[]byte("abcdefghij")},
	}
	fc := newForwardConn(mock)

	buf := make([]byte, 4)
	n, err := fc.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 || string(buf[:n]) != "abcd" {
		t.Errorf("first read: got %q", buf[:n])
	}

	n, err = fc.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 || string(buf[:n]) != "efgh" {
		t.Errorf("second read: got %q", buf[:n])
	}

	n, err = fc.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || string(buf[:n]) != "ij" {
		t.Errorf("third read: got %q", buf[:n])
	}
}

func TestForwardConn_Read_EOF(t *testing.T) {
	mock := &mockForwardStream{recvData: nil}
	fc := newForwardConn(mock)

	buf := make([]byte, 10)
	_, err := fc.Read(buf)
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestForwardConn_Close(t *testing.T) {
	fc := newForwardConn(&mockForwardStream{})
	if err := fc.Close(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// ── bridgeTCP tests ──────────────────────────────────────────────────────

func TestBridgeTCP(t *testing.T) {
	a := &rwcBuffer{Reader: bytes.NewReader([]byte("from-a")), Writer: &bytes.Buffer{}}
	b := &rwcBuffer{Reader: bytes.NewReader([]byte("from-b")), Writer: &bytes.Buffer{}}

	bridgeTCP(a, b)

	if got := a.Writer.(*bytes.Buffer).String(); got != "from-b" {
		t.Errorf("a received %q, want 'from-b'", got)
	}
	if got := b.Writer.(*bytes.Buffer).String(); got != "from-a" {
		t.Errorf("b received %q, want 'from-a'", got)
	}
}

type rwcBuffer struct {
	io.Reader
	io.Writer
	closed bool
}

func (r *rwcBuffer) Close() error {
	r.closed = true
	return nil
}

func TestBridgeTCP_OneSideClosed(t *testing.T) {
	a := &rwcBuffer{Reader: bytes.NewReader(nil), Writer: &bytes.Buffer{}}
	b := &rwcBuffer{Reader: bytes.NewReader([]byte("only-b")), Writer: &bytes.Buffer{}}

	bridgeTCP(a, b)

	if got := a.Writer.(*bytes.Buffer).String(); got != "only-b" {
		t.Errorf("a received %q, want 'only-b'", got)
	}
	if !a.closed || !b.closed {
		t.Error("both sides should be closed")
	}
}

// ── forwardConn error path tests ────────────────────────────────────────

type errorForwardStream struct {
	taipb.TaiTunnel_ForwardServer
}

func (e *errorForwardStream) Send(_ *taipb.ForwardData) error {
	return fmt.Errorf("send failed")
}

func (e *errorForwardStream) Recv() (*taipb.ForwardData, error) {
	return nil, fmt.Errorf("recv failed")
}

func TestForwardConn_Write_Error(t *testing.T) {
	fc := newForwardConn(&errorForwardStream{})
	_, err := fc.Write([]byte("data"))
	if err == nil {
		t.Fatal("expected error from Write")
	}
}

func TestForwardConn_Read_Error(t *testing.T) {
	fc := newForwardConn(&errorForwardStream{})
	buf := make([]byte, 10)
	_, err := fc.Read(buf)
	if err == nil {
		t.Fatal("expected error from Read")
	}
}

// ── authInfoFromStream with auth context ────────────────────────────────

func startTestServerWithAuth(t *testing.T) (taipb.TaiTunnelClient, *TunnelHandler, func()) {
	t.Helper()
	reg := registry.NewForTest()
	h := NewTunnelHandler(reg)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(
		grpc.StreamInterceptor(func(
			srvObj interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
		) error {
			ctx := auth.WithAuthorizedInfo(ss.Context(), &oauthtypes.AuthorizedInfo{
				Subject:  "user:123",
				UserID:   "u-123",
				ClientID: "client-abc",
				Scope:    "workspace:read",
				TeamID:   "team-1",
				TenantID: "tenant-1",
			})
			return handler(srvObj, &wrappedStreamCtx{ServerStream: ss, ctx: ctx})
		}),
	)
	taipb.RegisterTaiTunnelServer(srv, h)
	go srv.Serve(lis)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := taipb.NewTaiTunnelClient(conn)
	cleanup := func() {
		conn.Close()
		srv.Stop()
		lis.Close()
	}
	return client, h, cleanup
}

type wrappedStreamCtx struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStreamCtx) Context() context.Context { return w.ctx }

// ── RequestForward timeout ──────────────────────────────────────────────

func TestRequestForward_Timeout(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx := context.Background()
	stream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Send(&taipb.TunnelControl{
		Type: "register", NodeId: "timeout-node", MachineId: "timeout-machine",
		Ports: &taipb.Ports{Http: 8099},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	taiID := resp.TaiId

	// Drain any "open" from connectTunnelNode
	go func() {
		for {
			if _, err := stream.Recv(); err != nil {
				return
			}
		}
	}()
	time.Sleep(300 * time.Millisecond)

	// Override the timeout: patch pending with a short timeout by calling RequestForward
	// but never sending a Forward stream back. The default is 10s which is too long
	// for a unit test. We test the mechanism by directly checking pending cleanup.
	// To avoid waiting 10s we'll test the pending cleanup via a smaller helper:
	channelID := "timeout-test-channel"
	waitCh := make(chan taipb.TaiTunnel_ForwardServer, 1)
	h.pending.Store(channelID, waitCh)

	// Verify pending is stored
	if _, ok := h.pending.Load(channelID); !ok {
		t.Fatal("expected pending channel to be stored")
	}

	// Simulate timeout cleanup (what RequestForward's defer does)
	h.pending.Delete(channelID)
	if _, ok := h.pending.Load(channelID); ok {
		t.Fatal("pending should be cleaned up after delete")
	}

	// Now test actual RequestForward timeout behavior (with the real 10s timeout
	// by never sending Forward). We'll use a short context cancel to avoid waiting.
	done := make(chan error, 1)
	go func() {
		_, err := h.requestForwardRaw(taiID, 8099)
		done <- err
	}()

	// Cancel the register stream to trigger the regStream.Context().Done() branch
	stream.CloseSend()
	time.Sleep(200 * time.Millisecond)

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error from RequestForward")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RequestForward should have returned after stream close")
	}
}

// ── Concurrent Forward streams ──────────────────────────────────────────

func TestConcurrentForward(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	regStream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = regStream.Send(&taipb.TunnelControl{
		Type: "register", NodeId: "concurrent-node", MachineId: "concurrent-machine",
		Ports: &taipb.Ports{Http: 8099},
	})
	if err != nil {
		t.Fatal(err)
	}
	registered, err := regStream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	taiID := registered.TaiId

	type openInfo struct {
		channelID  string
		targetPort int32
	}
	openCh := make(chan openInfo, 20)
	go func() {
		for {
			msg, err := regStream.Recv()
			if err != nil {
				return
			}
			if msg.Type == "open" {
				openCh <- openInfo{channelID: msg.ChannelId, targetPort: msg.TargetPort}
			}
		}
	}()

	time.Sleep(300 * time.Millisecond)
	// Drain connectTunnelNode opens
	for {
		select {
		case <-openCh:
		default:
			goto drained
		}
	}
drained:

	const N = 5
	results := make(chan error, N)
	fwdStreams := make([]taipb.TaiTunnel_ForwardClient, 0, N)
	var mu sync.Mutex

	for i := 0; i < N; i++ {
		port := 8099 + i
		go func(port int) {
			_, err := h.requestForwardRaw(taiID, port)
			results <- err
		}(port)
	}

	// Act as Tai: respond to each open
	for i := 0; i < N; i++ {
		var oi openInfo
		select {
		case oi = <-openCh:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for open command #%d", i)
		}

		fwdCtx := metadata.AppendToOutgoingContext(ctx, "channel_id", oi.channelID)
		fwd, err := client.Forward(fwdCtx)
		if err != nil {
			t.Fatal(err)
		}
		if err := fwd.Send(&taipb.ForwardData{Data: []byte(fmt.Sprintf("data-%d", i))}); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		fwdStreams = append(fwdStreams, fwd)
		mu.Unlock()
	}

	// All RequestForward should succeed
	for i := 0; i < N; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Errorf("RequestForward #%d failed: %v", i, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for RequestForward result")
		}
	}

	mu.Lock()
	for _, fwd := range fwdStreams {
		fwd.CloseSend()
	}
	mu.Unlock()
	regStream.CloseSend()
}

// ── Disconnect detection: Forward terminates when Register stream closes ──

func TestDisconnect_ForwardTerminatesOnRegisterClose(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	regStream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = regStream.Send(&taipb.TunnelControl{
		Type: "register", NodeId: "disconnect-node", MachineId: "disconnect-machine",
		Ports: &taipb.Ports{Http: 8099},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := regStream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	taiID := resp.TaiId

	openCh := make(chan string, 10)
	go func() {
		for {
			msg, err := regStream.Recv()
			if err != nil {
				return
			}
			if msg.Type == "open" {
				openCh <- msg.ChannelId
			}
		}
	}()
	time.Sleep(300 * time.Millisecond)
	for {
		select {
		case <-openCh:
		default:
			goto drained2
		}
	}
drained2:

	// Start RequestForward
	fwdResult := make(chan error, 1)
	go func() {
		_, err := h.requestForwardRaw(taiID, 8099)
		fwdResult <- err
	}()

	// Receive the open
	var channelID string
	select {
	case channelID = <-openCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for open command")
	}

	// Open Forward stream
	fwdCtx := metadata.AppendToOutgoingContext(ctx, "channel_id", channelID)
	fwdStream, err := client.Forward(fwdCtx)
	if err != nil {
		t.Fatal(err)
	}
	_ = fwdStream.Send(&taipb.ForwardData{Data: []byte("hello")})

	// Wait for RequestForward to return
	select {
	case err := <-fwdResult:
		if err != nil {
			t.Fatal("RequestForward failed:", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for RequestForward")
	}

	// Close register stream — simulating Tai disconnect
	regStream.CloseSend()
	time.Sleep(300 * time.Millisecond)

	// Node should be unregistered
	_, ok := h.reg.Get(taiID)
	if ok {
		t.Error("node should be unregistered after register stream close")
	}

	// Forward stream should also end (context canceled)
	_, err = fwdStream.Recv()
	if err == nil {
		// It's possible the stream has remaining buffered data; try again
		_, err = fwdStream.Recv()
	}
	// We expect an error (EOF or canceled) since the server side closed
	if err == nil {
		t.Error("expected Forward stream to terminate after Register stream close")
	}
}

// ── Full HTTP proxy end-to-end test ─────────────────────────────────────

func TestHTTPProxy_EndToEnd(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Register a tunnel node
	regStream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = regStream.Send(&taipb.TunnelControl{
		Type: "register", NodeId: "proxy-node", MachineId: "proxy-machine",
		Ports: &taipb.Ports{Http: 8099},
	})
	if err != nil {
		t.Fatal(err)
	}
	registered, err := regStream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	taiID := registered.TaiId

	openCh := make(chan struct {
		channelID string
		port      int32
	}, 10)
	go func() {
		for {
			msg, err := regStream.Recv()
			if err != nil {
				return
			}
			if msg.Type == "open" {
				openCh <- struct {
					channelID string
					port      int32
				}{msg.ChannelId, msg.TargetPort}
			}
		}
	}()
	time.Sleep(300 * time.Millisecond)
	for {
		select {
		case <-openCh:
		default:
			goto proxyDrained
		}
	}
proxyDrained:

	// Start a mock Tai HTTP server
	taiHTTP, lisErr := net.Listen("tcp", "127.0.0.1:0")
	if lisErr != nil {
		t.Fatal(lisErr)
	}
	defer taiHTTP.Close()
	go func() {
		for {
			conn, err := taiHTTP.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				n, _ := c.Read(buf)
				_ = n
				response := "HTTP/1.1 200 OK\r\nContent-Length: 13\r\n\r\nHello Tunnel!"
				c.Write([]byte(response))
			}(conn)
		}
	}()

	// Simulate Tai: listen for open and connect local forward
	go func() {
		for oi := range openCh {
			go func(chID string, port int32) {
				fwdCtx := metadata.AppendToOutgoingContext(ctx, "channel_id", chID)
				fwd, err := client.Forward(fwdCtx)
				if err != nil {
					return
				}

				local, err := net.Dial("tcp", taiHTTP.Addr().String())
				if err != nil {
					return
				}
				defer local.Close()

				// Bridge: Forward stream ↔ local TCP
				done := make(chan struct{}, 2)
				go func() {
					defer func() { done <- struct{}{} }()
					for {
						data, err := fwd.Recv()
						if err != nil {
							return
						}
						local.Write(data.Data)
					}
				}()
				go func() {
					defer func() { done <- struct{}{} }()
					buf := make([]byte, 32*1024)
					for {
						n, err := local.Read(buf)
						if err != nil {
							return
						}
						fwd.Send(&taipb.ForwardData{Data: buf[:n]})
					}
				}()
				<-done
			}(oi.channelID, oi.port)
		}
	}()

	// Now do an actual RequestForward + simulate browser side
	fwd, err := h.requestForwardRaw(taiID, 8099)
	if err != nil {
		t.Fatal("RequestForward:", err)
	}

	// Send HTTP request through the tunnel
	httpReq := "GET /api/test HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if err := fwd.Send(&taipb.ForwardData{Data: []byte(httpReq)}); err != nil {
		t.Fatal("send request:", err)
	}

	// Read response
	var responseBuf bytes.Buffer
	for {
		data, err := fwd.Recv()
		if err != nil {
			break
		}
		responseBuf.Write(data.Data)
		if bytes.Contains(responseBuf.Bytes(), []byte("Hello Tunnel!")) {
			break
		}
	}

	response := responseBuf.String()
	if !bytes.Contains([]byte(response), []byte("200 OK")) {
		t.Errorf("expected 200 OK in response, got: %s", response)
	}
	if !bytes.Contains([]byte(response), []byte("Hello Tunnel!")) {
		t.Errorf("expected 'Hello Tunnel!' in response body, got: %s", response)
	}

	regStream.CloseSend()
}

// ── VNC-like WebSocket upgrade through tunnel ───────────────────────────

func TestVNCProxy_WSUpgrade(t *testing.T) {
	client, h, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	regStream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = regStream.Send(&taipb.TunnelControl{
		Type: "register", NodeId: "vnc-node", MachineId: "vnc-machine",
		Ports: &taipb.Ports{Vnc: 16080},
	})
	if err != nil {
		t.Fatal(err)
	}
	registered, err := regStream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	taiID := registered.TaiId

	openCh := make(chan struct {
		channelID string
		port      int32
	}, 10)
	go func() {
		for {
			msg, err := regStream.Recv()
			if err != nil {
				return
			}
			if msg.Type == "open" {
				openCh <- struct {
					channelID string
					port      int32
				}{msg.ChannelId, msg.TargetPort}
			}
		}
	}()
	time.Sleep(300 * time.Millisecond)
	for {
		select {
		case <-openCh:
		default:
			goto vncDrained
		}
	}
vncDrained:

	// Mock VNC server (responds to WS upgrade with 101 + echo)
	vncListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer vncListener.Close()
	go func() {
		for {
			conn, err := vncListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				n, _ := c.Read(buf)
				request := string(buf[:n])
				if bytes.Contains([]byte(request), []byte("Upgrade: websocket")) {
					wsResp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"
					c.Write([]byte(wsResp))
					// Echo back any data (simulating VNC binary frames)
					for {
						n, err := c.Read(buf)
						if err != nil {
							return
						}
						c.Write(buf[:n])
					}
				}
			}(conn)
		}
	}()

	// Act as Tai: respond to open by bridging to mock VNC
	go func() {
		for oi := range openCh {
			go func(chID string) {
				fwdCtx := metadata.AppendToOutgoingContext(ctx, "channel_id", chID)
				fwd, err := client.Forward(fwdCtx)
				if err != nil {
					return
				}

				local, err := net.Dial("tcp", vncListener.Addr().String())
				if err != nil {
					return
				}
				defer local.Close()

				done := make(chan struct{}, 2)
				go func() {
					defer func() { done <- struct{}{} }()
					for {
						data, err := fwd.Recv()
						if err != nil {
							return
						}
						local.Write(data.Data)
					}
				}()
				go func() {
					defer func() { done <- struct{}{} }()
					buf := make([]byte, 32*1024)
					for {
						n, err := local.Read(buf)
						if err != nil {
							return
						}
						fwd.Send(&taipb.ForwardData{Data: buf[:n]})
					}
				}()
				<-done
			}(oi.channelID)
		}
	}()

	// Send WS upgrade request through tunnel
	fwd, err := h.requestForwardRaw(taiID, 16080)
	if err != nil {
		t.Fatal("RequestForward:", err)
	}

	wsUpgrade := "GET /vnc/__host__/ws HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"
	if err := fwd.Send(&taipb.ForwardData{Data: []byte(wsUpgrade)}); err != nil {
		t.Fatal("send WS upgrade:", err)
	}

	// Read response
	var responseBuf bytes.Buffer
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout reading WS upgrade response, got so far: %s", responseBuf.String())
		default:
		}
		data, err := fwd.Recv()
		if err != nil {
			break
		}
		responseBuf.Write(data.Data)
		if bytes.Contains(responseBuf.Bytes(), []byte("101 Switching Protocols")) {
			break
		}
	}

	response := responseBuf.String()
	if !bytes.Contains([]byte(response), []byte("101 Switching Protocols")) {
		t.Fatalf("expected 101 Switching Protocols, got: %s", response)
	}

	// Send binary data (simulating VNC frame) and verify echo
	testFrame := []byte{0x00, 0x01, 0x02, 0x03, 0xAA, 0xBB}
	if err := fwd.Send(&taipb.ForwardData{Data: testFrame}); err != nil {
		t.Fatal("send VNC frame:", err)
	}

	echoData, err := fwd.Recv()
	if err != nil {
		t.Fatal("recv echo:", err)
	}
	if !bytes.Equal(echoData.Data, testFrame) {
		t.Errorf("expected echo %v, got %v", testFrame, echoData.Data)
	}

	regStream.CloseSend()
}

func TestRegister_WithAuthInfo(t *testing.T) {
	client, h, cleanup := startTestServerWithAuth(t)
	defer cleanup()

	stream, err := client.Register(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Send(&taipb.TunnelControl{
		Type:      "register",
		NodeId:    "auth-node",
		MachineId: "auth-machine",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	taiID := resp.TaiId

	node, ok := h.reg.Get(taiID)
	if !ok {
		t.Fatal("node not found")
	}
	if node.Auth.UserID != "u-123" {
		t.Errorf("expected user_id=u-123, got %q", node.Auth.UserID)
	}
	if node.Auth.ClientID != "client-abc" {
		t.Errorf("expected client_id=client-abc, got %q", node.Auth.ClientID)
	}
	if node.Auth.TeamID != "team-1" {
		t.Errorf("expected team_id=team-1, got %q", node.Auth.TeamID)
	}
	if node.Auth.Scope != "workspace:read" {
		t.Errorf("expected scope=workspace:read, got %q", node.Auth.Scope)
	}

	stream.CloseSend()
}
