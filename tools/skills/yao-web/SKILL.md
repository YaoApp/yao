---
name: yao-web
description: Web information retrieval expert. ALWAYS invoke this skill when the user needs to search the web, fetch a URL, or access real-time information beyond training data. Do not guess or use stale knowledge — use this skill first.
---

# Web Tools

Two tools for web information retrieval, called via bash.

## web_search

Search the web for real-time information. Returns structured results with title, URL, and content snippet.

```bash
tai tool web_search '{"query": "search terms", "limit": 5}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | Search query |
| `limit` | integer | no | Max results (default 10) |

## web_fetch

Fetch a web page and return its content in readable format.

```bash
tai tool web_fetch '{"url": "https://example.com", "format": "markdown"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | Fully-formed URL to fetch |
| `format` | string | no | `markdown` (default) or `html` |

## Guidelines

- Use web_search to discover URLs, then web_fetch to read the content
- Include the current year when searching for recent information
- All output is JSON
