# Authentication API Inventory

**Audience:** Web, desktop, and mobile frontend engineers and coding agents.

**Purpose:** a single list of the Yao authentication endpoints used by the frontend integration guides.

> **Environment:** see [Environment and API endpoints](../environment.md). All URLs are relative to the operator-provided base URL, for example `https://<your-yao-host>`.

**Related:** [Web](web.md) · [Desktop](desktop.md) · [Mobile](mobile.md).

---

## 1. Interactive Web login

These endpoints implement the first-party browser login described in [web.md](web.md).

| Method | Endpoint | Auth | Used by | Purpose |
| --- | --- | --- | --- | --- |
| `GET` | `/v1/user/entry?locale=<locale>` | Public | Web | login page configuration, local-login flag, providers, `success_url`, `failure_url` |
| `GET` | `/v1/user/entry/captcha` | Public | Web | optional image/Turnstile captcha for local login |
| `POST` | `/v1/user/entry/verify` | Public | Web | validate email/mobile; issue short-lived entry-verification token |
| `POST` | `/v1/user/entry/login` | Entry token | Web | password login; sets session cookies |
| `POST` | `/v1/user/entry/register` | Entry token | Web | register with password/OTP/invite; may auto-login |
| `POST` | `/v1/user/entry/otp` | Entry token | Web | resend registration OTP; the first code is sent by `/entry/verify` |
| `POST` | `/v1/user/entry/invite/verify` | Entry token | Web | redeem a required invitation code |
| `GET` | `/v1/user/oauth/providers/available` | Public | Web | provider availability (optional; `entry.providers` is authoritative) |
| `GET` | `/v1/user/oauth/tao/authorize?locale=<locale>` | Public | Web | start Tao login; returns `authorization_url` |
| `GET` | `/v1/user/oauth/tao/activate/callback` | Public browser redirect | Web | first-use Tao deployment activation continuation |
| `GET` | `/v1/user/oauth/tao/callback` | Public browser redirect | Web | Tao callback; sets Yao session cookies and redirects to configured `success_url` (default `/auth/entry/success`) or `failure_url` |
| `POST` | `/v1/user/logout` | Session/bearer | Web | revoke local session and clear auth cookies |
| `GET` | `/v1/user/profile` | Session/bearer | Web | current user, including `identity_source` |

Notes:

- The entry-verification token is returned by `/v1/user/entry/verify` and must be sent as `Authorization: Bearer <token>` to the entry login/register/otp/invite endpoints.
- `/v1/user/oauth/tao/authorize` sets a `session_id` cookie. The same browser cookie context is expected on `/v1/user/oauth/tao/callback`; if WebKit omits it on a cross-site redirect, Yao still validates the one-time state/PKCE transaction. A present-but-different session cookie is rejected.
- The callback is a browser endpoint, not a JSON API. It redirects to the configured entry `success_url` on success (default `/auth/entry/success`), and to the configured `failure_url` with `?code=<code>&message=<message>` on failure (default `/auth/entry/failure`).
- `local_login_enabled=false` makes the local login/register endpoints return `403 local_login_disabled`; hide those UI paths and use Tao.

---

## 2. Native authorization-code login

Desktop and mobile apps are OAuth **public clients**. They use the OAuth 2.1 authorization-code flow with PKCE; the browser shows the Yao login page, then redirects back to the app.

| Method | Endpoint | Auth | Used by | Purpose |
| --- | --- | --- | --- | --- |
| `GET` / `POST` | `/v1/oauth/authorize` | Public endpoint; requires browser login session to issue a code | Desktop, Mobile | authorization request; redirects to `/auth/entry?return_to=...` when not signed in, then to app callback with `code&state` |
| `POST` | `/v1/oauth/token` | Public client | Desktop, Mobile | exchange authorization code; refresh access token |
| `GET` / `POST` | `/v1/oauth/userinfo` | Bearer access token | Desktop, Mobile | identity claims for the token |
| `POST` | `/v1/oauth/revoke` | Public client token value | Desktop, Mobile | revoke access/refresh token |
| `GET` | `/v1/oauth/jwks` | Public | Desktop, Mobile | public signing keys for local JWT verification |
| `POST` | `/v1/oauth/register` | Public (when enabled) | Desktop, Mobile | dynamic registration of a public client |

### Authorization request

```http
GET /v1/oauth/authorize
  ?response_type=code
  &client_id=<client-id>
  &redirect_uri=<registered-redirect-uri>
  &scope=openid%20profile%20email%20offline_access
  &state=<state>
  &code_challenge=<S256-challenge>
  &code_challenge_method=S256
```

Yao requires PKCE `S256` for the authorization-code flow. Without a Yao browser session it `302`s to `/auth/entry?return_to=<encoded-authorize-request>`; the deployment's Web frontend must serve `/auth/entry`, complete login, and navigate back to `return_to`, after which authorization continues and redirects to the registered callback. `/auth/entry` is a Web UI route, not an endpoint served by the OpenAPI.

Success callback:

```text
<redirect_uri>?code=<authorization-code>&state=<state>
```

Error callback:

```text
<redirect_uri>?error=<code>&error_description=<description>&state=<state>
```

### Token request

`application/x-www-form-urlencoded`:

```text
grant_type=authorization_code
&code=<authorization-code>
&redirect_uri=<exact-registered-redirect-uri>
&client_id=<client-id>
&code_verifier=<PKCE-verifier>
```

Refresh uses the same endpoint:

```text
grant_type=refresh_token
&refresh_token=<refresh-token>
&client_id=<client-id>
```

A public client has no secret. The response contains `access_token`, `token_type`, `expires_in`, and `refresh_token` when the client is allowed to receive one. Do not depend on a `scope` field being present; the granted scopes are recorded server-side.

Use the access token on protected APIs:

```http
Authorization: Bearer <access-token>
```

---

## 3. Client registration

A deployment may pre-register clients or enable dynamic client registration. For a native public client, the essential fields are:

```json
{
  "client_name": "My App",
  "application_type": "native",
  "redirect_uris": ["<registered-callback>"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "none",
  "require_pkce": true,
  "scope": "openid profile email offline_access"
}
```

- `token_endpoint_auth_method: "none"` creates a public client. If omitted, the server may create a confidential client and return a secret.
- Redirect URIs must be allowlisted by the deployment. Loopback (`127.0.0.1`/`localhost`) is the desktop pattern. Custom schemes such as `yaosso://callback` require the scheme **and host** to be allowed; if dynamic registration rejects the URI, ask the operator for a pre-registered client.
- Dynamic registration can be disabled. Do not assume it is available in production.

---

## 4. Error and status conventions

| Status | Typical `error` | Meaning |
| --- | --- | --- |
| `400` | `invalid_request` | malformed/missing parameters, invalid PKCE, invalid redirect |
| `400` | `invalid_grant` | code/refresh token invalid, expired, or already used |
| `400` | `invalid_client` | client authentication/registration configuration failed |
| `401` | `invalid_token` | bearer token invalid or expired |
| `403` | `local_login_disabled` | local login is disabled by the deployment |
| `403` | account status error | account blocked, suspended, etc. |
| `302` | `login_required` (to client) | `prompt=none` without a session |
| `503` | `tao_not_configured` | Tao provider is not configured |

OAuth JSON errors have the shape:

```json
{"error": "invalid_grant", "error_description": "..."}
```

Browser authorization errors are delivered by redirect to the client URI (when the client and redirect URI are valid), not as an HTML page.

---

## 5. Where each endpoint is documented

| Guide | Endpoints covered |
| --- | --- |
| [Web](web.md) | interactive entry, local login, Tao login, cookie session, profile, logout |
| [Desktop](desktop.md) | public client, PKCE, system browser, loopback callback, token/refresh/revoke |
| [Mobile](mobile.md) | public client, PKCE, iOS/Android callbacks, token/refresh/revoke |
| [Computer connections](../features/computer-connections.md) | Computer and Tai APIs after authentication |

