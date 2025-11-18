package trace_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// ============================================================================
// Memory Leak Detection Tests
// ============================================================================

// TestMemoryLeakLocal checks for memory leaks with local driver
// Run with: go test -run=TestMemoryLeakLocal -v
func TestMemoryLeakLocal(t *testing.T) {
	ctx := context.Background()

	// Warm up - execute a few times to stabilize memory
	for i := 0; i < 10; i++ {
		traceID, manager, _ := trace.New(ctx, trace.Local, nil)
		manager.Add("test", types.TraceNodeOption{Label: "Test"})
		manager.Complete("result")
		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}

	// Force GC and get baseline memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Execute many iterations
	iterations := 1000
	for i := 0; i < iterations; i++ {
		traceID, manager, err := trace.New(ctx, trace.Local, nil)
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
			continue
		}

		_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
		if err != nil {
			t.Errorf("Add failed at iteration %d: %s", i, err.Error())
		}

		err = manager.Complete("result")
		if err != nil {
			t.Errorf("Complete failed at iteration %d: %s", i, err.Error())
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)

		// Periodic GC to help detect leaks faster
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force GC and check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	// Calculate memory growth
	baselineHeap := baseline.HeapAlloc
	finalHeap := final.HeapAlloc
	growth := int64(finalHeap) - int64(baselineHeap)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Local Driver):")
	t.Logf("  Iterations:              %d", iterations)
	t.Logf("  Baseline HeapAlloc:      %d bytes (%.2f MB)", baselineHeap, float64(baselineHeap)/1024/1024)
	t.Logf("  Final HeapAlloc:         %d bytes (%.2f MB)", finalHeap, float64(finalHeap)/1024/1024)
	t.Logf("  Total Growth:            %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth per iteration:    %.2f bytes", growthPerIteration)
	t.Logf("  Total Alloc:             %d bytes (%.2f MB)", final.TotalAlloc, float64(final.TotalAlloc)/1024/1024)
	t.Logf("  Mallocs:                 %d", final.Mallocs)
	t.Logf("  Frees:                   %d", final.Frees)
	t.Logf("  Live Objects:            %d", final.Mallocs-final.Frees)
	t.Logf("  GC Runs:                 %d", final.NumGC-baseline.NumGC)

	// Check for memory leak
	// Local driver involves file I/O, allow up to 10KB growth per iteration
	maxGrowthPerIteration := 10240.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak detected: %.2f bytes/iteration (threshold: %.2f bytes/iteration)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestMemoryLeakStore checks for memory leaks with store driver
// Run with: go test -run=TestMemoryLeakStore -v
func TestMemoryLeakStore(t *testing.T) {
	ctx := context.Background()

	// Warm up
	for i := 0; i < 10; i++ {
		traceID, manager, _ := trace.New(ctx, trace.Store, nil)
		manager.Add("test", types.TraceNodeOption{Label: "Test"})
		manager.Complete("result")
		trace.Release(traceID)
		trace.Remove(ctx, trace.Store, traceID)
	}

	// Force GC and get baseline memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Execute many iterations
	iterations := 1000
	for i := 0; i < iterations; i++ {
		traceID, manager, err := trace.New(ctx, trace.Store, nil)
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
			continue
		}

		_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
		if err != nil {
			t.Errorf("Add failed at iteration %d: %s", i, err.Error())
		}

		err = manager.Complete("result")
		if err != nil {
			t.Errorf("Complete failed at iteration %d: %s", i, err.Error())
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Store, traceID)

		// Periodic GC
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force GC and check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	// Calculate memory growth
	baselineHeap := baseline.HeapAlloc
	finalHeap := final.HeapAlloc
	growth := int64(finalHeap) - int64(baselineHeap)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Store Driver):")
	t.Logf("  Iterations:              %d", iterations)
	t.Logf("  Baseline HeapAlloc:      %d bytes (%.2f MB)", baselineHeap, float64(baselineHeap)/1024/1024)
	t.Logf("  Final HeapAlloc:         %d bytes (%.2f MB)", finalHeap, float64(finalHeap)/1024/1024)
	t.Logf("  Total Growth:            %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth per iteration:    %.2f bytes", growthPerIteration)
	t.Logf("  Total Alloc:             %d bytes (%.2f MB)", final.TotalAlloc, float64(final.TotalAlloc)/1024/1024)
	t.Logf("  Mallocs:                 %d", final.Mallocs)
	t.Logf("  Frees:                   %d", final.Frees)
	t.Logf("  Live Objects:            %d", final.Mallocs-final.Frees)
	t.Logf("  GC Runs:                 %d", final.NumGC-baseline.NumGC)

	// Store driver should have similar or better performance than local
	maxGrowthPerIteration := 10240.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak detected: %.2f bytes/iteration (threshold: %.2f bytes/iteration)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestMemoryLeakComplexScenarios checks for memory leaks with complex operations
// Run with: go test -run=TestMemoryLeakComplexScenarios -v
func TestMemoryLeakComplexScenarios(t *testing.T) {
	ctx := context.Background()

	scenarios := []struct {
		name    string
		execute func(types.Manager) error
	}{
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

				for i := 0; i < 10; i++ {
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
			name: "WithSubscription",
			execute: func(m types.Manager) error {
				updates, err := m.Subscribe()
				if err != nil {
					return err
				}

				// Drain updates in background with timeout
				done := make(chan bool)
				go func() {
					timeout := time.After(100 * time.Millisecond)
					for {
						select {
						case _, ok := <-updates:
							if !ok {
								done <- true
								return
							}
						case <-timeout:
							done <- true
							return
						}
					}
				}()

				_, err = m.Add("test", types.TraceNodeOption{Label: "Test"})
				if err != nil {
					return err
				}
				if err := m.Complete("result"); err != nil {
					return err
				}
				if err := m.MarkComplete(); err != nil {
					return err
				}

				// Wait for subscription to drain (with timeout)
				<-done
				return nil
			},
		},
	}

	// Warm up
	for i := 0; i < 10; i++ {
		traceID, manager, _ := trace.New(ctx, trace.Local, nil)
		manager.Add("warmup", types.TraceNodeOption{Label: "Warmup"})
		manager.Complete("done")
		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}

	// Test each scenario
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Get baseline
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			var baseline runtime.MemStats
			runtime.ReadMemStats(&baseline)

			// Execute iterations
			iterations := 200
			for i := 0; i < iterations; i++ {
				traceID, manager, err := trace.New(ctx, trace.Local, nil)
				if err != nil {
					t.Errorf("Create failed at iteration %d: %s", i, err.Error())
					continue
				}

				err = scenario.execute(manager)
				if err != nil {
					t.Errorf("Scenario failed at iteration %d: %s", i, err.Error())
				}

				trace.Release(traceID)
				trace.Remove(ctx, trace.Local, traceID)

				if i%50 == 0 {
					runtime.GC()
				}
			}

			// Check final memory
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			var final runtime.MemStats
			runtime.ReadMemStats(&final)

			growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
			growthPerIteration := float64(growth) / float64(iterations)

			t.Logf("  Baseline HeapAlloc: %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
			t.Logf("  Final HeapAlloc:    %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
			t.Logf("  Growth:             %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
			t.Logf("  Growth/iteration:   %.2f bytes", growthPerIteration)

			// Complex scenarios may have more memory usage
			maxGrowthPerIteration := 15360.0
			if growthPerIteration > maxGrowthPerIteration {
				t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
					growthPerIteration, maxGrowthPerIteration)
			} else {
				t.Logf("  ✓ Memory growth is within acceptable range")
			}
		})
	}
}

// TestMemoryLeakConcurrent checks for memory leaks under concurrent load
// Run with: go test -run=TestMemoryLeakConcurrent -v
func TestMemoryLeakConcurrent(t *testing.T) {
	ctx := context.Background()

	// Warm up
	for i := 0; i < 20; i++ {
		traceID, manager, _ := trace.New(ctx, trace.Local, nil)
		manager.Add("warmup", types.TraceNodeOption{Label: "Warmup"})
		manager.Complete("done")
		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Run concurrent load
	iterations := 1000
	concurrency := 10
	iterPerGoroutine := iterations / concurrency

	done := make(chan bool, concurrency)
	for g := 0; g < concurrency; g++ {
		go func(id int) {
			defer func() { done <- true }()
			for i := 0; i < iterPerGoroutine; i++ {
				traceID, manager, err := trace.New(ctx, trace.Local, nil)
				if err != nil {
					t.Errorf("Goroutine %d: Create failed at iteration %d: %s", id, i, err.Error())
					continue
				}

				_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
				if err != nil {
					t.Errorf("Goroutine %d: Add failed at iteration %d: %s", id, i, err.Error())
				}

				err = manager.Complete("result")
				if err != nil {
					t.Errorf("Goroutine %d: Complete failed at iteration %d: %s", id, i, err.Error())
				}

				trace.Release(traceID)
				trace.Remove(ctx, trace.Local, traceID)
			}
		}(g)
	}

	// Wait for all goroutines
	for g := 0; g < concurrency; g++ {
		<-done
	}

	// Check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Concurrent Load):")
	t.Logf("  Iterations:           %d", iterations)
	t.Logf("  Concurrency:          %d", concurrency)
	t.Logf("  Baseline HeapAlloc:   %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc:      %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth:               %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth/iteration:     %.2f bytes", growthPerIteration)
	t.Logf("  GC Runs:              %d", final.NumGC-baseline.NumGC)

	// Concurrent scenarios may have slightly more overhead
	maxGrowthPerIteration := 15360.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestMemoryLeakSpaceOperations checks for memory leaks with space operations
// Run with: go test -run=TestMemoryLeakSpaceOperations -v
func TestMemoryLeakSpaceOperations(t *testing.T) {
	ctx := context.Background()

	// Warm up
	for i := 0; i < 10; i++ {
		traceID, manager, _ := trace.New(ctx, trace.Local, nil)
		space, _ := manager.CreateSpace(types.TraceSpaceOption{Label: "Test"})
		manager.SetSpaceValue(space.ID, "key", "value")
		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)
	}

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Execute iterations with space operations
	iterations := 500
	for i := 0; i < iterations; i++ {
		traceID, manager, err := trace.New(ctx, trace.Local, nil)
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
			continue
		}

		// Create space and perform operations
		space, err := manager.CreateSpace(types.TraceSpaceOption{Label: "Test Space"})
		if err != nil {
			t.Errorf("CreateSpace failed at iteration %d: %s", i, err.Error())
		}

		// Set multiple values
		for j := 0; j < 20; j++ {
			err = manager.SetSpaceValue(space.ID, fmt.Sprintf("key_%d", j), fmt.Sprintf("value_%d", j))
			if err != nil {
				t.Errorf("SetSpaceValue failed at iteration %d: %s", i, err.Error())
			}
		}

		// Get values
		for j := 0; j < 20; j++ {
			_, err = manager.GetSpaceValue(space.ID, fmt.Sprintf("key_%d", j))
			if err != nil {
				t.Errorf("GetSpaceValue failed at iteration %d: %s", i, err.Error())
			}
		}

		// Delete some values
		for j := 0; j < 10; j++ {
			err = manager.DeleteSpaceValue(space.ID, fmt.Sprintf("key_%d", j))
			if err != nil {
				t.Errorf("DeleteSpaceValue failed at iteration %d: %s", i, err.Error())
			}
		}

		trace.Release(traceID)
		trace.Remove(ctx, trace.Local, traceID)

		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Space Operations):")
	t.Logf("  Iterations:           %d", iterations)
	t.Logf("  Baseline HeapAlloc:   %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc:      %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth:               %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth/iteration:     %.2f bytes", growthPerIteration)
	t.Logf("  GC Runs:              %d", final.NumGC-baseline.NumGC)

	// Space operations involve maps and persistence
	maxGrowthPerIteration := 20480.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestGoroutineLeak verifies that no goroutines are leaked
// Run with: go test -run=TestGoroutineLeak -v
func TestGoroutineLeak(t *testing.T) {
	ctx := context.Background()

	// Track goroutine count to detect goroutine leaks
	initialGoroutines := runtime.NumGoroutine()

	// Execute multiple iterations
	iterations := 100
	for i := 0; i < iterations; i++ {
		traceID, manager, err := trace.New(ctx, trace.Local, nil)
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
			continue
		}

		// Subscribe (creates goroutines)
		updates, err := manager.Subscribe()
		if err != nil {
			t.Errorf("Subscribe failed at iteration %d: %s", i, err.Error())
		}

		// Perform operations
		_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
		if err != nil {
			t.Errorf("Add failed at iteration %d: %s", i, err.Error())
		}

		err = manager.Complete("result")
		if err != nil {
			t.Errorf("Complete failed at iteration %d: %s", i, err.Error())
		}

		err = manager.MarkComplete()
		if err != nil {
			t.Errorf("MarkComplete failed at iteration %d: %s", i, err.Error())
		}

		// Drain subscription
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

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineGrowth := finalGoroutines - initialGoroutines

	t.Logf("Goroutine Statistics:")
	t.Logf("  Initial:  %d", initialGoroutines)
	t.Logf("  Final:    %d", finalGoroutines)
	t.Logf("  Growth:   %d", goroutineGrowth)

	// Allow some goroutine growth for runtime internals, but not proportional to iterations
	maxGoroutineGrowth := 20
	if goroutineGrowth > maxGoroutineGrowth {
		t.Errorf("Possible goroutine leak: %d new goroutines (threshold: %d)",
			goroutineGrowth, maxGoroutineGrowth)
	} else {
		t.Logf("✓ No goroutine leak detected")
	}
}
