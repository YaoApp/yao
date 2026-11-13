package computer

import (
	"context"
	"testing"

	"github.com/yaoapp/yao/modules/computer/types"
)

func TestSSHTaiControllerMethodsRequireTarget(t *testing.T) {
	c := NewSSHTaiController()
	ctx := context.Background()
	if err := c.Bootstrap(ctx, types.SSHTarget{}, types.BootstrapRequest{}); err == nil {
		t.Fatal("bootstrap should fail")
	}
	req := types.StartTaiRequest{NodeID: "node-x", DataDir: "/data", ServerURL: "https://yao", NodeToken: "tok", SandboxImage: "img", LogLevel: "info", LogFormat: "json", LogFile: "/tmp/tai.log"}
	if err := c.Start(ctx, types.SSHTarget{}, req); err == nil {
		t.Fatal("start should fail")
	}
	if err := c.Stop(ctx, types.SSHTarget{}); err == nil {
		t.Fatal("stop should fail")
	}
	if _, err := c.Status(ctx, types.SSHTarget{}); err == nil {
		t.Fatal("status should fail")
	}
	if _, err := c.Logs(ctx, types.SSHTarget{}, 0); err == nil {
		t.Fatal("logs should fail")
	}
}

func TestSSHTaiControllerTimeoutDefault(t *testing.T) {
	if got := (&SSHTaiController{}).timeout(); got == 0 {
		t.Fatal("default timeout is zero")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewSSHTaiController().run(ctx, types.SSHTarget{Host: "127.0.0.1", PrivateKey: "x"}, "true"); err == nil {
		t.Fatal("expected context error")
	}
}
