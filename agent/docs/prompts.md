# Prompts

## Default Prompts

Create `prompts.yml` in the assistant directory:

```yaml
- role: system
  content: |
    You are a helpful assistant.

    ## Guidelines
    - Be concise and accurate
    - Ask clarifying questions when needed

- role: system
  name: context
  content: |
    Current date: {{ $CTX.date }}
    User locale: {{ $CTX.locale }}
```

### Prompt Structure

```yaml
- role: system | user | assistant
  content: string
  name: string (optional)
```

### Context Variables

Use `$CTX.*` for runtime context:

| Variable         | Description                |
| ---------------- | -------------------------- |
| `$CTX.date`      | Current date               |
| `$CTX.time`      | Current time               |
| `$CTX.locale`    | User locale (e.g., en-us)  |
| `$CTX.timezone`  | User timezone              |
| `$CTX.user_id`   | Current user ID            |
| `$CTX.team_id`   | Current team ID            |
| `$CTX.chat_id`   | Current chat session ID    |

## Prompt Presets

Create presets in `prompts/` directory for different scenarios:

```
prompts/
├── chat.yml         # Casual conversation
├── task.yml         # Task-oriented
└── analysis.yml     # Data analysis
```

**prompts/chat.yml**

```yaml
- role: system
  content: |
    You are a friendly conversational assistant.
    Be warm and engaging.
```

**prompts/task.yml**

```yaml
- role: system
  content: |
    You are a task-focused assistant.
    Be precise and efficient.
```

### Using Presets

Select preset in Create hook:

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  return {
    messages,
    prompt_preset: "task", // Use prompts/task.yml
  };
}
```

Or via mode configuration in `package.yao`:

```json
{
  "modes": ["chat", "task"],
  "default_mode": "task"
}
```

## Global Prompts

Define global prompts in `agent/prompts.yml` (applies to all assistants):

```yaml
- role: system
  content: |
    # Global Guidelines
    - Always be helpful and respectful
    - Follow company policies
```

### Disabling Global Prompts

Per assistant:

```json
{
  "disable_global_prompts": true
}
```

Per request (in Create hook):

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  return {
    messages,
    disable_global_prompts: true,
  };
}
```

## Multi-line Content

Use YAML block scalars for long content:

```yaml
- role: system
  content: |
    # Assistant Role
    
    You are an expert in data analysis.
    
    ## Capabilities
    - Statistical analysis
    - Data visualization
    - Report generation
    
    ## Guidelines
    1. Always validate input data
    2. Explain your methodology
    3. Provide actionable insights
```

## Dynamic Prompts

Inject dynamic content in Create hook:

```typescript
function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const userPrefs = ctx.memory.user.Get("preferences");

  // Add dynamic system message
  const dynamicPrompt = {
    role: "system",
    content: `User preferences: ${JSON.stringify(userPrefs)}`,
  };

  return {
    messages: [dynamicPrompt, ...messages],
  };
}
```

## Prompt Best Practices

1. **Be specific** - Clear instructions produce better results
2. **Use structure** - Headers, lists, and sections improve readability
3. **Set boundaries** - Define what the assistant should and shouldn't do
4. **Include examples** - Show expected input/output formats
5. **Layer prompts** - Use global + assistant + dynamic prompts together
