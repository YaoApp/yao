package trace_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// Note: TestMain is defined in trace_basic_test.go and applies to all tests in this package

func TestManagerGetTraceInfo(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Create trace with custom metadata
			option := &types.TraceOption{
				CreatedBy: "test@example.com",
				TeamID:    "team-001",
				TenantID:  "tenant-001",
				Metadata:  map[string]any{"test_key": "test_value"},
			}

			traceID, manager, err := trace.New(ctx, d.DriverType, option, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Get trace info through manager
			info, err := manager.GetTraceInfo()
			assert.NoError(t, err)
			assert.NotNil(t, info)
			assert.Equal(t, traceID, info.ID)
			assert.Equal(t, "test@example.com", info.CreatedBy)
			assert.Equal(t, "team-001", info.TeamID)
			assert.Equal(t, "tenant-001", info.TenantID)
			assert.Equal(t, "test_value", info.Metadata["test_key"])
			assert.Equal(t, d.DriverType, info.Driver)
		})
	}
}

func TestManagerGetAllNodes(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Initially no nodes
			nodes, err := manager.GetAllNodes()
			assert.NoError(t, err)
			assert.Empty(t, nodes)

			// Add root node
			node1, err := manager.Add("input1", types.TraceNodeOption{Label: "Node 1", Icon: "icon1"})
			assert.NoError(t, err)

			// Add child node
			node2, err := manager.Add("input2", types.TraceNodeOption{Label: "Node 2", Icon: "icon2"})
			assert.NoError(t, err)

			// Add another child
			node3, err := manager.Add("input3", types.TraceNodeOption{Label: "Node 3", Icon: "icon3"})
			assert.NoError(t, err)

			// Complete nodes to ensure they are fully persisted
			err = node3.Complete()
			assert.NoError(t, err)
			err = node2.Complete()
			assert.NoError(t, err)
			err = node1.Complete()
			assert.NoError(t, err)

			// Get all nodes
			nodes, err = manager.GetAllNodes()
			assert.NoError(t, err)
			assert.Len(t, nodes, 3)

			// Verify node IDs are present
			nodeIDs := make(map[string]bool)
			for _, node := range nodes {
				nodeIDs[node.ID] = true
			}
			assert.True(t, nodeIDs[node1.ID()])
			assert.True(t, nodeIDs[node2.ID()])
			assert.True(t, nodeIDs[node3.ID()])

			// Verify node labels
			nodeLabels := make(map[string]string)
			for _, node := range nodes {
				nodeLabels[node.ID] = node.Label
			}
			assert.Equal(t, "Node 1", nodeLabels[node1.ID()])
			assert.Equal(t, "Node 2", nodeLabels[node2.ID()])
			assert.Equal(t, "Node 3", nodeLabels[node3.ID()])
		})
	}
}

func TestManagerGetNodeByID(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add a node
			node, err := manager.Add("test input", types.TraceNodeOption{
				Label:       "Test Node",
				Icon:        "test",
				Description: "Test Description",
			})
			assert.NoError(t, err)
			nodeID := node.ID()

			// Get node by ID
			retrievedNode, err := manager.GetNodeByID(nodeID)
			assert.NoError(t, err)
			assert.NotNil(t, retrievedNode)
			assert.Equal(t, nodeID, retrievedNode.ID)
			assert.Equal(t, "Test Node", retrievedNode.Label)
			assert.Equal(t, "test", retrievedNode.Icon)
			assert.Equal(t, "Test Description", retrievedNode.Description)
			assert.Equal(t, "test input", retrievedNode.Input)

			// Try to get non-existent node (should return error or nil)
			nonExistentNode, err := manager.GetNodeByID("non_existent_id")
			if err == nil {
				// If no error, node should be nil
				assert.Nil(t, nonExistentNode)
			} else {
				// If error, that's also acceptable
				assert.Nil(t, nonExistentNode)
			}
		})
	}
}

func TestManagerGetAllLogs(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Initially no logs
			logs, err := manager.GetAllLogs()
			assert.NoError(t, err)
			assert.Empty(t, logs)

			// Add a node and log some messages
			node, err := manager.Add("test", types.TraceNodeOption{Label: "Test Node"})
			assert.NoError(t, err)

			// Log different levels
			node.Info("Info message", map[string]any{"key1": "value1"})
			node.Debug("Debug message", map[string]any{"key2": "value2"})
			node.Warn("Warning message", map[string]any{"key3": "value3"})
			node.Error("Error message", map[string]any{"key4": "value4"})

			// Get all logs
			logs, err = manager.GetAllLogs()
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(logs), 4)

			// Verify log levels
			levels := make(map[string]int)
			for _, log := range logs {
				levels[log.Level]++
			}
			assert.GreaterOrEqual(t, levels["info"], 1)
			assert.GreaterOrEqual(t, levels["debug"], 1)
			assert.GreaterOrEqual(t, levels["warn"], 1)
			assert.GreaterOrEqual(t, levels["error"], 1)

			// Verify log messages
			messages := make([]string, 0)
			for _, log := range logs {
				messages = append(messages, log.Message)
			}
			assert.Contains(t, messages, "Info message")
			assert.Contains(t, messages, "Debug message")
			assert.Contains(t, messages, "Warning message")
			assert.Contains(t, messages, "Error message")
		})
	}
}

func TestManagerGetLogsByNode(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add two nodes
			node1, err := manager.Add("test1", types.TraceNodeOption{Label: "Node 1"})
			assert.NoError(t, err)

			node2, err := manager.Add("test2", types.TraceNodeOption{Label: "Node 2"})
			assert.NoError(t, err)

			// Log to node1
			node1.Info("Node 1 message 1")
			node1.Debug("Node 1 message 2")

			// Log to node2
			node2.Info("Node 2 message 1")
			node2.Warn("Node 2 message 2")
			node2.Error("Node 2 message 3")

			// Get logs for node1
			logs1, err := manager.GetLogsByNode(node1.ID())
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(logs1), 2)

			// Verify all logs belong to node1
			for _, log := range logs1 {
				assert.Equal(t, node1.ID(), log.NodeID)
			}

			// Get logs for node2
			logs2, err := manager.GetLogsByNode(node2.ID())
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(logs2), 3)

			// Verify all logs belong to node2
			for _, log := range logs2 {
				assert.Equal(t, node2.ID(), log.NodeID)
			}

			// Verify node1 and node2 logs are different
			assert.NotEqual(t, len(logs1), len(logs2))
		})
	}
}

func TestManagerResourceAccessAfterLoadFromStorage(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Create trace with metadata
			traceID, manager, err := trace.New(ctx, d.DriverType, &types.TraceOption{
				CreatedBy: "test@example.com",
				TeamID:    "team-001",
				TenantID:  "tenant-001",
				Metadata:  map[string]any{"test_key": "test_value"},
			}, d.DriverOptions...)
			assert.NoError(t, err)

			// Add root node
			node1, err := manager.Add("input1", types.TraceNodeOption{
				Label:       "Root Node",
				Icon:        "root",
				Description: "Root node description",
			})
			assert.NoError(t, err)
			node1.Info("Root node info log", map[string]any{"data": "info1"})
			node1.Debug("Root node debug log", map[string]any{"data": "debug1"})

			// Add child node
			node2, err := manager.Add("input2", types.TraceNodeOption{
				Label:       "Child Node",
				Icon:        "child",
				Description: "Child node description",
			})
			assert.NoError(t, err)
			node2.Info("Child node info log", map[string]any{"data": "info2"})
			node2.Warn("Child node warning log", map[string]any{"data": "warn2"})

			// Add another child node
			node3, err := manager.Add("input3", types.TraceNodeOption{
				Label:       "Second Child Node",
				Icon:        "child2",
				Description: "Second child description",
			})
			assert.NoError(t, err)
			node3.Error("Child node error log", map[string]any{"data": "error3"})

			// Complete nodes to ensure data is persisted
			err = node3.Complete(map[string]any{"result": "success3"})
			assert.NoError(t, err)
			err = node2.Complete(map[string]any{"result": "success2"})
			assert.NoError(t, err)
			err = node1.Complete(map[string]any{"result": "success1"})
			assert.NoError(t, err)

			// Release from registry
			err = trace.Release(traceID)
			assert.NoError(t, err)

			// Load from storage
			_, loadedManager, err := trace.LoadFromStorage(ctx, d.DriverType, traceID, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, loadedManager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Test GetTraceInfo
			info, err := loadedManager.GetTraceInfo()
			assert.NoError(t, err)
			assert.Equal(t, traceID, info.ID)
			assert.Equal(t, "test@example.com", info.CreatedBy)
			assert.Equal(t, "team-001", info.TeamID)
			assert.Equal(t, "tenant-001", info.TenantID)
			assert.Equal(t, "test_value", info.Metadata["test_key"])

			// Test GetAllNodes
			nodes, err := loadedManager.GetAllNodes()
			assert.NoError(t, err)
			assert.Len(t, nodes, 3, "Should have 3 nodes")

			// Verify node labels
			nodeLabels := make(map[string]bool)
			for _, node := range nodes {
				nodeLabels[node.Label] = true
			}
			assert.True(t, nodeLabels["Root Node"])
			assert.True(t, nodeLabels["Child Node"])
			assert.True(t, nodeLabels["Second Child Node"])

			// Test GetNodeByID
			retrievedNode, err := loadedManager.GetNodeByID(nodes[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, retrievedNode)
			assert.Equal(t, nodes[0].Label, retrievedNode.Label)

			// Test GetAllLogs (should have at least 5 logs)
			logs, err := loadedManager.GetAllLogs()
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(logs), 5, "Should have at least 5 logs")

			// Verify different log levels exist
			logLevels := make(map[string]bool)
			for _, log := range logs {
				logLevels[log.Level] = true
			}
			assert.True(t, logLevels["info"], "Should have info logs")
			assert.True(t, logLevels["debug"], "Should have debug logs")
			assert.True(t, logLevels["warn"], "Should have warn logs")
			assert.True(t, logLevels["error"], "Should have error logs")

			// Test GetLogsByNode (get logs for first node)
			nodeLogs, err := loadedManager.GetLogsByNode(nodes[0].ID)
			assert.NoError(t, err)
			assert.NotEmpty(t, nodeLogs)
			// Verify all logs belong to the same node
			for _, log := range nodeLogs {
				assert.Equal(t, nodes[0].ID, log.NodeID)
			}
		})
	}
}

func TestManagerGetEventsWithResourceAccess(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Add nodes
			node1, err := manager.Add("test1", types.TraceNodeOption{Label: "Node 1"})
			assert.NoError(t, err)

			node1.Info("Test message")
			err = node1.Complete(map[string]any{"result": "success"})
			assert.NoError(t, err)

			// Get events
			events, err := manager.GetEvents(0)
			assert.NoError(t, err)
			assert.NotEmpty(t, events)

			// Verify event types
			eventTypes := make(map[string]bool)
			for _, event := range events {
				eventTypes[event.Type] = true
			}
			assert.True(t, eventTypes[types.UpdateTypeInit])
			assert.True(t, eventTypes[types.UpdateTypeNodeStart])
			assert.True(t, eventTypes[types.UpdateTypeLogAdded])
			assert.True(t, eventTypes[types.UpdateTypeNodeComplete])

			// Get all nodes - should match nodes in events
			nodes, err := manager.GetAllNodes()
			assert.NoError(t, err)
			assert.Len(t, nodes, 1)
			assert.Equal(t, node1.ID(), nodes[0].ID)

			// Get logs - should match log events
			logs, err := manager.GetAllLogs()
			assert.NoError(t, err)
			assert.NotEmpty(t, logs)
		})
	}
}

func TestManagerGetAllSpaces(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Initially no spaces
			spaces, err := manager.GetAllSpaces()
			assert.NoError(t, err)
			assert.Empty(t, spaces)

			// Create spaces
			space1, err := manager.CreateSpace(types.TraceSpaceOption{
				Label:       "Space 1",
				Icon:        "memory",
				Description: "First test space",
			})
			assert.NoError(t, err)

			space2, err := manager.CreateSpace(types.TraceSpaceOption{
				Label:       "Space 2",
				Icon:        "cache",
				Description: "Second test space",
			})
			assert.NoError(t, err)

			space3, err := manager.CreateSpace(types.TraceSpaceOption{
				Label:       "Space 3",
				Icon:        "store",
				Description: "Third test space",
			})
			assert.NoError(t, err)

			// Get all spaces
			spaces, err = manager.GetAllSpaces()
			assert.NoError(t, err)
			assert.Len(t, spaces, 3)

			// Verify space IDs
			spaceIDs := make(map[string]bool)
			for _, space := range spaces {
				spaceIDs[space.ID] = true
			}
			assert.True(t, spaceIDs[space1.ID])
			assert.True(t, spaceIDs[space2.ID])
			assert.True(t, spaceIDs[space3.ID])

			// Verify space labels
			spaceLabels := make(map[string]string)
			for _, space := range spaces {
				spaceLabels[space.ID] = space.Label
			}
			assert.Equal(t, "Space 1", spaceLabels[space1.ID])
			assert.Equal(t, "Space 2", spaceLabels[space2.ID])
			assert.Equal(t, "Space 3", spaceLabels[space3.ID])
		})
	}
}

func TestManagerGetSpaceByID(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create a space
			space, err := manager.CreateSpace(types.TraceSpaceOption{
				Label:       "Test Space",
				Icon:        "memory",
				Description: "Test space with data",
				Metadata:    map[string]any{"type": "cache"},
			})
			assert.NoError(t, err)

			// Set some key-value pairs
			err = manager.SetSpaceValue(space.ID, "key1", "value1")
			assert.NoError(t, err)
			err = manager.SetSpaceValue(space.ID, "key2", 123)
			assert.NoError(t, err)
			err = manager.SetSpaceValue(space.ID, "key3", map[string]any{"nested": "data"})
			assert.NoError(t, err)

			// Get space by ID with all data
			spaceData, err := manager.GetSpaceByID(space.ID)
			assert.NoError(t, err)
			assert.NotNil(t, spaceData)
			assert.Equal(t, space.ID, spaceData.ID)
			assert.Equal(t, "Test Space", spaceData.Label)
			assert.Equal(t, "memory", spaceData.Icon)
			assert.Equal(t, "Test space with data", spaceData.Description)
			assert.Equal(t, "cache", spaceData.Metadata["type"])

			// Verify key-value data
			assert.Len(t, spaceData.Data, 3)
			assert.Equal(t, "value1", spaceData.Data["key1"])
			// Note: Store driver may serialize numbers as float64 through JSON
			key2Value := spaceData.Data["key2"]
			if floatVal, ok := key2Value.(float64); ok {
				assert.Equal(t, float64(123), floatVal)
			} else {
				assert.Equal(t, 123, key2Value)
			}
			nestedData, ok := spaceData.Data["key3"].(map[string]any)
			assert.True(t, ok)
			assert.Equal(t, "data", nestedData["nested"])

			// Try to get non-existent space
			nonExistentSpace, err := manager.GetSpaceByID("non_existent_id")
			if err == nil {
				assert.Nil(t, nonExistentSpace)
			} else {
				assert.Nil(t, nonExistentSpace)
			}
		})
	}
}
