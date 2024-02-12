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
	ctx := &Context{Pipe: pipe, id: uuid.NewString()}
	ctx.current = 0
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

// Exec and return error
func (ctx *Context) Exec(args ...any) (any, error) {
	fmt.Printf("name: %v\n", ctx.Name)
	fmt.Printf("global: %v\n", ctx.global)
	fmt.Printf("sid: %v\n", ctx.sid)
	fmt.Printf("whitelist: %v\n", ctx.Whitelist)
	return nil, nil
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
