# Page Configuration

Each SUI page can have a configuration file (`<page>.config`) that defines page-level settings including title, guards, caching, and API options.

## File Naming

Configuration files use the naming convention `<page>.config`:

```
/pages/users/
├── users.html
├── users.css
├── users.ts
├── users.json
├── users.config        # Page configuration
└── users.backend.ts
```

## Configuration Structure

```json
{
  "title": "Page Title",
  "description": "Page description",
  "guard": "oauth",
  "cache": 3600,
  "dataCache": 300,
  "cacheStore": "redis",
  "root": "/custom-root",
  "seo": {
    "title": "SEO Title",
    "description": "SEO Description",
    "keywords": "keyword1, keyword2",
    "image": "/images/og-image.png",
    "url": "https://example.com/page"
  },
  "api": {
    "prefix": "Api",
    "defaultGuard": "oauth",
    "guards": {
      "PublicMethod": "-",
      "AdminMethod": "bearer-jwt"
    }
  }
}
```

## Configuration Options

### Basic Options

| Option        | Type   | Description                      | Default |
| ------------- | ------ | -------------------------------- | ------- |
| `title`       | string | Page title                       | -       |
| `description` | string | Page description                 | -       |
| `guard`       | string | Guard for page rendering         | -       |
| `cache`       | number | Page cache duration in seconds   | 0       |
| `dataCache`   | number | Data cache duration in seconds   | 0       |
| `cacheStore`  | string | Cache store name (e.g., "redis") | -       |
| `root`        | string | Custom root path for the page    | -       |

### SEO Options

```json
{
  "seo": {
    "title": "SEO Title - Different from page title",
    "description": "Meta description for search engines",
    "keywords": "comma, separated, keywords",
    "image": "/images/og-image.png",
    "url": "https://example.com/canonical-url"
  }
}
```

### API Options

The `api` section configures guards for backend API methods (called via `$Backend().Call()`):

```json
{
  "api": {
    "prefix": "Api",
    "defaultGuard": "oauth",
    "guards": {
      "MethodName": "guard-name"
    }
  }
}
```

| Option         | Type   | Description                       | Default |
| -------------- | ------ | --------------------------------- | ------- |
| `prefix`       | string | Method prefix for API functions   | "Api"   |
| `defaultGuard` | string | Default guard for all API methods | -       |
| `guards`       | object | Per-method guard overrides        | -       |

## Guards

SUI supports the following built-in guards:

| Guard          | Description                                     |
| -------------- | ----------------------------------------------- |
| `oauth`        | OAuth 2.1 authentication (recommended)          |
| `bearer-jwt`   | Bearer token JWT authentication                 |
| `cookie-jwt`   | Cookie-based JWT authentication                 |
| `query-jwt`    | Query string JWT authentication (`?__tk=token`) |
| `cookie-trace` | Session tracking via cookie                     |
| `-`            | No authentication (public access)               |

### Page Guard vs API Guard

- **Page Guard** (`guard`): Controls access to page rendering
- **API Guard** (`api.defaultGuard` / `api.guards`): Controls access to backend API methods

```json
{
  "guard": "oauth",
  "api": {
    "defaultGuard": "oauth",
    "guards": {
      "PublicSearch": "-"
    }
  }
}
```

In this example:

- Page rendering requires OAuth authentication
- All API methods require OAuth by default
- `ApiPublicSearch` method is publicly accessible

## Examples

### Public Page

```json
{
  "title": "Welcome",
  "description": "Public landing page"
}
```

### Protected Page with OAuth

```json
{
  "title": "Dashboard",
  "guard": "oauth",
  "api": {
    "defaultGuard": "oauth"
  }
}
```

### Mixed Access Page

```json
{
  "title": "Product Catalog",
  "guard": "-",
  "api": {
    "defaultGuard": "-",
    "guards": {
      "AddToCart": "oauth",
      "Checkout": "oauth"
    }
  }
}
```

Page is public, most API methods are public, but cart and checkout require authentication.

### Cached Page

```json
{
  "title": "Blog Post",
  "cache": 3600,
  "dataCache": 300,
  "guard": "-"
}
```

### Full Configuration Example

```json
{
  "title": "User Settings",
  "description": "Manage your account settings",
  "guard": "oauth",
  "cache": 0,
  "dataCache": 60,
  "seo": {
    "title": "Account Settings | MyApp",
    "description": "Configure your account preferences and security settings"
  },
  "api": {
    "defaultGuard": "oauth",
    "guards": {
      "GetPublicProfile": "-",
      "UpdateProfile": "oauth",
      "DeleteAccount": "oauth"
    }
  }
}
```

## Accessing Authorized Info

When using `oauth` guard, the authorized user information is available in:

### Backend Scripts

```typescript
function ApiGetUserData(request: Request): any {
  // Access OAuth info from request.authorized
  const userId = request.authorized?.user_id;
  const teamId = request.authorized?.team_id;
  const clientId = request.authorized?.client_id;
  const scope = request.authorized?.scope;

  // Access data constraints (set by ACL)
  const ownerOnly = request.authorized?.constraints?.owner_only;
  const teamOnly = request.authorized?.constraints?.team_only;

  return Process("models.user.Find", userId);
}
```

### Data Binding (`.json`)

```json
{
  "userId": "$auth.user_id",
  "teamId": "$auth.team_id"
}
```

### HTML Templates

```html
<p>Welcome, User {{ $auth.user_id }}</p>
<p s:if="$auth.team_id">Team: {{ $auth.team_id }}</p>
```

## Custom Guards

You can use custom process-based guards:

```json
{
  "guard": "scripts.guards.CheckAdmin",
  "api": {
    "defaultGuard": "scripts.guards.CheckPermission"
  }
}
```

The guard process receives the request context and should throw an exception to deny access.
