package trace_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

func TestArchive(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Create a trace with AutoArchive disabled
			traceID, manager, err := trace.New(ctx, d.DriverType, &types.TraceOption{
				AutoArchive: false,
			}, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create some test data
			_, err = manager.Add("test input", types.TraceNodeOption{
				Label: "Test Node",
			})
			assert.NoError(t, err)

			manager.Info("Test log message", map[string]any{
				"key": "value",
			})

			err = manager.Complete()
			assert.NoError(t, err)

			err = manager.MarkComplete()
			assert.NoError(t, err)

			// Wait a bit for completion
			time.Sleep(100 * time.Millisecond)

			// Note: Archive functionality is tested at the driver level
			// Here we just verify that traces can be created and completed
			// without AutoArchive enabled
		})
	}
}

func TestAutoArchive(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			// Create a trace with AutoArchive enabled
			traceID, manager, err := trace.New(ctx, d.DriverType, &types.TraceOption{
				AutoArchive: true,
			}, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create some test data
			_, err = manager.Add("test input", types.TraceNodeOption{
				Label: "Test Node",
			})
			assert.NoError(t, err)

			manager.Info("Test log message", map[string]any{
				"key": "value",
			})

			err = manager.Complete()
			assert.NoError(t, err)

			err = manager.MarkComplete()
			assert.NoError(t, err)

			// Wait for auto-archive to complete
			time.Sleep(200 * time.Millisecond)

			// Trace should still be accessible after auto-archive
			assert.True(t, trace.IsLoaded(traceID))
		})
	}
}
