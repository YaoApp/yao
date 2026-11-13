package computer

import (
	"context"
	"testing"
)

func TestProductIDStable(t *testing.T) {
	a := productID("computer", "flex-2c2g")
	b := productID("computer", "flex-2c2g")
	if a != b || len(a) != 31 || a[:4] != "tao-" {
		t.Fatalf("productID=%q", a)
	}
	if productID("computer", "b") == a {
		t.Fatal("different resources must differ")
	}
}

func TestProductWriterCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (ModelProductWriter{}).UpsertProduct(ctx, Product{}); err == nil {
		t.Fatal("expected context error")
	}
}

func TestProductWriterDisableCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (ModelProductWriter{}).DisableMissing(ctx, "computer", map[string]struct{}{}); err == nil {
		t.Fatal("expected context error")
	}
}
