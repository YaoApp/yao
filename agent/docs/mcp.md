# MCP Integration

Model Context Protocol (MCP) enables tool integration with external services.

## Directory Structure

Assistants can define their own namespaced MCP servers in the `mcps/` directory:

```
assistants/
└── my-assistant/
    ├── package.yao
    └── mcps/
        ├── tools.mcp.yao           # → agents.my-assistant.tools
        ├── calculator.mcp.yao      # → agents.my-assistant.calculator
        └── mapping/
            └── tools/
                └── schemes/
                    ├── search.in.yao
                    └── search.out.yao
```

MCP servers are automatically loaded with `agents.<assistant-id>.` prefix.

## Defining MCP Servers

Create `mcps/tools.mcp.yao` in the assistant directory:

```json
{
  "label": "Tools",
  "description": "Custom tools for the assistant",
  "transport": "process",
  "tools": {
    "search": "scripts.tools.Search",
    "create": "models.data.Create"
  }
}
```

### Transport Types

**Process (Yao Internal)**

Map Yao Processes directly to MCP tools:

```json
{
  "transport": "process",
  "tools": {
    "search": "models.data.Paginate",
    "create": "models.data.Create"
  },
  "resources": {
    "detail": "models.data.Find"
  }
}
```

**STDIO (Local Server)**

```json
{
  "transport": "stdio",
  "command": "python",
  "arguments": ["mcp_server.py"],
  "env": { "API_KEY": "$ENV.API_KEY" }
}
```

**HTTP (REST API)**

```json
{
  "transport": "http",
  "url": "https://mcp.example.com/api",
  "authorization_token": "$ENV.TOKEN"
}
```

**SSE (Server-Sent Events)**

```json
{
  "transport": "sse",
  "url": "https://mcp.example.com/events",
  "authorization_token": "$ENV.TOKEN"
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
    "servers": [{ "server_id": "tools", "tools": ["search", "calculate"] }]
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
      { server_id: "data", resources: ["data://reports"] },
    ],
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
  limit: 10,
});
// { content: [{ type: "text", text: "..." }] }
```

### Batch Tool Calls

```typescript
// Sequential
const results = ctx.mcp.CallTools("server-id", [
  { name: "step1", arguments: { input: "a" } },
  { name: "step2", arguments: { input: "b" } },
]);

// Parallel
const results = ctx.mcp.CallToolsParallel("server-id", [
  { name: "api1", arguments: {} },
  { name: "api2", arguments: {} },
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

## Tool Schema Mapping

Define input schemas for process transport tools:

```
mcps/
└── mapping/
    └── <server-id>/
        └── schemes/
            ├── search.in.yao      # Input schema
            └── search.out.yao     # Output schema (optional)
```

**mapping/tools/schemes/search.in.yao**

```json
{
  "type": "object",
  "description": "Search data",
  "properties": {
    "keyword": { "type": "string" },
    "page": { "type": "integer" }
  },
  "x-process-args": [":arguments"]
}
```

The `x-process-args` maps MCP arguments to Yao Process parameters:

- `":arguments"` - Pass entire arguments object
- `"$args.field"` - Extract specific field

### Schema with Nested Objects

```json
{
  "type": "object",
  "description": "Extract structured data from input",
  "properties": {
    "intent": {
      "type": "string",
      "enum": ["query", "create", "update"],
      "description": "Operation intent"
    },
    "items": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "value": { "type": "number" }
        },
        "required": ["name", "value"]
      }
    }
  },
  "required": ["intent"],
  "x-process-args": [":arguments"]
}
```

## Using Assistant Models in MCP

MCP tools can reference assistant's own models:

**mcps/data.mcp.yao**

```json
{
  "label": "Data Tools",
  "transport": "process",
  "tools": {
    "list_orders": "models.agents.my-assistant.order.Paginate",
    "get_order": "models.agents.my-assistant.order.Find",
    "create_order": "models.agents.my-assistant.order.Create",
    "custom_query": "agents.my-assistant.orders.Query"
  }
}
```

See [Models](models.md) for defining assistant models.

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
  "label": "Calculator",
  "description": "Math operations",
  "transport": "process",
  "tools": {
    "add": "scripts.math.Add",
    "multiply": "scripts.math.Multiply"
  }
}
```

**package.yao**

```json
{
  "name": "Math Assistant",
  "connector": "gpt-4o",
  "mcp": {
    "servers": [{ "server_id": "calculator", "tools": ["add", "multiply"] }]
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
      mcp_servers: [{ server_id: "calculator" }],
    };
  }
  return { messages };
}

function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  const { tools } = payload;

  if (tools?.length > 0) {
    const calcResult = tools.find((t) => t.server === "calculator");
    if (calcResult?.result) {
      return {
        data: {
          answer: calcResult.result,
          expression: calcResult.arguments,
        },
      };
    }
  }

  return null;
}
```
