package llm

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed presets/*.yml
var presetsFS embed.FS

// PresetProvider holds all preset models for one provider, grouped by type.
type PresetProvider struct {
	ProviderID     string        `yaml:"provider_id"`
	Models         []PresetModel `yaml:"models,omitempty"`
	ImageModels    []PresetModel `yaml:"image_models,omitempty"`
	DecisionModels []PresetModel `yaml:"decision_models,omitempty"`
}

// PresetModel is a single model entry in the preset catalog.
// Fields align with the Model struct; only non-zero values are populated.
type PresetModel struct {
	ModelID  string   `yaml:"model_id"`
	Name     string   `yaml:"name"`
	API      string   `yaml:"api,omitempty"`
	ModelAPI string   `yaml:"model_api,omitempty"`
	BaseURL  string   `yaml:"base_url,omitempty"`
	Input    []string `yaml:"input,omitempty"`
	Output   []string `yaml:"output,omitempty"`

	ContextWindow   int  `yaml:"context_window,omitempty"`
	MaxOutputTokens int  `yaml:"max_output_tokens,omitempty"`
	Reasoning       bool `yaml:"reasoning,omitempty"`

	Capabilities []string `yaml:"capabilities,omitempty"`

	ThinkingLevelMap map[string]*string `yaml:"thinking_level_map,omitempty"`
	SamplingParams   map[string]any     `yaml:"sampling_params,omitempty"`

	InputCostPer1M      float64 `yaml:"input_cost_per_1m,omitempty"`
	OutputCostPer1M     float64 `yaml:"output_cost_per_1m,omitempty"`
	CacheReadCostPer1M  float64 `yaml:"cache_read_cost_per_1m,omitempty"`
	CacheWriteCostPer1M float64 `yaml:"cache_write_cost_per_1m,omitempty"`

	CostTiers   []PresetCostTier   `yaml:"cost_tiers,omitempty"`
	InputLimits *PresetInputLimits `yaml:"input_limits,omitempty"`
	PromptCache map[string]int     `yaml:"prompt_cache,omitempty"`
	Headers     map[string]string  `yaml:"headers,omitempty"`
	Compat      map[string]any     `yaml:"compat,omitempty"`

	Enabled *bool  `yaml:"enabled,omitempty"`
	Lab     string `yaml:"lab,omitempty"`
}

// PresetCostTier is a volume-based pricing tier in the preset catalog.
type PresetCostTier struct {
	InputTokensAbove int     `yaml:"input_tokens_above"`
	Input            float64 `yaml:"input"`
	Output           float64 `yaml:"output"`
	CacheRead        float64 `yaml:"cache_read,omitempty"`
	CacheWrite       float64 `yaml:"cache_write,omitempty"`
}

// PresetInputLimits mirrors ModelInputLimits in YAML-friendly form.
type PresetInputLimits struct {
	MaxRequestBytes int                    `yaml:"max_request_bytes,omitempty"`
	Images          *PresetImageInputLimit `yaml:"images,omitempty"`
}

// PresetImageInputLimit mirrors ModelImageInputLimits in YAML-friendly form.
type PresetImageInputLimit struct {
	MaxPerMessage int                `yaml:"max_per_message,omitempty"`
	MaxPerRequest int                `yaml:"max_per_request,omitempty"`
	Resize        *PresetImageResize `yaml:"resize,omitempty"`
}

// PresetImageResize mirrors ModelImageResizeOptions in YAML-friendly form.
type PresetImageResize struct {
	MaxWidth    int `yaml:"max_width,omitempty"`
	MaxHeight   int `yaml:"max_height,omitempty"`
	MaxBytes    int `yaml:"max_bytes,omitempty"`
	JPEGQuality int `yaml:"jpeg_quality,omitempty"`
}

var (
	presetOnce      sync.Once
	presetProviders map[string]*PresetProvider // keyed by provider_id
	presetIndex     map[string]*PresetModel    // keyed by "provider_id/model_id"
)

func ensurePresets() {
	presetOnce.Do(func() {
		presetProviders = make(map[string]*PresetProvider)
		presetIndex = make(map[string]*PresetModel)
		loadPresetsFromFS(presetsFS)
	})
}

func loadPresetsFromFS(fsys embed.FS) {
	entries, err := fs.ReadDir(fsys, "presets")
	if err != nil {
		panic("llm: failed to read presets directory: " + err.Error())
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		data, err := fsys.ReadFile("presets/" + entry.Name())
		if err != nil {
			panic(fmt.Sprintf("llm: failed to read preset file %s: %v", entry.Name(), err))
		}

		var pp PresetProvider
		if err := yaml.Unmarshal(data, &pp); err != nil {
			panic(fmt.Sprintf("llm: failed to parse preset file %s: %v", entry.Name(), err))
		}

		presetProviders[pp.ProviderID] = &pp

		for i := range pp.Models {
			m := &pp.Models[i]
			presetIndex[pp.ProviderID+"/"+m.ModelID] = m
		}
		for i := range pp.ImageModels {
			m := &pp.ImageModels[i]
			presetIndex[pp.ProviderID+"/"+m.ModelID] = m
		}
		for i := range pp.DecisionModels {
			m := &pp.DecisionModels[i]
			presetIndex[pp.ProviderID+"/"+m.ModelID] = m
		}
	}
}

// LookupPreset returns the preset model for the given provider and model ID.
// Returns nil if not found.
func LookupPreset(providerID, modelID string) *PresetModel {
	ensurePresets()
	m, ok := presetIndex[providerID+"/"+modelID]
	if !ok {
		return nil
	}
	cp := *m
	return &cp
}

// ListPresets returns all preset models for the given provider.
// Returns nil if the provider is not in the catalog.
func ListPresets(providerID string) []PresetModel {
	ensurePresets()
	pp, ok := presetProviders[providerID]
	if !ok {
		return nil
	}
	out := make([]PresetModel, 0, len(pp.Models)+len(pp.ImageModels)+len(pp.DecisionModels))
	out = append(out, pp.Models...)
	out = append(out, pp.ImageModels...)
	out = append(out, pp.DecisionModels...)
	return out
}

// ListAllPresets returns all preset providers with their model lists.
func ListAllPresets() []PresetProvider {
	ensurePresets()
	out := make([]PresetProvider, 0, len(presetProviders))
	for _, pp := range presetProviders {
		out = append(out, *pp)
	}
	return out
}

// ListPresetProviders returns a sorted list of all provider IDs in the catalog.
func ListPresetProviders() []string {
	ensurePresets()
	ids := make([]string, 0, len(presetProviders))
	for id := range presetProviders {
		ids = append(ids, id)
	}
	return ids
}

// ApplyPreset fills zero-value fields in the given Model from the preset
// catalog. User-set values take priority; only unset fields are populated.
func ApplyPreset(m *Model, providerID string) {
	pm := LookupPreset(providerID, m.ModelID)
	if pm == nil {
		return
	}

	if m.Name == "" {
		m.Name = pm.Name
	}
	if m.ContextWindow == 0 && pm.ContextWindow > 0 {
		m.ContextWindow = pm.ContextWindow
	}
	if m.MaxOutputTokens == 0 && pm.MaxOutputTokens > 0 {
		m.MaxOutputTokens = pm.MaxOutputTokens
	}
	if !m.Reasoning && pm.Reasoning {
		m.Reasoning = pm.Reasoning
	}
	if len(m.Capabilities) == 0 && len(pm.Capabilities) > 0 {
		m.Capabilities = pm.Capabilities
	}
	if len(m.Input) == 0 && len(pm.Input) > 0 {
		m.Input = pm.Input
	}
	if len(m.Output) == 0 && len(pm.Output) > 0 {
		m.Output = pm.Output
	}
	if m.ThinkingLevelMap == nil && pm.ThinkingLevelMap != nil {
		tlm := make(ThinkingLevelMap, len(pm.ThinkingLevelMap))
		for k, v := range pm.ThinkingLevelMap {
			tlm[ThinkingLevel(k)] = v
		}
		m.ThinkingLevelMap = tlm
	}
	if m.MaxThinking == "" && pm.Reasoning && pm.ThinkingLevelMap != nil {
		m.MaxThinking = inferMaxThinking(pm.ThinkingLevelMap)
	}
	if m.InputCostPer1M == 0 && pm.InputCostPer1M != 0 {
		m.InputCostPer1M = pm.InputCostPer1M
	}
	if m.OutputCostPer1M == 0 && pm.OutputCostPer1M != 0 {
		m.OutputCostPer1M = pm.OutputCostPer1M
	}
	if m.CacheReadCostPer1M == 0 && pm.CacheReadCostPer1M != 0 {
		m.CacheReadCostPer1M = pm.CacheReadCostPer1M
	}
	if m.CacheWriteCostPer1M == 0 && pm.CacheWriteCostPer1M != 0 {
		m.CacheWriteCostPer1M = pm.CacheWriteCostPer1M
	}
	if len(m.CostTiers) == 0 && len(pm.CostTiers) > 0 {
		tiers := make([]CostTier, len(pm.CostTiers))
		for i, t := range pm.CostTiers {
			tiers[i] = CostTier{
				InputTokensAbove:    t.InputTokensAbove,
				InputCostPer1M:      t.Input,
				OutputCostPer1M:     t.Output,
				CacheReadCostPer1M:  t.CacheRead,
				CacheWriteCostPer1M: t.CacheWrite,
			}
		}
		m.CostTiers = tiers
	}
	if m.InputLimits == nil && pm.InputLimits != nil {
		m.InputLimits = convertPresetInputLimits(pm.InputLimits)
	}
	if m.PromptCache == nil && pm.PromptCache != nil {
		m.PromptCache = pm.PromptCache
	}
	if m.Compat == nil && pm.Compat != nil {
		m.Compat = pm.Compat
	}
	if m.Lab == "" && pm.Lab != "" {
		m.Lab = pm.Lab
	}

	// SupportsImages / SupportsToolUse from capabilities
	if !m.SupportsImages {
		for _, c := range pm.Capabilities {
			if c == "image" {
				m.SupportsImages = true
				break
			}
		}
	}
	if !m.SupportsToolUse {
		for _, c := range pm.Capabilities {
			if c == "tool_calls" {
				m.SupportsToolUse = true
				break
			}
		}
	}
}

// inferMaxThinking determines the highest thinking level in a map.
func inferMaxThinking(m map[string]*string) ThinkingLevel {
	order := []ThinkingLevel{ThinkingMax, ThinkingXHigh, ThinkingHigh, ThinkingMedium, ThinkingLow, ThinkingMinimal, ThinkingOff}
	for _, lv := range order {
		if _, ok := m[string(lv)]; ok {
			return lv
		}
	}
	return ""
}

func convertPresetInputLimits(p *PresetInputLimits) *ModelInputLimits {
	l := &ModelInputLimits{
		MaxRequestBytes: p.MaxRequestBytes,
	}
	if p.Images != nil {
		l.Images = &ModelImageInputLimits{
			MaxPerMessage: p.Images.MaxPerMessage,
			MaxPerRequest: p.Images.MaxPerRequest,
		}
		if p.Images.Resize != nil {
			l.Images.Resize = &ModelImageResizeOptions{
				MaxWidth:    p.Images.Resize.MaxWidth,
				MaxHeight:   p.Images.Resize.MaxHeight,
				MaxBytes:    p.Images.Resize.MaxBytes,
				JPEGQuality: p.Images.Resize.JPEGQuality,
			}
		}
	}
	return l
}
