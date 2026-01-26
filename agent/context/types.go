package context

import (
	"context"
	"sync"
	"time"

	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/agent/memory"
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/openapi/oauth/types"
	traceTypes "github.com/yaoapp/yao/trace/types"
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

	// RefererAgent request from agent-to-agent delegate call (same context, saves history)
	RefererAgent = "agent"

	// RefererAgentFork request from agent-to-agent fork call (ctx.agent.Call/All/Any/Race, skips history)
	RefererAgentFork = "agent_fork"

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
	RefererAPI:       true,
	RefererProcess:   true,
	RefererMCP:       true,
	RefererJSSDK:     true,
	RefererAgent:     true,
	RefererAgentFork: true,
	RefererTool:      true,
	RefererHook:      true,
	RefererSchedule:  true,
	RefererScript:    true,
	RefererInternal:  true,
}

const (
	// StackStatusPending stack is created but not started yet
	StackStatusPending = "pending"

	// StackStatusRunning stack is currently executing
	StackStatusRunning = "running"

	// StackStatusCompleted stack completed successfully
	StackStatusCompleted = "completed"

	// StackStatusFailed stack failed with error
	StackStatusFailed = "failed"

	// StackStatusTimeout stack execution timeout
	StackStatusTimeout = "timeout"
)

// ValidStackStatus is the map of valid stack status types
var ValidStackStatus = map[string]bool{
	StackStatusPending:   true,
	StackStatusRunning:   true,
	StackStatusCompleted: true,
	StackStatusFailed:    true,
	StackStatusTimeout:   true,
}

// Interrupt Types and Constants
// ===============================

// InterruptType represents the type of interrupt
type InterruptType string

const (
	// InterruptGraceful waits for current step to complete before handling interrupt
	InterruptGraceful InterruptType = "graceful"

	// InterruptForce immediately cancels current operation and handles interrupt
	InterruptForce InterruptType = "force"
)

// InterruptAction represents the action to take after interrupt is handled
type InterruptAction string

const (
	// InterruptActionContinue appends new messages and continues execution
	InterruptActionContinue InterruptAction = "continue"

	// InterruptActionRestart restarts execution with only new messages
	InterruptActionRestart InterruptAction = "restart"

	// InterruptActionAbort terminates the request
	InterruptActionAbort InterruptAction = "abort"
)

// InterruptSignal represents an interrupt signal with new messages from user
type InterruptSignal struct {
	Type      InterruptType          `json:"type"`               // Interrupt type: graceful or force
	Messages  []Message              `json:"messages"`           // User's new messages (can be multiple)
	Timestamp int64                  `json:"timestamp"`          // Interrupt timestamp in milliseconds
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}

// InterruptHandler is the function signature for handling interrupts
// This handler is registered in the InterruptController and called when interrupt signal is received
// Parameters:
//   - ctx: The context being interrupted
//   - signal: The interrupt signal (contains Type and Messages)
//
// Returns:
//   - error: Error if interrupt handling failed
type InterruptHandler func(ctx *Context, signal *InterruptSignal) error

// InterruptController manages interrupt handling for a context
// All interrupt-related fields are encapsulated in this type
type InterruptController struct {
	queue           chan *InterruptSignal `json:"-"` // Queue to receive interrupt signals
	current         *InterruptSignal      `json:"-"` // Current interrupt being processed
	pending         []*InterruptSignal    `json:"-"` // Pending interrupts in queue
	mutex           sync.RWMutex          `json:"-"` // Protects current and pending
	ctx             context.Context       `json:"-"` // Interrupt control context (independent from HTTP context)
	cancel          context.CancelFunc    `json:"-"` // Cancel function for force interrupt
	listenerStarted bool                  `json:"-"` // Whether listener goroutine is started
	handler         InterruptHandler      `json:"-"` // Handler to process interrupt signals
	contextID       string                `json:"-"` // Context ID to retrieve the parent context
}

// AssistantInfo represents the assistant information structure
type AssistantInfo struct {
	ID          string `json:"assistant_id"`          // Assistant ID
	Type        string `json:"type,omitempty"`        // Assistant Type, default is assistant
	Name        string `json:"name,omitempty"`        // Assistant Name
	Avatar      string `json:"avatar,omitempty"`      // Assistant Avatar
	Description string `json:"description,omitempty"` // Assistant Description
}

// Skip configuration for what to skip in this request
type Skip struct {
	History        bool `json:"history"`         // Skip saving chat history (for internal calls like title/prompt generation)
	Trace          bool `json:"trace"`           // Skip trace logging
	Output         bool `json:"output"`          // Skip output to client (for internal A2A calls that only need response data)
	Keyword        bool `json:"keyword"`         // Skip keyword extraction for web search (use raw query directly)
	Search         bool `json:"search"`          // Skip auto search (for internal calls like needsearch intent detection)
	ContentParsing bool `json:"content_parsing"` // Skip content parsing (vision, PDF, docx, etc.), convert files to raw text directly
}

// MessageMetadata stores metadata for sent messages
// Used to inherit BlockID and ThreadID in delta operations
type MessageMetadata struct {
	MessageID  string    // Message ID
	BlockID    string    // Block ID
	ThreadID   string    // Thread ID
	Type       string    // Message type (text, thinking, etc.)
	StartTime  time.Time // Message start time (for calculating duration)
	ChunkCount int       // Number of chunks sent for this message
}

// BlockMetadata stores metadata for output blocks
type BlockMetadata struct {
	BlockID      string    // Block ID
	Type         string    // Block type (llm, mcp, agent, etc.)
	StartTime    time.Time // Block start time
	MessageCount int       // Number of messages in this block
}

// Context the context
type Context struct {

	// Context
	context.Context

	// External
	ID          string               `json:"id"` // Context ID for external interrupt identification
	Memory      *memory.Memory       `json:"-"`  // Agent memory with four spaces: User, Team, Chat, Context
	Cache       store.Store          `json:"-"`  // Cache store, it will be used to store the message cache, default is "__yao.agent.cache"
	Stack       *Stack               `json:"-"`  // Stack, current active stack of the request
	Stacks      map[string]*Stack    `json:"-"`  // Stacks, all stacks in this request (for trace logging)
	Writer      Writer               `json:"-"`  // Writer, it will be used to write response data to the client
	IDGenerator *message.IDGenerator `json:"-"`  // ID generator for this context (chunk, message, block, thread IDs)
	Logger      *RequestLogger       `json:"-"`  // Request-scoped async logger

	// ForkParent stores parent stack info for forked contexts (set by Fork())
	// This allows EnterStack to create a child stack instead of root stack
	// without sharing the actual Stack reference (which would cause race conditions)
	ForkParent *ForkParentInfo `json:"-"`

	// Chat buffer for batch saving messages and resume steps
	Buffer *ChatBuffer `json:"-"` // Chat buffer for batch saving at end of Stream()

	// Internal
	trace           traceTypes.Manager    `json:"-"` // Trace manager, lazy initialized on first access
	messageMetadata *messageMetadataStore `json:"-"` // Thread-safe message metadata store for delta operations

	// Model capabilities (set by assistant, used by output adapters)
	Capabilities *openai.Capabilities `json:"-"` // Model capabilities for the current connector

	// Interrupt control (all interrupt-related logic is encapsulated in InterruptController)
	Interrupt *InterruptController `json:"-"` // Interrupt controller for handling user interrupts during streaming

	// Authorized information
	Authorized  *types.AuthorizedInfo `json:"authorized,omitempty"`   // Authorized information
	ChatID      string                `json:"chat_id,omitempty"`      // Chat ID, use to select chat
	AssistantID string                `json:"assistant_id,omitempty"` // Assistant ID, use to select assistant

	// Locale information
	Locale string `json:"locale,omitempty"` // Locale
	Theme  string `json:"theme,omitempty"`  // Theme

	// Request information
	Client  Client `json:"client,omitempty"`  // Client information from HTTP request
	Referer string `json:"referer,omitempty"` // Request source: api, process, mcp, jssdk, agent, tool, hook, schedule, script, internal
	Accept  Accept `json:"accept,omitempty"`  // Response format: standard, cui-web, cui-native, cui-desktop

	// CUI Context information
	Route    string                 `json:"route,omitempty"`    // The route of the request, it will be used to identify the route of the request
	Metadata map[string]interface{} `json:"metadata,omitempty"` // The metadata of the request, it will be used to pass data to the page
}

// SearchIntent represents the result of search intent detection
// Used by Create hook to specify fine-grained search behavior
type SearchIntent struct {
	NeedSearch  bool     `json:"need_search"`            // Whether search is needed
	SearchTypes []string `json:"search_types,omitempty"` // Types of search to perform: "web", "kb", "db"
	Confidence  float64  `json:"confidence,omitempty"`   // Confidence level (0-1)
	Reason      string   `json:"reason,omitempty"`       // Reason for the decision
}

// Options represents the options for the context
type Options struct {

	// Original context, override the default context
	Context context.Context `json:"-"` // Context, it will be used to pass the context to the call

	// Writer, use to write response data to the client (override the default writer)
	Writer Writer `json:"writer,omitempty"` // Writer, use to write response data to the client

	// Skip configuration (history, trace, etc.), nil means don't skip anything
	Skip *Skip `json:"skip,omitempty"` // Skip configuration (history, trace, etc.), nil means don't skip anything

	// Connector, use to select the connector of the LLM Model, Default is Assistant.Connector
	Connector string `json:"connector,omitempty"` // Connector, use to select the connector of the LLM Model, Default is Assistant.Connector

	// Disable global prompts, default is false
	DisableGlobalPrompts bool `json:"disable_global_prompts,omitempty"` // Temporarily disable global prompts for this request

	// Search controls search behavior, supports multiple types:
	// - bool: true = enable all search types, false = disable all search
	// - SearchIntent: fine-grained control with specific types, confidence, etc.
	// - nil: use default behavior (determined by __yao.needsearch agent)
	Search any `json:"search,omitempty"` // Search mode: bool | SearchIntent | nil

	// Agent mode, use to select the mode of the request, default is "chat"
	Mode string `json:"mode,omitempty"` // Agent mode, use to select the mode of the request, default is "chat"

	// Uses configuration, allow hook to override wrapper configurations for vision, audio, search, and fetch
	Uses *Uses `json:"uses,omitempty"` // Uses configuration, allow hook to override wrapper configurations for vision, audio, search, and fetch

	// Metadata for passing custom data to hooks (e.g., scenario selection)
	Metadata map[string]any `json:"metadata,omitempty"` // Custom metadata passed to Create/Next hooks

	// OnMessage is called for each message sent via ctx.Send()
	// Used by ctx.agent.Call with onChunk callback to receive SSE messages
	// Returns: 0 = continue, non-zero = stop
	OnMessage OnMessageFunc `json:"-"`
}

// ForceA2A sets the options for Agent-to-Agent (A2A) calls.
// For A2A calls:
// - Output is NOT skipped - sub-agents output normally with ThreadID
// - History IS skipped - A2A messages should not be saved to chat history
// If Skip is nil, it creates a new Skip instance.
func (opts *Options) ForceA2A() {
	if opts.Skip == nil {
		opts.Skip = &Skip{}
	}
	opts.Skip.History = true
	// Note: skip.output is NOT set - sub-agents output normally with ThreadID
}

// OnMessageFunc is a callback function for receiving output messages
// Called for each message sent via ctx.Send() - same as SSE messages to client
// Returns: 0 = continue, non-zero = stop sending
type OnMessageFunc func(msg *message.Message) int

// ForkParentInfo stores parent stack information for forked contexts
// This is used by EnterStack to create a child stack with proper inheritance
// without sharing the actual Stack reference (which would cause race conditions in parallel calls)
type ForkParentInfo struct {
	StackID string   // Parent stack ID (used as ParentID for child stack)
	TraceID string   // Parent trace ID (inherited by child stack)
	Depth   int      // Parent depth (child depth = parent depth + 1)
	Path    []string // Parent path (child path = parent path + child ID)
}

// Stack represents the call stack node for tracing agent-to-agent calls
// Uses a flat structure to avoid circular references and memory overhead
type Stack struct {
	// Identity
	ID      string `json:"id"`       // Unique stack node ID, used to identify this specific call
	TraceID string `json:"trace_id"` // Shared trace ID for entire call tree, inherited from root

	// Options
	Options *Options `json:"options,omitempty"` // Options for the call

	// Call context
	AssistantID string `json:"assistant_id"`      // Assistant handling this call
	Referer     string `json:"referer,omitempty"` // Call source: api, agent, tool, process, etc.
	Depth       int    `json:"depth"`             // Call depth in the tree (0=root)

	// Relationships
	ParentID string   `json:"parent_id,omitempty"` // Parent stack ID (empty for root call)
	Path     []string `json:"path"`                // Full path from root: [root_id, parent_id, ..., this_id]

	// Tracking
	CreatedAt   int64  `json:"created_at"`             // Unix timestamp in milliseconds
	CompletedAt *int64 `json:"completed_at,omitempty"` // Unix timestamp when completed (nil if ongoing)
	Status      string `json:"status"`                 // Status: pending, running, completed, failed, timeout
	Error       string `json:"error,omitempty"`        // Error message if failed

	// Metrics
	DurationMs *int64 `json:"duration_ms,omitempty"` // Duration in milliseconds (calculated when completed)

	// Runtime cache (not serialized)
	output *output.Output `json:"-"` // Cached output instance for this stack
}

// Response the response
// 100% compatible with the OpenAI API
type Response struct {
	RequestID   string              `json:"request_id"`           // Request ID for the response
	ContextID   string              `json:"context_id"`           // Context ID for the response
	TraceID     string              `json:"trace_id"`             // Trace ID for the response
	ChatID      string              `json:"chat_id"`              // Chat ID for the response
	AssistantID string              `json:"assistant_id"`         // Assistant ID for the response
	Create      *HookCreateResponse `json:"create,omitempty"`     // Create response from the create hook
	Next        interface{}         `json:"next,omitempty"`       // Next response from the next hook
	Completion  *CompletionResponse `json:"completion,omitempty"` // Completion response from the completion hook
	Tools       []ToolCallResponse  `json:"tools,omitempty"`      // Tool call results (if any tools were executed)
}

// HookCreateResponse the response of the create hook
type HookCreateResponse struct {

	// Messages to be sent to the assistant
	Messages []Message `json:"messages,omitempty"`

	// Audio configuration (for models that support audio output)
	Audio *AudioConfig `json:"audio,omitempty"`

	// Generation parameters
	Temperature         *float64 `json:"temperature,omitempty"`
	MaxTokens           *int     `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int     `json:"max_completion_tokens,omitempty"`

	// MCP configuration - allow hook to add/override MCP servers for this request
	MCPServers []MCPServerConfig `json:"mcp_servers,omitempty"`

	// Prompt configuration
	PromptPreset         string `json:"prompt_preset,omitempty"`          // Select prompt preset (e.g., "chat.friendly", "task.analysis")
	DisableGlobalPrompts *bool  `json:"disable_global_prompts,omitempty"` // Temporarily disable global prompts for this request

	// Context adjustments - allow hook to modify context fields
	Connector string                 `json:"connector,omitempty"` // Override connector (call-level)
	Locale    string                 `json:"locale,omitempty"`    // Override locale (session-level)
	Theme     string                 `json:"theme,omitempty"`     // Override theme (session-level)
	Route     string                 `json:"route,omitempty"`     // Override route (session-level)
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // Override or merge metadata (session-level)

	// Uses configuration - allow hook to override wrapper configurations
	Uses *Uses `json:"uses,omitempty"` // Override wrapper configurations for vision, audio, search, and fetch

	// ForceUses controls whether to force using Uses tools even when model has native capabilities
	ForceUses *bool `json:"force_uses,omitempty"` // Force using Uses tools regardless of model capabilities

	// Search controls search behavior, supports multiple types:
	// - bool: true = enable all search types, false = disable all search
	// - SearchIntent: fine-grained control with specific types, confidence, etc.
	// - nil: use default behavior (determined by __yao.needsearch agent)
	Search any `json:"search,omitempty"` // Search mode: bool | SearchIntent | nil

	// Delegate: if provided, delegate to another agent immediately (skip LLM call)
	// This allows Create hook to route to sub-agents before any LLM processing
	Delegate *DelegateConfig `json:"delegate,omitempty"`
}

// NextHookPayload payload for the next hook
type NextHookPayload struct {
	Messages   []Message           `json:"messages,omitempty"`   // Messages to be sent to the assistant
	Completion *CompletionResponse `json:"completion,omitempty"` // Completion response from the completion hook
	Tools      []ToolCallResponse  `json:"tools,omitempty"`      // Tools results from the assistant
	Error      string              `json:"error,omitempty"`      // Error message if failed
}

// ToolCallResponse the response of a tool call
type ToolCallResponse struct {
	ToolCallID string      `json:"toolcall_id"`
	Server     string      `json:"server"`
	Tool       string      `json:"tool"`
	Arguments  interface{} `json:"arguments,omitempty"`
	Result     interface{} `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// NextHookResponse represents the response from Next hook
type NextHookResponse struct {
	// Delegate: if provided, delegate to another agent (recursive call)
	Delegate *DelegateConfig `json:"delegate,omitempty"`

	// Data: custom response data to return to user
	// If both Delegate and Data are nil, use standard CompletionResponse
	Data interface{} `json:"data,omitempty"`

	// Metadata: for debugging and logging
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DelegateConfig configuration for delegating to another agent
type DelegateConfig struct {
	AgentID  string                 `json:"agent_id"`          // Required: target agent ID
	Messages []Message              `json:"messages"`          // Messages to send to target agent
	Options  map[string]interface{} `json:"options,omitempty"` // Optional: call-level options for delegation
}

// NextAction defines the action determined by Next hook response
type NextAction string

const (
	// NextActionReturn returns data to user (standard or custom)
	NextActionReturn NextAction = "return"

	// NextActionDelegate delegates to another agent
	NextActionDelegate NextAction = "delegate"
)

// Action returns the determined action based on NextHookResponse fields
func (n *NextHookResponse) Action() NextAction {
	if n.Delegate != nil {
		return NextActionDelegate
	}
	return NextActionReturn
}

// ResponseHookNext the response of the next hook
type ResponseHookNext interface{}

// ResponseHookMCP the response of the mcp hook
type ResponseHookMCP struct{}

// ResponseHookFailback the response of the failback hook
type ResponseHookFailback struct{}

// HookInterruptedResponse the response of the interrupted hook
type HookInterruptedResponse struct {
	// Action to take after interrupt is handled
	Action InterruptAction `json:"action"` // continue, restart, or abort

	// Messages to use for next execution (if action is continue or restart)
	Messages []Message `json:"messages,omitempty"`

	// Context adjustments - allow hook to modify context fields
	AssistantID string                 `json:"assistant_id,omitempty"` // Override assistant ID
	Connector   string                 `json:"connector,omitempty"`    // Override connector
	Locale      string                 `json:"locale,omitempty"`       // Override locale
	Theme       string                 `json:"theme,omitempty"`        // Override theme
	Route       string                 `json:"route,omitempty"`        // Override route
	Metadata    map[string]interface{} `json:"metadata,omitempty"`     // Override or merge metadata

	// Notice to send to client
	Notice string `json:"notice,omitempty"` // Message to display to user (e.g., "Processing your new question...")
}

// Message Structure ( OpenAI Chat Completion Input Message Structure, https://platform.openai.com/docs/api-reference/chat/create#chat/create-messages )
// ===============================

// MessageRole represents the role of a message author
type MessageRole string

// Message role constants
const (
	RoleDeveloper MessageRole = "developer" // Developer-provided instructions (o1 models and newer)
	RoleSystem    MessageRole = "system"    // System instructions
	RoleUser      MessageRole = "user"      // User messages
	RoleAssistant MessageRole = "assistant" // Assistant responses
	RoleTool      MessageRole = "tool"      // Tool responses
)

// Message represents a message in the conversation, compatible with OpenAI's chat completion API
// Supports message types: developer, system, user, assistant, and tool
type Message struct {
	// Common fields for all message types
	Role    MessageRole `json:"role"`              // Required: message author role
	Content interface{} `json:"content,omitempty"` // string or array of ContentPart; Required for most types, optional for assistant with tool_calls
	Name    *string     `json:"name,omitempty"`    // Optional: participant name to differentiate between participants of the same role

	// Tool message specific fields
	ToolCallID *string `json:"tool_call_id,omitempty"` // Required for tool messages: tool call that this message is responding to

	// Assistant message specific fields
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // Optional for assistant: tool calls generated by the model
	Refusal   *string    `json:"refusal,omitempty"`    // Optional for assistant: refusal message (null when not refusing)
}

// ContentPartType represents the type of content part
type ContentPartType string

// Content part type constants
const (
	ContentText       ContentPartType = "text"        // Text content
	ContentImageURL   ContentPartType = "image_url"   // Image URL content (Vision)
	ContentInputAudio ContentPartType = "input_audio" // Input audio content (Audio)
	ContentFile       ContentPartType = "file"        // File attachment (documents, etc.)
	ContentData       ContentPartType = "data"        // Generic data content (base64, binary, etc.)
)

// ContentPart represents a part of the message content (for multimodal messages)
// Used when Content is an array instead of a simple string
type ContentPart struct {
	Type       ContentPartType `json:"type"`                  // Required: content part type
	Text       string          `json:"text,omitempty"`        // For type="text": the text content
	ImageURL   *ImageURL       `json:"image_url,omitempty"`   // For type="image_url": the image URL
	InputAudio *InputAudio     `json:"input_audio,omitempty"` // For type="input_audio": the input audio data
	File       *FileAttachment `json:"file,omitempty"`        // For type="file": file attachment
	Data       *DataContent    `json:"data,omitempty"`        // For type="data": generic data content
}

// ImageDetailLevel represents the detail level for image processing
type ImageDetailLevel string

// Image detail level constants
const (
	DetailAuto ImageDetailLevel = "auto" // Let the model decide
	DetailLow  ImageDetailLevel = "low"  // Low detail (faster, cheaper)
	DetailHigh ImageDetailLevel = "high" // High detail (slower, more expensive)
)

// ImageURL represents an image URL in the message content
type ImageURL struct {
	URL    string           `json:"url"`              // Required: URL of the image or base64 encoded image data
	Detail ImageDetailLevel `json:"detail,omitempty"` // Optional: how the model processes the image
}

// InputAudio represents input audio data in the message content
type InputAudio struct {
	Data   string `json:"data"`   // Required: Base64 encoded audio data
	Format string `json:"format"` // Required: Audio format (e.g., "wav", "mp3")
}

// FileAttachment represents a file attachment in the message content
// Compatible with frontend InputArea format: { type: 'file', file: { url, filename } }
type FileAttachment struct {
	URL      string `json:"url"`                // Required: URL of the file (http:// or __uploader://fileid wrapper)
	Filename string `json:"filename,omitempty"` // Optional: original filename
}

// DataSourceType represents the type of data source
type DataSourceType string

// Data source type constants
const (
	DataSourceModel        DataSourceType = "model"         // Data model
	DataSourceKBCollection DataSourceType = "kb_collection" // Knowledge base collection
	DataSourceKBDocument   DataSourceType = "kb_document"   // Knowledge base document/file
	DataSourceTable        DataSourceType = "table"         // Database table
	DataSourceAPI          DataSourceType = "api"           // API endpoint
	DataSourceMCPResource  DataSourceType = "mcp_resource"  // MCP (Model Context Protocol) resource
)

// DataSource represents a single data source reference
type DataSource struct {
	Type     DataSourceType         `json:"type"`               // Required: type of data source
	Name     string                 `json:"name"`               // Required: name/identifier of the data source
	ID       string                 `json:"id,omitempty"`       // Optional: specific ID (e.g., document ID, record ID)
	Filters  map[string]interface{} `json:"filters,omitempty"`  // Optional: filters to apply
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Optional: additional metadata
}

// DataContent represents data source references in the message
// Used to reference data models, knowledge base collections, KB documents, etc.
type DataContent struct {
	Sources []DataSource `json:"sources"` // Required: array of data source references
}

// ToolCallType represents the type of tool call
type ToolCallType string

// Tool call type constants
const (
	ToolTypeFunction ToolCallType = "function" // Function call
)

// ToolCall represents a tool call generated by the model (for assistant messages)
type ToolCall struct {
	ID       string       `json:"id"`       // Required: unique identifier for the tool call
	Type     ToolCallType `json:"type"`     // Required: type of tool call, currently only "function"
	Function Function     `json:"function"` // Required: function call details
}

// Function represents a function call with name and arguments
type Function struct {
	Name      string `json:"name"`                // Required: name of the function to call
	Arguments string `json:"arguments,omitempty"` // Optional: arguments to pass to the function, as a JSON string
}

// Completion Request Structure ( OpenAI Chat Completion Request, https://platform.openai.com/docs/api-reference/chat/create )
// ===============================

// CompletionRequest represents a chat completion request compatible with OpenAI's API
type CompletionRequest struct {
	// Required fields
	Model    string    `json:"model"`    // Required: ID of the model to use
	Messages []Message `json:"messages"` // Required: list of messages comprising the conversation so far

	// Audio configuration (for models that support audio output)
	Audio *AudioConfig `json:"audio,omitempty"` // Optional: audio output configuration

	// Generation parameters
	Temperature         *float64 `json:"temperature,omitempty"`           // Optional: sampling temperature (0-2), defaults to 1
	MaxTokens           *int     `json:"max_tokens,omitempty"`            // Optional: maximum number of tokens to generate (deprecated, use max_completion_tokens)
	MaxCompletionTokens *int     `json:"max_completion_tokens,omitempty"` // Optional: maximum number of tokens that can be generated in the completion

	// Streaming configuration
	Stream        *bool          `json:"stream,omitempty"`         // Optional: if true, stream partial message deltas
	StreamOptions *StreamOptions `json:"stream_options,omitempty"` // Optional: options for streaming response

	// CUI Context information
	Route    string                 `json:"route,omitempty"`    // Optional: route of the request for CUI context
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Optional: metadata to pass to the page for CUI context
	Skip     *Skip                  `json:"skip,omitempty"`     // Optional: skip configuration (history, trace, etc.)
}

// AudioConfig represents the audio output configuration for models that support audio
type AudioConfig struct {
	Voice  string `json:"voice"`  // Required: voice to use for audio output (e.g., "alloy", "echo", "fable", "onyx", "nova", "shimmer")
	Format string `json:"format"` // Required: audio output format (e.g., "wav", "mp3", "flac", "opus", "pcm16")
}

// StreamOptions represents options for streaming responses
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"` // If true, include usage statistics in the final chunk
}

// MCPServerConfig represents an MCP server configuration
// This mirrors agent/store/types.MCPServerConfig to avoid import cycles
type MCPServerConfig struct {
	ServerID  string   `json:"server_id"`           // MCP server ID (required)
	Tools     []string `json:"tools,omitempty"`     // Tool name filter (empty = all tools)
	Resources []string `json:"resources,omitempty"` // Resource URI filter (empty = all resources)
}
