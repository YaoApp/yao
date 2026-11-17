# Desktop App Authentication

**Audience:** engineers and coding agents building a native desktop application (macOS, Windows, Linux) for Yao 2.0.

**Scope:** OAuth 2.1 authorization-code login with PKCE, system-browser authorization, loopback callback, token storage, API calls, refresh, and logout. The desktop app is a **public client**; it never receives a client secret.

> **Environment:** see [Environment and API endpoints](../environment.md). All URLs are placeholders; use the deployment URL supplied by your Yao operator.

**Related:** [Web authentication](web.md) · [Mobile authentication](mobile.md) · [Computer connections](../features/computer-connections.md) · [Computer API](../features/computer-api.md) · [Authentication API inventory](api-endpoints.md).

**Status:** written against Yao `main` on 2026-11-17.

---

## 1. Flow at a glance

The desktop application never renders the Yao login form itself. It opens the system browser, where the user signs in on Yao's Web authorization page, and Yao redirects the authorization code back to a loopback URL owned by the app.

```text
Desktop app                         System browser / Yao
    |                                    |
    |-- 1. generate PKCE, bind loopback ->|
    |                                    |
    |-- 2. open /v1/oauth/authorize ------>|
    |                                    |  no session:
    |                                    |  302 /auth/entry?return_to=...
    |                                    |  user logs in with Tao/local
    |                                    |  back to /v1/oauth/authorize
    |<-- 3. 302 http://127.0.0.1:port/callback?code&state
    |                                    |
    |-- 4. POST /v1/oauth/token --------->|
    |<-- access_token + refresh_token ---|
    |                                    |
    |-- 5. API calls with Bearer token ->|
```

The authorization page is the deployment's Web login UI. The Yao API redirects to `/auth/entry?return_to=<authorize-request>` when the browser has no session; the Web frontend that implements `/auth/entry` must complete login and navigate back to `return_to`. The desktop app does not need to know whether the user signs in with Tao, email/password, or another configured method.

---

## 2. OAuth endpoints

| Purpose | Method | Endpoint |
| --- | --- | --- |
| Authorization | `GET` / `POST` | `/v1/oauth/authorize` |
| Token exchange and refresh | `POST` | `/v1/oauth/token` |
| UserInfo | `GET` / `POST` | `/v1/oauth/userinfo` |
| Revoke | `POST` | `/v1/oauth/revoke` |
| Dynamic client registration (optional) | `POST` | `/v1/oauth/register` |

The API base is the same Yao base used for other `/v1` endpoints. OAuth endpoints require HTTPS in production.

---

## 3. Register or obtain a client

Ask the Yao operator for a **public** client with:

- `redirect_uris`: the exact loopback callback, for example `http://127.0.0.1:49152/callback`
- `grant_types`: `authorization_code`, `refresh_token`
- `response_types`: `code`
- `token_endpoint_auth_method`: `none`
- `scope`: `openid profile email offline_access` (adjust to your product)
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
  "scope": "openid profile email offline_access"
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

A public client has **no client secret**. Do not send `client_secret` for a public client; there is no secret to protect.

Important:

- `token_endpoint_auth_method` must be explicitly `"none"` for a public client. If omitted, the server may create a confidential client and return a secret.
- The redirect URI must match exactly. There are no wildcard ports or paths.
- The operator must allow the redirect scheme/host. Loopback `127.0.0.1`/`localhost` is the standard desktop pattern, but a deployment may restrict dynamic registration.
- Dynamic registration may be disabled or reject your redirect URI; ask the operator for a pre-registered client in that case.

---

## 4. Loopback redirect

Bind a local HTTP listener before starting authorization:

- Bind only `127.0.0.1`, never `0.0.0.0`.
- Use a fixed, registered port (recommended for a stable redirect URI), for example `49152`.
- Use a single callback path such as `/callback`.
- Accept only a short-lived request for the authorization response; close the listener afterward.
- Do not expose the listener to the network or leave it running.

The exact redirect URI, including scheme, host, port, and path, must be registered on the Yao client. If the port changes, the URI must be re-registered or pre-registered with that exact URI.

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
  &scope=openid%20profile%20email%20offline_access
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

- If the browser has no Yao login session, Yao responds with `302` to `/auth/entry?return_to=<encoded-authorize-request>`. The user signs in there; Yao returns to the authorization request.
- If the user is already signed in, Yao immediately redirects to the loopback callback with the authorization code.
- Invalid client/redirect URI or missing PKCE S256 is rejected with an HTTP 400 response (not an app callback). Invalid scope and other authorization errors may instead redirect to the registered callback with `error=...&state=...`. Handle both a browser error page and an error callback.

Do not use an embedded WebView for the login page. The system browser is the expected authorization user-agent for a desktop OAuth client.

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
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "<jwt>"
}
```

The current implementation does not rely on an ID token for desktop login. Use `GET /v1/oauth/userinfo` with the access token for identity information.

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

