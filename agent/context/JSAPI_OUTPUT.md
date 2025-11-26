# Context Output JS API

The Context object provides `Send`, `SendGroup`, and `Flush` methods for sending messages to clients from JavaScript within Agent Hook functions.

## Hook Functions Overview

Agent Hook functions are lifecycle callbacks that allow you to customize the behavior of AI assistants. The Context object passed to these hooks includes output methods for real-time communication with clients.

### Available Hooks

- `Create(ctx, messages)` - Called before the assistant processes messages
- `Before(ctx, messages, response)` - Called before sending LLM response
- `After(ctx, messages, response)` - Called after receiving LLM response
- `Done(ctx, messages, response)` - Called after assistant completes
- `Error(ctx, messages, error)` - Called when an error occurs

## Quick Start

### Basic Usage in Create Hook

```javascript
/**
 * Create hook - send initial messages to client
 */
function Create(ctx, messages) {
  // Send welcome message (string shorthand)
  ctx.Send("Welcome! Let me help you with that...");
  ctx.Flush();

  // Send loading indicator
  ctx.Send({
    type: "loading",
    props: { message: "Analyzing your request..." },
  });
  ctx.Flush();

  // Continue with normal processing
  return { messages };
}
```

### Streaming Updates Example

```javascript
/**
 * Create hook - demonstrate streaming updates
 */
function Create(ctx, messages) {
  // Send initial message
  ctx.Send({
    type: "text",
    props: { content: "Processing" },
    id: "status_msg",
  });
  ctx.Flush();

  time.Sleep(500); // Simulate work

  // Append to message (delta update)
  ctx.Send({
    type: "text",
    props: { content: "..." },
    id: "status_msg",
    delta: true,
    delta_path: "content",
    delta_action: "append",
  });
  ctx.Flush();

  time.Sleep(500); // More work

  // Complete the message
  ctx.Send({
    type: "text",
    props: { content: " Done!" },
    id: "status_msg",
    delta: true,
    delta_path: "content",
    delta_action: "append",
  });
  ctx.Flush();

  return { messages };
}
```

## API Reference

### ctx.Send(message)

Send a single message to the client.

**String Shorthand:**

```javascript
ctx.Send("Hello World");
```

**Object Format:**

```javascript
// Text message
ctx.Send({
  type: "text",
  props: { content: "Hello from JavaScript" },
});

// Loading indicator
ctx.Send({
  type: "loading",
  props: { message: "Processing..." },
});

// Error message
ctx.Send({
  type: "error",
  props: { message: "Something went wrong", code: "ERR_500" },
});
```

**Complete Message Object:**

```javascript
ctx.Send({
  type: "text",
  props: { content: "Hello" },
  id: "msg_123", // Optional: message ID for delta updates
  delta: true, // Optional: incremental update flag
  done: false, // Optional: completion flag
  delta_path: "content", // Optional: update path
  delta_action: "append", // Optional: append, replace, merge, set
  group_id: "grp_1", // Optional: message group ID
  metadata: {
    // Optional: custom metadata
    timestamp: Date.now(),
    sequence: 1,
    trace_id: "trace_123",
  },
});
```

### ctx.SendGroup(group)

Send a group of related messages together.

```javascript
ctx.SendGroup({
  id: "group_123",
  messages: [
    { type: "text", props: { content: "First message" } },
    { type: "text", props: { content: "Second message" } },
  ],
  metadata: { timestamp: Date.now() },
});
```

### ctx.Flush()

Flush the output buffer to ensure messages are sent immediately.

```javascript
ctx.Send("Processing...");
ctx.Flush(); // Send immediately to client
```

## Complete Hook Examples

### 1. Create Hook - Welcome Message

```javascript
/**
 * Send welcome message when conversation starts
 */
function Create(ctx, messages) {
  // Send welcome message
  ctx.Send("Welcome to AI Assistant! How can I help you today?");
  ctx.Flush();

  // Return messages to continue processing
  return { messages };
}
```

### 2. Create Hook - Progress Updates

```javascript
/**
 * Show progress indicators during preprocessing
 */
function Create(ctx, messages) {
  // Step 1: Analyzing
  ctx.Send({
    type: "loading",
    props: { message: "Analyzing your request..." },
  });
  ctx.Flush();

  // Perform analysis...
  const userIntent = analyzeIntent(messages);

  // Step 2: Searching
  ctx.Send({
    type: "loading",
    props: { message: "Searching knowledge base..." },
  });
  ctx.Flush();

  // Search knowledge base...
  const context = searchKnowledgeBase(userIntent);

  // Add context to messages
  if (context) {
    messages.unshift({
      role: "system",
      content: `Context: ${context}`,
    });
  }

  return { messages };
}
```

### 3. Before Hook - Show Thinking Process

```javascript
/**
 * Display model's reasoning before sending response
 */
function Before(ctx, messages, response) {
  // If response includes thinking/reasoning
  if (response.thinking) {
    ctx.Send({
      type: "thinking",
      props: { content: response.thinking },
    });
    ctx.Flush();
  }

  return { response };
}
```

### 4. After Hook - Process Tool Calls

```javascript
/**
 * Handle tool calls and send results
 */
function After(ctx, messages, response) {
  // Process tool calls
  if (response.tool_calls && response.tool_calls.length > 0) {
    response.tool_calls.forEach((toolCall) => {
      // Show tool being called
      ctx.Send({
        type: "tool_call",
        props: {
          id: toolCall.id,
          name: toolCall.function.name,
          arguments: toolCall.function.arguments,
        },
      });
      ctx.Flush();

      // Execute tool and send result
      const result = executeTool(toolCall);
      ctx.Send({
        type: "text",
        props: { content: `Tool result: ${result}` },
      });
      ctx.Flush();
    });
  }

  return { response };
}
```

### 5. Done Hook - Completion Message

```javascript
/**
 * Send completion message and cleanup
 */
function Done(ctx, messages, response) {
  // Send completion indicator
  ctx.Send({
    type: "text",
    props: { content: "\n✅ Task completed successfully!" },
  });
  ctx.Flush();

  // Log metrics
  console.log("Conversation completed:", {
    chat_id: ctx.chat_id,
    message_count: messages.length,
    tokens_used: response.usage?.total_tokens,
  });

  return {};
}
```

### 6. Error Hook - Handle Errors Gracefully

```javascript
/**
 * Send user-friendly error messages
 */
function Error(ctx, messages, error) {
  console.error("Assistant error:", error);

  // Send error message to user
  ctx.Send({
    type: "error",
    props: {
      message: "I encountered an issue while processing your request.",
      code: error.code || "UNKNOWN_ERROR",
      details:
        process.env.YAO_ENV === "development" ? error.message : undefined,
    },
  });
  ctx.Flush();

  // Return error to be logged
  return { error };
}
```

### 7. Multi-Step Process with Progress

```javascript
/**
 * Complex processing with multiple steps
 */
function Create(ctx, messages) {
  const steps = [
    { name: "Validating input", duration: 500 },
    { name: "Loading context", duration: 1000 },
    { name: "Preparing response", duration: 800 },
  ];

  // Create progress message
  const progressId = "progress_" + Date.now();

  steps.forEach((step, index) => {
    // Update progress
    ctx.Send({
      type: "loading",
      props: {
        message: `${step.name}... (${index + 1}/${steps.length})`,
      },
      id: progressId,
      delta: index > 0,
    });
    ctx.Flush();

    // Simulate work
    time.Sleep(step.duration);
  });

  // Clear progress indicator
  ctx.Send({
    type: "loading",
    props: { message: "" },
    id: progressId,
    done: true,
  });
  ctx.Flush();

  return { messages };
}
```

### 8. Real-time Streaming Updates

```javascript
/**
 * Send streaming updates as processing progresses
 */
function Create(ctx, messages) {
  const messageId = "stream_" + Date.now();

  // Start message
  ctx.Send({
    type: "text",
    props: { content: "Processing" },
    id: messageId,
  });
  ctx.Flush();

  // Simulate incremental processing
  const updates = [".", ".", ".", " analyzing", ".", ".", ".", " complete!"];

  updates.forEach((update) => {
    time.Sleep(200);

    ctx.Send({
      type: "text",
      props: { content: update },
      id: messageId,
      delta: true,
      delta_path: "content",
      delta_action: "append",
    });
    ctx.Flush();
  });

  return { messages };
}
```

### 9. Message Groups for Related Content

```javascript
/**
 * Send groups of related messages together
 */
function Before(ctx, messages, response) {
  // Send a group of context information
  ctx.SendGroup({
    id: "context_info",
    messages: [
      {
        type: "text",
        props: { content: "**Context Information:**" },
      },
      {
        type: "text",
        props: { content: `User: ${ctx.authorized?.user_id || "Anonymous"}` },
      },
      {
        type: "text",
        props: { content: `Session: ${ctx.chat_id}` },
      },
      {
        type: "text",
        props: { content: `Locale: ${ctx.locale}` },
      },
    ],
    metadata: {
      timestamp: Date.now(),
      type: "context",
    },
  });
  ctx.Flush();

  return { response };
}
```

## Message Types

Built-in message types supported:

- `user_input` - User input (display only)
- `text` - Text content (supports Markdown)
- `thinking` - Reasoning/thinking process
- `loading` - Loading indicator
- `tool_call` - Tool/function call
- `error` - Error message
- `image` - Image content
- `audio` - Audio content
- `video` - Video content
- `action` - System action (silent in OpenAI clients)
- `event` - Lifecycle event (CUI only)

## Message Props by Type

### Text Message

```javascript
{
    type: "text",
    props: {
        content: "Text content (supports Markdown)"
    }
}
```

### Thinking Message

```javascript
{
    type: "thinking",
    props: {
        content: "Reasoning process..."
    }
}
```

### Loading Message

```javascript
{
    type: "loading",
    props: {
        message: "Loading message..."
    }
}
```

### Tool Call Message

```javascript
{
    type: "tool_call",
    props: {
        id: "call_123",
        name: "function_name",
        arguments: '{"key": "value"}'
    }
}
```

### Error Message

```javascript
{
    type: "error",
    props: {
        message: "Error message",
        code: "ERROR_CODE",
        details: "Additional details"
    }
}
```

### Image Message

```javascript
{
    type: "image",
    props: {
        url: "https://example.com/image.jpg",
        alt: "Image description",
        width: 800,
        height: 600
    }
}
```

### Audio Message

```javascript
{
    type: "audio",
    props: {
        url: "https://example.com/audio.mp3",
        format: "mp3",
        duration: 120.5,
        transcript: "Audio transcript...",
        autoplay: false,
        controls: true
    }
}
```

### Video Message

```javascript
{
    type: "video",
    props: {
        url: "https://example.com/video.mp4",
        format: "mp4",
        thumbnail: "https://example.com/thumb.jpg",
        width: 1920,
        height: 1080,
        autoplay: false,
        controls: true
    }
}
```

## Delta Updates

Use delta updates for streaming scenarios:

```javascript
// Initial message
ctx.Send({
  type: "text",
  props: { content: "Hello" },
  id: "msg_1",
  delta: false,
});

// Append to content
ctx.Send({
  type: "text",
  props: { content: " World" },
  id: "msg_1",
  delta: true,
  delta_path: "content",
  delta_action: "append",
});

// Mark as complete
ctx.Send({
  type: "text",
  props: {},
  id: "msg_1",
  done: true,
});
```

**Delta Actions:**

- `append` - Append to string or array
- `replace` - Replace value
- `merge` - Merge objects
- `set` - Set new field

## Hook Function Patterns

### Pattern 1: Fire-and-Forget Notifications

```javascript
function Create(ctx, messages) {
  ctx.Send("Starting processing...");
  ctx.Flush();
  // No need to wait, continue processing
  return { messages };
}
```

### Pattern 2: Progress Tracking

```javascript
function Create(ctx, messages) {
  const stages = ["validate", "analyze", "prepare"];
  stages.forEach((stage) => {
    ctx.Send({ type: "loading", props: { message: `${stage}...` } });
    ctx.Flush();
    performStage(stage);
  });
  return { messages };
}
```

### Pattern 3: Conditional Messaging

```javascript
function Before(ctx, messages, response) {
  // Only show reasoning for complex queries
  if (messages[messages.length - 1].content.length > 100) {
    ctx.Send({
      type: "thinking",
      props: { content: "Analyzing complex query..." },
    });
    ctx.Flush();
  }
  return { response };
}
```

### Pattern 4: Error Recovery

```javascript
function Error(ctx, messages, error) {
  if (error.code === "RATE_LIMIT") {
    ctx.Send("Service is busy, retrying...");
    ctx.Flush();
    time.Sleep(1000);
    return { retry: true };
  }

  ctx.Send({
    type: "error",
    props: { message: "Sorry, something went wrong.", code: error.code },
  });
  ctx.Flush();
  return { error };
}
```

## Important Notes

### 1. Hook Function Signatures

Each hook receives different parameters:

- `Create(ctx, messages)` - Context and input messages
- `Before(ctx, messages, response)` - Context, messages, and LLM response
- `After(ctx, messages, response)` - Context, messages, and LLM response
- `Done(ctx, messages, response)` - Context, messages, and final response
- `Error(ctx, messages, error)` - Context, messages, and error object

### 2. Always Flush for Real-time Updates

```javascript
// Good - user sees message immediately
ctx.Send("Processing...");
ctx.Flush();

// Bad - message buffered until hook returns
ctx.Send("Processing...");
// ... hook continues ...
```

### 3. Delta Updates Require Unique IDs

```javascript
// Initial message
ctx.Send({ type: "text", props: { content: "Step 1" }, id: "progress" });

// Update same message
ctx.Send({
  type: "text",
  props: { content: ", Step 2" },
  id: "progress",
  delta: true,
  delta_path: "content",
  delta_action: "append",
});
```

### 4. Message Types and Client Support

- **OpenAI Client** (`ctx.accept === "standard"`): Supports `text`, `thinking`, `tool_call`, `image`, `audio`, `video`
- **CUI Client** (`ctx.accept === "cui-web"` etc.): Supports all types including `loading`, `error`, `action`, `event`

### 5. Performance Considerations

- Use `Flush()` sparingly - only when immediate delivery is needed
- Batch related messages with `SendGroup()` when possible
- Avoid sending too many small updates (combine them)

### 6. Context Information Available

The `ctx` object provides access to:

```javascript
ctx.chat_id; // Chat session ID
ctx.assistant_id; // Assistant ID
ctx.locale; // User locale (e.g., "en", "zh-cn")
ctx.authorized; // User authorization info
ctx.metadata; // Custom metadata
ctx.client; // Client information (type, user_agent, ip)
```

## Migration Guide

### From Old Output API

**Before (Deprecated):**

```javascript
function Create(ctx, messages) {
  const output = new Output(ctx);
  output.Send("Hello");
  output.SendGroup({ id: "grp1", messages: [...] });
}
```

**After (Current):**

```javascript
function Create(ctx, messages) {
  ctx.Send("Hello");
  ctx.SendGroup({ id: "grp1", messages: [...] });
  ctx.Flush();
  return { messages };
}
```

## Best Practices

1. **Use String Shorthand**: `ctx.Send("Hello")` is simpler than `ctx.Send({ type: "text", props: { content: "Hello" } })`

2. **Flush After Each Step**: Ensure users see progress in real-time

3. **Handle Errors Gracefully**: Always provide user-friendly error messages

4. **Show Progress for Long Operations**: Use loading indicators for better UX

5. **Return Hook Results**: Always return required objects from hooks:

   - `Create`: `{ messages }`
   - `Before/After`: `{ response }`
   - `Done`: `{}` or `{ response }`
   - `Error`: `{ error }` or `{ retry: true }`

6. **Test with Different Clients**: Verify behavior with both OpenAI and CUI clients
