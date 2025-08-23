package i18n

import (
	"path/filepath"
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
			new = append(new, i18n.Parse(value).(string))
		}
		return new
	}

	return input
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

// Flatten flatten the i18n map
func (i18ns Map) Flatten() Map {
	new := Map{}
	for lang, i18n := range i18ns {
		new[lang] = I18n{Locale: lang, Messages: maps.MapOf(i18n.Messages).Dot()}

		// Add short lang
		parts := strings.Split(lang, "-")

		// en
		if parts[0] != lang {
			new[parts[0]] = new[lang]
		}

		// us
		if len(parts) > 1 {
			new[parts[1]] = new[lang]
		}
	}
	return new
}

// FlattenWithGlobal flatten the i18n map with global i18n
func (i18ns Map) FlattenWithGlobal() Map {

	// New i18n map
	new := Map{}

	// Global i18n
	globalI18ns, hasGlobal := Locales["__global__"]

	// Extend the i18n map with global i18n
	for lang, i18n := range i18ns {
		new[lang] = I18n{Locale: lang, Messages: maps.MapOf(i18n.Messages).Dot()}
		if hasGlobal {
			if global, has := globalI18ns[lang]; has {
				for key, value := range global.Messages {
					if _, ok := new[lang].Messages[key]; !ok {
						new[lang].Messages[key] = value
					}
				}
			}
		}

		// Add short lang
		parts := strings.Split(lang, "-")

		// en
		if parts[0] != lang {
			new[parts[0]] = new[lang]
		}

		// us
		if len(parts) > 1 {
			new[parts[1]] = new[lang]
		}
	}

	return new
}

// Translate translate the input
func Translate(assistantID string, locale string, input any) any {

	locale = strings.ToLower(strings.TrimSpace(locale))
	i18ns, has := Locales[assistantID]
	if !has {
		i18ns = map[string]I18n{}
	}

	i18n, has := i18ns[locale]
	if !has {
		parts := strings.Split(locale, "-")
		if len(parts) > 1 {
			i18n, has = i18ns[parts[1]]
		}
		if !has {
			i18n, has = i18ns[parts[0]]
		}
	}

	if !has {
		var hasGlobal bool = false
		i18ns, hasGlobal = Locales["__global__"]
		if hasGlobal {
			i18n, has = i18ns[locale]
		}
	}

	if has {
		return i18n.Parse(input)
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

	i18n, has := i18ns[locale]
	if !has {
		parts := strings.Split(locale, "-")
		if len(parts) > 1 {
			i18n, has = i18ns[parts[1]]
		}
		if !has {
			i18n, has = i18ns[parts[0]]
		}
	}

	if has {
		return i18n.Parse(input)
	}
	return input
}
