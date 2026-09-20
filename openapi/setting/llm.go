package setting

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
)

var llmRolesNS = llmprovider.RolesNamespace

func llmEnsureEncKey() {
	if llmprovider.Global != nil && config.Conf.DB.AESKey != "" {
		llmprovider.Global.SetEncryptionKey(config.Conf.DB.AESKey)
	}
}

func llmOwner(info *oauthTypes.AuthorizedInfo) *llmprovider.ProviderOwner {
	if info.TeamID != "" {
		return &llmprovider.ProviderOwner{Type: "team", TeamID: info.TeamID}
	}
	return &llmprovider.ProviderOwner{Type: "user", UserID: info.UserID}
}

func llmScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

func llmCheckOwnership(p *llmprovider.Provider, info *oauthTypes.AuthorizedInfo) error {
	owner := llmOwner(info)
	if p.Owner.Type != owner.Type {
		return fmt.Errorf("provider not found")
	}
	if owner.Type == "team" && p.Owner.TeamID != owner.TeamID {
		return fmt.Errorf("provider not found")
	}
	if owner.Type == "user" && p.Owner.UserID != owner.UserID {
		return fmt.Errorf("provider not found")
	}
	return nil
}

func enrichProvider(p *llmprovider.Provider) map[string]interface{} {
	raw, _ := json.Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(raw, &m)

	if p.PresetKey != "" {
		if preset := llmprovider.GetPreset(p.PresetKey); preset != nil {
			m["is_cloud"] = preset.IsCloud
			m["url_editable"] = preset.URLEditable
		} else if p.PresetKey == "taoservice" || p.PresetKey == "yaoagents" {
			m["is_cloud"] = true
			m["url_editable"] = false
		}
	}

	delete(m, "connector_id")
	delete(m, "source")
	delete(m, "owner")

	return m
}

// llmCompletionEndpoint returns the endpoint path for the given provider type.
func llmCompletionEndpoint(providerType string) string {
	if providerType == "anthropic" {
		return "/messages"
	}
	return "/chat/completions"
}

// llmSetAuthHeader sets the appropriate auth header for the provider type.
func llmSetAuthHeader(req *http.Request, providerType, apiKey string) {
	if apiKey == "" {
		return
	}
	if providerType == "anthropic" {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

// llmValidateKey tests connectivity and API key validity using a multi-step
// approach that works across all provider types (OpenAI, Anthropic, and
// third-party compatible APIs) without incurring any token costs.
//
// Step 1: POST to the completion endpoint with a fake model and empty messages.
// An HTML response means the URL is wrong (non-API path). Otherwise record
// the status and always fall through to step 2 — some providers return 401 for
// the test payload even when the key is valid.
//
// Step 2: GET /models. Combined with step-1 status, decide:
//   - step1=401 + step2=200(JSON) → key valid (step-1 rejected the test payload, not the key)
//   - step1=401 + step2=401/403   → key truly invalid
//   - step1=401 + step2=404       → inconclusive, allow save
//   - step1=404 + step2=404       → URL is wrong (both endpoints missing)
//   - step1=other + step2=404     → /models unsupported, trust step-1
//   - step2=200 + step1=404       → step 3 (public /models, re-check with real model)
//
// Step 3: Pick the first model from /models and POST again to trigger a real
// auth check (unchanged from before).
func llmValidateKey(providerType, apiURL, apiKey string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	// --- Step 1: POST completion endpoint with fake model + empty messages ---
	postURL := connector.BuildAPIURL(apiURL, llmCompletionEndpoint(providerType))
	req, err := http.NewRequest("POST", postURL, strings.NewReader(`{"model":"_","messages":[]}`))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	llmSetAuthHeader(req, providerType, apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	resp.Body.Close()

	if llmIsHTML(resp) {
		return fmt.Errorf("URL does not point to a valid API endpoint (received HTML response)")
	}
	postStatus := resp.StatusCode

	// --- Step 2: GET /models to confirm key ---
	modelsURL := connector.BuildAPIURL(apiURL, "/models")
	req2, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		if postStatus == http.StatusUnauthorized || postStatus == http.StatusForbidden {
			return fmt.Errorf("invalid API key (HTTP %d)", postStatus)
		}
		return nil
	}
	llmSetAuthHeader(req2, providerType, apiKey)

	resp2, err := client.Do(req2)
	if err != nil {
		if postStatus == http.StatusUnauthorized || postStatus == http.StatusForbidden {
			return fmt.Errorf("invalid API key (HTTP %d)", postStatus)
		}
		return nil
	}

	modelsStatus := resp2.StatusCode
	var modelsBody []byte
	if modelsStatus == http.StatusOK {
		modelsBody, _ = io.ReadAll(resp2.Body)
	}
	resp2.Body.Close()

	if llmIsHTML(resp2) {
		return fmt.Errorf("URL does not point to a valid API endpoint (received HTML response)")
	}

	// --- Combined decision ---
	postIs401 := postStatus == http.StatusUnauthorized || postStatus == http.StatusForbidden
	postIs404 := postStatus == http.StatusNotFound

	switch {
	case modelsStatus == http.StatusUnauthorized || modelsStatus == http.StatusForbidden:
		return fmt.Errorf("invalid API key (HTTP %d)", modelsStatus)

	case modelsStatus == http.StatusNotFound:
		if postIs404 {
			return fmt.Errorf("URL does not point to a valid API endpoint (both endpoints returned 404)")
		}
		if postIs401 {
			return nil
		}
		return nil

	case modelsStatus == http.StatusOK:
		if postIs401 {
			return nil
		}
		if postIs404 && len(modelsBody) > 0 {
			return llmValidateWithModel(client, providerType, apiURL, apiKey, modelsBody)
		}
		return nil

	default:
		if postIs401 {
			return fmt.Errorf("invalid API key (HTTP %d)", postStatus)
		}
		return fmt.Errorf("server returned HTTP %d", modelsStatus)
	}
}

// llmIsHTML checks whether the response Content-Type indicates HTML,
// which means the URL hit a web page instead of an API endpoint.
func llmIsHTML(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/html")
}

// llmValidateWithModel is the step-3 fallback for providers whose /models
// endpoint is public (always 200). It picks the first model from the /models
// response and POSTs to the completion endpoint with that model + empty
// messages to trigger a real auth check.
func llmValidateWithModel(client *http.Client, providerType, apiURL, apiKey string, modelsBody []byte) error {
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(modelsBody, &parsed); err != nil || len(parsed.Data) == 0 {
		return nil
	}

	postURL := connector.BuildAPIURL(apiURL, llmCompletionEndpoint(providerType))
	body := fmt.Sprintf(`{"model":%q,"messages":[]}`, parsed.Data[0].ID)
	req, err := http.NewRequest("POST", postURL, strings.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	llmSetAuthHeader(req, providerType, apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Remote model preset helpers
// ---------------------------------------------------------------------------

var (
	remoteModelCache     []llmprovider.ModelInfo
	remoteModelCacheURL  string
	remoteReasoningCache map[string]interface{} // modelID → raw reasoning field from Tao API
	remoteModelCacheMu   sync.Mutex
)

func buildTaoPreset(info *oauthTypes.AuthorizedInfo) {
	var saved map[string]interface{}
	if setting.Global != nil {
		saved, _ = setting.Global.GetMerged(info.UserID, info.TeamID, taoNS)
	}

	apiURL := resolveTaoAPIURL(saved)
	preset := llmprovider.ProviderPreset{
		Key:        "taoservice",
		Name:       "Tao Service",
		Type:       "openai",
		APIURL:     apiURL,
		RequireKey: false,
		IsCloud:    true,
	}

	status, _ := saved["status"].(string)
	if status == "connected" {
		if encKey, _ := saved["api_key"].(string); encKey != "" {
			models := fetchRemoteModels(apiURL, decryptValue(encKey))
			if len(models) > 0 {
				for i := range models {
					models[i].Enabled = true
				}
				preset.DefaultModels = models
			}
		}
	}

	llmprovider.RegisterPreset(preset)
}

// resolveTaoAPIURL returns the Tao API URL from saved config or falls back to well-known defaults.
func resolveTaoAPIURL(saved map[string]interface{}) string {
	if saved != nil {
		if v, ok := saved["base_url"].(string); ok && v != "" {
			return v
		}
	}
	return resolveTaoBaseURL("en-us")
}

func fetchRemoteModels(apiURL, apiKey string) []llmprovider.ModelInfo {
	remoteModelCacheMu.Lock()
	if remoteModelCache != nil && remoteModelCacheURL == apiURL {
		cached := remoteModelCache
		remoteModelCacheMu.Unlock()
		return cached
	}
	remoteModelCacheMu.Unlock()

	url := apiURL
	if strings.HasSuffix(url, "/") {
		url += "v1/models"
	} else {
		url += "/v1/models"
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	reasoningCache := make(map[string]interface{})
	remoteModelCacheMu.Lock()
	remoteReasoningCache = reasoningCache
	remoteModelCacheMu.Unlock()

	var expanded []llmprovider.ModelInfo
	for _, item := range result.Data {
		base := mapRemoteModel(item)
		if base == nil {
			continue
		}
		variants := expandReasoningVariants(base)
		expanded = append(expanded, variants...)
	}

	remoteModelCacheMu.Lock()
	remoteModelCache = expanded
	remoteModelCacheURL = apiURL
	remoteModelCacheMu.Unlock()

	return expanded
}

func invalidateRemoteModelCache() {
	remoteModelCacheMu.Lock()
	remoteModelCache = nil
	remoteModelCacheURL = ""
	remoteReasoningCache = nil
	remoteModelCacheMu.Unlock()
}

func mapRemoteModel(item map[string]interface{}) map[string]interface{} {
	id, _ := item["id"].(string)
	if id == "" {
		return nil
	}

	// OCR and fetch models are handled by dedicated tool providers, not the LLM provider.
	if svc, _ := item["service"].(string); svc == "ocr" || svc == "fetch" {
		return nil
	}

	name, _ := item["name"].(string)
	if name == "" {
		name = id
	}

	var caps []string
	if rawCaps, ok := item["capabilities"].([]interface{}); ok {
		for _, c := range rawCaps {
			if s, ok := c.(string); ok {
				caps = append(caps, s)
			}
		}
	}
	caps = normalizeCapabilities(caps)

	m := map[string]interface{}{
		"id":           id,
		"name":         name,
		"capabilities": caps,
	}

	if desc, ok := item["description"].(string); ok && desc != "" {
		m["description"] = desc
	}
	if v, ok := getNumber(item, "max_input_tokens"); ok && v > 0 {
		m["max_input_tokens"] = int(v)
	}
	if v, ok := getNumber(item, "max_output_tokens"); ok && v > 0 {
		m["max_output_tokens"] = int(v)
	}
	if r, ok := item["reasoning"]; ok && r != nil {
		m["reasoning"] = r
	}
	return m
}

// normalizeCapabilities maps upstream capability names to the canonical names the
// frontend expects (e.g. image_generate → image_generation, audio_transcribe → audio).
func normalizeCapabilities(caps []string) []string {
	if len(caps) == 0 {
		return []string{}
	}
	seen := make(map[string]bool, len(caps))
	out := make([]string, 0, len(caps)+2)
	for _, c := range caps {
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)

		var alias string
		switch c {
		case "image_generate":
			alias = "image_generation"
		case "image_edit":
			alias = "image_editing"
		case "audio_transcribe":
			alias = "audio"
		}
		if alias != "" && !seen[alias] {
			seen[alias] = true
			out = append(out, alias)
		}
	}
	return out
}

func getBool(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key].(bool)
	return ok && v
}

func getNumber(m map[string]interface{}, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	switch v := m[key].(type) {
	case float64:
		return v, true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	return 0, false
}

// ---------------------------------------------------------------------------
// Reasoning expansion — generates one ModelInfo per thinking-intensity config
// ---------------------------------------------------------------------------

// reasoningSpec holds the parsed reasoning field from a Tao API model.
type reasoningSpec struct {
	Switch      *reasoningSwitch
	Effort      *reasoningEffort
	Default     map[string]interface{}
	CanDisable  *bool                    // Tao can_disable: whether thinking can be turned off
	DisableWith []map[string]interface{} // Tao disable_with: parameter snippets that disable thinking
}

type reasoningSwitch struct {
	Param  string
	Values []interface{}
}

type reasoningEffort struct {
	Param  string
	Values []string
}

// parseReasoningSpec converts a raw reasoning field (interface{}) into a typed struct.
func parseReasoningSpec(raw interface{}) *reasoningSpec {
	m, ok := raw.(map[string]interface{})
	if !ok || m == nil {
		return nil
	}

	spec := &reasoningSpec{}

	if sw, ok := m["switch"].(map[string]interface{}); ok {
		param, _ := sw["param"].(string)
		if param != "" {
			rs := &reasoningSwitch{Param: param}
			if vals, ok := sw["values"].([]interface{}); ok {
				rs.Values = vals
			}
			spec.Switch = rs
		}
	}

	if eff, ok := m["effort"].(map[string]interface{}); ok {
		param, _ := eff["param"].(string)
		if param != "" {
			re := &reasoningEffort{Param: param}
			if vals, ok := eff["values"].([]interface{}); ok {
				for _, v := range vals {
					if s, ok := v.(string); ok {
						re.Values = append(re.Values, s)
					}
				}
			}
			spec.Effort = re
		}
	}

	if def, ok := m["default"].(map[string]interface{}); ok {
		spec.Default = def
	}

	if cd, ok := m["can_disable"].(bool); ok {
		spec.CanDisable = &cd
	}

	if dw, ok := m["disable_with"].([]interface{}); ok {
		for _, item := range dw {
			if dm, ok := item.(map[string]interface{}); ok {
				spec.DisableWith = append(spec.DisableWith, dm)
			}
		}
	}

	return spec
}

// classifySwitchValues separates switch values into disabled and enabled.
// Returns (disabledVal, enabledVal); either may be nil.
func classifySwitchValues(sw *reasoningSwitch) (disabled, enabled interface{}) {
	for _, v := range sw.Values {
		switch val := v.(type) {
		case bool:
			if val {
				enabled = v
			} else {
				disabled = v
			}
		case map[string]interface{}:
			t, _ := val["type"].(string)
			if t == "disabled" {
				disabled = v
			} else {
				enabled = v
			}
		default:
			enabled = v
		}
	}
	return
}

// isSwitchDisabled returns true if the switch value represents "disabled".
func isSwitchDisabled(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return !val
	case map[string]interface{}:
		t, _ := val["type"].(string)
		return t == "disabled"
	}
	return false
}

// expandReasoningVariants expands a single Tao model into multiple ModelInfo
// entries, one per thinking-intensity configuration.
func expandReasoningVariants(baseModel map[string]interface{}) []llmprovider.ModelInfo {
	id, _ := baseModel["id"].(string)
	if id == "" {
		return nil
	}

	spec := parseReasoningSpec(baseModel["reasoning"])
	baseCaps := toStringSlice(baseModel["capabilities"])

	if spec == nil {
		return []llmprovider.ModelInfo{buildModelInfo(id, "", baseCaps, nil, baseModel, true, "")}
	}

	cacheModelReasoning(id, baseModel["reasoning"])

	hasSwitch := spec.Switch != nil && len(spec.Switch.Values) > 0
	hasEffort := spec.Effort != nil && len(spec.Effort.Values) > 0
	var disabledVal, enabledVal interface{}
	if hasSwitch {
		disabledVal, enabledVal = classifySwitchValues(spec.Switch)
	}

	// Determine whether a disabled variant can be generated and its Options.
	// Priority: CanDisable > classifySwitchValues inference (backward compat).
	var disableOpts map[string]interface{}
	canDisable := false
	if spec.CanDisable != nil {
		if *spec.CanDisable && len(spec.DisableWith) > 0 {
			disableOpts = spec.DisableWith[0]
			canDisable = true
		} else if *spec.CanDisable && len(spec.DisableWith) == 0 && hasSwitch && disabledVal != nil {
			disableOpts = map[string]interface{}{spec.Switch.Param: disabledVal}
			canDisable = true
		}
		// can_disable==false → canDisable stays false regardless of switch values
	} else if hasSwitch && disabledVal != nil {
		disableOpts = map[string]interface{}{spec.Switch.Param: disabledVal}
		canDisable = true
	}

	// Case 1: switch + effort, can disable (e.g. deepseek-flash, qwen3.8-flash, kimi-k3)
	if hasSwitch && canDisable && hasEffort {
		variants := make([]llmprovider.ModelInfo, 0, 1+len(spec.Effort.Values))
		variants = append(variants, buildModelInfo(id, "", removeCap(baseCaps, "reasoning"), disableOpts, baseModel, true, ""))
		for _, effort := range spec.Effort.Values {
			if effort == "none" {
				continue
			}
			eOpts := map[string]interface{}{spec.Switch.Param: enabledVal, spec.Effort.Param: effort}
			connID := id + "-thinking-" + effort
			suffix := "(Thinking: " + capitalizeFirst(effort) + ")"
			variants = append(variants, buildModelInfo(connID, id, ensureCap(baseCaps, "reasoning"), eOpts, baseModel, false, suffix))
		}
		return variants
	}

	// Case 2: effort only, no switch (always-reasoning, no disable mechanism)
	if !hasSwitch && hasEffort {
		defaultEffort := ""
		if spec.Default != nil && spec.Effort != nil {
			defaultEffort, _ = spec.Default[spec.Effort.Param].(string)
		}
		variants := make([]llmprovider.ModelInfo, 0, len(spec.Effort.Values))
		for _, effort := range spec.Effort.Values {
			opts := map[string]interface{}{spec.Effort.Param: effort}
			thinkingCaps := ensureCap(baseCaps, "reasoning")
			if effort == defaultEffort {
				variants = append(variants, buildModelInfo(id, "", thinkingCaps, opts, baseModel, true, ""))
			} else {
				connID := id + "-effort-" + effort
				suffix := "(Effort: " + capitalizeFirst(effort) + ")"
				variants = append(variants, buildModelInfo(connID, id, thinkingCaps, opts, baseModel, false, suffix))
			}
		}
		return variants
	}

	// Case 3: switch only (enabled/disabled), no effort, can disable
	if hasSwitch && enabledVal != nil && canDisable && !hasEffort {
		opts2 := map[string]interface{}{spec.Switch.Param: enabledVal}
		return []llmprovider.ModelInfo{
			buildModelInfo(id, "", removeCap(baseCaps, "reasoning"), disableOpts, baseModel, true, ""),
			buildModelInfo(id+"-thinking", id, ensureCap(baseCaps, "reasoning"), opts2, baseModel, false, "(Thinking)"),
		}
	}

	// Case 4: switch with only enabled, or cannot disable (e.g. kimi-k2.7-code)
	if hasSwitch && enabledVal != nil && !canDisable {
		opts := map[string]interface{}{spec.Switch.Param: enabledVal}
		return []llmprovider.ModelInfo{buildModelInfo(id, "", ensureCap(baseCaps, "reasoning"), opts, baseModel, true, "")}
	}

	// Case 5: no switch + no effort but has default params
	if spec.Default != nil {
		return []llmprovider.ModelInfo{buildModelInfo(id, "", ensureCap(baseCaps, "reasoning"), spec.Default, baseModel, true, "")}
	}

	return []llmprovider.ModelInfo{buildModelInfo(id, "", baseCaps, nil, baseModel, true, "")}
}

// resolveConnectorID maps a role default (model + params) back to the expanded connector ID.
func resolveConnectorID(modelID string, params map[string]interface{}, reasoning interface{}) string {
	spec := parseReasoningSpec(reasoning)
	if spec == nil {
		return modelID
	}

	if len(params) == 0 {
		if spec.Default != nil {
			params = copyMap(spec.Default)
		} else {
			return modelID
		}
	}

	hasSwitch := spec.Switch != nil && len(spec.Switch.Values) > 0
	hasEffort := spec.Effort != nil && len(spec.Effort.Values) > 0

	switchIsDisabled := false
	if hasSwitch {
		if switchVal, ok := params[spec.Switch.Param]; ok {
			switchIsDisabled = isSwitchDisabled(switchVal)
		} else if spec.Default != nil {
			if defSwitch, ok := spec.Default[spec.Switch.Param]; ok {
				switchIsDisabled = isSwitchDisabled(defSwitch)
			}
		}
	}

	if hasSwitch && switchIsDisabled {
		return modelID
	}

	effortVal := ""
	if hasEffort {
		if ev, ok := params[spec.Effort.Param].(string); ok {
			effortVal = ev
		} else if spec.Default != nil {
			effortVal, _ = spec.Default[spec.Effort.Param].(string)
		}
	}

	if hasSwitch && hasEffort && effortVal != "" {
		if effortVal == "none" {
			return modelID
		}
		return modelID + "-thinking-" + effortVal
	}
	if hasSwitch && !hasEffort {
		return modelID + "-thinking"
	}
	if !hasSwitch && hasEffort && effortVal != "" {
		defaultEffort := ""
		if spec.Default != nil {
			defaultEffort, _ = spec.Default[spec.Effort.Param].(string)
		}
		if effortVal == defaultEffort {
			return modelID
		}
		return modelID + "-effort-" + effortVal
	}

	return modelID
}

// buildModelInfo constructs a ModelInfo from expansion parameters.
// nameSuffix is appended to the display name to distinguish thinking variants.
func buildModelInfo(connID, model string, caps []string, opts map[string]interface{}, base map[string]interface{}, enabled bool, nameSuffix string) llmprovider.ModelInfo {
	name, _ := base["name"].(string)
	if name == "" {
		name = connID
	}
	if nameSuffix != "" {
		name = name + " " + nameSuffix
	}
	mi := llmprovider.ModelInfo{
		ID:           connID,
		Name:         name,
		Capabilities: caps,
		Enabled:      enabled,
		Options:      opts,
	}
	if model != "" {
		mi.Model = model
	}
	if v, ok := getNumber(base, "max_input_tokens"); ok && v > 0 {
		mi.MaxInputTokens = int(v)
	}
	if v, ok := getNumber(base, "max_output_tokens"); ok && v > 0 {
		mi.MaxOutputTokens = int(v)
	}
	return mi
}

// capitalizeFirst returns s with the first letter uppercased.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// toStringSlice extracts a string slice from an interface value.
func toStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// removeCap returns a copy of caps without the specified capability.
func removeCap(caps []string, remove string) []string {
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		if c != remove {
			out = append(out, c)
		}
	}
	return out
}

// ensureCap returns a copy of caps with the specified capability added if missing.
func ensureCap(caps []string, add string) []string {
	for _, c := range caps {
		if c == add {
			dst := make([]string, len(caps))
			copy(dst, caps)
			return dst
		}
	}
	out := make([]string, len(caps)+1)
	copy(out, caps)
	out[len(caps)] = add
	return out
}

// cacheModelReasoning stores the raw reasoning spec for later resolveConnectorID lookups.
func cacheModelReasoning(modelID string, reasoning interface{}) {
	if reasoning != nil && remoteReasoningCache != nil {
		remoteReasoningCache[modelID] = reasoning
	}
}

// copyMap returns a shallow copy of a map.
func copyMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// handleLLMTest validates an API URL + Key without saving.
// POST /setting/llm/test
func handleLLMTest(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	var input struct {
		APIURL     string `json:"api_url"`
		APIKey     string `json:"api_key"`
		Type       string `json:"type"`
		RequireKey *bool  `json:"require_key"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.APIURL == "" {
		respondError(c, http.StatusBadRequest, "api_url is required")
		return
	}
	if input.APIKey == "" && (input.RequireKey == nil || *input.RequireKey) {
		response.RespondWithSuccess(c, http.StatusOK, llmprovider.ProviderTestResult{
			Success: false,
			Message: "API Key is required",
		})
		return
	}

	start := time.Now()
	err := llmValidateKey(input.Type, input.APIURL, input.APIKey)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		response.RespondWithSuccess(c, http.StatusOK, llmprovider.ProviderTestResult{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, llmprovider.ProviderTestResult{
		Success:   true,
		Message:   "Connection successful",
		LatencyMs: latency,
	})
}

// handleLLMGet returns the aggregated LLM configuration page data.
// GET /setting/llm
func handleLLMGet(c *gin.Context) {
	info := authorized.GetInfo(c)

	if llmprovider.Global == nil {
		respondError(c, http.StatusInternalServerError, "LLM provider registry not initialized")
		return
	}

	llmEnsureEncKey()

	owner := llmOwner(info)
	filter := &llmprovider.ProviderFilter{
		Owner:  owner,
		Source: llmprovider.ProviderSourceAll,
	}
	providers, err := llmprovider.Global.List(filter)
	if err != nil {
		providers = []llmprovider.Provider{}
	}

	enriched := make([]interface{}, 0, len(providers))
	for i := range providers {
		enriched = append(enriched, enrichProvider(&providers[i]))
	}

	var roles map[string]interface{}
	if setting.Global != nil {
		roles, _ = setting.Global.GetMerged(info.UserID, info.TeamID, llmRolesNS)
	}
	if roles == nil {
		roles = make(map[string]interface{})
	}

	buildTaoPreset(info)

	locale := c.Query("locale")
	var presetList []llmprovider.ProviderPreset
	if locale != "" {
		presetList = llmprovider.GetPresetsForLocale(locale)
	} else {
		presetList = llmprovider.GetPresets()
	}
	presetIface := make([]interface{}, len(presetList))
	for i, p := range presetList {
		raw, _ := json.Marshal(p)
		var m map[string]interface{}
		json.Unmarshal(raw, &m)
		presetIface[i] = m
	}

	response.RespondWithSuccess(c, http.StatusOK, LLMPageData{
		Providers:       enriched,
		Roles:           roles,
		PresetProviders: presetIface,
	})
}

// handleLLMRoles saves the role assignment (default models).
// PUT /setting/llm/roles
func handleLLMRoles(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := llmScope(info)

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, ok := body["default"]; !ok {
		respondError(c, http.StatusBadRequest, "\"default\" role is required")
		return
	}

	if llmprovider.Global == nil {
		respondError(c, http.StatusInternalServerError, "LLM provider registry not initialized")
		return
	}

	llmEnsureEncKey()

	var staleRoles []string
	for roleName, target := range body {
		targetMap, ok := target.(map[string]interface{})
		if !ok {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("invalid target for role \"%s\"", roleName))
			return
		}

		providerKey, _ := targetMap["provider"].(string)
		modelID, _ := targetMap["model"].(string)
		if providerKey == "" || modelID == "" {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("role \"%s\" requires provider and model", roleName))
			return
		}

		p, err := llmprovider.Global.Get(providerKey)
		if err != nil {
			staleRoles = append(staleRoles, roleName)
			continue
		}
		if !p.Enabled {
			staleRoles = append(staleRoles, roleName)
			continue
		}
		if err := llmCheckOwnership(p, info); err != nil {
			staleRoles = append(staleRoles, roleName)
			continue
		}

		modelFound := false
		for _, m := range p.Models {
			if m.ID == modelID {
				modelFound = true
				break
			}
		}
		if !modelFound {
			staleRoles = append(staleRoles, roleName)
		}
	}
	for _, role := range staleRoles {
		delete(body, role)
	}
	if _, ok := body["default"]; !ok {
		respondError(c, http.StatusBadRequest, "\"default\" role: the assigned provider no longer exists, please re-select")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	if _, err := setting.Global.Set(scope, llmRolesNS, body); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, body)
}

// handleLLMProviderCreate creates a new LLM provider (preset or custom).
// POST /setting/llm/providers
func handleLLMProviderCreate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)

	if llmprovider.Global == nil {
		respondError(c, http.StatusInternalServerError, "LLM provider registry not initialized")
		return
	}

	llmEnsureEncKey()

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var provider llmprovider.Provider
	owner := llmOwner(info)
	provider.Owner = *owner
	provider.Source = llmprovider.ProviderSourceDynamic
	provider.Enabled = true

	presetKey, _ := body["preset_key"].(string)

	if presetKey != "" {
		preset := llmprovider.GetPreset(presetKey)
		if preset == nil && (presetKey == "taoservice" || presetKey == "yaoagents") {
			buildTaoPreset(info)
			preset = llmprovider.GetPreset("taoservice")
		}
		if preset == nil {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("unknown preset: %s", presetKey))
			return
		}

		provider.Key = llmprovider.ScopedKey(owner, presetKey)
		provider.Name = preset.Name
		provider.Type = preset.Type
		provider.APIURL = preset.APIURL
		provider.RequireKey = preset.RequireKey
		provider.PresetKey = presetKey

		if v, ok := body["api_url"].(string); ok && v != "" {
			provider.APIURL = v
		}
		if v, ok := body["api_key"].(string); ok && v != "" {
			provider.APIKey = v
		}
		if v, ok := body["name"].(string); ok && v != "" {
			provider.Name = v
		}

		modelIDs, hasModelIDs := body["model_ids"].([]interface{})
		if hasModelIDs && len(modelIDs) > 0 {
			idSet := make(map[string]bool, len(modelIDs))
			for _, id := range modelIDs {
				if s, ok := id.(string); ok {
					idSet[s] = true
				}
			}
			for _, m := range preset.DefaultModels {
				if idSet[m.ID] {
					m.Enabled = true
					provider.Models = append(provider.Models, m)
				}
			}
		} else {
			provider.Models = make([]llmprovider.ModelInfo, len(preset.DefaultModels))
			copy(provider.Models, preset.DefaultModels)
		}

		if preset.IsCloud && provider.APIKey == "" {
			var saved map[string]interface{}
			if setting.Global != nil {
				saved, _ = setting.Global.GetMerged(info.UserID, info.TeamID, taoNS)
			}
			if encKey, _ := saved["api_key"].(string); encKey != "" {
				provider.APIKey = decryptValue(encKey)
			}
		}
	} else {
		provider.IsCustom = true

		key, _ := body["key"].(string)
		if key == "" {
			respondError(c, http.StatusBadRequest, "key is required for custom provider")
			return
		}
		provider.Key = llmprovider.ScopedKey(owner, key)

		name, _ := body["name"].(string)
		if name == "" {
			respondError(c, http.StatusBadRequest, "name is required")
			return
		}
		provider.Name = name

		typ, _ := body["type"].(string)
		if typ == "" {
			typ = "openai"
		}
		provider.Type = typ

		provider.APIURL, _ = body["api_url"].(string)
		provider.APIKey, _ = body["api_key"].(string)

		if modelsRaw, ok := body["models"]; ok {
			raw, _ := json.Marshal(modelsRaw)
			var models []llmprovider.ModelInfo
			if err := json.Unmarshal(raw, &models); err == nil {
				provider.Models = models
			}
		}

		if v, ok := body["require_key"].(bool); ok {
			provider.RequireKey = v
		}
	}

	if provider.Models == nil {
		provider.Models = []llmprovider.ModelInfo{}
	}

	if provider.RequireKey && provider.APIKey != "" && provider.APIURL != "" {
		if err := llmValidateKey(provider.Type, provider.APIURL, provider.APIKey); err != nil {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("API key validation failed: %s", err.Error()))
			return
		}
	}

	created, err := llmprovider.Global.Create(&provider)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			respondError(c, http.StatusConflict, err.Error())
		} else {
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	masked, err := llmprovider.Global.GetMasked(created.Key)
	if err != nil {
		created.APIKey = ""
		response.RespondWithSuccess(c, http.StatusCreated, enrichProvider(created))
		return
	}
	response.RespondWithSuccess(c, http.StatusCreated, enrichProvider(masked))
}

// handleLLMProviderUpdate replaces a provider's configuration.
// Full replacement: api_key empty string preserves existing value.
// PUT /setting/llm/providers/:key
func handleLLMProviderUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	key := c.Param("key")

	if llmprovider.Global == nil {
		respondError(c, http.StatusInternalServerError, "LLM provider registry not initialized")
		return
	}

	llmEnsureEncKey()

	existing, err := llmprovider.Global.Get(key, true)
	if err != nil {
		respondError(c, http.StatusNotFound, fmt.Sprintf("provider \"%s\" not found", key))
		return
	}
	if err := llmCheckOwnership(existing, info); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var provider llmprovider.Provider
	provider.Key = key
	provider.Owner = existing.Owner
	provider.Source = existing.Source
	provider.ConnectorID = existing.ConnectorID
	provider.PresetKey = existing.PresetKey
	provider.IsCustom = existing.IsCustom

	if v, ok := body["name"].(string); ok {
		provider.Name = v
	} else {
		provider.Name = existing.Name
	}
	if v, ok := body["type"].(string); ok {
		provider.Type = v
	} else {
		provider.Type = existing.Type
	}
	if v, ok := body["api_url"].(string); ok {
		provider.APIURL = v
	} else {
		provider.APIURL = existing.APIURL
	}

	if v, ok := body["api_key"].(string); ok && v != "" {
		provider.APIKey = v
	} else {
		provider.APIKey = existing.APIKey
	}

	if v, ok := body["enabled"].(bool); ok {
		provider.Enabled = v
	} else {
		provider.Enabled = existing.Enabled
	}
	if v, ok := body["require_key"].(bool); ok {
		provider.RequireKey = v
	} else {
		provider.RequireKey = existing.RequireKey
	}
	if v, ok := body["status"].(string); ok {
		provider.Status = v
	} else {
		provider.Status = existing.Status
	}

	if modelsRaw, ok := body["models"]; ok {
		raw, _ := json.Marshal(modelsRaw)
		var models []llmprovider.ModelInfo
		if err := json.Unmarshal(raw, &models); err == nil {
			provider.Models = models
		}
	} else {
		provider.Models = existing.Models
	}
	if provider.Models == nil {
		provider.Models = []llmprovider.ModelInfo{}
	}

	if _, err = llmprovider.Global.Update(key, &provider); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	masked, err := llmprovider.Global.GetMasked(key)
	if err != nil {
		provider.APIKey = ""
		response.RespondWithSuccess(c, http.StatusOK, enrichProvider(&provider))
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, enrichProvider(masked))
}

// handleLLMProviderDelete removes a provider and cleans up role references.
// DELETE /setting/llm/providers/:key
func handleLLMProviderDelete(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	key := c.Param("key")

	if llmprovider.Global == nil {
		respondError(c, http.StatusInternalServerError, "LLM provider registry not initialized")
		return
	}

	llmEnsureEncKey()

	existing, err := llmprovider.Global.Get(key)
	if err != nil {
		respondError(c, http.StatusNotFound, fmt.Sprintf("provider \"%s\" not found", key))
		return
	}
	if err := llmCheckOwnership(existing, info); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	var warning string
	if setting.Global != nil {
		scope := llmScope(info)
		roles, _ := setting.Global.Get(scope, llmRolesNS)
		if roles != nil {
			cleaned := false
			for roleName, target := range roles {
				if targetMap, ok := target.(map[string]interface{}); ok {
					if provKey, _ := targetMap["provider"].(string); provKey == key {
						delete(roles, roleName)
						cleaned = true
					}
				}
			}
			if cleaned {
				setting.Global.Set(scope, llmRolesNS, roles)
				warning = fmt.Sprintf("roles referencing provider \"%s\" have been cleared", key)
			}
		}
	}

	if err := llmprovider.Global.Delete(key); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := map[string]interface{}{"success": true}
	if warning != "" {
		result["warning"] = warning
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleLLMProviderTest tests connectivity for a provider and writes back status.
// POST /setting/llm/providers/:key/test
func handleLLMProviderTest(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	key := c.Param("key")

	if llmprovider.Global == nil {
		respondError(c, http.StatusInternalServerError, "LLM provider registry not initialized")
		return
	}

	llmEnsureEncKey()

	p, err := llmprovider.Global.Get(key, true)
	if err != nil {
		respondError(c, http.StatusNotFound, fmt.Sprintf("provider \"%s\" not found", key))
		return
	}
	if err := llmCheckOwnership(p, info); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	start := time.Now()
	err = llmValidateKey(p.Type, p.APIURL, p.APIKey)
	latency := time.Since(start).Milliseconds()

	var testResult llmprovider.ProviderTestResult
	if err != nil {
		testResult = llmprovider.ProviderTestResult{
			Success: false,
			Message: err.Error(),
		}
		p.Status = "disconnected"
	} else {
		testResult = llmprovider.ProviderTestResult{
			Success:   true,
			Message:   "Connection successful",
			LatencyMs: latency,
		}
		p.Status = "connected"
		llmprovider.Global.Update(key, p)
	}

	response.RespondWithSuccess(c, http.StatusOK, testResult)
}
