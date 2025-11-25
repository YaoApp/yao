# Output JSAPI

The Output JSAPI provides a JavaScript interface for sending output messages to clients from scripts (e.g., hooks, processes). It wraps the Go `output` package functionality and provides a convenient API for sending messages and message groups.

## Overview

The Output object allows you to:

- Send individual messages to clients in various formats (text, error, loading, etc.)
- Send groups of related messages
- Support streaming with delta updates
- Handle different message types with custom properties

## Constructor

### `new Output(ctx)`

Creates a new Output instance.

**Parameters:**

- `ctx` (Context): The agent context object

**Returns:**

- Output instance

**Example:**

```javascript
function Create(ctx, messages) {
  const output = new Output(ctx);
  // Use output methods...
}
```

## Methods

### `Send(message)`

Sends a single message to the client.

**Parameters:**

- `message` (string | object): The message to send
  - If string: Automatically converted to a text message
  - If object: Must have a `type` field and optional `props` and other fields

**Returns:**

- Output instance (for chaining)

**Message Object Structure:**

```javascript
{
  type: string,              // Required: Message type (e.g., "text", "error", "loading")
  props: object,             // Optional: Message properties (type-specific)
  id: string,                // Optional: Message ID (for streaming)
  delta: boolean,            // Optional: Whether this is a delta update
  done: boolean,             // Optional: Whether the message is complete
  delta_path: string,        // Optional: Path for delta updates (e.g., "content")
  delta_action: string,      // Optional: Delta action ("append", "replace", "merge", "set")
  type_change: boolean,      // Optional: Whether this is a type correction
  group_id: string,          // Optional: Parent message group ID
  group_start: boolean,      // Optional: Marks the start of a message group
  group_end: boolean,        // Optional: Marks the end of a message group
  metadata: {                // Optional: Message metadata
    timestamp: number,
    sequence: number,
    trace_id: string
  }
}
```

**Examples:**

Send a simple text message (shorthand):

```javascript
output.Send("Hello, world!");
```

Send a text message (full):

```javascript
output.Send({
  type: "text",
  props: {
    content: "Hello, world!",
  },
});
```

Send an error message:

```javascript
output.Send({
  type: "error",
  props: {
    message: "Something went wrong",
    code: "ERR_001",
    details: "Additional error details",
  },
});
```

Send a loading indicator:

```javascript
output.Send({
  type: "loading",
  props: {
    message: "Searching knowledge base...",
  },
});
```

Send streaming text with delta updates:

```javascript
// First chunk
output.Send({
  type: "text",
  id: "msg-1",
  props: { content: "Hello" },
  delta: true,
  done: false,
});

// Subsequent chunks
output.Send({
  type: "text",
  id: "msg-1",
  props: { content: " world" },
  delta: true,
  delta_path: "content",
  delta_action: "append",
  done: false,
});

// Final chunk
output.Send({
  type: "text",
  id: "msg-1",
  props: { content: "!" },
  delta: true,
  delta_path: "content",
  delta_action: "append",
  done: true,
});
```

Chain multiple sends:

```javascript
output
  .Send("First message")
  .Send("Second message")
  .Send({ type: "loading", props: { message: "Processing..." } });
```

### `SendGroup(group)`

Sends a group of related messages.

**Parameters:**

- `group` (object): The message group
  - `id` (string): Required - Group ID
  - `messages` (array): Required - Array of message objects
  - `metadata` (object): Optional - Group metadata

**Returns:**

- Output instance (for chaining)

**Group Object Structure:**

```javascript
{
  id: string,                // Required: Message group ID
  messages: [                // Required: Array of messages
    {
      type: string,
      props: object,
      // ... other message fields
    }
  ],
  metadata: {                // Optional: Group metadata
    timestamp: number,
    sequence: number,
    trace_id: string
  }
}
```

**Examples:**

Send a simple message group:

```javascript
output.SendGroup({
  id: "search-results",
  messages: [
    { type: "text", props: { content: "Found 3 results:" } },
    { type: "text", props: { content: "Result 1" } },
    { type: "text", props: { content: "Result 2" } },
    { type: "text", props: { content: "Result 3" } },
  ],
});
```

Send a group with metadata:

```javascript
output.SendGroup({
  id: "analysis-group",
  messages: [
    { type: "loading", props: { message: "Analyzing data..." } },
    { type: "text", props: { content: "Analysis complete" } },
  ],
  metadata: {
    timestamp: Date.now(),
    sequence: 1,
    trace_id: "trace-123",
  },
});
```

## Built-in Message Types

The Output JSAPI supports all built-in message types defined in the output package:

### User Interaction Types

- **`user_input`**: User input message (frontend display only)
  ```javascript
  { type: "user_input", props: { content: "User's message", role: "user" } }
  ```

### Content Types

- **`text`**: Plain text or Markdown content

  ```javascript
  { type: "text", props: { content: "Hello **world**" } }
  ```

- **`thinking`**: Reasoning/thinking process (e.g., o1 models)

  ```javascript
  { type: "thinking", props: { content: "Let me think about this..." } }
  ```

- **`loading`**: Loading/processing indicator

  ```javascript
  { type: "loading", props: { message: "Processing..." } }
  ```

- **`tool_call`**: LLM tool/function call

  ```javascript
  {
    type: "tool_call",
    props: {
      id: "call_123",
      name: "search",
      arguments: "{\"query\":\"test\"}"
    }
  }
  ```

- **`error`**: Error message
  ```javascript
  {
    type: "error",
    props: {
      message: "Error occurred",
      code: "ERR_001",
      details: "More info"
    }
  }
  ```

### Media Types

- **`image`**: Image content

  ```javascript
  {
    type: "image",
    props: {
      url: "https://example.com/image.jpg",
      alt: "Description",
      width: 800,
      height: 600
    }
  }
  ```

- **`audio`**: Audio content

  ```javascript
  {
    type: "audio",
    props: {
      url: "https://example.com/audio.mp3",
      format: "mp3",
      duration: 120.5
    }
  }
  ```

- **`video`**: Video content
  ```javascript
  {
    type: "video",
    props: {
      url: "https://example.com/video.mp4",
      format: "mp4",
      duration: 300
    }
  }
  ```

### System Types

- **`action`**: System action (silent in OpenAI clients)

  ```javascript
  {
    type: "action",
    props: {
      name: "open_panel",
      payload: { panel_id: "settings" }
    }
  }
  ```

- **`event`**: Lifecycle event (CUI only, silent in OpenAI clients)
  ```javascript
  {
    type: "event",
    props: {
      event: "stream_start",
      message: "Starting stream..."
    }
  }
  ```

## Usage in Hooks

### Create Hook Example

```javascript
/**
 * Create hook - Called before sending messages to the LLM
 * @param {Context} ctx - Agent context
 * @param {Array} messages - User messages
 * @returns {Object} Hook response
 */
function Create(ctx, messages) {
  const output = new Output(ctx);

  // Send a loading indicator
  output.Send({
    type: "loading",
    props: { message: "Processing your request..." },
  });

  // Send custom messages to the user
  output.Send({
    type: "text",
    props: { content: "I'm thinking about your question..." },
  });

  // Return hook response
  return {
    messages: messages,
    temperature: 0.7,
  };
}
```

### Done Hook Example

```javascript
/**
 * Done hook - Called after assistant completes response
 * @param {Context} ctx - Agent context
 * @param {Array} messages - Conversation messages
 * @param {Object} response - Assistant response
 */
function Done(ctx, messages, response) {
  const output = new Output(ctx);

  // Send a completion message
  output.Send({
    type: "text",
    props: { content: "Response complete!" },
  });

  // Send an action
  output.Send({
    type: "action",
    props: {
      name: "save_conversation",
      payload: { chat_id: ctx.chat_id },
    },
  });
}
```

### Progress Updates Example

```javascript
function ProcessData(ctx, data) {
  const output = new Output(ctx);

  // Show progress
  const steps = ["Loading", "Processing", "Analyzing", "Complete"];

  for (let i = 0; i < steps.length; i++) {
    output.Send({
      type: "loading",
      props: {
        message: `${steps[i]}... (${i + 1}/${steps.length})`,
      },
    });

    // Do some work...
    processStep(i);
  }

  // Send final result
  output.Send({
    type: "text",
    props: { content: "All done!" },
  });
}
```

## Error Handling

The Output JSAPI throws exceptions for invalid parameters:

```javascript
try {
  const output = new Output(ctx);

  // This will throw: message.type is required
  output.Send({ props: { content: "test" } });
} catch (e) {
  console.error("Output error:", e.toString());
}
```

Common errors:

- `"Output constructor requires a context argument"` - Missing ctx parameter
- `"Send requires a message argument"` - Missing message parameter
- `"message.type is required and must be a string"` - Missing or invalid type field
- `"SendGroup requires a group argument"` - Missing group parameter
- `"group.id is required and must be a string"` - Missing group ID
- `"group.messages is required and must be an array"` - Missing or invalid messages array

## Notes

1. **Context Requirement**: The Output object must be created with a valid agent context
2. **Writer Required**: The context must have a Writer set (automatically handled in API requests)
3. **Message Format**: Messages are automatically adapted based on the context's Accept type (standard, cui-web, cui-native, cui-desktop)
4. **Streaming**: For streaming responses, use delta updates with proper message IDs
5. **Method Chaining**: All methods return the Output instance for convenient chaining

## See Also

- [Output Package Documentation](../README.md)
- [Message Types](../BUILTIN_TYPES.md)
- [Agent Context](../../context/README.md)
- [Hook System](../../assistant/hook/README.md)
