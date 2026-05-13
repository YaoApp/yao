---
name: yao-agent
description: Agent management expert. ALWAYS invoke this skill when you need to list available agents, download or reference agent source code, deploy agent code to the host, or query the LLM connector matrix. Do not guess agent structures — use this skill first.
---

# Agent Tools

Five tools for managing agents on the host, called via bash.

## agent_list

List available agents. Returns ID, name, description, and capabilities for each agent.

```bash
tai tool agent_list '{}'
tai tool agent_list '{"namespace": "smith"}'
```

| Parameter   | Type   | Required | Description                                              |
|-------------|--------|----------|----------------------------------------------------------|
| `namespace` | string | no       | Filter by namespace (e.g. `yao`, `smith`). Omit for all. |

## agent_download

Download a **smith-namespace** agent into the development directory for editing. **Restricted to `smith` namespace only** — for other agents, use `agent_reference`.

```bash
tai tool agent_download '{"id": "smith.weather"}'
```

| Parameter | Type   | Required | Description                                              |
|-----------|--------|----------|----------------------------------------------------------|
| `id`      | string | yes      | Agent ID in dot notation. Must be `smith.*`.             |

Downloaded code lands in `agent-smith-dev/assistants/smith/<name>/`.

## agent_reference

Download agent source code from the host into `.references/` for **read-only study**. Any agent across all namespaces can be referenced.

```bash
tai tool agent_reference '{"id": "yao.slides"}'
tai tool agent_reference '{"id": "yao.keeper"}'
```

| Parameter | Type   | Required | Description                                        |
|-----------|--------|----------|----------------------------------------------------|
| `id`      | string | yes      | Agent ID in dot notation (e.g. `yao.slides`)       |

Referenced code lands in `agent-smith-dev/.references/<namespace>/<name>/`.

## agent_deploy

Deploy agent source code from the sandbox development directory to the host. **Restricted to the `smith` namespace only** — attempts to deploy to other namespaces will be rejected.

```bash
tai tool agent_deploy '{"id": "smith.weather"}'
tai tool agent_deploy '{"id": "smith.weather", "message": "add SUI page"}'
```

| Parameter | Type   | Required | Description                                            |
|-----------|--------|----------|--------------------------------------------------------|
| `id`      | string | yes      | Agent ID in dot notation. Must use `smith` namespace.  |
| `message` | string | no       | Optional deploy message for logging.                   |

## agent_connectors

Get the current user's LLM connector matrix. Returns metadata for each role (default, heavy, light, vision, etc.) **without API keys**. Use this to understand which models are available and their capabilities.

```bash
tai tool agent_connectors '{}'
```

No parameters required.

## Guidelines

- Use `agent_list` to discover agents before downloading or referencing
- `agent_download` is for editing smith agents — code lands in `agent-smith-dev/assistants/smith/<name>/`
- `agent_reference` is for studying any agent — code lands in `agent-smith-dev/.references/<namespace>/<name>/`
- Deploy is restricted to the `smith` namespace for safety
- Connector data never includes API keys, secrets, or tokens
- All output is JSON
