package claude

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

const defaultA2OPort = 3099
const defaultA2OMaxOutputTokens = 16384

var yaoSessionNS = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

var safeNameRe = regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)

func hashUserID(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:8]) // 16 hex chars — short yet collision-safe
}

func chatIDToSessionUUID(assistantID, chatID string) string {
	return uuid.NewSHA1(yaoSessionNS, []byte(assistantID+":"+chatID)).String()
}

func sanitizeSessionName(chatID string) string {
	return "yao-" + safeNameRe.ReplaceAllString(chatID, "_")
}

func chatSessionExists(storeKey string) bool {
	s, err := store.Get("__yao.store")
	if err != nil {
		return false
	}
	return s.Has(storeKey)
}

func markChatSession(storeKey, sessionUUID string, ttl time.Duration) {
	s, err := store.Get("__yao.store")
	if err != nil {
		return
	}
	s.Set(storeKey, sessionUUID, ttl)
}

type command struct {
	shell   []string
	env     map[string]string
	stdin   []byte
	workDir string
}

func (r *Runner) buildCommand(ctx context.Context, req *types.StreamRequest, p platform) command {
	computer := req.Computer
	workDir := computer.GetWorkDir()
	assistantID := req.AssistantID
	chatID := req.ChatID

	var isContinuation bool
	if chatID != "" {
		storeKey := "claude-session:" + assistantID + ":" + chatID + ":" + req.Config.WorkspaceID
		isContinuation = chatSessionExists(storeKey)
	} else {
		isContinuation = hasExistingSession(ctx, computer, p, assistantID)
	}

	env := buildEnv(req, p)
	args := buildArgs(req, r, p, isContinuation, assistantID, chatID)
	inputJSONL := buildLastUserMessageJSONL(req.Messages)

	var systemPrompt string
	envPrompt := buildSandboxEnvPrompt(p, workDir)
	if svcPrompt := buildServicePrompt(req.Config); svcPrompt != "" {
		envPrompt += "\n\n" + svcPrompt
	}
	if capPrompt := buildModelCapabilityPrompt(req); capPrompt != "" {
		envPrompt += "\n\n" + capPrompt
	}
	if !isContinuation && req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt + "\n\n" + envPrompt
	} else if !isContinuation {
		systemPrompt = envPrompt
	}

	promptFile := p.PathJoin(workDir, ".yao", "assistants", assistantID, "system-prompt.txt")
	if assistantID == "" {
		promptFile = p.PathJoin(workDir, ".yao", ".system-prompt.txt")
	}

	// On continuation turns the system prompt is not re-sent, but the previously
	// written prompt file is still on disk. Pass --append-system-prompt-file so
	// Claude keeps the same constraints (e.g. workspace path rules) across the
	// entire session without injecting a duplicate system turn.
	if isContinuation {
		args = append(args, "--append-system-prompt-file", promptFile)
	}

	script, stdin := p.BuildScript(scriptInput{
		args:         args,
		systemPrompt: systemPrompt,
		inputJSONL:   inputJSONL,
		workDir:      workDir,
		promptFile:   promptFile,
	})

	return command{
		shell:   p.ShellCmd(script),
		env:     env,
		stdin:   stdin,
		workDir: workDir,
	}
}

func buildEnv(req *types.StreamRequest, p platform) map[string]string {
	env := make(map[string]string)
	workDir := req.Computer.GetWorkDir()

	// Workspace ID
	workspace := req.Computer.Workplace()
	if workspace != nil {
		workspaceID, err := workspace.GetID()
		if err == nil {
			env["CTX_WORKSPACE_ID"] = workspaceID
		}
	}

	for k, v := range p.HomeEnv(workDir) {
		env[k] = v
	}
	env["WORKDIR"] = workDir

	if req.Locale != "" {
		env["CTX_LOCALE"] = req.Locale
	}

	assistantID := req.AssistantID
	if assistantID != "" {
		configDir := p.PathJoin(workDir, ".yao", "assistants", assistantID)
		env["CLAUDE_CONFIG_DIR"] = configDir
		env["CTX_ASSISTANT_ID"] = assistantID
		// CTX_SKILLS_DIR: absolute path to the skills directory inside the sandbox.
		// Use this in skill scripts instead of constructing the path manually,
		// so it works correctly on all platforms (Linux, macOS, Windows).
		env["CTX_SKILLS_DIR"] = p.PathJoin(workDir, ".yao", "assistants", assistantID, "skills")
	}

	if req.Config != nil && req.Config.NodeID != "" {
		env["CTX_NODE_ID"] = req.Config.NodeID
		if req.Config.ID != "" {
			env["CTX_TARGET_ID"] = req.Config.ID
		} else {
			env["CTX_TARGET_ID"] = "__host__"
		}
	}

	if req.Connector != nil {
		setting := req.Connector.Setting()

		var host, key, model string
		if lc, ok := req.Connector.(goullm.LLMConnector); ok {
			host = lc.GetURL()
			key = lc.GetKey()
			model = lc.GetModel()
		}
		if host == "" {
			host, _ = setting["host"].(string)
		}
		if key == "" {
			key, _ = setting["key"].(string)
		}
		if model == "" {
			model, _ = setting["model"].(string)
		}

		isAnthropic := req.Connector.Is(connector.ANTHROPIC)

		if isAnthropic {
			setAnthropicModelEnv(env, host, key, model, req.Connector)
			applyAnthropicRoleOverrides(env, host, req.Roles)
		} else {
			var a2oPort int
			if req.Computer != nil {
				a2oPort = req.Computer.ComputerInfo().Ports["a2o"]
			}
			setA2OModelEnv(env, req.Connector.ID(), model, req.Connector, a2oPort)
			applyA2ORoleOverrides(env, req.Roles)
		}

		if lc, ok := req.Connector.(goullm.LLMConnector); ok {
			if caps := lc.GetCapabilities(); caps != nil {
				if caps.MaxOutputTokens > 0 {
					env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = fmt.Sprintf("%d", caps.MaxOutputTokens)
				}
				if caps.MaxInputTokens > 0 {
					env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] = fmt.Sprintf("%d", caps.MaxInputTokens)
				}
			}
		}
		if _, ok := env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"]; !ok && !req.Connector.Is(connector.ANTHROPIC) {
			env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = fmt.Sprintf("%d", defaultA2OMaxOutputTokens)
		}

		if thinking, ok := setting["thinking"].(map[string]interface{}); ok {
			thinkType, _ := thinking["type"].(string)
			switch thinkType {
			case "disabled":
				env["MAX_THINKING_TOKENS"] = "0"
			case "enabled":
				if budget, ok := thinking["budget_tokens"].(float64); ok && budget > 0 {
					env["MAX_THINKING_TOKENS"] = fmt.Sprintf("%d", int(budget))
				}
			}
		}
	}

	if req.Token != nil {
		if req.Token.Token != "" {
			env["YAO_TOKEN"] = req.Token.Token
		}
		if req.Token.RefreshToken != "" {
			env["YAO_REFRESH_TOKEN"] = req.Token.RefreshToken
		}
	}

	if req.Computer != nil && req.Computer.ComputerInfo().Kind == "host" {
		if addr := infra.ResolveHostGRPCAddr(req.Computer.ComputerInfo().NodeID); addr != "" {
			env["YAO_GRPC_ADDR"] = addr
		}
	}

	logger := req.Logger
	if logger == nil {
		logger = agentContext.NoopLogger()
	}
	connectorID := ""
	if req.Connector != nil {
		connectorID = req.Connector.ID()
	}
	logger.Debug("claude-env: connector=%s isAnthropic=%v", connectorID, req.Connector != nil && req.Connector.Is(connector.ANTHROPIC))
	logger.Debug("claude-env: ANTHROPIC_MODEL=%s", env["ANTHROPIC_MODEL"])
	logger.Debug("claude-env: OPUS_MODEL=%s SONNET_MODEL=%s HAIKU_MODEL=%s",
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"],
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"],
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	logger.Debug("claude-env: CUSTOM_MODEL_OPTION=%s CAPABILITIES=%s",
		env["ANTHROPIC_CUSTOM_MODEL_OPTION"],
		env["ANTHROPIC_CUSTOM_MODEL_OPTION_SUPPORTED_CAPABILITIES"])
	logger.Debug("claude-env: MAX_THINKING_TOKENS=%s", env["MAX_THINKING_TOKENS"])

	// Override metadata.user_id with a sanitized value.
	// Claude CLI sets metadata.user_id to a JSON object that third-party
	// Anthropic-compatible APIs (e.g. DeepSeek) reject because the value
	// doesn't match ^[a-zA-Z0-9_-]+$.
	if _, ok := env["CLAUDE_CODE_EXTRA_BODY"]; !ok {
		uid := "yao-sandbox"
		if req.Config != nil && req.Config.Owner != "" {
			uid = hashUserID(req.Config.Owner)
		} else if assistantID != "" {
			uid = hashUserID(assistantID)
		}
		env["CLAUDE_CODE_EXTRA_BODY"] = fmt.Sprintf(`{"metadata":{"user_id":"%s"}}`, uid)
	}
	logger.Debug("claude-env: EXTRA_BODY=%s", env["CLAUDE_CODE_EXTRA_BODY"])

	return env
}

func buildArgs(req *types.StreamRequest, r *Runner, p platform, isContinuation bool, assistantID, chatID string) []string {
	var args []string

	permMode := ""
	if req.Config != nil && req.Config.Runner.Options != nil {
		if v, ok := req.Config.Runner.Options["permission_mode"]; ok {
			permMode = fmt.Sprintf("%v", v)
		}
	}
	if permMode == "bypassPermissions" {
		args = append(args, "--dangerously-skip-permissions")
		args = append(args, "--permission-mode", permMode)
	}

	args = append(args, "--input-format", "stream-json")
	args = append(args, "--output-format", "stream-json")
	args = append(args, "--include-partial-messages")
	args = append(args, "--verbose")

	if chatID != "" {
		sessionUUID := chatIDToSessionUUID(assistantID, chatID)
		sessionName := sanitizeSessionName(chatID)
		if isContinuation {
			args = append(args, "--resume", sessionUUID)
		} else {
			args = append(args, "--session-id", sessionUUID)
		}
		args = append(args, "--name", sessionName)
	} else if isContinuation {
		args = append(args, "--continue")
	}

	if req.Config != nil && req.Config.Runner.Options != nil {
		for key, val := range req.Config.Runner.Options {
			if flag, ok := claudeArgWhitelist[key]; ok {
				args = append(args, flag, fmt.Sprintf("%v", val))
			}
		}
	}

	if r.hasMCP {
		workDir := req.Computer.GetWorkDir()
		mcpPath := p.PathJoin(workDir, ".yao", "assistants", assistantID, "mcp.json")
		if assistantID == "" {
			mcpPath = p.PathJoin(workDir, ".claude", "mcp.json")
		}
		args = append(args, "--mcp-config", mcpPath)
		if r.mcpToolPattern != "" {
			args = append(args, "--allowedTools", r.mcpToolPattern)
		}
	}

	return args
}

func buildSandboxEnvPrompt(p platform, workDir string) string {
	osName := p.OS()
	if osName == "" {
		osName = "linux"
	}
	shell := p.Shell()
	if shell == "" {
		shell = "bash"
	}

	shellNote := p.EnvPromptNote()

	return fmt.Sprintf(`## Sandbox Environment

- **Operating System**: %[2]s
- **Shell**: %[3]s
- **Working Directory**: %[1]s
- **File Access**: You have full read/write access to %[1]s
%[4]s`, workDir, osName, shell, shellNote)
}

func buildServicePrompt(cfg *types.SandboxConfig) string {
	if cfg == nil || len(cfg.Computer.Ports) == 0 || cfg.NodeID == "" {
		return ""
	}
	targetID := cfg.ID
	if targetID == "" {
		targetID = "__host__"
	}
	var sb strings.Builder
	sb.WriteString("## Web Services\n\n")
	sb.WriteString("When you start a web server or the user asks to access a service, ")
	sb.WriteString("ALWAYS provide a clickable service link using markdown format with a descriptive title:\n\n")
	sb.WriteString(fmt.Sprintf("  [Title](service://%s/%s/{port}[/path][?title=URL+Encoded+Title])\n\n", cfg.NodeID, targetID))
	sb.WriteString("Examples:\n")
	sb.WriteString(fmt.Sprintf("- [My App](service://%s/%s/3000)\n", cfg.NodeID, targetID))
	sb.WriteString(fmt.Sprintf("- [API Docs](service://%s/%s/8080/docs?title=API+Documentation)\n\n", cfg.NodeID, targetID))
	sb.WriteString("The title in markdown link text takes priority for display. ")
	sb.WriteString("The ?title= query parameter is a fallback for bare URLs.\n\n")
	sb.WriteString("Configured ports:\n")
	for _, p := range cfg.Computer.Ports {
		if p.Label != "" {
			sb.WriteString(fmt.Sprintf("- %d (%s)\n", p.Port, p.Label))
		} else {
			sb.WriteString(fmt.Sprintf("- %d\n", p.Port))
		}
	}
	return sb.String()
}

func buildModelCapabilityPrompt(req *types.StreamRequest) string {
	if req.Connector == nil {
		return ""
	}

	lc, ok := req.Connector.(goullm.LLMConnector)
	if !ok {
		return ""
	}
	primaryModel := lc.GetModel()
	if primaryModel == "" {
		return ""
	}
	primaryCaps := lc.GetCapabilities()

	type tierInfo struct {
		tier  string
		alias string
		model string
		caps  *goullm.Capabilities
		conn  connector.Connector
	}

	tiers := []tierInfo{
		{tier: "Default", alias: "sonnet", model: primaryModel, caps: primaryCaps, conn: req.Connector},
	}

	hasDifferentTier := false
	if rc, exists := req.Roles["heavy"]; exists && rc != nil {
		m := connectorModel(rc)
		if m != "" {
			caps := connectorCaps(rc)
			tiers = append(tiers, tierInfo{tier: "Heavy", alias: "opus", model: m, caps: caps, conn: rc})
			if m != primaryModel {
				hasDifferentTier = true
			}
		}
	}
	if rc, exists := req.Roles["light"]; exists && rc != nil {
		m := connectorModel(rc)
		if m != "" {
			caps := connectorCaps(rc)
			tiers = append(tiers, tierInfo{tier: "Light", alias: "haiku", model: m, caps: caps, conn: rc})
			if m != primaryModel {
				hasDifferentTier = true
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("## Model Capabilities\n\n")
	sb.WriteString(fmt.Sprintf("Your current model: `%s`\n", primaryModel))

	if hasDifferentTier {
		sb.WriteString("\n### Available Model Tiers\n\n")
		sb.WriteString("| Tier | Alias | Model | Capabilities |\n")
		sb.WriteString("| ---- | ----- | ----- | ------------ |\n")
		for _, t := range tiers {
			capList := formatCapabilities(t.caps, t.conn)
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", t.tier, t.alias, t.model, capList))
		}
	}

	var guidance []string
	if hasDifferentTier {
		guidance = append(guidance,
			"For complex reasoning, multi-step analysis, or tasks requiring deep thought, delegate to a sub-agent with `model: \"opus\"`",
			"For simple tasks (formatting, translation, summarization), use `model: \"haiku\"` for faster responses",
		)
	}

	primaryHasVision := primaryCaps.HasVision()
	if !primaryHasVision {
		if _, hasVisionRole := req.Roles["vision"]; hasVisionRole {
			guidance = append(guidance,
				"**Image/Vision**: Your current model cannot process images directly. Use the `image_read` system tool (`tai tool image_read`) to analyze images — see the yao-image skill for details",
			)
		}
	}

	if len(guidance) > 0 {
		sb.WriteString("\n### Usage Guidance\n\n")
		for _, g := range guidance {
			sb.WriteString("- " + g + "\n")
		}
	}

	if !hasDifferentTier && len(guidance) == 0 {
		return ""
	}

	return sb.String()
}

func connectorModel(c connector.Connector) string {
	if lc, ok := c.(goullm.LLMConnector); ok {
		if m := lc.GetModel(); m != "" {
			return m
		}
	}
	m, _ := c.Setting()["model"].(string)
	return m
}

func connectorCaps(c connector.Connector) *goullm.Capabilities {
	if lc, ok := c.(goullm.LLMConnector); ok {
		return lc.GetCapabilities()
	}
	return nil
}

func formatCapabilities(caps *goullm.Capabilities, conn connector.Connector) string {
	var parts []string
	hasThinking := caps.HasReasoning()
	if !hasThinking && conn != nil {
		if thinking, ok := conn.Setting()["thinking"].(map[string]interface{}); ok {
			if t, _ := thinking["type"].(string); t == "enabled" {
				hasThinking = true
			}
		}
	}
	if hasThinking {
		parts = append(parts, "thinking")
	}
	if caps.HasVision() {
		parts = append(parts, "vision")
	}
	if caps.HasToolCalls() {
		parts = append(parts, "tool_calls")
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func hasExistingSession(ctx context.Context, computer infra.Computer, p platform, assistantID string) bool {
	workDir := computer.GetWorkDir()
	var sessionDir string
	if assistantID != "" {
		configDir := p.PathJoin(workDir, ".yao", "assistants", assistantID)
		sessionDir = p.PathJoin(configDir, "projects")
	} else {
		sessionDir = p.PathJoin(workDir, ".claude", "projects")
	}
	result, err := computer.Exec(ctx, p.ListDirCmd(sessionDir))
	if err != nil || result.ExitCode != 0 {
		return false
	}
	return strings.TrimSpace(result.Stdout) != ""
}

func buildMCPConfig(servers []types.MCPServer) []byte {
	mcpServers := make(map[string]any, len(servers))
	for _, s := range servers {
		name := s.ServerID
		if name == "" {
			continue
		}
		mcpServers[name] = map[string]any{
			"command": "tai",
			"args":    []string{"mcp", name},
		}
	}
	if len(mcpServers) == 0 {
		mcpServers["yao"] = map[string]any{
			"command": "tai",
			"args":    []string{"mcp"},
		}
	}
	config := map[string]any{"mcpServers": mcpServers}
	data, _ := json.Marshal(config)
	return data
}

func buildMCPAllowedTools(servers []types.MCPServer) string {
	patterns := make([]string, 0, len(servers))
	for _, s := range servers {
		if s.ServerID != "" {
			patterns = append(patterns, fmt.Sprintf("mcp__%s__*", s.ServerID))
		}
	}
	if len(patterns) == 0 {
		return "mcp__yao__*"
	}
	return strings.Join(patterns, ",")
}

func buildLastUserMessageJSONL(messages []agentContext.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			content := messages[i].Content
			if content == nil {
				content = ""
			}
			msg := map[string]any{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": content,
				},
			}
			data, _ := json.Marshal(msg)
			return string(data)
		}
	}
	return ""
}

// claudeRoleEnvMap maps abstract Yao model roles to Claude CLI environment
// variables. Only roles with matching Claude CLI env vars are listed here.
// ANTHROPIC_DEFAULT_SONNET_MODEL is set to the primary model in buildEnv.
var claudeRoleEnvMap = map[string]struct{ EnvVar string }{
	"default": {EnvVar: "ANTHROPIC_MODEL"},
	"heavy":   {EnvVar: "ANTHROPIC_DEFAULT_OPUS_MODEL"},
	"light":   {EnvVar: "ANTHROPIC_DEFAULT_HAIKU_MODEL"},
}

func connectorHost(c connector.Connector) string {
	if c == nil {
		return ""
	}
	if lc, ok := c.(goullm.LLMConnector); ok {
		if u := lc.GetURL(); u != "" {
			return u
		}
	}
	host, _ := c.Setting()["host"].(string)
	return host
}

func connectorProtocols(c connector.Connector) []string {
	if c == nil {
		return nil
	}
	setting := c.Setting()
	if ps, ok := setting["protocols"].([]string); ok && len(ps) > 0 {
		return ps
	}
	if c.Is(connector.ANTHROPIC) {
		return []string{"anthropic"}
	}
	return []string{"openai"}
}

func supportsProtocol(c connector.Connector, proto string) bool {
	for _, p := range connectorProtocols(c) {
		if p == proto {
			return true
		}
	}
	return false
}

var claudeArgWhitelist = map[string]string{
	"max_turns":        "--max-turns",
	"disallowed_tools": "--disallowed-tools",
	"allowed_tools":    "--allowedTools",
}

func isStandardAnthropicModel(model string) bool {
	return strings.HasPrefix(model, "claude-") || strings.HasPrefix(model, "anthropic.")
}

func buildClaudeCodeCapabilities(conn connector.Connector) string {
	if conn == nil {
		return ""
	}
	setting := conn.Setting()
	if setting == nil {
		return ""
	}
	var caps []string
	if thinking, ok := setting["thinking"].(map[string]interface{}); ok {
		if thinkType, _ := thinking["type"].(string); thinkType == "enabled" {
			caps = append(caps, "thinking")
		}
	}
	return strings.Join(caps, ",")
}

func setAnthropicModelEnv(env map[string]string, host, key, model string, conn connector.Connector) {
	env["ANTHROPIC_BASE_URL"] = host
	env["ANTHROPIC_API_KEY"] = key
	if model == "" {
		return
	}
	env["ANTHROPIC_MODEL"] = model
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model

	if isStandardAnthropicModel(model) {
		return
	}
	caps := buildClaudeCodeCapabilities(conn)
	env["ANTHROPIC_CUSTOM_MODEL_OPTION"] = model
	env["ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"] = model
	env["ANTHROPIC_DEFAULT_OPUS_MODEL_NAME"] = model
	env["ANTHROPIC_DEFAULT_SONNET_MODEL_NAME"] = model
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME"] = model
	if caps != "" {
		env["ANTHROPIC_CUSTOM_MODEL_OPTION_SUPPORTED_CAPABILITIES"] = caps
		env["ANTHROPIC_DEFAULT_OPUS_MODEL_SUPPORTED_CAPABILITIES"] = caps
		env["ANTHROPIC_DEFAULT_SONNET_MODEL_SUPPORTED_CAPABILITIES"] = caps
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL_SUPPORTED_CAPABILITIES"] = caps
	}
}

func applyAnthropicRoleOverrides(
	env map[string]string,
	primaryHost string,
	roles map[string]connector.Connector,
) {
	for role, rm := range claudeRoleEnvMap {
		if role == "default" {
			continue
		}
		rc, ok := roles[role]
		if !ok || rc == nil {
			continue
		}
		roleHost := connectorHost(rc)
		if roleHost != primaryHost {
			log.Warn("[claude] role %s: host mismatch (%s != %s), falling back to primary", role, roleHost, primaryHost)
			continue
		}
		if !supportsProtocol(rc, "anthropic") {
			log.Warn("[claude] role %s: not anthropic protocol, falling back to primary", role)
			continue
		}
		rcModel, _ := rc.Setting()["model"].(string)
		if rcModel == "" {
			continue
		}
		env[rm.EnvVar] = rcModel
		if isStandardAnthropicModel(rcModel) {
			continue
		}
		env[rm.EnvVar+"_NAME"] = rcModel
		if caps := buildClaudeCodeCapabilities(rc); caps != "" {
			env[rm.EnvVar+"_SUPPORTED_CAPABILITIES"] = caps
		}
	}
}

func setA2OModelEnv(env map[string]string, connectorID, model string, conn connector.Connector, a2oPort int) {
	port := a2oPort
	if port <= 0 {
		port = defaultA2OPort
	}
	env["ANTHROPIC_BASE_URL"] = fmt.Sprintf("http://127.0.0.1:%d/c/%s", port, connectorID)
	env["ANTHROPIC_API_KEY"] = "dummy"

	// Baseline: all tiers point to the "default" a2o route.
	// applyA2ORoleOverrides will override heavy/light if configured.
	env["ANTHROPIC_MODEL"] = "default"
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "default"
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "default"
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "default"

	env["ANTHROPIC_CUSTOM_MODEL_OPTION"] = "default"
	env["ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"] = model
	env["ANTHROPIC_DEFAULT_OPUS_MODEL_NAME"] = model
	env["ANTHROPIC_DEFAULT_SONNET_MODEL_NAME"] = model
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME"] = model

	if caps := buildClaudeCodeCapabilities(conn); caps != "" {
		env["ANTHROPIC_CUSTOM_MODEL_OPTION_SUPPORTED_CAPABILITIES"] = caps
		env["ANTHROPIC_DEFAULT_OPUS_MODEL_SUPPORTED_CAPABILITIES"] = caps
		env["ANTHROPIC_DEFAULT_SONNET_MODEL_SUPPORTED_CAPABILITIES"] = caps
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL_SUPPORTED_CAPABILITIES"] = caps
	}
}

func applyA2ORoleOverrides(
	env map[string]string,
	roles map[string]connector.Connector,
) {
	for role, rm := range claudeRoleEnvMap {
		if role == "default" {
			continue
		}
		rc, ok := roles[role]
		if !ok || rc == nil {
			continue
		}

		rcModel := connectorModel(rc)
		if rcModel == "" {
			continue
		}
		if connectorHost(rc) == "" {
			continue
		}

		// Set model name to role key — a2o routes by this key
		env[rm.EnvVar] = role
		env[rm.EnvVar+"_NAME"] = rcModel
		if caps := buildClaudeCodeCapabilities(rc); caps != "" {
			env[rm.EnvVar+"_SUPPORTED_CAPABILITIES"] = caps
		}
	}
}
