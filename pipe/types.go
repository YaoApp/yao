package pipe

import "context"

// Pipe the pipe
type Pipe struct {
	ID        string
	Name      string    `json:"name"`
	Nodes     []Node    `json:"nodes"`
	Label     string    `json:"label,omitempty"`
	Hooks     *Hooks    `json:"hooks,omitempty"`
	Output    any       `json:"output,omitempty"`    // $output
	Input     Input     `json:"input,omitempty"`     // $input
	Whitelist Whitelist `json:"whitelist,omitempty"` // the process whitelist

}

// Context the Context
type Context struct {
	*Pipe
	id      string
	context context.Context
	global  map[string]interface{} // $global
	sid     string                 // $sid
	current int                    // current position
}

// Hooks the Hooks
type Hooks struct {
	Progress string `json:"progress,omitempty"`
}

// Node the pip node
type Node struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type,omitempty"`      // user-input, ai, process, switch, request
	Label     string                 `json:"label,omitempty"`     // Display
	Process   *Process               `json:"process,omitempty"`   // Yao Process
	Prompts   []Prompt               `json:"prompts,omitempty"`   // AI prompts
	Request   *Request               `json:"request,omitempty"`   // Http Request
	Interface string                 `json:"interface,omitempty"` // User Interface command-line, web, app, wxapp ...
	Case      map[string]CaseSection `json:"case,omitempty"`      // Switch
	Input     Input                  `json:"input,omitempty"`     // $in
	Output    any                    `json:"output,omitempty"`    // $out
}

// Whitelist the Whitelist
type Whitelist map[string]bool

// Input the input
type Input []any

// Args the args
type Args []any

// CaseSection the switch case section
type CaseSection struct {
	Input  Input  `json:"input,omitempty"`  // $in
	Output any    `json:"output,omitempty"` // $out
	Nodes  []Node `json:"nodes,omitempty"`  // $out
	Goto   string `json:"goto,omitempty"`   // goto node name / EOF
}

// Prompt the switch
type Prompt struct {
	Role    string `json:"role,omitempty"`
	Message string `json:"message,omitempty"`
}

// Process the switch
type Process struct {
	Name string `json:"name"`
	Args Args   `json:"args,omitempty"`
}

// Request the request
type Request struct{}
