package local

import (
	"path/filepath"

	"github.com/yaoapp/yao/sui/core"
	"golang.org/x/text/language"
)

// Assets get the assets treelist
func (tmpl *Template) Assets() []string {
	return nil
}

// Locales get the global locales
func (tmpl *Template) Locales() []core.SelectOption {

	supportLocales := []core.SelectOption{}
	path := filepath.Join(tmpl.Root, "__locales")
	if !tmpl.local.fs.IsDir(path) {
		return nil
	}

	dirs, err := tmpl.local.fs.ReadDir(path, false)
	if err != nil {
		return nil
	}

	for _, dir := range dirs {
		locale := filepath.Base(dir)
		label := language.Make(locale).String()
		supportLocales = append(supportLocales, core.SelectOption{
			Value: locale,
			Label: label,
		})
	}

	return supportLocales
}

// Themes get the global themes
func (tmpl *Template) Themes() []core.SelectOption {
	return tmpl.Template.Themes
}
