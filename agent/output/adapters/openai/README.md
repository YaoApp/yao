# OpenAI Adapter

OpenAI adapter converts universal DSL messages to OpenAI-compatible format.

## Conversion Rules

### Built-in Types (Standard)

These types are defined in `output.types.go` and have standardized Props structures that all adapters must support:

| Message Type | Constant              | Props Structure | OpenAI Format             | Description                               |
| ------------ | --------------------- | --------------- | ------------------------- | ----------------------------------------- |
| `text`       | `output.TypeText`     | `TextProps`     | `delta.content`           | Plain text or Markdown                    |
| `thinking`   | `output.TypeThinking` | `ThinkingProps` | `delta.reasoning_content` | Reasoning process (o1 models)             |
| `loading`    | `output.TypeLoading`  | `LoadingProps`  | `delta.reasoning_content` | Loading indicator (shows as thinking)     |
| `tool_call`  | `output.TypeToolCall` | `ToolCallProps` | `delta.tool_calls`        | Tool/function calls                       |
| `error`      | `output.TypeError`    | `ErrorProps`    | `error`                   | Error messages                            |
| `action`     | `output.TypeAction`   | `ActionProps`   | (not sent)                | System actions (silent)                   |
| `event`      | `output.TypeEvent`    | `EventProps`    | (conditional)             | Lifecycle events (stream_start converted) |

### Event Type (Lifecycle Events)

The `event` type has special handling in the OpenAI adapter:

| Event Name     | Conversion                                  | Example Output                                      |
| -------------- | ------------------------------------------- | --------------------------------------------------- |
| `stream_start` | Converted to trace link (with i18n support) | 🔍 智能体正在处理 - [查看处理详情](/trace/xxx/view) |
| Other events   | Silent (not sent)                           | -                                                   |

**Conversion Logic for `stream_start`:**

1. **Extract trace data**: Gets `TraceID` from event data
2. **Check model capabilities**: Determines if model supports reasoning
3. **Format based on capabilities**:
   - **Reasoning models** (o1, DeepSeek R1): Uses `reasoning_content` field with 🔍 icon
   - **Regular models**: Uses `content` field with 🚀 icon
4. **Apply i18n**: Uses locale from context for localized text
5. **Generate trace link**: Creates clickable link to `/trace/{traceID}/view` for standalone viewing

**Example Conversion:**

```go
// Input (event message)
{
  "type": "event",
  "props": {
    "event": "stream_start",
    "message": "Stream started",
    "data": {
      "trace_id": "20251122779905354593",
      "request_id": "ctx-1763779905679380000",
      "chat_id": "uP4CWZCMHy84nCw7"
    }
  }
}

// Output (reasoning model - Chinese locale)
{
  "choices": [{
    "delta": {
      "reasoning_content": "🔍 智能体正在处理 - [查看处理详情](http://localhost:8000/__yao_admin_root/trace/20251122779905354593/view)\n"
    }
  }]
}

// Output (regular model - English locale)
{
  "choices": [{
    "delta": {
      "content": "🚀 Assistant is processing - [View process](http://localhost:8000/__yao_admin_root/trace/20251122779905354593/view)\n"
    }
  }]
}
```

**Internationalization:**

The adapter uses `i18n.T()` to provide localized text:

| Key                   | English (en-us)         | Chinese (zh-cn) |
| --------------------- | ----------------------- | --------------- |
| `output.stream_start` | Assistant is processing | 智能体正在处理  |
| `output.view_trace`   | View process            | 查看处理详情    |

### Custom Types

All other message types (not in the built-in list) are converted to Markdown links:

| Format                 | Example                         |
| ---------------------- | ------------------------------- |
| `delta.content` (link) | `"🖼️ [View Image](https://...)` |

## Usage

### Basic Usage

```go
import (
    "github.com/yaoapp/yao/agent/output/adapters/openai"
)

// Create adapter with default config
adapter := openai.NewAdapter()

// Convert message
chunks, err := adapter.Adapt(msg)
```

### With Custom Configuration

```go
// Create adapter with options
adapter := openai.NewAdapter(
    openai.WithBaseURL("https://api.example.com"),
    openai.WithModel("gpt-4"),
    openai.WithLinkTemplate("image", "🖼️ [View Image](%s)"),
    openai.WithLinkTransformer(myOTPTransformer),
)
```

### With Link Transformer (OTP)

```go
// Define OTP transformer
func otpTransformer(url string, msgType string, msgID string) (string, error) {
    // Generate OTP token
    otp := generateOTP(msgID, 3600) // 1 hour expiry

    // Create short link with OTP
    shortURL := fmt.Sprintf("https://api.example.com/s/%s?t=%s", msgID, otp)

    return shortURL, nil
}

// Use transformer
adapter := openai.NewAdapter(
    openai.WithLinkTransformer(otpTransformer),
)
```

### Custom Converter

```go
// Register custom converter for a specific type
adapter := openai.NewAdapter(
    openai.WithConverter("my_widget", func(msg *output.Message, config *openai.AdapterConfig) ([]interface{}, error) {
        // Custom conversion logic
        return []interface{}{
            // OpenAI format chunk
        }, nil
    }),
)
```

## Examples

### Text Message (Built-in Type)

**Input (DSL):**

```json
{
  "type": "text",
  "props": {
    "content": "Hello world"
  }
}
```

Or using helper:

```go
msg := output.NewTextMessage("Hello world")
```

**Output (OpenAI):**

```json
{
  "id": "M1",
  "object": "chat.completion.chunk",
  "model": "yao-agent",
  "choices": [
    {
      "delta": {
        "content": "Hello world"
      }
    }
  ]
}
```

### Image Message

**Input (DSL):**

```json
{
  "message_id": "M2",
  "type": "image",
  "props": {
    "url": "https://example.com/avatar.jpg"
  }
}
```

**Output (OpenAI):**

```json
{
  "id": "M2",
  "object": "chat.completion.chunk",
  "model": "yao-agent",
  "choices": [
    {
      "delta": {
        "content": "🖼️ [View Image](https://api.example.com/s/M2?t=abc123)"
      }
    }
  ]
}
```

### Button Message

**Input (DSL):**

```json
{
  "message_id": "M3",
  "type": "button",
  "props": {
    "text": "Approve",
    "action": "workflow.approve"
  }
}
```

**Output (OpenAI):**

```json
{
  "id": "M3",
  "object": "chat.completion.chunk",
  "model": "yao-agent",
  "choices": [
    {
      "delta": {
        "content": "🔘 [Approve](https://api.example.com/s/M3?t=abc123)"
      }
    }
  ]
}
```

## Link Templates

Default templates:

```go
"image":  "🖼️ [View Image](%s)"
"audio":  "🔊 [Play Audio](%s)"
"video":  "🎬 [Watch Video](%s)"
"file":   "📎 [Download File](%s)"
"page":   "📄 [Open Page](%s)"
"table":  "📊 [View Table](%s)"
"chart":  "📈 [View Chart](%s)"
"list":   "📋 [View List](%s)"
"form":   "📝 [Fill Form](%s)"
"button": "🔘 [%s](%s)" // Special: button text + link
```

Customize templates:

```go
adapter := openai.NewAdapter(
    openai.WithLinkTemplate("image", "📷 Image: %s"),
    openai.WithLinkTemplate("video", "🎥 Watch: %s"),
)
```

## Link Transformer (TODO)

The link transformer is currently left empty for future implementation of OTP/short link functionality.

**Planned features:**

- Generate one-time password (OTP) for secure access
- Create short URLs for better readability
- Set expiration time for links
- Track link access for analytics

**Example implementation:**

```go
func otpTransformer(url string, msgType string, msgID string) (string, error) {
    // TODO: Implement OTP generation
    // 1. Generate OTP token with expiry
    // 2. Store mapping: token -> (url, msgType, msgID, expiry)
    // 3. Create short URL with token
    // 4. Return short URL

    return url, nil // Currently pass-through
}
```
