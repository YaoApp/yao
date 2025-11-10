package context

import (
	"context"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// New create a new context
func New(parent context.Context, authorized *types.AuthorizedInfo, chatID, payload string) Context {

	if parent == nil {
		parent = context.Background()
	}

	// Validate the client type
	ctx := Context{
		Context: parent,
		Space:   plan.NewMemorySharedSpace(),
		ChatID:  chatID,
	}

	if payload == "" {
		return ctx
	}

	err := jsoniter.Unmarshal([]byte(payload), &ctx)
	if err != nil {
		log.Error("%s", err.Error())
	}

	return ctx
}

// NewWithCancel create a new context with cancel
func NewWithCancel(parent context.Context, authorized *types.AuthorizedInfo, chatID, payload string) (Context, context.CancelFunc) {
	ctx := New(parent, authorized, chatID, payload)
	return WithCancel(ctx)
}

// NewWithTimeout create a new context with timeout
func NewWithTimeout(parent context.Context, authorized *types.AuthorizedInfo, chatID, payload string, timeout time.Duration) (Context, context.CancelFunc) {
	ctx := New(parent, authorized, chatID, payload)
	return WithTimeout(ctx, timeout)
}

// WithCancel create a new context
func WithCancel(parent Context) (Context, context.CancelFunc) {
	new, cancel := context.WithCancel(parent.Context)
	parent.Context = new
	return parent, cancel
}

// WithTimeout create a new context
func WithTimeout(parent Context, timeout time.Duration) (Context, context.CancelFunc) {
	new, cancel := context.WithTimeout(parent.Context, timeout)
	parent.Context = new
	return parent, cancel
}

// Release the context
func (ctx *Context) Release() {
	ctx.Space.Clear()
	ctx.Space = nil
	ctx = nil
}

// Map the context to a map
func (ctx *Context) Map() map[string]interface{} {
	data := map[string]interface{}{}

	// Authorized information
	if ctx.ChatID != "" {
		data["chat_id"] = ctx.ChatID
	}
	if ctx.AssistantID != "" {
		data["assistant_id"] = ctx.AssistantID
	}

	// Arguments for call
	if len(ctx.Args) > 0 {
		data["args"] = ctx.Args
	}
	if ctx.Retry {
		data["retry"] = ctx.Retry
	}
	if ctx.RetryTimes > 0 {
		data["retry_times"] = ctx.RetryTimes
	}

	// Locale information
	if ctx.Locale != "" {
		data["locale"] = ctx.Locale
	}
	if ctx.Theme != "" {
		data["theme"] = ctx.Theme
	}

	// Request information
	if ctx.Client.Type != "" || ctx.Client.UserAgent != "" || ctx.Client.IP != "" {
		data["client"] = map[string]interface{}{
			"type":       ctx.Client.Type,
			"user_agent": ctx.Client.UserAgent,
			"ip":         ctx.Client.IP,
		}
	}
	if ctx.Referer != "" {
		data["referer"] = ctx.Referer
	}
	if ctx.Accept != "" {
		data["accept"] = ctx.Accept
	}

	// CUI Context information
	if ctx.Route != "" {
		data["route"] = ctx.Route
	}
	if len(ctx.Data) > 0 {
		data["data"] = ctx.Data
	}

	return data
}

// GenChatID generate a new chat ID
func GenChatID() string {
	return fmt.Sprintf("chat_%d", time.Now().UnixNano())
}
