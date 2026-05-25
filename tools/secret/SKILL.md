# Secret Management

Agents can read user-configured secrets at runtime using the `tai tool` CLI.
Secrets are encrypted at rest (AES-256-GCM) and decrypted only when read.

## Available Tools

| Tool | Description |
|------|-------------|
| `secret_list` | List secret names and descriptions (no values) |
| `secret_read` | Read a single secret value by name |

## Usage

### Bash

```bash
# List available secrets
tai tool secret_list

# Read a secret
TOKEN=$(tai tool secret_read '{"name": "GITHUB_TOKEN"}' | jq -r '.value')
git clone "https://${TOKEN}@github.com/org/repo.git"
```

### Node.js

```javascript
const { execSync } = require("child_process");

function readSecret(name) {
  const raw = execSync(
    `tai tool secret_read '${JSON.stringify({ name })}'`,
    { encoding: "utf-8" }
  );
  return JSON.parse(raw).value;
}

const token = readSecret("GITHUB_TOKEN");
```

### Python

```python
import json
import subprocess

def read_secret(name: str) -> str:
    result = subprocess.run(
        ["tai", "tool", "secret_read", json.dumps({"name": name})],
        capture_output=True, text=True, check=True,
    )
    return json.loads(result.stdout)["value"]

token = read_secret("GITHUB_TOKEN")
```

### PowerShell

```powershell
function Read-Secret {
    param([string]$Name)
    $json = @{ name = $Name } | ConvertTo-Json -Compress
    $result = tai tool secret_read $json | ConvertFrom-Json
    return $result.value
}

$token = Read-Secret -Name "GITHUB_TOKEN"
```

## Security Rules

1. **Never print or log secret values** — Do not write secrets to stdout, stderr, or any log file.
2. **Never write secrets to files** — Exception: SSH keys may be written to `~/.ssh/` with `chmod 600` permissions.
3. **Never send secrets to the LLM** — Secret values must not appear in prompt content, system messages, or tool call results that are forwarded to the model.
4. **Scope isolation** — Secrets are scoped per user per agent. An agent can only access secrets configured for it.
5. **Audit trail** — Every `secret_read` call is logged in the audit trail with the caller's identity.
