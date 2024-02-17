package pipe

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/kun/exception"
)

var contexts = sync.Map{}

// Create create new context
func (pipe *Pipe) Create() *Context {
	id := uuid.NewString()
	ctx := &Context{
		id:      id,
		Pipe:    pipe,
		in:      map[*Node][]any{},
		out:     map[*Node]any{},
		history: map[*Node][]Prompt{},
		current: nil,

		input:  []any{},
		output: nil,
	}

	// Set the current node
	if pipe.HasNodes() {
		ctx.current = &pipe.Nodes[0]
	}

	contexts.Store(id, ctx)
	return ctx
}

// Open the context
func Open(id string) (*Context, error) {
	ctx, ok := contexts.Load(id)
	if !ok {
		return nil, fmt.Errorf("context %s not found", id)
	}
	return ctx.(*Context), nil
}

// Close the context
func Close(id string) {
	contexts.Delete(id)
}

// Resume the context by id
func (ctx *Context) Resume(id string, args ...any) any {
	v, err := ctx.resume(args...)
	if err != nil {
		exception.New("pipe: %s %s", 500, ctx.Name, err).Throw()
	}
	return v
}

// resume the context by id
func (ctx *Context) resume(args ...any) (any, error) {
	if ctx.current == nil {
		return nil, ctx.Errorf("pipe %s has no nodes", ctx.Name)
	}

	node := ctx.current
	output, err := ctx.parseNodeOutput(node, args)
	if err != nil {
		return nil, node.Errorf(ctx, err.Error())
	}

	// Next node
	next, eof, err := ctx.next()
	if err != nil {
		return nil, err
	}

	// End of the pipe
	if eof {
		defer Close(ctx.id)
		output, err := ctx.parseOutput()
		if err != nil {
			return nil, err
		}
		return output, nil
	}

	return ctx.exec(next, anyToInput(output))
}

// Run the pipe
func (ctx *Context) Run(args ...any) any {
	v, err := ctx.Exec(args...)
	if err != nil {
		exception.New("pipe: %s %s", 500, ctx.Name, err).Throw()
	}
	return v
}

// Exec this is the entry point of the pipe
func (ctx *Context) Exec(args ...any) (any, error) {
	if ctx.current == nil {
		return nil, ctx.Errorf("pipe %s has no nodes", ctx.Name)
	}

	input, err := ctx.parseInput(args)
	if err != nil {
		return nil, err
	}

	return ctx.exec(ctx.current, input)
}

// Exec and return error
func (ctx *Context) exec(node *Node, input Input) (output any, err error) {

	var out any
	switch node.Type {

	case "process":
		out, err = node.YaoProcess(ctx, input)
		if err != nil {
			return nil, err
		}

	// case "request":
	// 	err := node.ExecRequest(ctx, args)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	case "ai":
		out, err = node.AI(ctx, input)
		if err != nil {
			return nil, err
		}

	case "switch":
		out, err = node.Case(ctx, input)
		if err != nil {
			return nil, err
		}

	case "user-input":
		var pause bool = false
		out, pause, err = node.Render(ctx, input)
		if err != nil {
			return nil, err
		}

		// Pause the pipe waiting for user input
		if pause {
			return out, nil
		}

	default:
		return nil, node.Errorf(ctx, "type '%s' not support", node.Type)
	}

	// Execute the next node
	next, eof, err := ctx.next()
	if err != nil {
		return nil, err
	}

	// End of the pipe
	if eof {
		defer Close(ctx.id)
		output, err := ctx.parseOutput()
		if err != nil {
			return nil, err
		}

		return output, nil
	}

	// Execute the next node
	return ctx.exec(next, anyToInput(out))
}

// Next the next node
func (ctx *Context) next() (*Node, bool, error) {

	if ctx.current == nil {
		return nil, true, nil
	}

	// if the goto is not empty, then goto the node
	if ctx.current.Goto != "" {
		data := ctx.data(ctx.current)
		next, err := data.replaceString(ctx.current.Goto)
		if err != nil {
			return nil, false, err
		}

		if next == "EOF" {
			return nil, true, nil
		}

		var has = false
		ctx.current, has = ctx.mapping[next]
		if !has {
			return nil, false, ctx.Errorf("node %s not found", next)
		}
		return ctx.current, false, nil
	}

	// continue to the next node
	next := ctx.current.index + 1
	if next >= len(ctx.Nodes) {
		return nil, true, nil
	}

	ctx.current = &ctx.Nodes[next]
	return ctx.current, false, nil
}

// ParseNodeInput parse the node input
func (ctx *Context) parseNodeInput(node *Node, input Input) (Input, error) {
	ctx.in[node] = input
	if node.Input != nil && len(node.Input) > 0 {
		data := ctx.data(node)
		input, err := data.replaceArray(node.Input)
		if err != nil {
			return nil, err
		}
		ctx.in[node] = input
		return input, nil
	}

	return input, nil
}

// ParseNodeOutput parse the node output
func (ctx *Context) parseNodeOutput(node *Node, output any) (any, error) {
	ctx.out[node] = output
	if node.Output != nil {
		data := ctx.data(node)
		output, err := data.replace(node.Output)
		if err != nil {
			return nil, err
		}
		ctx.out[node] = output
		return output, nil
	}

	return output, nil
}

// ParseInput parse the pipe input
func (ctx *Context) parseInput(input Input) (Input, error) {
	ctx.input = input
	if ctx.Input != nil && len(ctx.Input) > 0 {
		data := ctx.data(nil)
		input, err := data.replaceArray(ctx.Input)
		if err != nil {
			return nil, err
		}
		ctx.input = input
		return input, nil
	}
	return input, nil
}

// ParseOutput parse the pipe output
func (ctx *Context) parseOutput() (any, error) {

	if ctx.Output != nil {
		data := ctx.data(nil)
		output, err := data.replace(ctx.Output)
		if err != nil {
			return nil, err
		}
		ctx.output = output
		return output, nil
	}

	if ctx.current != nil {
		return ctx.out[ctx.current], nil
	}

	return nil, nil
}

func (ctx *Context) data(node *Node) Data {

	data := map[string]any{
		"$sid":    ctx.sid,
		"$global": ctx.global,
		"$input":  ctx.input,
		"$output": ctx.output,
	}

	if ctx.in != nil {
		for k, v := range ctx.in {
			key := fmt.Sprintf("$node.%s.in", k.Name)
			data[key] = v
		}
	}

	if ctx.out != nil {
		for k, v := range ctx.out {
			data[k.Name] = v
		}
	}

	if node != nil {
		data["$in"] = ctx.in[node]
		data["$out"] = ctx.out[node]
	}

	return data
}

// With with the context
func (ctx *Context) With(context context.Context) *Context {
	ctx.context = context
	return ctx
}

// WithGlobal with the global data
func (ctx *Context) WithGlobal(data map[string]interface{}) *Context {
	if data != nil {
		if ctx.global == nil {
			ctx.global = map[string]interface{}{}
		}
		for k, v := range data {
			ctx.global[k] = v
		}
	}
	return ctx
}

// WithSid with the sid
func (ctx *Context) WithSid(sid string) *Context {
	ctx.sid = sid
	return ctx
}

func (ctx *Context) inheritance(parent *Context) *Context {
	ctx.in = parent.in
	ctx.out = parent.out
	ctx.history = parent.history
	ctx.global = parent.global
	ctx.sid = parent.sid
	ctx.parent = parent
	return ctx
}

// Errorf format the error message
func (ctx *Context) Errorf(format string, a ...any) error {
	message := fmt.Sprintf(format, a...)
	return fmt.Errorf("pipe: %s(%s) %s %s", ctx.Name, ctx.Pipe.ID, ctx.id, message)
}
