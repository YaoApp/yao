package trace_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

func TestSpaceOperations(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create space
			space, err := manager.CreateSpace(types.TraceSpaceOption{
				Label:       "Test Space",
				Icon:        "database",
				Description: "Test space for unit tests",
				TTL:         3600,
			})
			assert.NoError(t, err)
			assert.NotNil(t, space)
			assert.NotEmpty(t, space.ID)

			// Set values
			err = manager.SetSpaceValue(space.ID, "key1", "value1")
			assert.NoError(t, err)

			err = manager.SetSpaceValue(space.ID, "key2", map[string]any{"nested": "data"})
			assert.NoError(t, err)

			err = manager.SetSpaceValue(space.ID, "key3", 12345)
			assert.NoError(t, err)

			// Get values
			val1, err := manager.GetSpaceValue(space.ID, "key1")
			assert.NoError(t, err)
			assert.Equal(t, "value1", val1)

			val2, err := manager.GetSpaceValue(space.ID, "key2")
			assert.NoError(t, err)
			assert.NotNil(t, val2)

			// Has value
			exists := manager.HasSpaceValue(space.ID, "key1")
			assert.True(t, exists)

			exists = manager.HasSpaceValue(space.ID, "nonexistent")
			assert.False(t, exists)

			// List keys
			keys := manager.ListSpaceKeys(space.ID)
			assert.Len(t, keys, 3)

			// Delete value
			err = manager.DeleteSpaceValue(space.ID, "key1")
			assert.NoError(t, err)

			exists = manager.HasSpaceValue(space.ID, "key1")
			assert.False(t, exists)

			// Clear all values
			err = manager.ClearSpaceValues(space.ID)
			assert.NoError(t, err)

			keys = manager.ListSpaceKeys(space.ID)
			assert.Empty(t, keys)

			// List spaces
			spaces := manager.ListSpaces()
			assert.NotEmpty(t, spaces)
			assert.True(t, manager.HasSpace(space.ID))

			// Delete space
			err = manager.DeleteSpace(space.ID)
			assert.NoError(t, err)

			assert.False(t, manager.HasSpace(space.ID))
		})
	}
}

func TestMultipleSpaces(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create multiple spaces
			space1, err := manager.CreateSpace(types.TraceSpaceOption{
				Label: "Context",
				Icon:  "context",
			})
			assert.NoError(t, err)

			space2, err := manager.CreateSpace(types.TraceSpaceOption{
				Label: "Memory",
				Icon:  "memory",
			})
			assert.NoError(t, err)

			space3, err := manager.CreateSpace(types.TraceSpaceOption{
				Label: "Cache",
				Icon:  "cache",
			})
			assert.NoError(t, err)

			// Set values in different spaces
			err = manager.SetSpaceValue(space1.ID, "context_key", "context_value")
			assert.NoError(t, err)

			err = manager.SetSpaceValue(space2.ID, "memory_key", "memory_value")
			assert.NoError(t, err)

			err = manager.SetSpaceValue(space3.ID, "cache_key", "cache_value")
			assert.NoError(t, err)

			// Verify isolation
			val1, err := manager.GetSpaceValue(space1.ID, "context_key")
			assert.NoError(t, err)
			assert.Equal(t, "context_value", val1)

			// Key from space1 should not exist in space2
			exists := manager.HasSpaceValue(space2.ID, "context_key")
			assert.False(t, exists)

			// List all spaces
			spaces := manager.ListSpaces()
			assert.Len(t, spaces, 3)
		})
	}
}

func TestSpaceGetSpace(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create space
			space, err := manager.CreateSpace(types.TraceSpaceOption{
				Label:       "Test Space",
				Description: "Test description",
				TTL:         7200,
			})
			assert.NoError(t, err)

			// Get space by ID
			retrieved, err := manager.GetSpace(space.ID)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved)
			assert.Equal(t, space.ID, retrieved.ID)
			assert.Equal(t, "Test Space", retrieved.Label)
			assert.Equal(t, "Test description", retrieved.Description)
			assert.Equal(t, int64(7200), retrieved.TTL)

			// Get non-existent space (returns nil, nil)
			nonExistent, err := manager.GetSpace("nonexistent")
			assert.NoError(t, err)
			assert.Nil(t, nonExistent)
		})
	}
}
