package sandbox_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
)

func setupHostManager(t *testing.T, tgt hostExecTarget) *sandbox.Manager {
	t.Helper()
	addr := fmt.Sprintf("tai://%s", tgt.Addr)
	pool := sandbox.Pool{Name: tgt.Name, Addr: addr}
	cfg := sandbox.Config{Pool: []sandbox.Pool{pool}}
	if err := sandbox.Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m := sandbox.M()
	t.Cleanup(func() { m.Close() })
	return m
}

func TestHost_Exec_Echo(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, tgt)

			host, err := m.Host(context.Background(), tgt.Name)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd, args := linuxCmd(tgt, "echo", "hello", "from", "host")
			result, err := host.Exec(ctx, cmd, args)
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if result.Error != "" {
				if strings.Contains(result.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("error: %s", result.Error)
			}
			if result.ExitCode != 0 {
				t.Errorf("exit_code = %d, want 0", result.ExitCode)
			}
			got := strings.TrimSpace(string(result.Stdout))
			if !strings.Contains(got, "hello") {
				t.Errorf("stdout = %q, want contains 'hello'", got)
			}
		})
	}
}

func TestHost_Exec_Env(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, tgt)

			host, err := m.Host(context.Background(), tgt.Name)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			var cmd string
			var args []string
			if tgt.IsWinNative {
				cmd = "cmd.exe"
				args = []string{"/c", "echo", "%MY_VAR%"}
			} else {
				cmd = "sh"
				args = []string{"-c", "echo $MY_VAR"}
			}

			result, err := host.Exec(ctx, cmd, args, sandbox.WithHostEnv(map[string]string{"MY_VAR": "host_test_value"}))
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if result.Error != "" {
				if strings.Contains(result.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("error: %s", result.Error)
			}
			got := strings.TrimSpace(string(result.Stdout))
			if !strings.Contains(got, "host_test_value") {
				t.Errorf("stdout = %q, want contains 'host_test_value'", got)
			}
		})
	}
}

func TestHost_Workspace(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, tgt)

			host, err := m.Host(context.Background(), tgt.Name)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			sessionID := fmt.Sprintf("host-test-%d", time.Now().UnixNano())
			ws := host.Workspace(sessionID)
			if ws == nil {
				t.Fatal("Workspace returned nil")
			}

			content := []byte("hello from host workspace test")
			if err := ws.WriteFile("test.txt", content, 0644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			got, err := ws.ReadFile("test.txt")
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(got) != string(content) {
				t.Errorf("ReadFile = %q, want %q", got, content)
			}

			if err := ws.MkdirAll("sub/dir", 0755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}
			if err := ws.WriteFile("sub/dir/nested.txt", []byte("nested"), 0644); err != nil {
				t.Fatalf("WriteFile nested: %v", err)
			}

			entries, err := ws.ReadDir("sub/dir")
			if err != nil {
				t.Fatalf("ReadDir: %v", err)
			}
			if len(entries) != 1 {
				t.Errorf("ReadDir len = %d, want 1", len(entries))
			}

			if err := ws.RemoveAll(sessionID); err != nil && !strings.Contains(err.Error(), "not found") {
				t.Logf("cleanup RemoveAll: %v", err)
			}
		})
	}
}

func TestHost_Stream_Incremental(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		if tgt.IsWinNative {
			continue
		}
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, tgt)

			host, err := m.Host(context.Background(), tgt.Name)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, "sh", []string{"-c",
				"for i in 1 2 3 4 5; do echo chunk$i; sleep 0.2; done"})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			var chunks []string
			for chunk := range stream.Stdout {
				chunks = append(chunks, string(chunk))
			}

			exitCode, err := stream.Wait()
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				if strings.Contains(err.Error(), "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("Wait: %v", err)
			}
			if exitCode != 0 {
				t.Errorf("exit_code = %d, want 0", exitCode)
			}

			combined := strings.Join(chunks, "")
			for _, expect := range []string{"chunk1", "chunk3", "chunk5"} {
				if !strings.Contains(combined, expect) {
					t.Errorf("output = %q, want contains %q", combined, expect)
				}
			}

			if len(chunks) < 2 {
				t.Errorf("received %d chunks, want >= 2 (proves streaming, not buffered)", len(chunks))
			}
			t.Logf("received %d chunks over stream", len(chunks))
		})
	}
}

func TestHost_Stream_MultiLine(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		if tgt.IsWinNative {
			continue
		}
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, tgt)

			host, err := m.Host(context.Background(), tgt.Name)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, "sh", []string{"-c", "for i in 1 2 3; do echo line$i; done"})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			var stdout []byte
			for chunk := range stream.Stdout {
				stdout = append(stdout, chunk...)
			}

			exitCode, err := stream.Wait()
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				if strings.Contains(err.Error(), "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("Wait: %v", err)
			}
			if exitCode != 0 {
				t.Errorf("exit_code = %d, want 0", exitCode)
			}
			got := strings.TrimSpace(string(stdout))
			for _, expect := range []string{"line1", "line2", "line3"} {
				if !strings.Contains(got, expect) {
					t.Errorf("stdout = %q, want contains %q", got, expect)
				}
			}
		})
	}
}

func TestHost_Stream_Stderr(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		if tgt.IsWinNative {
			continue
		}
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, tgt)

			host, err := m.Host(context.Background(), tgt.Name)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, "sh", []string{"-c", "echo err-msg >&2"})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			var stderr []byte
			done := make(chan struct{})
			go func() {
				for chunk := range stream.Stdout {
					_ = chunk
				}
				close(done)
			}()
			for chunk := range stream.Stderr {
				stderr = append(stderr, chunk...)
			}
			<-done

			exitCode, err := stream.Wait()
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				if strings.Contains(err.Error(), "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("Wait: %v", err)
			}
			if exitCode != 0 {
				t.Errorf("exit_code = %d, want 0", exitCode)
			}
			got := strings.TrimSpace(string(stderr))
			if !strings.Contains(got, "err-msg") {
				t.Errorf("stderr = %q, want contains 'err-msg'", got)
			}
		})
	}
}

func TestHost_Stream_Cancel(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		if tgt.IsWinNative {
			continue
		}
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, tgt)

			host, err := m.Host(context.Background(), tgt.Name)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, "sh", []string{"-c", "while true; do echo tick; sleep 0.1; done"})
			if err != nil {
				if strings.Contains(err.Error(), "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("Stream: %v", err)
			}

			received := 0
			for chunk := range stream.Stdout {
				_ = chunk
				received++
				if received >= 3 {
					stream.Cancel()
					break
				}
			}

			_, waitErr := stream.Wait()
			if waitErr != nil && strings.Contains(waitErr.Error(), "not in the allowed list") {
				t.Skipf("command not allowed on %s", tgt.Name)
			}
			if received < 3 && waitErr == nil {
				t.Errorf("received %d chunks before cancel, want >= 3", received)
			}
		})
	}
}

func TestHost_CreateRejectsNoContainerPool(t *testing.T) {
	// Use the Windows native HostExec target which has no Docker.
	tgt := findHostExecOnly(t)
	if tgt == nil {
		t.Skip("no host-exec-only target available")
	}

	m := setupHostManager(t, *tgt)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := m.Create(ctx, sandbox.CreateOptions{
		Image: "alpine:latest",
		Owner: "test",
		Pool:  tgt.Name,
	})
	if err == nil {
		t.Fatal("expected error for Create on host-exec-only pool, got nil")
	}
	if !strings.Contains(err.Error(), "no container runtime") {
		t.Errorf("error = %q, want contains 'no container runtime'", err.Error())
	}
}

func TestHost_PoolNotFound(t *testing.T) {
	skipIfNoHostExec(t)
	tgt := hostExecTargets()[0]
	m := setupHostManager(t, tgt)

	_, err := m.Host(context.Background(), "nonexistent-pool")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// findHostExecOnly returns a hostExecTarget that is likely host-exec-only
// (Windows native Tai without Docker).
func findHostExecOnly(t *testing.T) *hostExecTarget {
	t.Helper()
	for _, tgt := range hostExecTargets() {
		if tgt.IsWinNative {
			// Windows native Tai typically has no Docker
			addr := fmt.Sprintf("tai://%s", tgt.Addr)
			client, err := tai.New(addr)
			if err != nil {
				continue
			}
			hasNoSandbox := client.Sandbox() == nil
			client.Close()
			if hasNoSandbox {
				return &tgt
			}
		}
	}
	return nil
}
