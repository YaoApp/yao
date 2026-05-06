package opencode

import (
	"encoding/json"
	"testing"

	"github.com/yaoapp/gou/connector"
	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

type fakeConn struct {
	id       string
	typ      int
	settings map[string]interface{}
}

func (f *fakeConn) Register(string, string, []byte) error { return nil }
func (f *fakeConn) Query() (query.Query, error)           { return nil, nil }
func (f *fakeConn) Schema() (schema.Schema, error)        { return nil, nil }
func (f *fakeConn) Close() error                          { return nil }
func (f *fakeConn) ID() string                            { return f.id }
func (f *fakeConn) Is(t int) bool                         { return f.typ == t }
func (f *fakeConn) Setting() map[string]interface{}       { return f.settings }
func (f *fakeConn) GetMetaInfo() gouTypes.MetaInfo        { return gouTypes.MetaInfo{} }

func newFakeOpenAI(id, host, model, key string) *fakeConn {
	return &fakeConn{
		id:  id,
		typ: connector.OPENAI,
		settings: map[string]interface{}{
			"host":  host,
			"model": model,
			"key":   key,
		},
	}
}

func newFakeAnthropic(id, host, model, key string) *fakeConn {
	return &fakeConn{
		id:  id,
		typ: connector.ANTHROPIC,
		settings: map[string]interface{}{
			"host":  host,
			"model": model,
			"key":   key,
		},
	}
}

// ---------------------------------------------------------------------------
// injectRoleProviders tests
// ---------------------------------------------------------------------------

func TestInjectRoleProviders_VisionCustomProvider(t *testing.T) {
	visionConn := newFakeOpenAI("vis", "https://api.mymaas.com/v1", "gpt-4o-mini", "sk-vis")

	primaryConn := newFakeOpenAI("primary", "https://api.deepseek.com", "deepseek-v4-flash", "sk-ds")
	cfg := map[string]any{
		"provider":          map[string]any{"custom": map[string]any{"npm": "@ai-sdk/openai-compatible"}},
		"model":             "custom/deepseek-v4-flash",
		"enabled_providers": []string{"custom"},
	}

	req := &types.PrepareRequest{
		Connector: primaryConn,
		Config:    &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"default": primaryConn,
			"vision":  visionConn,
		},
	}

	injectRoleProviders(cfg, req, primaryConn)

	providers := cfg["provider"].(map[string]any)
	visionBlock, ok := providers["vision"]
	if !ok {
		t.Fatal("should have injected 'vision' provider block")
	}

	vBlock := visionBlock.(map[string]any)
	models := vBlock["models"].(map[string]any)
	if _, ok := models["gpt-4o-mini"]; !ok {
		t.Fatal("vision provider should contain gpt-4o-mini model")
	}

	enabled := cfg["enabled_providers"].([]string)
	hasVision := false
	for _, e := range enabled {
		if e == "vision" {
			hasVision = true
		}
	}
	if !hasVision {
		t.Error("enabled_providers should contain 'vision'")
	}
}

func TestInjectRoleProviders_VisionNativeOpenAI(t *testing.T) {
	visionConn := newFakeOpenAI("vis", "", "gpt-4o-mini", "sk-oai")

	primaryConn := newFakeOpenAI("primary", "https://api.deepseek.com", "deepseek-v4-flash", "sk-ds")
	cfg := map[string]any{
		"provider":          map[string]any{"custom": map[string]any{"npm": "@ai-sdk/openai-compatible"}},
		"model":             "custom/deepseek-v4-flash",
		"enabled_providers": []string{"custom"},
	}

	req := &types.PrepareRequest{
		Connector: primaryConn,
		Config:    &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"default": primaryConn,
			"vision":  visionConn,
		},
	}

	injectRoleProviders(cfg, req, primaryConn)

	providers := cfg["provider"].(map[string]any)
	if _, ok := providers["vision"]; !ok {
		t.Fatal("should have separate 'vision' provider (different host from primary)")
	}
}

func TestInjectRoleProviders_LightWithDifferentHost(t *testing.T) {
	lightConn := newFakeOpenAI("moonshot", "https://api.moonshot.cn/v1", "moonshot-v1-8k", "sk-moon")

	primaryConn := newFakeOpenAI("primary", "https://api.deepseek.com", "deepseek-v4-flash", "sk-ds")
	cfg := map[string]any{
		"provider":          map[string]any{"custom": map[string]any{"npm": "@ai-sdk/openai-compatible"}},
		"model":             "custom/deepseek-v4-flash",
		"enabled_providers": []string{"custom"},
	}

	req := &types.PrepareRequest{
		Connector: primaryConn,
		Config:    &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"default": primaryConn,
			"light":   lightConn,
		},
	}

	injectRoleProviders(cfg, req, primaryConn)

	providers := cfg["provider"].(map[string]any)
	if _, ok := providers["light"]; !ok {
		t.Fatal("light role should have its own provider block when host differs from primary")
	}

	smallModel, ok := cfg["small_model"].(string)
	if !ok || smallModel == "" {
		t.Fatal("small_model should be set for light role")
	}
	if smallModel != "light/moonshot-v1-8k" {
		t.Errorf("small_model = %q, want 'light/moonshot-v1-8k'", smallModel)
	}

	enabled := cfg["enabled_providers"].([]string)
	hasLight := false
	for _, e := range enabled {
		if e == "light" {
			hasLight = true
		}
	}
	if !hasLight {
		t.Error("enabled_providers should contain 'light'")
	}
}

func TestInjectRoleProviders_LightSameHostAsPrimary(t *testing.T) {
	lightConn := newFakeOpenAI("ds-light", "https://api.deepseek.com", "deepseek-chat", "sk-ds")

	primaryConn := newFakeOpenAI("primary", "https://api.deepseek.com", "deepseek-v4-flash", "sk-ds")
	primaryProviderID, primaryCfg, modelStr := buildProviderConfig(primaryConn)

	cfg := map[string]any{
		"provider":          map[string]any{primaryProviderID: primaryCfg},
		"model":             modelStr,
		"enabled_providers": []string{primaryProviderID},
	}

	req := &types.PrepareRequest{
		Connector: primaryConn,
		Config:    &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"default": primaryConn,
			"light":   lightConn,
		},
	}

	injectRoleProviders(cfg, req, primaryConn)

	providers := cfg["provider"].(map[string]any)
	if _, ok := providers["light"]; ok {
		t.Error("light should merge into primary block when same host, not create separate block")
	}

	customBlock := providers["custom"].(map[string]any)
	models := customBlock["models"].(map[string]any)
	if _, ok := models["deepseek-chat"]; !ok {
		t.Error("light model should be merged into primary's 'custom' provider models")
	}

	smallModel := cfg["small_model"].(string)
	if smallModel != "custom/deepseek-chat" {
		t.Errorf("small_model = %q, want 'custom/deepseek-chat'", smallModel)
	}
}

func TestInjectRoleProviders_NoConnectors(t *testing.T) {
	cfg := map[string]any{
		"provider":          map[string]any{"openai": map[string]any{}},
		"model":             "openai/gpt-4o",
		"enabled_providers": []string{"openai"},
	}

	primaryConn := newFakeOpenAI("primary", "", "gpt-4o", "sk-oai")
	req := &types.PrepareRequest{
		Connector: primaryConn,
		Config:    &types.SandboxConfig{},
	}

	injectRoleProviders(cfg, req, primaryConn)

	enabled := cfg["enabled_providers"].([]string)
	if len(enabled) != 1 || enabled[0] != "openai" {
		t.Errorf("enabled_providers should be unchanged: %v", enabled)
	}
}

func TestInjectRoleProviders_AnthropicVision(t *testing.T) {
	visionConn := newFakeAnthropic("claude-vis", "https://api.anthropic.com", "claude-sonnet-4-5-20250929", "sk-ant")

	primaryConn := newFakeOpenAI("primary", "https://api.deepseek.com", "deepseek-v4-flash", "sk-ds")
	cfg := map[string]any{
		"provider":          map[string]any{"custom": map[string]any{"npm": "@ai-sdk/openai-compatible"}},
		"model":             "custom/deepseek-v4-flash",
		"enabled_providers": []string{"custom"},
	}

	req := &types.PrepareRequest{
		Connector: primaryConn,
		Config:    &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"default": primaryConn,
			"vision":  visionConn,
		},
	}

	injectRoleProviders(cfg, req, primaryConn)

	providers := cfg["provider"].(map[string]any)
	visionBlock, ok := providers["vision"]
	if !ok {
		t.Fatal("should inject 'vision' provider for Anthropic connector")
	}

	vBlock := visionBlock.(map[string]any)
	if vBlock["npm"] != nil {
		t.Error("Anthropic provider should NOT have npm field")
	}
}

// ---------------------------------------------------------------------------
// buildEnv role injection tests
// ---------------------------------------------------------------------------

func TestInjectRoleEnvVars_Vision(t *testing.T) {
	visionConn := newFakeOpenAI("vis", "https://api.mymaas.com/v1", "gpt-4o-mini", "sk-vis-key")

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"vision": visionConn,
		},
	}

	env := map[string]string{}
	injectRoleEnvVars(env, req)

	if env["YAO_VISION_KEY"] != "sk-vis-key" {
		t.Errorf("YAO_VISION_KEY = %q, want 'sk-vis-key'", env["YAO_VISION_KEY"])
	}
	if env["YAO_VISION_BASE_URL"] != "https://api.mymaas.com/v1" {
		t.Errorf("YAO_VISION_BASE_URL = %q, want 'https://api.mymaas.com/v1'", env["YAO_VISION_BASE_URL"])
	}
	if env["YAO_VISION_MODEL"] != "gpt-4o-mini" {
		t.Errorf("YAO_VISION_MODEL = %q, want 'gpt-4o-mini'", env["YAO_VISION_MODEL"])
	}
}

func TestInjectRoleEnvVars_Light(t *testing.T) {
	lightConn := newFakeOpenAI("moon", "https://api.moonshot.cn/v1", "moonshot-v1-8k", "sk-moon")

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"light": lightConn,
		},
	}

	env := map[string]string{}
	injectRoleEnvVars(env, req)

	if env["YAO_LIGHT_KEY"] != "sk-moon" {
		t.Errorf("YAO_LIGHT_KEY = %q, want 'sk-moon'", env["YAO_LIGHT_KEY"])
	}
	if env["YAO_LIGHT_BASE_URL"] != "https://api.moonshot.cn/v1" {
		t.Errorf("YAO_LIGHT_BASE_URL = %q, want 'https://api.moonshot.cn/v1'", env["YAO_LIGHT_BASE_URL"])
	}
	if env["YAO_LIGHT_MODEL"] != "moonshot-v1-8k" {
		t.Errorf("YAO_LIGHT_MODEL = %q, want 'moonshot-v1-8k'", env["YAO_LIGHT_MODEL"])
	}
}

func TestInjectRoleEnvVars_NoConnectors(t *testing.T) {
	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
	}

	env := map[string]string{}
	injectRoleEnvVars(env, req)

	for _, prefix := range []string{"YAO_VISION", "YAO_LIGHT"} {
		for _, suffix := range []string{"_KEY", "_BASE_URL", "_MODEL"} {
			if v, ok := env[prefix+suffix]; ok {
				t.Errorf("unexpected env %s=%s with no connectors", prefix+suffix, v)
			}
		}
	}
}

func TestInjectRoleEnvVars_MultipleRoles(t *testing.T) {
	visionConn := newFakeOpenAI("vis", "https://api.vision.com", "vis-model", "sk-vis")
	lightConn := newFakeOpenAI("light-c", "https://api.light.com", "light-model", "sk-light")

	req := &types.StreamRequest{
		Config: &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"vision": visionConn,
			"light":  lightConn,
		},
	}

	env := map[string]string{}
	injectRoleEnvVars(env, req)

	if env["YAO_VISION_KEY"] != "sk-vis" {
		t.Errorf("YAO_VISION_KEY = %q", env["YAO_VISION_KEY"])
	}
	if env["YAO_LIGHT_KEY"] != "sk-light" {
		t.Errorf("YAO_LIGHT_KEY = %q", env["YAO_LIGHT_KEY"])
	}
}

// ---------------------------------------------------------------------------
// Full integration: buildOpenCodeConfig with role connectors
// ---------------------------------------------------------------------------

func TestBuildOpenCodeConfig_WithVisionAndLight(t *testing.T) {
	visionConn := newFakeOpenAI("vis", "https://api.mymaas.com/v1", "gpt-4o-mini", "sk-vis")
	lightConn := newFakeOpenAI("moon", "https://api.moonshot.cn/v1", "moonshot-v1-8k", "sk-moon")

	primaryConn := newFakeOpenAI("primary", "https://api.deepseek.com", "deepseek-v4-flash", "sk-ds")
	req := &types.PrepareRequest{
		AssistantID: "test-assistant",
		Connector:   primaryConn,
		Config:      &types.SandboxConfig{},
		Roles: map[string]connector.Connector{
			"default": primaryConn,
			"vision":  visionConn,
			"light":   lightConn,
		},
	}

	data := buildOpenCodeConfig(req, nil)
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	providers := cfg["provider"].(map[string]any)

	if _, ok := providers["custom"]; !ok {
		t.Error("should have 'custom' provider for primary DeepSeek")
	}
	if _, ok := providers["vision"]; !ok {
		t.Error("should have 'vision' provider block")
	}
	if _, ok := providers["light"]; !ok {
		t.Error("should have 'light' provider block (different host from primary)")
	}

	if cfg["model"] != "custom/deepseek-v4-flash" {
		t.Errorf("model = %v, want custom/deepseek-v4-flash", cfg["model"])
	}
	if cfg["small_model"] != "light/moonshot-v1-8k" {
		t.Errorf("small_model = %v, want light/moonshot-v1-8k", cfg["small_model"])
	}

	enabled := cfg["enabled_providers"].([]any)
	enabledSet := map[string]bool{}
	for _, e := range enabled {
		enabledSet[e.(string)] = true
	}
	for _, want := range []string{"custom", "vision", "light"} {
		if !enabledSet[want] {
			t.Errorf("enabled_providers should contain %q", want)
		}
	}
}

// ---------------------------------------------------------------------------
// resolvePrimaryConnector tests
// ---------------------------------------------------------------------------

func TestResolvePrimaryConnector_HeavyConfigured(t *testing.T) {
	defaultConn := newFakeOpenAI("default", "https://api.deepseek.com", "deepseek-chat", "sk-ds")
	heavyConn := newFakeOpenAI("heavy", "https://api.openai.com", "o3-pro", "sk-oai")

	roles := map[string]connector.Connector{
		"default": defaultConn,
		"heavy":   heavyConn,
	}

	result := resolvePrimaryConnector(defaultConn, roles)
	if result != heavyConn {
		t.Error("should return heavy connector when configured")
	}
}

func TestResolvePrimaryConnector_NoHeavy(t *testing.T) {
	defaultConn := newFakeOpenAI("default", "https://api.deepseek.com", "deepseek-chat", "sk-ds")
	lightConn := newFakeOpenAI("light", "https://api.moonshot.cn", "moon-v1", "sk-moon")

	roles := map[string]connector.Connector{
		"default": defaultConn,
		"light":   lightConn,
	}

	result := resolvePrimaryConnector(defaultConn, roles)
	if result != defaultConn {
		t.Error("should fallback to default when heavy not configured")
	}
}

func TestResolvePrimaryConnector_NilRoles(t *testing.T) {
	defaultConn := newFakeOpenAI("default", "https://api.deepseek.com", "deepseek-chat", "sk-ds")

	result := resolvePrimaryConnector(defaultConn, nil)
	if result != defaultConn {
		t.Error("should return default when roles is nil")
	}
}
