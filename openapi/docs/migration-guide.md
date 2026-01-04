# OpenAPI Migration Guide

This guide helps developers migrate their Yao applications to use the new OpenAPI mode. OpenAPI mode enables OAuth 2.1 authentication, AI Agent integration, and other advanced features.

## Overview

When OpenAPI is enabled, your application gains:

- **OAuth 2.1 Authentication** - Industry-standard secure authentication
- **AI Agent Integration** - Built-in AI agent and chat capabilities
- **Knowledge Base** - Vector search and RAG support
- **MCP Protocol Support** - Model Context Protocol for AI tooling
- **API Hot Reload** - Update APIs without server restart

## Quick Start

### 1. Enable OpenAPI

Add the OpenAPI configuration to your `app.yao`:

```json
{
  "name": "My Application",
  "openapi": {
    "enabled": true,
    "baseURL": "/v1"
  }
}
```

### 2. Update Frontend API Calls

The API path prefix changes when OpenAPI is enabled:

| Before (Traditional) | After (OpenAPI)      |
| -------------------- | -------------------- |
| `/api/user/login`    | `/v1/api/user/login` |
| `/api/product/list`  | `/v1/api/product/list` |

**Recommended**: Use a configuration variable for the API prefix:

```javascript
// config.js
export const API_PREFIX = process.env.OPENAPI_ENABLED ? '/v1/api' : '/api';

// usage
fetch(`${API_PREFIX}/user/login`, { ... });
```

### 3. Update Authentication

Replace JWT tokens with OAuth tokens:

```javascript
// Before: JWT
fetch('/api/user/profile', {
  headers: {
    'Authorization': 'Bearer <jwt-token>'
  }
});

// After: OAuth
fetch('/v1/api/user/profile', {
  headers: {
    'Authorization': 'Bearer <oauth-access-token>'
  }
});
```

## Route Changes

### Route Structure

```
/{baseURL}/
├── api/          # Your custom APIs (isolated namespace)
│   ├── user/
│   ├── product/
│   └── ...
├── __yao/        # Built-in Widgets
│   ├── table/
│   ├── form/
│   ├── list/
│   ├── chart/
│   ├── dashboard/
│   └── sui/v1/
├── oauth/        # OAuth endpoints
├── agent/        # AI Agent
├── chat/         # Chat sessions
├── kb/           # Knowledge Base
└── ...           # Other system features
```

### Route Mapping Examples

Assuming `baseURL = "/v1"`:

| Type | Traditional Mode | OpenAPI Mode |
| ---- | ---------------- | ------------ |
| Custom API | `/api/user/login` | `/v1/api/user/login` |
| Table Widget | `/api/__yao/table/pet/search` | `/v1/__yao/table/pet/search` |
| Form Widget | `/api/__yao/form/pet/find/1` | `/v1/__yao/form/pet/find/1` |
| SUI Render | `/api/__yao/sui/v1/render/home` | `/v1/__yao/sui/v1/render/home` |
| OAuth Token | N/A | `/v1/oauth/token` |
| AI Agent | N/A | `/v1/agent/chat` |

## Authentication Changes

### Guard Mapping

Your existing guard configurations are automatically mapped:

| Guard Name | Traditional Mode | OpenAPI Mode |
| ---------- | ---------------- | ------------ |
| `bearer-jwt` | JWT Bearer Token | OAuth Access Token |
| `query-jwt` | JWT in Query String | OAuth Access Token |
| `cookie-jwt` | JWT in Cookie | OAuth Secure Cookie |
| `cookie-trace` | Session Tracking | OAuth Session |
| `-` (public) | No auth | No auth |

### No Code Changes Required

Your API definitions remain unchanged:

```json
{
  "name": "User API",
  "version": "1.0.0",
  "guard": "bearer-jwt",
  "paths": [
    {
      "path": "/profile",
      "method": "GET",
      "process": "scripts.user.Profile"
    }
  ]
}
```

The `bearer-jwt` guard automatically uses OAuth authentication when OpenAPI is enabled.

### Custom Guards

Custom guards defined via processes continue to work unchanged:

```json
{
  "guard": "scripts.auth.CustomGuard",
  "paths": [...]
}
```

### Public APIs

Public APIs (`guard: "-"`) work identically in both modes:

```json
{
  "guard": "-",
  "paths": [
    {
      "path": "/health",
      "method": "GET",
      "process": "scripts.health.Check"
    }
  ]
}
```

## OAuth Integration

### Obtaining Access Tokens

Use the OAuth token endpoint to obtain access tokens:

```bash
# Authorization Code Flow
curl -X POST /v1/oauth/token \
  -d "grant_type=authorization_code" \
  -d "code=<authorization_code>" \
  -d "client_id=<client_id>" \
  -d "redirect_uri=<redirect_uri>" \
  -d "code_verifier=<pkce_verifier>"
```

### Refreshing Tokens

```bash
curl -X POST /v1/oauth/token \
  -d "grant_type=refresh_token" \
  -d "refresh_token=<refresh_token>" \
  -d "client_id=<client_id>"
```

### Available OAuth Endpoints

| Endpoint | Method | Purpose |
| -------- | ------ | ------- |
| `/v1/oauth/authorize` | GET, POST | Authorization request |
| `/v1/oauth/token` | POST | Token exchange |
| `/v1/oauth/revoke` | POST | Revoke tokens |
| `/v1/oauth/introspect` | POST | Token introspection |
| `/v1/oauth/userinfo` | GET | User information |
| `/v1/oauth/jwks` | GET | JSON Web Key Set |

See [OAuth Documentation](./oauth.md) for complete endpoint reference.

## API Hot Reload

OpenAPI mode supports hot reloading of custom APIs without server restart.

### Triggering Hot Reload

After modifying `apis/*.http.yao` files:

**Option 1: Via API call**

```bash
curl -X POST /v1/api/__reload
```

**Option 2: Via Process**

```javascript
Process("yao.api.Reload");
```

**Option 3: Automatic (Development Mode)**

In development mode, file changes are automatically detected and APIs are reloaded.

### What Gets Reloaded

- Custom API definitions (`apis/*.http.yao`)
- Route mappings
- Guard configurations

### What Does NOT Get Reloaded

- Widget definitions (require restart)
- OpenAPI system routes
- Process/Script code (handled separately)

## SUI Frontend Integration

SUI pages work seamlessly with OpenAPI mode.

### Backend Script Calls

Update your SUI backend scripts to use the new API prefix:

```typescript
// pages/home/home.backend.ts
import { Process } from '@yao/runtime';

export function getData() {
  // Process calls remain unchanged
  return Process('models.user.Find', 1, {});
}
```

### Frontend API Calls

```html
<!-- pages/home/home.html -->
<script>
  // Use the configured API prefix
  const API_PREFIX = window.__yao?.apiPrefix || '/api';
  
  fetch(`${API_PREFIX}/user/profile`)
    .then(res => res.json())
    .then(data => console.log(data));
</script>
```

## Checklist

### Before Migration

- [ ] Back up your application
- [ ] Review all API endpoints in use
- [ ] Identify frontend API calls that need updating
- [ ] Plan OAuth client registration

### During Migration

- [ ] Enable OpenAPI in `app.yao`
- [ ] Update frontend API prefix configuration
- [ ] Register OAuth clients
- [ ] Test authentication flows
- [ ] Verify all API endpoints

### After Migration

- [ ] Remove legacy JWT token generation code
- [ ] Update documentation
- [ ] Train team on OAuth flows
- [ ] Monitor for authentication issues

## Troubleshooting

### 404 Not Found

**Symptom**: API returns 404 after enabling OpenAPI.

**Solution**: Update the API path to include the new prefix:

```javascript
// Wrong
fetch('/api/user/profile');

// Correct
fetch('/v1/api/user/profile');
```

### 401 Unauthorized

**Symptom**: API returns 401 with valid JWT token.

**Solution**: Use OAuth access token instead of JWT:

```javascript
// Wrong: Using old JWT
headers: { 'Authorization': 'Bearer <jwt-token>' }

// Correct: Using OAuth access token
headers: { 'Authorization': 'Bearer <oauth-access-token>' }
```

### CORS Issues

**Symptom**: CORS errors when calling APIs from frontend.

**Solution**: Ensure your OAuth client is registered with the correct redirect URIs and origins.

### Hot Reload Not Working

**Symptom**: API changes not reflected after modification.

**Solution**: 
1. Ensure you're in development mode
2. Manually trigger reload: `curl -X POST /v1/api/__reload`
3. Check for syntax errors in API definition files

## FAQ

### Can I use both JWT and OAuth?

No. When OpenAPI is enabled, all authentication uses OAuth. The JWT guards are automatically mapped to OAuth for backward compatibility.

### Do I need to modify my API definition files?

No. Your `apis/*.http.yao` files remain unchanged. The guard names are automatically mapped to the appropriate authentication method.

### What happens to existing JWT tokens?

Existing JWT tokens will no longer work. Users need to re-authenticate using OAuth.

### Can I disable OpenAPI after enabling it?

Yes. Remove or set `openapi.enabled: false` in `app.yao`. Note that this will break OAuth-dependent features.

### Is the performance impacted?

The performance impact is negligible (< 0.01%). The dynamic routing proxy adds approximately 0.1 microseconds per request.

## Related Documentation

- [OAuth Reference](./oauth.md) - Complete OAuth endpoint documentation
- [AI Agent Guide](./agent.md) - Using AI Agent features
- [Knowledge Base Guide](./kb.md) - Setting up Knowledge Base
