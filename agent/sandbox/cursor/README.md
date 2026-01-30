# Cursor Executor

## Status

**Not Implemented** - This is a placeholder for future Cursor CLI integration.

## Planned Features

The Cursor executor will provide similar functionality to the Claude executor:

- Execute Cursor CLI in a Docker sandbox container
- Stream output in real-time
- File system operations (ReadFile, WriteFile, ListDir)
- Command execution (Exec)
- Integration with Yao's MCP servers

## Configuration

When implemented, the Cursor executor will be configured in assistant `package.yao`:

```jsonc
{
  "name": "Coder Assistant",
  "connector": "deepseek.v3",
  "sandbox": {
    "command": "cursor",              // Use Cursor CLI
    "image": "yaoapp/sandbox-cursor:latest",
    "timeout": "10m"
  }
}
```

## Implementation Notes

The implementation should follow the same pattern as `claude/executor.go`:

1. Create `cursor/executor.go` implementing the `sandbox.Executor` interface
2. Create `cursor/command.go` for building Cursor CLI commands
3. Create `cursor/types.go` for Cursor-specific types
4. Add appropriate tests

## Docker Image

A `yaoapp/sandbox-cursor` Docker image will need to be created with:

- Ubuntu 24.04 LTS base
- Node.js 22 LTS
- Python 3.12
- Cursor CLI installed and configured

## References

- [Cursor CLI Documentation](https://cursor.sh/docs)
- [Claude Executor Implementation](../claude/executor.go)
- [Sandbox Design Document](../DESIGN.md)
