# Output Module

The output module provides a unified API for sending messages to different client types (CUI, OpenAI-compatible, etc.) with support for streaming and rich media content.

## Architecture

```
agent/output/
├── message/              # Core types and interfaces (no dependencies)
│   ├── types.go         # Message, MessageGroup, Props structures
│   └── interfaces.go    # Writer, Adapter, Factory interfaces
├── adapters/            # Client-specific adapters
│   ├── cui/             # CUI adapter (native DSL)
│   │   ├── adapter.go
│   │   └── writer.go
│   └── openai/          # OpenAI adapter (converts to OpenAI format)
│       ├── adapter.go
│       ├── converter.go
│       ├── writer.go
│       ├── types.go
│       └── factory.go
├── output.go            # Main API (Send, GetWriter, etc.)
├── builtin.go           # Helper functions for built-in types
└── BUILTIN_TYPES.md     # Documentation for built-in types
```

## DSL Structure

### Message Structure

The universal message DSL is a JSON structure that supports streaming, rich media, and incremental updates:

```go
type Message struct {
    // Core fields
    Type  string                 `json:"type"`            // Message type (e.g., "text", "image", "action")
    Props map[string]interface{} `json:"props,omitempty"` // Type-specific properties

    // Streaming control
    ID    string `json:"id,omitempty"`    // Unique message ID (for merging in streaming)
    Delta bool   `json:"delta,omitempty"` // Whether this is an incremental update
    Done  bool   `json:"done,omitempty"`  // Whether the message is complete

    // Delta update control (for incremental props updates)
    DeltaPath   string `json:"delta_path,omitempty"`   // Which field to update (e.g., "content", "items.0.name")
    DeltaAction string `json:"delta_action,omitempty"` // How to update ("append", "replace", "merge", "set")

    // Type correction (for streaming type inference)
    TypeChange bool `json:"type_change,omitempty"` // Marks this as a type correction message

    // Message grouping (for semantically related messages)
    GroupID    string `json:"group_id,omitempty"`    // Parent message group ID
    GroupStart bool   `json:"group_start,omitempty"` // Marks the start of a group
    GroupEnd   bool   `json:"group_end,omitempty"`   // Marks the end of a group

    // Metadata
    Metadata *Metadata `json:"metadata,omitempty"` // Timestamp, sequence, trace ID
}
```

### Field Descriptions

#### Core Fields

- **`Type`** (required): Determines how the message should be rendered

  - Built-in types: `text`, `thinking`, `loading`, `tool_call`, `error`, `image`, `audio`, `video`, `action`, `event`
  - Custom types: Any string (frontend must have corresponding component)

- **`Props`** (optional): Type-specific properties passed to the rendering component
  - For `text`: `{"content": "Hello"}`
  - For `image`: `{"url": "...", "alt": "..."}`
  - For custom types: Any JSON-serializable data

#### Streaming Control

- **`ID`** (optional): Unique identifier for message tracking

  - Used to merge multiple delta updates into a single message
  - Auto-generated if not provided
  - Example: `"msg_1234567890_9876543210"`

- **`Delta`** (optional): Marks this as an incremental update

  - `true`: Append/update to existing message with same ID
  - `false`: Complete message (default)
  - Used for streaming LLM responses

- **`Done`** (optional): Marks message as complete
  - `true`: No more updates will come for this message ID
  - `false`: More updates may follow
  - Typically sent as final message in a delta sequence

#### Delta Update Control

For complex, structured messages that need field-level updates:

- **`DeltaPath`** (optional): JSON path to the field being updated

  - Simple: `"content"` (updates `props.content`)
  - Nested: `"user.name"` (updates `props.user.name`)
  - Array: `"items.0.title"` (updates `props.items[0].title`)

- **`DeltaAction`** (optional): How to apply the delta update
  - `"append"`: Concatenate to existing string/array
  - `"replace"`: Replace entire value
  - `"merge"`: Merge objects (shallow merge)
  - `"set"`: Set new field (if doesn't exist)

#### Type Correction

- **`TypeChange`** (optional): Indicates message type was corrected
  - Used when initial type inference was wrong
  - Frontend should re-render with new type
  - Example: Initially sent as `text`, corrected to `thinking`

#### Message Grouping

For grouping semantically related messages (e.g., image + caption):

- **`GroupID`** (optional): Identifier for the message group
- **`GroupStart`** (optional): Marks the beginning of a group
- **`GroupEnd`** (optional): Marks the end of a group

#### Metadata

- **`Metadata`** (optional): Additional message metadata
  ```go
  type Metadata struct {
      Timestamp int64  // Unix nanoseconds
      Sequence  int    // Message sequence number
      TraceID   string // For debugging/logging
  }
  ```

### Message Examples

#### Simple Text Message

```json
{
  "type": "text",
  "props": {
    "content": "Hello, world!"
  }
}
```

#### Streaming Text (Delta Updates)

```json
// First chunk
{
  "id": "msg_123",
  "type": "text",
  "delta": true,
  "props": {
    "content": "Hello"
  }
}

// Second chunk (appends)
{
  "id": "msg_123",
  "type": "text",
  "delta": true,
  "props": {
    "content": ", world"
  }
}

// Final chunk (marks done)
{
  "id": "msg_123",
  "type": "text",
  "delta": true,
  "done": true,
  "props": {
    "content": "!"
  }
}
```

#### Complex Type with Nested Updates

```json
// Initial message
{
  "id": "msg_456",
  "type": "table",
  "props": {
    "columns": ["Name", "Age"],
    "rows": []
  }
}

// Add first row
{
  "id": "msg_456",
  "type": "table",
  "delta": true,
  "delta_path": "rows",
  "delta_action": "append",
  "props": {
    "rows": [{"name": "Alice", "age": 30}]
  }
}

// Add second row
{
  "id": "msg_456",
  "type": "table",
  "delta": true,
  "delta_path": "rows",
  "delta_action": "append",
  "props": {
    "rows": [{"name": "Bob", "age": 25}]
  }
}
```

#### Type Correction

```json
// Initial guess (text)
{
  "id": "msg_789",
  "type": "text",
  "delta": true,
  "props": {
    "content": "Let me think..."
  }
}

// Correction (actually thinking)
{
  "id": "msg_789",
  "type": "thinking",
  "type_change": true,
  "props": {
    "content": "Let me think..."
  }
}
```

#### Message Group

```json
// Group start
{
  "group_id": "grp_001",
  "group_start": true
}

// Image in group
{
  "type": "image",
  "group_id": "grp_001",
  "props": {
    "url": "https://example.com/photo.jpg",
    "alt": "Beautiful sunset"
  }
}

// Caption in group
{
  "type": "text",
  "group_id": "grp_001",
  "props": {
    "content": "Captured at Golden Gate Bridge"
  }
}

// Group end
{
  "group_id": "grp_001",
  "group_end": true
}
```

## Key Design Decisions

### 1. Separate `message` Package

To avoid circular dependencies, all core types and interfaces are defined in the `message` sub-package:

- `message.Message` - Universal message DSL
- `message.Writer` - Interface for writing messages
- `message.Adapter` - Interface for format conversion

This allows:

- `handlers` → `output` → `message` ✅
- `output/adapters` → `message` ✅
- No circular dependencies!

### 2. Adapter Pattern

Different clients require different formats:

**CUI Clients:**

```json
{
  "type": "text",
  "props": { "content": "Hello" }
}
```

**OpenAI Clients:**

```json
{
  "choices": [
    {
      "delta": { "content": "Hello" }
    }
  ]
}
```

Adapters handle the transformation automatically based on `ctx.Accept`.

### 3. Built-in Types

10 standardized message types with defined Props structures:

| Type        | Purpose            | CUI     | OpenAI                    |
| ----------- | ------------------ | ------- | ------------------------- |
| `text`      | Text content       | Direct  | `delta.content`           |
| `thinking`  | LLM reasoning      | Direct  | `delta.reasoning_content` |
| `loading`   | Progress indicator | Direct  | `delta.reasoning_content` |
| `tool_call` | Function calls     | Direct  | `delta.tool_calls`        |
| `error`     | Error messages     | Direct  | `error`                   |
| `image`     | Images             | Render  | `![](url)` markdown       |
| `audio`     | Audio              | Player  | Link                      |
| `video`     | Video              | Player  | Link                      |
| `action`    | System commands    | Execute | Silent                    |
| `event`     | Lifecycle events   | Track   | Silent                    |

## Usage

### Basic Usage

```go
import (
    "github.com/yaoapp/yao/agent/output"
    "github.com/yaoapp/yao/agent/output/message"
)

// Send a text message
msg := output.NewTextMessage("Hello world")
output.Send(ctx, msg)

// Send a loading indicator
loading := output.NewLoadingMessage("Searching knowledge base...")
output.Send(ctx, loading)

// Send an image
img := output.NewImageMessage("https://example.com/image.jpg", "Description")
output.Send(ctx, img)

// Send an error
err := output.NewErrorMessage("Connection failed", "TIMEOUT")
output.Send(ctx, err)
```

### Streaming Messages

```go
// Send delta (incremental) updates
msg := &message.Message{
    ID:    "msg_123",
    Type:  message.TypeText,
    Delta: true,  // Incremental update
    Props: map[string]interface{}{
        "content": "Hello",
    },
}
output.Send(ctx, msg)

// Mark as complete
msg.Delta = false
msg.Done = true
msg.Props["content"] = "Hello world!"  // Full content
output.Send(ctx, msg)
```

### Custom Writers

```go
// Register a custom writer factory
factory := &MyCustomFactory{}
output.SetWriterFactory(factory)

// Now all calls to output.Send will use your custom writer
```

## Integration with Handlers

The `handlers` package uses the output module for streaming:

```go
func DefaultStreamHandler(ctx *context.Context) context.StreamFunc {
    return func(chunkType context.StreamChunkType, data []byte) int {
        switch chunkType {
        case context.ChunkText:
            msg := output.NewTextMessage(string(data))
            output.Send(ctx, msg)
        case context.ChunkThinking:
            msg := output.NewThinkingMessage(string(data))
            output.Send(ctx, msg)
        // ... handle other types
        }
        return 0 // Continue
    }
}
```

## Context-based Routing

The output module automatically selects the right writer based on `ctx.Accept`:

| `ctx.Accept`  | Writer | Format                |
| ------------- | ------ | --------------------- |
| `standard`    | OpenAI | OpenAI-compatible SSE |
| `cui-web`     | CUI    | Universal DSL JSON    |
| `cui-native`  | CUI    | Universal DSL JSON    |
| `cui-desktop` | CUI    | Universal DSL JSON    |

## Writer Caching

Writers are cached per context to avoid recreating them:

```go
// Get or create writer (cached)
writer := output.GetWriter(ctx)

// Clear cache when done
output.Close(ctx)  // Also closes the writer
```

## See Also

- [BUILTIN_TYPES.md](./BUILTIN_TYPES.md) - Complete documentation of built-in message types
- [adapters/openai/README.md](./adapters/openai/README.md) - OpenAI adapter documentation
