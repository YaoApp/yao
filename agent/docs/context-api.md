# Context API

The `ctx` object provides access to messaging, memory, tracing, and MCP operations.

## Properties

```typescript
interface Context {
  chat_id: string;           // Chat session ID
  assistant_id: string;      // Assistant ID
  locale: string;            // User locale (e.g., "en-us")
  theme: string;             // UI theme
  route: string;             // Request route
  referer: string;           // Request source
  metadata: Record<string, any>;    // Custom metadata
  authorized: Record<string, any>;  // Auth info

  memory: Memory;            // Memory namespaces
  trace: Trace;              // Tracing API
  mcp: MCP;                  // MCP operations
  search: Search;            // Search API
  agent: Agent;              // Agent-to-Agent calls (A2A)
  llm: LLM;                  // Direct LLM calls
}
```

## Messaging

### Send Complete Message

```typescript
ctx.Send({ type: "text", props: { content: "Hello!" } });
ctx.Send("Hello!");  // Shorthand for text
```

### Streaming Messages

```typescript
const msgId = ctx.SendStream("Starting...");
ctx.Append(msgId, " processing...");
ctx.Append(msgId, " done!");
ctx.End(msgId);
```

### Update Streaming Message

```typescript
const msgId = ctx.SendStream({ type: "loading", props: { message: "Loading..." } });
// ... do work ...
ctx.Replace(msgId, { type: "text", props: { content: "Complete!" } });
ctx.End(msgId);
```

### Merge Data

```typescript
const msgId = ctx.SendStream({ type: "status", props: { progress: 0 } });
ctx.Merge(msgId, { progress: 50 }, "props");
ctx.Merge(msgId, { progress: 100, status: "done" }, "props");
ctx.End(msgId);
```

### Set Field

```typescript
const msgId = ctx.SendStream({ type: "result", props: {} });
ctx.Set(msgId, "success", "props.status");
ctx.Set(msgId, { count: 10 }, "props.data");
ctx.End(msgId);
```

### Block Grouping

```typescript
const blockId = ctx.BlockID();
ctx.Send("Step 1", blockId);
ctx.Send("Step 2", blockId);
ctx.Send("Step 3", blockId);
ctx.EndBlock(blockId);
```

### ID Generators

```typescript
const msgId = ctx.MessageID();    // "M1", "M2", ...
const blockId = ctx.BlockID();    // "B1", "B2", ...
const threadId = ctx.ThreadID();  // "T1", "T2", ...
```

## Memory

Four-level hierarchical memory system:

| Namespace            | Scope        | Persistence |
| -------------------- | ------------ | ----------- |
| `ctx.memory.user`    | Per user     | Persistent  |
| `ctx.memory.team`    | Per team     | Persistent  |
| `ctx.memory.chat`    | Per chat     | Persistent  |
| `ctx.memory.context` | Per request  | Temporary   |

### Basic Operations

```typescript
// Get/Set
ctx.memory.user.Set("theme", "dark");
const theme = ctx.memory.user.Get("theme");

// With TTL (seconds)
ctx.memory.context.Set("temp", data, 300);

// Check/Delete
if (ctx.memory.chat.Has("topic")) {
  ctx.memory.chat.Del("topic");
}

// Get and delete atomically
const token = ctx.memory.context.GetDel("one_time_token");

// Collection operations
const keys = ctx.memory.user.Keys();
const count = ctx.memory.chat.Len();
ctx.memory.context.Clear();
```

### Counters

```typescript
const views = ctx.memory.user.Incr("page_views");
const credits = ctx.memory.user.Decr("credits", 5);
```

### Lists

```typescript
ctx.memory.chat.Push("history", [msg1, msg2]);
const last = ctx.memory.chat.Pop("queue");
const items = ctx.memory.chat.Pull("queue", 5);
const all = ctx.memory.chat.PullAll("queue");
```

### Sets

```typescript
ctx.memory.user.AddToSet("visited", ["/home", "/about"]);
```

### Array Access

```typescript
const len = ctx.memory.chat.ArrayLen("messages");
const first = ctx.memory.chat.ArrayGet("messages", 0);
const last = ctx.memory.chat.ArrayGet("messages", -1);
ctx.memory.chat.ArraySet("messages", 0, newMsg);
const slice = ctx.memory.chat.ArraySlice("messages", -10, -1);
const page = ctx.memory.chat.ArrayPage("messages", 1, 20);
const all = ctx.memory.chat.ArrayAll("messages");
```

## Trace

### Create Nodes

```typescript
const node = ctx.trace.Add(
  { query: "input data" },
  {
    label: "Processing",
    type: "process",
    icon: "play",
    description: "Processing user request"
  }
);
```

### Logging

```typescript
ctx.trace.Info("Starting process");
ctx.trace.Debug("Variable: " + value);
ctx.trace.Warn("Deprecated feature");
ctx.trace.Error("Operation failed");

// Or on node
node.Info("Step completed");
```

### Node Lifecycle

```typescript
node.SetOutput({ result: data });
node.SetMetadata("duration", 1500);
node.Complete({ status: "done" });
// or
node.Fail("Error message");
```

### Parallel Nodes

```typescript
const nodes = ctx.trace.Parallel([
  { input: { url: "api1" }, option: { label: "API 1" } },
  { input: { url: "api2" }, option: { label: "API 2" } }
]);
```

### Child Nodes

```typescript
const parent = ctx.trace.Add({}, { label: "Parent" });
const child = parent.Add({}, { label: "Child" });
```

## MCP

### Tools

```typescript
// List tools
const tools = ctx.mcp.ListTools("server-id");

// Call single tool - returns parsed result directly
const result = ctx.mcp.CallTool("server-id", "tool-name", { arg: "value" });
console.log(result.field);  // Direct access to parsed data

// Call multiple sequentially - returns array of parsed results
const results = ctx.mcp.CallTools("server-id", [
  { name: "tool1", arguments: { a: 1 } },
  { name: "tool2", arguments: { b: 2 } }
]);
results.forEach(r => console.log(r));

// Call multiple in parallel - returns array of parsed results
const results = ctx.mcp.CallToolsParallel("server-id", [
  { name: "tool1", arguments: {} },
  { name: "tool2", arguments: {} }
]);
results.forEach(r => console.log(r));
```

### Cross-Server Tool Calls

```typescript
// Call tools across multiple MCP servers (like Promise.all)
const results = ctx.mcp.All([
  { mcp: "server1", tool: "search", arguments: { q: "query" } },
  { mcp: "server2", tool: "fetch", arguments: { id: 123 } }
]);

// First success wins (like Promise.any)
const results = ctx.mcp.Any([
  { mcp: "primary", tool: "search", arguments: { q: "query" } },
  { mcp: "backup", tool: "search", arguments: { q: "query" } }
]);

// First complete wins (like Promise.race)
const results = ctx.mcp.Race([
  { mcp: "region-us", tool: "ping", arguments: {} },
  { mcp: "region-eu", tool: "ping", arguments: {} }
]);

// Result structure
interface MCPToolResult {
  mcp: string;      // Server ID
  tool: string;     // Tool name
  result?: any;     // Parsed result content
  error?: string;   // Error if failed
}
```

### Resources

```typescript
const resources = ctx.mcp.ListResources("server-id");
const data = ctx.mcp.ReadResource("server-id", "resource://uri");
```

### Prompts

```typescript
const prompts = ctx.mcp.ListPrompts("server-id");
const prompt = ctx.mcp.GetPrompt("server-id", "prompt-name", { arg: "value" });
```

## Search

### Single Search

```typescript
// Web search
const webResult = ctx.search.Web("query", {
  limit: 10,
  sites: ["example.com"],
  time_range: "week"
});

// Knowledge base
const kbResult = ctx.search.KB("query", {
  collections: ["docs"],
  threshold: 0.7,
  graph: true
});

// Database
const dbResult = ctx.search.DB("query", {
  models: ["model.name"],
  wheres: [{ column: "status", value: "active" }],
  limit: 20
});
```

### Parallel Search

```typescript
// Wait for all
const results = ctx.search.All([
  { type: "web", query: "topic" },
  { type: "kb", query: "topic", collections: ["docs"] }
]);

// First success
const results = ctx.search.Any([
  { type: "web", query: "topic" },
  { type: "kb", query: "topic" }
]);

// First complete
const results = ctx.search.Race([
  { type: "web", query: "topic" },
  { type: "kb", query: "topic" }
]);
```

### Result Structure

```typescript
interface SearchResult {
  type: "web" | "kb" | "db";
  query: string;
  source: "hook" | "auto" | "user";
  items: {
    citation_id: string;
    title: string;
    url: string;
    content: string;
    score: number;
  }[];
  error?: string;
}
```

## Agent API

The `ctx.agent` object provides methods to call other agents from within hooks, enabling agent-to-agent communication (A2A).

### Single Agent Call

```typescript
// Basic call
const result = ctx.agent.Call("assistant-id", messages);

// With options and callback
const result = ctx.agent.Call("assistant-id", messages, {
  connector: "gpt-4o",
  mode: "chat",
  metadata: { source: "hook" },
  skip: { history: false, trace: false, output: false },
  onChunk: (msg) => {
    console.log("Received:", msg.type, msg.props);
    return 0; // 0 = continue, non-zero = stop
  }
});
```

### Agent Options

```typescript
interface AgentCallOptions {
  connector?: string;            // Override LLM connector
  mode?: string;                 // Agent mode ("chat", "task")
  metadata?: Record<string, any>; // Custom metadata passed to hooks
  skip?: {
    history?: boolean;           // Skip loading chat history
    trace?: boolean;             // Skip trace recording
    output?: boolean;            // Skip output to client
    keyword?: boolean;           // Skip keyword extraction
    search?: boolean;            // Skip search
    content_parsing?: boolean;   // Skip content parsing
  };
  onChunk?: (msg: Message) => number; // Callback (0=continue, non-zero=stop)
}
```

### Parallel Agent Calls

```typescript
// Wait for all agents to complete (like Promise.all)
const results = ctx.agent.All([
  { agent: "agent-1", messages: [...] },
  { agent: "agent-2", messages: [...] }
]);

// Return first successful result (like Promise.any)
const results = ctx.agent.Any([
  { agent: "agent-1", messages: [...] },
  { agent: "agent-2", messages: [...] }
]);

// Return first completed result (like Promise.race)
const results = ctx.agent.Race([
  { agent: "agent-1", messages: [...] },
  { agent: "agent-2", messages: [...] }
]);

// With global callback for all responses
const results = ctx.agent.All([
  { agent: "agent-1", messages: [...] },
  { agent: "agent-2", messages: [...] }
], {
  onChunk: (agentId, index, msg) => {
    console.log(`Agent ${agentId} [${index}]:`, msg.type);
    return 0;
  }
});
```

### Result Structure

```typescript
interface AgentResult {
  agent_id: string;
  response?: Response;
  content?: string;
  error?: string;
}
```

### Message Object (onChunk callback)

```typescript
interface Message {
  type: string;                // "text", "thinking", "tool_call", "error"
  props?: Record<string, any>; // e.g., { content: "Hello" }
  chunk_id?: string;           // C1, C2, ...
  message_id?: string;         // M1, M2, ...
  delta?: boolean;             // Incremental update flag
}
```

## LLM API

The `ctx.llm` object provides direct access to LLM connectors for streaming completions.

### Single LLM Call

```typescript
// Basic streaming call
const result = ctx.llm.Stream("gpt-4o", [
  { role: "user", content: "Hello" }
]);

// With options and callback
const result = ctx.llm.Stream("gpt-4o", messages, {
  temperature: 0.7,
  max_tokens: 2000,
  onChunk: (msg) => {
    console.log("Chunk:", msg.props?.content);
    return 0;
  }
});
```

### Parallel LLM Calls

```typescript
// Wait for all LLM calls (like Promise.all)
const results = ctx.llm.All([
  { connector: "gpt-4o", messages: [...] },
  { connector: "claude-3", messages: [...] }
]);

// Return first successful result (like Promise.any)
const results = ctx.llm.Any([
  { connector: "gpt-4o", messages: [...] },
  { connector: "claude-3", messages: [...] }
]);

// Return first completed result (like Promise.race)
const results = ctx.llm.Race([
  { connector: "gpt-4o", messages: [...] },
  { connector: "claude-3", messages: [...] }
]);

// With global callback
const results = ctx.llm.All([
  { connector: "gpt-4o", messages: [...] },
  { connector: "claude-3", messages: [...] }
], {
  onChunk: (connectorId, index, msg) => {
    console.log(`LLM ${connectorId} [${index}]:`, msg.type);
    return 0;
  }
});
```

### LLM Options

```typescript
interface LlmOptions {
  temperature?: number;
  max_tokens?: number;
  max_completion_tokens?: number;
  top_p?: number;
  presence_penalty?: number;
  frequency_penalty?: number;
  stop?: string | string[];
  user?: string;
  seed?: number;
  tools?: object[];
  tool_choice?: string | object;
  response_format?: { type: string; json_schema?: object };
  reasoning_effort?: string;
  onChunk?: (msg: Message) => number;
}
```

### Result Structure

```typescript
interface LlmResult {
  connector: string;
  response?: CompletionResponse;
  content?: string;
  error?: string;
}
