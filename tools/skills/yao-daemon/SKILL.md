---
name: yao-daemon
description: "Daemon process management — start, monitor, restart, stop long-running services. ALWAYS invoke this skill when the user asks to start, check, restart, or stop background services, dev servers, or daemon processes."
---

# Daemon Management

## Tools

- `yao_daemon_start` — Start a long-running service. Always runs in background; returns `daemon_id` immediately.
- `yao_daemon_list` — List daemons. Supports `--status` filter and `--limit`/`--offset` pagination.
- `yao_daemon_status` — Get full details for a daemon (`daemon_port`, `daemon_url`, `started_at`, `daemon_restarts`).
- `yao_daemon_stop` — Stop a running daemon.
- `yao_daemon_restart` — Restart a daemon (preserves the same ID, increments `daemon_restarts`).
- `yao_job_output` — Read daemon output (pass `daemon_id` as `job_id`).

## Parameters

### yao_daemon_start

| Param | Required | Default | Description |
|-------|----------|---------|-------------|
| `command` | yes | — | Shell command to execute |
| `description` | yes | — | Concise 5–10 word summary |
| `port` | no | auto-detect | Expected listen port; the system verifies the port is ready before reporting it |
| `args` | no | — | Additional arguments |
| `working_dir` | no | workspace | Working directory |

### yao_daemon_list

| Param | Default | Description |
|-------|---------|-------------|
| `status` | all | Filter: `running`, `stopping`, `completed`, `failed`, `cancelled`, `port_detecting`, `all` |
| `limit` | 20 | Max items per page |
| `offset` | 0 | Skip first N items |

### yao_daemon_restart / yao_daemon_status / yao_daemon_stop

| Param | Required | Description |
|-------|----------|-------------|
| `daemon_id` | yes | The daemon ID |

## Response Fields

All responses are JSON. Daemon objects contain:

```
id, kind, session_id, status, description,
command, command_argv, command_line, pid, pgid, user,
started_at, finished_at, duration_ms, exit_code, signal, detail,
daemon_port, daemon_url, daemon_restarts, seq
```

- `daemon_port` — The verified listen port (0 if not yet detected or not specified).
- `daemon_url` — `http://localhost:<port>` (empty until port is verified as listening by the daemon's own process tree).
- `daemon_restarts` — Number of restarts via `yao_daemon_restart`.
- `started_at` — Unix millisecond timestamp of the most recent start.

**Port detection behavior:**

Both modes start by returning `port_detecting` with `daemon_port: 0` if the port is not confirmed within the initial ~2s settle window. If the port is confirmed within ~2s, `start` returns `running` with `daemon_port`/`daemon_url` directly.

- **`--port N`** (explicit): Verifies port N is listening AND owned by this daemon's process tree. If not confirmed within ~2s, returns `status: "port_detecting"`. Poll `yao_daemon_status` until status changes to `running` — either with `daemon_port: N` and `daemon_url` (port confirmed), or with `daemon_port: 0` and `detail: "specified port N not detected within timeout"` (~60s).
- **No `--port`** (auto-detect): Scans the daemon's process tree for any listening TCP port. If not found within ~2s, returns `status: "port_detecting"`. The scan continues in background (~60s); if a port is found, status changes to `running` with `daemon_port`/`daemon_url` and a `<BackgroundJobReceipt>` is delivered. If no port is found after ~60s, status changes to `running` with `daemon_port: 0`.

### List (paginated)

```json
{ "daemons": [...], "total": 5, "offset": 0, "limit": 20, "has_more": false }
```

### Stop (already terminal)

```json
{ "job": { ... }, "detail": "already in terminal state (cancelled); no action taken" }
```

## Errors

Structured JSON with `code` and `message`:

```json
{ "error": { "code": "not_found", "message": "daemon \"dmn_xxx\" not found" } }
```

Codes: `invalid_argument`, `not_found`, `internal`.

## Usage Rules

1. **Always provide a `description`**.
2. **Specify `port`** when starting a server — the system confirms the port is actually listening before reporting `daemon_url`.
3. **Daemons run indefinitely** — use `yao_daemon_stop` to terminate.
4. **Use `yao_daemon_restart`** to restart a crashed or misconfigured daemon without losing its ID.
5. **Monitor output** with `yao_job_output` passing the `daemon_id` as `job_id`.

## BackgroundJobReceipt

When a daemon's port becomes ready, the system delivers a `<BackgroundJobReceipt>` XML message containing the `daemon_url`. Use that URL for subsequent HTTP requests to the daemon.
