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

| Concern            | Module            | Table                |
| ------------------ | ----------------- | -------------------- |
| Request tracking   | `openapi/request` | `openapi_request`    |
| Billing (tokens)   | `openapi/request` | `openapi_request`    |
| Rate limiting      | `openapi/request` | -                    |
| Chat conversations | `agent/store`     | `agent_conversation` |
| Chat messages      | `agent/store`     | `agent_message`      |
| Execution steps    | `agent/store`     | `agent_step`         |

The `request_id` from OpenAPI middleware is passed to Agent and stored in messages/steps for correlation.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Chat Storage                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐                                        │
│  │  Conversation   │  Metadata: title, assistant, user      │
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
│  │     Step        │  Execution: type, status, input/output │
│  └─────────────────┘                                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Data Models

### 1. Conversation Table

Stores conversation metadata and session information.

**Table Name:** `agent_conversation`

| Column            | Type        | Nullable | Index  | Description                         |
| ----------------- | ----------- | -------- | ------ | ----------------------------------- |
| `id`              | ID          | No       | PK     | Auto-increment primary key          |
| `conversation_id` | string(64)  | No       | Unique | Unique conversation identifier      |
| `title`           | string(500) | Yes      | -      | Conversation title                  |
| `assistant_id`    | string(200) | No       | Yes    | Associated assistant ID             |
| `user_id`         | string(200) | No       | Yes    | Owner user ID                       |
| `team_id`         | string(200) | Yes      | Yes    | Team ID for access control          |
| `mode`            | string(50)  | No       | -      | Conversation mode (default: "chat") |
| `status`          | enum        | No       | Yes    | Status: `active`, `archived`        |
| `last_message_at` | timestamp   | Yes      | Yes    | Timestamp of last message           |
| `metadata`        | json        | Yes      | -      | Additional metadata                 |
| `created_at`      | timestamp   | No       | Yes    | Creation timestamp                  |
| `updated_at`      | timestamp   | No       | -      | Last update timestamp               |

**Indexes:**

| Name                 | Columns             | Type  |
| -------------------- | ------------------- | ----- |
| `idx_conv_user`      | `user_id`, `status` | index |
| `idx_conv_team`      | `team_id`, `status` | index |
| `idx_conv_assistant` | `assistant_id`      | index |
| `idx_conv_last_msg`  | `last_message_at`   | index |

### 2. Message Table

Stores user-visible messages (both user input and assistant responses).

**Table Name:** `agent_message`

| Column            | Type        | Nullable | Index  | Description                               |
| ----------------- | ----------- | -------- | ------ | ----------------------------------------- |
| `id`              | ID          | No       | PK     | Auto-increment primary key                |
| `message_id`      | string(64)  | No       | Unique | Unique message identifier                 |
| `conversation_id` | string(64)  | No       | Yes    | Parent conversation ID                    |
| `request_id`      | string(64)  | Yes      | Yes    | Request ID for grouping                   |
| `role`            | enum        | No       | Yes    | Role: `user`, `assistant`                 |
| `type`            | string(50)  | No       | -      | Message type (text, image, loading, etc.) |
| `props`           | json        | No       | -      | Message properties (content, url, etc.)   |
| `block_id`        | string(64)  | Yes      | Yes    | Block grouping ID                         |
| `thread_id`       | string(64)  | Yes      | Yes    | Thread grouping ID                        |
| `assistant_id`    | string(200) | Yes      | Yes    | Assistant ID (join to get name/avatar)    |
| `sequence`        | integer     | No       | Yes    | Message order within conversation         |
| `metadata`        | json        | Yes      | -      | Additional metadata                       |
| `created_at`      | timestamp   | No       | Yes    | Creation timestamp                        |
| `updated_at`      | timestamp   | No       | -      | Last update timestamp                     |

**Indexes:**

| Name                | Columns                       | Type  |
| ------------------- | ----------------------------- | ----- |
| `idx_msg_conv_seq`  | `conversation_id`, `sequence` | index |
| `idx_msg_request`   | `request_id`                  | index |
| `idx_msg_block`     | `block_id`                    | index |
| `idx_msg_assistant` | `assistant_id`                | index |

**Message Types:**

| Type      | Description       | Props Example                                     |
| --------- | ----------------- | ------------------------------------------------- |
| `text`    | Text message      | `{"content": "Hello world"}`                      |
| `image`   | Image message     | `{"url": "...", "alt": "...", "caption": "..."}`  |
| `loading` | Loading indicator | `{"message": "Processing...", "done": false}`     |
| `error`   | Error message     | `{"message": "...", "code": "..."}`               |
| `action`  | Action buttons    | `{"buttons": [...]}`                              |
| `file`    | File attachment   | `{"url": "...", "filename": "...", "size": 1024}` |

### 3. Step Table

Stores execution steps for resume/retry functionality.

**Table Name:** `agent_step`

| Column            | Type        | Nullable | Index  | Description                      |
| ----------------- | ----------- | -------- | ------ | -------------------------------- |
| `id`              | ID          | No       | PK     | Auto-increment primary key       |
| `step_id`         | string(64)  | No       | Unique | Unique step identifier           |
| `conversation_id` | string(64)  | No       | Yes    | Parent conversation ID           |
| `request_id`      | string(64)  | No       | Yes    | Request ID                       |
| `assistant_id`    | string(200) | No       | Yes    | Assistant executing this step    |
| `stack_id`        | string(64)  | No       | Yes    | Stack node ID for this execution |
| `stack_parent_id` | string(64)  | Yes      | Yes    | Parent stack ID (for A2A calls)  |
| `stack_depth`     | integer     | No       | -      | Call depth (0=root, 1+=nested)   |
| `type`            | enum        | No       | Yes    | Step type                        |
| `status`          | enum        | No       | Yes    | Step status                      |
| `input`           | json        | Yes      | -      | Step input data                  |
| `output`          | json        | Yes      | -      | Step output data                 |
| `error`           | text        | Yes      | -      | Error message if failed          |
| `sequence`        | integer     | No       | Yes    | Step order within request        |
| `metadata`        | json        | Yes      | -      | Additional metadata              |
| `created_at`      | timestamp   | No       | Yes    | Creation timestamp               |
| `updated_at`      | timestamp   | No       | -      | Last update timestamp            |

**Step Types:**

| Type          | Description           | Input                  | Output                                |
| ------------- | --------------------- | ---------------------- | ------------------------------------- |
| `input`       | User input received   | `{messages: [...]}`    | -                                     |
| `hook_create` | Create hook execution | `{messages: [...]}`    | `{messages: [...], ...}`              |
| `llm`         | LLM completion call   | `{messages: [...]}`    | `{content: "...", tool_calls: [...]}` |
| `tool`        | Tool/MCP execution    | `{server, tool, args}` | `{result: ...}`                       |
| `hook_next`   | Next hook execution   | `{completion, tools}`  | `{data: ...}`                         |
| `delegate`    | A2A delegation        | `{agent_id, messages}` | `{response: ...}`                     |

**Step Status:**

| Status        | Description           | Can Resume     |
| ------------- | --------------------- | -------------- |
| `pending`     | Not started           | Yes            |
| `running`     | In progress           | Yes (restart)  |
| `completed`   | Finished successfully | No             |
| `failed`      | Failed with error     | Yes (retry)    |
| `interrupted` | User interrupted      | Yes (continue) |

**Indexes:**

| Name                 | Columns                  | Type  |
| -------------------- | ------------------------ | ----- |
| `idx_step_conv`      | `conversation_id`        | index |
| `idx_step_request`   | `request_id`, `sequence` | index |
| `idx_step_status`    | `status`                 | index |
| `idx_step_stack`     | `stack_id`               | index |
| `idx_step_parent`    | `stack_parent_id`        | index |
| `idx_step_assistant` | `assistant_id`           | index |

## Write Strategy

### Two-Write Strategy

All data is buffered in memory during execution and written to database only **twice**:

1. **Write 1 (Entry)**: When `Stream()` starts - save user input message
2. **Write 2 (Exit)**: When `Stream()` exits - batch save all assistant messages and steps

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
        - Batch write all assistant messages
        - Batch write all steps (with final status)
        - Update token usage in openapi_request (via request_id)
```

### Write Points

| Event            | Message Table        | Step Table                                |
| ---------------- | -------------------- | ----------------------------------------- |
| Stream entry     | Write 1 (user input) | -                                         |
| During execution | Buffer in memory     | Buffer in memory                          |
| **Stream exit**  | **Batch write all**  | **Batch write all (status=completed)**    |
| On interrupt     | Batch write buffered | Batch write buffered (status=interrupted) |
| On error         | Batch write buffered | Batch write buffered (status=failed)      |

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

// createStep creates a step with context information
func createStep(ctx *Context, stepType, status string, input, output interface{}) *Step {
    return &Step{
        StepID:         generateID(),
        ConversationID: ctx.ChatID,       // ChatID = conversation_id
        RequestID:      ctx.RequestID,    // From OpenAPI middleware
        AssistantID:    ctx.AssistantID,
        StackID:        ctx.Stack.ID,
        StackParentID:  ctx.Stack.ParentID,
        StackDepth:     ctx.Stack.Depth,
        Type:           stepType,
        Status:         status,
        Input:          input,
        Output:         output,
        Sequence:       nextSequence(),
    }
}

## API Interface

### ChatStore Interface

```go
// ChatStore defines the chat storage interface
type ChatStore interface {
    // Conversation Management
    CreateConversation(conv *Conversation) error
    GetConversation(conversationID string) (*Conversation, error)
    UpdateConversation(conversationID string, updates map[string]interface{}) error
    DeleteConversation(conversationID string) error
    ListConversations(filter ConversationFilter) (*ConversationList, error)

    // Message Management
    SaveMessages(conversationID string, messages []*Message) error
    GetMessages(conversationID string, filter MessageFilter) ([]*Message, error)
    UpdateMessage(messageID string, updates map[string]interface{}) error
    DeleteMessages(conversationID string, messageIDs []string) error

    // Step Management
    SaveStep(step *Step) error
    UpdateStep(stepID string, updates map[string]interface{}) error
    GetSteps(requestID string) ([]*Step, error)
    GetLastIncompleteStep(conversationID string) (*Step, error)
}
````

### Data Structures

```go
// Conversation represents a chat conversation
type Conversation struct {
    ConversationID string                 `json:"conversation_id"`
    Title          string                 `json:"title,omitempty"`
    AssistantID    string                 `json:"assistant_id"`
    UserID         string                 `json:"user_id"`
    TeamID         string                 `json:"team_id,omitempty"`
    Mode           string                 `json:"mode"`
    Status         string                 `json:"status"`
    LastMessageAt  *time.Time             `json:"last_message_at,omitempty"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
}

// Message represents a chat message
type Message struct {
    MessageID      string                 `json:"message_id"`
    ConversationID string                 `json:"conversation_id"`
    RequestID      string                 `json:"request_id,omitempty"`
    Role           string                 `json:"role"`
    Type           string                 `json:"type"`
    Props          map[string]interface{} `json:"props"`
    BlockID        string                 `json:"block_id,omitempty"`
    ThreadID       string                 `json:"thread_id,omitempty"`
    AssistantID    string                 `json:"assistant_id,omitempty"`
    Sequence       int                    `json:"sequence"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
}

// Step represents an execution step
type Step struct {
    StepID         string                 `json:"step_id"`
    ConversationID string                 `json:"conversation_id"`
    RequestID      string                 `json:"request_id"`
    AssistantID    string                 `json:"assistant_id"`
    StackID        string                 `json:"stack_id"`
    StackParentID  string                 `json:"stack_parent_id,omitempty"`
    StackDepth     int                    `json:"stack_depth"`
    Type           string                 `json:"type"`
    Status         string                 `json:"status"`
    Input          map[string]interface{} `json:"input,omitempty"`
    Output         map[string]interface{} `json:"output,omitempty"`
    Error          string                 `json:"error,omitempty"`
    Sequence       int                    `json:"sequence"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
}
```

### Filter Structures

```go
// ConversationFilter for listing conversations
type ConversationFilter struct {
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

// ConversationList paginated response
type ConversationList struct {
    Data      []*Conversation `json:"data"`
    Page      int             `json:"page"`
    PageSize  int             `json:"pagesize"`
    PageCount int             `json:"pagecount"`
    Total     int             `json:"total"`
}
```

## Usage Examples

### 1. Normal Request Flow

See [Write Strategy - Implementation](#implementation) for the complete flow with two-write strategy.

### 2. Load Chat History

```go
// Get conversation list
convs, _ := chatStore.ListConversations(ConversationFilter{
    UserID:   "user123",
    Status:   "active",
    Page:     1,
    PageSize: 20,
})

// Get messages for a conversation
messages, _ := chatStore.GetMessages("conv_123", MessageFilter{
    Limit: 100,
})

// Return to frontend
return map[string]interface{}{
    "conversation": conv,
    "messages":     messages,
}
```

### 3. Resume from Interruption

```go
func (ast *Assistant) Resume(ctx *Context) error {
    // 1. Find last incomplete step
    step, _ := chatStore.GetLastIncompleteStep(ctx.ConversationID)
    if step == nil {
        return nil // Nothing to resume
    }

    // 2. Check if this is an A2A nested call
    if step.StackDepth > 0 {
        // Need to rebuild the call stack
        return ast.ResumeNestedCall(ctx, step)
    }

    // 3. Resume based on step type
    switch step.Type {
    case "llm":
        // Re-execute LLM call with saved input
        messages := step.Input["messages"].([]Message)
        return ast.executeLLMStream(ctx, messages, ...)

    case "tool":
        // Retry tool call
        return ast.retryToolCall(ctx, step)

    case "hook_next":
        // Re-execute hook
        return ast.executeHookNext(ctx, step.Input)
    }

    return nil
}
```

### 4. Resume A2A Nested Calls

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

### 4. Handle Interruption

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
┌─────┬─────────────┬─────────────┬──────────┬────────────┬───────┬─────────────┐
│ seq │ assistant   │ stack_id    │ parent   │ depth      │ type  │ status      │
├─────┼─────────────┼─────────────┼──────────┼────────────┼───────┼─────────────┤
│  1  │ analyzer    │ stk_001     │ null     │ 0          │ input │ completed   │
│  2  │ analyzer    │ stk_001     │ null     │ 0          │ llm   │ completed   │
│  3  │ analyzer    │ stk_001     │ null     │ 0          │ delegate │ running  │ ← delegating
│  4  │ visualizer  │ stk_002     │ stk_001  │ 1          │ input │ completed   │
│  5  │ visualizer  │ stk_002     │ stk_001  │ 1          │ llm   │ interrupted │ ← interrupted here
└─────┴─────────────┴─────────────┴──────────┴────────────┴───────┴─────────────┘

Resume Flow:
1. Find step with status="interrupted" → step 5
2. Check stack_depth=1 → nested call
3. Get stack path: [stk_001, stk_002]
4. Resume visualizer assistant with step 5's input
5. When visualizer completes, update step 3 (delegate) to completed
```

## Migration Notes

### From Old Schema

The old `agent_history` and `agent_chat` tables are replaced by:

| Old Table       | New Table            | Notes                                                  |
| --------------- | -------------------- | ------------------------------------------------------ |
| `agent_chat`    | `agent_conversation` | Similar structure, added `mode`, `metadata`            |
| `agent_history` | `agent_message`      | Changed to store `type`/`props` instead of raw content |
| -               | `agent_step`         | New table for execution tracking                       |

### Data Migration

```sql
-- Migrate conversations
INSERT INTO agent_conversation (conversation_id, title, assistant_id, ...)
SELECT chat_id, title, assistant_id, ...
FROM agent_chat;

-- Migrate messages (simplified, actual migration needs content transformation)
INSERT INTO agent_message (message_id, conversation_id, role, type, props, ...)
SELECT id, chat_id, role, 'text', JSON_OBJECT('content', content), ...
FROM agent_history;
```

## Related Documents

- [OpenAPI Request Design](../../openapi/request/REQUEST_DESIGN.md) - Global request tracking, billing, rate limiting
- [Trace Module](../../trace/README.md) - Detailed execution tracing for debugging
- [Agent Context](../context/README.md) - Context and message handling
