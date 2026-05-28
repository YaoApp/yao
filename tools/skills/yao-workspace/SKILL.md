---
name: yao-workspace
description: Workspace file I/O expert. ALWAYS invoke this skill when you need to list workspaces, read or write files in a workspace on a remote node, or browse workspace directories. Use this for cross-node file operations — for local sandbox files, use standard filesystem tools instead.
---

# Workspace Tools

Five tools for managing workspaces and file I/O, called via bash.

## workspace_list

List workspaces with summary info. Use `workspace_get` for full details.

```bash
tai tool workspace_list '{}'
tai tool workspace_list '{"node": "tai-node-1"}'
```

| Parameter | Type   | Required | Description              |
|-----------|--------|----------|--------------------------|
| `node`    | string | no       | Filter by Tai node ID    |

## workspace_get

Get full workspace details: name, owner, node, labels, timestamps.

```bash
tai tool workspace_get '{"id": "ws-abc123"}'
```

| Parameter | Type   | Required | Description   |
|-----------|--------|----------|---------------|
| `id`      | string | yes      | Workspace ID  |

## workspace_file_list

List files and directories in a workspace path.

```bash
tai tool workspace_file_list '{"id": "ws-abc123"}'
tai tool workspace_file_list '{"id": "ws-abc123", "path": "src/components"}'
```

| Parameter | Type   | Required | Description                    |
|-----------|--------|----------|--------------------------------|
| `id`      | string | yes      | Workspace ID                   |
| `path`    | string | no       | Directory path (default ".")   |

## workspace_file_read

Read file content from workspace. Returns text (UTF-8) or base64 encoded.

```bash
tai tool workspace_file_read '{"id": "ws-abc123", "path": "README.md"}'
tai tool workspace_file_read '{"id": "ws-abc123", "path": "image.png", "encoding": "base64"}'
```

| Parameter  | Type   | Required | Description                              |
|------------|--------|----------|------------------------------------------|
| `id`       | string | yes      | Workspace ID                             |
| `path`     | string | yes      | File path relative to workspace root     |
| `encoding` | string | no       | "text" (default) or "base64"             |

## workspace_file_write

Write content to a file in workspace. For binary files, use base64 encoding.

```bash
tai tool workspace_file_write '{"id": "ws-abc123", "path": "output.txt", "content": "Hello World"}'
tai tool workspace_file_write '{"id": "ws-abc123", "path": "data.bin", "content": "base64data...", "encoding": "base64"}'
```

| Parameter  | Type   | Required | Description                              |
|------------|--------|----------|------------------------------------------|
| `id`       | string | yes      | Workspace ID                             |
| `path`     | string | yes      | File path relative to workspace root     |
| `content`  | string | yes      | File content (text or base64 encoded)    |
| `encoding` | string | no       | "text" (default) or "base64"             |

## Typical Workflow

1. `workspace_list` — discover available workspaces
2. `workspace_get` — get full details (node, labels)
3. `workspace_file_list` — browse directory structure
4. `workspace_file_read` — read specific files
5. `workspace_file_write` — write output files

## Guidelines

- Use `workspace_list` for discovery, `workspace_get` for details (list returns summaries only)
- For large files (>1MB), prefer local filesystem operations if the workspace is mounted in the sandbox
- Use `encoding: "base64"` for binary files (images, archives, etc.)
- File paths are relative to the workspace root — do not use absolute paths
- The current sandbox workspace ID is available via `$CTX_WORKSPACE_ID`
- All output is JSON
