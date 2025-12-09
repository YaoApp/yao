# Chat Storage Design

This document describes the design for storing chat conversations, messages, and execution steps in the YAO Agent system.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Data Models](#data-models)
- [Write Strategy](#write-strategy)
- [API Interface](#api-interface)
- [Usage Examples](#usage-examples)
- [Related Documents](#related-documents)

## Overview

The chat storage system is designed to:

1. **Store user-visible messages** - All messages sent via `ctx.Send()`, including text, images, loading states, etc.
2. **Support resume/retry** - Track execution steps to enable recovery from interruptions or failures
3. **Efficient writes** - Batch message writes at request end

### Design Goals

| Goal                     | Solution                                         |
| ------------------------ | ------------------------------------------------ |
| Complete chat history    | Store final content of all `ctx.Send()` messages |
| Resume from interruption | Track step status and input/output               |
| Retry failed operations  | Store step input for re-execution                |
| Minimize database writes | Batch writes at request end                      |

### Non-Goals

- **Tracing/debugging** - Handled by separate [Trace module](../../trace/README.md)
- **Streaming replay** - Not needed, history shows final content only
- **Request tracking/billing** - Handled by [OpenAPI Request module](../../openapi/request/REQUEST_DESIGN.md)

### Relationship with OpenAPI Request

The Agent storage focuses on **chat content and execution state**, while request tracking (billing, rate limiting, auditing) is handled globally by the OpenAPI layer:

| Concern          | Module            | Table             |
| ---------------- | ----------------- | ----------------- |
| Request tracking | `openapi/request` | `openapi_request` |
| Billing (tokens) | `openapi/request` | `openapi_request` |
| Rate limiting    | `openapi/request` | -                 |
| Chat sessions    | `agent/store`     | `agent_chat`      |
| Chat messages    | `agent/store`     | `agent_message`   |
| Resume/Retry     | `agent/store`     | `agent_resume`    |

The `request_id` from OpenAPI middleware is passed to Agent and stored in messages/steps for correlation.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Chat Storage                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐                                        │
│  │      Chat       │  Metadata: title, assistant, user      │
│  └────────┬────────┘                                        │
│           │                                                  │
│           │ 1:N                                              │
│           ▼                                                  │
│  ┌─────────────────┐                                        │
│  │    Message      │  User-visible: type, props, role       │
│  └────────┬────────┘                                        │
│           │                                                  │
│           │ N:N (via request_id)                            │
│           ▼                                                  │
│  ┌─────────────────┐                                        │
│  │     Resume      │  Recovery: type, status, input/output  │
│  │  (only on fail) │  Only saved when interrupted/failed    │
│  └─────────────────┘                                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Data Models

### 1. Chat Table

Stores chat metadata and session information.

**Table Name:** `agent_chat`

| Column            | Type        | Nullable | Index  | Description                      |
| ----------------- | ----------- | -------- | ------ | -------------------------------- |
| `id`              | ID          | No       | PK     | Auto-increment primary key       |
| `chat_id`         | string(64)  | No       | Unique | Unique chat identifier           |
| `title`           | string(500) | Yes      | -      | Chat title                       |
| `assistant_id`    | string(200) | No       | Yes    | Associated assistant ID          |
| `mode`            | string(50)  | No       | -      | Chat mode (default: "chat")      |
| `status`          | enum        | No       | Yes    | Status: `active`, `archived`     |
| `preset`          | boolean     | No       | -      | Whether this is a preset chat    |
| `public`          | boolean     | No       | -      | Whether shared across all teams  |
| `share`           | enum        | No       | Yes    | Sharing scope: `private`, `team` |
| `sort`            | integer     | No       | -      | Sort order for display           |
| `last_message_at` | timestamp   | Yes      | Yes    | Timestamp of last message        |
| `metadata`        | json        | Yes      | -      | Additional metadata              |
| `created_at`      | timestamp   | No       | Yes    | Creation timestamp               |
| `updated_at`      | timestamp   | No       | -      | Last update timestamp            |

**Model Options:**

```json
{
  "option": {
    "soft_deletes": true,
    "permission": true,
    "timestamps": true
  }
}
```

**Note:** `permission: true` enables Yao's built-in permission management, which automatically adds the following fields:

| Field              | Type        | Description                    |
| ------------------ | ----------- | ------------------------------ |
| `__yao_created_by` | string(200) | User ID who created the record |
| `__yao_updated_by` | string(200) | User ID who last updated       |
| `__yao_team_id`    | string(200) | Team ID for team-level access  |
| `__yao_tenant_id`  | string(200) | Tenant ID for multi-tenancy    |

These fields are automatically managed by the framework and used for access control filtering.

**Indexes:**

| Name                 | Columns           | Type  |
| -------------------- | ----------------- | ----- |
| `idx_chat_assistant` | `assistant_id`    | index |
| `idx_chat_status`    | `status`          | index |
| `idx_chat_share`     | `share`           | index |
| `idx_chat_last_msg`  | `last_message_at` | index |

### 2. Message Table

Stores user-visible messages (both user input and assistant responses).

**Table Name:** `agent_message`

| Column         | Type        | Nullable | Index  | Description                               |
| -------------- | ----------- | -------- | ------ | ----------------------------------------- |
| `id`           | ID          | No       | PK     | Auto-increment primary key                |
| `message_id`   | string(64)  | No       | Unique | Unique message identifier                 |
| `chat_id`      | string(64)  | No       | Yes    | Parent chat ID                            |
| `request_id`   | string(64)  | Yes      | Yes    | Request ID for grouping                   |
| `role`         | enum        | No       | Yes    | Role: `user`, `assistant`                 |
| `type`         | string(50)  | No       | -      | Message type (text, image, loading, etc.) |
| `props`        | json        | No       | -      | Message properties (content, url, etc.)   |
| `block_id`     | string(64)  | Yes      | Yes    | Block grouping ID                         |
| `thread_id`    | string(64)  | Yes      | Yes    | Thread grouping ID                        |
| `assistant_id` | string(200) | Yes      | Yes    | Assistant ID (join to get name/avatar)    |
| `sequence`     | integer     | No       | Yes    | Message order within chat                 |
| `metadata`     | json        | Yes      | -      | Additional metadata                       |
| `created_at`   | timestamp   | No       | Yes    | Creation timestamp                        |
| `updated_at`   | timestamp   | No       | -      | Last update timestamp                     |

**Indexes:**

| Name                | Columns               | Type  |
| ------------------- | --------------------- | ----- |
| `idx_msg_chat_seq`  | `chat_id`, `sequence` | index |
| `idx_msg_request`   | `request_id`          | index |
| `idx_msg_block`     | `block_id`            | index |
| `idx_msg_assistant` | `assistant_id`        | index |

**Message Types (Built-in):**

All built-in types defined in `agent/output/BUILTIN_TYPES.md` are stored. See that document for complete Props structures.

| Type         | Description                      | Props Example                                                                               | Stored?     |
| ------------ | -------------------------------- | ------------------------------------------------------------------------------------------- | ----------- |
| `user_input` | User input (frontend display)    | `{"content": "Hello", "role": "user", "name": "John"}`                                      | ✅ Yes      |
| `text`       | Text/Markdown content            | `{"content": "Hello **world**!"}`                                                           | ✅ Yes      |
| `thinking`   | Reasoning process (o1, DeepSeek) | `{"content": "Let me analyze..."}`                                                          | ✅ Yes      |
| `loading`    | Loading/processing indicator     | `{"message": "Searching knowledge base..."}`                                                | ✅ Yes      |
| `tool_call`  | LLM tool/function call           | `{"id": "call_abc123", "name": "get_weather", "arguments": "{\"location\":\"SF\"}"}`        | ✅ Yes      |
| `retrieval`  | KB/Web search results            | `{"query": "...", "sources": [...], "total_results": 10}`                                   | ✅ Yes      |
| `error`      | Error message                    | `{"message": "Connection timeout", "code": "TIMEOUT", "details": "..."}`                    | ✅ Yes      |
| `image`      | Image content                    | `{"url": "...", "alt": "...", "width": 200, "height": 200, "detail": "auto"}`               | ✅ Yes      |
| `audio`      | Audio content                    | `{"url": "...", "format": "mp3", "duration": 120.5, "transcript": "...", "controls": true}` | ✅ Yes      |
| `video`      | Video content                    | `{"url": "...", "format": "mp4", "thumbnail": "...", "width": 640, "height": 360}`          | ✅ Yes      |
| `action`     | System action (CUI only)         | `{"name": "open_panel", "payload": {"panel_id": "user_profile"}}`                           | ✅ Yes      |
| `event`      | Lifecycle event (CUI only)       | `{"event": "stream_start", "message": "...", "data": {...}}`                                | ⚠️ Optional |

**Note on `event` type:** Lifecycle events (`stream_start`, `stream_end`, etc.) are typically transient and may not need persistent storage. Consider storing only significant events or skipping entirely based on use case.

**Tool Call Storage:**

Tool calls from LLM responses are stored as `tool_call` type messages. The raw tool call data is preserved in `props`:

```json
{
  "message_id": "msg_001",
  "chat_id": "chat_123",
  "role": "assistant",
  "type": "tool_call",
  "props": {
    "id": "call_abc123",
    "name": "get_weather",
    "arguments": "{\"location\": \"San Francisco\", \"unit\": \"celsius\"}"
  },
  "block_id": "B1",
  "sequence": 5
}
```

**Tool Result Storage:**

Tool execution results can be stored as `text` type with metadata indicating it's a tool result:

```json
{
  "message_id": "msg_002",
  "chat_id": "chat_123",
  "role": "assistant",
  "type": "text",
  "props": {
    "content": "The weather in San Francisco is 18°C and sunny."
  },
  "metadata": {
    "tool_call_id": "call_abc123",
    "tool_name": "get_weather",
    "is_tool_result": true
  },
  "block_id": "B1",
  "sequence": 6
}
```

**Custom Types:**

Any type not in the built-in list is considered a custom type and stored with its original structure:

```json
{
  "type": "chart",
  "props": {
    "chartType": "bar",
    "data": [...],
    "options": {...}
  }
}
```

**Multimodal User Input:**

User input with multimodal content (text + images + files) is stored as `user_input` type:

```json
{
  "message_id": "msg_000",
  "chat_id": "chat_123",
  "role": "user",
  "type": "user_input",
  "props": {
    "content": [
      { "type": "text", "text": "What's in this image?" },
      {
        "type": "image_url",
        "image_url": {
          "url": "https://example.com/photo.jpg",
          "detail": "high"
        }
      }
    ],
    "role": "user",
    "name": "John"
  },
  "sequence": 1
}
```

### Knowledge Base & Web Search Results

Retrieval results from knowledge bases and web searches need to be stored for:

1. **User Feedback** - Users can rate (👍/👎) individual sources
2. **Quality Analytics** - Track which documents/sources are most useful
3. **Source Attribution** - Display citations in the UI
4. **RAG Optimization** - Improve retrieval based on feedback

**Storage Approach:** Store retrieval results as a special message type `retrieval` with structured props.

**Retrieval Message Structure:**

```json
{
  "message_id": "msg_retrieval_001",
  "chat_id": "chat_123",
  "request_id": "req_abc",
  "role": "assistant",
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
      },
      {
        "id": "src_002",
        "type": "kb",
        "collection_id": "col_docs",
        "document_id": "doc_124",
        "chunk_id": "chunk_789",
        "title": "Advanced Model Options",
        "content": "Models support various options including soft_deletes...",
        "score": 0.87,
        "metadata": {
          "file_path": "/docs/advanced.md",
          "page": 12
        }
      },
      {
        "id": "src_003",
        "type": "web",
        "url": "https://yaoapps.com/docs/models",
        "title": "Yao Models Documentation",
        "content": "Official documentation for Yao model system...",
        "score": 0.85,
        "metadata": {
          "domain": "yaoapps.com",
          "fetched_at": "2024-01-15T10:30:00Z"
        }
      }
    ],
    "total_results": 15,
    "query_time_ms": 120
  },
  "block_id": "B1",
  "assistant_id": "docs_assistant",
  "sequence": 2
}
```

**Source Types:**

| Type   | Description             | Key Fields                                 |
| ------ | ----------------------- | ------------------------------------------ |
| `kb`   | Knowledge base document | `collection_id`, `document_id`, `chunk_id` |
| `web`  | Web search result       | `url`, `domain`                            |
| `file` | Uploaded file           | `file_id`, `file_path`                     |
| `api`  | External API result     | `api_name`, `endpoint`                     |
| `mcp`  | MCP tool result         | `server`, `tool`                           |

**Source Feedback:**

User feedback on retrieval sources is handled by the Knowledge Base module. See [KB Feedback](../../kb/README.md) for details.

**Example: KB Search in Create Hook:**

```typescript
// In Create hook, search knowledge base and store results
const results = await ctx.kb.search("col_docs", query, { limit: 5 });

// Send retrieval message (stored automatically)
ctx.Send({
  type: "retrieval",
  props: {
    query: query,
    sources: results.documents.map((doc, idx) => ({
      id: `src_${idx}`,
      type: "kb",
      collection_id: "col_docs",
      document_id: doc.document.metadata.document_id,
      chunk_id: doc.document.id,
      title: doc.document.metadata.title || "Untitled",
      content: doc.document.content,
      score: doc.score,
      metadata: doc.document.metadata,
    })),
    total_results: results.total,
    query_time_ms: results.query_time_ms,
  },
});

// Also send loading message for user feedback
ctx.Send({
  type: "loading",
  props: { message: `Found ${results.total} relevant documents...` },
});
```

**Example: Web Search Results:**

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
          "published_at": "2024-01-10",
          "fetched_at": "2024-01-15T10:30:00Z",
          "snippet": "The year 2024 has seen remarkable..."
        }
      }
    ],
    "provider": "tavily",
    "total_results": 10,
    "query_time_ms": 850
  }
}
```

### 3. Resume Table

Stores execution state for resume/retry functionality. **Only written when request is interrupted or failed.**

**Table Name:** `agent_resume`

| Column            | Type        | Nullable | Index  | Description                      |
| ----------------- | ----------- | -------- | ------ | -------------------------------- |
| `id`              | ID          | No       | PK     | Auto-increment primary key       |
| `resume_id`       | string(64)  | No       | Unique | Unique resume record identifier  |
| `chat_id`         | string(64)  | No       | Yes    | Parent chat ID                   |
| `request_id`      | string(64)  | No       | Yes    | Request ID                       |
| `assistant_id`    | string(200) | No       | Yes    | Assistant executing this step    |
| `stack_id`        | string(64)  | No       | Yes    | Stack node ID for this execution |
| `stack_parent_id` | string(64)  | Yes      | Yes    | Parent stack ID (for A2A calls)  |
| `stack_depth`     | integer     | No       | -      | Call depth (0=root, 1+=nested)   |
| `type`            | enum        | No       | Yes    | Step type                        |
| `status`          | enum        | No       | Yes    | Status: `interrupted`, `failed`  |
| `input`           | json        | Yes      | -      | Step input data                  |
| `output`          | json        | Yes      | -      | Step output data (partial)       |
| `space_snapshot`  | json        | Yes      | -      | Space data snapshot for recovery |
| `error`           | text        | Yes      | -      | Error message if failed          |
| `sequence`        | integer     | No       | Yes    | Step order within request        |
| `metadata`        | json        | Yes      | -      | Additional metadata              |
| `created_at`      | timestamp   | No       | Yes    | Creation timestamp               |
| `updated_at`      | timestamp   | No       | -      | Last update timestamp            |

**Space Snapshot:**

The `space_snapshot` field stores the shared data space (`ctx.Space`) at each step for recovery purposes.

```typescript
// Example: In Next hook, set data to Space before delegate
ctx.space.Set("choose_prompt", "query");
return {
  delegate: { agent_id: "expense", messages: payload.messages },
};
```

If interrupted during delegate, the `space_snapshot` allows restoring `ctx.Space` state:

```json
{
  "choose_prompt": "query",
  "user_preferences": { "currency": "USD" }
}
```

**Resume Step Types:**

| Type          | Description           | Input                  | Output                                |
| ------------- | --------------------- | ---------------------- | ------------------------------------- |
| `input`       | User input received   | `{messages: [...]}`    | -                                     |
| `hook_create` | Create hook execution | `{messages: [...]}`    | `{messages: [...], ...}`              |
| `llm`         | LLM completion call   | `{messages: [...]}`    | `{content: "...", tool_calls: [...]}` |
| `tool`        | Tool/MCP execution    | `{server, tool, args}` | `{result: ...}`                       |
| `hook_next`   | Next hook execution   | `{completion, tools}`  | `{data: ...}`                         |
| `delegate`    | A2A delegation        | `{agent_id, messages}` | `{response: ...}`                     |

**Resume Status (only two values - table only stores failed/interrupted):**

| Status        | Description       | Action   |
| ------------- | ----------------- | -------- |
| `failed`      | Failed with error | Retry    |
| `interrupted` | User interrupted  | Continue |

**Indexes:**

| Name                   | Columns                  | Type  |
| ---------------------- | ------------------------ | ----- |
| `idx_resume_chat`      | `chat_id`                | index |
| `idx_resume_request`   | `request_id`, `sequence` | index |
| `idx_resume_status`    | `status`                 | index |
| `idx_resume_stack`     | `stack_id`               | index |
| `idx_resume_parent`    | `stack_parent_id`        | index |
| `idx_resume_assistant` | `assistant_id`           | index |

## Write Strategy

### Two-Write Strategy

All data is buffered in memory during execution and written to database only **twice**:

1. **Write 1 (Entry)**: When `Stream()` starts - save user input message
2. **Write 2 (Exit)**: When `Stream()` exits - batch save messages (and steps only on error/interrupt)

**Note**: Request tracking (status, tokens, duration) is handled by [OpenAPI Request Middleware](../../openapi/request/REQUEST_DESIGN.md).

```
Stream() Entry
    │
    ├── 【Write 1】Save user input
    │   - User message (role=user)
    │
    ├── Execution (all in memory)
    │   - ctx.Send()    → messageBuffer
    │   - ctx.Append()  → update messageBuffer
    │   - ctx.Replace() → update messageBuffer
    │   - Each step     → stepBuffer
    │
    └── 【Write 2】Save final state (via defer)
        │
        ├── Always:
        │   - Batch write all assistant messages
        │   - Update token usage in openapi_request (via request_id)
        │
        └── Only on error/interrupt:
            - Batch write all steps (for resume/retry)
```

### Write Points

| Event            | Message Table        | Step Table                          | Token Usage |
| ---------------- | -------------------- | ----------------------------------- | ----------- |
| Stream entry     | Write 1 (user input) | -                                   | -           |
| During execution | Buffer in memory     | Buffer in memory                    | -           |
| **Completed**    | **Batch write all**  | **❌ Skip (no need to resume)**     | ✅ Update   |
| On interrupt     | Batch write buffered | ✅ Batch write (status=interrupted) | ✅ Update   |
| On error         | Batch write buffered | ✅ Batch write (status=failed)      | ✅ Update   |

**Why skip Steps on success?**

- Steps are only needed for resume/retry operations
- If completed successfully, there's nothing to resume
- Reduces database writes and keeps Step table clean

### Why Two Writes?

| Scenario           | What Happens                        | Data Safe? |
| ------------------ | ----------------------------------- | ---------- |
| Normal completion  | `defer` triggers → Write 2 executes | ✅         |
| User clicks stop   | `defer` triggers → Write 2 executes | ✅         |
| LLM timeout        | `defer` triggers → Write 2 executes | ✅         |
| Tool failure       | `defer` triggers → Write 2 executes | ✅         |
| Network disconnect | `defer` triggers → Write 2 executes | ✅         |
| Process crash      | Service is down, user must retry    | N/A        |

**Note**: Process crash is a catastrophic failure handled at infrastructure level, not application level.

### Write Count Comparison

For a typical request: user input → hook_create → llm → tool → llm → hook_next → 5 messages

| Strategy               | Database Writes | Notes              |
| ---------------------- | --------------- | ------------------ |
| Write per operation    | 1 + 5 + 5 = 11  | One write per step |
| **Two-write strategy** | **2**           | Entry + Exit only  |

### Implementation

````go
func (ast *Assistant) Stream(ctx, inputMessages, options) {
    // ========== Write 1: Entry ==========
    userMsg := createUserMessage(ctx, inputMessages)
    chatStore.SaveMessages(ctx.ChatID, []*Message{userMsg})

    // ========== Memory Buffers ==========
    messageBuffer := NewMessageBuffer()
    stepBuffer := NewStepBuffer()

    // Track current step for error handling
    var currentStep *Step

    defer func() {
        // ========== Write 2: Exit (always executes) ==========
        // Determine final status for incomplete steps
        finalStatus := "completed"
        if ctx.IsInterrupted() {
            finalStatus = "interrupted"
        }
        if r := recover(); r != nil {
            finalStatus = "failed"
        }

        // Update status of any incomplete step
        if currentStep != nil && currentStep.Status == "running" {
            currentStep.Status = finalStatus
        }

        // Batch write all buffered data
        chatStore.SaveMessages(ctx.ChatID, messageBuffer.GetAll())
        chatStore.SaveSteps(stepBuffer.GetAll())

        // Update token usage in OpenAPI request record
        if ctx.RequestID != "" && completionResponse != nil {
            request.UpdateTokenUsage(
                ctx.RequestID,
                completionResponse.Usage.PromptTokens,
                completionResponse.Usage.CompletionTokens,
            )
        }
    }()

    // ========== Execution (all in memory) ==========
    // Note: request_id = ctx.RequestID (from OpenAPI middleware)

    // hook_create
    currentStep = stepBuffer.Add(createStep(ctx, "hook_create", "running", input, nil))
    createResponse := ast.HookScript.Create(...)
    currentStep.Output = createResponse
    currentStep.Status = "completed"

    // llm
    currentStep = stepBuffer.Add(createStep(ctx, "llm", "running", messages, nil))
    completionResponse := ast.executeLLMStream(...)
    currentStep.Output = completionResponse
    currentStep.Status = "completed"

    // tool (if any)
    for _, toolCall := range completionResponse.ToolCalls {
        currentStep = stepBuffer.Add(createStep(ctx, "tool", "running", toolCall, nil))
        result := executeToolCall(toolCall)
        currentStep.Output = result
        currentStep.Status = "completed"
    }

    // hook_next
    currentStep = stepBuffer.Add(createStep(ctx, "hook_next", "running", payload, nil))
    nextResponse := ast.HookScript.Next(...)
    currentStep.Output = nextResponse
    currentStep.Status = "completed"
    currentStep = nil // All done

    // Messages are automatically buffered via ctx.Send()
}

// createResumeRecord creates a resume record with context information
// Only called when request fails or is interrupted
func createResumeRecord(ctx *Context, stepType, status string, input, output interface{}, err error) *Resume {
    // Capture Space snapshot for recovery
    var spaceSnapshot map[string]interface{}
    if ctx.Space != nil {
        spaceSnapshot = ctx.Space.Snapshot() // Get all key-value pairs
    }

    errorMsg := ""
    if err != nil {
        errorMsg = err.Error()
    }

    return &Resume{
        ResumeID:      generateID(),
        ChatID:        ctx.ChatID,        // ChatID
        RequestID:     ctx.RequestID,     // From OpenAPI middleware
        AssistantID:   ctx.AssistantID,
        StackID:       ctx.Stack.ID,
        StackParentID: ctx.Stack.ParentID,
        StackDepth:    ctx.Stack.Depth,
        Type:          stepType,
        Status:        status,            // "failed" or "interrupted"
        Input:         input,
        Output:        output,
        SpaceSnapshot: spaceSnapshot,     // Shared space data for recovery
        Error:         errorMsg,
        Sequence:      nextSequence(),
    }
}

## API Interface

### ChatStore Interface

```go
// ChatStore defines the chat storage interface
type ChatStore interface {
    // Chat Management
    CreateChat(chat *Chat) error
    GetChat(chatID string) (*Chat, error)
    UpdateChat(chatID string, updates map[string]interface{}) error
    DeleteChat(chatID string) error
    ListChats(filter ChatFilter) (*ChatList, error)

    // Message Management
    SaveMessages(chatID string, messages []*Message) error
    GetMessages(chatID string, filter MessageFilter) ([]*Message, error)
    UpdateMessage(messageID string, updates map[string]interface{}) error
    DeleteMessages(chatID string, messageIDs []string) error

    // Resume Management (only called on failure/interrupt)
    SaveResume(records []*Resume) error
    GetResume(chatID string) ([]*Resume, error)
    GetLastResume(chatID string) (*Resume, error)
    GetResumeByStackID(stackID string) ([]*Resume, error)
    GetStackPath(stackID string) ([]string, error) // Returns [root_stack_id, ..., current_stack_id]
    DeleteResume(chatID string) error              // Clean up after successful resume
}

// SpaceStore defines the interface for Space snapshot operations
// Note: Space itself uses plan.Space interface, this is for persistence
type SpaceStore interface {
    // Snapshot returns all key-value pairs in the space
    Snapshot() map[string]interface{}

    // Restore sets multiple key-value pairs from a snapshot
    Restore(data map[string]interface{}) error
}
````

### Data Structures

```go
// Chat represents a chat session
type Chat struct {
    ChatID        string                 `json:"chat_id"`
    Title         string                 `json:"title,omitempty"`
    AssistantID   string                 `json:"assistant_id"`
    Mode          string                 `json:"mode"`
    Status        string                 `json:"status"`
    Preset        bool                   `json:"preset"`
    Public        bool                   `json:"public"`
    Share         string                 `json:"share"` // "private" or "team"
    Sort          int                    `json:"sort"`
    LastMessageAt *time.Time             `json:"last_message_at,omitempty"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt     time.Time              `json:"created_at"`
    UpdatedAt     time.Time              `json:"updated_at"`
}

// Message represents a chat message
type Message struct {
    MessageID   string                 `json:"message_id"`
    ChatID      string                 `json:"chat_id"`
    RequestID   string                 `json:"request_id,omitempty"`
    Role        string                 `json:"role"`
    Type        string                 `json:"type"`
    Props       map[string]interface{} `json:"props"`
    BlockID     string                 `json:"block_id,omitempty"`
    ThreadID    string                 `json:"thread_id,omitempty"`
    AssistantID string                 `json:"assistant_id,omitempty"`
    Sequence    int                    `json:"sequence"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// Resume represents an execution state for recovery (only stored on failure/interrupt)
type Resume struct {
    ResumeID      string                 `json:"resume_id"`
    ChatID        string                 `json:"chat_id"`
    RequestID     string                 `json:"request_id"`
    AssistantID   string                 `json:"assistant_id"`
    StackID       string                 `json:"stack_id"`
    StackParentID string                 `json:"stack_parent_id,omitempty"`
    StackDepth    int                    `json:"stack_depth"`
    Type          string                 `json:"type"`
    Status        string                 `json:"status"` // "failed" or "interrupted"
    Input         map[string]interface{} `json:"input,omitempty"`
    Output        map[string]interface{} `json:"output,omitempty"`
    SpaceSnapshot map[string]interface{} `json:"space_snapshot,omitempty"` // Shared space data for recovery
    Error         string                 `json:"error,omitempty"`
    Sequence      int                    `json:"sequence"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt     time.Time              `json:"created_at"`
    UpdatedAt     time.Time              `json:"updated_at"`
}
```

### Filter Structures

```go
// ChatFilter for listing chats
type ChatFilter struct {
    UserID      string `json:"user_id,omitempty"`
    TeamID      string `json:"team_id,omitempty"`
    AssistantID string `json:"assistant_id,omitempty"`
    Status      string `json:"status,omitempty"`
    Keywords    string `json:"keywords,omitempty"`
    Page        int    `json:"page,omitempty"`
    PageSize    int    `json:"pagesize,omitempty"`
}

// MessageFilter for listing messages
type MessageFilter struct {
    RequestID string `json:"request_id,omitempty"`
    Role      string `json:"role,omitempty"`
    BlockID   string `json:"block_id,omitempty"`
    Limit     int    `json:"limit,omitempty"`
    Offset    int    `json:"offset,omitempty"`
}

// ChatList paginated response
type ChatList struct {
    Data      []*Chat `json:"data"`
    Page      int     `json:"page"`
    PageSize  int     `json:"pagesize"`
    PageCount int     `json:"pagecount"`
    Total     int     `json:"total"`
}
```

## Usage Examples

### 1. Complete Message Storage Example

A typical conversation with various message types stored in `agent_message`:

```
User: "What's the weather in SF? Also show me a chart."

Timeline:
1. User sends multimodal input
2. Hook shows loading state
3. LLM thinks and calls tool
4. Tool returns result
5. LLM generates text response
6. Hook sends image chart
```

**Stored Messages:**

```json
[
  // 1. User input (role=user, type=user_input)
  {
    "message_id": "msg_001",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "user",
    "type": "user_input",
    "props": {
      "content": "What's the weather in SF? Also show me a chart.",
      "role": "user"
    },
    "sequence": 1
  },

  // 2. Loading state from Create hook (role=assistant, type=loading)
  {
    "message_id": "msg_002",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "loading",
    "props": {
      "message": "Searching knowledge base..."
    },
    "block_id": "B1",
    "assistant_id": "weather_assistant",
    "sequence": 2
  },

  // 3. LLM thinking process (role=assistant, type=thinking)
  {
    "message_id": "msg_003",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "thinking",
    "props": {
      "content": "User wants weather info for San Francisco. I should use the get_weather tool..."
    },
    "block_id": "B2",
    "assistant_id": "weather_assistant",
    "sequence": 3
  },

  // 4. LLM tool call (role=assistant, type=tool_call)
  {
    "message_id": "msg_004",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "tool_call",
    "props": {
      "id": "call_weather_001",
      "name": "get_weather",
      "arguments": "{\"location\": \"San Francisco\", \"unit\": \"celsius\"}"
    },
    "block_id": "B2",
    "assistant_id": "weather_assistant",
    "sequence": 4
  },

  // 5. Tool result (role=assistant, type=text, with tool metadata)
  {
    "message_id": "msg_005",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "text",
    "props": {
      "content": "Weather data retrieved: 18°C, sunny, humidity 65%"
    },
    "block_id": "B2",
    "metadata": {
      "tool_call_id": "call_weather_001",
      "tool_name": "get_weather",
      "is_tool_result": true
    },
    "assistant_id": "weather_assistant",
    "sequence": 5
  },

  // 6. LLM text response (role=assistant, type=text)
  {
    "message_id": "msg_006",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "text",
    "props": {
      "content": "The weather in San Francisco is currently **18°C** and sunny with 65% humidity. Perfect weather for outdoor activities!"
    },
    "block_id": "B2",
    "assistant_id": "weather_assistant",
    "sequence": 6
  },

  // 7. Chart image from Next hook (role=assistant, type=image)
  {
    "message_id": "msg_007",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "image",
    "props": {
      "url": "https://charts.example.com/weather_sf.png",
      "alt": "San Francisco 7-day weather forecast",
      "width": 800,
      "height": 400
    },
    "block_id": "B3",
    "assistant_id": "weather_assistant",
    "sequence": 7
  }
]
```

**Streaming IDs (from `STREAMING.md`):**

During streaming, messages include additional fields for real-time delivery:

| Field        | Purpose                        | Stored? |
| ------------ | ------------------------------ | ------- |
| `chunk_id`   | Deduplication, ordering, debug | ❌ No   |
| `message_id` | Delta merge target             | ✅ Yes  |
| `block_id`   | UI block/section grouping      | ✅ Yes  |
| `thread_id`  | Concurrent stream distinction  | ✅ Yes  |
| `delta`      | Whether this is a delta chunk  | ❌ No   |
| `delta_path` | Path for delta merge           | ❌ No   |

**Note:** `chunk_id`, `delta`, and `delta_path` are transient streaming control fields and are NOT stored. Only the final merged content is persisted.

### 2. Error Message Storage

When errors occur, they are stored as `error` type:

```json
{
  "message_id": "msg_err_001",
  "chat_id": "chat_123",
  "request_id": "req_abc",
  "role": "assistant",
  "type": "error",
  "props": {
    "message": "Failed to connect to weather service",
    "code": "SERVICE_UNAVAILABLE",
    "details": "Connection timeout after 30 seconds"
  },
  "block_id": "B2",
  "assistant_id": "weather_assistant",
  "sequence": 5
}
```

### 3. Action Message Storage (CUI clients)

System actions are stored but only processed by CUI clients:

```json
{
  "message_id": "msg_action_001",
  "chat_id": "chat_123",
  "request_id": "req_abc",
  "role": "assistant",
  "type": "action",
  "props": {
    "name": "open_panel",
    "payload": {
      "panel_id": "weather_details",
      "location": "San Francisco"
    }
  },
  "block_id": "B2",
  "assistant_id": "weather_assistant",
  "sequence": 6
}
```

### 4. Audio/Video Message Storage

Multimedia content storage:

```json
// Audio message
{
  "message_id": "msg_audio_001",
  "chat_id": "chat_123",
  "role": "assistant",
  "type": "audio",
  "props": {
    "url": "https://storage.example.com/audio/response.mp3",
    "format": "mp3",
    "duration": 45.5,
    "transcript": "Here's the weather forecast for today...",
    "controls": true
  },
  "sequence": 7
}

// Video message
{
  "message_id": "msg_video_001",
  "chat_id": "chat_123",
  "role": "assistant",
  "type": "video",
  "props": {
    "url": "https://storage.example.com/video/weather_report.mp4",
    "format": "mp4",
    "thumbnail": "https://storage.example.com/video/weather_report_thumb.jpg",
    "duration": 120.0,
    "width": 1280,
    "height": 720,
    "controls": true
  },
  "sequence": 8
}
```

### 5. Load Chat History

```go
// Get chat list
chats, _ := chatStore.ListChats(ChatFilter{
    UserID:   "user123",
    Status:   "active",
    Page:     1,
    PageSize: 20,
})

// Get messages for a chat
messages, _ := chatStore.GetMessages("chat_123", MessageFilter{
    Limit: 100,
})

// Return to frontend
return map[string]interface{}{
    "chat":     chat,
    "messages": messages,
}
```

### 6. Resume from Interruption

```go
func (ast *Assistant) Resume(ctx *Context) error {
    // 1. Find last resume record
    record, _ := chatStore.GetLastResume(ctx.ChatID)
    if record == nil {
        return nil // Nothing to resume
    }

    // 2. Restore Space data from snapshot
    if record.SpaceSnapshot != nil && ctx.Space != nil {
        for key, value := range record.SpaceSnapshot {
            ctx.Space.Set(key, value)
        }
    }

    // 3. Check if this is an A2A nested call
    if record.StackDepth > 0 {
        // Need to rebuild the call stack
        return ast.ResumeNestedCall(ctx, record)
    }

    // 4. Resume based on step type
    var err error
    switch record.Type {
    case "llm":
        // Re-execute LLM call with saved input
        messages := record.Input["messages"].([]Message)
        err = ast.executeLLMStream(ctx, messages, ...)

    case "tool":
        // Retry tool call
        err = ast.retryToolCall(ctx, record)

    case "hook_next":
        // Re-execute hook
        err = ast.executeHookNext(ctx, record.Input)

    case "delegate":
        // Resume delegated agent call
        agentID := record.Input["agent_id"].(string)
        messages := record.Input["messages"].([]Message)
        err = ast.delegateToAgent(ctx, agentID, messages)
    }

    // 5. Clean up resume records on success
    if err == nil {
        chatStore.DeleteResume(ctx.ChatID)
    }

    return err
}
```

### 7. Resume A2A Nested Calls

For agent-to-agent (A2A) recursive calls, the stack information is essential for proper recovery.

```go
func (ast *Assistant) ResumeNestedCall(ctx *Context, step *Step) error {
    // 1. Rebuild the call stack from root to interrupted point
    stackPath, _ := chatStore.GetStackPath(step.StackID)
    // stackPath: [root_stack_id, parent_stack_id, ..., current_stack_id]

    // 2. Get all steps for each stack level
    for _, stackID := range stackPath {
        steps, _ := chatStore.GetStepsByStackID(stackID)
        // Restore context for each level
    }

    // 3. Resume from the interrupted assistant
    targetAssistant := assistant.Select(step.AssistantID)
    return targetAssistant.Stream(ctx, step.Input["messages"], ...)
}
```

### 8. Handle Interruption

Interruption is handled automatically by the `defer` block in the two-write strategy. When `ctx.IsInterrupted()` returns true, the status is set to `interrupted` and all buffered data is saved.

```go
// Inside the defer block (see Write Strategy - Implementation)
if ctx.IsInterrupted() {
    status = "interrupted"
}
// Then batch write all buffered messages and steps
```

## A2A (Agent-to-Agent) Call Example

When Assistant A delegates to Assistant B, the step records look like:

```
Request: User asks "analyze this data and visualize it"

Step Records:
┌─────┬─────────────┬─────────────┬──────────┬───────┬───────┬─────────────┬─────────────────────────────┐
│ seq │ assistant   │ stack_id    │ parent   │ depth │ type  │ status      │ space_snapshot              │
├─────┼─────────────┼─────────────┼──────────┼───────┼───────┼─────────────┼─────────────────────────────┤
│  1  │ analyzer    │ stk_001     │ null     │ 0     │ input │ completed   │ {}                          │
│  2  │ analyzer    │ stk_001     │ null     │ 0     │ llm   │ completed   │ {}                          │
│  3  │ analyzer    │ stk_001     │ null     │ 0     │ delegate │ running  │ {"choose_prompt": "query"}  │ ← Space data set before delegate
│  4  │ visualizer  │ stk_002     │ stk_001  │ 1     │ input │ completed   │ {"choose_prompt": "query"}  │
│  5  │ visualizer  │ stk_002     │ stk_001  │ 1     │ llm   │ interrupted │ {"choose_prompt": "query"}  │ ← interrupted here
└─────┴─────────────┴─────────────┴──────────┴───────┴───────┴─────────────┴─────────────────────────────┘

Resume Flow:
1. Find step with status="interrupted" → step 5
2. Restore Space from space_snapshot: {"choose_prompt": "query"}
3. Check stack_depth=1 → nested call
4. Get stack path: [stk_001, stk_002]
5. Resume visualizer assistant with step 5's input
6. When visualizer completes, update step 3 (delegate) to completed
```

**Space Snapshot Use Case (from expense assistant):**

```typescript
// In Next hook, before delegating to another agent
ctx.space.Set("choose_prompt", "query");
return {
  delegate: { agent_id: "expense", messages: payload.messages },
};

// If interrupted during delegate, Resume will:
// 1. Restore space_snapshot → ctx.space now has "choose_prompt": "query"
// 2. The delegated agent's Create hook can read: ctx.space.GetDel("choose_prompt")
```

## Related Documents

- [OpenAPI Request Design](../../openapi/request/REQUEST_DESIGN.md) - Global request tracking, billing, rate limiting
- [Trace Module](../../trace/README.md) - Detailed execution tracing for debugging
- [Agent Context](../context/README.md) - Context and message handling
