package types

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
)

// Setting represents the conversation configuration structure
// Used to configure basic conversation parameters including connector, user field, table name, etc.
type Setting struct {
	Connector string                 `json:"connector,omitempty" yaml:"connector,omitempty"` // Connector name, default is "default"
	MaxSize   int                    `json:"max_size,omitempty" yaml:"max_size,omitempty"`   // Maximum storage size limit, default is 20
	TTL       int                    `json:"ttl,omitempty" yaml:"ttl,omitempty"`             // Time To Live in seconds, default is 90 * 24 * 60 * 60 (90 days)
	Options   map[string]interface{} `json:"optional,omitempty" yaml:"optional,omitempty"`   // The options for the store
}

// ChatInfo represents the chat information structure
// Contains basic information and history for a single chat
type ChatInfo struct {
	Chat    map[string]interface{}   `json:"chat"`    // Basic chat information
	History []map[string]interface{} `json:"history"` // Chat history records
}

// ChatFilter represents the chat filter structure
// Used for filtering and pagination when retrieving chat lists
type ChatFilter struct {
	Keywords string `json:"keywords,omitempty"` // Keyword search
	Page     int    `json:"page,omitempty"`     // Page number, starting from 1
	PageSize int    `json:"pagesize,omitempty"` // Number of items per page
	Order    string `json:"order,omitempty"`    // Sort order: desc/asc
	Silent   *bool  `json:"silent,omitempty"`   // Include silent messages (default: false)
}

// ChatGroup represents the chat group structure
// Groups chats by date
type ChatGroup struct {
	Label string                   `json:"label"` // Group label (typically a date)
	Chats []map[string]interface{} `json:"chats"` // List of chats in this group
}

// ChatGroupResponse represents the paginated chat group response
// Contains paginated chat group information
type ChatGroupResponse struct {
	Groups   []ChatGroup `json:"groups"`    // List of chat groups
	Page     int         `json:"page"`      // Current page number
	PageSize int         `json:"pagesize"`  // Items per page
	Total    int64       `json:"total"`     // Total number of records
	LastPage int         `json:"last_page"` // Last page number
}

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
	Optional   bool              `json:"optional,omitempty"`   // Whether connector is optional for user selection
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
	CreatedAt            int64                  `json:"created_at"`                       // Creation timestamp
	UpdatedAt            int64                  `json:"updated_at"`                       // Last update timestamp

	// Permission management fields (not exposed in JSON API responses)
	YaoCreatedBy string `json:"-"` // User who created the assistant (not exposed in JSON)
	YaoUpdatedBy string `json:"-"` // User who last updated the assistant (not exposed in JSON)
	YaoTeamID    string `json:"-"` // Team ID for team-based access control (not exposed in JSON)
	YaoTenantID  string `json:"-"` // Tenant ID for multi-tenancy support (not exposed in JSON)
}
