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
function Page(component: HTMLElement) {
  this.root = component;

  this.handleClick = (event: Event, data: any, context: EventContext) => {
    // event - The DOM event
    // data - Combined data from s:data-* and s:json-*
    // context - Event context with element references
  };
}
```

### EventContext

```typescript
interface EventContext {
  rootElement: HTMLElement; // Component root element
  targetElement: HTMLElement; // Element that triggered the event
}
```

### Example

```html
<div class="item-list">
  <div s:for="{{ items }}" s:for-item="item">
    <span>{{ item.name }}</span>
    <button
      s:on-click="deleteItem"
      s:data-id="{{ item.id }}"
      s:json-item="{{ item }}"
    >
      Delete
    </button>
  </div>
</div>
```

```typescript
function ItemList(component: HTMLElement) {
  this.root = component;

  this.deleteItem = async (event: Event, data: any, context: EventContext) => {
    const id = data.id; // String from s:data-id
    const item = data.item; // Object from s:json-item

    if (confirm(`Delete ${item.name}?`)) {
      await this.backend.ApiDeleteItem(id);
      context.targetElement.closest(".item").remove();
    }
  };
}
```

## State Management

### State Object

```typescript
function Counter(component: HTMLElement) {
  this.root = component;
  this.state = new __sui_state(this);

  // Initial state
  this.state.Set("count", 0);
}
```

### State Watchers

React to state changes with watchers:

```typescript
function Counter(component: HTMLElement) {
  this.root = component;
  this.state = new __sui_state(this);

  // Define watchers
  this.watch = {
    count: (value: number, state: State) => {
      this.root.querySelector(".count").textContent = value;
    },

    items: (value: any[], state: State) => {
      this.renderItems(value);
    },
  };

  this.increment = () => {
    const count = this.state.Get("count") || 0;
    this.state.Set("count", count + 1); // Triggers watcher
  };
}
```

### Stop Propagation

Prevent state changes from bubbling to parent:

```typescript
this.watch = {
  localState: (value: any, state: State) => {
    // Handle locally
    this.updateUI(value);

    // Stop propagation to parent components
    state.stopPropagation();
  },
};
```

## Store (Data Attributes)

Store manages `data-*` attributes on the component:

### Basic Usage

```typescript
function Card(component: HTMLElement) {
  this.root = component;
  this.store = new __sui_store(component);

  // Get/Set string values
  const id = this.store.Get("id");
  this.store.Set("id", "123");

  // Get/Set JSON values
  const items = this.store.GetJSON("items");
  this.store.SetJSON("items", [{ id: 1 }, { id: 2 }]);
}
```

### Component Data

Get data from BeforeRender:

```typescript
// Backend returns: { user: { name: "John" }, settings: {...} }
const data = this.store.GetData();
console.log(data.user.name); // "John"
```

## Custom Events

### Emit Events

```typescript
function ItemCard(component: HTMLElement) {
  this.root = component;

  this.selectItem = () => {
    const item = this.store.GetJSON("item");

    // Emit custom event
    this.emit("item:selected", { item });
  };
}
```

### Listen to Events

```typescript
function ItemList(component: HTMLElement) {
  this.root = component;

  // Listen to child events
  this.root.addEventListener("item:selected", (e: CustomEvent) => {
    const { item } = e.detail;
    console.log("Selected:", item);
  });
}
```

### State Change Events

Parent components can listen to state changes:

```typescript
function Parent(component: HTMLElement) {
  this.root = component;

  this.root.addEventListener("state:change", (e: CustomEvent) => {
    const { key, value, target } = e.detail;
    console.log(`State ${key} changed to ${value} in`, target);
  });
}
```

## Form Handling

### Form Submit

```html
<form s:on-submit="handleSubmit">
  <input name="email" type="email" required />
  <input name="password" type="password" required />
  <button type="submit">Login</button>
</form>
```

```typescript
function LoginForm(component: HTMLElement) {
  this.root = component;

  this.handleSubmit = async (event: Event) => {
    event.preventDefault();

    const form = event.target as HTMLFormElement;
    const formData = new FormData(form);

    const email = formData.get("email");
    const password = formData.get("password");

    try {
      await this.backend.ApiLogin(email, password);
      window.location.href = "/dashboard";
    } catch (error) {
      alert("Login failed");
    }
  };
}
```

### Input Binding

```html
<input type="text" s:on-input="handleInput" s:data-field="name" />
```

```typescript
function Form(component: HTMLElement) {
  this.root = component;
  this.formData = {};

  this.handleInput = (event: Event, data: any) => {
    const input = event.target as HTMLInputElement;
    this.formData[data.field] = input.value;
  };
}
```

## Keyboard Events

```html
<input s:on-keydown="handleKeydown" s:on-keyup="handleKeyup" />
```

```typescript
function Search(component: HTMLElement) {
  this.root = component;

  this.handleKeydown = (event: KeyboardEvent) => {
    if (event.key === "Enter") {
      this.search();
    }

    if (event.key === "Escape") {
      this.clear();
    }
  };
}
```

## Complete Example

```html
<div class="todo-app">
  <form s:on-submit="addTodo">
    <input
      name="title"
      placeholder="Add todo..."
      s:on-keydown="handleKeydown"
    />
    <button type="submit">Add</button>
  </form>

  <ul class="todo-list">
    <li s:for="{{ todos }}" s:for-item="todo">
      <input
        type="checkbox"
        s:on-change="toggleTodo"
        s:data-id="{{ todo.id }}"
        s:attr-checked="{{ todo.completed }}"
      />
      <span class="{{ todo.completed ? 'completed' : '' }}">
        {{ todo.title }}
      </span>
      <button s:on-click="deleteTodo" s:data-id="{{ todo.id }}">×</button>
    </li>
  </ul>
</div>
```

```typescript
function TodoApp(component: HTMLElement) {
  this.root = component;
  this.state = new __sui_state(this);
  this.store = new __sui_store(component);

  this.watch = {
    todos: (todos: any[]) => {
      this.render("todoList", { todos });
    },
  };

  this.addTodo = async (event: Event) => {
    event.preventDefault();
    const form = event.target as HTMLFormElement;
    const input = form.querySelector("input") as HTMLInputElement;

    if (input.value.trim()) {
      const todo = await this.backend.ApiAddTodo(input.value);
      const todos = this.state.Get("todos") || [];
      this.state.Set("todos", [...todos, todo]);
      input.value = "";
    }
  };

  this.toggleTodo = async (event: Event, data: any) => {
    const checkbox = event.target as HTMLInputElement;
    await this.backend.ApiToggleTodo(data.id, checkbox.checked);
  };

  this.deleteTodo = async (event: Event, data: any) => {
    await this.backend.ApiDeleteTodo(data.id);
    const todos = this.state.Get("todos").filter((t) => t.id !== data.id);
    this.state.Set("todos", todos);
  };
}
```
