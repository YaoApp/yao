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
		id:     id,
		Pipe:   pipe,
		in:     map[string][]any{},
		out:    map[string]any{},
		input:  map[string][]any{},
		output: map[string]any{},
	}

	if pipe.Nodes != nil {
		ctx.current = pipe.Nodes[0].Namespace()
	}
	contexts.Store(id, ctx)
	return ctx
}

// Open the context
func Open(id string) *Context {
	ctx, ok := contexts.Load(id)
	if !ok {
		exception.New("pipe: %s not found", 404, id).Throw()
	}
	return ctx.(*Context)
}

// Close the context
func Close(id string) {
	contexts.Delete(id)
}

// Run the pipe
func (ctx *Context) Run(args ...any) any {
	v, err := ctx.Exec(args...)
	if err != nil {
		exception.New("pipe: %s %s", 500, ctx.Name, err).Throw()
	}
	return v
}

// ID the context id
func (ctx *Context) ID() string {
	return ctx.id
}

// Current the current node
func (ctx *Context) Current() (*Node, error) {
	node, has := ctx.mapping[ctx.current]
	if !has {
		return nil, fmt.Errorf("pipe: %s %s", ctx.Name, "node not found")
	}
	return node, nil
}

// Next the next node
func (ctx *Context) Next() (*Node, error) {
	node, err := ctx.Current()
	if err != nil {
		return nil, err
	}

	if node.Goto != "" {
		next, err := ctx.replaceString(node.Goto)
		if err != nil {
			return nil, err
		}

		if next == "EOF" {
			return nil, fmt.Errorf("EOF")
		}

		ctx.current = next
		return ctx.Current()
	}

	next := node.index[len(node.index)-1] + 1
	if next < len(ctx.Nodes) {
		ctx.current = ctx.Nodes[next].Namespace()
		return ctx.Current()
	}
	return nil, fmt.Errorf("EOF")
}

// IsEOF check if the error is EOF
func IsEOF(err error) bool {
	return err != nil && err.Error() == "EOF"
}

// Exec the pipe
func (ctx *Context) Exec(args ...any) (any, error) {
	node, err := ctx.Current()
	if err != nil {
		return nil, err
	}
	return ctx.exec(node, args...)
}

// Exec and return error
func (ctx *Context) exec(node *Node, args ...any) (any, error) {

	switch node.Type {

	case "process":
		err := node.ExecProcess(ctx, args)
		if err != nil {
			return nil, err
		}

	case "request":
		err := node.ExecRequest(ctx, args)
		if err != nil {
			return nil, err
		}

	case "ai":
		err := node.ExecAI(ctx, args)
		if err != nil {
			return nil, err
		}

	case "switch":
		err := node.ExecSwitch(ctx, args)
		if err != nil {
			return nil, err
		}

	case "user-input":
		err := node.Render(ctx, args)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("pipe: %s %s", ctx.Name, "node type error")
	}

	return nil, nil
}

func (ctx *Context) replace(value any) (any, error) {

	switch v := value.(type) {
	case string:
		return ctx.replaceAny(v)

	case []any:
		return ctx.replaceArray(v)

	case map[string]any:
		return ctx.replaceMap(v)

	case Input:
		return ctx.replaceArray(v)
	}

	return value, nil
}

func (ctx *Context) replaceAny(value string) (any, error) {

	if !IsExpression(value) {
		return value, nil
	}

	data, err := ctx.data()
	if err != nil {
		return "", err
	}

	v, err := data.Exec(value)
	if err != nil {
		return "", err
	}
	return v, nil
}

// replaceString replace the string
func (ctx *Context) replaceString(value string) (string, error) {

	if !IsExpression(value) {
		return value, nil
	}

	data, err := ctx.data()
	if err != nil {
		return "", err
	}

	v, err := data.ExecString(value)
	if err != nil {
		return "", err
	}
	return v, nil
}

func (ctx *Context) replaceMap(value map[string]any) (map[string]any, error) {
	newValue := map[string]any{}
	for k, v := range value {
		res, err := ctx.replace(v)
		if err != nil {
			return nil, err
		}
		newValue[k] = res
	}
	return newValue, nil
}

func (ctx *Context) replaceArray(value []any) ([]any, error) {
	newValue := []any{}
	for _, v := range value {
		res, err := ctx.replace(v)
		if err != nil {
			return nil, err
		}
		newValue = append(newValue, res)
	}

	return newValue, nil
}

func (ctx *Context) replaceInput(value Input) (Input, error) {
	return ctx.replaceArray(value)
}

func (ctx *Context) data() (Data, error) {
	node, err := ctx.Current()
	if err != nil {
		return Data{}, err
	}

	name := node.Namespace()
	data := map[string]any{
		"$sid":    ctx.sid,
		"$global": ctx.global,
		"$in":     ctx.in[name],
		"$out":    ctx.out[name],
		"$input":  ctx.input,
		"$output": ctx.output,
	}

	if ctx.output != nil {
		for k, v := range ctx.output {
			data[k] = v
		}
	}
	return data, nil

}

// With with the context
func (ctx *Context) With(context context.Context) *Context {
	ctx.context = context
	return ctx
}

// WithGlobal with the global data
func (ctx *Context) WithGlobal(data map[string]interface{}) *Context {
	ctx.global = data
	return ctx
}

// WithSid with the sid
func (ctx *Context) WithSid(sid string) *Context {
	ctx.sid = sid
	return ctx
}
