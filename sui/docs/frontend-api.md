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

### Via $Backend

The backend automatically adds the `Api` prefix to method names, so you call without the prefix:

```typescript
import { $Backend } from "@yao/sui";

// Call backend API methods (backend functions are ApiGetUsers, ApiGetUser, ApiCreateUser)
const users = await $Backend().Call("GetUsers");
const user = await $Backend().Call("GetUser", 123);
const result = await $Backend().Call("CreateUser", "John", "john@example.com");
```

### Direct Call

```typescript
// __sui_backend_call(route, headers, method, ...args)
// Note: method name here also gets Api prefix added automatically
const result = await __sui_backend_call(
  "/users/list", // Page route
  { "X-Custom-Header": "value" }, // Custom headers
  "GetUsers", // Method name (backend has ApiGetUsers)
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
import { $Backend, Component } from "@yao/sui";

const self = this as Component;

self.RefreshUsers = async () => {
  const users = await $Backend().Call("GetUsers");

  // Render with data
  await self.render("userList", { users });
};
```

### Render Options

```typescript
await self.render("targetName", data, {
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
import { Component } from "@yao/sui";

const self = this as Component;

self.Select = () => {
  self.emit("card:selected", { id: self.store.Get("id") });
};
```

### Listen

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

self.root.addEventListener("card:selected", (e: CustomEvent) => {
  console.log("Selected:", e.detail.id);
});
```

### State Change Events

```typescript
// Listen to child state changes
self.root.addEventListener("state:change", (e: CustomEvent) => {
  const { key, value, target } = e.detail;
  console.log(`${key} = ${value}`);
});
```

## Complete Example

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

// Initialize API
const api = new OpenAPI({ baseURL: "/api" });
const fileApi = new FileAPI(api);

// State watchers
self.watch = {
  users: (users: any[]) => self.render("userList", { users }),
  loading: (loading: boolean) => {
    self.root.classList.toggle("loading", loading);
  },
};

// Load users
async function loadUsers() {
  self.state.Set("loading", true);

  const response = await api.Get<User[]>("/users");
  if (!api.IsError(response)) {
    self.state.Set("users", response.data);
  }

  self.state.Set("loading", false);
}

// Create user
self.CreateUser = async (event: Event, data: EventData) => {
  const response = await $Backend().Call("CreateUser", data.name, data.email);
  const users = self.state.Get("users");
  self.state.Set("users", [...users, response]);
};

// Upload avatar
self.UploadAvatar = async (event: Event) => {
  const input = event.target as HTMLInputElement;
  const file = input.files![0];

  const response = await fileApi.Upload(file, { path: "avatars" });
  if (!api.IsError(response)) {
    self.emit("avatar:uploaded", { url: response.data.url });
  }
};

// Initialize
loadUsers();
```

## CUI Integration

When SUI pages are embedded in CUI via `/web/` routes, they can communicate with the CUI host.

### URL Parameters

CUI automatically replaces special parameter values:

| Value      | Replaced With                    |
| ---------- | -------------------------------- |
| `__theme`  | Current theme (`light` / `dark`) |
| `__locale` | Current locale (e.g., `en-us`)   |

> **Note**: Authentication uses secure HTTP-only cookies, no token parameter needed.

### Receiving Messages from CUI

```typescript
window.addEventListener("message", (e) => {
  // Only accept messages from same origin
  if (e.origin !== window.location.origin) return;

  const { type, message } = e.data;
  switch (type) {
    case "setup":
      // Initial context from CUI
      document.documentElement.setAttribute("data-theme", message.theme);
      console.log("Locale:", message.locale);
      break;
    case "update":
      // Data updates from CUI
      handleUpdate(message);
      break;
  }
});
```

### Sending Actions to CUI

Use the unified Action system to trigger CUI operations:

```typescript
// Helper function
const sendAction = (name: string, payload?: any) => {
  window.parent.postMessage(
    { type: "action", message: { name, payload } },
    window.location.origin
  );
};

// Show notification
sendAction("notify.success", { message: "Operation completed!" });
sendAction("notify.error", { message: "Something went wrong" });

// Navigate to page
sendAction("navigate", {
  route: "/agents/my-app/detail",
  title: "Details",
  query: { id: "123" },
});

// Open in new tab
sendAction("navigate", {
  route: "/agents/my-app/report",
  target: "_blank",
});

// Refresh menu
sendAction("app.menu.reload");

// Close sidebar
sendAction("event.emit", { key: "app/closeSidebar", value: {} });
```

### Available Actions

| Category | Action            | Description               | Payload                                     |
| -------- | ----------------- | ------------------------- | ------------------------------------------- |
| Navigate | `navigate`        | Open page in sidebar/tab  | `{ route, title?, icon?, query?, target? }` |
|          | `navigate.back`   | Go back in history        | -                                           |
| Notify   | `notify.success`  | Success notification      | `{ message, duration?, closable? }`         |
|          | `notify.error`    | Error notification        | `{ message, duration?, closable? }`         |
|          | `notify.warning`  | Warning notification      | `{ message, duration?, closable? }`         |
|          | `notify.info`     | Info notification         | `{ message, duration?, closable? }`         |
| App      | `app.menu.reload` | Refresh application menu  | -                                           |
| Modal    | `modal.open`      | Open modal dialog         | `{ ... }`                                   |
|          | `modal.close`     | Close modal               | -                                           |
| Table    | `table.search`    | Trigger table search      | `{ keywords }`                              |
|          | `table.refresh`   | Refresh table data        | -                                           |
| Form     | `form.submit`     | Submit form               | -                                           |
|          | `form.reset`      | Reset form                | -                                           |
| Event    | `event.emit`      | Emit custom event         | `{ key, value }`                            |
| Confirm  | `confirm`         | Show confirmation dialog  | `{ title, content }`                        |

### Complete Example

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

// Helper: Send action to CUI
const sendAction = (name: string, payload?: any) => {
  window.parent.postMessage(
    { type: "action", message: { name, payload } },
    window.location.origin
  );
};

// Initialize CUI communication
function init() {
  window.addEventListener("message", (e) => {
    if (e.origin !== window.location.origin) return;

    if (e.data.type === "setup") {
      const { theme, locale } = e.data.message;
      document.documentElement.setAttribute("data-theme", theme);
    }
  });

  (window as any).sendAction = sendAction;
}

init();

// Event handlers
self.HandleSave = async (event: Event, data: EventData) => {
  try {
    await $Backend().Call("Save", data);
    sendAction("notify.success", { message: "Saved successfully!" });
  } catch (error: any) {
    sendAction("notify.error", { message: error.message });
  }
};

self.HandleViewDetail = (event: Event, data: EventData) => {
  sendAction("navigate", {
    route: `/agents/my-app/detail`,
    title: "Details",
    query: { id: data.id },
  });
};

self.HandleClose = () => {
  sendAction("event.emit", { key: "app/closeSidebar", value: {} });
};
```
