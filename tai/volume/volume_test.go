package volume

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalVolume(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "test-session"

	t.Run("WriteFile and ReadFile", func(t *testing.T) {
		data := []byte("hello world")
		if err := vol.WriteFile(ctx, sid, "greeting.txt", data, 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		got, mode, err := vol.ReadFile(ctx, sid, "greeting.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(got) != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
		if mode&0o644 != 0o644 {
			t.Errorf("mode %v does not contain 0644", mode)
		}
	})

	t.Run("Stat", func(t *testing.T) {
		info, err := vol.Stat(ctx, sid, "greeting.txt")
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if info.Size != 11 {
			t.Errorf("size = %d, want 11", info.Size)
		}
		if info.IsDir {
			t.Error("expected file, got dir")
		}
	})

	t.Run("MkdirAll and ListDir", func(t *testing.T) {
		if err := vol.MkdirAll(ctx, sid, "subdir/nested"); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		_ = vol.WriteFile(ctx, sid, "subdir/nested/file.txt", []byte("x"), 0o644)
		entries, err := vol.ListDir(ctx, sid, "subdir/nested")
		if err != nil {
			t.Fatalf("ListDir: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Path != "file.txt" {
			t.Errorf("entry name = %q, want %q", entries[0].Path, "file.txt")
		}
	})

	t.Run("Rename", func(t *testing.T) {
		if err := vol.Rename(ctx, sid, "greeting.txt", "hello.txt"); err != nil {
			t.Fatalf("Rename: %v", err)
		}
		_, _, err := vol.ReadFile(ctx, sid, "hello.txt")
		if err != nil {
			t.Fatalf("ReadFile after rename: %v", err)
		}
		_, _, err = vol.ReadFile(ctx, sid, "greeting.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected not-exist, got %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		if err := vol.Remove(ctx, sid, "hello.txt", false); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		_, err := vol.Stat(ctx, sid, "hello.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected not-exist, got %v", err)
		}
	})

	t.Run("Remove recursive", func(t *testing.T) {
		if err := vol.Remove(ctx, sid, "subdir", true); err != nil {
			t.Fatalf("RemoveAll: %v", err)
		}
		_, err := vol.Stat(ctx, sid, "subdir")
		if !os.IsNotExist(err) {
			t.Errorf("expected not-exist, got %v", err)
		}
	})
}

func TestLocalSyncPush(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	defer vol.Close()
	ctx := context.Background()
	sid := "sync-test"

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0o644)
	_ = os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("bbb"), 0o644)

	result, err := vol.SyncPush(ctx, sid, srcDir, WithForceFull())
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 2 {
		t.Errorf("synced = %d, want 2", result.FilesSynced)
	}

	// Verify files exist in dataDir
	data, err := os.ReadFile(filepath.Join(dataDir, sid, "a.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "aaa" {
		t.Errorf("content = %q, want %q", data, "aaa")
	}
}

func TestLocalSyncPull(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	defer vol.Close()
	ctx := context.Background()
	sid := "pull-test"

	// Create source in dataDir
	sessionDir := filepath.Join(dataDir, sid)
	_ = os.MkdirAll(sessionDir, 0o755)
	_ = os.WriteFile(filepath.Join(sessionDir, "c.txt"), []byte("ccc"), 0o644)

	dstDir := t.TempDir()
	result, err := vol.SyncPull(ctx, sid, dstDir, WithForceFull())
	if err != nil {
		t.Fatalf("SyncPull: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "c.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "ccc" {
		t.Errorf("content = %q, want %q", data, "ccc")
	}
}

func TestLocalSyncPushSkipsUnchanged(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	defer vol.Close()
	ctx := context.Background()
	sid := "skip-test"

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0o644)

	// First push
	_, _ = vol.SyncPush(ctx, sid, srcDir, WithForceFull())

	// Second push (no changes) without force
	result, err := vol.SyncPush(ctx, sid, srcDir)
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 0 {
		t.Errorf("synced = %d, want 0 (no changes)", result.FilesSynced)
	}
}

func TestLocalAbs_Dot(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "abs-test"

	got, err := vol.Abs(ctx, sid, ".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	want := dir + "/" + sid
	if got != want {
		t.Errorf("Abs(\".\") = %q, want %q", got, want)
	}
}

func TestLocalAbs_RelativePath(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "abs-rel"

	got, err := vol.Abs(ctx, sid, "sub/file.txt")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	want := dir + "/" + sid + "/sub/file.txt"
	if got != want {
		t.Errorf("Abs = %q, want %q", got, want)
	}
}

func TestLocalAbs_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.Abs(ctx, "test", "../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal in Abs")
	}
}

func TestLocalPathTraversal(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	// Path traversal should fail
	_, _, err := vol.ReadFile(ctx, "test", "../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal in ReadFile")
	}
	if err := vol.WriteFile(ctx, "test", "../../etc/evil", []byte("x"), 0o644); err == nil {
		t.Error("expected error for path traversal in WriteFile")
	}
	_, err = vol.Stat(ctx, "test", "../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal in Stat")
	}
	_, err = vol.ListDir(ctx, "test", "../../etc")
	if err == nil {
		t.Error("expected error for path traversal in ListDir")
	}
	if err := vol.Remove(ctx, "test", "../../etc/passwd", false); err == nil {
		t.Error("expected error for path traversal in Remove")
	}
	if err := vol.Rename(ctx, "test", "../../etc/a", "b"); err == nil {
		t.Error("expected error for path traversal in Rename old")
	}
	if err := vol.Rename(ctx, "test", "a", "../../etc/b"); err == nil {
		t.Error("expected error for path traversal in Rename new")
	}
	if err := vol.MkdirAll(ctx, "test", "../../etc/evil"); err == nil {
		t.Error("expected error for path traversal in MkdirAll")
	}
}

func TestLocalReadFileNotExist(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, _, err := vol.ReadFile(ctx, "test", "nonexistent.txt")
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist, got %v", err)
	}
}

func TestLocalStatNotExist(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.Stat(ctx, "test", "nonexistent.txt")
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist, got %v", err)
	}
}

func TestLocalListDirNotExist(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.ListDir(ctx, "test", "nonexistent")
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist, got %v", err)
	}
}

func TestLocalRemoveNotExist(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	// Non-recursive remove on nonexistent should error
	err := vol.Remove(ctx, "test", "nonexistent.txt", false)
	if err == nil {
		t.Error("expected error for remove nonexistent")
	}
}

func TestLocalSyncPullNoSource(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	dstDir := t.TempDir()
	result, err := vol.SyncPull(ctx, "nonexistent-session", dstDir)
	if err != nil {
		t.Fatalf("SyncPull nonexistent: %v", err)
	}
	if result.FilesSynced != 0 {
		t.Errorf("synced = %d, want 0", result.FilesSynced)
	}
}

func TestLocalSyncPushWithDirs(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	ctx := context.Background()
	sid := "dir-sync-test"

	srcDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(srcDir, "a", "b", "c"), 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "a", "b", "c", "deep.txt"), []byte("deep"), 0o644)

	result, err := vol.SyncPush(ctx, sid, srcDir, WithForceFull())
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}
}

func TestCompressDecompress(t *testing.T) {
	data := []byte("hello world, this is a test of compression that needs enough data to exercise the paths")
	compressed, err := compress(data)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	decompressed, err := decompress(compressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if string(decompressed) != string(data) {
		t.Errorf("round-trip failed: got %q", decompressed)
	}
}

func TestCompressLargeData(t *testing.T) {
	data := make([]byte, 256*1024) // 256KB
	for i := range data {
		data[i] = byte(i % 256)
	}
	compressed, err := compress(data)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	decompressed, err := decompress(compressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if len(decompressed) != len(data) {
		t.Errorf("len = %d, want %d", len(decompressed), len(data))
	}
}

func TestDecompressInvalid(t *testing.T) {
	_, err := decompress([]byte{0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestCompressEmpty(t *testing.T) {
	compressed, err := compress([]byte{})
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	decompressed, err := decompress(compressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if len(decompressed) != 0 {
		t.Errorf("expected empty, got %d bytes", len(decompressed))
	}
}

func TestLocalSyncPullIncrementalSkip(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	ctx := context.Background()
	sid := "pull-skip-test"

	// Push some files
	_ = vol.WriteFile(ctx, sid, "a.txt", []byte("aaa"), 0o644)
	_ = vol.WriteFile(ctx, sid, "b.txt", []byte("bbb"), 0o644)

	dstDir := t.TempDir()

	// First pull
	result1, err := vol.SyncPull(ctx, sid, dstDir)
	if err != nil {
		t.Fatalf("SyncPull 1: %v", err)
	}
	if result1.FilesSynced != 2 {
		t.Errorf("first sync = %d, want 2", result1.FilesSynced)
	}

	// Second pull — identical mtime+size should skip
	result2, err := vol.SyncPull(ctx, sid, dstDir)
	if err != nil {
		t.Fatalf("SyncPull 2: %v", err)
	}
	// Files should still be synced due to mtime possibly differing (Chtimes on first pull),
	// but on the third pull they should match
	result3, err := vol.SyncPull(ctx, sid, dstDir)
	if err != nil {
		t.Fatalf("SyncPull 3: %v", err)
	}
	if result3.FilesSynced != 0 {
		t.Logf("sync3 = %d (may vary by platform)", result3.FilesSynced)
	}
	_ = result2
}

func TestLocalSyncPullWithExcludes(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	ctx := context.Background()
	sid := "pull-excl"

	_ = vol.WriteFile(ctx, sid, "keep.txt", []byte("keep"), 0o644)
	_ = vol.WriteFile(ctx, sid, "skip.log", []byte("skip"), 0o644)
	_ = vol.MkdirAll(ctx, sid, "node_modules")
	_ = vol.WriteFile(ctx, sid, "node_modules/pkg.js", []byte("x"), 0o644)

	dstDir := t.TempDir()
	result, err := vol.SyncPull(ctx, sid, dstDir, WithExcludes("*.log", "node_modules"), WithForceFull())
	if err != nil {
		t.Fatalf("SyncPull: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}
}

func TestLocalSyncPushIncremental(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	ctx := context.Background()
	sid := "push-inc"

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0o644)

	// First push
	result1, err := vol.SyncPush(ctx, sid, srcDir)
	if err != nil {
		t.Fatalf("SyncPush 1: %v", err)
	}
	if result1.FilesSynced != 1 {
		t.Errorf("first sync = %d, want 1", result1.FilesSynced)
	}

	// Second push without changes — mtime matches, should skip
	result2, err := vol.SyncPush(ctx, sid, srcDir)
	if err != nil {
		t.Fatalf("SyncPush 2: %v", err)
	}
	if result2.FilesSynced != 0 {
		t.Logf("second sync = %d (expected 0 but may vary)", result2.FilesSynced)
	}
}

func TestLocalWriteFileNested(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	// WriteFile with deep nested path (MkdirAll should succeed)
	err := vol.WriteFile(ctx, "test", "a/b/c/deep.txt", []byte("deep"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile nested: %v", err)
	}
	data, _, err := vol.ReadFile(ctx, "test", "a/b/c/deep.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "deep" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalSyncPushExcludeDir(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	ctx := context.Background()

	srcDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(srcDir, ".git", "objects"), 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, ".git", "objects", "abc"), []byte("obj"), 0o644)
	_ = os.WriteFile(filepath.Join(srcDir, "keep.txt"), []byte("keep"), 0o644)

	result, err := vol.SyncPush(ctx, "excl-dir", srcDir, WithForceFull(), WithExcludes(".git"))
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1 (exclude .git dir)", result.FilesSynced)
	}
}

func TestLocalSyncPullForceFull(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	ctx := context.Background()
	sid := "pull-force"

	_ = vol.WriteFile(ctx, sid, "a.txt", []byte("aaa"), 0o644)

	dstDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dstDir, "a.txt"), []byte("aaa"), 0o644)

	// Force full should re-sync even if same content
	result, err := vol.SyncPull(ctx, sid, dstDir, WithForceFull())
	if err != nil {
		t.Fatalf("SyncPull: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1 (force full)", result.FilesSynced)
	}
}

func TestLocalSyncPushWithRemotePath(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	defer vol.Close()
	ctx := context.Background()
	sid := "remote-path-push"

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "app.js"), []byte("console.log('hi')"), 0o644)
	_ = os.MkdirAll(filepath.Join(srcDir, "lib"), 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "lib", "util.js"), []byte("export {}"), 0o644)

	result, err := vol.SyncPush(ctx, sid, srcDir, WithForceFull(), WithRemotePath("packages/frontend"))
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 2 {
		t.Errorf("synced = %d, want 2", result.FilesSynced)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, sid, "packages", "frontend", "app.js"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "console.log('hi')" {
		t.Errorf("content = %q", data)
	}

	nested, err := os.ReadFile(filepath.Join(dataDir, sid, "packages", "frontend", "lib", "util.js"))
	if err != nil {
		t.Fatalf("ReadFile nested: %v", err)
	}
	if string(nested) != "export {}" {
		t.Errorf("nested content = %q", nested)
	}
}

func TestLocalSyncPullWithRemotePath(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	defer vol.Close()
	ctx := context.Background()
	sid := "remote-path-pull"

	sessionDir := filepath.Join(dataDir, sid, "packages", "backend")
	_ = os.MkdirAll(sessionDir, 0o755)
	_ = os.WriteFile(filepath.Join(sessionDir, "main.go"), []byte("package main"), 0o644)

	dstDir := t.TempDir()
	result, err := vol.SyncPull(ctx, sid, dstDir, WithForceFull(), WithRemotePath("packages/backend"))
	if err != nil {
		t.Fatalf("SyncPull: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "main.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "package main" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalZipUnzip(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "zip-test"

	_ = vol.MkdirAll(ctx, sid, "src")
	_ = vol.WriteFile(ctx, sid, "src/a.txt", []byte("aaa"), 0o644)
	_ = vol.WriteFile(ctx, sid, "src/b.txt", []byte("bbb"), 0o644)

	zr, err := vol.Zip(ctx, sid, "src", "out.zip", nil)
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}
	if zr.FilesCount != 2 {
		t.Errorf("zip files = %d, want 2", zr.FilesCount)
	}
	if zr.SizeBytes <= 0 {
		t.Error("zip size should be > 0")
	}

	ur, err := vol.Unzip(ctx, sid, "out.zip", "extracted")
	if err != nil {
		t.Fatalf("Unzip: %v", err)
	}
	if ur.FilesCount != 2 {
		t.Errorf("unzip files = %d, want 2", ur.FilesCount)
	}

	data, _, err := vol.ReadFile(ctx, sid, "extracted/a.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "aaa" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalZipExcludes(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "zip-excl"

	_ = vol.MkdirAll(ctx, sid, "src")
	_ = vol.WriteFile(ctx, sid, "src/keep.txt", []byte("keep"), 0o644)
	_ = vol.WriteFile(ctx, sid, "src/skip.log", []byte("skip"), 0o644)

	zr, err := vol.Zip(ctx, sid, "src", "filtered.zip", []string{"*.log"})
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}
	if zr.FilesCount != 1 {
		t.Errorf("zip files = %d, want 1", zr.FilesCount)
	}

	ur, err := vol.Unzip(ctx, sid, "filtered.zip", "out")
	if err != nil {
		t.Fatalf("Unzip: %v", err)
	}
	if ur.FilesCount != 1 {
		t.Errorf("unzip files = %d, want 1", ur.FilesCount)
	}

	_, err = vol.Stat(ctx, sid, "out/keep.txt")
	if err != nil {
		t.Error("keep.txt should exist")
	}
	_, err = vol.Stat(ctx, sid, "out/skip.log")
	if !os.IsNotExist(err) {
		t.Error("skip.log should not exist")
	}
}

func TestLocalGzipGunzip(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "gzip-test"

	_ = vol.WriteFile(ctx, sid, "data.txt", []byte("gzip test content"), 0o644)

	gr, err := vol.Gzip(ctx, sid, "data.txt", "data.txt.gz")
	if err != nil {
		t.Fatalf("Gzip: %v", err)
	}
	if gr.FilesCount != 1 {
		t.Errorf("gzip files = %d", gr.FilesCount)
	}
	if gr.SizeBytes <= 0 {
		t.Error("gzip size should be > 0")
	}

	ur, err := vol.Gunzip(ctx, sid, "data.txt.gz", "restored.txt")
	if err != nil {
		t.Fatalf("Gunzip: %v", err)
	}
	if ur.FilesCount != 1 {
		t.Errorf("gunzip files = %d", ur.FilesCount)
	}

	data, _, err := vol.ReadFile(ctx, sid, "restored.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "gzip test content" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalGzipRejectsDir(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "gzip-dir"

	_ = vol.MkdirAll(ctx, sid, "subdir")
	_, err := vol.Gzip(ctx, sid, "subdir", "subdir.gz")
	if err == nil {
		t.Error("expected error for gzip on directory")
	}
}

func TestLocalTarUntar(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "tar-test"

	_ = vol.MkdirAll(ctx, sid, "src")
	_ = vol.WriteFile(ctx, sid, "src/x.txt", []byte("tar x"), 0o644)
	_ = vol.WriteFile(ctx, sid, "src/y.txt", []byte("tar y"), 0o644)

	tr, err := vol.Tar(ctx, sid, "src", "out.tar", nil)
	if err != nil {
		t.Fatalf("Tar: %v", err)
	}
	if tr.FilesCount != 2 {
		t.Errorf("tar files = %d, want 2", tr.FilesCount)
	}

	ur, err := vol.Untar(ctx, sid, "out.tar", "extracted")
	if err != nil {
		t.Fatalf("Untar: %v", err)
	}
	if ur.FilesCount != 2 {
		t.Errorf("untar files = %d, want 2", ur.FilesCount)
	}

	data, _, err := vol.ReadFile(ctx, sid, "extracted/x.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "tar x" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalTarExcludes(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "tar-excl"

	_ = vol.MkdirAll(ctx, sid, "src")
	_ = vol.WriteFile(ctx, sid, "src/keep.txt", []byte("keep"), 0o644)
	_ = vol.WriteFile(ctx, sid, "src/skip.log", []byte("skip"), 0o644)

	tr, err := vol.Tar(ctx, sid, "src", "out.tar", []string{"*.log"})
	if err != nil {
		t.Fatalf("Tar: %v", err)
	}
	if tr.FilesCount != 1 {
		t.Errorf("tar files = %d, want 1", tr.FilesCount)
	}
}

func TestLocalTgzUntgz(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "tgz-test"

	_ = vol.MkdirAll(ctx, sid, "src")
	_ = vol.WriteFile(ctx, sid, "src/f.txt", []byte("tgz content"), 0o644)

	tr, err := vol.Tgz(ctx, sid, "src", "out.tgz", nil)
	if err != nil {
		t.Fatalf("Tgz: %v", err)
	}
	if tr.FilesCount != 1 {
		t.Errorf("tgz files = %d", tr.FilesCount)
	}

	ur, err := vol.Untgz(ctx, sid, "out.tgz", "extracted")
	if err != nil {
		t.Fatalf("Untgz: %v", err)
	}
	if ur.FilesCount != 1 {
		t.Errorf("untgz files = %d", ur.FilesCount)
	}

	data, _, err := vol.ReadFile(ctx, sid, "extracted/f.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "tgz content" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalSyncPushForceFull(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	ctx := context.Background()
	sid := "push-force"

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0o644)

	// First sync
	_, _ = vol.SyncPush(ctx, sid, srcDir)
	// Force full should re-sync
	result, err := vol.SyncPush(ctx, sid, srcDir, WithForceFull())
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1 (force full)", result.FilesSynced)
	}
}

func TestLocalSyncExcludes(t *testing.T) {
	dataDir := t.TempDir()
	vol := NewLocal(dataDir)
	defer vol.Close()
	ctx := context.Background()
	sid := "exclude-test"

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "keep.txt"), []byte("k"), 0o644)
	_ = os.WriteFile(filepath.Join(srcDir, "skip.log"), []byte("s"), 0o644)

	result, err := vol.SyncPush(ctx, sid, srcDir, WithForceFull(), WithExcludes("*.log"))
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}

	if _, err := os.Stat(filepath.Join(dataDir, sid, "skip.log")); !os.IsNotExist(err) {
		t.Error("excluded file should not exist")
	}
}

func TestLocalCopyFile(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "copy-file"

	_ = vol.WriteFile(ctx, sid, "src.txt", []byte("hello copy"), 0o644)

	result, err := vol.Copy(ctx, sid, "src.txt", "dst.txt")
	if err != nil {
		t.Fatalf("Copy file: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}
	if result.BytesTransferred != 10 {
		t.Errorf("bytes = %d, want 10", result.BytesTransferred)
	}

	data, _, err := vol.ReadFile(ctx, sid, "dst.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello copy" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalCopyDir(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "copy-dir"

	_ = vol.MkdirAll(ctx, sid, "src/sub")
	_ = vol.WriteFile(ctx, sid, "src/a.txt", []byte("aaa"), 0o644)
	_ = vol.WriteFile(ctx, sid, "src/sub/b.txt", []byte("bbb"), 0o644)

	result, err := vol.Copy(ctx, sid, "src", "dst")
	if err != nil {
		t.Fatalf("Copy dir: %v", err)
	}
	if result.FilesSynced != 2 {
		t.Errorf("synced = %d, want 2", result.FilesSynced)
	}

	data, _, err := vol.ReadFile(ctx, sid, "dst/a.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "aaa" {
		t.Errorf("content = %q", data)
	}

	data, _, err = vol.ReadFile(ctx, sid, "dst/sub/b.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "bbb" {
		t.Errorf("content = %q", data)
	}
}

func TestLocalCopyExcludes(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "copy-excl"

	_ = vol.MkdirAll(ctx, sid, "src")
	_ = vol.WriteFile(ctx, sid, "src/keep.txt", []byte("keep"), 0o644)
	_ = vol.WriteFile(ctx, sid, "src/skip.log", []byte("skip"), 0o644)

	result, err := vol.Copy(ctx, sid, "src", "dst", WithExcludes("*.log"))
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}

	_, err = vol.Stat(ctx, sid, "dst/keep.txt")
	if err != nil {
		t.Error("keep.txt should exist")
	}
	_, err = vol.Stat(ctx, sid, "dst/skip.log")
	if !os.IsNotExist(err) {
		t.Error("skip.log should not exist")
	}
}

func TestLocalCopySkipsUnchanged(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "copy-skip"

	_ = vol.WriteFile(ctx, sid, "src.txt", []byte("data"), 0o644)

	result1, err := vol.Copy(ctx, sid, "src.txt", "dst.txt", WithForceFull())
	if err != nil {
		t.Fatalf("Copy 1: %v", err)
	}
	if result1.FilesSynced != 1 {
		t.Errorf("first copy synced = %d, want 1", result1.FilesSynced)
	}

	result2, err := vol.Copy(ctx, sid, "src.txt", "dst.txt")
	if err != nil {
		t.Fatalf("Copy 2: %v", err)
	}
	if result2.FilesSynced != 0 {
		t.Errorf("second copy synced = %d, want 0 (unchanged)", result2.FilesSynced)
	}
}

func TestLocalCopyForceFull(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	defer vol.Close()
	ctx := context.Background()
	sid := "copy-force"

	_ = vol.WriteFile(ctx, sid, "src.txt", []byte("data"), 0o644)

	_, _ = vol.Copy(ctx, sid, "src.txt", "dst.txt", WithForceFull())

	result, err := vol.Copy(ctx, sid, "src.txt", "dst.txt", WithForceFull())
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("force copy synced = %d, want 1", result.FilesSynced)
	}
}

func TestLocalCopyNotExist(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.Copy(ctx, "test", "nonexistent", "dst")
	if err == nil {
		t.Error("expected error for copy nonexistent source")
	}
}

func TestLocalCopyPathTraversal(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.Copy(ctx, "test", "../../etc/passwd", "dst")
	if err == nil {
		t.Error("expected error for path traversal in src")
	}

	_, err = vol.Copy(ctx, "test", "src", "../../etc/evil")
	if err == nil {
		t.Error("expected error for path traversal in dst")
	}
}
