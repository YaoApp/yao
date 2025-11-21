package trace_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// TestAutoCompleteParentDefault tests the default behavior where parent nodes are auto-completed
func TestAutoCompleteParentDefault(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add first node
			node1, err := manager.Add("step1", types.TraceNodeOption{
				Label: "Step 1",
			})
			assert.NoError(t, err)

			// Verify node1 is running
			currentNodes, err := manager.GetCurrentNodes()
			assert.NoError(t, err)
			assert.Len(t, currentNodes, 1)
			assert.Equal(t, types.StatusRunning, currentNodes[0].Status)

			// Add second node - node1 should be auto-completed
			node2, err := manager.Add("step2", types.TraceNodeOption{
				Label: "Step 2",
			})
			assert.NoError(t, err)

			// Verify node1 is now completed
			node1Data, err := manager.GetNodeByID(node1.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusCompleted, node1Data.Status)

			// Verify node2 is running
			currentNodes, err = manager.GetCurrentNodes()
			assert.NoError(t, err)
			assert.Len(t, currentNodes, 1)
			assert.Equal(t, node2.ID(), currentNodes[0].ID)
			assert.Equal(t, types.StatusRunning, currentNodes[0].Status)
		})
	}
}

// TestAutoCompleteParentDisabled tests disabling auto-complete behavior
func TestAutoCompleteParentDisabled(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add first node
			node1, err := manager.Add("step1", types.TraceNodeOption{
				Label: "Step 1",
			})
			assert.NoError(t, err)

			// Add second node with auto-complete disabled
			falseVal := false
			node2, err := manager.Add("step2", types.TraceNodeOption{
				Label:              "Step 2",
				AutoCompleteParent: &falseVal,
			})
			assert.NoError(t, err)

			// Verify node1 is still running (not auto-completed)
			node1Data, err := manager.GetNodeByID(node1.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusRunning, node1Data.Status)

			// Verify node2 is running
			node2Data, err := manager.GetNodeByID(node2.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusRunning, node2Data.Status)
		})
	}
}

// TestAutoCompleteParentParallel tests auto-complete with parallel nodes
func TestAutoCompleteParentParallel(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add root node
			rootNode, err := manager.Add("root", types.TraceNodeOption{
				Label: "Root",
			})
			assert.NoError(t, err)

			// Create parallel nodes
			parallelNodes, err := manager.Parallel([]types.TraceParallelInput{
				{Input: "task1", Option: types.TraceNodeOption{Label: "Task 1"}},
				{Input: "task2", Option: types.TraceNodeOption{Label: "Task 2"}},
				{Input: "task3", Option: types.TraceNodeOption{Label: "Task 3"}},
			})
			assert.NoError(t, err)
			assert.Len(t, parallelNodes, 3)

			// Root node should be auto-completed
			rootData, err := manager.GetNodeByID(rootNode.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusCompleted, rootData.Status)

			// All parallel nodes should be running
			for _, node := range parallelNodes {
				nodeData, err := manager.GetNodeByID(node.ID())
				assert.NoError(t, err)
				assert.Equal(t, types.StatusRunning, nodeData.Status)
			}

			// Add a merge node - all parallel nodes should be auto-completed
			mergeNode, err := manager.Add("merge", types.TraceNodeOption{
				Label: "Merge",
			})
			assert.NoError(t, err)

			// All parallel nodes should now be completed
			for _, node := range parallelNodes {
				nodeData, err := manager.GetNodeByID(node.ID())
				assert.NoError(t, err)
				assert.Equal(t, types.StatusCompleted, nodeData.Status)
			}

			// Merge node should be running
			mergeData, err := manager.GetNodeByID(mergeNode.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusRunning, mergeData.Status)
		})
	}
}

// TestAutoCompleteParentConcurrent tests auto-complete with concurrent operations
func TestAutoCompleteParentConcurrent(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add root node
			_, err = manager.Add("root", types.TraceNodeOption{Label: "Root"})
			assert.NoError(t, err)

			// Create parallel nodes
			parallelNodes, err := manager.Parallel([]types.TraceParallelInput{
				{Input: "task1", Option: types.TraceNodeOption{Label: "Task 1"}},
				{Input: "task2", Option: types.TraceNodeOption{Label: "Task 2"}},
				{Input: "task3", Option: types.TraceNodeOption{Label: "Task 3"}},
			})
			assert.NoError(t, err)

			// Complete all parallel nodes (simulating concurrent work)
			var wg sync.WaitGroup
			for _, pNode := range parallelNodes {
				wg.Add(1)
				go func(node types.Node) {
					defer wg.Done()
					// Small delay to simulate real work
					time.Sleep(10 * time.Millisecond)
					err := node.Complete(map[string]any{"done": true})
					assert.NoError(t, err)
				}(pNode)
			}
			wg.Wait()

			// All parallel nodes should be completed
			for _, node := range parallelNodes {
				nodeData, err := manager.GetNodeByID(node.ID())
				assert.NoError(t, err)
				assert.Equal(t, types.StatusCompleted, nodeData.Status)
			}

			// Add a merge node - should work correctly with all parents completed
			mergeNode, err := manager.Add("merge", types.TraceNodeOption{
				Label: "Merge Results",
			})
			assert.NoError(t, err)

			// Verify merge node is running
			mergeData, err := manager.GetNodeByID(mergeNode.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusRunning, mergeData.Status)

			// Verify merge node has all parallel nodes as parents
			assert.Len(t, mergeData.ParentIDs, 3)
		})
	}
}

// TestAutoCompleteParentWithFailedNode tests that failed nodes are not auto-completed
func TestAutoCompleteParentWithFailedNode(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add first node
			node1, err := manager.Add("step1", types.TraceNodeOption{
				Label: "Step 1",
			})
			assert.NoError(t, err)

			// Fail node1
			err = node1.Fail(assert.AnError)
			assert.NoError(t, err)

			// Verify node1 is failed
			node1Data, err := manager.GetNodeByID(node1.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusFailed, node1Data.Status)

			// Add second node with auto-complete enabled
			node2, err := manager.Add("step2", types.TraceNodeOption{
				Label: "Step 2",
			})
			assert.NoError(t, err)

			// node1 should still be failed (not auto-completed to success)
			node1Data, err = manager.GetNodeByID(node1.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusFailed, node1Data.Status)

			// node2 should be running
			node2Data, err := manager.GetNodeByID(node2.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusRunning, node2Data.Status)
		})
	}
}

// TestAutoCompleteParentWithCompletedNode tests that already completed nodes are not affected
func TestAutoCompleteParentWithCompletedNode(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add first node
			node1, err := manager.Add("step1", types.TraceNodeOption{
				Label: "Step 1",
			})
			assert.NoError(t, err)

			// Complete node1 manually with output
			err = node1.Complete(map[string]any{"result": "manual"})
			assert.NoError(t, err)

			// Verify node1 is completed with output
			node1Data, err := manager.GetNodeByID(node1.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusCompleted, node1Data.Status)
			assert.Equal(t, map[string]any{"result": "manual"}, node1Data.Output)

			// Add second node with auto-complete enabled
			node2, err := manager.Add("step2", types.TraceNodeOption{
				Label: "Step 2",
			})
			assert.NoError(t, err)

			// node1 should still be completed with same output
			node1Data, err = manager.GetNodeByID(node1.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusCompleted, node1Data.Status)
			assert.Equal(t, map[string]any{"result": "manual"}, node1Data.Output)

			// node2 should be running
			node2Data, err := manager.GetNodeByID(node2.ID())
			assert.NoError(t, err)
			assert.Equal(t, types.StatusRunning, node2Data.Status)
		})
	}
}

// TestAutoCompleteParentEvents tests that auto-complete generates proper events
func TestAutoCompleteParentEvents(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Subscribe to updates
			updates, err := manager.Subscribe()
			assert.NoError(t, err)

			// Collect updates in background
			var receivedUpdates []*types.TraceUpdate
			var updatesMu sync.Mutex
			done := make(chan bool)

			go func() {
				timeout := time.After(2 * time.Second)
				for {
					select {
					case update, ok := <-updates:
						if !ok {
							done <- true
							return
						}
						updatesMu.Lock()
						receivedUpdates = append(receivedUpdates, update)
						updatesMu.Unlock()
					case <-timeout:
						done <- true
						return
					}
				}
			}()

			// Add first node
			node1, err := manager.Add("step1", types.TraceNodeOption{
				Label: "Step 1",
			})
			assert.NoError(t, err)

			// Small delay
			time.Sleep(50 * time.Millisecond)

			// Add second node - should trigger auto-complete of node1
			_, err = manager.Add("step2", types.TraceNodeOption{
				Label: "Step 2",
			})
			assert.NoError(t, err)

			// Wait a bit for events to be processed
			time.Sleep(100 * time.Millisecond)

			// Wait for goroutine to finish (with timeout)
			<-done

			// Verify we received auto-complete event for node1
			updatesMu.Lock()
			defer updatesMu.Unlock()

			var foundNode1Start bool
			var foundNode1Complete bool
			var foundNode2Start bool

			for _, update := range receivedUpdates {
				switch update.Type {
				case types.UpdateTypeNodeStart:
					if data, ok := update.Data.(*types.NodeStartData); ok && data.Node != nil {
						if data.Node.ID == node1.ID() {
							foundNode1Start = true
						} else if data.Node.Label == "Step 2" {
							foundNode2Start = true
						}
					}
				case types.UpdateTypeNodeComplete:
					if data, ok := update.Data.(*types.NodeCompleteData); ok {
						if data.NodeID == node1.ID() {
							foundNode1Complete = true
						}
					}
				}
			}

			assert.True(t, foundNode1Start, "Should receive node_start event for node1")
			assert.True(t, foundNode1Complete, "Should receive node_complete event for node1 (auto-completed)")
			assert.True(t, foundNode2Start, "Should receive node_start event for node2")
		})
	}
}

// TestAutoCompleteParentSequentialChain tests a long sequential chain
func TestAutoCompleteParentSequentialChain(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create a chain of 10 nodes
			nodeCount := 10
			nodes := make([]types.Node, nodeCount)

			for i := 0; i < nodeCount; i++ {
				node, err := manager.Add("input", types.TraceNodeOption{
					Label: "Step " + string(rune('0'+i)),
				})
				assert.NoError(t, err)
				nodes[i] = node

				// All previous nodes should be completed
				for j := 0; j < i; j++ {
					nodeData, err := manager.GetNodeByID(nodes[j].ID())
					assert.NoError(t, err)
					assert.Equal(t, types.StatusCompleted, nodeData.Status,
						"Node %d should be completed when node %d is added", j, i)
				}

				// Current node should be running
				currentData, err := manager.GetNodeByID(node.ID())
				assert.NoError(t, err)
				assert.Equal(t, types.StatusRunning, currentData.Status)
			}

			// Get all nodes
			allNodes, err := manager.GetAllNodes()
			assert.NoError(t, err)
			assert.Len(t, allNodes, nodeCount)

			// All except the last should be completed
			completedCount := 0
			runningCount := 0
			for _, nodeData := range allNodes {
				if nodeData.Status == types.StatusCompleted {
					completedCount++
				} else if nodeData.Status == types.StatusRunning {
					runningCount++
				}
			}
			assert.Equal(t, nodeCount-1, completedCount, "Should have %d completed nodes", nodeCount-1)
			assert.Equal(t, 1, runningCount, "Should have 1 running node")
		})
	}
}
