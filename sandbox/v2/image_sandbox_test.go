package sandbox_test

import (
	"context"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func TestImage_Exists(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ok, err := m.ImageExists(ctx, nodeID, sandboxtest.TestImage())
	if err != nil {
		t.Fatalf("ImageExists: %v", err)
	}
	if !ok {
		t.Fatalf("expected image %s to exist after EnsureImage", sandboxtest.TestImage())
	}

	ok2, err := m.ImageExists(ctx, nodeID, "nonexistent-image:v999.999")
	if err != nil {
		t.Fatalf("ImageExists for nonexistent: %v", err)
	}
	if ok2 {
		t.Fatal("expected nonexistent image to not exist")
	}
}

func TestImage_Pull(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	ch, err := m.PullImage(ctx, nodeID, sandboxtest.TestImage(), sandbox.ImagePullOptions{})
	if err != nil {
		t.Fatalf("PullImage: %v", err)
	}
	for range ch {
		// drain progress
	}

	ok, err := m.ImageExists(ctx, nodeID, sandboxtest.TestImage())
	if err != nil {
		t.Fatalf("ImageExists after pull: %v", err)
	}
	if !ok {
		t.Fatal("image should exist after pull")
	}
}

func TestImage_Ensure(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := m.EnsureImage(ctx, nodeID, sandboxtest.TestImage(), sandbox.ImagePullOptions{}); err != nil {
		t.Fatalf("EnsureImage first: %v", err)
	}
	if err := m.EnsureImage(ctx, nodeID, sandboxtest.TestImage(), sandbox.ImagePullOptions{}); err != nil {
		t.Fatalf("EnsureImage second (idempotent): %v", err)
	}
}

func TestImage_EnsureBadRef(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := m.EnsureImage(ctx, nodeID, "invalid/repo:nonexistent-tag-xyz", sandbox.ImagePullOptions{})
	if err == nil {
		t.Fatal("expected error for invalid image reference")
	}
}
