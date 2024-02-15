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
	Output    any       `json:"output,omitempty"`    // the pipe output expression
	Input     Input     `json:"input,omitempty"`     // the pipe input expression
	Whitelist Whitelist `json:"whitelist,omitempty"` // the process whitelist
	Goto      string    `json:"goto,omitempty"`      // goto node name / EOF

	parent    *Pipe            // the parent pipe
	namespace string           // the namespace of the pipe
	mapping   map[string]*Node // the mapping of the nodes Key:name Value:index
}

// Context the Context
type Context struct {
	*Pipe
	id     string
	parent *Context // the parent context id

	context context.Context
	global  map[string]interface{} // $global
	sid     string                 // $sid
	current *Node                  // current position

	in      map[*Node][]any    // $in the current node input value
	out     map[*Node]any      // $out the current node output value
	history map[*Node][]Prompt // history of prompts, this is for the AI and auto merge to the prompts of the node

	input  []any // $input the pipe input value
	output any   // $output the pipe output value
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
	Model    string           `json:"model,omitempty"`    // AI model name (optional)
	Options  map[string]any   `json:"options,omitempty"`  // AI or Request options (optional)
	Request  *Request         `json:"request,omitempty"`  // Http Request
	UI       string           `json:"ui,omitempty"`       // The User Interface cli, web, app, wxapp ...
	AutoFill *AutoFill        `json:"autofill,omitempty"` // Autofill the user input with the expression
	Switch   map[string]*Pipe `json:"case,omitempty"`     // Switch
	Input    Input            `json:"input,omitempty"`    // the node input expression
	Output   any              `json:"output,omitempty"`   // the node output expression
	Goto     string           `json:"goto,omitempty"`     // goto node name / EOF

	index int // the index of the node
}

// Whitelist the Whitelist
type Whitelist map[string]bool

// Input the input
type Input []any

// Args the args
type Args []any

// Data data for the template
type Data map[string]interface{}

// ResumeContext the resume context
type ResumeContext struct {
	ID    string `json:"__id"`
	Type  string `json:"__type"`
	UI    string `json:"__ui"`
	Input Input  `json:"input"`
	Node  *Node  `json:"node"`
	Data  Data   `json:"data"`
}

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
	Content string `json:"content,omitempty"`
}

// Process the switch
type Process struct {
	Name string `json:"name"`
	Args Args   `json:"args,omitempty"`
}

// Request the request
type Request struct{}

// ChatCompletionChunk the chat completion chunk
type ChatCompletionChunk struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	Created           int64       `json:"created"`
	Model             string      `json:"model"`
	SystemFingerprint interface{} `json:"system_fingerprint"`
	Choices           []struct {
		Index        int         `json:"index"`
		Delta        DeltaStruct `json:"delta"`
		Logprobs     interface{} `json:"logprobs"`
		FinishReason interface{} `json:"finish_reason"`
	} `json:"choices"`
}

// DeltaStruct the delta struct
type DeltaStruct struct {
	Content string `json:"content"`
}
