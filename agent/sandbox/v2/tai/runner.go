package tai

import (
	"context"
	"fmt"
	"time"

	goullm "github.com/yaoapp/gou/llm"
	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tools"
)

// TaiRunner implements the sandbox Runner interface by launching
// "tai agent run --format jsonl" as a subprocess and parsing NDJSON
// events from stdout. Same subprocess+stdio architecture as the DSH
// and OpenCode runners, keeping yao free of SDK dependencies.
type TaiRunner struct {
	lastChatID    string
	lastCompleted bool
	logger        *agentContext.RequestLogger
}

// New creates a new TaiRunner.
func New() *TaiRunner { return &TaiRunner{} }

// Name returns the runner identifier.
func (r *TaiRunner) Name() string { return "tai" }

// Prepare executes user-defined and runner-specific prepare steps.
func (r *TaiRunner) Prepare(ctx context.Context, req *types.PrepareRequest) error {
	if r.logger == nil {
		r.logger = agentContext.NoopLogger()
	}

	prefix := ".tai"
	if req.AssistantID != "" {
		prefix = ".yao/assistants/" + req.AssistantID
	}

	steps := append([]types.PrepareStep{}, req.Config.Prepare...)

	var hasVision bool
	if lc, ok := req.Connector.(goullm.LLMConnector); ok {
		if caps := lc.GetCapabilities(); caps != nil {
			hasVision = caps.HasVision()
		}
	}

	if ws := req.Computer.Workplace(); ws != nil {
		if err := shared.InjectSystemSkills(ws, tools.SkillsFS, prefix+"/skills"); err != nil && r.logger != nil {
			r.logger.Warn("inject system skills: %v", err)
		}
		if err := shared.InjectAgentDefinitions(ws, tools.AgentsFS, prefix+"/agents"); err != nil && r.logger != nil {
			r.logger.Warn("inject agent definitions: %v", err)
		}

		prompt, _ := tools.RenderSystemPrompt(hasVision)
		if err := shared.AppendSystemPrompt(ws, "AGENTS.md", prompt); err != nil && r.logger != nil {
			r.logger.Warn("append AGENTS.md: %v", err)
		}
	}

	if req.SkillsDir != "" {
		if ws := req.Computer.Workplace(); ws != nil {
			src := "local:///" + req.SkillsDir
			dst := prefix + "/skills"
			if _, err := ws.Copy(src, dst); err != nil && r.logger != nil {
				r.logger.Warn("copy skills %s -> %s: %v", src, dst, err)
			}
		}
	}

	if req.RunSteps != nil && len(steps) > 0 {
		if err := req.RunSteps(ctx, steps, req.Computer, req.AssistantID, req.ConfigHash, req.AssistantDir); err != nil {
			return fmt.Errorf("tai prepare steps: %w", err)
		}
	}

	return nil
}

// Stream launches "tai agent run --format jsonl" as a subprocess and
// translates NDJSON events to StreamFunc calls.
func (r *TaiRunner) Stream(ctx context.Context, req *types.StreamRequest, handler message.StreamFunc) error {
	computer := req.Computer
	if computer == nil {
		return fmt.Errorf("computer is nil")
	}

	r.logger = req.Logger
	if r.logger == nil {
		r.logger = agentContext.NoopLogger()
	}

	workDir := computer.GetWorkDir()
	chatID := req.ChatID
	r.lastChatID = chatID

	// Inject per-chat memory/instruction/preference (aligned with DSH runner).
	if chatID != "" {
		if ws := computer.Workplace(); ws != nil {
			if err := shared.InjectChatMemory(ws, chatID, "AGENTS.md"); err != nil {
				r.logger.Warn("inject chat memory: %v", err)
			}
		}
	}

	// Process attachments (aligned with DSH runner).
	var msgParts *shared.MessageParts
	if chatID != "" {
		if ws := computer.Workplace(); ws != nil {
			vision := shared.HasVision(req.Connector)
			processed, mp, err := shared.ExtractMessageParts(ctx, req.Messages, chatID, ws, vision)
			if err != nil {
				r.logger.Warn("extractMessageParts: %v", err)
			} else {
				req.Messages = processed
				msgParts = mp
			}
		}
	}

	cmd := buildCommand(req, workDir, msgParts)

	r.logger.Info("exec: tai agent run session=%s", chatID)

	sess, err := startSession(ctx, computer, cmd, chatID, r.logger)
	if err != nil {
		return err
	}

	streamStart := time.Now()
	completed, err := sess.runStream(handler, req.Locale, req.PrepareLoadingMsgID)
	r.lastCompleted = completed
	elapsed := time.Since(streamStart).Round(time.Second)
	log.Trace("[tai-runner] Stream finished: chatID=%s completed=%v elapsed=%v err=%v",
		chatID, completed, elapsed, err)
	r.logger.Debug("Stream: runStream returned completed=%v err=%v elapsed=%v", completed, err, elapsed)

	if completed || err == nil {
		sess.shutdown()
	}
	return err
}

// Cleanup kills any remaining tai agent process.
func (r *TaiRunner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}
	if r.lastCompleted {
		if r.logger != nil {
			r.logger.Info("cleanup: stream completed normally")
		}
		return nil
	}
	return nil
}
