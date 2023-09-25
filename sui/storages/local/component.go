package local

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

// Components get the components
func (tmpl *Template) Components() ([]core.IComponent, error) {

	path := filepath.Join(tmpl.Root, "__components")
	components := []core.IComponent{}
	if exist, _ := tmpl.local.fs.Exists(path); !exist {
		return components, nil
	}

	dirs, err := tmpl.local.fs.ReadDir(path, false)
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if !tmpl.local.fs.IsDir(dir) {
			continue
		}

		block, err := tmpl.getComponentFrom(dir)
		if err != nil {
			log.Error("Get block error: %v", err)
			continue
		}

		components = append(components, block)
	}

	return components, nil
}

// Component get the component
func (tmpl *Template) Component(id string) (core.IComponent, error) {

	path := filepath.Join(tmpl.Root, "__components", id)
	if exist, _ := tmpl.local.fs.Exists(path); !exist {
		return nil, fmt.Errorf("Block %s not found", id)
	}

	component, err := tmpl.getComponentFrom(path)
	if err != nil {
		return nil, err
	}

	err = component.Load()
	if err != nil {
		return nil, err
	}

	_, err = component.Compile()
	if err != nil {
		return nil, err
	}

	return component, nil
}

// Load get the component from the storage
func (component *Component) Load() error {

	root := filepath.Join(component.tmpl.Root, "__components")

	// Type script is the default language
	tsFile := filepath.Join(root, component.Codes.TS.File)
	if exist, _ := component.tmpl.local.fs.Exists(tsFile); exist {
		tsCode, err := component.tmpl.local.fs.ReadFile(tsFile)
		if err != nil {
			return err
		}
		component.Codes.TS.Code = string(tsCode)

	} else {
		jsFile := filepath.Join(root, component.Codes.JS.File)
		jsCode, err := component.tmpl.local.fs.ReadFile(jsFile)
		if err != nil {
			return err
		}
		component.Codes.JS.Code = string(jsCode)
	}

	htmlFile := filepath.Join(root, component.Codes.HTML.File)
	if exist, _ := component.tmpl.local.fs.Exists(htmlFile); exist {
		htmlCode, err := component.tmpl.local.fs.ReadFile(htmlFile)
		if err != nil {
			return err
		}
		component.Codes.HTML.Code = string(htmlCode)
	}

	return nil
}

func (tmpl *Template) getComponentFrom(path string) (core.IComponent, error) {
	id := tmpl.getComponentID(path)
	return tmpl.getComponent(id)
}

func (tmpl *Template) getComponent(id string) (core.IComponent, error) {

	path := filepath.Join(tmpl.Root, "__components", id)
	if !tmpl.local.fs.IsDir(path) {
		return nil, fmt.Errorf("Component %s not found", id)
	}

	jsFile := filepath.Join("/", id, fmt.Sprintf("%s.js", id))
	tsFile := filepath.Join("/", id, fmt.Sprintf("%s.ts", id))
	htmlFile := filepath.Join("/", id, fmt.Sprintf("%s.html", id))
	component := &Component{
		tmpl: tmpl,
		Component: &core.Component{
			ID: id,
			Codes: core.SourceCodes{
				HTML: core.Source{File: htmlFile},
				JS:   core.Source{File: jsFile},
				TS:   core.Source{File: tsFile},
			},
		},
	}
	return component, nil
}

func (tmpl *Template) getComponentID(path string) string {
	return filepath.Base(path)
}
