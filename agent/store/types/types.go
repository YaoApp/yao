package types

import (
	"encoding/json"
	"fmt"
	"time"

	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
)

// Setting represents the conversation configuration structure
// Used to configure basic conversation parameters including connector, user field, table name, etc.
type Setting struct {
	Connector string                 `json:"connector,omitempty" yaml:"connector,omitempty"` // Connector name, default is "default"
	MaxSize   int                    `json:"max_size,omitempty" yaml:"max_size,omitempty"`   // Maximum storage size limit, default is 20
	TTL       int                    `json:"ttl,omitempty" yaml:"ttl,omitempty"`             // Time To Live in seconds, default is 90 * 24 * 60 * 60 (90 days)
	Options   map[string]interface{} `json:"optional,omitempty" yaml:"optional,omitempty"`   // The options for the store
}

// =============================================================================
// Chat Types
// =============================================================================

// Chat represents a chat session
type Chat struct {
	ChatID        string                 `json:"chat_id"`
	Title         string                 `json:"title,omitempty"`
	AssistantID   string                 `json:"assistant_id"`
	LastConnector string                 `json:"last_connector,omitempty"` // Last used connector ID (updated on each message)
	LastMode      string                 `json:"last_mode,omitempty"`      // Last used chat mode (updated on each message)
	Status        string                 `json:"status"`                   // "active" or "archived"
	Public        bool                   `json:"public"`                   // Whether shared across all teams
	Share         string                 `json:"share"`                    // "private" or "team"
	Sort          int                    `json:"sort"`                     // Sort order for display
	LastMessageAt *time.Time             `json:"last_message_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`

	// Permission fields (managed by Yao framework when permission: true)
	CreatedBy string `json:"__yao_created_by,omitempty"` // User ID who created the record
	UpdatedBy string `json:"__yao_updated_by,omitempty"` // User ID who last updated
	TeamID    string `json:"__yao_team_id,omitempty"`    // Team ID for team-level access
	TenantID  string `json:"__yao_tenant_id,omitempty"`  // Tenant ID for multi-tenancy
}

// ChatFilter for listing chats
type ChatFilter struct {
	UserID      string `json:"user_id,omitempty"`
	TeamID      string `json:"team_id,omitempty"`
	AssistantID string `json:"assistant_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Keywords    string `json:"keywords,omitempty"`

	// Time range filter
	StartTime *time.Time `json:"start_time,omitempty"` // Filter chats after this time
	EndTime   *time.Time `json:"end_time,omitempty"`   // Filter chats before this time
	TimeField string     `json:"time_field,omitempty"` // Field for time filter: "created_at" or "last_message_at" (default)

	// Sorting
	OrderBy string `json:"order_by,omitempty"` // Field to sort by (default: "last_message_at")
	Order   string `json:"order,omitempty"`    // Sort order: "desc" (default) or "asc"

	// Response format
	GroupBy string `json:"group_by,omitempty"` // "time" for time-based groups, empty for flat list

	// Pagination
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pagesize,omitempty"`

	// Permission filter (not serialized)
	QueryFilter func(query.Query) `json:"-"` // Custom query function for permission filtering
}

// ChatList paginated response with time-based grouping
type ChatList struct {
	Data      []*Chat      `json:"data"`
	Groups    []*ChatGroup `json:"groups,omitempty"` // Time-based groups for UI display
	Page      int          `json:"page"`
	PageSize  int          `json:"pagesize"`
	PageCount int          `json:"pagecount"`
	Total     int          `json:"total"`
}

// ChatGroup represents a time-based group of chats
type ChatGroup struct {
	Label string  `json:"label"` // "Today", "Yesterday", "This Week", "This Month", "Earlier"
	Key   string  `json:"key"`   // "today", "yesterday", "this_week", "this_month", "earlier"
	Chats []*Chat `json:"chats"` // Chats in this group
	Count int     `json:"count"` // Number of chats in group
}

// =============================================================================
// Message Types
// =============================================================================

// Message represents a chat message
type Message struct {
	MessageID   string                 `json:"message_id"`
	ChatID      string                 `json:"chat_id"`
	RequestID   string                 `json:"request_id,omitempty"`
	Role        string                 `json:"role"` // "user" or "assistant"
	Type        string                 `json:"type"` // "text", "image", "loading", "tool_call", "retrieval", etc.
	Props       map[string]interface{} `json:"props"`
	BlockID     string                 `json:"block_id,omitempty"`
	ThreadID    string                 `json:"thread_id,omitempty"`
	AssistantID string                 `json:"assistant_id,omitempty"`
	Connector   string                 `json:"connector,omitempty"` // Connector ID used for this message
	Mode        string                 `json:"mode,omitempty"`      // Chat mode used for this message (chat or task)
	Sequence    int                    `json:"sequence"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// MessageFilter for listing messages
type MessageFilter struct {
	RequestID string `json:"request_id,omitempty"`
	Role      string `json:"role,omitempty"`
	BlockID   string `json:"block_id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty"`
	Type      string `json:"type,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// =============================================================================
// Resume Types (for recovery from interruption/failure)
// =============================================================================

// Resume represents an execution state for recovery
// Only stored when request is interrupted or failed
type Resume struct {
	ResumeID      string                 `json:"resume_id"`
	ChatID        string                 `json:"chat_id"`
	RequestID     string                 `json:"request_id"`
	AssistantID   string                 `json:"assistant_id"`
	StackID       string                 `json:"stack_id"`
	StackParentID string                 `json:"stack_parent_id,omitempty"`
	StackDepth    int                    `json:"stack_depth"`
	Type          string                 `json:"type"`   // "input", "hook_create", "llm", "tool", "hook_next", "delegate"
	Status        string                 `json:"status"` // "failed" or "interrupted"
	Input         map[string]interface{} `json:"input,omitempty"`
	Output        map[string]interface{} `json:"output,omitempty"`
	SpaceSnapshot map[string]interface{} `json:"space_snapshot,omitempty"` // Shared space data for recovery
	Error         string                 `json:"error,omitempty"`
	Sequence      int                    `json:"sequence"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ResumeStatus constants
const (
	ResumeStatusFailed      = "failed"
	ResumeStatusInterrupted = "interrupted"
)

// ResumeType constants
const (
	ResumeTypeInput      = "input"
	ResumeTypeHookCreate = "hook_create"
	ResumeTypeLLM        = "llm"
	ResumeTypeTool       = "tool"
	ResumeTypeHookNext   = "hook_next"
	ResumeTypeDelegate   = "delegate"
)

// AssistantFilter represents the assistant filter structure
// Used for filtering and pagination when retrieving assistant lists
type AssistantFilter struct {
	Tags         []string          `json:"tags,omitempty"`          // Filter by tags
	Type         string            `json:"type,omitempty"`          // Filter by type
	Keywords     string            `json:"keywords,omitempty"`      // Search in name and description
	Connector    string            `json:"connector,omitempty"`     // Filter by connector
	AssistantID  string            `json:"assistant_id,omitempty"`  // Filter by assistant ID
	AssistantIDs []string          `json:"assistant_ids,omitempty"` // Filter by assistant IDs
	Mentionable  *bool             `json:"mentionable,omitempty"`   // Filter by mentionable status
	Automated    *bool             `json:"automated,omitempty"`     // Filter by automation status
	BuiltIn      *bool             `json:"built_in,omitempty"`      // Filter by built-in status
	Page         int               `json:"page,omitempty"`          // Page number, starting from 1
	PageSize     int               `json:"pagesize,omitempty"`      // Items per page
	Select       []string          `json:"select,omitempty"`        // Fields to return, returns all fields if empty
	QueryFilter  func(query.Query) `json:"-"`                       // Custom query function for permission filtering (not serialized)
}

// AssistantList represents the paginated assistant list response structure
// Used for returning paginated assistant lists with metadata
type AssistantList struct {
	Data      []*AssistantModel `json:"data"`      // List of assistants
	Page      int               `json:"page"`      // Current page number (1-based)
	PageSize  int               `json:"pagesize"`  // Number of items per page
	PageCount int               `json:"pagecount"` // Total number of pages
	Next      int               `json:"next"`      // Next page number (0 if no next page)
	Prev      int               `json:"prev"`      // Previous page number (0 if no previous page)
	Total     int               `json:"total"`     // Total number of items across all pages
}

// AssistantInfo contains basic assistant information for display
// Used in chat history to show assistant details with i18n support
type AssistantInfo struct {
	AssistantID string `json:"assistant_id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar,omitempty"`
	Description string `json:"description,omitempty"`
}

// Tag represents a tag
type Tag struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Prompt a prompt
type Prompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// KBSetting Knowledge Base configuration for agent (from agent/kb.yml)
type KBSetting struct {
	Chat *ChatKBSetting `json:"chat,omitempty" yaml:"chat,omitempty"` // Chat session KB settings
}

// ChatKBSetting represents KB settings for chat sessions
type ChatKBSetting struct {
	EmbeddingProviderID string                                 `json:"embedding_provider_id" yaml:"embedding_provider_id"`             // Embedding provider ID
	EmbeddingOptionID   string                                 `json:"embedding_option_id" yaml:"embedding_option_id"`                 // Embedding option ID
	Locale              string                                 `json:"locale,omitempty" yaml:"locale,omitempty"`                       // Locale for content processing
	Config              *graphragtypes.CreateCollectionOptions `json:"config,omitempty" yaml:"config,omitempty"`                       // Vector index configuration
	Metadata            map[string]interface{}                 `json:"metadata,omitempty" yaml:"metadata,omitempty"`                   // Collection metadata defaults
	DocumentDefaults    *DocumentDefaults                      `json:"document_defaults,omitempty" yaml:"document_defaults,omitempty"` // Document processing defaults
}

// DocumentDefaults represents default settings for document processing
type DocumentDefaults struct {
	Chunking   *ProviderOption `json:"chunking,omitempty" yaml:"chunking,omitempty"`     // Chunking provider configuration
	Extraction *ProviderOption `json:"extraction,omitempty" yaml:"extraction,omitempty"` // Extraction provider configuration
	Converter  *ProviderOption `json:"converter,omitempty" yaml:"converter,omitempty"`   // Converter provider configuration
}

// ProviderOption represents a provider and option ID pair
type ProviderOption struct {
	ProviderID string `json:"provider_id" yaml:"provider_id"` // Provider ID
	OptionID   string `json:"option_id" yaml:"option_id"`     // Option ID within the provider
}

// KnowledgeBase the knowledge base configuration
type KnowledgeBase struct {
	Collections []string               `json:"collections,omitempty"` // Knowledge base collection IDs
	Options     map[string]interface{} `json:"options,omitempty"`     // Additional options for knowledge base
}

// Database the database configuration
type Database struct {
	Models  []string               `json:"models,omitempty"`  // Database models
	Options map[string]interface{} `json:"options,omitempty"` // Additional options for database
}

// MCPServers the MCP servers configuration
// Supports multiple formats in the servers array:
// - Simple string: "server_id"
// - With tools: {"server_id": ["tool1", "tool2"]}
// - With resources and tools: {"server_id": {"resources": [...], "tools": [...]}}
type MCPServers struct {
	Servers []MCPServerConfig      `json:"servers,omitempty"` // MCP server configurations
	Options map[string]interface{} `json:"options,omitempty"` // Additional options for MCP servers
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	ServerID  string   `json:"server_id,omitempty"` // MCP server ID
	Resources []string `json:"resources,omitempty"` // Resources to use (optional)
	Tools     []string `json:"tools,omitempty"`     // Tools to use (optional)
}

// UnmarshalJSON implements custom JSON unmarshaling for MCPServerConfig
// Supports multiple input formats:
// 1. Simple string: "server_id"
// 2. Standard object: {"server_id": "server1", "resources": [...], "tools": [...]}
// 3. Tools array: {"server_id": ["tool1", "tool2"]}
// 4. Full config: {"server_id": {"resources": [...], "tools": [...]}}
func (m *MCPServerConfig) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		m.ServerID = str
		return nil
	}

	// Try to unmarshal as standard object with server_id field
	type Alias MCPServerConfig
	var stdObj Alias
	if err := json.Unmarshal(data, &stdObj); err == nil && stdObj.ServerID != "" {
		*m = MCPServerConfig(stdObj)
		return nil
	}

	// Try to unmarshal as object with single key (alternative formats)
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	// Should have exactly one key (the server ID)
	if len(obj) != 1 {
		return fmt.Errorf("MCPServerConfig object must have exactly one key or server_id field")
	}

	// Get the server ID (the only key)
	for serverID, value := range obj {
		m.ServerID = serverID

		// Try to unmarshal value as array of strings (format c: tools only)
		var tools []string
		if err := json.Unmarshal(value, &tools); err == nil {
			m.Tools = tools
			return nil
		}

		// Try to unmarshal as object with resources and tools (format b)
		var detail struct {
			Resources []string `json:"resources,omitempty"`
			Tools     []string `json:"tools,omitempty"`
		}
		if err := json.Unmarshal(value, &detail); err == nil {
			m.Resources = detail.Resources
			m.Tools = detail.Tools
			return nil
		}

		return fmt.Errorf("invalid format for server '%s'", serverID)
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for MCPServerConfig
// Serializes to different formats based on content:
// 1. If only ServerID: "server_id"
// 2. If has Resources or Tools: {"server_id": "...", "resources": [...], "tools": [...]}
func (m MCPServerConfig) MarshalJSON() ([]byte, error) {
	// If only ServerID, serialize as simple string
	if len(m.Resources) == 0 && len(m.Tools) == 0 {
		return json.Marshal(m.ServerID)
	}

	// Otherwise, use standard object format
	type Alias MCPServerConfig
	return json.Marshal(Alias(m))
}

// Workflow the workflow configuration
type Workflow struct {
	Workflows []string               `json:"workflows,omitempty"` // Workflow IDs
	Options   map[string]interface{} `json:"options,omitempty"`   // Additional workflow options
}

// Tool represents a tool configuration for storage
type Tool struct {
	Type        string                 `json:"type,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ToolCalls the tool calls
type ToolCalls struct {
	Tools   []Tool   `json:"tools,omitempty"`
	Prompts []Prompt `json:"prompts,omitempty"`
}

// Placeholder the assistant placeholder
type Placeholder struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Prompts     []string `json:"prompts,omitempty"`
}

// ModelCapability defines the available model capability filters
type ModelCapability string

// Model capability constants for filtering connectors
const (
	CapVision                ModelCapability = "vision"
	CapAudio                 ModelCapability = "audio"
	CapToolCalls             ModelCapability = "tool_calls"
	CapReasoning             ModelCapability = "reasoning"
	CapStreaming             ModelCapability = "streaming"
	CapJSON                  ModelCapability = "json"
	CapMultimodal            ModelCapability = "multimodal"
	CapTemperatureAdjustable ModelCapability = "temperature_adjustable"
)

// ConnectorOptions the connector selection options
// Allows defining optional connector selection with filtering capabilities
type ConnectorOptions struct {
	Optional   *bool             `json:"optional"`             // Whether connector is optional for user selection (nil=default, false=hidden, true=shown)
	Connectors []string          `json:"connectors,omitempty"` // List of available connectors, empty means all connectors are available
	Filters    []ModelCapability `json:"filters,omitempty"`    // Filter by model capabilities, conditions can be stacked
}

// AssistantModel the assistant database model
type AssistantModel struct {
	ID                   string                 `json:"assistant_id"`                     // Assistant ID
	Type                 string                 `json:"type,omitempty"`                   // Assistant Type, default is assistant
	Name                 string                 `json:"name,omitempty"`                   // Assistant Name
	Avatar               string                 `json:"avatar,omitempty"`                 // Assistant Avatar
	Connector            string                 `json:"connector"`                        // AI Connector (default connector)
	ConnectorOptions     *ConnectorOptions      `json:"connector_options,omitempty"`      // Connector selection options for user to choose from
	Path                 string                 `json:"path,omitempty"`                   // Assistant Path
	BuiltIn              bool                   `json:"built_in,omitempty"`               // Whether this is a built-in assistant
	Sort                 int                    `json:"sort,omitempty"`                   // Assistant Sort
	Description          string                 `json:"description,omitempty"`            // Assistant Description
	Tags                 []string               `json:"tags,omitempty"`                   // Assistant Tags
	Modes                []string               `json:"modes,omitempty"`                  // Supported modes (e.g., ["task", "chat"]), null means all modes are supported
	DefaultMode          string                 `json:"default_mode,omitempty"`           // Default mode, can be empty
	Readonly             bool                   `json:"readonly,omitempty"`               // Whether this assistant is readonly
	Public               bool                   `json:"public,omitempty"`                 // Whether this assistant is shared across all teams in the platform
	Share                string                 `json:"share,omitempty"`                  // Assistant sharing scope (private/team)
	Mentionable          bool                   `json:"mentionable,omitempty"`            // Whether this assistant is mentionable
	Automated            bool                   `json:"automated,omitempty"`              // Whether this assistant is automated
	Options              map[string]interface{} `json:"options,omitempty"`                // AI Options
	Prompts              []Prompt               `json:"prompts,omitempty"`                // AI Prompts (default prompts)
	PromptPresets        map[string][]Prompt    `json:"prompt_presets,omitempty"`         // Prompt presets organized by mode (e.g., "chat", "task", etc.)
	DisableGlobalPrompts bool                   `json:"disable_global_prompts,omitempty"` // Whether to disable global prompts, default is false
	KB                   *KnowledgeBase         `json:"kb,omitempty"`                     // Knowledge base configuration
	DB                   *Database              `json:"db,omitempty"`                     // Database configuration
	MCP                  *MCPServers            `json:"mcp,omitempty"`                    // MCP servers configuration
	Workflow             *Workflow              `json:"workflow,omitempty"`               // Workflow configuration
	Placeholder          *Placeholder           `json:"placeholder,omitempty"`            // Assistant Placeholder
	Source               string                 `json:"source,omitempty"`                 // Hook script source code
	Locales              i18n.Map               `json:"locales,omitempty"`                // Assistant Locales
	Uses                 *context.Uses          `json:"uses,omitempty"`                   // Assistant-specific wrapper configurations for vision, audio, etc. If not set, use global settings
	Search               *searchTypes.Config    `json:"search,omitempty"`                 // Search configuration (web, kb, db, citation, weights, etc.)
	CreatedAt            int64                  `json:"created_at"`                       // Creation timestamp
	UpdatedAt            int64                  `json:"updated_at"`                       // Last update timestamp

	// Permission management fields (not exposed in JSON API responses)
	YaoCreatedBy string `json:"-"` // User who created the assistant (not exposed in JSON)
	YaoUpdatedBy string `json:"-"` // User who last updated the assistant (not exposed in JSON)
	YaoTeamID    string `json:"-"` // Team ID for team-based access control (not exposed in JSON)
	YaoTenantID  string `json:"-"` // Tenant ID for multi-tenancy support (not exposed in JSON)
}

// =============================================================================
// Search Types (for search result storage)
// =============================================================================

// Search represents stored search results for a request
// Stores all intermediate processing results for debugging, replay, and citation
type Search struct {
	ID         int64          `json:"id"`
	RequestID  string         `json:"request_id"`
	ChatID     string         `json:"chat_id"`
	Query      string         `json:"query"`               // Original query
	Config     map[string]any `json:"config,omitempty"`    // Search config used (for tuning)
	Keywords   []string       `json:"keywords,omitempty"`  // Extracted keywords (Web/NLP)
	Entities   []Entity       `json:"entities,omitempty"`  // Extracted entities (Graph)
	Relations  []Relation     `json:"relations,omitempty"` // Extracted relations (Graph)
	DSL        map[string]any `json:"dsl,omitempty"`       // Generated QueryDSL (DB)
	Source     string         `json:"source"`              // web/kb/db/auto
	References []Reference    `json:"references"`          // References with global index
	Graph      []GraphNode    `json:"graph,omitempty"`     // Graph nodes from KB
	XML        string         `json:"xml,omitempty"`       // Formatted XML for LLM
	Prompt     string         `json:"prompt,omitempty"`    // Citation prompt
	Duration   int64          `json:"duration_ms"`         // Search duration in ms
	Error      string         `json:"error,omitempty"`     // Error if failed
	CreatedAt  time.Time      `json:"created_at"`
}

// Reference represents a single reference with global index (for storage)
type Reference struct {
	Index    int            `json:"index"`             // Global index (1-based, unique within request)
	Type     string         `json:"type"`              // web/kb/db
	Title    string         `json:"title"`             // Reference title
	URL      string         `json:"url,omitempty"`     // URL (for web)
	Snippet  string         `json:"snippet,omitempty"` // Short snippet
	Content  string         `json:"content,omitempty"` // Full content
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SearchFilter for listing searches
type SearchFilter struct {
	RequestID string `json:"request_id,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	Source    string `json:"source,omitempty"`
}

// Entity represents an extracted entity (for Graph RAG)
type Entity struct {
	Name   string `json:"name"`
	Type   string `json:"type,omitempty"`
	Source string `json:"source,omitempty"`
}

// Relation represents an extracted relation (for Graph RAG)
type Relation struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Source    string `json:"source,omitempty"`
}

// GraphNode represents a node from knowledge graph
type GraphNode struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Label      string         `json:"label,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Score      float64        `json:"score,omitempty"`
}
