---
name: yao-process
description: Yao process execution expert. ALWAYS invoke this skill when the user needs to call a Yao process, query data models, run scripts, or check process permissions. Do not call processes without checking this skill first.
---

# Process Tools

Two tools for Yao process execution, called via bash.

## process_call

Execute a Yao Process by its fully qualified name.

```bash
tai tool process_call '{"name": "models.user.Find", "args": [1, {}]}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Process name (e.g. `models.user.Find`) |
| `args` | array | no | Positional arguments |

## process_allowed

Check which processes are permitted, or verify a specific process.

```bash
# List all allowed rules
tai tool process_allowed '{}'

# Check a specific process
tai tool process_allowed '{"name": "models.user.Find"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | no | Process name to check. Omit to list all rules. |

## Workflow

1. Use **process_allowed** to check what is permitted
2. Use **doc_inspect** (from yao-doc skill) to understand the process signature
3. Use **process_call** to execute

## Guidelines

- Always check documentation before calling an unfamiliar process
- A 403 error means the process is not in the allowed list
- Process names follow the pattern `group.id.Method` (e.g. `models.user.Find`, `scripts.auth.Check`)
- Rules use prefix matching: `models.*` matches all model processes
