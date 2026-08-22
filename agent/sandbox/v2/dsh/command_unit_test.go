//go:build unit

package dsh_test

import (
	"os"
	"strings"
	"testing"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/dsh"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

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
	if got != "You are an agent.\n\nWorking directory: /workspace\n\nAlways respond in Chinese (Simplified)." {
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
	if !containsAll(s, "dsh-yaoapp-jsonrpc-stream", "dsh-llm-deepseek", "dsh-bash-local", "dsh-fs-local") {
		t.Errorf("config missing expected plugins: %s", s)
	}
	if !containsAll(s, "session-persistence-jsonl", "session-checkpoint-policy") {
		t.Error("config should contain session persistence plugins")
	}
}

func TestRenderCordisConfig_WithBaseURL(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		BaseURL: "https://api.custom.com/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "baseURL: https://api.custom.com/v1") {
		t.Errorf("config = %s", s)
	}
}

func TestRenderCordisConfig_WithMaxTokens(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		MaxTokens: 8192,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "maxTokens: 8192") {
		t.Errorf("config = %s", s)
	}
}

func TestRenderCordisConfig_ThinkingDisabled(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		Thinking: "disabled",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "thinking: disabled") {
		t.Errorf("config should have thinking: disabled, got: %s", s)
	}
}

func TestRenderCordisConfig_ThinkingEnabled(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		Thinking:        "enabled",
		ReasoningEffort: "medium",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "thinking: enabled", "reasoningEffort: medium") {
		t.Errorf("config = %s", s)
	}
}

func TestRenderCordisConfig_ThinkingDefaultsToEnabled(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "thinking: enabled") {
		t.Errorf("empty config should default thinking to enabled, got: %s", s)
	}
}

func TestRenderCordisConfig_NoBaseURL(t *testing.T) {
	data, _ := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{})
	s := string(data)
	if containsAny(s, "baseURL:") {
		t.Error("should not contain baseURL when empty")
	}
}

func TestRenderCordisConfig_WithModelsTextOnly(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		Models: []dsh.ModelConfig{
			{ID: "deepseek-chat", InputModalities: []string{"text"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "models:", "id: deepseek-chat", "inputModalities: [text]") {
		t.Errorf("config should render models section, got: %s", s)
	}
	if containsAny(s, "dsh-attachment-local") {
		t.Error("text-only models should not load attachment-local")
	}
}

func TestRenderCordisConfig_WithVision(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		Models: []dsh.ModelConfig{
			{ID: "deepseek-v4-flash", InputModalities: []string{"text", "image"}},
		},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "models:", "id: deepseek-v4-flash", "inputModalities: [text, image]") {
		t.Errorf("config should render vision model, got: %s", s)
	}
	if !containsAll(s, "dsh-attachment-local") {
		t.Error("vision=true should load attachment-local plugin")
	}
}

func TestRenderCordisConfig_MultipleModels(t *testing.T) {
	data, err := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{
		Models: []dsh.ModelConfig{
			{ID: "deepseek-chat", InputModalities: []string{"text"}},
			{ID: "deepseek-v4-flash", InputModalities: []string{"text", "image"}},
		},
		Vision: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !containsAll(s, "id: deepseek-chat", "id: deepseek-v4-flash") {
		t.Errorf("config should contain both models, got: %s", s)
	}
	if !containsAll(s, "dsh-attachment-local") {
		t.Error("should load attachment-local when vision=true")
	}
}

func TestRenderCordisConfig_NoModels(t *testing.T) {
	data, _ := dsh.ExportRenderCordisConfig(&dsh.ConnectorConfig{})
	s := string(data)
	if containsAny(s, "models:", "inputModalities:") {
		t.Error("empty Models should not render models section")
	}
	if containsAny(s, "dsh-attachment-local") {
		t.Error("empty config should not load attachment-local")
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
