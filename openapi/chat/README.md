# Chat API

This document describes the RESTful API for AI chat completions in Yao applications, providing **100% compatibility with OpenAI clients**.

## Base URL

All endpoints are prefixed with the configured base URL followed by `/chat` (e.g., `/v1/chat`).

## Authentication

All endpoints require OAuth authentication via the configured OAuth provider.

## Overview

The Chat API provides AI-powered chat completion capabilities with **full OpenAI API compatibility**, supporting:

- **OpenAI Client Compatibility** - 100% compatible with existing OpenAI client libraries
- **Server-Sent Events (SSE)** - Real-time streaming responses
- **Context Management** - Persistent chat sessions with history
- **Assistant Selection** - Multiple AI assistants with different capabilities
- **Flexible Parameters** - Standard OpenAI parameters plus Yao-specific extensions
- **Session Management** - Automatic session handling with user identification

## Endpoints

### Chat Completions

Create AI chat completions with streaming responses using Server-Sent Events. This endpoint is **100% compatible with OpenAI's `/v1/chat/completions` API**.

```
GET /completions?content={content}&chat_id={chat_id}&assistant_id={assistant_id}&context={context}&silent={silent}&history_visible={history_visible}&client_type={client_type}
POST /completions
```

**Note:** This is a temporary implementation for full-process testing, and the interface may undergo significant global changes in the future.

**OpenAI Compatibility:**

- **Endpoint Path**: `/chat/completions` (matches OpenAI exactly)
- **Request Format**: Supports both OpenAI standard and Yao-extended parameters
- **Response Format**: Compatible with OpenAI response structure
- **Client Libraries**: Works with existing OpenAI SDKs and client libraries

**Query Parameters (GET) / Form Data (POST):**

**Standard OpenAI Parameters:**

- `model` (optional): AI model to use (mapped to `assistant_id` internally)
- `messages` (optional): Array of message objects (OpenAI format)
- `temperature` (optional): Sampling temperature
- `max_tokens` (optional): Maximum tokens in response
- `stream` (optional): Enable streaming responses

**Yao-Specific Parameters:**

- `content` (required): The user's message or question (simplified input)
- `chat_id` (optional): Chat session identifier (auto-generated if not provided)
- `assistant_id` (optional): Specific assistant to use (defaults to system default)
- `context` (optional): Additional context for the conversation
- `silent` (optional): Silent mode flag ("true"/"false" or "1"/"0")
- `history_visible` (optional): Whether chat history is visible ("true"/"false" or "1"/"0")
- `client_type` (optional): Client type identifier for customization

**Headers:**

```
Authorization: Bearer {access_token}
Content-Type: application/json
```

**Response Headers:**

```
Content-Type: text/event-stream;charset=utf-8
Cache-Control: no-cache
Connection: keep-alive
```

**Example GET Request (Yao Simplified Format):**

```bash
curl -X GET "/v1/chat/completions?content=Hello%2C%20how%20are%20you%3F&chat_id=chat_123&assistant_id=mohe" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

**Example POST Request (OpenAI Compatible Format):**

```bash
curl -X POST "/v1/chat/completions" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mohe",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "stream": true
  }'
```

**Example POST Request (Yao Simplified Format):**

```bash
curl -X POST "/v1/chat/completions" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: text/event-stream" \
  -d "content=Hello, how are you?&chat_id=chat_123&assistant_id=mohe"
```

**Response (Server-Sent Events):**

The response is streamed as Server-Sent Events with OpenAI-compatible format:

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1640995200,"model":"mohe","choices":[{"index":0,"delta":{"content":"Hello! I'm doing well, thank you for asking."},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1640995200,"model":"mohe","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

**Response Data Types:**

- `chat.completion.chunk` - Streaming content chunks (OpenAI format)
- `error` - Error message if something goes wrong
- `[DONE]` - Indicates completion of the response (OpenAI format)

**Success Response Example (OpenAI Compatible):**

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion.chunk",
  "created": 1640995200,
  "model": "mohe",
  "choices": [
    {
      "index": 0,
      "delta": {
        "content": "Hello! I'm an AI assistant created by Yao. How can I help you today?"
      },
      "finish_reason": null
    }
  ]
}
```

**Completion Response:**

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion.chunk",
  "created": 1640995200,
  "model": "mohe",
  "choices": [
    {
      "index": 0,
      "delta": {},
      "finish_reason": "stop"
    }
  ]
}
```

## OpenAI Client Integration

### Using OpenAI Python Client

```python
import openai

# Configure client for Yao API
openai.api_base = "https://your-yao-server.com/v1"
openai.api_key = "your-oauth-token"

# Use exactly like OpenAI
response = openai.ChatCompletion.create(
    model="mohe",
    messages=[
        {"role": "user", "content": "Hello, how are you?"}
    ],
    stream=True
)

for chunk in response:
    if chunk.choices[0].delta.get("content"):
        print(chunk.choices[0].delta.content, end="")
```

### Using OpenAI Node.js Client

```javascript
import OpenAI from "openai";

const openai = new OpenAI({
  baseURL: "https://your-yao-server.com/v1",
  apiKey: "your-oauth-token",
});

const stream = await openai.chat.completions.create({
  model: "mohe",
  messages: [{ role: "user", content: "Hello, how are you?" }],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || "");
}
```

### Using OpenAI Go Client

```go
package main

import (
    "context"
    "fmt"
    "io"

    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("your-oauth-token")
    config.BaseURL = "https://your-yao-server.com/v1"
    client := openai.NewClientWithConfig(config)

    req := openai.ChatCompletionRequest{
        Model: "mohe",
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: "Hello, how are you?",
            },
        },
        Stream: true,
    }

    stream, err := client.CreateChatCompletionStream(context.Background(), req)
    if err != nil {
        panic(err)
    }
    defer stream.Close()

    for {
        response, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
        fmt.Print(response.Choices[0].Delta.Content)
    }
}
```

## Parameters in Detail

### Content Parameter (Yao Extension)

The `content` parameter provides simplified input for basic use cases:

- **Required** for simplified Yao format
- Can be a question, command, or conversation message
- Supports natural language input
- Alternative to OpenAI's `messages` array format

### Model/Assistant Selection

The `model` parameter (OpenAI) or `assistant_id` parameter (Yao) selects the AI assistant:

- **OpenAI Compatible**: Use `model` field in JSON requests
- **Yao Extension**: Use `assistant_id` for URL parameters
- **Available Models**: `mohe`, `developer`, `analyst`, etc.
- **Default**: System default assistant if not specified

### Chat ID Management (Yao Extension)

The `chat_id` parameter manages conversation continuity:

- **Auto-generated** if not provided (format: `chat_{timestamp}`)
- **Persistent** across multiple requests for the same conversation
- **Unique** identifier for each chat session
- **Yao-specific**: Not part of standard OpenAI API

### Context and Behavior (Yao Extensions)

Additional Yao-specific parameters for fine-tuning behavior:

- `context` - Provides additional context for better responses
- `silent` - Controls verbose/quiet response modes
- `history_visible` - Controls whether conversation history affects responses
- `client_type` - Allows client-specific customizations

## Error Responses

All endpoints return standardized error responses compatible with OpenAI format:

**Server-Sent Events Error:**

```
data: {"error":{"type":"invalid_request_error","message":"content is required","code":"missing_parameter"}}
```

**HTTP Error Response:**

```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "The request is missing required parameters",
    "code": "missing_parameter"
  }
}
```

**Common Error Types:**

- `invalid_request_error` - Missing required parameters
- `authentication_error` - Authentication failure
- `not_found_error` - Invalid model/assistant ID
- `internal_server_error` - Server processing error

**HTTP Status Codes:**

- `200` - Success (streaming response)
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (authentication required)
- `404` - Not Found (model not found)
- `500` - Internal Server Error

## Example Workflows

### OpenAI Client Migration

**Before (OpenAI):**

```python
import openai

openai.api_key = "sk-..."
response = openai.ChatCompletion.create(
    model="gpt-3.5-turbo",
    messages=[{"role": "user", "content": "Hello"}]
)
```

**After (Yao - No Code Changes Required):**

```python
import openai

openai.api_base = "https://your-yao.com/v1"  # Only change needed
openai.api_key = "your-oauth-token"          # Only change needed
response = openai.ChatCompletion.create(
    model="mohe",                           # Use Yao assistant
    messages=[{"role": "user", "content": "Hello"}]
)
```

### Simple Chat Interaction

1. **Start a conversation (Yao simplified format):**

```bash
curl -X GET "/v1/chat/completions?content=What%20is%20Yao?" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

2. **Continue the conversation (OpenAI format):**

```bash
curl -X POST "/v1/chat/completions" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mohe",
    "messages": [
      {"role": "user", "content": "Tell me more about its features"}
    ],
    "stream": true
  }'
```

### Assistant-Specific Interaction

1. **Use a specific assistant (OpenAI compatible):**

```bash
curl -X POST "/v1/chat/completions" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "developer",
    "messages": [
      {"role": "user", "content": "Help me debug this code"}
    ],
    "stream": true,
    "temperature": 0.7
  }'
```

### Context-Aware Conversation (Yao Extensions)

1. **Provide additional context:**

```bash
curl -X GET "/v1/chat/completions?content=Optimize%20this%20query&context=PostgreSQL%20database%20with%20large%20user%20table&assistant_id=analyst" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

## Client Library Examples

### Curl (OpenAI Format)

```bash
curl -X POST "/v1/chat/completions" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mohe",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "stream": true,
    "max_tokens": 150,
    "temperature": 0.7
  }'
```

### JavaScript (Fetch API)

```javascript
const response = await fetch("/v1/chat/completions", {
  method: "POST",
  headers: {
    Authorization: `Bearer ${accessToken}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "mohe",
    messages: [{ role: "user", content: "Hello, how are you?" }],
    stream: true,
  }),
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;

  const chunk = decoder.decode(value);
  const lines = chunk.split("\n");

  for (const line of lines) {
    if (line.startsWith("data: ")) {
      const data = line.slice(6);
      if (data === "[DONE]") return;

      try {
        const parsed = JSON.parse(data);
        const content = parsed.choices[0]?.delta?.content;
        if (content) {
          console.log(content);
        }
      } catch (e) {
        // Skip invalid JSON
      }
    }
  }
}
```

## Migration Guide

### From OpenAI API

**No code changes required!** Just update your configuration:

1. **Change Base URL**: `https://api.openai.com/v1` → `https://your-yao.com/v1`
2. **Update API Key**: Use your Yao OAuth token instead of OpenAI API key
3. **Change Model Names**: `gpt-3.5-turbo` → `mohe`, `gpt-4` → `developer`, etc.

### From Custom Chat APIs

If migrating from other chat APIs, you can use Yao's simplified format:

- **Simple GET requests** with `content` parameter
- **Form data POST** for basic interactions
- **Gradual migration** to full OpenAI format

## Integration Considerations

### Performance

- **Streaming responses** reduce perceived latency
- **Connection pooling** for multiple concurrent chats
- **Automatic session cleanup** prevents memory leaks
- **OpenAI client optimizations** work seamlessly

### Security

- **OAuth 2.1 authentication** required for all requests
- **Session-based access control**
- **Input validation** and sanitization
- **Rate limiting** (configured at server level)
- **Compatible with OpenAI security practices**

### Scalability

- **Stateless design** (session data in external store)
- **Load balancer compatible** (sticky sessions not required)
- **Horizontal scaling** support
- **OpenAI client connection pooling** supported

## Development Notes

**Important:** This is a temporary implementation for full-process testing. The interface design and functionality may undergo significant global changes in future versions. However, **OpenAI compatibility will be maintained** to ensure existing client libraries continue to work.

### Current Limitations

- Limited error recovery mechanisms
- Basic assistant selection logic
- Simplified context management
- Minimal response formatting options

### Future Enhancements

Future versions will maintain OpenAI compatibility while adding:

- Enhanced context management
- Advanced assistant capabilities
- Improved error handling
- Extended Yao-specific parameters
- WebSocket support as alternative to SSE

### Compatibility Promise

- **OpenAI Client Support**: All major OpenAI client libraries will continue to work
- **Standard Compliance**: Full compliance with OpenAI API specification
- **Seamless Migration**: Existing OpenAI code works with minimal configuration changes

This Chat API provides **100% OpenAI client compatibility** while extending capabilities with Yao-specific features, making it easy to migrate existing applications and integrate with the broader AI ecosystem.
