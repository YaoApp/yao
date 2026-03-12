package sandbox_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func setupHostManager(t *testing.T, tgt *hostExecTarget) *sandbox.Manager {
	t.Helper()
	addr := fmt.Sprintf("tai://%s", tgt.Addr)
	m, nodes := setupManager(t, nodeConfig{Name: tgt.Name, Addr: addr})
	tgt.TaiID = nodes[0].TaiID
	return m
}

func TestHost_Exec_Echo(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd := hostCmd(tgt, "echo", "hello", "from", "host")
			result, err := host.Exec(ctx, cmd)
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
			got := strings.TrimSpace(result.Stdout)
			if !strings.Contains(got, "hello") {
				t.Errorf("stdout = %q, want contains 'hello'", got)
			}
		})
	}
}

func TestHost_Exec_Env(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			var cmd []string
			if tgt.IsWinNative {
				cmd = []string{"cmd.exe", "/c", "echo", "%MY_VAR%"}
			} else {
				cmd = []string{"sh", "-c", "echo $MY_VAR"}
			}

			result, err := host.Exec(ctx, cmd, sandbox.WithEnv(map[string]string{"MY_VAR": "host_test_value"}))
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if result.Error != "" {
				if strings.Contains(result.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("error: %s", result.Error)
			}
			got := strings.TrimSpace(result.Stdout)
			if !strings.Contains(got, "host_test_value") {
				t.Errorf("stdout = %q, want contains 'host_test_value'", got)
			}
		})
	}
}

func TestHost_Workplace(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			sessionID := fmt.Sprintf("host-test-%d", time.Now().UnixNano())
			host.BindWorkplace(sessionID)
			ws := host.Workplace()
			if ws == nil {
				t.Fatal("Workplace returned nil after BindWorkplace")
			}

			content := []byte("hello from host workplace test")
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
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, []string{"sh", "-c",
				"for i in 1 2 3 4 5; do echo chunk$i; sleep 0.2; done"})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			var chunks []string
			buf := make([]byte, 4096)
			for {
				n, err := stream.Stdout.Read(buf)
				if n > 0 {
					chunks = append(chunks, string(buf[:n]))
				}
				if err != nil {
					break
				}
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
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, []string{"sh", "-c", "for i in 1 2 3; do echo line$i; done"})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			stdout, _ := io.ReadAll(stream.Stdout)

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
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, []string{"sh", "-c", "echo err-msg >&2"})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			done := make(chan struct{})
			go func() {
				io.ReadAll(stream.Stdout)
				close(done)
			}()
			stderr, _ := io.ReadAll(stream.Stderr)
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
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := host.Stream(ctx, []string{"sh", "-c", "while true; do echo tick; sleep 0.1; done"})
			if err != nil {
				if strings.Contains(err.Error(), "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("Stream: %v", err)
			}

			received := 0
			buf := make([]byte, 4096)
			for {
				n, err := stream.Stdout.Read(buf)
				if n > 0 {
					received++
				}
				if received >= 3 {
					stream.Cancel()
					break
				}
				if err != nil {
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

func TestHost_ComputerInfo(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			info := host.ComputerInfo()
			if info.Kind != "host" {
				t.Errorf("Kind = %q, want 'host'", info.Kind)
			}
			if info.NodeID != tgt.TaiID {
				t.Errorf("NodeID = %q, want %q", info.NodeID, tgt.TaiID)
			}
		})
	}
}

func TestHost_ComputerInterface(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			host, err := m.Host(context.Background(), tgt.TaiID)
			if err != nil {
				t.Skipf("Host(%s): %v", tgt.Name, err)
			}

			// Verify Host satisfies Computer interface at runtime.
			var c sandbox.Computer = host
			info := c.ComputerInfo()
			if info.Kind != "host" {
				t.Errorf("Computer.ComputerInfo().Kind = %q, want 'host'", info.Kind)
			}
		})
	}
}

func TestHost_CreateRejectsNoContainerNode(t *testing.T) {
	tgt := findHostExecOnly(t)
	if tgt == nil {
		t.Skip("no host-exec-only target available")
	}

	m := setupHostManager(t, tgt)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := m.Create(ctx, sandbox.CreateOptions{
		Image:  "alpine:latest",
		Owner:  "test",
		NodeID: tgt.TaiID,
	})
	if err == nil {
		t.Fatal("expected error for Create on host-exec-only node, got nil")
	}
	if !strings.Contains(err.Error(), "no container runtime") {
		t.Errorf("error = %q, want contains 'no container runtime'", err.Error())
	}
}

func TestHost_NodeNotFound(t *testing.T) {
	skipIfNoHostExec(t)
	tgt := hostExecTargets()[0]
	m := setupHostManager(t, &tgt)

	_, err := m.Host(context.Background(), "nonexistent-node")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func findHostExecOnly(t *testing.T) *hostExecTarget {
	t.Helper()
	for _, tgt := range hostExecTargets() {
		if tgt.IsWinNative {
			addr := fmt.Sprintf("tai://%s", tgt.Addr)
			res, err := dialForTest(addr)
			if err != nil {
				continue
			}
			hasNoSandbox := res.Runtime == nil
			res.Close()
			if hasNoSandbox {
				return &tgt
			}
		}
	}
	return nil
}

// hostCmd builds a []string command, adapting for Windows targets.
func hostCmd(tgt hostExecTarget, prog string, args ...string) []string {
	if tgt.IsWinNative {
		cmd, wArgs := linuxCmd(tgt, prog, args...)
		return append([]string{cmd}, wArgs...)
	}
	return append([]string{prog}, args...)
}
