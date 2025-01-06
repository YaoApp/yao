package context

import (
	"context"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// Context the context
type Context struct {
	context.Context
	Sid         string                 `json:"sid" yaml:"-"`           // Session ID
	ChatID      string                 `json:"chat_id,omitempty"`      // Chat ID, use to select chat
	AssistantID string                 `json:"assistant_id,omitempty"` // Assistant ID, use to select assistant
	Stack       string                 `json:"stack,omitempty"`
	Path        string                 `json:"pathname,omitempty"`
	FormData    map[string]interface{} `json:"formdata,omitempty"`
	Field       *Field                 `json:"field,omitempty"`
	Namespace   string                 `json:"namespace,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Signal      interface{}            `json:"signal,omitempty"`
	Upload      *FileUpload            `json:"upload,omitempty"`
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
