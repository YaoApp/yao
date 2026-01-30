package sandbox

import (
	"fmt"

	"github.com/yaoapp/yao/agent/sandbox/claude"
	infraSandbox "github.com/yaoapp/yao/sandbox"
)

// New creates a new Executor based on the command type
func New(manager *infraSandbox.Manager, opts *Options) (Executor, error) {
	if opts == nil {
		return nil, fmt.Errorf("options is required")
	}

	if !IsValidCommand(opts.Command) {
		return nil, fmt.Errorf("unsupported command type: %s, supported: %v", opts.Command, CommandTypes)
	}

	// Set default image if not specified
	if opts.Image == "" {
		opts.Image = DefaultImage(opts.Command)
	}

	switch opts.Command {
	case "claude":
		// Convert to claude.Options
		claudeOpts := &claude.Options{
			Command:       opts.Command,
			Image:         opts.Image,
			MaxMemory:     opts.MaxMemory,
			MaxCPU:        opts.MaxCPU,
			Timeout:       opts.Timeout,
			Arguments:     opts.Arguments,
			UserID:        opts.UserID,
			ChatID:        opts.ChatID,
			MCPConfig:     opts.MCPConfig,
			MCPTools:      opts.MCPTools, // MCP tools to expose via IPC
			SkillsDir:     opts.SkillsDir,
			ConnectorHost: opts.ConnectorHost,
			ConnectorKey:  opts.ConnectorKey,
			Model:         opts.Model,
		}
		return claude.NewExecutor(manager, claudeOpts)
	case "cursor":
		return nil, fmt.Errorf("cursor executor not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported command type: %s", opts.Command)
	}
}
