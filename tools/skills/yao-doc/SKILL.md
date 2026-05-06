---
name: yao-doc
description: Yao process documentation expert. ALWAYS invoke this skill when the user needs to discover available processes, read process signatures, or validate process names. Do not guess process APIs — use this skill first.
---

# Documentation Tools

Three tools for browsing Yao process documentation, called via bash.

## doc_list

Search and list available process documentation entries.

```bash
tai tool doc_list '{"keyword": "user", "limit": 10}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `keyword` | string | no | Search keyword (empty to list all) |
| `limit` | integer | no | Max results (default 20) |

## doc_inspect

Get detailed documentation for a specific process: arguments, return type, methods.

```bash
tai tool doc_inspect '{"name": "models.user.Find"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Process name (e.g. `models.user.Find`) |

## doc_validate

Check if a process name is valid. Returns suggestions for similar processes if not found.

```bash
tai tool doc_validate '{"name": "models.user.Findd"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Process name to validate |

## Workflow

1. **doc_list** — discover available processes by keyword
2. **doc_inspect** — read the full signature before calling
3. **doc_validate** — fix typos when a process name doesn't work

## Guidelines

- Always use doc_inspect before calling an unfamiliar process via process_call
- Use doc_validate when you get unexpected errors — the name might be misspelled
