# Hooks

Hooks allow you to customize agent behavior at key points in the execution lifecycle.

## Lifecycle

```
User Input → Create Hook → LLM Call → Tool Execution → Next Hook → Response
```

## Create Hook

Called before LLM call. Use to preprocess messages, configure request, or delegate.

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  // Return null for default behavior
  return null;

  // Or return configuration
  return {
    messages,                    // Modified messages
    temperature: 0.7,            // Override temperature
    max_tokens: 2000,            // Override max tokens
    connector: "gpt-4o-mini",    // Override connector
    prompt_preset: "task",       // Select prompt preset
    disable_global_prompts: true,// Skip global prompts
    mcp_servers: [               // Add MCP servers
      { server_id: "tools", tools: ["search"] }
    ],
    uses: {                      // Override wrapper tools
      vision: "vision-agent",
      search: "disabled"
    },
    force_uses: true,            // Force use wrapper tools
    locale: "zh-cn",             // Override locale
    metadata: { key: "value" },  // Pass data to context
  };
}
```

### Delegation (Skip LLM)

Route to another agent immediately:

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  if (shouldDelegate(messages)) {
    return {
      delegate: {
        agent_id: "specialist.agent",
        messages: messages,
        options: { metadata: { source: "main" } }
      }
    };
  }
  return { messages };
}
```

## Next Hook

Called after LLM response and tool execution. Use to post-process or delegate.

```typescript
function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  const { messages, completion, tools, error } = payload;

  // Handle errors
  if (error) {
    return { data: { status: "error", message: error } };
  }

  // Process tool results
  if (tools?.length > 0) {
    const results = tools.map(t => t.result);
    return { data: { status: "success", results } };
  }

  // Delegate based on response
  if (completion?.content?.includes("transfer")) {
    return {
      delegate: {
        agent_id: "transfer.agent",
        messages: payload.messages
      }
    };
  }

  // Return null for standard response
  return null;
}
```

### Payload Structure

```typescript
interface Payload {
  messages: Message[];           // Messages sent to LLM
  completion?: {
    content: string;             // LLM text response
    tool_calls?: ToolCall[];     // Tool calls from LLM
    usage?: UsageInfo;           // Token usage
  };
  tools?: ToolCallResponse[];    // Tool execution results
  error?: string;                // Error message
}

interface ToolCallResponse {
  toolcall_id: string;
  server: string;                // MCP server ID
  tool: string;                  // Tool name
  arguments?: any;               // Tool arguments
  result?: any;                  // Tool result
  error?: string;                // Tool error
}
```

### Return Values

```typescript
interface NextResponse {
  delegate?: {                   // Route to another agent
    agent_id: string;
    messages: Message[];
    options?: Record<string, any>;
  };
  data?: any;                    // Custom response data
  metadata?: Record<string, any>;// Debug metadata
}
```

## Sending Messages

Use `ctx` to send messages to the client:

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  // Send complete message
  ctx.Send({ type: "text", props: { content: "Processing..." } });

  // Streaming message
  const msgId = ctx.SendStream("Starting...");
  ctx.Append(msgId, " step 1...");
  ctx.Append(msgId, " step 2...");
  ctx.End(msgId);

  return { messages };
}
```

## Memory

Share data between hooks:

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  // Store in request-scoped memory
  ctx.memory.context.Set("start_time", Date.now());
  ctx.memory.context.Set("query", messages[0]?.content);

  return { messages };
}

function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  // Retrieve data
  const startTime = ctx.memory.context.Get("start_time");
  const duration = Date.now() - startTime;

  return { data: { duration_ms: duration } };
}
```

## Tracing

Add trace nodes for debugging and UI:

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const node = ctx.trace.Add(
    { query: messages[0]?.content },
    { label: "Preprocessing", type: "process", icon: "play" }
  );

  node.Info("Starting analysis");
  // ... processing ...
  node.Complete({ status: "done" });

  return { messages };
}
```

## Error Handling

```typescript
function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  try {
    if (payload.error) {
      ctx.trace.Error(payload.error);
      return {
        data: { status: "error", message: "Something went wrong" }
      };
    }
    // ... normal processing
  } catch (e) {
    ctx.trace.Error(e.message);
    return { data: { status: "error", message: e.message } };
  }
}
```

## Multi-Agent Orchestration

```typescript
// Main agent delegates based on intent
function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  const { tools } = payload;

  // Route based on tool result
  const intent = tools?.[0]?.result?.intent;
  const agentMap = {
    "search": "search.agent",
    "calculate": "calc.agent",
    "translate": "translate.agent"
  };

  if (intent && agentMap[intent]) {
    return {
      delegate: {
        agent_id: agentMap[intent],
        messages: payload.messages
      }
    };
  }

  return null;
}
```

## Complete Example

```typescript
import { agent } from "@yao/runtime";

function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const query = messages[messages.length - 1]?.content || "";
  
  // Store for Next hook
  ctx.memory.context.Set("query", query);
  ctx.memory.context.Set("start", Date.now());

  // Add trace
  ctx.trace.Add({ query }, { label: "Create", type: "hook" });

  // Check if needs special handling
  if (query.toLowerCase().includes("urgent")) {
    return {
      messages,
      temperature: 0,
      prompt_preset: "task"
    };
  }

  return { messages };
}

function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  const { completion, tools, error } = payload;
  const start = ctx.memory.context.Get("start");
  const duration = Date.now() - start;

  ctx.trace.Add(
    { duration },
    { label: "Next", type: "hook" }
  ).Complete();

  if (error) {
    return { data: { status: "error", error } };
  }

  if (tools?.length > 0) {
    return {
      data: {
        status: "success",
        response: completion?.content,
        tools: tools.map(t => ({ name: t.tool, result: t.result })),
        duration_ms: duration
      }
    };
  }

  return null;
}
```
