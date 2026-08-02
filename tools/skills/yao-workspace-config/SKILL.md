---
name: yao-workspace-config
description: "Workspace Git identity and credential management. ALWAYS invoke this skill when the user asks about configuring Git user info, adding HTTPS tokens, importing SSH keys, or managing workspace-level Git authentication."
---

# Workspace Git Config Tools

Manage workspace-level Git configuration, HTTPS credentials, and SSH keys.

## workspace_git_config

Get or set workspace-level Git configuration values.

```bash
# Get all config
tai tool workspace_git_config '{"workspace_id":"ws-xxx","action":"get"}'

# Get specific key
tai tool workspace_git_config '{"workspace_id":"ws-xxx","action":"get","key":"user.name"}'

# Set config
tai tool workspace_git_config '{"workspace_id":"ws-xxx","action":"set","key":"user.name","value":"John"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| workspace_id | string | yes | Workspace ID |
| action | string | yes | `get` or `set` |
| key | string | no | Config key (e.g. `user.name`). Empty returns all. |
| value | string | for set | Config value |

## workspace_git_credential

Manage HTTPS Git credentials (personal access tokens).

```bash
# Set credential
tai tool workspace_git_credential '{"workspace_id":"ws-xxx","action":"set","host":"github.com","token":"ghp_xxx"}'

# List credentials (tokens are not exposed)
tai tool workspace_git_credential '{"workspace_id":"ws-xxx","action":"list"}'

# Delete credential
tai tool workspace_git_credential '{"workspace_id":"ws-xxx","action":"delete","host":"github.com"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| workspace_id | string | yes | Workspace ID |
| action | string | yes | `set`, `list`, or `delete` |
| host | string | for set/delete | Git host (e.g. `github.com`) |
| username | string | no | Username (defaults to `x-access-token`) |
| token | string | for set | Access token |

## workspace_ssh_key

Manage SSH keys for Git authentication.

```bash
# Import SSH key
tai tool workspace_ssh_key '{"workspace_id":"ws-xxx","action":"import","name":"github","private_key":"-----BEGIN...","host":"github.com"}'

# List SSH keys (private keys not exposed)
tai tool workspace_ssh_key '{"workspace_id":"ws-xxx","action":"list"}'

# Delete SSH key
tai tool workspace_ssh_key '{"workspace_id":"ws-xxx","action":"delete","name":"github"}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| workspace_id | string | yes | Workspace ID |
| action | string | yes | `import`, `list`, or `delete` |
| name | string | for import/delete | Key name |
| private_key | string | for import | PEM-encoded private key |
| public_key | string | no | Public key (derived from private_key if omitted) |
| host | string | no | Git host to associate (e.g. `github.com`) |
