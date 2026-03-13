package workspace

import (
	"context"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/yaoapp/yao/tai/volume"
)

// FS extends Go's fs.FS with write operations.
// Backed by volume.Volume — works for both Remote and Local transparently.
type FS interface {
	fs.FS
	fs.StatFS
	fs.ReadFileFS
	fs.ReadDirFS
	io.Closer

	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
	RemoveAll(name string) error
	Rename(oldname, newname string) error
	MkdirAll(name string, perm os.FileMode) error

	// Copy copies files between workspace paths and/or host paths.
	// Host paths use "local:///" (absolute system path) or "tmp:///" (os.TempDir-relative).
	// ws↔ws uses Volume.Copy (server-side for remote volumes, avoiding 2N network round-trips).
	// Returns non-nil *SyncResult for host↔workspace and ws↔ws transfers; nil for host↔host.
	Copy(src, dst string, opts ...volume.SyncOption) (*volume.SyncResult, error)

	// GetRoot returns the absolute path of this workspace's root directory on the host filesystem.
	GetRoot() (string, error)
}

// New creates an FS backed by the given Volume for the specified session.
func New(vol volume.Volume, sessionID string) FS {
	return &workspaceFS{vol: vol, session: sessionID}
}

type workspaceFS struct {
	vol     volume.Volume
	session string
}

func (w *workspaceFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	ctx := context.Background()
	info, err := w.vol.Stat(ctx, w.session, name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	if info.IsDir {
		return &dirFile{w: w, name: name, info: info}, nil
	}
	data, _, err := w.vol.ReadFile(ctx, w.session, name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &memFile{name: name, info: info, data: data}, nil
}

func (w *workspaceFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	info, err := w.vol.Stat(context.Background(), w.session, name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	return toFSInfo(name, info), nil
}

func (w *workspaceFS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	data, _, err := w.vol.ReadFile(context.Background(), w.session, name)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: err}
	}
	return data, nil
}

func (w *workspaceFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	entries, err := w.vol.ListDir(context.Background(), w.session, name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}
	result := make([]fs.DirEntry, 0, len(entries))
	for i := range entries {
		result = append(result, &dirEntry{info: &entries[i]})
	}
	return result, nil
}

func (w *workspaceFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return w.vol.WriteFile(context.Background(), w.session, name, data, perm)
}

func (w *workspaceFS) Remove(name string) error {
	return w.vol.Remove(context.Background(), w.session, name, false)
}

func (w *workspaceFS) RemoveAll(name string) error {
	return w.vol.Remove(context.Background(), w.session, name, true)
}

func (w *workspaceFS) Rename(oldname, newname string) error {
	return w.vol.Rename(context.Background(), w.session, oldname, newname)
}

func (w *workspaceFS) MkdirAll(name string, _ os.FileMode) error {
	return w.vol.MkdirAll(context.Background(), w.session, name)
}

func (w *workspaceFS) GetRoot() (string, error) {
	return w.vol.Abs(context.Background(), w.session, ".")
}

func (w *workspaceFS) Close() error { return nil }

// --- fs.FileInfo adapter ---

type fileInfoAdapter struct {
	name  string
	size  int64
	mode  fs.FileMode
	mtime time.Time
	isDir bool
}

func toFSInfo(name string, vi *volume.FileInfo) *fileInfoAdapter {
	base := name
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		base = name[idx+1:]
	}
	if base == "" {
		base = "."
	}
	return &fileInfoAdapter{
		name:  base,
		size:  vi.Size,
		mode:  vi.Mode,
		mtime: vi.Mtime,
		isDir: vi.IsDir,
	}
}

func (f *fileInfoAdapter) Name() string       { return f.name }
func (f *fileInfoAdapter) Size() int64        { return f.size }
func (f *fileInfoAdapter) Mode() fs.FileMode  { return f.mode }
func (f *fileInfoAdapter) ModTime() time.Time { return f.mtime }
func (f *fileInfoAdapter) IsDir() bool        { return f.isDir }
func (f *fileInfoAdapter) Sys() any           { return nil }

// --- fs.DirEntry adapter ---

type dirEntry struct {
	info *volume.FileInfo
}

func (d *dirEntry) Name() string               { return d.info.Path }
func (d *dirEntry) IsDir() bool                { return d.info.IsDir }
func (d *dirEntry) Type() fs.FileMode          { return d.info.Mode.Type() }
func (d *dirEntry) Info() (fs.FileInfo, error) { return toFSInfo(d.info.Path, d.info), nil }

// --- in-memory file (for Open on regular files) ---

type memFile struct {
	name   string
	info   *volume.FileInfo
	data   []byte
	offset int
}

func (f *memFile) Stat() (fs.FileInfo, error) { return toFSInfo(f.name, f.info), nil }
func (f *memFile) Read(b []byte) (int, error) {
	if f.offset >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.offset:])
	f.offset += n
	return n, nil
}
func (f *memFile) Close() error { return nil }

// --- directory file (for Open on directories) ---

type dirFile struct {
	w    *workspaceFS
	name string
	info *volume.FileInfo
}

func (d *dirFile) Stat() (fs.FileInfo, error) { return toFSInfo(d.name, d.info), nil }
func (d *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: fs.ErrInvalid}
}
func (d *dirFile) Close() error { return nil }
