package context

import (
	"context"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/kun/log"
)

// Context the context
type Context struct {
	context.Context
	Sid         string                 `json:"sid" yaml:"-"`           // Session ID
	ChatID      string                 `json:"chat_id,omitempty"`      // Chat ID, use to select chat
	AssistantID string                 `json:"assistant_id,omitempty"` // Assistant ID, use to select assistant
	Stack       string                 `json:"stack,omitempty"`        // will be removed in the future
	Path        string                 `json:"pathname,omitempty"`     // wiil be rename to path
	FormData    map[string]interface{} `json:"formdata,omitempty"`
	Field       *Field                 `json:"field,omitempty"`
	Namespace   string                 `json:"namespace,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Signal      interface{}            `json:"signal,omitempty"`
	Silent      bool                   `json:"silent,omitempty"`      // Silent mode
	Retry       bool                   `json:"retry,omitempty"`       // Retry mode
	RetryTimes  uint8                  `json:"retry_times,omitempty"` // Retry times
	Upload      *FileUpload            `json:"upload,omitempty"`
	Version     bool                   `json:"version,omitempty"` // Version support
	RAG         bool                   `json:"rag,omitempty"`     // RAG support
	SharedSpace plan.Space             `json:"-"`                 // Shared space
}

// Field the context field
type Field struct {
	Name     string                 `json:"name,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Bind     string                 `json:"bind,omitempty"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Children []interface{}          `json:"children,omitempty"`
}

// FileUpload the file upload
type FileUpload struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Size     int64  `json:"size,omitempty"`
	TempFile string `json:"temp_file,omitempty"`
}

// New create a new context
func New(sid, cid, payload string) Context {

	ctx := Context{Context: context.Background(), Sid: sid, ChatID: cid, SharedSpace: plan.NewMemorySharedSpace()}
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
func NewWithCancel(sid, cid, payload string) (Context, context.CancelFunc) {
	ctx := New(sid, cid, payload)
	return WithCancel(ctx)
}

// NewWithTimeout create a new context with timeout
func NewWithTimeout(sid, cid, payload string, timeout time.Duration) (Context, context.CancelFunc) {
	ctx := New(sid, cid, payload)
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
	ctx.SharedSpace.Clear()
	ctx.SharedSpace = nil
	ctx = nil
}

// Map the context to a map
func (ctx *Context) Map() map[string]interface{} {
	data := map[string]interface{}{
		"sid":     ctx.Sid,
		"rag":     ctx.RAG,
		"version": ctx.Version,
	}

	if ctx.ChatID != "" {
		data["chat_id"] = ctx.ChatID
	}
	if ctx.AssistantID != "" {
		data["assistant_id"] = ctx.AssistantID
	}
	if ctx.Stack != "" {
		data["stack"] = ctx.Stack
	}

	// Silent mode
	if ctx.Silent {
		data["silent"] = ctx.Silent
	}

	// Retry mode
	if ctx.Retry {
		data["retry"] = ctx.Retry
	}

	// Retry times
	data["retry_times"] = ctx.RetryTimes

	if ctx.Path != "" {
		data["pathname"] = ctx.Path
	}
	if len(ctx.FormData) > 0 {
		data["formdata"] = ctx.FormData
	}
	if ctx.Field != nil {
		data["field"] = ctx.Field
	}
	if ctx.Namespace != "" {
		data["namespace"] = ctx.Namespace
	}
	if len(ctx.Config) > 0 {
		data["config"] = ctx.Config
	}
	if ctx.Signal != nil {
		data["signal"] = ctx.Signal
	}
	if ctx.Upload != nil {
		data["upload"] = ctx.Upload
	}

	return data
}
