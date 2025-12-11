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

func TestSubscription(t *testing.T) {
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
			assert.NotNil(t, updates)

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
							// Channel closed
							done <- true
							return
						}

						updatesMu.Lock()
						receivedUpdates = append(receivedUpdates, update)
						updatesMu.Unlock()

						// Check for trace completion
						if update.Type == types.UpdateTypeComplete {
							done <- true
							return
						}
					case <-timeout:
						done <- true
						return
					}
				}
			}()

			// Perform operations
			manager.Info("Test operation")
			_, err = manager.Add("test node", types.TraceNodeOption{Label: "Test"})
			assert.NoError(t, err)

			err = manager.Complete(map[string]any{"test": "data"})
			assert.NoError(t, err)

			// Create space and set value
			space, err := manager.CreateSpace(types.TraceSpaceOption{Label: "Test Space"})
			assert.NoError(t, err)

			err = manager.SetSpaceValue(space.ID, "key", "value")
			assert.NoError(t, err)

			// Mark trace complete
			err = manager.MarkComplete()
			assert.NoError(t, err)

			// Wait for completion or timeout
			<-done

			// Verify we received updates
			updatesMu.Lock()
			defer updatesMu.Unlock()

			assert.NotEmpty(t, receivedUpdates)

			// Check for specific event types
			eventTypes := make(map[string]bool)
			for _, update := range receivedUpdates {
				eventTypes[update.Type] = true
			}

			assert.True(t, eventTypes[types.UpdateTypeInit], "Should receive init event")
			assert.True(t, eventTypes[types.UpdateTypeNodeStart], "Should receive node_start event")
			assert.True(t, eventTypes[types.UpdateTypeComplete], "Should receive complete event")
		})
	}
}

func TestSubscribeFrom(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Real scenario: User starts a trace, performs some operations
			_, err = manager.Add("Step 1", types.TraceNodeOption{Label: "Processing"})
			assert.NoError(t, err)
			manager.Info("Processing step 1")
			err = manager.Complete("step1 result")
			assert.NoError(t, err)

			// Wait to ensure different timestamp (simulate time passing)
			// Use a longer sleep to account for CI environment variability
			time.Sleep(1100 * time.Millisecond)

			// Record timestamp (simulate user noting current time before refresh)
			// Subtract 1ms to ensure we capture events that happen "now"
			// This accounts for millisecond precision and timing variability in CI
			resumeTimestamp := time.Now().UnixMilli() - 1

			// Wait to ensure next operations have a clearly different timestamp
			// Use longer sleep for CI reliability
			time.Sleep(500 * time.Millisecond)

			// Continue with more operations
			_, err = manager.Add("Step 2", types.TraceNodeOption{Label: "Finalizing"})
			assert.NoError(t, err)
			manager.Info("Processing step 2")
			err = manager.Complete("step2 result")
			assert.NoError(t, err)

			// Mark trace complete
			err = manager.MarkComplete()
			assert.NoError(t, err)

			// Real scenario: User refreshes page and resumes from last known timestamp
			// This should replay events from resumeTimestamp onwards
			updates, err := manager.SubscribeFrom(resumeTimestamp)
			assert.NoError(t, err)
			assert.NotNil(t, updates)

			// Collect updates
			var receivedUpdates []*types.TraceUpdate
			timeout := time.After(1 * time.Second)
			foundStep2 := false

		collectLoop:
			for {
				select {
				case update, ok := <-updates:
					if !ok {
						// Channel closed
						break collectLoop
					}
					receivedUpdates = append(receivedUpdates, update)
					// Check if we received step 2 events
					if update.Type == types.UpdateTypeNodeStart {
						if data, ok := update.Data.(*types.NodeStartData); ok {
							if data.Node != nil && data.Node.Label == "Finalizing" {
								foundStep2 = true
							}
						}
					}
					// Stop after receiving trace_complete
					if update.Type == types.UpdateTypeComplete {
						break collectLoop
					}
				case <-timeout:
					break collectLoop
				}
			}

			// Verify we received events from step 2 onwards
			assert.NotEmpty(t, receivedUpdates, "Should receive events from resume point")
			assert.True(t, foundStep2, "Should receive Step 2 events")

			// All events should be at or after the resume timestamp
			for _, update := range receivedUpdates {
				assert.GreaterOrEqual(t, update.Timestamp, resumeTimestamp,
					"Event timestamp %d should be >= resume timestamp %d (event type: %s)",
					update.Timestamp, resumeTimestamp, update.Type)
			}
		})
	}
}

func TestIsComplete(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Initially not complete
			assert.False(t, manager.IsComplete())

			// Mark complete
			err = manager.MarkComplete()
			assert.NoError(t, err)

			// Now should be complete
			assert.True(t, manager.IsComplete())
		})
	}
}

func TestMultipleSubscribers(t *testing.T) {
	drivers := trace.GetTestDrivers()

	for _, d := range drivers {
		t.Run(d.Name, func(t *testing.T) {
			ctx := context.Background()

			traceID, manager, err := trace.New(ctx, d.DriverType, nil, d.DriverOptions...)
			assert.NoError(t, err)
			defer trace.Release(traceID)
			defer trace.Remove(ctx, d.DriverType, traceID, d.DriverOptions...)

			// Create multiple subscribers
			sub1, err := manager.Subscribe()
			assert.NoError(t, err)

			sub2, err := manager.Subscribe()
			assert.NoError(t, err)

			sub3, err := manager.Subscribe()
			assert.NoError(t, err)

			// Collect updates from all subscribers
			var wg sync.WaitGroup
			counts := make([]int, 3)
			var mu sync.Mutex

			for i, sub := range []<-chan *types.TraceUpdate{sub1, sub2, sub3} {
				wg.Add(1)
				go func(idx int, ch <-chan *types.TraceUpdate) {
					defer wg.Done()
					timeout := time.After(1 * time.Second)
					for {
						select {
						case update := <-ch:
							if update != nil {
								mu.Lock()
								counts[idx]++
								mu.Unlock()
								if update.Type == types.UpdateTypeComplete {
									return
								}
							}
						case <-timeout:
							return
						}
					}
				}(i, sub)
			}

			// Perform operations
			_, err = manager.Add("test", types.TraceNodeOption{Label: "Test"})
			assert.NoError(t, err)
			err = manager.Complete(nil)
			assert.NoError(t, err)
			err = manager.MarkComplete()
			assert.NoError(t, err)

			// Wait for all subscribers
			wg.Wait()

			// All subscribers should receive updates
			mu.Lock()
			defer mu.Unlock()
			for i, count := range counts {
				assert.Greater(t, count, 0, "Subscriber %d should receive updates", i+1)
			}
		})
	}
}
