# Context JavaScript API Documentation

## Overview

The Context JavaScript API provides a comprehensive interface for interacting with the Yao Agent system from JavaScript/TypeScript hooks (Create, Next, Done). The Context object exposes agent state, configuration, messaging capabilities, trace operations, and MCP (Model Context Protocol) integrations.

## Context Object

The Context object is automatically passed to hook functions and provides access to the agent's execution environment.

### Basic Properties

```typescript
interface Context {
  // Identifiers
  chat_id: string; // Current chat session ID
  assistant_id: string; // Assistant identifier

  // Configuration
  connector: string; // LLM connector name
  search?: string; // Search engine configuration
  locale: string; // User locale (e.g., "en", "zh-cn")
  theme: string; // UI theme preference
  accept: string; // Output format ("openai", "cui", etc.)
  route: string; // Request route path
  referer: string; // Request referer

  // Retry Configuration
  retry: boolean; // Whether retry is enabled
  retry_times: number; // Number of retry attempts

  // Client Information
  client: {
    type: string; // Client type
    user_agent: string; // User agent string
    ip: string; // Client IP address
  };

  // Dynamic Data
  args?: any[]; // Additional arguments
  metadata?: Record<string, any>; // Custom metadata
  authorized?: Record<string, any>; // Authorization data
}
```

## Methods

### Send Messages

#### `ctx.Send(message): string`

Sends a message to the client and automatically flushes the output.

**Parameters:**

- `message`: Message object or string

**Returns:**

- `string`: The message ID (auto-generated if not provided in the message object)

**Message Object Structure:**

```typescript
interface Message {
  type: string; // Message type: "text", "tool", "image", etc.
  props: Record<string, any>; // Message properties
  message_id?: string; // Optional message ID (auto-generated if omitted)
}
```

**Examples:**

```javascript
// Send text message (object format) and capture message ID
const messageId = ctx.Send({
  type: "text",
  props: { content: "Hello, World!" },
});
console.log("Sent message:", messageId);

// Send text message (shorthand)
const textId = ctx.Send("Hello, World!");

// Send tool message with custom ID
const toolId = ctx.Send({
  type: "tool",
  message_id: "custom-tool-msg-1",
  props: {
    name: "calculator",
    result: { sum: 42 },
  },
});

// Send image message
const imageId = ctx.Send({
  type: "image",
  props: {
    url: "https://example.com/image.png",
    alt: "Example Image",
  },
});
```

**Notes:**

- Message ID is automatically generated if not provided
- Returns the message ID for reference in subsequent operations
- Output is automatically flushed after sending
- Throws exception on failure

#### `ctx.Replace(messageId, message): string`

Replaces an existing message with new content. This is useful for updating progress messages or correcting previously sent information.

**Parameters:**

- `messageId`: String - The ID of the message to replace
- `message`: Message object or string - The new message content

**Returns:**

- `string`: The message ID (same as the provided messageId)

**Examples:**

```javascript
// Send initial message
const msgId = ctx.Send("Processing...");

// Later, replace with updated content
ctx.Replace(msgId, "Processing complete!");

// Replace with complex message
ctx.Replace(msgId, {
  type: "text",
  props: {
    content: "Task finished",
    status: "success",
  },
});

// Replace with shorthand text
ctx.Replace(msgId, "Updated text content");
```

**Use Cases:**

```javascript
// Progress updates
const progressId = ctx.Send("Step 1/3: Starting...");
// ... do work ...
ctx.Replace(progressId, "Step 2/3: Processing...");
// ... do more work ...
ctx.Replace(progressId, "Step 3/3: Finalizing...");
// ... finish ...
ctx.Replace(progressId, "Complete! ✓");

// Error correction
const msgId = ctx.Send("Found 5 results");
// Oops, counted wrong
ctx.Replace(msgId, "Found 8 results");
```

**Notes:**

- The message must exist (must have been sent previously)
- Replaces the entire message content, not just specific fields
- Output is automatically flushed after replacing
- Throws exception on failure

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
const node = ctx.Trace.Add(
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
const nodes = ctx.Trace.Parallel([
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
const node = ctx.Trace.Add({ query: "search" }, options);
node.SetOutput({ results: [...] });
```

#### `node.SetMetadata(key, value)`

Sets metadata for a node.

```javascript
node.SetMetadata("duration", 1500);
node.SetMetadata("cache_hit", true);
```

#### `node.Complete(output?)`

Marks a node as completed (optionally with output).

```javascript
node.Complete({ status: "success", data: [...] });
```

#### `node.Fail(error)`

Marks a node as failed with an error.

```javascript
try {
  // Operation
} catch (error) {
  node.Fail(error);
}
```

### Query Operations

#### `ctx.Trace.GetRootNode()`

Returns the root node of the trace tree.

```javascript
const root = ctx.Trace.GetRootNode();
console.log(root.id, root.label);
```

#### `ctx.Trace.GetNode(id)`

Retrieves a specific node by ID.

```javascript
const node = ctx.Trace.GetNode("node-123");
```

#### `ctx.Trace.GetCurrentNodes()`

Returns the current active nodes (may be multiple if in parallel state).

```javascript
const currentNodes = ctx.Trace.GetCurrentNodes();
```

### Memory Space Operations

#### `ctx.Trace.CreateSpace(option)`

Creates a memory space for storing key-value data.

```javascript
const space = ctx.Trace.CreateSpace({
  label: "Context Memory",
  type: "context",
  icon: "database",
  description: "Stores conversation context",
});
```

#### `ctx.Trace.GetSpace(id)`

Retrieves a memory space by ID.

```javascript
const space = ctx.Trace.GetSpace("context");
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
const spaces = ctx.Trace.ListSpaces();
spaces.forEach((space) => {
  console.log(space.id, space.label);
});
```

## MCP API

The `ctx.MCP` object provides access to Model Context Protocol operations for interacting with external tools, resources, and prompts.

### Resource Operations

#### `ctx.MCP.ListResources(client)`

Lists available resources from an MCP client.

```javascript
const resources = ctx.MCP.ListResources("filesystem");
```

#### `ctx.MCP.ReadResource(client, uri)`

Reads a specific resource.

```javascript
const content = ctx.MCP.ReadResource("filesystem", "file:///path/to/file.txt");
```

### Tool Operations

#### `ctx.MCP.ListTools(client)`

Lists available tools from an MCP client.

```javascript
const tools = ctx.MCP.ListTools("toolkit");
```

#### `ctx.MCP.CallTool(client, name, args)`

Calls a single tool.

```javascript
const result = ctx.MCP.CallTool("calculator", "add", {
  a: 10,
  b: 32,
});
```

#### `ctx.MCP.CallTools(client, calls)`

Calls multiple tools sequentially.

```javascript
const results = ctx.MCP.CallTools("toolkit", [
  { name: "tool1", args: { param: "value1" } },
  { name: "tool2", args: { param: "value2" } },
]);
```

#### `ctx.MCP.CallToolsParallel(client, calls)`

Calls multiple tools in parallel.

```javascript
const results = ctx.MCP.CallToolsParallel("toolkit", [
  { name: "api1", args: { endpoint: "/users" } },
  { name: "api2", args: { endpoint: "/posts" } },
]);
```

### Prompt Operations

#### `ctx.MCP.ListPrompts(client)`

Lists available prompts from an MCP client.

```javascript
const prompts = ctx.MCP.ListPrompts("prompt_library");
```

#### `ctx.MCP.GetPrompt(client, name, args?)`

Retrieves a specific prompt.

```javascript
const prompt = ctx.MCP.GetPrompt("prompt_library", "code_review", {
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

## Complete Example

Here's a comprehensive example using various Context API features:

```javascript
/**
 * Next Hook - Process LLM response and enhance with tools
 */
function Next(ctx, messages, completion, tools) {
  try {
    // Create trace node for custom processing
    const processNode = ctx.Trace.Add(
      { completion, tools },
      {
        label: "Custom Processing",
        type: "custom",
        icon: "settings",
        description: "Enhancing response with external data",
      }
    );

    // Log processing start
    ctx.Trace.Info("Starting custom processing", {
      tool_count: tools?.length || 0,
    });

    // Send progress message and capture message ID
    const progressId = ctx.Send("Searching for articles...");

    // Call MCP tool for additional data
    const searchResults = ctx.MCP.CallTool("search_engine", "search", {
      query: "latest AI news",
      limit: 5,
    });

    // Update trace with results
    processNode.SetMetadata("search_results_count", searchResults.length);

    // Update the progress message with results
    ctx.Replace(progressId, `Found ${searchResults.length} relevant articles.`);

    // Log the message ID for tracking
    ctx.Trace.Debug("Updated progress message", { message_id: progressId });

    // Process and format response
    const enhancedResponse = {
      text: completion.content,
      sources: searchResults,
      timestamp: Date.now(),
    };

    // Mark node as complete
    processNode.Complete(enhancedResponse);

    // Return enhanced response
    return {
      data: enhancedResponse,
      done: true,
    };
  } catch (error) {
    ctx.Trace.Error("Processing failed", { error: error.message });
    throw error;
  } finally {
    // Optional: Manual cleanup
    ctx.Release();
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
7. **Memory Spaces**: Use memory spaces for persistent data across agent calls

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

function Next(
  ctx: Context,
  messages: Message[],
  completion: any,
  tools: any[]
): any {
  // Your code with full type checking
}
```

## See Also

- [Agent Hooks Documentation](../hooks/README.md)
- [MCP Protocol Specification](../mcp/README.md)
- [Trace System Documentation](../../trace/README.md)
- [Message Format Specification](../message/README.md)
