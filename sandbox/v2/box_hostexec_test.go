package sandbox_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	hepb "github.com/yaoapp/yao/tai/hostexec/pb"
)

func hostExecClient(t *testing.T, tgt hostExecTarget) hepb.HostExecClient {
	t.Helper()
	addr := fmt.Sprintf("tai://%s", tgt.Addr)
	client, err := tai.New(addr)
	if err != nil {
		t.Skipf("tai.New(%s): %v", addr, err)
		return nil
	}
	t.Cleanup(func() { client.Close() })
	he := client.HostExec()
	if he == nil {
		t.Skipf("hostexec not available on %s", tgt.Name)
		return nil
	}

	probeCmd, probeArgs := linuxCmd(tgt, "echo", "probe")
	probe, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = he.Exec(probe, &hepb.ExecRequest{Command: probeCmd, Args: probeArgs})
	if err != nil {
		client.Close()
		t.Skipf("hostexec on %s unreachable: %v", tgt.Name, err)
		return nil
	}
	return he
}

func linuxCmd(tgt hostExecTarget, cmd string, args ...string) (string, []string) {
	if tgt.IsWinNative {
		switch cmd {
		case "echo":
			return "cmd.exe", append([]string{"/c", "echo"}, args...)
		case "pwd":
			return "cmd.exe", []string{"/c", "cd"}
		case "env":
			return "cmd.exe", []string{"/c", "set"}
		case "sleep":
			return "cmd.exe", []string{"/c", "ping", "-n", "10", "127.0.0.1"}
		case "cat":
			return "cmd.exe", []string{"/c", "more"}
		case "sh":
			if len(args) >= 2 && args[0] == "-c" {
				return "cmd.exe", []string{"/c", args[1]}
			}
			return "cmd.exe", append([]string{"/c"}, args...)
		default:
			return cmd, args
		}
	}
	return cmd, args
}

func TestHostExec_Echo(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			he := hostExecClient(t, tgt)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd, args := linuxCmd(tgt, "echo", "hello", "from", "host")
			resp, err := he.Exec(ctx, &hepb.ExecRequest{Command: cmd, Args: args})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if resp.Error != "" {
				if strings.Contains(resp.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("error: %s", resp.Error)
			}
			if resp.ExitCode != 0 {
				t.Errorf("exit_code = %d, want 0", resp.ExitCode)
			}
			got := strings.TrimSpace(string(resp.Stdout))
			if !strings.Contains(got, "hello") {
				t.Errorf("stdout = %q, want contains 'hello'", got)
			}
		})
	}
}

func TestHostExec_Env(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			he := hostExecClient(t, tgt)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd, args := linuxCmd(tgt, "env")
			resp, err := he.Exec(ctx, &hepb.ExecRequest{Command: cmd, Args: args})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if resp.Error != "" {
				if strings.Contains(resp.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("error: %s", resp.Error)
			}
			out := string(resp.Stdout)
			if out == "" {
				t.Error("stdout is empty, expected environment variables")
			}
		})
	}
}

func TestHostExec_Timeout(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			he := hostExecClient(t, tgt)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd, args := linuxCmd(tgt, "sleep", "10")
			resp, err := he.Exec(ctx, &hepb.ExecRequest{
				Command:   cmd,
				Args:      args,
				TimeoutMs: 200,
			})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if resp.Error != "" && strings.Contains(resp.Error, "not in the allowed list") {
				t.Skipf("command not allowed on %s", tgt.Name)
			}
			if !strings.Contains(resp.Error, "timed out") {
				t.Errorf("error = %q, want contains 'timed out'", resp.Error)
			}
		})
	}
}

func TestHostExec_WorkingDir(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			he := hostExecClient(t, tgt)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd, args := linuxCmd(tgt, "pwd")
			workDir := "/tmp"
			if tgt.IsWinNative {
				workDir = "C:\\Windows\\Temp"
			}

			resp, err := he.Exec(ctx, &hepb.ExecRequest{
				Command:    cmd,
				Args:       args,
				WorkingDir: workDir,
			})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if resp.Error != "" {
				if strings.Contains(resp.Error, "not in") && strings.Contains(resp.Error, "allowed") {
					t.Skipf("working_dir not allowed on %s: %s", tgt.Name, resp.Error)
				}
				t.Fatalf("error: %s", resp.Error)
			}
			got := strings.TrimSpace(string(resp.Stdout))
			if got == "" {
				t.Error("stdout is empty")
			}
		})
	}
}

func TestHostExec_Stdin(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			he := hostExecClient(t, tgt)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd, args := linuxCmd(tgt, "cat")
			resp, err := he.Exec(ctx, &hepb.ExecRequest{
				Command: cmd,
				Args:    args,
				Stdin:   []byte("piped input"),
			})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if resp.Error != "" {
				if strings.Contains(resp.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("error: %s", resp.Error)
			}
			got := string(resp.Stdout)
			if !strings.Contains(got, "piped input") {
				t.Errorf("stdout = %q, want contains 'piped input'", got)
			}
		})
	}
}

func TestHostExec_NonZeroExit(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			he := hostExecClient(t, tgt)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			var cmd string
			var args []string
			if tgt.IsWinNative {
				cmd = "cmd.exe"
				args = []string{"/c", "exit", "42"}
			} else {
				cmd = "sh"
				args = []string{"-c", "exit 42"}
			}

			resp, err := he.Exec(ctx, &hepb.ExecRequest{Command: cmd, Args: args})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if resp.Error != "" {
				if strings.Contains(resp.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
			}
			if resp.ExitCode != 42 {
				t.Errorf("exit_code = %d, want 42", resp.ExitCode)
			}
		})
	}
}

func TestHostExec_UserEnv(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostExecTargets() {
		t.Run(tgt.Name, func(t *testing.T) {
			he := hostExecClient(t, tgt)
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

			resp, err := he.Exec(ctx, &hepb.ExecRequest{
				Command: cmd,
				Args:    args,
				Env:     map[string]string{"MY_VAR": "test_value"},
			})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if resp.Error != "" {
				if strings.Contains(resp.Error, "not in the allowed list") {
					t.Skipf("command not allowed on %s", tgt.Name)
				}
				t.Fatalf("error: %s", resp.Error)
			}
			got := strings.TrimSpace(string(resp.Stdout))
			if !strings.Contains(got, "test_value") {
				t.Errorf("stdout = %q, want contains 'test_value'", got)
			}
		})
	}
}

// TestHostExec_LocalUnavailable verifies ExecOnHost returns an error for local pools.
func TestHostExec_LocalUnavailable(t *testing.T) {
	skipIfNoDocker(t)

	m := setupManagerForPool(t, poolConfig{Name: "local", Addr: testLocalAddr()})
	box := createTestBox(t, m)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := box.ExecOnHost(ctx, "echo", []string{"should fail"})
	if err == nil {
		t.Fatal("expected error for local pool, got nil")
	}
	if !strings.Contains(err.Error(), "not available") {
		t.Errorf("error = %q, expected 'not available'", err.Error())
	}
}

// TestHostExec_BoxIntegration verifies ExecOnHost works through a sandbox Box
// (requires container creation — only tests pools with Docker/K8s support).
func TestHostExec_BoxIntegration(t *testing.T) {
	skipIfNoTai(t)

	for _, pc := range testPools() {
		if pc.Name == "local" {
			continue
		}
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			pool := pc.Name
			if err := m.EnsureImage(ctx, pool, testImage(), sandbox.ImagePullOptions{}); err != nil {
				t.Skipf("pool %s unavailable (image check): %v", pool, err)
			}
			box, err := m.Create(ctx, sandbox.CreateOptions{Image: testImage(), Owner: "test-user"})
			if err != nil {
				t.Skipf("pool %s unavailable (create): %v", pool, err)
			}
			t.Cleanup(func() { m.Remove(context.Background(), box.ID()) })

			result, err := box.ExecOnHost(ctx, "echo", []string{"box", "integration"})
			if err != nil {
				t.Skipf("ExecOnHost unavailable on pool %s: %v", pc.Name, err)
			}
			if result.Error != "" {
				if strings.Contains(result.Error, "not in the allowed list") {
					t.Skipf("echo not in allowed commands on pool %s", pc.Name)
				}
				t.Fatalf("hostexec error: %s", result.Error)
			}
			got := strings.TrimSpace(string(result.Stdout))
			if !strings.Contains(got, "box") || !strings.Contains(got, "integration") {
				t.Errorf("stdout = %q, want contains 'box integration'", got)
			}
		})
	}
}
