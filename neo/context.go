package neo

import (
	"context"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// NewContext create a new context
func NewContext(sid, cid, payload string) Context {
	ctx := Context{Context: context.Background(), Sid: sid, ChatID: cid}
	if payload == "" {
		return ctx
	}

	err := jsoniter.Unmarshal([]byte(payload), &ctx)
	if err != nil {
		log.Error("%s", err.Error())
	}
	return ctx
}

// NewContextWithCancel create a new context with cancel
func NewContextWithCancel(sid, cid, payload string) (Context, context.CancelFunc) {
	ctx := NewContext(sid, cid, payload)
	return ContextWithCancel(ctx)
}

// NewContextWithTimeout create a new context with timeout
func NewContextWithTimeout(sid, cid, payload string, timeout time.Duration) (Context, context.CancelFunc) {
	ctx := NewContext(sid, cid, payload)
	return ContextWithTimeout(ctx, timeout)
}

// ContextWithCancel create a new context
func ContextWithCancel(parent Context) (Context, context.CancelFunc) {
	new, cancel := context.WithCancel(parent.Context)
	parent.Context = new
	return parent, cancel
}

// ContextWithTimeout create a new context
func ContextWithTimeout(parent Context, timeout time.Duration) (Context, context.CancelFunc) {
	new, cancel := context.WithTimeout(parent.Context, timeout)
	parent.Context = new
	return parent, cancel
}
