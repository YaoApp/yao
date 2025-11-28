# Context Resource Management

This document explains the resource management strategy for Context and Trace objects in JavaScript.

## Overview

Both `Context` and `Trace` objects provide two cleanup methods:

- **`__release()`** - Internal method called automatically by:
  - V8 garbage collector (when object is collected)
  - `Use()` function (immediate cleanup after callback)

- **`Release()`** - Public method for explicit manual cleanup:
  - Called in `try-finally` blocks
  - Provides immediate resource cleanup
  - Same implementation as `__release()` - they do the same thing

## Resource Hierarchy

When `Context.Release()` is called, it automatically releases:

1. **Trace object** - If present, calls `Trace.__release()` to cleanup:
   - Go bridge registry entries
   - Trace manager resources
   - Background goroutines

2. **Context object** - Releases:
   - Go bridge registry entry for the Context itself

This ensures proper cleanup of the entire resource tree.

## Usage Patterns

### Pattern 1: Automatic Cleanup with `Use()` (Recommended)

**Best for**: Most cases, clean code, automatic resource management

```javascript
// Context is released automatically after callback
Use(Context, contextData, (ctx) => {
  // Access Trace (released automatically with context)
  const trace = ctx.Trace
  const node = trace.Add({ type: "step" }, { label: "Processing" })
  
  trace.Info("Doing work")
  node.Complete({ result: "done" })
  
  return result
})
// ctx.Release() called automatically, which also releases Trace
```

### Pattern 2: Manual Cleanup with `try-finally`

**Best for**: Explicit control, critical memory scenarios

```javascript
const ctx = getContext() // or passed as parameter
const trace = ctx.Trace

try {
  const node = trace.Add({ type: "step" }, { label: "Processing" })
  
  trace.Info("Doing work")
  node.Complete({ result: "done" })
  
  return result
} finally {
  // Explicit cleanup (also releases Trace)
  ctx.Release()
}
```

### Pattern 3: Separate Trace Cleanup

**Best for**: When you want to release Trace independently

```javascript
const ctx = getContext()
const trace = ctx.Trace

try {
  const node = trace.Add({ type: "step" }, { label: "Processing" })
  trace.Info("Doing work")
  node.Complete({ result: "done" })
  
  // Release trace early if needed
  trace.Release()
  
  // Continue using ctx...
  return result
} finally {
  // Release context (Trace already released, safe to call again)
  ctx.Release()
}
```

### Pattern 4: No Explicit Cleanup (Not Recommended)

**Avoid in production**: Relies on GC, unpredictable timing

```javascript
function processData(ctx) {
  const trace = ctx.Trace
  const node = trace.Add({ type: "step" }, { label: "Processing" })
  
  trace.Info("Doing work")
  node.Complete({ result: "done" })
  
  return result
  // Waits for V8 GC to call __release() - SLOW!
}
```

## No-op Trace Handling

When Trace is not initialized, `ctx.Trace` returns a no-op object:

- All methods are no-ops (do nothing)
- `Release()` is safe to call (no-op)
- No errors are thrown
- Provides consistent API regardless of trace initialization

```javascript
// Works even if Trace is not initialized
const ctx = getContext()
const trace = ctx.Trace // might be no-op

trace.Info("Message") // safe even if no-op
trace.Release()       // safe even if no-op
ctx.Release()         // always safe
```

## Error Handling

Cleanup happens even when errors occur:

```javascript
const ctx = getContext()
try {
  const trace = ctx.Trace
  const node = trace.Add({ type: "step" }, { label: "Processing" })
  
  throw new Error("Something went wrong")
  
} finally {
  // Cleanup still happens
  ctx.Release() // also releases Trace
}
```

With `Use()`:

```javascript
try {
  Use(Context, contextData, (ctx) => {
    throw new Error("Something went wrong")
  })
} catch (error) {
  // Error is caught
  // ctx.Release() was already called automatically
}
```

## Memory Management

### ✅ Good: Immediate Cleanup

```javascript
// Loop with immediate cleanup
for (let i = 0; i < 10000; i++) {
  Use(Context, data, (ctx) => {
    const trace = ctx.Trace
    trace.Info(`Processing item ${i}`)
    // Released immediately after each iteration
  })
}
```

### ❌ Bad: Waiting for GC

```javascript
// Memory accumulates until GC runs
for (let i = 0; i < 10000; i++) {
  const ctx = getContext()
  const trace = ctx.Trace
  trace.Info(`Processing item ${i}`)
  // No cleanup - may run out of memory!
}
```

## Implementation Details

### Context.Release() / Context.__release()

1. Checks if `ctx.Trace` exists
2. If yes, calls `trace.__release()` to cleanup Trace resources
3. Releases Context from bridge registry
4. Safe to call multiple times (idempotent)
5. Errors in cleanup are silently ignored

### Trace.Release() / Trace.__release()

1. Releases Go manager object from bridge registry
2. Calls `trace.Release(traceID)` to cleanup:
   - Remove from global trace registry
   - Stop background goroutines
   - Free associated resources
3. Safe to call multiple times (idempotent)

### No-op Objects

Both no-op Trace and no-op Node provide:
- All methods as no-ops
- `Release()` and `__release()` methods
- Consistent API for error-free operation
- Zero memory overhead

## Best Practices

1. **✅ Use `Use()` for automatic cleanup** in most cases
2. **✅ Use `try-finally` with `Release()`** when you need explicit control
3. **✅ Release Context** (which also releases Trace) rather than releasing each separately
4. **✅ Release resources in loops** to prevent memory accumulation
5. **❌ Don't rely on GC** for resource cleanup in production code
6. **❌ Don't worry about calling `Release()` twice** - it's idempotent

## Testing

See `jsapi_release_test.go` for comprehensive tests of:
- Context Release
- Trace Release  
- Cascading cleanup (Context → Trace)
- try-finally pattern
- No-op object Release
- Error handling with cleanup

Run tests:
```bash
cd yao
go test -v ./agent/context -run Release
```

