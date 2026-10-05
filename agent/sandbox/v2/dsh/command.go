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
	"github.com/yaoapp/yao/llmprovider"
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

	// Extract primary connector profile (connector vocab → pi-ai vocab)
	primary := llmprovider.ExtractProfile(req.Connector)
	if primary.Model == "" {
		primary.Model = "deepseek-chat"
	}

	cfg := &ConnectorConfig{
		IsWindows: p.OS() == "windows",
	}

	cfg.PiAiRoutes = append(cfg.PiAiRoutes, profileToPiAiRoute(primary, "yao-primary", "DSH_KEY_PRIMARY"))

	// Build pi-ai routes for all roles (each role gets its own route)
	for roleName, c := range req.Roles {
		if c == nil || roleName == "default" {
			continue
		}
		role := llmprovider.ExtractProfile(c)
		if role.Model == "" {
			continue
		}

		envKey := "DSH_KEY_" + strings.ToUpper(roleName)
		routeName := "yao-" + roleName
		cfg.PiAiRoutes = append(cfg.PiAiRoutes, profileToPiAiRoute(role, routeName, envKey))
	}

	// Determine vision from all models (primary + roles)
	vision := hasVisionInput(primary.Input)
	if !vision {
		for _, c := range req.Roles {
			if c == nil {
				continue
			}
			role := llmprovider.ExtractProfile(c)
			if hasVisionInput(role.Input) {
				vision = true
				break
			}
		}
	}
	cfg.Vision = vision
	cfg.MaxInstructionBytes = resolveMaxInstructionBytes(primary.ContextWindow)

	// Render cordis.yml
	cordisYAML, err := RenderCordisConfig(cfg)
	if err != nil {
		return command{}, fmt.Errorf("render cordis config: %w", err)
	}

	// debugDumpCordis(cordisYAML)

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
	initMsg, err := buildInitializeMsg(workDir, "yao-primary", primary.Model, primary.MaxTokens)
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
	env := buildEnv(req, p, workDir, primary.APIKey, primary.BaseURL, systemPrompt)

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
	if req.ChatID != "" {
		env["CTX_CHAT_ID"] = req.ChatID
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

// resolveMaxInstructionBytes computes the dsh-agent-instructions maxBytes
// budget based on the model's context window. DSH official presets use 65536
// (64KB) as the standard baseline; we scale up for larger context models
// and use the official value as the floor.
func resolveMaxInstructionBytes(contextWindow int) int {
	switch {
	case contextWindow >= 500_000:
		return 131072 // 128KB — 1M context models (DeepSeek, Gemini)
	default:
		return 65536 // 64KB — DSH official preset baseline
	}
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

// profileToPiAiRoute converts a ConnectorProfile (already in pi-ai vocabulary)
// to a PiAiRoute with DSH-specific URL normalization.
func profileToPiAiRoute(p llmprovider.ConnectorProfile, name, apiKeyEnv string) PiAiRoute {
	return PiAiRoute{
		Name:            name,
		API:             p.API,
		BaseURL:         normalizeBaseURL(p.API, p.BaseURL),
		APIKeyEnv:       apiKeyEnv,
		Reasoning:       p.Reasoning,
		BudgetTokens:    p.BudgetTokens,
		NoDeveloperRole: p.NoDeveloperRole,
		ThinkingFormat:  p.ThinkingFormat,
		Models: []PiAiModelConfig{{
			ID:               p.Model,
			Input:            p.Input,
			ContextWindow:    p.ContextWindow,
			MaxTokens:        p.MaxTokens,
			ReasoningEfforts: p.ReasoningEfforts,
			DisableReasoning: p.DisableReasoning,
		}},
	}
}

// normalizeBaseURL applies DSH-specific URL normalization based on the wire protocol.
func normalizeBaseURL(api, baseURL string) string {
	if api == llmprovider.APIAnthropicMessages {
		return strings.TrimSuffix(baseURL, "/")
	}
	return connector.BuildAPIURL(baseURL, "")
}

// hasVisionInput reports whether the input modality list includes "image".
func hasVisionInput(input []string) bool {
	for _, m := range input {
		if m == "image" {
			return true
		}
	}
	return false
}

// debugDumpCordis prints the pi-ai section of cordis.yml to stderr for diagnostics.
// Commented out in normal operation; uncomment the call in buildCommand to enable.
//
// func debugDumpCordis(yaml []byte) {
// 	s := string(yaml)
// 	end := strings.Index(s, "- id: attachment")
// 	if end < 0 {
// 		end = strings.Index(s, "- id: subprocess")
// 	}
// 	if end < 0 && len(s) > 600 {
// 		end = 600
// 	}
// 	if end > 0 {
// 		fmt.Fprintf(os.Stderr, "\n[buildCommand] cordis.yml pi-ai:\n%s\n", s[:end])
// 	}
// }

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
