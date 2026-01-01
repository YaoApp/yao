# Agent Pages

Agent Pages provide a built-in SUI (Simple User Interface) framework for building web interfaces for AI agents. Pages are automatically loaded from the `/agent/template/` directory for global templates and `/assistants/<name>/pages/` for individual assistant pages.

## Directory Structure

```
<app>/
├── agent/
│   └── template/              # Global template directory
│       ├── __document.html    # Document template
│       ├── __data.json        # Global data
│       ├── __assets/          # Global assets
│       │   ├── css/
│       │   ├── js/
│       │   └── images/
│       ├── pages/             # Global pages (login, error, etc.)
│       │   └── login/
│       │       └── login.html
│       └── __locales/         # Internationalization
│
└── assistants/
    └── my-assistant/
        ├── package.yao
        └── pages/             # Assistant-specific pages
            ├── index/
            │   ├── index.html
            │   ├── index.css
            │   ├── index.ts
            │   └── index.backend.ts
            └── __assets/      # Optional assistant assets
```

## Route Mapping

| File Path                                 | Public URL           |
| ----------------------------------------- | -------------------- |
| `/agent/template/pages/login/login.html`  | `/agents/login`      |
| `/assistants/demo/pages/index/index.html` | `/agents/demo/index` |
| `/assistants/demo/pages/chat/chat.html`   | `/agents/demo/chat`  |

## Quick Start

### 1. Create Document Template

**`/agent/template/__document.html`**:

```html
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <title>{{ $global.title }}</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" href="/agents/assets/images/favicon.png" />
  </head>
  <body>
    <div class="container">{{ __page }}</div>
  </body>
</html>
```

### 2. Create Global Data

**`/agent/template/__data.json`**:

```json
{
  "title": "AI Agent",
  "version": "1.0.0"
}
```

### 3. Create a Page

**`/assistants/my-assistant/pages/index/index.html`**:

```html
<div id="chat-page" class="page">
  <h1>{{ title }}</h1>
  <div class="messages" s:for="{{ messages }}" s:for-item="msg">
    <div class="message {{ msg.role }}">{{ msg.content }}</div>
  </div>
  <input
    type="text"
    s:on-keypress="handleInput"
    placeholder="Type a message..."
  />
</div>
```

**`/assistants/my-assistant/pages/index/index.json`**:

```json
{
  "title": "Chat",
  "messages": []
}
```

**`/assistants/my-assistant/pages/index/index.css`**:

```css
.page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}

.messages {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message.user {
  align-self: flex-end;
  background: #007bff;
  color: white;
}

.message.assistant {
  align-self: flex-start;
  background: #f0f0f0;
}
```

### 4. Add Backend Script

**`/assistants/my-assistant/pages/index/index.backend.ts`**:

```typescript
function BeforeRender(request: Request): Record<string, any> {
  const chatId = request.query.chat_id;
  return {
    messages: chatId ? Process("scripts.chat.GetHistory", chatId) : [],
    user: request.authorized?.user_id,
  };
}

function ApiGetData(request: Request): any {
  const { id } = request.payload;
  return Process("models.data.Find", id, {});
}
```

### 5. Add Frontend Script

**`/assistants/my-assistant/pages/index/index.ts`**:

```typescript
function index(component: HTMLElement) {
  this.root = component;
  this.store = new __sui_store(component);

  this.handleClick = async (event: Event) => {
    const data = await this.backend.ApiGetData({ id: 1 });
    console.log(data);
  };
}
```

### 6. Build and Run

```bash
# Build pages
yao sui build agent

# Or watch for changes
yao sui watch agent

# Start server
yao start
```

Access at: `http://localhost:5099/agents/my-assistant/index`

## Template Syntax

### Data Binding

```html
<!-- Simple binding -->
<h1>{{ title }}</h1>

<!-- Object properties -->
<p>{{ user.name }}</p>

<!-- With default value -->
<p>{{ description || "No description" }}</p>
```

### Conditionals

```html
<div s:if="{{ isLoggedIn }}">Welcome, {{ user.name }}!</div>
<div s:elif="{{ isGuest }}">Welcome, Guest!</div>
<div s:else>Please log in</div>
```

### Loops

```html
<ul>
  <li s:for="{{ items }}" s:for-item="item" s:for-index="i">
    {{ i + 1 }}. {{ item.name }}
  </li>
</ul>
```

### Events

```html
<button s:on-click="handleClick">Click Me</button>
<input s:on-change="handleChange" s:on-keypress="handleKeypress" />
```

### Components

Pages can use other pages as components:

```html
<import s:as="Header" s:from="/shared/header" />
<import s:as="Footer" s:from="/shared/footer" />

<div class="page">
  <header title="My Page" />
  <main>Content</main>
  <footer />
</div>
```

## Built-in Variables

| Variable   | Description                                 |
| ---------- | ------------------------------------------- |
| `$global`  | Global data from `__data.json`              |
| `$query`   | URL query parameters                        |
| `$param`   | URL path parameters                         |
| `$payload` | POST request body                           |
| `$cookie`  | Cookie values                               |
| `$url`     | Current URL info                            |
| `$theme`   | Current theme                               |
| `$locale`  | Current locale                              |
| `$auth`    | OAuth authorization info (if authenticated) |

## Page Configuration

Create `<page>.config` for page settings:

```json
{
  "title": "Page Title",
  "guard": "bearer-jwt",
  "cache": 3600,
  "data": {
    "key": "value"
  }
}
```

## Asset Paths

- **Global assets**: `/agents/assets/...` → `/agent/template/__assets/...`
- **Assistant assets**: `/agents/<id>/assets/...` → `/assistants/<id>/pages/__assets/...`

## Build Output

```
<app>/public/agents/
├── assets/
│   ├── libsui.min.js      # SUI frontend SDK
│   ├── css/               # Global CSS
│   ├── js/                # Global JS
│   └── images/            # Global images
│
├── login.sui              # Global page
├── login.cfg
│
└── my-assistant/
    ├── index.sui          # Assistant page
    └── index.cfg
```

## Authentication

Pages default to public access. To require authentication:

**`/assistants/my-assistant/pages/dashboard/dashboard.config`**:

```json
{
  "guard": "bearer-jwt"
}
```

Available guards:

| Guard        | Description                       |
| ------------ | --------------------------------- |
| `-`          | No authentication (default)       |
| `bearer-jwt` | JWT token in Authorization header |
| `cookie-jwt` | JWT token in cookie               |
| `oauth`      | OAuth 2.0 authentication          |

## Triggering Pages from Hooks

Use `action` messages to open pages in the sidebar during conversation:

```typescript
// Navigate to a page in sidebar
ctx.Send({
  type: "action",
  props: {
    name: "navigate",
    payload: {
      route: "/agents/my-assistant/result", // Page route
      title: "Query Results", // Sidebar title
      query: { id: "123" }, // Passed as $query in page
    },
  },
});

// Open in new tab
ctx.Send({
  type: "action",
  props: {
    name: "navigate",
    payload: {
      route: "/agents/my-assistant/detail",
      target: "_blank",
    },
  },
});
```

### Action Reference

#### Navigate

Open a route in the sidebar or new window.

**Payload:**

| Field    | Type                     | Required | Description                                          |
| -------- | ------------------------ | -------- | ---------------------------------------------------- |
| `route`  | `string`                 | ✅       | Target route or URL                                  |
| `title`  | `string`                 | -        | Page title (shows custom title bar with back button) |
| `icon`   | `string`                 | -        | Tab icon (e.g., `material-folder`)                   |
| `query`  | `Record<string, string>` | -        | Query parameters (passed as `$query` in page)        |
| `target` | `'_self'` \| `'_blank'`  | -        | `_self` (sidebar, default) or `_blank` (new window)  |

**Route Types:**

| Prefix            | Type     | Description                                     |
| ----------------- | -------- | ----------------------------------------------- |
| `$dashboard/`     | CUI Page | Dashboard pages (e.g., `$dashboard/kb` → `/kb`) |
| `/`               | SUI Page | Custom pages (e.g., `/agents/demo/result`)      |
| `http://https://` | External | External URL (loaded in iframe)                 |

**Examples:**

```typescript
// Open agent page in sidebar with title
ctx.Send({
  type: "action",
  props: {
    name: "navigate",
    payload: {
      route: "/agents/my-assistant/result",
      title: "Query Results",
      icon: "material-table_chart",
      query: { id: "123" },
    },
  },
});

// Open CUI dashboard page
ctx.Send({
  type: "action",
  props: {
    name: "navigate",
    payload: { route: "$dashboard/users" },
  },
});

// Open external URL in new tab
ctx.Send({
  type: "action",
  props: {
    name: "navigate",
    payload: {
      route: "https://docs.example.com",
      target: "_blank",
    },
  },
});
```

#### Navigate Back

Navigate back in history.

```typescript
ctx.Send({
  type: "action",
  props: { name: "navigate.back" },
});
```

#### Notify

Show notification messages.

**Actions:**

| Action           | Description                   |
| ---------------- | ----------------------------- |
| `notify.success` | Success notification (green)  |
| `notify.error`   | Error notification (red)      |
| `notify.warning` | Warning notification (yellow) |
| `notify.info`    | Info notification (blue)      |

**Payload:**

| Field      | Type      | Required | Description                                    |
| ---------- | --------- | -------- | ---------------------------------------------- |
| `message`  | `string`  | ✅       | Notification message                           |
| `duration` | `number`  | -        | Auto-close seconds (default: 3, 0 = keep open) |
| `icon`     | `string`  | -        | Custom icon (overrides default)                |
| `closable` | `boolean` | -        | Show close button (default: false)             |

**Examples:**

```typescript
// Success notification
ctx.Send({
  type: "action",
  props: {
    name: "notify.success",
    payload: { message: "Data saved successfully!" },
  },
});

// Error with custom duration
ctx.Send({
  type: "action",
  props: {
    name: "notify.error",
    payload: {
      message: "Operation failed",
      duration: 5,
      closable: true,
    },
  },
});
```

#### App Menu

Refresh application menu/navigation.

```typescript
ctx.Send({
  type: "action",
  props: { name: "app.menu.reload" },
});
```

#### All Actions

| Category | Action              | Description                     |
| -------- | ------------------- | ------------------------------- |
| Navigate | `navigate`          | Open page in sidebar or new tab |
|          | `navigate.back`     | Navigate back in history        |
| Notify   | `notify.success`    | Show success notification       |
|          | `notify.error`      | Show error notification         |
|          | `notify.warning`    | Show warning notification       |
|          | `notify.info`       | Show info notification          |
| App      | `app.menu.reload`   | Refresh application menu        |
| Modal    | `modal.open`        | Open content in modal dialog    |
|          | `modal.close`       | Close modal                     |
| Table    | `table.search`      | Trigger table search            |
|          | `table.refresh`     | Refresh table data              |
|          | `table.save`        | Save table row data             |
|          | `table.delete`      | Delete table row(s)             |
| Form     | `form.find`         | Load form data by ID            |
|          | `form.submit`       | Submit form data                |
|          | `form.reset`        | Reset form to initial state     |
|          | `form.setFields`    | Set form field values           |
| MCP      | `mcp.tool.call`     | Execute MCP tool (client-side)  |
|          | `mcp.resource.read` | Read MCP resource               |
| Event    | `event.emit`        | Emit custom event               |
| Confirm  | `confirm`           | Show confirmation dialog        |

## Frontend API

The SUI frontend SDK provides:

```typescript
// Backend calls
const data = await this.backend.ApiMethodName(payload);

// State management
this.store.Set("key", value);
const value = this.store.Get("key");

// OpenAPI client (if using oauth guard)
const response = await OpenAPI.Get("/api/endpoint");
await OpenAPI.Post("/api/endpoint", data);
```

## Related Documentation

- [SUI Template Syntax](../../sui/docs/template-syntax.md)
- [SUI Data Binding](../../sui/docs/data-binding.md)
- [SUI Components](../../sui/docs/components.md)
- [SUI Frontend API](../../sui/docs/frontend-api.md)
