package llmprovider

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
)

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
		caps[c] = true
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

	reserved := map[string]bool{"host": true, "key": true, "model": true, "capabilities": true, "_connector_type": true}
	extraBody := map[string]interface{}{}
	for k, v := range m.Options {
		if !reserved[k] {
			extraBody[k] = v
		}
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
	dsl := map[string]interface{}{
		"type":    connType,
		"name":    name,
		"label":   name,
		"options": opts,
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
		if r.store.Has(storeKey(id)) {
			continue
		}

		conn, err := connector.Select(id)
		if err != nil {
			continue
		}

		p := providerFromConnector(id, conn)
		m, err := providerToMap(&p, r.encKey)
		if err != nil {
			continue
		}
		sk := storeKey(id)
		_ = r.store.Set(sk, m, 0)
		if r.cache != nil {
			_ = r.cache.Set(sk, m, 0)
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
		}}
	}

	return Provider{
		Key:         id,
		ConnectorID: id,
		Name:        name,
		Type:        typ,
		APIURL:      apiURL,
		APIKey:      apiKey,
		Models:      models,
		Enabled:     true,
		Status:      "connected",
		Source:      ProviderSourceBuiltIn,
		Owner:       ProviderOwner{Type: "system"},
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
