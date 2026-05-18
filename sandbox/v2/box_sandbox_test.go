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

func TestBox_Exec(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := box.Exec(ctx, []string{"echo", "hello"})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Fatalf("expected 'hello' in stdout, got %q", res.Stdout)
	}
}

func TestBox_ExecWithWorkDir(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := box.Exec(ctx, []string{"pwd"}, sandbox.WithWorkDir("/tmp"))
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !strings.Contains(res.Stdout, "/tmp") {
		t.Fatalf("expected /tmp in stdout, got %q", res.Stdout)
	}
}

func TestBox_ExecWithEnv(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := box.Exec(ctx, []string{"sh", "-c", "echo $MY_VAR"}, sandbox.WithEnv(map[string]string{"MY_VAR": "test_value"}))
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !strings.Contains(res.Stdout, "test_value") {
		t.Fatalf("expected 'test_value' in stdout, got %q", res.Stdout)
	}
}

func TestBox_Stream(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stream, err := box.Stream(ctx, []string{"echo", "streaming"})
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
	if !strings.Contains(string(out), "streaming") {
		t.Fatalf("expected 'streaming' in output, got %q", string(out))
	}
}

func TestBox_StreamCancel(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stream, err := box.Stream(ctx, []string{"sleep", "30"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	stream.Cancel()
	_, waitErr := stream.Wait()
	// Cancel should terminate; error is expected
	_ = waitErr
}

func TestBox_StreamStderr(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stream, err := box.Stream(ctx, []string{"sh", "-c", "echo errout >&2"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer stream.Cancel()

	stderr, err := io.ReadAll(stream.Stderr)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	stream.Wait()
	if !strings.Contains(string(stderr), "errout") {
		t.Fatalf("expected 'errout' in stderr, got %q", string(stderr))
	}
}

func TestBox_Info(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := box.Info(ctx)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil BoxInfo")
	}
}

func TestBox_StopAndStart(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	box := sandboxtest.CreateBox(t, m, sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr()))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := box.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := box.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	res, err := box.Exec(ctx, []string{"echo", "back"})
	if err != nil {
		t.Fatalf("Exec after restart: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code %d after restart", res.ExitCode)
	}
}
