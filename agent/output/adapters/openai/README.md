# OpenAI Adapter

OpenAI adapter converts universal DSL messages to OpenAI-compatible format.

## Conversion Rules

### Built-in Types (Standard)

These types are defined in `output.types.go` and have standardized Props structures that all adapters must support:

| Message Type | Constant              | Props Structure | OpenAI Format             | Description                           |
| ------------ | --------------------- | --------------- | ------------------------- | ------------------------------------- |
| `text`       | `output.TypeText`     | `TextProps`     | `delta.content`           | Plain text or Markdown                |
| `thinking`   | `output.TypeThinking` | `ThinkingProps` | `delta.reasoning_content` | Reasoning process (o1 models)         |
| `loading`    | `output.TypeLoading`  | `LoadingProps`  | `delta.reasoning_content` | Loading indicator (shows as thinking) |
| `tool_call`  | `output.TypeToolCall` | `ToolCallProps` | `delta.tool_calls`        | Tool/function calls                   |
| `error`      | `output.TypeError`    | `ErrorProps`    | `error`                   | Error messages                        |
| `action`     | `output.TypeAction`   | `ActionProps`   | (not sent)                | System actions (silent)               |

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
  "id": "M2",
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
  "id": "M3",
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
