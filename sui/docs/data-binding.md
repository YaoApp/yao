# Data Binding

SUI provides built-in variables and functions for accessing request data and executing server-side logic.

## Built-in Variables

### Request Variables

| Variable   | Description          | Example                 |
| ---------- | -------------------- | ----------------------- |
| `$payload` | POST request body    | `{{ $payload.name }}`   |
| `$query`   | URL query parameters | `{{ $query.search }}`   |
| `$param`   | Route parameters     | `{{ $param.id }}`       |
| `$cookie`  | Request cookies      | `{{ $cookie.session }}` |

### URL Variables

| Variable      | Description | Example Value               |
| ------------- | ----------- | --------------------------- |
| `$url.path`   | URL path    | `/users/123`                |
| `$url.host`   | Full host   | `example.com:8080`          |
| `$url.domain` | Domain only | `example.com`               |
| `$url.scheme` | Protocol    | `https`                     |
| `$url.url`    | Full URL    | `https://example.com/users` |

### Context Variables

| Variable     | Description                    | Example Value        |
| ------------ | ------------------------------ | -------------------- |
| `$theme`     | Current theme                  | `light`, `dark`      |
| `$locale`    | Current locale                 | `en-us`, `zh-cn`     |
| `$timezone`  | System timezone                | `Asia/Shanghai`      |
| `$direction` | Text direction                 | `ltr`, `rtl`         |
| `$global`    | Global data from `__data.json` | `{ title: "App" }`   |
| `$auth`      | OAuth authorized info (if guard is `oauth`) | `{ user_id: "123" }` |

## Usage Examples

### Query Parameters

URL: `/search?q=hello&page=2`

```html
<h1>Search: {{ $query.q }}</h1>
<p>Page: {{ $query.page ?? 1 }}</p>
```

### Route Parameters

Route: `/users/[id]/posts/[postId]`
URL: `/users/123/posts/456`

```html
<h1>User {{ $param.id }}</h1>
<p>Post {{ $param.postId }}</p>
```

### POST Payload

```html
<form method="POST">
  <input name="email" value="{{ $payload.email }}" />
  <div s:if="{{ $payload.error }}">{{ $payload.error }}</div>
</form>
```

### Theme and Locale

```html
<html class="{{ $theme }}" lang="{{ $locale }}">
  <body dir="{{ $direction }}">
    <h1>{{ $global.title }}</h1>
  </body>
</html>
```

## Data Configuration (`<page>.json`)

Define page data using JSON configuration:

### Static Data

```json
{
  "title": "My Page",
  "items": [
    { "id": 1, "name": "Item 1" },
    { "id": 2, "name": "Item 2" }
  ]
}
```

### Process Calls

```json
{
  "$users": "models.user.Get",
  "$settings": {
    "process": "models.settings.Find",
    "args": [1]
  }
}
```

Keys starting with `$` trigger process calls. The result is available as the variable name (without `$`).

### Using Request Variables

```json
{
  "$user": {
    "process": "models.user.Find",
    "args": ["$param.id"]
  },
  "searchQuery": "$query.q",
  "currentPath": "$url.path"
}
```

Available request variables in JSON config:

- `$query.<name>` - Query parameters
- `$param.<name>` - Route parameters
- `$payload.<name>` - POST payload
- `$header.<name>` - Request headers
- `$url.path` / `$url.host` / `$url.domain` / `$url.scheme`

Note: `$header` is only available in JSON configuration, not in HTML templates.

### Complex Example

```json
{
  "pageTitle": "User Profile",
  "userId": "$param.id",
  "$user": {
    "process": "models.user.Find",
    "args": ["$param.id"]
  },
  "$posts": {
    "process": "models.post.Get",
    "args": [
      {
        "wheres": [{ "column": "user_id", "value": "$param.id" }],
        "limit": 10
      }
    ]
  },
  "isOwner": "$query.edit == 'true'"
}
```

### Calling Backend Script Methods

Use the `@MethodName` syntax to call functions defined in the page's `.backend.ts` file:

```json
{
  "$record": "@GetRecord",
  "$items": {
    "process": "@GetItems",
    "args": ["active", 20]
  }
}
```

**Important**: The Request object is automatically appended as the **last argument** to the backend function.

**`page.backend.ts`**:

```typescript
// Called from .json as: "$record": "@GetRecord"
// Receives: (request)
function GetRecord(request: Request): any {
  const id = request.params.id; // Access route params via request
  return Process("models.record.Find", id);
}

// Called from .json as: { "process": "@GetItems", "args": ["active", 20] }
// Receives: ("active", 20, request)
function GetItems(status: string, limit: number, request: Request): any[] {
  return Process("models.item.Get", {
    wheres: [{ column: "status", value: status }],
    limit: limit,
  });
}
```

> **⚠️ Common Mistake**: You cannot use `$param.id` directly in backend scripts. The `$param`, `$query`, etc. variables are only available in HTML templates and `.json` configurations. In backend scripts, access these values via the `request` parameter: `request.params.id`, `request.query.search`, etc.

## Built-in Functions

### P\_() - Process Call

Call a Yao process directly in templates:

```html
<!-- Simple call -->
<span>{{ P_('utils.formatDate', createdAt) }}</span>

<!-- With multiple arguments -->
<span>{{ P_('utils.calculate', price, quantity, discount) }}</span>

<!-- In conditions -->
<div s:if="{{ P_('auth.hasPermission', 'admin') }}">Admin Panel</div>
```

### True() / False()

Check boolean values:

```html
<div s:if="{{ True(user) }}">User exists</div>
<div s:if="{{ False(error) }}">No error</div>

<!-- Equivalent to -->
<div s:if="{{ user != null && user != false && user != 0 }}">User exists</div>
```

### Empty()

Check if array or object is empty:

```html
<div s:if="{{ Empty(items) }}">No items</div>
<div s:if="{{ !Empty(items) }}">{{ items.length }} items found</div>

<!-- Works with objects too -->
<div s:if="{{ Empty(settings) }}">No settings configured</div>
```

## Global Data (`__data.json`)

Define global data available to all pages:

**`/templates/<template>/__data.json`**:

```json
{
  "title": "My Application",
  "version": "1.0.0",
  "company": {
    "name": "ACME Inc",
    "email": "contact@acme.com"
  },
  "navigation": [
    { "label": "Home", "href": "/" },
    { "label": "About", "href": "/about" }
  ]
}
```

Access in templates:

```html
<title>{{ $global.title }}</title>
<footer>© {{ $global.company.name }}</footer>

<nav>
  <a s:for="{{ $global.navigation }}" s:for-item="item" href="{{ item.href }}">
    {{ item.label }}
  </a>
</nav>
```

## Backend Script Data

Data returned from `BeforeRender` is merged with page data:

**`<page>.backend.ts`**:

```typescript
function BeforeRender(request: Request): Record<string, any> {
  return {
    user: Process("session.Get", "user"),
    notifications: Process("models.notification.Get", {
      wheres: [{ column: "read", value: false }],
      limit: 5,
    }),
    serverTime: new Date().toISOString(),
  };
}
```

**`<page>.html`**:

```html
<div s:if="{{ user }}">
  Welcome, {{ user.name }}!
  <span s:if="{{ !Empty(notifications) }}">
    {{ notifications.length }} new notifications
  </span>
</div>
<footer>Server time: {{ serverTime }}</footer>
```

## Data Priority

When the same key exists in multiple sources, priority is:

1. **BeforeRender** (highest) - Backend script data
2. **`<page>.json`** - Page data configuration
3. **`__data.json`** (lowest) - Global data

```typescript
// BeforeRender returns { title: "From Backend" }
// page.json has { title: "From JSON" }
// __data.json has { title: "Global Title" }

// Result: title = "From Backend"
```
