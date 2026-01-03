# Event Handling

SUI provides a declarative event binding system with state management and component communication.

## Event Binding

### Basic Events

Use `s:on-<event>` to bind events:

```html
<button s:on-click="handleClick">Click Me</button>
<input s:on-input="handleInput" />
<form s:on-submit="handleSubmit">...</form>
```

### Common Events

| Attribute         | Event      | Description        |
| ----------------- | ---------- | ------------------ |
| `s:on-click`      | click      | Mouse click        |
| `s:on-dblclick`   | dblclick   | Double click       |
| `s:on-input`      | input      | Input value change |
| `s:on-change`     | change     | Value changed      |
| `s:on-submit`     | submit     | Form submission    |
| `s:on-focus`      | focus      | Element focused    |
| `s:on-blur`       | blur       | Element lost focus |
| `s:on-keydown`    | keydown    | Key pressed        |
| `s:on-keyup`      | keyup      | Key released       |
| `s:on-mouseenter` | mouseenter | Mouse entered      |
| `s:on-mouseleave` | mouseleave | Mouse left         |

### Multiple Events

```html
<input
  s:on-input="handleInput"
  s:on-focus="handleFocus"
  s:on-blur="handleBlur"
  s:on-keydown="handleKeydown"
/>
```

## Passing Data

### Data Attributes

Use `s:data-*` to pass string data:

```html
<button
  s:on-click="deleteItem"
  s:data-id="{{ item.id }}"
  s:data-name="{{ item.name }}"
>
  Delete
</button>
```

### JSON Data

Use `s:json-*` to pass complex data:

```html
<button
  s:on-click="editItem"
  s:json-item="{{ item }}"
  s:json-options="{{ { confirm: true, redirect: '/list' } }}"
>
  Edit
</button>
```

## Event Handlers

### Handler Signature

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

self.HandleClick = (event: Event, data: EventData) => {
  // event - The DOM event
  // data - Combined data from s:data-* and s:json-*
};
```

### EventData

```typescript
interface EventData {
  [key: string]: any; // Data from s:data-* and s:json-* attributes
}
```

### Example

```html
<div class="item-list">
  <div s:for="{{ items }}" s:for-item="item">
    <span>{{ item.name }}</span>
    <button
      s:on-click="DeleteItem"
      s:data-id="{{ item.id }}"
      s:json-item="{{ item }}"
    >
      Delete
    </button>
  </div>
</div>
```

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

self.DeleteItem = async (event: Event, data: EventData) => {
  const id = data.id; // String from s:data-id
  const item = data.item; // Object from s:json-item

  if (confirm(`Delete ${item.name}?`)) {
    await $Backend().Call("DeleteItem", id);
    (event.target as HTMLElement).closest(".item")?.remove();
  }
};
```

## State Management

### State Object

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

// Initial state
self.state.Set("count", 0);
```

### State Watchers

React to state changes with watchers:

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

// Define watchers
self.watch = {
  count: (value: number) => {
    self.root.querySelector(".count")!.textContent = String(value);
  },

  items: (value: any[]) => {
    renderItems(value);
  },
};

self.Increment = () => {
  const count = self.state.Get("count") || 0;
  self.state.Set("count", count + 1); // Triggers watcher
};
```

### Stop Propagation

Prevent state changes from bubbling to parent:

```typescript
self.watch = {
  localState: (value: any, state: any) => {
    // Handle locally
    updateUI(value);

    // Stop propagation to parent components
    state.stopPropagation();
  },
};
```

## Store (Data Attributes)

Store manages `data-*` attributes on the component:

### Basic Usage

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

// Get/Set string values
const id = self.store.Get("id");
self.store.Set("id", "123");

// Get/Set JSON values
const items = self.store.GetJSON("items");
self.store.SetJSON("items", [{ id: 1 }, { id: 2 }]);
```

### Component Data

Get data from BeforeRender:

```typescript
// Backend returns: { user: { name: "John" }, settings: {...} }
const data = self.store.GetData();
console.log(data.user.name); // "John"
```

## Custom Events

### Emit Events

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

self.SelectItem = () => {
  const item = self.store.GetJSON("item");

  // Emit custom event
  self.emit("item:selected", { item });
};
```

### Listen to Events

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

// Listen to child events
self.root.addEventListener("item:selected", (e: CustomEvent) => {
  const { item } = e.detail;
  console.log("Selected:", item);
});
```

### State Change Events

Parent components can listen to state changes:

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

self.root.addEventListener("state:change", (e: CustomEvent) => {
  const { key, value, target } = e.detail;
  console.log(`State ${key} changed to ${value} in`, target);
});
```

## Form Handling

### Form Submit

```html
<form s:on-submit="HandleSubmit">
  <input name="email" type="email" required />
  <input name="password" type="password" required />
  <button type="submit">Login</button>
</form>
```

```typescript
import { $Backend, Component } from "@yao/sui";

const self = this as Component;

self.HandleSubmit = async (event: Event) => {
  event.preventDefault();

  const form = event.target as HTMLFormElement;
  const formData = new FormData(form);

  const email = formData.get("email");
  const password = formData.get("password");

  try {
    await $Backend().Call("Login", email, password);
    window.location.href = "/dashboard";
  } catch (error) {
    alert("Login failed");
  }
};
```

### Input Binding

```html
<input type="text" s:on-input="HandleInput" s:data-field="name" />
```

```typescript
import { Component, EventData } from "@yao/sui";

const self = this as Component;
const formData: Record<string, string> = {};

self.HandleInput = (event: Event, data: EventData) => {
  const input = event.target as HTMLInputElement;
  formData[data.field] = input.value;
};
```

## Keyboard Events

```html
<input s:on-keydown="HandleKeydown" s:on-keyup="HandleKeyup" />
```

```typescript
import { Component } from "@yao/sui";

const self = this as Component;

self.HandleKeydown = (event: KeyboardEvent) => {
  if (event.key === "Enter") {
    search();
  }

  if (event.key === "Escape") {
    clear();
  }
};
```

## Complete Example

```html
<div class="todo-app">
  <form s:on-submit="AddTodo">
    <input
      name="title"
      placeholder="Add todo..."
      s:on-keydown="HandleKeydown"
    />
    <button type="submit">Add</button>
  </form>

  <ul class="todo-list">
    <li s:for="{{ todos }}" s:for-item="todo">
      <input
        type="checkbox"
        s:on-change="ToggleTodo"
        s:data-id="{{ todo.id }}"
        s:attr-checked="{{ todo.completed }}"
      />
      <span class="{{ todo.completed ? 'completed' : '' }}">
        {{ todo.title }}
      </span>
      <button s:on-click="DeleteTodo" s:data-id="{{ todo.id }}">×</button>
    </li>
  </ul>
</div>
```

```typescript
import { $Backend, Component, EventData } from "@yao/sui";

const self = this as Component;

self.watch = {
  todos: (todos: any[]) => {
    self.render("todoList", { todos });
  },
};

self.AddTodo = async (event: Event) => {
  event.preventDefault();
  const form = event.target as HTMLFormElement;
  const input = form.querySelector("input") as HTMLInputElement;

  if (input.value.trim()) {
    const todo = await $Backend().Call("AddTodo", input.value);
    const todos = self.state.Get("todos") || [];
    self.state.Set("todos", [...todos, todo]);
    input.value = "";
  }
};

self.ToggleTodo = async (event: Event, data: EventData) => {
  const checkbox = event.target as HTMLInputElement;
  await $Backend().Call("ToggleTodo", data.id, checkbox.checked);
};

self.DeleteTodo = async (event: Event, data: EventData) => {
  await $Backend().Call("DeleteTodo", data.id);
  const todos = self.state.Get("todos").filter((t: any) => t.id !== data.id);
  self.state.Set("todos", todos);
};
```
