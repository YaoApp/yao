---
name: yao-robot
description: Robot management expert. ALWAYS invoke this skill when you need to list, create, update, or manage robots, check robot status, trigger robot executions, cancel tasks, or retrieve execution results. Use this skill before guessing robot APIs.
---

# Robot Tools

Ten tools for managing robots and their executions, called via bash.

## robot_list

List robots with summary info. Use `robot_get` for full details.

```bash
tai tool robot_list '{}'
tai tool robot_list '{"keywords": "weather", "page": 1, "pagesize": 10}'
```

| Parameter  | Type    | Required | Description                                    |
|------------|---------|----------|------------------------------------------------|
| `team_id`  | string  | no       | Filter by team ID                              |
| `status`   | string  | no       | Filter by robot status                         |
| `keywords` | string  | no       | Search in name/bio                             |
| `page`     | integer | no       | Page number (default 1)                        |
| `pagesize` | integer | no       | Page size (default 20, max 100)                |

## robot_get

Get full robot profile including config, agents, prompts.

```bash
tai tool robot_get '{"member_id": "rob-abc123"}'
```

| Parameter   | Type   | Required | Description      |
|-------------|--------|----------|------------------|
| `member_id` | string | yes      | Robot member ID  |

## robot_create

Create a new robot with profile and config.

```bash
tai tool robot_create '{"display_name": "Weather Bot", "bio": "Reports weather daily"}'
tai tool robot_create '{"display_name": "Code Reviewer", "system_prompt": "You review code", "agents": ["yao.agent-smith"]}'
```

| Parameter         | Type    | Required | Description                              |
|-------------------|---------|----------|------------------------------------------|
| `display_name`    | string  | yes      | Robot display name                       |
| `bio`             | string  | no       | Robot description                        |
| `system_prompt`   | string  | no       | System prompt                            |
| `agents`          | array   | no       | Agent IDs the robot can use              |
| `workspace`       | string  | no       | Workspace ID to bind                     |
| `autonomous_mode` | boolean | no       | Enable autonomous mode                   |
| `robot_config`    | object  | no       | See robot_config structure below           |

### robot_config structure

Only these sub-fields are accepted (others are silently ignored for security):

| Sub-field        | Type   | Description                                                    |
|------------------|--------|----------------------------------------------------------------|
| `identity`       | object | `{role, duties[], rules[]}` — role definition (role required)  |
| `quota`          | object | `{max, queue, priority}` — concurrency limits                  |
| `clock`          | object | `{mode, times[], days[], every, tz, timeout}` — scheduling     |
| `triggers`       | object | `{clock: {enabled}, intervene: {enabled}, event: {enabled}}`   |
| `executor`       | object | `{mode, max_duration}` — execution mode (standard/dryrun/sandbox) |
| `default_locale` | string | Default language, e.g. "en", "zh"                              |

## robot_update

Update robot profile or config fields. Only provided fields are changed.

> **Warning**: `robot_config` is a **full replace**, not a merge. If you only pass
> `{"quota": {"max": 5}}`, other sub-configs (identity, clock, etc.) will be cleared.
> Always pass the complete robot_config when updating.

```bash
tai tool robot_update '{"member_id": "rob-abc123", "bio": "Updated description"}'
```

| Parameter   | Type   | Required | Description                      |
|-------------|--------|----------|----------------------------------|
| `member_id` | string | yes      | Robot member ID                  |
| Others      | varies | no       | Same as robot_create (optional)  |

## robot_status

Check if robot is busy: running task count, available slots, last/next run times.

```bash
tai tool robot_status '{"member_id": "rob-abc123"}'
```

| Parameter   | Type   | Required | Description      |
|-------------|--------|----------|------------------|
| `member_id` | string | yes      | Robot member ID  |

## robot_execution_list

List executions (tasks in progress or recent history).

```bash
tai tool robot_execution_list '{"member_id": "rob-abc123"}'
tai tool robot_execution_list '{"member_id": "rob-abc123", "status": "running"}'
```

| Parameter   | Type    | Required | Description                                              |
|-------------|---------|----------|----------------------------------------------------------|
| `member_id` | string  | yes      | Robot member ID                                          |
| `status`    | string  | no       | Filter: pending, running, paused, completed, failed, cancelled, confirming, waiting |
| `page`      | integer | no       | Page number (default 1)                                  |
| `pagesize`  | integer | no       | Page size (default 20)                                   |

## robot_execution_get

Get execution details: goals, tasks, progress, errors.

```bash
tai tool robot_execution_get '{"member_id": "rob-abc123", "execution_id": "exec-xyz"}'
```

| Parameter      | Type   | Required | Description      |
|----------------|--------|----------|------------------|
| `member_id`    | string | yes      | Robot member ID  |
| `execution_id` | string | yes      | Execution ID     |

## robot_execution_create

Trigger a new execution. Returns execution_id for tracking.

```bash
tai tool robot_execution_create '{"member_id": "rob-abc123", "messages": [{"role": "user", "content": "Generate report"}]}'
tai tool robot_execution_create '{"member_id": "rob-abc123", "trigger_type": "human"}'
```

| Parameter      | Type   | Required | Description                      |
|----------------|--------|----------|----------------------------------|
| `member_id`    | string | yes      | Robot member ID                  |
| `messages`     | array  | no       | Input messages (text/images)     |
| `trigger_type` | string | no       | "human" or "event" (default human)|

## robot_execution_cancel

Cancel a running execution.

```bash
tai tool robot_execution_cancel '{"member_id": "rob-abc123", "execution_id": "exec-xyz"}'
```

| Parameter      | Type   | Required | Description      |
|----------------|--------|----------|------------------|
| `member_id`    | string | yes      | Robot member ID  |
| `execution_id` | string | yes      | Execution ID     |

## robot_result_list

List completed executions with delivery outputs (reports, files).

```bash
tai tool robot_result_list '{"member_id": "rob-abc123"}'
```

| Parameter   | Type    | Required | Description                  |
|-------------|---------|----------|------------------------------|
| `member_id` | string  | yes      | Robot member ID              |
| `page`      | integer | no       | Page number (default 1)      |
| `pagesize`  | integer | no       | Page size (default 20)       |

## Typical Workflow

1. `robot_list` — discover available robots
2. `robot_get` — get full details of a specific robot
3. `robot_status` — check if the robot has available slots
4. `robot_execution_create` — trigger a task, get `execution_id`
5. `robot_execution_get` — poll execution progress using `execution_id`
6. `robot_result_list` — retrieve completed results

## Guidelines

- Use `robot_list` for discovery, `robot_get` for details (list returns summaries only)
- `robot_status` tells you if the robot is busy before triggering
- After `robot_execution_create`, use the returned `execution_id` with `robot_execution_get` to track progress
- `robot_result_list` only shows completed executions with outputs
- All output is JSON
