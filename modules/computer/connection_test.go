package computer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/computer/types"
)

func TestConnectValidation(t *testing.T) {
	svc := computer.New()
	cases := []types.ConnectRequest{
		{},
		{Name: "x", Visibility: "invalid"},
		{Name: "x", Scope: types.ScopeUser},
		{Name: "x", Scope: types.ScopeTeam, UserID: "u"},
	}
	for _, req := range cases {
		_, err := svc.Connect(context.Background(), req)
		if !errors.Is(err, computer.ErrInvalid) {
			t.Fatalf("request %+v error=%v, want invalid", req, err)
		}
	}
}

func TestUpdateVisibilityValidation(t *testing.T) {
	bad := "public"
	_, err := computer.New().Update(context.Background(), "missing", types.UpdateRequest{Visibility: &bad})
	if !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("error=%v, want invalid", err)
	}
}
