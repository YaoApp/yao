package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
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
	lastChatID     string
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
				r.logger.Warn("copy skills %s -> %s: %v", src, dst, err)
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

	// Inject connector config into a2o proxy (best-effort, errors ignored).
	if req.Connector != nil && req.Connector.Is(connector.OPENAI) {
		injectA2OConfig(ctx, computer, req.Connector)
	}

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

	chatID := req.ChatID
	r.lastChatID = chatID
	assistantID := ""
	if req.Config != nil {
		assistantID = req.Config.ID
	}

	log.Trace("[claude-runner] Stream started: assistantID=%s chatID=%s promptLen=%d", assistantID, chatID, len(cmd.shell))

	sess, err := startSession(ctx, computer, p, cmd, chatID, r.logger)
	if err != nil {
		return err
	}

	streamStart := time.Now()
	completed, err := sess.runStream(handler)
	r.lastCompleted = completed
	elapsed := time.Since(streamStart).Round(time.Second)
	log.Trace("[claude-runner] Stream finished: assistantID=%s chatID=%s completed=%v elapsed=%v err=%v", assistantID, chatID, completed, elapsed, err)
	r.logger.Debug("Stream: runStream returned completed=%v err=%v elapsed=%v", completed, err, elapsed)
	if completed {
		sess.shutdown()
		if chatID != "" {
			assistantID := ""
			if req.Config != nil {
				assistantID = req.Config.ID
			}
			storeKey := "claude-session:" + assistantID + ":" + chatID
			sessionUUID := chatIDToSessionUUID(assistantID, chatID)
			markChatSession(storeKey, sessionUUID, 90*24*time.Hour)
		}
	}
	return err
}

// Cleanup kills any remaining claude processes. If the stream completed
// normally (received "result"), child processes are preserved.
func (r *ClaudeRunner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}

	log.Trace("[claude-runner] Cleanup: chatID=%s lastCompleted=%v", r.lastChatID, r.lastCompleted)

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
			computer.Exec(ctx, p.KillCmd("claude"))
		}
	}

	return nil
}

type a2oConnectorConfig struct {
	Backend string                 `json:"backend"`
	Model   string                 `json:"model"`
	APIKey  string                 `json:"api_key"`
	Options map[string]interface{} `json:"options,omitempty"`
}

func buildSingleA2OConfig(conn connector.Connector) *a2oConnectorConfig {
	settings := conn.Setting()
	if settings == nil {
		return nil
	}

	cfg := &a2oConnectorConfig{}

	if host, ok := settings["host"].(string); ok && host != "" {
		cfg.Backend = connector.BuildAPIURL(host, "/chat/completions")
	} else if proxy, ok := settings["proxy"].(string); ok && proxy != "" {
		cfg.Backend = connector.BuildAPIURL(proxy, "/chat/completions")
	}
	if model, ok := settings["model"].(string); ok && model != "" {
		cfg.Model = model
	}
	if key, ok := settings["key"].(string); ok && key != "" {
		cfg.APIKey = key
	}

	extra := make(map[string]interface{})
	for k, v := range settings {
		switch k {
		case "host", "model", "key", "proxy", "type":
			continue
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		cfg.Options = extra
	}

	if cfg.Backend == "" {
		return nil
	}
	return cfg
}

// injectA2OConfig pushes the connector config to the a2o proxy.
// For box (Linux container): uses sh pipe since Docker exec stdin may not work.
// For host: uses WithStdin which works reliably on all platforms.
// Best-effort: errors are logged and ignored.
func injectA2OConfig(ctx context.Context, computer infra.Computer, conn connector.Connector) {
	cfg := buildSingleA2OConfig(conn)
	if cfg == nil {
		log.Trace("[claude] injectA2OConfig: no valid config for connector %s", conn.ID())
		return
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		log.Trace("[claude] injectA2OConfig: marshal error: %v", err)
		return
	}

	connID := conn.ID()
	var result *infra.ExecResult

	info := computer.ComputerInfo()
	if info.Kind == "host" {
		result, err = computer.Exec(ctx, []string{"tai", "a2o", "config", "put", connID}, infra.WithStdin(data))
	} else {
		escaped := strings.ReplaceAll(string(data), "'", "'\\''")
		script := fmt.Sprintf("echo '%s' | tai a2o config put %s", escaped, connID)
		result, err = computer.Exec(ctx, []string{"sh", "-c", script})
	}

	if err != nil {
		log.Trace("[claude] injectA2OConfig: exec error (ignored): %v", err)
		return
	}
	if result.ExitCode != 0 {
		log.Trace("[claude] injectA2OConfig: exit %d stderr=%s (ignored)", result.ExitCode, result.Stderr)
		return
	}

	log.Trace("[claude] injectA2OConfig: connector=%s injected ok", connID)
}
