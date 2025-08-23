# OAuth 2.0/2.1 Route Mapping (RFC Standards + MCP Protocol Support)

## OAuth Core Endpoints (OAuth 2.1 Required)

| Endpoint            | HTTP Method | Purpose                                          | RFC Standard            | MCP Requirement       |
| ------------------- | ----------- | ------------------------------------------------ | ----------------------- | --------------------- |
| `/oauth/authorize`  | GET, POST   | Authorization request, obtain authorization code | RFC 6749 Section 3.1    | ✅ Required           |
| `/oauth/token`      | POST        | Token request, exchange for access token         | RFC 6749 Section 3.2    | ✅ Required           |
| `/oauth/revoke`     | POST        | Revoke access token or refresh token             | RFC 7009                | ✅ Required           |
| `/oauth/introspect` | POST        | Check token status and metadata                  | RFC 7662                | ✅ Recommended        |
| `/oauth/jwks`       | GET         | JSON Web Key Set for token verification          | RFC 7517                | ✅ Required (for JWT) |
| `/oauth/userinfo`   | GET, POST   | Retrieve user information                        | OpenID Connect Core 1.0 | ✅ Recommended        |

## OAuth Extended Endpoints

| Endpoint                      | HTTP Method | Purpose                       | RFC Standard | MCP Requirement |
| ----------------------------- | ----------- | ----------------------------- | ------------ | --------------- |
| `/oauth/register`             | POST        | Dynamic client registration   | RFC 7591     | ✅ Required     |
| `/oauth/register/:client_id`  | GET         | Retrieve client configuration | RFC 7592     | ✅ Optional     |
| `/oauth/register/:client_id`  | PUT         | Update client configuration   | RFC 7592     | ✅ Optional     |
| `/oauth/register/:client_id`  | DELETE      | Delete client configuration   | RFC 7592     | ✅ Optional     |
| `/oauth/device_authorization` | POST        | Device authorization flow     | RFC 8628     | ✅ Optional     |
| `/oauth/par`                  | POST        | Pushed Authorization Request  | RFC 9126     | ✅ Recommended  |
| `/oauth/token_exchange`       | POST        | Token exchange                | RFC 8693     | ✅ Optional     |

## Discovery and Metadata Endpoints

| Endpoint                                  | HTTP Method | Purpose                       | RFC Standard                 | MCP Requirement |
| ----------------------------------------- | ----------- | ----------------------------- | ---------------------------- | --------------- |
| `/.well-known/oauth-authorization-server` | GET         | Authorization server metadata | RFC 8414                     | ✅ Required     |
| `/.well-known/openid_configuration`       | GET         | OpenID Connect configuration  | OpenID Connect Discovery 1.0 | ✅ Optional     |
| `/.well-known/oauth-protected-resource`   | GET         | Protected resource metadata   | RFC 9728                     | ✅ Required     |

## Interface Method Mapping

Each route handler corresponds to interface methods:

- `oauthAuthorize` → `OAuth.Authorize()`
- `oauthToken` → `OAuth.Token()`, `OAuth.RefreshToken()`
- `oauthRevoke` → `OAuth.Revoke()`
- `oauthIntrospect` → `OAuth.Introspect()`
- `oauthJWKS` → `OAuth.JWKS()`
- `oauthUserInfo` → `OAuth.UserInfo()`
- `oauthRegister` → `OAuth.Register()`, `OAuth.DynamicClientRegistration()`
- `oauthGetClient` → Client query methods
- `oauthUpdateClient` → `OAuth.UpdateClient()`
- `oauthDeleteClient` → `OAuth.DeleteClient()`
- `oauthDeviceAuthorization` → `OAuth.DeviceAuthorization()`
- `oauthPushedAuthorizationRequest` → `OAuth.PushAuthorizationRequest()`
- `oauthTokenExchange` → `OAuth.TokenExchange()`
- `oauthServerMetadata` → `OAuth.GetServerMetadata()`
- `oauthProtectedResourceMetadata` → `OAuth.GetProtectedResourceMetadata()`

## MCP Protocol Special Requirements

1. **Resource Parameter Validation**: Using `OAuth.ValidateResourceParameter()`
2. **Canonical Resource URI**: Using `OAuth.GetCanonicalResourceURI()`
3. **State Parameter Security**: Using `OAuth.ValidateStateParameter()`, `OAuth.GenerateStateParameter()`
4. **Redirect URI Validation**: Using `OAuth.ValidateRedirectURI()`
5. **Token Binding**: Using `OAuth.ValidateTokenBinding()`
6. **Refresh Token Rotation**: Using `OAuth.RotateRefreshToken()`

## Security Considerations

- All POST endpoints should validate CSRF protection
- `/oauth/authorize` supports both GET and POST, but POST is recommended for enhanced security
- PKCE (Proof Key for Code Exchange) should be enforced in all authorization code flows
- All endpoints should support HTTPS
- Token endpoints require client authentication
- State parameters are required in authorization flows

## Typical Flows

1. **Authorization Code Flow**: `/oauth/authorize` → `/oauth/token`
2. **Refresh Token**: `/oauth/token` (grant_type=refresh_token)
3. **Token Revocation**: `/oauth/revoke`
4. **Device Flow**: `/oauth/device_authorization` → `/oauth/token`
5. **Token Introspection**: `/oauth/introspect`
6. **Dynamic Registration**: `/oauth/register`

## MCP Authorization Flow Overview

The Model Context Protocol requires specific OAuth 2.1 implementation patterns:

### Authorization Server Discovery

1. **Protected Resource Metadata**: MCP servers MUST implement RFC 9728
2. **WWW-Authenticate Header**: Used in 401 responses to indicate authorization server location
3. **Server Metadata**: Authorization servers MUST provide RFC 8414 metadata

### Resource Parameter Implementation

MCP clients MUST implement Resource Indicators (RFC 8707):

```
&resource=https%3A%2F%2Fmcp.example.com
```

- MUST be included in both authorization and token requests
- MUST identify the target MCP server
- MUST use canonical URI format

### Canonical Server URI Examples

**Valid canonical URIs:**

- `https://mcp.example.com/mcp`
- `https://mcp.example.com`
- `https://mcp.example.com:8443`
- `https://mcp.example.com/server/mcp`

**Invalid canonical URIs:**

- `mcp.example.com` (missing scheme)
- `https://mcp.example.com#fragment` (contains fragment)

### Access Token Usage

- MUST use Authorization header: `Authorization: Bearer <access-token>`
- MUST NOT include tokens in URI query strings
- MUST validate token audience binding
- MUST implement token theft protection

### Dynamic Client Registration

Authorization servers SHOULD support RFC 7591 for seamless client onboarding:

- Enables automatic registration with new authorization servers
- Reduces user friction
- Allows authorization servers to implement custom registration policies

### Security Requirements

1. **Token Audience Binding**: Tokens MUST be bound to intended audiences
2. **Communication Security**: All endpoints MUST use HTTPS
3. **PKCE Protection**: MUST implement PKCE for authorization code flows
4. **Open Redirection Prevention**: MUST validate redirect URIs exactly
5. **Refresh Token Rotation**: MUST rotate refresh tokens for public clients

This route planning follows OAuth 2.1 best practices, supports all MCP protocol requirements, and provides complete OAuth authorization server functionality with enhanced security measures specifically designed for Model Context Protocol implementations.
