package dsh

import (
	"context"
	"fmt"
	"strings"

	goullm "github.com/yaoapp/gou/llm"
	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tools"
)

// Runner implements the sandbox Runner interface for DSH (DeepSeek Harness).
type Runner struct {
	mode          string
	lastCompleted bool
	lastChatID    string
	logger        *agentContext.RequestLogger
}

// New creates a new DSH Runner.
func New() *Runner {
	return &Runner{mode: "cli", logger: agentContext.NoopLogger()}
}

// Name returns the runner identifier.
func (r *Runner) Name() string { return "dsh" }

// Prepare executes user-defined and runner-specific prepare steps (skills, prompts, configs).
func (r *Runner) Prepare(ctx context.Context, req *types.PrepareRequest) error {
	if r.logger == nil {
		r.logger = agentContext.NoopLogger()
	}

	r.mode = req.Config.Runner.Mode
	if r.mode == "" {
		r.mode = "cli"
	}

	assistantID := req.AssistantID
	prefix := ".yao/assistants/" + assistantID
	if assistantID == "" {
		prefix = ".dsh"
	}

	steps := append([]types.PrepareStep{}, req.Config.Prepare...)

	var hasVision bool
	if lc, ok := req.Connector.(goullm.LLMConnector); ok {
		if caps := lc.GetCapabilities(); caps != nil {
			hasVision = caps.HasVision()
		}
	}

	if ws := req.Computer.Workplace(); ws != nil {
		if err := shared.InjectSystemSkills(ws, tools.SkillsFS, ".dsh/skills"); err != nil {
			r.logger.Warn("inject system skills: %v", err)
		}
		if err := shared.InjectAgentDefinitions(ws, tools.AgentsFS, prefix+"/agents"); err != nil {
			r.logger.Warn("inject agent definitions: %v", err)
		}

		prompt, _ := tools.RenderSystemPrompt(hasVision)

		if err := shared.AppendSystemPrompt(ws, "AGENTS.md", prompt); err != nil {
			r.logger.Warn("append AGENTS.md: %v", err)
		}
	}

	if req.SkillsDir != "" {
		ws := req.Computer.Workplace()
		if ws != nil {
			src := "local:///" + req.SkillsDir
			dst := prefix + "/skills"
			if _, err := ws.Copy(src, dst); err != nil {
				r.logger.Warn("copy skills %s -> %s: %v", src, dst, err)
			}
		}
	}

	if req.RunSteps != nil && len(steps) > 0 {
		if err := req.RunSteps(ctx, steps, req.Computer, req.AssistantID, req.ConfigHash, req.AssistantDir); err != nil {
			return fmt.Errorf("dsh prepare steps: %w", err)
		}
	}

	return nil
}

// Stream executes `tai dsh` and streams output to handler.
func (r *Runner) Stream(ctx context.Context, req *types.StreamRequest, handler message.StreamFunc) error {
	computer := req.Computer
	if computer == nil {
		return fmt.Errorf("computer is nil")
	}

	p := resolvePlatform(computer)

	r.logger = req.Logger
	if r.logger == nil {
		r.logger = agentContext.NoopLogger()
	}

	// Process attachments (aligned with Claude Runner)
	var msgParts *shared.MessageParts
	if req.ChatID != "" {
		if ws := computer.Workplace(); ws != nil {
			vision := shared.HasVision(req.Connector)
			processed, mp, err := shared.ExtractMessageParts(ctx, req.Messages, req.ChatID, ws, vision)
			if err != nil {
				r.logger.Warn("extractMessageParts: %v", err)
			} else {
				req.Messages = processed
				msgParts = mp
			}
		}
	}

	cmd, err := r.buildCommand(req, p, msgParts)
	if err != nil {
		return fmt.Errorf("buildCommand: %w", err)
	}

	chatID := req.ChatID
	r.lastChatID = chatID
	assistantID := req.AssistantID

	log.Trace("[dsh-runner] Stream started: assistantID=%s chatID=%s", assistantID, chatID)
	r.logger.Debug("env vars passed to session (%d total):", len(cmd.env))
	for k, v := range cmd.env {
		if strings.HasPrefix(k, "CTX_") || k == "DSH_CWD" || k == "HOME" || k == "WORKDIR" {
			r.logger.Debug("  %s=%s", k, v)
		} else {
			r.logger.Debug("  %s=(set, len=%d)", k, len(v))
		}
	}

	if msgParts != nil && len(msgParts.ImageBlocks) > 0 {
		totalBytes := 0
		for _, img := range msgParts.ImageBlocks {
			totalBytes += len(img.Data)
		}
		r.logger.Info("vision input: images=%d totalBase64=%d bytes", len(msgParts.ImageBlocks), totalBytes)
	}

	sess, err := startSession(ctx, computer, p, cmd, cmd.sessionID, r.logger, req.Locale, req.PrepareLoadingMsgID)
	if err != nil {
		return err
	}
	defer sess.exec.Cancel()

	completed, streamErr := sess.runStream(handler)
	r.lastCompleted = completed

	if completed {
		sess.shutdown()
		return nil
	}

	if streamErr != nil {
		return streamErr
	}

	return nil
}

// Cleanup kills any remaining DSH processes.
func (r *Runner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}

	log.Trace("[dsh-runner] Cleanup: chatID=%s lastCompleted=%v", r.lastChatID, r.lastCompleted)

	if r.lastCompleted {
		if r.logger != nil {
			r.logger.Info("cleanup: stream completed normally, preserving child processes")
		}
		return nil
	}

	p := resolvePlatform(computer)
	if r.lastChatID != "" {
		computer.Exec(ctx, p.KillSessionCmd("dsh-"+r.lastChatID))
	} else {
		computer.Exec(ctx, p.KillCmd("dsh-jsonrpc-agent"))
	}

	return nil
}
