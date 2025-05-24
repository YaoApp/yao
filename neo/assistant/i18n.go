package assistant

import (
	"strings"
)

// Locales the locales
var Locales = map[string]map[string]I18n{}

// I18n the i18n struct
type I18n struct {
	Locale   string         `json:"locale,omitempty" yaml:"locale,omitempty"`
	Messages map[string]any `json:"messages,omitempty" yaml:"messages,omitempty"`
}

// Parse parse the input
func (i18n I18n) Parse(input any) any {

	switch in := input.(type) {
	case string:
		trimed := strings.TrimSpace(in)
		hasExp := strings.HasPrefix(trimed, "{{") && strings.HasSuffix(trimed, "}}")
		if hasExp {
			exp := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(trimed, "}}"), "{{"))
			if _, ok := i18n.Messages[exp]; ok {
				return i18n.Messages[exp]
			}
			return exp
		}

		if _, ok := i18n.Messages[trimed]; ok {
			return i18n.Messages[trimed]
		}

		return in

	case map[string]any:
		for key, value := range in {
			in[key] = i18n.Parse(value)
		}
		return in

	case []any:
		for i, value := range in {
			in[i] = i18n.Parse(value)
		}
		return in
	}

	return input
}
