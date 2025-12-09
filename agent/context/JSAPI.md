# Context JavaScript API Documentation

## Overview

The Context JavaScript API provides a comprehensive interface for interacting with the Yao Agent system from JavaScript/TypeScript hooks (Create, Next). The Context object exposes agent state, configuration, messaging capabilities, trace operations, and MCP (Model Context Protocol) integrations.

## Context Object

The Context object is automatically passed to hook functions and provides access to the agent's execution environment.

### Basic Properties

```typescript
interface Context {
  // Identifiers
  chat_id: string; // Current chat session ID
  assistant_id: string; // Assistant identifier

  // Configuration
  locale: string; // User locale (e.g., "en", "zh-cn")
  theme: string; // UI theme preference
  accept: string; // Output format ("standard", "cui-web", "cui-native", etc.)
  route: string; // Request route path
  referer: string; // Request referer

  // Client Information
  client: {
    type: string; // Client type
    user_agent: string; // User agent string
    ip: string; // Client IP address
  };

  // Dynamic Data
  metadata: Record<string, any>; // Custom metadata (empty object if not set)
  authorized: Record<string, any>; // Authorization data (empty object if not set)

  // Objects
  space: Space; // Shared data space for passing data between requests
  Trace: Trace; // Trace object for debugging and monitoring
  MCP: MCP; // MCP object for external tool/resource access
}
```

## Methods

### Send Messages

The Context provides several methods for sending messages to the client:

| Method                               | Description                 | Auto `message_end` | Updatable |
| ------------------------------------ | --------------------------- | ------------------ | --------- |
| `Send(message, block_id?)`           | Send a complete message     | ✅ Yes             | ❌ No     |
| `SendStream(message, block_id?)`     | Start a streaming message   | ❌ No              | ✅ Yes    |
| `Append(message_id, content, path?)` | Append content to a message | -                  | -         |
| `Replace(message_id, message)`       | Replace message content     | -                  | -         |
| `Merge(message_id, data, path?)`     | Merge data into message     | -                  | -         |
| `Set(message_id, data, path)`        | Set a field in message      | -                  | -         |
| `End(message_id, final_content?)`    | Finalize streaming message  | ✅ Yes             | -         |

> **Note:** `Append`, `Replace`, `Merge`, and `Set` only work with messages started via `SendStream()`. Messages sent via `Send()` are immediately finalized and cannot be updated.

#### `ctx.Send(message, block_id?): string`

Sends a message to the client and automatically flushes the output.

**Parameters:**

- `message`: Message object or string
- `block_id`: String (optional) - Block ID to send this message in. If omitted, no block ID is assigned.

**Returns:**

- `string`: The message ID (auto-generated if not provided in the message object)

**Message Object Structure:**

```typescript
interface Message {
  // Required
  type: string; // Message type: "text", "tool", "image", etc.

  // Common fields
  props?: Record<string, any>; // Message properties (passed to frontend component)
  message_id?: string; // Message ID (auto-generated if omitted)
  block_id?: string; // Block ID (NOT auto-generated, has priority over block_id parameter)
  thread_id?: string; // Thread ID (auto-set from Stack for nested agents)

  // Metadata (optional)
  metadata?: Record<string, any>; // Custom metadata
}
```

**Examples:**

```javascript
// Send text message (object format) and capture message ID
const message_id = ctx.Send({
  type: "text",
  props: { content: "Hello, World!" },
});
console.log("Sent message:", message_id);

// Send text message (shorthand) - no block ID by default
const text_id = ctx.Send("Hello, World!");

// Send multiple messages in the same block (same bubble/card in UI)
const block_id = ctx.BlockID(); // Generate block ID first
const msg1 = ctx.Send("Step 1: Analyzing...", block_id);
const msg2 = ctx.Send("Step 2: Processing...", block_id);
const msg3 = ctx.Send("Step 3: Complete!", block_id);

// Specify block_id in message object (highest priority)
const msg4 = ctx.Send({
  type: "text",
  props: { content: "In specific block" },
  block_id: "B2", // This takes priority over second parameter
});

// Send tool message with custom IDs
const tool_id = ctx.Send({
  type: "tool",
  message_id: "custom-tool-msg-1",
  block_id: "B_tools",
  props: {
    name: "calculator",
    result: { sum: 42 },
  },
});

// Send image message
const image_id = ctx.Send({
  type: "image",
  props: {
    url: "https://example.com/image.png",
    alt: "Example Image",
  },
});
```

**Block Management:**

```javascript
// Scenario 1: Simple message (most common)
function Next(ctx, payload) {
  const { completion } = payload;

  // Send a complete message
  ctx.Send({
    type: "text",
    props: { content: completion.content },
  });
}

// Scenario 2: Loading indicator before slow operation
function Next(ctx, payload) {
  // Start a streaming message for loading
  const loading_id = ctx.SendStream({
    type: "loading",
    props: { message: "Fetching data..." },
  });

  // Do slow operation (e.g., external API call)
  const result = fetchExternalData();

  // Replace loading with result
  ctx.Replace(loading_id, {
    type: "text",
    props: { content: result },
  });
  ctx.End(loading_id);
}

// Scenario 3: Grouping messages in one block (special case)
function Create(ctx, messages) {
  // Generate a block ID for grouping
  const block_id = ctx.BlockID(); // "B1"

  ctx.Send("# Analysis Results", block_id);
  ctx.Send("- Finding 1: ...", block_id);
  ctx.Send("- Finding 2: ...", block_id);
  ctx.Send("- Finding 3: ...", block_id);

  // All messages appear in the same card/bubble in the UI
}

// Scenario 4: LLM response + follow-up card in same block
function Next(ctx, payload) {
  const { completion } = payload;
  const block_id = ctx.BlockID();

  // LLM response
  ctx.Send({
    type: "text",
    props: { content: completion.content },
    block_id: block_id,
  });

  // Action card (grouped with LLM response)
  ctx.Send({
    type: "card",
    props: {
      title: "Related Actions",
      actions: ["action1", "action2"],
    },
    block_id: block_id,
  });
}
```

**Notes:**

- **Message ID** is automatically generated if not provided
- **Block ID** is NOT auto-generated by default (remains empty unless manually specified)
  - Most messages don't need a Block ID (each message is independent)
  - Only specify Block ID in special cases (e.g., grouping LLM output with a follow-up card)
  - **Block ID priority**: message.block_id > block_id parameter > empty
- **Thread ID** is automatically set from Stack for non-root calls (nested agents)
- Returns the message ID for reference in subsequent operations
- Output is automatically flushed after sending
- Throws exception on failure
- `Send()` automatically sends `message_end` event - the message is complete and cannot be updated
- **For updatable messages**, use `ctx.SendStream()` instead (see below)

#### `ctx.SendStream(message, block_id?): string`

Sends a streaming message that can be appended to later. Unlike `Send()`, this does NOT automatically send `message_end` event. Use `ctx.Append()` to add content, then `ctx.End()` to finalize.

**Parameters:**

- `message`: Message object or string
- `block_id`: String (optional) - Block ID to send this message in

**Returns:**

- `string`: The message ID (for use with `Append` and `End`)

**Examples:**

```javascript
// Start a streaming message
const msg_id = ctx.SendStream({
  type: "text",
  props: { content: "# Title\n\n" },
});

// Append content in chunks (simulating streaming)
ctx.Append(msg_id, "First paragraph. ");
ctx.Append(msg_id, "Second sentence. ");
ctx.Append(msg_id, "Third sentence.\n\n");

// Finalize the message (sends message_end event)
ctx.End(msg_id);
```

**String Shorthand:**

```javascript
// SendStream with string shorthand
const msg_id = ctx.SendStream("Starting analysis...");
ctx.Append(msg_id, " processing...");
ctx.Append(msg_id, " done!");
ctx.End(msg_id);
// Final content: "Starting analysis... processing... done!"
```

**With Block ID:**

```javascript
const block_id = ctx.BlockID();
const msg_id = ctx.SendStream("Step 1: ", block_id);
ctx.Append(msg_id, "Analyzing data...");
ctx.End(msg_id);
```

**Notes:**

- Returns the message ID immediately for use with `Append` and `End`
- Sends `message_start` event but NOT `message_end` (unlike `Send`)
- Must call `ctx.End(msg_id)` to finalize the message
- Content appended via `ctx.Append()` is accumulated for storage
- Ideal for streaming text output where you control the timing

#### `ctx.End(message_id, final_content?): string`

Finalizes a streaming message started with `SendStream()`. Sends `message_end` event with the complete accumulated content.

**Parameters:**

- `message_id`: String - The message ID returned by `SendStream()`
- `final_content`: String (optional) - Final content to append before ending

**Returns:**

- `string`: The message ID

**Examples:**

```javascript
// Basic usage
const msg_id = ctx.SendStream("Hello");
ctx.Append(msg_id, " World");
ctx.End(msg_id);
// Final: "Hello World"

// End with final content
const msg_id2 = ctx.SendStream("Processing");
ctx.Append(msg_id2, "...");
ctx.End(msg_id2, " Complete!");
// Final: "Processing... Complete!"
```

**Notes:**

- Must be called after `SendStream()` to send `message_end` event
- Optional `final_content` is appended before sending `message_end`
- The complete accumulated content is included in `message_end.extra.content`
- Throws exception if `message_id` is not a string

**Send vs SendStream Comparison:**

| Feature               | `Send()`          | `SendStream()`      |
| --------------------- | ----------------- | ------------------- |
| `message_start` event | ✅ Auto           | ✅ Auto             |
| `message_end` event   | ✅ Auto           | ❌ Manual (`End()`) |
| Use case              | Complete messages | Streaming output    |
| Content accumulation  | N/A               | Via `Append()`      |
| Storage               | Immediate         | On `End()`          |

**Streaming Workflow Example:**

```javascript
function Create(ctx, messages) {
  // Start streaming output
  const msg_id = ctx.SendStream({
    type: "text",
    props: { content: "# Analysis Report\n\n" },
  });

  // Simulate streaming chunks
  ctx.Append(msg_id, "## Section 1\n");
  ctx.Append(msg_id, "Processing data...\n\n");

  // Do some work
  const result = analyzeData();

  ctx.Append(msg_id, "## Section 2\n");
  ctx.Append(msg_id, `Found ${result.count} items.\n\n`);

  // Finalize with conclusion
  ctx.End(msg_id, "## Conclusion\nAnalysis complete.");

  return { messages };
}
```

#### `ctx.Replace(message_id, message): string`

Replaces the content of a streaming message. **Only works with messages started via `SendStream()`**.

**Parameters:**

- `message_id`: String - The ID of the streaming message (returned by `SendStream()`)
- `message`: Message object or string - The new message content

**Returns:**

- `string`: The message ID (same as the provided message_id)

**Examples:**

```javascript
// Start a streaming message
const msg_id = ctx.SendStream({
  type: "loading",
  props: { message: "Loading..." },
});

// Replace with new content
ctx.Replace(msg_id, {
  type: "text",
  props: { content: "Data loaded successfully!" },
});

// Finalize the message
ctx.End(msg_id);
```

**Use Cases:**

```javascript
// Progress updates with replacement
function Next(ctx, payload) {
  const msg_id = ctx.SendStream("Step 1/3: Starting...");

  // ... do work ...
  ctx.Replace(msg_id, "Step 2/3: Processing...");

  // ... do more work ...
  ctx.Replace(msg_id, "Step 3/3: Finalizing...");

  // ... finish ...
  ctx.Replace(msg_id, "Complete! ✓");
  ctx.End(msg_id);
}

// Loading to result transition
function Next(ctx, payload) {
  const msg_id = ctx.SendStream({
    type: "loading",
    props: { message: "Fetching results..." },
  });

  const results = fetchData();

  ctx.Replace(msg_id, {
    type: "text",
    props: { content: `Found ${results.length} results` },
  });
  ctx.End(msg_id);
}
```

**Notes:**

- **Only works with `SendStream()` messages** - `Send()` messages cannot be replaced
- Replaces the entire message content, not just specific fields
- Must call `ctx.End(msg_id)` after all updates to finalize the message
- Output is automatically flushed after replacing
- Throws exception on failure

#### `ctx.Append(message_id, content, path?): string`

Appends content to a streaming message. **Only works with messages started via `SendStream()`**.

**Parameters:**

- `message_id`: String - The ID of the streaming message (returned by `SendStream()`)
- `content`: Message object or string - The content to append
- `path`: String (optional) - The delta path to append to (e.g., "props.content", "props.data")

**Returns:**

- `string`: The message ID (same as the provided message_id)

**Examples:**

```javascript
// Start a streaming message
const msg_id = ctx.SendStream("Starting");

// Append more text (default path)
ctx.Append(msg_id, "... processing");
ctx.Append(msg_id, "... done!");

// Finalize the message
ctx.End(msg_id);
// Final content: "Starting... processing... done!"

// Append to specific path
const data_id = ctx.SendStream({
  type: "data",
  props: {
    content: "Item 1\n",
    status: "loading",
  },
});

ctx.Append(data_id, "Item 2\n", "props.content");
ctx.Append(data_id, "Item 3\n", "props.content");
ctx.End(data_id);
// Final: props.content = "Item 1\nItem 2\nItem 3\n"
```

**Use Cases:**

```javascript
// Streaming text output (simulating LLM-like output)
function Create(ctx, messages) {
  const msg_id = ctx.SendStream("");

  ctx.Append(msg_id, "The");
  ctx.Append(msg_id, " quick");
  ctx.Append(msg_id, " brown");
  ctx.Append(msg_id, " fox");

  ctx.End(msg_id);
  // Final: "The quick brown fox"

  return { messages };
}

// Progress logs
function Next(ctx, payload) {
  const log_id = ctx.SendStream({
    type: "log",
    props: { content: "Starting process\n" },
  });

  // Step 1
  doStep1();
  ctx.Append(log_id, "Step 1 complete\n", "props.content");

  // Step 2
  doStep2();
  ctx.Append(log_id, "Step 2 complete\n", "props.content");

  // Finish
  ctx.Append(log_id, "All done!\n", "props.content");
  ctx.End(log_id);
}
```

**Notes:**

- **Only works with `SendStream()` messages** - `Send()` messages cannot be appended to
- Uses delta append operation (adds to existing content, doesn't replace)
- If `path` is omitted, appends to the default content location (`props.content`)
- Must call `ctx.End(msg_id)` after all appends to finalize the message
- Output is automatically flushed after appending
- Throws exception on failure
- block_id and ThreadID are inherited from the original message

#### `ctx.Merge(message_id, data, path?): string`

Merges data into a streaming message object. **Only works with messages started via `SendStream()`**.

**Parameters:**

- `message_id`: String - The ID of the streaming message (returned by `SendStream()`)
- `data`: Object - The data to merge (should be an object)
- `path`: String (optional) - The delta path to merge into (e.g., "props", "props.metadata")

**Returns:**

- `string`: The message ID (same as the provided message_id)

**Examples:**

```javascript
// Start a streaming message with object data
const msg_id = ctx.SendStream({
  type: "status",
  props: {
    status: "running",
    progress: 0,
    started: true,
  },
});

// Merge updates into props (adds/updates fields, keeps others unchanged)
ctx.Merge(msg_id, { progress: 50 }, "props");
// Result: props = { status: "running", progress: 50, started: true }

ctx.Merge(msg_id, { progress: 100, status: "completed" }, "props");
// Result: props = { status: "completed", progress: 100, started: true }

// Finalize the message
ctx.End(msg_id);
```

**Use Cases:**

```javascript
// Updating task progress
function Next(ctx, payload) {
  const task_id = ctx.SendStream({
    type: "task",
    props: {
      name: "Data Processing",
      status: "pending",
      progress: 0,
    },
  });

  ctx.Merge(task_id, { status: "running" }, "props");
  doStep1();
  ctx.Merge(task_id, { progress: 25 }, "props");
  doStep2();
  ctx.Merge(task_id, { progress: 50 }, "props");
  doStep3();
  ctx.Merge(task_id, { progress: 100, status: "completed" }, "props");

  ctx.End(task_id);
}

// Building metadata incrementally
function Create(ctx, messages) {
  const data_id = ctx.SendStream({
    type: "data",
    props: { content: "Result data" },
  });

  ctx.Merge(data_id, { metadata: { source: "api" } }, "props");
  ctx.Merge(data_id, { metadata: { timestamp: Date.now() } }, "props");
  // metadata fields are merged together

  ctx.End(data_id);
  return { messages };
}
```

**Notes:**

- **Only works with `SendStream()` messages** - `Send()` messages cannot be merged into
- Uses delta merge operation (merges objects, doesn't replace)
- Only works with object data (for merging key-value pairs)
- Existing fields not in the merge data remain unchanged
- If `path` is omitted, merges into the default object location
- Must call `ctx.End(msg_id)` after all merges to finalize the message
- Output is automatically flushed after merging
- Throws exception on failure
- block_id and ThreadID are inherited from the original message

#### `ctx.Set(message_id, data, path): string`

Sets a new field or value in a streaming message. **Only works with messages started via `SendStream()`**.

**Parameters:**

- `message_id`: String - The ID of the streaming message (returned by `SendStream()`)
- `data`: Any - The value to set
- `path`: String (required) - The delta path where to set the value (e.g., "props.newField", "props.metadata.key")

**Returns:**

- `string`: The message ID (same as the provided message_id)

**Examples:**

```javascript
// Start a streaming message
const msg_id = ctx.SendStream({
  type: "result",
  props: {
    content: "Initial content",
  },
});

// Set a new field
ctx.Set(msg_id, "success", "props.status");
// Result: props.status = "success"

// Set a nested object
ctx.Set(msg_id, { duration: 1500, cached: true }, "props.metadata");
// Result: props.metadata = { duration: 1500, cached: true }

// Finalize the message
ctx.End(msg_id);
```

**Use Cases:**

```javascript
// Adding computed metadata after initial send
function Next(ctx, payload) {
  const result_id = ctx.SendStream({
    type: "search_result",
    props: { results: search_results },
  });

  ctx.Set(result_id, search_results.length, "props.count");
  ctx.Set(result_id, Date.now(), "props.timestamp");
  ctx.Set(result_id, "relevance", "props.sort_by");

  ctx.End(result_id);
}

// Conditionally adding fields
function Create(ctx, messages) {
  const msg_id = ctx.SendStream({
    type: "operation",
    props: { name: "Process Data" },
  });

  try {
    const result = processData();
    ctx.Set(msg_id, "success", "props.status");
    ctx.Set(msg_id, result, "props.data");
  } catch (e) {
    ctx.Set(msg_id, e.message, "props.error");
    ctx.Set(msg_id, "error", "props.status");
  }

  ctx.End(msg_id);
  return { messages };
}
```

**Notes:**

- **Only works with `SendStream()` messages** - `Send()` messages cannot be modified
- Uses delta set operation (creates/sets new fields)
- The `path` parameter is **required** (must specify where to set the value)
- Creates the path if it doesn't exist
- Use for adding new fields or completely replacing a field's value
- For updating existing object fields, consider using `Merge` instead
- Must call `ctx.End(msg_id)` after all sets to finalize the message
- Output is automatically flushed after setting
- Throws exception on failure
- block_id and ThreadID are inherited from the original message

### ID Generators

These methods generate unique IDs for manual message management. Useful when you need to specify IDs before sending messages or for advanced Block/Thread management.

#### `ctx.MessageID(): string`

Generates a unique message ID.

**Returns:**

- `string`: Message ID in format "M1", "M2", "M3"...

**Example:**

```javascript
// Generate IDs manually
const id_1 = ctx.MessageID(); // "M1"
const id_2 = ctx.MessageID(); // "M2"

// Use custom ID
ctx.Send({
  type: "text",
  message_id: id_1,
  props: { content: "Hello" },
});
```

#### `ctx.BlockID(): string`

Generates a unique block ID for grouping messages.

**Returns:**

- `string`: Block ID in format "B1", "B2", "B3"...

**Example:**

```javascript
// Generate block ID for grouping messages
const block_id = ctx.BlockID(); // "B1"

// Send multiple messages in the same block
ctx.Send("Step 1: Analyzing...", block_id);
ctx.Send("Step 2: Processing...", block_id);
ctx.Send("Step 3: Complete!", block_id);

// All three messages appear in the same card/bubble in UI
```

**Use Cases:**

```javascript
// Scenario: LLM output + follow-up card in same block
const block_id = ctx.BlockID();

// LLM response
const llm_result = Process("llms.chat", {...});
ctx.Send({
  type: "text",
  props: { content: llm_result.content },
  block_id: block_id,
});

// Follow-up action card (grouped with LLM output)
ctx.Send({
  type: "card",
  props: {
    title: "Related Actions",
    actions: [...]
  },
  block_id: block_id,
});
```

#### `ctx.ThreadID(): string`

Generates a unique thread ID for concurrent operations.

**Returns:**

- `string`: Thread ID in format "T1", "T2", "T3"...

**Example:**

```javascript
// For advanced parallel processing scenarios
const thread_id = ctx.ThreadID(); // "T1"

// Send messages in a specific thread
ctx.Send({
  type: "text",
  props: { content: "Parallel task 1" },
  thread_id: thread_id,
});
```

**Notes:**

- IDs are generated sequentially within each context
- Each context has its own ID counter (starts from 1)
- IDs are guaranteed to be unique within the same request/stream
- ThreadID is usually auto-managed by Stack, manual generation is for advanced use cases

### Lifecycle Management

#### `ctx.EndBlock(block_id): void`

Manually sends a `block_end` event for the specified block. Use this to explicitly mark the end of a block.

**Parameters:**

- `block_id`: String - The block ID to end

**Returns:**

- `void`

**Example:**

```javascript
// Create a block for grouped messages
const block_id = ctx.BlockID(); // "B1"

// Send messages in the block
ctx.Send("Analyzing data...", block_id);
ctx.Send("Processing results...", block_id);
ctx.Send("Complete!", block_id);

// Manually end the block
ctx.EndBlock(block_id);
```

**Block Lifecycle Events:**

When you send messages with a `block_id`:

1. **First message**: Automatically sends `block_start` event
2. **Subsequent messages**: No additional block events
3. **Manual end**: Call `ctx.EndBlock(block_id)` to send `block_end` event

**block_end Event Format:**

```json
{
  "type": "event",
  "props": {
    "event": "block_end",
    "message": "Block ended",
    "data": {
      "block_id": "B1",
      "timestamp": 1764483531624,
      "duration_ms": 1523,
      "message_count": 3,
      "status": "completed"
    }
  }
}
```

**Notes:**

- `block_start` is sent automatically when the first message with a new `block_id` is sent
- `block_end` must be called manually via `ctx.EndBlock()`
- You can track multiple blocks simultaneously (each has independent lifecycle)
- Automatically flushes output after sending the event

**Use Cases:**

```javascript
// Use case 1: Progress reporting in a block
function Create(ctx, messages) {
  const block_id = ctx.BlockID();

  ctx.Send("Step 1: Analyzing data...", block_id);
  // ... analysis logic ...

  ctx.Send("Step 2: Processing results...", block_id);
  // ... processing logic ...

  ctx.Send("Step 3: Complete!", block_id);

  // Mark the block as complete
  ctx.EndBlock(block_id);

  return { messages };
}

// Use case 2: Multiple parallel blocks
function Create(ctx, messages) {
  const llm_block = ctx.BlockID(); // "B1"
  const mcp_block = ctx.BlockID(); // "B2"

  // LLM output block
  ctx.Send("Thinking...", llm_block);
  const response = callLLM();
  ctx.Send(response, llm_block);
  ctx.EndBlock(llm_block);

  // MCP tool call block
  ctx.Send("Fetching data...", mcp_block);
  const data = ctx.MCP.CallTool("tool", "method", {});
  ctx.Send(`Found ${data.length} results`, mcp_block);
  ctx.EndBlock(mcp_block);

  return { messages };
}
```

### Resource Cleanup

#### `ctx.Release()`

Manually releases Context resources. This is optional as cleanup happens automatically via garbage collection.

**Example:**

```javascript
try {
  // Use context
  ctx.Send("Processing...");
} finally {
  ctx.Release(); // Manual cleanup
}
```

## Trace API

The `ctx.Trace` object provides comprehensive tracing capabilities for debugging and monitoring agent execution.

### Node Operations

#### `ctx.Trace.Add(input, options)`

Creates a new trace node (sequential step).

**Parameters:**

- `input`: Input data for the node
- `options`: Node configuration object

**Options Structure:**

```typescript
interface TraceNodeOption {
  label: string; // Display label
  type: string; // Node type identifier
  icon: string; // Icon identifier
  description: string; // Node description
  metadata?: Record<string, any>; // Additional metadata
}
```

**Example:**

```javascript
const search_node = ctx.Trace.Add(
  { query: "What is AI?" },
  {
    label: "Search Query",
    type: "search",
    icon: "search",
    description: "Searching for AI information",
  }
);
```

#### `ctx.Trace.Parallel(inputs)`

Creates multiple parallel trace nodes for concurrent operations.

**Parameters:**

- `inputs`: Array of parallel input objects

**Input Structure:**

```typescript
interface ParallelInput {
  input: any; // Input data
  option: TraceNodeOption; // Node configuration
}
```

**Example:**

```javascript
const parallel_nodes = ctx.Trace.Parallel([
  {
    input: { url: "https://api1.com" },
    option: {
      label: "API Call 1",
      type: "api",
      icon: "cloud",
      description: "Fetching from API 1",
    },
  },
  {
    input: { url: "https://api2.com" },
    option: {
      label: "API Call 2",
      type: "api",
      icon: "cloud",
      description: "Fetching from API 2",
    },
  },
]);
```

### Logging Methods

Add log entries to the current trace node:

```javascript
// Information logs
ctx.Trace.Info("Processing started", { step: 1 });

// Debug logs
ctx.Trace.Debug("Variable value", { value: 42 });

// Warning logs
ctx.Trace.Warn("Deprecated feature used", { feature: "old_api" });

// Error logs
ctx.Trace.Error("Operation failed", { error: "timeout" });
```

### Node Status Operations

#### `node.SetOutput(output)`

Sets the output data for a node.

```javascript
const search_node = ctx.Trace.Add({ query: "search" }, options);
search_node.SetOutput({ results: [...] });
```

#### `node.SetMetadata(key, value)`

Sets metadata for a node.

```javascript
search_node.SetMetadata("duration", 1500);
search_node.SetMetadata("cache_hit", true);
```

#### `node.Complete(output?)`

Marks a node as completed (optionally with output).

```javascript
search_node.Complete({ status: "success", data: [...] });
```

#### `node.Fail(error)`

Marks a node as failed with an error.

```javascript
try {
  // Operation
} catch (error) {
  search_node.Fail(error);
}
```

### Query Operations

#### `ctx.Trace.GetRootNode()`

Returns the root node of the trace tree.

```javascript
const root_node = ctx.Trace.GetRootNode();
console.log(root_node.id, root_node.label);
```

#### `ctx.Trace.GetNode(id)`

Retrieves a specific node by ID.

```javascript
const target_node = ctx.Trace.GetNode("node-123");
```

#### `ctx.Trace.GetCurrentNodes()`

Returns the current active nodes (may be multiple if in parallel state).

```javascript
const current_nodes = ctx.Trace.GetCurrentNodes();
```

### Memory Space Operations

#### `ctx.Trace.CreateSpace(option)`

Creates a memory space for storing key-value data.

```javascript
const memory_space = ctx.Trace.CreateSpace({
  label: "Context Memory",
  type: "context",
  icon: "database",
  description: "Stores conversation context",
});
```

#### `ctx.Trace.GetSpace(id)`

Retrieves a memory space by ID.

```javascript
const context_space = ctx.Trace.GetSpace("context");
```

#### `ctx.Trace.HasSpace(id)`

Checks if a memory space exists.

```javascript
if (ctx.Trace.HasSpace("context")) {
  // Space exists
}
```

#### `ctx.Trace.DeleteSpace(id)`

Deletes a memory space.

```javascript
ctx.Trace.DeleteSpace("temp_storage");
```

#### `ctx.Trace.ListSpaces()`

Lists all memory spaces.

```javascript
const all_spaces = ctx.Trace.ListSpaces();
all_spaces.forEach((space) => {
  console.log(space.id, space.label);
});
```

## Space API

The `ctx.space` object provides a shared data space for passing data between requests and agent calls. This is useful for storing temporary data that needs to be accessed across different hooks or nested agent calls.

### Methods

#### `ctx.space.Get(key): any`

Gets a value from the space.

**Parameters:**

- `key`: String - The key to retrieve

**Returns:**

- `any`: The value, or `null` if not found

**Example:**

```javascript
const user_data = ctx.space.Get("user_data");
if (user_data) {
  console.log("Found user:", user_data.name);
}
```

#### `ctx.space.Set(key, value): void`

Sets a value in the space.

**Parameters:**

- `key`: String - The key to set
- `value`: Any - The value to store

**Example:**

```javascript
ctx.space.Set("user_data", { name: "John", id: 123 });
ctx.space.Set("processing_status", "started");
```

#### `ctx.space.Delete(key): void`

Deletes a key from the space.

**Parameters:**

- `key`: String - The key to delete

**Example:**

```javascript
ctx.space.Delete("temp_data");
```

#### `ctx.space.GetDel(key): any`

Gets a value and immediately deletes it. Convenient for one-time use data.

**Parameters:**

- `key`: String - The key to retrieve and delete

**Returns:**

- `any`: The value, or `null` if not found

**Example:**

```javascript
// Store file metadata in parent agent
ctx.space.Set("file_metadata", { name: "report.pdf", size: 1024 });

// In child agent, get and consume the data
const metadata = ctx.space.GetDel("file_metadata");
// metadata is now deleted from space
```

### Use Cases

```javascript
// Use case 1: Pass data between hooks
function Create(ctx, messages) {
  // Store data for later use
  ctx.space.Set("original_query", messages[0].content);
  return { messages };
}

function Next(ctx, payload) {
  // Retrieve data from Create hook
  const query = ctx.space.Get("original_query");
  console.log("Original query was:", query);
}

// Use case 2: Pass data to nested agent calls
function Create(ctx, messages) {
  // Prepare context for child agent
  ctx.space.Set("parent_context", {
    user_id: ctx.authorized.user_id,
    session_start: Date.now(),
  });

  // Call child agent...
}

// Use case 3: One-time data consumption
function Next(ctx, payload) {
  // Get and delete in one operation
  const temp_data = ctx.space.GetDel("temp_processing_data");
  if (temp_data) {
    // Process and discard
  }
}
```

**Notes:**

- Space is shared across all hooks within the same request
- Space persists across nested agent calls (A2A)
- Values can be any JSON-serializable data
- Use `GetDel` for data that should only be consumed once

## MCP API

The `ctx.MCP` object provides access to Model Context Protocol operations for interacting with external tools, resources, and prompts.

### Resource Operations

#### `ctx.MCP.ListResources(client)`

Lists available resources from an MCP client.

```javascript
const fs_resources = ctx.MCP.ListResources("filesystem");
```

#### `ctx.MCP.ReadResource(client, uri)`

Reads a specific resource.

```javascript
const file_content = ctx.MCP.ReadResource(
  "filesystem",
  "file:///path/to/file.txt"
);
```

### Tool Operations

#### `ctx.MCP.ListTools(client)`

Lists available tools from an MCP client.

```javascript
const available_tools = ctx.MCP.ListTools("toolkit");
```

#### `ctx.MCP.CallTool(client, name, args)`

Calls a single tool.

```javascript
const calc_result = ctx.MCP.CallTool("calculator", "add", {
  a: 10,
  b: 32,
});
```

#### `ctx.MCP.CallTools(client, calls)`

Calls multiple tools sequentially.

```javascript
const tool_results = ctx.MCP.CallTools("toolkit", [
  { name: "tool1", args: { param: "value1" } },
  { name: "tool2", args: { param: "value2" } },
]);
```

#### `ctx.MCP.CallToolsParallel(client, calls)`

Calls multiple tools in parallel.

```javascript
const parallel_results = ctx.MCP.CallToolsParallel("toolkit", [
  { name: "api1", args: { endpoint: "/users" } },
  { name: "api2", args: { endpoint: "/posts" } },
]);
```

### Prompt Operations

#### `ctx.MCP.ListPrompts(client)`

Lists available prompts from an MCP client.

```javascript
const available_prompts = ctx.MCP.ListPrompts("prompt_library");
```

#### `ctx.MCP.GetPrompt(client, name, args?)`

Retrieves a specific prompt.

```javascript
const review_prompt = ctx.MCP.GetPrompt("prompt_library", "code_review", {
  language: "javascript",
});
```

### Sample Operations

#### `ctx.MCP.CreateSample(client, uri, sample)`

Creates a sample for a resource.

```javascript
ctx.MCP.CreateSample("filesystem", "file:///examples", {
  name: "example1",
  content: "Sample content",
});
```

## Hooks

The Agent system supports two hooks that can be defined in the assistant's `index.ts` file:

### Create Hook

Called before the LLM call. Use this to preprocess messages, add context, or configure the LLM request.

**Signature:**

```typescript
function Create(ctx: Context, messages: Message[]): HookCreateResponse | null;
```

**Parameters:**

- `ctx`: Context object
- `messages`: Array of input messages (including chat history if enabled)

**Return Value (`HookCreateResponse`):**

```typescript
interface HookCreateResponse {
  // Messages to be sent to the assistant (can modify/replace input messages)
  messages?: Message[];

  // Audio configuration (for models that support audio output)
  audio?: AudioConfig;

  // Generation parameters (override assistant defaults)
  temperature?: number;
  max_tokens?: number;
  max_completion_tokens?: number;

  // MCP configuration - add/override MCP servers for this request
  mcp_servers?: MCPServerConfig[];

  // Prompt configuration
  prompts?: string; // Prompt preset key to use
  disable_global_prompts?: boolean; // Disable global prompts

  // Tool configuration
  tools?: ToolConfig[]; // Override tools for this request
  disable_tools?: boolean; // Disable all tools
}
```

**Example:**

```javascript
function Create(ctx, messages) {
  // Store data for Next hook
  ctx.space.Set("user_query", messages[0]?.content);

  // Modify messages
  const enhanced_messages = messages.map((msg) => ({
    ...msg,
    content: msg.content + "\n\nPlease be concise.",
  }));

  // Return configuration
  return {
    messages: enhanced_messages,
    temperature: 0.7,
    max_tokens: 2000,
  };
}
```

### Next Hook

Called after the LLM response (and tool calls if any). Use this to post-process the response, send custom messages, or delegate to another agent.

**Signature:**

```typescript
function Next(ctx: Context, payload: NextHookPayload): NextHookResponse | null;
```

**Parameters:**

- `ctx`: Context object
- `payload`: Object containing:

```typescript
interface NextHookPayload {
  messages: Message[]; // Messages sent to the assistant
  completion?: CompletionResponse; // LLM response
  tools?: ToolCallResponse[]; // Tool call results (if any)
  error?: string; // Error message if LLM call failed
}

interface CompletionResponse {
  content: string; // LLM text response
  tool_calls?: ToolCall[]; // Tool calls requested by LLM
  usage?: UsageInfo; // Token usage statistics
}

interface ToolCallResponse {
  toolcall_id: string;
  server: string; // MCP server name
  tool: string; // Tool name
  arguments?: any; // Arguments passed to tool
  result?: any; // Tool execution result
  error?: string; // Error if tool failed
}
```

**Return Value (`NextHookResponse`):**

```typescript
interface NextHookResponse {
  // Delegate to another agent (recursive call)
  delegate?: {
    agent_id: string; // Target agent ID
    messages: Message[]; // Messages to send
  };

  // Custom response data (returned to user)
  data?: any;

  // Metadata for debugging
  metadata?: Record<string, any>;
}
```

**Example:**

```javascript
function Next(ctx, payload) {
  const { messages, completion, tools, error } = payload;

  if (error) {
    ctx.Send({
      type: "error",
      props: { message: error },
    });
    return null;
  }

  // Process tool results
  if (tools && tools.length > 0) {
    const results = tools.map((t) => t.result);
    ctx.Send(`Tool results: ${JSON.stringify(results)}`);
  }

  // Return custom data
  return {
    data: {
      response: completion?.content,
      processed: true,
    },
    metadata: {
      tool_count: tools?.length || 0,
    },
  };
}
```

### Hook Execution Flow

```
User Input
    ↓
[Create Hook] → Preprocess messages, configure LLM
    ↓
[LLM Call] → Get completion from language model
    ↓
[Tool Calls] → Execute any tool calls (if requested by LLM)
    ↓
[Next Hook] → Post-process response, send messages
    ↓
Response to User
```

**Notes:**

- Hooks are optional - if not defined, the agent uses default behavior
- Return `null` or `undefined` from hooks to use default behavior
- Hooks can send messages directly via `ctx.Send()`, `ctx.SendStream()`, etc.
- Use `ctx.space` to pass data between Create and Next hooks

## Complete Example

Here's a comprehensive example using various Context API features:

```javascript
/**
 * Create Hook - Initialize and prepare for LLM call
 * @param {Context} ctx - Agent context
 * @param {Array} messages - Input messages
 */
function Create(ctx, messages) {
  // Store original query in space for later use
  ctx.space.Set("original_query", messages[0]?.content || "");

  // Add trace node
  ctx.Trace.Add(
    { messages },
    {
      label: "Create Hook",
      type: "hook",
      icon: "play",
      description: "Preparing messages for LLM",
    }
  );

  return { messages };
}

/**
 * Next Hook - Process LLM response and enhance with tools
 * @param {Context} ctx - Agent context
 * @param {Object} payload - Hook payload
 * @param {Array} payload.messages - Messages sent to the assistant
 * @param {Object} payload.completion - Completion response from LLM
 * @param {Array} payload.tools - Tool call results
 * @param {string} payload.error - Error message if failed
 */
function Next(ctx, payload) {
  try {
    const { messages, completion, tools, error } = payload;

    // Retrieve data from Create hook
    const original_query = ctx.space.Get("original_query");

    // Create trace node for custom processing
    const process_node = ctx.Trace.Add(
      { completion, tools },
      {
        label: "Custom Processing",
        type: "custom",
        icon: "settings",
        description: "Enhancing response with external data",
      }
    );

    ctx.Trace.Info("Starting custom processing", {
      original_query: original_query,
      tool_count: tools?.length || 0,
    });

    // Start streaming output
    const msg_id = ctx.SendStream("# Search Results\n\n");

    // Call MCP tool for additional data
    const search_results = ctx.MCP.CallTool("search_engine", "search", {
      query: "latest AI news",
      limit: 5,
    });

    // Stream results as they come
    ctx.Append(msg_id, `Found ${search_results.length} articles:\n\n`);

    search_results.forEach((result, i) => {
      ctx.Append(msg_id, `${i + 1}. **${result.title}**\n`);
      ctx.Append(msg_id, `   ${result.summary}\n\n`);
    });

    // Finalize the streaming message
    ctx.End(msg_id, "---\n*Search complete*");

    // Update trace
    process_node.SetMetadata("search_results_count", search_results.length);
    process_node.Complete({ status: "success" });

    return {
      data: { sources: search_results },
      metadata: { processed: true },
    };
  } catch (error) {
    ctx.Trace.Error("Processing failed", { error: error.message });
    throw error;
  }
}
```

## Best Practices

1. **Error Handling**: Always wrap Context operations in try-catch blocks
2. **Resource Cleanup**: Use try-finally pattern for manual cleanup if needed
3. **Trace Organization**: Create meaningful trace nodes with descriptive labels
4. **Logging Levels**: Use appropriate log levels (Debug for development, Info for progress, Error for failures)
5. **Message IDs**: Let the system auto-generate message IDs unless you need specific tracking
6. **Parallel Operations**: Use `Trace.Parallel()` for concurrent operations to maintain trace clarity
7. **Space Usage**: Use `ctx.space` for passing data between hooks and nested agent calls
8. **Streaming Messages**: Use `SendStream()` + `Append()` + `End()` for streaming output; use `Send()` for complete messages
9. **Block Grouping**: Only use Block IDs when you need to group multiple messages together (e.g., LLM output + follow-up card)

## Error Handling

All Context methods throw exceptions on failure. Always handle errors appropriately:

```javascript
try {
  ctx.Send(message);
} catch (error) {
  ctx.Trace.Error("Failed to send message", { error: error.message });
  throw error;
}
```

## TypeScript Support

For TypeScript projects, the Context types are automatically inferred. You can also import explicit types:

```typescript
import { Context, Message, TraceNodeOption } from "@yaoapps/types";

interface NextPayload {
  messages: Message[];
  completion: any;
  tools: any[];
  error?: string;
}

function Next(ctx: Context, payload: NextPayload): any {
  // Your code with full type checking
  const { messages, completion, tools, error } = payload;
  // ...
}
```

## See Also

- [Agent Hooks Documentation](../hooks/README.md)
- [MCP Protocol Specification](../mcp/README.md)
- [Trace System Documentation](../../trace/README.md)
- [Message Format Specification](../message/README.md)
