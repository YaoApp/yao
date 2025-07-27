# Agent API

This document describes the RESTful API for AI agent interactions and chat completions in Yao applications.

## Base URL

All endpoints are prefixed with the configured base URL followed by `/agent` (e.g., `/v1/agent`).

## Authentication

All endpoints require OAuth authentication via the configured OAuth provider.

## Overview

The Agent API provides AI-powered chat completion capabilities with support for:

- **Server-Sent Events (SSE)** - Real-time streaming responses
- **Context Management** - Persistent chat sessions with history
- **Assistant Selection** - Multiple AI assistants with different capabilities
- **Flexible Parameters** - Configurable behavior for different use cases
- **Session Management** - Automatic session handling with user identification

## Endpoints

### Chat Completions

Create AI chat completions with streaming responses using Server-Sent Events.

```
GET /chat/completions?content={content}&chat_id={chat_id}&assistant_id={assistant_id}&context={context}&silent={silent}&history_visible={history_visible}&client_type={client_type}
POST /chat/completions
```

**Note:** This is a temporary implementation for full-process testing, and the interface may undergo significant global changes in the future.

**Query Parameters (GET) / Form Data (POST):**

- `content` (required): The user's message or question
- `chat_id` (optional): Chat session identifier (auto-generated if not provided)
- `assistant_id` (optional): Specific assistant to use (defaults to system default)
- `context` (optional): Additional context for the conversation
- `silent` (optional): Silent mode flag ("true"/"false" or "1"/"0")
- `history_visible` (optional): Whether chat history is visible ("true"/"false" or "1"/"0")
- `client_type` (optional): Client type identifier for customization

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response Headers:**

```
Content-Type: text/event-stream;charset=utf-8
Cache-Control: no-cache
Connection: keep-alive
```

**Example GET Request:**

```bash
curl -X GET "/v1/agent/chat/completions?content=Hello%2C%20how%20are%20you%3F&chat_id=chat_123&assistant_id=mohe" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

**Example POST Request:**

```bash
curl -X POST "/v1/agent/chat/completions" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: text/event-stream" \
  -d "content=Hello, how are you?&chat_id=chat_123&assistant_id=mohe"
```

**Response (Server-Sent Events):**

The response is streamed as Server-Sent Events with the following format:

```
data: {"type":"message","content":"Hello! I'm doing well, thank you for asking.","role":"assistant","timestamp":1640995200}

data: {"type":"done","chat_id":"chat_123","session_id":"session_456"}
```

**Response Data Types:**

- `message` - Content chunk from the AI assistant
- `error` - Error message if something goes wrong
- `done` - Indicates completion of the response

**Success Response Example:**

```json
{
  "type": "message",
  "content": "Hello! I'm an AI assistant created by Yao. How can I help you today?",
  "role": "assistant",
  "chat_id": "chat_123",
  "timestamp": 1640995200
}
```

**Completion Response:**

```json
{
  "type": "done",
  "chat_id": "chat_123",
  "session_id": "session_456"
}
```

## Parameters in Detail

### Content Parameter

The `content` parameter is the main user input:

- **Required** for all requests
- Can be a question, command, or conversation message
- Supports natural language input
- Maximum length depends on assistant configuration

### Chat ID Management

The `chat_id` parameter manages conversation continuity:

- **Auto-generated** if not provided (format: `chat_{timestamp}`)
- **Persistent** across multiple requests for the same conversation
- **Unique** identifier for each chat session
- Used for conversation history and context management

### Assistant Selection

The `assistant_id` parameter allows choosing specific AI assistants:

- **Optional** - defaults to system default assistant
- **Different assistants** may have different capabilities, knowledge bases, or personalities
- **Examples**: `mohe`, `developer`, `analyst`

### Context and Behavior

Additional parameters for fine-tuning behavior:

- `context` - Provides additional context for better responses
- `silent` - Controls verbose/quiet response modes
- `history_visible` - Controls whether conversation history affects responses
- `client_type` - Allows client-specific customizations

## Error Responses

All endpoints return standardized error responses via Server-Sent Events:

```
data: {"type":"error","error":"invalid_request","error_description":"content is required"}
```

**Common Error Types:**

- `invalid_request` - Missing required parameters
- `unauthorized` - Authentication failure
- `assistant_not_found` - Invalid assistant ID
- `internal_error` - Server processing error

**HTTP Status Codes:**

- `200` - Success (streaming response)
- `400` - Bad Request (missing content parameter)
- `401` - Unauthorized (authentication required)
- `500` - Internal Server Error

## Session Management

The Agent API automatically manages sessions:

### Session Creation

- **Automatic** session creation when `__sid` is not present
- **UUID generation** for new sessions
- **Session persistence** across requests

### Session Context

- **User identification** through session data
- **Conversation history** maintained per chat_id
- **Context preservation** between messages

## Real-Time Streaming

The API uses Server-Sent Events for real-time communication:

### Connection Management

- **Keep-alive** connections for streaming
- **Automatic reconnection** support
- **Graceful error handling**

### Event Types

- **message** - Streaming content chunks
- **error** - Error notifications
- **done** - Completion indicators

## Example Workflows

### Simple Chat Interaction

1. **Start a conversation:**

```bash
curl -X GET "/v1/agent/chat/completions?content=What%20is%20Yao?" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

2. **Continue the conversation:**

```bash
curl -X GET "/v1/agent/chat/completions?content=Tell%20me%20more%20about%20its%20features&chat_id=chat_123" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

### Assistant-Specific Interaction

1. **Use a specific assistant:**

```bash
curl -X POST "/v1/agent/chat/completions" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: text/event-stream" \
  -d "content=Help me debug this code&assistant_id=developer&chat_id=debug_session_456"
```

### Context-Aware Conversation

1. **Provide additional context:**

```bash
curl -X GET "/v1/agent/chat/completions?content=Optimize%20this%20query&context=PostgreSQL%20database%20with%20large%20user%20table&assistant_id=analyst" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

### Silent Mode Operation

1. **Use silent mode for concise responses:**

```bash
curl -X GET "/v1/agent/chat/completions?content=Generate%20user%20model&silent=true&client_type=api" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: text/event-stream"
```

## JavaScript Client Example

Here's how to consume the streaming API in JavaScript:

```javascript
const eventSource = new EventSource(
  "/v1/agent/chat/completions?content=Hello&chat_id=chat_123",
  {
    headers: {
      Authorization: "Bearer " + accessToken,
    },
  }
);

eventSource.onmessage = function (event) {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case "message":
      console.log("Assistant:", data.content);
      break;
    case "error":
      console.error("Error:", data.error_description);
      break;
    case "done":
      console.log("Conversation completed");
      eventSource.close();
      break;
  }
};

eventSource.onerror = function (error) {
  console.error("Connection error:", error);
};
```

## Integration Considerations

### Performance

- **Streaming responses** reduce perceived latency
- **Connection pooling** for multiple concurrent chats
- **Automatic session cleanup** prevents memory leaks

### Security

- **OAuth 2.1 authentication** required for all requests
- **Session-based access control**
- **Input validation** and sanitization
- **Rate limiting** (configured at server level)

### Scalability

- **Stateless design** (session data in external store)
- **Load balancer compatible** (sticky sessions not required)
- **Horizontal scaling** support

## Development Notes

**Important:** This is a temporary implementation for full-process testing. The interface design and functionality may undergo significant global changes in future versions. Consider this API experimental and subject to breaking changes.

### Current Limitations

- Limited error recovery mechanisms
- Basic assistant selection logic
- Simplified context management
- Minimal response formatting options

### Future Enhancements

Future versions may include:

- Enhanced context management
- Advanced assistant capabilities
- Improved error handling
- Extended parameter options
- WebSocket support as alternative to SSE

This Agent API provides a foundation for AI-powered interactions in Yao applications with real-time streaming capabilities and flexible configuration options.
