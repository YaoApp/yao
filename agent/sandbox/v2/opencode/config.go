package opencode

import (
	"encoding/json"
	"strings"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

type roleSpec struct {
	EnvKeyPrefix string
	TopLevel     string
	Modalities   map[string][]string
}

var openCodeRoleMap = map[string]roleSpec{
	"light": {
		EnvKeyPrefix: "YAO_LIGHT",
		TopLevel:     "small_model",
	},
	"vision": {
		EnvKeyPrefix: "YAO_VISION",
		Modalities: map[string][]string{
			"input":  {"text", "image"},
			"output": {"text"},
		},
	},
	"heavy": {
		EnvKeyPrefix: "YAO_HEAVY",
	},
	"subagent": {
		EnvKeyPrefix: "YAO_SUBAGENT",
	},
}

// buildOpenCodeConfig generates the opencode.json project configuration.
// All provider configuration is direct (no a2o proxy).
func buildOpenCodeConfig(req *types.PrepareRequest, mcpServers []types.MCPServer) []byte {
	cfg := map[string]any{
		"$schema":    "https://opencode.ai/config.json",
		"autoupdate": false,
		"snapshot":   false,
		"share":      "disabled",
		"watcher":    map[string]any{"ignore": []string{".yao/**", ".attachments/**"}},
		"permission": map[string]any{"*": "allow"},
	}

	if req.Connector != nil {
		providerID, providerCfg, modelStr := buildProviderConfig(req.Connector)
		cfg["provider"] = map[string]any{providerID: providerCfg}
		cfg["model"] = modelStr
		cfg["enabled_providers"] = []string{providerID}
	}

	injectRoleProviders(cfg, req)

	if len(mcpServers) > 0 {
		cfg["mcp"] = buildMCPConfig(mcpServers)
	}

	prefix := ".yao/assistants/" + req.AssistantID
	if req.AssistantID == "" {
		prefix = ".opencode"
	}
	cfg["instructions"] = []string{prefix + "/system-prompt.md"}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	return data
}

// buildProviderConfig maps a Yao connector to an OpenCode provider configuration.
// Anthropic connectors map directly; OpenAI/OpenAI-compatible map to "openai".
//
// OpenCode appends its own endpoint paths (e.g. /responses) to baseURL,
// so we must NOT include /chat/completions. For native OpenAI (api.openai.com)
// we omit baseURL entirely and let OpenCode use its built-in default.
// For custom hosts (OpenAI-compatible proxies), we pass the bare host URL.
func buildProviderConfig(conn connector.Connector) (providerID string, cfg map[string]any, model string) {
	setting := conn.Setting()
	host, _ := setting["host"].(string)
	modelName, _ := setting["model"].(string)

	opts := map[string]any{
		"apiKey": "{env:YAO_PROVIDER_KEY}",
	}

	if conn.Is(connector.ANTHROPIC) {
		if host != "" {
			opts["baseURL"] = host
		}
		return "anthropic", map[string]any{"options": opts}, "anthropic/" + modelName
	}

	// Native OpenAI (api.openai.com): use built-in "openai" provider which
	// already knows all official models — no models declaration needed.
	if host == "" || isNativeOpenAI(host) {
		return "openai", map[string]any{"options": opts}, "openai/" + modelName
	}

	// OpenAI-compatible provider (DeepSeek, Moonshot, etc.): must use
	// @ai-sdk/openai-compatible and explicitly declare models, otherwise
	// OpenCode throws ProviderModelNotFoundError.
	opts["baseURL"] = normalizeBaseURL(host)

	modelCfg := map[string]any{
		"name": modelName,
	}

	// DeepSeek (and similar) thinking models return reasoning_content in
	// assistant messages. OpenCode must be told to preserve and replay
	// this field on conversation continuation, otherwise the API returns:
	//   "The reasoning_content in the thinking mode must be passed back to the API."
	// Adding "interleaved" is safe for non-thinking models (no-op if absent).
	modelCfg["interleaved"] = map[string]any{"field": "reasoning_content"}

	// Pass through thinking configuration from the Yao connector so OpenCode
	// sends it to the upstream API. DeepSeek defaults thinking to "enabled";
	// without explicitly sending {"thinking":{"type":"disabled"}}, the API
	// returns reasoning_content that OpenCode (AI SDK bug) fails to replay.
	modelOpts := buildModelOptions(setting)
	if len(modelOpts) > 0 {
		modelCfg["options"] = modelOpts
	}

	return "custom", map[string]any{
		"npm":     "@ai-sdk/openai-compatible",
		"options": opts,
		"models": map[string]any{
			modelName: modelCfg,
		},
	}, "custom/" + modelName
}

// buildModelOptions extracts connector-level model options (thinking, etc.)
// and maps them to the OpenCode model options format.
func buildModelOptions(setting map[string]any) map[string]any {
	opts := map[string]any{}

	// Forward thinking configuration as-is (e.g. {"type":"disabled"}).
	// DeepSeek V4 models default thinking to "enabled"; the only way to
	// suppress reasoning_content is to explicitly send {"type":"disabled"}.
	if thinking, ok := setting["thinking"]; ok && thinking != nil {
		opts["thinking"] = thinking
	}

	return opts
}

// isNativeOpenAI returns true if host points to official OpenAI API,
// where OpenCode already knows the correct base URL.
func isNativeOpenAI(host string) bool {
	h := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://"), "/")
	return h == "api.openai.com" ||
		strings.HasPrefix(h, "api.openai.com/")
}

// normalizeBaseURL strips trailing /chat/completions or /v1/chat/completions
// that Yao connectors may include, because OpenCode appends its own paths.
func normalizeBaseURL(host string) string {
	u := strings.TrimRight(host, "/")
	for _, suffix := range []string{"/chat/completions", "/completions"} {
		if strings.HasSuffix(u, suffix) {
			u = strings.TrimSuffix(u, suffix)
			break
		}
	}
	return strings.TrimRight(u, "/")
}

// injectRoleProviders iterates openCodeRoleMap and injects provider blocks
// for every role that has a configured connector. For the "light" role it
// also sets the top-level "small_model" field. This replaces the old
// buildSmallModel function and adds support for vision/heavy/subagent roles.
func injectRoleProviders(cfg map[string]any, req *types.PrepareRequest) {
	if req.Config == nil || req.Config.Runner.Connectors == nil {
		return
	}

	providers, _ := cfg["provider"].(map[string]any)
	if providers == nil {
		providers = map[string]any{}
		cfg["provider"] = providers
	}

	enabledSlice, _ := cfg["enabled_providers"].([]string)
	enabledSet := map[string]bool{}
	for _, e := range enabledSlice {
		enabledSet[e] = true
	}

	primaryHost := ""
	primaryType := ""
	if req.Connector != nil {
		primaryHost = connectorHost(req.Connector)
		if req.Connector.Is(connector.ANTHROPIC) {
			primaryType = "anthropic"
		} else {
			primaryType = "openai"
		}
	}

	for role, spec := range openCodeRoleMap {
		rc, ok := req.Config.Runner.Connectors[role]
		if !ok || rc == nil || rc.Connector == "" {
			continue
		}

		c, exists := connector.Connectors[rc.Connector]
		if !exists || c == nil {
			continue
		}

		setting := c.Setting()
		modelName, _ := setting["model"].(string)
		if modelName == "" {
			continue
		}

		roleHost := connectorHost(c)
		roleType := "openai"
		if c.Is(connector.ANTHROPIC) {
			roleType = "anthropic"
		}

		sameProvider := roleType == primaryType && roleHost == primaryHost
		if sameProvider && primaryHost != "" {
			sameProvider = true
		} else if sameProvider && primaryHost == "" && roleHost == "" {
			sameProvider = true
		} else if roleHost != primaryHost {
			sameProvider = false
		}

		var providerID string
		var modelRef string

		if sameProvider {
			providerID = resolveExistingProviderID(providers, primaryType)
			modelRef = providerID + "/" + modelName
			mergeModelIntoProvider(providers, providerID, modelName, spec.Modalities)
		} else {
			providerID = role
			providerCfg := buildRoleProviderConfig(c, spec.EnvKeyPrefix, spec.Modalities)
			providers[providerID] = providerCfg
			modelRef = providerID + "/" + modelName
		}

		if !enabledSet[providerID] {
			enabledSlice = append(enabledSlice, providerID)
			enabledSet[providerID] = true
		}

		if spec.TopLevel != "" {
			cfg[spec.TopLevel] = modelRef
		}
	}

	cfg["enabled_providers"] = enabledSlice
}

// resolveExistingProviderID finds the actual provider ID key used in the
// providers map for a given type. For "openai" type, it could be "openai"
// or "custom" (for openai-compatible). Returns the type as fallback.
func resolveExistingProviderID(providers map[string]any, pType string) string {
	if _, ok := providers[pType]; ok {
		return pType
	}
	if pType == "openai" {
		if _, ok := providers["custom"]; ok {
			return "custom"
		}
	}
	return pType
}

// mergeModelIntoProvider adds a model entry to an existing provider block.
func mergeModelIntoProvider(providers map[string]any, providerID, modelName string, modalities map[string][]string) {
	block, ok := providers[providerID].(map[string]any)
	if !ok {
		return
	}
	models, _ := block["models"].(map[string]any)
	if models == nil {
		models = map[string]any{}
		block["models"] = models
	}
	modelCfg := map[string]any{"name": modelName}
	if len(modalities) > 0 {
		modelCfg["modalities"] = modalities
	}
	models[modelName] = modelCfg
}

// buildRoleProviderConfig creates a provider configuration block for a
// non-primary role connector. Uses the role's env key prefix for API key
// and base URL references.
func buildRoleProviderConfig(conn connector.Connector, envKeyPrefix string, modalities map[string][]string) map[string]any {
	setting := conn.Setting()
	modelName, _ := setting["model"].(string)
	host, _ := setting["host"].(string)

	opts := map[string]any{
		"apiKey": "{env:" + envKeyPrefix + "_KEY}",
	}

	modelCfg := map[string]any{"name": modelName}
	if len(modalities) > 0 {
		modelCfg["modalities"] = modalities
	}

	if conn.Is(connector.ANTHROPIC) {
		if host != "" {
			opts["baseURL"] = host
		}
		return map[string]any{
			"options": opts,
			"models":  map[string]any{modelName: modelCfg},
		}
	}

	if host == "" || isNativeOpenAI(host) {
		return map[string]any{
			"options": opts,
			"models":  map[string]any{modelName: modelCfg},
		}
	}

	opts["baseURL"] = normalizeBaseURL(host)
	modelCfg["interleaved"] = map[string]any{"field": "reasoning_content"}

	return map[string]any{
		"npm":     "@ai-sdk/openai-compatible",
		"options": opts,
		"models":  map[string]any{modelName: modelCfg},
	}
}

// buildMCPConfig produces the "mcp" object for opencode.json.
// OpenCode uses "command" as an array (not command + args like Claude).
func buildMCPConfig(servers []types.MCPServer) map[string]any {
	result := make(map[string]any, len(servers))
	for _, s := range servers {
		name := s.ServerID
		if name == "" {
			continue
		}
		result[name] = map[string]any{
			"type":        "local",
			"command":     []string{"tai", "mcp", name},
			"enabled":     true,
			"environment": map[string]string{"YAO_TOKEN": "{env:YAO_TOKEN}"},
		}
	}
	if len(result) == 0 {
		result["yao"] = map[string]any{
			"type":        "local",
			"command":     []string{"tai", "mcp"},
			"enabled":     true,
			"environment": map[string]string{"YAO_TOKEN": "{env:YAO_TOKEN}"},
		}
	}
	return result
}
