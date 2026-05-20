package sandbox_test

import (
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestSandbox_InitAndM(t *testing.T) {
	testprepare.PrepareSandbox(t)

	m := sandbox.M()
	if m == nil {
		t.Fatal("expected non-nil Manager after PrepareSandbox")
	}
}

func TestSandbox_MReturnsConsistentInstance(t *testing.T) {
	testprepare.PrepareSandbox(t)

	m1 := sandbox.M()
	m2 := sandbox.M()
	if m1 != m2 {
		t.Fatal("expected M() to return the same Manager instance across calls")
	}
}
