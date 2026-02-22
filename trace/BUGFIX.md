# Trace Module Bug Analysis & Fix

## 1. Problem Summary

`yao start` crashes with `SIGSEGV` when Agent `All()` runs concurrent operations.
Root cause: three inter-related race conditions in the trace module's channel
lifecycle management, triggered when `agent/context.Release()` calls
`trace.MarkCancelled/MarkComplete` followed by `trace.Release`.

## 2. Bug Chain

```
context.Release()
  -> MarkCancelled/MarkComplete  (many safeSend calls to stateCmdChan)
    -> stateMarkCompleted triggers state.completed = true
    -> state worker starts 100ms drain then exits (BUG 1)
  -> trace.Release(traceID)
    -> close(mgr.stateCmdChan) (BUG 3: races with concurrent safeSend)
    -> if stateExecuteSpaceOp also writing (BUG 2: bare channel send)
  -> panic on closed channel
  -> in CGO/V8 callback stack, recover() may fail -> SIGSEGV
```

### BUG 1: State worker premature exit (state.go)

`startStateWorker` exits after `state.completed = true` + 100ms drain, but the
channel remains open. Subsequent safeSend calls write to an unconsumed channel.
If the buffer (100) fills up, safeSend blocks forever. Commands with response
channels (e.g. stateMarkCompleted) deadlock permanently.

### BUG 2: stateExecuteSpaceOp bare channel write (state.go)

```go
m.stateCmdChan <- &cmdSpaceKVOp{...}  // no safeSend, panics on closed channel
```

### BUG 3: safeSend vs close race (state.go + trace.go)

Even with `defer/recover`, the window between `select` choosing the send case
and the actual send allows a concurrent `close()` to trigger a panic that
cannot be recovered in CGO callback stacks.

## 3. Triggering Scenario (All() + context.Release)

```
Parent ctx (llm/process.go or caller/process.go)
  ├── Fork child ctx1 -> goroutine 1 (Orchestrator.All)
  ├── Fork child ctx2 -> goroutine 2
  └── All returns, defer ctx.Release()

Key: Fork sets trace=nil, but ForkParent.TraceID points to same traceID.
All goroutines share one trace manager, writing to one stateCmdChan.

ctx.Release():
  1. MarkCancelled/MarkComplete -> many safeSend calls
  2. trace.Release -> close(stateCmdChan) + cancel()

If child goroutines still have residual operations:
  -> safeSend to closed channel -> panic -> SIGSEGV
```

## 4. Fix Applied

### FIX 1: State worker uses for-range (state.go)

Removed the premature exit after `state.completed`. Worker now uses idiomatic
`for cmd := range m.stateCmdChan` which only exits when the channel is closed
by `Release()`. Go guarantees that `for range` drains all buffered commands
before returning.

### FIX 2: stateExecuteSpaceOp uses safeSend (state.go)

Replaced bare `m.stateCmdChan <- cmd` with `m.safeSend(cmd)`. Returns a
descriptive error when the state worker has stopped.

### FIX 3: Three-step safe shutdown (manager.go + state.go + trace.go)

Added `closed int32` atomic flag to `manager` struct. Release now follows a
strict three-step sequence:

1. `atomic.StoreInt32(&mgr.closed, 1)` — blocks new `safeSend` calls
2. `mgr.cancel()` — unblocks any `safeSend` stuck in `select` via `ctx.Done`
3. `close(mgr.stateCmdChan)` — terminates state worker (drains buffer first)

`safeSend` checks the atomic flag before touching the channel, providing a
fast-path rejection that is safe even in CGO callback stacks where `recover()`
may not work.

### FIX 4: context.Release timing (no code change needed)

`MarkCancelled/MarkComplete` calls are synchronous (wait for resp channel).
The only fire-and-forget call is `stateAddUpdate` inside `addUpdateAndBroadcast`.
Under FIX 3 protection, this call returns `false` instead of panicking if
Release has already started. Losing one update during shutdown is acceptable.

## 5. Files Modified

| File | Change |
|------|--------|
| trace/state.go | FIX 1 (for range) + FIX 2 (safeSend for SpaceOp) + FIX 3 (atomic check in safeSend) |
| trace/manager.go | FIX 3 (closed int32 field) |
| trace/trace.go | FIX 3 (three-step Release shutdown) |
| trace/trace_lifecycle_test.go | NEW: 7 boundary condition tests |

## 6. Test Coverage

New tests in `trace_lifecycle_test.go`:

- **TestReleaseWhileWriting** — 20 writers + concurrent Release
- **TestReleaseDuringSpaceOp** — space KV ops + concurrent Release
- **TestReleaseAfterMarkComplete** — MarkComplete -> Release -> late operations
- **TestConcurrentReleaseAndMarkCancelled** — MarkCancelled and Release race
- **TestSafeSendAfterClosed** — operations after closed flag is set
- **TestRapidCreateReleaseLoop** — 100x create/release stress test
- **TestConcurrentAllPattern** — simulates real All() fork + parent Release

All existing tests pass unchanged (interfaces not modified).

---

_Last updated: February 2026_
