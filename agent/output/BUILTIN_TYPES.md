# Built-in Message Types

Built-in message types are standardized types that all adapters must support. These types have predefined Props structures to ensure consistency across different output formats.

## Type Constants

Defined in `types.go`:

```go
const (
    TypeText     = "text"      // Plain text or Markdown content
    TypeThinking = "thinking"  // Reasoning/thinking process
    TypeLoading  = "loading"   // Loading/processing indicator
    TypeToolCall = "tool_call" // LLM tool/function call
    TypeError    = "error"     // Error message
    TypeImage    = "image"     // Image content
    TypeAudio    = "audio"     // Audio content
    TypeVideo    = "video"     // Video content
    TypeAction   = "action"    // System action (silent in standard clients)
    TypeEvent    = "event"     // Lifecycle event (silent in standard clients)
)
```

## Standard Props Structures

### 1. Text (`text`)

**Purpose:** Plain text or Markdown content

**Props Structure:**

```go
type TextProps struct {
    Content string `json:"content"` // Text content (supports Markdown)
}
```

**Example:**

```json
{
  "type": "text",
  "props": {
    "content": "Hello **world**!"
  }
}
```

**Helper:**

```go
msg := output.NewTextMessage("Hello **world**!")
```

---

### 2. Thinking (`thinking`)

**Purpose:** Reasoning or thinking process (used by o1 models, DeepSeek R1, etc.)

**Props Structure:**

```go
type ThinkingProps struct {
    Content string `json:"content"` // Reasoning/thinking content
}
```

**Example:**

```json
{
  "type": "thinking",
  "props": {
    "content": "Let me analyze this step by step..."
  }
}
```

**Helper:**

```go
msg := output.NewThinkingMessage("Let me analyze this step by step...")
```

---

### 3. Loading (`loading`)

**Purpose:** Loading or processing indicator (preprocessing, knowledge base search, data fetching, etc.)

**Props Structure:**

```go
type LoadingProps struct {
    Message string `json:"message"` // Loading message
}
```

**Example:**

```json
{
  "type": "loading",
  "props": {
    "message": "Searching knowledge base..."
  }
}
```

**Helper:**

```go
msg := output.NewLoadingMessage("Searching knowledge base...")
```

**Use Cases:**

- Knowledge base search: `"Searching knowledge base..."`
- Data preprocessing: `"Processing uploaded file..."`
- External API calls: `"Fetching data from API..."`
- Database queries: `"Querying database..."`

**Example in Hook:**

```go
// In Create hook, show preprocessing steps
func Create(ctx *context.Context, messages []context.Message) (*context.HookCreateResponse, error) {
    // Send loading message for knowledge base search
    output.Send(ctx, output.NewLoadingMessage("Searching knowledge base..."))

    // Do the actual search
    results := searchKnowledgeBase(messages)

    // Send another loading message for processing
    output.Send(ctx, output.NewLoadingMessage("Processing results..."))

    // Process and return
    return &context.HookCreateResponse{
        Messages: buildMessages(results),
    }, nil
}
```

**Result in OpenAI Client:**

- Shows as thinking/reasoning process
- User sees "Searching knowledge base..." and "Processing results..."
- Provides transparency into what's happening

---

### 4. Tool Call (`tool_call`)

**Purpose:** LLM tool or function call

**Props Structure:**

```go
type ToolCallProps struct {
    ID        string `json:"id"`                  // Tool call ID
    Name      string `json:"name"`                // Function/tool name
    Arguments string `json:"arguments,omitempty"` // JSON string of arguments
}
```

**Example:**

```json
{
  "type": "tool_call",
  "props": {
    "id": "call_abc123",
    "name": "get_weather",
    "arguments": "{\"location\": \"San Francisco\"}"
  }
}
```

**Helper:**

```go
msg := output.NewToolCallMessage(
    "call_abc123",
    "get_weather",
    "{\"location\": \"San Francisco\"}",
)
```

---

### 5. Error (`error`)

**Purpose:** Error message

**Props Structure:**

```go
type ErrorProps struct {
    Message string `json:"message"`           // Error message
    Code    string `json:"code,omitempty"`    // Error code
    Details string `json:"details,omitempty"` // Additional error details
}
```

**Example:**

```json
{
  "type": "error",
  "props": {
    "message": "Connection timeout",
    "code": "TIMEOUT",
    "details": "Failed to connect to database after 30s"
  }
}
```

**Helper:**

```go
msg := output.NewErrorMessage("Connection timeout", "TIMEOUT")
```

---

### 6. Action (`action`)

**Purpose:** System-level action/command (not displayed to user, only processed by client)

**Props Structure:**

```go
type ActionProps struct {
    Name    string                 `json:"name"`              // Action name
    Payload map[string]interface{} `json:"payload,omitempty"` // Action parameters
}
```

**Example:**

```json
{
  "type": "action",
  "props": {
    "name": "open_panel",
    "payload": {
      "panel_id": "user_profile",
      "user_id": "123"
    }
  }
}
```

**Helper:**

```go
msg := output.NewActionMessage("open_panel", map[string]interface{}{
    "panel_id": "user_profile",
    "user_id": "123",
})
```

**Use Cases:**

- Open sidebar/panel: `"open_panel"`
- Navigate to page: `"navigate"`
- Trigger UI update: `"refresh_view"`
- Close modal: `"close_modal"`
- Scroll to element: `"scroll_to"`

**Important Notes:**

- **Silent in OpenAI clients**: Action messages are NOT sent to standard chat clients
- **CUI clients only**: Only CUI clients process action messages
- **System-level**: Used for controlling the UI/application, not chat content

**Example in Hook:**

```go
// Send action to open a panel with user details
output.Send(ctx, output.NewActionMessage("open_panel", map[string]interface{}{
    "panel_id": "user_details",
    "user_id":  user.ID,
}))

// Send text message (visible to user)
output.Send(ctx, output.NewTextMessage("I've opened the user details panel for you."))
```

**Result:**

- **CUI client**: Panel opens, text message displays
- **OpenAI client**: Only text message displays (action is silent)

---

### 7. Event (`event`)

**Purpose:** Lifecycle event messages (stream_start, stream_end, connecting, etc.)

**Props Structure:**

```go
type EventProps struct {
    Event   string                 `json:"event"`             // Event type
    Message string                 `json:"message,omitempty"` // Human-readable message
    Data    map[string]interface{} `json:"data,omitempty"`    // Additional event data
}
```

**Example:**

```json
{
  "type": "event",
  "props": {
    "event": "stream_start",
    "message": "Starting stream...",
    "data": {
      "model": "gpt-4",
      "session_id": "sess_123"
    }
  }
}
```

**Helper:**

```go
msg := output.NewEventMessage("stream_start", "Starting stream...", map[string]interface{}{
    "model": "gpt-4",
    "session_id": "sess_123",
})
```

**Use Cases:**

- Stream lifecycle: `"stream_start"`, `"stream_end"`
- Connection status: `"connecting"`, `"connected"`, `"disconnected"`
- Processing stages: `"preprocessing"`, `"postprocessing"`
- Agent state: `"thinking"`, `"executing"`, `"completed"`

**Important Notes:**

- **Silent in OpenAI clients**: Event messages are NOT sent to standard chat clients
- **CUI clients only**: Only CUI clients process event messages
- **Lifecycle tracking**: Used for tracking agent/stream lifecycle, not chat content
- **Non-blocking**: Events don't interrupt the main message flow

**Example in Hook:**

```go
// Send stream start event
output.Send(ctx, output.NewEventMessage("stream_start", "Initializing...", map[string]interface{}{
    "timestamp": time.Now().Unix(),
}))

// Do processing
processData()

// Send stream end event
output.Send(ctx, output.NewEventMessage("stream_end", "Stream completed", map[string]interface{}{
    "duration_ms": 1500,
}))
```

**Result:**

- **CUI client**: Tracks lifecycle, may show status indicators
- **OpenAI client**: Events are silent (not sent to client)

---

### 8. Image (`image`)

**Purpose:** Image content

**Props Structure:**

```go
type ImageProps struct {
    URL    string  // Required: Image URL or base64 data
    Alt    string  // Alternative text
    Width  int     // Image width in pixels
    Height int     // Image height in pixels
    Detail string  // OpenAI detail level: "auto", "low", "high"
}
```

**Example:**

```json
{
  "type": "image",
  "props": {
    "url": "https://example.com/avatar.jpg",
    "alt": "User avatar",
    "width": 200,
    "height": 200
  }
}
```

**Helper:**

```go
msg := output.NewImageMessage("https://example.com/avatar.jpg", "User avatar")
```

**Adapter Behavior:**

- **CUI**: Renders image directly with `<img>` tag
- **OpenAI**: Converts to Markdown `![alt](url)` - **displays inline** in Markdown-supporting clients

---

### 9. Audio (`audio`)

**Purpose:** Audio content

**Props Structure:**

```go
type AudioProps struct {
    URL        string   // Required: Audio URL or base64 data
    Format     string   // Audio format: "mp3", "wav", "ogg"
    Duration   float64  // Duration in seconds
    Transcript string   // Audio transcript text
    Autoplay   bool     // Whether to autoplay
    Controls   bool     // Whether to show controls
}
```

**Example:**

```json
{
  "type": "audio",
  "props": {
    "url": "https://example.com/audio.mp3",
    "format": "mp3",
    "duration": 120.5,
    "transcript": "This is the audio content...",
    "controls": true
  }
}
```

**Helper:**

```go
msg := output.NewAudioMessage("https://example.com/audio.mp3", "mp3")
```

**Adapter Behavior:**

- **CUI**: Renders audio player with controls
- **OpenAI**: Converts to link `🔊 [Play Audio](url)` - can't display inline

---

### 10. Video (`video`)

**Purpose:** Video content

**Props Structure:**

```go
type VideoProps struct {
    URL       string   // Required: Video URL
    Format    string   // Video format: "mp4", "webm"
    Duration  float64  // Duration in seconds
    Thumbnail string   // Thumbnail/poster image URL
    Width     int      // Video width in pixels
    Height    int      // Video height in pixels
    Autoplay  bool     // Whether to autoplay
    Controls  bool     // Whether to show controls
    Loop      bool     // Whether to loop
}
```

**Example:**

```json
{
  "type": "video",
  "props": {
    "url": "https://example.com/video.mp4",
    "format": "mp4",
    "thumbnail": "https://example.com/poster.jpg",
    "width": 640,
    "height": 360,
    "controls": true
  }
}
```

**Helper:**

```go
msg := output.NewVideoMessage("https://example.com/video.mp4")
```

**Adapter Behavior:**

- **CUI**: Renders video player with controls
- **OpenAI**: Converts to link `🎬 [Watch Video](url)` - can't display inline

---

## Adapter Requirements

All adapters (CUI, OpenAI, etc.) **must** support these built-in types with their standard Props structures.

### CUI Adapter

CUI adapter passes built-in types through without transformation:

```json
{
  "type": "text",
  "props": {
    "content": "Hello world"
  }
}
```

### OpenAI Adapter

OpenAI adapter converts built-in types to OpenAI format:

| Type        | OpenAI Format             | Field                         | Note                                      |
| ----------- | ------------------------- | ----------------------------- | ----------------------------------------- |
| `text`      | `delta.content`           | `props.content`               |                                           |
| `thinking`  | `delta.reasoning_content` | `props.content`               | Reasoning content (o1 models)             |
| `loading`   | `delta.reasoning_content` | `props.message`               | Shows as thinking in OpenAI clients       |
| `tool_call` | `delta.tool_calls`        | `props.{id, name, arguments}` |                                           |
| `error`     | `error`                   | `props.{message, code}`       |                                           |
| `image`     | `delta.content`           | `props.{url, alt}`            | Markdown: `![alt](url)` - displays inline |
| `audio`     | `delta.content`           | `props.url`                   | Markdown link (can't display inline)      |
| `video`     | `delta.content`           | `props.url`                   | Markdown link (can't display inline)      |
| `action`    | (not sent)                | -                             | Silent - system actions only              |
| `event`     | (not sent)                | -                             | Silent - lifecycle events only            |

---

## Custom Types

Any type **not** in the built-in list is considered a custom type. Adapters may handle custom types differently:

- **CUI:** Pass through as-is
- **OpenAI:** Convert to Markdown link

Example custom type:

```json
{
  "type": "image",
  "props": {
    "url": "https://example.com/image.jpg",
    "alt": "Description"
  }
}
```

---

## Checking Built-in Types

```go
// Check if a type is built-in
if output.IsBuiltinType(msg.Type) {
    // Handle as standard type
} else {
    // Handle as custom type
}
```

---

## Guidelines for New Built-in Types

When adding new built-in types:

1. ✅ Add constant to `types.go`
2. ✅ Define Props structure
3. ✅ Add helper function in `builtin.go`
4. ✅ Update all adapters to support it
5. ✅ Document in this file
6. ✅ Add tests

**Only add built-in types for:**

- Universal concepts (text, errors, etc.)
- LLM-specific features (thinking, tool_calls)
- Types that need cross-adapter consistency

**Do NOT add built-in types for:**

- UI components (buttons, forms, etc.)
- Application-specific widgets
- Domain-specific data types

These should remain custom types.
