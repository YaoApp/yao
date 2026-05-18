package sandbox_test

import (
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestExecOption_WithWorkDir(t *testing.T) {
	dir, _, _, _, _ := sandbox.ExportApplyExecOptions(sandbox.WithWorkDir("/app"))
	if dir != "/app" {
		t.Fatalf("WithWorkDir: got %q, want %q", dir, "/app")
	}
}

func TestExecOption_WithEnv(t *testing.T) {
	env := map[string]string{"A": "1", "B": "2"}
	_, got, _, _, _ := sandbox.ExportApplyExecOptions(sandbox.WithEnv(env))
	if len(got) != 2 || got["A"] != "1" || got["B"] != "2" {
		t.Fatalf("WithEnv: got %v, want %v", got, env)
	}
}

func TestExecOption_WithTimeout(t *testing.T) {
	_, _, got, _, _ := sandbox.ExportApplyExecOptions(sandbox.WithTimeout(5 * time.Second))
	want := int64(5 * time.Second)
	if got != want {
		t.Fatalf("WithTimeout: got %d, want %d", got, want)
	}
}

func TestExecOption_WithStdin(t *testing.T) {
	data := []byte("hello")
	_, _, _, got, _ := sandbox.ExportApplyExecOptions(sandbox.WithStdin(data))
	if string(got) != "hello" {
		t.Fatalf("WithStdin: got %q, want %q", got, "hello")
	}
}

func TestExecOption_WithMaxOutput(t *testing.T) {
	_, _, _, _, got := sandbox.ExportApplyExecOptions(sandbox.WithMaxOutput(1024))
	if got != 1024 {
		t.Fatalf("WithMaxOutput: got %d, want %d", got, 1024)
	}
}

func TestExecOption_Combined(t *testing.T) {
	env := map[string]string{"K": "V"}
	dir, gotEnv, timeout, stdin, maxOut := sandbox.ExportApplyExecOptions(
		sandbox.WithWorkDir("/w"),
		sandbox.WithEnv(env),
		sandbox.WithTimeout(10*time.Second),
		sandbox.WithStdin([]byte("in")),
		sandbox.WithMaxOutput(2048),
	)
	if dir != "/w" {
		t.Errorf("WorkDir: got %q", dir)
	}
	if gotEnv["K"] != "V" {
		t.Errorf("Env: got %v", gotEnv)
	}
	if timeout != int64(10*time.Second) {
		t.Errorf("Timeout: got %d", timeout)
	}
	if string(stdin) != "in" {
		t.Errorf("Stdin: got %q", stdin)
	}
	if maxOut != 2048 {
		t.Errorf("MaxOutput: got %d", maxOut)
	}
}

func TestExecOption_LastWins(t *testing.T) {
	dir, _, _, _, _ := sandbox.ExportApplyExecOptions(
		sandbox.WithWorkDir("/first"),
		sandbox.WithWorkDir("/second"),
	)
	if dir != "/second" {
		t.Fatalf("last-wins: got %q, want %q", dir, "/second")
	}
}

func TestAttachOption_WithProtocol(t *testing.T) {
	proto, _, _ := sandbox.ExportApplyAttachOptions(sandbox.WithProtocol("websocket"))
	if proto != "websocket" {
		t.Fatalf("WithProtocol: got %q, want %q", proto, "websocket")
	}
}

func TestAttachOption_WithPath(t *testing.T) {
	_, path, _ := sandbox.ExportApplyAttachOptions(sandbox.WithPath("/api/v1"))
	if path != "/api/v1" {
		t.Fatalf("WithPath: got %q, want %q", path, "/api/v1")
	}
}

func TestAttachOption_WithHeaders(t *testing.T) {
	h := map[string]string{"Authorization": "Bearer tok"}
	_, _, got := sandbox.ExportApplyAttachOptions(sandbox.WithHeaders(h))
	if got["Authorization"] != "Bearer tok" {
		t.Fatalf("WithHeaders: got %v", got)
	}
}

func TestAttachOption_Combined(t *testing.T) {
	h := map[string]string{"X-Key": "val"}
	proto, path, headers := sandbox.ExportApplyAttachOptions(
		sandbox.WithProtocol("tcp"),
		sandbox.WithPath("/ws"),
		sandbox.WithHeaders(h),
	)
	if proto != "tcp" || path != "/ws" || headers["X-Key"] != "val" {
		t.Fatalf("Combined: proto=%q path=%q headers=%v", proto, path, headers)
	}
}

func TestLifecyclePolicyConstants(t *testing.T) {
	cases := []struct {
		got  sandbox.LifecyclePolicy
		want string
	}{
		{sandbox.OneShot, "oneshot"},
		{sandbox.Session, "session"},
		{sandbox.LongRunning, "longrunning"},
		{sandbox.Persistent, "persistent"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("LifecyclePolicy: got %q, want %q", tc.got, tc.want)
		}
	}
}

func TestDefaultTimeouts(t *testing.T) {
	if sandbox.DefaultStopTimeout != 2*time.Second {
		t.Errorf("DefaultStopTimeout: got %v", sandbox.DefaultStopTimeout)
	}
	if sandbox.DefaultSessionIdleTimeout != 30*time.Minute {
		t.Errorf("DefaultSessionIdleTimeout: got %v", sandbox.DefaultSessionIdleTimeout)
	}
	if sandbox.DefaultLongRunningIdleTimeout != 2*time.Hour {
		t.Errorf("DefaultLongRunningIdleTimeout: got %v", sandbox.DefaultLongRunningIdleTimeout)
	}
	if sandbox.DefaultOneShotMaxAge != 8*time.Hour {
		t.Errorf("DefaultOneShotMaxAge: got %v", sandbox.DefaultOneShotMaxAge)
	}
}
