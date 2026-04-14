package claude

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/workspace"
)

func testPlatform() platform {
	return &darwinPlatform{posixBase: posixBase{
		os: "darwin", workDir: "/workspace", shell: "bash", tempDir: "/tmp",
	}}
}

// --- buildEnv ---

func TestBuildEnv_HomeEnv(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Equal(t, "/workspace", env["HOME"])
}

func TestBuildEnv_ConfigDirIsolation(t *testing.T) {
	req := &types.StreamRequest{
		Config:      &types.SandboxConfig{ID: "my-assistant"},
		AssistantID: "my-assistant",
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	expected := "/workspace/.yao/assistants/my-assistant"
	assert.Equal(t, expected, env["CLAUDE_CONFIG_DIR"])
}

func TestBuildEnv_NoAssistantID(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	_, hasConfigDir := env["CLAUDE_CONFIG_DIR"]
	assert.False(t, hasConfigDir, "should not set CLAUDE_CONFIG_DIR without assistantID")
}

func TestBuildEnv_Token(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Token: &types.SandboxToken{
			Token:        "tok123",
			RefreshToken: "ref456",
		},
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Equal(t, "tok123", env["YAO_TOKEN"])
	assert.Equal(t, "ref456", env["YAO_REFRESH_TOKEN"])
}

func TestBuildEnv_Secrets(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Secrets: map[string]string{
				"MY_SECRET": "secret_val",
			},
		},
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Equal(t, "secret_val", env["MY_SECRET"])
}

// --- buildArgs ---

func TestBuildArgs_Default(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{}
	p := testPlatform()

	args := buildArgs(req, r, p, false, "", "")
	assert.Contains(t, args, "--input-format")
	assert.Contains(t, args, "stream-json")
	assert.Contains(t, args, "--output-format")
	assert.Contains(t, args, "--verbose")
	assert.Contains(t, args, "--include-partial-messages")
}

func TestBuildArgs_Continuation(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{}
	p := testPlatform()

	args := buildArgs(req, r, p, true, "", "")
	assert.Contains(t, args, "--continue")
}

func TestBuildArgs_PermissionMode(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Runner: types.RunnerConfig{
				Options: map[string]interface{}{
					"permission_mode": "bypassPermissions",
				},
			},
		},
	}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{}
	p := testPlatform()

	args := buildArgs(req, r, p, false, "", "")
	assert.Contains(t, args, "--dangerously-skip-permissions")
	assert.Contains(t, args, "--permission-mode")
}

func TestBuildArgs_MCP(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{hasMCP: true, mcpToolPattern: "mcp__yao__*"}
	p := testPlatform()

	args := buildArgs(req, r, p, false, "test-assistant", "")
	assert.Contains(t, args, "--mcp-config")
	assert.Contains(t, args, "--allowedTools")
	assert.Contains(t, args, "mcp__yao__*")

	mcpIdx := -1
	for i, a := range args {
		if a == "--mcp-config" {
			mcpIdx = i
			break
		}
	}
	require.Greater(t, mcpIdx, -1)
	mcpPath := args[mcpIdx+1]
	assert.Contains(t, mcpPath, "test-assistant")
}

func TestBuildArgs_WhitelistOptions(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Runner: types.RunnerConfig{
				Options: map[string]interface{}{
					"max_turns": 10,
				},
			},
		},
	}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{}
	p := testPlatform()

	args := buildArgs(req, r, p, false, "", "")
	assert.Contains(t, args, "--max-turns")
}

// --- buildLastUserMessageJSONL (was buildInput / buildFirstRequestJSONL) ---

func TestBuildLastUserMessageJSONL_SkipsSystem(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello"},
	}
	result := buildLastUserMessageJSONL(msgs)
	var parsed map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(result)), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "user", parsed["type"])
	msg, _ := parsed["message"].(map[string]any)
	assert.Equal(t, "hello", msg["content"])
}

func TestBuildLastUserMessageJSONL_OnlyLastUser(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "user", Content: "q1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "q2"},
	}
	result := buildLastUserMessageJSONL(msgs)
	var parsed map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(result)), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "user", parsed["type"])
	msg, _ := parsed["message"].(map[string]any)
	assert.Equal(t, "q2", msg["content"])
	assert.NotContains(t, result, "q1")
}

func TestBuildLastUserMessageJSONL_NilContent(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "user", Content: nil},
	}
	result := buildLastUserMessageJSONL(msgs)
	var parsed map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(result)), &parsed)
	require.NoError(t, err)
	msg, _ := parsed["message"].(map[string]any)
	assert.Equal(t, "", msg["content"])
}

// --- buildLastUserMessageJSONL ---

func TestBuildLastUserMessageJSONL_Found(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "reply"},
		{Role: "user", Content: "second"},
	}
	result := buildLastUserMessageJSONL(msgs)
	var parsed map[string]any
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	msg, _ := parsed["message"].(map[string]any)
	assert.Equal(t, "second", msg["content"])
}

func TestBuildLastUserMessageJSONL_NoUser(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "assistant", Content: "only assistant"},
	}
	assert.Empty(t, buildLastUserMessageJSONL(msgs))
}

func TestBuildLastUserMessageJSONL_Empty(t *testing.T) {
	assert.Empty(t, buildLastUserMessageJSONL(nil))
}

// --- buildMCPConfig ---

func TestBuildMCPConfig_WithServers(t *testing.T) {
	servers := []types.MCPServer{
		{ServerID: "server1"},
		{ServerID: "server2"},
	}
	data := buildMCPConfig(servers)
	var cfg map[string]any
	err := json.Unmarshal(data, &cfg)
	require.NoError(t, err)
	mcpServers, ok := cfg["mcpServers"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, mcpServers, "server1")
	assert.Contains(t, mcpServers, "server2")
}

func TestBuildMCPConfig_Empty(t *testing.T) {
	data := buildMCPConfig(nil)
	var cfg map[string]any
	err := json.Unmarshal(data, &cfg)
	require.NoError(t, err)
	mcpServers, ok := cfg["mcpServers"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, mcpServers, "yao", "should default to yao server")
}

func TestBuildMCPConfig_EmptyServerID(t *testing.T) {
	servers := []types.MCPServer{{ServerID: ""}}
	data := buildMCPConfig(servers)
	var cfg map[string]any
	json.Unmarshal(data, &cfg)
	mcpServers, _ := cfg["mcpServers"].(map[string]any)
	assert.Contains(t, mcpServers, "yao", "empty serverID should fall back to default")
}

// --- buildMCPAllowedTools ---

func TestBuildMCPAllowedTools_WithServers(t *testing.T) {
	servers := []types.MCPServer{
		{ServerID: "s1"},
		{ServerID: "s2"},
	}
	result := buildMCPAllowedTools(servers)
	assert.Contains(t, result, "mcp__s1__*")
	assert.Contains(t, result, "mcp__s2__*")
	assert.Contains(t, result, ",")
}

func TestBuildMCPAllowedTools_Empty(t *testing.T) {
	assert.Equal(t, "mcp__yao__*", buildMCPAllowedTools(nil))
}

// --- buildSandboxEnvPrompt ---

func TestBuildSandboxEnvPrompt(t *testing.T) {
	p := testPlatform()
	prompt := buildSandboxEnvPrompt(p, "/workspace")
	assert.Contains(t, prompt, "/workspace")
	assert.Contains(t, prompt, "darwin")
	assert.Contains(t, prompt, "bash")
	assert.Contains(t, prompt, "Sandbox Environment")
	assert.Contains(t, prompt, ".attachments")
}

func TestBuildSandboxEnvPrompt_WindowsPlatform(t *testing.T) {
	p := newWindowsPlatform(`C:\ws`, "pwsh", "")
	prompt := buildSandboxEnvPrompt(p, `C:\ws`)
	assert.Contains(t, prompt, "windows")
	assert.Contains(t, prompt, "pwsh")
	assert.Contains(t, prompt, `C:\ws`)
	assert.Contains(t, prompt, "Windows desktop")
}

// --- fakeComputer implements infra.Computer for unit tests ---

type fakeComputer struct {
	workDir string
}

func newFakeComputer(workDir string) *fakeComputer {
	return &fakeComputer{workDir: workDir}
}

func (f *fakeComputer) GetWorkDir() string      { return f.workDir }
func (f *fakeComputer) BindWorkplace(string)    {}
func (f *fakeComputer) Workplace() workspace.FS { return nil }
func (f *fakeComputer) ComputerInfo() infra.ComputerInfo {
	return infra.ComputerInfo{System: infra.SystemInfo{OS: "linux", Shell: "bash"}}
}
func (f *fakeComputer) Exec(_ context.Context, _ []string, _ ...infra.ExecOption) (*infra.ExecResult, error) {
	return &infra.ExecResult{}, nil
}
func (f *fakeComputer) Stream(_ context.Context, _ []string, _ ...infra.ExecOption) (*infra.ExecStream, error) {
	return nil, nil
}
func (f *fakeComputer) VNC(_ context.Context) (string, error)                    { return "", nil }
func (f *fakeComputer) Proxy(_ context.Context, _ int, _ string) (string, error) { return "", nil }

// --- chatIDToSessionUUID ---

func TestChatIDToSessionUUID_Deterministic(t *testing.T) {
	u1 := chatIDToSessionUUID("asst-1", "robot_m1_e1")
	u2 := chatIDToSessionUUID("asst-1", "robot_m1_e1")
	assert.Equal(t, u1, u2, "same inputs should produce same UUID")
}

func TestChatIDToSessionUUID_DifferentAssistant(t *testing.T) {
	u1 := chatIDToSessionUUID("asst-1", "robot_m1_e1")
	u2 := chatIDToSessionUUID("asst-2", "robot_m1_e1")
	assert.NotEqual(t, u1, u2, "different assistantID should produce different UUID")
}

func TestChatIDToSessionUUID_ValidFormat(t *testing.T) {
	u := chatIDToSessionUUID("asst-1", "robot_m1_e1")
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, u)
}

// --- sanitizeSessionName ---

func TestSanitizeSessionName_Normal(t *testing.T) {
	assert.Equal(t, "yao-robot_m1_e1", sanitizeSessionName("robot_m1_e1"))
}

func TestSanitizeSessionName_SpecialChars(t *testing.T) {
	assert.Equal(t, "yao-user_s__chat_", sanitizeSessionName("user's \"chat\""))
}

func TestSanitizeSessionName_Empty(t *testing.T) {
	assert.Equal(t, "yao-", sanitizeSessionName(""))
}

// --- buildArgs with session ---

func TestBuildArgs_SessionID_NewSession(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{}
	p := testPlatform()

	args := buildArgs(req, r, p, false, "asst-1", "robot_m1_e1")
	assert.Contains(t, args, "--session-id")
	assert.Contains(t, args, "--name")
	assert.Contains(t, args, "yao-robot_m1_e1")
	assert.NotContains(t, args, "--resume")
	assert.NotContains(t, args, "--continue")

	sidIdx := -1
	for i, a := range args {
		if a == "--session-id" {
			sidIdx = i
			break
		}
	}
	require.Greater(t, sidIdx, -1)
	assert.Regexp(t, `^[0-9a-f]{8}-`, args[sidIdx+1])
}

func TestBuildArgs_SessionID_Continuation(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{}
	p := testPlatform()

	args := buildArgs(req, r, p, true, "asst-1", "robot_m1_e1")
	assert.Contains(t, args, "--resume")
	assert.Contains(t, args, "--name")
	assert.NotContains(t, args, "--session-id")
	assert.NotContains(t, args, "--continue")

	resumeIdx := -1
	for i, a := range args {
		if a == "--resume" {
			resumeIdx = i
			break
		}
	}
	require.Greater(t, resumeIdx, -1)
	assert.Regexp(t, `^[0-9a-f]{8}-`, args[resumeIdx+1])
}

func TestBuildArgs_EmptyChatID_Continuation(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	r := &Runner{}
	p := testPlatform()

	args := buildArgs(req, r, p, true, "", "")
	assert.Contains(t, args, "--continue")
	assert.NotContains(t, args, "--session-id")
	assert.NotContains(t, args, "--name")
}

// --- fakeConnector implements connector.Connector for unit tests ---

type fakeConnector struct {
	id       string
	typ      int
	settings map[string]interface{}
}

func (f *fakeConnector) Register(string, string, []byte) error { return nil }
func (f *fakeConnector) Query() (query.Query, error)           { return nil, nil }
func (f *fakeConnector) Schema() (schema.Schema, error)        { return nil, nil }
func (f *fakeConnector) Close() error                          { return nil }
func (f *fakeConnector) ID() string                            { return f.id }
func (f *fakeConnector) Is(t int) bool                         { return f.typ == t }
func (f *fakeConnector) Setting() map[string]interface{}       { return f.settings }
func (f *fakeConnector) GetMetaInfo() gouTypes.MetaInfo        { return gouTypes.MetaInfo{} }

func newOpenAIConnector(id, host, model, key string) *fakeConnector {
	return &fakeConnector{
		id:  id,
		typ: connector.OPENAI,
		settings: map[string]interface{}{
			"host":  host,
			"model": model,
			"key":   key,
		},
	}
}

func newAnthropicConnector(id, host, model, key string) *fakeConnector {
	return &fakeConnector{
		id:  id,
		typ: connector.ANTHROPIC,
		settings: map[string]interface{}{
			"host":  host,
			"model": model,
			"key":   key,
		},
	}
}

func newDualProtoConnector(id, host, model, key string) *fakeConnector {
	return &fakeConnector{
		id:  id,
		typ: connector.OPENAI,
		settings: map[string]interface{}{
			"host":      host,
			"model":     model,
			"key":       key,
			"protocols": []string{"openai", "anthropic"},
		},
	}
}

// --- helper functions tests ---

func TestConnectorHost(t *testing.T) {
	c := newOpenAIConnector("test", "https://api.openai.com", "gpt-4", "k")
	assert.Equal(t, "https://api.openai.com", connectorHost(c))
	assert.Equal(t, "", connectorHost(nil))
}

func TestConnectorProtocols(t *testing.T) {
	oai := newOpenAIConnector("oai", "https://api.openai.com", "gpt-4", "k")
	assert.Equal(t, []string{"openai"}, connectorProtocols(oai))

	anth := newAnthropicConnector("anth", "https://api.anthropic.com", "claude", "k")
	assert.Equal(t, []string{"anthropic"}, connectorProtocols(anth))

	dual := newDualProtoConnector("dual", "https://api.yao.run", "model", "k")
	assert.Equal(t, []string{"openai", "anthropic"}, connectorProtocols(dual))

	assert.Nil(t, connectorProtocols(nil))
}

func TestSupportsProtocol(t *testing.T) {
	dual := newDualProtoConnector("dual", "https://api.yao.run", "model", "k")
	assert.True(t, supportsProtocol(dual, "anthropic"))
	assert.True(t, supportsProtocol(dual, "openai"))
	assert.False(t, supportsProtocol(dual, "grpc"))

	oai := newOpenAIConnector("oai", "https://api.openai.com", "gpt-4", "k")
	assert.False(t, supportsProtocol(oai, "anthropic"))
	assert.True(t, supportsProtocol(oai, "openai"))
}

func TestResolveRoleConnector_Undeclared(t *testing.T) {
	roles := map[string]*types.RoleConnector{}
	result := resolveRoleConnector("heavy", roles, false, func(id string) connector.Connector { return nil })
	assert.Nil(t, result)
}

func TestResolveRoleConnector_Force(t *testing.T) {
	heavyConn := newOpenAIConnector("thinking", "https://api.thinking.com", "think-model", "k")
	roles := map[string]*types.RoleConnector{
		"heavy": {Connector: "thinking", Override: "force"},
	}
	result := resolveRoleConnector("heavy", roles, true, func(id string) connector.Connector {
		if id == "thinking" {
			return heavyConn
		}
		return nil
	})
	assert.Equal(t, heavyConn, result)
}

func TestResolveRoleConnector_UserExplicit(t *testing.T) {
	roles := map[string]*types.RoleConnector{
		"heavy": {Connector: "thinking", Override: "user"},
	}
	result := resolveRoleConnector("heavy", roles, true, func(id string) connector.Connector {
		return newOpenAIConnector("thinking", "h", "m", "k")
	})
	assert.Nil(t, result, "override=user + userExplicit=true => use user's connector")
}

func TestResolveRoleConnector_UserNotExplicit(t *testing.T) {
	heavyConn := newOpenAIConnector("thinking", "h", "m", "k")
	roles := map[string]*types.RoleConnector{
		"heavy": {Connector: "thinking", Override: "user"},
	}
	result := resolveRoleConnector("heavy", roles, false, func(id string) connector.Connector {
		if id == "thinking" {
			return heavyConn
		}
		return nil
	})
	assert.Equal(t, heavyConn, result, "override=user + userExplicit=false => use sandbox connector")
}

// --- buildEnv with multi-connector ---

func registerTestConnectors(t *testing.T, connectors map[string]connector.Connector) func() {
	t.Helper()
	for id, c := range connectors {
		connector.Connectors[id] = c
	}
	return func() {
		for id := range connectors {
			delete(connector.Connectors, id)
		}
	}
}

func TestBuildEnv_OpenAI_SingleConnector(t *testing.T) {
	oai := newOpenAIConnector("kimi", "https://api.moonshot.cn", "kimi-k2.5", "sk-test")
	req := &types.StreamRequest{
		Config:    &types.SandboxConfig{},
		Connector: oai,
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Contains(t, env["ANTHROPIC_BASE_URL"], "127.0.0.1")
	assert.Contains(t, env["ANTHROPIC_BASE_URL"], "kimi")
	assert.Equal(t, "claude-sonnet-4-6", env["ANTHROPIC_MODEL"])
	assert.Equal(t, "claude-sonnet-4-6", env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
}

func TestBuildEnv_OpenAI_MultiConnector(t *testing.T) {
	primary := newOpenAIConnector("kimi", "https://api.moonshot.cn", "kimi-k2.5", "sk-test")
	vision := newOpenAIConnector("vision-conn", "https://api.vision.com", "vis-model", "sk-v")

	cleanup := registerTestConnectors(t, map[string]connector.Connector{
		"vision-conn": vision,
	})
	defer cleanup()

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Runner: types.RunnerConfig{
				Connectors: map[string]*types.RoleConnector{
					"vision": {Connector: "vision-conn", Override: "force"},
				},
			},
		},
		Connector: primary,
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Equal(t, "claude-vision-4-5", env["ANTHROPIC_DEFAULT_SONNET_MODEL"],
		"vision role should get its virtual model name for A2O routing")
	assert.Equal(t, "claude-sonnet-4-6", env["ANTHROPIC_MODEL"],
		"primary should keep default virtual model")
}

func TestBuildEnv_Anthropic_SingleConnector(t *testing.T) {
	anth := newAnthropicConnector("claude", "https://api.anthropic.com", "claude-sonnet-4-20250514", "sk-ant-test")
	req := &types.StreamRequest{
		Config:    &types.SandboxConfig{},
		Connector: anth,
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Equal(t, "https://api.anthropic.com", env["ANTHROPIC_BASE_URL"])
	assert.Equal(t, "sk-ant-test", env["ANTHROPIC_API_KEY"])
	assert.Equal(t, "claude-sonnet-4-20250514", env["ANTHROPIC_MODEL"])
}

func TestBuildEnv_Anthropic_MultiConnector_Compatible(t *testing.T) {
	primary := newAnthropicConnector("claude", "https://api.yao.run", "claude-sonnet-4-20250514", "sk-ant")
	lightConn := newDualProtoConnector("light-conn", "https://api.yao.run", "claude-haiku-3-5-20241022", "sk-light")

	cleanup := registerTestConnectors(t, map[string]connector.Connector{
		"light-conn": lightConn,
	})
	defer cleanup()

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Runner: types.RunnerConfig{
				Connectors: map[string]*types.RoleConnector{
					"light": {Connector: "light-conn", Override: "force"},
				},
			},
		},
		Connector: primary,
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Equal(t, "claude-haiku-3-5-20241022", env["ANTHROPIC_DEFAULT_HAIKU_MODEL"],
		"compatible connector: use real model name")
	assert.Equal(t, "https://api.yao.run", env["ANTHROPIC_BASE_URL"],
		"primary base URL unchanged")
}

// --- buildSingleA2OConfig + injectA2OConfigWithRoutes ---

func TestBuildSingleA2OConfig_Basic(t *testing.T) {
	conn := newOpenAIConnector("test", "https://api.openai.com", "gpt-4", "sk-test")
	cfg := buildSingleA2OConfig(conn)
	require.NotNil(t, cfg)
	assert.Contains(t, cfg.Backend, "api.openai.com")
	assert.Contains(t, cfg.Backend, "chat/completions")
	assert.Equal(t, "gpt-4", cfg.Model)
	assert.Equal(t, "sk-test", cfg.APIKey)
}

func TestBuildSingleA2OConfig_Nil(t *testing.T) {
	conn := &fakeConnector{id: "empty", typ: connector.OPENAI, settings: map[string]interface{}{}}
	cfg := buildSingleA2OConfig(conn)
	assert.Nil(t, cfg, "no host => nil config")
}

func TestInjectA2OConfigWithRoutes_BuildsCorrectJSON(t *testing.T) {
	primary := newOpenAIConnector("kimi", "https://api.moonshot.cn", "kimi-k2.5", "sk-kimi")
	vision := newOpenAIConnector("vision", "https://api.vision.com", "vis-model", "sk-v")

	roleConnectors := map[string]connector.Connector{
		"claude-vision-4-5": vision,
	}

	primaryCfg := buildSingleA2OConfig(primary)
	require.NotNil(t, primaryCfg)

	routes := make(map[string]*a2oConnectorConfig, len(roleConnectors))
	for modelName, rc := range roleConnectors {
		routeCfg := buildSingleA2OConfig(rc)
		if routeCfg != nil {
			routes[modelName] = routeCfg
		}
	}
	primaryCfg.Routes = routes

	data, err := json.Marshal(primaryCfg)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	routesMap, ok := parsed["routes"].(map[string]interface{})
	require.True(t, ok, "routes should be present in JSON")
	assert.Len(t, routesMap, 1)

	visionRoute, ok := routesMap["claude-vision-4-5"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "vis-model", visionRoute["model"])
	assert.Contains(t, visionRoute["backend"], "api.vision.com")
}

func TestResolveAllRoleConnectors_Empty(t *testing.T) {
	req := &types.StreamRequest{
		Config:    &types.SandboxConfig{},
		Connector: newOpenAIConnector("test", "h", "m", "k"),
	}
	result := resolveAllRoleConnectors(req)
	assert.Nil(t, result)
}

func TestResolveAllRoleConnectors_WithRoles(t *testing.T) {
	vision := newOpenAIConnector("vis", "https://vis.com", "vis-m", "sk")
	cleanup := registerTestConnectors(t, map[string]connector.Connector{"vis": vision})
	defer cleanup()

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Runner: types.RunnerConfig{
				Connectors: map[string]*types.RoleConnector{
					"vision": {Connector: "vis", Override: "force"},
				},
			},
		},
		Connector: newOpenAIConnector("primary", "h", "m", "k"),
	}
	result := resolveAllRoleConnectors(req)
	assert.Len(t, result, 1)
	assert.Equal(t, vision, result["claude-vision-4-5"])
}

func TestBuildEnv_Anthropic_MultiConnector_Incompatible(t *testing.T) {
	primary := newAnthropicConnector("claude", "https://api.anthropic.com", "claude-sonnet-4-20250514", "sk-ant")
	visionConn := newOpenAIConnector("vision-oai", "https://api.openai.com", "gpt-4o", "sk-oai")

	cleanup := registerTestConnectors(t, map[string]connector.Connector{
		"vision-oai": visionConn,
	})
	defer cleanup()

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Runner: types.RunnerConfig{
				Connectors: map[string]*types.RoleConnector{
					"vision": {Connector: "vision-oai", Override: "force"},
				},
			},
		},
		Connector: primary,
	}
	req.Computer = newFakeComputer("/workspace")
	p := testPlatform()

	env := buildEnv(req, p)
	assert.Equal(t, "claude-sonnet-4-20250514", env["ANTHROPIC_DEFAULT_SONNET_MODEL"],
		"incompatible connector: vision should keep primary model (different host, no anthropic protocol)")
}
