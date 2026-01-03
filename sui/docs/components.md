# Components

In SUI, **every page is a component**. Any page can be embedded into another page using the `is` attribute.

## Core Concept

When a page is used as a component:

1. The page's HTML becomes the component template
2. The page's CSS is automatically scoped
3. The page's TypeScript becomes the component class
4. The page's `backend.ts` provides server-side logic via `BeforeRender`

## Creating a Component

A component is just a page with a single root element:

**`/card/card.html`**:

```html
<div class="card">
  <h3>{{ title }}</h3>
  <div class="card-body">
    <children></children>
  </div>
</div>
```

**`/card/card.css`**:

```css
.card {
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 16px;
}

.card h3 {
  margin: 0 0 12px;
}
```

**`/card/card.ts`**:

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

// self.root - Root element
// self.store - Data store
// self.props - Props from attributes
```

## Using Components

### Basic Usage

Use the `is` attribute to embed a page as a component:

```html
<div is="/card" title="My Card">
  <p>Card content goes here</p>
</div>
```

### With Import Alias

Use `<import>` for cleaner syntax:

```html
<import s:as="Card" s:from="/card" />
<import s:as="Button" s:from="/shared/button" />

<Card title="My Card">
  <p>Content</p>
</Card>

<Button variant="primary">Click Me</Button>
```

## Props

Props are passed as attributes:

```html
<div
  is="/user-card"
  name="{{ user.name }}"
  email="{{ user.email }}"
  avatar="{{ user.avatar }}"
  role="admin"
/>
```

Access props in the component script:

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

// Get single prop
const name = self.props.Get("name");

// Get all props
const allProps = self.props.List();
// { name: "John", email: "john@example.com", avatar: "...", role: "admin" }
```

Access props in backend script:

```typescript
function BeforeRender(
  request: Request,
  props: Record<string, any>
): Record<string, any> {
  const userId = props.userId;
  return {
    user: Process("models.user.Find", userId),
  };
}
```

## Children and Slots

### Children

Use `<children></children>` to render child content:

**Component (`/panel/panel.html`)**:

```html
<div class="panel">
  <div class="panel-header">{{ title }}</div>
  <div class="panel-body">
    <children></children>
  </div>
</div>
```

**Usage**:

```html
<div is="/panel" title="Settings">
  <p>This content appears in the panel body</p>
  <button>Save</button>
</div>
```

### Named Slots

Use `<slot name="xxx">` for multiple content areas:

**Component (`/modal/modal.html`)**:

```html
<div class="modal">
  <div class="modal-header">
    <slot name="header"></slot>
  </div>
  <div class="modal-body">
    <children></children>
  </div>
  <div class="modal-footer">
    <slot name="footer"></slot>
  </div>
</div>
```

**Usage**:

```html
<div is="/modal">
  <slot name="header">
    <h2>Confirmation</h2>
  </slot>

  <p>Are you sure you want to proceed?</p>

  <slot name="footer">
    <button>Cancel</button>
    <button>Confirm</button>
  </slot>
</div>
```

## Dynamic Components

### Variable Component Route

```html
<div is="{{ '/widgets/' + widgetType }}" ...widgetProps></div>
```

### Dynamic Tag

```html
<dynamic route="/components/{{ componentName }}" />
```

## Component Script

### Structure

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

// self.root - Root element (HTMLElement)
// self.store - Data store (data-* attributes)
// self.props - Props (passed attributes)
// self.state - State management

// State watchers
self.watch = {
  propertyName: (value: any, state: any) => {
    // React to state changes
  },
};

// Event handlers (bound to s:on-click="HandleClick")
self.HandleClick = async (event: Event, data: EventData) => {
  const result = await $Backend().Call("Method", data.id);
  // Handle result
};
```

### Store API

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

// String data
self.store.Get("key");
self.store.Set("key", "value");

// JSON data
self.store.GetJSON("items");
self.store.SetJSON("items", [{ id: 1 }]);

// Component data (from BeforeRender)
self.store.GetData();
```

### Props API

```typescript
// Get single prop
const value = self.props.Get("propName");

// Get all props
const props = self.props.List();
```

### State API

```typescript
// Set state (triggers watchers)
self.state.Set("count", 10);

// Watch state changes
self.watch = {
  count: (value: number, state: any) => {
    self.root.querySelector(".count")!.textContent = String(value);
    // state.stopPropagation(); // Prevent bubbling to parent
  },
};
```

## Nested Components

Components can include other components:

```html
<!-- /dashboard/dashboard.html -->
<div class="dashboard">
  <div is="/shared/header" title="Dashboard" />

  <div class="content">
    <div is="/dashboard/stats" data="{{ stats }}" />
    <div is="/dashboard/chart" type="line" data="{{ chartData }}" />
  </div>

  <div is="/shared/footer" />
</div>
```

## Component Backend Script

**`/user-card/user-card.backend.ts`**:

```typescript
function BeforeRender(
  request: Request,
  props: Record<string, any>
): Record<string, any> {
  const userId = props.userId;

  return {
    user: Process("models.user.Find", userId),
    permissions: Process("scripts.auth.GetPermissions", userId),
  };
}

function ApiUpdateUser(userId: string, data: any, request: Request): any {
  return Process("models.user.Save", userId, data);
}
```

## CSS Scoping

Component CSS is automatically scoped using namespace attributes:

**Original CSS**:

```css
.card {
  border: 1px solid #ddd;
}
.card h3 {
  color: #333;
}
```

**Compiled CSS** (scoped):

```css
[s:ns="ns_abc123"] .card {
  border: 1px solid #ddd;
}
[s:ns="ns_abc123"] .card h3 {
  color: #333;
}
```

## Important Notes

1. **Single Root Element**: Components must have exactly one root element
2. **Scoped Styles**: CSS is automatically scoped to prevent conflicts
3. **Recursive Prevention**: SUI detects and prevents recursive component inclusion
4. **Component Pattern**: Use `const self = this as Component` to access component APIs
