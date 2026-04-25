package opencode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// Runner implements the sandbox Runner interface for OpenCode CLI.
type Runner struct {
	mode          string
	hasMCP        bool
	mcpServers    []types.MCPServer
	lastCompleted bool
	lastChatID    string
	logger        *agentContext.RequestLogger
}

// New creates a new OpenCode Runner.
func New() *Runner {
	return &Runner{mode: "cli"}
}

// Name returns the runner identifier. Must NOT be "yao" (see agent.go branching).
func (r *Runner) Name() string { return "opencode" }

// Prepare executes user-defined and runner-specific prepare steps.
func (r *Runner) Prepare(ctx context.Context, req *types.PrepareRequest) error {
	r.mode = req.Config.Runner.Mode
	if r.mode == "" {
		r.mode = "cli"
	}

	assistantID := req.AssistantID
	prefix := ".yao/assistants/" + assistantID
	if assistantID == "" {
		prefix = ".opencode"
	}

	steps := append([]types.PrepareStep{}, req.Config.Prepare...)

	// 1. Skills copy (aligned with Claude Runner)
	if req.SkillsDir != "" {
		ws := req.Computer.Workplace()
		if ws != nil {
			src := "local:///" + req.SkillsDir
			dst := prefix + "/skills"
			if _, err := ws.Copy(src, dst); err != nil {
				log.Warn("[opencode-runner] copy skills %s -> %s: %v", src, dst, err)
			}
		}
	}

	// 2. MCP servers -> stored for opencode.json generation
	if len(req.MCPServers) > 0 {
		r.hasMCP = true
		r.mcpServers = req.MCPServers
	}

	// 3. Create OPENCODE_*_DIR directories (data/config/state/cache) via
	// workspace.FS so directory ownership matches the workspace mount.
	// OpenCode writes files (e.g. .gitignore, SQLite DB) into these dirs
	// on first startup and crashes if they don't exist.
	if ws := req.Computer.Workplace(); ws != nil {
		for _, sub := range []string{"data", "config", "state", "cache"} {
			ws.MkdirAll(prefix+"/opencode/"+sub, 0777)
		}
	}

	// 4. Copy custom tools (e.g. read.ts for vision) into OpenCode global
	// config dir ($HOME/.config/opencode/tools/).  Only needed when a
	// vision connector is configured — the custom read tool overrides the
	// built-in read to route image files through the vision API.
	if req.Config != nil && req.Config.Runner.Connectors != nil {
		if vc, ok := req.Config.Runner.Connectors["vision"]; ok && vc != nil && vc.Connector != "" {
			steps = append(steps, types.PrepareStep{
				Action:      "exec",
				Cmd:         "mkdir -p $HOME/.config/opencode/tools && for f in /opt/opencode-tools/*.ts; do [ -f \"$f\" ] && cp -f \"$f\" $HOME/.config/opencode/tools/; done",
				Once:        true,
				IgnoreError: true,
			})
		}
	}

	// 5. Generate opencode.json (project config at workspace root)
	configJSON := buildOpenCodeConfig(req, r.mcpServers)
	steps = append(steps, types.PrepareStep{
		Action:  "file",
		Path:    "opencode.json",
		Content: configJSON,
	})

	// 5. System prompt file
	// Written via a prepare step so configHash dedup applies.
	// opencode.json instructions field references this path.
	// (System prompt is injected at Stream time if not a continuation.)

	// 6. Execute all prepare steps via RunPrepareSteps (configHash dedup)
	if req.RunSteps != nil && len(steps) > 0 {
		if err := req.RunSteps(ctx, steps, req.Computer, req.AssistantID, req.ConfigHash, req.AssistantDir); err != nil {
			return fmt.Errorf("opencode prepare steps: %w", err)
		}
	}

	return nil
}

// Stream executes the OpenCode CLI and streams output to handler.
func (r *Runner) Stream(ctx context.Context, req *types.StreamRequest, handler message.StreamFunc) error {
	computer := req.Computer
	if computer == nil {
		return fmt.Errorf("computer is nil")
	}

	p := resolvePlatform(computer)

	// Resolve attachments (shared with Claude runner).
	var attachmentPaths []string
	if req.ChatID != "" {
		if ws := computer.Workplace(); ws != nil {
			_, resolved, err := shared.PrepareAttachments(ctx, req.Messages, req.ChatID, ws)
			if err != nil {
				return fmt.Errorf("prepareAttachments: %w", err)
			}
			workDir := computer.GetWorkDir()
			for _, ar := range resolved {
				attachmentPaths = append(attachmentPaths, p.PathJoin(workDir, ar.Path))
			}
		}
	}

	// Write system prompt if this is the first turn.
	assistantID := req.AssistantID
	chatID := req.ChatID
	storeKey := "opencode-session:" + assistantID + ":" + chatID
	isContinuation := chatID != "" && chatSessionExists(storeKey)

	if !isContinuation && req.SystemPrompt != "" {
		if ws := computer.Workplace(); ws != nil {
			prefix := ".yao/assistants/" + assistantID
			if assistantID == "" {
				prefix = ".opencode"
			}
			promptPath := prefix + "/system-prompt.md"
			envPrompt := buildSandboxEnvPrompt(p, computer.GetWorkDir())
			fullPrompt := req.SystemPrompt + "\n\n" + envPrompt
			ws.MkdirAll(prefix, 0755)
			ws.WriteFile(promptPath, []byte(fullPrompt), 0644)
		}
	}

	cmd := r.buildCommand(req, p, attachmentPaths)

	r.logger = req.Logger
	if r.logger == nil {
		r.logger = agentContext.NoopLogger()
	}

	r.lastChatID = chatID

	log.Trace("[opencode-runner] Stream started: assistantID=%s chatID=%s", assistantID, chatID)
	r.logger.Debug("env vars passed to session (%d total):", len(cmd.env))
	for k, v := range cmd.env {
		if strings.HasPrefix(k, "CTX_") || k == "OPENCODE_DATA_DIR" || k == "HOME" || k == "WORKDIR" {
			r.logger.Debug("  %s=%s", k, v)
		} else {
			r.logger.Debug("  %s=(set, len=%d)", k, len(v))
		}
	}

	sess, err := startSession(ctx, computer, p, cmd, chatID, r.logger)
	if err != nil {
		return err
	}

	streamStart := time.Now()
	completed, err := sess.runStream(handler)
	r.lastCompleted = completed
	elapsed := time.Since(streamStart).Round(time.Second)
	log.Trace("[opencode-runner] Stream finished: assistantID=%s chatID=%s completed=%v elapsed=%v err=%v",
		assistantID, chatID, completed, elapsed, err)
	r.logger.Debug("Stream: runStream returned completed=%v err=%v elapsed=%v", completed, err, elapsed)

	// Mark session in store after a clean run (err==nil) so future
	// requests can use --continue --session to resume. This covers both
	// completed=true (single-step stop) and completed=false (multi-step
	// with tool-calls where the process exited normally).
	if err == nil && chatID != "" {
		sessionID := chatIDToSessionID(assistantID, chatID)
		markChatSession(storeKey, sessionID, 90*24*time.Hour)
	}

	if completed || err == nil {
		sess.shutdown()
	}
	return err
}

// Cleanup kills any remaining opencode processes. If the stream completed
// normally (received step_finish with stop), child processes are preserved.
func (r *Runner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}

	log.Trace("[opencode-runner] Cleanup: chatID=%s lastCompleted=%v", r.lastChatID, r.lastCompleted)

	if r.lastCompleted {
		if r.logger != nil {
			r.logger.Info("cleanup: stream completed normally, preserving child processes")
		}
		return nil
	}

	if r.mode != "service" {
		p := resolvePlatform(computer)
		if r.lastChatID != "" {
			computer.Exec(ctx, p.KillSessionCmd(sanitizeSessionName(r.lastChatID)))
		} else {
			computer.Exec(ctx, p.KillCmd("opencode"))
		}
	}

	return nil
}
