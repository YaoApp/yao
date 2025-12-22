package memory

import (
	"sync"
)

// Global manager instance
var globalManager Manager

// Init initializes the global memory manager with the given configuration
// Called by agent.Load() after loading agent DSL
func Init(config *Config) {
	globalManager = NewManager(config)
}

// GetMemory returns a memory instance for the given identifiers using the global manager
// This is the main entry point for creating Memory instances from agent/context
func GetMemory(userID, teamID, chatID, contextID string) (*Memory, error) {
	if globalManager == nil {
		// Initialize with defaults if not configured
		globalManager = NewManagerWithDefaults()
	}
	return globalManager.Memory(userID, teamID, chatID, contextID)
}

// Close closes the global manager and releases resources
func Close() error {
	if globalManager != nil {
		err := globalManager.Close()
		globalManager = nil
		return err
	}
	return nil
}

// DefaultManager is the default memory manager implementation
type DefaultManager struct {
	config   *Config
	memories sync.Map // map[string]*Memory, key is composite of userID:teamID:chatID:contextID
}

// NewManager creates a new memory manager with the given configuration
func NewManager(config *Config) Manager {
	if config == nil {
		config = &Config{}
	}
	return &DefaultManager{
		config: config,
	}
}

// NewManagerWithDefaults creates a new memory manager with default configuration
func NewManagerWithDefaults() Manager {
	return NewManager(&Config{
		User:    DefaultUserStore,
		Team:    DefaultTeamStore,
		Chat:    DefaultChatStore,
		Context: DefaultContextStore,
	})
}

// memoryKey generates a unique key for the memory instance
func memoryKey(userID, teamID, chatID, contextID string) string {
	return userID + ":" + teamID + ":" + chatID + ":" + contextID
}

// Memory returns the memory instance for given identifiers
func (m *DefaultManager) Memory(userID, teamID, chatID, contextID string) (*Memory, error) {
	key := memoryKey(userID, teamID, chatID, contextID)

	// Check if memory already exists
	if val, ok := m.memories.Load(key); ok {
		return val.(*Memory), nil
	}

	// Create new memory instance
	mem, err := New(m.config, userID, teamID, chatID, contextID)
	if err != nil {
		return nil, err
	}

	// Store and return (use LoadOrStore for thread safety)
	actual, _ := m.memories.LoadOrStore(key, mem)
	return actual.(*Memory), nil
}

// Close closes all stores and releases resources
func (m *DefaultManager) Close() error {
	// Clear all cached memory instances
	m.memories.Range(func(key, value interface{}) bool {
		m.memories.Delete(key)
		return true
	})
	return nil
}

// Ensure DefaultManager implements Manager
var _ Manager = (*DefaultManager)(nil)
