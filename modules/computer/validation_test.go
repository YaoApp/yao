package computer_test

import (
	"context"
	"testing"

	"github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/computer/types"
)

func TestConnectValidationBranches(t *testing.T) {
	svc := computer.New()
	ctx := context.Background()
	if _, err := svc.Connect(ctx, types.ConnectRequest{}); err == nil {
		t.Fatal("name required")
	}
	if _, err := svc.Connect(ctx, types.ConnectRequest{Name: "x", Visibility: "invalid"}); err == nil {
		t.Fatal("visibility invalid")
	}
	if _, err := svc.Connect(ctx, types.ConnectRequest{Name: "x", Scope: types.ScopeUser}); err == nil {
		t.Fatal("user required")
	}
	if _, err := svc.Connect(ctx, types.ConnectRequest{Name: "x", Scope: types.ScopeTeam}); err == nil {
		t.Fatal("team required")
	}
}
