# Search

The agent search system provides automatic search across web, knowledge base (KB), and database (DB).

## Auto Search Flow

1. **Intent Detection** - `__yao.needsearch` agent analyzes user message
2. **Search Execution** - Executes web/kb/db searches based on intent
3. **Context Injection** - Results injected as system message
4. **Citation** - LLM can cite results using `[1]`, `[2]` format

## Configuration

### Global (agent/search.yml)

```yaml
web:
  provider: tavily  # tavily, serper, serpapi
  api_key_env: TAVILY_API_KEY
  max_results: 10

kb:
  threshold: 0.7
  graph: true

db:
  max_results: 20

keyword:
  max_keywords: 5
  language: en

citation:
  format: "[{index}]"
  auto_inject_prompt: true
```

### Per Assistant (package.yao)

```json
{
  "search": {
    "web": { "max_results": 5 },
    "kb": { "threshold": 0.8 },
    "citation": { "format": "[{index}]" }
  },
  "kb": {
    "collections": ["docs", "faq"]
  },
  "db": {
    "models": ["articles", "products"]
  }
}
```

## Controlling Search in Hooks

### Disable Search

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  return {
    messages,
    search: false
  };
}
```

### Enable Specific Types

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  return {
    messages,
    search: {
      need_search: true,
      search_types: ["kb", "db"],  // Only KB and DB
      confidence: 1.0,
      reason: "controlled by hook"
    }
  };
}
```

### Disable via Uses

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  return {
    messages,
    uses: { search: "disabled" }
  };
}
```

## Search API (ctx.search)

### Web Search

```typescript
const result = ctx.search.Web("query", {
  limit: 10,
  sites: ["example.com", "docs.example.com"],
  time_range: "week",  // day, week, month, year
  rerank: { top_n: 5 }
});
```

### Knowledge Base Search

```typescript
const result = ctx.search.KB("query", {
  collections: ["docs", "faq"],
  threshold: 0.7,
  limit: 10,
  graph: true,
  rerank: { top_n: 5 }
});
```

### Database Search

```typescript
const result = ctx.search.DB("query", {
  models: ["articles"],
  wheres: [{ column: "status", value: "published" }],
  orders: [{ column: "created_at", option: "desc" }],
  select: ["id", "title", "content"],
  limit: 20,
  rerank: { top_n: 10 }
});
```

### Parallel Search

```typescript
// Wait for all
const results = ctx.search.All([
  { type: "web", query: "topic" },
  { type: "kb", query: "topic", collections: ["docs"] },
  { type: "db", query: "topic", models: ["articles"] }
]);

// First success with results
const results = ctx.search.Any([
  { type: "web", query: "topic" },
  { type: "kb", query: "topic" }
]);

// First to complete
const results = ctx.search.Race([
  { type: "web", query: "topic" },
  { type: "kb", query: "topic" }
]);
```

## Result Structure

```typescript
interface SearchResult {
  type: "web" | "kb" | "db";
  query: string;
  source: "hook" | "auto" | "user";
  items: SearchItem[];
  error?: string;
}

interface SearchItem {
  citation_id: string;  // "1", "2", etc.
  title: string;
  url: string;
  content: string;
  score: number;
}
```

## Custom Search in Hooks

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const query = messages[messages.length - 1]?.content || "";

  // Custom KB search
  const kbResult = ctx.search.KB(query, {
    collections: ["internal_docs"],
    threshold: 0.8
  });

  if (kbResult.items?.length > 0) {
    // Format results as context
    const context = kbResult.items
      .map((item, i) => `[${i + 1}] ${item.title}\n${item.content}`)
      .join("\n\n");

    // Inject as system message
    const contextMsg = {
      role: "system",
      content: `Reference information:\n${context}`
    };

    return {
      messages: [contextMsg, ...messages],
      search: false  // Skip auto search
    };
  }

  return { messages };
}
```

## Authorization

### KB Collections

Collections are filtered by user authorization:

```typescript
// Only collections user has access to are searched
const result = ctx.search.KB("query", {
  collections: ["public", "internal", "secret"]
  // User without "secret" access won't search that collection
});
```

### DB Models

Database queries include permission filters:

```typescript
// Auth filters are automatically added
// e.g., { column: "__yao_created_by", value: user_id }
const result = ctx.search.DB("query", {
  models: ["user_documents"]
});
```

## Web Search Providers

### Tavily

```yaml
web:
  provider: tavily
  api_key_env: TAVILY_API_KEY
```

### Serper

```yaml
web:
  provider: serper
  api_key_env: SERPER_API_KEY
```

### SerpAPI

```yaml
web:
  provider: serpapi
  api_key_env: SERPAPI_API_KEY
```

## Citation

LLM responses can include citations:

```
Based on the documentation [1], the feature works by... [2]

References:
[1] Getting Started Guide - https://docs.example.com/start
[2] API Reference - https://docs.example.com/api
```

### Citation Format

```yaml
citation:
  format: "[{index}]"  # or "({index})" or "[^{index}]"
  auto_inject_prompt: true
  custom_prompt: |
    When citing sources, use the format [N] where N is the reference number.
```
