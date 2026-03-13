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
