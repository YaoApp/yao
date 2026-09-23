package llm

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	agentllm "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Provider represents an LLM provider option
type Provider struct {
	Label        string                 `json:"label"`
	Value        string                 `json:"value"`
	Type         string                 `json:"type"`              // "openai"
	Builtin      bool                   `json:"builtin"`           // true for system built-in, false for user-defined
	Capabilities map[string]interface{} `json:"capabilities"`      // Model capabilities from connector settings
	Options      map[string]interface{} `json:"options,omitempty"` // Model options: reasoning_effort, thinking_levels, model
}

// modelEntry associates a ModelInfo with its parent Provider for sibling lookup.
type modelEntry struct {
	model    *llmprovider.ModelInfo
	provider *llmprovider.Provider
}

// Attach attaches the LLM management handlers to the router with OAuth protection
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {

	// Create providers group with OAuth guard
	group.Use(oauth.Guard)

	// LLM Providers endpoints
	group.GET("/providers", listProviders)      // GET /providers - List all LLM providers
	group.GET("/model-groups", listModelGroups) // GET /model-groups - Grouped models with effort levels

	// OpenAI-compatible endpoints
	group.POST("/chat/completions", handleChatCompletions) // POST /chat/completions - LLM proxy
	group.GET("/models", handleListModels)                 // GET /models - List available models
}

// listProviders lists all available LLM providers (built-in + user-defined).
// Supports filtering by capabilities using query parameter: ?filters=vision,tool_calls,audio
func listProviders(c *gin.Context) {
	filtersParam := c.Query("filters")
	var filters []string
	if filtersParam != "" {
		filters = strings.Split(filtersParam, ",")
		for i, filter := range filters {
			filters[i] = strings.TrimSpace(strings.ToLower(filter))
		}
	}

	info := authorized.GetInfo(c)
	allProviders := collectProviders(info, filters)
	response.RespondWithSuccess(c, response.StatusOK, allProviders)
}

// ModelGroupsResponse is the response for GET /llm/model-groups.
type ModelGroupsResponse struct {
	Groups           []ProviderGroup `json:"groups"`
	DefaultConnector string          `json:"default_connector"`
}

// ProviderGroup groups models by provider/vendor.
type ProviderGroup struct {
	Name   string       `json:"name"`
	Models []ModelGroup `json:"models"`
}

// ModelGroup aggregates connectors of the same model family with different reasoning efforts.
type ModelGroup struct {
	ModelName   string            `json:"model_name"`
	ModelFamily string            `json:"model_family"`
	Connectors  map[string]string `json:"connectors"`
	Levels      []string          `json:"levels"`
}

// listModelGroups returns model groups with effort levels for the model selector.
func listModelGroups(c *gin.Context) {
	filtersParam := c.Query("filters")
	var filters []string
	if filtersParam != "" {
		filters = strings.Split(filtersParam, ",")
		for i, filter := range filters {
			filters[i] = strings.TrimSpace(strings.ToLower(filter))
		}
	}

	info := authorized.GetInfo(c)
	providers := collectProviders(info, filters)
	modelIndex := buildModelIndex()
	groups := buildGroups(providers, modelIndex)

	defaultConn := ""
	if llmprovider.Global != nil {
		if cid, err := llmprovider.Global.GetRoleBy("default", info); err == nil {
			defaultConn = cid
		}
	}

	response.RespondWithSuccess(c, response.StatusOK, ModelGroupsResponse{
		Groups:           groups,
		DefaultConnector: defaultConn,
	})
}

// collectProviders builds the flat provider list (shared by listProviders and listModelGroups).
func collectProviders(info *oauthTypes.AuthorizedInfo, filters []string) []Provider {
	var opts []connector.Option
	if llmprovider.Global != nil {
		opts = llmprovider.Global.ListModelsBy(info)
	} else {
		opts = connector.AIConnectors
	}

	allProviders := make([]Provider, 0, len(opts))
	for _, opt := range opts {
		var conn connector.Connector
		var err error
		if llmprovider.Global != nil {
			conn, err = llmprovider.Global.GetModel(opt.Value)
		} else {
			conn, err = connector.Select(opt.Value)
		}
		if err != nil {
			continue
		}

		connType := connectorType(conn)
		if connType != "openai" && connType != "anthropic" {
			continue
		}

		capabilities := getCapabilitiesFromConn(conn)
		if isNonChatModel(capabilities) && !hasFilter(filters, "embedding") && !hasFilter(filters, "image_generation") {
			continue
		}
		if len(filters) > 0 && !matchesFilters(capabilities, filters) {
			continue
		}

		allProviders = append(allProviders, Provider{
			Label:        opt.Label,
			Value:        opt.Value,
			Type:         connType,
			Builtin:      conn.GetMetaInfo().Builtin,
			Capabilities: capabilities,
		})
	}
	return allProviders
}

// effortOrder defines the fixed display order for reasoning effort levels.
var effortOrder = map[string]int{
	"none": 0, "low": 1, "medium": 2, "high": 3, "max": 4, "thinking": 5,
}

// buildGroups aggregates providers into ProviderGroups by model_family metadata.
// Connectors that share the same model_family are merged into one ModelGroup.
// Connectors without metadata remain as individual entries.
func buildGroups(providers []Provider, modelIndex map[string]*modelEntry) []ProviderGroup {
	// merged tracks ModelGroups keyed by model_family (metadata-driven merge).
	type merged struct {
		modelName  string
		connectors map[string]string
		groupName  string // ProviderGroup display name (first connector determines it)
	}

	familyMap := make(map[string]*merged)
	familyOrder := make([]string, 0)

	for _, p := range providers {
		var meta map[string]interface{}
		groupName := p.Label

		if modelIndex != nil {
			if entry := modelIndex[p.Value]; entry != nil {
				if entry.model != nil {
					meta = entry.model.Metadata
				}
				if entry.provider != nil {
					groupName = entry.provider.Name
				}
			}
		}

		// metadata group_name takes priority over provider.Name
		if meta != nil {
			if v, ok := meta["group_name"].(string); ok && v != "" {
				groupName = v
			}
		}

		// No metadata → individual entry, no merging
		if meta == nil {
			family := p.Value
			familyMap[family] = &merged{
				modelName:  p.Label,
				connectors: map[string]string{"none": p.Value},
				groupName:  groupName,
			}
			familyOrder = append(familyOrder, family)
			continue
		}

		modelName := p.Label
		modelFamily := p.Value
		effort := ""

		if v, ok := meta["model_name"].(string); ok && v != "" {
			modelName = v
		}
		if v, ok := meta["model_family"].(string); ok && v != "" {
			modelFamily = v
		}
		if v, ok := meta["reasoning_effort"].(string); ok {
			effort = v
		}
		if effort == "" {
			effort = "none"
		}

		mg, exists := familyMap[modelFamily]
		if !exists {
			mg = &merged{
				modelName:  modelName,
				connectors: make(map[string]string),
				groupName:  groupName,
			}
			familyMap[modelFamily] = mg
			familyOrder = append(familyOrder, modelFamily)
		}
		mg.connectors[effort] = p.Value
	}

	// Organize merged families into ProviderGroups
	groupOrder := make([]string, 0)
	groupMap := make(map[string]*ProviderGroup)

	for _, family := range familyOrder {
		mg := familyMap[family]
		pg, exists := groupMap[mg.groupName]
		if !exists {
			pg = &ProviderGroup{Name: mg.groupName}
			groupMap[mg.groupName] = pg
			groupOrder = append(groupOrder, mg.groupName)
		}
		pg.Models = append(pg.Models, ModelGroup{
			ModelName:   mg.modelName,
			ModelFamily: family,
			Connectors:  mg.connectors,
			Levels:      sortedEffortLevels(mg.connectors),
		})
	}

	result := make([]ProviderGroup, 0, len(groupOrder))
	for _, name := range groupOrder {
		result = append(result, *groupMap[name])
	}
	return result
}

// sortedEffortLevels returns the keys of the connectors map sorted by effortOrder.
// Unknown levels are appended at the end in their natural order.
func sortedEffortLevels(connectors map[string]string) []string {
	levels := make([]string, 0, len(connectors))
	for k := range connectors {
		levels = append(levels, k)
	}
	sort.Slice(levels, func(i, j int) bool {
		oi, oki := effortOrder[levels[i]]
		oj, okj := effortOrder[levels[j]]
		if !oki {
			oi = 100
		}
		if !okj {
			oj = 100
		}
		if oi != oj {
			return oi < oj
		}
		return levels[i] < levels[j]
	})
	return levels
}

// connectorType returns the type string for a connector.
func connectorType(conn connector.Connector) string {
	if conn.Is(connector.OPENAI) {
		return "openai"
	}
	if conn.Is(connector.ANTHROPIC) {
		return "anthropic"
	}
	return "unknown"
}

// getCapabilitiesFromConn extracts capabilities from connector settings
func getCapabilitiesFromConn(conn connector.Connector) map[string]interface{} {
	if conn == nil {
		return nil
	}

	caps := agentllm.GetCapabilitiesFromConn(conn)
	return agentllm.ToMap(caps)
}

// isNonChatModel returns true if capabilities indicate a non-chat model (embedding or image generation).
func isNonChatModel(caps map[string]interface{}) bool {
	if v, ok := caps["embedding"].(bool); ok && v {
		return true
	}
	if v, ok := caps["image_generation"].(bool); ok && v {
		return true
	}
	return false
}

// hasFilter checks whether a specific filter string is present in the filters list.
func hasFilter(filters []string, name string) bool {
	for _, f := range filters {
		if f == name {
			return true
		}
	}
	return false
}

// buildModelIndex constructs a connectorID → modelEntry index from the llmprovider Registry.
// Used by buildGroups to look up ModelInfo.Metadata for grouping.
func buildModelIndex() map[string]*modelEntry {
	if llmprovider.Global == nil {
		return nil
	}

	enabled := true
	providers, err := llmprovider.Global.List(&llmprovider.ProviderFilter{
		Source:  llmprovider.ProviderSourceAll,
		Enabled: &enabled,
	})
	if err != nil {
		return nil
	}

	index := make(map[string]*modelEntry)
	for i := range providers {
		p := &providers[i]
		for j := range p.Models {
			m := &p.Models[j]
			if !m.Enabled {
				continue
			}
			entry := &modelEntry{model: m, provider: p}

			// Dynamic provider models use "providerCID:modelID" as connector ID
			if p.Source == llmprovider.ProviderSourceDynamic {
				index[p.ConnectorID+":"+m.ID] = entry
			}
			// Builtin: connectorID == p.ConnectorID; also register by m.ID as fallback
			index[p.ConnectorID] = entry
			index[m.ID] = entry
		}
	}
	return index
}

// matchesFilters checks if capabilities match all requested filters
// Filters are matched case-insensitively and support the following capability keys:
// - vision: true or string value like "openai", "claude"
// - audio: bool (LLM supports audio input/understanding)
// - stt: bool (Speech-to-Text / audio transcription model, e.g. Whisper)
// - tool_calls: bool
// - reasoning: bool
// - streaming: bool
// - json: bool
// - multimodal: bool
// - embedding: bool
// - image_generation: bool
// - image_editing: bool or string
// - ocr: bool (OCR text extraction model, e.g. Qwen3.5-OCR)
// - temperature_adjustable: bool
func matchesFilters(capabilities map[string]interface{}, filters []string) bool {
	if capabilities == nil {
		return false
	}

	// All filters must match (AND logic)
	for _, filter := range filters {
		matched := false

		// Check each capability field
		for key, value := range capabilities {
			keyLower := strings.ToLower(key)

			// Match the filter against capability key
			if keyLower == filter {
				// For vision, check if it's true or a non-empty string
				if filter == "vision" {
					if boolVal, ok := value.(bool); ok && boolVal {
						matched = true
						break
					}
					if strVal, ok := value.(string); ok && strVal != "" {
						matched = true
						break
					}
				} else {
					// For other capabilities, check if it's true
					if boolVal, ok := value.(bool); ok && boolVal {
						matched = true
						break
					}
				}
			}
		}

		// If any filter doesn't match, return false
		if !matched {
			return false
		}
	}

	return true
}
