# Desktop App Authentication

**Audience:** engineers and coding agents building a native desktop application (macOS, Windows, Linux) for Yao 2.0.

**Scope:** OAuth 2.1 authorization-code login with PKCE, system-browser authorization, loopback callback, token storage, API calls, refresh, and logout. The desktop app is a **public client**; it never receives a client secret.

> **Environment:** see [Environment and API endpoints](../environment.md). All URLs are placeholders; use the deployment URL supplied by your Yao operator.

**Related:** [Web authentication](web.md) · [Mobile authentication](mobile.md) · [Computer connections](../features/computer-connections.md) · [Computer API](../features/computer-api.md) · [Authentication API inventory](api-endpoints.md).

**Status:** written against Yao `main` on 2026-11-17.

---

## 1. Flow at a glance

The desktop application never renders the Yao login form itself. It opens the system browser, where the user signs in with Tao (or with the deployment's Web login UI when local login is configured), and Yao redirects the authorization code back to a loopback URL owned by the app.

```text
Desktop app                         System browser / Yao
    |                                    |
    |-- 1. generate PKCE, bind loopback ->|
    |                                    |
    |-- 2. open /v1/oauth/authorize ------>|
    |                                    |  no session, native client:
    |                                    |  302 Tao login (no page)
    |                                    |  after login: consent page
    |                                    |  POST /v1/oauth/authorize
    |<-- 3. 302 http://127.0.0.1:port/callback?code&state
    |                                    |
    |-- 4. POST /v1/oauth/token --------->|
    |<-- access_token + refresh_token ---|
    |                                    |
    |-- 5. API calls with Bearer token ->|
```

For a native client when Tao is configured, Yao handles the no-session case
directly: it redirects the system browser to Tao, and after Tao login returns to
the original authorize request. No deployment **login** page is required for the
Tao-only desktop flow; the deployment consent page described below is still
shown before the loopback callback.

If Tao is not configured, or the client is a Web client, Yao keeps using
`/auth/entry?return_to=<authorize-request>` as the deployment login entry; that
is where local email/password login can be added later.

After the Tao callback, Yao resumes the original authorize request with a
one-time server-side login handoff. It does not rely on `SameSite=Strict`
session cookies being sent on the cross-site redirect from Tao, which browsers
may drop.

When a session exists, Yao does not silently issue a code. It redirects to the
deployment consent page (default `/app/consent`), which displays the current
account and asks the user to continue. The consent page POSTs back to
`/v1/oauth/authorize` with a one-time `consent_token`; only then does Yao
redirect to the loopback callback.

---

## 2. OAuth endpoints

| Purpose | Method | Endpoint |
| --- | --- | --- |
| Authorization | `GET` / `POST` | `/v1/oauth/authorize` |
| Pushed authorization request (optional) | `POST` | `/v1/oauth/par` |
| Token exchange and refresh | `POST` | `/v1/oauth/token` |
| UserInfo | `GET` / `POST` | `/v1/oauth/userinfo` |
| Revoke | `POST` | `/v1/oauth/revoke` |
| Dynamic client registration (optional) | `POST` | `/v1/oauth/register` |
| RP-initiated logout | `GET` / `POST` | `/v1/oauth/logout` |

The API base is the same Yao base used for other `/v1` endpoints. OAuth endpoints require HTTPS in production.

---

## 3. Register or obtain a client

Ask the Yao operator for a **public** client with:

- `redirect_uris`: the exact loopback callback, for example `http://127.0.0.1:49152/callback`
- `grant_types`: `authorization_code`, `refresh_token`
- `response_types`: `code`
- `token_endpoint_auth_method`: `none`
- `scope`: the desktop scope set in [§3.1](#31-desktop-scope-set); OIDC scopes alone are not enough for Yao business APIs
- `require_pkce`: `true`

If dynamic client registration is enabled, the app can create a public client itself:

```http
POST /v1/oauth/register
Content-Type: application/json

{
  "client_name": "My Desktop App",
  "application_type": "native",
  "redirect_uris": ["http://127.0.0.1:49152/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "none",
  "require_pkce": true,
  "scope": "openid profile email offline_access user:profile:read:own builtin:teams:selection"
}
```

Response:

```json
{
  "client_id": "dcr_...",
  "client_id_issued_at": 1760000000,
  "redirect_uris": ["http://127.0.0.1:49152/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "none"
}
```

A public client has **no client secret**. Do not send `client_secret` for a public client; there is no secret to protect. If registration fails, read `error` and `error_description`; the server returns the specific validation reason, such as a disallowed redirect host, instead of only the generic `invalid_client_metadata`.

Important:

- `token_endpoint_auth_method` must be explicitly `"none"` for a public client. If omitted, the server may create a confidential client and return a secret.
- The redirect URI must match exactly. There are no wildcard ports or paths.
- The operator must allow the redirect scheme/host. Loopback `127.0.0.1`/`localhost` is the standard desktop pattern, but a deployment may restrict dynamic registration.
- Dynamic registration may be disabled or reject your redirect URI; ask the operator for a pre-registered client in that case.

### 3.1 Desktop scope set

OIDC scopes only identify the user. They do **not** authorize Yao business
APIs. Request the following set for a full desktop client, or a subset that
matches the features you ship:

```text
openid profile email offline_access
user:profile:read:own user:profile:write:own
builtin:teams:selection
user:teams:read:public user:teams:write:public
user:teams:read:config user:teams:write:select
user:teams:read:own user:teams:write:own user:teams:delete:own
user:teams:read:team user:teams:write:team user:teams:delete:team
user:members:read:own user:members:write:own user:members:delete:own
user:members:read:team user:members:write:team user:members:delete:team
user:members:write:own:profile
user:invitations:read:own user:invitations:write:own user:invitations:delete:own
user:invitations:read:team user:invitations:write:team user:invitations:delete:team
user:features:read:own
computer:computers:read:own computer:computers:write:own computer:computers:delete:own
computer:computers:read:team computer:computers:write:team computer:computers:delete:team
tao:computers:read:own tao:computers:write:own
```

The minimum for login plus profile is:

```text
openid profile email offline_access user:profile:read:own
```

Add `builtin:teams:selection` when the client needs to list teams and select
a team after login.

#### Scope to endpoint map

| Scope | Endpoints |
| --- | --- |
| `openid`, `profile`, `email` | OIDC identity; `/v1/oauth/userinfo` |
| `offline_access` | refresh token |
| `user:profile:read:own` | `GET /v1/user/profile` |
| `user:profile:write:own` | `PUT /v1/user/profile`, `PUT /v1/user/account/password` |
| `builtin:teams:selection` | team-selection surface: `GET /v1/user/profile`, `GET /v1/user/teams`, `GET /v1/user/teams/config`, `POST /v1/user/teams/select` |
| `user:teams:read:config` | `GET /v1/user/teams/config` |
| `user:teams:write:select` | `POST /v1/user/teams/select` |
| `user:teams:read:own` | `GET /v1/user/teams`, `GET /v1/user/teams/current`, `GET /v1/user/teams/:id` |
| `user:teams:write:own` | `POST /v1/user/teams`, `PUT /v1/user/teams/:id` |
| `user:teams:delete:own` | `DELETE /v1/user/teams/:id` |
| `user:teams:read:team`, `write:team`, `delete:team` | same team endpoints in the selected-team context |
| `user:teams:read:public`, `write:public` | invitation view/accept endpoints |
| `user:members:*` | `GET/POST/PUT/DELETE /v1/user/teams/:id/members...` |
| `user:invitations:*` | `GET/POST/PUT/DELETE /v1/user/teams/:id/invitations...` |
| `user:features:read:own` | `GET /v1/user/features` |
| `computer:computers:read:own` | `GET /v1/computer/options`, `GET /v1/computers`, computer detail/workspaces/runtimes/containers |
| `computer:computers:write:own` | `POST /v1/computers/connect`, `PATCH /v1/computers/:id`, disconnect/reconnect/renew |
| `computer:computers:delete:own` | `DELETE /v1/computers/:id` |
| `computer:computers:*:team` | the same computer endpoints when the computer belongs to the selected team |
| `tao:computers:read:own` | `GET /v1/tao/computers`, `GET /v1/tao/computers/:id`, `GET /v1/tao/account` |
| `tao:computers:write:own` | `POST /v1/tao/computers`, start/stop/reboot/delete, `PUT/DELETE /v1/tao/account` |

The `:team` scopes apply after the user selects a team; use `:own` for the
personal context.

#### Who configures the scopes

- **Recommended:** the operator pre-registers a public client with this scope
  set. The client sends the same `scope` in `/v1/oauth/authorize`; it may omit
  it, in which case Yao uses the client's registered scope as the default.
- With dynamic registration, the client submits `scope` in the registration
  metadata and then requests the same set at authorize time. Registration
  metadata is the allowlist; it does not bypass the authorize request.
- Request only scopes your build uses. An unsupported scope fails with
  `invalid_scope`; a missing scope on an API call fails with `403` and
  `missing required scopes`.

---

## 4. Loopback redirect

Bind a local HTTP listener before starting authorization:

- Bind only `127.0.0.1`, never `0.0.0.0`.
- A fixed port is recommended, but RFC 8252 loopback validation ignores the port: a client registered with `http://127.0.0.1/callback` may use `http://127.0.0.1:<port>/callback`. Other hosts/schemes remain exact-match.
- Use a single callback path such as `/callback`.
- Accept only a short-lived request for the authorization response; close the listener afterward.
- Do not expose the listener to the network or leave it running.

For loopback HTTP, the scheme, host, and path must match; the port may vary. For non-loopback HTTPS callbacks, the full URI (including port) must match.

The listener receives either:

```text
http://127.0.0.1:49152/callback?code=<authorization-code>&state=<state>
```

or an error:

```text
http://127.0.0.1:49152/callback?error=access_denied&error_description=...&state=<state>
```

Validate `state` before exchanging the code. Reject missing, mismatched, or replayed state values.

---

## 5. PKCE

Yao OAuth 2.1 requires PKCE with `S256` for the authorization-code flow.

1. Generate a high-entropy `code_verifier` (43-128 characters from the unreserved set; a random 32-64 byte value encoded base64url is common).
2. Compute:

```text
code_challenge = BASE64URL(SHA256(ASCII(code_verifier)))
```

3. Send `code_challenge` with `code_challenge_method=S256` on the authorization request.
4. Send the original `code_verifier` on the token request.

Example verifier/challenge construction:

```ts
const verifier = base64UrlEncode(crypto.getRandomValues(new Uint8Array(64)));
const challenge = base64UrlEncode(
  new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier)))
);
```

Do not derive the verifier from a password or reuse it across login attempts.

---

## 6. Authorization request

Build a URL and open it in the system browser:

```text
GET https://<your-yao-host>/v1/oauth/authorize
  ?response_type=code
  &client_id=<client-id>
  &redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback
  &scope=openid%20profile%20email%20offline_access%20user%3Aprofile%3Aread%3Aown%20builtin%3Ateams%3Aselection
  &state=<random-state>
  &code_challenge=<challenge>
  &code_challenge_method=S256
```

Use the platform browser opener:

| OS | Example |
| --- | --- |
| macOS | `open "<url>"` |
| Windows | `start "" "<url>"` |
| Linux | `xdg-open "<url>"` |

Behavior:

- If the browser has no Yao login session and the client is native with Tao configured, Yao responds with `302` directly to Tao. After Tao login, Yao sets its session and returns to the authorization request; no deployment **login** page is involved (the consent page still follows).
- If Tao is not configured, Yao responds with `302` to `/auth/entry?return_to=<encoded-authorize-request>` for the deployment's Web login UI. Local login can be added there later without changing the desktop client.
- When a session exists and `prompt` is absent or `consent`, Yao redirects to the consent page instead of issuing a code. `prompt=login` / `select_account` force reauthentication first; `prompt=none` never shows a page and either issues a code or returns `error=login_required` to the callback.
- The consent page submits the original parameters plus `consent_token` with `POST /v1/oauth/authorize`; only a successful POST returns the loopback `code`.
- Invalid client/redirect URI or missing PKCE S256 is rejected with an HTTP 400 response (not an app callback). Invalid scope and other authorization errors may instead redirect to the registered callback with `error=...&state=...`. Handle both a browser error page and an error callback.

Optional team selection uses the RFC 6749 extension parameter `team_id` on the authorize request. It must be a team the signed-in user belongs to; an unknown team returns `access_denied`. The issued token carries `team_id` / `tenant_id`, and `GET /v1/user/profile?team=true` returns `yao:team`.

Do not use an embedded WebView for the login page. The system browser is the expected authorization user-agent for a desktop OAuth client.

### 6.1 Consent page and prompt

The consent page is a deployment Web route, configured by
`openapi/user/entry/<locale>.yao -> consent_url` (default `/app/consent`). Yao
redirects there with the original authorization parameters plus a one-time
`consent_token`.

The page must POST `application/x-www-form-urlencoded` to
`/v1/oauth/authorize` with the original parameters and `consent_token`. A
successful POST returns the loopback `code`; an invalid token or mismatched
request returns an OAuth error. The token is one-time and expires after 10
minutes. `team_id` may be supplied in this POST as the user's selection even if
it was not present on the initial GET; it is validated against membership and
written into the token.

| `prompt` | Behavior |
| --- | --- |
| absent / `consent` | show the consent page |
| `none` | no page; issue a code with an existing session, otherwise redirect `error=login_required&state=...` |
| `login` | force reauthentication, then show consent |
| `select_account` | same session-switch behavior as `login`; no account-list API is required |
| `max_age=0` | force reauthentication before consent |

Yao forwards `prompt`, `login_hint`, and `max_age` to the Tao authorization
request. Tao enforces reauthentication/account selection; if it cannot
reauthenticate, it returns `error=login_required` instead of silently issuing a
code.

### 6.2 Pushed authorization request (PAR)

If the deployment advertises:

```json
{"require_pushed_authorization_requests": true}
```

or if you choose PAR for hardening, send the authorization parameters to:

```http
POST /v1/oauth/par
Content-Type: application/x-www-form-urlencoded

client_id=<client-id>
&response_type=code
&redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback
&scope=openid%20profile%20email%20offline_access%20user%3Aprofile%3Aread%3Aown%20builtin%3Ateams%3Aselection
&state=<random-state>
&code_challenge=<S256-challenge>
&code_challenge_method=S256
```

The `code_challenge`/`state` are stored with the PAR record. The response is:

```json
{"request_uri":"urn:ietf:params:oauth:request_uri:...","expires_in":600}
```

Then open only:

```text
GET /v1/oauth/authorize?client_id=<client-id>&request_uri=<encoded-request-uri>
```

Yao resolves the stored parameters and shows the login page if needed. The `request_uri` remains valid through the login redirect and is consumed only after a code is successfully issued; an expired/reused request URI is rejected.

The current server advertises PAR as an optional endpoint and does **not** require it when `require_pushed_authorization_requests` is absent or false. If a deployment sets that metadata to true, always use PAR.

---

## 7. Token exchange

Handle the loopback request, validate `state`, then call:

```http
POST /v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=<authorization-code>
&redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback
&client_id=<client-id>
&code_verifier=<original-code-verifier>
```

The authorization code is single-use and short-lived. If the exchange fails, restart the authorization flow with a new state and a new PKCE pair.

Success response (standard OAuth fields plus Yao extensions):

```json
{
  "access_token": "<jwt>",
  "id_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "<jwt>"
}
```

`id_token` is returned for requests that include `openid`; it is optional for this flow. Use `GET /v1/oauth/userinfo` when you need current identity claims.

---

## 8. API calls

Use the access token as a bearer token for all protected Yao APIs:

```http
GET /v1/user/profile
Authorization: Bearer <access-token>
```

Store tokens in the operating system's secure credential store:

| OS | Storage |
| --- | --- |
| macOS | Keychain |
| Windows | Credential Locker / DPAPI-backed store |
| Linux | Secret Service (libsecret) or equivalent |

Do not put access or refresh tokens in plaintext config files, browser `localStorage`, application logs, crash reports, or analytics.

Access tokens expire; a typical desktop session stores the refresh token and refreshes silently before or after expiry.

---

## 9. Refresh

```http
POST /v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=<refresh-token>
&client_id=<client-id>
```

Response has a new access token and, when rotation is enabled, a new refresh token. Persist the rotated refresh token; the old one may be invalid.

Refresh failures with `invalid_grant`, `invalid_token`, or `invalid_client` mean the session cannot be recovered: clear stored tokens and send the user through the authorization flow again.

---

## 10. Logout

The OIDC RP-Initiated Logout endpoint is:

```text
GET /v1/oauth/logout?client_id=<client-id>&post_logout_redirect_uri=<registered-uri>&state=<state>
```

It clears the Yao session cookies and redirects to the registered
`post_logout_redirect_uri` with `state`. `id_token_hint` is accepted when
available but is not required by the current server. The Web `POST /v1/user/logout`
remains available for browser sessions.

Revoke tokens server-side:

```http
POST /v1/oauth/revoke
Content-Type: application/x-www-form-urlencoded

token=<refresh-token>&token_type_hint=refresh_token
```

Then clear local secure storage. The revocation endpoint is idempotent and returns success even when the token is already unknown.

Revocation ends the OAuth client session. It does not guarantee logout from a Tao browser SSO session; that is a separate browser/IdP concern.

---

## 11. Errors and retry behavior

| Condition | Handling |
| --- | --- |
| browser opens but user closes it | cancel listener after a visible timeout; offer "Try again" |
| callback has `error` | show provider/error description; discard state/PKCE pair |
| `state` mismatch | do not exchange code; start a new flow |
| token endpoint returns `invalid_grant` | code expired/reused; start a new flow |
| token endpoint returns `invalid_client` | client configuration is wrong; surface an operator-actionable error |
| refresh fails | clear tokens and request login |
| loopback port is already bound | choose a registered alternate port or fail with a clear message |

Never retry an authorization code or a rotated refresh token with the same value.

---

## 12. Security checklist

- Public client, PKCE S256, no client secret.
- Bind the loopback listener to `127.0.0.1` only.
- Validate `state` before token exchange.
- Accept only the registered callback path and port.
- Store tokens only in the OS secure store.
- Never log authorization codes, access tokens, or refresh tokens.
- Never let a renderer process handle OAuth redirects without a controlled main-process handoff.
- Close the loopback listener as soon as the callback is processed.

---

## 13. Implementation references

| Behavior | Source |
| --- | --- |
| Authorization endpoint and login redirect | `openapi/oauth.go`, `openapi/oauth/guard.go` |
| Authorization code and PKCE validation | `openapi/oauth/core.go` |
| Token endpoint and public-client handling | `openapi/oauth.go`, `openapi/oauth/providers/client/default.go` |
| Client registration | `openapi/oauth/client.go` |
| Redirect URI schemes and hosts | `openapi/oauth/types/types.go` |
| UserInfo | `openapi/oauth.go` |

