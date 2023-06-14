package widget

import (
	"sync"

	"github.com/yaoapp/gou/api"
)

// DSL is the widget DSL
type DSL struct {
	ID          string            `json:"-"`
	File        string            `json:"-"`
	Instances   sync.Map          `json:"-"`
	FS          FS                `json:"-"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Path        string            `json:"path,omitempty"`
	Extensions  []string          `json:"extensions,omitempty"`
	Remote      *RemoteDSL        `json:"remote,omitempty"`
	Loader      LoaderDSL         `json:"loader"`
	Process     map[string]string `json:"process,omitempty"`
	API         *api.HTTP         `json:"api,omitempty"`
}

// RemoteDSL is the remote widget DSL
type RemoteDSL struct {
	Connector string `json:"connector,omitempty"`
	Table     string `json:"table,omitempty"`
	Reload    bool   `json:"reload,omitempty"`
}

// LoaderDSL is the loader widget DSL
type LoaderDSL struct {
	Load   string `json:"load,omitempty"`
	Reload string `json:"reload,omitempty"`
	Unload string `json:"unload,omitempty"`
}

// Instance is the widget instance
type Instance struct {
	source map[string]interface{}
	dsl    interface{}
	loader LoaderDSL
	id     string
	widget string
}

// FS is the DSL File system
type FS interface {
	Walk(cb func(id string, source map[string]interface{})) error
	Save(file string, source map[string]interface{}) error
	Remove(file string) error
}
