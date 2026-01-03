# Iframe Integration

Agent Pages can be embedded in CUI via `/web/` routes. This document covers the iframe communication mechanism between embedded pages and the CUI host.

## Route Mapping

Pages are accessible via:

```
/web/<assistant-id>/<page-path>
```

Example:

| Page File                  | URL                               |
| -------------------------- | --------------------------------- |
| `pages/index/index.html`   | `/web/my-assistant/index`         |
| `pages/result/index.html`  | `/web/my-assistant/result`        |
| `pages/report/detail.html` | `/web/my-assistant/report/detail` |

## URL Parameters

CUI automatically injects context via URL parameters:

| Parameter  | Value                  | Description   |
| ---------- | ---------------------- | ------------- |
| `__theme`  | `light` / `dark`       | Current theme |
| `__locale` | `en-us`, `zh-cn`, etc. | User locale   |

> **Note**: Authentication uses secure HTTP-only cookies, so `__token` parameter is not needed.

**Usage in page URL:**

```
/web/my-assistant/result?theme=__theme&locale=__locale
```

CUI replaces `__theme`, `__locale` with actual values before loading.

## Message Communication

### Receiving Setup Message

When the iframe loads, CUI sends a `setup` message:

```typescript
// In your page script
window.addEventListener("message", (e) => {
  if (e.data.type === "setup") {
    const { theme, locale } = e.data.message;
    // Apply theme, set locale
    document.documentElement.setAttribute("data-theme", theme);
  }
});
```

### Sending Actions to CUI

Pages can trigger CUI actions via `postMessage` using the unified Action system:

```typescript
// Send action to parent CUI
window.parent.postMessage(
  {
    type: "action",
    message: {
      name: "notify.success",
      payload: { message: "Operation completed" },
    },
  },
  window.location.origin
);
```

### Action Types

#### Navigate

| Action          | Description                     | Payload                                     |
| --------------- | ------------------------------- | ------------------------------------------- |
| `navigate`      | Open page in sidebar or new tab | `{ route, title?, icon?, query?, target? }` |
| `navigate.back` | Navigate back in history        | -                                           |

**Navigate Payload:**

| Field    | Type                     | Required | Description                                     |
| -------- | ------------------------ | -------- | ----------------------------------------------- |
| `route`  | `string`                 | ✅       | Target route (`$dashboard/xxx`, `/xxx`, or URL) |
| `title`  | `string`                 | -        | Page title (shows title bar with back button)   |
| `icon`   | `string`                 | -        | Tab icon (e.g., `material-folder`)              |
| `query`  | `Record<string, string>` | -        | Query parameters                                |
| `target` | `'_self'` \| `'_blank'`  | -        | `_self` (sidebar) or `_blank` (new window)      |

#### Notify

| Action           | Description               | Payload                                    |
| ---------------- | ------------------------- | ------------------------------------------ |
| `notify.success` | Show success notification | `{ message, duration?, icon?, closable? }` |
| `notify.error`   | Show error notification   | `{ message, duration?, icon?, closable? }` |
| `notify.warning` | Show warning notification | `{ message, duration?, icon?, closable? }` |
| `notify.info`    | Show info notification    | `{ message, duration?, icon?, closable? }` |

#### App

| Action            | Description              |
| ----------------- | ------------------------ |
| `app.menu.reload` | Refresh application menu |

#### Modal

| Action        | Description       |
| ------------- | ----------------- |
| `modal.open`  | Open modal dialog |
| `modal.close` | Close modal       |

#### Table

| Action          | Description          |
| --------------- | -------------------- |
| `table.search`  | Trigger table search |
| `table.refresh` | Refresh table data   |
| `table.save`    | Save table row       |
| `table.delete`  | Delete table row(s)  |

#### Form

| Action            | Description           |
| ----------------- | --------------------- |
| `form.find`       | Load form data by ID  |
| `form.submit`     | Submit form           |
| `form.reset`      | Reset form            |
| `form.setFields`  | Set form field values |
| `form.fullscreen` | Toggle fullscreen     |

#### MCP (Client-side)

| Action              | Description        |
| ------------------- | ------------------ |
| `mcp.tool.call`     | Execute MCP tool   |
| `mcp.resource.read` | Read MCP resource  |
| `mcp.resource.list` | List MCP resources |
| `mcp.prompt.get`    | Get MCP prompt     |
| `mcp.prompt.list`   | List MCP prompts   |

#### Event

| Action       | Description       |
| ------------ | ----------------- |
| `event.emit` | Emit custom event |

#### Confirm

| Action    | Description              |
| --------- | ------------------------ |
| `confirm` | Show confirmation dialog |

### Receiving Events from CUI

CUI can send messages to iframe via `web/sendMessage` event:

```typescript
// In your page script
window.addEventListener("message", (e) => {
  const { type, message } = e.data;

  switch (type) {
    case "setup":
      // Initial setup with theme, locale
      break;
    case "refresh":
      // CUI requests page refresh
      location.reload();
      break;
    case "data":
      // CUI sends data update
      handleDataUpdate(message);
      break;
  }
});
```

## Complete Example

### Page HTML (pages/result/index.html)

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Result Page</title>
    <script src="@assets/js/result.js"></script>
  </head>
  <body>
    <div id="app"></div>
  </body>
</html>
```

### Page Script (pages/result/result.ts)

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

// Helper: Send action to CUI parent
const sendAction = (name: string, payload?: any) => {
  try {
    window.parent.postMessage(
      { type: "action", message: { name, payload } },
      window.location.origin
    );
  } catch (err) {
    console.error("Failed to send action to parent:", err);
  }
};

// Initialize message listener
function init() {
  window.addEventListener("message", (e) => {
    if (e.origin !== window.location.origin) return;

    const { type, message } = e.data;
    switch (type) {
      case "setup":
        // Apply theme, locale from CUI
        document.documentElement.setAttribute("data-theme", message.theme);
        break;
      case "update":
        // Handle data updates from CUI
        console.log("Received update:", message);
        break;
    }
  });

  // Make helper available globally
  (window as any).sendAction = sendAction;
}

init();

// Event handler: Show success notification
self.HandleSuccess = (event: Event, data: EventData) => {
  sendAction("notify.success", { message: data.message || "Success!" });
};

// Event handler: Navigate to page
self.HandleNavigate = (event: Event, data: EventData) => {
  sendAction("navigate", {
    route: data.path,
    title: data.title,
  });
};

// Event handler: Close sidebar
self.HandleClose = () => {
  sendAction("event.emit", { key: "app/closeSidebar", value: {} });
};

// Event handler: Call backend and display result
self.HandleQuery = async (event: Event, data: EventData) => {
  try {
    const result = await $Backend().Call("Query", data.id);
    console.log(result);
  } catch (error: any) {
    sendAction("notify.error", { message: error.message });
  }
};
```

## Triggering from Hooks

Open page in sidebar from agent hooks:

```typescript
function Next(ctx: agent.Context, payload: agent.Payload): agent.Next {
  // Open result page in sidebar
  ctx.Send({
    type: "action",
    props: {
      name: "navigate",
      payload: {
        route: `/agents/my-assistant/result`,
        title: "Results",
        query: { id: resultId },
      },
    },
  });

  return null;
}
```

See [Pages](pages.md) for more details on triggering pages from hooks.

## Security Notes

1. **Same-origin only**: Messages are only processed from same-origin iframes
2. **Secure cookies**: Authentication uses HTTP-only cookies, no token in URL
3. **Validate messages**: Always validate message structure before processing
