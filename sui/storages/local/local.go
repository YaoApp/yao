package local

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
	sui "github.com/yaoapp/yao/sui/core"
)

// New create a new local sui
func New(dsl *sui.DSL) (*Local, error) {

	templateRoot := "/data/sui/templates"
	if dsl.Storage.Option != nil && dsl.Storage.Option["root"] != nil {
		templateRoot = dsl.Storage.Option["root"].(string)
	}

	root := "/"
	host := "/"
	index := "/index"
	matcher := ""
	if dsl.Public != nil {
		if dsl.Public.Root != "" {
			root = dsl.Public.Root
		}

		if dsl.Public.Host != "" {
			host = dsl.Public.Host
		}

		if dsl.Public.Index != "" {
			index = dsl.Public.Index
		}

		if dsl.Public.Matcher != "" {
			matcher = dsl.Public.Matcher
		}
	}

	dataFS, err := fs.Get("system")
	if err != nil {
		return nil, err
	}

	dsl.Public = &sui.Public{
		Host:    host,
		Root:    root,
		Index:   index,
		Matcher: matcher,
	}

	return &Local{
		root: templateRoot,
		fs:   dataFS,
		DSL:  dsl,
	}, nil
}

// GetTemplates get the templates
func (local *Local) GetTemplates() ([]sui.ITemplate, error) {

	templates := []sui.ITemplate{}
	dirs, err := local.fs.ReadDir(local.root, false)
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if !local.fs.IsDir(dir) {
			continue
		}

		tmpl, err := local.getTemplateFrom(dir)
		if err != nil {
			log.Error("GetTemplates %s error: %s", dir, err.Error())
			continue
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// GetTemplate get the template
func (local *Local) GetTemplate(id string) (sui.ITemplate, error) {
	path := path.Join(local.root, id)
	return local.getTemplate(id, path)
}

// UploadTemplate upload the template
func (local *Local) UploadTemplate(src string, dst string) (sui.ITemplate, error) {
	return nil, nil
}

// GetTemplateFrom get the template from the path
func (local *Local) getTemplateFrom(path string) (*Template, error) {
	id := local.getTemplateID(path)
	return local.getTemplate(id, path)
}

// getTemplate get the template
func (local *Local) getTemplate(id string, path string) (*Template, error) {

	if !local.fs.IsDir(path) {
		return nil, fmt.Errorf("Template %s not found", id)
	}

	tmpl := Template{
		local: local,
		Root:  path,
		Template: &core.Template{
			ID:          id,
			Name:        strings.ToUpper(id),
			Version:     1,
			Screenshots: []string{},
			Themes:      []core.SelectOption{},
		}}

	// load the template.json
	configFile := filepath.Join(path, "template.json")
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

	// load the __document.html
	documentFile := filepath.Join(path, "__document.html")
	if local.fs.IsFile(documentFile) {
		documentBytes, err := local.fs.ReadFile(documentFile)
		if err != nil {
			return nil, err
		}
		tmpl.Document = documentBytes
	}

	// load the __data.json
	dataFile := filepath.Join(path, "__data.json")
	if local.fs.IsFile(dataFile) {
		dataBytes, err := local.fs.ReadFile(dataFile)
		if err != nil {
			return nil, err
		}
		tmpl.GlobalData = dataBytes
	}

	// load the __build.backend.ts / __build.backend.js
	err := tmpl.loadBuildScript()
	if err != nil {
		return nil, err
	}

	return &tmpl, nil
}

// GetTemplateID get the template ID
func (local *Local) getTemplateID(path string) string {
	return filepath.Base(path)
}

// AppRoot get the app root
func (local *Local) AppRoot() string {
	approot := application.App.Root()
	localroot := local.fs.Root()
	return strings.TrimPrefix(localroot, approot)
}
