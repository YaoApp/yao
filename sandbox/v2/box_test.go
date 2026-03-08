package sandbox_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestBoxExec(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := box.Exec(ctx, []string{"echo", "box-exec"})
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if result.Stdout != "box-exec\n" {
				t.Errorf("stdout = %q, want %q", result.Stdout, "box-exec\n")
			}
		})
	}
}

func TestBoxExecWithOptions(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			ctx := context.Background()
			result, err := box.Exec(ctx, []string{"pwd"},
				sandbox.WithWorkDir("/tmp"),
			)
			if err != nil {
				t.Fatalf("Exec: %v", err)
			}
			if result.Stdout != "/tmp\n" {
				t.Errorf("stdout = %q, want %q", result.Stdout, "/tmp\n")
			}
		})
	}
}

func TestBoxStream(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			ctx := context.Background()
			stream, err := box.Stream(ctx, []string{"sh", "-c", "echo line1; echo line2"})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			out, err := io.ReadAll(stream.Stdout)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if string(out) != "line1\nline2\n" {
				t.Errorf("stdout = %q, want %q", string(out), "line1\nline2\n")
			}

			code, err := stream.Wait()
			if err != nil {
				t.Fatalf("Wait: %v", err)
			}
			if code != 0 {
				t.Errorf("exit code = %d, want 0", code)
			}
		})
	}
}

func TestBoxWorkspace(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			ws := box.Workspace()
			if ws == nil {
				t.Skip("Workspace returned nil (volume not available)")
			}

			content := []byte("package main\n")
			if err := ws.WriteFile("main.go", content, 0644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			data, err := ws.ReadFile("main.go")
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(data) != string(content) {
				t.Errorf("content = %q, want %q", string(data), string(content))
			}

			if err := ws.MkdirAll("src/pkg", 0755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}

			entries, err := ws.ReadDir("src")
			if err != nil {
				t.Fatalf("ReadDir: %v", err)
			}
			if len(entries) == 0 {
				t.Error("expected non-empty directory listing")
			}
		})
	}
}

func TestBoxInfo(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			ctx := context.Background()
			info, err := box.Info(ctx)
			if err != nil {
				t.Fatalf("Info: %v", err)
			}
			if info.ID != box.ID() {
				t.Errorf("ID = %q, want %q", info.ID, box.ID())
			}
			if s := strings.ToLower(info.Status); s != "running" {
				t.Errorf("status = %q, want running", info.Status)
			}
			if info.Owner != "test-user" {
				t.Errorf("owner = %q, want test-user", info.Owner)
			}
		})
	}
}

func TestBoxStopStart(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)
			ctx := context.Background()

			if err := box.Stop(ctx); err != nil {
				t.Fatalf("Stop: %v", err)
			}

			if err := box.Start(ctx); err != nil {
				t.Fatalf("Start: %v", err)
			}

			result, err := box.Exec(ctx, []string{"echo", "after-restart"})
			if err != nil {
				t.Fatalf("Exec after restart: %v", err)
			}
			if result.Stdout != "after-restart\n" {
				t.Errorf("stdout = %q", result.Stdout)
			}
		})
	}
}

func TestBoxGetOrCreate(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			ctx := context.Background()
			box1, err := m.GetOrCreate(ctx, sandbox.CreateOptions{
				ID:     "goc-" + pc.Name,
				Image:  testImage(),
				Owner:  "test-user",
				NodeID: pc.TaiID,
			})
			if err != nil {
				t.Fatalf("GetOrCreate first: %v", err)
			}
			defer m.Remove(ctx, box1.ID())

			box2, err := m.GetOrCreate(ctx, sandbox.CreateOptions{
				ID:     "goc-" + pc.Name,
				Image:  testImage(),
				Owner:  "test-user",
				NodeID: pc.TaiID,
			})
			if err != nil {
				t.Fatalf("GetOrCreate second: %v", err)
			}
			if box2.ContainerID() != box1.ContainerID() {
				t.Error("expected same container for GetOrCreate with same ID")
			}
		})
	}
}
