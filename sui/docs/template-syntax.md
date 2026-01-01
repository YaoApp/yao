# Template Syntax

SUI uses a simple template syntax for data binding, conditional rendering, and list iteration.

## Data Interpolation

Use double curly braces `{{ }}` to output data:

```html
<!-- Variable binding -->
<span>{{ name }}</span>
<span>{{ user.email }}</span>
<span>{{ items[0].title }}</span>

<!-- Default values (null coalescing) -->
<span>{{ title ?? 'Default Title' }}</span>
<span>{{ user.name ?? 'Anonymous' }}</span>

<!-- Expressions -->
<span>{{ price * quantity }}</span>
<span>{{ firstName + ' ' + lastName }}</span>
<span>{{ count > 0 ? 'Has items' : 'Empty' }}</span>
```

## Conditional Rendering

### Basic If

```html
<div s:if="{{ isActive }}">Active</div>
<div s:if="{{ count > 0 }}">Has items</div>
<div s:if="{{ user != null }}">Logged in</div>
```

### If-Elif-Else

```html
<div s:if="{{ status == 'active' }}">Active</div>
<div s:elif="{{ status == 'pending' }}">Pending</div>
<div s:elif="{{ status == 'suspended' }}">Suspended</div>
<div s:else>Unknown</div>
```

### Comparison Operators

| Operator | Description           |
| -------- | --------------------- |
| `==`     | Equal                 |
| `!=`     | Not equal             |
| `>`      | Greater than          |
| `<`      | Less than             |
| `>=`     | Greater than or equal |
| `<=`     | Less than or equal    |
| `&&`     | Logical AND           |
| `\|\|`   | Logical OR            |
| `!`      | Logical NOT           |

### Examples

```html
<!-- Multiple conditions -->
<div s:if="{{ isAdmin && isActive }}">Admin Panel</div>
<div s:if="{{ age >= 18 || hasPermission }}">Access Granted</div>

<!-- Negation -->
<div s:if="{{ !isLoading }}">Content loaded</div>

<!-- Null checks -->
<div s:if="{{ user != null && user.verified }}">Verified User</div>
```

## List Rendering

### Basic Loop

```html
<ul>
  <li s:for="{{ items }}" s:for-item="item">{{ item.name }}</li>
</ul>
```

### With Index

```html
<ul>
  <li s:for="{{ items }}" s:for-item="item" s:for-index="index">
    {{ index + 1 }}. {{ item.name }}
  </li>
</ul>
```

### Nested Loops

```html
<div s:for="{{ categories }}" s:for-item="category">
  <h3>{{ category.name }}</h3>
  <ul>
    <li s:for="{{ category.items }}" s:for-item="item">{{ item.title }}</li>
  </ul>
</div>
```

### Loop with Conditional

```html
<div s:for="{{ users }}" s:for-item="user" s:if="{{ user.active }}">
  {{ user.name }}
</div>
```

### Object Iteration

```html
<dl s:for="{{ settings }}" s:for-item="value" s:for-index="key">
  <dt>{{ key }}</dt>
  <dd>{{ value }}</dd>
</dl>
```

## Variable Assignment

Use `<s:set>` to define variables:

```html
<!-- Simple assignment -->
<s:set name="total" value="{{ price * quantity }}" />
<span>Total: {{ total }}</span>

<!-- Computed values -->
<s:set name="fullName" value="{{ firstName + ' ' + lastName }}" />
<s:set name="isExpensive" value="{{ price > 100 }}" />

<!-- From expressions -->
<s:set name="discountedPrice" value="{{ price * (1 - discount / 100) }}" />
```

## Attribute Binding

### Dynamic Attributes

```html
<input value="{{ formData.email }}" />
<a href="{{ '/user/' + userId }}">Profile</a>
<img src="{{ imageUrl }}" alt="{{ imageAlt }}" />
```

### Conditional Attributes

```html
<!-- Attribute with condition -->
<button s:attr-disabled="{{ !isValid }}">Submit</button>
<input s:attr-readonly="{{ isLocked }}" />
<div s:attr-hidden="{{ !showPanel }}">Panel</div>

<!-- Class binding -->
<div class="base {{ isActive ? 'active' : '' }}">Content</div>
```

### Spread Attributes

```html
<!-- Spread object as attributes -->
<div ...props></div>
<input ...inputAttrs />
```

## Raw HTML Output

By default, output is HTML-escaped. Use `s:raw` for raw HTML:

```html
<!-- Escaped (safe) -->
<div>{{ htmlContent }}</div>

<!-- Raw HTML (use with caution) -->
<div s:raw="true">{{ htmlContent }}</div>
```

## Expression Engine

SUI uses [Expr](https://expr-lang.org/) (v1.17) as the expression engine. Expr provides a powerful expression language with operators, functions, and more.

### SUI Custom Functions

| Function        | Description                    | Example                           |
| --------------- | ------------------------------ | --------------------------------- |
| `P_(proc, ...)` | Call a Yao process             | `{{ P_('models.user.Find', 1) }}` |
| `True(value)`   | Check if value is truthy       | `{{ True(user) }}`                |
| `False(value)`  | Check if value is falsy        | `{{ False(error) }}`              |
| `Empty(value)`  | Check if array/object is empty | `{{ Empty(items) }}`              |

### Expr Built-in Functions

Expr provides many built-in functions. Here are commonly used ones:

| Function              | Description                    | Example                           |
| --------------------- | ------------------------------ | --------------------------------- |
| `len(array)`          | Get length of array/string/map | `{{ len(items) }}`                |
| `all(array, pred)`    | Check if all elements match    | `{{ all(users, .active) }}`       |
| `any(array, pred)`    | Check if any element matches   | `{{ any(items, .price > 100) }}`  |
| `one(array, pred)`    | Check if exactly one matches   | `{{ one(users, .admin) }}`        |
| `none(array, pred)`   | Check if no elements match     | `{{ none(items, .deleted) }}`     |
| `map(array, mapper)`  | Transform array elements       | `{{ map(users, .name) }}`         |
| `filter(array, pred)` | Filter array by predicate      | `{{ filter(items, .active) }}`    |
| `find(array, pred)`   | Find first matching element    | `{{ find(users, .id == 1) }}`     |
| `count(array, pred)`  | Count matching elements        | `{{ count(items, .price > 50) }}` |
| `sum(array)`          | Sum of array elements          | `{{ sum(prices) }}`               |
| `mean(array)`         | Average of array elements      | `{{ mean(scores) }}`              |
| `min(array)`          | Minimum value                  | `{{ min(prices) }}`               |
| `max(array)`          | Maximum value                  | `{{ max(scores) }}`               |
| `first(array)`        | First element                  | `{{ first(items) }}`              |
| `last(array)`         | Last element                   | `{{ last(items) }}`               |
| `take(array, n)`      | Take first n elements          | `{{ take(items, 5) }}`            |
| `keys(map)`           | Get map keys                   | `{{ keys(settings) }}`            |
| `values(map)`         | Get map values                 | `{{ values(settings) }}`          |
| `contains(a, b)`      | Check if a contains b          | `{{ contains(name, 'test') }}`    |
| `startsWith(s, pre)`  | Check string prefix            | `{{ startsWith(url, 'https') }}`  |
| `endsWith(s, suf)`    | Check string suffix            | `{{ endsWith(file, '.pdf') }}`    |
| `upper(s)`            | Uppercase string               | `{{ upper(name) }}`               |
| `lower(s)`            | Lowercase string               | `{{ lower(email) }}`              |
| `trim(s)`             | Trim whitespace                | `{{ trim(input) }}`               |
| `split(s, sep)`       | Split string                   | `{{ split(tags, ',') }}`          |
| `join(array, sep)`    | Join array to string           | `{{ join(names, ', ') }}`         |
| `int(v)`              | Convert to integer             | `{{ int(value) }}`                |
| `float(v)`            | Convert to float               | `{{ float(value) }}`              |
| `string(v)`           | Convert to string              | `{{ string(count) }}`             |
| `now()`               | Current time                   | `{{ now() }}`                     |
| `date(s)`             | Parse date string              | `{{ date('2024-01-01') }}`        |
| `duration(s)`         | Parse duration string          | `{{ duration('1h30m') }}`         |

For the complete list of built-in functions and operators, see the [Expr Language Definition](https://expr-lang.org/docs/language-definition).

### Examples

```html
<!-- SUI custom functions -->
<div s:if="{{ Empty(users) }}">No users found</div>
<span>{{ P_('utils.formatDate', createdAt) }}</span>

<!-- Array operations -->
<span>Total: {{ len(items) }} items</span>
<span>Active: {{ count(users, .active) }}</span>
<span>Sum: {{ sum(map(items, .price)) }}</span>

<!-- String operations -->
<span>{{ upper(first(split(name, ' '))) }}</span>

<!-- Filtering -->
<div s:for="{{ filter(items, .price > 100) }}" s:for-item="item">
  {{ item.name }}
</div>
```

## String Operations

```html
<!-- Concatenation -->
<span>{{ 'Hello, ' + name + '!' }}</span>

<!-- Template literals (in expressions) -->
<a href="{{ '/users/' + userId + '/edit' }}">Edit</a>
```

## Arithmetic Operations

```html
<!-- Basic math -->
<span>{{ price * quantity }}</span>
<span>{{ total / count }}</span>
<span>{{ value + 10 }}</span>
<span>{{ index - 1 }}</span>

<!-- Percentage -->
<span>{{ (completed / total) * 100 }}%</span>
```

## Comments

HTML comments are preserved in output:

```html
<!-- This comment appears in output -->
```

## Whitespace Control

SUI preserves whitespace by default. For minified output, use build options:

```bash
yao sui build <sui> <template>       # Minified (production)
yao sui build <sui> <template> -D    # Preserved (development, --debug)
```
