# Context Output JS API

The Context object now provides `Send`, `SendGroup`, and `Flush` methods directly for sending messages to clients from JavaScript.

## Usage

### ctx.Send(message)

Send a single message to the client.

**Parameters:**

- `message`: Can be a string (shorthand) or an object

**String Shorthand:**

```javascript
// Automatically converts to a text message
ctx.Send("Hello World");
```

**Object Format:**

```javascript
// Send text message
ctx.Send({
  type: "text",
  props: {
    content: "Hello from JavaScript",
  },
});

// Send loading message
ctx.Send({
  type: "loading",
  props: {
    message: "Processing...",
  },
});

// Send error message
ctx.Send({
  type: "error",
  props: {
    message: "Something went wrong",
    code: "ERR_500",
  },
});

// Send custom message
ctx.Send({
  type: "custom_widget",
  props: {
    data: { foo: "bar" },
  },
});
```

**Complete Message Object:**

```javascript
ctx.Send({
  type: "text",
  props: {
    content: "Hello",
  },
  id: "msg_123", // Optional: message ID
  delta: true, // Optional: incremental update
  done: false, // Optional: whether complete
  delta_path: "content", // Optional: update path
  delta_action: "append", // Optional: update action (append, replace, merge, set)
  group_id: "grp_1", // Optional: message group ID
  metadata: {
    // Optional: metadata
    timestamp: Date.now(),
    sequence: 1,
    trace_id: "trace_123",
  },
});
```

### ctx.SendGroup(group)

Send a group of messages to the client.

**Parameters:**

- `group`: Message group object

**Example:**

```javascript
ctx.SendGroup({
  id: "group_123",
  messages: [
    {
      type: "text",
      props: { content: "First message" },
    },
    {
      type: "text",
      props: { content: "Second message" },
    },
  ],
  metadata: {
    timestamp: Date.now(),
  },
});
```

### ctx.Flush()

Flush the output buffer to ensure all messages are sent to the client.

**Example:**

```javascript
ctx.Send("Processing...");
ctx.Flush(); // Send immediately
```

## Complete Examples

### Using in Hook Functions

```javascript
/**
 * Create hook - called before assistant processes
 */
function Create(input, options) {
  const ctx = input.context;

  // Send welcome message
  ctx.Send("Welcome to AI Assistant!");

  // Send loading indicator
  ctx.Send({
    type: "loading",
    props: {
      message: "Thinking...",
    },
  });

  return { messages: input.messages };
}

/**
 * Done hook - called after assistant completes
 */
function Done(input, output) {
  const ctx = input.context;

  // Send completion message
  ctx.Send({
    type: "text",
    props: {
      content: "Processing completed!",
    },
  });

  // Flush output
  ctx.Flush();

  return {};
}
```

### Streaming Response Example

```javascript
function StreamingResponse(input) {
  const ctx = input.context;

  // Send initial message
  ctx.Send({
    type: "text",
    props: { content: "Starting process" },
    id: "msg_1",
    delta: false,
  });

  // Send incremental updates
  ctx.Send({
    type: "text",
    props: { content: "..." },
    id: "msg_1",
    delta: true,
    delta_path: "content",
    delta_action: "append",
  });

  // Send completion marker
  ctx.Send({
    type: "text",
    props: { content: "" },
    id: "msg_1",
    delta: false,
    done: true,
  });

  ctx.Flush();
}
```

### Error Handling Example

```javascript
function ProcessWithErrorHandling(input) {
  const ctx = input.context;

  try {
    // Processing logic
    ctx.Send("Processing...");

    // Simulate error
    throw new Error("Something went wrong");
  } catch (error) {
    // Send error message
    ctx.Send({
      type: "error",
      props: {
        message: error.message,
        code: "ERR_PROCESSING",
      },
    });

    ctx.Flush();
  }
}
```

### Multi-step Process Example

```javascript
function MultiStepProcess(input) {
  const ctx = input.context;

  // Step 1
  ctx.Send({
    type: "loading",
    props: { message: "Step 1: Analyzing input..." },
  });
  ctx.Flush();

  // ... processing ...

  // Step 2
  ctx.Send({
    type: "loading",
    props: { message: "Step 2: Generating response..." },
  });
  ctx.Flush();

  // ... processing ...

  // Final result
  ctx.Send({
    type: "text",
    props: { content: "Process completed successfully!" },
  });
  ctx.Flush();
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

## Notes

1. **No Separate Output API Needed**: The previous `const output = new Output(ctx)` approach is deprecated. Now use `ctx.Send()` methods directly.

2. **Automatic Client Handling**: Messages are automatically converted to the appropriate format based on `ctx.accept`:

   - `standard` → OpenAI format
   - `cui-web`/`cui-native`/`cui-desktop` → CUI native format

3. **Performance Optimization**: Output objects are automatically cached and managed, no manual management needed.

4. **Error Handling**: All methods throw JavaScript exceptions on failure, which can be caught with try-catch.

5. **Streaming Support**: Use delta updates with unique message IDs for real-time streaming scenarios.

6. **Metadata**: Optional metadata can be attached to messages for tracking, debugging, or custom processing.

## Migration Guide

**Before (Deprecated):**

```javascript
// Old way - no longer needed
const output = new Output(ctx)
output.Send("Hello")
output.SendGroup({ id: "grp1", messages: [...] })
```

**After (Current):**

```javascript
// New way - simpler and cleaner
ctx.Send("Hello")
ctx.SendGroup({ id: "grp1", messages: [...] })
ctx.Flush()
```

## Best Practices

1. **Use String Shorthand for Simple Messages**: `ctx.Send("Hello")` instead of `ctx.Send({ type: "text", props: { content: "Hello" } })`

2. **Always Flush After Important Messages**: Use `ctx.Flush()` to ensure messages are sent immediately

3. **Use Unique IDs for Delta Updates**: Assign unique IDs to messages that will receive incremental updates

4. **Handle Errors Gracefully**: Wrap Send operations in try-catch blocks for robust error handling

5. **Use Loading Indicators**: Show loading messages for long-running operations to improve UX

6. **Group Related Messages**: Use `SendGroup` for semantically related messages that should be displayed together
