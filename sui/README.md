# SUI - Simple User Interface

SUI is a full-stack web development framework that allows you to create web applications using HTML, CSS, and TypeScript/JavaScript without complex build tools.

## Features

- **Page as Component**: Every page is a component, unifying the development model
- **Template Syntax**: Intuitive data binding, conditionals, and loops
- **Backend Scripts**: Server-side logic with TypeScript
- **Scoped Styles**: Automatic CSS scoping per component
- **i18n Support**: Built-in internationalization
- **Agent SUI**: Special configuration for AI Agent applications

## Quick Start

### Directory Structure

```
/templates/<template_name>/
├── __document.html         # Global document template
├── __data.json             # Global data (accessible via $global)
├── __assets/               # Static assets (reference via @assets/)
├── __locales/              # Locale files
└── pages/                  # All pages go here
    └── <page>/             # Route = folder name (can be nested)
        ├── <page>.html     # HTML template (filename must match folder)
        ├── <page>.css      # Styles
        ├── <page>.ts       # Frontend script
        ├── <page>.json     # Data configuration
        ├── <page>.config   # Page configuration
        ├── <page>.backend.ts  # Backend script
        └── __locales/      # Page-level locale files
```

### Basic Page

**`/home/home.html`**:

```html
<div class="home">
  <h1>{{ title }}</h1>
  <p s:if="{{ showMessage }}">{{ message }}</p>
</div>
```

**`/home/home.json`**:

```json
{
  "title": "Welcome",
  "showMessage": true,
  "message": "Hello, World!"
}
```

### Commands

```bash
# Build templates
yao sui build <sui> [template]

# Watch for changes
yao sui watch <sui> [template]

# Build Agent SUI
yao sui build agent

# Watch Agent SUI
yao sui watch agent
```

## Documentation

- [Template Syntax](docs/template-syntax.md) - Data binding, conditionals, loops
- [Components](docs/components.md) - Page as component, props, slots
- [Backend Scripts](docs/backend-scripts.md) - Server-side logic
- [Data Binding](docs/data-binding.md) - Built-in variables and functions
- [Event Handling](docs/event-handling.md) - Event binding and state management
- [Internationalization](docs/i18n.md) - Translation and localization
- [Frontend API](docs/frontend-api.md) - Component query, backend calls, render API, CUI integration
- [Agent SUI](docs/agent-sui.md) - AI Agent application setup

## Agent SUI

Agent SUI is designed for AI Agent applications with automatic page loading from assistants:

```
<app>/
├── agent/
│   └── template/              # Agent SUI template (shared)
│       ├── __document.html
│       ├── __data.json
│       ├── __assets/
│       └── pages/             # Global pages (401, 404, etc.)
│           └── <page>/
└── assistants/
    └── <name>/
        └── pages/             # Assistant pages → /agents/<name>/<route>
            └── <page>/
```

Build with: `yao sui build agent`

See [Agent SUI Documentation](docs/agent-sui.md) for details.

## License

This project is part of the Yao App Engine and follows the [Yao Open Source License](../LICENSE).
