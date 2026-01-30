package ipc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// Manager manages IPC sessions
type Manager struct {
	sessions sync.Map // sessionID → *Session
	sockDir  string   // Socket directory
}

// NewManager creates a new IPC manager
func NewManager(sockDir string) *Manager {
	return &Manager{
		sockDir: sockDir,
	}
}

// Create creates a new IPC session
func (m *Manager) Create(ctx context.Context, sessionID string, agentCtx *AgentContext, mcpTools map[string]*MCPTool) (*Session, error) {
	// Close existing session if any
	m.Close(sessionID)

	// Create socket path using hash to avoid path length issues
	// Unix socket paths are limited to ~104-108 bytes
	socketPath := m.socketPath(sessionID)

	// Ensure directory exists
	if err := os.MkdirAll(m.sockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket file if any
	os.Remove(socketPath)

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Unix socket: %w", err)
	}

	// Set socket permissions (readable/writable by all users)
	// This allows container processes running as non-root to connect
	if err := os.Chmod(socketPath, 0666); err != nil {
		listener.Close()
		os.Remove(socketPath)
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Create cancellable context
	sessionCtx, cancel := context.WithCancel(ctx)

	session := &Session{
		ID:         sessionID,
		SocketPath: socketPath,
		Listener:   listener,
		Context:    agentCtx,
		MCPTools:   mcpTools,
		cancel:     cancel,
	}

	// Start serving in background
	go session.serve(sessionCtx)

	// Store session
	m.sessions.Store(sessionID, session)

	return session, nil
}

// Close closes an IPC session
func (m *Manager) Close(sessionID string) error {
	if s, ok := m.sessions.LoadAndDelete(sessionID); ok {
		session := s.(*Session)
		return session.Close()
	}
	return nil
}

// Get returns an existing session
func (m *Manager) Get(sessionID string) (*Session, bool) {
	if s, ok := m.sessions.Load(sessionID); ok {
		return s.(*Session), true
	}
	return nil, false
}

// CloseAll closes all sessions
func (m *Manager) CloseAll() {
	m.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.Close()
		m.sessions.Delete(key)
		return true
	})
}

// socketPath generates a short socket path using hash
// Unix socket paths are limited to ~104-108 bytes on most systems
func (m *Manager) socketPath(sessionID string) string {
	hash := sha256.Sum256([]byte(sessionID))
	shortHash := hex.EncodeToString(hash[:8]) // 16 chars
	return filepath.Join(m.sockDir, shortHash+".sock")
}

// GetSocketPath returns the socket path for a session ID (for external use)
func (m *Manager) GetSocketPath(sessionID string) string {
	return m.socketPath(sessionID)
}
