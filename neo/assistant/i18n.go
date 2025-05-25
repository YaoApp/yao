package assistant

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

// GetI18n load the i18n from path
func GetI18n(path string) (map[string]I18n, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, err
	}

	// Get the global i18n
	globalI18ns, hasGlobal := Locales["__global__"]
	// i18ns
	localesdir := filepath.Join(path, "locales")
	var i18ns map[string]I18n = map[string]I18n{}
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
			// Merge the global i18n
			if hasGlobal {
				global, has := globalI18ns[name]
				if has {
					for key, value := range global.Messages {
						if _, ok := messages[key]; !ok {
							messages[key] = value
						}
					}
				}
			}

			i18ns[name] = I18n{Locale: name, Messages: messages.Dot()}
			namer := strings.Split(name, "-")
			if len(namer) > 1 {
				// Merge the global i18n
				if hasGlobal {
					global, has := globalI18ns[namer[0]]
					if has {
						for key, value := range global.Messages {
							if _, ok := messages[key]; !ok {
								messages[key] = value
							}
						}
					}
				}
				i18ns[namer[0]] = I18n{Locale: name, Messages: messages.Dot()}
			}

		}
	}
	return i18ns, nil
}

// Translate translate the input
func Translate(locale string, id string, input any) any {

	locale = strings.ToLower(strings.TrimSpace(locale))
	i18ns, has := Locales[id]
	if !has {
		i18ns = map[string]I18n{}
	}

	i18n, has := i18ns[locale]
	if !has {
		namer := strings.Split(locale, "-")
		lang := namer[0]
		i18n, has = i18ns[lang]
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
