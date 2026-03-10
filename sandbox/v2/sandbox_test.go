package sandbox_test

import (
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestInit(t *testing.T) {
	sandbox.Init()
	m := sandbox.M()
	if m == nil {
		t.Fatal("M() returned nil")
	}
	m.Close()
}

func TestMPanicWithoutInit(t *testing.T) {
	sandbox.ResetForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from M() without Init")
		}
	}()
	sandbox.M()
}
