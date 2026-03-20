package claude

import (
	"context"
	"fmt"
	"os"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// ClaudeRunner implements the Runner interface for Claude CLI (mode=cli).
type ClaudeRunner struct {
	mode           string
	hasMCP         bool
	mcpToolPattern string
	lastCompleted  bool
	logger         *agentContext.RequestLogger
}

// New creates a new ClaudeRunner.
func New() *ClaudeRunner {
	return &ClaudeRunner{mode: "cli"}
}

func (r *ClaudeRunner) Name() string { return "claude" }

// Prepare executes user-defined and runner-specific prepare steps.
func (r *ClaudeRunner) Prepare(ctx context.Context, req *types.PrepareRequest) error {
	r.mode = req.Config.Runner.Mode
	if r.mode == "" {
		r.mode = "cli"
	}

	assistantID := req.Config.ID
	prefix := ".yao/assistants/" + assistantID
	if assistantID == "" {
		prefix = ".claude"
	}

	steps := append([]types.PrepareStep{}, req.Config.Prepare...)

	if req.SkillsDir != "" {
		ws := req.Computer.Workplace()
		if ws != nil {
			src := "local:///" + req.SkillsDir
			dst := prefix + "/skills"
			if _, err := ws.Copy(src, dst); err != nil {
				fmt.Fprintf(os.Stderr, "[claude] warn: copy skills %s -> %s: %v\n", src, dst, err)
			}
		}
	}

	if len(req.MCPServers) > 0 {
		r.hasMCP = true
		r.mcpToolPattern = buildMCPAllowedTools(req.MCPServers)
		mcpJSON := buildMCPConfig(req.MCPServers)
		steps = append(steps, types.PrepareStep{
			Action:  "file",
			Path:    prefix + "/mcp.json",
			Content: mcpJSON,
		})
	}

	if req.RunSteps != nil && len(steps) > 0 {
		if err := req.RunSteps(ctx, steps, req.Computer, req.Config.ID, req.ConfigHash, req.AssistantDir); err != nil {
			return fmt.Errorf("claude prepare steps: %w", err)
		}
	}

	return nil
}

// Stream executes the Claude CLI and streams output to handler.
func (r *ClaudeRunner) Stream(ctx context.Context, req *types.StreamRequest, handler message.StreamFunc) error {
	computer := req.Computer
	if computer == nil {
		return fmt.Errorf("computer is nil")
	}

	p := resolvePlatform(computer)

	if req.ChatID != "" {
		if ws := computer.Workplace(); ws != nil {
			processed, err := prepareAttachments(ctx, req.Messages, req.ChatID, ws)
			if err != nil {
				return fmt.Errorf("prepareAttachments: %w", err)
			}
			req.Messages = processed
		}
	}

	cmd := r.buildCommand(ctx, req, p)

	r.logger = req.Logger
	if r.logger == nil {
		r.logger = agentContext.NoopLogger()
	}

	sess, err := startSession(ctx, computer, p, cmd, r.logger)
	if err != nil {
		return err
	}

	completed, err := sess.runStream(handler)
	r.lastCompleted = completed
	return err
}

// Cleanup kills any remaining claude processes. If the stream completed
// normally (received "result"), child processes are preserved.
func (r *ClaudeRunner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}

	if r.lastCompleted {
		if r.logger != nil {
			r.logger.Info("cleanup: stream completed normally, preserving child processes")
		}
		return nil
	}

	if r.mode != "service" {
		p := resolvePlatform(computer)
		computer.Exec(ctx, p.KillCmd("claude"))
	}

	return nil
}
