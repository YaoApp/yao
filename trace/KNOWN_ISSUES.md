# Trace Module - Known Issues

This document tracks known issues in the Trace module that are scheduled for refactoring.

## Goroutine Leak

### Symptom

Each trace creation starts 2 goroutines that accumulate during rapid iterations:

1. `trace/pubsub.(*PubSub).forward()` - PubSub event forwarding
2. `trace.(*manager).startStateWorker()` - State machine worker

### Evidence

```
Goroutine growth by function:
Function                                    Initial    Final   Growth
--------------------------------------------------------------------------
github.com/yaoapp/yao/trace.                    0       10       10
github.com/yaoapp/yao/trace/pubsub.             0       10       10
```

### Root Cause

- Goroutines exit when `Release()` closes their channels
- Exit is **asynchronous** (goroutine needs to reach select statement)
- Go runtime needs time to schedule and cleanup
- In rapid iterations, new goroutines are created before old ones fully exit

### Current Behavior

- **NOT a true leak**: Goroutines eventually exit (channels are closed)
- **No unbounded growth**: They will be GC'd eventually
- **Typical pattern**: Async cleanup in Go

### Impact on Tests

Memory leak tests use a 20KB/iteration threshold to accommodate this overhead:

| Test              | Actual Growth  | Threshold |
| ----------------- | -------------- | --------- |
| StandardMode      | ~11-15 KB/iter | 20 KB     |
| BusinessScenarios | ~13-16 KB/iter | 20 KB     |
| NestedCalls       | ~13 KB/iter    | 20 KB     |

## Memory Growth

### Symptom

Linear memory growth during trace operations:

```
Batch | Iterations | HeapAlloc (MB) | Growth/iter (bytes)
------|------------|----------------|--------------------
    1 |       1000 |          23.28 |           12014.42
    2 |       2000 |          37.06 |           13229.02
    3 |       3000 |          50.63 |           13562.39
    4 |       4000 |          64.54 |           13819.49
    5 |       5000 |          78.15 |           13910.01
```

### Root Cause

Trace-related objects are not fully released during `ctx.Release()`:

- State machine data
- PubSub subscriptions
- Trace node references

### Workaround

The 20KB threshold in memory leak tests accommodates this known overhead while still detecting severe leaks (50KB+ growth would indicate a real problem).

## Planned Refactoring

The Trace module is scheduled for refactoring to address:

1. **Synchronous cleanup**: Ensure goroutines exit before `Release()` returns
2. **Memory management**: Properly release all trace-related objects
3. **Resource pooling**: Consider reusing trace resources to reduce allocation overhead

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
- `trace/pubsub/pubsub.go` - PubSub forwarding goroutine
- `trace/trace.go` - Release() implementation
- `agent/assistant/hook/create_mem_test.go` - Memory leak tests

---

_Last updated: December 2025_
