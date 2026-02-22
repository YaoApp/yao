# Trace Module - Known Issues

This document tracks known issues in the Trace module that are scheduled for refactoring.

## Goroutine Leak (Mitigated)

### Symptom

Each trace creation starts 2 goroutines that accumulate during rapid iterations:

1. `trace/pubsub.(*PubSub).forward()` - PubSub event forwarding
2. `trace.(*manager).startStateWorker()` - State machine worker

### Current Status

**Fixed in Feb 2026**: State worker now uses `for range` on the command channel,
which exits cleanly when `Release()` closes the channel. The three-step shutdown
sequence (atomic flag -> cancel -> close) ensures:

- No new commands are accepted after the flag is set
- In-flight `safeSend` calls unblock via `ctx.Done`
- Worker drains remaining buffered commands before exiting

### Residual Behavior

- PubSub `forward()` goroutine still exits asynchronously after `Stop()` closes its channel
- In rapid create/release loops, there may be a brief overlap where old goroutines
  haven't exited before new ones start. This is normal Go async cleanup behavior.
- **NOT a true leak**: goroutines eventually exit (channels are closed)

### Impact on Tests

Memory leak tests use a 20KB/iteration threshold to accommodate this overhead:

| Test              | Actual Growth  | Threshold |
| ----------------- | -------------- | --------- |
| StandardMode      | ~15 bytes/iter | 20 KB     |
| BusinessScenarios | ~80 bytes/iter | 20 KB     |
| NestedCalls       | ~13 KB/iter    | 20 KB     |

## Memory Growth (Reduced)

### Symptom

Linear memory growth during trace operations.

### Current Status

**Improved in Feb 2026**: The state worker lifecycle fix eliminates the scenario
where the worker exits prematurely while the channel remains open, which could
cause command objects to accumulate in the buffer without being consumed.

The three-step shutdown ensures all buffered commands are processed before the
worker exits, reducing memory retention from unconsumed channel entries.

### Residual Growth Sources

- PubSub subscription objects (cleaned up on Stop)
- Driver I/O buffers (transient, GC-eligible)
- Trace node references in memory state (released on worker exit)

### Workaround

The 20KB threshold in memory leak tests accommodates known overhead while still
detecting severe leaks (50KB+ growth would indicate a real problem).

## Planned Refactoring

The Trace module is scheduled for further refactoring:

1. **Global Event Service**: Decouple event broadcasting from trace manager into
   a process-level daemon (separate plan)
2. **Resource pooling**: Consider reusing trace resources to reduce allocation overhead
3. **PubSub synchronous cleanup**: Ensure forward() goroutine exits before Stop() returns

## Testing Notes

When running memory leak tests:

- `TestMemoryLeakStandardMode`: 20KB threshold
- `TestMemoryLeakBusinessScenarios`: 20KB threshold
- `TestMemoryLeakNestedCalls`: 20KB threshold
- `TestMemoryLeakNestedConcurrent`: 25KB threshold (concurrent + DB operations)

These thresholds are intentionally higher than actual growth to:

1. Accommodate CI environment variations
2. Allow for GC timing differences
3. Still catch severe leaks (50KB+ would be concerning)

## Related Files

- `trace/manager.go` - State machine and goroutine management
- `trace/state.go` - Channel-based state worker and safeSend
- `trace/trace.go` - Release() three-step shutdown
- `trace/pubsub/pubsub.go` - PubSub forwarding goroutine
- `trace/trace_lifecycle_test.go` - Boundary condition tests for shutdown races
- `trace/BUGFIX.md` - Detailed bug analysis and fix documentation

---

_Last updated: February 2026_
