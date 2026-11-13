//go:build e2e

package computer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yaoapp/gou/model"
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
	if _, err := svc.ListRuntimes(ctx, result.ComputerID); err != nil {
		t.Fatalf("list runtimes: %v", err)
	}
	if _, err := svc.ListContainers(ctx, result.ComputerID); err != nil {
		t.Fatalf("list containers: %v", err)
	}
	if _, err := svc.ListWorkspaces(ctx, result.ComputerID); err != nil {
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
	if _, err := svc.ListRuntimes(ctx, created.ID); err != nil {
		t.Fatalf("service runtimes: %v", err)
	}
	if _, err := svc.ListContainers(ctx, created.ID); err != nil {
		t.Fatalf("service containers: %v", err)
	}
	if _, err := svc.ListWorkspaces(ctx, created.ID); err != nil {
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
