package claude

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Config: &types.SandboxConfig{ID: "my-assistant"},
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
	r := &ClaudeRunner{}
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
	r := &ClaudeRunner{}
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
	r := &ClaudeRunner{}
	p := testPlatform()

	args := buildArgs(req, r, p, false, "", "")
	assert.Contains(t, args, "--dangerously-skip-permissions")
	assert.Contains(t, args, "--permission-mode")
}

func TestBuildArgs_MCP(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = newFakeComputer("/workspace")
	r := &ClaudeRunner{hasMCP: true, mcpToolPattern: "mcp__yao__*"}
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
	r := &ClaudeRunner{}
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
	r := &ClaudeRunner{}
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
	r := &ClaudeRunner{}
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
	r := &ClaudeRunner{}
	p := testPlatform()

	args := buildArgs(req, r, p, true, "", "")
	assert.Contains(t, args, "--continue")
	assert.NotContains(t, args, "--session-id")
	assert.NotContains(t, args, "--name")
}
