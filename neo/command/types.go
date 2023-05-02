package command

import (
	"context"

	"github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/neo/command/driver"
)

// Request the command request
type Request struct {
	id       string
	sid      string
	ctx      Context
	messages []map[string]interface{}
	*Command
}

// Command the command struct
type Command struct {
	ID          string   `json:"-" yaml:"-"`
	Name        string   `json:"name,omitempty"`
	Connector   string   `json:"connector"`
	Process     string   `json:"process"`
	Prepare     Prepare  `json:"prepare"`
	Description string   `json:"description,omitempty"`
	Optional    Optional `json:"optional,omitempty"`
	Args        []Arg    `json:"args,omitempty"`
	Stack       string   `json:"stack,omitempty"` // query stack
	Path        string   `json:"path,omitempty"`  // query path
	AI          aigc.AI  `json:"-" yaml:"-"`
}

// Arg the argument
type Arg struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required,omitempty"`
}

// Prepare the prepare struct
type Prepare struct {
	Before  string                 `json:"before,omitempty"`
	After   string                 `json:"after,omitempty"`
	Prompts []Prompt               `json:"prompts"`
	Option  map[string]interface{} `json:"option"`
}

// Prompt a prompt
type Prompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Optional optional
type Optional struct {
	Autopilot   bool `json:"autopilot,omitempty"`
	Confirm     bool `json:"confirm,omitempty"`
	MaxAttempts int  `json:"maxAttempts,omitempty"` // default 10
}

// Context the context
type Context struct {
	Sid             string `json:"-" yaml:"-"`
	Stack           string `json:"stack,omitempty"`
	Path            string `json:"path,omitempty"`
	context.Context `json:"-" yaml:"-"`
}

// Store the command driver
type Store interface {
	Match(query driver.Query, content string) (string, error)
	Set(id string, cmd driver.Command) error
	Get(id string) (driver.Command, bool)
	Del(id string)
	SetRequest(sid, id, cid string) error
	GetRequest(sid string) (string, string, bool)
	DelRequest(sid string)
}
