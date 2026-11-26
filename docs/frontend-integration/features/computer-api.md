# Computer API Inventory

**Audience:** Web/desktop/mobile frontend engineers and coding agents implementing Computer UI or a host-side Tai agent.

**Purpose:** a single list of the Yao Computer and Tai transport endpoints used by the integration guide.

> **Environment:** see [Environment and API endpoints](../environment.md). All URLs are relative to the operator-provided base URL, for example `https://<your-yao-host>`; the transport discovery endpoint is at the site root, not under `/v1`.

**Related:** [Computer connections](computer-connections.md) · [Web authentication](../auth/web.md) · [Desktop authentication](../auth/desktop.md) · [Mobile authentication](../auth/mobile.md).

**Status:** written against Yao `main` on 2026-11-17.

---

## 1. Transport discovery

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/.well-known/yao-transport` | Public | discover the Tai transport port and protocols |

Example:

```json
{
  "transport_port": 6099,
  "protocols": ["HTTP/2", "HTTP/3"],
  "version": "yao"
}
```

The value is deployment-specific. The frontend does not connect to the transport port directly; the host-side Tai node does.

---

## 2. Computer management

All `/v1/computers` endpoints require a signed-in user session or a bearer access token.

| Method | Endpoint | Auth | Purpose | Success |
| --- | --- | --- | --- | --- |
| `GET` | `/v1/computers` | User | list Computers for the caller (owner/team filter; admins: all) | `200`, `{"data":[...]}` |
| `POST` | `/v1/computers/connect` | User | create a Computer and issue a node token | `200`, `ConnectResult` |
| `GET` | `/v1/computers/{computer_id}` | Owner/admin | get one Computer | `200`, `Computer` |
| `GET` | `/v1/computers/{computer_id}/status` | Owner/admin | lightweight runtime status for event fallback | `200`, `{computer_id,status}` |
| `POST` | `/v1/tao/computers` | User | enqueue cloud purchase/provision task | `202`, `{task_id,status}` |
| `GET` | `/v1/tao/computers/tasks/{task_id}` | User | cloud task status | `200`, task |
| `GET` | `/v1/products` | User | live Tao sellable products | `200`, `{data:[...]}` |
| `PATCH` | `/v1/computers/{computer_id}` | Owner/admin | rename or change visibility | `200`, `Computer` |
| `DELETE` | `/v1/computers/{computer_id}` | Owner/admin | delete Computer and revoke its node token | `204`, no body |
| `POST` | `/v1/computers/{computer_id}/disconnect` | Owner/admin | intentionally disconnect; block new connections | `200`, `Computer` |
| `POST` | `/v1/computers/{computer_id}/reconnect` | Owner/admin | allow the existing node to reconnect | `200`, `Computer` |
| `POST` | `/v1/computers/{computer_id}/renew` | Owner/admin | extend or restore a lease | `200`, lease result |
| `GET` | `/v1/computers/{computer_id}/workspaces` | Owner/admin | list node workspaces | `200` or `501 not_supported` |
| `GET` | `/v1/computers/{computer_id}/runtimes` | Owner/admin | list node runtimes | `200` or `501 not_supported` |
| `GET` | `/v1/computers/{computer_id}/containers` | Owner/admin | list containers | `501 not_supported` in the current Tai SDK |

### 2.1 Create

```http
POST /v1/computers/connect
Content-Type: application/json

{
  "name": "My computer",
  "data_dir": "/path/on/host",
  "visibility": "private"
}
```

The response is direct JSON; important fields include (the example `server` is a placeholder):


```json
{
  "computer": {"id": "cmp_...", "node_id": "node-...", "status": "pending"},
  "computer_id": "cmp_...",
  "node_id": "node-...",
  "node_token": "<one-time node token>",
  "expires_at": "...",
  "server": "http://<yao-host>:<port>",
  "transport_port": 6099,
  "protocols": ["HTTP/2", "HTTP/3"]
}
```

`server` is currently built from `YAO_HOST`/`YAO_PORT` and is always formatted as `http://<host>:<port>`. If Yao listens on `0.0.0.0`, the value is a loopback URL and must be replaced with a reachable deployment URL (for example, the HTTPS public URL) by a host on another machine. There is no separate public-URL override in this response path.

`node_token` is a secret for the Tai node, not for the browser UI. Pass it to the host agent and never store or log it in the UI.

`lease_source` and `lease_expires_at` are administrator-only inputs. `visibility=global` is administrator-only.

### 2.2 List and detail

```http
GET /v1/computers?page=1&page_size=20
```

List response is wrapped in `data`:

```json
{"data": [{"id": "cmp_...", "node_id": "node-...", "status": "online"}]}
```

Detail `GET /v1/computers/{id}` returns one Computer directly, including
the structured hardware snapshot reported by Tai when the node is online:

```json
{
  "id": "cmp_...",
  "status": "online",
  "hardware": {
    "os": "darwin",
    "arch": "arm64",
    "cpu_cores": 12,
    "memory_mb": 32768,
    "disk_free_mb": 512000,
    "gpus": [],
    "capabilities": ["session"],
    "container_runtimes": [{"name": "docker", "version": "27", "socket": "/var/run/docker.sock"}],
    "tai_version": "2.0.0"
  },
  "ip": "192.168.3.110",
  "networks": [
    {"id": "en0", "name": "en0", "kind": "lan", "family": "ipv4", "address": "192.168.3.110", "prefix_length": 24, "primary": true}
  ],
  "last_seen_at": "2026-11-25T10:00:00Z"
}
```

`gpus` is present as an array when Tai reports GPUs; current Tai builds may
send an empty array. Missing values are omitted rather than filled with fake
zeros. `networks` may include loopback addresses; display the primary
non-loopback address by default. The list endpoint returns the same snapshot
shape as detail.

`PATCH` accepts:

```json
{"name": "New name", "visibility": "team"}
```

### 2.3 Lifecycle

```http
POST /v1/computers/{id}/disconnect
POST /v1/computers/{id}/reconnect
DELETE /v1/computers/{id}
```

- `disconnect` sets status `disconnected` and blocks new connections.
- `reconnect` moves the existing node back to `pending`; it does not mint a new node token.
- `DELETE` returns `204` and revokes the node token. Workspaces are not archived.

### 2.4 Lease renewal

```http
POST /v1/computers/{id}/renew
Content-Type: application/json

{"duration": "24h"}
```

or:

```json
{"expires_at": "2027-01-01T00:00:00Z"}
```

The current `/renew` handler forwards `lease_source` for an owner; do not expose it in a normal user UI and treat it as administrator-only. If an expired Computer is renewed with `"issue_node_token": true`, the response carries a fresh node token; treat it as a secret. Re-read the Computer via `GET /v1/computers/{id}` for canonical status.

---

## 3. Node-token lifecycle (host agent)

These endpoints are for the host-side Tai agent, not the browser UI.

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/v1/auth/node-tokens/refresh` | `Authorization: Bearer <node_token>` | rotate/refresh the node token |
| `POST` | `/v1/auth/node-tokens/rotate` | User owner/admin | rotate a Computer's node token |
| `DELETE` | `/v1/auth/node-tokens/{jti}` | User owner/admin | revoke a node token by JTI |

Refresh request/response:

```http
POST /v1/auth/node-tokens/refresh
Authorization: Bearer <node-token>
```

```json
{
  "node_token": "<new node token>",
  "expires_at": "2026-11-24T00:00:00Z"
}
```

The old token is invalid after a successful refresh. Store the new token in the host's secure storage.

---

## 4. Status values

| Status | Meaning |
| --- | --- |
| `pending` | waiting for Tai to connect |
| `online` | connected and heartbeating |
| `offline` | lost connection; can normally reconnect |
| `disconnected` | intentionally disconnected; new connections blocked |
| `provisioning` | cloud host is being created |
| `expired` | lease expired; node token revoked |
| `released` | deleted/released |

---

## 5. Errors

| Status | Meaning | UI/agent action |
| --- | --- | --- |
| `400` | invalid request, lease duration, or fields | show validation error |
| `401` | missing/invalid user or node token | send user to login, or refresh node token |
| `403` | not owner/admin or forbidden field | hide action / show permissions error |
| `404` | Computer not found | refresh list |
| `501` `not_supported` | current node/SDK cannot provide data | show unavailable state |
| `204` | delete succeeded | remove from list |

---

## 6. Authentication dependency

Computer APIs use the same user session as the rest of Yao:

- Web: HttpOnly cookies from [Web authentication](../auth/web.md).
- Desktop/mobile: OAuth bearer access token from [Desktop](../auth/desktop.md) or [Mobile](../auth/mobile.md).

The host-side Tai agent uses the Computer's `node_token` for node-token endpoints, not the user's access token.

---

## 7. Related documents

- [Computer connections](computer-connections.md) — lifecycle, state flow, and frontend/host responsibilities.
- [Web authentication](../auth/web.md) — browser session.
- [Desktop authentication](../auth/desktop.md) — native desktop OAuth.
- [Mobile authentication](../auth/mobile.md) — iOS/Android OAuth.
- [Authentication API inventory](../auth/api-endpoints.md) — login endpoints.

