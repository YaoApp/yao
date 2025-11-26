# Output Module

The output module provides a unified API for sending messages to different client types (CUI, OpenAI-compatible, etc.) with support for streaming and rich media content.

## Architecture

```
agent/output/
├── message/              # Core types and interfaces (no dependencies)
│   ├── types.go         # Message, Group, Props structures
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

    // Streaming control - Hierarchical structure for Agent/LLM/MCP streaming
    ChunkID   string `json:"chunk_id,omitempty"`   // Unique chunk ID (C1, C2, C3...; for dedup/ordering/debugging)
    MessageID string `json:"message_id,omitempty"` // Logical message ID (M1, M2, M3...; delta merge target; multiple chunks → one message)
    BlockID   string `json:"block_id,omitempty"`   // Block ID (B1, B2, B3...; Agent-level grouping for UI sections)
    ThreadID  string `json:"thread_id,omitempty"`  // Thread ID (T1, T2, T3...; optional; for concurrent streams)

    // Delta control
    Delta       bool   `json:"delta,omitempty"`        // Whether this is an incremental update
    DeltaPath   string `json:"delta_path,omitempty"`   // Which field to update (e.g., "content", "items.0.name")
    DeltaAction string `json:"delta_action,omitempty"` // How to update ("append", "replace", "merge", "set")

    // Type correction (for streaming type inference)
    TypeChange bool `json:"type_change,omitempty"` // Marks this as a type correction message

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

Hierarchical structure for fine-grained control over streaming in complex Agent/LLM/MCP scenarios:

- **`ChunkID`** (optional): Unique chunk identifier

  - Auto-generated (C1, C2, C3...)
  - For deduplication, ordering, and debugging
  - Each raw stream fragment gets a unique ChunkID

- **`MessageID`** (optional): Logical message identifier

  - Auto-generated (M1, M2, M3...)
  - Delta merge target - multiple chunks with same MessageID are merged
  - Represents one complete logical message (e.g., one thinking output, one text response)
  - Example: `"M1"`

- **`BlockID`** (optional): Output block identifier

  - Auto-generated (B1, B2, B3...)
  - Agent-level grouping for UI sections
  - One LLM call, one MCP call, or one Agent sub-task
  - Used for rendering blocks/sections in the UI

- **`ThreadID`** (optional): Thread identifier

  - Auto-generated (T1, T2, T3...)
  - For concurrent Agent/LLM/MCP calls
  - Distinguishes multiple parallel output streams

- **`Delta`** (optional): Marks this as an incremental update
  - `true`: Append/update to existing message with same MessageID
  - `false`: Complete message (default)
  - Used for streaming LLM responses

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
  "chunk_id": "C1",
  "message_id": "M1",
  "type": "text",
  "delta": true,
  "props": {
    "content": "Hello"
  }
}

// Second chunk (appends)
{
  "chunk_id": "C2",
  "message_id": "M1",
  "type": "text",
  "delta": true,
  "props": {
    "content": ", world"
  }
}

// Third chunk
{
  "chunk_id": "C3",
  "message_id": "M1",
  "type": "text",
  "delta": true,
  "props": {
    "content": "!"
  }
}

// Completion signaled by message_end event (sent separately)
{
  "type": "event",
  "props": {
    "event": "message_end",
    "data": {
      "message_id": "M1",
      "type": "text",
      "chunk_count": 3,
      "status": "completed"
    }
  }
}
```

#### Complex Type with Nested Updates

```json
// Initial message
{
  "message_id": "M2",
  "type": "table",
  "props": {
    "columns": ["Name", "Age"],
    "rows": []
  }
}

// Add first row
{
  "chunk_id": "C4",
  "message_id": "M2",
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
  "chunk_id": "C5",
  "message_id": "M2",
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
  "chunk_id": "C6",
  "message_id": "M3",
  "type": "text",
  "delta": true,
  "props": {
    "content": "Let me think..."
  }
}

// Correction (actually thinking)
{
  "chunk_id": "C7",
  "message_id": "M3",
  "type": "thinking",
  "type_change": true,
  "props": {
    "content": "Let me think..."
  }
}
```

#### Block Grouping (Agent-level)

```json
// Block start event
{
  "type": "event",
  "props": {
    "event": "block_start",
    "data": {
      "block_id": "B1",
      "type": "llm",
      "label": "Analyzing image"
    }
  }
}

// Thinking message in block
{
  "message_id": "M4",
  "block_id": "B1",
  "type": "thinking",
  "props": {
    "content": "Let me analyze this image..."
  }
}

// Text message in block
{
  "message_id": "M5",
  "block_id": "B1",
  "type": "text",
  "props": {
    "content": "This is a beautiful sunset at Golden Gate Bridge"
  }
}

// Block end event
{
  "type": "event",
  "props": {
    "event": "block_end",
    "data": {
      "block_id": "B1",
      "message_count": 2,
      "status": "completed"
    }
  }
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

| Type        | Purpose            | CUI     | OpenAI                                       |
| ----------- | ------------------ | ------- | -------------------------------------------- |
| `text`      | Text content       | Direct  | `delta.content`                              |
| `thinking`  | LLM reasoning      | Direct  | `delta.reasoning_content`                    |
| `loading`   | Progress indicator | Direct  | `delta.reasoning_content`                    |
| `tool_call` | Function calls     | Direct  | `delta.tool_calls`                           |
| `error`     | Error messages     | Direct  | `error`                                      |
| `image`     | Images             | Render  | `![](url)` markdown                          |
| `audio`     | Audio              | Player  | Link                                         |
| `video`     | Video              | Player  | Link                                         |
| `action`    | System commands    | Execute | Silent                                       |
| `event`     | Lifecycle events   | Track   | Conditional (stream_start converted to link) |

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
// Get ID generator from context
idGen := ctx.IDGenerator

// Send delta (incremental) updates
msg := &message.Message{
    ChunkID:   idGen.GenerateChunkID(),   // C1
    MessageID: idGen.GenerateMessageID(), // M1
    Type:      message.TypeText,
    Delta:     true,  // Incremental update
    Props: map[string]interface{}{
        "content": "Hello",
    },
}
output.Send(ctx, msg)

// Send more delta updates (same MessageID for merging)
msg2 := &message.Message{
    ChunkID:   idGen.GenerateChunkID(),   // C2
    MessageID: msg.MessageID,             // M1 (same as before)
    Type:      message.TypeText,
    Delta:     true,
    Props: map[string]interface{}{
        "content": " world",
    },
}
output.Send(ctx, msg2)

// Mark completion with message_end event
endData := message.EventMessageEndData{
    MessageID:  msg.MessageID, // M1
    Type:       "text",
    Status:     "completed",
    ChunkCount: 2,
    Extra: map[string]interface{}{
        "content": "Hello world!", // Full content
    },
}
eventMsg := output.NewEventMessage(message.EventMessageEnd, "Message completed", endData)
output.Send(ctx, eventMsg)
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
