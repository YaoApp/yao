package workspace

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/tai/volume"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
)

var mgr = NewManager()

// M returns the global Manager.
func M() *Manager {
	return mgr
}

// Manager owns workspace CRUD, file I/O, and node management.
type Manager struct {
	gitRepoCache sync.Map // map[workspaceID]*gitRepoCacheEntry
}

type gitRepoCacheEntry struct {
	repos    []volume.GitRepo
	cachedAt time.Time
}

// NewManager creates a workspace manager.
func NewManager() *Manager {
	return &Manager{}
}

// Create allocates storage on the target node and persists metadata.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Workspace, error) {
	if opts.Node == "" {
		return nil, ErrNodeMissing
	}

	res, ok := tai.GetResources(opts.Node)
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

	vol := res.Volume
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
func (m *Manager) Get(ctx context.Context, id string) (*Workspace, error) {
	for _, snap := range listNodes() {
		res, ok := tai.GetResources(snap.TaiID)
		if !ok {
			continue
		}
		ws, err := readMeta(ctx, res.Volume, id)
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
		res, ok := tai.GetResources(snap.TaiID)
		if !ok {
			continue
		}
		entries, err := res.Volume.ListDir(ctx, "", ".")
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir {
				continue
			}
			ws, err := readMeta(ctx, res.Volume, e.Path)
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
func (m *Manager) Update(ctx context.Context, id string, opts UpdateOptions) (*Workspace, error) {
	ws, vol, err := m.resolve(ctx, id)
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
	if err := vol.WriteFile(ctx, id, metadataFile, data, 0644); err != nil {
		return nil, fmt.Errorf("workspace: write metadata: %w", err)
	}
	return ws, nil
}

// Delete removes workspace storage from the node.
func (m *Manager) Delete(ctx context.Context, id string, force bool) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	if err := vol.Remove(ctx, id, ".", true); err != nil {
		return fmt.Errorf("workspace: remove: %w", err)
	}
	m.gitRepoCache.Delete(id)
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
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return taiworkspace.New(vol, id), nil
}

// ReadFile reads a file from the workspace.
func (m *Manager) ReadFile(ctx context.Context, id string, path string) ([]byte, *volume.FileInfo, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return vol.ReadFile(ctx, id, path)
}

// WriteFile writes a file to the workspace.
func (m *Manager) WriteFile(ctx context.Context, id string, path string, data []byte, perm os.FileMode) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.WriteFile(ctx, id, path, data, perm)
}

// ListDir lists entries in a workspace directory.
func (m *Manager) ListDir(ctx context.Context, id string, path string) ([]DirEntry, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	entries, err := vol.ListDir(ctx, id, path)
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = DirEntry{
			Name:    e.Path,
			IsDir:   e.IsDir,
			Size:    e.Size,
			ModTime: e.Mtime,
		}
	}
	return result, nil
}

// Remove deletes a file or directory from the workspace.
func (m *Manager) Remove(ctx context.Context, id string, path string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.Remove(ctx, id, path, true)
}

// Rename renames a file or directory within the workspace.
func (m *Manager) Rename(ctx context.Context, id string, oldPath, newPath string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.Rename(ctx, id, oldPath, newPath)
}

// MkdirAll creates a directory (and parents) in the workspace.
func (m *Manager) MkdirAll(ctx context.Context, id string, path string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.MkdirAll(ctx, id, path)
}

// Volume returns the Volume interface for the node hosting the given workspace.
func (m *Manager) Volume(ctx context.Context, id string) (volume.Volume, string, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return vol, id, nil
}

// NodeForWorkspace returns the node name for a given workspace ID.
func (m *Manager) NodeForWorkspace(ctx context.Context, id string) (string, error) {
	ws, _, err := m.resolve(ctx, id)
	if err != nil {
		return "", err
	}
	return ws.Node, nil
}

// MountPath returns the host-side directory path for a workspace.
func (m *Manager) MountPath(ctx context.Context, id string) (string, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return "", err
	}
	return vol.Abs(ctx, id, ".")
}

// --- Git ---

// GitListRepos lists Git repositories in a workspace.
// When refresh is false, returns cached results if available.
func (m *Manager) GitListRepos(ctx context.Context, id string, refresh bool) ([]volume.GitRepo, error) {
	if !refresh {
		if entry, ok := m.gitRepoCache.Load(id); ok {
			return entry.(*gitRepoCacheEntry).repos, nil
		}
	}

	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	repos, err := vol.GitListRepos(ctx, id, ".")
	if err != nil {
		return nil, err
	}
	m.gitRepoCache.Store(id, &gitRepoCacheEntry{repos: repos, cachedAt: time.Now()})
	return repos, nil
}

// GitStatus returns the status of a Git repository.
func (m *Manager) GitStatus(ctx context.Context, id, repoPath string) (*volume.GitStatusResult, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return vol.GitStatus(ctx, id, repoPath)
}

// GitFileDiff returns the diff for a single file.
func (m *Manager) GitFileDiff(ctx context.Context, id, repoPath, filePath string, staged bool) (*volume.GitFileDiffResult, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return vol.GitFileDiff(ctx, id, repoPath, filePath, staged)
}

// GitAdd stages files in a Git repository.
func (m *Manager) GitAdd(ctx context.Context, id, repoPath string, files []string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitAdd(ctx, id, repoPath, files)
}

// GitReset unstages files in a Git repository.
func (m *Manager) GitReset(ctx context.Context, id, repoPath string, files []string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitReset(ctx, id, repoPath, files)
}

// GitCommit creates a commit in a Git repository.
func (m *Manager) GitCommit(ctx context.Context, id, repoPath, message, authorName, authorEmail string, allowEmpty bool) (*volume.GitCommitResult, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return vol.GitCommit(ctx, id, repoPath, message, authorName, authorEmail, allowEmpty)
}

// GitDiscardChanges discards uncommitted changes in a Git repository.
func (m *Manager) GitDiscardChanges(ctx context.Context, id, repoPath string, files []string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitDiscardChanges(ctx, id, repoPath, files)
}

// --- Workspace Git Config ---

// GitConfigGet retrieves workspace-level Git config values.
func (m *Manager) GitConfigGet(ctx context.Context, id, key string) (map[string]string, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return vol.GitConfigGet(ctx, id, key)
}

// GitConfigSet sets a workspace-level Git config value.
func (m *Manager) GitConfigSet(ctx context.Context, id, key, value string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitConfigSet(ctx, id, key, value)
}

// GitCredentialSet stores an HTTPS credential for the workspace.
func (m *Manager) GitCredentialSet(ctx context.Context, id, host, username, token string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitCredentialSet(ctx, id, host, username, token)
}

// GitCredentialList lists HTTPS credentials for the workspace.
func (m *Manager) GitCredentialList(ctx context.Context, id string) ([]volume.GitCredentialEntry, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return vol.GitCredentialList(ctx, id)
}

// GitCredentialDelete removes an HTTPS credential by host.
func (m *Manager) GitCredentialDelete(ctx context.Context, id, host string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitCredentialDelete(ctx, id, host)
}

// GitSSHKeyImport imports an SSH key pair for the workspace.
func (m *Manager) GitSSHKeyImport(ctx context.Context, id, name, privateKey, publicKey, host string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitSSHKeyImport(ctx, id, name, privateKey, publicKey, host)
}

// GitSSHKeyList lists SSH keys for the workspace.
func (m *Manager) GitSSHKeyList(ctx context.Context, id string) ([]volume.GitSSHKeyEntry, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return vol.GitSSHKeyList(ctx, id)
}

// GitSSHKeyDelete removes an SSH key by name.
func (m *Manager) GitSSHKeyDelete(ctx context.Context, id, name string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitSSHKeyDelete(ctx, id, name)
}

// --- Git Remote Sync ---

// GitFetch fetches from a remote in the workspace.
func (m *Manager) GitFetch(ctx context.Context, id, repoPath, remote string) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitFetch(ctx, id, repoPath, remote)
}

// GitPull pulls from a remote in the workspace.
func (m *Manager) GitPull(ctx context.Context, id, repoPath, remote string, rebase bool) (*volume.GitPullResult, error) {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	return vol.GitPull(ctx, id, repoPath, remote, rebase)
}

// GitPush pushes to a remote in the workspace.
func (m *Manager) GitPush(ctx context.Context, id, repoPath, remote string, force, setUpstream bool) error {
	_, vol, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return vol.GitPush(ctx, id, repoPath, remote, force, setUpstream)
}

// --- internal ---

// resolve finds the workspace and its Volume by scanning all registered nodes.
func (m *Manager) resolve(ctx context.Context, id string) (*Workspace, volume.Volume, error) {
	for _, snap := range listNodes() {
		res, ok := tai.GetResources(snap.TaiID)
		if !ok {
			continue
		}
		ws, err := readMeta(ctx, res.Volume, id)
		if err != nil {
			continue
		}
		return ws, res.Volume, nil
	}
	return nil, nil, ErrNotFound
}

func readMeta(ctx context.Context, vol volume.Volume, id string) (*Workspace, error) {
	data, _, err := vol.ReadFile(ctx, id, metadataFile)
	if err != nil {
		return nil, err
	}
	return unmarshalMeta(data)
}

func listNodes() []taitypes.NodeMeta {
	reg := registry.Global()
	if reg == nil {
		return nil
	}
	return reg.List()
}

// DirEntry represents a file or directory entry in a workspace listing.
type DirEntry struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time,omitempty"`
}
