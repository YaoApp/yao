# Built-in Message Types

Built-in message types are standardized types that all adapters must support. These types have predefined Props structures to ensure consistency across different output formats.

## Type Constants

Defined in `types.go`:

```go
const (
    TypeUserInput  = "user_input"  // User input message (frontend display only)
    TypeText       = "text"        // Plain text or Markdown content
    TypeThinking   = "thinking"    // Reasoning/thinking process
    TypeLoading    = "loading"     // Loading/processing indicator
    TypeToolCall   = "tool_call"   // LLM tool/function call
    TypeRetrieval  = "retrieval"   // KB/Web search results (for feedback & analytics)
    TypeError      = "error"       // Error message
    TypeImage      = "image"       // Image content
    TypeAudio      = "audio"       // Audio content
    TypeVideo      = "video"       // Video content
    TypeAction     = "action"      // System action (silent in standard clients)
    TypeEvent      = "event"       // Lifecycle event (silent in standard clients)
)
```

## Standard Props Structures

### 1. User Input (`user_input`)

**Purpose:** User input message (for frontend display only)

**Props Structure:**

```go
type UserInputProps struct {
    Content interface{} `json:"content"`        // User input (text string or multimodal ContentPart[])
    Role    string      `json:"role,omitempty"` // User role: "user", "system", "developer" (default: "user")
    Name    string      `json:"name,omitempty"` // Optional participant name
}
```

**Example:**

```json
{
  "type": "user_input",
  "props": {
    "content": "Hello, can you help me?",
    "role": "user"
  }
}
```

**Multimodal Example:**

```json
{
  "type": "user_input",
  "props": {
    "content": [
      {
        "type": "text",
        "text": "What's in this image?"
      },
      {
        "type": "image_url",
        "image_url": {
          "url": "https://example.com/photo.jpg"
        }
      }
    ],
    "role": "user"
  }
}
```

**Helper:**

```go
// Simple text input
msg := output.NewUserInputMessage("Hello, can you help me?", "user", "")

// With name
msg := output.NewUserInputMessage("I need assistance", "user", "John")

// Multimodal content
content := []map[string]interface{}{
    {
        "type": "text",
        "text": "What's in this image?",
    },
    {
        "type": "image_url",
        "image_url": map[string]string{
            "url": "https://example.com/photo.jpg",
        },
    },
}
msg := output.NewUserInputMessage(content, "user", "")
```

**Important Notes:**

- **Frontend display only**: This type is used by the frontend to display user input in the chat UI
- **Not sent to backend**: User input is sent to backend as `UserMessage` (OpenAI format), not as `Message`
- **Preserves role**: Unlike `text` type, preserves the original user role (`user`, `system`, `developer`)
- **Supports multimodal**: Can contain text, images, audio, or files

**Data Flow:**

```
User types → UserMessage (sent to API) → Backend processes → Message types (AI response)
           ↓
           UserInputMessage (frontend display)
```

---

### 2. Text (`text`)

**Purpose:** Plain text or Markdown content (AI responses)

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

### 3. Thinking (`thinking`)

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

### 4. Loading (`loading`)

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

### 5. Tool Call (`tool_call`)

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

### 6. Retrieval (`retrieval`)

**Purpose:** Knowledge base and web search results (for feedback, analytics, and source attribution)

**Props Structure:**

```go
type RetrievalProps struct {
    Query        string           `json:"query"`                   // Search query
    Sources      []RetrievalSource `json:"sources"`                // Retrieved sources
    TotalResults int              `json:"total_results,omitempty"` // Total matching results
    QueryTimeMs  int64            `json:"query_time_ms,omitempty"` // Query execution time
    Provider     string           `json:"provider,omitempty"`      // Search provider (e.g., "tavily", "bing")
}

type RetrievalSource struct {
    ID           string                 `json:"id"`                      // Unique source ID within this retrieval
    Type         string                 `json:"type"`                    // Source type: "kb", "web", "file", "api", "mcp"
    Title        string                 `json:"title,omitempty"`         // Source title
    Content      string                 `json:"content"`                 // Retrieved content/snippet
    Score        float64                `json:"score,omitempty"`         // Relevance score
    URL          string                 `json:"url,omitempty"`           // URL for web sources
    CollectionID string                 `json:"collection_id,omitempty"` // KB collection ID
    DocumentID   string                 `json:"document_id,omitempty"`   // KB document ID
    ChunkID      string                 `json:"chunk_id,omitempty"`      // KB chunk ID
    Metadata     map[string]interface{} `json:"metadata,omitempty"`      // Additional metadata
}
```

**Example (Knowledge Base):**

```json
{
  "type": "retrieval",
  "props": {
    "query": "How to configure Yao models?",
    "sources": [
      {
        "id": "src_001",
        "type": "kb",
        "collection_id": "col_docs",
        "document_id": "doc_123",
        "chunk_id": "chunk_456",
        "title": "Model Configuration Guide",
        "content": "To configure a model in Yao, create a .mod.yao file...",
        "score": 0.92,
        "metadata": {
          "file_path": "/docs/model.md",
          "page": 3
        }
      }
    ],
    "total_results": 15,
    "query_time_ms": 120
  }
}
```

**Example (Web Search):**

```json
{
  "type": "retrieval",
  "props": {
    "query": "latest AI news 2024",
    "sources": [
      {
        "id": "src_001",
        "type": "web",
        "url": "https://example.com/ai-news",
        "title": "AI Breakthroughs in 2024",
        "content": "Summary of the article...",
        "score": 0.95,
        "metadata": {
          "domain": "example.com",
          "published_at": "2024-01-10"
        }
      }
    ],
    "provider": "tavily",
    "total_results": 10,
    "query_time_ms": 850
  }
}
```

**Helper:**

```go
msg := output.NewRetrievalMessage(
    "How to configure Yao models?",
    []output.RetrievalSource{
        {
            ID:           "src_001",
            Type:         "kb",
            CollectionID: "col_docs",
            DocumentID:   "doc_123",
            ChunkID:      "chunk_456",
            Title:        "Model Configuration Guide",
            Content:      "To configure a model in Yao...",
            Score:        0.92,
        },
    },
)
```

**Source Types:**

| Type   | Description             | Key Fields                                 |
| ------ | ----------------------- | ------------------------------------------ |
| `kb`   | Knowledge base document | `collection_id`, `document_id`, `chunk_id` |
| `web`  | Web search result       | `url`                                      |
| `file` | Uploaded file           | `file_id`, `file_path`                     |
| `api`  | External API result     | `api_name`, `endpoint`                     |
| `mcp`  | MCP tool result         | `server`, `tool`                           |

**Use Cases:**

- **Source Attribution**: Display citations in the chat UI
- **User Feedback**: Allow users to rate individual sources (👍/👎)
- **Analytics**: Track which documents/sources are most useful
- **RAG Optimization**: Improve retrieval based on feedback data

**Adapter Behavior:**

- **CUI**: Renders as expandable source cards with feedback buttons
- **OpenAI**: Converts to markdown citations or footnotes

---

### 7. Error (`error`)

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

### 8. Action (`action`)

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

### 9. Event (`event`)

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

- **Converted in OpenAI clients**: Event messages are typically NOT sent to OpenAI clients, **except** `stream_start`:
  - `stream_start`: Converted to a clickable trace link in either `reasoning_content` (thinking models) or `content` (regular models)
  - Other events: Silent (not sent to OpenAI clients)
- **CUI clients**: All event messages are processed and may show status indicators
- **Lifecycle tracking**: Used for tracking agent/stream lifecycle
- **Non-blocking**: Events don't interrupt the main message flow

**Example in Hook:**

```go
// Send stream start event (automatically generated by assistant)
// This is typically handled by the framework, not manually sent
startData := message.EventStreamStartData{
    RequestID: ctx.RequestID,
    Timestamp: time.Now().UnixMilli(),
    TraceID:   ctx.Stack.TraceID,
    ChatID:    ctx.ChatID,
}
output.Send(ctx, output.NewEventMessage("stream_start", "Stream started", startData))

// Do processing
processData()

// Send stream end event
endData := message.EventStreamEndData{
    RequestID:  ctx.RequestID,
    Timestamp:  time.Now().UnixMilli(),
    DurationMs: 1500,
    Status:     "completed",
}
output.Send(ctx, output.NewEventMessage("stream_end", "Stream completed", endData))
```

**Result:**

- **CUI client**: Tracks lifecycle, may show status indicators
- **OpenAI client (stream_start only)**:
  - Reasoning models: Shows as 🔍 with trace link in `reasoning_content` field
  - Regular models: Shows as 🚀 with trace link in `content` field
  - Example: "🔍 智能体正在处理 - [查看处理详情](baseURL/trace/traceID/view)"
- **OpenAI client (other events)**: Silent (not sent)

---

### 10. Image (`image`)

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

### 11. Audio (`audio`)

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

### 12. Video (`video`)

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

| Type         | OpenAI Format             | Field                         | Note                                                                 |
| ------------ | ------------------------- | ----------------------------- | -------------------------------------------------------------------- |
| `user_input` | (not sent)                | -                             | Frontend display only - not sent to OpenAI clients                   |
| `text`       | `delta.content`           | `props.content`               |                                                                      |
| `thinking`   | `delta.reasoning_content` | `props.content`               | Reasoning content (o1 models)                                        |
| `loading`    | `delta.reasoning_content` | `props.message`               | Shows as thinking in OpenAI clients                                  |
| `tool_call`  | `delta.tool_calls`        | `props.{id, name, arguments}` |                                                                      |
| `retrieval`  | `delta.content`           | `props.sources`               | Markdown citations/footnotes with source links                       |
| `error`      | `error`                   | `props.{message, code}`       |                                                                      |
| `image`      | `delta.content`           | `props.{url, alt}`            | Markdown: `![alt](url)` - displays inline                            |
| `audio`      | `delta.content`           | `props.url`                   | Markdown link (can't display inline)                                 |
| `video`      | `delta.content`           | `props.url`                   | Markdown link (can't display inline)                                 |
| `action`     | (not sent)                | -                             | Silent - system actions only                                         |
| `event`      | (conditional)             | `props.{event, data}`         | Most events silent; `stream_start` converted to trace link with i18n |

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
