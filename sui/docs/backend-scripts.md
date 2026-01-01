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

Functions prefixed with `Api` are exposed as callable endpoints:

```typescript
// Callable from frontend as: this.backend.ApiGetUsers()
function ApiGetUsers(request: Request): any[] {
  return Process("models.user.Get", {});
}

// Callable from frontend as: this.backend.ApiCreateUser(name, email)
function ApiCreateUser(name: string, email: string, request: Request): any {
  return Process("models.user.Create", { name, email });
}

// Callable from frontend as: this.backend.ApiDeleteUser(id)
function ApiDeleteUser(id: string, request: Request): boolean {
  Process("models.user.Delete", id);
  return true;
}
```

### Calling from Frontend

```typescript
function Page(component: HTMLElement) {
  this.root = component;

  this.loadUsers = async () => {
    const users = await this.backend.ApiGetUsers();
    console.log(users);
  };

  this.createUser = async () => {
    const user = await this.backend.ApiCreateUser("John", "john@example.com");
    console.log("Created:", user);
  };
}
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
function Page(component: HTMLElement) {
  console.log(this.constants.API_URL); // "/api/v1"
  console.log(this.constants.MAX_ITEMS); // 100
}
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
function Page(component: HTMLElement) {
  const formatted = this.helpers.formatDate("2024-01-15");
  const price = this.helpers.formatCurrency(99.99);
  const isValid = this.helpers.validateEmail("test@example.com");
}
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
  authorized?: Record<string, any>; // OAuth info (when guard is "oauth")
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
