//go:build e2e

package computer_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/yaoapp/gou/model"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

func ensureUser(t *testing.T) string {
	t.Helper()
	m := model.Select("__yao.user")
	if m == nil {
		t.Fatal("user model is not loaded")
	}
	for _, email := range []string{"user@e2e.local", "admin@e2e.local"} {
		rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "email", Value: email}}})
		if err == nil && len(rows) > 0 {
			return fmt.Sprint(rows[0]["user_id"])
		}
	}
	t.Fatal("seeded e2e user not found")
	return ""
}

func TestE2EBindAndResolveTaoKey(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	userID := ensureUser(t)
	key := "sk-tao-e2e-secret"
	if err := taocomputer.BindKey(ctx, userID, "bad-key"); err == nil {
		t.Fatal("expected invalid key error")
	}
	if err := taocomputer.BindKey(ctx, userID, key); err != nil {
		t.Fatalf("bind: %v", err)
	}
	got, err := (taocomputer.ModelKeyResolver{}).ResolveTaoKey(ctx, userID)
	if err != nil || got != key {
		t.Fatalf("resolve key=%q err=%v", got, err)
	}
	if _, err := (taocomputer.ModelKeyResolver{}).ResolveTaoKey(ctx, "missing"); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestE2EBindKeyValidation(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	if err := taocomputer.BindKey(ctx, "", "sk-tao-x"); err == nil {
		t.Fatal("expected user error")
	}
	if _, err := (taocomputer.ModelKeyResolver{}).ResolveTaoKey(ctx, ""); err == nil {
		t.Fatal("expected resolver user error")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := taocomputer.BindKey(canceled, "u", "sk-tao-x"); err == nil {
		t.Fatal("expected context error")
	}
}

func TestE2EDeleteAndHasTaoKey(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	userID := ensureUser(t)
	if err := taocomputer.BindKey(ctx, userID, "sk-tao-delete"); err != nil {
		t.Fatal(err)
	}
	if ok, err := taocomputer.HasKey(ctx, userID); err != nil || !ok {
		t.Fatalf("has=%v err=%v", ok, err)
	}
	if err := taocomputer.DeleteKey(ctx, userID); err != nil {
		t.Fatal(err)
	}
	if ok, err := taocomputer.HasKey(ctx, userID); err != nil || ok {
		t.Fatalf("has after delete=%v err=%v", ok, err)
	}
}

func TestE2EKeyContextCanceled(t *testing.T) {
	_ = testEnvironment(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := taocomputer.DeleteKey(ctx, "u"); err == nil {
		t.Fatal("delete should fail")
	}
	if _, err := taocomputer.HasKey(ctx, "u"); err == nil {
		t.Fatal("has should fail")
	}
}

func TestE2EBindKeyUpdatesExisting(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	userID := ensureUser(t)
	if err := taocomputer.BindKey(ctx, userID, "sk-tao-one"); err != nil {
		t.Fatal(err)
	}
	if err := taocomputer.BindKey(ctx, userID, "sk-tao-two"); err != nil {
		t.Fatal(err)
	}
	key, err := (taocomputer.ModelKeyResolver{}).ResolveTaoKey(ctx, userID)
	if err != nil || key != "sk-tao-two" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	_ = taocomputer.DeleteKey(ctx, userID)
}

func TestE2EKeyMissingBranches(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	if err := taocomputer.DeleteKey(ctx, "missing-key-user"); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
	if ok, err := taocomputer.HasKey(ctx, "missing-key-user"); err != nil || ok {
		t.Fatalf("has missing=%v err=%v", ok, err)
	}
}

func TestE2EKeyMissingUserAndEmptyKey(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	if err := taocomputer.BindKey(ctx, "missing-user-xyz", "sk-tao-x"); err == nil {
		t.Fatal("expected missing user error")
	}
	userID := ensureUser(t)
	if err := taocomputer.BindKey(ctx, userID, "sk-tao-empty"); err != nil {
		t.Fatal(err)
	}
	m := model.Select("__yao.user")
	rows, _ := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "user_id", Value: userID}}})
	if len(rows) > 0 {
		_ = m.Update(fmt.Sprint(rows[0]["id"]), map[string]interface{}{"tao_api_key": ""})
	}
	if _, err := (taocomputer.ModelKeyResolver{}).ResolveTaoKey(ctx, userID); err == nil {
		t.Fatal("expected empty key error")
	}
	_ = taocomputer.DeleteKey(ctx, userID)
}
