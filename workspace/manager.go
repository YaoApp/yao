package workspace

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yaoapp/yao/tai"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
)

// Manager owns workspace CRUD, file I/O, and node management.
// Pools are shared with sandbox.Manager — both reference the same tai.Client instances.
type Manager struct {
	pools map[string]*tai.Client
	mu    sync.RWMutex
}

// NewManager creates a workspace manager with the given pools.
func NewManager(pools map[string]*tai.Client) *Manager {
	if pools == nil {
		pools = make(map[string]*tai.Client)
	}
	return &Manager{pools: pools}
}

// Create allocates storage on the target node and persists metadata.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Workspace, error) {
	if opts.Node == "" {
		return nil, ErrNodeMissing
	}

	client, err := m.getClient(opts.Node)
	if err != nil {
		return nil, err
	}

	id := opts.ID
	if id == "" {
		id = generateID()
	}

	now := time.Now().UTC()
	ws := &Workspace{
		ID:        id,
		Name:      opts.Name,
		Owner:     opts.Owner,
		Node:      opts.Node,
		Labels:    opts.Labels,
		CreatedAt: now,
		UpdatedAt: now,
	}

	vol := client.Volume()

	if err := vol.MkdirAll(ctx, id, "."); err != nil {
		return nil, fmt.Errorf("workspace: create directory: %w", err)
	}

	data, err := marshalMeta(ws)
	if err != nil {
		return nil, err
	}
	if err := vol.WriteFile(ctx, id, metadataFile, data, 0644); err != nil {
		return nil, fmt.Errorf("workspace: write metadata: %w", err)
	}

	return ws, nil
}

// Get returns a workspace by ID.
// If the node is unknown, scans all pools.
func (m *Manager) Get(ctx context.Context, id string) (*Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for nodeName, client := range m.pools {
		ws, err := m.readMeta(ctx, client, id)
		if err != nil {
			continue
		}
		if ws.Node == "" {
			ws.Node = nodeName
		}
		return ws, nil
	}
	return nil, ErrNotFound
}

// List returns workspaces, optionally filtered by owner and/or node.
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Workspace
	for nodeName, client := range m.pools {
		if opts.Node != "" && nodeName != opts.Node {
			continue
		}
		entries, err := client.Volume().ListDir(ctx, "", ".")
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir {
				continue
			}
			ws, err := m.readMeta(ctx, client, e.Path)
			if err != nil {
				continue
			}
			if ws.Node == "" {
				ws.Node = nodeName
			}
			if opts.Owner != "" && ws.Owner != opts.Owner {
				continue
			}
			result = append(result, ws)
		}
	}
	return result, nil
}

// Update modifies workspace metadata (Name, Labels).
// Node and Owner are immutable after creation.
func (m *Manager) Update(ctx context.Context, id string, opts UpdateOptions) (*Workspace, error) {
	ws, client, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}

	if opts.Name != nil {
		ws.Name = *opts.Name
	}
	if opts.Labels != nil {
		ws.Labels = opts.Labels
	}
	ws.UpdatedAt = time.Now().UTC()

	data, err := marshalMeta(ws)
	if err != nil {
		return nil, err
	}
	if err := client.Volume().WriteFile(ctx, id, metadataFile, data, 0644); err != nil {
		return nil, fmt.Errorf("workspace: write metadata: %w", err)
	}
	return ws, nil
}

// Delete removes workspace storage from the node.
func (m *Manager) Delete(ctx context.Context, id string, force bool) error {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}

	vol := client.Volume()
	if err := vol.Remove(ctx, id, ".", true); err != nil {
		return fmt.Errorf("workspace: remove: %w", err)
	}
	return nil
}

// Nodes returns all configured Tai nodes with their online status.
func (m *Manager) Nodes() []NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]NodeInfo, 0, len(m.pools))
	for name := range m.pools {
		nodes = append(nodes, NodeInfo{
			Name:   name,
			Online: true,
		})
	}
	return nodes
}

// FS returns an fs.FS-compatible filesystem for the given workspace.
func (m *Manager) FS(ctx context.Context, id string) (taiworkspace.FS, error) {
	ws, client, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = ws
	return client.Workspace(id), nil
}

// ReadFile reads a file from the workspace.
func (m *Manager) ReadFile(ctx context.Context, id string, path string) ([]byte, error) {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	data, _, err := client.Volume().ReadFile(ctx, id, path)
	return data, err
}

// WriteFile writes a file to the workspace.
func (m *Manager) WriteFile(ctx context.Context, id string, path string, data []byte, perm os.FileMode) error {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return client.Volume().WriteFile(ctx, id, path, data, perm)
}

// ListDir lists entries in a workspace directory.
func (m *Manager) ListDir(ctx context.Context, id string, path string) ([]DirEntry, error) {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	entries, err := client.Volume().ListDir(ctx, id, path)
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = DirEntry{
			Name:  e.Path,
			IsDir: e.IsDir,
			Size:  e.Size,
		}
	}
	return result, nil
}

// Remove deletes a file or directory from the workspace.
func (m *Manager) Remove(ctx context.Context, id string, path string) error {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return client.Volume().Remove(ctx, id, path, true)
}

// AddPool registers a new Tai node.
func (m *Manager) AddPool(name string, client *tai.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pools[name] = client
}

// RemovePool unregisters a Tai node.
func (m *Manager) RemovePool(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pools, name)
}

// NodeForWorkspace returns the node name for a given workspace ID.
// Used by sandbox.Manager to route container creation to the correct pool.
func (m *Manager) NodeForWorkspace(ctx context.Context, id string) (string, error) {
	ws, _, err := m.resolve(ctx, id)
	if err != nil {
		return "", err
	}
	return ws.Node, nil
}

// MountPath returns the host-side directory path for a workspace,
// suitable for use as a Docker bind mount source.
// For local volumes this is dataDir/{id}; for remote (Tai) the server handles mounts.
func (m *Manager) MountPath(ctx context.Context, id string) (string, error) {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return "", err
	}
	dataDir := client.DataDir()
	if dataDir == "" {
		return "", nil
	}
	return dataDir + "/" + id, nil
}

// --- internal ---

func (m *Manager) getClient(node string) (*tai.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.pools[node]
	if !ok {
		return nil, ErrNodeOffline
	}
	return client, nil
}

// resolve finds the workspace and its tai.Client by scanning pools.
func (m *Manager) resolve(ctx context.Context, id string) (*Workspace, *tai.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.pools {
		ws, err := m.readMeta(ctx, client, id)
		if err != nil {
			continue
		}
		return ws, client, nil
	}
	return nil, nil, ErrNotFound
}

func (m *Manager) readMeta(ctx context.Context, client *tai.Client, id string) (*Workspace, error) {
	data, _, err := client.Volume().ReadFile(ctx, id, metadataFile)
	if err != nil {
		return nil, err
	}
	return unmarshalMeta(data)
}

// DirEntry represents a file or directory entry in a workspace listing.
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}
