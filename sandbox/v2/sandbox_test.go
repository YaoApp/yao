package sandbox_test

import (
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestInit(t *testing.T) {
	cfg := sandbox.Config{
		Pool: []sandbox.Pool{
			{Name: "test", Addr: "local"},
		},
	}
	if err := sandbox.Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m := sandbox.M()
	if m == nil {
		t.Fatal("M() returned nil")
	}
	m.Close()
}

func TestInitEmpty(t *testing.T) {
	cfg := sandbox.Config{}
	if err := sandbox.Init(cfg); err != nil {
		t.Fatalf("Init with empty config: %v", err)
	}
	sandbox.M().Close()
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
