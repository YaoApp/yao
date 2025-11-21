package trace_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

func TestMain(m *testing.M) {
	// Prepare test environment (initializes stores, models, etc.)
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Run tests
	os.Exit(m.Run())
}

func TestTraceNew(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Create new trace
			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotEmpty(t, traceID)
			assert.NotNil(t, manager)

			// Clean up
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Verify trace is loaded
			assert.True(t, trace.IsLoaded(traceID))

			// Root node should be nil initially (lazy initialization)
			root, err := manager.GetRootNode()
			assert.NoError(t, err)
			assert.Nil(t, root)

			// Add first node - this should become the root
			node, err := manager.Add("test input", types.TraceNodeOption{
				Label: "First Node",
				Type:  "test",
				Icon:  "test",
			})
			assert.NoError(t, err)
			assert.NotNil(t, node)

			// Now root node should exist
			root, err = manager.GetRootNode()
			assert.NoError(t, err)
			assert.NotNil(t, root)
			assert.Equal(t, "First Node", root.Label)
			assert.Equal(t, "test", root.Type)
			assert.Equal(t, "test", root.Icon)
		})
	}
}

func TestTraceWithCustomID(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()
			customID := trace.GenTraceID()

			option := &types.TraceOption{
				ID:        customID,
				CreatedBy: "test@example.com",
				TeamID:    "team-001",
				TenantID:  "tenant-001",
				Metadata:  map[string]any{"test": "value"},
			}

			traceID, manager, err := trace.New(ctx, d.DriverType, option, d.DriverOptions...)
			assert.NoError(t, err)
			assert.Equal(t, customID, traceID)
			assert.NotNil(t, manager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Verify trace info
			info, err := trace.GetInfo(ctx, d.DriverType, traceID, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, info)
			assert.Equal(t, customID, info.ID)
			assert.Equal(t, "test@example.com", info.CreatedBy)
			assert.Equal(t, "team-001", info.TeamID)
			assert.Equal(t, "tenant-001", info.TenantID)
		})
	}
}

func TestTraceLoadFromStorage(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Create and persist a trace
			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)

			// Add some data
			_, err = manager.Add("test", types.TraceNodeOption{
				Label: "Test",
				Type:  "test_node",
				Icon:  "node",
			})
			assert.NoError(t, err)

			space, err := manager.CreateSpace(types.TraceSpaceOption{
				Label: "Test Space",
				Type:  "test_space",
				Icon:  "space",
			})
			assert.NoError(t, err)

			err = manager.SetSpaceValue(space.ID, "key", "value")
			assert.NoError(t, err)

			// Release from registry
			err = trace.Release(traceID)
			assert.NoError(t, err)
			assert.False(t, trace.IsLoaded(traceID))

			// Load from storage
			loadedTraceID, loadedManager, err := trace.LoadFromStorage(ctx, d.DriverType, traceID, d.DriverOptions...)
			assert.NoError(t, err)
			assert.Equal(t, traceID, loadedTraceID)
			assert.NotNil(t, loadedManager)

			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Verify loaded
			assert.True(t, trace.IsLoaded(traceID))

			// Verify data still exists
			spaces := loadedManager.ListSpaces()
			assert.NotEmpty(t, spaces)
		})
	}
}

func TestTraceExistsAndRemove(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			// Check exists
			exists, err := trace.Exists(ctx, d.DriverType, traceID, d.DriverOptions...)
			assert.NoError(t, err)
			assert.True(t, exists)

			// Remove trace
			err = trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)
			assert.NoError(t, err)

			// Check not exists
			exists, err = trace.Exists(ctx, d.DriverType, traceID, d.DriverOptions...)
			assert.NoError(t, err)
			assert.False(t, exists)

			// Check not loaded
			assert.False(t, trace.IsLoaded(traceID))
		})
	}
}

func TestTraceList(t *testing.T) {
	ctx := context.Background()

	// Create multiple traces
	var traces []string
	for i := 0; i < 3; i++ {
		traceID, _, err := trace.New(ctx, trace.Local, nil)
		assert.NoError(t, err)
		traces = append(traces, traceID)
	}

	// Clean up
	defer func() {
		for _, traceID := range traces {
			trace.Release(traceID)
			trace.Remove(ctx, trace.Local, traceID)
		}
	}()

	// List active traces
	activeTraces := trace.List()
	assert.GreaterOrEqual(t, len(activeTraces), 3)

	// Verify our traces are in the list
	for _, traceID := range traces {
		found := false
		for _, activeID := range activeTraces {
			if activeID == traceID {
				found = true
				break
			}
		}
		assert.True(t, found, "Trace %s should be in active list", traceID)
	}
}

func TestContextCancellation(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(context.Background(), d.DriverType, traceID, d.DriverOptions...)

			// Cancel context
			cancel()

			// Operations should fail with context error
			_, err = manager.Add("test", types.TraceNodeOption{
				Label: "Test",
				Type:  "test",
			})
			assert.Error(t, err)
		})
	}
}
