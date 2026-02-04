# Anthropic Provider Implementation Proposal

## Overview

This proposal outlines the implementation plan for native Anthropic Claude API support in Yao Agent. Currently, all LLM connectors use `type: "openai"`, and Anthropic detection relies on URL pattern matching, which is unreliable and architecturally incorrect.

## Current Architecture

```
gou/connector/
├── openai/          # type: "openai" - handles all OpenAI-compatible APIs
├── moapi/           # type: "moapi"
├── redis/           # type: "redis"
└── ...

yao/agent/llm/
├── providers/
│   ├── factory.go   # SelectProvider() - selects provider based on connector type
│   ├── base/        # Base provider implementation
│   └── openai/      # OpenAI-compatible provider
```

**Current Flow:**
1. All LLM connectors declare `"type": "openai"`
2. `factory.go` uses `conn.Is(connector.OPENAI)` → always true for LLMs
3. `DetectAPIFormat()` guesses API format by URL patterns (unreliable)

## Problem Statement

1. **No type distinction**: Cannot differentiate Anthropic from OpenAI at connector level
2. **URL-based detection is fragile**: Relies on hardcoded patterns like `"anthropic.com"`
3. **API incompatibility**: Anthropic API uses different:
   - Endpoint: `/messages` vs `/chat/completions`
   - Auth header: `x-api-key` vs `Bearer` token
   - Request format: `system` as separate field, `max_tokens` required
   - Response format: Different structure

## Proposed Solution

### Phase 1: gou/connector - Add Anthropic Connector Type

**New files:**
```
gou/connector/
├── anthropic/
│   ├── anthropic.go    # Connector implementation
│   ├── types.go        # Options, Capabilities structs
│   └── defaults.go     # Default model capabilities
```

**connector/types.go changes:**
```go
const (
    // ... existing types
    ANTHROPIC = 7  // New connector type
)
```

**connector/anthropic/anthropic.go:**
```go
package anthropic

type Connector struct {
    id      string
    file    string
    Name    string  `json:"name"`
    Options Options `json:"options"`
}

type Options struct {
    Host         string        `json:"host,omitempty"`   // Default: https://api.anthropic.com
    Model        string        `json:"model,omitempty"`  // e.g., claude-sonnet-4-5
    Key          string        `json:"key"`              // API key
    Version      string        `json:"version,omitempty"` // API version, default: 2024-01-01
    Capabilities *Capabilities `json:"capabilities,omitempty"`
}

type Capabilities struct {
    Vision     interface{} `json:"vision,omitempty"`
    ToolCalls  bool        `json:"tool_calls,omitempty"`
    Streaming  bool        `json:"streaming,omitempty"`
    // ... same as openai.Capabilities
}
```

**DSL Example:**
```json
{
  "label": "Claude Sonnet 4.5",
  "type": "anthropic",
  "options": {
    "model": "claude-sonnet-4-5",
    "key": "$ENV.ANTHROPIC_API_KEY",
    "capabilities": {
      "vision": "claude",
      "tool_calls": true,
      "streaming": true
    }
  }
}
```

### Phase 2: yao/agent/llm - Add Anthropic Provider

**New files:**
```
yao/agent/llm/providers/
├── anthropic/
│   ├── anthropic.go    # Provider implementation
│   └── types.go        # Request/Response types
```

**anthropic/anthropic.go:**
```go
package anthropic

type Provider struct {
    *base.Provider
    adapters []adapters.CapabilityAdapter
}

func New(conn connector.Connector, capabilities *Capabilities) *Provider

func (p *Provider) Stream(ctx, messages, options, handler) (*CompletionResponse, error)
func (p *Provider) Post(ctx, messages, options) (*CompletionResponse, error)

// Internal methods
func (p *Provider) buildRequestBody(messages, options, streaming) (map[string]interface{}, error)
func (p *Provider) convertMessages(messages []context.Message) []map[string]interface{}
```

**Key Implementation Details:**

1. **Message Conversion** (OpenAI format → Anthropic format):
```go
// OpenAI format:
// {"role": "system", "content": "..."}
// {"role": "user", "content": "..."}

// Anthropic format:
// system: "..." (separate field)
// messages: [{"role": "user", "content": "..."}]
```

2. **Request Building:**
```go
body := map[string]interface{}{
    "model":      model,
    "max_tokens": maxTokens,  // Required in Anthropic
    "messages":   convertedMessages,
}
if systemPrompt != "" {
    body["system"] = systemPrompt
}
```

3. **HTTP Headers:**
```go
req.SetHeader("Content-Type", "application/json")
req.SetHeader("x-api-key", apiKey)           // Not Bearer token
req.SetHeader("anthropic-version", "2024-01-01")
```

4. **SSE Parsing** (different from OpenAI):
```go
// Anthropic SSE events:
// event: message_start
// event: content_block_start
// event: content_block_delta
// event: content_block_stop
// event: message_delta
// event: message_stop
```

**factory.go changes:**
```go
func SelectProvider(conn connector.Connector, options *CompletionOptions) (LLM, error) {
    // ...
    
    // Check connector type directly
    if conn.Is(connector.ANTHROPIC) {
        return anthropic.New(conn, options.Capabilities), nil
    }
    
    if conn.Is(connector.OPENAI) {
        return openai.New(conn, options.Capabilities), nil
    }
    
    // Default fallback
    return openai.New(conn, options.Capabilities), nil
}
```

## Implementation Effort

| Component | Files | Lines (est.) | Effort |
|-----------|-------|--------------|--------|
| gou/connector/anthropic | 3 | ~250 | 2-3 hours |
| yao/agent/llm/providers/anthropic | 2 | ~600 | 4-6 hours |
| Tests | 4 | ~400 | 2-3 hours |
| **Total** | **9** | **~1250** | **8-12 hours** |

## Migration Path

1. **Backward Compatible**: Existing `type: "openai"` connectors continue to work
2. **New Connectors**: Use `type: "anthropic"` for direct Anthropic API access
3. **Proxy Services**: OpenRouter, AWS Bedrock still use `type: "openai"` (they provide OpenAI-compatible endpoints)

## Testing Strategy

1. **Unit Tests**: Message conversion, request building
2. **Integration Tests**: Real API calls (with test API key)
3. **Connector Tests**: gou connector parsing and validation

## Alternative Considered

**URL-based detection in yao layer only** (current approach):
- Pros: No gou changes needed
- Cons: Fragile, architecturally incorrect, no connector-level validation

**Conclusion**: Rejected. Proper connector type is the cleaner solution.

## References

- [Anthropic API Documentation](https://docs.anthropic.com/en/api)
- [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go)
- [OpenAI Compatibility Guide](https://platform.claude.com/docs/en/api/openai-sdk)

## Next Steps

1. Review and approve this proposal
2. Implement gou/connector/anthropic (Phase 1)
3. Implement yao/agent/llm/providers/anthropic (Phase 2)
4. Update yao-init connectors to use `type: "anthropic"`
5. Write tests and documentation
