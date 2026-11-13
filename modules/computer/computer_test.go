//go:build !e2e

package computer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/computer/types"
)

func TestNewService(t *testing.T) {
	if computer.New() == nil {
		t.Fatal("New returned nil")
	}
}

func TestServiceWithoutModel(t *testing.T) {
	svc := computer.New()
	ctx := context.Background()
	if _, err := svc.Get(ctx, "id"); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("get without model: %v", err)
	}
	if _, err := svc.List(ctx, types.ListFilter{}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("list without model: %v", err)
	}
	if _, err := svc.Create(ctx, types.CreateComputerRequest{NodeID: "node-abcdefgh", Name: "x"}); !errors.Is(err, computer.ErrInvalid) {
		t.Fatalf("create without model: %v", err)
	}
}
