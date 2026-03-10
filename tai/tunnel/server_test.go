package tunnel

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yaoapp/yao/tai/registry"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRegistry() *registry.Registry {
	r := registry.NewForTest()
	registry.SetGlobalForTest(r)
	return r
}

func mockAuth(info registry.AuthInfo, authErr error) func() {
	old := authenticateBearerFunc
	authenticateBearerFunc = func(token string) (registry.AuthInfo, error) {
		return info, authErr
	}
	return func() { authenticateBearerFunc = old }
}

// --- extractBearer ---

func TestExtractBearer(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"valid", "Bearer abc123", "abc123"},
		{"lowercase", "bearer xyz", "xyz"},
		{"empty", "", ""},
		{"no_scheme", "abc123", ""},
		{"only_bearer", "Bearer ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if tt.header != "" {
				r.Header.Set("Authorization", tt.header)
			}
			got := extractBearer(r)
			if got != tt.want {
				t.Errorf("extractBearer(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

// --- wsConn ---

func TestWSConn_EchoRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		wc := newWSConn(conn)
		buf := make([]byte, 256)
		n, err := wc.Read(buf)
		if err != nil {
			return
		}
		wc.Write(buf[:n])
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("handshake status = %d, want 101", resp.StatusCode)
	}

	msg := []byte("hello tunnel")
	if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	mt, reply, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if mt != websocket.BinaryMessage {
		t.Errorf("type = %d, want BinaryMessage(%d)", mt, websocket.BinaryMessage)
	}
	if string(reply) != "hello tunnel" {
		t.Errorf("reply = %q, want %q", reply, "hello tunnel")
	}
}

func TestWSConn_MultipleMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		wc := newWSConn(conn)
		for i := 0; i < 3; i++ {
			buf := make([]byte, 256)
			n, err := wc.Read(buf)
			if err != nil {
				return
			}
			wc.Write(buf[:n])
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	for i, msg := range []string{"one", "two", "three"} {
		conn.WriteMessage(websocket.BinaryMessage, []byte(msg))
		_, reply, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("round %d read: %v", i, err)
		}
		if string(reply) != msg {
			t.Errorf("round %d: got %q, want %q", i, reply, msg)
		}
	}
}

func TestWSConn_ImplementsNetConn(t *testing.T) {
	var _ net.Conn = (*wsConn)(nil)
}

func TestWSConn_LocalRemoteAddr(t *testing.T) {
	addrCh := make(chan [2]net.Addr, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		wc := newWSConn(conn)
		addrCh <- [2]net.Addr{wc.LocalAddr(), wc.RemoteAddr()}
		wc.Close()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	select {
	case addrs := <-addrCh:
		if addrs[0] == nil {
			t.Error("LocalAddr should not be nil")
		}
		if addrs[1] == nil {
			t.Error("RemoteAddr should not be nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for addresses")
	}
}

// --- HandleControl ---

func newGinRouter() *gin.Engine {
	r := gin.New()
	r.GET("/ws/tai", HandleControl)
	r.GET("/ws/tai/data/:channel_id", HandleData)
	return r
}

func TestHandleControl_NoRegistry(t *testing.T) {
	registry.SetGlobalForTest(nil)
	defer setupTestRegistry()

	restore := mockAuth(registry.AuthInfo{ClientID: "tai-001"}, nil)
	defer restore()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer test-token"},
	})
	if err == nil {
		t.Fatal("expected dial to fail when registry is nil")
	}
	if resp != nil && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandleControl_NoAuth(t *testing.T) {
	setupTestRegistry()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial to fail without auth")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandleControl_AuthFailed(t *testing.T) {
	setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{}, fmt.Errorf("bad token"))
	defer restore()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer bad-token"},
	})
	if err == nil {
		t.Fatal("expected dial to fail with bad auth")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandleControl_RegisterAndPing(t *testing.T) {
	reg := setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{
		ClientID: "tai-001",
		Subject:  "user-test",
		Scope:    "tai:tunnel",
	}, nil)
	defer restore()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("handshake = %d, want 101", resp.StatusCode)
	}

	regMsg := registerMessage{
		Type:      "register",
		NodeID:    "9100",
		MachineID: "m-test",
		Version:   "2.0",
		Ports:     map[string]int{"grpc": 9100},
	}
	if err := conn.WriteJSON(regMsg); err != nil {
		t.Fatalf("write register: %v", err)
	}

	var registered map[string]string
	if err := conn.ReadJSON(&registered); err != nil {
		t.Fatalf("read registered: %v", err)
	}
	if registered["type"] != "registered" {
		t.Errorf("response type = %q, want registered", registered["type"])
	}
	gotTaiID := registered["tai_id"]
	if gotTaiID == "" || len(gotTaiID) < 5 || gotTaiID[:4] != "tai-" {
		t.Errorf("response tai_id = %q, want server-generated tai-xxx", gotTaiID)
	}

	snap, ok := reg.Get(gotTaiID)
	if !ok {
		t.Fatal("node not found in registry after register")
	}
	if snap.Status != "online" {
		t.Errorf("Status = %q, want online", snap.Status)
	}
	if snap.MachineID != "m-test" {
		t.Errorf("MachineID = %q, want m-test", snap.MachineID)
	}
	if snap.Version != "2.0" {
		t.Errorf("Version = %q, want 2.0", snap.Version)
	}
	if snap.Mode != "tunnel" {
		t.Errorf("Mode = %q, want tunnel", snap.Mode)
	}
	if snap.Auth.ClientID != "tai-001" {
		t.Errorf("Auth.ClientID = %q, want tai-001", snap.Auth.ClientID)
	}
	if snap.Auth.Subject != "user-test" {
		t.Errorf("Auth.Subject = %q, want user-test", snap.Auth.Subject)
	}
	if snap.Ports["grpc"] != 9100 {
		t.Errorf("Ports[grpc] = %d, want 9100", snap.Ports["grpc"])
	}

	time.Sleep(10 * time.Millisecond)
	if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	// Read messages until we get the pong; connectTunnelNode may inject
	// "open" messages (with numeric fields) before our pong arrives.
	var gotPong bool
	for i := 0; i < 10; i++ {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read message: %v", err)
		}
		if msg["type"] == "pong" {
			gotPong = true
			break
		}
	}
	if !gotPong {
		t.Error("did not receive pong after ping")
	}

	snap2, _ := reg.Get(gotTaiID)
	if !snap2.LastPing.After(snap.LastPing) {
		t.Error("LastPing should be updated after ping")
	}

	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(100 * time.Millisecond)

	if _, ok := reg.Get("tai-001"); ok {
		t.Error("node should be unregistered after connection close")
	}
}

func TestHandleControl_BadRegisterType(t *testing.T) {
	setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{ClientID: "tai-001"}, nil)
	defer restore()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	conn.WriteJSON(map[string]string{"type": "not-register"})
	_, _, readErr := conn.ReadMessage()
	if readErr == nil {
		t.Error("expected connection to close for bad register type")
	}
}

func TestHandleControl_MissingTaiID(t *testing.T) {
	setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{ClientID: "tai-001"}, nil)
	defer restore()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	conn.WriteJSON(map[string]string{"type": "register"})
	_, _, readErr := conn.ReadMessage()
	if readErr == nil {
		t.Error("expected connection to close for missing tai_id")
	}
}

// --- HandleData ---

func TestHandleData_NoAuth(t *testing.T) {
	setupTestRegistry()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai/data/ch-001"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial to fail without auth")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandleData_AcceptSuccess(t *testing.T) {
	reg := setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{ClientID: "tai-001"}, nil)
	defer restore()

	resultCh := make(chan net.Conn, 1)
	timer := time.AfterFunc(5*time.Second, func() {})
	reg.SetPendingForTest("ch-test-123", "tai-001", resultCh, timer)

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai/data/ch-test-123"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("status = %d, want 101", resp.StatusCode)
	}

	select {
	case c := <-resultCh:
		if c == nil {
			t.Fatal("expected non-nil conn from resultCh")
		}
		c.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for conn on resultCh")
	}
}

func TestHandleData_ChannelNotPending(t *testing.T) {
	setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{ClientID: "tai-001"}, nil)
	defer restore()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai/data/nonexistent"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		return
	}
	defer conn.Close()

	_, _, readErr := conn.ReadMessage()
	if readErr == nil {
		t.Error("expected connection to close for non-pending channel")
	}
}

func TestHandleData_TaiIDMismatch(t *testing.T) {
	reg := setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{ClientID: "tai-intruder"}, nil)
	defer restore()

	resultCh := make(chan net.Conn, 1)
	timer := time.AfterFunc(5*time.Second, func() {})
	reg.SetPendingForTest("ch-mismatch", "tai-owner", resultCh, timer)

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai/data/ch-mismatch"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		return
	}
	defer conn.Close()

	_, _, readErr := conn.ReadMessage()
	if readErr == nil {
		t.Error("expected connection to close for tai_id mismatch")
	}
}

// --- Full open-channel flow ---

func TestHandleControl_OpenChannelAndBridge(t *testing.T) {
	reg := setupTestRegistry()
	restore := mockAuth(registry.AuthInfo{
		ClientID: "tai-001",
		Subject:  "user-test",
	}, nil)
	defer restore()

	srv := httptest.NewServer(newGinRouter())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai"
	ctrlConn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		t.Fatalf("dial control: %v", err)
	}
	defer ctrlConn.Close()

	ctrlConn.WriteJSON(registerMessage{
		Type:      "register",
		NodeID:    "9100",
		MachineID: "m-test",
		Ports:     map[string]int{"grpc": 9100},
	})
	var registered map[string]string
	if err := ctrlConn.ReadJSON(&registered); err != nil {
		t.Fatalf("read registered: %v", err)
	}
	if registered["type"] != "registered" {
		t.Fatalf("expected registered, got %v", registered)
	}
	taiID := registered["tai_id"]

	var wg sync.WaitGroup
	wg.Add(1)
	var requestErr error
	var channelConn net.Conn
	go func() {
		defer wg.Done()
		_, resultCh, err := reg.RequestChannel(taiID, 9100)
		if err != nil {
			requestErr = err
			return
		}
		channelConn = <-resultCh
	}()

	time.Sleep(50 * time.Millisecond)

	var openCmd map[string]interface{}
	if err := ctrlConn.ReadJSON(&openCmd); err != nil {
		t.Fatalf("read open cmd: %v", err)
	}
	if openCmd["type"] != "open" {
		t.Errorf("open type = %v, want open", openCmd["type"])
	}
	channelID, ok := openCmd["channel_id"].(string)
	if !ok || channelID == "" {
		t.Fatalf("missing channel_id: %v", openCmd)
	}
	if tp, ok := openCmd["target_port"].(float64); !ok || int(tp) != 9100 {
		t.Errorf("target_port = %v, want 9100", openCmd["target_port"])
	}

	dataURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/tai/data/" + channelID
	dataConn, _, err := websocket.DefaultDialer.Dial(dataURL, http.Header{
		"Authorization": []string{"Bearer valid-token"},
	})
	if err != nil {
		t.Fatalf("dial data: %v", err)
	}
	defer dataConn.Close()

	wg.Wait()
	if requestErr != nil {
		t.Fatalf("RequestChannel: %v", requestErr)
	}
	if channelConn == nil {
		t.Fatal("expected non-nil conn from RequestChannel")
	}
	defer channelConn.Close()

	payload := []byte("grpc-payload-test")
	dataConn.WriteMessage(websocket.BinaryMessage, payload)

	buf := make([]byte, 256)
	n, err := channelConn.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("read bridged: %v", err)
	}
	if string(buf[:n]) != "grpc-payload-test" {
		t.Errorf("bridged data = %q, want %q", buf[:n], "grpc-payload-test")
	}
}
