package sandboxv2_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

// ---------------------------------------------------------------------------
// Box tests (local + remote)
// ---------------------------------------------------------------------------

func TestRunPrepareSteps_Exec(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "exec", Cmd: "echo hello > /tmp/prep-test"},
				{Action: "exec", Cmd: "echo world >> /tmp/prep-test"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, box, "test-assistant", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps: %v", err)
			}

			result, err := box.Exec(ctx, []string{"cat", "/tmp/prep-test"})
			if err != nil {
				t.Fatalf("cat: %v", err)
			}
			got := strings.TrimSpace(result.Stdout)
			if got != "hello\nworld" {
				t.Errorf("content = %q, want %q", got, "hello\nworld")
			}
		})
	}
}

func TestRunPrepareSteps_File(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)
			wsID := fmt.Sprintf("test-file-%d", time.Now().UnixNano())
			box.BindWorkplace(wsID)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "file", Path: "config/test.txt", Content: []byte("file-content-v2")},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, box, "test-assistant", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps: %v", err)
			}

			ws := box.Workplace()
			data, err := ws.ReadFile("config/test.txt")
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(data) != "file-content-v2" {
				t.Errorf("content = %q, want %q", string(data), "file-content-v2")
			}
		})
	}
}

func TestRunPrepareSteps_Copy(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)
			wsID := fmt.Sprintf("test-copy-%d", time.Now().UnixNano())
			box.BindWorkplace(wsID)

			ws := box.Workplace()
			ws.WriteFile("src.txt", []byte("copy-src"), 0644)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "copy", Src: "src.txt", Dst: "dst.txt"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, box, "test-assistant", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps: %v", err)
			}

			data, err := ws.ReadFile("dst.txt")
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(data) != "copy-src" {
				t.Errorf("content = %q, want %q", string(data), "copy-src")
			}
		})
	}
}

func TestRunPrepareSteps_OnceMarker(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)
			wsID := fmt.Sprintf("test-once-%d", time.Now().UnixNano())
			box.BindWorkplace(wsID)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			counter := "/tmp/once-counter"
			steps := []types.PrepareStep{
				{Action: "exec", Cmd: "echo -n x >> " + counter, Once: true},
			}

			hash := "abc123"
			assistantID := "test-once"

			if err := sandboxv2.RunPrepareSteps(ctx, steps, box, assistantID, hash, ""); err != nil {
				t.Fatalf("first run: %v", err)
			}
			r1, _ := box.Exec(ctx, []string{"cat", counter})
			if r1.Stdout != "x" {
				t.Fatalf("first run: got %q, want %q", r1.Stdout, "x")
			}

			if err := sandboxv2.RunPrepareSteps(ctx, steps, box, assistantID, hash, ""); err != nil {
				t.Fatalf("second run: %v", err)
			}
			r2, _ := box.Exec(ctx, []string{"cat", counter})
			if r2.Stdout != "x" {
				t.Errorf("second run: got %q, want %q (once step should be skipped)", r2.Stdout, "x")
			}

			if err := sandboxv2.RunPrepareSteps(ctx, steps, box, assistantID, "new-hash", ""); err != nil {
				t.Fatalf("third run: %v", err)
			}
			r3, _ := box.Exec(ctx, []string{"cat", counter})
			if r3.Stdout != "xx" {
				t.Errorf("third run: got %q, want %q (hash changed, should re-execute)", r3.Stdout, "xx")
			}
		})
	}
}

func TestRunPrepareSteps_OnceIsolation(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)
			wsID := fmt.Sprintf("test-iso-%d", time.Now().UnixNano())
			box.BindWorkplace(wsID)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			stepsA := []types.PrepareStep{
				{Action: "exec", Cmd: "echo -n A >> /tmp/iso-a", Once: true},
			}
			stepsB := []types.PrepareStep{
				{Action: "exec", Cmd: "echo -n B >> /tmp/iso-b", Once: true},
			}

			hash := "same-hash"

			if err := sandboxv2.RunPrepareSteps(ctx, stepsA, box, "assistant-a", hash, ""); err != nil {
				t.Fatalf("assistant-a: %v", err)
			}
			if err := sandboxv2.RunPrepareSteps(ctx, stepsB, box, "assistant-b", hash, ""); err != nil {
				t.Fatalf("assistant-b: %v", err)
			}

			rA, _ := box.Exec(ctx, []string{"cat", "/tmp/iso-a"})
			rB, _ := box.Exec(ctx, []string{"cat", "/tmp/iso-b"})
			if rA.Stdout != "A" {
				t.Errorf("assistant-a: got %q, want %q", rA.Stdout, "A")
			}
			if rB.Stdout != "B" {
				t.Errorf("assistant-b: got %q, want %q", rB.Stdout, "B")
			}

			if err := sandboxv2.RunPrepareSteps(ctx, stepsA, box, "assistant-a", hash, ""); err != nil {
				t.Fatalf("assistant-a re-run: %v", err)
			}
			if err := sandboxv2.RunPrepareSteps(ctx, stepsB, box, "assistant-b", hash, ""); err != nil {
				t.Fatalf("assistant-b re-run: %v", err)
			}
			rA2, _ := box.Exec(ctx, []string{"cat", "/tmp/iso-a"})
			rB2, _ := box.Exec(ctx, []string{"cat", "/tmp/iso-b"})
			if rA2.Stdout != "A" {
				t.Errorf("assistant-a re-run: got %q, want %q (should be skipped)", rA2.Stdout, "A")
			}
			if rB2.Stdout != "B" {
				t.Errorf("assistant-b re-run: got %q, want %q (should be skipped)", rB2.Stdout, "B")
			}
		})
	}
}

func TestRunPrepareSteps_IgnoreError(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "exec", Cmd: "false", IgnoreError: true},
				{Action: "exec", Cmd: "echo survived > /tmp/survived"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, box, "test-assistant", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps: %v (ignore_error should have prevented failure)", err)
			}

			result, _ := box.Exec(ctx, []string{"cat", "/tmp/survived"})
			if strings.TrimSpace(result.Stdout) != "survived" {
				t.Errorf("second step should have executed, got %q", result.Stdout)
			}
		})
	}
}

func TestRunPrepareSteps_FailOnError(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "exec", Cmd: "false"},
				{Action: "exec", Cmd: "echo should-not-reach > /tmp/unreachable"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, box, "test-assistant", "", "")
			if err == nil {
				t.Fatal("expected error from failing step without ignore_error")
			}

			result, _ := box.Exec(ctx, []string{"cat", "/tmp/unreachable"})
			if result.ExitCode == 0 {
				t.Error("second step should not have executed")
			}
		})
	}
}

func TestRunPrepareSteps_UnknownAction(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			_ = createBox(t, m, nc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "unknown_action"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, nil, "test-assistant", "", "")
			if err == nil {
				t.Fatal("expected error for unknown action")
			}
			if !strings.Contains(err.Error(), "unknown_action") {
				t.Errorf("error should mention action name, got: %v", err)
			}
		})
	}
}

func TestRunPrepareSteps_EmptySteps(t *testing.T) {
	err := sandboxv2.RunPrepareSteps(context.Background(), nil, nil, "test-assistant", "hash", "")
	if err != nil {
		t.Fatalf("empty steps should succeed: %v", err)
	}
}

func TestRunPrepareSteps_Background(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "exec", Cmd: "sleep 30", Background: true},
				{Action: "exec", Cmd: "echo after-bg > /tmp/after-bg"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, box, "test-assistant", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps: %v", err)
			}

			result, _ := box.Exec(ctx, []string{"cat", "/tmp/after-bg"})
			if strings.TrimSpace(result.Stdout) != "after-bg" {
				t.Errorf("background step blocked execution, got %q", result.Stdout)
			}
		})
	}
}

func TestRunPrepareSteps_MixedActions(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			box := createBox(t, m, nc)
			wsID := fmt.Sprintf("test-mixed-%d", time.Now().UnixNano())
			box.BindWorkplace(wsID)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "file", Path: "mixed.conf", Content: []byte("key=value")},
				{Action: "exec", Cmd: "echo exec-ok > /tmp/mixed-exec"},
				{Action: "copy", Src: "mixed.conf", Dst: "mixed-copy.conf"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, box, "test-assistant", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps: %v", err)
			}

			ws := box.Workplace()
			data, err := ws.ReadFile("mixed-copy.conf")
			if err != nil {
				t.Fatalf("ReadFile mixed-copy.conf: %v", err)
			}
			if string(data) != "key=value" {
				t.Errorf("copy result: got %q, want %q", string(data), "key=value")
			}

			result, _ := box.Exec(ctx, []string{"cat", "/tmp/mixed-exec"})
			if strings.TrimSpace(result.Stdout) != "exec-ok" {
				t.Errorf("exec result: got %q, want %q", result.Stdout, "exec-ok")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HostExec tests
// ---------------------------------------------------------------------------

func TestRunPrepareSteps_HostExec(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)
			host := createHost(t, m, tgt)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Logf("SystemInfo: OS=%q Shell=%q TempDir=%q",
				host.ComputerInfo().System.OS,
				host.ComputerInfo().System.Shell,
				host.ComputerInfo().System.TempDir)

			isWin := tgt.Name == "win-native"
			var cmd string
			if isWin {
				cmd = `Write-Output 'host-ok'`
			} else {
				cmd = "echo host-ok"
			}
			steps := []types.PrepareStep{
				{Action: "exec", Cmd: cmd},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, host, "test-host", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps on host: %v", err)
			}
		})
	}
}

func TestRunPrepareSteps_HostExecFile(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)
			host := createHost(t, m, tgt)
			wsID := fmt.Sprintf("test-hostfile-%d", time.Now().UnixNano())
			host.BindWorkplace(wsID)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "file", Path: "host-test.txt", Content: []byte("host-file-data")},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, host, "test-host", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps file: %v", err)
			}

			ws := host.Workplace()
			data, err := ws.ReadFile("host-test.txt")
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(data) != "host-file-data" {
				t.Errorf("content = %q, want %q", string(data), "host-file-data")
			}
		})
	}
}

func TestRunPrepareSteps_HostExecCopy(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)
			host := createHost(t, m, tgt)
			wsID := fmt.Sprintf("test-hostcopy-%d", time.Now().UnixNano())
			host.BindWorkplace(wsID)

			ws := host.Workplace()
			ws.WriteFile("copy-src.txt", []byte("copy-data"), 0644)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			steps := []types.PrepareStep{
				{Action: "copy", Src: "copy-src.txt", Dst: "copy-dst.txt"},
			}

			err := sandboxv2.RunPrepareSteps(ctx, steps, host, "test-host", "", "")
			if err != nil {
				t.Fatalf("RunPrepareSteps copy: %v", err)
			}

			data, err := ws.ReadFile("copy-dst.txt")
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(data) != "copy-data" {
				t.Errorf("content = %q, want %q", string(data), "copy-data")
			}
		})
	}
}

func TestRunPrepareSteps_HostExecOnce(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)
			host := createHost(t, m, tgt)
			wsID := fmt.Sprintf("test-hostonce-%d", time.Now().UnixNano())
			host.BindWorkplace(wsID)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			isWin := tgt.Name == "win-native"
			var cmd string
			if isWin {
				cmd = `Write-Output 'once-ok'`
			} else {
				cmd = "echo once-ok"
			}
			steps := []types.PrepareStep{
				{Action: "exec", Cmd: cmd, Once: true},
			}
			hash := "host-once-hash"
			aid := "host-once-aid"

			if err := sandboxv2.RunPrepareSteps(ctx, steps, host, aid, hash, ""); err != nil {
				t.Fatalf("first run: %v", err)
			}

			ws := host.Workplace()
			markerData, err := ws.ReadFile(".yao/prepare/" + aid + "/done")
			if err != nil {
				t.Fatalf("marker not written: %v", err)
			}
			if string(markerData) != hash {
				t.Errorf("marker = %q, want %q", string(markerData), hash)
			}
		})
	}
}
