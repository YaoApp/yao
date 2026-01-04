# Agent SUI

Agent SUI is a special SUI configuration designed for AI Agent applications. It automatically loads pages from the `/agent/template/` directory and individual assistant pages from `/assistants/<name>/pages/`.

## Directory Structure

```
<app>/
├── agent/
│   ├── agent.yao              # Agent configuration
│   └── template/              # Agent SUI template directory
│       ├── template.json      # Optional template configuration
│       ├── __document.html    # Global document template
│       ├── __data.json        # Global data (accessible via $global)
│       ├── __assets/          # Global assets (reference via @assets/)
│       │   ├── css/
│       │   ├── js/
│       │   └── images/
│       ├── __locales/         # Global locale files
│       └── pages/             # Global pages (401, 404, login, etc.)
│           └── <page>/        # Route = folder name
│               ├── <page>.html
│               ├── <page>.css
│               ├── <page>.ts
│               └── __locales/ # Page-level locale files
│
└── assistants/                # Assistants directory
    └── <name>/                # Assistant
        ├── package.yao        # Assistant configuration
        └── pages/             # Assistant pages → /agents/<name>/<route>
            └── <page>/        # Route = folder name (can be nested)
                ├── <page>.html
                ├── <page>.css
                ├── <page>.ts
                └── __locales/
```

## Route Mapping

| File Path                                          | Public URL                 |
| -------------------------------------------------- | -------------------------- |
| `/agent/template/pages/login/login.html`           | `/agents/login`            |
| `/assistants/demo/pages/index/index.html`          | `/agents/demo/index`       |
| `/assistants/another/pages/settings/settings.html` | `/agents/another/settings` |

## Asset Paths

- **Global assets**: `/agents/assets/...` → `/agent/template/__assets/...`
- **Assistant assets**: `/agents/<assistant-id>/assets/...` → `/assistants/<assistant-id>/pages/__assets/...`

## Build Commands

```bash
# Build Agent SUI
yao sui build agent

# Watch Agent SUI for changes
yao sui watch agent
```

## Build Output

After running `yao sui build agent`, the following structure is generated:

```
<app>/public/
└── agents/                        # Public root for Agent SUI
    ├── assets/                    # Static assets
    │   ├── libsui.min.js          # SUI frontend SDK
    │   ├── libsui.min.js.map      # Source map
    │   ├── css/                   # From /agent/template/__assets/css/
    │   ├── js/                    # From /agent/template/__assets/js/
    │   └── images/                # From /agent/template/__assets/images/
    │
    ├── login.sui                  # Compiled page
    ├── login.cfg                  # Page configuration
    │
    ├── demo/                      # Assistant: demo
    │   ├── index.sui              # Compiled page
    │   └── index.cfg              # Page configuration
    │
    └── another/                   # Assistant: another
        ├── settings.sui           # Compiled page
        └── settings.cfg           # Page configuration
```

**File Types:**

| Extension | Description                                             |
| --------- | ------------------------------------------------------- |
| `.sui`    | Compiled HTML page (includes template, styles, scripts) |
| `.cfg`    | Page configuration (JSON format)                        |
| `.jit`    | JIT component (for dynamic loading)                     |

## Auto-Loading

Agent SUI is automatically loaded when:

1. The `/agent/template/` directory exists
2. At least one assistant has a `pages/` directory

No additional configuration is required.

## Document Template

Create `/agent/template/__document.html`:

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

## Global Data

Create `/agent/template/__data.json`:

```json
{
  "title": "AI Agent",
  "version": "1.0.0",
  "theme": "light"
}
```

## Example Assistant Page

**`/assistants/demo/pages/index/index.html`**:

```html
<div id="demo-index" class="page">
  <h1>{{ title }}</h1>
  <div class="content">
    <p>{{ description }}</p>
  </div>
</div>
```

**`/assistants/demo/pages/index/index.json`**:

```json
{
  "title": "Welcome",
  "description": "This is a demo page"
}
```

**`/assistants/demo/pages/index/index.css`**:

```css
.page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}
```

## Page Configuration

Create `<page>.config` for page settings:

```json
{
  "title": "Page Title",
  "guard": "oauth",
  "api": {
    "defaultGuard": "oauth"
  }
}
```

### Available Guards

| Guard          | Description                                |
| -------------- | ------------------------------------------ |
| `oauth`        | OAuth 2.1 authentication (recommended)     |
| `bearer-jwt`   | Bearer token JWT authentication            |
| `cookie-jwt`   | Cookie-based JWT authentication            |
| `-`            | No authentication (public access)          |

> See [Page Configuration](./page-config.md) for complete configuration options.

## Backend Scripts

Each page can have a backend script:

**`/assistants/demo/pages/index/index.backend.ts`**:

```typescript
function BeforeRender(request: Request): Record<string, any> {
  return {
    user: Process("session.Get", "user"),
    data: Process("models.data.Get", {}),
  };
}

function ApiGetData(request: Request): any {
  return Process("models.data.Get", {});
}
```

## Using Components

Pages can use other pages as components:

```html
<import s:as="Header" s:from="/shared/header" />
<import s:as="Footer" s:from="/shared/footer" />

<div class="page">
  <header title="Demo" />
  <main>
    <p>Content here</p>
  </main>
  <footer />
</div>
```

## Accessing in Templates

Use standard SUI template syntax:

```html
<!-- Data binding -->
<h1>{{ title }}</h1>

<!-- Conditionals -->
<div s:if="{{ isLoggedIn }}">Welcome!</div>

<!-- Loops -->
<ul>
  <li s:for="{{ items }}" s:for-item="item">{{ item.name }}</li>
</ul>

<!-- Events -->
<button s:on-click="handleClick">Click Me</button>
```

## Frontend Script

Frontend scripts can be written in two styles:

### Direct Style (Simple Pages)

```typescript
// Runs immediately when script loads
document.addEventListener("DOMContentLoaded", () => {
  const form = document.querySelector("#myForm") as HTMLFormElement;

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    // Handle submission
  });
});

// Smooth scrolling
document.querySelectorAll('a[href^="#"]').forEach((anchor) => {
  anchor.addEventListener("click", function (e) {
    e.preventDefault();
    const target = document.querySelector(this.getAttribute("href"));
    target?.scrollIntoView({ behavior: "smooth" });
  });
});
```

### Component Style (Interactive Pages)

**`/assistants/demo/pages/index/index.ts`**:

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

// Event handler bound to s:on-click="HandleClick"
self.HandleClick = async (event: Event, data: EventData) => {
  const result = await $Backend().Call("GetData", data.id);
  console.log(result);
};

// Form submission
self.HandleSubmit = async (event: Event) => {
  event.preventDefault();
  const form = event.target as HTMLFormElement;
  const formData = new FormData(form);
  await $Backend().Call("Submit", Object.fromEntries(formData));
};
```

## CUI Integration

When Agent SUI pages are embedded in CUI via `/web/` routes, they can communicate with the CUI host.

### Receiving Context

```typescript
window.addEventListener("message", (e) => {
  if (e.origin !== window.location.origin) return;

  if (e.data.type === "setup") {
    const { theme, locale } = e.data.message;
    document.documentElement.setAttribute("data-theme", theme);
  }
});
```

### Sending Actions

```typescript
// Helper function
const sendAction = (name: string, payload?: any) => {
  window.parent.postMessage(
    { type: "action", message: { name, payload } },
    window.location.origin
  );
};

// Show notification
sendAction("notify.success", { message: "Done!" });

// Navigate
sendAction("navigate", {
  route: "/agents/demo/detail",
  title: "Details",
});

// Close sidebar
sendAction("event.emit", { key: "app/closeSidebar", value: {} });
```

See [Frontend API - CUI Integration](frontend-api.md#cui-integration) for complete documentation.
