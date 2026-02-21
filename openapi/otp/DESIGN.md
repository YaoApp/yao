# OTP Passwordless Authentication

## Overview

OTP (One-Time Password) provides passwordless authentication via magic links.
AI or system generates a short link `https://host/<prefix>/v/<code>`, user clicks it,
CUI verifies the code against the backend, backend issues tokens, and frontend redirects.

## Architecture

```
AI/System                     Yao Backend                      CUI Frontend
   |                              |                                |
   |-- otp.Create(params) ------->|                                |
   |<---- code (nanoid 12) -------|                                |
   |                              |                                |
   | (compose link, send to user) |                                |
   |                              |                                |
   |                              |     GET /<prefix>/v/<code>     |
   |                              |<-------------------------------|
   |                              |                                |
   |                              |  POST /api/otp/login {code}    |
   |                              |<-------------------------------|
   |                              |                                |
   |                              |-- otp.Login(code) ------------>|
   |                              |   ├─ Verify code in store      |
   |                              |   ├─ Resolve identity          |
   |                              |   ├─ LoginByTeamID → tokens    |
   |                              |   └─ SendLoginCookies          |
   |                              |                                |
   |                              |  {redirect} ----------------->|
   |                              |                                |
   |                              |              window.location = redirect
```

## Design Principles

1. **Four clean APIs**: Create, Verify, Login, Revoke — each with a single responsibility.
2. **Verify is pure**: returns stored payload, no side effects.
   Login is the full flow: verify + identity resolution + token issuance.
3. **Package location `openapi/otp/`**: can freely import `user` and `oauth`.
   Dependency chain is one-directional: `openapi/otp → user → oauth` (no cycles).
4. **Shared store, unified key namespace**: reuses OAuth's store.Store,
   keys under `{prefix}oauth:otp:{code}` alongside refresh_token/access_token.

## Package Structure

```
openapi/
  otp/
    DESIGN.md          ← this file
    otp.go             ← Service, Payload, NewService
    generate.go        ← Create (nanoid + collision check + store.Set)
    verify.go          ← Verify (store.Get + type coercion)
    revoke.go          ← Revoke (store.Del)
    login.go           ← Login (Verify + resolve identity + issue tokens)
    handler.go         ← GinOTPCreate + GinOTPLogin HTTP handlers + Attach(group, oauth)
    process.go         ← Yao processors: otp.Create, otp.Verify, otp.Login, otp.Revoke

  openapi.go           ← init OTP service in Load(), register route in Attach()
```

```
cui/packages/cui/
  openapi/user/auth.ts ← add OTPLogin method
  pages/auth/v/$.tsx   ← OTP verification page (route: /v/<code>)
```

## Dependency Graph

```
openapi/otp
  ├── imports user      (LoginByTeamID, LoginWithOptions, SendLoginCookies)
  ├── imports oauth     (OAuth.GetUserProvider, OAuth.GetStore for identity resolution)
  └── imports store     (store.Store for code persistence)

user
  └── imports oauth     (existing, unchanged)
```

One-directional: `openapi/otp → user → oauth`. No cycles.

## Data Structures

### Payload (stored in store)

```go
type Payload struct {
    TeamID   string `json:"team_id,omitempty"`
    MemberID string `json:"member_id,omitempty"`
    UserID   string `json:"user_id,omitempty"`
    Redirect string `json:"redirect"`
    Scope    string `json:"scope,omitempty"`
}
```

### GenerateParams

```go
type GenerateParams struct {
    TeamID    string
    MemberID  string
    UserID    string
    ExpiresIn int    // seconds, default 24h
    Redirect  string // required
    Scope     string // optional, space-separated
}
```

## Store Key Format

```
{prefix}oauth:otp:{code}
```

Example: `yao_:oauth:otp:abc123def456`

Consistent with existing OAuth keys:
- `{prefix}oauth:refresh_token:{token}`
- `{prefix}oauth:access_token:{token}`
- `{prefix}oauth:otp:{code}`

NanoID: 12 chars, alphabet `23456789abcdefghjkmnpqrstuvwxyz` (no ambiguous chars).
Collision check: retry up to 5 times.

## Processor Interface

Four processors, CRUD-style naming.

### otp.Create

```javascript
code = Process("otp.Create", {
  "user_id":    "user_xxx",                // required (or member_id)
  "team_id":    "team_xxx",                // optional
  "member_id":  "member_xxx",              // optional; when set, team_id is required
  "expires_in": 86400,                     // optional; seconds, default 24h
  "redirect":   "/chat",                   // required; target path after login
  "scope":      "read write"               // optional; space-separated scopes
})
// returns: "abc123def456" (string)
// developer composes the full link: `${host}/${prefix}/v/${code}`
```

Single map argument. Returns code string.

### otp.Verify

```javascript
payload = Process("otp.Verify", "abc123def456")
// returns: {
//   "team_id":   "team_xxx",
//   "member_id": "member_xxx",
//   "user_id":   "user_xxx",
//   "redirect":  "/chat",
//   "scope":     "read write"
// }
```

Pure validation. Returns stored Payload. Does NOT consume code (valid within TTL).
Use case: inspect payload before login, or use OTP for non-login purposes.

### otp.Login

```javascript
result = Process("otp.Login", "abc123def456", "zh-CN")
// returns: {
//   "access_token": "Bearer ...",
//   "id_token":     "eyJ...",
//   "redirect":     "/chat",
//   "expires_in":   3600,
//   ...
// }
```

Full login flow: verify code -> resolve identity -> issue tokens.
- args[0]: code (string, required)
- args[1]: locale (string, optional)

Internally:
1. Verify(code) -> Payload
2. Resolve identity (member_id -> user_id if needed)
3. LoginByTeamID or LoginWithOptions (when scope override)
4. Return LoginResponse + redirect

Does NOT set HTTP cookies (no gin.Context). The HTTP handler wraps this
and additionally calls SendLoginCookies.

### otp.Revoke

```javascript
Process("otp.Revoke", "abc123def456")
// returns: null
```

Immediately removes code from store. Silent on missing/expired.

## HTTP APIs

### POST /api/otp/create (protected)

Requires authentication (OpenAPI Guard). Permission managed by Scope/ACL.
The caller's `team_id` is forced from the authenticated identity; request body `team_id` is ignored.
The `member_id` must belong to the caller's team.

**Request:**
```json
{
  "member_id": "member_xxx",
  "user_id": "user_xxx",
  "expires_in": 86400,
  "redirect": "/chat",
  "scope": "read write"
}
```

**Success (200):**
```json
{ "code": "abc123def456" }
```

**Errors:** 400 (missing fields), 403 (member not in team), 500 (internal).

### POST /api/otp/login (public)

Public endpoint (no auth guard). The OTP code itself is the credential.

**Request:**
```json
{ "code": "abc123def456", "locale": "zh-CN" }
```

**Success (200):**
```json
{ "redirect": "/chat" }
```
Cookies set: `access_token`, `refresh_token`, `session_id`.

**Errors:** 400 (missing code), 401 (invalid/expired), 500 (internal).

### Handler Flows (handler.go)

#### GinOTPCreate
```
1. authorized.GetInfo(c) -> authInfo (teamID, userID)
2. Bind JSON -> request body
3. Force teamID from authInfo
4. Validate member belongs to team
5. service.Create(params) -> code
6. Respond {code}
```

#### GinOTPLogin
```
1. Bind JSON -> {code, locale}
2. service.Login(code, locale) -> LoginResponse + Payload
3. sessionID = utils.GetSessionID(c) or generateSessionID()
4. user.SendLoginCookies(c, loginResp, sessionID)
5. Respond {redirect: payload.Redirect}
```

The handlers are thin — business logic lives in the Service methods.

## Scope Override (user package change)

When `payload.Scope` is non-empty, `LoginByTeamID` cannot be used directly
because it resolves scopes internally. Add one function to `user` package:

```go
// user/types.go
type LoginOptions struct {
    Scopes []string
}

// user/login.go
func LoginWithOptions(userid, teamID string, loginCtx *LoginContext, opts *LoginOptions) (*LoginResponse, error)
```

Same logic as `LoginByTeamID`, uses `opts.Scopes` when non-nil.
This is the **only** change to `user` package.

## Initialization (openapi.go)

In `Load()`:
```go
otp.NewService(oauth.OAuth.GetStore(), oauth.OAuth.GetPrefix())
```

Note: Since oauth.Service.prefix is private, the OTP service constructs
the prefix independently using `share.App.Prefix` for consistency.

In `Attach()`:
```go
otp.Attach(group.Group("/otp"), openapi.OAuth)
```

## CUI Page (pages/auth/v/$.tsx)

```
1. Extract code from URL path: /v/<code>
2. Call userClient.auth.OTPLogin(code, locale)
3. On success:
   - Call GetProfile() to get UserInfo (cookies already set by backend)
   - AfterLogin(global, { user: profileData, entry: redirect })
   - window.location.href = redirect
4. On error: show error UI with "Go Back" button
```

### auth.ts Addition

```typescript
async OTPLogin(code: string, locale?: string): Promise<ApiResponse<{ redirect: string }>> {
    return this.api.Post<{ redirect: string }>('/otp/login', { code, locale: locale || '' })
}
```

## Test Plan

- **Unit tests** (openapi/otp package):
  - Create: produces 12-char code, stores payload, respects TTL
  - Create: validates required fields (user_id/member_id, redirect)
  - Create: handles collision retry
  - Verify: returns payload, rejects expired/invalid/empty
  - Verify: does NOT consume code (multi-verify within TTL)
  - Revoke: removes code, silent on missing
  - Login: full flow with valid code returns tokens
  - Login: rejects invalid code
- **Handler tests**:
  - POST /api/otp/create with valid auth -> 200 + code
  - POST /api/otp/create without auth -> 401
  - POST /api/otp/create with cross-team member -> 403
  - POST /api/otp/login with valid code -> 200 + cookies + redirect
  - POST /api/otp/login with invalid code -> 401
  - POST /api/otp/login with missing code -> 400
