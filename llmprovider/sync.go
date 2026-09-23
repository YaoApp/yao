package llmprovider

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
)

// builtinSchemaVersion is monotonically incremented when import logic changes.
// v1: initial import (implicit, legacy records store 0)
// v2: extractOptionsFromSetting adds reasoning_effort/thinking/model to ModelInfo.Options
// v3: metadata — model_name, model_family, reasoning_efforts, reasoning_effort
// v4: metadata — group_name for ProviderGroup display
const builtinSchemaVersion = 4

// ScopedKey returns a provider key prefixed with the owner scope.
// This ensures unique keys per user/team in the store.
//
//	user  -> "u<userID>.<baseKey>"
//	team  -> "t<teamID>.<baseKey>"
//	other -> baseKey (unchanged)
func ScopedKey(owner *ProviderOwner, baseKey string) string {
	switch owner.Type {
	case "user":
		return "u" + owner.UserID + "." + baseKey
	case "team":
		return "t" + owner.TeamID + "." + baseKey
	default:
		return baseKey
	}
}

// connectorID builds the runtime ID for registering into connector.Connectors.
// For user/team providers the Key is already scoped, so use it directly.
func connectorID(p *Provider) string {
	switch p.Owner.Type {
	case "user", "team":
		return p.Key
	default:
		return "s." + p.Key
	}
}

// defaultModel returns the first enabled model ID, or empty string.
func defaultModel(p *Provider) string {
	for _, m := range p.Models {
		if m.Enabled {
			return m.ID
		}
	}
	if len(p.Models) > 0 {
		return p.Models[0].ID
	}
	return ""
}

// marshalDSL builds a connector DSL JSON from the flat Provider fields.
func marshalDSL(p *Provider) ([]byte, error) {
	opts := map[string]interface{}{
		"host":  p.APIURL,
		"key":   p.APIKey,
		"model": defaultModel(p),
	}

	if caps := aggregateCapabilities(p); len(caps) > 0 {
		opts["capabilities"] = caps
	}

	dsl := map[string]interface{}{
		"type":    p.Type,
		"name":    p.Name,
		"label":   p.Name,
		"options": opts,
	}

	// Propagate auth_mode for providers that use non-Bearer authentication
	if p.PresetKey == "azure" {
		dsl["auth_mode"] = "api-key"
	}

	return json.Marshal(dsl)
}

// aggregateCapabilities merges all model capabilities into a single map.
// Falls back to type-based defaults when no model declares explicit caps.
func aggregateCapabilities(p *Provider) map[string]bool {
	caps := make(map[string]bool)
	for _, m := range p.Models {
		for _, c := range m.Capabilities {
			caps[c] = true
		}
	}
	if len(caps) == 0 {
		switch p.Type {
		case "openai", "anthropic":
			caps["streaming"] = true
			caps["tool_calls"] = true
			caps["temperature_adjustable"] = true
		}
	}
	return caps
}

// ensureConnector makes sure the provider's connector is registered in the runtime.
// Builtin providers are managed by engine.Load and skipped here.
func ensureConnector(p *Provider) error {
	if p.Source == ProviderSourceBuiltIn {
		return nil
	}
	if !p.Enabled {
		return nil
	}

	cid := p.ConnectorID
	if cid == "" {
		cid = connectorID(p)
	}

	if _, err := connector.Select(cid); err == nil {
		return nil
	}

	dslJSON, err := marshalDSL(p)
	if err != nil {
		return fmt.Errorf("ensureConnector %s: marshal DSL: %w", p.Key, err)
	}

	_, err = connector.LoadSourceSync(dslJSON, cid, "__registry/"+cid+".conn.yao")
	if err != nil {
		return fmt.Errorf("ensureConnector %s: LoadSourceSync: %w", p.Key, err)
	}

	return nil
}

// ensureModelConnector registers a per-model connector for a dynamic provider.
// The connector ID format is "{providerConnectorID}:{modelID}".
func ensureModelConnector(p *Provider, m *ModelInfo) error {
	if p.Source == ProviderSourceBuiltIn {
		return nil
	}
	if !p.Enabled || !m.Enabled {
		return nil
	}

	baseCID := p.ConnectorID
	if baseCID == "" {
		baseCID = connectorID(p)
	}
	cid := baseCID + ":" + m.ID

	if _, err := connector.Select(cid); err == nil {
		return nil
	}

	dslJSON, err := marshalModelDSL(p, m)
	if err != nil {
		return fmt.Errorf("ensureModelConnector %s:%s: %w", p.Key, m.ID, err)
	}

	_, err = connector.LoadSourceSync(dslJSON, cid, "__registry/"+baseCID+"/"+m.ID+".conn.yao")
	if err != nil {
		return fmt.Errorf("ensureModelConnector %s:%s: LoadSourceSync: %w", p.Key, m.ID, err)
	}
	return nil
}

// marshalModelDSL builds a connector DSL for a specific model within a provider.
func marshalModelDSL(p *Provider, m *ModelInfo) ([]byte, error) {
	caps := make(map[string]interface{})
	for _, c := range m.Capabilities {
		if optVal, ok := m.Options[c]; ok {
			caps[c] = optVal
		} else {
			caps[c] = true
		}
	}
	if len(caps) == 0 {
		switch p.Type {
		case "openai", "anthropic":
			caps["streaming"] = true
			caps["tool_calls"] = true
			caps["temperature_adjustable"] = true
		}
	}
	if m.MaxInputTokens > 0 {
		caps["max_input_tokens"] = m.MaxInputTokens
	}
	if m.MaxOutputTokens > 0 {
		caps["max_output_tokens"] = m.MaxOutputTokens
	}

	apiModel := m.ID
	if m.Model != "" {
		apiModel = m.Model
	}
	opts := map[string]interface{}{
		"host":  p.APIURL,
		"key":   p.APIKey,
		"model": apiModel,
	}
	if len(caps) > 0 {
		opts["capabilities"] = caps
	}

	reserved := map[string]bool{"host": true, "key": true, "model": true, "capabilities": true, "_connector_type": true, "endpoint": true}
	extraBody := map[string]interface{}{}
	for k, v := range m.Options {
		if reserved[k] || caps[k] != nil {
			continue
		}
		extraBody[k] = v
	}
	if len(extraBody) > 0 {
		opts["extra_body"] = extraBody
	}

	connType := p.Type
	if ct, ok := m.Options["_connector_type"].(string); ok && ct != "" {
		connType = ct
	}

	name := m.Name
	if name == "" {
		name = m.ID
	}
	// TypeSafe connector needs endpoint in opts directly (not in extra_body)
	if connType == "typesafe" {
		if ep, ok := m.Options["endpoint"].(string); ok && ep != "" {
			opts["endpoint"] = ep
		}
	}

	dsl := map[string]interface{}{
		"type":    connType,
		"name":    name,
		"label":   name,
		"options": opts,
	}
	if m.Metadata != nil {
		dsl["metadata"] = m.Metadata
	}

	if p.PresetKey == "azure" {
		dsl["auth_mode"] = "api-key"
	}

	return json.Marshal(dsl)
}

// unregisterConnector removes the provider's connector from the runtime.
func unregisterConnector(p *Provider) error {
	if p.Source == ProviderSourceBuiltIn {
		return nil
	}
	cid := p.ConnectorID
	if cid == "" {
		cid = connectorID(p)
	}

	for _, m := range p.Models {
		_ = connector.Unregister(cid + ":" + m.ID)
	}

	return connector.Unregister(cid)
}

// importFromConnectors scans existing AI connectors loaded by engine.Load
// and imports them as builtin providers into the Registry store.
// If a store record with the same key already exists (dynamic), it is not overwritten.
func importFromConnectors(r *Registry) error {
	for _, opt := range connector.AIConnectors {
		id := opt.Value
		conn, err := connector.Select(id)
		if err != nil {
			continue
		}

		fresh := providerFromConnector(id, conn)
		sk := storeKey(id)

		if r.store.Has(sk) {
			existing, err := storeGet(r.store, r.cache, id, r.encKey)
			if err != nil || existing == nil {
				continue
			}
			if existing.Source != ProviderSourceBuiltIn {
				continue
			}
			if existing.SchemaVersion >= builtinSchemaVersion {
				continue
			}

			// Selective merge: take fresh connector data, preserve user adjustments
			enabledMap := make(map[string]bool)
			for _, m := range existing.Models {
				enabledMap[m.ID] = m.Enabled
			}
			for i := range fresh.Models {
				if enabled, ok := enabledMap[fresh.Models[i].ID]; ok {
					fresh.Models[i].Enabled = enabled
				}
			}
			fresh.Enabled = existing.Enabled
			fresh.Status = existing.Status

			if err := storeSet(r.store, r.cache, &fresh, r.encKey); err != nil {
				continue
			}
			continue
		}

		// New connector — normal import
		if err := storeSet(r.store, r.cache, &fresh, r.encKey); err != nil {
			continue
		}
		_ = indexAdd(r.store, r.cache, id)
	}
	return nil
}

// providerFromConnector builds a Provider from a runtime Connector interface.
func providerFromConnector(id string, conn connector.Connector) Provider {
	meta := conn.GetMetaInfo()
	setting := conn.Setting()

	name := meta.Label
	if name == "" {
		name = id
	}

	typ := connectorType(conn)

	var apiURL, apiKey, model string
	if lc, ok := conn.(goullm.LLMConnector); ok {
		apiURL = lc.GetURL()
		apiKey = lc.GetKey()
		model = lc.GetModel()
	}
	if apiURL == "" {
		apiURL, _ = setting["host"].(string)
	}
	if apiKey == "" {
		apiKey, _ = setting["key"].(string)
	}
	if model == "" {
		model, _ = setting["model"].(string)
	}

	var models []ModelInfo
	if model != "" {
		var caps []string
		if lc, ok := conn.(goullm.LLMConnector); ok {
			if c := lc.GetCapabilities(); c != nil {
				caps = capabilitiesFromCapabilities(c)
			}
		}
		if len(caps) == 0 {
			caps = capabilitiesFromSetting(setting)
		}
		models = []ModelInfo{{
			ID:           model,
			Name:         model,
			Capabilities: caps,
			Enabled:      true,
			Options:      extractOptionsFromSetting(setting),
			Metadata:     conn.GetMetadata(),
		}}
	}

	return Provider{
		Key:           id,
		ConnectorID:   id,
		Name:          name,
		Type:          typ,
		APIURL:        apiURL,
		APIKey:        apiKey,
		Models:        models,
		Enabled:       true,
		Status:        "connected",
		Source:        ProviderSourceBuiltIn,
		Owner:         ProviderOwner{Type: "system"},
		SchemaVersion: builtinSchemaVersion,
	}
}

// capabilitiesFromCapabilities converts a typed Capabilities struct to a string slice.
func capabilitiesFromCapabilities(c *goullm.Capabilities) []string {
	var out []string
	if c.Streaming {
		out = append(out, "streaming")
	}
	if c.ToolCalls {
		out = append(out, "tool_calls")
	}
	if c.TemperatureAdjustable {
		out = append(out, "temperature_adjustable")
	}
	if c.Vision != nil {
		switch v := c.Vision.(type) {
		case bool:
			if v {
				out = append(out, "vision")
			}
		case string:
			if v != "" {
				out = append(out, "vision")
			}
		}
	}
	if c.Audio {
		out = append(out, "audio")
	}
	if c.STT {
		out = append(out, "stt")
	}
	if c.Reasoning {
		out = append(out, "reasoning")
	}
	if c.JSON {
		out = append(out, "json")
	}
	if c.Multimodal {
		out = append(out, "multimodal")
	}
	if c.Embedding {
		out = append(out, "embedding")
	}
	if c.ImageGeneration {
		out = append(out, "image_generation")
	}
	if c.HasImageEditing() {
		out = append(out, "image_editing")
	}
	if c.OCR {
		out = append(out, "ocr")
	}
	if c.Decision {
		out = append(out, "decision")
	}
	return out
}

func connectorType(conn connector.Connector) string {
	switch {
	case conn.Is(6): // OPENAI
		return "openai"
	case conn.Is(11): // ANTHROPIC
		return "anthropic"
	case conn.Is(9): // FASTEMBED
		return "fastembed"
	case conn.Is(8): // MOAPI
		return "moapi"
	default:
		return "custom"
	}
}

// extractOptionsFromSetting extracts model options (reasoning_effort, thinking)
// from a connector's Setting() map for builtin connectors.
// openai.Setting() merges ExtraBody entries to the top level, so reasoning_effort
// is read directly from setting, not from a nested "extra_body" key.
func extractOptionsFromSetting(setting map[string]interface{}) map[string]interface{} {
	if setting == nil {
		return nil
	}

	opts := make(map[string]interface{})

	// reasoning_effort is at the top level (merged from ExtraBody by openai.Setting)
	if effort, ok := setting["reasoning_effort"].(string); ok {
		opts["reasoning_effort"] = effort
	}

	// thinking config (e.g., {"type": "enabled", "budget_tokens": 8192})
	if thinking, ok := setting["thinking"]; ok {
		opts["thinking"] = thinking
	}

	if len(opts) == 0 {
		return nil
	}
	return opts
}

func capabilitiesFromSetting(setting map[string]interface{}) []string {
	raw, ok := setting["capabilities"]
	if !ok {
		return nil
	}

	switch caps := raw.(type) {
	case map[string]interface{}:
		var out []string
		for k, v := range caps {
			if b, ok := v.(bool); ok && b {
				out = append(out, k)
			}
		}
		return out
	default:
		return nil
	}
}
