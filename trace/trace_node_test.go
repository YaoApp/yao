package trace_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

func TestNodeOperations(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add sequential node
			node1, err := manager.Add("input data", types.TraceNodeOption{
				Label:       "Input Processing",
				Icon:        "processor",
				Description: "Process input data",
			})
			assert.NoError(t, err)
			assert.NotNil(t, node1)

			// Log messages (chainable)
			manager.Info("Processing started").
				Debug("Debug info").
				Warn("Warning message")

			// Set output and complete
			err = manager.Complete(map[string]any{"result": "success"})
			assert.NoError(t, err)

			// Add another node
			node2, err := manager.Add("processing", types.TraceNodeOption{
				Label: "Processing",
				Icon:  "cpu",
			})
			assert.NoError(t, err)
			assert.NotNil(t, node2)

			// Set metadata
			err = manager.SetMetadata("key1", "value1")
			assert.NoError(t, err)

			err = manager.Complete(map[string]any{"status": "done"})
			assert.NoError(t, err)

			// Get current nodes
			currentNodes, err := manager.GetCurrentNodes()
			assert.NoError(t, err)
			assert.NotEmpty(t, currentNodes)
			assert.Equal(t, types.StatusCompleted, currentNodes[0].Status)
		})
	}
}

func TestParallelOperations(t *testing.T) {
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
				{
					Input:  "task A",
					Option: types.TraceNodeOption{Label: "Worker A", Icon: "cpu"},
				},
				{
					Input:  "task B",
					Option: types.TraceNodeOption{Label: "Worker B", Icon: "cpu"},
				},
				{
					Input:  "task C",
					Option: types.TraceNodeOption{Label: "Worker C", Icon: "cpu"},
				},
			})
			assert.NoError(t, err)
			assert.Len(t, nodes, 3)

			// Each node completes itself
			var wg sync.WaitGroup
			for i, node := range nodes {
				wg.Add(1)
				go func(idx int, n types.Node) {
					defer wg.Done()

					n.Info("Worker %d processing", idx+1)
					time.Sleep(10 * time.Millisecond)
					err := n.Complete(map[string]any{"worker": idx + 1, "status": "done"})
					assert.NoError(t, err)
				}(i, node)
			}
			wg.Wait()

			// Add node after parallel (auto-join)
			node, err := manager.Add("merge", types.TraceNodeOption{
				Label: "Merge",
				Icon:  "merge",
			})
			assert.NoError(t, err)
			assert.NotNil(t, node)

			err = manager.Complete(map[string]any{"merged": true})
			assert.NoError(t, err)
		})
	}
}

func TestNodeFailOperation(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add node
			_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
			assert.NoError(t, err)

			// Fail node
			testErr := fmt.Errorf("test error")
			err = manager.Fail(testErr)
			assert.NoError(t, err)

			// Verify node status
			currentNodes, err := manager.GetCurrentNodes()
			assert.NoError(t, err)
			assert.NotEmpty(t, currentNodes)
			assert.Equal(t, types.StatusFailed, currentNodes[0].Status)
		})
	}
}

func TestNodeChaining(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Test chainable logging
			result := manager.Info("Step 1").
				Debug("Debug step 1").
				Warn("Warning step 1")

			// Should return Manager interface
			assert.NotNil(t, result)

			// Should still be able to call Manager methods
			_, err = result.Add("next", types.TraceNodeOption{Label: "Next"})
			assert.NoError(t, err)
		})
	}
}

func TestCompleteWithOutput(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add node
			_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
			assert.NoError(t, err)

			// Complete with output directly
			output := map[string]any{"result": "success", "count": 42}
			err = manager.Complete(output)
			assert.NoError(t, err)

			// Verify output was set
			currentNodes, err := manager.GetCurrentNodes()
			assert.NoError(t, err)
			assert.NotEmpty(t, currentNodes)
			assert.Equal(t, output, currentNodes[0].Output)
		})
	}
}
