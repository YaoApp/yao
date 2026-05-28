---
name: yao-secret
description: Secret management expert. ALWAYS invoke this skill when you need to read API keys, tokens, or other secrets configured by the user. Never hardcode credentials — use this skill to retrieve them securely.
---

# Secret Tools

Two tools for accessing user-configured secrets, called via bash.

## secret_list

List available secret names and descriptions. **Does not return secret values** — use `secret_read` for that.

```bash
tai tool secret_list '{}'
```

No parameters required. Returns secrets configured for the current assistant.

## secret_read

Read a secret value by name. Returns the decrypted value for use in scripts.

```bash
tai tool secret_read '{"name": "GITHUB_TOKEN"}'
tai tool secret_read '{"name": "AWS_SECRET_KEY"}'
```

| Parameter | Type   | Required | Description                                              |
|-----------|--------|----------|----------------------------------------------------------|
| `name`    | string | yes      | Secret key name (e.g. `GITHUB_TOKEN`, `AWS_SECRET_KEY`)  |

**Security**: Never log, print, or expose the returned secret value in output visible to users.

## Typical Workflow

1. `secret_list` — discover what secrets are available
2. `secret_read` — retrieve a specific secret by name
3. Use the value in API calls, git auth, etc.

## Guidelines

- Always call `secret_list` first to check if a required secret exists before reading
- Never hardcode API keys or tokens — always use `secret_read`
- Secret values are decrypted at read time; treat them as sensitive
- If a secret is not found, prompt the user to configure it in their settings
- All output is JSON
