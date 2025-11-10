package context

import (
	"context"

	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Accept the accept of the request, it will be used to identify the accept of the request.
type Accept string

// Referer the referer of the request, it will be used to identify the referer of the request.
type Referer string

// Client represents the client information from HTTP request
type Client struct {
	Type      string `json:"type,omitempty"`       // Client type: web, android, ios, windows, macos, linux, agent, jssdk
	UserAgent string `json:"user_agent,omitempty"` // Original User-Agent header
	IP        string `json:"ip,omitempty"`         // Client IP address
}

const (
	// AcceptStandard standard response format compatible with OpenAI API and general chat UIs (default)
	AcceptStandard = "standard"

	// AcceptWebCUI web-based CUI format with action request support for Yao Chat User Interface
	AcceptWebCUI = "cui-web"

	// AccepNativeCUI native mobile/tablet CUI format with action request support
	AccepNativeCUI = "cui-native"

	// AcceptDesktopCUI desktop CUI format with action request support
	AcceptDesktopCUI = "cui-desktop"
)

// ValidAccepts is the map of valid accept types
var ValidAccepts = map[string]bool{
	AcceptStandard:   true,
	AcceptWebCUI:     true,
	AccepNativeCUI:   true,
	AcceptDesktopCUI: true,
}

const (
	// RefererAPI request from HTTP API endpoint
	RefererAPI = "api"

	// RefererProcess request from Yao Process call
	RefererProcess = "process"

	// RefererMCP request from MCP (Model Context Protocol) server
	RefererMCP = "mcp"

	// RefererJSSDK request from JavaScript SDK
	RefererJSSDK = "jssdk"

	// RefererAgent request from agent-to-agent recursive call (assistant calling another assistant)
	RefererAgent = "agent"

	// RefererTool request from tool/function execution
	RefererTool = "tool"

	// RefererHook request from hook trigger (on_message, on_error, etc.)
	RefererHook = "hook"

	// RefererSchedule request from scheduled task or cron job
	RefererSchedule = "schedule"

	// RefererScript request from custom script execution
	RefererScript = "script"

	// RefererInternal request from internal system call
	RefererInternal = "internal"
)

// ValidReferers is the map of valid referer types
var ValidReferers = map[string]bool{
	RefererAPI:      true,
	RefererProcess:  true,
	RefererMCP:      true,
	RefererJSSDK:    true,
	RefererAgent:    true,
	RefererTool:     true,
	RefererHook:     true,
	RefererSchedule: true,
	RefererScript:   true,
	RefererInternal: true,
}

// Context the context
type Context struct {

	// Context
	context.Context
	Space plan.Space `json:"-"` // Shared data space, it will be used to share data between the request and the call

	// Authorized information
	Authorized  *types.AuthorizedInfo `json:"authorized,omitempty"`   // Authorized information
	ChatID      string                `json:"chat_id,omitempty"`      // Chat ID, use to select chat
	AssistantID string                `json:"assistant_id,omitempty"` // Assistant ID, use to select assistant
	Sid         string                `json:"sid" yaml:"-"`           // Session ID (Deprecated, use Authorized instead)

	// Arguments for call
	Args       []interface{} `json:"args,omitempty"`        // Arguments for call, it will be used to pass data to the call
	Retry      bool          `json:"retry,omitempty"`       // Retry mode
	RetryTimes uint8         `json:"retry_times,omitempty"` // Retry times

	// Locale information
	Locale string `json:"locale,omitempty"` // Locale
	Theme  string `json:"theme,omitempty"`  // Theme

	// Request information
	Client  Client `json:"client,omitempty"`  // Client information from HTTP request
	Referer string `json:"referer,omitempty"` // Request source: api, process, mcp, jssdk, agent, tool, hook, schedule, script, internal
	Accept  Accept `json:"accept,omitempty"`  // Response format: standard, cui-web, cui-native, cui-desktop

	// CUI Context information
	Route string                 `json:"route,omitempty"` // The route of the request, it will be used to identify the route of the request
	Data  map[string]interface{} `json:"data,omitempty"`  // The data of the request, it will be used to pass data to the page

	Silent bool `json:"silent,omitempty"` // Silent mode (Deprecated, use Referer instead)
}
