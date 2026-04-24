package opencode

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/str"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

var (
	yaoSessionNS = uuid.MustParse("e37bc21a-72dd-4a8f-b567-1f02c3d4e590")
	safeNameRe   = regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)
)

type command struct {
	shell   []string
	env     map[string]string
	stdin   string
	workDir string
}

func chatIDToSessionID(assistantID, chatID string) string {
	return uuid.NewSHA1(yaoSessionNS, []byte(assistantID+":"+chatID)).String()
}

func sanitizeSessionName(chatID string) string {
	return "yao-oc-" + safeNameRe.ReplaceAllString(chatID, "_")
}

func chatSessionExists(storeKey string) bool {
	s, err := store.Get("__yao.store")
	if err != nil {
		return false
	}
	return s.Has(storeKey)
}

func markChatSession(storeKey, sessionID string, ttl time.Duration) {
	s, err := store.Get("__yao.store")
	if err != nil {
		return
	}
	s.Set(storeKey, sessionID, ttl)
}

func (r *Runner) buildCommand(req *types.StreamRequest, p platform, attachmentPaths []string) command {
	workDir := req.Computer.GetWorkDir()
	assistantID := req.AssistantID
	chatID := req.ChatID

	var isContinuation bool
	if chatID != "" {
		storeKey := "opencode-session:" + assistantID + ":" + chatID
		isContinuation = chatSessionExists(storeKey)
	}

	env := buildEnv(req, p)
	args := buildArgs(req, r, isContinuation, chatID)

	stdinMsg := buildStdinMessage(req.Messages, attachmentPaths)

	script := shellQuote("opencode", args...)

	return command{
		shell:   p.ShellCmd(script),
		env:     env,
		stdin:   stdinMsg,
		workDir: workDir,
	}
}

func buildEnv(req *types.StreamRequest, p platform) map[string]string {
	env := make(map[string]string)
	workDir := req.Computer.GetWorkDir()

	ws := req.Computer.Workplace()
	if ws != nil {
		workspaceID, err := ws.GetID()
		if err == nil {
			env["CTX_WORKSPACE_ID"] = workspaceID
		}
	}

	for k, v := range p.HomeEnv(workDir) {
		env[k] = v
	}
	env["WORKDIR"] = workDir

	assistantID := req.AssistantID
	prefix := p.PathJoin(workDir, ".yao", "assistants", assistantID, "opencode")
	if assistantID == "" {
		prefix = p.PathJoin(workDir, ".opencode-data")
	}
	if assistantID != "" {
		env["CTX_ASSISTANT_ID"] = assistantID
		env["CTX_SKILLS_DIR"] = p.PathJoin(workDir, ".yao", "assistants", assistantID, "skills")
	}

	env["OPENCODE_DATA_DIR"] = p.PathJoin(prefix, "data")
	env["OPENCODE_CACHE_DIR"] = p.PathJoin(prefix, "cache")
	env["OPENCODE_STATE_DIR"] = p.PathJoin(prefix, "state")
	env["OPENCODE_CONFIG_DIR"] = p.PathJoin(prefix, "config")

	env["OPENCODE_DISABLE_AUTOUPDATE"] = "true"
	env["OPENCODE_DISABLE_MODELS_FETCH"] = "true"
	env["OPENCODE_DISABLE_LSP_DOWNLOAD"] = "true"
	env["OPENCODE_DISABLE_DEFAULT_PLUGINS"] = "true"
	env["OPENCODE_DISABLE_TERMINAL_TITLE"] = "true"
	env["OPENCODE_DISABLE_MOUSE"] = "true"
	env["OPENCODE_DISABLE_CLAUDE_CODE"] = "true"
	env["OPENCODE_CLIENT"] = "cli"

	// Lower bash default timeout from 120s to 30s. Long-running commands
	// like browsers should be nohup'd; this prevents accidental 2-min hangs.
	env["OPENCODE_EXPERIMENTAL_BASH_DEFAULT_TIMEOUT_MS"] = "30000"

	if req.Connector != nil {
		setting := req.Connector.Setting()
		key, _ := setting["key"].(string)
		if key != "" {
			env["YAO_PROVIDER_KEY"] = key
		}

		if req.Connector.Is(connector.ANTHROPIC) {
			apiKey, _ := setting["key"].(string)
			if apiKey != "" {
				env["ANTHROPIC_API_KEY"] = apiKey
			}
		}
	}

	injectRoleEnvVars(env, req)

	if req.Config != nil && len(req.Config.Secrets) > 0 {
		for k, v := range req.Config.Secrets {
			env[k] = str.EnvVar(v)
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

	return env
}

func buildArgs(req *types.StreamRequest, r *Runner, isContinuation bool, chatID string) []string {
	args := []string{"run", "--format", "json"}

	permMode := ""
	if req.Config != nil && req.Config.Runner.Options != nil {
		if v, ok := req.Config.Runner.Options["permission_mode"]; ok {
			permMode = fmt.Sprintf("%v", v)
		}
	}
	if permMode == "bypassPermissions" {
		args = append(args, "--dangerously-skip-permissions")
	}

	if chatID != "" && isContinuation {
		sessionID := chatIDToSessionID(req.AssistantID, chatID)
		args = append(args, "--continue", "--session", sessionID)
	}

	if req.Connector != nil {
		if mid := connectorModelID(req.Connector); mid != "" {
			args = append(args, "--model", mid)
		}
	}

	// User message and attachments are passed via stdin (heredoc pipe),
	// NOT as positional args. This avoids shell escaping issues with
	// special characters, CJK text, long messages, and --file ambiguity.

	return args
}

// buildStdinMessage builds the text piped to `opencode run` via stdin.
// It combines the user's text message with attachment references so OpenCode
// receives everything through stdin — no positional args, no --file flags.
// This mirrors the Claude runner approach and avoids shell escaping pitfalls.
func buildStdinMessage(messages []agentContext.Message, attachmentPaths []string) string {
	var parts []string

	if len(attachmentPaths) > 0 {
		parts = append(parts, "The user has attached the following files — read them to understand context:")
		for _, p := range attachmentPaths {
			parts = append(parts, fmt.Sprintf("  - %s", p))
		}
		parts = append(parts, "")
	}

	text := lastUserText(messages)
	if text != "" {
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n")
}

// lastUserText extracts the plain text from the last user message,
// handling string, []ContentPart, and []any (generic JSON) content types.
func lastUserText(messages []agentContext.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		switch c := messages[i].Content.(type) {
		case string:
			return c
		case []agentContext.ContentPart:
			var texts []string
			for _, part := range c {
				if part.Type == agentContext.ContentText && part.Text != "" {
					texts = append(texts, part.Text)
				}
			}
			return strings.Join(texts, "\n")
		case []any:
			var texts []string
			for _, item := range c {
				if m, ok := item.(map[string]any); ok {
					if t, _ := m["type"].(string); t == "text" {
						if text, _ := m["text"].(string); text != "" {
							texts = append(texts, text)
						}
					}
				}
			}
			return strings.Join(texts, "\n")
		}
	}
	return ""
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

	return fmt.Sprintf(`## Sandbox Environment

- **Operating System**: %[2]s
- **Shell**: %[3]s
- **Working Directory**: %[1]s
- **File Access**: You have full read/write access to %[1]s

## User Attachments

User-uploaded files are placed in %[1]s/.attachments/{chatID}/
Each chat session has its own subdirectory.
When the user attaches files, their paths are listed at the top of the message.
**Read these files yourself** using the Read or Bash tool — they are NOT passed as CLI arguments.
`, workDir, osName, shell)
}

func getProviderPrefix(conn connector.Connector) string {
	if conn != nil && conn.Is(connector.ANTHROPIC) {
		return "anthropic"
	}
	return "openai"
}

// resolveRoleConnector determines which connector to use for a given role.
func resolveRoleConnector(
	role string,
	roleConnectors map[string]*types.RoleConnector,
	userExplicit bool,
	getConnector func(id string) connector.Connector,
) connector.Connector {
	rc, ok := roleConnectors[role]
	if !ok || rc == nil {
		return nil
	}
	if rc.Override == "user" && userExplicit {
		return nil
	}
	return getConnector(rc.Connector)
}

func getRoleConnectors(req *types.StreamRequest) map[string]*types.RoleConnector {
	if req.Config == nil {
		return nil
	}
	return req.Config.Runner.Connectors
}

// shellQuote builds a shell-safe command string from program and args.
func shellQuote(program string, args ...string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, program)
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'\\$`!#&|;(){}[]<>?*~") {
			parts = append(parts, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}

// connectorModelID returns the "provider/model" string matching the
// provider ID used in opencode.json (see buildProviderConfig).
func connectorModelID(c connector.Connector) string {
	setting := c.Setting()
	modelName, _ := setting["model"].(string)
	host, _ := setting["host"].(string)

	if c.Is(connector.ANTHROPIC) {
		return "anthropic/" + modelName
	}
	if host == "" || isNativeOpenAI(host) {
		return "openai/" + modelName
	}
	return "custom/" + modelName
}

// injectRoleEnvVars adds API key, base URL, and model environment variables
// for each role connector defined in openCodeRoleMap. These env vars are
// consumed by opencode.json provider blocks (via {env:...} references) and
// by the custom read.ts tool (for vision API calls).
func injectRoleEnvVars(env map[string]string, req *types.StreamRequest) {
	if req.Config == nil || req.Config.Runner.Connectors == nil {
		return
	}
	for role, spec := range openCodeRoleMap {
		if spec.EnvKeyPrefix == "" {
			continue
		}
		rc, ok := req.Config.Runner.Connectors[role]
		if !ok || rc == nil || rc.Connector == "" {
			continue
		}
		c, exists := connector.Connectors[rc.Connector]
		if !exists || c == nil {
			continue
		}
		setting := c.Setting()
		if key, _ := setting["key"].(string); key != "" {
			env[spec.EnvKeyPrefix+"_KEY"] = key
		}
		if host, _ := setting["host"].(string); host != "" {
			env[spec.EnvKeyPrefix+"_BASE_URL"] = normalizeBaseURL(host)
		}
		if model, _ := setting["model"].(string); model != "" {
			env[spec.EnvKeyPrefix+"_MODEL"] = model
		}
	}
}

func connectorHost(c connector.Connector) string {
	if c == nil {
		return ""
	}
	host, _ := c.Setting()["host"].(string)
	return strings.TrimSpace(host)
}
