package workspace

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/tai/volume"
)

// Copy implements the FS.Copy method with 4-way dispatch:
//
//	ws   -> ws   : Volume.Copy (server-side for remote, local copy for local)
//	host -> ws   : Volume.SyncPush
//	ws   -> host : Volume.SyncPull
//	host -> host : os-level recursive copy
func (w *workspaceFS) Copy(src, dst string, opts ...volume.SyncOption) (*volume.SyncResult, error) {
	srcURI := parseHostURI(src)
	dstURI := parseHostURI(dst)
	ctx := context.Background()

	switch {
	case !srcURI.IsHost && !dstURI.IsHost:
		return w.vol.Copy(ctx, w.session, srcURI.Path, dstURI.Path, opts...)

	case srcURI.IsHost && !dstURI.IsHost:
		hostPath, err := resolveAbsHostPath(srcURI)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(hostPath)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, w.pushSingleFile(ctx, hostPath, dstURI.Path, info)
		}
		pushOpts := append(sliceClone(opts), volume.WithRemotePath(dstURI.Path))
		return w.vol.SyncPush(ctx, w.session, hostPath, pushOpts...)

	case !srcURI.IsHost && dstURI.IsHost:
		hostPath, err := resolveAbsHostPath(dstURI)
		if err != nil {
			return nil, err
		}
		srcInfo, statErr := w.vol.Stat(ctx, w.session, srcURI.Path)
		if statErr != nil {
			return nil, statErr
		}
		if !srcInfo.IsDir {
			return nil, w.pullSingleFile(ctx, srcURI.Path, hostPath)
		}
		pullOpts := append(sliceClone(opts), volume.WithRemotePath(srcURI.Path))
		return w.vol.SyncPull(ctx, w.session, hostPath, pullOpts...)

	default:
		srcPath, err := resolveAbsHostPath(srcURI)
		if err != nil {
			return nil, err
		}
		dstPath, err := resolveAbsHostPath(dstURI)
		if err != nil {
			return nil, err
		}
		cfg := volume.ApplySyncOpts(opts)
		return nil, copyLocalToLocal(srcPath, dstPath, cfg.Excludes)
	}
}

// pushSingleFile reads a host file and writes it into the workspace at dstPath.
func (w *workspaceFS) pushSingleFile(ctx context.Context, hostPath, dstPath string, info os.FileInfo) error {
	data, err := os.ReadFile(hostPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dstPath)
	if dir != "" && dir != "." {
		if err := w.vol.MkdirAll(ctx, w.session, dir); err != nil {
			return err
		}
	}
	perm := info.Mode()
	if perm == 0 {
		perm = 0o644
	}
	return w.vol.WriteFile(ctx, w.session, dstPath, data, perm)
}

// pullSingleFile reads a workspace file and writes it to hostPath.
func (w *workspaceFS) pullSingleFile(ctx context.Context, srcPath, hostPath string) error {
	data, perm, err := w.vol.ReadFile(ctx, w.session, srcPath)
	if err != nil {
		return err
	}
	if perm == 0 {
		perm = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(hostPath, data, perm)
}

func copyLocalToLocal(src, dst string, excludes []string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, info.Mode())
	}

	return filepath.WalkDir(src, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, abs)
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		for _, p := range excludes {
			if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		fi, _ := d.Info()
		perm := os.FileMode(0o644)
		if fi != nil {
			perm = fi.Mode()
		}
		return os.WriteFile(target, data, perm)
	})
}

func sliceClone(opts []volume.SyncOption) []volume.SyncOption {
	cp := make([]volume.SyncOption, len(opts))
	copy(cp, opts)
	return cp
}
