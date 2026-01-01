# Frontend API

SUI provides a rich frontend API for component interaction, backend calls, and rendering.

## Component Query

### $$() Function

Get a component instance by selector or element:

```typescript
// By ID
const card = $$("#my-card");

// By element
const element = document.querySelector(".card");
const card = $$(element);

// Access component methods
card.toggle();
card.state.Set("expanded", true);
```

### Query Methods

```typescript
const component = $$("#my-component");

// Find child component (returns __Query wrapper)
const button = component.find("button");

// Query single element
const title = component.query(".title"); // Returns Element

// Query all elements
const items = component.queryAll(".item"); // Returns NodeList
```

## Backend Calls

### Via Component

```typescript
function Page(component: HTMLElement) {
  this.root = component;

  this.loadData = async () => {
    // Call backend API methods
    const users = await this.backend.ApiGetUsers();
    const user = await this.backend.ApiGetUser(123);
    const result = await this.backend.ApiCreateUser("John", "john@example.com");
  };
}
```

### Direct Call

```typescript
// __sui_backend_call(route, headers, method, ...args)
const result = await __sui_backend_call(
  "/users/list", // Page route
  { "X-Custom-Header": "value" }, // Custom headers
  "ApiGetUsers", // Method name
  { page: 1, limit: 10 } // Arguments
);
```

## Render API

### Render Target

Define render targets in HTML:

```html
<div s:render="userList" class="user-list">
  <!-- Content will be replaced here -->
</div>
```

### Render Method

```typescript
function Page(component: HTMLElement) {
  this.root = component;

  this.refreshUsers = async () => {
    const users = await this.backend.ApiGetUsers();

    // Render with data
    await this.render("userList", { users });
  };
}
```

### Render Options

```typescript
await this.render("targetName", data, {
  replace: true, // Replace content (default: true)
  showLoader: true, // Show loading indicator
  withPageData: true, // Include page data in render context
  route: "/custom/route", // Use custom route for rendering
});
```

## Yao SDK (Legacy)

The `Yao` class provides HTTP client functionality:

```typescript
const yao = new Yao();

// GET request
const data = await yao.Get("/api/users", { page: 1 });

// POST request
const result = await yao.Post("/api/users", { name: "John" });

// Download file
await yao.Download("/api/export", { format: "csv" }, "export.csv");

// Token management
const token = yao.Token();
yao.SetCookie("key", "value", 30); // 30 days
yao.DeleteCookie("key");
```

## OpenAPI Client (Recommended)

The `OpenAPI` client provides a modern HTTP client with type safety and error handling.

### Initialization

```typescript
const api = new OpenAPI({ baseURL: "/api" });
```

### HTTP Methods

```typescript
// GET
const response = await api.Get<User[]>("/users");

// POST
const response = await api.Post<User>("/users", {
  name: "John",
  email: "john@example.com",
});

// PUT
const response = await api.Put<User>("/users/123", {
  name: "John Updated",
});

// DELETE
const response = await api.Delete<void>("/users/123");
```

### Error Handling

```typescript
const response = await api.Get<User[]>("/users");

if (api.IsError(response)) {
  console.error(`Error: ${response.error.error_description}`);
  return;
}

const users = response.data;
```

### Response Types

```typescript
interface APIResponse<T> {
  data: T;
}

interface APIError {
  error: {
    error: string;
    error_description: string;
  };
}
```

## File API

### Initialization

```typescript
const api = new OpenAPI({ baseURL: "/api" });
const fileApi = new FileAPI(api);
```

### Upload

```typescript
const fileInput = document.querySelector<HTMLInputElement>("#file");
const file = fileInput.files[0];

// Upload with progress
const response = await fileApi.Upload(
  file,
  {
    path: "documents",
    groups: ["team-a"],
    compressImage: true,
  },
  (progress) => {
    console.log(`${progress.percentage}%`);
  }
);
```

### Upload Multiple

```typescript
const responses = await fileApi.UploadMultiple(
  Array.from(fileInput.files),
  { path: "uploads" },
  (fileIndex, progress) => {
    console.log(`File ${fileIndex}: ${progress.percentage}%`);
  }
);
```

### File Operations

```typescript
// List files
const files = await fileApi.List({
  page: 1,
  pageSize: 20,
  contentType: "image/*",
  orderBy: "created_at desc",
});

// Get file info
const info = await fileApi.Retrieve("file-id");

// Download
const blob = await fileApi.Download("file-id");
if (!api.IsError(blob)) {
  const url = URL.createObjectURL(blob.data);
  window.open(url);
}

// Delete
await fileApi.Delete("file-id");

// Check existence
const exists = await fileApi.Exists("file-id");
```

### Utility Methods

```typescript
// Format file size
FileAPI.FormatSize(1024); // "1 KB"
FileAPI.FormatSize(1048576); // "1 MB"

// Get extension
FileAPI.GetExtension("doc.pdf"); // "pdf"

// Check type
FileAPI.IsImage("image/png"); // true
FileAPI.IsDocument("application/pdf"); // true
```

## Cross-Origin Support

```typescript
const api = new OpenAPI({ baseURL: "https://api.example.com" });

if (api.IsCrossOrigin()) {
  console.log("Cross-origin API");
}

// Set CSRF token after login
const loginResponse = await api.Post("/auth/login", credentials);
if (!api.IsError(loginResponse) && loginResponse.data.csrf_token) {
  api.SetCSRFToken(loginResponse.data.csrf_token);
}

// Clear tokens on logout
api.ClearTokens();
```

## Custom Events

### Emit

```typescript
function Card(component: HTMLElement) {
  this.root = component;

  this.select = () => {
    this.emit("card:selected", { id: this.store.Get("id") });
  };
}
```

### Listen

```typescript
function CardList(component: HTMLElement) {
  this.root = component;

  this.root.addEventListener("card:selected", (e: CustomEvent) => {
    console.log("Selected:", e.detail.id);
  });
}
```

### State Change Events

```typescript
// Listen to child state changes
this.root.addEventListener("state:change", (e: CustomEvent) => {
  const { key, value, target } = e.detail;
  console.log(`${key} = ${value}`);
});
```

## Complete Example

```typescript
function UserDashboard(component: HTMLElement) {
  this.root = component;
  this.store = new __sui_store(component);
  this.state = new __sui_state(this);

  // Initialize API
  const api = new OpenAPI({ baseURL: "/api" });
  const fileApi = new FileAPI(api);

  // State watchers
  this.watch = {
    users: (users) => this.render("userList", { users }),
    loading: (loading) => {
      this.root.classList.toggle("loading", loading);
    },
  };

  // Load users
  this.loadUsers = async () => {
    this.state.Set("loading", true);

    const response = await api.Get<User[]>("/users");
    if (!api.IsError(response)) {
      this.state.Set("users", response.data);
    }

    this.state.Set("loading", false);
  };

  // Create user
  this.createUser = async (event: Event, data: any) => {
    const response = await this.backend.ApiCreateUser(data.name, data.email);
    const users = this.state.Get("users");
    this.state.Set("users", [...users, response]);
  };

  // Upload avatar
  this.uploadAvatar = async (event: Event) => {
    const input = event.target as HTMLInputElement;
    const file = input.files[0];

    const response = await fileApi.Upload(file, { path: "avatars" });
    if (!api.IsError(response)) {
      this.emit("avatar:uploaded", { url: response.data.url });
    }
  };

  // Initialize
  this.loadUsers();
}
```
