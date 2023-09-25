package local

import (
	"path/filepath"
	"strings"
)

// Get get the template
func (tmpl *Template) Get() error {
	return nil
}

// Save save the template
func (tmpl *Template) Save() error {
	return nil
}

func (tmpl *Template) getPageRoute(path string) string {
	return strings.TrimSuffix(path[len(tmpl.Root):], filepath.Ext(path))
}

// Styles get the global styles
func (tmpl *Template) Styles() []string {
	return nil
}

// Locales get the global locales
func (tmpl *Template) Locales() []string {
	return nil
}

// Themes get the global themes
func (tmpl *Template) Themes() []string {
	return nil
}
