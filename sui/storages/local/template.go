package local

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/sui/core"
)

// NewTemplate create a new local template
func (local *Local) NewTemplate(path string) (*Template, error) {
	id := local.GetTemplateID(path)

	tmpl := Template{
		Path: path,
		Template: &core.Template{
			ID:          id,
			Name:        strings.ToUpper(id),
			Version:     1,
			Screenshots: []string{},
		}}

	// load the template.json
	configFile := filepath.Join(path, fmt.Sprintf("%s.json", id))
	if local.fs.IsFile(configFile) {
		configBytes, err := local.fs.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		err = jsoniter.Unmarshal(configBytes, &tmpl.Template)
		if err != nil {
			return nil, err
		}
	}

	return &tmpl, nil
}

// GetTemplateID get the template ID
func (local *Local) GetTemplateID(path string) string {
	return filepath.Base(path)
}

// Get get the template
func (tmpl *Template) Get() error {
	return nil
}

// Save save the template
func (tmpl *Template) Save() error {
	return nil
}

// Pages get the pages
func (tmpl *Template) Pages() ([]core.IPage, error) {
	return nil, nil
}

// Blocks get the blocks
func (tmpl *Template) Blocks() ([]core.IBlock, error) {
	return nil, nil
}

// Components get the components
func (tmpl *Template) Components() ([]core.IComponent, error) {
	return nil, nil
}

// Page get the page
func (tmpl *Template) Page(route string) (core.IPage, error) {
	return nil, nil
}

// Block get the block
func (tmpl *Template) Block(name string) (core.IBlock, error) {
	return nil, nil
}

// Component get the component
func (tmpl *Template) Component(name string) (core.IComponent, error) {
	return nil, nil
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
