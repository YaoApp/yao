package dsh

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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

	// Resolve primary connector settings
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

	// Detect connector type and select wire protocol.
	// Anthropic connectors → anthropic-messages.
	// OpenAI/unknown connectors → openai-completions (pi-ai detectCompat handles wire details).
	isAnthropic := req.Connector != nil && req.Connector.Is(connector.ANTHROPIC)

	var primaryAPI string
	var primaryBaseURL string
	if isAnthropic {
		primaryAPI = "anthropic-messages"
		primaryBaseURL = normalizeAnthropicBaseURL(baseURL)
	} else {
		primaryAPI = "openai-completions"
		primaryBaseURL = normalizePiAiBaseURL(baseURL)
	}

	primaryProvider := "yao-primary"
	cfg := &ConnectorConfig{
		IsWindows: p.OS() == "windows",
	}

	primaryInfo := extractReasoningInfo(req.Connector)

	reasoning, budgetTokens := toPiAiThinking(primaryInfo)

	// Reasoning-capable models default to "high" when no explicit level is set,
	// so pi-ai declares model.reasoning=true and the Off toggle works.
	if primaryInfo.hasReasoning && reasoning == "" {
		reasoning = "high"
	}

	input := []string{"text"}
	if connectorHasVision(req.Connector) {
		input = []string{"text", "image"}
	}
	cfg.PiAiRoutes = append(cfg.PiAiRoutes, PiAiRoute{
		Name:            "yao-primary",
		API:             primaryAPI,
		BaseURL:         primaryBaseURL,
		APIKeyEnv:       "DSH_KEY_PRIMARY",
		Models:          []PiAiModelConfig{{ID: model, Input: input, Reasoning: primaryInfo.hasReasoning || reasoning != ""}},
		Reasoning:       reasoning,
		BudgetTokens:    budgetTokens,
		NoDeveloperRole: !isAnthropic,
		ThinkingFormat:  nonAnthropicThinkingFormat(isAnthropic, req.Connector),
	})

	// Build pi-ai routes for all roles (each role gets its own route)
	for roleName, c := range req.Roles {
		if c == nil {
			continue
		}
		lc, ok := c.(goullm.LLMConnector)
		if !ok {
			continue
		}
		roleModel := lc.GetModel()
		if roleModel == "" {
			continue
		}

		// Skip "default" role — already handled as primary
		if roleName == "default" {
			continue
		}

		roleInfo := extractReasoningInfo(c)
		roleReasoning, roleBudgetTokens := toPiAiThinking(roleInfo)
		input := []string{"text"}
		if caps := lc.GetCapabilities(); caps != nil && caps.HasVision() {
			input = []string{"text", "image"}
		}

		// Select wire protocol based on connector type
		roleIsAnthropic := c.Is(connector.ANTHROPIC)
		var api string
		var roleBaseURL string
		if roleIsAnthropic {
			api = "anthropic-messages"
			roleBaseURL = normalizeAnthropicBaseURL(lc.GetURL())
		} else {
			api = "openai-completions"
			roleBaseURL = normalizePiAiBaseURL(lc.GetURL())
		}

		if roleInfo.hasReasoning && roleReasoning == "" {
			roleReasoning = "high"
		}

		envKey := "DSH_KEY_" + strings.ToUpper(roleName)
		routeName := "yao-" + roleName

		cfg.PiAiRoutes = append(cfg.PiAiRoutes, PiAiRoute{
			Name:            routeName,
			API:             api,
			BaseURL:         roleBaseURL,
			APIKeyEnv:       envKey,
			Models:          []PiAiModelConfig{{ID: roleModel, Input: input, Reasoning: roleInfo.hasReasoning || roleReasoning != ""}},
			Reasoning:       roleReasoning,
			BudgetTokens:    roleBudgetTokens,
			NoDeveloperRole: !roleIsAnthropic,
			ThinkingFormat:  nonAnthropicThinkingFormat(roleIsAnthropic, c),
		})
	}

	// Determine vision from all models (primary + roles)
	vision := connectorHasVision(req.Connector)
	if !vision {
		for _, c := range req.Roles {
			if connectorHasVision(c) {
				vision = true
				break
			}
		}
	}
	cfg.Vision = vision

	// Render cordis.yml
	cordisYAML, err := RenderCordisConfig(cfg)
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

	// Pre-write .anonymous-user-id so all workspaces share the same DSH user_id.
	// DSH reads this file during initialize; writing before process launch is required.
	if req.ClientID != "" {
		dshHome := filepath.Join(workDir, ".dsh")
		os.MkdirAll(dshHome, 0755)
		idFile := filepath.Join(dshHome, ".anonymous-user-id")
		instanceUUID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(req.ClientID)).String()
		os.WriteFile(idFile, []byte(instanceUUID+"\n"), 0644)
	}

	// Build JSON-RPC input
	initMsg, err := buildInitializeMsg(workDir, primaryProvider, model, maxTokens)
	if err != nil {
		return command{}, err
	}

	// Enrich context vars with runner-local paths before building prefix
	if req.ContextVars == nil {
		req.ContextVars = make(map[string]string)
	}
	skillsPrefix := ".dsh"
	if req.AssistantID != "" {
		skillsPrefix = ".yao/assistants/" + req.AssistantID
	}
	req.ContextVars["SKILLS_DIR"] = filepath.Join(workDir, skillsPrefix, "skills")
	req.ContextVars["EXT_SKILLS_DIR"] = filepath.Join(workDir, ".yao", "skills")

	ctxPrefix := shared.BuildContextPrefix(req.ContextVars)
	var promptMsg string
	if msgParts != nil && len(msgParts.ImageBlocks) > 0 {
		msgParts.TextParts = append([]string{ctxPrefix}, msgParts.TextParts...)
		blocks := buildContentBlocks(msgParts)
		promptMsg, err = buildSessionPromptMsgFromBlocks(sessionID, blocks)
	} else {
		lastMsg := extractLastUserMessage(req.Messages)
		lastMsg = ctxPrefix + "\n\n" + lastMsg
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

	// DSH-specific — DEEPSEEK_API_KEY always set for backward compatibility
	env["DEEPSEEK_API_KEY"] = apiKey
	if baseURL != "" {
		env["DEEPSEEK_BASE_URL"] = baseURL
	}

	// Per-role API keys for pi-ai routes
	env["DSH_KEY_PRIMARY"] = apiKey
	for roleName, c := range req.Roles {
		if c == nil || roleName == "default" {
			continue
		}
		if lc, ok := c.(goullm.LLMConnector); ok {
			envName := "DSH_KEY_" + strings.ToUpper(roleName)
			env[envName] = lc.GetKey()
		}
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

	return strings.Join(parts, "\n\n")
}

func buildSandboxEnvPrompt(workDir string) string {
	return fmt.Sprintf("Working directory: %s", workDir)
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

// reasoningInfo holds extracted thinking/reasoning configuration from a connector.
// Populated by extractReasoningInfo using a three-level priority chain.
type reasoningInfo struct {
	effort       string // "none"/"thinking"/"low"/"medium"/"high"/"max"/"xhigh"/""
	budgetTokens int    // from Setting()["thinking"]["budget_tokens"]
	thinkingType string // from Setting()["thinking"]["type"]: "enabled"/"disabled"/"adaptive"/""
	hasReasoning bool   // from GetCapabilities().Reasoning
}

// extractReasoningInfo reads thinking/reasoning configuration from a connector
// using a three-level priority chain:
//  1. Setting()["reasoning_effort"] — llmprovider path via ExtraBody promotion
//  2. GetMetadata()["reasoning_effort"] — metadata backup (same source, independent path)
//  3. Setting()["thinking"] — legacy .conn.yao compatibility
//
// "none" is a definitive value meaning "reasoning disabled", not an absence
// indicator. When priority 1 or 2 returns "none", no further fallback occurs.
func extractReasoningInfo(c connector.Connector) reasoningInfo {
	if c == nil {
		return reasoningInfo{}
	}

	info := reasoningInfo{}

	// budgetTokens always from Setting()["thinking"]["budget_tokens"]
	settings := c.Setting()
	if settings != nil {
		if thinking, ok := settings["thinking"].(map[string]interface{}); ok {
			if t, ok := thinking["type"].(string); ok {
				info.thinkingType = t
			}
			if bt, ok := thinking["budget_tokens"].(float64); ok {
				info.budgetTokens = int(bt)
			} else if bt, ok := thinking["budget_tokens"].(int); ok {
				info.budgetTokens = bt
			}
		}
	}

	// hasReasoning from capabilities
	if lc, ok := c.(goullm.LLMConnector); ok {
		if caps := lc.GetCapabilities(); caps != nil {
			info.hasReasoning = caps.Reasoning
		}
	}

	// Priority 1: Setting()["reasoning_effort"]
	if settings != nil {
		if v, ok := settings["reasoning_effort"].(string); ok && v != "" {
			info.effort = v
			return info
		}
	}

	// Priority 2: GetMetadata()["reasoning_effort"]
	if meta := c.GetMetadata(); meta != nil {
		if v, ok := meta["reasoning_effort"].(string); ok && v != "" {
			info.effort = v
			return info
		}
	}

	// Priority 3: Setting()["thinking"] — legacy fallback, effort stays ""
	return info
}

// toPiAiThinking converts reasoningInfo to dsh-llm-pi-ai config values.
// Returns (reasoning, budgetTokens) for the cordis.yml template.
// Pi-ai vocabulary: "off"|"minimal"|"low"|"medium"|"high"|"xhigh"|"max",
// or empty string to omit (let pi-ai decide from its model catalog).
func toPiAiThinking(info reasoningInfo) (string, int) {
	// Effort has value (priority 1/2 matched)
	if info.effort != "" {
		switch info.effort {
		case "none":
			return "off", 0
		case "thinking":
			return "high", info.budgetTokens
		default:
			return normalizePiAiReasoning(info.effort), info.budgetTokens
		}
	}

	// Effort empty — fallback to legacy priority 3
	switch info.thinkingType {
	case "disabled":
		return "off", 0
	case "enabled", "adaptive":
		return "high", info.budgetTokens
	}
	if info.hasReasoning {
		return "high", 0
	}
	return "", 0 // omit — no preference, let pi-ai decide from model catalog
}

// normalizePiAiReasoning maps a reasoning_effort string to a valid pi-ai
// ModelThinkingLevel. "off"/"none" explicitly disable reasoning; empty string
// omits the field (pi-ai decides from its model catalog).
func normalizePiAiReasoning(v string) string {
	switch v {
	case "off", "none":
		return "off"
	case "":
		return ""
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "xhigh":
		return "xhigh"
	case "max":
		return "max"
	default:
		return "high"
	}
}

// normalizePiAiBaseURL builds the pi-ai base URL from a connector host using
// the same convention as gou/connector.BuildAPIURL: trailing "/" means the user
// specified a full base path; otherwise "/v1" is appended automatically.
func normalizePiAiBaseURL(u string) string {
	return connector.BuildAPIURL(u, "")
}

// normalizeAnthropicBaseURL trims trailing slashes for Anthropic-protocol routes.
// pi-ai's anthropic-messages API type appends the correct path itself.
func normalizeAnthropicBaseURL(u string) string {
	return strings.TrimSuffix(u, "/")
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

// nonAnthropicThinkingFormat returns connectorThinkingFormat for non-Anthropic
// routes. Anthropic routes use pi-ai's native anthropic-messages thinking
// handling; thinkingFormat (an openai-completions compat field) does not apply.
func nonAnthropicThinkingFormat(isAnthropic bool, c connector.Connector) string {
	if isAnthropic {
		return ""
	}
	return connectorThinkingFormat(c)
}

// connectorThinkingFormat reads the thinking wire format from a connector.
// Returns a format string (e.g. "deepseek") for pi-ai's compat.thinkingFormat,
// or "" to let pi-ai detect the format from the endpoint URL.
//
// Detection:
//  1. metadata["thinking_format"] — explicit declaration
//  2. settings["thinking"]["type"] present — connector speaks the thinking.type
//     wire protocol (used by DeepSeek, Moonshot, and compatible providers)
func connectorThinkingFormat(c connector.Connector) string {
	if c == nil {
		return ""
	}
	if meta := c.GetMetadata(); meta != nil {
		if v, ok := meta["thinking_format"].(string); ok && v != "" {
			return v
		}
	}
	if settings := c.Setting(); settings != nil {
		if thinking, ok := settings["thinking"].(map[string]interface{}); ok {
			if _, hasType := thinking["type"]; hasType {
				return "deepseek"
			}
		}
	}
	return ""
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
