package llmprovider

import (
	_ "embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed presets.yml
var presetsYAML []byte

var presets []ProviderPreset

func init() {
	presets = loadPresets()
}

func loadPresets() []ProviderPreset {
	var list []ProviderPreset
	if err := yaml.Unmarshal(presetsYAML, &list); err != nil {
		panic("llmprovider: failed to parse presets.yml: " + err.Error())
	}
	return list
}

// GetPresets returns a copy of the embedded preset list.
func GetPresets() []ProviderPreset {
	out := make([]ProviderPreset, len(presets))
	copy(out, presets)
	return out
}

// GetPresetsForLocale returns presets filtered by locale.
// Presets with empty Locale are always included (global).
// Presets with a non-empty Locale are included only when it matches.
func GetPresetsForLocale(locale string) []ProviderPreset {
	norm := strings.ToLower(locale)
	var out []ProviderPreset
	for _, p := range presets {
		if p.Locale == "" || strings.ToLower(p.Locale) == norm {
			out = append(out, p)
		}
	}
	return out
}

// GetPreset returns the preset for the given key, or nil if not found.
func GetPreset(key string) *ProviderPreset {
	for i := range presets {
		if presets[i].Key == key {
			cp := presets[i]
			return &cp
		}
	}
	return nil
}

// RegisterPreset adds or updates a dynamic preset in the global list.
// New entries are prepended so they appear first; existing entries are updated in place.
func RegisterPreset(p ProviderPreset) {
	for i := range presets {
		if presets[i].Key == p.Key {
			presets[i] = p
			return
		}
	}
	presets = append([]ProviderPreset{p}, presets...)
}
