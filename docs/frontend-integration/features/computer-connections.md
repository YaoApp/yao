# Computer Connections

**Audience:** frontend engineers and coding agents building the Web/desktop/mobile UI for Yao 2.0, and engineers writing the host-side agent that runs a local Tai node.

**Scope:** the Computer API, the one-time `node_token`, the Tai node handshake, connection lifecycle, and the authenticated UI calls around it. It describes the current Yao 2.0 implementation.

> **Environment:** see [Environment and API endpoints](../environment.md). All URLs in this guide are placeholders; use the base URL supplied by your Yao operator.

**Companion guides:** [Web authentication](../auth/web.md) · [Desktop authentication](../auth/desktop.md) · [Mobile authentication](../auth/mobile.md) · [Computer API inventory](computer-api.md).

**Status:** written against Yao `main` on 2026-11-17.

---

## 1. What a Computer is

A Yao **Computer** is a record of a Tai node, not a physical device.

- The same machine can run multiple Tai nodes, producing multiple Computer rows.
- A Computer has a server-generated `node_id` and a one-time `node_token`.
- The node token is used by the Tai process, not by a browser UI.
- `status` is the Yao-side view: `pending`, `online`, `offline`, `disconnected`, `expired`, and so on.

### 1.1 Who does what

| Component | Responsibility |
| --- | --- |
| Web/Android UI | lists computers, shows status, edits names/visibility, disconnect/reconnect, renew, opens workspaces |
| BFF or native host agent | creates a Computer (`POST /v1/computers/connect`), starts Tai, holds the `node_token` |
| Tai node | connects to Yao transport on port `6099`, registers with the node token |
| Yao | validates the node token, maintains status, exposes runtimes/workspaces |

The browser UI must not persist or log `node_token`. If the UI creates a Computer directly, hand the token immediately to the host-side process and remove it from UI state. For a pure browser UI, put `POST /connect` behind a BFF.

---

## 2. Authentication and origin

All Computer endpoints require a Yao user session.

```http
GET /v1/computers
Cookie: session_id=...; access_token=Bearer ...
```

or:

```http
GET /v1/computers
Authorization: Bearer <raw-access-token>
```

Use the same-origin/BFF model described in the [Web authentication guide](../auth/web.md#11-prefer-a-same-origin-deployment). Browser requests must use:

```ts
fetch("/v1/computers", { credentials: "include" })
```

The OpenAPI does not provide a general CORS setup for cross-origin cookie sessions. Serve the UI and `/v1` from one origin, or use a BFF.

Use the same base URL and first-party session for Computer API calls.

---

## 3. Transport discovery

The public discovery endpoint tells a host how to reach the Tai transport:

```http
GET /.well-known/yao-transport
```

Example response:

```json
{
  "transport_port": 6099,
  "protocols": ["HTTP/2", "HTTP/3"],
  "version": "yao"
}
```

The transport port is deployment-specific; the value above is an example. The OpenAPI gRPC listener is separate and may be disabled. Do not assume the transport port is gRPC for a frontend HTTP client.

The actual Tai node connects over the Yao HTTP server plus the transport port. The frontend does not connect to `6099` directly; it manages the Computer and observes status.

---

## 4. Create a computer and start Tai

### 4.1 Request

```http
POST /v1/computers/connect
Content-Type: application/json

{
  "name": "Max's MacBook",
  "data_dir": "/Users/max/Documents/YaoAgents/Volumes",
  "visibility": "private"
}
```

`scope` is derived from the authenticated user/team; do not send it.

`lease_source` and `lease_expires_at` are admin-only fields. Sending them as a normal user returns `403`.

### 4.2 Response

The response is direct JSON (there is no outer envelope):

```json
{
  "computer": {
    "id": "cmp_xxx",
    "node_id": "node-26928183",
    "name": "Max's MacBook",
    "provider_type": "local",
    "scope": "user",
    "visibility": "private",
    "status": "pending",
    "user_id": "user_xxx"
  },
  "computer_id": "cmp_xxx",
  "node_id": "node-26928183",
  "node_token": "eyJ...",
  "expires_at": "2026-11-24T00:00:00Z",
  "server": "http://<yao-host>:<port>",
  "transport_port": 6099,
  "protocols": ["HTTP/2", "HTTP/3"]
}
```

Treat `node_token` as a secret:

- return it only to the host-side Tai process;
- never put it in browser storage, analytics, logs, or crash reports;
- do not expose it in a rendered page;
- it expires with `expires_at`.

The new Computer starts in `pending`. It becomes `online` after a Tai node registers with Yao using this token.

`server` is currently built from Yao's listen host/port (`YAO_HOST`/`YAO_PORT`) and is always formatted as `http://<host>:<port>`. When Yao listens on `0.0.0.0`, it becomes `http://127.0.0.1:<port>`, which is not reachable from another device. A host on another machine must replace `server` with the deployment's reachable base URL (for example, the HTTPS public URL), or the deployment must set `YAO_HOST` to a reachable address. There is no separate public-URL override in this response path.

### 4.3 Start the Tai node

The host-side agent uses the response values. The integration tests and current Tai runtime use this shape:

```bash
tai start \
  --node-id "$node_id" \
  --node-token "$node_token" \
  --server "$server" \
  --data "$data_dir"
```

Check the Tai version shipped with your deployment with `tai help start`; flag names are owned by Tai. The important values are the generated `node_id`, the one-time `node_token`, the Yao `server` URL, and the workspace data directory.

Then poll the Computer:

```http
GET /v1/computers/{computer_id}
```

Recommended UI polling is every 2-5 seconds while the status is `pending`/`offline`, with a timeout and a manual refresh action. A push event channel is available; see [Computer status events](computer-status-events.md). Use `GET /v1/computers/{id}/status` as the lightweight fallback when the event connection is unavailable.

### 4.4 Connection state flow

```text
POST /computers/connect
        |
        v
     pending
        |
        | Tai connects with node_token
        v
     online  <------------------+
        |                       |
        | heartbeat lost        | Tai reconnects
        v                       |
     offline -------------------+

owner action:
  online/offline -> POST /disconnect -> disconnected
  disconnected   -> POST /reconnect  -> pending
```

Meaning:

| Status | Meaning |
| --- | --- |
| `pending` | waiting for the Tai node to connect |
| `online` | node is connected and heartbeating |
| `offline` | node was connected but stopped heartbeating |
| `disconnected` | owner intentionally disconnected; new connections are blocked |
| `provisioning` | cloud host is still being created |
| `expired` | lease expired; node token was revoked |
| `released` | deleted/released |

An `offline` node can normally reconnect without a new `node_token`. A `disconnected` node must be set back to `pending` via `/reconnect`. An `expired` node cannot reconnect until a new token is issued.

---

## 5. Computer API

| Method | Path | Purpose | Success |
| --- | --- | --- | --- |
| `GET` | `/v1/computers` | list visible computers | `200`, `{"data":[...]}` |
| `POST` | `/v1/computers/connect` | create Computer, node id, node token | `200`, `ConnectResult` |
| `GET` | `/v1/computers/{computer_id}` | get one Computer | `200`, `Computer` |
| `PATCH` | `/v1/computers/{computer_id}` | rename or change visibility | `200`, `Computer` |
| `DELETE` | `/v1/computers/{computer_id}` | delete and revoke node token | `204`, no body |
| `POST` | `/v1/computers/{computer_id}/disconnect` | block new connections | `200`, `Computer` |
| `POST` | `/v1/computers/{computer_id}/reconnect` | allow the existing node again | `200`, `Computer` |
| `POST` | `/v1/computers/{computer_id}/renew` | extend/restore a lease | `200`, lease result |
| `GET` | `/v1/computers/{computer_id}/workspaces` | list node workspaces | `200` or `501` |
| `GET` | `/v1/computers/{computer_id}/runtimes` | list node container runtimes | `200` or `501` |
| `GET` | `/v1/computers/{computer_id}/containers` | list containers | `501` in the current SDK |

### 5.1 List

```http
GET /v1/computers?page=1&page_size=20
```

Response:

```json
{
  "data": [
    {
      "id": "cmp_xxx",
      "node_id": "node-26928183",
      "name": "Max's MacBook",
      "provider_type": "local",
      "scope": "user",
      "visibility": "private",
      "status": "online",
      "user_id": "user_xxx",
      "created_at": "2026-11-17T00:00:00Z",
      "updated_at": "2026-11-17T00:00:00Z"
    }
  ]
}
```

For a non-admin, the current handler filters by the caller's user id and current team id; it does not return every Computer shared with that team. Admins may list all; query fields are `page` and `page_size`.

### 5.2 Get and update

```http
GET /v1/computers/cmp_xxx
```

The detail response is a single Computer object.

```http
PATCH /v1/computers/cmp_xxx
Content-Type: application/json

{"name": "Build machine", "visibility": "team"}
```

`visibility` may be `private`, `members`, `team`, or `global`. `global` is administrator-only.

### 5.3 Disconnect and reconnect

```http
POST /v1/computers/cmp_xxx/disconnect
```

The response is the updated Computer with `status: "disconnected"`.

```http
POST /v1/computers/cmp_xxx/reconnect
```

The response is the updated Computer with `status: "pending"`. The existing node id/token are reused; reconnect is not token renewal.

### 5.4 Renew a lease

```http
POST /v1/computers/cmp_xxx/renew
Content-Type: application/json

{"duration": "24h"}
```

or:

```json
{"expires_at": "2027-01-01T00:00:00Z"}
```

- At least one of `duration` and `expires_at` is required.
- `Connect` rejects `lease_source`/`lease_expires_at` for non-admin users. The current `/renew` handler forwards `lease_source` for an owner; do not expose it in a normal user UI, and treat it as administrator-only.
- When an expired Computer is renewed with `"issue_node_token": true`, the response contains a new node token. Treat it as a secret and hand it directly to the host agent; do not persist it in UI state.
- After renewal, re-read `GET /v1/computers/{id}` for the canonical Computer status. This avoids depending on lease-result field naming in a frontend model.

### 5.5 Delete

```http
DELETE /v1/computers/cmp_xxx
```

Returns `204 No Content`. The node token is revoked. Workspaces are not archived or deleted by this call.

---

## 6. Runtimes, workspaces, containers

These endpoints are available only when a Tai node provider is registered and the node is connected.

### Workspaces

```http
GET /v1/computers/cmp_xxx/workspaces
```

The response is the current workspace list from the node.

### Runtimes

```http
GET /v1/computers/cmp_xxx/runtimes
```

The response lists container runtimes advertised by the Tai node, for example:

```json
[
  {"name": "docker", "version": "28.0.0", "ready": true}
]
```

### Containers

```http
GET /v1/computers/cmp_xxx/containers
```

The current Tai SDK does not expose a container-list API, so this endpoint returns:

```http
501 Not Implemented
```

```json
{
  "error": "not_supported",
  "error_description": "computer capability is not supported"
}
```

A `501` here is expected; render an unavailable/empty state, not a hard error. The same `501` can be returned for runtimes/workspaces when the provider has no data source.

---

## 7. Node token lifecycle

The host-side agent owns the node token.

- `POST /v1/computers/connect` returns a new token.
- `POST /v1/computers/{id}/renew` can return a new token when the lease was expired and `issue_node_token` is set.
- Token expiration is `expires_at` in the connect response (the current default node-token TTL is seven days).
- If the token is near expiry, the host agent refreshes it:

```http
POST /v1/auth/node-tokens/refresh
Authorization: Bearer <node_token>
```

The response contains the rotated `node_token` and its `expires_at`. The old token is invalid after a successful refresh.

Do not call this endpoint from a browser UI. It is a node-agent operation.

---

## 8. Permissions and sharing

- The current Computer HTTP handlers authorize the owner and administrators only; any other user gets `403`.
- `visibility` and `scope` are stored and can be set, but these handlers do not currently implement member/team access-grant checks. Do not treat `visibility` as an authorization guarantee in the UI.
- `visibility=global` is administrator-only at write time.
- `scope` is `user` or `team`; it is not a visibility value.
- A Computer that is `disconnected` rejects new connections until `/reconnect`.
- A Computer that is `expired` rejects connections until renewed with a new node token.

When building the UI, treat `online` as a capability signal and `runtimes/workspaces` as optional details; a connected UI must handle `501` and empty lists.

---

## 9. Web and Android patterns

### Web SPA

- List/get computers via same-origin `/v1`.
- Poll during `pending`/`offline`.
- Call patch/disconnect/reconnect/renew from the UI.
- Put `POST /connect` and Tai startup behind a BFF or native host agent if possible.
- Never render or log `node_token`.

### Android

- The Android app can manage Computer records via a BFF, or use the system-browser OAuth flow described in [Mobile authentication](../auth/mobile.md).
- The Android app can itself be the Tai host only if it can run the Tai runtime and start it with the returned `node_id`, `node_token`, `server`, and data directory; current Tai support depends on the Tai build for that platform.
- Keep the node token in Android Keystore-backed secure storage, never in plain preferences or logs.
- If the frontend uses a native HTTP stack, make sure its cookie/session handling matches the WebView login; otherwise use a BFF.

If a test deployment uses plain HTTP, allow cleartext only for that test host in debug builds. Production requires HTTPS.

---

## 10. Errors

Error responses are direct JSON:

```json
{"error": "forbidden", "error_description": "..."}
```

| HTTP | Meaning | UI action |
| --- | --- | --- |
| `401` | missing/invalid session | send to login |
| `403` | not owner/admin, local-login guard, or forbidden field | hide action or show permission message |
| `400` | invalid request/lease duration | show validation error |
| `404` | Computer not found | remove from list, refresh |
| `501` `not_supported` | current node/SDK has no such data | show unavailable state |
| `204` | delete succeeded | remove from list |

Do not retry `403` or `400` automatically. Retry transient `5xx` with backoff.

---

## 11. Smoke checklist

```bash
export YAO_BASE_URL=https://<your-yao-host>

# 1. Transport metadata
curl -fsS "$YAO_BASE_URL/.well-known/yao-transport" | jq

# 2. Authenticated list (use a logged-in cookie jar)
curl -fsS -b /tmp/yao-cookies.txt "$YAO_BASE_URL/v1/computers" | jq

# 3. Create a local computer and inspect the one-time token
curl -fsS -b /tmp/yao-cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"name":"Test host","data_dir":"/tmp/yao-test-host","visibility":"private"}' \
  "$YAO_BASE_URL/v1/computers/connect" | jq

# 4. Start Tai on the host with the returned node_id/node_token/server.
# 5. Poll:
curl -fsS -b /tmp/yao-cookies.txt "$YAO_BASE_URL/v1/computers/<computer_id>" | jq '.status'
# pending -> online
```

---

## 12. Implementation references

| Behavior | Source |
| --- | --- |
| Computer HTTP routes and handlers | `openapi/computers/computers.go` |
| Connect/Renew/lease logic | `modules/computer/computer.go`, `modules/computer/lease.go` |
| Computer and transport response types | `modules/computer/types/computer.go` |
| Tai transport metadata | `openapi/well-known.go`, `tai/server.go` |
| Auth guard and session | `openapi/oauth/guard.go` |
| Computer and node-token design | See the OpenAPI/Computer documentation for the deployed Yao version. |

