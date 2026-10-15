package tai

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/llmprovider"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// command holds the fully resolved subprocess invocation for tai agent run.
type command struct {
	shell     []string
	env       map[string]string
	workDir   string
	stdinData []byte // JSON image blocks to write to tai's stdin (nil = no images)
}

// buildCommand constructs the "tai agent run --format jsonl" command and
// its environment. The API key is passed exclusively via env for security.
// msgParts carries extracted text+image content; when images are present they
// are serialized as JSON and written to tai's stdin.
func buildCommand(req *types.StreamRequest, workDir string, msgParts *shared.MessageParts) command {
	primary := llmprovider.ExtractProfile(req.Connector)
	if primary.Model == "" {
		primary.Model = "deepseek-chat"
	}

	assistantID := req.AssistantID
	prefix := ".tai"
	if assistantID != "" {
		prefix = ".yao/assistants/" + assistantID
	}

	// Build context vars prefix.
	if req.ContextVars == nil {
		req.ContextVars = make(map[string]string)
	}
	req.ContextVars["SKILLS_DIR"] = filepath.Join(workDir, prefix, "skills")
	req.ContextVars["EXT_SKILLS_DIR"] = filepath.Join(workDir, ".yao", "skills")
	ctxPrefix := shared.BuildContextPrefix(req.ContextVars)

	userMsg := extractLastUserMessage(req.Messages)
	if ctxPrefix != "" {
		userMsg = ctxPrefix + "\n\n" + userMsg
	}

	// If msgParts has images, prepend context to text parts and override userMsg.
	if msgParts != nil && len(msgParts.ImageBlocks) > 0 {
		if ctxPrefix != "" {
			msgParts.TextParts = append([]string{ctxPrefix}, msgParts.TextParts...)
		}
		userMsg = strings.Join(msgParts.TextParts, "\n\n")
	}

	systemPrompt := buildSystemPrompt(req, workDir)

	args := []string{
		"agent", "run",
		"--format", "jsonl",
		"--model", primary.Model,
		"--work-dir", workDir,
	}

	if req.ChatID != "" {
		// tai session IDs must start/end with alphanumeric characters;
		// yao chat IDs may start with '-', so prefix to guarantee validity.
		sid := sanitizeSessionID(req.ChatID)
		args = append(args, "--session", sid)
	}

	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}

	if primary.BaseURL != "" {
		args = append(args, "--base-url", primary.BaseURL)
	}

	if primary.API != "" {
		args = append(args, "--api", primary.API)
	}

	providerID := inferProviderID(primary)
	if providerID != "" {
		args = append(args, "--provider", providerID)
	}

	args = append(args, userMsg)

	// Serialize image blocks for stdin if present.
	var stdinData []byte
	if msgParts != nil && len(msgParts.ImageBlocks) > 0 {
		args = append(args, "--images-stdin")
		type imagePayload struct {
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
		}
		var images []imagePayload
		for _, img := range msgParts.ImageBlocks {
			images = append(images, imagePayload{
				MediaType: img.MediaType,
				Data:      img.Data,
			})
		}
		stdinData, _ = json.Marshal(images)
	}

	env := buildEnv(req, primary, workDir)

	return command{
		shell:     append([]string{"tai"}, args...),
		env:       env,
		workDir:   workDir,
		stdinData: stdinData,
	}
}

// buildEnv assembles environment variables for the tai subprocess.
// Aligned with DSH buildEnv to provide the full context tai tools need.
func buildEnv(req *types.StreamRequest, primary llmprovider.ConnectorProfile, workDir string) map[string]string {
	env := map[string]string{
		"DEEPSEEK_API_KEY": primary.APIKey,
	}
	if primary.BaseURL != "" {
		env["DEEPSEEK_BASE_URL"] = primary.BaseURL
	}

	// Sandbox context
	env["WORKDIR"] = workDir

	// Workspace identity
	if req.Computer != nil {
		if ws := req.Computer.Workplace(); ws != nil {
			if wsID, err := ws.GetID(); err == nil {
				env["CTX_WORKSPACE_ID"] = wsID
			}
			wsBase := filepath.Join(workDir, ".workspace")
			env["GIT_CONFIG_GLOBAL"] = filepath.Join(wsBase, "git", "config")
			env["GIT_SSH_COMMAND"] = fmt.Sprintf("ssh -F %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new", filepath.Join(wsBase, "ssh", "config"))
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
	if req.Token != nil && req.Token.Token != "" {
		env["YAO_TOKEN"] = req.Token.Token
	}
	if req.Token != nil && req.Token.RefreshToken != "" {
		env["YAO_REFRESH_TOKEN"] = req.Token.RefreshToken
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

	// CTX_* context variables for tool integration.
	for k, v := range req.ContextVars {
		env["CTX_"+strings.ToUpper(k)] = v
	}

	return env
}

func hashUserID(id string) string {
	h := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", h[:8])
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func inferProviderID(p llmprovider.ConnectorProfile) string {
	u := strings.ToLower(p.BaseURL)
	switch {
	case strings.Contains(u, "deepseek.com"):
		return "deepseek"
	case strings.Contains(u, "moonshot.cn"):
		return "moonshotai"
	case strings.Contains(u, "openai.com"):
		return "openai"
	case strings.Contains(u, "anthropic.com"):
		return "anthropic"
	case strings.Contains(u, "openrouter.ai"):
		return "openrouter"
	}
	return ""
}

func buildSystemPrompt(req *types.StreamRequest, workDir string) string {
	var parts []string
	if req.SystemPrompt != "" {
		parts = append(parts, req.SystemPrompt)
	}
	parts = append(parts, fmt.Sprintf(
		"Working directory: %s\nAll project files and attachments (.attachments/) are inside this directory.\n"+
			"A file-write sandbox is active: write operations outside the workspace are denied by the OS.\n"+
			"Do not scan the entire filesystem.",
		workDir,
	))
	return strings.Join(parts, "\n\n")
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
	case []agentContext.ContentPart:
		var texts []string
		for _, part := range v {
			if part.Type == agentContext.ContentText && part.Text != "" {
				texts = append(texts, part.Text)
			}
		}
		return strings.Join(texts, "\n\n")
	}
	return fmt.Sprintf("%v", last.Content)
}

// sanitizeSessionID ensures the ID conforms to tai's session ID rules:
// alphanumeric, '-', '_', '.' only; must start and end with alphanumeric.
func sanitizeSessionID(id string) string {
	var b strings.Builder
	for _, c := range id {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' {
			b.WriteRune(c)
		}
	}
	s := b.String()
	s = strings.TrimLeft(s, "-_.")
	s = strings.TrimRight(s, "-_.")
	if s == "" {
		s = "s0"
	}
	return s
}
