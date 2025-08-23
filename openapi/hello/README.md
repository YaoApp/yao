# Hello World API

Simple endpoints for testing connectivity, server status, and OAuth authentication functionality.

## Base URL

All endpoints are prefixed with the configured base URL followed by `/helloworld` (e.g., `/v1/helloworld`).

## Endpoints

### Public Endpoint

Test basic server connectivity without authentication.

```
GET /helloworld/public
POST /helloworld/public
```

**No authentication required.**

**Example:**

```bash
curl -X GET "/v1/helloworld/public"
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

**Response Fields:**

- `MESSAGE` - Static "HELLO, WORLD" message
- `SERVER_TIME` - Current server time in RFC3339 format
- `VERSION` - Yao framework version
- `PRVERSION` - Yao framework preview version
- `CUI` - Yao CUI version
- `PRCUI` - Yao CUI preview version
- `APP` - Application name
- `APP_VERSION` - Application version

### Protected Endpoint

Test OAuth authentication and authorization.

```
GET /helloworld/protected
POST /helloworld/protected
```

**Authentication:** Required (OAuth Bearer token)

**Headers:**

```
Authorization: Bearer {access_token}
```

**Example:**

```bash
# First, obtain an access token
curl -X POST "/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=your_client_id&client_secret=your_client_secret"

# Then use the token to access the protected endpoint
curl -X GET "/v1/helloworld/protected" \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..."
```

**Response:**

Same response format as the public endpoint, confirming that authentication is working correctly.

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

## HTTP Methods

Both endpoints support both GET and POST methods, allowing flexibility for different client requirements and testing scenarios.

## Error Responses

### Unauthorized Access

When accessing the protected endpoint without proper authentication:

**HTTP Status:** `401 Unauthorized`

**Response:**

```json
{
  "error": "invalid_token",
  "error_description": "The access token provided is expired, revoked, malformed, or invalid"
}
```

### Invalid Token

When using an invalid or expired token:

**HTTP Status:** `401 Unauthorized`

**Response:**

```json
{
  "error": "invalid_token",
  "error_description": "The access token provided is invalid"
}
```

## Use Cases

### Health Check

Use the public endpoint as a health check for monitoring systems:

```bash
# Simple health check
curl -f "/v1/helloworld/public" > /dev/null 2>&1 && echo "Service is healthy" || echo "Service is down"
```

### Authentication Testing

Use the protected endpoint to verify OAuth authentication setup:

```bash
# Test authentication flow
TOKEN=$(curl -s -X POST "/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET" \
  | jq -r '.access_token')

curl -X GET "/v1/helloworld/protected" \
  -H "Authorization: Bearer $TOKEN"
```

### Development and Debugging

These endpoints are useful for:

- **API Gateway Testing** - Verify routing and load balancer configuration
- **Authentication Debugging** - Test OAuth token validation
- **Environment Verification** - Check server version and configuration
- **Network Connectivity** - Basic reachability testing
- **Performance Baseline** - Minimal response time measurement

## Integration Examples

### JavaScript/Browser

```javascript
// Public endpoint
fetch("/v1/helloworld/public")
  .then((response) => response.json())
  .then((data) => console.log(data));

// Protected endpoint
const token = localStorage.getItem("access_token");
fetch("/v1/helloworld/protected", {
  headers: {
    Authorization: `Bearer ${token}`,
  },
})
  .then((response) => response.json())
  .then((data) => console.log(data));
```

### Python

```python
import requests

# Public endpoint
response = requests.get('/v1/helloworld/public')
print(response.json())

# Protected endpoint
headers = {'Authorization': f'Bearer {access_token}'}
response = requests.get('/v1/helloworld/protected', headers=headers)
print(response.json())
```

### Go

```go
package main

import (
    "fmt"
    "io"
    "net/http"
)

func testPublicEndpoint() {
    resp, err := http.Get("/v1/helloworld/public")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    fmt.Println(string(body))
}

func testProtectedEndpoint(token string) {
    req, _ := http.NewRequest("GET", "/v1/helloworld/protected", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    fmt.Println(string(body))
}
```

## Security Considerations

### Public Endpoint

- No sensitive information is exposed
- Safe for use in monitoring and health checks
- Should be rate-limited to prevent abuse

### Protected Endpoint

- Requires valid OAuth access token
- Validates token signature and expiration
- Can be used to test authorization scopes (if implemented)

## Server Information

The response includes various version and application information useful for:

- **Version Compatibility** - Ensure client compatibility with server version
- **Environment Identification** - Distinguish between development, staging, and production
- **Debugging** - Identify exact server version when reporting issues
- **Monitoring** - Track server deployments and version rollouts

This API serves as a foundation for testing and validating the Yao OpenAPI infrastructure.
