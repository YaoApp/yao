package trace_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// TestReleaseWhileWriting verifies that calling Release while multiple
// goroutines are actively writing to the trace does not panic or deadlock.
func TestReleaseWhileWriting(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add a root node so logging operations have a target
			_, err = manager.Add("root", types.TraceNodeOption{Label: "Root"})
			assert.NoError(t, err)

			// Start 20 goroutines continuously writing
			var wg sync.WaitGroup
			const numWriters = 20
			stop := make(chan struct{})

			for i := 0; i < numWriters; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					for {
						select {
						case <-stop:
							return
						default:
							manager.Info("writer %d tick", idx)
						}
					}
				}(i)
			}

			// Let writers run briefly, then Release while they are still writing
			time.Sleep(10 * time.Millisecond)
			err = trace.Release(traceID)
			assert.NoError(t, err)

			// Signal writers to stop and wait
			close(stop)
			wg.Wait()
		})
	}
}

// TestReleaseDuringSpaceOp verifies that calling Release while space
// key-value operations are in flight does not panic.
func TestReleaseDuringSpaceOp(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			space, err := manager.CreateSpace(types.TraceSpaceOption{Label: "Test Space"})
			assert.NoError(t, err)

			var wg sync.WaitGroup
			const numOps = 10
			stop := make(chan struct{})

			for i := 0; i < numOps; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					for {
						select {
						case <-stop:
							return
						default:
							key := fmt.Sprintf("key_%d", idx)
							// Errors are expected after Release; we only care about no panic.
							_ = manager.SetSpaceValue(space.ID, key, idx)
							_, _ = manager.GetSpaceValue(space.ID, key)
						}
					}
				}(i)
			}

			time.Sleep(10 * time.Millisecond)
			err = trace.Release(traceID)
			assert.NoError(t, err)

			close(stop)
			wg.Wait()
		})
	}
}

// TestReleaseAfterMarkComplete verifies that MarkComplete followed by
// immediate Release and further operations does not panic.
func TestReleaseAfterMarkComplete(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			_, err = manager.Add("root", types.TraceNodeOption{Label: "Root"})
			assert.NoError(t, err)

			err = manager.MarkComplete()
			assert.NoError(t, err)

			err = trace.Release(traceID)
			assert.NoError(t, err)

			// Post-release operations should not panic.
			// They may return errors or be silently dropped.
			manager.Info("after release")
			_ = manager.SetOutput("stale output")
			_, _ = manager.Add("late", types.TraceNodeOption{Label: "Late"})
		})
	}
}

// TestConcurrentReleaseAndMarkCancelled verifies that calling MarkCancelled
// and Release concurrently does not panic or deadlock.
func TestConcurrentReleaseAndMarkCancelled(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			_, err = manager.Add("root", types.TraceNodeOption{Label: "Root"})
			assert.NoError(t, err)

			done := make(chan struct{})
			go func() {
				defer close(done)

				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()
					_ = trace.MarkCancelled(traceID, "test cancel")
				}()
				go func() {
					defer wg.Done()
					_ = trace.Release(traceID)
				}()

				wg.Wait()
			}()

			select {
			case <-done:
				// success — no deadlock
			case <-time.After(5 * time.Second):
				t.Fatal("deadlock: concurrent MarkCancelled + Release did not finish in 5s")
			}
		})
	}
}

// TestSafeSendAfterClosed verifies that operations using safeSend after
// Release return gracefully instead of panicking.
func TestSafeSendAfterClosed(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			_, err = manager.Add("root", types.TraceNodeOption{Label: "Root"})
			assert.NoError(t, err)

			err = trace.Release(traceID)
			assert.NoError(t, err)

			// All of these internally use safeSend. After Release they should
			// return nil/error/zero-value, never panic.
			manager.Info("post-close info")
			manager.Debug("post-close debug")
			manager.Error("post-close error")
			manager.Warn("post-close warn")

			root, _ := manager.GetRootNode()
			assert.Nil(t, root)

			nodes, _ := manager.GetCurrentNodes()
			assert.Nil(t, nodes)

			status := manager.IsComplete()
			assert.True(t, status)
		})
	}
}

// TestRapidCreateReleaseLoop stress-tests the create/release cycle to ensure
// no goroutine accumulation or panics over many iterations.
func TestRapidCreateReleaseLoop(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Warm up and let baseline stabilize
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			baseGoroutines := runtime.NumGoroutine()

			const iterations = 100
			for i := 0; i < iterations; i++ {
				traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
				assert.NoError(t, err)

				_, err = manager.Add("input", types.TraceNodeOption{Label: "Node"})
				assert.NoError(t, err)
				manager.Info("iteration %d", i)

				err = manager.MarkComplete()
				assert.NoError(t, err)

				err = trace.Release(traceID)
				assert.NoError(t, err)

				err = trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)
				assert.NoError(t, err)
			}

			// Allow goroutines to wind down
			runtime.GC()
			time.Sleep(200 * time.Millisecond)

			finalGoroutines := runtime.NumGoroutine()
			delta := finalGoroutines - baseGoroutines

			// Allow a small margin for runtime goroutines; flag severe leaks
			assert.LessOrEqual(t, delta, 20,
				"goroutine leak: base=%d final=%d delta=%d", baseGoroutines, finalGoroutines, delta)
		})
	}
}

// TestConcurrentAllPattern simulates the real All() orchestrator pattern:
// parent creates a trace, forks N goroutines that each write to the shared
// trace manager, then parent calls MarkComplete + Release while children
// may still be writing.
func TestConcurrentAllPattern(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Root node
			_, err = manager.Add("orchestrator", types.TraceNodeOption{Label: "Orchestrator"})
			assert.NoError(t, err)

			// Create a shared space
			space, err := manager.CreateSpace(types.TraceSpaceOption{Label: "Shared"})
			assert.NoError(t, err)

			// Fork N "child" goroutines, each doing work on the shared trace
			const numChildren = 10
			childStarted := make(chan struct{})
			var wg sync.WaitGroup

			for i := 0; i < numChildren; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					// Signal that this child has started
					select {
					case childStarted <- struct{}{}:
					default:
					}

					// Simulate work: log, set space values, complete
					for j := 0; j < 20; j++ {
						manager.Info("child %d step %d", idx, j)
						_ = manager.SetSpaceValue(space.ID, fmt.Sprintf("child_%d_%d", idx, j), j)
					}
				}(i)
			}

			// Wait for at least a few children to start, then trigger shutdown
			time.Sleep(5 * time.Millisecond)

			// Parent completes and releases — some children are still writing
			_ = manager.MarkComplete()
			_ = trace.Release(traceID)

			// Wait for all children to finish (they should not panic)
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// success
			case <-time.After(10 * time.Second):
				t.Fatal("deadlock: child goroutines did not finish in 10s")
			}
		})
	}
}
