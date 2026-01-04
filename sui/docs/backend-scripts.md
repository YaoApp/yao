# Backend Scripts

Backend scripts provide server-side logic for SUI pages, including data fetching, API endpoints, and helper functions.

## File Naming

Backend scripts use the naming convention `<page>.backend.ts` or `<page>.backend.js`:

```
/users/list/
├── list.html
├── list.css
├── list.ts
└── list.backend.ts    # Backend script
```

## Important Notes

> **⚠️ No ES Module Exports**: Backend scripts do NOT support ES Module `export` syntax. Simply define functions directly - they will be automatically available based on naming conventions.

> **⚠️ `$param` Not Available**: Unlike HTML templates, you cannot use `$param.id` directly in backend scripts. Route parameters must be accessed via the `request.params` object passed to your functions.

## BeforeRender

The `BeforeRender` function is called before the page is rendered:

```typescript
function BeforeRender(
  request: Request,
  props?: Record<string, any>
): Record<string, any> {
  return {
    user: Process("session.Get", "user"),
    items: Process("models.item.Get", { limit: 10 }),
  };
}
```

### Parameters

- `request` - The HTTP request object
- `props` - Props passed when used as a component (optional)

### Return Value

Return an object that will be merged with page data:

```typescript
function BeforeRender(request: Request): Record<string, any> {
  const userId = request.query.userId;

  return {
    user: Process("models.user.Find", userId),
    posts: Process("models.post.Get", {
      wheres: [{ column: "user_id", value: userId }],
    }),
    stats: {
      views: 100,
      likes: 50,
    },
  };
}
```

## API Methods

Functions prefixed with `Api` are exposed as callable endpoints. The backend automatically adds the `Api` prefix, so frontend calls use the method name without the prefix:

```typescript
// Callable from frontend as: $Backend().Call("GetUsers")
function ApiGetUsers(request: Request): any[] {
  return Process("models.user.Get", {});
}

// Callable from frontend as: $Backend().Call("CreateUser", name, email)
function ApiCreateUser(name: string, email: string, request: Request): any {
  return Process("models.user.Create", { name, email });
}

// Callable from frontend as: $Backend().Call("DeleteUser", id)
function ApiDeleteUser(id: string, request: Request): boolean {
  Process("models.user.Delete", id);
  return true;
}
```

### Calling from Frontend

```typescript
import { $Backend, Component } from "@yao/sui";

const self = this as Component;

self.LoadUsers = async () => {
  // Call "ApiGetUsers" in backend script (without "Api" prefix)
  const users = await $Backend().Call("GetUsers");
  console.log(users);
};

self.CreateUser = async () => {
  const user = await $Backend().Call("CreateUser", "John", "john@example.com");
  console.log("Created:", user);
};
```

## Constants

Export constants to the frontend using `__sui_constants`:

```typescript
const __sui_constants = {
  API_URL: "/api/v1",
  MAX_ITEMS: 100,
  SUPPORTED_FORMATS: ["jpg", "png", "gif"],
  CONFIG: {
    timeout: 5000,
    retries: 3,
  },
};
```

Access in frontend:

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

console.log(self.constants.API_URL); // "/api/v1"
console.log(self.constants.MAX_ITEMS); // 100
```

## Helpers

Export helper functions to the frontend using `__sui_helpers`:

```typescript
const __sui_helpers = ["formatDate", "formatCurrency", "validateEmail"];

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString();
}

function formatCurrency(amount: number, currency: string = "USD"): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency,
  }).format(amount);
}

function validateEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}
```

Access in frontend:

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

const formatted = self.helpers.formatDate("2024-01-15");
const price = self.helpers.formatCurrency(99.99);
const isValid = self.helpers.validateEmail("test@example.com");
```

## Request Object

The request object contains:

```typescript
interface Request {
  method: string; // HTTP method
  url: {
    path: string;
    host: string;
    domain: string;
    scheme: string;
  };
  query: Record<string, string>; // Query parameters
  params: Record<string, string>; // Route parameters
  payload: Record<string, any>; // POST body
  headers: Record<string, string>; // HTTP headers
  sid: string; // Session ID
  theme: string; // Current theme
  locale: string; // Current locale
  authorized?: {
    // OAuth info (when guard is "oauth")
    sub?: string; // Subject identifier
    user_id?: string; // User ID
    team_id?: string; // Team ID (if team login)
    tenant_id?: string; // Tenant ID (multi-tenancy)
    client_id?: string; // OAuth client ID
    session_id?: string; // Session ID
    scope?: string; // OAuth scopes
    remember_me?: boolean; // Remember me flag

    // Data access constraints (set by ACL)
    constraints?: {
      owner_only?: boolean; // Only access owner's data
      creator_only?: boolean; // Only access creator's data
      editor_only?: boolean; // Only access editor's data
      team_only?: boolean; // Only access team's data
      extra?: Record<string, any>; // Custom constraints
    };
  };
}
```

### Example Usage

```typescript
function BeforeRender(request: Request): Record<string, any> {
  // Access query parameters
  const search = request.query.q;
  const page = parseInt(request.query.page) || 1;

  // Access route parameters
  const userId = request.params.id;

  // Access headers
  const authToken = request.headers["Authorization"];

  // Access session
  const sessionId = request.sid;

  return {
    search,
    page,
    userId,
  };
}
```

## Process Calls

Use `Process()` to call Yao processes:

```typescript
// Model operations
const users = Process("models.user.Get", { limit: 10 });
const user = Process("models.user.Find", userId);
Process("models.user.Save", userId, { name: "Updated" });
Process("models.user.Delete", userId);

// Custom scripts
const result = Process("scripts.utils.calculate", arg1, arg2);

// Session
const sessionUser = Process("session.Get", "user");
Process("session.Set", "key", "value");

// Flows
const output = Process("flows.myflow", input);
```

## Error Handling

```typescript
function ApiUpdateUser(id: string, data: any, request: Request): any {
  try {
    const user = Process("models.user.Find", id);
    if (!user) {
      throw new Error("User not found");
    }

    return Process("models.user.Save", id, data);
  } catch (error) {
    // Error will be returned to frontend
    throw new Error(`Failed to update user: ${error.message}`);
  }
}
```

## Data Binding Methods (Called from `.json`)

In addition to `Api` prefixed methods (for frontend calls) and `BeforeRender`, you can define methods that are called directly from the page's `.json` configuration using the `@MethodName` syntax.

### Naming Convention

| Call Source                  | Function Name   | Example Call                    |
| ---------------------------- | --------------- | ------------------------------- |
| Frontend `$Backend().Call()` | `ApiMethodName` | `$Backend().Call("MethodName")` |
| `.json` data binding         | `MethodName`    | `"$data": "@MethodName"`        |
| Before render                | `BeforeRender`  | Automatic                       |

### How It Works

When using `@MethodName` in `.json`, SUI calls the backend function with the **Request object appended as the last argument**:

```typescript
// In .json: "$record": "@GetRecord"
// SUI internally calls: GetRecord(request)

function GetRecord(request: Request): any {
  // Access route parameters via request.params
  const id = request.params.id;
  return Process("models.record.Find", id);
}
```

### With Additional Arguments

You can also pass arguments from `.json`:

```json
{
  "$items": {
    "process": "@GetItems",
    "args": ["category_a", 10]
  }
}
```

```typescript
// SUI calls: GetItems("category_a", 10, request)
// Arguments from .json come first, request is appended last

function GetItems(category: string, limit: number, request: Request): any[] {
  return Process("models.item.Get", {
    wheres: [{ column: "category", value: category }],
    limit: limit,
  });
}
```

### Common Pitfall: Accessing Route Parameters

❌ **Wrong** - `$param` is not available in backend scripts:

```typescript
function GetRecord(): any {
  const id = $param.id; // ReferenceError: $param is not defined
  return Process("models.record.Find", id);
}
```

✅ **Correct** - Use `request.params`:

```typescript
function GetRecord(request: Request): any {
  const id = request.params.id; // Works!
  return Process("models.record.Find", id);
}
```

## Complete Example

**`/users/profile/profile.backend.ts`**:

```typescript
// Constants exported to frontend
const __sui_constants = {
  MAX_BIO_LENGTH: 500,
  ALLOWED_AVATAR_TYPES: ["image/jpeg", "image/png"],
};

// Helper functions exported to frontend
const __sui_helpers = ["formatDate", "truncate"];

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString();
}

function truncate(text: string, length: number): string {
  if (text.length <= length) return text;
  return text.slice(0, length) + "...";
}

// Called before page render
function BeforeRender(request: Request): Record<string, any> {
  const userId = request.params.id;
  const user = Process("models.user.Find", userId);

  if (!user) {
    return { error: "User not found" };
  }

  const posts = Process("models.post.Get", {
    wheres: [{ column: "user_id", value: userId }],
    orders: [{ column: "created_at", option: "desc" }],
    limit: 10,
  });

  return {
    user,
    posts,
    isOwner: request.sid === user.session_id,
  };
}

// API: Get user posts
function ApiGetPosts(userId: string, page: number, request: Request): any {
  return Process("models.post.Paginate", {
    wheres: [{ column: "user_id", value: userId }],
    orders: [{ column: "created_at", option: "desc" }],
    page,
    pageSize: 10,
  });
}

// API: Update profile
function ApiUpdateProfile(data: any, request: Request): any {
  const sessionUser = Process("session.Get", "user");
  if (!sessionUser) {
    throw new Error("Not authenticated");
  }

  return Process("models.user.Save", sessionUser.id, {
    name: data.name,
    bio: data.bio?.slice(0, 500),
  });
}

// API: Upload avatar
function ApiUploadAvatar(file: any, request: Request): any {
  const sessionUser = Process("session.Get", "user");
  if (!sessionUser) {
    throw new Error("Not authenticated");
  }

  const result = Process("fs.system.Upload", file);
  Process("models.user.Save", sessionUser.id, {
    avatar: result.path,
  });

  return { avatar: result.path };
}
```
