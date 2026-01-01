# Assistant Configuration

## Directory Structure

```
assistants/
└── <assistant-id>/
    ├── package.yao          # Required: Configuration
    ├── prompts.yml          # Optional: Default prompts
    ├── prompts/             # Optional: Prompt presets
    │   ├── chat.yml
    │   └── task.yml
    ├── locales/             # Optional: Translations
    │   ├── en-us.yml
    │   └── zh-cn.yml
    ├── src/                 # Optional: Hook scripts
    │   └── index.ts
    └── mcps/                # Optional: MCP servers
        └── tools.mcp.yao
```

## package.yao

### Basic Fields

```json
{
  "name": "{{ name }}",
  "type": "assistant",
  "avatar": "/assets/avatar.png",
  "description": "{{ description }}",
  "connector": "gpt-4o",
  "tags": ["Category1", "Category2"],
  "sort": 1
}
```

| Field         | Type     | Description                          |
| ------------- | -------- | ------------------------------------ |
| `name`        | string   | Display name (supports i18n `{{ }}`) |
| `type`        | string   | Type: `assistant` (default)          |
| `avatar`      | string   | Avatar image path                    |
| `description` | string   | Description (supports i18n)          |
| `connector`   | string   | LLM connector ID                     |
| `tags`        | string[] | Categorization tags                  |
| `sort`        | number   | Display order                        |

### Connector Options

```json
{
  "connector": "gpt-4o",
  "connector_options": {
    "optional": true,
    "connectors": ["gpt-4o", "gpt-4o-mini", "claude-3"],
    "filters": ["tool_calls", "vision"]
  }
}
```

| Field        | Type     | Description                                  |
| ------------ | -------- | -------------------------------------------- |
| `optional`   | boolean  | Allow user to select connector               |
| `connectors` | string[] | Available connectors (empty = all)           |
| `filters`    | string[] | Required capabilities: `vision`, `audio`, `tool_calls`, `reasoning` |

### Generation Options

```json
{
  "options": {
    "temperature": 0.7,
    "max_tokens": 4096
  }
}
```

### Placeholder (UI Hints)

```json
{
  "placeholder": {
    "title": "{{ chat.title }}",
    "description": "{{ chat.description }}",
    "prompts": [
      "{{ chat.prompts.0 }}",
      "{{ chat.prompts.1 }}"
    ]
  }
}
```

### Visibility & Access

```json
{
  "public": true,
  "share": "team",
  "readonly": true,
  "built_in": true,
  "mentionable": true,
  "automated": false
}
```

| Field         | Type    | Description                       |
| ------------- | ------- | --------------------------------- |
| `public`      | boolean | Visible to all users              |
| `share`       | string  | Sharing scope: `private`, `team`  |
| `readonly`    | boolean | Prevent user modifications        |
| `built_in`    | boolean | System-managed assistant          |
| `mentionable` | boolean | Can be @mentioned in chat         |
| `automated`   | boolean | Can be triggered automatically    |

### Modes

```json
{
  "modes": ["chat", "task"],
  "default_mode": "task"
}
```

### MCP Servers

```json
{
  "mcp": {
    "servers": [
      "server-id",
      { "server_id": "tools", "tools": ["tool1", "tool2"] },
      { "server_id": "resources", "resources": ["uri://pattern"] }
    ]
  }
}
```

### Knowledge Base

```json
{
  "kb": {
    "collections": ["collection-id-1", "collection-id-2"]
  }
}
```

### Database Models

```json
{
  "db": {
    "models": ["model.name", "another.model"]
  }
}
```

### Uses (Wrapper Tools)

```json
{
  "uses": {
    "vision": "vision-agent",
    "audio": "audio-agent",
    "search": "disabled",
    "fetch": "mcp:fetcher"
  }
}
```

| Field    | Description                                        |
| -------- | -------------------------------------------------- |
| `vision` | Vision processing: `<agent-id>` or `mcp:<server>`  |
| `audio`  | Audio processing: `<agent-id>` or `mcp:<server>`   |
| `search` | Search: `disabled`, `<agent-id>`, or `mcp:<server>`|
| `fetch`  | HTTP fetching: `<agent-id>` or `mcp:<server>`      |

### Search Configuration

```json
{
  "search": {
    "web": {
      "provider": "tavily",
      "max_results": 10
    },
    "kb": {
      "threshold": 0.7,
      "graph": true
    },
    "db": {
      "max_results": 20
    },
    "citation": {
      "format": "[{index}]",
      "auto_inject_prompt": true
    }
  }
}
```

## Environment Variables

Use `$ENV.VAR_NAME` for sensitive values:

```json
{
  "connector": "$ENV.LLM_CONNECTOR"
}
```

## Complete Example

```json
{
  "name": "{{ name }}",
  "type": "assistant",
  "avatar": "/assets/assistant.png",
  "connector": "gpt-4o",
  "connector_options": {
    "optional": true,
    "connectors": ["gpt-4o", "gpt-4o-mini"],
    "filters": ["tool_calls"]
  },
  "mcp": {
    "servers": [{ "server_id": "tools", "tools": ["search", "calculate"] }]
  },
  "description": "{{ description }}",
  "options": { "temperature": 0.7 },
  "public": true,
  "placeholder": {
    "title": "{{ chat.title }}",
    "description": "{{ chat.description }}",
    "prompts": ["{{ chat.prompts.0 }}", "{{ chat.prompts.1 }}"]
  },
  "tags": ["Productivity"],
  "modes": ["chat", "task"],
  "default_mode": "chat",
  "sort": 1,
  "readonly": true,
  "mentionable": true
}
```
