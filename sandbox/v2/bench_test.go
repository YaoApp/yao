package sandbox_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/registry"
)

// BenchmarkContainerLifecycle measures the full Create → Exec → Remove cycle.
func BenchmarkContainerLifecycle(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForBench(b, &pc)
			ensureTestImageBench(b, m, pc.TaiID)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()

				box, err := m.Create(ctx, sandbox.CreateOptions{
					Image: testImage(),
					Owner: "bench",
				})
				if err != nil {
					b.Fatalf("Create: %v", err)
				}

				_, err = box.Exec(ctx, []string{"echo", "ok"})
				if err != nil {
					b.Fatalf("Exec: %v", err)
				}

				m.Remove(ctx, box.ID())
			}
		})
	}
}

// BenchmarkCreate measures container creation time only.
func BenchmarkCreate(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForBench(b, &pc)
			ensureTestImageBench(b, m, pc.TaiID)

			ids := make([]string, 0, b.N)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				box, err := m.Create(context.Background(), sandbox.CreateOptions{
					Image: testImage(),
					Owner: "bench",
				})
				if err != nil {
					b.Fatalf("Create: %v", err)
				}
				ids = append(ids, box.ID())
			}
			b.StopTimer()

			for _, id := range ids {
				m.Remove(context.Background(), id)
			}
		})
	}
}

// BenchmarkExec measures command execution latency on a pre-created container.
func BenchmarkExec(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForBench(b, &pc)
			box := createBoxForBench(b, m)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := box.Exec(context.Background(), []string{"echo", "bench"})
				if err != nil {
					b.Fatalf("Exec: %v", err)
				}
				if result.ExitCode != 0 {
					b.Fatalf("exit code = %d", result.ExitCode)
				}
			}
		})
	}
}

// BenchmarkExecHeavy measures execution of a heavier command (write + read file).
func BenchmarkExecHeavy(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForBench(b, &pc)
			box := createBoxForBench(b, m)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cmd := []string{"sh", "-c", fmt.Sprintf("echo bench-%d > /tmp/b.txt && cat /tmp/b.txt", i)}
				result, err := box.Exec(context.Background(), cmd)
				if err != nil {
					b.Fatalf("Exec: %v", err)
				}
				if result.ExitCode != 0 {
					b.Fatalf("exit code = %d", result.ExitCode)
				}
			}
		})
	}
}

// BenchmarkRemove measures container removal time.
func BenchmarkRemove(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForBench(b, &pc)
			ensureTestImageBench(b, m, pc.TaiID)

			boxes := make([]*sandbox.Box, b.N)
			for i := 0; i < b.N; i++ {
				box, err := m.Create(context.Background(), sandbox.CreateOptions{
					Image: testImage(),
					Owner: "bench",
				})
				if err != nil {
					b.Fatalf("Create: %v", err)
				}
				boxes[i] = box
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := m.Remove(context.Background(), boxes[i].ID()); err != nil {
					b.Fatalf("Remove: %v", err)
				}
			}
		})
	}
}

// BenchmarkInfo measures Info() latency on a running container.
func BenchmarkInfo(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForBench(b, &pc)
			box := createBoxForBench(b, m)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := box.Info(context.Background())
				if err != nil {
					b.Fatalf("Info: %v", err)
				}
			}
		})
	}
}

// BenchmarkStopStart measures Stop → Start cycle time.
func BenchmarkStopStart(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			if pc.Name == "k8s" {
				b.Skip("K8s Stop deletes Pod; Stop→Start cycle not applicable")
			}
			m := setupManagerForBench(b, &pc)
			box := createBoxForBench(b, m)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := box.Stop(context.Background()); err != nil {
					b.Fatalf("Stop: %v", err)
				}
				if err := box.Start(context.Background()); err != nil {
					b.Fatalf("Start: %v", err)
				}
			}
		})
	}
}

// BenchmarkWorkspaceReadWrite measures workspace file read/write via container Box.
func BenchmarkWorkspaceReadWrite(b *testing.B) {
	for _, pc := range testNodes() {
		pc := pc
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForBench(b, &pc)
			box := createBoxForBench(b, m)
			ws := box.Workspace()
			if ws == nil {
				b.Skip("workspace not available")
			}

			payload := []byte("package main\nfunc main() { println(\"hello\") }\n")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				name := fmt.Sprintf("f%d.go", i)
				if err := ws.WriteFile(name, payload, 0644); err != nil {
					b.Fatalf("WriteFile: %v", err)
				}
				data, err := ws.ReadFile(name)
				if err != nil {
					b.Fatalf("ReadFile: %v", err)
				}
				if len(data) != len(payload) {
					b.Fatalf("size mismatch: %d vs %d", len(data), len(payload))
				}
			}
		})
	}
}

// --- helpers ---

func setupManagerForBench(b *testing.B, pc *nodeConfig) *sandbox.Manager {
	b.Helper()
	if registry.Global() == nil {
		registry.Init(nil)
	}
	taiID, res := registerForTest(b, pc.Addr, pc.DialOps...)
	pc.TaiID = taiID
	b.Cleanup(func() { res.Close() })
	sandbox.Init()
	m := sandbox.M()
	b.Cleanup(func() { m.Close() })
	return m
}

func ensureTestImageBench(b *testing.B, m *sandbox.Manager, nodeID string) {
	b.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := m.EnsureImage(ctx, nodeID, testImage(), sandbox.ImagePullOptions{}); err != nil {
		b.Fatalf("EnsureImage: %v", err)
	}
}

func createBoxForBench(b *testing.B, m *sandbox.Manager) *sandbox.Box {
	b.Helper()
	nodes := m.Nodes()
	var nodeID string
	if len(nodes) > 0 {
		nodeID = nodes[0].TaiID
		ensureTestImageBench(b, m, nodeID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	box, err := m.Create(ctx, sandbox.CreateOptions{
		Image:  testImage(),
		Owner:  "bench",
		NodeID: nodeID,
	})
	if err != nil {
		b.Fatalf("Create: %v", err)
	}
	b.Cleanup(func() { m.Remove(context.Background(), box.ID()) })
	return box
}
