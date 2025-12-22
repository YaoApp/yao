package memory

import "github.com/yaoapp/gou/store"

// Manager defines the interface for managing agent memory
type Manager interface {
	// Memory returns the memory instance for given identifiers
	Memory(userID, teamID, chatID, contextID string) (*Memory, error)

	// Close closes all stores and releases resources
	Close() error
}

// Accessor defines the interface for accessing memory from agent context
// This is the primary interface used by agent hooks and tools
type Accessor interface {
	// User returns the user-level memory namespace
	User() NamespaceAccessor

	// Team returns the team-level memory namespace
	Team() NamespaceAccessor

	// Chat returns the chat-level memory namespace
	Chat() NamespaceAccessor

	// Context returns the context-level memory namespace
	Context() NamespaceAccessor

	// Space returns a memory namespace by space type
	Space(space Space) NamespaceAccessor

	// Stats returns memory statistics
	Stats() *Stats
}

// NamespaceAccessor defines the interface for accessing a single memory namespace
// Embeds store.Store for all KV and list operations
type NamespaceAccessor interface {
	store.Store

	// GetID returns the namespace identifier (user_id, team_id, chat_id, or context_id)
	GetID() string

	// GetSpace returns the space type of this namespace
	GetSpace() Space

	// Stats returns statistics for this namespace
	Stats() *NamespaceStats
}

// Factory defines the interface for creating memory instances
type Factory interface {
	// Create creates a new memory instance with the given configuration
	Create(config *Config) (Manager, error)

	// CreateWithDefaults creates a new memory instance with default configuration
	CreateWithDefaults() (Manager, error)
}
