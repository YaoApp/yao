package i18n

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/maps"
)

// Locales the locales
var Locales = map[string]map[string]I18n{}

// I18n the i18n struct
type I18n struct {
	Locale   string         `json:"locale,omitempty" yaml:"locale,omitempty"`
	Messages map[string]any `json:"messages,omitempty" yaml:"messages,omitempty"`
}

// Map the i18n map
type Map map[string]I18n

// Parse parse the input
func (i18n I18n) Parse(input any) any {
	if input == nil {
		return nil
	}

	switch in := input.(type) {
	case string:
		return i18n.parseString(in)

	case map[string]any:
		new := map[string]any{}
		for key, value := range in {
			new[key] = i18n.Parse(value)
		}
		return new

	case []any:
		new := []any{}
		for _, value := range in {
			new = append(new, i18n.Parse(value))
		}
		return new

	case []string:
		new := []string{}
		for _, value := range in {
			if parsed := i18n.Parse(value); parsed != nil {
				if s, ok := parsed.(string); ok {
					new = append(new, s)
				} else {
					new = append(new, value)
				}
			} else {
				new = append(new, value)
			}
		}
		return new
	}

	return input
}

// parseString parse a string value
func (i18n I18n) parseString(in string) string {
	trimed := strings.TrimSpace(in)

	// Check if it's a direct message key (no template markers)
	if !strings.Contains(trimed, "{{") && !strings.Contains(trimed, "}}") {
		if val, ok := i18n.Messages[trimed]; ok {
			if s, ok := val.(string); ok {
				return s
			}
		}
		return in
	}

	// Check if it's a full template expression {{...}} (exact match - entire string is one template)
	hasExp := strings.HasPrefix(trimed, "{{") && strings.HasSuffix(trimed, "}}")
	if hasExp {
		// Check if there's only ONE template pattern (no text before/after or multiple templates)
		re := regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
		matches := re.FindAllString(trimed, -1)

		// Only treat as full template if there's exactly one match and it equals the trimmed string
		if len(matches) == 1 && matches[0] == trimed {
			exp := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(trimed, "}}"), "{{"))
			if val, ok := i18n.Messages[exp]; ok {
				if s, ok := val.(string); ok {
					return s
				}
			}
			return in
		}
	}

	// Handle embedded template variables: "text {{var}} more {{var2}}"
	if strings.Contains(in, "{{") && strings.Contains(in, "}}") {
		result := in
		// Use regex to find all {{...}} patterns
		re := regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
		matches := re.FindAllStringSubmatch(in, -1)

		for _, match := range matches {
			if len(match) >= 2 {
				fullMatch := match[0]                  // Full match including {{ }}
				varName := strings.TrimSpace(match[1]) // Variable name without {{ }}

				// Try to replace with value from Messages
				if val, ok := i18n.Messages[varName]; ok {
					if s, ok := val.(string); ok {
						result = strings.Replace(result, fullMatch, s, 1)
					}
				}
			}
		}
		return result
	}

	return in
}

// GetLocales get the locales from path
func GetLocales(path string) (Map, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, err
	}

	i18ns := Map{}
	localesdir := filepath.Join(path, "locales")
	if has, _ := app.Exists(localesdir); has {
		locales, err := app.ReadDir(localesdir, true)
		if err != nil {
			return nil, err
		}

		// load locales
		for _, locale := range locales {
			localeData, err := app.ReadFile(locale)
			if err != nil {
				return nil, err
			}
			var messages maps.Map = map[string]any{}
			err = application.Parse(locale, localeData, &messages)
			if err != nil {
				return nil, err
			}
			name := strings.ToLower(strings.TrimSuffix(filepath.Base(locale), ".yml"))
			i18ns[name] = I18n{Locale: name, Messages: messages}
		}
	}
	return i18ns, nil
}

// Flatten flattens the map of locales by adding short language codes and region codes
// e.g., "en-us" will also create "en" and "us" entries
// If __global__ locales exist, they are merged (local/user messages override global built-in messages)
func (m Map) Flatten() Map {
	flattened := make(Map)

	// First, process local messages with Dot() flattening
	for localeCode, i18n := range m {
		// Flatten nested messages to dot notation (e.g., {"local": {"key": "value"}} -> {"local.key": "value"})
		flattened[localeCode] = I18n{
			Locale:   localeCode,
			Messages: maps.MapOf(i18n.Messages).Dot(),
		}

		// Add short language codes
		parts := strings.Split(localeCode, "-")
		if len(parts) > 1 {
			// Add short language code (e.g., "en" from "en-us")
			if _, ok := flattened[parts[0]]; !ok {
				flattened[parts[0]] = flattened[localeCode]
			}
			// Add region code (e.g., "us" from "en-us")
			if _, ok := flattened[parts[1]]; !ok {
				flattened[parts[1]] = flattened[localeCode]
			}
		}
	}

	// Merge with global locales if they exist
	// Strategy: Start with global (built-in), then override with local (user)
	globalLocales, hasGlobal := Locales["__global__"]
	if !hasGlobal {
		return flattened
	}

	for globalLocaleCode, globalI18n := range globalLocales {
		// Ensure global messages are also flattened (though builtin.go already uses flat keys)
		globalFlattened := maps.MapOf(globalI18n.Messages).Dot()

		if localI18n, ok := flattened[globalLocaleCode]; ok {
			// Both global and local exist: merge with local overriding global
			mergedMessages := make(map[string]any)
			// First copy all global messages
			for k, v := range globalFlattened {
				mergedMessages[k] = v
			}
			// Then override with local messages
			for k, v := range localI18n.Messages {
				mergedMessages[k] = v
			}
			flattened[globalLocaleCode] = I18n{
				Locale:   globalLocaleCode,
				Messages: mergedMessages,
			}
		} else {
			// Only global exists, add it with flattened messages
			flattened[globalLocaleCode] = I18n{
				Locale:   globalLocaleCode,
				Messages: globalFlattened,
			}
		}
	}

	return flattened
}

// FlattenWithGlobal is deprecated. Use Flatten() instead, which now automatically merges with global locales.
// Kept for backward compatibility.
func (m Map) FlattenWithGlobal() Map {
	return m.Flatten()
}

// Translate translate the input with recursive variable resolution
// Fallback strategy: assistant locale -> assistant short codes -> global locale -> global short codes
func Translate(assistantID string, locale string, input any) any {

	locale = strings.ToLower(strings.TrimSpace(locale))

	// Helper function to try translation with a specific i18n object
	tryTranslate := func(i18n I18n, input any) (any, bool) {
		result := i18n.Parse(input)
		// For string input, check if translation was found by comparing with input
		// For other types, Parse always returns a result (transformed or original)
		if inputStr, ok := input.(string); ok {
			if resultStr, ok := result.(string); ok {
				// Translation found if result is different from input
				if resultStr != inputStr {
					return result, true
				}
				return input, false
			}
		}
		// For non-string inputs (maps, slices), Parse always processes them
		return result, true
	}

	// Helper function to process recursive templates
	processTemplates := func(result any, assistantID string, locale string) any {
		if resultStr, ok := result.(string); ok && strings.Contains(resultStr, "{{") && strings.Contains(resultStr, "}}") {
			re := regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
			resultStr = re.ReplaceAllStringFunc(resultStr, func(match string) string {
				varName := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{"))
				translated := Translate(assistantID, locale, varName)
				if translatedStr, ok := translated.(string); ok && translatedStr != varName {
					return translatedStr
				}
				return match
			})
			return resultStr
		}
		return result
	}

	// Try assistant locale first
	if i18ns, has := Locales[assistantID]; has {
		// Try exact locale
		if i18n, hasLocale := i18ns[locale]; hasLocale {
			if result, found := tryTranslate(i18n, input); found {
				return processTemplates(result, assistantID, locale)
			}
		}

		// Try short codes
		parts := strings.Split(locale, "-")
		if len(parts) > 1 {
			if i18n, hasLocale := i18ns[parts[1]]; hasLocale {
				if result, found := tryTranslate(i18n, input); found {
					return processTemplates(result, assistantID, locale)
				}
			}
			if i18n, hasLocale := i18ns[parts[0]]; hasLocale {
				if result, found := tryTranslate(i18n, input); found {
					return processTemplates(result, assistantID, locale)
				}
			}
		}
	}

	// Fallback to global locales
	if globalI18ns, hasGlobal := Locales["__global__"]; hasGlobal {
		// Try exact locale
		if i18n, hasLocale := globalI18ns[locale]; hasLocale {
			if result, found := tryTranslate(i18n, input); found {
				return processTemplates(result, assistantID, locale)
			}
		}

		// Try short codes
		parts := strings.Split(locale, "-")
		if len(parts) > 1 {
			if i18n, hasLocale := globalI18ns[parts[1]]; hasLocale {
				if result, found := tryTranslate(i18n, input); found {
					return processTemplates(result, assistantID, locale)
				}
			}
			if i18n, hasLocale := globalI18ns[parts[0]]; hasLocale {
				if result, found := tryTranslate(i18n, input); found {
					return processTemplates(result, assistantID, locale)
				}
			}
		}
	}

	return input
}

// TranslateGlobal translate the input with global i18n
func TranslateGlobal(locale string, input any) any {
	locale = strings.ToLower(strings.TrimSpace(locale))
	i18ns, has := Locales["__global__"]
	if !has {
		i18ns = map[string]I18n{}
	}

	// Try the exact locale first
	i18n, has := i18ns[locale]
	if has {
		result := i18n.Parse(input)
		// If the result is the same as input (not translated), try fallback
		if result != input {
			return result
		}
	}

	// Fallback logic: for "en-us", try "en"
	parts := strings.Split(locale, "-")
	if len(parts) > 1 {
		// Try the language code (e.g., "en" for "en-us")
		if fallbackI18n, hasFallback := i18ns[parts[0]]; hasFallback {
			result := fallbackI18n.Parse(input)
			if result != input {
				return result
			}
		}
		// Try the country code (e.g., "us" for "en-us")
		if fallbackI18n, hasFallback := i18ns[parts[1]]; hasFallback {
			result := fallbackI18n.Parse(input)
			if result != input {
				return result
			}
		}
	}

	return input
}

// T is a short alias for TranslateGlobal that returns string
// Usage: i18n.T(ctx.Locale, "assistant.agent.stream.label")
// Variables in templates like {{variable}} will be recursively resolved from the global language pack
func T(locale string, key string) string {
	result := TranslateGlobal(locale, key)
	if str, ok := result.(string); ok {
		return str
	}
	return key
}

// Tr translates with assistantID and returns string
// Supports recursive translation of {{variable}} templates
// Usage: i18n.Tr(assistantID, locale, "key")
func Tr(assistantID string, locale string, key string) string {
	result := Translate(assistantID, locale, key)
	if str, ok := result.(string); ok {
		return str
	}
	return key
}
