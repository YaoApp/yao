# Mobile App Authentication (iOS and Android)

**Audience:** iOS and Android engineers and coding agents building native Yao 2.0 clients.

**Scope:** OAuth 2.1 authorization-code login with PKCE, system-browser authorization, custom scheme or universal-link callback, token storage, API calls, refresh, and logout. The mobile app is a **public client**; it never receives a client secret.

> **Environment:** see [Environment and API endpoints](../environment.md). All URLs are placeholders; use the deployment URL supplied by your Yao operator.

**Related:** [Web authentication](web.md) · [Desktop authentication](desktop.md) · [Computer connections](../features/computer-connections.md) · [Computer API](../features/computer-api.md) · [Authentication API inventory](api-endpoints.md).

**Status:** written against Yao `main` on 2026-11-17.

---

## 1. Flow at a glance

The app opens Yao's Web authorization page in the system browser. The user signs in there (Tao, local password, or another configured method), and Yao redirects the authorization code back to the app through a registered callback URI.

```text
iOS / Android app                  System browser / Yao
       |                                  |
       |-- 1. PKCE + state --------------->|
       |-- 2. open /v1/oauth/authorize --->|
       |                                  | no session:
       |                                  |  /auth/entry?return_to=...
       |                                  | user signs in
       |<-- 3. redirect to app callback --|
       |        ?code=...&state=...       |
       |                                  |
       |-- 4. POST /v1/oauth/token ------>|
       |<-- access + refresh tokens ------|
       |                                  |
       |-- 5. Bearer API calls ---------->|
```

The app must not render Yao's login form in an embedded `WebView`. The system browser is the authorization user-agent; it also supports existing browser SSO cookies. When no Yao session exists, `/v1/oauth/authorize` redirects to `/auth/entry?return_to=...`; the deployment's Web frontend must serve that route and return to the authorization request after login.

---

## 2. OAuth endpoints

| Purpose | Method | Endpoint |
| --- | --- | --- |
| Authorization | `GET` / `POST` | `/v1/oauth/authorize` |
| Token exchange and refresh | `POST` | `/v1/oauth/token` |
| UserInfo | `GET` / `POST` | `/v1/oauth/userinfo` |
| Revoke | `POST` | `/v1/oauth/revoke` |
| Dynamic client registration (optional) | `POST` | `/v1/oauth/register` |

---

## 3. Register or obtain a client

Ask the Yao operator for a **public** client with:

- `client_type`: public
- `redirect_uris`: a custom scheme such as `yaosso://callback`, and/or a verified HTTPS callback such as `https://app.example.com/mobile/callback`
- `grant_types`: `authorization_code`, `refresh_token`
- `response_types`: `code`
- `token_endpoint_auth_method`: `none`
- `scope`: `openid profile email offline_access`
- `require_pkce`: `true`

Example dynamic registration for a mobile public client:

```http
POST /v1/oauth/register
Content-Type: application/json

{
  "client_name": "My Mobile App",
  "application_type": "native",
  "redirect_uris": ["yaosso://callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "none",
  "require_pkce": true,
  "scope": "openid profile email offline_access"
}
```

Response includes a `client_id`. A public client has no client secret.

The server must allow the redirect scheme and host used by the client. For custom schemes this is controlled by the OAuth client policy (`allowed_redirect_uri_schemes`); the host portion is also validated against `allowed_redirect_uri_hosts`, so a URI like `yaosso://callback` has host `callback`. Some deployments allow the scheme but not that host; in that case dynamic registration is rejected and the operator must pre-register the client. `yaosso` is a supported example, but the deployment may use a different scheme or require an HTTPS universal/app link. Confirm the exact callback URI with the operator before shipping.

> `yaosso://callback` is an example. The scheme must be registered in the app, in the Yao client, and allowed by the Yao deployment.

---

## 4. PKCE and state

Yao requires PKCE with `S256` for the authorization-code flow.

1. Generate a random `code_verifier` (43-128 unreserved characters; a random 32-64-byte value encoded base64url is common).
2. Compute `code_challenge = BASE64URL(SHA256(ASCII(code_verifier)))`.
3. Send `code_challenge` and `code_challenge_method=S256` on `/v1/oauth/authorize`.
4. Send the original `code_verifier` when exchanging the code.
5. Generate a random `state`. Store it with the PKCE verifier in memory for the duration of the flow.

Never reuse a verifier or state across attempts.

---

## 5. Authorization request

Build and open:

```text
GET https://<your-yao-host>/v1/oauth/authorize
  ?response_type=code
  &client_id=<client-id>
  &redirect_uri=yaosso%3A%2F%2Fcallback
  &scope=openid%20profile%20email%20offline_access
  &state=<random-state>
  &code_challenge=<challenge>
  &code_challenge_method=S256
```

Behavior:

- Without a Yao browser session, Yao redirects to `/auth/entry?return_to=<encoded-authorize-request>`. The user signs in on the Web page, and Yao returns to the authorization request.
- With a session, Yao immediately redirects to the registered callback with `code` and `state`.
- Invalid client/redirect URI or missing PKCE S256 is rejected with an HTTP 400 response (not an app callback). Invalid scope and other authorization errors may redirect to the registered callback with `error=...&state=...`; handle both cases.

On success the app callback receives:

```text
yaosso://callback?code=<authorization-code>&state=<state>
```

On error:

```text
yaosso://callback?error=access_denied&error_description=...&state=<state>
```

Validate `state` before doing anything else.

---

## 6. iOS

### 6.1 Register the URL scheme

Add a URL type to `Info.plist`:

```xml
<key>CFBundleURLTypes</key>
<array>
  <dict>
    <key>CFBundleURLSchemes</key>
    <array>
      <string>yaosso</string>
    </array>
  </dict>
</array>
```

If the operator registers an HTTPS callback, use an Associated Domain/universal link instead; the app then handles the callback through the universal-link delegate.

### 6.2 Open the system browser session

Use `ASWebAuthenticationSession` so the callback returns directly to the app. Do not use `SFSafariViewController` alone: it does not return the callback URL to your code.

```swift
import AuthenticationServices

final class AuthSession: NSObject, ASWebAuthenticationPresentationContextProviding {
    private var session: ASWebAuthenticationSession?

    func start(authURL: URL) {
        let session = ASWebAuthenticationSession(
            url: authURL,
            callbackURLScheme: "yaosso"
        ) { callbackURL, error in
            // callbackURL contains code/state on success
            // error is set when the user cancels or the session fails
            self.session = nil
        }
        session.presentationContextProvider = self
        session.prefersEphemeralWebBrowserSession = false // share SSO when desired
        self.session = session
        session.start()
    }

    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        // Return the active key window.
        UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .flatMap(\.windows)
            .first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
}
```

Read `code` and `state` from `callbackURL` query parameters, verify state, then exchange the code.

### 6.3 Store tokens

Store access/refresh tokens in Keychain, preferably accessible only when the device is unlocked. Do not store them in `UserDefaults`, files, logs, or analytics.

### 6.4 Token exchange

```http
POST /v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=<code>
&redirect_uri=yaosso%3A%2F%2Fcallback
&client_id=<client-id>
&code_verifier=<verifier>
```

---

## 7. Android

### 7.1 Register the callback

Declare an activity or a dedicated callback activity with an intent filter:

```xml
<activity
    android:name=".AuthCallbackActivity"
    android:exported="true"
    android:launchMode="singleTask">
    <intent-filter>
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="yaosso" android:host="callback" />
    </intent-filter>
</activity>
```

If the operator uses HTTPS app links, declare the verified host and use an App Link intent filter instead. Custom schemes can be claimed by other apps; App Links/universal links are the stronger production option.

### 7.2 Open a Custom Tab

Use AndroidX Browser Custom Tabs for the authorization page:

```kotlin
val authUri = Uri.parse(authorizeUrl)
CustomTabsIntent.Builder()
    .setShowTitle(true)
    .build()
    .launchUrl(context, authUri)
```

Do not use an embedded WebView for the OAuth login page.

### 7.3 Handle the callback

Handle the callback in `onCreate` and `onNewIntent`:

```kotlin
override fun onNewIntent(intent: Intent) {
    super.onNewIntent(intent)
    handleAuthCallback(intent.data)
}

private fun handleAuthCallback(uri: Uri?) {
    if (uri == null) return
    if (uri.scheme != "yaosso" || uri.host != "callback") return

    val code = uri.getQueryParameter("code")
    val state = uri.getQueryParameter("state")
    val error = uri.getQueryParameter("error")
    if (error != null) {
        // show error_description
        return
    }
    // verify state, then exchange code
}
```

### 7.4 Store tokens

Use `EncryptedSharedPreferences` backed by the Android Keystore, or an equivalent hardware-backed secure storage. Do not store tokens in plain `SharedPreferences` or log them.

### 7.5 Token exchange

Same form request as iOS, with `redirect_uri=yaosso://callback` (exact registered value).

---

## 8. Exchange and use tokens

```http
POST /v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=<authorization-code>
&redirect_uri=<exact-registered-redirect-uri>
&client_id=<client-id>
&code_verifier=<original-verifier>
```

Success:

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "<jwt>"
}
```

The code is single-use and short-lived. Use the access token for API calls:

```http
GET /v1/user/profile
Authorization: Bearer <access-token>
```

Get identity information from:

```http
GET /v1/oauth/userinfo
Authorization: Bearer <access-token>
```

---

## 9. Refresh

```http
POST /v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=<refresh-token>
&client_id=<client-id>
```

Persist the returned refresh token when rotation is enabled; the old token may be invalid immediately. On `invalid_grant`/`invalid_token`/`invalid_client`, clear tokens and restart the authorization flow.

---

## 10. Logout

Revoke the refresh token server-side:

```http
POST /v1/oauth/revoke
Content-Type: application/x-www-form-urlencoded

token=<refresh-token>&token_type_hint=refresh_token
```

Then clear Keychain/Keystore state. Revocation does not guarantee logout from a Tao browser SSO session; Tao-side logout is a separate concern.

---

## 11. Native app security checklist

- Use the system browser (`ASWebAuthenticationSession`, Custom Tabs); avoid embedded WebViews for credentials.
- Use PKCE S256 and validate `state`.
- Use a public client; never ship a client secret.
- Register and verify the callback exactly; prefer universal links/app links where possible.
- Treat custom schemes as hijackable; PKCE and short code lifetimes reduce, but do not eliminate, the risk.
- Store tokens in Keychain/Keystore-backed storage.
- Never log authorization codes, access tokens, or refresh tokens.
- Cancel the flow cleanly when the user returns without `code`.

---

## 12. Errors and retry behavior

| Condition | Handling |
| --- | --- |
| user cancels browser | discard state/verifier; offer retry |
| callback has `error` / `error_description` | display the provider error; do not exchange |
| `state` mismatch | reject callback; start a new flow |
| code expired/reused | start a new authorization flow |
| client/redirect configuration error | surface an operator-actionable message |
| refresh fails | clear tokens and re-authenticate |

---

## 13. Implementation references

| Behavior | Source |
| --- | --- |
| Authorization endpoint and login redirect | `openapi/oauth.go`, `openapi/oauth/guard.go` |
| PKCE validation and authorization code exchange | `openapi/oauth/core.go` |
| Token and refresh handlers | `openapi/oauth.go` |
| Dynamic client registration | `openapi/oauth/client.go` |
| Redirect schemes and allowed hosts | `openapi/oauth/types/types.go` |
| UserInfo | `openapi/oauth.go` |

