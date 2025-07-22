# Yao OpenAPI

The Yao OpenAPI provides a comprehensive set of RESTful APIs for managing Yao applications, including OAuth 2.1/OpenID Connect authentication, DSL resource management, and development utilities.

## Base URL

All API endpoints are prefixed with a configurable base URL (e.g., `/v1`).

## Authentication

The Yao OpenAPI implements OAuth 2.1 and OpenID Connect Core 1.0 specifications for secure authentication and authorization.

### Supported Grant Types

- **Authorization Code Flow** - RFC 6749 (recommended for web applications)
- **Client Credentials Flow** - RFC 6749 (for server-to-server communication)
- **Device Authorization Flow** - RFC 8628 (for devices with limited input)
- **Refresh Token Flow** - RFC 6749 (for token renewal)
- **Token Exchange** - RFC 8693 (for token delegation)

### Discovery Endpoints

The OpenAPI server provides standard OAuth 2.1 discovery endpoints:

```
GET /.well-known/oauth-authorization-server
```

Returns server metadata including supported endpoints, grant types, and security features.

### OAuth Endpoints

#### Authorization Endpoint

Initiate the authorization code flow:

```
GET /oauth/authorize?client_id={client_id}&response_type=code&redirect_uri={redirect_uri}&scope={scope}&state={state}
```

**Parameters:**

- `client_id` (required): Client identifier
- `response_type` (required): Must be "code"
- `redirect_uri` (required): Client redirect URI
- `scope` (optional): Requested scopes
- `state` (recommended): CSRF protection state parameter
- `code_challenge` (optional): PKCE code challenge
- `code_challenge_method` (optional): PKCE challenge method

#### Token Endpoint

Exchange authorization code for access token:

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&code={code}&redirect_uri={redirect_uri}&client_id={client_id}&client_secret={client_secret}
```

**Client Credentials Flow:**

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&client_id={client_id}&client_secret={client_secret}&scope={scope}
```

**Response:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "def50200...",
  "scope": "openid profile"
}
```

#### Token Introspection

Validate and inspect access tokens (RFC 7662):

```
POST /oauth/introspect
Content-Type: application/x-www-form-urlencoded
Authorization: Basic {base64(client_id:client_secret)}

token={access_token}
```

**Response:**

```json
{
  "active": true,
  "scope": "openid profile",
  "client_id": "your_client_id",
  "username": "user@example.com",
  "token_type": "Bearer",
  "exp": 1640995200,
  "iat": 1640991600
}
```

#### Token Revocation

Revoke access or refresh tokens (RFC 7009):

```
POST /oauth/revoke
Content-Type: application/x-www-form-urlencoded
Authorization: Basic {base64(client_id:client_secret)}

token={token}&token_type_hint={access_token|refresh_token}
```

#### Dynamic Client Registration

Register OAuth clients dynamically (RFC 7591):

```
POST /oauth/register
Content-Type: application/json

{
  "client_name": "My Application",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "scope": "openid profile"
}
```

**Response:**

```json
{
  "client_id": "generated_client_id",
  "client_secret": "generated_client_secret",
  "client_name": "My Application",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "scope": "openid profile"
}
```

#### JSON Web Key Set

Retrieve public keys for token verification (RFC 7517):

```
GET /oauth/jwks
```

**Response:**

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-id-1",
      "use": "sig",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

### Authentication Usage

#### Bearer Token Authentication

Include the access token in API requests:

```bash
curl -X GET "/v1/dsl/list/model" \
  -H "Authorization: Bearer {access_token}"
```

#### Client Credentials

For server-to-server authentication, use the client credentials flow to obtain an access token, then include it in subsequent API requests.

## Hello World API

Simple endpoints for testing connectivity and authentication.

### Public Endpoint

Test basic connectivity without authentication:

```
GET /helloworld/public
POST /helloworld/public
```

**Response:**

```json
{
  "MESSAGE": "HELLO, WORLD",
  "SERVER_TIME": "2024-01-15T10:30:00Z",
  "VERSION": "1.0.0",
  "PRVERSION": "1.0.0-preview",
  "CUI": "1.0.0",
  "PRCUI": "1.0.0-preview",
  "APP": "YaoApp",
  "APP_VERSION": "1.0.0"
}
```

### Protected Endpoint

Test OAuth authentication:

```
GET /helloworld/protected
POST /helloworld/protected
```

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response:**
Same as public endpoint, but requires valid authentication.

**Example:**

```bash
# Get access token first
curl -X POST "/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=your_client&client_secret=your_secret"

# Use token to access protected endpoint
curl -X GET "/v1/helloworld/protected" \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..."
```

## DSL Management API

Comprehensive API for managing Yao DSL resources (models, connectors, MCP clients, etc.).

**[View Full DSL API Documentation →](dsl/README.md)**

The DSL Management API provides:

- **Resource Management**: Create, read, update, delete DSL resources
- **Load Management**: Load, unload, reload DSL resources
- **Validation**: Validate DSL source code syntax
- **Execution**: Execute methods on loaded DSL resources
- **Discovery**: List and inspect available DSL resources

**Key Endpoints:**

- `GET /dsl/list/{type}` - List DSL resources
- `POST /dsl/create/{type}` - Create new DSL resource
- `GET /dsl/inspect/{type}/{id}` - Inspect DSL resource details
- `PUT /dsl/update/{type}` - Update existing DSL resource
- `DELETE /dsl/delete/{type}/{id}` - Delete DSL resource

All DSL endpoints require OAuth authentication.

## Error Responses

All endpoints return standardized error responses:

```json
{
  "error": "invalid_request",
  "error_description": "The request is missing a required parameter"
}
```

**Common HTTP Status Codes:**

- `200` - Success
- `201` - Created
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (authentication required)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `500` - Internal Server Error

**OAuth Error Codes:**

- `invalid_request` - Request is malformed
- `invalid_client` - Client authentication failed
- `invalid_grant` - Grant is invalid or expired
- `unauthorized_client` - Client not authorized for grant type
- `unsupported_grant_type` - Grant type not supported
- `invalid_scope` - Requested scope is invalid

## Security Features

The OpenAPI implements comprehensive security measures:

### OAuth 2.1 Security

- **PKCE (Proof Key for Code Exchange)** - Required for public clients
- **State Parameter** - CSRF protection for authorization requests
- **Secure Token Storage** - Access tokens with appropriate expiration
- **Client Authentication** - Multiple authentication methods supported

### HTTP Security Headers

All responses include security headers:

- `Cache-Control: no-store, no-cache, must-revalidate`
- `Pragma: no-cache`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`

### Rate Limiting

API endpoints are protected against abuse with configurable rate limiting.

## Example Workflows

### Web Application Authentication

1. **Register your client** (if using dynamic registration):

```bash
curl -X POST "/v1/oauth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "My Web App",
    "redirect_uris": ["https://myapp.com/callback"],
    "grant_types": ["authorization_code", "refresh_token"]
  }'
```

2. **Initiate authorization flow**:

```
https://api.example.com/v1/oauth/authorize?client_id=your_client_id&response_type=code&redirect_uri=https://myapp.com/callback&scope=openid+profile&state=random_state
```

3. **Exchange authorization code for tokens**:

```bash
curl -X POST "/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=auth_code&redirect_uri=https://myapp.com/callback&client_id=your_client_id&client_secret=your_secret"
```

4. **Use access token to call APIs**:

```bash
curl -X GET "/v1/dsl/list/model" \
  -H "Authorization: Bearer access_token_here"
```

### Server-to-Server Integration

1. **Obtain client credentials token**:

```bash
curl -X POST "/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=server_client&client_secret=server_secret&scope=dsl:manage"
```

2. **Manage DSL resources**:

```bash
# Create a new model
curl -X POST "/v1/dsl/create/model" \
  -H "Authorization: Bearer {access_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "product",
    "source": "{ \"name\": \"product\", \"table\": { \"name\": \"products\" }, \"columns\": [...] }"
  }'
```

## Configuration

The OpenAPI server is configured through `openapi/openapi.yao`.

**[View Complete Configuration Examples →](https://github.com/YaoApp/yao-dev-app/tree/main/openapi)**

This includes comprehensive configuration examples for:

- OAuth 2.1 server settings
- Client registration and management
- Security and authentication policies
- Development and production environments
- API endpoint configuration
