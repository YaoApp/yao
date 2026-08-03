package volume

import (
	"context"
	"io/fs"
	"os"
	"time"
)

// Volume provides filesystem IO, directory synchronization, and archive operations.
// Remote connects to Tai gRPC :19100; Local operates directly on disk.
type Volume interface {
	ReadFile(ctx context.Context, sessionID, path string) ([]byte, os.FileMode, error)
	WriteFile(ctx context.Context, sessionID, path string, data []byte, perm os.FileMode) error
	Stat(ctx context.Context, sessionID, path string) (*FileInfo, error)
	ListDir(ctx context.Context, sessionID, path string) ([]FileInfo, error)
	Remove(ctx context.Context, sessionID, path string, recursive bool) error
	Rename(ctx context.Context, sessionID, oldPath, newPath string) error
	MkdirAll(ctx context.Context, sessionID, path string) error
	Abs(ctx context.Context, sessionID, path string) (string, error)
	ListSkills(ctx context.Context, sessionID, dir string) ([]SkillInfo, error)

	SyncPush(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error)
	SyncPull(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error)
	Copy(ctx context.Context, sessionID, src, dst string, opts ...SyncOption) (*SyncResult, error)

	Zip(ctx context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error)
	Unzip(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error)
	Gzip(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error)
	Gunzip(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error)
	Tar(ctx context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error)
	Untar(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error)
	Tgz(ctx context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error)
	Untgz(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error)

	// --- Git ---
	GitListRepos(ctx context.Context, sessionID, basePath string) ([]GitRepo, error)
	GitStatus(ctx context.Context, sessionID, repoPath string) (*GitStatusResult, error)
	GitFileDiff(ctx context.Context, sessionID, repoPath, filePath string, staged bool) (*GitFileDiffResult, error)
	GitAdd(ctx context.Context, sessionID, repoPath string, files []string) error
	GitReset(ctx context.Context, sessionID, repoPath string, files []string) error
	GitCommit(ctx context.Context, sessionID, repoPath, message, authorName, authorEmail string, allowEmpty bool) (*GitCommitResult, error)
	GitDiscardChanges(ctx context.Context, sessionID, repoPath string, files []string) error

	// --- Workspace Git Config ---
	GitConfigGet(ctx context.Context, sessionID, key string) (map[string]string, error)
	GitConfigSet(ctx context.Context, sessionID, key, value string) error
	GitCredentialSet(ctx context.Context, sessionID, host, username, token string) error
	GitCredentialList(ctx context.Context, sessionID string) ([]GitCredentialEntry, error)
	GitCredentialDelete(ctx context.Context, sessionID, host string) error
	GitSSHKeyImport(ctx context.Context, sessionID, name, privateKey, publicKey, host string) error
	GitSSHKeyList(ctx context.Context, sessionID string) ([]GitSSHKeyEntry, error)
	GitSSHKeyDelete(ctx context.Context, sessionID, name string) error

	// --- Git Remote Sync ---
	GitFetch(ctx context.Context, sessionID, repoPath, remote string) error
	GitPull(ctx context.Context, sessionID, repoPath, remote string, rebase bool) (*GitPullResult, error)
	GitPush(ctx context.Context, sessionID, repoPath, remote string, force, setUpstream bool) error

	Close() error
}

// FileInfo describes a single file or directory.
type FileInfo struct {
	Path  string
	Size  int64
	Mtime time.Time
	Mode  fs.FileMode
	IsDir bool
}

// SyncResult summarizes a SyncPush or SyncPull operation.
type SyncResult struct {
	FilesSynced      int
	BytesTransferred int64
	Duration         time.Duration
}

// ArchiveResult summarizes an archive/compression operation.
type ArchiveResult struct {
	SizeBytes  int64
	FilesCount int
}

// SkillInfo describes a discovered skill from a SKILL.md file.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// SyncOption configures sync behavior.
type SyncOption func(*SyncConfig)

// SyncConfig holds resolved sync options.
type SyncConfig struct {
	ForceFull  bool
	Excludes   []string
	RemotePath string
}

// WithForceFull skips snapshot caches and diffs against actual disk.
func WithForceFull() SyncOption {
	return func(c *SyncConfig) { c.ForceFull = true }
}

// WithExcludes adds glob patterns to exclude from sync.
func WithExcludes(patterns ...string) SyncOption {
	return func(c *SyncConfig) { c.Excludes = append(c.Excludes, patterns...) }
}

// WithRemotePath sets a sub-path within the workspace root for sync operations.
func WithRemotePath(path string) SyncOption {
	return func(c *SyncConfig) { c.RemotePath = path }
}

// ApplySyncOpts resolves a slice of SyncOption into a SyncConfig.
func ApplySyncOpts(opts []SyncOption) SyncConfig {
	var cfg SyncConfig
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// GitRepo describes a Git repository found within a workspace.
type GitRepo struct {
	Path        string `json:"path"`
	Branch      string `json:"branch"`
	RemoteURL   string `json:"remote_url"`
	HasChanges  bool   `json:"has_changes"`
	Ahead       int    `json:"ahead"`
	Behind      int    `json:"behind"`
	HasUpstream bool   `json:"has_upstream"`
}

// GitChangedFile represents a file with uncommitted changes.
type GitChangedFile struct {
	Path           string `json:"path"`
	IndexStatus    string `json:"index_status"`
	WorktreeStatus string `json:"worktree_status"`
	OldPath        string `json:"old_path,omitempty"`
}

// GitStatusResult aggregates the status of a Git repository.
type GitStatusResult struct {
	Branch          string           `json:"branch"`
	Files           []GitChangedFile `json:"files"`
	Ahead           int              `json:"ahead"`
	Behind          int              `json:"behind"`
	TotalInsertions int              `json:"total_insertions"`
	TotalDeletions  int              `json:"total_deletions"`
	IsDetached      bool             `json:"is_detached"`
	IsEmpty         bool             `json:"is_empty"`
	RemoteName      string           `json:"remote_name"`
	RemoteURL       string           `json:"remote_url"`
	UpstreamBranch  string           `json:"upstream_branch"`
	HasUpstream     bool             `json:"has_upstream"`
}

// GitFileDiffResult contains diff content for a single file.
type GitFileDiffResult struct {
	Original   string `json:"original"`
	Modified   string `json:"modified"`
	Language   string `json:"language"`
	IsBinary   bool   `json:"is_binary"`
	IsNew      bool   `json:"is_new"`
	IsDeleted  bool   `json:"is_deleted"`
	IsTooLarge bool   `json:"is_too_large"`
}

// GitCommitResult contains the result of a commit operation.
type GitCommitResult struct {
	CommitHash string `json:"commit_hash"`
	Message    string `json:"message"`
}

// GitCredentialEntry represents an HTTPS credential (token not exposed).
type GitCredentialEntry struct {
	Host     string `json:"host"`
	Username string `json:"username"`
}

// GitSSHKeyEntry represents an imported SSH key (private key not exposed).
type GitSSHKeyEntry struct {
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}

// GitPullResult contains the result of a git pull operation.
type GitPullResult struct {
	HasConflicts bool `json:"has_conflicts"`
}
