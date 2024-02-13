package pipe

import (
	"context"
)

// Pipe the pipe
type Pipe struct {
	ID        string
	Name      string    `json:"name"`
	Nodes     []Node    `json:"nodes"`
	Label     string    `json:"label,omitempty"`
	Hooks     *Hooks    `json:"hooks,omitempty"`
	Output    any       `json:"output,omitempty"`
	Input     Input     `json:"input,omitempty"`
	Whitelist Whitelist `json:"whitelist,omitempty"` // the process whitelist
	Goto      string    `json:"goto,omitempty"`      // goto node name / EOF

	namespace string           // the namespace of the pipe
	mapping   map[string]*Node // the mapping of the nodes Key:namespace.name Value:index
}

// Context the Context
type Context struct {
	*Pipe
	id      string
	context context.Context
	global  map[string]interface{} // $global
	sid     string                 // $sid
	current string                 // current position
	in      map[string][]any       // $in the node input key:namespace.name Value:[]
	out     map[string]any         // $out the node output key:namespace.name Value:any
	input   map[string][]any       // $input the pipe input key:namespace.name Value:[]
	output  map[string]any         // $output the pipe output key:namespace.name Value:any
}

// Hooks the Hooks
type Hooks struct {
	Progress string `json:"progress,omitempty"`
}

// Node the pip node
type Node struct {
	Name     string           `json:"name"`
	Type     string           `json:"type,omitempty"`     // user-input, ai, process, switch, request
	Label    string           `json:"label,omitempty"`    // Display
	Process  *Process         `json:"process,omitempty"`  // Yao Process
	Prompts  []Prompt         `json:"prompts,omitempty"`  // AI prompts
	Request  *Request         `json:"request,omitempty"`  // Http Request
	UI       string           `json:"ui,omitempty"`       // The User Interface cli, web, app, wxapp ...
	AutoFill *AutoFill        `json:"autofill,omitempty"` // Autofill the user input with the expression
	Case     map[string]*Pipe `json:"case,omitempty"`     // Switch
	Input    Input            `json:"input,omitempty"`    //
	Output   any              `json:"output,omitempty"`   //
	Goto     string           `json:"goto,omitempty"`     // goto node name / EOF

	index     []int    // the index of the node
	namespace string   // the namespace of the node
	history   []Prompt // history of prompts, this is for the AI and auto merge to the prompts
}

// Whitelist the Whitelist
type Whitelist map[string]bool

// Input the input
type Input []any

// Args the args
type Args []any

// Data data for the template
type Data map[string]interface{}

// AutoFill the autofill
type AutoFill struct {
	Value  any    `json:"value"`
	Action string `json:"action,omitempty"`
}

// Case the switch case section
type Case struct {
	Input  Input  `json:"input,omitempty"`  // $in
	Output any    `json:"output,omitempty"` // $out
	Nodes  []Node `json:"nodes,omitempty"`  // $out
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
