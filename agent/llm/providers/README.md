# LLM Providers Architecture (New)

## Overview

This directory contains LLM provider implementations using the **Capability Adapters** pattern. The new architecture separates API format handling from capability handling.

## Architecture Design

```
┌─────────────────────────────────────────────────┐
│           LLM Provider (API Format)             │
│  - OpenAI-compatible                            │
│  - Claude (TODO)                                │
│  - Custom (TODO)                                │
└──────────────┬──────────────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────────────┐
│        Capability Adapters (Modular)            │
│  - ToolCallAdapter    (native or prompt eng.)   │
│  - VisionAdapter      (native or removal)       │
│  - AudioAdapter       (native or removal)       │
│  - ReasoningAdapter   (o1/R1/GPT-Think)         │
└─────────────────────────────────────────────────┘
```

## Key Concepts

### 1. Provider = API Format

Providers handle the **API communication format**:
- OpenAI-compatible API (`/v1/chat/completions`)
- Claude API (TODO)
- Custom API formats (TODO)

### 2. Adapters = Capabilities

Adapters handle **model capabilities** independently:
- **ToolCallAdapter**: Tool calling (native or prompt engineering)
- **VisionAdapter**: Image input (native or removal/conversion)
- **AudioAdapter**: Audio input (native or removal/conversion)
- **ReasoningAdapter**: Reasoning content (o1/DeepSeek R1/GPT-4o thinking)

## Provider Selection

```go
// factory.go
func SelectProvider(conn connector.Connector, options *context.CompletionOptions) (LLM, error) {
    apiFormat := DetectAPIFormat(conn)
    
    switch apiFormat {
    case "openai":
        // Adapters automatically configured based on capabilities
        return openai.New(conn, options.Capabilities), nil
    case "claude":
        return claude.New(conn, options.Capabilities), nil
    default:
        return openai.New(conn, options.Capabilities), nil
    }
}
```

## Directory Structure

```
providers/
├── factory.go          # Provider selection based on API format
├── base/               # Common functionality
│   └── base.go
├── openai/             # OpenAI-compatible API provider
│   └── openai.go       # Includes adapter integration
└── README.md           # This file

../adapters/            # Capability adapters (separate package)
├── adapter.go          # Base interface
├── toolcall.go         # Tool calling adapter
├── vision.go           # Vision adapter
├── audio.go            # Audio adapter
└── reasoning.go        # Reasoning adapter
```

## OpenAI Provider

The OpenAI provider supports **all capabilities** through adapters:

```go
type Provider struct {
    *base.Provider
    adapters []adapters.CapabilityAdapter
}

func New(conn connector.Connector, capabilities *context.ModelCapabilities) *Provider {
    return &Provider{
        Provider: base.NewProvider(conn, capabilities),
        adapters: buildAdapters(capabilities), // Auto-configured
    }
}
```

### Adapter Pipeline

**Preprocessing** (before API call):
```
Messages → ToolCallAdapter → VisionAdapter → AudioAdapter → API Request
```

**Streaming** (during API call):
```
API Chunk → ReasoningAdapter → ToolCallAdapter → Output
```

**Postprocessing** (after API call):
```
API Response → All Adapters → Final Response
```

## Model Examples

### Full-Featured Model (GPT-4o)

```yaml
# connectors.yml
gpt-4o:
  vision: true
  tool_calls: true
  audio: true
  reasoning: false
```

**Adapters created**:
- ToolCallAdapter(native=true)
- VisionAdapter(native=true)
- AudioAdapter(native=true)

### Reasoning Model with Tools (OpenAI o1)

```yaml
o1-preview:
  reasoning: true
  tool_calls: true
```

**Adapters created**:
- ToolCallAdapter(native=true)
- ReasoningAdapter(format=openai-o1)

### Reasoning Model without Tools (DeepSeek R1)

```yaml
deepseek-reasoner:
  reasoning: true
  tool_calls: false
```

**Adapters created**:
- ToolCallAdapter(native=false) → Uses prompt engineering
- ReasoningAdapter(format=deepseek-r1)

### Legacy Model (GPT-3.5-instruct)

```yaml
gpt-3.5-turbo-instruct:
  tool_calls: false
  vision: false
  audio: false
```

**Adapters created**:
- ToolCallAdapter(native=false) → Prompt engineering
- VisionAdapter(native=false) → Removes images
- AudioAdapter(native=false) → Removes audio

## Capability Adapters

### ToolCallAdapter

**When native=true**:
- Passes tool definitions to API
- Parses structured tool_calls from response

**When native=false**:
- Injects tool schemas into system prompt
- Extracts tool calls from text response (JSON parsing)

### VisionAdapter

**When native=true**:
- Passes image URLs/data directly to API

**When native=false**:
- Removes image content from messages
- Optionally converts to text descriptions

### AudioAdapter

**When native=true**:
- Passes audio data directly to API

**When native=false**:
- Removes audio content from messages
- Optionally converts to text transcriptions

### ReasoningAdapter

Handles different reasoning formats:

**OpenAI o1** (`reasoning_content` field):
```json
{
  "delta": {
    "reasoning_content": "Let me think...",
    "content": "The answer is 42"
  }
}
```

**DeepSeek R1** (may have different format):
```json
{
  "delta": {
    "content": "<think>Let me think...</think>The answer is 42"
  }
}
```

**GPT-4o thinking** (future):
```json
{
  "delta": {
    "thinking": "Let me think...",
    "content": "The answer is 42"
  }
}
```

## Adding New Capabilities

1. Create new adapter in `../adapters/`:
   ```go
   type NewCapabilityAdapter struct {
       *BaseAdapter
       nativeSupport bool
   }
   ```

2. Implement CapabilityAdapter interface

3. Add to `buildAdapters()` in `openai/openai.go`:
   ```go
   if cap.NewCapability != nil {
       result = append(result, adapters.NewNewCapabilityAdapter(*cap.NewCapability))
   }
   ```

## Adding New API Format Provider

1. Create new directory: `providers/newapi/`

2. Implement LLM interface:
   ```go
   type Provider struct {
       *base.Provider
       adapters []adapters.CapabilityAdapter
   }
   
   func (p *Provider) Stream(...) (*CompletionResponse, error) {
       // Apply adapter preprocessing
       // Make API call
       // Apply adapter postprocessing
   }
   ```

3. Update `factory.go`:
   ```go
   case "newapi":
       return newapi.New(conn, options.Capabilities), nil
   ```

## Benefits of New Architecture

1. **Separation of Concerns**:
   - Providers handle API format
   - Adapters handle capabilities

2. **Code Reuse**:
   - Same adapters work across different providers
   - No duplication of capability logic

3. **Easy Extension**:
   - Add new capability = add one adapter
   - Add new API = add one provider

4. **Flexible Combinations**:
   - Any provider can use any adapter combination
   - Capabilities are composable

5. **Clear Responsibility**:
   - Each adapter handles exactly one capability dimension
   - Easy to test and maintain

## Testing Strategy

### Unit Tests (per adapter)
- Test preprocessing logic
- Test postprocessing logic
- Test stream chunk processing

### Integration Tests (per provider)
- Test with different adapter combinations
- Test full request/response flow
- Test error handling

### End-to-End Tests
- Test real API calls with different models
- Verify capability detection
- Verify adapter selection

## Migration Notes

### Old Architecture → New Architecture

**Before**:
```
reasoning.Provider  → Reasoning models (o1, R1)
openai.Provider     → Full-featured models (GPT-4o)
legacy.Provider     → Old models (GPT-3)
```

**After**:
```
openai.Provider + adapters → ALL models
```

The same OpenAI provider now handles all cases through different adapter combinations.

