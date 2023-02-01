package action

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/widgets/hook"
)

// Process action.search ...
type Process struct {
	Name        string        `json:"-"`
	Process     string        `json:"process,omitempty"`
	ProcessBind string        `json:"bind,omitempty"`
	Guard       string        `json:"guard,omitempty"`
	Default     []interface{} `json:"default,omitempty"`
	Disable     bool          `json:"disable,omitempty"`
	Before      *hook.Before  `json:"-"`
	After       *hook.After   `json:"-"`
	Handler     Handler       `json:"-"`
}

// Handler action handler
type Handler func(p *Process, process *process.Process) (interface{}, error)
