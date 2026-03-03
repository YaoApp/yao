package volume

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type localStorage struct {
	dataDir string
}

// NewLocal creates a Volume backed by direct disk IO under dataDir/{sessionID}/.
func NewLocal(dataDir string) Volume {
	return &localStorage{dataDir: dataDir}
}

func (l *localStorage) root(sessionID string) string {
	return filepath.Join(l.dataDir, sessionID)
}

func (l *localStorage) abs(sessionID, path string) (string, error) {
	base := l.root(sessionID)
	resolved := filepath.Join(base, filepath.Clean(path))
	if !strings.HasPrefix(resolved, base+string(filepath.Separator)) && resolved != base {
		return "", os.ErrPermission
	}
	return resolved, nil
}

func (l *localStorage) ReadFile(_ context.Context, sessionID, path string) ([]byte, os.FileMode, error) {
	abs, err := l.abs(sessionID, path)
	if err != nil {
		return nil, 0, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, 0, err
	}
	return data, info.Mode(), nil
}

func (l *localStorage) WriteFile(_ context.Context, sessionID, path string, data []byte, perm os.FileMode) error {
	abs, err := l.abs(sessionID, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, perm)
}

func (l *localStorage) Stat(_ context.Context, sessionID, path string) (*FileInfo, error) {
	abs, err := l.abs(sessionID, path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Path:  path,
		Size:  info.Size(),
		Mtime: info.ModTime(),
		Mode:  info.Mode(),
		IsDir: info.IsDir(),
	}, nil
}

func (l *localStorage) ListDir(_ context.Context, sessionID, path string) ([]FileInfo, error) {
	abs, err := l.abs(sessionID, path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	var result []FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, FileInfo{
			Path:  e.Name(),
			Size:  info.Size(),
			Mtime: info.ModTime(),
			Mode:  info.Mode(),
			IsDir: e.IsDir(),
		})
	}
	return result, nil
}

func (l *localStorage) Remove(_ context.Context, sessionID, path string, recursive bool) error {
	abs, err := l.abs(sessionID, path)
	if err != nil {
		return err
	}
	if recursive {
		return os.RemoveAll(abs)
	}
	return os.Remove(abs)
}

func (l *localStorage) Rename(_ context.Context, sessionID, oldPath, newPath string) error {
	oldAbs, err := l.abs(sessionID, oldPath)
	if err != nil {
		return err
	}
	newAbs, err := l.abs(sessionID, newPath)
	if err != nil {
		return err
	}
	return os.Rename(oldAbs, newAbs)
}

func (l *localStorage) MkdirAll(_ context.Context, sessionID, path string) error {
	abs, err := l.abs(sessionID, path)
	if err != nil {
		return err
	}
	return os.MkdirAll(abs, 0o755)
}

// SyncPush copies changed files from localDir to dataDir/{sessionID}/.
// Uses mtime+size to detect changes. Files that vanish during sync are skipped.
func (l *localStorage) SyncPush(_ context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error) {
	start := time.Now()
	cfg := applySyncOpts(opts)
	dst := l.root(sessionID)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, err
	}

	var synced int
	var transferred int64

	err := filepath.WalkDir(localDir, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(localDir, abs)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if isExcluded(rel, d.IsDir(), cfg.excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(dst, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		srcInfo, err := d.Info()
		if err != nil {
			return nil // file vanished between readdir and stat; skip
		}

		if !cfg.forceFull {
			if dstInfo, e := os.Stat(target); e == nil {
				if dstInfo.Size() == srcInfo.Size() && dstInfo.ModTime().Equal(srcInfo.ModTime()) {
					return nil
				}
			}
		}

		data, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // file vanished between stat and read; skip
			}
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, srcInfo.Mode()); err != nil {
			return err
		}
		_ = os.Chtimes(target, srcInfo.ModTime(), srcInfo.ModTime())
		synced++
		transferred += srcInfo.Size()
		return nil
	})

	return &SyncResult{
		FilesSynced:      synced,
		BytesTransferred: transferred,
		Duration:         time.Since(start),
	}, err
}

// SyncPull copies changed files from dataDir/{sessionID}/ to localDir.
// Files that vanish during sync are skipped.
func (l *localStorage) SyncPull(_ context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error) {
	start := time.Now()
	cfg := applySyncOpts(opts)
	src := l.root(sessionID)
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return nil, err
	}

	var synced int
	var transferred int64

	err := filepath.WalkDir(src, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(src, abs)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if isExcluded(rel, d.IsDir(), cfg.excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(localDir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		srcInfo, err := d.Info()
		if err != nil {
			return nil // file vanished between readdir and stat; skip
		}

		if !cfg.forceFull {
			if dstInfo, e := os.Stat(target); e == nil {
				if dstInfo.Size() == srcInfo.Size() && dstInfo.ModTime().Equal(srcInfo.ModTime()) {
					return nil
				}
			}
		}

		data, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // file vanished between stat and read; skip
			}
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, srcInfo.Mode()); err != nil {
			return err
		}
		_ = os.Chtimes(target, srcInfo.ModTime(), srcInfo.ModTime())
		synced++
		transferred += srcInfo.Size()
		return nil
	})

	return &SyncResult{
		FilesSynced:      synced,
		BytesTransferred: transferred,
		Duration:         time.Since(start),
	}, err
}

func (l *localStorage) Close() error { return nil }

func isExcluded(rel string, isDir bool, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
			return true
		}
	}
	return false
}
