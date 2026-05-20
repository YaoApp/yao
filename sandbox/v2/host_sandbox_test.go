package sandbox_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func TestHost_Exec(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	host := sandboxtest.CreateHost(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd, args := sandboxtest.TranslateCmd(false, "echo", "hello-host")
	res, err := host.Exec(ctx, append([]string{cmd}, args...))
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "hello-host") {
		t.Fatalf("expected 'hello-host' in stdout, got %q", res.Stdout)
	}
}

func TestHost_ExecEnv(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	host := sandboxtest.CreateHost(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd, args := sandboxtest.TranslateCmd(false, "sh", "-c", "echo $TEST_HOST_VAR")
	res, err := host.Exec(ctx, append([]string{cmd}, args...), sandbox.WithEnv(map[string]string{"TEST_HOST_VAR": "host_value"}))
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !strings.Contains(res.Stdout, "host_value") {
		t.Fatalf("expected 'host_value' in stdout, got %q", res.Stdout)
	}
}

func TestHost_Stream(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	host := sandboxtest.CreateHost(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd, args := sandboxtest.TranslateCmd(false, "echo", "stream-host")
	stream, err := host.Stream(ctx, append([]string{cmd}, args...))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer stream.Cancel()

	out, err := io.ReadAll(stream.Stdout)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	exitCode, err := stream.Wait()
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exit code %d", exitCode)
	}
	if !strings.Contains(string(out), "stream-host") {
		t.Fatalf("expected 'stream-host' in output, got %q", string(out))
	}
}

func TestHost_StreamCancel(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	host := sandboxtest.CreateHost(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd, args := sandboxtest.TranslateCmd(false, "sleep", "30")
	stream, err := host.Stream(ctx, append([]string{cmd}, args...))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	stream.Cancel()
	_, _ = stream.Wait()
}

func TestHost_ComputerInfo(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	host := sandboxtest.CreateHost(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr()))

	info := host.ComputerInfo()
	if info.Kind != "host" {
		t.Fatalf("expected Kind='host', got %q", info.Kind)
	}
	if info.NodeID == "" {
		t.Fatal("expected non-empty NodeID")
	}
}

func TestHost_BindWorkplace(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	host := sandboxtest.CreateHost(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr()))

	host.BindWorkplace("test-workspace-id")
	wp := host.Workplace()
	_ = wp // Workplace may be nil if workspace not created; validate no panic
}
