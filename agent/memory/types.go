package memory

import (
	"time"

	"github.com/yaoapp/gou/store"
)

// Space defines the memory space type
type Space string

const (
	// SpaceUser user-level memory, persists across all chats for a user
	// Use case: user preferences, long-term knowledge, personal settings
	SpaceUser Space = "user"

	// SpaceTeam team-level memory, shared across all users in a team
	// Use case: team knowledge, shared settings, collaborative data
	SpaceTeam Space = "team"

	// SpaceChat chat-level memory, persists within a single chat session
	// Use case: conversation context, chat-specific settings, accumulated knowledge
	SpaceChat Space = "chat"

	// SpaceContext context-level memory, temporary within a single request context
	// Use case: intermediate results, temporary variables, request-scoped cache
	SpaceContext Space = "context"
)

// Config represents the memory configuration
// Each field is a Store ID referencing gou/store, empty string uses built-in default
// All spaces use xun-based storage by default for persistence and reliability
type Config struct {
	User    string `json:"user,omitempty" yaml:"user,omitempty"`       // Store ID for user-level memory (default: xun-based)
	Team    string `json:"team,omitempty" yaml:"team,omitempty"`       // Store ID for team-level memory (default: xun-based)
	Chat    string `json:"chat,omitempty" yaml:"chat,omitempty"`       // Store ID for chat-level memory (default: xun-based)
	Context string `json:"context,omitempty" yaml:"context,omitempty"` // Store ID for context-level memory (default: xun-based, shorter TTL)
}

// DefaultStoreID constants for built-in stores
const (
	DefaultUserStore    = "__yao.agent.memory.user"
	DefaultTeamStore    = "__yao.agent.memory.team"
	DefaultChatStore    = "__yao.agent.memory.chat"
	DefaultContextStore = "__yao.agent.memory.context"
)

// Entry represents a memory entry
type Entry struct {
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	Space     Space                  `json:"space"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	TTL       time.Duration          `json:"ttl,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
}

// Namespace represents a memory namespace for a specific space
type Namespace struct {
	Space   Space         `json:"space"`
	ID      string        `json:"id"` // UserID, TeamID, ChatID, or ContextID depending on space
	Store   store.Store   `json:"-"`  // Underlying store
	StoreID string        `json:"-"`  // Store ID
	Prefix  string        `json:"-"`  // Computed key prefix (e.g., "user:123:", "team:456:")
	Default time.Duration `json:"-"`  // Default TTL for this namespace
}

// Memory represents the complete memory system for an agent
// It manages four separate namespaces: User, Team, Chat, and Context
type Memory struct {
	UserID    string `json:"user_id"`
	TeamID    string `json:"team_id"`
	ChatID    string `json:"chat_id"`
	ContextID string `json:"context_id"`

	User    *Namespace `json:"-"` // User-level memory namespace
	Team    *Namespace `json:"-"` // Team-level memory namespace
	Chat    *Namespace `json:"-"` // Chat-level memory namespace
	Context *Namespace `json:"-"` // Context-level memory namespace
	Config  *Config    `json:"-"` // Memory configuration
}

// Stats represents memory statistics
type Stats struct {
	User    *NamespaceStats `json:"user,omitempty"`
	Team    *NamespaceStats `json:"team,omitempty"`
	Chat    *NamespaceStats `json:"chat,omitempty"`
	Context *NamespaceStats `json:"context,omitempty"`
}

// NamespaceStats represents statistics for a single memory namespace
type NamespaceStats struct {
	Space    Space  `json:"space"`
	ID       string `json:"id"`
	KeyCount int    `json:"key_count"`
	StoreID  string `json:"store_id"`
}
