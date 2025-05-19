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

	Locale string `json:"locale,omitempty"` // Locale
	Theme  string `json:"theme,omitempty"`  // Theme

	Silent         bool        `json:"silent,omitempty"`          // Silent mode
	ClientType     string      `json:"client_type,omitempty"`     // The request client type. SDK, Desktop, Web, JSSDK, etc. the default is Web
	HistoryVisible bool        `json:"history_visible,omitempty"` // History visible, default is true, if false, the history will not be displayed in the UI
	Retry          bool        `json:"retry,omitempty"`           // Retry mode
	RetryTimes     uint8       `json:"retry_times,omitempty"`     // Retry times
	Upload         *FileUpload `json:"upload,omitempty"`          // Will be removed in the future

	Vision bool `json:"vision,omitempty"` // Vision support
	Search bool `json:"search,omitempty"` // Search support
	RAG    bool `json:"rag,omitempty"`    // RAG support

	Args        []interface{} `json:"args,omitempty"` // Arguments for call
	SharedSpace plan.Space    `json:"-"`              // Shared space
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

const (

	// ClientTypeAgent is the client type for Agent
	ClientTypeAgent = "agent"

	// ClientTypeWeb is the client type for Web (Default UI)
	ClientTypeWeb = "web"

	// ClientTypeAndroid is the client type for Android
	ClientTypeAndroid = "android"

	// ClientTypeIOS is the client type for IOS
	ClientTypeIOS = "ios"

	// ClientTypeJSSDK is the client type for JSSDK
	ClientTypeJSSDK = "jssdk"

	// ClientTypeMacOS is the client type for MacOS Desktop
	ClientTypeMacOS = "macos"

	// ClientTypeWindows is the client type for Windows Desktop
	ClientTypeWindows = "windows"

	// ClientTypeLinux is the client type for Linux Desktop
	ClientTypeLinux = "linux"
)

// SupportedClientTypes is the supported client types
var SupportedClientTypes = map[string]bool{
	ClientTypeAgent:   true,
	ClientTypeWeb:     true,
	ClientTypeAndroid: true,
	ClientTypeIOS:     true,
	ClientTypeJSSDK:   true,
	ClientTypeMacOS:   true,
	ClientTypeWindows: true,
	ClientTypeLinux:   true,
}

// New create a new context
func New(sid, cid, payload string) Context {

	// Validate the client type
	ctx := Context{
		Context:        context.Background(),
		SharedSpace:    plan.NewMemorySharedSpace(),
		Sid:            sid,
		ChatID:         cid,
		HistoryVisible: true,
		ClientType:     ClientTypeWeb,
		Silent:         false,
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
func NewWithCancel(sid, cid, payload string) (Context, context.CancelFunc) {
	ctx := New(sid, cid, payload)
	return WithCancel(ctx)
}

// WithAssistantID set the assistant ID
func WithAssistantID(ctx Context, assistantID string) Context {
	ctx.AssistantID = assistantID
	return ctx
}

// WithSilent set the silent mode
func WithSilent(ctx Context, silent bool) Context {
	ctx.Silent = silent
	return ctx
}

// WithClientType set the client type
func WithClientType(ctx Context, clientType string) Context {
	// Validate the client type
	if !SupportedClientTypes[clientType] {
		log.Error("[Neo] Invalid client type: %s", clientType)
		return ctx
	}
	ctx.ClientType = clientType
	return ctx
}

// WithHistoryVisible set the history visible
func WithHistoryVisible(ctx Context, historyVisible bool) Context {
	ctx.HistoryVisible = historyVisible
	return ctx
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
		"sid":    ctx.Sid,
		"rag":    ctx.RAG,
		"vision": ctx.Vision,
		"search": ctx.Search,
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

	// History visible
	data["history_visible"] = ctx.HistoryVisible

	// Client type
	data["client_type"] = ctx.ClientType

	// Retry mode
	if ctx.Retry {
		data["retry"] = ctx.Retry
	}

	// Arguments for call
	if ctx.Args != nil && len(ctx.Args) > 0 {
		data["args"] = ctx.Args
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

	// Locale
	data["locale"] = ctx.Locale

	// Theme
	data["theme"] = ctx.Theme

	return data
}
