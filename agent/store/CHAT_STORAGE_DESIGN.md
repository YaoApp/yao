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
| `last_connector`  | string(200) | Yes      | Yes    | Last used connector ID           |
| `last_mode`       | string(50)  | Yes      | -      | Last used chat mode (chat/task)  |
| `status`          | enum        | No       | Yes    | Status: `active`, `archived`     |
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
| `idx_chat_last_conn` | `last_connector`  | index |
| `idx_chat_status`    | `status`          | index |
| `idx_chat_share`     | `share`           | index |
| `idx_chat_last_msg`  | `last_message_at` | index |

### 2. Message Table

Stores user-visible messages (both user input and assistant responses).

**Table Name:** `agent_message`

| Column         | Type        | Nullable | Index | Description                                 |
| -------------- | ----------- | -------- | ----- | ------------------------------------------- |
| `id`           | ID          | No       | PK    | Auto-increment primary key                  |
| `message_id`   | string(64)  | No       | -     | Message identifier (unique within request)  |
| `chat_id`      | string(64)  | No       | Yes   | Parent chat ID                              |
| `request_id`   | string(64)  | Yes      | Yes   | Request ID for grouping                     |
| `role`         | enum        | No       | Yes   | Role: `user`, `assistant`                   |
| `type`         | string(50)  | No       | -     | Message type (text, image, loading, etc.)   |
| `props`        | json        | No       | -     | Message properties (content, url, etc.)     |
| `block_id`     | string(64)  | Yes      | Yes   | Block grouping ID                           |
| `thread_id`    | string(64)  | Yes      | Yes   | Thread grouping ID                          |
| `assistant_id` | string(200) | Yes      | Yes   | Assistant ID (join to get name/avatar)      |
| `connector`    | string(200) | Yes      | Yes   | Connector ID used for this message          |
| `mode`         | string(50)  | Yes      | -     | Chat mode used for this message (chat/task) |
| `sequence`     | integer     | No       | -     | Message order within chat (in composite)    |
| `metadata`     | json        | Yes      | -     | Additional metadata                         |
| `created_at`   | timestamp   | No       | Yes   | Creation timestamp                          |
| `updated_at`   | timestamp   | No       | -     | Last update timestamp                       |

**Indexes:**

| Name                      | Columns                    | Type   |
| ------------------------- | -------------------------- | ------ |
| `idx_msg_chat_seq`        | `chat_id`, `sequence`      | index  |
| `idx_msg_request_message` | `request_id`, `message_id` | unique |
| `idx_msg_request`         | `request_id`               | index  |
| `idx_msg_role`            | `role`                     | index  |
| `idx_msg_block`           | `block_id`                 | index  |
| `idx_msg_thread`          | `thread_id`                | index  |
| `idx_msg_assistant`       | `assistant_id`             | index  |

**Message Ordering:**

Messages are ordered by `created_at` first, then by `sequence` within the same timestamp. This ensures correct chronological order when there are multiple requests with overlapping sequence numbers:

```sql
ORDER BY created_at ASC, sequence ASC
```

**Why this ordering?**

- `sequence` is assigned per-request, so different requests may have the same sequence numbers
- `created_at` groups messages by request time, ensuring messages from earlier requests appear first
- Within the same request (same `created_at`), `sequence` preserves the internal ordering

**Message Types:**

All message types are stored, including built-in types and custom types. See `agent/output/BUILTIN_TYPES.md` for built-in Props structures.

| Type         | Description                      | Props Example                                                                               | Stored? |
| ------------ | -------------------------------- | ------------------------------------------------------------------------------------------- | ------- |
| `user_input` | User input (frontend display)    | `{"content": "Hello", "role": "user", "name": "John"}`                                      | ✅ Yes  |
| `text`       | Text/Markdown content            | `{"content": "Hello **world**!"}`                                                           | ✅ Yes  |
| `thinking`   | Reasoning process (o1, DeepSeek) | `{"content": "Let me analyze..."}`                                                          | ✅ Yes  |
| `loading`    | Loading/processing indicator     | `{"message": "Searching knowledge base..."}`                                                | ✅ Yes  |
| `tool_call`  | LLM tool/function call           | `{"id": "call_abc123", "name": "get_weather", "arguments": "{\"location\":\"SF\"}"}`        | ✅ Yes  |
| `retrieval`  | KB/Web search results            | `{"query": "...", "sources": [...], "total_results": 10}`                                   | ✅ Yes  |
| `error`      | Error message                    | `{"message": "Connection timeout", "code": "TIMEOUT", "details": "..."}`                    | ✅ Yes  |
| `image`      | Image content                    | `{"url": "...", "alt": "...", "width": 200, "height": 200, "detail": "auto"}`               | ✅ Yes  |
| `audio`      | Audio content                    | `{"url": "...", "format": "mp3", "duration": 120.5, "transcript": "...", "controls": true}` | ✅ Yes  |
| `video`      | Video content                    | `{"url": "...", "format": "mp4", "thumbnail": "...", "width": 640, "height": 360}`          | ✅ Yes  |
| `action`     | System action (CUI only)         | `{"name": "open_panel", "payload": {"panel_id": "user_profile"}}`                           | ✅ Yes  |
| `event`      | Lifecycle event (CUI only)       | `{"event": "stream_start", "message": "...", "data": {...}}`                                | ❌ No   |
| `*` (custom) | Any custom type                  | `{"chartType": "bar", "data": [...], "options": {...}}`                                     | ✅ Yes  |

**Note on `event` type:** Lifecycle events (`stream_start`, `stream_end`, etc.) are transient control signals and are NOT stored. They are only used for real-time streaming coordination.

**Note on custom types:** Any type not in the built-in list is stored as-is with its original `type` and `props` structure.

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

| Column            | Type        | Nullable | Index  | Description                              |
| ----------------- | ----------- | -------- | ------ | ---------------------------------------- |
| `id`              | ID          | No       | PK     | Auto-increment primary key               |
| `resume_id`       | string(64)  | No       | Unique | Unique resume record identifier          |
| `chat_id`         | string(64)  | No       | Yes    | Parent chat ID                           |
| `request_id`      | string(64)  | No       | Yes    | Request ID                               |
| `assistant_id`    | string(200) | No       | Yes    | Assistant executing this step            |
| `stack_id`        | string(64)  | No       | Yes    | Stack node ID for this execution         |
| `stack_parent_id` | string(64)  | Yes      | Yes    | Parent stack ID (for A2A calls)          |
| `stack_depth`     | integer     | No       | -      | Call depth (0=root, 1+=nested)           |
| `type`            | enum        | No       | Yes    | Step type                                |
| `status`          | enum        | No       | Yes    | Status: `interrupted`, `failed`          |
| `input`           | json        | Yes      | -      | Step input data                          |
| `output`          | json        | Yes      | -      | Step output data (partial)               |
| `space_snapshot`  | json        | Yes      | -      | Space data snapshot for recovery         |
| `error`           | text        | Yes      | -      | Error message if failed                  |
| `sequence`        | integer     | No       | -      | Step order within request (in composite) |
| `metadata`        | json        | Yes      | -      | Additional metadata                      |
| `created_at`      | timestamp   | No       | Yes    | Creation timestamp                       |
| `updated_at`      | timestamp   | No       | -      | Last update timestamp                    |

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
| `idx_resume_type`      | `type`                   | index |
| `idx_resume_status`    | `status`                 | index |
| `idx_resume_stack`     | `stack_id`               | index |
| `idx_resume_parent`    | `stack_parent_id`        | index |
| `idx_resume_assistant` | `assistant_id`           | index |

## Write Strategy

### Single-Write Strategy

All data is buffered in memory during execution and written to database **only once** when `Stream()` exits:

**Note**: Request tracking (status, tokens, duration) is handled by [OpenAPI Request Middleware](../../openapi/request/REQUEST_DESIGN.md).

```
Stream() Entry
    │
    ├── Buffer user input message (role=user)
    │
    ├── Execution (all in memory)
    │   - ctx.Send()    → messageBuffer
    │   - ctx.Append()  → update messageBuffer
    │   - ctx.Replace() → update messageBuffer
    │   - Each step     → stepBuffer
    │
    └── 【Single Write】Save final state (via defer)
        │
        ├── Always:
        │   - Batch write all messages (user input + assistant responses)
        │   - Update token usage in openapi_request (via request_id)
        │
        └── Only on error/interrupt:
            - Batch write all steps (for resume/retry)
```

### Write Points

| Event            | Message Table                          | Step Table                          | Token Usage |
| ---------------- | -------------------------------------- | ----------------------------------- | ----------- |
| Stream entry     | Buffer user input                      | -                                   | -           |
| During execution | Buffer in memory                       | Buffer in memory                    | -           |
| **Completed**    | **Batch write all (user + assistant)** | **❌ Skip (no need to resume)**     | ✅ Update   |
| On interrupt     | Batch write all buffered               | ✅ Batch write (status=interrupted) | ✅ Update   |
| On error         | Batch write all buffered               | ✅ Batch write (status=failed)      | ✅ Update   |

**Why skip Steps on success?**

- Steps are only needed for resume/retry operations
- If completed successfully, there's nothing to resume
- Reduces database writes and keeps Resume table clean

### Why Single Write?

| Scenario           | What Happens                      | Data Safe? |
| ------------------ | --------------------------------- | ---------- |
| Normal completion  | `defer` triggers → Write executes | ✅         |
| User clicks stop   | `defer` triggers → Write executes | ✅         |
| LLM timeout        | `defer` triggers → Write executes | ✅         |
| Tool failure       | `defer` triggers → Write executes | ✅         |
| Network disconnect | `defer` triggers → Write executes | ✅         |
| Process crash      | Service is down, user must retry  | N/A        |

**Note**: Process crash is a catastrophic failure handled at infrastructure level, not application level.

### Write Count Comparison

For a typical request: user input → hook_create → llm → tool → hook_next → 5 messages

| Strategy                  | Database Writes | Notes                 |
| ------------------------- | --------------- | --------------------- |
| Write per operation       | 1 + 5 + 5 = 11  | One write per step    |
| **Single-write strategy** | **1**           | Exit only (via defer) |

### Implementation

````go
func (ast *Assistant) Stream(ctx, inputMessages, options) {
    // ========== Memory Buffers ==========
    messageBuffer := NewMessageBuffer()
    stepBuffer := NewStepBuffer()

    // Buffer user input message (not written yet)
    userMsg := createUserMessage(ctx, inputMessages)
    messageBuffer.Add(userMsg)

    // Track current step for error handling
    var currentStep *Step

    defer func() {
        // ========== Single Write: Exit (always executes) ==========
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

        // Batch write all buffered messages (user input + assistant responses)
        chatStore.SaveMessages(ctx.ChatID, messageBuffer.GetAll())

        // Only save steps on error/interrupt (not on success)
        if finalStatus != "completed" {
            chatStore.SaveResume(stepBuffer.GetAll())
        }

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
// Provides operations for chat, message, and resume management
type ChatStore interface {
    // ==========================================================================
    // Chat Management
    // ==========================================================================

    // CreateChat creates a new chat session
    CreateChat(chat *Chat) error

    // GetChat retrieves a single chat by ID
    GetChat(chatID string) (*Chat, error)

    // UpdateChat updates chat fields
    UpdateChat(chatID string, updates map[string]interface{}) error

    // DeleteChat deletes a chat and its associated messages
    DeleteChat(chatID string) error

    // ListChats retrieves a paginated list of chats with optional grouping
    ListChats(filter ChatFilter) (*ChatList, error)

    // ==========================================================================
    // Message Management
    // ==========================================================================

    // SaveMessages batch saves messages for a chat
    // This is the primary write method - messages are buffered during execution
    // and batch-written at the end of a request
    SaveMessages(chatID string, messages []*Message) error

    // GetMessages retrieves messages for a chat with filtering
    GetMessages(chatID string, filter MessageFilter) ([]*Message, error)

    // UpdateMessage updates a single message
    UpdateMessage(messageID string, updates map[string]interface{}) error

    // DeleteMessages deletes specific messages from a chat
    DeleteMessages(chatID string, messageIDs []string) error

    // ==========================================================================
    // Resume Management (only called on failure/interrupt)
    // ==========================================================================

    // SaveResume batch saves resume records
    // Only called when request is interrupted or failed
    SaveResume(records []*Resume) error

    // GetResume retrieves all resume records for a chat
    GetResume(chatID string) ([]*Resume, error)

    // GetLastResume retrieves the last (most recent) resume record for a chat
    GetLastResume(chatID string) (*Resume, error)

    // GetResumeByStackID retrieves resume records for a specific stack
    GetResumeByStackID(stackID string) ([]*Resume, error)

    // GetStackPath returns the stack path from root to the given stack
    // Returns: [root_stack_id, ..., current_stack_id]
    GetStackPath(stackID string) ([]string, error)

    // DeleteResume deletes all resume records for a chat
    // Called after successful resume to clean up
    DeleteResume(chatID string) error
}

// AssistantStore defines the assistant storage interface
// Separated from ChatStore for clearer responsibility
type AssistantStore interface {
    // SaveAssistant saves assistant information
    SaveAssistant(assistant *AssistantModel) (string, error)

    // UpdateAssistant updates assistant fields
    UpdateAssistant(assistantID string, updates map[string]interface{}) error

    // DeleteAssistant deletes an assistant
    DeleteAssistant(assistantID string) error

    // GetAssistants retrieves a paginated list of assistants with filtering
    GetAssistants(filter AssistantFilter, locale ...string) (*AssistantList, error)

    // GetAssistantTags retrieves all unique tags from assistants with filtering
    GetAssistantTags(filter AssistantFilter, locale ...string) ([]Tag, error)

    // GetAssistant retrieves a single assistant by ID
    GetAssistant(assistantID string, fields []string, locale ...string) (*AssistantModel, error)

    // DeleteAssistants deletes assistants based on filter conditions
    DeleteAssistants(filter AssistantFilter) (int64, error)
}

// Store combines ChatStore and AssistantStore interfaces
// This is the main interface for the storage layer
type Store interface {
    ChatStore
    AssistantStore
}

// SpaceStore defines the interface for Space snapshot operations
// Note: Space itself uses plan.Space interface, this is for persistence
type SpaceStore interface {
    // Snapshot returns all key-value pairs in the space
    Snapshot() map[string]interface{}

    // Restore sets multiple key-value pairs from a snapshot
    Restore(data map[string]interface{}) error
}
```

### Data Structures

```go
// Chat represents a chat session
type Chat struct {
    ChatID        string                 `json:"chat_id"`
    Title         string                 `json:"title,omitempty"`
    AssistantID   string                 `json:"assistant_id"`
    LastConnector string                 `json:"last_connector,omitempty"` // Last used connector ID (updated on each message)
    LastMode      string                 `json:"last_mode,omitempty"`      // Last used chat mode (updated on each message)
    Status        string                 `json:"status"`          // "active" or "archived"
    Public        bool                   `json:"public"`          // Whether shared across all teams
    Share         string                 `json:"share"`           // "private" or "team"
    Sort          int                    `json:"sort"`            // Sort order for display
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
    Role        string                 `json:"role"` // "user" or "assistant"
    Type        string                 `json:"type"` // "text", "image", "loading", "tool_call", "retrieval", etc.
    Props       map[string]interface{} `json:"props"`
    BlockID     string                 `json:"block_id,omitempty"`
    ThreadID    string                 `json:"thread_id,omitempty"`
    AssistantID string                 `json:"assistant_id,omitempty"`
    Connector   string                 `json:"connector,omitempty"` // Connector ID used for this message
    Mode        string                 `json:"mode,omitempty"`      // Chat mode used for this message (chat or task)
    Sequence    int                    `json:"sequence"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// Resume represents an execution state for recovery
// Only stored when request is interrupted or failed
type Resume struct {
    ResumeID      string                 `json:"resume_id"`
    ChatID        string                 `json:"chat_id"`
    RequestID     string                 `json:"request_id"`
    AssistantID   string                 `json:"assistant_id"`
    StackID       string                 `json:"stack_id"`
    StackParentID string                 `json:"stack_parent_id,omitempty"`
    StackDepth    int                    `json:"stack_depth"`
    Type          string                 `json:"type"`   // "input", "hook_create", "llm", "tool", "hook_next", "delegate"
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

// ResumeStatus constants
const (
    ResumeStatusFailed      = "failed"
    ResumeStatusInterrupted = "interrupted"
)

// ResumeType constants
const (
    ResumeTypeInput      = "input"
    ResumeTypeHookCreate = "hook_create"
    ResumeTypeLLM        = "llm"
    ResumeTypeTool       = "tool"
    ResumeTypeHookNext   = "hook_next"
    ResumeTypeDelegate   = "delegate"
)
```

### Filter Structures

```go
// ChatFilter for listing chats
type ChatFilter struct {
    // Permission filters (direct filtering on Yao permission fields)
    UserID string `json:"user_id,omitempty"` // Filter by __yao_created_by
    TeamID string `json:"team_id,omitempty"` // Filter by __yao_team_id

    // Business filters
    AssistantID string `json:"assistant_id,omitempty"`
    Status      string `json:"status,omitempty"`
    Keywords    string `json:"keywords,omitempty"`

    // Time range filter
    StartTime *time.Time `json:"start_time,omitempty"` // Filter chats after this time
    EndTime   *time.Time `json:"end_time,omitempty"`   // Filter chats before this time
    TimeField string     `json:"time_field,omitempty"` // Field for time filter: "created_at" or "last_message_at" (default)

    // Sorting
    OrderBy string `json:"order_by,omitempty"` // Field to sort by (default: "last_message_at")
    Order   string `json:"order,omitempty"`    // Sort order: "desc" (default) or "asc"

    // Response format
    GroupBy string `json:"group_by,omitempty"` // "time" for time-based groups, empty for flat list

    // Pagination
    Page     int `json:"page,omitempty"`
    PageSize int `json:"pagesize,omitempty"`

    // Advanced permission filter (not serialized)
    // Use for complex conditions like: (created_by = user OR team_id = team)
    QueryFilter func(query.Query) `json:"-"`
}

// MessageFilter for listing messages
type MessageFilter struct {
    RequestID string `json:"request_id,omitempty"`
    Role      string `json:"role,omitempty"`
    BlockID   string `json:"block_id,omitempty"`
    ThreadID  string `json:"thread_id,omitempty"`
    Type      string `json:"type,omitempty"`
    Limit     int    `json:"limit,omitempty"`
    Offset    int    `json:"offset,omitempty"`
}

// ChatList paginated response with time-based grouping
type ChatList struct {
    Data      []*Chat      `json:"data"`
    Groups    []*ChatGroup `json:"groups,omitempty"` // Time-based groups for UI display
    Page      int          `json:"page"`
    PageSize  int          `json:"pagesize"`
    PageCount int          `json:"pagecount"`
    Total     int          `json:"total"`
}

// ChatGroup represents a time-based group of chats
type ChatGroup struct {
    Label string   `json:"label"` // "Today", "Yesterday", "This Week", "This Month", "Earlier"
    Key   string   `json:"key"`   // "today", "yesterday", "this_week", "this_month", "earlier"
    Chats []*Chat  `json:"chats"` // Chats in this group
    Count int      `json:"count"` // Number of chats in group
}
```

## Usage Examples

### 1. Complete Message Storage Example

A typical conversation with various message types stored in `agent_message`:

```
User: "What's the weather in SF? Also show me a chart."

Timeline (user input → hook_create → llm → tool → hook_next):
1. User sends input
2. Create hook shows loading state
3. LLM thinks and calls tool
4. Tool executes and returns result
5. Next hook generates text response and image chart
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

  // 5. Tool result from Next hook (role=assistant, type=text, with tool metadata)
  {
    "message_id": "msg_005",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "text",
    "props": {
      "content": "The weather in San Francisco is currently **18°C** and sunny with 65% humidity. Perfect weather for outdoor activities!"
    },
    "block_id": "B3",
    "metadata": {
      "tool_call_id": "call_weather_001",
      "tool_name": "get_weather"
    },
    "assistant_id": "weather_assistant",
    "sequence": 5
  },

  // 6. Chart image from Next hook (role=assistant, type=image)
  {
    "message_id": "msg_006",
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
    "sequence": 6
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
// Example 1: Filter by user (simple permission check)
chats, _ := chatStore.ListChats(ChatFilter{
    UserID:   "user123",  // Filters by __yao_created_by
    Status:   "active",
    OrderBy:  "last_message_at",
    Order:    "desc",
    Page:     1,
    PageSize: 20,
})
// Response: chats.Data = [...], chats.Groups = nil

// Example 2: Filter by team
chats, _ := chatStore.ListChats(ChatFilter{
    TeamID:   "team456",  // Filters by __yao_team_id
    Status:   "active",
    Page:     1,
    PageSize: 20,
})

// Example 3: Filter by user AND team (both must match)
chats, _ := chatStore.ListChats(ChatFilter{
    UserID:   "user123",
    TeamID:   "team456",
    Page:     1,
    PageSize: 20,
})

// Example 4: Complex permission filter (user OR team) using QueryFilter
chats, _ := chatStore.ListChats(ChatFilter{
    Page:     1,
    PageSize: 20,
    QueryFilter: func(qb query.Query) {
        qb.Where(func(sub query.Query) {
            sub.Where("__yao_created_by", "user123").
                OrWhere("__yao_team_id", "team456")
        })
    },
})

// Example 5: Grouped by time
chats, _ := chatStore.ListChats(ChatFilter{
    UserID:   "user123",
    GroupBy:  "time", // Enable time-based grouping
    OrderBy:  "last_message_at",
    Order:    "desc",
    Page:     1,
    PageSize: 20,
})
// Response includes time-based groups:
// chats.Groups = [
//   { Key: "today", Label: "Today", Chats: [...], Count: 3 },
//   { Key: "yesterday", Label: "Yesterday", Chats: [...], Count: 5 },
//   { Key: "this_week", Label: "This Week", Chats: [...], Count: 8 },
//   { Key: "this_month", Label: "This Month", Chats: [...], Count: 4 },
//   { Key: "earlier", Label: "Earlier", Chats: [...], Count: 0 },
// ]

// Example 6: Filter by time range
startTime := time.Now().AddDate(0, 0, -7) // Last 7 days
chats, _ := chatStore.ListChats(ChatFilter{
    UserID:    "user123",
    StartTime: &startTime,
    TimeField: "last_message_at", // Filter by last message time
    OrderBy:   "last_message_at",
    Order:     "desc",
})

// Example 7: Filter specific date range
start := time.Date(2024, 12, 1, 0, 0, 0, 0, time.Local)
end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.Local)
chats, _ := chatStore.ListChats(ChatFilter{
    UserID:    "user123",
    StartTime: &start,
    EndTime:   &end,
    TimeField: "created_at", // Filter by creation time
})

// Example 8: Combine permission with business filters
chats, _ := chatStore.ListChats(ChatFilter{
    UserID:      "user123",
    TeamID:      "team456",
    AssistantID: "weather_assistant",
    Status:      "active",
    Keywords:    "weather",
    Page:        1,
    PageSize:    20,
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

## Concurrent Operations Storage

When an Agent makes parallel calls (e.g., multiple MCP tools, multiple sub-agents), messages use `block_id` and `thread_id` for grouping:

```
Main Agent concurrently calls 3 tasks:
├── Thread T1: Weather query (MCP)
├── Thread T2: News search (MCP)
├── Thread T3: Stock query (MCP)
└── Wait for all to complete, then summarize
```

**Stored Messages:**

```json
[
  // All concurrent messages share the same block_id, different thread_id
  // Messages may arrive in any order due to concurrency

  // Thread T1: Weather result
  {
    "message_id": "msg_t1_001",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "text",
    "props": { "content": "Weather in SF: 18°C, sunny" },
    "block_id": "B1",
    "thread_id": "T1",
    "assistant_id": "main_assistant",
    "sequence": 2
  },

  // Thread T2: News result
  {
    "message_id": "msg_t2_001",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "text",
    "props": { "content": "Top news: AI breakthrough announced..." },
    "block_id": "B1",
    "thread_id": "T2",
    "assistant_id": "main_assistant",
    "sequence": 3
  },

  // Thread T3: Stock result
  {
    "message_id": "msg_t3_001",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "text",
    "props": { "content": "AAPL: $185.50 (+1.2%)" },
    "block_id": "B1",
    "thread_id": "T3",
    "assistant_id": "main_assistant",
    "sequence": 4
  },

  // After all threads complete, main agent summarizes (new block)
  {
    "message_id": "msg_summary",
    "chat_id": "chat_123",
    "request_id": "req_abc",
    "role": "assistant",
    "type": "text",
    "props": {
      "content": "Here's your daily briefing: The weather is great at 18°C..."
    },
    "block_id": "B2",
    "thread_id": null,
    "assistant_id": "main_assistant",
    "sequence": 5
  }
]
```

**Key Points:**

| Field       | Concurrent Usage                                   |
| ----------- | -------------------------------------------------- |
| `block_id`  | Same for all parallel operations (B1)              |
| `thread_id` | Different for each concurrent task (T1, T2, T3)    |
| `sequence`  | Reflects actual arrival order (may be interleaved) |

**Frontend Rendering:**

- Group messages by `block_id` for visual blocks
- Within a block, optionally group by `thread_id` to show parallel results
- Use `sequence` for chronological display

## HTTP API

The chat storage provides RESTful HTTP APIs for managing chat sessions and messages.

**Base Path:** `/v1/chat`

### Chat Sessions

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/sessions` | List chat sessions with pagination and filtering |
| `GET` | `/sessions/:chat_id` | Get a single chat session |
| `PUT` | `/sessions/:chat_id` | Update chat session (title, status, metadata) |
| `DELETE` | `/sessions/:chat_id` | Delete chat session |
| `GET` | `/sessions/:chat_id/messages` | Get messages for a chat session |

### List Chat Sessions

**Request:**

```
GET /v1/chat/sessions?page=1&pagesize=20&assistant_id=xxx&status=active&keywords=search&group_by=time
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `pagesize` | int | 20 | Items per page (max 100) |
| `assistant_id` | string | - | Filter by assistant ID |
| `status` | string | - | Filter by status: `active`, `archived` |
| `keywords` | string | - | Search in title |
| `start_time` | RFC3339 | - | Filter chats after this time |
| `end_time` | RFC3339 | - | Filter chats before this time |
| `time_field` | string | `last_message_at` | Field for time filter: `created_at` or `last_message_at` |
| `order_by` | string | `last_message_at` | Sort field |
| `order` | string | `desc` | Sort order: `asc` or `desc` |
| `group_by` | string | - | Set to `time` for time-based grouping |

**Response:**

```json
{
  "data": [
    {
      "chat_id": "chat_123",
      "title": "Weather Query",
      "assistant_id": "weather_assistant",
      "status": "active",
      "last_message_at": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "groups": [
    {
      "key": "today",
      "label": "Today",
      "chats": [...],
      "count": 3
    },
    {
      "key": "yesterday",
      "label": "Yesterday",
      "chats": [...],
      "count": 5
    }
  ],
  "page": 1,
  "pagesize": 20,
  "pagecount": 5,
  "total": 100
}
```

### Get Chat Session

**Request:**

```
GET /v1/chat/sessions/chat_123
```

**Response:**

```json
{
  "chat_id": "chat_123",
  "title": "Weather Query",
  "assistant_id": "weather_assistant",
  "last_connector": "deepseek.v3",
  "last_mode": "chat",
  "status": "active",
  "public": false,
  "share": "private",
  "last_message_at": "2024-01-15T10:30:00Z",
  "metadata": {},
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Update Chat Session

**Request:**

```
PUT /v1/chat/sessions/chat_123
Content-Type: application/json

{
  "title": "New Title",
  "status": "archived",
  "metadata": {"custom_field": "value"}
}
```

**Response:**

```json
{
  "message": "Chat updated successfully",
  "chat_id": "chat_123"
}
```

### Delete Chat Session

**Request:**

```
DELETE /v1/chat/sessions/chat_123
```

**Response:**

```json
{
  "message": "Chat deleted successfully",
  "chat_id": "chat_123"
}
```

### Get Chat Messages

**Request:**

```
GET /v1/chat/sessions/chat_123/messages?limit=100&offset=0&role=assistant&type=text
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `request_id` | string | - | Filter by request ID |
| `role` | string | - | Filter by role: `user`, `assistant` |
| `block_id` | string | - | Filter by block ID |
| `thread_id` | string | - | Filter by thread ID |
| `type` | string | - | Filter by message type |
| `limit` | int | 100 | Max messages to return (max 1000) |
| `offset` | int | 0 | Offset for pagination |
| `locale` | string | - | Locale for assistant info (e.g., `zh-cn`, `en-us`). Falls back to `Accept-Language` header |

**Locale Resolution Priority:**
1. Query parameter `locale`
2. HTTP header `Accept-Language`

**Response:**

```json
{
  "chat_id": "chat_123",
  "messages": [
    {
      "message_id": "msg_001",
      "chat_id": "chat_123",
      "request_id": "req_abc",
      "role": "user",
      "type": "user_input",
      "props": {
        "content": "What's the weather?",
        "role": "user"
      },
      "sequence": 1,
      "created_at": "2024-01-15T10:00:00Z"
    },
    {
      "message_id": "msg_002",
      "chat_id": "chat_123",
      "request_id": "req_abc",
      "role": "assistant",
      "type": "text",
      "props": {
        "content": "The weather in San Francisco is 18°C and sunny."
      },
      "block_id": "B1",
      "assistant_id": "weather_assistant",
      "sequence": 2,
      "created_at": "2024-01-15T10:00:05Z"
    }
  ],
  "count": 2,
  "assistants": {
    "weather_assistant": {
      "assistant_id": "weather_assistant",
      "name": "Weather Assistant",
      "avatar": "https://example.com/weather-avatar.png",
      "description": "Get weather information for any location"
    }
  }
}
```

**Note:** The `assistants` field contains localized assistant information (name, avatar, description) for all unique `assistant_id` values found in the messages. This allows the frontend to display assistant details without additional API calls. The locale is determined by the `locale` query parameter or `Accept-Language` header.

### Permission Filtering

All endpoints respect Yao's permission system:

| Constraint | Behavior |
|------------|----------|
| `OwnerOnly` | User can only access their own chats (`__yao_created_by` matches) |
| `TeamOnly` | User can access own chats OR team-shared chats (`share = "team"`) |
| No constraints | Full access (for admin users) |

**Permission Fields Used:**

- `__yao_created_by`: User who created the chat
- `__yao_team_id`: Team ID for team-level access
- `public`: Whether chat is public to all
- `share`: Sharing scope (`private` or `team`)

## Related Documents

- [OpenAPI Request Design](../../openapi/request/REQUEST_DESIGN.md) - Global request tracking, billing, rate limiting
- [Trace Module](../../trace/README.md) - Detailed execution tracing for debugging
- [Agent Context](../context/README.md) - Context and message handling
````
