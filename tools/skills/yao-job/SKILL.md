---
name: yao-job
description: "Background job management — start, monitor, wait, stop long-running commands. ALWAYS invoke this skill when the user asks to run background tasks, check job status, read job output, or manage long-running commands."
---

# Background Jobs

## Tools

- `yao_job_start` — Start a command. `background false` (default) blocks until done and returns `output_tail`; `background true` returns `job_id` immediately.
- `yao_job_list` — List jobs. Supports `--status` filter and `--limit`/`--offset` pagination.
- `yao_job_get` — Get full details for a job (or daemon by ID).
- `yao_job_output` — Read process output (works for both job and daemon IDs).
- `yao_job_wait` — Block until a background job finishes or timeout expires. Jobs only, not daemons.
- `yao_job_stop` — Kill a running job.

## Parameters

### yao_job_start

| Param | Required | Default | Description |
|-------|----------|---------|-------------|
| `command` | yes | — | Shell command to execute |
| `description` | yes | — | Concise 5–10 word summary |
| `background` | no | `false` | `true` returns immediately; `false` blocks until done |
| `args` | no | — | Additional arguments (rarely needed; prefer inline in `command`) |
| `working_dir` | no | workspace | Working directory |
| `max_duration_ms` | no | — | Kill the job after this many milliseconds |

### yao_job_list

| Param | Default | Description |
|-------|---------|-------------|
| `status` | all | Filter: `running`, `stopping`, `completed`, `failed`, `cancelled`, `all` |
| `limit` | 20 | Max items per page |
| `offset` | 0 | Skip first N items |

### yao_job_output

| Param | Required | Description |
|-------|----------|-------------|
| `job_id` | yes | Job or daemon ID |
| `offset` | no | Byte offset to read from (0 = beginning) |

**Output read behavior:**
- Returns all available content from `offset` in a single read (no `limit` parameter).
- Ring buffer caps at **10 MB**; older content beyond this is discarded.
- When content has been discarded, `lossy=true` in the response.
- `size` = total bytes the process has produced; `offset` = earliest readable byte position.
- Example: a 20 MB output has `size=20000000, offset≈10000000, lossy=true` — the first ~10 MB was discarded.

### yao_job_wait

| Param | Required | Default | Description |
|-------|----------|---------|-------------|
| `job_id` | yes | — | Job ID |
| `timeout_ms` | no | 60000 | Timeout in milliseconds |

## Response Fields

All responses are JSON. Job objects contain:

```
id, kind, session_id, status, description, background,
command, command_argv, command_line, pid, pgid, user,
started_at, finished_at, duration_ms, exit_code, signal, detail, seq
```

### Foreground start (background=false)

Returns the job object plus `output_tail` (≤4KB tail of stdout+stderr):

```json
{ "id": "job_...", "status": "completed", "exit_code": 0, ..., "output_tail": "hello world\n" }
```

### List (paginated)

```json
{ "jobs": [...], "total": 130, "offset": 0, "limit": 20, "has_more": true }
```

### Wait

```json
{ "job": { ... }, "timed_out": false }
```

### Stop (already terminal)

```json
{ "job": { ... }, "detail": "already in terminal state (failed); no action taken" }
```

## Errors

Structured JSON with `code` and `message`:

```json
{ "error": { "code": "invalid_argument", "message": "job_id is required" } }
```

Codes: `invalid_argument`, `not_found`, `internal`.

## Usage Rules

1. **Use `background true`** (not bare `--background`) for commands expected to run longer than 30 seconds.
2. **Always provide a `description`**.
3. **Foreground is default** — the result and output come back in one call.
4. **Monitor background output** with `yao_job_output` to check progress.
5. **Use `yao_job_wait`** to block until a background job completes rather than polling.

## BackgroundJobReceipt

When a background job finishes, the system delivers a `<BackgroundJobReceipt>` XML message. Upon receiving it:

1. Review the `status`, `exit_code`, `cause`, and `output_tail`.
2. If `status=failed`, investigate and take corrective action.
3. If `status=completed`, acknowledge and continue the task.
