package local

import (
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/sui/core"
)

// Local is the struct for the local sui
type Local struct {
	root string
	fs   fs.FileSystem
	*core.DSL
}

// Template is the struct for the local sui template
type Template struct {
	Root    string `json:"-"`
	local   *Local
	locales []core.SelectOption
	loaded  map[string]core.IPage
	*core.Template
}

// Page is the struct for the local sui page
type Page struct {
	tmpl *Template
	*core.Page
}

// Block is the struct for the local sui block
type Block struct {
	tmpl *Template
	*core.Block
}

// Component is the struct for the local sui component
type Component struct {
	tmpl *Template
	*core.Component
}
