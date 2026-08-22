package dsh

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

func hashUserID(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:8])
}

type command struct {
	shell     []string
	env       map[string]string
	stdin     []byte
	workDir   string
	sessionID string
}

func (r *Runner) buildCommand(req *types.StreamRequest, p platform, msgParts *shared.MessageParts) (command, error) {
	computer := req.Computer
	workDir := computer.GetWorkDir()

	// Resolve connector settings
	apiKey := ""
	baseURL := ""
	model := "deepseek-chat"
	maxTokens := 0

	if lc, ok := req.Connector.(goullm.LLMConnector); ok {
		apiKey = lc.GetKey()
		baseURL = lc.GetURL()
		if m := lc.GetModel(); m != "" {
			model = m
		}
		if caps := lc.GetCapabilities(); caps != nil {
			maxTokens = caps.MaxOutputTokens
		}
	} else if req.Connector != nil {
		apiKey = connectorSetting(req.Connector, "api_key")
		baseURL = connectorSetting(req.Connector, "base_url")
		if m := connectorSetting(req.Connector, "model"); m != "" {
			model = m
		}
	}

	// DSH appends /chat/completions to baseURL; strip protocol-specific suffixes
	baseURL = normalizeDSHBaseURL(baseURL)

	// Extract thinking configuration from nested connector setting map.
	// Connector stores thinking as {"type": "enabled"|"disabled", "budget_tokens": N}.
	thinking, reasoningEffort := extractThinkingConfig(req.Connector)

	// Collect all models on the same endpoint (primary + roles) for the DSH catalog.
	// DSH only recognizes capabilities declared in the catalog; uncatalogued models
	// are forced text-only regardless of actual support.
	var models []ModelConfig
	seen := map[string]bool{}

	if lc, ok := req.Connector.(goullm.LLMConnector); ok {
		modalities := []string{"text"}
		if caps := lc.GetCapabilities(); caps != nil && caps.HasVision() {
			modalities = []string{"text", "image"}
		}
		models = append(models, ModelConfig{ID: model, InputModalities: modalities})
		seen[model] = true
	} else if req.Connector != nil {
		models = append(models, ModelConfig{ID: model, InputModalities: []string{"text"}})
		seen[model] = true
	}

	for _, c := range req.Roles {
		if c == nil {
			continue
		}
		lc, ok := c.(goullm.LLMConnector)
		if !ok {
			continue
		}
		roleModel := lc.GetModel()
		if roleModel == "" || seen[roleModel] {
			continue
		}
		roleURL := normalizeDSHBaseURL(lc.GetURL())
		if roleURL != baseURL {
			continue
		}
		modalities := []string{"text"}
		if caps := lc.GetCapabilities(); caps != nil && caps.HasVision() {
			modalities = []string{"text", "image"}
		}
		models = append(models, ModelConfig{ID: roleModel, InputModalities: modalities})
		seen[roleModel] = true
	}

	vision := false
	for _, m := range models {
		for _, mod := range m.InputModalities {
			if mod == "image" {
				vision = true
				break
			}
		}
		if vision {
			break
		}
	}

	// Render cordis.yml
	cordisYAML, err := RenderCordisConfig(&ConnectorConfig{
		BaseURL:         baseURL,
		Thinking:        thinking,
		ReasoningEffort: reasoningEffort,
		MaxTokens:       maxTokens,
		IsWindows:       p.OS() == "windows",
		Models:          models,
		Vision:          vision,
	})
	if err != nil {
		return command{}, fmt.Errorf("render cordis config: %w", err)
	}

	// Build system prompt
	systemPrompt := buildSystemPrompt(req, workDir)

	// Use chatID directly as DSH session ID — same chat shares session across
	// assistant switches (as long as runner is DSH).
	sessionID := req.ChatID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Build JSON-RPC input
	initMsg, err := buildInitializeMsg(workDir, model, maxTokens)
	if err != nil {
		return command{}, err
	}

	var promptMsg string
	if msgParts != nil && len(msgParts.ImageBlocks) > 0 {
		blocks := buildContentBlocks(msgParts)
		promptMsg, err = buildSessionPromptMsgFromBlocks(sessionID, blocks)
	} else {
		lastMsg := extractLastUserMessage(req.Messages)
		promptMsg, err = buildSessionPromptMsg(sessionID, lastMsg)
	}
	if err != nil {
		return command{}, err
	}

	inputJSONRPC := initMsg + "\n" + promptMsg

	// Build environment
	env := buildEnv(req, p, workDir, apiKey, baseURL, systemPrompt)

	// Config file path (in workspace, accessible from Host and Box)
	configFile := p.PathJoin(workDir, ".yao", "dsh", "cordis.yml")

	// Build platform-specific script
	script, stdin := p.BuildScript(scriptInput{
		cordisYAML:   string(cordisYAML),
		configFile:   configFile,
		inputJSONRPC: inputJSONRPC,
	})

	return command{
		shell:     p.ShellCmd(script),
		env:       env,
		stdin:     stdin,
		workDir:   workDir,
		sessionID: sessionID,
	}, nil
}

func buildEnv(req *types.StreamRequest, p platform, workDir, apiKey, baseURL, systemPrompt string) map[string]string {
	env := make(map[string]string)

	// DSH-specific
	env["DEEPSEEK_API_KEY"] = apiKey
	if baseURL != "" {
		env["DEEPSEEK_BASE_URL"] = baseURL
	}
	env["DSH_CWD"] = workDir
	env["DSH_SESSION_ROOT"] = p.PathJoin(workDir, ".yao", "dsh", "sessions")
	env["DSH_SYSTEM_PROMPT"] = systemPrompt
	env["NODE_NO_WARNINGS"] = "1"

	// Sandbox context (aligned with Claude Runner)
	env["WORKDIR"] = workDir

	// Workspace identity: git config, SSH, and XDG for sandbox git operations.
	if req.Computer != nil {
		if ws := req.Computer.Workplace(); ws != nil {
			if wsID, err := ws.GetID(); err == nil {
				env["CTX_WORKSPACE_ID"] = wsID
			}
			wsBase := p.PathJoin(workDir, ".workspace")
			env["GIT_CONFIG_GLOBAL"] = p.PathJoin(wsBase, "git", "config")
			env["GIT_SSH_COMMAND"] = fmt.Sprintf("ssh -F %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new", p.PathJoin(wsBase, "ssh", "config"))
			env["XDG_CONFIG_HOME"] = wsBase
		}
	}
	if req.Config != nil && req.Config.WorkspaceID != "" {
		env["CTX_WORKSPACE_ID"] = req.Config.WorkspaceID
	}
	if req.AssistantID != "" {
		env["CTX_ASSISTANT_ID"] = req.AssistantID
	}
	if req.Locale != "" {
		env["CTX_LOCALE"] = req.Locale
	}

	// Skills directories
	prefix := ".dsh"
	if req.AssistantID != "" {
		prefix = ".yao/assistants/" + req.AssistantID
	}
	env["CTX_SKILLS_DIR"] = p.PathJoin(workDir, prefix, "skills")
	env["CTX_EXT_SKILLS_DIR"] = p.PathJoin(workDir, ".yao", "skills")

	// Home environment (platform-specific)
	for k, v := range p.HomeEnv(workDir) {
		env[k] = v
	}

	// Node identity for tai tool routing
	if req.Config != nil && req.Config.NodeID != "" {
		env["CTX_NODE_ID"] = req.Config.NodeID
		if req.Config.ID != "" {
			env["CTX_TARGET_ID"] = req.Config.ID
		} else {
			env["CTX_TARGET_ID"] = "__host__"
		}
	}

	// Auth tokens for tai tool callbacks
	if req.Token != nil {
		if req.Token.Token != "" {
			env["YAO_TOKEN"] = req.Token.Token
		}
		if req.Token.RefreshToken != "" {
			env["YAO_REFRESH_TOKEN"] = req.Token.RefreshToken
		}
	}

	// gRPC address for host-mode tai tool calls
	if req.Computer != nil && req.Computer.ComputerInfo().Kind == "host" {
		if addr := infra.ResolveHostGRPCAddr(req.Computer.ComputerInfo().NodeID); addr != "" {
			env["YAO_GRPC_ADDR"] = addr
		}
	}

	if req.Config != nil && req.Config.Owner != "" {
		env["CTX_OWNER_HASH"] = hashUserID(req.Config.Owner)
	}

	return env
}

func buildSystemPrompt(req *types.StreamRequest, workDir string) string {
	var parts []string

	if req.SystemPrompt != "" {
		parts = append(parts, req.SystemPrompt)
	}

	parts = append(parts, buildSandboxEnvPrompt(workDir))

	if req.Locale != "" {
		if lp := buildLocalePrompt(req.Locale); lp != "" {
			parts = append(parts, lp)
		}
	}

	return strings.Join(parts, "\n\n")
}

func buildSandboxEnvPrompt(workDir string) string {
	return fmt.Sprintf("Working directory: %s", workDir)
}

func buildLocalePrompt(locale string) string {
	switch {
	case strings.HasPrefix(locale, "zh"):
		return "Always respond in Chinese (Simplified)."
	default:
		return ""
	}
}

func extractLastUserMessage(messages []agentContext.Message) string {
	if len(messages) == 0 {
		return ""
	}
	last := messages[len(messages)-1]
	switch v := last.Content.(type) {
	case string:
		return v
	case []interface{}:
		var texts []string
		for _, part := range v {
			if pm, ok := part.(map[string]interface{}); ok {
				if text, ok := pm["text"].(string); ok {
					texts = append(texts, text)
				}
			}
		}
		return strings.Join(texts, "\n\n")
	}
	return fmt.Sprintf("%v", last.Content)
}

func connectorSetting(c connector.Connector, key string) string {
	if c == nil {
		return ""
	}
	settings := c.Setting()
	if settings == nil {
		return ""
	}
	if v, ok := settings[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractThinkingConfig reads the nested thinking map from connector settings
// and maps Yao connector values to DSH-compatible values.
//
// DSH llm-deepseek accepts:
//   - thinking: "enabled" | "disabled"
//   - reasoningEffort: "off" | "high" | "max"
//
// Yao connectors may set reasoning_effort to "low" or "medium", which DSH
// does not support; both map to "high" (DeepSeek API maps them server-side).
// When thinking is disabled, reasoningEffort must be "off".
func extractThinkingConfig(c connector.Connector) (string, string) {
	if c == nil {
		return "", ""
	}
	settings := c.Setting()
	if settings == nil {
		return "", ""
	}

	thinkingType := ""
	if thinking, ok := settings["thinking"].(map[string]interface{}); ok {
		if t, ok := thinking["type"].(string); ok {
			thinkingType = t
		}
	}

	if thinkingType == "disabled" {
		return "disabled", "off"
	}

	reasoningEffort := ""
	if v, ok := settings["reasoning_effort"].(string); ok {
		reasoningEffort = normalizeDSHReasoningEffort(v)
	}

	return thinkingType, reasoningEffort
}

// normalizeDSHReasoningEffort maps Yao reasoning_effort values to the
// subset DSH llm-deepseek accepts: "off", "high", or "max".
func normalizeDSHReasoningEffort(v string) string {
	switch v {
	case "max":
		return "max"
	case "off":
		return "off"
	case "high", "medium", "low":
		return "high"
	default:
		return ""
	}
}

// normalizeDSHBaseURL converts a Yao connector base URL to the format DSH expects.
// DSH appends /chat/completions directly, so the URL needs the /v1 prefix that
// standard OpenAI-compatible endpoints require. Yao's GetURL() returns the base
// without /v1; Anthropic-style suffixes are replaced.
func normalizeDSHBaseURL(u string) string {
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, "/anthropic")
	if !strings.HasSuffix(u, "/v1") {
		u += "/v1"
	}
	return u
}

func connectorHasVision(c connector.Connector) bool {
	if c == nil {
		return false
	}
	lc, ok := c.(goullm.LLMConnector)
	if !ok {
		return false
	}
	caps := lc.GetCapabilities()
	if caps == nil {
		return false
	}
	return caps.HasVision()
}

// buildContentBlocks creates mixed text+image content blocks from MessageParts.
func buildContentBlocks(parts *shared.MessageParts) []contentBlock {
	var blocks []contentBlock
	for _, text := range parts.TextParts {
		if text == "" {
			continue
		}
		blocks = append(blocks, contentBlock{Type: "text", Text: text})
	}
	for _, img := range parts.ImageBlocks {
		blocks = append(blocks, contentBlock{
			Type:      "image",
			MediaType: img.MediaType,
			Data:      img.Data,
			Name:      img.Filename,
		})
	}
	return blocks
}
