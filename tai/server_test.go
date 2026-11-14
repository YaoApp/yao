package tai

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/yao/config"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

type fakeOAuth struct {
	validateErr error
	refreshErr  error
	info        *oauth.NodeInfo
	token       *oauth.NodeToken
}

func (f fakeOAuth) ValidateNodeToken(context.Context, string) (*oauth.NodeInfo, error) {
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	return f.info, nil
}
func (f fakeOAuth) RefreshNodeToken(context.Context, string) (*oauth.NodeToken, error) {
	if f.refreshErr != nil {
		return nil, f.refreshErr
	}
	return f.token, nil
}

type fakeComputerStore struct {
	current *computertypes.Computer
	err     error
	last    computertypes.CreateOrUpdateComputerRequest
}

func (f *fakeComputerStore) GetByNodeID(context.Context, string) (*computertypes.Computer, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.current, nil
}
func (f *fakeComputerStore) CreateOrUpdate(_ context.Context, req computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error) {
	f.last = req
	return &computertypes.UpsertResult{Computer: f.current}, nil
}

func TestAuthProviderNodeTokens(t *testing.T) {
	info := &oauth.NodeInfo{NodeID: "node-1", UserID: "user-1"}
	a := &AuthProvider{OAuth: fakeOAuth{info: info}}
	got, err := a.ValidateNodeToken("token")
	if err != nil || got.NodeID != "node-1" || got.UserID != "user-1" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	a.OAuth = fakeOAuth{validateErr: errors.New("bad")}
	if _, err := a.ValidateNodeToken("bad"); err == nil {
		t.Fatal("expected error")
	}
	a.OAuth = fakeOAuth{token: &oauth.NodeToken{Token: "new"}}
	if token, err := a.RefreshNodeToken("old"); err != nil || token != "new" {
		t.Fatalf("token=%q err=%v", token, err)
	}
	a.OAuth = fakeOAuth{refreshErr: errors.New("bad")}
	if _, err := a.RefreshNodeToken("old"); err == nil {
		t.Fatal("expected refresh error")
	}
}

func TestAuthProviderIssueAndUnsupported(t *testing.T) {
	a := &AuthProvider{IssueNode: func(context.Context, string) (*oauth.NodeToken, error) { return &oauth.NodeToken{Token: "issued"}, nil }}
	if token, err := a.IssueNodeToken("node"); err != nil || token != "issued" {
		t.Fatalf("token=%q err=%v", token, err)
	}
	a.IssueNode = func(context.Context, string) (*oauth.NodeToken, error) { return nil, errors.New("bad") }
	if _, err := a.IssueNodeToken("node"); err == nil {
		t.Fatal("expected issue error")
	}
	if _, err := (&AuthProvider{}).IssueNodeToken("node"); err == nil {
		t.Fatal("expected missing issuer")
	}
	if _, _, err := a.IssueUserToken("node", nil); err == nil {
		t.Fatal("user token must be unsupported")
	}
	if _, err := a.RefreshUserToken("x"); err == nil {
		t.Fatal("user refresh must be unsupported")
	}
	if _, err := a.ValidateUserToken("x"); err == nil {
		t.Fatal("user validate must be unsupported")
	}
	if info := a.WellKnownInfo(); info != nil {
		t.Fatalf("expected nil inactive info, got %+v", info)
	}
}

func TestAuthProviderMissingOAuth(t *testing.T) {
	a := &AuthProvider{}
	if _, err := a.ValidateNodeToken("x"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := a.RefreshNodeToken("x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestObserver(t *testing.T) {
	store := &fakeComputerStore{current: &computertypes.Computer{ID: "cmp-1", NodeID: "node-1"}}
	o := &ComputerObserver{Computers: store}
	if err := o.NodeRegistered(context.Background(), Node{ID: "node-1"}); err != nil {
		t.Fatal(err)
	}
	if store.last.Status != computertypes.StatusOnline {
		t.Fatalf("req=%+v", store.last)
	}
	o.NodeDisconnected(context.Background(), "node-1")
	if store.last.Status != computertypes.StatusOffline {
		t.Fatalf("req=%+v", store.last)
	}
	store.err = errors.New("missing")
	if err := o.NodeRegistered(context.Background(), Node{ID: "node-1"}); err == nil {
		t.Fatal("expected error")
	}
	o.NodeDisconnected(context.Background(), "node-1")
	(&ComputerObserver{}).NodeDisconnected(context.Background(), "x")
	if err := (&ComputerObserver{}).NodeRegistered(context.Background(), Node{ID: "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestProviderMissingAndServerLifecycle(t *testing.T) {
	if _, ok := (Provider{}).Node("missing"); ok {
		t.Fatal("unexpected node")
	}
	if _, err := (Provider{}).ListWorkspaces(context.Background(), "missing"); err == nil {
		t.Fatal("expected error")
	}
	if _, ok := (ComputerProvider{}).Node("missing"); ok {
		t.Fatal("unexpected node")
	}
	if _, err := (ComputerProvider{}).ListWorkspaces(context.Background(), "missing"); err == nil {
		t.Fatal("expected error")
	}

	port := freePort(t)
	server, err := NewServer(Config{TransportPort: port, OAuth: fakeOAuth{}, Observer: &ComputerObserver{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	_ = server.Close()
	if _, err := NewServer(Config{}); err == nil {
		t.Fatal("expected missing auth")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

var _ = time.Second

func TestServerDefaultsAndStartErrors(t *testing.T) {
	server, err := NewServer(Config{OAuth: fakeOAuth{}})
	if err != nil {
		t.Fatal(err)
	}
	// Default ports are applied internally; the well-known provider must report them.
	if server.tr == nil || server.reg == nil {
		t.Fatal("transport nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := server.Start(ctx); err == nil {
		t.Fatal("expected start error")
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	var nilServer *Server
	if err := nilServer.Start(context.Background()); err == nil {
		t.Fatal("expected nil start error")
	}
	if err := nilServer.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestComputerProviderMissing(t *testing.T) {
	if _, ok := (ComputerProvider{}).Node("missing"); ok {
		t.Fatal("unexpected node")
	}
	if _, err := (ComputerProvider{}).ListWorkspaces(context.Background(), "missing"); err == nil {
		t.Fatal("expected error")
	}
	if err := (ComputerProvider{}).GoAway(context.Background(), "missing", "test"); err != nil {
		t.Fatal(err)
	}
}

func TestServerURL(t *testing.T) {
	oldHost, oldPort := config.Conf.Host, config.Conf.Port
	t.Cleanup(func() {
		config.Conf.Host = oldHost
		config.Conf.Port = oldPort
	})
	t.Setenv("YAO_SERVER_URL", "http://example.com:5099/")
	if got := ServerURL(); got != "http://example.com:5099" {
		t.Fatalf("explicit=%q", got)
	}
	t.Setenv("YAO_SERVER_URL", "")
	config.Conf.Host = "0.0.0.0"
	config.Conf.Port = 6100
	if got := ServerURL(); got != "http://127.0.0.1:6100" {
		t.Fatalf("default=%q", got)
	}
}

func TestHTTPHandlerTokenAndHeartbeat(t *testing.T) {
	port := freePort(t)
	server, err := NewServer(Config{TransportPort: port, OAuth: fakeOAuth{info: &oauth.NodeInfo{NodeID: "node"}, token: &oauth.NodeToken{Token: "new"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	h := HTTPHandler()
	if h == nil {
		t.Fatal("nil handler")
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/tai/token/refresh", strings.NewReader(`{"node_token":"old"}`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("refresh status=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "new") {
		t.Fatalf("refresh body=%s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/tai/heartbeat", strings.NewReader(`{"node_token":"old"}`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/tai/token/refresh", strings.NewReader(`{`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad refresh=%d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/tai/heartbeat", strings.NewReader(`{`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad heartbeat=%d", rr.Code)
	}
}

func TestHTTPHandlerInactiveAndTokenErrors(t *testing.T) {
	rr := httptest.NewRecorder()
	HTTPHandler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/tai/token/refresh", strings.NewReader(`{}`)))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("inactive=%d", rr.Code)
	}

	port := freePort(t)
	server, err := NewServer(Config{TransportPort: port, OAuth: fakeOAuth{refreshErr: errors.New("bad"), validateErr: errors.New("bad")}})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	h := HTTPHandler()
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/tai/token/refresh", strings.NewReader(`{"node_token":"old"}`)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("refresh error=%d", rr.Code)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/tai/heartbeat", strings.NewReader(`{"node_token":"old"}`)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("heartbeat error=%d", rr.Code)
	}
}
