package volume

import (
	"context"
	"io/fs"
	"os"
	"time"
)

// Volume provides filesystem IO and directory synchronization.
// Remote connects to Tai gRPC :9100; Local operates directly on disk.
type Volume interface {
	ReadFile(ctx context.Context, sessionID, path string) ([]byte, os.FileMode, error)
	WriteFile(ctx context.Context, sessionID, path string, data []byte, perm os.FileMode) error
	Stat(ctx context.Context, sessionID, path string) (*FileInfo, error)
	ListDir(ctx context.Context, sessionID, path string) ([]FileInfo, error)
	Remove(ctx context.Context, sessionID, path string, recursive bool) error
	Rename(ctx context.Context, sessionID, oldPath, newPath string) error
	MkdirAll(ctx context.Context, sessionID, path string) error

	SyncPush(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error)
	SyncPull(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error)

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

// SyncOption configures sync behavior.
type SyncOption func(*syncConfig)

type syncConfig struct {
	forceFull bool
	excludes  []string
}

// WithForceFull skips snapshot caches and diffs against actual disk.
func WithForceFull() SyncOption {
	return func(c *syncConfig) { c.forceFull = true }
}

// WithExcludes adds glob patterns to exclude from sync.
func WithExcludes(patterns ...string) SyncOption {
	return func(c *syncConfig) { c.excludes = append(c.excludes, patterns...) }
}

func applySyncOpts(opts []SyncOption) syncConfig {
	var cfg syncConfig
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
