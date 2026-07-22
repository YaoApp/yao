//go:build unit

package claude_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/claude"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/tai/registry"
	taiTypes "github.com/yaoapp/yao/tai/types"
)

// --- fake connector types ---

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
		id: id, typ: connector.OPENAI,
		settings: map[string]interface{}{"host": host, "model": model, "key": key},
	}
}

func newAnthropicConnector(id, host, model, key string) *fakeConnector {
	return &fakeConnector{
		id: id, typ: connector.ANTHROPIC,
		settings: map[string]interface{}{"host": host, "model": model, "key": key},
	}
}

func newDualProtoConnector(id, host, model, key string) *fakeConnector {
	return &fakeConnector{
		id: id, typ: connector.OPENAI,
		settings: map[string]interface{}{
			"host": host, "model": model, "key": key,
			"protocols": []string{"openai", "anthropic"},
		},
	}
}

type fakeLLMConnector struct {
	fakeConnector
	model string
	caps  *goullm.Capabilities
}

func (f *fakeLLMConnector) GetAuthMode() goullm.AuthMode                     { return goullm.AuthBearer }
func (f *fakeLLMConnector) GetURL() string                                   { return "" }
func (f *fakeLLMConnector) GetKey() string                                   { return "" }
func (f *fakeLLMConnector) GetModel() string                                 { return f.model }
func (f *fakeLLMConnector) GetSupportedParams() map[string]*goullm.ParamSpec { return nil }
func (f *fakeLLMConnector) GetCapabilities() *goullm.Capabilities            { return f.caps }

func testPlatform() claude.ExportPlatform {
	return claude.ExportNewDarwinPlatform("/workspace", "bash", "/tmp")
}

// --- hashUserID ---

func TestHashUserID_Deterministic(t *testing.T) {
	h1 := claude.ExportHashUserID("user-123")
	h2 := claude.ExportHashUserID("user-123")
	if h1 != h2 {
		t.Errorf("expected deterministic hash, got %q and %q", h1, h2)
	}
}

func TestHashUserID_Different(t *testing.T) {
	h1 := claude.ExportHashUserID("user-1")
	h2 := claude.ExportHashUserID("user-2")
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestHashUserID_Length(t *testing.T) {
	h := claude.ExportHashUserID("test")
	if len(h) != 16 {
		t.Errorf("expected 16 hex chars, got %d (%q)", len(h), h)
	}
}

// --- chatIDToSessionUUID ---

func TestChatIDToSessionUUID_Deterministic(t *testing.T) {
	u1 := claude.ExportChatIDToSessionUUID("asst-1", "robot_m1_e1")
	u2 := claude.ExportChatIDToSessionUUID("asst-1", "robot_m1_e1")
	if u1 != u2 {
		t.Error("same inputs should produce same UUID")
	}
}

func TestChatIDToSessionUUID_DifferentAssistant(t *testing.T) {
	u1 := claude.ExportChatIDToSessionUUID("asst-1", "robot_m1_e1")
	u2 := claude.ExportChatIDToSessionUUID("asst-2", "robot_m1_e1")
	if u1 == u2 {
		t.Error("different assistantID should produce different UUID")
	}
}

// --- sanitizeSessionName ---

func TestSanitizeSessionName_Normal(t *testing.T) {
	got := claude.ExportSanitizeSessionName("robot_m1_e1")
	if got != "yao-robot_m1_e1" {
		t.Errorf("got %q", got)
	}
}

func TestSanitizeSessionName_SpecialChars(t *testing.T) {
	got := claude.ExportSanitizeSessionName("user's \"chat\"")
	if got != "yao-user_s__chat_" {
		t.Errorf("got %q", got)
	}
}

func TestSanitizeSessionName_Empty(t *testing.T) {
	got := claude.ExportSanitizeSessionName("")
	if got != "yao-" {
		t.Errorf("got %q", got)
	}
}

// --- isStandardAnthropicModel ---

func TestIsStandardAnthropicModel(t *testing.T) {
	cases := []struct {
		model string
		want  bool
	}{
		{"claude-sonnet-4-20250514", true},
		{"claude-3-5-haiku-20241022", true},
		{"anthropic.claude-3-haiku", true},
		{"deepseek-v4-flash", false},
		{"gpt-4o", false},
		{"", false},
	}
	for _, tc := range cases {
		got := claude.ExportIsStandardAnthropicModel(tc.model)
		if got != tc.want {
			t.Errorf("isStandardAnthropicModel(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

// --- buildMCPConfig ---

func TestBuildMCPConfig_WithServers(t *testing.T) {
	data := claude.ExportBuildMCPConfig([]types.MCPServer{{ServerID: "s1"}, {ServerID: "s2"}})
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	mcpServers, _ := cfg["mcpServers"].(map[string]any)
	if _, ok := mcpServers["s1"]; !ok {
		t.Error("missing s1")
	}
	if _, ok := mcpServers["s2"]; !ok {
		t.Error("missing s2")
	}
}

func TestBuildMCPConfig_Empty(t *testing.T) {
	data := claude.ExportBuildMCPConfig(nil)
	var cfg map[string]any
	json.Unmarshal(data, &cfg)
	mcpServers, _ := cfg["mcpServers"].(map[string]any)
	if _, ok := mcpServers["yao"]; !ok {
		t.Error("should default to yao server")
	}
}

func TestBuildMCPConfig_EmptyServerID(t *testing.T) {
	data := claude.ExportBuildMCPConfig([]types.MCPServer{{ServerID: ""}})
	var cfg map[string]any
	json.Unmarshal(data, &cfg)
	mcpServers, _ := cfg["mcpServers"].(map[string]any)
	if _, ok := mcpServers["yao"]; !ok {
		t.Error("empty serverID should fall back to default")
	}
}

// --- buildMCPAllowedTools ---

func TestBuildMCPAllowedTools_WithServers(t *testing.T) {
	result := claude.ExportBuildMCPAllowedTools([]types.MCPServer{{ServerID: "s1"}, {ServerID: "s2"}})
	if !strings.Contains(result, "mcp__s1__*") || !strings.Contains(result, "mcp__s2__*") {
		t.Errorf("got %q", result)
	}
}

func TestBuildMCPAllowedTools_Empty(t *testing.T) {
	if got := claude.ExportBuildMCPAllowedTools(nil); got != "mcp__yao__*" {
		t.Errorf("got %q", got)
	}
}

// --- buildLastUserMessageJSONL ---

func TestBuildLastUserMessageJSONL_SkipsSystem(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello"},
	}
	result := claude.ExportBuildLastUserJSONL(msgs)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatal(err)
	}
	msg, _ := parsed["message"].(map[string]any)
	if msg["content"] != "hello" {
		t.Errorf("content = %v", msg["content"])
	}
}

func TestBuildLastUserMessageJSONL_OnlyLastUser(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "user", Content: "q1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "q2"},
	}
	result := claude.ExportBuildLastUserJSONL(msgs)
	var parsed map[string]any
	json.Unmarshal([]byte(result), &parsed)
	msg, _ := parsed["message"].(map[string]any)
	if msg["content"] != "q2" {
		t.Errorf("content = %v", msg["content"])
	}
}

func TestBuildLastUserMessageJSONL_NilContent(t *testing.T) {
	msgs := []agentContext.Message{{Role: "user", Content: nil}}
	result := claude.ExportBuildLastUserJSONL(msgs)
	var parsed map[string]any
	json.Unmarshal([]byte(result), &parsed)
	msg, _ := parsed["message"].(map[string]any)
	if msg["content"] != "" {
		t.Errorf("content = %v", msg["content"])
	}
}

func TestBuildLastUserMessageJSONL_NoUser(t *testing.T) {
	msgs := []agentContext.Message{{Role: "assistant", Content: "only assistant"}}
	if got := claude.ExportBuildLastUserJSONL(msgs); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildLastUserMessageJSONL_Empty(t *testing.T) {
	if got := claude.ExportBuildLastUserJSONL(nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// --- buildSandboxEnvPrompt ---

func TestBuildSandboxEnvPrompt_Darwin(t *testing.T) {
	p := testPlatform()
	prompt := claude.ExportBuildSandboxEnvPrompt(p, "/workspace")
	if !strings.Contains(prompt, "/workspace") || !strings.Contains(prompt, "Sandbox Environment") {
		t.Errorf("prompt = %q", prompt)
	}
}

func TestBuildSandboxEnvPrompt_Windows(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "pwsh", "")
	prompt := claude.ExportBuildSandboxEnvPrompt(p, `C:\ws`)
	if !strings.Contains(prompt, "windows") || !strings.Contains(prompt, "Windows desktop") {
		t.Errorf("prompt = %q", prompt)
	}
}

// --- buildEnv ---

func TestBuildEnv_HomeEnv(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = claude.NewFakeComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["HOME"] != "/workspace" {
		t.Errorf("HOME = %q", env["HOME"])
	}
}

func TestBuildEnv_ConfigDirIsolation(t *testing.T) {
	req := &types.StreamRequest{
		Config:      &types.SandboxConfig{ID: "my-assistant"},
		AssistantID: "my-assistant",
	}
	req.Computer = claude.NewFakeComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["CLAUDE_CONFIG_DIR"] != "/workspace/.yao/assistants/my-assistant" {
		t.Errorf("CLAUDE_CONFIG_DIR = %q", env["CLAUDE_CONFIG_DIR"])
	}
}

func TestBuildEnv_Token(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Token:  &types.SandboxToken{Token: "tok123", RefreshToken: "ref456"},
	}
	req.Computer = claude.NewFakeComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["YAO_TOKEN"] != "tok123" || env["YAO_REFRESH_TOKEN"] != "ref456" {
		t.Errorf("tokens: %q, %q", env["YAO_TOKEN"], env["YAO_REFRESH_TOKEN"])
	}
}

func TestBuildEnv_GRPCAddr_HostMode(t *testing.T) {
	reg := registry.NewForTest()
	reg.Register(&registry.TaiNode{TaiID: "local", Mode: "local"})
	registry.SetGlobalForTest(reg)
	t.Cleanup(func() { registry.SetGlobalForTest(nil) })

	config.Conf.GRPC.Port = 9099

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Token:  &types.SandboxToken{Token: "tok123", RefreshToken: "ref456"},
	}
	req.Computer = claude.NewFakeHostComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["YAO_GRPC_ADDR"] != "127.0.0.1:9099" {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", env["YAO_GRPC_ADDR"], "127.0.0.1:9099")
	}
}

func TestBuildEnv_GRPCAddr_HostMode_Cloud(t *testing.T) {
	reg := registry.NewForTest()
	reg.Register(&registry.TaiNode{
		TaiID: "cloud-node",
		Mode:  "cloud",
		Ports: taiTypes.Ports{GRPC: 54321},
	})
	registry.SetGlobalForTest(reg)
	t.Cleanup(func() { registry.SetGlobalForTest(nil) })

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Token:  &types.SandboxToken{Token: "tok", RefreshToken: "ref"},
	}
	req.Computer = claude.NewFakeHostComputerWithNode("/workspace", "cloud-node")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["YAO_GRPC_ADDR"] != "127.0.0.1:54321" {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q (dynamic Tai port)", env["YAO_GRPC_ADDR"], "127.0.0.1:54321")
	}
}

func TestBuildEnv_GRPCAddr_HostMode_Tunnel(t *testing.T) {
	reg := registry.NewForTest()
	reg.Register(&registry.TaiNode{
		TaiID: "tunnel-node",
		Mode:  "tunnel",
		Ports: taiTypes.Ports{GRPC: 19200},
	})
	registry.SetGlobalForTest(reg)
	t.Cleanup(func() { registry.SetGlobalForTest(nil) })

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Token:  &types.SandboxToken{Token: "tok", RefreshToken: "ref"},
	}
	req.Computer = claude.NewFakeHostComputerWithNode("/workspace", "tunnel-node")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["YAO_GRPC_ADDR"] != "127.0.0.1:19200" {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q (dynamic Tai port)", env["YAO_GRPC_ADDR"], "127.0.0.1:19200")
	}
}

func TestBuildEnv_GRPCAddr_BoxMode(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
	}
	req.Computer = claude.NewFakeComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["YAO_GRPC_ADDR"] != "" {
		t.Errorf("expected YAO_GRPC_ADDR to be empty for non-host mode, got %q", env["YAO_GRPC_ADDR"])
	}
}

func TestBuildEnv_OpenAI(t *testing.T) {
	conn := newOpenAIConnector("kimi", "https://api.moonshot.cn", "kimi-k2.5", "sk-test")
	req := &types.StreamRequest{Config: &types.SandboxConfig{}, Connector: conn}
	req.Computer = claude.NewFakeComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if !strings.Contains(env["ANTHROPIC_BASE_URL"], "127.0.0.1") {
		t.Errorf("ANTHROPIC_BASE_URL = %q", env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_MODEL"] != "default" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q", env["ANTHROPIC_MODEL"], "default")
	}
	if env["ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"] != "kimi-k2.5" {
		t.Errorf("ANTHROPIC_CUSTOM_MODEL_OPTION_NAME = %q, want %q", env["ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"], "kimi-k2.5")
	}
}

func TestBuildEnv_Anthropic(t *testing.T) {
	conn := newAnthropicConnector("claude", "https://api.anthropic.com", "claude-sonnet-4-20250514", "sk-ant-test")
	req := &types.StreamRequest{Config: &types.SandboxConfig{}, Connector: conn}
	req.Computer = claude.NewFakeComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["ANTHROPIC_BASE_URL"] != "https://api.anthropic.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %q", env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_API_KEY"] != "sk-ant-test" {
		t.Errorf("ANTHROPIC_API_KEY = %q", env["ANTHROPIC_API_KEY"])
	}
}

func TestBuildEnv_Anthropic_MultiConnector_Incompatible(t *testing.T) {
	primary := newAnthropicConnector("claude", "https://api.anthropic.com", "claude-sonnet-4-20250514", "sk-ant")
	heavyConn := newOpenAIConnector("heavy-oai", "https://api.openai.com", "gpt-4o", "sk-oai")
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{}, Connector: primary,
		Roles: map[string]connector.Connector{"default": primary, "heavy": heavyConn},
	}
	req.Computer = claude.NewFakeComputer("/workspace")
	env := claude.ExportBuildEnv(req, testPlatform())
	if env["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "claude-sonnet-4-20250514" {
		t.Errorf("incompatible heavy should keep primary model, got %q", env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
}

// --- buildArgs ---

func TestBuildArgs_Default(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = claude.NewFakeComputer("/workspace")
	args := claude.ExportBuildArgs(req, false, "", testPlatform(), false, "", "")
	found := map[string]bool{}
	for _, a := range args {
		found[a] = true
	}
	for _, want := range []string{"--input-format", "--output-format", "--verbose", "--include-partial-messages"} {
		if !found[want] {
			t.Errorf("missing arg %q", want)
		}
	}
}

func TestBuildArgs_Continuation(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = claude.NewFakeComputer("/workspace")
	args := claude.ExportBuildArgs(req, false, "", testPlatform(), true, "", "")
	found := false
	for _, a := range args {
		if a == "--continue" {
			found = true
		}
	}
	if !found {
		t.Error("missing --continue")
	}
}

func TestBuildArgs_PermissionMode(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{
			Runner: types.RunnerConfig{
				Options: map[string]interface{}{"permission_mode": "bypassPermissions"},
			},
		},
	}
	req.Computer = claude.NewFakeComputer("/workspace")
	args := claude.ExportBuildArgs(req, false, "", testPlatform(), false, "", "")
	found := map[string]bool{}
	for _, a := range args {
		found[a] = true
	}
	if !found["--dangerously-skip-permissions"] || !found["--permission-mode"] {
		t.Error("missing permission args")
	}
}

func TestBuildArgs_MCP(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = claude.NewFakeComputer("/workspace")
	args := claude.ExportBuildArgs(req, true, "mcp__yao__*", testPlatform(), false, "test-assistant", "")
	found := map[string]bool{}
	for _, a := range args {
		found[a] = true
	}
	if !found["--mcp-config"] || !found["--allowedTools"] {
		t.Error("missing MCP args")
	}
}

func TestBuildArgs_SessionID(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = claude.NewFakeComputer("/workspace")
	args := claude.ExportBuildArgs(req, false, "", testPlatform(), false, "asst-1", "robot_m1_e1")
	found := map[string]bool{}
	for _, a := range args {
		found[a] = true
	}
	if !found["--session-id"] || !found["--name"] {
		t.Error("missing session args")
	}
	if found["--resume"] || found["--continue"] {
		t.Error("should not have resume/continue for new session")
	}
}

// --- connectorHost / connectorProtocols / supportsProtocol ---

func TestConnectorHost(t *testing.T) {
	c := newOpenAIConnector("test", "https://api.openai.com", "gpt-4", "k")
	if got := claude.ExportConnectorHost(c); got != "https://api.openai.com" {
		t.Errorf("got %q", got)
	}
	if got := claude.ExportConnectorHost(nil); got != "" {
		t.Errorf("nil: got %q", got)
	}
}

func TestConnectorProtocols(t *testing.T) {
	oai := newOpenAIConnector("oai", "h", "m", "k")
	protos := claude.ExportConnectorProtocols(oai)
	if len(protos) != 1 || protos[0] != "openai" {
		t.Errorf("openai connector protocols = %v", protos)
	}

	anth := newAnthropicConnector("anth", "h", "m", "k")
	protos = claude.ExportConnectorProtocols(anth)
	if len(protos) != 1 || protos[0] != "anthropic" {
		t.Errorf("anthropic connector protocols = %v", protos)
	}

	dual := newDualProtoConnector("dual", "h", "m", "k")
	protos = claude.ExportConnectorProtocols(dual)
	if len(protos) != 2 {
		t.Errorf("dual connector protocols = %v", protos)
	}
}

func TestSupportsProtocol(t *testing.T) {
	dual := newDualProtoConnector("dual", "h", "m", "k")
	if !claude.ExportSupportsProtocol(dual, "anthropic") {
		t.Error("should support anthropic")
	}
	if claude.ExportSupportsProtocol(dual, "grpc") {
		t.Error("should not support grpc")
	}
}

// --- buildModelCapabilityPrompt ---

func TestBuildModelCapabilityPrompt_NilConnector(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	if got := claude.ExportBuildModelCapabilityPrompt(req); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildModelCapabilityPrompt_WithHeavyAndLight(t *testing.T) {
	primary := &fakeLLMConnector{
		fakeConnector: fakeConnector{id: "ds-flash", typ: connector.OPENAI, settings: map[string]interface{}{"model": "deepseek-v4-flash"}},
		model:         "deepseek-v4-flash", caps: &goullm.Capabilities{ToolCalls: true},
	}
	heavy := &fakeLLMConnector{
		fakeConnector: fakeConnector{id: "ds-pro", typ: connector.OPENAI, settings: map[string]interface{}{"model": "deepseek-v4-pro"}},
		model:         "deepseek-v4-pro", caps: &goullm.Capabilities{Reasoning: true, ToolCalls: true},
	}
	light := &fakeLLMConnector{
		fakeConnector: fakeConnector{id: "ds-lite", typ: connector.OPENAI, settings: map[string]interface{}{"model": "deepseek-v4-flash"}},
		model:         "deepseek-v4-flash", caps: &goullm.Capabilities{ToolCalls: true},
	}
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{}, Connector: primary,
		Roles: map[string]connector.Connector{"default": primary, "heavy": heavy, "light": light},
	}
	result := claude.ExportBuildModelCapabilityPrompt(req)
	if !strings.Contains(result, "deepseek-v4-pro") || !strings.Contains(result, "thinking") {
		t.Errorf("result = %q", result)
	}
}

func TestBuildModelCapabilityPrompt_VisionGuidance(t *testing.T) {
	primary := &fakeLLMConnector{
		fakeConnector: fakeConnector{id: "ds", typ: connector.OPENAI, settings: map[string]interface{}{"model": "deepseek-v4-flash"}},
		model:         "deepseek-v4-flash", caps: &goullm.Capabilities{ToolCalls: true},
	}
	visionConn := &fakeLLMConnector{
		fakeConnector: fakeConnector{id: "vis", typ: connector.OPENAI, settings: map[string]interface{}{"model": "gpt-4o"}},
		model:         "gpt-4o", caps: &goullm.Capabilities{Vision: true, ToolCalls: true},
	}
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{}, Connector: primary,
		Roles: map[string]connector.Connector{"default": primary, "vision": visionConn},
	}
	result := claude.ExportBuildModelCapabilityPrompt(req)
	if !strings.Contains(result, "image_read") {
		t.Errorf("result = %q", result)
	}
}

// --- buildClaudeCodeCapabilities ---

func TestBuildClaudeCodeCapabilities_Nil(t *testing.T) {
	if got := claude.ExportBuildClaudeCodeCapabilities(nil); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestBuildClaudeCodeCapabilities_WithThinking(t *testing.T) {
	conn := &fakeConnector{
		id: "test", typ: connector.OPENAI,
		settings: map[string]interface{}{
			"thinking": map[string]interface{}{"type": "enabled"},
		},
	}
	if got := claude.ExportBuildClaudeCodeCapabilities(conn); got != "thinking" {
		t.Errorf("got %q", got)
	}
}

func TestBuildClaudeCodeCapabilities_NoThinking(t *testing.T) {
	conn := &fakeConnector{
		id: "test", typ: connector.OPENAI,
		settings: map[string]interface{}{"model": "gpt-4o"},
	}
	if got := claude.ExportBuildClaudeCodeCapabilities(conn); got != "" {
		t.Errorf("got %q", got)
	}
}

// --- resolveAllRoleConnectors ---

func TestResolveAllRoleConnectors_SkipsDefault(t *testing.T) {
	primary := newOpenAIConnector("ds", "https://api.deepseek.com/v1", "deepseek-v4-flash", "sk-1")
	light := newOpenAIConnector("gpt", "https://api.openai.com/v1", "gpt-4o-mini", "sk-2")
	req := &types.StreamRequest{
		Roles: map[string]connector.Connector{
			"default": primary,
			"light":   light,
		},
	}
	result := claude.ExportResolveAllRoleConnectors(req)
	if _, ok := result["default"]; ok {
		t.Error("default role should be excluded from routes")
	}
	if _, ok := result["light"]; !ok {
		t.Error("light role should be included")
	}
}

func TestResolveAllRoleConnectors_KeepsAnthropicProtocol(t *testing.T) {
	primary := newOpenAIConnector("ds", "https://api.deepseek.com/v1", "deepseek-v4-flash", "sk-1")
	anthLight := newAnthropicConnector("ds-ant", "https://api.deepseek.com/anthropic/v1", "deepseek-v4-flash", "sk-2")
	oaiHeavy := newOpenAIConnector("gpt", "https://api.openai.com/v1", "gpt-4o", "sk-3")
	req := &types.StreamRequest{
		Roles: map[string]connector.Connector{
			"default": primary,
			"light":   anthLight,
			"heavy":   oaiHeavy,
		},
	}
	result := claude.ExportResolveAllRoleConnectors(req)
	if _, ok := result["light"]; !ok {
		t.Error("anthropic-protocol connector should be kept (no protocol filtering)")
	}
	if _, ok := result["heavy"]; !ok {
		t.Error("openai-protocol heavy connector should be included")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 role connectors, got %d", len(result))
	}
}

func TestResolveAllRoleConnectors_SameModelDifferentRoles(t *testing.T) {
	primary := newOpenAIConnector("ds-oai", "https://api.deepseek.com/v1", "deepseek-v4-flash", "sk-1")
	anthLight := newAnthropicConnector("ds-ant", "https://api.deepseek.com/anthropic/v1", "deepseek-v4-flash", "sk-2")
	req := &types.StreamRequest{
		Roles: map[string]connector.Connector{
			"default": primary,
			"light":   anthLight,
		},
	}
	result := claude.ExportResolveAllRoleConnectors(req)
	if len(result) != 1 {
		t.Errorf("expected 1 role connector (light), got %d", len(result))
	}
	if _, ok := result["light"]; !ok {
		t.Error("light role should be present even with same model name as primary")
	}
}

func TestResolveAllRoleConnectors_EmptyRoles(t *testing.T) {
	req := &types.StreamRequest{}
	result := claude.ExportResolveAllRoleConnectors(req)
	if result != nil {
		t.Errorf("expected nil for empty roles, got %v", result)
	}
}

func TestResolveAllRoleConnectors_DualProtoKept(t *testing.T) {
	primary := newOpenAIConnector("ds", "https://api.deepseek.com/v1", "deepseek-v4-flash", "sk-1")
	dual := newDualProtoConnector("dual", "https://api.example.com/v1", "dual-model", "sk-2")
	req := &types.StreamRequest{
		Roles: map[string]connector.Connector{
			"default": primary,
			"heavy":   dual,
		},
	}
	result := claude.ExportResolveAllRoleConnectors(req)
	if _, ok := result["heavy"]; !ok {
		t.Error("dual-protocol connector (openai+anthropic) should be kept with role key")
	}
}
