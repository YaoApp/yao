# LLM Providers Architecture

## Overview

This directory contains different LLM provider implementations, each optimized for specific model capabilities.

## Provider Selection Strategy

The `factory.SelectProvider()` function automatically selects the appropriate provider based on model capabilities:

```go
Priority 1: Reasoning models → reasoning.Provider
Priority 2: Native tool support → openai.Provider
Priority 3: Legacy models → legacy.Provider
```

## Provider Types

### 1. Base Provider (`base/`)

**Purpose**: Common functionality shared across all providers

**Features**:

- Message preprocessing
- Request body building
- Response parsing

**Usage**: Embedded in all other providers

---

### 2. OpenAI Provider (`openai/`)

**Purpose**: OpenAI-compatible models with full feature support

**Capabilities**:

- ✅ Vision (image input)
- ✅ Native tool calls
- ✅ Streaming
- ✅ JSON mode

**Models**:

- GPT-4, GPT-4o, GPT-4-turbo
- GPT-3.5-turbo
- Claude (via OpenAI-compatible API)

---

### 3. Reasoning Provider (`reasoning/`)

**Purpose**: Reasoning models with special response format

**Capabilities**:

- ✅ Reasoning content (`reasoning_content` field)
- ✅ Thinking + Answer phases
- ⚠️ Tool calls support varies by model

**Models**:

- **OpenAI o1** (supports native tool calls)
- **DeepSeek R1** (no native tool calls, uses prompt engineering)

**Special Handling**:

```go
// DeepSeek R1 scenario
if !supportsNativeTools && hasTools {
    // Inject tool instructions into prompt
    messages = injectToolInstructions(messages, tools)
    // Extract tool calls from text response
    toolCalls = extractToolCallsFromText(response.Content)
}
```

**Response Format**:

```json
{
  "content": "The answer is 42",
  "reasoning_content": "Let me think... first we need to...",
  "content_types": ["text", "reasoning"]
}
```

---

### 4. Legacy Provider (`legacy/`)

**Purpose**: Older models without native tool calling

**Capabilities**:

- ✅ Text generation
- ⚠️ Tool calls via prompt engineering
- ❌ No native vision support
- ❌ No native tool API

**Models**:

- GPT-3 (davinci, curie)
- Older open-source models
- Custom models without tool API

**Tool Call Flow**:

1. Inject tool schemas into system prompt
2. Model returns tool call in text format (JSON)
3. Extract and parse tool calls from text
4. Execute tools
5. Continue conversation

---

### 5. Vision Utils (`vision/`)

**Purpose**: Vision-related preprocessing utilities

**Functions**:

- `PreprocessVisionMessages()` - Handle image content
- `ConvertImageToText()` - Convert images to descriptions (for non-vision models)
- `ValidateImageURL()` - Validate image URLs
- `ExtractImagesFromMessages()` - Extract all images from messages

**Usage**:

```go
// When model doesn't support vision
if !supportsVision {
    messages = vision.PreprocessVisionMessages(messages, false)
    // Images converted to text descriptions
}
```

---

### 6. Audio Utils (`audio/`)

**Purpose**: Audio-related preprocessing utilities

**Functions**:

- `PreprocessAudioMessages()` - Handle audio content
- `ConvertAudioToText()` - Convert audio to text transcription (for non-audio models)
- `ValidateAudioFormat()` - Validate audio format and encoding
- `ExtractAudioFromMessages()` - Extract all audio data from messages
- `RemoveAudioConfig()` - Remove audio configuration from options

**Usage**:

```go
// When model doesn't support audio
if !supportsAudio {
    messages = audio.PreprocessAudioMessages(messages, false)
    options = audio.RemoveAudioConfig(options)
    // Audio converted to text transcriptions
}
```

---

## Special Scenarios

### Scenario 1: DeepSeek R1 (Reasoning + No Tool Support)

**Provider**: `reasoning.Provider`

**Handling**:

```go
// Check if reasoning model supports tools
if !p.supportsNativeTools && len(options.Tools) > 0 {
    // Use prompt engineering approach
    messages = p.injectToolInstructions(messages, tools)
    options = p.removeToolsFromOptions(options)
}

// After getting response
if !p.supportsNativeTools {
    toolCalls = p.extractToolCallsFromText(response.Content)
}
```

**Why reasoning provider?**

- Primary characteristic is reasoning (special response format)
- Tool handling is secondary concern
- Reuses tool injection logic from legacy approach

---

### Scenario 2: Legacy Model + Vision/Audio Request

**Provider**: `legacy.Provider`

**Handling**:

```go
import (
    "github.com/yaoapp/yao/agent/llm/providers/vision"
    "github.com/yaoapp/yao/agent/llm/providers/audio"
)

// Preprocess to remove/convert vision content
if !supportsVision {
    messages = vision.PreprocessVisionMessages(messages, false)
    // Images converted to text: "[Image: description]"
}

// Preprocess to remove/convert audio content
if !supportsAudio {
    messages = audio.PreprocessAudioMessages(messages, false)
    options = audio.RemoveAudioConfig(options)
    // Audio converted to text: "[Audio transcription: ...]"
}
```

---

### Scenario 3: OpenAI o1 (Reasoning + Tool Support)

**Provider**: `reasoning.Provider`

**Handling**:

```go
// o1 supports native tools, no special handling needed
if p.supportsNativeTools {
    // Use standard OpenAI tool calling API
}
```

---

## Configuration Example

In `connectors.yml`:

```yaml
# GPT-4o with all features
gpt-4o:
  vision: true
  tool_calls: true
  audio: true
  streaming: true
  json: true
  multimodal: true

# OpenAI o1 - reasoning with tool support
o1-preview:
  reasoning: true
  tool_calls: true
  streaming: true

# DeepSeek R1 - reasoning without tool support
deepseek-reasoner:
  reasoning: true
  tool_calls: false # Will use prompt engineering
  streaming: true

# GPT-3 - legacy model
gpt-3.5-turbo-instruct:
  tool_calls: false # Will use prompt engineering
  vision: false # Will convert images to text
  audio: false # Will convert audio to text
  streaming: false

# GPT-4 Vision only
gpt-4-vision:
  vision: true
  tool_calls: true
  audio: false # No audio support
  streaming: true
```

---

## Adding a New Provider

1. Create new directory: `providers/newprovider/`
2. Implement `LLM` interface:

   ```go
   type Provider struct {
       *base.Provider
   }

   func (p *Provider) Stream(...) (*CompletionResponse, error)
   func (p *Provider) Post(...) (*CompletionResponse, error)
   ```

3. Update `factory.SelectProvider()` selection logic
4. Add capability flags to `ConnectorSetting`

---

## Testing

Each provider should have tests for:

- Standard completion
- Streaming completion
- Tool calling (if supported)
- Vision input (if supported)
- Error handling
- Response parsing

---

## Performance Considerations

- **Caching**: Consider caching connector instances
- **Pooling**: HTTP connection pooling for high throughput
- **Timeouts**: Configurable timeouts per provider
- **Retries**: Exponential backoff for transient errors
