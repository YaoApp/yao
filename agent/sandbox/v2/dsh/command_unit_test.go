//go:build unit

package dsh_test

import (
	"os"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/dsh"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// fakeConnForThinking implements connector.Connector for test cases.
type fakeConnForThinking struct {
	settings map[string]interface{}
	metadata map[string]interface{}
}

func (f *fakeConnForThinking) Register(string, string, []byte) error { return nil }
func (f *fakeConnForThinking) Query() (query.Query, error)           { return nil, nil }
func (f *fakeConnForThinking) Schema() (schema.Schema, error)        { return nil, nil }
func (f *fakeConnForThinking) Close() error                          { return nil }
func (f *fakeConnForThinking) ID() string                            { return "fake" }
func (f *fakeConnForThinking) Is(int) bool                           { return false }
func (f *fakeConnForThinking) Setting() map[string]interface{}       { return f.settings }
func (f *fakeConnForThinking) GetMetaInfo() gouTypes.MetaInfo        { return gouTypes.MetaInfo{} }
func (f *fakeConnForThinking) GetMetadata() map[string]interface{}   { return f.metadata }

// fakeLLMConn implements both connector.Connector and goullm.LLMConnector.
type fakeLLMConn struct {
	typ      int
	key      string
	url      string
	model    string
	settings map[string]interface{}
	metadata map[string]interface{}
	caps     *goullm.Capabilities
}

func (m *fakeLLMConn) Register(string, string, []byte) error            { return nil }
func (m *fakeLLMConn) Query() (query.Query, error)                      { return nil, nil }
func (m *fakeLLMConn) Schema() (schema.Schema, error)                   { return nil, nil }
func (m *fakeLLMConn) Close() error                                     { return nil }
func (m *fakeLLMConn) ID() string                                       { return "mock" }
func (m *fakeLLMConn) Is(t int) bool                                    { return m.typ == t }
func (m *fakeLLMConn) Setting() map[string]interface{}                  { return m.settings }
func (m *fakeLLMConn) GetMetaInfo() gouTypes.MetaInfo                   { return gouTypes.MetaInfo{} }
func (m *fakeLLMConn) GetMetadata() map[string]interface{}              { return m.metadata }
func (m *fakeLLMConn) GetAuthMode() goullm.AuthMode                     { return goullm.AuthBearer }
func (m *fakeLLMConn) GetURL() string                                   { return m.url }
func (m *fakeLLMConn) GetKey() string                                   { return m.key }
func (m *fakeLLMConn) GetModel() string                                 { return m.model }
func (m *fakeLLMConn) GetSupportedParams() map[string]*goullm.ParamSpec { return nil }
func (m *fakeLLMConn) GetCapabilities() *goullm.Capabilities            { return m.caps }

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	os.Exit(m.Run())
}

// --- extractLastUserMessage ---

func TestExtractLastUserMessage_Simple(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "user", Content: "hello world"},
	}
	got := dsh.ExportExtractLastUserMessage(msgs)
	if got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestExtractLastUserMessage_MultiPart(t *testing.T) {
	msgs := []agentContext.Message{
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "part one"},
				map[string]interface{}{"type": "text", "text": "part two"},
			},
		},
	}
	got := dsh.ExportExtractLastUserMessage(msgs)
	if got != "part one\n\npart two" {
		t.Errorf("got %q", got)
	}
}

func TestExtractLastUserMessage_OnlyLast(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "reply"},
		{Role: "user", Content: "second"},
	}
	got := dsh.ExportExtractLastUserMessage(msgs)
	if got != "second" {
		t.Errorf("got %q, want %q", got, "second")
	}
}

func TestExtractLastUserMessage_Empty(t *testing.T) {
	if got := dsh.ExportExtractLastUserMessage(nil); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestExtractLastUserMessage_NilContent(t *testing.T) {
	msgs := []agentContext.Message{{Role: "user", Content: nil}}
	got := dsh.ExportExtractLastUserMessage(msgs)
	if got != "<nil>" {
		t.Errorf("got %q", got)
	}
}

// --- buildSystemPrompt ---

func TestBuildSystemPrompt_WithLocale(t *testing.T) {
	req := &types.StreamRequest{
		Config:       &types.SandboxConfig{},
		SystemPrompt: "You are an agent.",
		Locale:       "zh-cn",
	}
	got := dsh.ExportBuildSystemPrompt(req, "/workspace")
	if got == "" {
		t.Fatal("empty prompt")
	}
	// Locale no longer injected into system prompt (moved to user message prefix for cache optimization).
	if !strings.Contains(got, "You are an agent.") {
		t.Errorf("prompt missing base system prompt, got %q", got)
	}
	if !strings.Contains(got, "Working directory: /workspace") {
		t.Errorf("prompt missing working directory, got %q", got)
	}
	if !strings.Contains(got, "Background Jobs & Daemons") {
		t.Errorf("prompt missing background jobs section, got %q", got)
	}
}

func TestBuildSystemPrompt_NoLocale(t *testing.T) {
	req := &types.StreamRequest{
		Config:       &types.SandboxConfig{},
		SystemPrompt: "You are an agent.",
	}
	got := dsh.ExportBuildSystemPrompt(req, "/workspace")
	if !strings.Contains(got, "You are an agent.") {
		t.Errorf("prompt missing base system prompt, got %q", got)
	}
	if !strings.Contains(got, "Working directory: /workspace") {
		t.Errorf("prompt missing working directory, got %q", got)
	}
	if !strings.Contains(got, "Background Jobs & Daemons") {
		t.Errorf("prompt missing background jobs section, got %q", got)
	}
}

func TestBuildSystemPrompt_EnLocale_NoExtra(t *testing.T) {
	req := &types.StreamRequest{
		Config:       &types.SandboxConfig{},
		SystemPrompt: "Agent",
		Locale:       "en-us",
	}
	got := dsh.ExportBuildSystemPrompt(req, "/workspace")
	if !strings.Contains(got, "Agent") {
		t.Errorf("prompt missing base system prompt, got %q", got)
	}
	if !strings.Contains(got, "Working directory: /workspace") {
		t.Errorf("prompt missing working directory, got %q", got)
	}
}

// --- buildEnv ---

func TestBuildEnv_HomeEnv(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "sk-test", "", "prompt")
	if env["HOME"] != "/workspace" {
		t.Errorf("HOME = %q", env["HOME"])
	}
}

func TestBuildEnv_DSH_Vars(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "sk-test", "https://api.ds.com", "my prompt")
	if env["DEEPSEEK_API_KEY"] != "sk-test" {
		t.Errorf("DEEPSEEK_API_KEY = %q", env["DEEPSEEK_API_KEY"])
	}
	if env["DSH_KEY_PRIMARY"] != "sk-test" {
		t.Errorf("DSH_KEY_PRIMARY = %q", env["DSH_KEY_PRIMARY"])
	}
	if env["DEEPSEEK_BASE_URL"] != "https://api.ds.com" {
		t.Errorf("DEEPSEEK_BASE_URL = %q", env["DEEPSEEK_BASE_URL"])
	}
	if env["DSH_CWD"] != "/workspace" {
		t.Errorf("DSH_CWD = %q", env["DSH_CWD"])
	}
	if env["DSH_SYSTEM_PROMPT"] != "my prompt" {
		t.Errorf("DSH_SYSTEM_PROMPT = %q", env["DSH_SYSTEM_PROMPT"])
	}
}

func TestBuildEnv_NoBaseURL(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "sk-test", "", "prompt")
	if _, ok := env["DEEPSEEK_BASE_URL"]; ok {
		t.Error("DEEPSEEK_BASE_URL should not be set when empty")
	}
}

func TestBuildEnv_Token(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Token:  &types.SandboxToken{Token: "tok123", RefreshToken: "ref456"},
	}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "", "", "")
	if env["YAO_TOKEN"] != "tok123" || env["YAO_REFRESH_TOKEN"] != "ref456" {
		t.Errorf("tokens: %q, %q", env["YAO_TOKEN"], env["YAO_REFRESH_TOKEN"])
	}
}

func TestBuildEnv_WorkspaceID(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{WorkspaceID: "ws-test-123"},
	}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "", "", "")
	if env["CTX_WORKSPACE_ID"] != "ws-test-123" {
		t.Errorf("CTX_WORKSPACE_ID = %q", env["CTX_WORKSPACE_ID"])
	}
}

func TestBuildEnv_SkillsDirs(t *testing.T) {
	req := &types.StreamRequest{
		Config:      &types.SandboxConfig{},
		AssistantID: "my-asst",
	}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "", "", "")
	if env["CTX_SKILLS_DIR"] != "/workspace/.yao/assistants/my-asst/skills" {
		t.Errorf("CTX_SKILLS_DIR = %q", env["CTX_SKILLS_DIR"])
	}
	if env["CTX_EXT_SKILLS_DIR"] != "/workspace/.yao/skills" {
		t.Errorf("CTX_EXT_SKILLS_DIR = %q", env["CTX_EXT_SKILLS_DIR"])
	}
}

func TestBuildEnv_SkillsDirs_NoAssistant(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "", "", "")
	if env["CTX_SKILLS_DIR"] != "/workspace/.dsh/skills" {
		t.Errorf("CTX_SKILLS_DIR = %q", env["CTX_SKILLS_DIR"])
	}
}

func TestBuildEnv_Windows(t *testing.T) {
	req := &types.StreamRequest{Config: &types.SandboxConfig{}}
	req.Computer = dsh.NewFakeWindowsComputer(`C:\workspace`)
	p := dsh.ExportNewWindowsPlatform("pwsh")
	env := dsh.ExportBuildEnv(req, p, `C:\workspace`, "sk-test", "", "")
	if env["USERPROFILE"] != `C:\workspace` {
		t.Errorf("USERPROFILE = %q", env["USERPROFILE"])
	}
	if env["HOMEDRIVE"] != "C:" {
		t.Errorf("HOMEDRIVE = %q", env["HOMEDRIVE"])
	}
}

// --- buildEnv: CTX_NODE_ID / CTX_TARGET_ID ---

func TestBuildEnv_NodeIDAndTargetID(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{NodeID: "node-abc", ID: "target-xyz"},
	}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "", "", "")
	if env["CTX_NODE_ID"] != "node-abc" {
		t.Errorf("CTX_NODE_ID = %q", env["CTX_NODE_ID"])
	}
	if env["CTX_TARGET_ID"] != "target-xyz" {
		t.Errorf("CTX_TARGET_ID = %q", env["CTX_TARGET_ID"])
	}
}

func TestBuildEnv_NodeID_DefaultTarget(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{NodeID: "node-abc"},
	}
	req.Computer = dsh.NewFakeComputer("/workspace")
	p := dsh.ExportNewPosixPlatform()
	env := dsh.ExportBuildEnv(req, p, "/workspace", "", "", "")
	if env["CTX_TARGET_ID"] != "__host__" {
		t.Errorf("CTX_TARGET_ID = %q, want __host__", env["CTX_TARGET_ID"])
	}
}

// --- RenderCordisConfig ---

func TestRenderCordisConfig_Default(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if len(s) == 0 {
		t.Fatal("empty config")
	}
	if !containsAll(s, "dsh-yaoapp-jsonrpc-stream", "dsh-bash-local", "dsh-fs-local") {
		t.Errorf("config missing expected plugins: %s", s)
	}
	if containsAny(s, "dsh-llm-deepseek", "dsh-llm-pi-ai") {
		t.Error("empty config should not render any LLM block")
	}
	if !containsAll(s, "session-persistence-jsonl", "session-checkpoint-policy") {
		t.Error("config should contain session persistence plugins")
	}
}

func TestRenderCordisConfig_PiAiOnly(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://api.openai.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models: []dsh.PiAiModelConfig{{
				ID:               "gpt-5.5",
				Input:            []string{"text"},
				ContextWindow:    1050000,
				MaxTokens:        128000,
				ReasoningEfforts: map[string]string{"off": "none", "low": "low", "medium": "medium", "high": "high"},
			}},
			Reasoning: "high",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if containsAny(s, "dsh-llm-deepseek") {
		t.Error("should not render deepseek block")
	}
	if !containsAll(s, "dsh-llm-pi-ai", "yao-primary:", "openai-completions",
		"https://api.openai.com/v1", "DSH_KEY_PRIMARY", "gpt-5.5", "reasoning: high",
		"contextWindow: 1050000", "maxTokens: 128000") {
		t.Errorf("pi-ai route not rendered correctly, got:\n%s", s)
	}
}

func TestRenderCordisConfig_PiAiWithReasoning(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://api.yaoagents.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models: []dsh.PiAiModelConfig{{
				ID:               "deepseek-flash",
				Input:            []string{"text", "image"},
				ReasoningEfforts: map[string]string{"off": "none", "low": "low", "high": "high", "max": "max"},
			}},
			Reasoning: "max",
		}},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "reasoningEfforts:", "off: none", "low: low", "high: high", "max: max") {
		t.Errorf("reasoning model should render reasoningEfforts block, got:\n%s", s)
	}
	if !containsAll(s, "reasoning: max") {
		t.Errorf("route reasoning should be max, got:\n%s", s)
	}
}

func TestRenderCordisConfig_PiAiNoReasoning(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://api.example.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models:    []dsh.PiAiModelConfig{{ID: "some-model", Input: []string{"text"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if containsAny(s, "reasoning:", "reasoningEfforts:") {
		t.Errorf("non-reasoning model should not render reasoning blocks, got:\n%s", s)
	}
}

func TestRenderCordisConfig_PiAiReasoningOmitted(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://api.example.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models:    []dsh.PiAiModelConfig{{ID: "some-model", Input: []string{"text"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if containsAny(s, "reasoning:") {
		t.Errorf("empty reasoning should be omitted, got:\n%s", s)
	}
}

func TestRenderCordisConfig_PiAiWithBudget(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://proxy.example.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models: []dsh.PiAiModelConfig{{
				ID:               "deepseek-chat",
				Input:            []string{"text", "image"},
				ReasoningEfforts: map[string]string{"off": "none", "high": "high", "max": "max"},
			}},
			Reasoning:    "high",
			BudgetTokens: 32000,
		}},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "thinkingBudgets:", "high: 32000") {
		t.Errorf("budget should be rendered, got:\n%s", s)
	}
}

func TestRenderCordisConfig_MultipleRoutes(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{
			{
				Name:      "yao-primary",
				API:       "openai-completions",
				BaseURL:   "https://api.yaoagents.com/v1",
				APIKeyEnv: "DSH_KEY_PRIMARY",
				Models: []dsh.PiAiModelConfig{{
					ID:               "deepseek-flash",
					Input:            []string{"text"},
					ContextWindow:    1048576,
					ReasoningEfforts: map[string]string{"off": "none", "high": "high", "max": "max"},
				}},
				Reasoning: "high",
			},
			{
				Name:      "yao-heavy",
				API:       "openai-completions",
				BaseURL:   "https://api.yaoagents.com/v1",
				APIKeyEnv: "DSH_KEY_HEAVY",
				Models: []dsh.PiAiModelConfig{{
					ID:               "deepseek-chat",
					Input:            []string{"text"},
					ContextWindow:    1048576,
					MaxTokens:        384000,
					ReasoningEfforts: map[string]string{"off": "none", "high": "high", "max": "max"},
				}},
				Reasoning: "max",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "yao-primary:", "yao-heavy:", "deepseek-flash", "deepseek-chat",
		"reasoning: high", "reasoning: max", "contextWindow: 1048576") {
		t.Errorf("multiple routes not rendered correctly, got:\n%s", s)
	}
}

func TestRenderCordisConfig_WithVision(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:            "yao-primary",
			API:             "openai-completions",
			BaseURL:         "https://api.yaoagents.com/v1",
			APIKeyEnv:       "DSH_KEY_PRIMARY",
			Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text", "image"}}},
			Reasoning:       "high",
			NoDeveloperRole: true,
		}},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "dsh-attachment-local") {
		t.Error("vision=true should load attachment-local plugin")
	}
}

func TestRenderCordisConfig_NoDeveloperRoleCompat(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:            "yao-primary",
			API:             "openai-completions",
			BaseURL:         "https://api.deepseek.com/v1",
			APIKeyEnv:       "DSH_KEY_PRIMARY",
			Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text"}}},
			Reasoning:       "high",
			NoDeveloperRole: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "compat:", "supportsDeveloperRole: false") {
		t.Errorf("NoDeveloperRole route should render supportsDeveloperRole: false, got:\n%s", s)
	}
	if containsAny(s, "thinkingFormat:", "maxTokensField:", "requiresReasoningContentOnAssistantMessages:") {
		t.Errorf("compat block should NOT contain provider-specific fields, got:\n%s", s)
	}
}

func TestRenderCordisConfig_ThinkingFormat(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:            "yao-primary",
			API:             "openai-completions",
			BaseURL:         "https://api.yaoagents.com/v1",
			APIKeyEnv:       "DSH_KEY_PRIMARY",
			Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text"}}},
			Reasoning:       "high",
			NoDeveloperRole: true,
			ThinkingFormat:  "deepseek",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "compat:", "supportsDeveloperRole: false", "thinkingFormat: deepseek") {
		t.Errorf("ThinkingFormat route should render full compat block, got:\n%s", s)
	}
}

func TestRenderCordisConfig_ThinkingFormat_WithoutNoDeveloperRole(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:           "yao-primary",
			API:            "anthropic-messages",
			BaseURL:        "https://api.anthropic.com",
			APIKeyEnv:      "DSH_KEY_PRIMARY",
			Models:         []dsh.PiAiModelConfig{{ID: "claude-sonnet-4", Input: []string{"text"}}},
			Reasoning:      "high",
			ThinkingFormat: "some-format",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "compat:", "thinkingFormat: some-format") {
		t.Errorf("ThinkingFormat without NoDeveloperRole should still render compat, got:\n%s", s)
	}
	if containsAny(s, "supportsDeveloperRole") {
		t.Errorf("should not have supportsDeveloperRole when NoDeveloperRole is false, got:\n%s", s)
	}
}

// --- normalizeBaseURL ---

func TestNormalizeBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		api     string
		baseURL string
		want    string
	}{
		{"openai_no_trailing_slash", "openai-completions", "https://api.deepseek.com", "https://api.deepseek.com/v1"},
		{"openai_with_custom_path", "openai-completions", "https://ark.cn-beijing.volces.com/api/v3/", "https://ark.cn-beijing.volces.com/api/v3"},
		{"anthropic_no_trailing_slash", "anthropic-messages", "https://api.anthropic.com", "https://api.anthropic.com"},
		{"anthropic_trailing_slash", "anthropic-messages", "https://api.deepseek.com/anthropic/", "https://api.deepseek.com/anthropic"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dsh.ExportNormalizeBaseURL(c.api, c.baseURL); got != c.want {
				t.Errorf("normalizeBaseURL(%q, %q) = %q, want %q", c.api, c.baseURL, got, c.want)
			}
		})
	}
}

// --- hasVisionInput ---

func TestHasVisionInput(t *testing.T) {
	if dsh.ExportHasVisionInput([]string{"text"}) {
		t.Error("text-only should not be vision")
	}
	if !dsh.ExportHasVisionInput([]string{"text", "image"}) {
		t.Error("text+image should be vision")
	}
	if dsh.ExportHasVisionInput(nil) {
		t.Error("nil should not be vision")
	}
}

func TestRenderCordisConfig_DisableReasoning(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://api.example.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models: []dsh.PiAiModelConfig{{
				ID:               "text-only-model",
				Input:            []string{"text"},
				DisableReasoning: true,
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "reasoningEfforts: false") {
		t.Errorf("DisableReasoning should render 'reasoningEfforts: false', got:\n%s", s)
	}
	if containsAny(s, "off:", "low:", "high:", "max:") {
		t.Errorf("DisableReasoning should not render effort map, got:\n%s", s)
	}
}

func TestRenderCordisConfig_ContextWindowAndMaxTokens(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://api.example.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models: []dsh.PiAiModelConfig{{
				ID:            "some-model",
				Input:         []string{"text"},
				ContextWindow: 196608,
				MaxTokens:     32000,
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "contextWindow: 196608", "maxTokens: 32000") {
		t.Errorf("contextWindow and maxTokens should be rendered, got:\n%s", s)
	}
}

func TestRenderCordisConfig_ZeroContextWindow(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "openai-completions",
			BaseURL:   "https://api.example.com/v1",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models:    []dsh.PiAiModelConfig{{ID: "some-model", Input: []string{"text"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if containsAny(s, "contextWindow:") {
		t.Errorf("zero contextWindow should not be rendered, got:\n%s", s)
	}
	// Extract the model entry block (from model ID to next "- id:" or end of providers section).
	// The template-level maxTokensAsSuccess and compaction maxTokens are unrelated.
	start := strings.Index(s, "some-model")
	end := strings.Index(s[start:], "\n\n")
	if end < 0 {
		end = len(s) - start
	}
	modelBlock := s[start : start+end]
	if containsAny(modelBlock, "maxTokens:") {
		t.Errorf("zero maxTokens should not be rendered in model block, got:\n%s", modelBlock)
	}
}

func TestRenderCordisConfig_NonDeepSeekOpenAI(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:            "yao-primary",
			API:             "openai-completions",
			BaseURL:         "https://api.moonshot.cn",
			APIKeyEnv:       "DSH_KEY_PRIMARY",
			Models:          []dsh.PiAiModelConfig{{ID: "kimi-k2.5", Input: []string{"text"}}},
			NoDeveloperRole: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "compat:", "supportsDeveloperRole: false") {
		t.Errorf("non-DeepSeek OpenAI should still have supportsDeveloperRole: false, got:\n%s", s)
	}
	if containsAny(s, "thinkingFormat:", "requiresReasoningContentOnAssistantMessages") {
		t.Errorf("non-DeepSeek OpenAI should NOT have provider-specific compat, got:\n%s", s)
	}
}

func TestRenderCordisConfig_AnthropicRoute(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{{
			Name:      "yao-primary",
			API:       "anthropic-messages",
			BaseURL:   "https://api.anthropic.com",
			APIKeyEnv: "DSH_KEY_PRIMARY",
			Models: []dsh.PiAiModelConfig{{
				ID:               "claude-sonnet-4",
				Input:            []string{"text", "image"},
				ContextWindow:    1000000,
				MaxTokens:        64000,
				ReasoningEfforts: map[string]string{"off": "none", "high": "high"},
			}},
			Reasoning: "high",
		}},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "anthropic-messages", "https://api.anthropic.com", "claude-sonnet-4") {
		t.Errorf("Anthropic route not rendered correctly, got:\n%s", s)
	}
	if containsAny(s, "supportsDeveloperRole", "thinkingFormat: deepseek") {
		t.Errorf("Anthropic route should NOT have DeepSeek compat, got:\n%s", s)
	}
}

func TestRenderCordisConfig_MixedRoutes(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		PiAiRoutes: []dsh.PiAiRoute{
			{
				Name:            "yao-primary",
				API:             "openai-completions",
				BaseURL:         "https://api.deepseek.com/v1",
				APIKeyEnv:       "DSH_KEY_PRIMARY",
				Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text"}}},
				Reasoning:       "high",
				NoDeveloperRole: true,
			},
			{
				Name:      "yao-heavy",
				API:       "anthropic-messages",
				BaseURL:   "https://api.anthropic.com",
				APIKeyEnv: "DSH_KEY_HEAVY",
				Models:    []dsh.PiAiModelConfig{{ID: "claude-sonnet-4", Input: []string{"text", "image"}}},
				Reasoning: "high",
			},
		},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "yao-primary:", "openai-completions", "compat:", "supportsDeveloperRole: false",
		"yao-heavy:", "anthropic-messages", "https://api.anthropic.com") {
		t.Errorf("mixed routes not rendered correctly, got:\n%s", s)
	}
	if containsAny(s, "thinkingFormat:", "maxTokensField:", "requiresReasoningContentOnAssistantMessages:") {
		t.Errorf("compat block should NOT contain provider-specific fields, got:\n%s", s)
	}
}

// --- profileToPiAiRoute ---

func TestProfileToPiAiRoute(t *testing.T) {
	p := llmprovider.ConnectorProfile{
		APIKey:           "sk-test",
		BaseURL:          "https://api.deepseek.com",
		API:              "openai-completions",
		Model:            "deepseek-flash",
		Input:            []string{"text", "image"},
		ContextWindow:    1048576,
		MaxTokens:        384000,
		Reasoning:        "high",
		ReasoningEfforts: map[string]string{"off": "none", "high": "high", "max": "max"},
		BudgetTokens:     0,
		ThinkingFormat:   "deepseek",
		NoDeveloperRole:  true,
	}
	route := dsh.ExportProfileToPiAiRoute(p, "yao-primary", "DSH_KEY_PRIMARY")
	if route.Name != "yao-primary" {
		t.Errorf("Name = %q", route.Name)
	}
	if route.API != "openai-completions" {
		t.Errorf("API = %q", route.API)
	}
	// normalizeBaseURL appends /v1 for openai-completions
	if route.BaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("BaseURL = %q", route.BaseURL)
	}
	if route.APIKeyEnv != "DSH_KEY_PRIMARY" {
		t.Errorf("APIKeyEnv = %q", route.APIKeyEnv)
	}
	if route.Reasoning != "high" {
		t.Errorf("Reasoning = %q", route.Reasoning)
	}
	if route.ThinkingFormat != "deepseek" {
		t.Errorf("ThinkingFormat = %q", route.ThinkingFormat)
	}
	if !route.NoDeveloperRole {
		t.Error("NoDeveloperRole should be true")
	}
	if len(route.Models) != 1 {
		t.Fatalf("Models len = %d", len(route.Models))
	}
	m := route.Models[0]
	if m.ID != "deepseek-flash" {
		t.Errorf("Model.ID = %q", m.ID)
	}
	if m.ContextWindow != 1048576 {
		t.Errorf("Model.ContextWindow = %d", m.ContextWindow)
	}
	if m.MaxTokens != 384000 {
		t.Errorf("Model.MaxTokens = %d", m.MaxTokens)
	}
	if m.ReasoningEfforts["off"] != "none" || m.ReasoningEfforts["high"] != "high" || m.ReasoningEfforts["max"] != "max" {
		t.Errorf("Model.ReasoningEfforts = %v", m.ReasoningEfforts)
	}
}

func TestProfileToPiAiRoute_Anthropic(t *testing.T) {
	p := llmprovider.ConnectorProfile{
		APIKey:        "sk-ant",
		BaseURL:       "https://api.anthropic.com/",
		API:           "anthropic-messages",
		Model:         "claude-sonnet-4",
		Input:         []string{"text", "image"},
		ContextWindow: 1000000,
		MaxTokens:     64000,
		Reasoning:     "high",
		BudgetTokens:  10000,
	}
	route := dsh.ExportProfileToPiAiRoute(p, "yao-primary", "DSH_KEY_PRIMARY")
	// normalizeBaseURL trims trailing / for anthropic-messages
	if route.BaseURL != "https://api.anthropic.com" {
		t.Errorf("BaseURL = %q", route.BaseURL)
	}
	if route.BudgetTokens != 10000 {
		t.Errorf("BudgetTokens = %d", route.BudgetTokens)
	}
}

// --- helpers ---

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && strings.Contains(s, sub)
}

func TestBuildContentBlocks_TextOnly(t *testing.T) {
	parts := &shared.MessageParts{
		TextParts: []string{"hello world"},
	}
	blocks := dsh.ExportBuildContentBlocks(parts)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestBuildContentBlocks_WithImages(t *testing.T) {
	parts := &shared.MessageParts{
		TextParts:   []string{"describe this"},
		ImageBlocks: []shared.ImageBlock{{MediaType: "image/png", Data: "AAAA", Filename: "test.png"}},
	}
	blocks := dsh.ExportBuildContentBlocks(parts)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks (text+image), got %d", len(blocks))
	}
}

func TestBuildContentBlocks_SkipEmptyText(t *testing.T) {
	parts := &shared.MessageParts{
		TextParts:   []string{"", "hello"},
		ImageBlocks: []shared.ImageBlock{{MediaType: "image/jpeg", Data: "AAAA", Filename: ""}},
	}
	blocks := dsh.ExportBuildContentBlocks(parts)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks (non-empty text+image), got %d", len(blocks))
	}
}

func TestBuildSessionPromptMsgFromBlocks_JSON(t *testing.T) {
	parts := &shared.MessageParts{
		TextParts:   []string{"what is in this image?"},
		ImageBlocks: []shared.ImageBlock{{MediaType: "image/png", Data: "AAAA", Filename: "img.png"}},
	}
	blocks := dsh.ExportBuildContentBlocks(parts)
	msg, err := dsh.ExportBuildSessionPromptMsgFromBlocks("test-session-id", blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(msg, `"sessionId":"test-session-id"`) {
		t.Errorf("missing sessionId in output: %s", msg)
	}
	if !contains(msg, `"mediaType":"image/png"`) {
		t.Errorf("missing mediaType in output: %s", msg)
	}
	if !contains(msg, `"data":"AAAA"`) {
		t.Errorf("missing data in output: %s", msg)
	}
}

// --- ExtractProfile ---

func TestExtractProfile_Nil(t *testing.T) {
	p := llmprovider.ExtractProfile(nil)
	if p.APIKey != "" || p.BaseURL != "" || p.Model != "" {
		t.Errorf("nil connector: key=%q url=%q model=%q", p.APIKey, p.BaseURL, p.Model)
	}
	if len(p.Input) != 1 || p.Input[0] != "text" {
		t.Errorf("nil connector Input = %v", p.Input)
	}
}

func TestExtractProfile_Connection(t *testing.T) {
	c := &fakeLLMConn{
		typ:   connector.OPENAI,
		key:   "sk-test",
		url:   "https://api.deepseek.com",
		model: "deepseek-flash",
	}
	p := llmprovider.ExtractProfile(c)
	if p.APIKey != "sk-test" {
		t.Errorf("APIKey = %q", p.APIKey)
	}
	if p.BaseURL != "https://api.deepseek.com" {
		t.Errorf("BaseURL = %q", p.BaseURL)
	}
	if p.Model != "deepseek-flash" {
		t.Errorf("Model = %q", p.Model)
	}
}

func TestExtractProfile_PlainConnectorFallback(t *testing.T) {
	c := &fakeConnForThinking{
		settings: map[string]interface{}{
			"key":   "sk-plain",
			"host":  "https://example.com",
			"model": "test-model",
		},
	}
	p := llmprovider.ExtractProfile(c)
	if p.APIKey != "sk-plain" {
		t.Errorf("APIKey = %q", p.APIKey)
	}
	if p.BaseURL != "https://example.com" {
		t.Errorf("BaseURL = %q", p.BaseURL)
	}
	if p.Model != "test-model" {
		t.Errorf("Model = %q", p.Model)
	}
}

func TestExtractProfile_ContextWindowAndMaxTokens(t *testing.T) {
	c := &fakeLLMConn{
		caps: &goullm.Capabilities{
			MaxInputTokens:  1048576,
			MaxOutputTokens: 384000,
		},
	}
	p := llmprovider.ExtractProfile(c)
	if p.ContextWindow != 1048576 {
		t.Errorf("ContextWindow = %d", p.ContextWindow)
	}
	if p.MaxTokens != 384000 {
		t.Errorf("MaxTokens = %d", p.MaxTokens)
	}
}

func TestExtractProfile_NoCaps(t *testing.T) {
	c := &fakeLLMConn{}
	p := llmprovider.ExtractProfile(c)
	if p.ContextWindow != 0 || p.MaxTokens != 0 {
		t.Errorf("no caps: context=%d max=%d", p.ContextWindow, p.MaxTokens)
	}
}

func TestExtractProfile_Vision(t *testing.T) {
	c := &fakeLLMConn{caps: &goullm.Capabilities{Vision: true}}
	p := llmprovider.ExtractProfile(c)
	if len(p.Input) != 2 || p.Input[1] != "image" {
		t.Errorf("vision Input = %v", p.Input)
	}
}

func TestExtractProfile_API_Anthropic(t *testing.T) {
	c := &fakeLLMConn{typ: connector.ANTHROPIC}
	p := llmprovider.ExtractProfile(c)
	if p.API != llmprovider.APIAnthropicMessages {
		t.Errorf("API = %q", p.API)
	}
	if p.NoDeveloperRole {
		t.Error("Anthropic NoDeveloperRole should be false")
	}
}

func TestExtractProfile_API_OpenAI(t *testing.T) {
	c := &fakeLLMConn{typ: connector.OPENAI}
	p := llmprovider.ExtractProfile(c)
	if p.API != llmprovider.APIOpenAICompletions {
		t.Errorf("API = %q", p.API)
	}
	if !p.NoDeveloperRole {
		t.Error("OpenAI NoDeveloperRole should be true")
	}
}

func TestExtractProfile_API_ProtocolsOverride(t *testing.T) {
	c := &fakeLLMConn{
		typ: connector.OPENAI,
		settings: map[string]interface{}{
			"protocols": []interface{}{"openai", "anthropic"},
		},
	}
	p := llmprovider.ExtractProfile(c)
	if p.API != llmprovider.APIAnthropicMessages {
		t.Errorf("protocols should override to anthropic-messages, got %q", p.API)
	}
}

func TestExtractProfile_Reasoning_SettingPriority(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{"reasoning_effort": "low"},
		metadata: map[string]interface{}{"reasoning_effort": "high"},
	}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "low" {
		t.Errorf("Setting should take priority, Reasoning=%q", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_MetadataFallback(t *testing.T) {
	c := &fakeLLMConn{
		metadata: map[string]interface{}{"reasoning_effort": "max"},
	}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "max" {
		t.Errorf("Reasoning = %q, want max", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_NoneTranslation(t *testing.T) {
	c := &fakeLLMConn{settings: map[string]interface{}{"reasoning_effort": "none"}}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "off" {
		t.Errorf("none→off, got Reasoning=%q", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_ThinkingTranslation(t *testing.T) {
	c := &fakeLLMConn{metadata: map[string]interface{}{"reasoning_effort": "thinking"}}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "high" {
		t.Errorf("thinking→high, got Reasoning=%q", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_UnknownValueDefaultsHigh(t *testing.T) {
	c := &fakeLLMConn{settings: map[string]interface{}{"reasoning_effort": "adaptive"}}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "high" {
		t.Errorf("unknown value 'adaptive' should default to high, got Reasoning=%q", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_ThinkingTypeDisabled(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}},
	}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "off" {
		t.Errorf("disabled fallback→off, got Reasoning=%q", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_ThinkingTypeEnabled(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
	}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "high" {
		t.Errorf("enabled fallback→high, got Reasoning=%q", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_HasReasoningFallback(t *testing.T) {
	c := &fakeLLMConn{caps: &goullm.Capabilities{Reasoning: true}}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "high" {
		t.Errorf("hasReasoning fallback→high, got Reasoning=%q", p.Reasoning)
	}
}

func TestExtractProfile_Reasoning_NoPreference(t *testing.T) {
	c := &fakeLLMConn{}
	p := llmprovider.ExtractProfile(c)
	// No reasoning info at all → pure non-reasoning model
	if p.Reasoning != "off" {
		t.Errorf("no info→off, got Reasoning=%q", p.Reasoning)
	}
	if !p.DisableReasoning {
		t.Error("no info→DisableReasoning=true")
	}
}

func TestExtractProfile_BudgetTokens(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{
			"thinking": map[string]interface{}{"type": "enabled", "budget_tokens": float64(10000)},
		},
		metadata: map[string]interface{}{"reasoning_effort": "thinking"},
	}
	p := llmprovider.ExtractProfile(c)
	if p.BudgetTokens != 10000 {
		t.Errorf("BudgetTokens = %d", p.BudgetTokens)
	}
}

func TestExtractProfile_ThinkingFormat_MetadataDeclared(t *testing.T) {
	c := &fakeLLMConn{metadata: map[string]interface{}{"thinking_format": "deepseek"}}
	p := llmprovider.ExtractProfile(c)
	if p.ThinkingFormat != "deepseek" {
		t.Errorf("ThinkingFormat = %q", p.ThinkingFormat)
	}
}

func TestExtractProfile_ThinkingFormat_ClearedForAnthropic(t *testing.T) {
	c := &fakeLLMConn{
		typ:      connector.ANTHROPIC,
		settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
	}
	p := llmprovider.ExtractProfile(c)
	if p.ThinkingFormat != "" {
		t.Errorf("ThinkingFormat should be cleared for anthropic, got %q", p.ThinkingFormat)
	}
}

func TestExtractProfile_ReasoningEfforts_DeepSeek(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}, "reasoning_effort": "none"},
		metadata: map[string]interface{}{"reasoning_efforts": []interface{}{"none", "low", "high", "max"}, "reasoning_effort": "none"},
	}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "off" {
		t.Errorf("Reasoning = %q, want off", p.Reasoning)
	}
	// deepseek dialect: off wire="" (YAML null), plus one non-off for PI validation
	expected := map[string]string{"off": "", "high": "high"}
	if len(p.ReasoningEfforts) != len(expected) {
		t.Fatalf("ReasoningEfforts = %v, want %v", p.ReasoningEfforts, expected)
	}
	for k, v := range expected {
		if p.ReasoningEfforts[k] != v {
			t.Errorf("ReasoningEfforts[%q] = %q, want %q", k, p.ReasoningEfforts[k], v)
		}
	}
}

func TestExtractProfile_ReasoningEfforts_Claude(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
		metadata: map[string]interface{}{"reasoning_efforts": []interface{}{"none", "thinking"}, "reasoning_effort": "thinking"},
	}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "high" {
		t.Errorf("Reasoning = %q, want high", p.Reasoning)
	}
	if len(p.ReasoningEfforts) != 1 || p.ReasoningEfforts["high"] != "high" {
		t.Errorf("ReasoningEfforts = %v, want {high:high}", p.ReasoningEfforts)
	}
}

func TestExtractProfile_ReasoningEfforts_NoneOnly(t *testing.T) {
	c := &fakeLLMConn{
		metadata: map[string]interface{}{"reasoning_efforts": []interface{}{"none"}},
	}
	p := llmprovider.ExtractProfile(c)
	// No reasoning_effort set → p.Reasoning="" → falls through to inferReasoningEfforts
	if p.ReasoningEfforts != nil {
		t.Errorf("ReasoningEfforts should be nil for none-only, got %v", p.ReasoningEfforts)
	}
	if !p.DisableReasoning {
		t.Error("DisableReasoning should be true")
	}
	if p.Reasoning != "off" {
		t.Errorf("Reasoning should be off, got %q", p.Reasoning)
	}
}

func TestExtractProfile_ReasoningEfforts_NilMetadata(t *testing.T) {
	c := &fakeLLMConn{}
	p := llmprovider.ExtractProfile(c)
	// No metadata, no reasoning hints → pure non-reasoning
	if p.ReasoningEfforts != nil {
		t.Errorf("ReasoningEfforts should be nil, got %v", p.ReasoningEfforts)
	}
	if !p.DisableReasoning {
		t.Error("DisableReasoning should be true")
	}
}

func TestExtractProfile_ReasoningEfforts_ThinkingOnly(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
		metadata: map[string]interface{}{"reasoning_efforts": []interface{}{"thinking"}, "reasoning_effort": "thinking"},
	}
	p := llmprovider.ExtractProfile(c)
	if p.DisableReasoning {
		t.Error("thinking variant should NOT disable reasoning")
	}
	if p.ReasoningEfforts == nil || len(p.ReasoningEfforts) != 1 || p.ReasoningEfforts["high"] != "high" {
		t.Errorf("ReasoningEfforts = %v, want {high:high}", p.ReasoningEfforts)
	}
}

func TestExtractProfile_ReasoningEfforts_NoNone(t *testing.T) {
	c := &fakeLLMConn{
		metadata: map[string]interface{}{"reasoning_efforts": []interface{}{"low", "high", "max"}, "reasoning_effort": "high"},
	}
	p := llmprovider.ExtractProfile(c)
	// Active variant at high: only {high:high}
	if _, ok := p.ReasoningEfforts["off"]; ok {
		t.Error("active variant should not have off entry")
	}
	if len(p.ReasoningEfforts) != 1 || p.ReasoningEfforts["high"] != "high" {
		t.Errorf("ReasoningEfforts = %v, want {high:high}", p.ReasoningEfforts)
	}
}

func TestExtractProfile_Protocols(t *testing.T) {
	c := &fakeLLMConn{
		settings: map[string]interface{}{"protocols": []interface{}{"openai", "anthropic"}},
	}
	p := llmprovider.ExtractProfile(c)
	if len(p.Protocols) != 2 || p.Protocols[0] != "openai" || p.Protocols[1] != "anthropic" {
		t.Errorf("Protocols = %v", p.Protocols)
	}
}

// --- Full model scenarios ---

func TestExtractProfile_DeepSeekV4FlashThinking(t *testing.T) {
	c := &fakeLLMConn{
		typ:   connector.OPENAI,
		key:   "sk-ds",
		url:   "https://api.deepseek.com",
		model: "deepseek-v4-flash",
		settings: map[string]interface{}{
			"thinking": map[string]interface{}{"type": "enabled"},
		},
		metadata: map[string]interface{}{
			"reasoning_effort":  "high",
			"reasoning_efforts": []interface{}{"none", "low", "high", "max"},
			"thinking_format":   "deepseek",
		},
		caps: &goullm.Capabilities{Reasoning: true, MaxInputTokens: 1048576, MaxOutputTokens: 384000},
	}
	p := llmprovider.ExtractProfile(c)
	if p.Reasoning != "high" {
		t.Errorf("Reasoning = %q", p.Reasoning)
	}
	if p.ContextWindow != 1048576 || p.MaxTokens != 384000 {
		t.Errorf("context=%d max=%d", p.ContextWindow, p.MaxTokens)
	}
	if p.ThinkingFormat != "deepseek" {
		t.Errorf("ThinkingFormat = %q", p.ThinkingFormat)
	}
	// Active variant at high: only {high:high}
	if len(p.ReasoningEfforts) != 1 || p.ReasoningEfforts["high"] != "high" {
		t.Errorf("ReasoningEfforts = %v, want {high:high}", p.ReasoningEfforts)
	}
}

func TestExtractProfile_ClaudeWithBudget(t *testing.T) {
	c := &fakeLLMConn{
		typ:   connector.ANTHROPIC,
		key:   "sk-ant",
		url:   "https://api.anthropic.com",
		model: "claude-sonnet-4-20250514",
		settings: map[string]interface{}{
			"thinking": map[string]interface{}{"type": "enabled", "budget_tokens": float64(10000)},
		},
		metadata: map[string]interface{}{
			"reasoning_effort":  "thinking",
			"reasoning_efforts": []interface{}{"none", "thinking"},
		},
		caps: &goullm.Capabilities{Reasoning: true, Vision: true, MaxInputTokens: 1000000, MaxOutputTokens: 64000},
	}
	p := llmprovider.ExtractProfile(c)
	if p.API != llmprovider.APIAnthropicMessages {
		t.Errorf("API = %q", p.API)
	}
	if p.BudgetTokens != 10000 {
		t.Errorf("BudgetTokens = %d", p.BudgetTokens)
	}
	if p.ThinkingFormat != "" {
		t.Errorf("ThinkingFormat should be cleared, got %q", p.ThinkingFormat)
	}
	if p.ContextWindow != 1000000 {
		t.Errorf("ContextWindow = %d", p.ContextWindow)
	}
	// Active variant at high: only {high:high}
	if len(p.ReasoningEfforts) != 1 || p.ReasoningEfforts["high"] != "high" {
		t.Errorf("ReasoningEfforts = %v, want {high:high}", p.ReasoningEfforts)
	}
}

func TestResolveMaxInstructionBytes(t *testing.T) {
	cases := []struct {
		ctx  int
		want int
	}{
		{0, 65536},
		{128000, 65536},
		{500000, 131072},
		{1000000, 131072},
	}
	for _, tc := range cases {
		got := dsh.ExportResolveMaxInstructionBytes(tc.ctx)
		if got != tc.want {
			t.Errorf("resolveMaxInstructionBytes(%d) = %d, want %d", tc.ctx, got, tc.want)
		}
	}
}

func TestRenderCordisConfig_DynamicMaxBytes(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		MaxInstructionBytes: 131072,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "maxBytes: 131072") {
		t.Errorf("expected maxBytes: 131072 in config, got:\n%s", s)
	}
}

func TestRenderCordisConfig_DefaultMaxBytes(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "maxBytes: 65536") {
		t.Errorf("expected maxBytes: 65536 (default) in config, got:\n%s", s)
	}
}
