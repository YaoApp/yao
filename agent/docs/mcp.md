# MCP Integration

Model Context Protocol (MCP) enables tool integration with external services.

## Defining MCP Servers

Create `mcps/tools.mcp.yao` in the assistant directory:

```json
{
  "name": "Tools",
  "description": "Custom tools for the assistant",
  "transport": "process",
  "process": {
    "command": "node",
    "args": ["server.js"]
  }
}
```

### Transport Types

**Process (Local)**

```json
{
  "transport": "process",
  "process": {
    "command": "python",
    "args": ["mcp_server.py"],
    "env": { "API_KEY": "$ENV.API_KEY" }
  }
}
```

**HTTP**

```json
{
  "transport": "http",
  "http": {
    "url": "https://mcp.example.com",
    "headers": { "Authorization": "Bearer $ENV.TOKEN" }
  }
}
```

**SSE (Server-Sent Events)**

```json
{
  "transport": "sse",
  "sse": {
    "url": "https://mcp.example.com/events"
  }
}
```

## Configuring in package.yao

### All Tools

```json
{
  "mcp": {
    "servers": ["tools"]
  }
}
```

### Specific Tools

```json
{
  "mcp": {
    "servers": [
      { "server_id": "tools", "tools": ["search", "calculate"] }
    ]
  }
}
```

### With Resources

```json
{
  "mcp": {
    "servers": [
      {
        "server_id": "data",
        "tools": ["query"],
        "resources": ["data://users/*"]
      }
    ]
  }
}
```

## Dynamic Configuration in Hooks

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  return {
    messages,
    mcp_servers: [
      { server_id: "tools", tools: ["search"] },
      { server_id: "data", resources: ["data://reports"] }
    ]
  };
}
```

## Using MCP in Hooks

### List Available Tools

```typescript
const tools = ctx.mcp.ListTools("server-id");
// { tools: [{ name: "search", description: "...", inputSchema: {...} }] }
```

### Call Tool

```typescript
const result = ctx.mcp.CallTool("server-id", "search", {
  query: "example",
  limit: 10
});
// { content: [{ type: "text", text: "..." }] }
```

### Batch Tool Calls

```typescript
// Sequential
const results = ctx.mcp.CallTools("server-id", [
  { name: "step1", arguments: { input: "a" } },
  { name: "step2", arguments: { input: "b" } }
]);

// Parallel
const results = ctx.mcp.CallToolsParallel("server-id", [
  { name: "api1", arguments: {} },
  { name: "api2", arguments: {} }
]);
```

### Read Resources

```typescript
const resources = ctx.mcp.ListResources("server-id");
const data = ctx.mcp.ReadResource("server-id", "data://users/123");
```

### Get Prompts

```typescript
const prompts = ctx.mcp.ListPrompts("server-id");
const prompt = ctx.mcp.GetPrompt("server-id", "system", { role: "helper" });
```

## Tool Mapping

Map MCP tools to Yao processes:

```
mcps/
└── mapping/
    └── tools/
        ├── search.yao
        └── calculate.yao
```

**mapping/tools/search.yao**

```json
{
  "process": "scripts.search.Execute",
  "args": ["{{input}}"]
}
```

## Error Handling

```typescript
function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  const { tools } = payload;

  if (tools) {
    for (const tool of tools) {
      if (tool.error) {
        ctx.trace.Error(`Tool ${tool.tool} failed: ${tool.error}`);
        // Handle error
      } else {
        // Process result
        console.log(tool.result);
      }
    }
  }

  return null;
}
```

## Complete Example

**mcps/calculator.mcp.yao**

```json
{
  "name": "Calculator",
  "description": "Math operations",
  "transport": "process",
  "process": {
    "command": "node",
    "args": ["calc-server.js"]
  }
}
```

**package.yao**

```json
{
  "name": "Math Assistant",
  "connector": "gpt-4o",
  "mcp": {
    "servers": [
      { "server_id": "agents.assistant.calculator", "tools": ["add", "multiply"] }
    ]
  }
}
```

**src/index.ts**

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  // Check if calculation is needed
  const query = messages[messages.length - 1]?.content || "";
  if (/\d+\s*[\+\-\*\/]\s*\d+/.test(query)) {
    // Enable calculator
    return {
      messages,
      mcp_servers: [{ server_id: "agents.assistant.calculator" }]
    };
  }
  return { messages };
}

function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  const { tools } = payload;

  if (tools?.length > 0) {
    const calcResult = tools.find(t => t.server.includes("calculator"));
    if (calcResult?.result) {
      return {
        data: {
          answer: calcResult.result,
          expression: calcResult.arguments
        }
      };
    }
  }

  return null;
}
```
