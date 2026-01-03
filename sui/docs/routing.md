# Routing

SUI supports file-system based routing with dynamic route parameters and URL rewriting.

## File-System Routing

Pages are organized in directories, with each directory containing a page's files:

```
/pages/
├── index/
│   ├── index.html
│   ├── index.css
│   └── index.ts
├── about/
│   ├── about.html
│   └── about.css
└── users/
    ├── users.html
    └── [id]/              # Dynamic route
        ├── [id].html
        ├── [id].css
        └── [id].ts
```

## Dynamic Routes

Use square brackets `[param]` to create dynamic route segments:

| Directory Structure | URL Pattern      | Example URL          |
| ------------------- | ---------------- | -------------------- |
| `/users/[id]/`      | `/users/:id`     | `/users/123`         |
| `/posts/[slug]/`    | `/posts/:slug`   | `/posts/hello-world` |
| `/[category]/[id]/` | `/:category/:id` | `/electronics/456`   |

### Accessing Route Parameters

**In HTML templates** - Use `$param`:

```html
<h1>User ID: {{ $param.id }}</h1>
<p>Category: {{ $param.category }}</p>
```

**In `.json` configuration**:

```json
{
  "userId": "$param.id",
  "$user": {
    "process": "models.user.Find",
    "args": ["$param.id"]
  }
}
```

**In backend scripts** - Via Request object:

```typescript
function GetRecord(request: Request): any {
  const id = request.params.id;
  return Process("models.record.Find", id);
}
```

> **Note**: `$param` is NOT available as a global variable in backend scripts. You must access route parameters through the `request.params` object.

## URL Rewriting

SUI pages require URL rewriting to map clean URLs to `.sui` page files. Configure rewrite rules in `app.yao`:

```json
{
  "public": {
    "rewrite": [
      { "^\\/assets\\/(.*)$": "/assets/$1" },
      { "^\\/users\\/([^\\/]+)$": "/users/[id].sui" },
      { "^\\/(.*)$": "/$1.sui" }
    ]
  }
}
```

### Rewrite Rule Syntax

Each rule is a JSON object with a regex pattern as the key and the target path as the value:

```json
{ "REGEX_PATTERN": "TARGET_PATH" }
```

- **REGEX_PATTERN**: A regular expression to match the incoming URL
- **TARGET_PATH**: The internal path to route to, can use capture groups (`$1`, `$2`, etc.)

### Rule Processing Order

Rules are processed **in order from top to bottom**. The first matching rule wins. Always place more specific rules before general ones.

### Common Patterns

#### Static Assets (Passthrough)

```json
{ "^\\/assets\\/(.*)$": "/assets/$1" }
```

Passes asset requests directly without modification.

#### Simple Dynamic Route

```json
{ "^\\/users\\/([^\\/]+)$": "/users/[id].sui" }
```

Maps `/users/123` to `/users/[id].sui`, making `123` available as `$param.id`.

#### Nested Dynamic Route

```json
{
  "^\\/users\\/([^\\/]+)\\/posts\\/([^\\/]+)$": "/users/[id]/posts/[postId].sui"
}
```

Maps `/users/123/posts/456` to the nested page, with `$param.id = "123"` and `$param.postId = "456"`.

#### Catch-All for SUI Pages

```json
{ "^\\/(.*)$": "/$1.sui" }
```

Maps any URL to its corresponding `.sui` file. Place this **last** as a fallback.

#### Specific Page Override

```json
{ "^\\/dashboard\\/login(.*)$": "/dashboard/login.sui" },
{ "^\\/dashboard\\/(.*)$": "/dashboard/[id].sui" }
```

The login page is matched first (specific), then other dashboard pages use dynamic routing.

### Complete Example

```json
{
  "public": {
    "rewrite": [
      // Static assets - passthrough
      { "^\\/assets\\/(.*)$": "/assets/$1" },
      { "^\\/images\\/(.*)$": "/images/$1" },

      // Specific pages (before dynamic routes)
      { "^\\/blog\\/new$": "/blog/new.sui" },
      { "^\\/blog\\/([^\\/]+)\\/edit$": "/blog/[id]/edit.sui" },

      // Dynamic routes
      { "^\\/blog\\/([^\\/]+)$": "/blog/[id].sui" },
      {
        "^\\/users\\/([^\\/]+)\\/posts\\/([^\\/]+)$": "/users/[id]/posts/[postId].sui"
      },
      { "^\\/users\\/([^\\/]+)$": "/users/[id].sui" },

      // Fallback - must be last
      { "^\\/(.*)$": "/$1.sui" }
    ]
  }
}
```

### Regex Tips

| Pattern     | Matches            | Description                          |
| ----------- | ------------------ | ------------------------------------ |
| `([^\\/]+)` | Any segment        | Matches characters until next `/`    |
| `(.*)`      | Everything         | Matches any characters including `/` |
| `(\\d+)`    | Numbers only       | Matches numeric IDs                  |
| `([a-z-]+)` | Lowercase + hyphen | Matches slugs like `hello-world`     |

### Debugging Rewrite Rules

1. Check the server logs for route matching information
2. Ensure regex escaping is correct (double backslashes in JSON: `\\/` for `/`)
3. Test specific URLs to verify capture groups work correctly
4. Remember that the `.sui` extension is internal - users access pages without it

## Route Parameters in Different Contexts

| Context         | Access Method       | Example                         |
| --------------- | ------------------- | ------------------------------- |
| HTML Template   | `{{ $param.id }}`   | `<h1>{{ $param.id }}</h1>`      |
| `.json` Config  | `"$param.id"`       | `"userId": "$param.id"`         |
| Backend Script  | `request.params.id` | `const id = request.params.id;` |
| Frontend Script | Read from DOM       | `document.body.dataset.id`      |

### Frontend Access Pattern

Since frontend scripts run in the browser, route params aren't directly available. Pass them via data attributes:

**HTML**:

```html
<div id="page" data-id="{{ $param.id }}">
  <!-- content -->
</div>
```

**Frontend TypeScript**:

```typescript
const pageEl = document.getElementById("page");
const id = pageEl?.dataset.id;
```
