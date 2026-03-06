package workspace_test

import (
	"context"
	"fmt"
	"io/fs"
	"testing"

	"github.com/yaoapp/yao/workspace"
)

// BenchmarkWriteFile measures workspace file write latency.
func BenchmarkWriteFile(b *testing.B) {
	for _, pc := range testPools() {
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForPool(b, pc)
			ws := createWorkspace(b, m, pc.Name)
			ctx := context.Background()
			payload := []byte("package main\nfunc main() { println(\"bench\") }\n")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := m.WriteFile(ctx, ws.ID, fmt.Sprintf("f%d.go", i), payload, 0644); err != nil {
					b.Fatalf("WriteFile: %v", err)
				}
			}
		})
	}
}

// BenchmarkReadFile measures workspace file read latency.
func BenchmarkReadFile(b *testing.B) {
	for _, pc := range testPools() {
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForPool(b, pc)
			ws := createWorkspace(b, m, pc.Name)
			ctx := context.Background()
			if err := m.WriteFile(ctx, ws.ID, "bench.txt", []byte("benchmark data here"), 0644); err != nil {
				b.Fatalf("setup WriteFile: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				data, err := m.ReadFile(ctx, ws.ID, "bench.txt")
				if err != nil {
					b.Fatalf("ReadFile: %v", err)
				}
				if len(data) == 0 {
					b.Fatal("empty data")
				}
			}
		})
	}
}

// BenchmarkReadWriteCycle measures a full write-then-read cycle.
func BenchmarkReadWriteCycle(b *testing.B) {
	for _, pc := range testPools() {
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForPool(b, pc)
			ws := createWorkspace(b, m, pc.Name)
			ctx := context.Background()
			payload := []byte("package main\nfunc main() { println(\"cycle\") }\n")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				name := fmt.Sprintf("c%d.go", i)
				if err := m.WriteFile(ctx, ws.ID, name, payload, 0644); err != nil {
					b.Fatalf("WriteFile: %v", err)
				}
				data, err := m.ReadFile(ctx, ws.ID, name)
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

// BenchmarkWriteLargeFile measures write throughput with a 1MB payload.
func BenchmarkWriteLargeFile(b *testing.B) {
	for _, pc := range testPools() {
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForPool(b, pc)
			ws := createWorkspace(b, m, pc.Name)
			ctx := context.Background()
			payload := make([]byte, 1<<20) // 1 MB
			for i := range payload {
				payload[i] = byte('A' + i%26)
			}

			b.SetBytes(int64(len(payload)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := m.WriteFile(ctx, ws.ID, fmt.Sprintf("large%d.bin", i), payload, 0644); err != nil {
					b.Fatalf("WriteFile: %v", err)
				}
			}
		})
	}
}

// BenchmarkListDir measures directory listing latency (50 files).
func BenchmarkListDir(b *testing.B) {
	for _, pc := range testPools() {
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForPool(b, pc)
			ws := createWorkspace(b, m, pc.Name)
			ctx := context.Background()

			for i := 0; i < 50; i++ {
				m.WriteFile(ctx, ws.ID, fmt.Sprintf("file%d.txt", i), []byte("x"), 0644)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				entries, err := m.ListDir(ctx, ws.ID, ".")
				if err != nil {
					b.Fatalf("ListDir: %v", err)
				}
				if len(entries) < 50 {
					b.Fatalf("expected >= 50 entries, got %d", len(entries))
				}
			}
		})
	}
}

// BenchmarkFSWalkDir measures fs.WalkDir performance over a directory tree (45+ entries).
func BenchmarkFSWalkDir(b *testing.B) {
	for _, pc := range testPools() {
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForPool(b, pc)
			ws := createWorkspace(b, m, pc.Name)
			ctx := context.Background()

			wfs, err := m.FS(ctx, ws.ID)
			if err != nil {
				b.Fatalf("FS: %v", err)
			}

			for _, dir := range []string{"src", "src/pkg", "src/cmd", "lib"} {
				wfs.MkdirAll(dir, 0755)
			}
			for i := 0; i < 20; i++ {
				wfs.WriteFile(fmt.Sprintf("src/f%d.go", i), []byte("package src"), 0644)
			}
			for i := 0; i < 10; i++ {
				wfs.WriteFile(fmt.Sprintf("src/pkg/p%d.go", i), []byte("package pkg"), 0644)
			}
			for i := 0; i < 10; i++ {
				wfs.WriteFile(fmt.Sprintf("lib/l%d.go", i), []byte("package lib"), 0644)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				count := 0
				fs.WalkDir(wfs, ".", func(_ string, _ fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					count++
					return nil
				})
				if count < 40 {
					b.Fatalf("walk returned only %d entries", count)
				}
			}
		})
	}
}

// BenchmarkCreateDelete measures workspace CRUD cycle.
func BenchmarkCreateDelete(b *testing.B) {
	for _, pc := range testPools() {
		b.Run(pc.Name, func(b *testing.B) {
			m := setupManagerForPool(b, pc)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ws, err := m.Create(ctx, workspace.CreateOptions{
					Name:  "bench-workspace",
					Owner: "bench-user",
					Node:  pc.Name,
				})
				if err != nil {
					b.Fatalf("Create: %v", err)
				}
				if err := m.Delete(ctx, ws.ID, true); err != nil {
					b.Fatalf("Delete: %v", err)
				}
			}
		})
	}
}
