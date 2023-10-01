package local

import (
	"fmt"
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

// Asset get the asset
func (tmpl *Template) Asset(file string) (*core.Asset, error) {

	file = filepath.Join(tmpl.Root, "__assets", file)
	if exist, _ := tmpl.local.fs.Exists(file); exist {

		content, err := tmpl.local.fs.ReadFile(file)
		if err != nil {
			return nil, err
		}

		typ := "text/plain"
		switch filepath.Ext(file) {
		case ".css":
			typ = "text/css; charset=utf-8"
			break

		case ".js":
			typ = "application/javascript; charset=utf-8"
			break

		case ".ts":
			typ = "application/javascript; charset=utf-8"
			break

		case ".json":
			typ = "application/json; charset=utf-8"
			break

		case ".html":
			typ = "text/html; charset=utf-8"
			break

		default:
			typ, err = tmpl.local.fs.MimeType(file)
			if err != nil {
				return nil, err
			}
		}

		return &core.Asset{Type: typ, Content: content}, nil
	}

	return nil, fmt.Errorf("Asset %s not found", file)
}
