# Claude API Proxy

A lightweight API proxy that allows Claude CLI to use any OpenAI-compatible backend API (Volcengine, DeepSeek, GLM, etc.).

## Features

- **Zero dependencies**: Uses only Go standard library
- **Lightweight**: Single executable binary
- **True streaming**: Direct SSE forwarding, no buffering
- **Full tool calling support**: Both streaming and non-streaming
- **Image content support**: Base64 and URL formats
- **Multi-architecture**: Supports amd64 and arm64

## Architecture

```
Claude CLI (Anthropic Messages API)
    │
    ▼
claude-proxy (localhost:3456)
    │
    │ Convert: Anthropic → OpenAI
    ▼
OpenAI-compatible Backend (Volcengine/DeepSeek/GLM...)
    │
    │ Convert: OpenAI → Anthropic
    ▼
Claude CLI (Real-time streaming output)
```

## Command Line Options

```bash
claude-proxy [options]

Options:
  -p, --port <port>         Listen port (default: 3456)
  -b, --backend <url>       Backend API URL (required)
  -m, --model <model>       Backend model name (required)
  -k, --api-key <key>       Backend API key (required)
  -l, --log <path>          Log file path
  -t, --timeout <seconds>   Request timeout (default: 300)
  -v, --verbose             Verbose logging
  -h, --help                Show help

Environment Variables:
  CLAUDE_PROXY_PORT         Listen port
  CLAUDE_PROXY_BACKEND      Backend API URL
  CLAUDE_PROXY_MODEL        Backend model name
  CLAUDE_PROXY_API_KEY      Backend API key
  CLAUDE_PROXY_TIMEOUT      Timeout in seconds
```

## Usage

### Method 1: Command Line Arguments

```bash
# Using Volcengine GLM-4
claude-proxy -b https://ark.cn-beijing.volces.com/api/v3/chat/completions \
             -m glm-4-7-251222 \
             -k your-api-key \
             -v

# Using DeepSeek
claude-proxy -b https://api.deepseek.com/chat/completions \
             -m deepseek-chat \
             -k your-api-key
```

### Method 2: Environment Variables

```bash
export CLAUDE_PROXY_BACKEND="https://ark.cn-beijing.volces.com/api/v3/chat/completions"
export CLAUDE_PROXY_API_KEY="your-api-key"
export CLAUDE_PROXY_MODEL="glm-4-7-251222"
claude-proxy -v
```

### Method 3: Config File (Inside Container)

Create config file at `/workspace/.claude-proxy.json`:

```json
{
  "backend": "https://ark.cn-beijing.volces.com/api/v3/chat/completions",
  "api_key": "your-api-key",
  "model": "glm-4-7-251222"
}
```

Then run:

```bash
start-claude-proxy
```

## Container Usage

### Docker Run (via Environment Variables)

```bash
docker run -d \
  -e CLAUDE_PROXY_BACKEND="https://ark.cn-beijing.volces.com/api/v3/chat/completions" \
  -e CLAUDE_PROXY_API_KEY="your-api-key" \
  -e CLAUDE_PROXY_MODEL="your-model-name" \
  yaoapp/sandbox-claude:latest
```

The container will automatically start claude-proxy on startup.

### Using claude-run Wrapper

```bash
# Enter the container
docker exec -it <container> bash

# Set environment variables
export CLAUDE_PROXY_BACKEND="https://ark.cn-beijing.volces.com/api/v3/chat/completions"
export CLAUDE_PROXY_API_KEY="your-api-key"
export CLAUDE_PROXY_MODEL="your-model-name"

# Use claude-run wrapper (auto-starts proxy)
claude-run --dangerously-skip-permissions "Build me a website"
```

### Direct Claude CLI Usage

```bash
# Ensure proxy is running
curl http://127.0.0.1:3456/health

# Set Claude CLI environment
export ANTHROPIC_BASE_URL=http://127.0.0.1:3456
export ANTHROPIC_API_KEY=dummy

# Use Claude CLI
claude -p --dangerously-skip-permissions --permission-mode bypassPermissions "Build me a website"
```

## Claude CLI Common Options

```bash
# Basic usage (max permissions, no questions)
claude -p --dangerously-skip-permissions --permission-mode bypassPermissions "your task"

# Streaming JSON output
claude -p --dangerously-skip-permissions --output-format stream-json --verbose "your task"

# Interactive mode (real-time streaming output)
claude --dangerously-skip-permissions "your task"
```

### Option Reference

| Option                                | Description                               |
| ------------------------------------- | ----------------------------------------- |
| `-p, --print`                         | Print mode, exit after output             |
| `--dangerously-skip-permissions`      | Skip all permission checks                |
| `--permission-mode bypassPermissions` | Bypass permission mode                    |
| `--output-format stream-json`         | Output JSON stream                        |
| `--verbose`                           | Verbose output (required for stream-json) |

## Viewing Logs

```bash
# View proxy logs inside container
tail -f /workspace/proxy.log

# Check health status
curl http://127.0.0.1:3456/health
```

## Supported Backends

| Backend                      | API URL                                                     |
| ---------------------------- | ----------------------------------------------------------- |
| Volcengine GLM               | `https://ark.cn-beijing.volces.com/api/v3/chat/completions` |
| Volcengine DeepSeek          | `https://ark.cn-beijing.volces.com/api/v3/chat/completions` |
| DeepSeek Official            | `https://api.deepseek.com/chat/completions`                 |
| OpenAI                       | `https://api.openai.com/v1/chat/completions`                |
| Other OpenAI-compatible APIs | Custom URL                                                  |

## API Endpoints

### POST /v1/messages

Main endpoint, accepts Anthropic Messages API format requests.

### GET /health

Health check endpoint, returns `{"status": "ok"}`.

## Building

```bash
# Local build
go build -o claude-proxy ./cmd/claude-proxy/

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o claude-proxy-amd64 ./cmd/claude-proxy/
GOOS=linux GOARCH=arm64 go build -o claude-proxy-arm64 ./cmd/claude-proxy/
```

## Yao Integration

Yao's sandbox executor automatically:

1. Writes connector config to `/workspace/.claude-proxy.json` when creating container
2. Calls `start-claude-proxy` to start the proxy
3. Sets `ANTHROPIC_BASE_URL` and `ANTHROPIC_API_KEY` environment variables
4. Executes Claude CLI commands

No manual configuration required.
