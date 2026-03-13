package volume

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
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

func (l *localStorage) Abs(_ context.Context, sessionID, path string) (string, error) {
	return l.abs(sessionID, path)
}

// Copy duplicates src to dst within the same workspace session.
// Supports single files and directories (recursive). Uses excludes from SyncOption
// and forceFull to overwrite even when mtime+size match.
func (l *localStorage) Copy(_ context.Context, sessionID, src, dst string, opts ...SyncOption) (*SyncResult, error) {
	start := time.Now()
	cfg := ApplySyncOpts(opts)

	srcAbs, err := l.abs(sessionID, src)
	if err != nil {
		return nil, err
	}
	dstAbs, err := l.abs(sessionID, dst)
	if err != nil {
		return nil, err
	}

	srcInfo, err := os.Stat(srcAbs)
	if err != nil {
		return nil, err
	}

	if !srcInfo.IsDir() {
		n, err := l.copyFile(srcAbs, dstAbs, srcInfo, cfg.ForceFull)
		if err != nil {
			return nil, err
		}
		synced := 0
		if n > 0 {
			synced = 1
		}
		return &SyncResult{
			FilesSynced:      synced,
			BytesTransferred: n,
			Duration:         time.Since(start),
		}, nil
	}

	var synced int
	var transferred int64
	err = filepath.WalkDir(srcAbs, func(abs string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		rel, _ := filepath.Rel(srcAbs, abs)
		if rel == "." {
			return os.MkdirAll(dstAbs, 0o755)
		}

		if isExcluded(rel, d.IsDir(), cfg.Excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(dstAbs, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		n, err := l.copyFile(abs, target, info, cfg.ForceFull)
		if err != nil {
			return err
		}
		if n > 0 {
			synced++
			transferred += n
		}
		return nil
	})

	return &SyncResult{
		FilesSynced:      synced,
		BytesTransferred: transferred,
		Duration:         time.Since(start),
	}, err
}

func (l *localStorage) copyFile(srcAbs, dstAbs string, srcInfo os.FileInfo, force bool) (int64, error) {
	if !force {
		if dstInfo, e := os.Stat(dstAbs); e == nil {
			if dstInfo.Size() == srcInfo.Size() && dstInfo.ModTime().Equal(srcInfo.ModTime()) {
				return 0, nil
			}
		}
	}

	data, err := os.ReadFile(srcAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(dstAbs, data, srcInfo.Mode()); err != nil {
		return 0, err
	}
	_ = os.Chtimes(dstAbs, srcInfo.ModTime(), srcInfo.ModTime())
	return int64(len(data)), nil
}

// SyncPush copies changed files from localDir to dataDir/{sessionID}/.
// Uses mtime+size to detect changes. Files that vanish during sync are skipped.
func (l *localStorage) SyncPush(_ context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error) {
	start := time.Now()
	cfg := ApplySyncOpts(opts)
	dst := l.root(sessionID)
	if cfg.RemotePath != "" {
		dst = filepath.Join(dst, filepath.Clean(cfg.RemotePath))
	}
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

		if isExcluded(rel, d.IsDir(), cfg.Excludes) {
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

		if !cfg.ForceFull {
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
	cfg := ApplySyncOpts(opts)
	src := l.root(sessionID)
	if cfg.RemotePath != "" {
		src = filepath.Join(src, filepath.Clean(cfg.RemotePath))
	}
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

		if isExcluded(rel, d.IsDir(), cfg.Excludes) {
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

		if !cfg.ForceFull {
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

func (l *localStorage) Zip(_ context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error) {
	srcAbs, err := l.abs(sessionID, src)
	if err != nil {
		return nil, err
	}
	dstAbs, err := l.abs(sessionID, dst)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return nil, err
	}
	out, err := os.Create(dstAbs)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	w := zip.NewWriter(out)
	defer w.Close()
	var count int
	if err := filepath.WalkDir(srcAbs, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcAbs, abs)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if isExcluded(rel, d.IsDir(), excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			_, e := w.Create(rel + "/")
			return e
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(abs)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(writer, f)
		if err == nil {
			count++
		}
		return err
	}); err != nil {
		return nil, err
	}
	w.Close()
	out.Close()
	fi, _ := os.Stat(dstAbs)
	return &ArchiveResult{SizeBytes: fi.Size(), FilesCount: count}, nil
}

func (l *localStorage) Unzip(_ context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	srcAbs, err := l.abs(sessionID, src)
	if err != nil {
		return nil, err
	}
	dstAbs, err := l.abs(sessionID, dst)
	if err != nil {
		return nil, err
	}
	r, err := zip.OpenReader(srcAbs)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err := os.MkdirAll(dstAbs, 0o755); err != nil {
		return nil, err
	}
	var count int
	var totalSize int64
	for _, f := range r.File {
		target := filepath.Join(dstAbs, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, dstAbs+string(filepath.Separator)) && target != dstAbs {
			return nil, fmt.Errorf("zip slip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return nil, err
		}
		n, err := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return nil, err
		}
		totalSize += n
		count++
	}
	return &ArchiveResult{SizeBytes: totalSize, FilesCount: count}, nil
}

func (l *localStorage) Gzip(_ context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	srcAbs, err := l.abs(sessionID, src)
	if err != nil {
		return nil, err
	}
	dstAbs, err := l.abs(sessionID, dst)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(srcAbs)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("gzip requires a file, not directory")
	}
	in, err := os.Open(srcAbs)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return nil, err
	}
	out, err := os.Create(dstAbs)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	w := gzip.NewWriter(out)
	w.Name = filepath.Base(srcAbs)
	if _, err := io.Copy(w, in); err != nil {
		w.Close()
		return nil, err
	}
	w.Close()
	out.Close()
	fi, _ := os.Stat(dstAbs)
	return &ArchiveResult{SizeBytes: fi.Size(), FilesCount: 1}, nil
}

func (l *localStorage) Gunzip(_ context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	srcAbs, err := l.abs(sessionID, src)
	if err != nil {
		return nil, err
	}
	dstAbs, err := l.abs(sessionID, dst)
	if err != nil {
		return nil, err
	}
	in, err := os.Open(srcAbs)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	r, err := gzip.NewReader(in)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return nil, err
	}
	out, err := os.Create(dstAbs)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	n, err := io.Copy(out, r)
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: n, FilesCount: 1}, nil
}

func (l *localStorage) Tar(_ context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error) {
	return l.tarImpl(sessionID, src, dst, excludes, false)
}

func (l *localStorage) Tgz(_ context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error) {
	return l.tarImpl(sessionID, src, dst, excludes, true)
}

func (l *localStorage) tarImpl(sessionID, src, dst string, excludes []string, useGzip bool) (*ArchiveResult, error) {
	srcAbs, err := l.abs(sessionID, src)
	if err != nil {
		return nil, err
	}
	dstAbs, err := l.abs(sessionID, dst)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return nil, err
	}
	out, err := os.Create(dstAbs)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	var tw *tar.Writer
	var gw *gzip.Writer
	if useGzip {
		gw = gzip.NewWriter(out)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(out)
	}
	defer tw.Close()
	var count int
	if err := filepath.WalkDir(srcAbs, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcAbs, abs)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if isExcluded(rel, d.IsDir(), excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(abs)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		if err == nil {
			count++
		}
		return err
	}); err != nil {
		return nil, err
	}
	tw.Close()
	if gw != nil {
		gw.Close()
	}
	out.Close()
	fi, _ := os.Stat(dstAbs)
	return &ArchiveResult{SizeBytes: fi.Size(), FilesCount: count}, nil
}

func (l *localStorage) Untar(_ context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	return l.untarImpl(sessionID, src, dst, false)
}

func (l *localStorage) Untgz(_ context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	return l.untarImpl(sessionID, src, dst, true)
}

func (l *localStorage) untarImpl(sessionID, src, dst string, useGzip bool) (*ArchiveResult, error) {
	srcAbs, err := l.abs(sessionID, src)
	if err != nil {
		return nil, err
	}
	dstAbs, err := l.abs(sessionID, dst)
	if err != nil {
		return nil, err
	}
	in, err := os.Open(srcAbs)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	var reader io.Reader = in
	if useGzip {
		gr, err := gzip.NewReader(in)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		reader = gr
	}
	tr := tar.NewReader(reader)
	if err := os.MkdirAll(dstAbs, 0o755); err != nil {
		return nil, err
	}
	var count int
	var totalSize int64
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		target := filepath.Join(dstAbs, filepath.FromSlash(header.Name))
		if !strings.HasPrefix(target, dstAbs+string(filepath.Separator)) && target != dstAbs {
			return nil, fmt.Errorf("tar slip: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
			n, err := io.Copy(out, tr)
			out.Close()
			if err != nil {
				return nil, err
			}
			totalSize += n
			count++
		}
	}
	return &ArchiveResult{SizeBytes: totalSize, FilesCount: count}, nil
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
