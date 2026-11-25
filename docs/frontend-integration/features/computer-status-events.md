# Computer Status Events and Fallback Status

**Audience:** frontend engineers and host agents building Computer status UI.

**Scope:** the Computer event channel, desktop WebSocket ticket authentication,
and the lightweight status endpoint used when an event connection is unavailable.

## 1. Event contract

Computer changes are published through the existing `/v1/events` WebSocket.

```text
computer.created
computer.status
```

Both events carry only a summary:

```json
{
  "computer_id": "cmp_...",
  "status": "online"
}
```

`computer.status` is emitted when the persisted status changes, including:

```text
pending, provisioning, online, offline, disconnected, expired, error, released
```

Events are scoped to the current user or team. A different user's Computer
events are not pushed to the connection. Events are not replayed; after a
disconnect, use the status endpoint or the Computer list to resynchronize.

Required token scopes for the WebSocket:

```text
event:events:read:own
event:events:read:team
```

## 2. Browser / Cookie WebSocket

Browsers can use the existing Cookie-authenticated endpoint:

```text
wss://<yao-host>/v1/events
```

No query token is needed. The access token must include an event read scope.

## 3. Desktop / Bearer WebSocket ticket

A native WebSocket cannot set an `Authorization` header. Exchange the Bearer
session for a one-time, short-lived ticket first:

```http
POST /v1/events/ticket
Authorization: Bearer <access-token>
```

```json
{
  "ticket": "opaque-one-time-ticket",
  "expires_in": 30
}
```

Open the WebSocket with the ticket in a subprotocol (preferred; the ticket does
not appear in the URL or access logs):

```text
new WebSocket("wss://<yao-host>/v1/events", [
  "yao-events",
  "yao-ticket.<ticket>"
])
```

Query fallback is accepted for clients that cannot set subprotocols:

```text
wss://<yao-host>/v1/events?ticket=<ticket>
```

Rules:

- the ticket is one-time and expires in 30 seconds;
- it is bound to the user/team/session that requested it;
- never put a long-lived access token in a WebSocket URL;
- request a fresh ticket for each reconnect.

## 4. Lightweight status fallback

```http
GET /v1/computers/{computer_id}/status
Authorization: Bearer <access-token>
```

Owner/admin only. Response:

```json
{
  "computer_id": "cmp_...",
  "status": "online"
}
```

It returns no hardware, metadata, or node-token fields. Use it only for
fallback polling after an event disconnect, not as a high-frequency full-detail
poll. The full Computer is still available at `GET /v1/computers/{id}`.

Required token scopes:

```text
computer:computers:read:own
computer:computers:read:team
```
