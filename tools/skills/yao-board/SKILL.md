---
name: yao-board
description: Kanban board and task query expert. ALWAYS invoke this skill when the user asks about boards, tasks, task status, or project progress. Do not guess task state — use this skill first.
---

# Board & Task Tools

Two tools for querying kanban boards and tasks, called via bash.

## board_list

List all kanban boards with their columns and task counts.

```bash
tai tool board_list '{}'
```

## task_list

List tasks with optional filtering.

```bash
tai tool task_list '{"board_id": "xxx", "run_status": "running"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `run_status` | string | no | Filter: pending/running/waiting/completed/failed |
| `assistant_id` | string | no | Filter by assistant ID |
| `board_id` | string | no | Filter by board ID |
| `page` | number | no | Page number (default 1) |
| `page_size` | number | no | Items per page (default 50) |

## Guidelines

- Use board_list first to discover available boards and their IDs
- Use task_list with filters to find specific tasks
- All output is JSON
