//go:build e2e

package computer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

func TestE2ECreateOrUpdateCloudComputer(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)

	first, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{
		InstanceID: "inst-e2e-1", Name: "cloud e2e", ProviderType: "cloud",
		Scope: types.ScopeUser, Visibility: types.VisibilityPrivate,
		Status: types.StatusProvisioning, UserID: "user_cloud_e2e",
		DataDir: "/tmp/cloud-e2e", LeaseSource: types.LeaseSourceTao,
	})
	if err != nil {
		t.Fatalf("create provisioning: %v", err)
	}
	if first.Computer == nil || first.Computer.Status != types.StatusProvisioning || first.Computer.InstanceID != "inst-e2e-1" {
		t.Fatalf("unexpected first computer: %+v", first.Computer)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), first.Computer.ID) })

	second, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{
		InstanceID: "inst-e2e-1", Status: types.StatusPending, IssueNodeToken: true,
	})
	if err != nil {
		t.Fatalf("advance running: %v", err)
	}
	if second.Computer.ID != first.Computer.ID {
		t.Fatalf("upsert created a new computer: %s != %s", second.Computer.ID, first.Computer.ID)
	}
	if second.Computer.Status != types.StatusPending || second.NodeToken == "" {
		t.Fatalf("unexpected second computer/token: %+v token=%q", second.Computer, second.NodeToken)
	}
	if _, err := oauth.OAuth.ValidateNodeToken(ctx, second.NodeToken); err != nil {
		t.Fatalf("issued token invalid: %v", err)
	}
}

func TestE2ELeaseExpireAndRenew(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	past := time.Now().UTC().Add(-time.Hour)
	created, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{
		InstanceID: "inst-e2e-lease", Name: "lease", ProviderType: "cloud",
		Scope: types.ScopeUser, Visibility: types.VisibilityPrivate, Status: types.StatusPending,
		UserID: "user_lease_e2e", LeaseSource: types.LeaseSourceInternal, LeaseExpiresAt: &past,
	})
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.Computer.ID) })
	n, err := svc.ExpireDue(ctx, time.Now().UTC())
	if err != nil || n == 0 {
		t.Fatalf("expire due n=%d err=%v", n, err)
	}
	got, _ := svc.Get(ctx, created.Computer.ID)
	if got.Status != types.StatusExpired {
		t.Fatalf("status=%s", got.Status)
	}
	renewed, err := svc.Renew(ctx, created.Computer.ID, types.RenewLeaseRequest{Duration: time.Hour, LeaseSource: types.LeaseSourceAdmin, IssueNodeToken: true})
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if renewed.Computer.Status != types.StatusPending || renewed.NodeToken == "" {
		t.Fatalf("renewed=%+v token=%q", renewed.Computer, renewed.NodeToken)
	}
}

func TestE2ECreateOrUpdateValidationAndUpdate(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	if _, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{}); err == nil {
		t.Fatal("expected instance_id error")
	}
	created, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-update", Name: "before", ProviderType: "cloud", UserID: "user_update_e2e", Status: types.StatusPending})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.Computer.ID) })
	later := time.Now().UTC().Add(2 * time.Hour)
	updated, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-update", NodeID: "node-override", Name: "after", Status: types.StatusOnline, LeaseSource: types.LeaseSourceAdmin, LeaseExpiresAt: &later})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Computer.ID != created.Computer.ID || updated.Computer.Name != "after" || updated.Computer.Status != types.StatusOnline || updated.Computer.LeaseExpiresAt == nil {
		t.Fatalf("updated=%+v", updated.Computer)
	}
}

func TestE2ELeaseValidation(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	stored, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-lease-validation", Status: types.StatusPending, UserID: "user_lease_validation"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), stored.Computer.ID) })
	if _, err := svc.Renew(ctx, stored.Computer.ID, types.RenewLeaseRequest{}); err == nil {
		t.Fatal("expected expiry error")
	}
	if _, err := svc.Renew(ctx, "missing", types.RenewLeaseRequest{Duration: time.Hour}); err == nil {
		t.Fatal("expected missing computer error")
	}
	n, err := svc.ExpireDue(ctx, time.Now().UTC().Add(-24*time.Hour))
	if err != nil || n != 0 {
		t.Fatalf("expire none n=%d err=%v", n, err)
	}
}

type errorTokens struct{}

func (errorTokens) IssueNodeToken(context.Context, oauth.IssueRequest) (*oauth.NodeToken, error) {
	return nil, errors.New("token error")
}
func (errorTokens) RevokeNodeTokenByComputer(context.Context, string) error {
	return errors.New("revoke error")
}

func TestE2ELeaseWithoutTokenService(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(errorTokens{})
	past := time.Now().UTC().Add(-time.Hour)
	created, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-no-token", Status: types.StatusExpired, UserID: "user_no_token", LeaseExpiresAt: &past})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.Computer.ID) })
	if _, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-no-token", Status: types.StatusPending, IssueNodeToken: true}); err == nil {
		t.Fatal("expected token service error")
	}
}

func TestE2EExpireDueSkipsReleased(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	past := time.Now().UTC().Add(-time.Hour)
	created, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-released", Status: types.StatusReleased, UserID: "user_released", LeaseExpiresAt: &past})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.Computer.ID) })
	if _, err := svc.ExpireDue(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("expire due: %v", err)
	}
	got, _ := svc.Get(ctx, created.Computer.ID)
	if got.Status != types.StatusReleased {
		t.Fatalf("released computer changed to %s", got.Status)
	}
}

func TestE2EExpireDueRevokeError(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(errorTokens{})
	past := time.Now().UTC().Add(-time.Hour)
	created, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-revoke-error", Status: types.StatusPending, UserID: "user_revoke_error", LeaseExpiresAt: &past})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = computer.New().Delete(context.Background(), created.Computer.ID) })
	if _, err := svc.ExpireDue(ctx, time.Now().UTC()); err == nil {
		t.Fatal("expected revoke error")
	}
}

func TestE2ERenewExpiredIssuesToken(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	past := time.Now().UTC().Add(-time.Hour)
	created, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{InstanceID: "inst-e2e-renew-expired", Status: types.StatusExpired, UserID: "user_renew_expired", LeaseExpiresAt: &past})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.Computer.ID) })
	renewed, err := svc.Renew(ctx, created.Computer.ID, types.RenewLeaseRequest{Duration: time.Hour, LeaseSource: types.LeaseSourceAdmin, IssueNodeToken: true})
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if renewed.NodeToken == "" || renewed.Computer.Status != types.StatusPending {
		t.Fatalf("renewed=%+v token=%q", renewed.Computer, renewed.NodeToken)
	}
}

func TestE2ELeaseContextCanceled(t *testing.T) {
	_ = testEnvironment(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := computer.New()
	if _, err := svc.Renew(ctx, "x", types.RenewLeaseRequest{}); err == nil {
		t.Fatal("renew should fail")
	}
	if _, err := svc.ExpireDue(ctx, time.Now()); err == nil {
		t.Fatal("expire should fail")
	}
	if _, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{}); err == nil {
		t.Fatal("create should fail")
	}
}

func TestE2ECreateOrUpdateByComputerID(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	created, err := svc.Create(ctx, types.CreateComputerRequest{NodeID: "node-by-id", Name: "by-id", Status: types.StatusPending, UserID: "user_by_id"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ID) })
	updated, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{ID: created.ID, Name: "by-id-updated", Status: types.StatusOnline})
	if err != nil {
		t.Fatalf("upsert by computer_id: %v", err)
	}
	if updated.Computer.ID != created.ID || updated.Computer.Name != "by-id-updated" || updated.Computer.Status != types.StatusOnline {
		t.Fatalf("updated=%+v", updated.Computer)
	}
	newID := "cmp-upsert-by-id"
	newComputer, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{ID: newID, Name: "new-by-id", Status: types.StatusProvisioning, NodeID: "node-upsert-by-id"})
	if err != nil {
		t.Fatalf("create by computer_id: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), newComputer.Computer.ID) })
	if newComputer.Computer.ID != newID {
		t.Fatalf("id=%s", newComputer.Computer.ID)
	}
}

func TestE2ECreateOrUpdateDualKeyBranches(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	if _, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{}); err == nil {
		t.Fatal("expected missing key error")
	}
	created, err := svc.Create(ctx, types.CreateComputerRequest{NodeID: "node-dual", Name: "dual", Status: types.StatusPending, UserID: "user_dual"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ID) })
	updated, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{ID: created.ID, InstanceID: "inst-dual", Status: types.StatusProvisioning})
	if err != nil {
		t.Fatalf("update instance_id by id: %v", err)
	}
	if updated.Computer.InstanceID != "inst-dual" {
		t.Fatalf("instance=%q", updated.Computer.InstanceID)
	}
}
