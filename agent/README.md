# Agent API Documentation

Agent is a chat/AI assistant API that provides endpoints for managing conversations, assistants, file uploads, and more.

## Base URL

All endpoints are relative to your base URL + `/api/__yao/agent`

Example: `http://localhost:5099/api/__yao/agent`

## Authentication

All endpoints require a `token` parameter for authentication. The token can be provided as:

- Query parameter: `?token=your_token_here`
- Authorization header: `Authorization: Bearer your_token_here`

## CORS Support

The API supports Cross-Origin Resource Sharing (CORS) and handles preflight OPTIONS requests.

## API Endpoints

### 1. Chat Endpoints

#### 1.1 Chat with AI

Start or continue a conversation with an AI assistant.

**Endpoints:**

- `GET /` - Chat via query parameters
- `POST /` - Chat via JSON body

**Parameters:**

- `content` (required) - The message content
- `chat_id` (optional) - Chat session ID. If not provided, a new one will be generated
- `context` (optional) - Additional context for the conversation
- `assistant_id` (optional) - Specific assistant to use
- `silent` (optional) - Silent mode: `true` or `1`
- `history_visible` (optional) - Show history: `true` or `1`
- `client_type` (optional) - Client type identifier

**Examples:**

```bash
# GET request
curl -X GET 'http://localhost:5099/api/__yao/agent?content=Hello&chat_id=chat_123&token=xxx'

# POST request
curl -X POST 'http://localhost:5099/api/__yao/agent' \
  -H 'Content-Type: application/json' \
  -d '{"content": "Hello", "chat_id": "chat_123", "token": "xxx"}'
```

**Response:**
Server-Sent Events (SSE) stream with chat messages.

#### 1.2 Chat History

Get conversation history for a specific chat.

**Endpoint:** `GET /history`

**Parameters:**

- `chat_id` (required) - Chat session ID

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/history?chat_id=chat_123&token=xxx'
```

**Response:**

```json
{
  "data": [
    {
      "role": "user",
      "content": "Hello",
      "timestamp": "2024-01-01T00:00:00Z"
    },
    {
      "role": "assistant",
      "content": "Hi there!",
      "timestamp": "2024-01-01T00:00:01Z"
    }
  ]
}
```

### 2. Chat Management

#### 2.1 List Chats

Get a paginated list of chat conversations.

**Endpoint:** `GET /chats`

**Parameters:**

- `page` (optional) - Page number (default: 1)
- `pagesize` (optional) - Items per page (default: 20)
- `keywords` (optional) - Search keywords
- `order` (optional) - Sort order (`asc` or `desc`)
- `locale` (optional) - Locale code (default: `en-us`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/chats?page=1&pagesize=20&keywords=search+term&order=desc&token=xxx'
```

**Response:**

```json
{
  "data": {
    "groups": [
      {
        "date": "2024-01-01",
        "chats": [
          {
            "chat_id": "chat_123",
            "title": "Chat Title",
            "updated_at": "2024-01-01T00:00:00Z"
          }
        ]
      }
    ],
    "total": 50,
    "page": 1,
    "pagesize": 20
  }
}
```

#### 2.2 Get Latest Chat

Get the most recent chat or create a new one if none exists.

**Endpoint:** `GET /chats/latest`

**Parameters:**

- `assistant_id` (optional) - Preferred assistant ID
- `locale` (optional) - Locale code (default: `en-us`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/chats/latest?assistant_id=assistant_123&token=xxx'
```

#### 2.3 Get Chat Details

Get detailed information about a specific chat.

**Endpoint:** `GET /chats/:id`

**Parameters:**

- `locale` (optional) - Locale code (default: `en-us`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/chats/chat_123?token=xxx'
```

#### 2.4 Update Chat

Update chat metadata (e.g., title).

**Endpoint:** `POST /chats/:id`

**Body:**

```json
{
  "title": "New Title",
  "content": "Chat content for title generation"
}
```

**Example:**

```bash
curl -X POST 'http://localhost:5099/api/__yao/agent/chats/chat_123' \
  -H 'Content-Type: application/json' \
  -d '{"title": "New Title", "content": "Chat content", "token": "xxx"}'
```

#### 2.5 Delete Chat

Delete a specific chat conversation.

**Endpoint:** `DELETE /chats/:id`

**Example:**

```bash
curl -X DELETE 'http://localhost:5099/api/__yao/agent/chats/chat_123?token=xxx'
```

### 3. Assistant Management

#### 3.1 List Assistants

Get a paginated list of available assistants.

**Endpoint:** `GET /assistants`

**Parameters:**

- `page` (optional) - Page number (default: 1)
- `pagesize` (optional) - Items per page (default: 20)
- `tags` (optional) - Comma-separated list of tags
- `keywords` (optional) - Search keywords
- `connector` (optional) - Connector name filter
- `select` (optional) - Comma-separated fields to select
- `built_in` (optional) - Filter built-in assistants (`true`/`false`/`1`/`0`)
- `mentionable` (optional) - Filter mentionable assistants (`true`/`false`/`1`/`0`)
- `automated` (optional) - Filter automated assistants (`true`/`false`/`1`/`0`)
- `assistant_id` (optional) - Specific assistant ID
- `locale` (optional) - Locale code (default: `en-us`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/assistants?page=1&pagesize=20&tags=tag1,tag2&token=xxx'
```

#### 3.2 Get Assistant Tags

Get all available assistant tags.

**Endpoint:** `GET /assistants/tags`

**Parameters:**

- `locale` (optional) - Locale code (default: `en-us`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/assistants/tags?token=xxx'
```

#### 3.3 Get Assistant Details

Get detailed information about a specific assistant.

**Endpoint:** `GET /assistants/:id`

**Parameters:**

- `locale` (optional) - Locale code (default: `en-us`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/assistants/assistant_123?token=xxx'
```

#### 3.4 Execute Assistant API

Call a specific assistant's API functionality.

**Endpoint:** `POST /assistants/:id/call`

**Body:**

```json
{
  "name": "Test",
  "payload": {
    "name": "yao",
    "age": 18
  }
}
```

**Example:**

```bash
curl -X POST 'http://localhost:5099/api/__yao/agent/assistants/assistant_123/call' \
  -H 'Content-Type: application/json' \
  -d '{"name": "Test", "payload": {"name": "yao", "age": 18}}'
```

#### 3.5 Create/Update Assistant

Create a new assistant or update an existing one.

**Endpoint:** `POST /assistants`

**Body:**

```json
{
  "name": "My Assistant",
  "type": "chat",
  "tags": ["tag1", "tag2"],
  "mentionable": true,
  "avatar": "path/to/avatar.png"
}
```

**Example:**

```bash
curl -X POST 'http://localhost:5099/api/__yao/agent/assistants' \
  -H 'Content-Type: application/json' \
  -d '{"name": "My Assistant", "type": "chat", "tags": ["tag1"], "token": "xxx"}'
```

#### 3.6 Delete Assistant

Delete a specific assistant.

**Endpoint:** `DELETE /assistants/:id`

**Example:**

```bash
curl -X DELETE 'http://localhost:5099/api/__yao/agent/assistants/assistant_123?token=xxx'
```

### 4. File Management

#### 4.1 Upload File

Upload files to different storage types.

**Endpoint:** `POST /upload/:storage`

**Storage Types:**

- `chat` - Chat-related files
- `knowledge` - Knowledge base files
- `assets` - General assets

**Form Data:**

- `file` (required) - The file to upload
- `chat_id` (required for chat storage) - Chat session ID
- `collection_id` (required for knowledge storage) - Knowledge collection ID
- `public` (optional) - Make file public
- `gzip` (optional) - Enable gzip compression

**Example:**

```bash
curl -X POST 'http://localhost:5099/api/__yao/agent/upload/chat?chat_id=chat_123&token=xxx' \
  -F 'file=@/path/to/file.txt'
```

**Response:**

```json
{
  "data": {
    "id": "file_123",
    "content_type": "text/plain",
    "bytes": 1024,
    "status": "uploaded"
  }
}
```

#### 4.2 Download File

Download a previously uploaded file.

**Endpoint:** `GET /download`

**Parameters:**

- `file_id` (required) - File ID to download
- `disposition` (optional) - Content disposition (default: `attachment`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/download?file_id=file_123&disposition=attachment&token=xxx' \
  -o downloaded_file.txt
```

### 5. Mentions

#### 5.1 Get Mentions

Get mentionable assistants for autocomplete.

**Endpoint:** `GET /mentions`

**Parameters:**

- `keywords` (optional) - Search keywords
- `locale` (optional) - Locale code (default: `en-us`)

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/mentions?keywords=assistant&token=xxx'
```

**Response:**

```json
{
  "data": [
    {
      "id": "assistant_123",
      "name": "Assistant Name",
      "type": "chat",
      "avatar": "avatar_url"
    }
  ]
}
```

### 6. Generation Endpoints

#### 6.1 Generate Title

Generate a title for chat content.

**Endpoints:**

- `GET /generate/title` - Generate via query parameters
- `POST /generate/title` - Generate via JSON body

**Parameters:**

- `content` (required) - Content to generate title for
- `chat_id` (optional) - Associated chat ID
- `context` (optional) - Additional context

**Examples:**

```bash
# GET request
curl -X GET 'http://localhost:5099/api/__yao/agent/generate/title?content=Chat+content&chat_id=chat_123&token=xxx'

# POST request
curl -X POST 'http://localhost:5099/api/__yao/agent/generate/title' \
  -H 'Content-Type: application/json' \
  -d '{"content": "Chat content", "chat_id": "chat_123", "token": "xxx"}'
```

**Response:** SSE stream with generated title

#### 6.2 Generate Prompts

Generate prompts based on content.

**Endpoints:**

- `GET /generate/prompts` - Generate via query parameters
- `POST /generate/prompts` - Generate via JSON body

**Parameters:**

- `content` (required) - Content to generate prompts for
- `chat_id` (optional) - Associated chat ID
- `context` (optional) - Additional context

**Examples:**

```bash
# GET request
curl -X GET 'http://localhost:5099/api/__yao/agent/generate/prompts?content=Generate+prompts&chat_id=chat_123&token=xxx'

# POST request
curl -X POST 'http://localhost:5099/api/__yao/agent/generate/prompts' \
  -H 'Content-Type: application/json' \
  -d '{"content": "Generate prompts", "chat_id": "chat_123", "token": "xxx"}'
```

**Response:** SSE stream with generated prompts

### 7. Utility Endpoints

#### 7.1 List Connectors

Get available AI connectors.

**Endpoint:** `GET /utility/connectors`

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/utility/connectors?token=xxx'
```

**Response:**

```json
{
  "data": [
    {
      "label": "OpenAI",
      "value": "openai"
    },
    {
      "label": "Custom API",
      "value": "custom_api"
    }
  ]
}
```

#### 7.2 Status Check

Check API service status.

**Endpoint:** `GET /status`

**Example:**

```bash
curl -X GET 'http://localhost:5099/api/__yao/agent/status?token=xxx'
```

**Response:** HTTP 200 status code

### 8. Dangerous Operations

#### 8.1 Clear All Chats

Delete all chat conversations for the authenticated user.

**Endpoint:** `DELETE /dangerous/clear_chats`

**Example:**

```bash
curl -X DELETE 'http://localhost:5099/api/__yao/agent/dangerous/clear_chats?token=xxx'
```

**Response:**

```json
{
  "message": "ok"
}
```

## Error Responses

All endpoints return JSON error responses in the following format:

```json
{
  "message": "Error description",
  "code": 400
}
```

Common error codes:

- `400` - Bad Request (missing or invalid parameters)
- `401` - Unauthorized (invalid or missing token)
- `403` - Forbidden (access denied)
- `404` - Not Found (resource not found)
- `500` - Internal Server Error

## Server-Sent Events (SSE)

Chat and generation endpoints return Server-Sent Events for real-time streaming:

**Headers:**

- `Content-Type: text/event-stream;charset=utf-8`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

**Event Format:**

```
data: {"type": "message", "content": "Hello"}

data: {"type": "done"}
```

## Rate Limiting

Rate limiting may be applied based on your authentication token and usage patterns.

## Support

For support and questions, please refer to the Yao App Engine documentation.
