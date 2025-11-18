package trace_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// ============================================================================
// Simple Scenario Benchmarks
// ============================================================================

// BenchmarkSimpleTraceLocal benchmarks simple trace operations with local driver
// Run with: go test -bench=BenchmarkSimpleTraceLocal -benchmem -benchtime=100x
func BenchmarkSimpleTraceLocal(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		traceID, manager, err := trace.New(ctx, trace.Local, nil)
		if err != nil {
			b.Fatalf("Failed to create trace: %s", err.Error())
		}

		_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
		if err != nil {
			b.Fatalf("Failed to add node: %s", err.Error())
		}

		err = manager.Complete("result")
		if err != nil {
			b.Fatalf("Failed to complete: %s", err.Error())
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}
}

// BenchmarkSimpleTraceStore benchmarks simple trace operations with store driver
// Run with: go test -bench=BenchmarkSimpleTraceStore -benchmem -benchtime=100x
func BenchmarkSimpleTraceStore(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		traceID, manager, err := trace.New(ctx, trace.Store, nil)
		if err != nil {
			b.Fatalf("Failed to create trace: %s", err.Error())
		}

		_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
		if err != nil {
			b.Fatalf("Failed to add node: %s", err.Error())
		}

		err = manager.Complete("result")
		if err != nil {
			b.Fatalf("Failed to complete: %s", err.Error())
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Store, traceID)
	}
}

// ============================================================================
// Complex Scenario Benchmarks (with Parallel, Space, Subscription)
// ============================================================================

// BenchmarkComplexTraceLocal benchmarks complex trace operations with local driver
// Run with: go test -bench=BenchmarkComplexTraceLocal -benchmem -benchtime=100x
func BenchmarkComplexTraceLocal(b *testing.B) {
	ctx := context.Background()

	scenarios := getTraceScenarios()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scenario := scenarios[i%len(scenarios)]

		traceID, manager, err := trace.New(ctx, trace.Local, nil)
		if err != nil {
			b.Fatalf("Failed to create trace: %s", err.Error())
		}

		err = scenario.execute(manager)
		if err != nil {
			b.Errorf("%s failed: %s", scenario.name, err.Error())
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}
}

// BenchmarkComplexTraceStore benchmarks complex trace operations with store driver
// Run with: go test -bench=BenchmarkComplexTraceStore -benchmem -benchtime=100x
func BenchmarkComplexTraceStore(b *testing.B) {
	ctx := context.Background()

	scenarios := getTraceScenarios()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scenario := scenarios[i%len(scenarios)]

		traceID, manager, err := trace.New(ctx, trace.Store, nil)
		if err != nil {
			b.Fatalf("Failed to create trace: %s", err.Error())
		}

		err = scenario.execute(manager)
		if err != nil {
			b.Errorf("%s failed: %s", scenario.name, err.Error())
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Store, traceID)
	}
}

// ============================================================================
// Concurrent Benchmarks
// ============================================================================

// BenchmarkConcurrentSimpleLocal benchmarks concurrent simple operations with local driver
// Run with: go test -bench=BenchmarkConcurrentSimpleLocal -benchmem -benchtime=100x
func BenchmarkConcurrentSimpleLocal(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			traceID, manager, err := trace.New(ctx, trace.Local, nil)
			if err != nil {
				b.Errorf("Failed to create trace: %s", err.Error())
				continue
			}

			_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
			if err != nil {
				b.Errorf("Failed to add node: %s", err.Error())
			}

			err = manager.Complete("result")
			if err != nil {
				b.Errorf("Failed to complete: %s", err.Error())
			}

			trace.Release(traceID)
			trace.Remove(ctx, trace.Local, traceID)
		}
	})
}

// BenchmarkConcurrentSimpleStore benchmarks concurrent simple operations with store driver
// Run with: go test -bench=BenchmarkConcurrentSimpleStore -benchmem -benchtime=100x
func BenchmarkConcurrentSimpleStore(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			traceID, manager, err := trace.New(ctx, trace.Store, nil)
			if err != nil {
				b.Errorf("Failed to create trace: %s", err.Error())
				continue
			}

			_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
			if err != nil {
				b.Errorf("Failed to add node: %s", err.Error())
			}

			err = manager.Complete("result")
			if err != nil {
				b.Errorf("Failed to complete: %s", err.Error())
			}

			trace.Release(traceID)
			trace.Remove(ctx, trace.Store, traceID)
		}
	})
}

// BenchmarkConcurrentComplexLocal benchmarks concurrent complex operations with local driver
// Run with: go test -bench=BenchmarkConcurrentComplexLocal -benchmem -benchtime=100x
func BenchmarkConcurrentComplexLocal(b *testing.B) {
	ctx := context.Background()
	scenarios := getTraceScenarios()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			scenario := scenarios[i%len(scenarios)]
			i++

			traceID, manager, err := trace.New(ctx, trace.Local, nil)
			if err != nil {
				b.Errorf("Failed to create trace: %s", err.Error())
				continue
			}

			err = scenario.execute(manager)
			if err != nil {
				b.Errorf("%s failed: %s", scenario.name, err.Error())
			}

			trace.Release(traceID)
			trace.Remove(ctx, trace.Local, traceID)
		}
	})
}

// BenchmarkConcurrentComplexStore benchmarks concurrent complex operations with store driver
// Run with: go test -bench=BenchmarkConcurrentComplexStore -benchmem -benchtime=100x
func BenchmarkConcurrentComplexStore(b *testing.B) {
	ctx := context.Background()
	scenarios := getTraceScenarios()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			scenario := scenarios[i%len(scenarios)]
			i++

			traceID, manager, err := trace.New(ctx, trace.Store, nil)
			if err != nil {
				b.Errorf("Failed to create trace: %s", err.Error())
				continue
			}

			err = scenario.execute(manager)
			if err != nil {
				b.Errorf("%s failed: %s", scenario.name, err.Error())
			}

			trace.Release(traceID)
			trace.Remove(ctx, trace.Store, traceID)
		}
	})
}

// ============================================================================
// Subscription Benchmarks
// ============================================================================

// BenchmarkSubscription benchmarks subscription operations
// Run with: go test -bench=BenchmarkSubscription -benchmem -benchtime=100x
func BenchmarkSubscription(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		traceID, manager, err := trace.New(ctx, trace.Local, nil)
		if err != nil {
			b.Fatalf("Failed to create trace: %s", err.Error())
		}

		// Subscribe
		updates, err := manager.Subscribe()
		if err != nil {
			b.Fatalf("Failed to subscribe: %s", err.Error())
		}

		// Perform operations
		_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
		if err != nil {
			b.Fatalf("Failed to add node: %s", err.Error())
		}

		err = manager.Complete("result")
		if err != nil {
			b.Fatalf("Failed to complete: %s", err.Error())
		}

		err = manager.MarkComplete()
		if err != nil {
			b.Fatalf("Failed to mark complete: %s", err.Error())
		}

		// Drain updates
		timeout := time.After(10 * time.Millisecond)
	drainLoop:
		for {
			select {
			case _, ok := <-updates:
				if !ok {
					break drainLoop
				}
			case <-timeout:
				break drainLoop
			}
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}
}

// ============================================================================
// Space Operations Benchmarks
// ============================================================================

// BenchmarkSpaceOperations benchmarks space operations
// Run with: go test -bench=BenchmarkSpaceOperations -benchmem -benchtime=100x
func BenchmarkSpaceOperations(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		traceID, manager, err := trace.New(ctx, trace.Local, nil)
		if err != nil {
			b.Fatalf("Failed to create trace: %s", err.Error())
		}

		// Create space
		space, err := manager.CreateSpace(types.TraceSpaceOption{Label: "Test Space"})
		if err != nil {
			b.Fatalf("Failed to create space: %s", err.Error())
		}

		// Set values
		for j := 0; j < 10; j++ {
			err = manager.SetSpaceValue(space.ID, fmt.Sprintf("key_%d", j), fmt.Sprintf("value_%d", j))
			if err != nil {
				b.Fatalf("Failed to set space value: %s", err.Error())
			}
		}

		// Get values
		for j := 0; j < 10; j++ {
			_, err = manager.GetSpaceValue(space.ID, fmt.Sprintf("key_%d", j))
			if err != nil {
				b.Fatalf("Failed to get space value: %s", err.Error())
			}
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

type traceScenario struct {
	name    string
	execute func(types.Manager) error
}

func getTraceScenarios() []traceScenario {
	return []traceScenario{
		{
			name: "SequentialNodes",
			execute: func(m types.Manager) error {
				for i := 0; i < 5; i++ {
					_, err := m.Add(fmt.Sprintf("step_%d", i), types.TraceNodeOption{Label: fmt.Sprintf("Step %d", i)})
					if err != nil {
						return err
					}
					if err := m.Complete(fmt.Sprintf("result_%d", i)); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			name: "ParallelNodes",
			execute: func(m types.Manager) error {
				// Add first node as root
				_, err := m.Add("root", types.TraceNodeOption{Label: "Root"})
				if err != nil {
					return err
				}

				nodes, err := m.Parallel([]types.TraceParallelInput{
					{Input: "task1", Option: types.TraceNodeOption{Label: "Task 1"}},
					{Input: "task2", Option: types.TraceNodeOption{Label: "Task 2"}},
					{Input: "task3", Option: types.TraceNodeOption{Label: "Task 3"}},
				})
				if err != nil {
					return err
				}

				var wg sync.WaitGroup
				for i, node := range nodes {
					wg.Add(1)
					go func(idx int, n types.Node) {
						defer wg.Done()
						n.Complete(fmt.Sprintf("result_%d", idx))
					}(i, node)
				}
				wg.Wait()
				return nil
			},
		},
		{
			name: "WithSpace",
			execute: func(m types.Manager) error {
				space, err := m.CreateSpace(types.TraceSpaceOption{Label: "Context"})
				if err != nil {
					return err
				}

				for i := 0; i < 5; i++ {
					if err := m.SetSpaceValue(space.ID, fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i)); err != nil {
						return err
					}
				}

				_, err = m.Add("process", types.TraceNodeOption{Label: "Process"})
				if err != nil {
					return err
				}
				return m.Complete("done")
			},
		},
		{
			name: "WithLogging",
			execute: func(m types.Manager) error {
				m.Info("Starting process")
				_, err := m.Add("step1", types.TraceNodeOption{Label: "Step 1"})
				if err != nil {
					return err
				}
				m.Debug("Debug info")
				if err := m.Complete("result1"); err != nil {
					return err
				}

				_, err = m.Add("step2", types.TraceNodeOption{Label: "Step 2"})
				if err != nil {
					return err
				}
				m.Warn("Warning message")
				return m.Complete("result2")
			},
		},
		{
			name: "ComplexFlow",
			execute: func(m types.Manager) error {
				// Create space
				space, err := m.CreateSpace(types.TraceSpaceOption{Label: "Shared"})
				if err != nil {
					return err
				}

				// Sequential node
				_, err = m.Add("prepare", types.TraceNodeOption{Label: "Prepare"})
				if err != nil {
					return err
				}
				m.Info("Preparing data")
				if err := m.Complete("prepared"); err != nil {
					return err
				}

				// Parallel nodes
				nodes, err := m.Parallel([]types.TraceParallelInput{
					{Input: "taskA", Option: types.TraceNodeOption{Label: "Task A"}},
					{Input: "taskB", Option: types.TraceNodeOption{Label: "Task B"}},
				})
				if err != nil {
					return err
				}

				var wg sync.WaitGroup
				for i, node := range nodes {
					wg.Add(1)
					go func(idx int, n types.Node) {
						defer wg.Done()
						n.Info("Processing task %d", idx)
						m.SetSpaceValue(space.ID, fmt.Sprintf("result_%d", idx), fmt.Sprintf("done_%d", idx))
						n.Complete(fmt.Sprintf("result_%d", idx))
					}(i, node)
				}
				wg.Wait()

				// Final node
				_, err = m.Add("finalize", types.TraceNodeOption{Label: "Finalize"})
				if err != nil {
					return err
				}
				return m.Complete("completed")
			},
		},
	}
}
