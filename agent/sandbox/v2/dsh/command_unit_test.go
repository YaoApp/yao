//go:build unit

package dsh_test

import (
	"os"
	"strings"
	"testing"

	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/dsh"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// --- fakeConnForThinking implements connector.Connector for connectorThinkingFormat tests ---

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

// --- connectorSetting ---

func TestConnectorSetting_Nil(t *testing.T) {
	if got := dsh.ExportConnectorSetting(nil, "key"); got != "" {
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
	if got != "You are an agent.\n\nWorking directory: /workspace" {
		t.Errorf("prompt = %q", got)
	}
}

func TestBuildSystemPrompt_NoLocale(t *testing.T) {
	req := &types.StreamRequest{
		Config:       &types.SandboxConfig{},
		SystemPrompt: "You are an agent.",
	}
	got := dsh.ExportBuildSystemPrompt(req, "/workspace")
	if got != "You are an agent.\n\nWorking directory: /workspace" {
		t.Errorf("prompt = %q", got)
	}
}

func TestBuildSystemPrompt_EnLocale_NoExtra(t *testing.T) {
	req := &types.StreamRequest{
		Config:       &types.SandboxConfig{},
		SystemPrompt: "Agent",
		Locale:       "en-us",
	}
	got := dsh.ExportBuildSystemPrompt(req, "/workspace")
	if got != "Agent\n\nWorking directory: /workspace" {
		t.Errorf("prompt = %q (should not have locale suffix)", got)
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
			Models:    []dsh.PiAiModelConfig{{ID: "gpt-5.5", Input: []string{"text"}, Reasoning: true}},
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
		"https://api.openai.com/v1", "DSH_KEY_PRIMARY", "gpt-5.5", "reasoning: high") {
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
			Models:    []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text", "image"}, Reasoning: true}},
			Reasoning: "max",
		}},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "reasoningEfforts:", "off: none", "low: low", "medium: medium", "high: high", "max: max") {
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
			Models:    []dsh.PiAiModelConfig{{ID: "some-model", Input: []string{"text"}, Reasoning: false}},
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
			Name:         "yao-primary",
			API:          "openai-completions",
			BaseURL:      "https://proxy.example.com/v1",
			APIKeyEnv:    "DSH_KEY_PRIMARY",
			Models:       []dsh.PiAiModelConfig{{ID: "deepseek-chat", Input: []string{"text", "image"}, Reasoning: true}},
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
				Models:    []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text"}, Reasoning: true}},
				Reasoning: "high",
			},
			{
				Name:      "yao-heavy",
				API:       "openai-completions",
				BaseURL:   "https://api.yaoagents.com/v1",
				APIKeyEnv: "DSH_KEY_HEAVY",
				Models:    []dsh.PiAiModelConfig{{ID: "deepseek-chat", Input: []string{"text"}, Reasoning: true}},
				Reasoning: "max",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "yao-primary:", "yao-heavy:", "deepseek-flash", "deepseek-chat",
		"reasoning: high", "reasoning: max") {
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
			Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text", "image"}, Reasoning: true}},
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
			Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text"}, Reasoning: true}},
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
			Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text"}, Reasoning: true}},
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
			Models:         []dsh.PiAiModelConfig{{ID: "claude-sonnet-4", Input: []string{"text"}, Reasoning: true}},
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

// --- connectorThinkingFormat matrix ---

func TestConnectorThinkingFormat(t *testing.T) {
	cases := []struct {
		name     string
		settings map[string]interface{}
		metadata map[string]interface{}
		want     string
	}{
		{
			name: "nil_connector",
			want: "",
		},
		{
			name:     "no_metadata_no_thinking_settings",
			settings: map[string]interface{}{"host": "example.com"},
			want:     "",
		},
		{
			name:     "metadata_declares_deepseek",
			metadata: map[string]interface{}{"thinking_format": "deepseek"},
			want:     "deepseek",
		},
		{
			name:     "metadata_overrides_settings",
			metadata: map[string]interface{}{"thinking_format": "other-format"},
			settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
			want:     "other-format",
		},
		{
			name:     "settings_thinking_type_enabled",
			settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
			want:     "deepseek",
		},
		{
			name:     "settings_thinking_type_disabled",
			settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}},
			want:     "deepseek",
		},
		{
			name:     "empty_metadata_falls_through_to_settings",
			metadata: map[string]interface{}{},
			settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
			want:     "deepseek",
		},
		{
			name:     "thinking_map_without_type_key",
			settings: map[string]interface{}{"thinking": map[string]interface{}{"budget_tokens": 32000}},
			want:     "",
		},
		{
			name:     "metadata_empty_string_falls_through",
			metadata: map[string]interface{}{"thinking_format": ""},
			settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
			want:     "deepseek",
		},
		{
			name:     "metadata_non_string_ignored",
			metadata: map[string]interface{}{"thinking_format": 123},
			settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
			want:     "deepseek",
		},
		{
			name:     "settings_thinking_not_a_map",
			settings: map[string]interface{}{"thinking": "enabled"},
			want:     "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.settings == nil && c.metadata == nil && c.name == "nil_connector" {
				if got := dsh.ExportConnectorThinkingFormat(nil); got != c.want {
					t.Errorf("connectorThinkingFormat(nil) = %q, want %q", got, c.want)
				}
				return
			}
			conn := &fakeConnForThinking{settings: c.settings, metadata: c.metadata}
			if got := dsh.ExportConnectorThinkingFormat(conn); got != c.want {
				t.Errorf("connectorThinkingFormat() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestNonAnthropicThinkingFormat(t *testing.T) {
	connWithThinking := &fakeConnForThinking{
		settings: map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}},
	}
	connWithoutThinking := &fakeConnForThinking{
		settings: map[string]interface{}{"host": "example.com"},
	}

	cases := []struct {
		name        string
		isAnthropic bool
		conn        *fakeConnForThinking
		want        string
	}{
		{"openai_with_thinking", false, connWithThinking, "deepseek"},
		{"openai_without_thinking", false, connWithoutThinking, ""},
		{"anthropic_with_thinking", true, connWithThinking, ""},
		{"anthropic_without_thinking", true, connWithoutThinking, ""},
		{"nil_connector", false, nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got string
			if c.conn == nil {
				got = dsh.ExportNonAnthropicThinkingFormat(c.isAnthropic, nil)
			} else {
				got = dsh.ExportNonAnthropicThinkingFormat(c.isAnthropic, c.conn)
			}
			if got != c.want {
				t.Errorf("nonAnthropicThinkingFormat(anthropic=%v) = %q, want %q", c.isAnthropic, got, c.want)
			}
		})
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
			Models:    []dsh.PiAiModelConfig{{ID: "claude-sonnet-4", Input: []string{"text", "image"}, Reasoning: true}},
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
				Models:          []dsh.PiAiModelConfig{{ID: "deepseek-flash", Input: []string{"text"}, Reasoning: true}},
				Reasoning:       "high",
				NoDeveloperRole: true,
			},
			{
				Name:      "yao-heavy",
				API:       "anthropic-messages",
				BaseURL:   "https://api.anthropic.com",
				APIKeyEnv: "DSH_KEY_HEAVY",
				Models:    []dsh.PiAiModelConfig{{ID: "claude-sonnet-4", Input: []string{"text", "image"}, Reasoning: true}},
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

// --- Thinking/Reasoning mapping ---

func TestNormalizePiAiReasoning(t *testing.T) {
	cases := []struct{ in, want string }{
		{"off", "off"}, {"none", "off"},
		{"", ""},
		{"low", "low"}, {"medium", "medium"}, {"high", "high"},
		{"xhigh", "xhigh"}, {"max", "max"},
		{"unknown", "high"},
	}
	for _, c := range cases {
		if got := dsh.ExportNormalizePiAiReasoning(c.in); got != c.want {
			t.Errorf("normalizePiAiReasoning(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToPiAiThinking_EffortValues(t *testing.T) {
	cases := []struct {
		effort        string
		budgetTokens  int
		wantReasoning string
		wantBudget    int
	}{
		{"none", 0, "off", 0},
		{"thinking", 32000, "high", 32000},
		{"thinking", 0, "high", 0},
		{"low", 0, "low", 0},
		{"medium", 0, "medium", 0},
		{"high", 0, "high", 0},
		{"xhigh", 0, "xhigh", 0},
		{"max", 0, "max", 0},
	}
	for _, c := range cases {
		info := dsh.NewExportReasoningInfo(c.effort, c.budgetTokens, "", false)
		reasoning, budget := dsh.ExportToPiAiThinking(info)
		if reasoning != c.wantReasoning || budget != c.wantBudget {
			t.Errorf("toPiAiThinking(effort=%q, budget=%d) = (%q, %d), want (%q, %d)",
				c.effort, c.budgetTokens, reasoning, budget, c.wantReasoning, c.wantBudget)
		}
	}
}

func TestToPiAiThinking_LegacyFallback(t *testing.T) {
	info := dsh.NewExportReasoningInfo("", 0, "disabled", false)
	reasoning, budget := dsh.ExportToPiAiThinking(info)
	if reasoning != "off" || budget != 0 {
		t.Errorf("disabled fallback = (%q, %d)", reasoning, budget)
	}

	info = dsh.NewExportReasoningInfo("", 16000, "enabled", false)
	reasoning, budget = dsh.ExportToPiAiThinking(info)
	if reasoning != "high" || budget != 16000 {
		t.Errorf("enabled fallback = (%q, %d)", reasoning, budget)
	}

	info = dsh.NewExportReasoningInfo("", 0, "", true)
	reasoning, budget = dsh.ExportToPiAiThinking(info)
	if reasoning != "high" || budget != 0 {
		t.Errorf("capabilities fallback = (%q, %d)", reasoning, budget)
	}

	info = dsh.NewExportReasoningInfo("", 0, "", false)
	reasoning, budget = dsh.ExportToPiAiThinking(info)
	if reasoning != "" || budget != 0 {
		t.Errorf("empty fallback = (%q, %d)", reasoning, budget)
	}
}

func TestNormalizePiAiBaseURL(t *testing.T) {
	cases := []struct{ in, want string }{
		// No trailing "/" → append /v1 (same as BuildAPIURL convention)
		{"https://api.deepseek.com", "https://api.deepseek.com/v1"},
		{"https://api.yaoagents.com", "https://api.yaoagents.com/v1"},
		{"https://api.moonshot.cn", "https://api.moonshot.cn/v1"},
		{"http://127.0.0.1:8080", "http://127.0.0.1:8080/v1"},
		// Trailing "/" → user-specified base path, strip / only
		{"https://ark.cn-beijing.volces.com/api/v3/", "https://ark.cn-beijing.volces.com/api/v3"},
		{"https://api.yaoagents.com/v1/", "https://api.yaoagents.com/v1"},
		{"https://maas.example.com/api/", "https://maas.example.com/api"},
	}
	for _, c := range cases {
		if got := dsh.ExportNormalizePiAiBaseURL(c.in); got != c.want {
			t.Errorf("normalizePiAiBaseURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeAnthropicBaseURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://api.anthropic.com", "https://api.anthropic.com"},
		{"https://api.anthropic.com/", "https://api.anthropic.com"},
		{"https://api.yaoagents.com/anthropic", "https://api.yaoagents.com/anthropic"},
		{"https://api.yaoagents.com/anthropic/", "https://api.yaoagents.com/anthropic"},
	}
	for _, c := range cases {
		if got := dsh.ExportNormalizeAnthropicBaseURL(c.in); got != c.want {
			t.Errorf("normalizeAnthropicBaseURL(%q) = %q, want %q", c.in, got, c.want)
		}
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
