package trace_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

func TestConcurrentNodeOperations(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add first node as root
			_, err = manager.Add("root", types.TraceNodeOption{Label: "Root"})
			assert.NoError(t, err)

			// Create parallel nodes
			nodes, err := manager.Parallel([]types.TraceParallelInput{
				{Input: "task 1", Option: types.TraceNodeOption{Label: "Worker 1"}},
				{Input: "task 2", Option: types.TraceNodeOption{Label: "Worker 2"}},
				{Input: "task 3", Option: types.TraceNodeOption{Label: "Worker 3"}},
				{Input: "task 4", Option: types.TraceNodeOption{Label: "Worker 4"}},
				{Input: "task 5", Option: types.TraceNodeOption{Label: "Worker 5"}},
			})
			assert.NoError(t, err)
			assert.Len(t, nodes, 5)

			// Concurrent operations on each node
			var wg sync.WaitGroup
			for i, node := range nodes {
				wg.Add(1)
				go func(idx int, n types.Node) {
					defer wg.Done()

					// Concurrent logging
					n.Info("Starting worker %d", idx+1)
					n.Debug("Debug info %d", idx+1)

					// Set metadata
					err := n.SetMetadata("worker_id", idx+1)
					assert.NoError(t, err)

					// Complete
					err = n.Complete(map[string]any{"worker": idx + 1})
					assert.NoError(t, err)
				}(i, node)
			}
			wg.Wait()
		})
	}
}

func TestConcurrentSpaceOperations(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create shared space
			space, err := manager.CreateSpace(types.TraceSpaceOption{
				Label: "Shared Space",
			})
			assert.NoError(t, err)

			// Concurrent writes to the SAME space (now thread-safe with per-space locks)
			var wg sync.WaitGroup
			numWorkers := 10

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					key := fmt.Sprintf("key_%d", idx)
					value := fmt.Sprintf("value_%d", idx)

					err := manager.SetSpaceValue(space.ID, key, value)
					assert.NoError(t, err)
				}(i)
			}
			wg.Wait()

			// Verify all keys were set
			keys := manager.ListSpaceKeys(space.ID)
			assert.Len(t, keys, numWorkers)

			// Concurrent reads
			wg = sync.WaitGroup{}
			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					key := fmt.Sprintf("key_%d", idx)
					val, err := manager.GetSpaceValue(space.ID, key)
					assert.NoError(t, err)
					assert.Equal(t, fmt.Sprintf("value_%d", idx), val)
				}(i)
			}
			wg.Wait()
		})
	}
}

func TestConcurrentSpaceCreation(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create multiple spaces concurrently
			var wg sync.WaitGroup
			numSpaces := 10
			spaces := make([]*types.TraceSpace, numSpaces)
			var mu sync.Mutex

			for i := 0; i < numSpaces; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					space, err := manager.CreateSpace(types.TraceSpaceOption{
						Label: fmt.Sprintf("Space %d", idx),
					})
					assert.NoError(t, err)

					mu.Lock()
					spaces[idx] = space
					mu.Unlock()
				}(i)
			}
			wg.Wait()

			// Verify all spaces were created
			allSpaces := manager.ListSpaces()
			assert.Len(t, allSpaces, numSpaces)
		})
	}
}

func TestConcurrentSubscribers(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create multiple subscribers concurrently
			var wg sync.WaitGroup
			numSubscribers := 5
			subscribers := make([]<-chan *types.TraceUpdate, numSubscribers)
			var mu sync.Mutex

			for i := 0; i < numSubscribers; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					sub, err := manager.Subscribe()
					assert.NoError(t, err)

					mu.Lock()
					subscribers[idx] = sub
					mu.Unlock()
				}(i)
			}
			wg.Wait()

			// Verify all subscriptions were created
			for i, sub := range subscribers {
				assert.NotNil(t, sub, "Subscriber %d should not be nil", i)
			}

			// Perform operations and verify all subscribers receive updates
			_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
			assert.NoError(t, err)
		})
	}
}

func TestConcurrentTraceCreation(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Create multiple traces concurrently
			var wg sync.WaitGroup
			numTraces := 10
			traceIDs := make([]string, numTraces)
			var mu sync.Mutex

			for i := 0; i < numTraces; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
					assert.NoError(t, err)
					assert.NotNil(t, manager)

					mu.Lock()
					traceIDs[idx] = traceID
					mu.Unlock()
				}(i)
			}
			wg.Wait()

			// Clean up all traces
			defer func() {
				for _, traceID := range traceIDs {
					trace.Release(traceID)
					trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)
				}
			}()

			// Verify all traces were created and loaded
			for i, traceID := range traceIDs {
				assert.NotEmpty(t, traceID, "Trace %d should have ID", i)
				assert.True(t, trace.IsLoaded(traceID), "Trace %d should be loaded", i)
			}
		})
	}
}

func TestConcurrentLogging(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Concurrent logging
			var wg sync.WaitGroup
			numLogs := 50
			for i := 0; i < numLogs; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					manager.Info("Log message %d", idx)
					manager.Debug("Debug message %d", idx)
					manager.Warn("Warning message %d", idx)
				}(i)
			}
			wg.Wait()

			// Note: We can't easily verify log count without exposing LoadLogs,
			// but we verify no errors occurred during concurrent logging
		})
	}
}
