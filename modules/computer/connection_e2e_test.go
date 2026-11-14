//go:build e2e

package computer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

func TestE2EComputerLifecycle(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)

	result, err := svc.Connect(ctx, types.ConnectRequest{
		Name:       "E2E Local Computer",
		Visibility: types.VisibilityPrivate,
		Scope:      types.ScopeUser,
		UserID:     "user_computer_e2e",
	})
	if err != nil {
		t.Fatalf("connect computer: %v", err)
	}
	if result.ComputerID == "" || result.NodeID == "" || result.NodeToken == "" {
		t.Fatalf("incomplete connect result: %+v", result)
	}
	if result.Computer.Status != types.StatusPending {
		t.Fatalf("status=%s, want pending", result.Computer.Status)
	}
	if _, err := oauth.OAuth.ValidateNodeToken(ctx, result.NodeToken); err != nil {
		t.Fatalf("issued node token invalid: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), result.ComputerID) })

	got, err := svc.Get(ctx, result.ComputerID)
	if err != nil {
		t.Fatalf("get computer: %v", err)
	}
	if got.NodeID != result.NodeID {
		t.Fatalf("node_id=%s, want %s", got.NodeID, result.NodeID)
	}
	items, err := svc.List(ctx, types.ListFilter{UserID: "user_computer_e2e"})
	if err != nil {
		t.Fatalf("list computers: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("computer list is empty")
	}
	if err := svc.Disconnect(ctx, result.ComputerID); err != nil {
		t.Fatalf("disconnect computer: %v", err)
	}
	got, _ = svc.Get(ctx, result.ComputerID)
	if got.Status != types.StatusDisconnected {
		t.Fatalf("status=%s, want disconnected", got.Status)
	}
	if err := svc.Reconnect(ctx, result.ComputerID); err != nil {
		t.Fatalf("reconnect computer: %v", err)
	}
	got, _ = svc.Get(ctx, result.ComputerID)
	if got.Status != types.StatusPending {
		t.Fatalf("status=%s, want pending", got.Status)
	}
	if _, err := svc.ListRuntimes(ctx, result.ComputerID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("list runtimes: %v", err)
	}
	if _, err := svc.ListContainers(ctx, result.ComputerID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("list containers: %v", err)
	}
	if _, err := svc.ListWorkspaces(ctx, result.ComputerID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("list workspaces: %v", err)
	}
	if err := svc.Delete(ctx, result.ComputerID); err != nil {
		t.Fatalf("delete computer: %v", err)
	}
	if _, err := svc.Get(ctx, result.ComputerID); err == nil {
		t.Fatal("computer still exists after delete")
	}
	if _, err := oauth.OAuth.ValidateNodeToken(ctx, result.NodeToken); err == nil {
		t.Fatal("node token still valid after delete")
	}
}

// TestE2EComputerServiceCRUD exercises the service against the standard E2E
// environment and its real SQLite/PostgreSQL database.
func TestE2EComputerServiceCRUD(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	created, err := svc.Create(ctx, types.CreateComputerRequest{
		NodeID: "node-service1", Name: "Service E2E", UserID: "user_service_e2e",
		DataDir: "/tmp/service-e2e",
	})
	if err != nil {
		t.Fatalf("create service computer: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ID) })
	if created.ID[:4] != "cmp-" || created.Status != types.StatusPending {
		t.Fatalf("unexpected created computer: %+v", created)
	}
	if _, err := svc.Get(ctx, created.ID); err != nil {
		t.Fatalf("get service computer: %v", err)
	}
	items, err := svc.List(ctx, types.ListFilter{UserID: "user_service_e2e"})
	if err != nil || len(items) == 0 {
		t.Fatalf("list service computer=%v err=%v", items, err)
	}
	name, visibility := "Service E2E Renamed", types.VisibilityMembers
	updated, err := svc.Update(ctx, created.ID, types.UpdateRequest{Name: &name, Visibility: &visibility})
	if err != nil || updated.Name != name || updated.Visibility != visibility {
		t.Fatalf("update service computer=%+v err=%v", updated, err)
	}
	if err := svc.Disconnect(ctx, created.ID); err != nil {
		t.Fatalf("disconnect service computer: %v", err)
	}
	if err := svc.Reconnect(ctx, created.ID); err != nil {
		t.Fatalf("reconnect service computer: %v", err)
	}
	if _, err := svc.ListRuntimes(ctx, created.ID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("service runtimes: %v", err)
	}
	if _, err := svc.ListContainers(ctx, created.ID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("service containers: %v", err)
	}
	if _, err := svc.ListWorkspaces(ctx, created.ID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("service workspaces: %v", err)
	}
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete service computer: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); err == nil {
		t.Fatal("service computer still exists after delete")
	}
}

// TestE2EComputerServiceErrorPaths covers service validation and cancellation
// paths using the standard E2E environment; no custom DB or fake store.
func TestE2EComputerServiceErrorPaths(t *testing.T) {
	_ = testEnvironment(t)
	svc := computer.New().WithTokens(oauth.OAuth)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.Create(ctx, types.CreateComputerRequest{NodeID: "n", Name: "n"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create canceled: %v", err)
	}
	if _, err := svc.Create(context.Background(), types.CreateComputerRequest{}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("create invalid: %v", err)
	}
	if _, err := svc.Get(context.Background(), ""); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("get empty: %v", err)
	}
	if _, err := svc.Get(context.Background(), "not-found"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("get missing: %v", err)
	}
	if _, err := svc.List(ctx, types.ListFilter{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("list canceled: %v", err)
	}
	if _, err := svc.Update(context.Background(), "not-found", types.UpdateRequest{Name: stringPtr("x")}); err == nil {
		t.Fatal("update missing should fail")
	}
	bad := "public"
	if _, err := svc.Update(context.Background(), "not-found", types.UpdateRequest{Visibility: &bad}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("update invalid visibility: %v", err)
	}
	if _, err := svc.Update(ctx, "not-found", types.UpdateRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("update canceled: %v", err)
	}
	for _, req := range []types.ConnectRequest{
		{}, {Name: "x", Visibility: "invalid"}, {Name: "x"},
		{Name: "x", Scope: types.ScopeUser}, {Name: "x", Scope: types.ScopeTeam, UserID: "u"},
	} {
		if _, err := svc.Connect(context.Background(), req); !errors.Is(err, computer.ErrInvalid) {
			t.Fatalf("connect invalid %+v: %v", req, err)
		}
	}
	if _, err := svc.Connect(ctx, types.ConnectRequest{Name: "x", UserID: "u"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("connect canceled: %v", err)
	}
	if err := svc.Delete(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete canceled: %v", err)
	}
	if err := svc.Delete(context.Background(), "not-found"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("delete missing: %v", err)
	}
	if err := svc.Disconnect(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("disconnect canceled: %v", err)
	}
	if err := svc.Reconnect(context.Background(), "not-found"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("reconnect missing: %v", err)
	}
	if _, err := svc.ListRuntimes(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("runtimes canceled: %v", err)
	}
	if _, err := svc.ListContainers(context.Background(), "not-found"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("containers missing: %v", err)
	}
	if _, err := svc.ListWorkspaces(context.Background(), "not-found"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("workspaces missing: %v", err)
	}
}

func stringPtr(value string) *string { return &value }

// TestE2EComputerServiceMorePaths uses only the standard E2E environment and
// covers filter, empty-update, and missing-model branches.
func TestE2EComputerServiceMorePaths(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	created, err := svc.Create(ctx, types.CreateComputerRequest{
		NodeID: "node-service2", Name: "More Paths", UserID: "user_more_paths",
		Scope: types.ScopeUser, Visibility: types.VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("create more paths: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ID) })
	if _, err := svc.List(ctx, types.ListFilter{
		Scope: types.ScopeUser, Visibility: types.VisibilityPrivate, Status: types.StatusPending,
		UserID: "user_more_paths", Page: 1, PageSize: 10,
	}); err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if _, err := svc.List(ctx, types.ListFilter{Page: 1}); err != nil {
		t.Fatalf("list pagination: %v", err)
	}
	if _, err := svc.Update(ctx, created.ID, types.UpdateRequest{}); err != nil {
		t.Fatalf("empty update: %v", err)
	}
	if _, err := svc.ListRuntimes(ctx, "missing"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("runtimes missing: %v", err)
	}
	if _, err := svc.ListWorkspaces(ctx, "missing"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("workspaces missing: %v", err)
	}
	// No-model branch: still the standard environment, only remove the loaded
	// model temporarily; restore immediately afterward.
	saved := model.Models["__yao.computers"]
	delete(model.Models, "__yao.computers")
	if _, err := (&computer.Service{}).Get(ctx, "id"); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("get without model: %v", err)
	}
	model.Models["__yao.computers"] = saved
}

func TestE2EComputerNoTokenService(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	created, err := svc.Create(ctx, types.CreateComputerRequest{NodeID: "node-notok1", Name: "No Token", UserID: "u"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	saved := oauth.OAuth
	oauth.OAuth = nil
	defer func() { oauth.OAuth = saved }()
	bare := computer.New()
	if _, err := bare.Connect(ctx, types.ConnectRequest{Name: "no-token", UserID: "u"}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("connect without token: %v", err)
	}
	if err := bare.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete without token: %v", err)
	}
}

func TestE2EComputerContainerWorkspaceCanceled(t *testing.T) {
	_ = testEnvironment(t)
	svc := computer.New().WithTokens(oauth.OAuth)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.ListContainers(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("containers canceled: %v", err)
	}
	if _, err := svc.ListWorkspaces(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("workspaces canceled: %v", err)
	}
}

func TestE2EComputerNoModel(t *testing.T) {
	_ = testEnvironment(t)
	saved, had := model.Models["__yao.computers"]
	delete(model.Models, "__yao.computers")
	defer func() {
		if had {
			model.Models["__yao.computers"] = saved
		}
	}()
	svc := computer.New()
	ctx := context.Background()
	if _, err := svc.Get(ctx, "id"); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("get no model: %v", err)
	}
	if _, err := svc.List(ctx, types.ListFilter{}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("list no model: %v", err)
	}
	if _, err := svc.Create(ctx, types.CreateComputerRequest{NodeID: "node-abcdefgh", Name: "x"}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("create no model: %v", err)
	}
	if _, err := svc.Connect(ctx, types.ConnectRequest{Name: "x", UserID: "u"}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("connect no model: %v", err)
	}
	if err := svc.Disconnect(ctx, "id"); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("disconnect no model: %v", err)
	}
	if err := svc.Delete(ctx, "id"); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("delete no model: %v", err)
	}
	if _, err := svc.ListRuntimes(ctx, "id"); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("runtimes no model: %v", err)
	}
}

func TestE2EConnectWithLease(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	expiry := time.Now().UTC().Add(time.Hour)
	svc := computer.New().WithTokens(oauth.OAuth)
	result, err := svc.Connect(ctx, types.ConnectRequest{
		Name: "Lease Computer", Visibility: types.VisibilityPrivate,
		Scope: types.ScopeUser, UserID: "user_connect_lease",
		LeaseSource: types.LeaseSourceAdmin, LeaseExpiresAt: &expiry,
	})
	if err != nil {
		t.Fatalf("connect lease: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), result.ComputerID) })
	got, err := svc.Get(ctx, result.ComputerID)
	if err != nil || got.LeaseSource != types.LeaseSourceAdmin || got.LeaseExpiresAt == nil {
		t.Fatalf("lease result=%+v err=%v", got, err)
	}
}

type fakeTaiNodeProvider struct{}

func (fakeTaiNodeProvider) Node(nodeID string) (types.TaiNode, bool) {
	return types.TaiNode{ID: nodeID, ContainerRuntimes: []types.Runtime{{Name: "docker", Version: "27", Ready: true}}}, true
}
func (fakeTaiNodeProvider) ListWorkspaces(context.Context, string) ([]types.TaiWorkspace, error) {
	return []types.TaiWorkspace{{ID: "ws-1", Name: "Workspace"}}, nil
}

func TestE2ETaiNodeProviderRuntimeAndWorkspace(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth).WithTaiNodeProvider(fakeTaiNodeProvider{})
	result, err := svc.Connect(ctx, types.ConnectRequest{Name: "Tai Provider", Visibility: types.VisibilityPrivate, Scope: types.ScopeUser, UserID: "user_tai_provider"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), result.ComputerID) })
	runtimes, err := svc.ListRuntimes(ctx, result.ComputerID)
	if err != nil || len(runtimes) != 1 || runtimes[0].Name != "docker" {
		t.Fatalf("runtimes=%+v err=%v", runtimes, err)
	}
	workspaces, err := svc.ListWorkspaces(ctx, result.ComputerID)
	if err != nil || len(workspaces) != 1 || workspaces[0].ID != "ws-1" {
		t.Fatalf("workspaces=%+v err=%v", workspaces, err)
	}
	if _, err := svc.ListContainers(ctx, result.ComputerID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("containers err=%v", err)
	}
}

func TestE2EGetByNodeID(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	created, err := svc.Connect(ctx, types.ConnectRequest{Name: "By Node", Visibility: types.VisibilityPrivate, Scope: types.ScopeUser, UserID: "user_by_node"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ComputerID) })
	got, err := svc.GetByNodeID(ctx, created.NodeID)
	if err != nil || got.ID != created.ComputerID {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if _, err := svc.GetByNodeID(ctx, ""); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("empty err=%v", err)
	}
	if _, err := svc.GetByNodeID(ctx, "missing-node"); !errors.Is(err, computer.ErrNotFound) {
		t.Fatalf("missing err=%v", err)
	}
}

type missingTaiProvider struct{}

func (missingTaiProvider) Node(string) (types.TaiNode, bool) { return types.TaiNode{}, false }
func (missingTaiProvider) ListWorkspaces(context.Context, string) ([]types.TaiWorkspace, error) {
	return nil, errors.New("tai unavailable")
}

func TestE2ETaiNodeProviderErrors(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth).WithTaiNodeProvider(missingTaiProvider{})
	created, err := svc.Connect(ctx, types.ConnectRequest{Name: "Missing Tai", Visibility: types.VisibilityPrivate, Scope: types.ScopeUser, UserID: "user_missing_tai"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ComputerID) })
	if _, err := svc.ListRuntimes(ctx, created.ComputerID); !errors.Is(err, computer.ErrNotSupported) {
		t.Fatalf("runtimes err=%v", err)
	}
	if _, err := svc.ListWorkspaces(ctx, created.ComputerID); err == nil {
		t.Fatal("expected workspace error")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := svc.ListRuntimes(canceled, created.ComputerID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled err=%v", err)
	}
	if _, err := svc.ListWorkspaces(canceled, created.ComputerID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled ws err=%v", err)
	}
	if _, err := svc.ListContainers(canceled, created.ComputerID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled containers err=%v", err)
	}
}

type fakeDisconnectProvider struct {
	fakeTaiNodeProvider
	called bool
	err    error
}

func (p *fakeDisconnectProvider) GoAway(context.Context, string, string) error {
	p.called = true
	return p.err
}

func TestE2EDisconnectTaiNode(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	provider := &fakeDisconnectProvider{}
	svc := computer.New().WithTokens(oauth.OAuth).WithTaiNodeProvider(provider)
	created, err := svc.Connect(ctx, types.ConnectRequest{Name: "Disconnect Tai", Visibility: types.VisibilityPrivate, Scope: types.ScopeUser, UserID: "user_disconnect_tai"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ComputerID) })
	if err := svc.Disconnect(ctx, created.ComputerID); err != nil {
		t.Fatal(err)
	}
	if !provider.called {
		t.Fatal("GoAway was not called")
	}
	got, _ := svc.Get(ctx, created.ComputerID)
	if got.Status != types.StatusDisconnected {
		t.Fatalf("status=%s", got.Status)
	}
	provider.err = errors.New("goaway failed")
	if err := svc.Disconnect(ctx, created.ComputerID); err == nil {
		t.Fatal("expected goaway error")
	}
	provider.err = nil
	if err := svc.Reconnect(ctx, created.ComputerID); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.Get(ctx, created.ComputerID)
	if got.Status != types.StatusPending {
		t.Fatalf("reconnect status=%s", got.Status)
	}
}

func TestE2EConnectTransportPort(t *testing.T) {
	_ = testEnvironment(t)
	old := config.Conf.Tai.Server.TransportPort
	config.Conf.Tai.Server.TransportPort = 6123
	t.Cleanup(func() { config.Conf.Tai.Server.TransportPort = old })
	svc := computer.New().WithTokens(oauth.OAuth)
	created, err := svc.Connect(context.Background(), types.ConnectRequest{Name: "Port", Visibility: types.VisibilityPrivate, Scope: types.ScopeUser, UserID: "user_port"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ComputerID) })
	if created.TransportPort != 6123 {
		t.Fatalf("port=%d", created.TransportPort)
	}
}
