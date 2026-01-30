# Agent Sandbox Design

Inject Coding Agent capabilities (Claude CLI, Cursor CLI) into Yao's LLM request pipeline via Docker-based sandbox execution.

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Yao LLM Pipeline                             │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Before Hooks / Auth / Logging                │  │
│  └─────────────────────────┬─────────────────────────────────┘  │
│                            │                                    │
│                            ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                  LLM Request Handler                      │  │
│  │                                                           │  │
│  │   sandbox: nil     → Direct LLM API call (default)       │  │
│  │   sandbox: config  → Sandbox + Claude CLI                │  │ ← Inject here
│  │   sandbox: config  → Sandbox + Cursor CLI (future)       │  │
│  │                                                           │  │
│  └─────────────────────────┬─────────────────────────────────┘  │
│                            │                                    │
│                            ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              After Hooks / Billing / Audit                │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## 2. Design Benefits

| Aspect        | Standalone Executor      | Pipeline Injection      |
| ------------- | ------------------------ | ----------------------- |
| **Hooks**     | Need re-implementation   | ✅ Fully reused         |
| **Auth**      | Need re-implementation   | ✅ Fully reused         |
| **Logging**   | Need re-implementation   | ✅ Fully reused         |
| **Billing**   | Need re-implementation   | ✅ Fully reused         |
| **Errors**    | Need re-implementation   | ✅ Fully reused         |
| **Code size** | Large (full executor)    | Small (one branch)      |
| **Extend**    | One executor per agent   | Just add agent type     |

## 3. Configuration

### 3.1 Assistant Configuration

Sandbox is configured at the **Assistant level** (not Connector), because:

- Same Connector can be shared by multiple Assistants
- Some Assistants need Sandbox (Coding), others don't (Q&A)
- Assistant defines "behavior", Connector defines "connection"

```jsonc
// assistants/coder/package.yao
{
  "name": "Coder Assistant",
  "connector": "deepseek.v3",

  // Sandbox configuration
  "sandbox": {
    "command": "claude",                         // claude | cursor (future)
    "image": "yaoapp/sandbox-claude:latest",     // Optional, auto-selected by agent
    "max_memory": "4g",                          // Optional
    "max_cpu": 2.0,                              // Optional
    "timeout": "10m",                            // Optional, execution timeout
    "arguments": {
      // Command-specific arguments (passed to Claude CLI)
      "max_turns": 20,
      "permission_mode": "acceptEdits"
      // Different commands may have different arguments
    }
  },

  // MCP servers
  "mcp": {
    "servers": [
      {
        "server_id": "filesystem",
        "tools": ["read_file", "write_file", "list_directory"]
      }
    ]
  }
}
```

### 3.2 Directory Structure

Skills are auto-discovered from `skills/` directory following the [Agent Skills](https://agentskills.io) open standard:

```
assistants/coder/
├── package.yao       # Assistant config
├── prompts.yml       # Prompts
├── mcps/             # MCP server definitions
└── skills/           # Skills directory (auto-discovered)
    ├── code-review/
    │   ├── SKILL.md       # Required: instructions + metadata
    │   ├── scripts/       # Optional: executable code (Python, Bash, JS)
    │   ├── references/    # Optional: additional documentation
    │   └── assets/        # Optional: templates, images, data files
    └── deploy/
        ├── SKILL.md
        └── scripts/
            └── deploy.sh
```

### 3.3 SKILL.md Format

Each skill must have a `SKILL.md` file with YAML frontmatter:

```yaml
---
name: code-review                    # Required: must match parent directory name
description: >                       # Required: when to use this skill
  Review code for bugs, security issues, and best practices.
  Use when the user asks to review, audit, or analyze code quality.
license: Apache-2.0                  # Optional
compatibility: Requires git          # Optional: environment requirements
metadata:                            # Optional: arbitrary key-value pairs
  author: yao-team
  version: "1.0"
allowed-tools: Bash(git:*) Read      # Optional: pre-approved tools (experimental)
---

# Code Review

## When to use this skill
Use this skill when the user asks to review code...

## Steps
1. Check for security vulnerabilities
2. Review code style and best practices
3. Identify potential bugs
...
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | 1-64 chars, lowercase + hyphens, must match directory name |
| `description` | Yes | 1-1024 chars, describes what skill does and when to use it |
| `license` | No | License name or reference to bundled LICENSE file |
| `compatibility` | No | Environment requirements (tools, network access, etc.) |
| `metadata` | No | Arbitrary key-value pairs for additional properties |
| `allowed-tools` | No | Space-delimited list of pre-approved tools (experimental) |

### 3.4 Skills Progressive Disclosure

Skills use progressive disclosure to manage context efficiently:

1. **Discovery**: At startup, agent loads only `name` and `description` of each skill
2. **Activation**: When task matches a skill's description, agent reads full `SKILL.md`
3. **Execution**: Agent follows instructions, loading `scripts/`, `references/`, `assets/` as needed

### 3.5 No Sandbox (Default)

If `sandbox` is not configured, Assistant uses direct LLM API calls:

```jsonc
// assistants/qa/package.yao
{
  "name": "QA Assistant",
  "connector": "deepseek.v3"
  // No sandbox config → direct API call
}
```

## 4. Implementation

### 4.1 Package Structure

```
yao/
├── sandbox/                    # Low-level sandbox infrastructure
│   ├── manager.go              # Container management (✅ Done)
│   ├── config.go               # Configuration (✅ Done)
│   ├── ipc/                    # IPC communication (✅ Done)
│   │   ├── manager.go
│   │   ├── session.go
│   │   └── types.go
│   ├── bridge/                 # yao-bridge binary (✅ Done)
│   │   └── main.go
│   └── docker/                 # Docker images (✅ Done)
│       ├── base/
│       ├── claude/
│       └── build.sh
│
└── agent/
    └── sandbox/                # Agent-level sandbox integration (NEW)
        ├── DESIGN.md           # This document
        ├── types.go            # Common types and interfaces
        ├── executor.go         # Factory function and registry
        ├── claude/             # Claude CLI agent
        │   ├── types.go        # Claude-specific types (StreamMessage, ToolCall, etc.)
        │   ├── executor.go     # Executor implementation
        │   ├── command.go      # CLI command builder
        │   ├── stream.go       # Stream output parser
        │   ├── environment.go  # Container environment setup
        │   └── executor_test.go
        ├── cursor/             # Cursor CLI agent (future)
        │   └── README.md       # Placeholder for future implementation
        └── sandbox_test.go     # Integration tests
```

### 4.2 Types and Interfaces

```go
// agent/sandbox/types.go
package sandbox

import (
    "time"

    "github.com/yaoapp/yao/agent/context"
    "github.com/yaoapp/yao/agent/output/message"
)

// Executor executes LLM requests in sandbox
type Executor interface {
    // Execute runs the request and returns response
    Execute(ctx *context.Context, messages []context.Message, opts *Options) (*context.CompletionResponse, error)

    // Stream runs the request with streaming output
    Stream(ctx *context.Context, messages []context.Message, opts *Options, handler message.StreamFunc) (*context.CompletionResponse, error)

    // Filesystem operations (for Hooks)
    ReadFile(ctx context.Context, path string) ([]byte, error)
    WriteFile(ctx context.Context, path string, content []byte) error
    ListDir(ctx context.Context, path string) ([]os.FileInfo, error)

    // Command execution (for Hooks)
    Exec(ctx context.Context, cmd []string) (string, error)

    // Close releases container resources
    Close() error
}

// Options for sandbox execution
type Options struct {
    // Command type (claude, cursor)
    Command string `json:"command"`

    // Docker image (optional, auto-selected by agent)
    Image string `json:"image,omitempty"`

    // Resource limits
    MaxMemory string  `json:"max_memory,omitempty"`
    MaxCPU    float64 `json:"max_cpu,omitempty"`

    // Execution timeout
    Timeout time.Duration `json:"timeout,omitempty"`

    // Command-specific arguments (passed to CLI)
    Arguments map[string]interface{} `json:"arguments,omitempty"`

    // ========================================
    // Internal fields (auto-resolved by Yao)
    // Do NOT set these in package.yao config
    // ========================================

    // MCP configuration - auto-loaded from assistants/{name}/mcps/
    MCPConfig []byte `json:"-"`

    // Skills directory - auto-resolved to assistants/{name}/skills/
    SkillsDir string `json:"-"`

    // Connector settings - auto-resolved from connector config file
    // e.g., connectors/deepseek/v3.conn.yao → host, key, model
    ConnectorHost string `json:"-"`
    ConnectorKey  string `json:"-"`
    Model         string `json:"-"`
}
```

### 4.3 Executor Factory

```go
// agent/sandbox/executor.go
package sandbox

import (
    "fmt"

    "github.com/yaoapp/yao/agent/sandbox/claude"
    "github.com/yaoapp/yao/agent/sandbox/cursor"
    "github.com/yaoapp/yao/sandbox"
)

// New creates an executor based on agent type
func New(manager *sandbox.Manager, opts *Options) (Executor, error) {
    if opts == nil {
        return nil, fmt.Errorf("options cannot be nil")
    }

    // Set default image if not specified
    if opts.Image == "" {
        opts.Image = DefaultImage(opts.Command)
    }

    switch opts.Command {
    case "claude":
        return claude.New(manager, opts)
    case "cursor":
        return cursor.New(manager, opts)
    default:
        return nil, fmt.Errorf("unknown command type: %s", opts.Command)
    }
}

// DefaultImage returns the default Docker image for a command type
func DefaultImage(command string) string {
    switch command {
    case "claude":
        return "yaoapp/sandbox-claude:latest"
    case "cursor":
        return "yaoapp/sandbox-cursor:latest"
    default:
        return ""
    }
}
```

### 4.4 Claude Agent Implementation

The Claude agent is split into multiple files for maintainability:

#### 4.4.1 Executor (claude/executor.go)

```go
// agent/sandbox/claude/executor.go
package claude

import (
    "fmt"

    "github.com/yaoapp/yao/agent/context"
    "github.com/yaoapp/yao/agent/output/message"
    sandbox "github.com/yaoapp/yao/agent/sandbox"
    infra "github.com/yaoapp/yao/sandbox"
)

// Executor executes requests via Claude CLI in sandbox
type Executor struct {
    manager *infra.Manager
    opts    *sandbox.Options
}

// New creates a new Claude executor
func New(manager *infra.Manager, opts *sandbox.Options) (*Executor, error) {
    return &Executor{
        manager: manager,
        opts:    opts,
    }, nil
}

// Stream executes with streaming output
func (e *Executor) Stream(
    ctx *context.Context,
    messages []context.Message,
    opts *sandbox.Options,
    handler message.StreamFunc,
) (*context.CompletionResponse, error) {

    // 1. Get or create container
    container, err := e.manager.GetOrCreate(ctx.Context(), ctx.Authorized.UserID, ctx.ChatID)
    if err != nil {
        return nil, fmt.Errorf("failed to get container: %w", err)
    }

    // 2. Prepare environment
    workDir := fmt.Sprintf("/workspace/%s/chat-%s", ctx.Authorized.UserID, ctx.ChatID)
    if err := e.prepareEnvironment(ctx, container, workDir); err != nil {
        return nil, fmt.Errorf("failed to prepare environment: %w", err)
    }

    // 3. Build CLI command
    cmd, env := BuildCommand(messages, opts, workDir)

    // 4. Execute in container with streaming
    reader, err := e.manager.Stream(ctx.Context(), container.Name, cmd, &infra.ExecOptions{
        Env:        env,
        WorkingDir: workDir,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to execute: %w", err)
    }
    defer reader.Close()

    // 5. Parse streaming output
    return ParseStream(reader, handler)
}

// Execute runs without streaming (wraps Stream)
func (e *Executor) Execute(ctx *context.Context, messages []context.Message, opts *sandbox.Options) (*context.CompletionResponse, error) {
    var result *context.CompletionResponse
    _, err := e.Stream(ctx, messages, opts, func(msg *message.Message) error {
        // Collect final result
        return nil
    })
    return result, err
}

// Close releases resources
func (e *Executor) Close() error {
    return nil
}
```

#### 4.4.2 Environment Setup (claude/environment.go)

```go
// agent/sandbox/claude/environment.go
package claude

import (
    "path/filepath"

    "github.com/yaoapp/yao/agent/context"
    sandbox "github.com/yaoapp/yao/agent/sandbox"
    infra "github.com/yaoapp/yao/sandbox"
)

// prepareEnvironment sets up the container environment
func (e *Executor) prepareEnvironment(ctx *context.Context, container *infra.Container, workDir string) error {
    // Create work directory
    if err := e.manager.MkDir(ctx.Context(), container.Name, workDir); err != nil {
        return err
    }

    // Write MCP config
    if len(e.opts.MCPConfig) > 0 {
        mcpPath := filepath.Join(workDir, ".mcp.json")
        if err := e.manager.WriteFile(ctx.Context(), container.Name, mcpPath, e.opts.MCPConfig); err != nil {
            return err
        }
    }

    // Copy skills if configured
    if e.opts.SkillsDir != "" {
        claudeDir := filepath.Join(workDir, ".claude")
        if err := e.manager.MkDir(ctx.Context(), container.Name, claudeDir); err != nil {
            return err
        }
        targetSkillsDir := filepath.Join(claudeDir, "skills")
        if err := e.manager.CopyToContainer(ctx.Context(), container.Name, e.opts.SkillsDir, targetSkillsDir); err != nil {
            return err
        }
    }

    return nil
}
```

#### 4.4.3 Command Builder (claude/command.go)

```go
// agent/sandbox/claude/command.go
package claude

import (
    "fmt"
    "strconv"
    "strings"

    "github.com/yaoapp/yao/agent/context"
    sandbox "github.com/yaoapp/yao/agent/sandbox"
)

// BuildCommand constructs the Claude CLI command
func BuildCommand(messages []context.Message, opts *sandbox.Options, workDir string) ([]string, map[string]string) {
    cmd := []string{
        "claude",
        "--print",
        "--output-format", "stream-json",
    }

    // Model
    if opts.Model != "" {
        cmd = append(cmd, "--model", opts.Model)
    }

    // Agent-specific options from sandbox.options
    if opts.Arguments != nil {
        if maxTurns, ok := opts.Arguments["max_turns"].(int); ok && maxTurns > 0 {
            cmd = append(cmd, "--max-turns", strconv.Itoa(maxTurns))
        }
        if permMode, ok := opts.Arguments["permission_mode"].(string); ok && permMode != "" {
            cmd = append(cmd, "--permission-mode", permMode)
        }
    }

    // MCP config
    cmd = append(cmd, "--mcp-config", ".mcp.json")

    // System prompt (with history)
    systemPrompt := buildSystemPrompt(messages)
    if systemPrompt != "" {
        cmd = append(cmd, "--system-prompt", systemPrompt)
    }

    // User prompt (last user message)
    prompt := extractUserPrompt(messages)
    cmd = append(cmd, prompt)

    // Environment variables
    env := map[string]string{}
    if opts.ConnectorHost != "" {
        env["ANTHROPIC_BASE_URL"] = opts.ConnectorHost
    }
    if opts.ConnectorKey != "" {
        env["ANTHROPIC_API_KEY"] = opts.ConnectorKey
    }

    return cmd, env
}

// buildSystemPrompt builds system prompt with conversation history
func buildSystemPrompt(messages []context.Message) string {
    var systemParts []string
    var history []string

    for i, msg := range messages {
        if msg.Role == "system" {
            if content, ok := msg.Content.(string); ok {
                systemParts = append(systemParts, content)
            }
            continue
        }

        // Skip last user message (it becomes the prompt)
        if i == len(messages)-1 && msg.Role == "user" {
            continue
        }

        // Add to history
        if content, ok := msg.Content.(string); ok {
            history = append(history, fmt.Sprintf("[%s]: %s", msg.Role, content))
        }
    }

    if len(history) > 0 {
        systemParts = append(systemParts,
            "\n## Conversation History:\n"+strings.Join(history, "\n\n"))
    }

    return strings.Join(systemParts, "\n")
}

// extractUserPrompt gets the last user message
func extractUserPrompt(messages []context.Message) string {
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].Role == "user" {
            if content, ok := messages[i].Content.(string); ok {
                return content
            }
        }
    }
    return ""
}
```

#### 4.4.4 Types (claude/types.go)

```go
// agent/sandbox/claude/types.go
package claude

// StreamMessage represents a Claude CLI stream-json message
type StreamMessage struct {
    Type    string `json:"type"`
    Message struct {
        Content []struct {
            Type string `json:"type"`
            Text string `json:"text"`
        } `json:"content"`
    } `json:"message,omitempty"`
    Result       string  `json:"result,omitempty"`
    TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
    NumTurns     int     `json:"num_turns,omitempty"`
    DurationMs   int64   `json:"duration_ms,omitempty"`
    Usage        *struct {
        InputTokens  int `json:"input_tokens"`
        OutputTokens int `json:"output_tokens"`
    } `json:"usage,omitempty"`
}

// ToolCall represents a tool invocation from Claude CLI
type ToolCall struct {
    ID        string                 `json:"id"`
    Name      string                 `json:"name"`
    Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents a tool execution result
type ToolResult struct {
    ID      string `json:"id"`
    Content string `json:"content"`
    IsError bool   `json:"is_error,omitempty"`
}
```

#### 4.4.5 Stream Parser (claude/stream.go)

```go
// agent/sandbox/claude/stream.go
package claude

import (
    "bufio"
    "encoding/json"
    "io"
    "strings"

    "github.com/yaoapp/yao/agent/context"
    "github.com/yaoapp/yao/agent/output/message"
)

// ParseStream parses Claude CLI stream-json output
func ParseStream(reader io.ReadCloser, handler message.StreamFunc) (*context.CompletionResponse, error) {
    resp := &context.CompletionResponse{}
    var fullContent strings.Builder

    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
        line := scanner.Text()
        if line == "" {
            continue
        }

        var msg StreamMessage
        if err := json.Unmarshal([]byte(line), &msg); err != nil {
            continue
        }

        switch msg.Type {
        case "assistant":
            // Extract text content
            for _, content := range msg.Message.Content {
                if content.Type == "text" && content.Text != "" {
                    fullContent.WriteString(content.Text)

                    // Send to stream handler
                    if handler != nil {
                        handler(&message.Message{
                            Type: "text",
                            Data: content.Text,
                        })
                    }
                }
            }

        case "result":
            // Final result
            if msg.Result != "" {
                resp.Content = msg.Result
            }
            if msg.Usage != nil {
                resp.Usage = &context.Usage{
                    PromptTokens:     msg.Usage.InputTokens,
                    CompletionTokens: msg.Usage.OutputTokens,
                    TotalTokens:      msg.Usage.InputTokens + msg.Usage.OutputTokens,
                }
            }
            resp.Extra = map[string]interface{}{
                "total_cost_usd": msg.TotalCostUSD,
                "num_turns":      msg.NumTurns,
                "duration_ms":    msg.DurationMs,
            }
        }
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    if resp.Content == "" {
        resp.Content = fullContent.String()
    }

    return resp, nil
}
```

### 4.5 Sandbox Lifecycle

#### 4.5.1 Design Principles

The sandbox follows a **stateless container + persistent workspace** model:

| Component | Lifecycle | Storage |
|-----------|-----------|---------|
| **Container** | Per-request, disposable | None (stateless) |
| **Workspace** | Persistent across requests | `{YAO_DATA_ROOT}/sandbox/workspace/{user}/chat-{chat_id}/` |
| **Message History** | Managed by Yao | Yao's session store |

This means:
- Container can be recreated anytime without losing state
- All files are preserved in the mounted workspace
- Conversation history is passed in each request (not stored in container)

#### 4.5.2 Container Lifecycle

```
Request Start
     │
     ▼
┌─────────────────────────────────────────────────────────────┐
│  1. Create Executor (docker run --rm)                       │
│     - Mount workspace directory                             │
│     - Set resource limits                                   │
│     - ~500ms-1s cold start                                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  2. Before Hook (can access Executor)                       │
│     - Write config files                                    │
│     - Check/prepare environment                             │
│     - Can reject request (container auto-cleaned)           │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  3. LLM Execute (Claude CLI in container)                   │
│     - Run with full message history                         │
│     - Stream output to handler                              │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  4. After Hook (can access Executor)                        │
│     - Read generated files                                  │
│     - Execute post-commands (git commit, etc.)              │
│     - Cleanup temp files                                    │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  5. Close Executor (defer)                                  │
│     - Container removed (--rm)                              │
│     - Workspace persists                                    │
└─────────────────────────────────────────────────────────────┘
```

#### 4.5.3 Workspace Cleanup

Workspace cleanup is separate from container lifecycle:

```go
// Global cleanup configuration
type CleanupConfig struct {
    // Workspace retention period (default: 7 days)
    WorkspaceRetention time.Duration `json:"workspace_retention"`
    
    // Run cleanup on schedule (default: daily at 3am)
    CleanupSchedule string `json:"cleanup_schedule"`
}

// Cleanup worker
func (m *Manager) cleanupStaleWorkspaces() {
    // Find workspaces older than retention period
    // Delete directories not accessed within the period
}
```

### 4.6 Hook Integration (via JSAPI)

Hooks interact with the sandbox via **Context JSAPI**, not Go code. The Executor methods are exposed to JavaScript through `ctx.sandbox`:

#### 4.6.1 Context JSAPI Extension

```typescript
// Extension to Context interface (see agent/context/JSAPI.md)
interface Context {
  // ... existing properties ...
  
  // Sandbox operations (only available when sandbox is configured)
  sandbox?: {
    // Filesystem operations
    ReadFile(path: string): string;              // Read file content
    WriteFile(path: string, content: string): void;  // Write file
    ListDir(path: string): FileInfo[];           // List directory
    
    // Command execution
    Exec(cmd: string[]): string;                 // Execute command, return output
    
    // Workspace info
    workdir: string;                             // Container workspace path
  };
}

interface FileInfo {
  name: string;
  size: number;
  is_dir: boolean;
  mod_time: string;
}
```

#### 4.6.2 Create Hook Examples (JavaScript)

```javascript
// assistants/coder/src/index.ts

/**
 * Create Hook - runs after container created, before LLM execution
 */
function Create(ctx, messages, options) {
  // Check if sandbox is available
  if (!ctx.sandbox) {
    return { messages };
  }

  // Send loading message to user
  const loadingId = ctx.SendStream({
    type: "loading",
    props: { message: "Preparing sandbox environment..." }
  });

  // Write project configuration
  ctx.sandbox.WriteFile(".env", "DEBUG=true\nNODE_ENV=development");
  
  // Check environment prerequisites
  try {
    const nodeVersion = ctx.sandbox.Exec(["node", "--version"]);
    log.Info(`Node.js version: ${nodeVersion}`);
  } catch (e) {
    log.Error("Node.js not available");
    ctx.End(loadingId);
    throw new Error("Node.js is required");
  }
  
  // List existing files
  const files = ctx.sandbox.ListDir(".");
  log.Debug(`Workspace files: ${files.map(f => f.name).join(", ")}`);
  
  // Update loading message and end it
  ctx.Replace(loadingId, {
    type: "text",
    props: { content: "Sandbox ready" }
  });
  ctx.End(loadingId);
  
  return { messages };
}
```

#### 4.6.3 Next Hook Examples (JavaScript)

```javascript
/**
 * Next Hook - runs after LLM execution, before container cleanup
 */
function Next(ctx, payload, options) {
  const { completion, error } = payload;
  
  if (error || !ctx.sandbox) {
    return null;
  }

  // Read generated files
  try {
    const result = ctx.sandbox.ReadFile("output/result.json");
    log.Info(`Generated output: ${result}`);
    ctx.memory.context.Set("generated_result", JSON.parse(result));
  } catch (e) {
    log.Debug("No result file generated");
  }
  
  // List generated source files
  const srcFiles = ctx.sandbox.ListDir("src/");
  for (const file of srcFiles) {
    log.Debug(`Generated: ${file.name} (${file.size} bytes)`);
  }
  
  // Execute git commit if files changed
  try {
    ctx.sandbox.Exec(["git", "add", "."]);
    ctx.sandbox.Exec(["git", "commit", "-m", "auto-commit by sandbox"]);
    log.Info("Changes committed");
  } catch (e) {
    log.Debug(`git commit skipped: ${e.message}`);
  }
  
  // Cleanup temporary files
  try {
    ctx.sandbox.Exec(["rm", "-rf", "tmp/"]);
  } catch (e) {
    // Ignore cleanup errors
  }
  
  return null;  // Use default response
}
```

#### 4.6.4 Go Implementation (Internal)

The Go layer only handles exposing the Executor to JSAPI:

```go
// In agent/context/sandbox.go - expose Executor to JS runtime

// SandboxJSAPI wraps Executor for JavaScript access
type SandboxJSAPI struct {
    executor sandbox.Executor
    workdir  string
}

// Methods are called from JavaScript via v8go bindings
func (s *SandboxJSAPI) ReadFile(path string) (string, error) {
    content, err := s.executor.ReadFile(context.Background(), path)
    return string(content), err
}

func (s *SandboxJSAPI) WriteFile(path string, content string) error {
    return s.executor.WriteFile(context.Background(), path, []byte(content))
}

func (s *SandboxJSAPI) ListDir(path string) ([]FileInfo, error) {
    return s.executor.ListDir(context.Background(), path)
}

func (s *SandboxJSAPI) Exec(cmd []string) (string, error) {
    return s.executor.Exec(context.Background(), cmd)
}
```

### 4.7 Integration with Assistant

The sandbox integration is separated into two files:

- `agent/assistant/sandbox.go` - Sandbox handler (new file, contains all sandbox logic)
- `agent/assistant/llm.go` - Add sandbox detection (minimal change)

#### 4.7.1 Sandbox Detection (`agent/assistant/llm.go`)

```go
// In agent/assistant/llm.go - add sandbox detection

func (ast *Assistant) executeLLMStream(...) (*context.CompletionResponse, error) {
    // Check if sandbox is configured
    if ast.Sandbox != nil && ast.Sandbox.Command != "" {
        return ast.executeSandboxStream(ctx, completionMessages, completionOptions, agentNode, streamHandler, opts)
    }

    // Default: direct LLM API call
    // ... existing code ...
}
```

#### 4.7.2 Sandbox Handler (`agent/assistant/sandbox.go`)

```go
// In agent/assistant/sandbox.go - sandbox execution logic

func (ast *Assistant) executeSandboxStream(...) (*context.CompletionResponse, error) {
    // Get sandbox manager (singleton)
    manager := sandbox.GetManager()
    if manager == nil {
        return nil, fmt.Errorf("sandbox manager not initialized")
    }

    // Build executor options
    execOpts := &agentsandbox.Options{
        Command:      ast.Sandbox.Command,
        Image:        ast.Sandbox.Image,
        MaxMemory:    ast.Sandbox.MaxMemory,
        MaxCPU:       ast.Sandbox.MaxCPU,
        Timeout:      ast.Sandbox.Timeout,
        Arguments:    ast.Sandbox.Arguments,
        SkillsDir:    filepath.Join(ast.Path, "skills"),
    }

    // Get connector settings (resolved from connector config)
    conn, _, err := ast.GetConnector(ctx, opts)
    if err != nil {
        return nil, err
    }
    setting := conn.Setting()
    if host, ok := setting["host"].(string); ok {
        execOpts.ConnectorHost = host
    }
    if key, ok := setting["key"].(string); ok {
        execOpts.ConnectorKey = key
    }
    if model, ok := setting["model"].(string); ok {
        execOpts.Model = model
    }

    // Build MCP config
    execOpts.MCPConfig = ast.buildMCPConfig(ctx)

    // 1. Create executor (container starts here)
    ctx.Trace().Info("Creating sandbox container...")
    executor, err := agentsandbox.New(manager, execOpts)
    if err != nil {
        ctx.Trace().Error(fmt.Sprintf("Sandbox creation failed: %v", err))
        return nil, err
    }
    ctx.Trace().Info("Sandbox container ready")
    defer executor.Close()  // 5. Container cleanup on exit

    // 2. Before Hook - can access executor
    if err := ast.runBeforeHook(ctx, executor); err != nil {
        return nil, err  // Container cleaned by defer
    }

    // 3. LLM Execute
    resp, err := executor.Stream(ctx, completionMessages, execOpts, streamHandler)
    if err != nil {
        return nil, err
    }

    // 4. After Hook - can access executor
    if err := ast.runAfterHook(ctx, executor, resp); err != nil {
        log.Printf("after hook error: %v", err)  // Log but don't fail
    }

    return resp, nil
}
```

## 5. IPC Communication (Yao ↔ Sandbox)

The sandbox can call Yao processes via MCP:

```
┌─────────────────────────────────────────────────────────────────┐
│  Sandbox Container                                              │
│  ┌─────────────────┐    ┌──────────────────┐                   │
│  │   Claude CLI    │───▶│    yao-bridge    │                   │
│  │                 │    │                  │                   │
│  │  MCP: tools/call│    │  Unix Socket     │                   │
│  │  "yao.process"  │    │  /tmp/yao.sock   │                   │
│  └─────────────────┘    └────────┬─────────┘                   │
└──────────────────────────────────┼─────────────────────────────┘
                                   │ JSON-RPC
                                   ▼
┌──────────────────────────────────────────────────────────────────┐
│  Yao Host                                                        │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    IPC Manager                              │ │
│  │                                                             │ │
│  │  Session: user123-chat456                                   │ │
│  │  Socket: /data/sandbox/ipc/user123-chat456.sock             │ │
│  │                                                             │ │
│  │  Handlers:                                                  │ │
│  │    tools/list  → Return authorized tool list                │ │
│  │    tools/call  → Execute Yao Process                        │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

## 6. Skills Support

Skills follow [Agent Skills](https://agentskills.io) standard:

```markdown
---
name: code-review
description: "Review code for security, performance, and best practices"
allowed-tools:
  - Read
  - Grep
  - Glob
---

# Code Review

When reviewing code:

## Security
- Check for SQL injection, XSS, CSRF
- Verify input validation

## Performance
- Look for N+1 queries
- Check for unnecessary re-renders
```

Skills are auto-copied to container at `{workDir}/.claude/skills/`.

## 7. Docker Images

### Available Images

| Image | Contents | Size |
|-------|----------|------|
| `yaoapp/sandbox-base` | Ubuntu 24.04, git, vim, network tools, yao-bridge | ~200MB |
| `yaoapp/sandbox-claude` | base + Node.js 22, Python 3.12, Claude CLI, CCR | ~900MB |
| `yaoapp/sandbox-claude-full` | claude + Go 1.23 | ~1.2GB |

### Included Tools

- **Editors**: vim, less, tree
- **Network**: ping, netstat, nslookup, nc, telnet
- **Compression**: zip, unzip, tar, gzip
- **System**: htop, ps, sed, awk, grep
- **Development**: git, Node.js, Python, Claude CLI, CCR

## 8. Implementation Status

See [PLAN.md](./PLAN.md) for detailed implementation tasks and testing requirements.

### Completed (sandbox package)

- [x] Container management (`sandbox/manager.go`)
- [x] Configuration (`sandbox/config.go`)
- [x] IPC communication (`sandbox/ipc/`)
- [x] yao-bridge binary (`sandbox/bridge/`)
- [x] Docker images (`sandbox/docker/`)

### To Implement (agent/sandbox package)

- [ ] Types and interfaces
- [ ] Executor factory
- [ ] Claude executor (claude/)
- [ ] Context JSAPI bindings
- [ ] Assistant integration
- [ ] Workspace cleanup

## 9. Testing

See [PLAN.md](./PLAN.md) for complete testing strategy.

**Quick test:**

```bash
# Test Docker image
docker run --rm yaoapp/sandbox-claude:latest bash -c "
  node --version && python3 --version && claude --version
"

# Run integration tests
source /Users/max/Yao/yao/env.local.sh
go test -v ./agent/sandbox/...
```

## 10. Usage Example

```bash
# Test script at yao-dev-app/agents/claude/run.sh
./run.sh "Hello, what is 1+1?"                    # Default: deepseek/v3
./run.sh -c deepseek/r1 "Hello"                   # Use R1 model
./run.sh -c claude/sonnet-4_0 "Hello"             # Use Claude
```
