# Package `api`

HTTP handlers for Tai node registration, heartbeat, and unregistration. Built on [Gin](https://github.com/gin-gonic/gin), these handlers are mounted on the Yao server to allow remote Tai instances to register themselves.

## Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `POST` | `/tai-nodes/register` | `HandleRegister` | Register a Tai node |
| `POST` | `/tai-nodes/heartbeat` | `HandleHeartbeat` | Update heartbeat timestamp |
| `DELETE` | `/tai-nodes/register/:tai_id` | `HandleUnregister` | Remove a Tai node |

All endpoints require a `Bearer` token in the `Authorization` header. Tokens are validated via the Yao OAuth service.

## Authentication

```
Authorization: Bearer <access_token>
```

The token is validated against `oauth.OAuth.AuthenticateToken()`. On success, an `AuthInfo` is extracted containing `Subject`, `UserID`, `ClientID`, `Scope`, `TeamID`, and `TenantID`. The `ClientID` is used for ownership checks on heartbeat and unregister.

## Endpoints

### POST /tai-nodes/register

Registers a new Tai node in the global registry.

**Request Body:**

```json
{
  "tai_id":     "tai-abc123",
  "machine_id": "m-001",
  "version":    "1.2.0",
  "addr":       "192.168.1.100",
  "ports":      {"grpc": 19100, "http": 8099, "vnc": 16080, "docker": 12375},
  "capabilities": {"docker": true, "host_exec": true},
  "system": {
    "os": "linux",
    "arch": "amd64",
    "hostname": "docker-host-01",
    "num_cpu": 16,
    "total_mem": 34359738368
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tai_id` | string | yes | Unique identifier for this Tai instance |
| `machine_id` | string | no | Host machine identifier |
| `version` | string | no | Tai version string |
| `addr` | string | no | Reachable address of the Tai server |
| `ports` | map[string]int | no | Service ports (grpc, http, vnc, docker, k8s) |
| `capabilities` | map[string]bool | no | Supported features (docker, k8s, host_exec) |
| `system` | object | no | Host system information |

**Response (200):**

```json
{
  "status":    "registered",
  "tai_id":    "tai-abc123",
  "remote_ip": "203.0.113.50"
}
```

**Errors:**

| Code | Condition |
|------|-----------|
| 400 | Missing `tai_id` or invalid JSON body |
| 401 | Missing or invalid Bearer token |
| 500 | Registry not initialized |

### POST /tai-nodes/heartbeat

Updates the `LastPing` timestamp for a registered node. The node's `ClientID` must match the token's `ClientID`.

**Request Body:**

```json
{
  "tai_id": "tai-abc123"
}
```

**Response (200):**

```json
{
  "status": "ok"
}
```

**Errors:**

| Code | Condition |
|------|-----------|
| 400 | Missing `tai_id` or invalid JSON body |
| 401 | Missing or invalid Bearer token |
| 403 | `tai_id` belongs to a different client |
| 404 | `tai_id` not found in registry |
| 500 | Registry not initialized |

### DELETE /tai-nodes/register/:tai_id

Removes a registered node. The node's `ClientID` must match the token's `ClientID`.

**Response (200):**

```json
{
  "status": "unregistered"
}
```

**Errors:**

| Code | Condition |
|------|-----------|
| 400 | Missing `tai_id` path parameter |
| 401 | Missing or invalid Bearer token |
| 403 | `tai_id` belongs to a different client |
| 404 | `tai_id` not found in registry |
| 500 | Registry not initialized |

## Node Mode

Nodes registered via this HTTP API are marked with `Mode: "direct"`. This means the Yao server can reach the Tai instance directly over the network. For tunnel-mode nodes (registered via WebSocket), see [registry.md](registry.md).

## Health Check

The registry runs a background health checker (started via `Registry.StartHealthCheck`). Direct-mode nodes that miss heartbeats beyond the configured timeout are marked `"offline"`. Nodes that remain offline longer than the cleanup threshold are automatically unregistered.
