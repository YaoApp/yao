# OTP — Passwordless Authentication

One-time password (OTP) module for passwordless login. An authorized caller generates a short-lived code bound to a user/member and a redirect URL. The recipient opens `/v/<code>` in a browser to authenticate without credentials.

## Process

| Process | Args | Returns | Description |
|---|---|---|---|
| `otp.Create` | `params` (map) | code (string) | Generate an OTP code |
| `otp.Verify` | `code` | payload (map) | Look up a code without consuming it |
| `otp.Login` | `code`, `locale?` | LoginResult | Verify code, issue access token, optionally consume |
| `otp.Revoke` | `code` | nil | Delete a code immediately |

### otp.Create

```
yao run otp.Create '::{"team_id":"T1","member_id":"M1","redirect":"/dashboard"}'
```

**Parameters:**

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `team_id` | string | When `member_id` set | — | Team context |
| `member_id` | string | Either this or `user_id` | — | Target member (resolved to user_id at login) |
| `user_id` | string | Either this or `member_id` | — | Target user |
| `redirect` | string | Yes | — | Post-login redirect URL |
| `expires_in` | int | No | 86400 | Code TTL in seconds |
| `token_expires_in` | int | No | system default | Access token lifetime override (seconds) |
| `scope` | string | No | — | Space-separated scopes for the issued token |
| `consume` | bool | No | true | Revoke code after first login |

### otp.Verify

```
yao run otp.Verify abc123def456
```

Returns the stored payload without consuming the code.

### otp.Login

```
yao run otp.Login abc123def456 en-US
```

Verifies the code, resolves identity (`member_id` → `user_id` if needed), issues an access token (no refresh token), and returns `LoginResult`. Consumes the code if `consume` is true.

### otp.Revoke

```
yao run otp.Revoke abc123def456
```

Deletes the code. Silent if the code does not exist.

## HTTP API

| Method | Path | Auth | Description |
|---|---|---|---|
| ~~`POST`~~ | ~~`/otp/create`~~ | ~~Bearer token~~ | **Disabled** — use `otp.Create` process instead |
| `POST` | `/otp/login` | Public | Verify code, set session cookies |

> **Note:** The `/otp/create` HTTP endpoint is intentionally disabled. Exposing it would allow any team member to generate OTP codes for other members, effectively logging in as them without credentials. OTP codes must be created server-side via the `otp.Create` process only.

### POST /otp/login

Public endpoint. Checks for an existing valid session first — if found, returns `already_logged_in` without issuing new tokens. Otherwise performs login and sets `access_token` cookie (no `refresh_token`).

**Request:**
```json
{"code": "abc123def456", "locale": "en-US"}
```

**Response:**
```json
{"status": "success", "redirect": "/agents/keeper/entry/xxx"}
```

Status is either `success` (new session) or `already_logged_in` (existing session).

## Security

- Codes are 12-char NanoID (`[2-9a-hjkmnp-z]`), ~62 bits of entropy
- Default TTL: 24 hours
- Codes are single-use by default (`consume: true`)
- No refresh token issued — access token only
- `POST /otp/create` enforces team membership validation
- `team_id` is always derived from the caller's token, not the request body
