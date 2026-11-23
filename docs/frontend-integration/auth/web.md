# Web App Authentication

**Audience:** frontend engineers and coding agents building a browser-based Web SPA for Yao 2.0.

**Scope:** first-party browser login, session establishment, profile check, and logout. Desktop and mobile native apps use the OAuth authorization-code flow described in [desktop.md](desktop.md) and [mobile.md](mobile.md).

> **Environment:** see [Environment and API endpoints](../environment.md). All URLs in this guide are placeholders; use the base URL supplied by your Yao operator.

**Companion guides:** [Desktop authentication](desktop.md) · [Mobile authentication](mobile.md) · [Computer connections](../features/computer-connections.md) · [Computer API](../features/computer-api.md) · [Authentication API inventory](api-endpoints.md).

**Status:** written against Yao `main` on 2026-11-17.

---

## 1. Integration model

Yao exposes the OpenAPI under `/v1`. A signed-in browser has an HttpOnly cookie session; API calls may alternatively send a bearer access token.

| Item | Current behavior |
| --- | --- |
| API base | `https://<your-yao-host>/v1` |
| Browser auth | HttpOnly cookies set by Yao |
| Native auth | `Authorization: Bearer <access-token>` or the same cookies |
| Login provider | Tao OAuth, configured by the Yao operator |
| Local password login | controlled by `local_login_enabled` in the entry configuration |

### 1.1 Prefer a same-origin deployment

The login callback and session cookies are designed for a **first-party origin**. Serve the frontend and Yao under the same site, for example with a reverse proxy:

```text
https://app.example.com/          -> frontend SPA
https://app.example.com/v1/*      -> Yao OpenAPI
```

A local frontend dev server can use the same pattern: serve the app at one origin and proxy `/v1` to your local Yao instance.

Do not design a login flow that depends on third-party cookies. Cookie `SameSite` policies and browser privacy rules can block it. If the frontend must run on a different origin, use a backend-for-frontend (BFF) that owns the Yao session and exposes your own API to the browser.

### 1.2 Cookies

Yao sets these cookies after a successful login:

| Cookie | Contents | Purpose |
| --- | --- | --- |
| `session_id` | session id | associates OAuth transaction/session state |
| `access_token` | `Bearer <JWT>` | API authentication |
| `refresh_token` | `Bearer <JWT>` | transparent access-token renewal |

All are HttpOnly. The `access_token` and `refresh_token` cookies use `SameSite=Strict`; `session_id` uses `SameSite=Lax`.

The cookie names depend on `openapi.yao -> oauth.security.secure_cookie`:

| `secure_cookie` | Cookie names | Transport |
| --- | --- | --- |
| `false` (local development) | `session_id`, `access_token`, `refresh_token` | HTTP or HTTPS |
| `true` (production default) | `__Host-session_id`, `__Host-access_token`, `__Host-refresh_token` | HTTPS only |

Do not read HttpOnly cookies from JavaScript or store token values in `localStorage`/`sessionStorage`. Let the browser send them with `credentials: "include"`.

When an access token expires, the Yao guard can rotate it with the refresh cookie and returns updated `Set-Cookie` headers. Your HTTP client must accept those headers; do not cache or freeze a token value in application code.

### 1.3 First-party client scopes

Web session scopes come from the app-level `openapi/user/client.yao` file. After
adding a feature that needs new scopes, update that app config and **sign in
again**; an existing browser token keeps its old scope until a new token is
issued.

Member pickers call:

```http
GET /v1/user/teams/:team_id/members
```

The first-party Web client must include:

```text
user:members:read:own
user:members:write:own
user:members:delete:own
user:members:read:team
user:members:write:team
user:members:delete:team
user:members:write:own:profile
```

Missing these scopes returns `403 forbidden` with
`missing_scopes`/`required_scopes` in the response.

---

## 2. Discover the login methods

```http
GET /v1/user/entry?locale=en
```

This endpoint is public and returns the login page configuration. A trimmed example response:

```json
{
  "title": "Yao",
  "description": "Sign in to continue",
  "success_url": "/app",
  "failure_url": "/app/login-error",
  "local_login_enabled": true,
  "role": "user:free",
  "type": "free",
  "form": {
    "username": { "placeholder": "Email", "fields": ["email"] },
    "password": { "placeholder": "Password" },
    "remember_me": true
  },
  "providers": [
    {
      "id": "tao",
      "kind": "oidc-tao",
      "label": "Tao",
      "title": "Sign in with Tao",
      "issuer": "https://<tao-issuer>",
      "available": true
    }
  ]
}
```

Rules:

- Use the top-level `providers` array. For each provider, check `available === true` before rendering a login button.
- `id: "tao"` is the recommended login path for Yao 2.0.
- `local_login_enabled` controls whether email/password endpoints are usable. When it is `false`, local login endpoints return `403 local_login_disabled`.
- The response never contains Tao client secrets.
- `third_party.providers` is a legacy summary. Do not use it as the authoritative provider list.

---

## 3. Tao login (recommended)

Tao login is a browser redirect flow. Yao owns the OAuth transaction, PKCE, code exchange, and session creation; the frontend only starts the redirect and then verifies the session.

### 3.1 Start the login

```http
GET /v1/user/oauth/tao/authorize?locale=en
```

Send this request with credentials enabled, **from the same cookie context that will receive the callback**:

```ts
const response = await fetch("/v1/user/oauth/tao/authorize?locale=en", {
  credentials: "include",
  headers: { Accept: "application/json" },
});

if (!response.ok) {
  const error = await response.json();
  throw new Error(error.error_description || "Tao login is unavailable");
}

const { authorization_url } = await response.json();
// Remember where the user should return inside your SPA.
sessionStorage.setItem("post_login_route", location.pathname + location.search);
window.location.assign(authorization_url);
```

Success response:

```json
{
  "authorization_url": "https://<tao-issuer>/oauth/authorize?...",
  "state": "..."
}
```

If no `session_id` cookie exists yet, the response sets one. The `state`, nonce, and PKCE verifier are stored server-side. When the callback arrives with a `session_id`, it must match the transaction; a different cookie value is rejected as `invalid_state`.

Safari/WebKit may omit the `SameSite=Lax` session cookie on the cross-site redirect from the Tao domain back to Yao. The callback therefore still validates the one-time `state` and PKCE verifier when the cookie is absent; only a present-but-different session cookie is treated as a state mismatch. The same applies to the first-login Tao activation callback.

First deployment may return a Tao **activation** URL instead of a normal authorization URL. That is expected: Yao handles activation and automatically continues to the normal login. The frontend passes the returned URL to the browser unchanged.

### 3.2 Redirect the top-level browser context

Use a top-level navigation, not an iframe or `fetch`:

```ts
window.location.assign(authorization_url);
```

The browser visits Tao, the user signs in, and Tao redirects to Yao:

```text
GET /v1/user/oauth/tao/callback?code=...&state=...
```

This callback is a browser endpoint. Yao exchanges the code, loads Tao user information, creates or updates the Yao user, sets the login cookies, and redirects to the generated callback destination (the configured `success_url`, or `/auth/entry/success` when unset).

The frontend never sees or handles a Tao authorization code or token.

### 3.3 Verify the session

After returning to `/`, call your normal authenticated endpoint:

```ts
const me = await fetch("/v1/user/profile", { credentials: "include" });
if (me.status === 401) {
  // Not signed in, or callback was not completed.
  return;
}

const profile = await me.json();
// For Tao login: profile.identity_source === "tao"

// Account menu: plan + remaining Tao credits.
const usageResponse = await fetch("/v1/user/usage", { credentials: "include" });
const usage = await usageResponse.json();

if (usageResponse.status === 403 && usage.error === "owner_balance_hidden") {
  // owner billing mode: the current member must not see the owner's balance.
  // Hide the balance/plan values; do not show 0 or a fake balance.
} else if (!usageResponse.ok) {
  // Failed usage request: render an error state, never substitute a fake balance.
} else {
  // usage.plan, usage.remaining, usage.unit === "credits"
}
```

Example:

```bash
curl -sS -b /tmp/yao-cookies.txt \
  https://<your-yao-host>/v1/user/profile
```

Use the returned `identity_source` to distinguish Tao and local users. The access token may also contain `auth_source`/`identity_source` claims.

### 3.4 Return route

The Tao callback does not accept an arbitrary frontend `return_to` URL. Its success destination comes from the OpenAPI entry configuration:

```text
openapi/user/entry/<locale>.yao
```

Example:

```jsonc
{
  "success_url": "/app", // or /dashboard, /inbox, etc.
  "failure_url": "/app/login-error"
}
```

`success_url` and `failure_url` are returned by `GET /v1/user/entry`. Set them to real, stable routes in the frontend:

- `success_url`: redirect target after successful Tao login. Default: `/auth/entry/success`.
- `failure_url`: base failure route. Yao appends `code` and `message` query parameters. Default: `/auth/entry/failure`.

For example, the default failure redirect is:

```text
/auth/entry/failure?code=access_denied&message=Login+was+cancelled+or+denied
```

`failure_url` values that already contain a query string have `&` appended instead of `?`.

A SPA should:

1. Configure `success_url` in the OpenAPI entry file for the deployment.
2. After the callback returns, call `GET /v1/user/profile` to confirm the session.
3. Render the configured route directly; do not put tokens or sensitive data in a client-side saved route.

### 3.5 Callback errors

On failure Yao redirects to the configured failure route:

```text
/auth/entry/failure?code=<code>&message=<message>
```

Display a retry action for these values:

| Error | Meaning |
| --- | --- |
| `invalid_state` | login transaction expired or cookie/state mismatch |
| `access_denied` | user cancelled at Tao |
| `tao_not_configured` | Tao provider is not configured on the server |
| `server_error` | code exchange, validation, or user provisioning failed |

Set `TAO_OAUTH_ISSUER` on the Yao server if `tao_not_configured` is returned. The `message` parameter is human-readable; treat `code` as the stable value for UI logic.

---

## 4. Local password login

Local login is available only when `local_login_enabled` is `true`. Production deployments may disable it in favor of Tao.

The flow uses a short-lived **entry verification token**. It is not a normal API token and must be sent in the `Authorization` header for the next step.

### 4.1 Verify the identifier

```http
POST /v1/user/entry/verify
Content-Type: application/json

{"username": "user@example.com", "locale": "en"}
```

If the entry configuration enables a captcha, first call `GET /v1/user/entry/captcha` and include the returned `captcha_id` plus the user's answer or Turnstile token.

The response contains:

```json
{
  "status": "login",
  "access_token": "<entry-verification-token>",
  "expires_in": 600,
  "token_type": "Bearer",
  "scope": "entry_verification",
  "user_exists": true
}
```

`status` is `login` for an existing account and `register` for a new account. For `register`, the response may also contain `otp_id` and `verification_sent`; when verification is required, Yao sends the code during this verify call.

### 4.2 Login

```http
POST /v1/user/entry/login
Authorization: Bearer <entry-verification-token>
Content-Type: application/json

{"password": "user-password", "remember_me": true, "locale": "en"}
```

Success sets the same session cookies as Tao login and returns a token payload:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "id_token": "...",
  "expires_in": 7200,
  "refresh_token_expires_in": 604800,
  "status": "ok"
}
```

For a normal frontend, prefer the HttpOnly cookies and do not persist these values. The JSON token fields are useful for native or automation clients.

### 4.3 Registration

Registration uses the same entry-verification token and follows the `register` status:

1. `POST /v1/user/entry/verify` with the email/phone. When verification is required, this call sends the OTP and returns `otp_id` and `verification_sent`.
2. Call `POST /v1/user/entry/otp` only when the user asks to resend the code; it returns a new `otp_id`.
3. `POST /v1/user/entry/register` with `password`, `confirm_password`, `otp_id`, and `verification_code`.
4. On success Yao may auto-login and return the normal login response, depending on `auto_login`.

If registration requires an invitation, handle the invitation response before continuing.

---

## 5. Session lifecycle

### 5.1 Authenticated API calls

Web, same-origin:

```ts
const response = await fetch("/v1/user/profile", {
  credentials: "include",
  headers: { Accept: "application/json" },
});
```

Native client that stores cookies:

```http
GET /v1/user/profile HTTP/1.1
Cookie: session_id=...; access_token=Bearer ...; refresh_token=Bearer ...
```

Native client using a bearer token:

```http
GET /v1/user/profile HTTP/1.1
Authorization: Bearer <raw-access-token>
```

The cookie value includes the `Bearer ` prefix; an `Authorization` header must contain only the JWT after `Bearer `.

### 5.2 Refresh

The Yao guard automatically refreshes an expired access token when a valid refresh cookie is present and returns new `Set-Cookie` headers. This is the recommended browser path.

For a native bearer-only client, do not assume the cookie refresh path applies: the `Authorization` header takes precedence over cookies. Either keep a browser-style cookie session, implement OAuth token refresh against the Yao OAuth token endpoint for your client, or use the app BFF.

### 5.3 Logout

```http
POST /v1/user/logout
```

Send credentials. Yao revokes the local tokens, best-effort revokes the Tao refresh token, clears the custody record, and deletes auth cookies. The response is:

```json
{"message": "Logout successful"}
```

The endpoint does not guarantee logout from Tao's browser SSO session. If users must be signed out of Tao itself, handle that as a separate Tao-side flow.

### 5.4 Common errors

| HTTP | `error` | Action |
| --- | --- | --- |
| 401 | `token_missing` | start login |
| 401 | `invalid_token` | clear local session, start login |
| 401 | `invalid_refresh_token` | start login |
| 403 | `local_login_disabled` | hide local login; use Tao |
| 403 | account status error | show account/support message |
| 400 | `invalid_request` | validate the form payload |
| 503 | `tao_not_configured` | server configuration issue |

Error bodies are direct JSON, for example:

```json
{"error": "local_login_disabled", "error_description": "Local login is disabled; use Tao account login"}
```

---

## 6. Native applications

Browser login is specific to a first-party Web app. Native desktop and mobile clients should not reuse the browser cookie flow; they use the OAuth 2.1 authorization-code flow with PKCE and bring the authorization code back to the app:

- [Desktop authentication](desktop.md) — system browser + loopback redirect.
- [Mobile authentication](mobile.md) — system browser + custom scheme/universal link, with iOS and Android examples.

## 7. Smoke checklist

Run these against the deployment URL supplied by your operator:

```bash
export YAO_BASE_URL=https://<your-yao-host>

curl -fsS "$YAO_BASE_URL/v1/user/entry?locale=en" | jq '.providers[] | select(.id=="tao")'
# expect available: true when Tao is configured

curl -fsS -c /tmp/yao-cookies.txt \
  "$YAO_BASE_URL/v1/user/oauth/tao/authorize?locale=en" | jq
# expect authorization_url + state, and a session_id cookie file

# Open authorization_url in the SAME browser cookie context.
# After callback, verify the session:
curl -fsS -b /tmp/yao-cookies.txt "$YAO_BASE_URL/v1/user/profile"
```

---

## 8. Implementation references

| Behavior | Source |
| --- | --- |
| Entry configuration and local login guard | `openapi/user/entry.go` |
| Tao authorize/callback and PKCE | `openapi/user/tao.go` |
| Login/session cookies | `openapi/user/login.go`, `openapi/response/response.go` |
| Token extraction and refresh | `openapi/oauth/guard.go` |
| Entry/profile response types | `openapi/user/types.go` |
| User-facing API design | See the OpenAPI/user documentation for the deployed Yao version. |

