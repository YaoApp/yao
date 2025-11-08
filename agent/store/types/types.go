package types

import (
	"github.com/yaoapp/xun/dbal/query"
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

// MCPServers the MCP servers configuration
type MCPServers struct {
	Servers []string               `json:"servers,omitempty"` // MCP server IDs
	Options map[string]interface{} `json:"options,omitempty"` // Additional options for MCP servers
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

// AssistantModel the assistant database model
type AssistantModel struct {
	ID          string                 `json:"assistant_id"`          // Assistant ID
	Type        string                 `json:"type,omitempty"`        // Assistant Type, default is assistant
	Name        string                 `json:"name,omitempty"`        // Assistant Name
	Avatar      string                 `json:"avatar,omitempty"`      // Assistant Avatar
	Connector   string                 `json:"connector"`             // AI Connector
	Path        string                 `json:"path,omitempty"`        // Assistant Path
	BuiltIn     bool                   `json:"built_in,omitempty"`    // Whether this is a built-in assistant
	Sort        int                    `json:"sort,omitempty"`        // Assistant Sort
	Description string                 `json:"description,omitempty"` // Assistant Description
	Tags        []string               `json:"tags,omitempty"`        // Assistant Tags
	Readonly    bool                   `json:"readonly,omitempty"`    // Whether this assistant is readonly
	Public      bool                   `json:"public,omitempty"`      // Whether this assistant is shared across all teams in the platform
	Share       string                 `json:"share,omitempty"`       // Assistant sharing scope (private/team)
	Mentionable bool                   `json:"mentionable,omitempty"` // Whether this assistant is mentionable
	Automated   bool                   `json:"automated,omitempty"`   // Whether this assistant is automated
	Options     map[string]interface{} `json:"options,omitempty"`     // AI Options
	Prompts     []Prompt               `json:"prompts,omitempty"`     // AI Prompts
	KB          *KnowledgeBase         `json:"kb,omitempty"`          // Knowledge base configuration
	MCP         *MCPServers            `json:"mcp,omitempty"`         // MCP servers configuration
	Tools       *ToolCalls             `json:"tools,omitempty"`       // Assistant Tools
	Workflow    *Workflow              `json:"workflow,omitempty"`    // Workflow configuration
	Placeholder *Placeholder           `json:"placeholder,omitempty"` // Assistant Placeholder
	Locales     i18n.Map               `json:"locales,omitempty"`     // Assistant Locales
	CreatedAt   int64                  `json:"created_at"`            // Creation timestamp
	UpdatedAt   int64                  `json:"updated_at"`            // Last update timestamp

	// Permission management fields (not exposed in JSON API responses)
	YaoCreatedBy string `json:"-"` // User who created the assistant (not exposed in JSON)
	YaoUpdatedBy string `json:"-"` // User who last updated the assistant (not exposed in JSON)
	YaoTeamID    string `json:"-"` // Team ID for team-based access control (not exposed in JSON)
	YaoTenantID  string `json:"-"` // Tenant ID for multi-tenancy support (not exposed in JSON)
}
