package memory

import (
	"fmt"
	"time"

	"github.com/yaoapp/gou/store"
)

// Default TTL values for each memory space
const (
	DefaultUserTTL    = 0                // No expiration for user-level memory
	DefaultTeamTTL    = 0                // No expiration for team-level memory
	DefaultChatTTL    = 24 * time.Hour   // 24 hours for chat-level memory
	DefaultContextTTL = 30 * time.Minute // 30 minutes for context-level memory
)

// New creates a new Memory instance with the given configuration and identifiers
func New(cfg *Config, userID, teamID, chatID, contextID string) (*Memory, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	m := &Memory{
		UserID:    userID,
		TeamID:    teamID,
		ChatID:    chatID,
		ContextID: contextID,
		Config:    cfg,
	}

	// Initialize user namespace
	if userID != "" {
		ns, err := newNamespace(SpaceUser, userID, cfg.User, DefaultUserTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to create user namespace: %w", err)
		}
		m.User = ns
	}

	// Initialize team namespace
	if teamID != "" {
		ns, err := newNamespace(SpaceTeam, teamID, cfg.Team, DefaultTeamTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to create team namespace: %w", err)
		}
		m.Team = ns
	}

	// Initialize chat namespace
	if chatID != "" {
		ns, err := newNamespace(SpaceChat, chatID, cfg.Chat, DefaultChatTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to create chat namespace: %w", err)
		}
		m.Chat = ns
	}

	// Initialize context namespace
	if contextID != "" {
		ns, err := newNamespace(SpaceContext, contextID, cfg.Context, DefaultContextTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to create context namespace: %w", err)
		}
		m.Context = ns
	}

	return m, nil
}

// newNamespace creates a new Namespace with the given parameters
func newNamespace(space Space, id, storeID string, defaultTTL time.Duration) (*Namespace, error) {
	// Use default store ID if not specified
	if storeID == "" {
		switch space {
		case SpaceUser:
			storeID = DefaultUserStore
		case SpaceTeam:
			storeID = DefaultTeamStore
		case SpaceChat:
			storeID = DefaultChatStore
		case SpaceContext:
			storeID = DefaultContextStore
		}
	}

	// Get store instance
	s, err := store.Get(storeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get store %s: %w", storeID, err)
	}

	return &Namespace{
		Space:   space,
		ID:      id,
		Store:   s,
		StoreID: storeID,
		Prefix:  fmt.Sprintf("%s:%s:", space, id),
		Default: defaultTTL,
	}, nil
}

// GetUser returns the user-level memory namespace accessor
func (m *Memory) GetUser() NamespaceAccessor {
	if m.User == nil {
		return nil
	}
	return m.User
}

// GetTeam returns the team-level memory namespace accessor
func (m *Memory) GetTeam() NamespaceAccessor {
	if m.Team == nil {
		return nil
	}
	return m.Team
}

// GetChat returns the chat-level memory namespace accessor
func (m *Memory) GetChat() NamespaceAccessor {
	if m.Chat == nil {
		return nil
	}
	return m.Chat
}

// GetContext returns the context-level memory namespace accessor
func (m *Memory) GetContext() NamespaceAccessor {
	if m.Context == nil {
		return nil
	}
	return m.Context
}

// GetSpace returns a memory namespace by space type
func (m *Memory) GetSpace(space Space) NamespaceAccessor {
	switch space {
	case SpaceUser:
		return m.GetUser()
	case SpaceTeam:
		return m.GetTeam()
	case SpaceChat:
		return m.GetChat()
	case SpaceContext:
		return m.GetContext()
	default:
		return nil
	}
}

// GetStats returns memory statistics for all namespaces
func (m *Memory) GetStats() *Stats {
	stats := &Stats{}

	if m.User != nil {
		stats.User = m.User.Stats()
	}
	if m.Team != nil {
		stats.Team = m.Team.Stats()
	}
	if m.Chat != nil {
		stats.Chat = m.Chat.Stats()
	}
	if m.Context != nil {
		stats.Context = m.Context.Stats()
	}

	return stats
}

// Clear clears all memory in all namespaces for this memory instance
func (m *Memory) Clear() {
	if m.User != nil {
		m.User.Clear()
	}
	if m.Team != nil {
		m.Team.Clear()
	}
	if m.Chat != nil {
		m.Chat.Clear()
	}
	if m.Context != nil {
		m.Context.Clear()
	}
}
