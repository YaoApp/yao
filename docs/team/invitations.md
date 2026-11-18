# Team Invitations API

**Audience:** Web and native frontend engineers integrating team invitations.

**Base URL:** the Yao OpenAPI origin plus `/v1`.

**Related:** [Web authentication](../frontend-integration/auth/web.md) · [Authentication API inventory](../frontend-integration/auth/api-endpoints.md) · [Environment](../frontend-integration/environment.md).

> **Current contract (R-9):** accepting an invitation requires only a signed-in
> user. It does not require `user:teams:write:public`, and an invitation link can
> be accepted before choosing an existing team.

## 1. Invitation lifecycle

There are three different actions:

| Action | Endpoint | Auth |
| --- | --- | --- |
| Preview an invitation | `GET /v1/user/teams/invitations/:invitation_id` | public |
| Accept an invitation | `POST /v1/user/teams/invitations/:invitation_id/accept` | signed-in user |
| Manage invitations | `/v1/user/teams/:team_id/invitations...` | team owner/admin |

The invitation link is a capability link: possession of the invitation token is
the proof used to accept it. Yao does not require the signed-in account's email
to match the invited email.

## 2. Preview an invitation

```http
GET /v1/user/teams/invitations/inv_abc123?locale=en
```

No authentication is required. Use this before login to render the invitation
landing page.

Response:

```json
{
  "invitation_id": "inv_abc123",
  "team_name": "Acme",
  "team_logo": "",
  "team_description": "Product team",
  "role_label": "Member",
  "status": "pending",
  "invited_at": "2026-11-18T08:00:00Z",
  "invitation_expires_at": "2026-11-25T08:00:00Z",
  "message": "Join our team",
  "inviter_info": {
    "user_id": "user_123",
    "name": "Alice",
    "picture": ""
  }
}
```

The preview never returns the invitation token.

## 3. Accept an invitation

The frontend should keep the invitation URL and token while sending the user
through login, then return to the same invitation page.

```http
POST /v1/user/teams/invitations/inv_abc123/accept
Cookie: session_id=...
Content-Type: application/json

{"token": "<invitation-token>"}
```

The access token may also be sent as `Authorization: Bearer <access_token>`.

**Do not send the user to `/app/team/select` before this call.** A multi-team
user with only a temporary `builtin:teams:selection` token may call accept
directly; the server switches the session to the invited team. Accept does not
require an invitation-management scope; the signed-in identity and invitation
token are sufficient.

Success: `200`, matching the normal login response.

```json
{
  "user_id": "user_123",
  "subject": "1234567890123456",
  "access_token": "...",
  "id_token": "...",
  "refresh_token": "...",
  "expires_in": 7200,
  "refresh_token_expires_in": 604800,
  "token_type": "Bearer",
  "scope": "openid profile ..."
}
```

Errors:

| HTTP | Meaning | Frontend behavior |
| --- | --- | --- |
| `400` | missing/invalid token | show error, do not retry automatically |
| `401` | not signed in | send to login, return to invitation page |
| `404` | invitation not found, expired, or already accepted | show invalid/expired invitation |
| `409` | user is already a member of the team | use `team_id` to enter/switch team |

The `409` response carries the target team when available:

```json
{
  "error": "already_member",
  "error_description": "User is already a member of this team",
  "team_id": "617403793931",
  "team_name": "Acme"
}
```

## 4. Invitation management

Management endpoints require the current user to be an owner/admin of the team.
They are not available to ordinary members.

| Method | Endpoint | Scopes |
| --- | --- | --- |
| `GET` | `/v1/user/teams/:team_id/invitations` | `user:invitations:read:own` or `user:invitations:read:team` |
| `POST` | `/v1/user/teams/:team_id/invitations` | `user:invitations:write:own` or `user:invitations:write:team` |
| `GET` | `/v1/user/teams/:team_id/invitations/:invitation_id` | `user:invitations:read:own` or `user:invitations:read:team` |
| `PUT` | `/v1/user/teams/:team_id/invitations/:invitation_id/resend` | `user:invitations:write:own` or `user:invitations:write:team` |
| `DELETE` | `/v1/user/teams/:team_id/invitations/:invitation_id` | `user:invitations:delete:own` or `user:invitations:delete:team` |

### 4.1 List invitations

```http
GET /v1/user/teams/617403793931/invitations?status=pending&page=1&pagesize=20
```

Query parameters:

| Name | Default | Meaning |
| --- | --- | --- |
| `status` | `pending` | invitation status filter |
| `page` | `1` | page number |
| `pagesize` | `20` | page size, max 100 |

Response:

```json
{
  "data": [
    {
      "invitation_id": "inv_abc123",
      "team_id": "617403793931",
      "role_id": "user",
      "status": "pending",
      "invited_at": "2026-11-18T08:00:00Z",
      "invitation_expires_at": "2026-11-25T08:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "pagesize": 20
}
```

### 4.2 Create an invitation

```http
POST /v1/user/teams/617403793931/invitations
Content-Type: application/json

{
  "user_id": "user_123",
  "email": "invitee@example.com",
  "member_type": "user",
  "role_id": "user",
  "message": "Join our team",
  "expiry": "7d",
  "send_email": false,
  "locale": "en"
}
```

`role_id` is required. `user_id` is optional for an unregistered invitee; `email`
is required when the user is not identified by `user_id`. `member_type` defaults
to `user`. `send_email` defaults to `false`.

Success: `201` with the invitation response, including `invitation_id`,
`invitation_token`, `invitation_link`, `status`, and expiry fields.

Errors include `403` when the current user is not an owner/admin, `409` when the
user is already a member or already has a pending invitation, and `400` for an
invalid request.

### 4.3 Get, resend, cancel

- `GET /v1/user/teams/:team_id/invitations/:invitation_id` returns the invitation
  detail (including the invitation token/link for an owner/admin).
- `PUT /v1/user/teams/:team_id/invitations/:invitation_id/resend` accepts an
  optional `{"locale": "en"}` body or `?locale=en` and returns
  `{"message": "Invitation resent successfully"}`.
- `DELETE /v1/user/teams/:team_id/invitations/:invitation_id` cancels a pending
  invitation and returns `{"message": "Invitation cancelled successfully"}`.

## 5. Frontend flow

```text
open /app/team/join/:invitation_id/:token
  -> GET public preview
  -> not signed in: "Sign in and join" -> login
  -> login returns to the same URL
  -> signed in: POST accept {token}
       200 -> /app/new-session
       409 -> use team_id; POST /user/teams/select; enter team
       401 -> login again
       400/404 -> error state
```

Do not call `/user/teams/select` before accept. The accept handler can switch the
session to the invited team even when the user has no current team.

## 6. Deployment notes

- Web sessions must be allowed to use the invitation scopes in their OpenAPI
  client configuration.
- Invitation management requires `user:invitations:read/write/delete:own` or the
  corresponding `:team` scopes; ordinary members cannot manage invitations.
- The public preview endpoint intentionally returns display data only and no
  invitation token.
- The `/v1/user/invitations/:token` placeholder routes are not part of this
  integration.
