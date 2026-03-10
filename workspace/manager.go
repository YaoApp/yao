package workspace

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/volume"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
)

var mgr = NewManager()

// M returns the global Manager.
func M() *Manager {
	return mgr
}

// Manager owns workspace CRUD, file I/O, and node management.
// All node/client lookups go through tai.GetClient → registry.
type Manager struct{}

// NewManager creates a workspace manager.
func NewManager() *Manager {
	return &Manager{}
}

// Create allocates storage on the target node and persists metadata.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Workspace, error) {
	if opts.Node == "" {
		return nil, ErrNodeMissing
	}

	client, ok := tai.GetClient(opts.Node)
	if !ok {
		return nil, ErrNodeOffline
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
// Scans all registered nodes.
func (m *Manager) Get(ctx context.Context, id string) (*Workspace, error) {
	for _, snap := range listNodes() {
		client, ok := tai.GetClient(snap.TaiID)
		if !ok {
			continue
		}
		ws, err := readMeta(ctx, client, id)
		if err != nil {
			continue
		}
		if ws.Node == "" {
			ws.Node = snap.TaiID
		}
		return ws, nil
	}
	return nil, ErrNotFound
}

// List returns workspaces, optionally filtered by owner and/or node.
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Workspace, error) {
	var result []*Workspace
	for _, snap := range listNodes() {
		if opts.Node != "" && snap.TaiID != opts.Node {
			continue
		}
		client, ok := tai.GetClient(snap.TaiID)
		if !ok {
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
			ws, err := readMeta(ctx, client, e.Path)
			if err != nil {
				continue
			}
			if ws.Node == "" {
				ws.Node = snap.TaiID
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

// Nodes returns all registered Tai nodes with their online status.
func (m *Manager) Nodes() []NodeInfo {
	nodes := listNodes()
	result := make([]NodeInfo, 0, len(nodes))
	for _, snap := range nodes {
		result = append(result, NodeInfo{
			Name:   snap.TaiID,
			Online: snap.Status == "online" || snap.Status == "",
		})
	}
	return result
}

// FS returns an fs.FS-compatible filesystem for the given workspace.
func (m *Manager) FS(ctx context.Context, id string) (taiworkspace.FS, error) {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
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

// Rename renames a file or directory within the workspace.
func (m *Manager) Rename(ctx context.Context, id string, oldPath, newPath string) error {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return client.Volume().Rename(ctx, id, oldPath, newPath)
}

// MkdirAll creates a directory (and parents) in the workspace.
func (m *Manager) MkdirAll(ctx context.Context, id string, path string) error {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return client.Volume().MkdirAll(ctx, id, path)
}

// Volume returns the Volume interface for the node hosting the given workspace.
func (m *Manager) Volume(ctx context.Context, id string) (volume.Volume, string, error) {
	_, client, err := m.resolve(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return client.Volume(), id, nil
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

// resolve finds the workspace and its tai.Client by scanning all registered nodes.
func (m *Manager) resolve(ctx context.Context, id string) (*Workspace, *tai.Client, error) {
	for _, snap := range listNodes() {
		client, ok := tai.GetClient(snap.TaiID)
		if !ok {
			continue
		}
		ws, err := readMeta(ctx, client, id)
		if err != nil {
			continue
		}
		return ws, client, nil
	}
	return nil, nil, ErrNotFound
}

func readMeta(ctx context.Context, client *tai.Client, id string) (*Workspace, error) {
	data, _, err := client.Volume().ReadFile(ctx, id, metadataFile)
	if err != nil {
		return nil, err
	}
	return unmarshalMeta(data)
}

func listNodes() []registry.NodeSnapshot {
	reg := registry.Global()
	if reg == nil {
		return nil
	}
	return reg.List()
}

// DirEntry represents a file or directory entry in a workspace listing.
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}
