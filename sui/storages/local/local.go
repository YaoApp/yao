package local

import (
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/log"
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
	}

	dataFS, err := fs.Get("system")
	if err != nil {
		return nil, err
	}

	dsl.Public = &sui.Public{
		Host:  host,
		Root:  root,
		Index: index,
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

		tmpl, err := local.NewTemplate(dir)
		if err != nil {
			log.Error("GetTemplates %s error: %s", dir, err.Error())
			continue
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// GetTemplate get the template
func (local *Local) GetTemplate(name string) (sui.ITemplate, error) {
	return nil, nil
}

// UploadTemplate upload the template
func (local *Local) UploadTemplate(src string, dst string) (sui.ITemplate, error) {
	return nil, nil
}
