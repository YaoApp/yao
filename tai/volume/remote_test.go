package volume_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	code := m.Run()
	testprepare.Cleanup()
	os.Exit(code)
}

func dialRemoteTai(t *testing.T) *grpc.ClientConn {
	t.Helper()
	testprepare.PrepareSandbox(t)

	reg := registry.Global()
	if reg == nil {
		t.Fatal("dialRemoteTai: registry not initialized")
	}
	for _, n := range reg.List() {
		if n.Mode == "local" || n.Status != "online" || n.Ports.GRPC == 0 {
			continue
		}
		addr := fmt.Sprintf("127.0.0.1:%d", n.Ports.GRPC)
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("dialRemoteTai: grpc.NewClient %s: %v", addr, err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	t.Fatal("dialRemoteTai: no online tunnel node with gRPC port found")
	return nil
}

func TestRemoteVolume(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	defer vol.Close()
	ctx := context.Background()
	sid := "sdk-remote-test"

	t.Run("MkdirAll", func(t *testing.T) {
		if err := vol.MkdirAll(ctx, sid, "sub/dir"); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	})

	t.Run("WriteFile and ReadFile", func(t *testing.T) {
		data := []byte("remote test content")
		if err := vol.WriteFile(ctx, sid, "test.txt", data, 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		got, mode, err := vol.ReadFile(ctx, sid, "test.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(got) != "remote test content" {
			t.Errorf("got %q", got)
		}
		if mode == 0 {
			t.Error("mode should be nonzero")
		}
	})

	t.Run("WriteFile empty", func(t *testing.T) {
		if err := vol.WriteFile(ctx, sid, "empty.txt", []byte{}, 0o644); err != nil {
			t.Fatalf("WriteFile empty: %v", err)
		}
		got, _, err := vol.ReadFile(ctx, sid, "empty.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %d bytes", len(got))
		}
	})

	t.Run("Stat", func(t *testing.T) {
		info, err := vol.Stat(ctx, sid, "test.txt")
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if info.Size != 19 {
			t.Errorf("size = %d, want 19", info.Size)
		}
	})

	t.Run("ListDir", func(t *testing.T) {
		entries, err := vol.ListDir(ctx, sid, ".")
		if err != nil {
			t.Fatalf("ListDir: %v", err)
		}
		if len(entries) == 0 {
			t.Error("expected entries")
		}
	})

	t.Run("Rename", func(t *testing.T) {
		if err := vol.Rename(ctx, sid, "test.txt", "renamed.txt"); err != nil {
			t.Fatalf("Rename: %v", err)
		}
		_, _, err := vol.ReadFile(ctx, sid, "renamed.txt")
		if err != nil {
			t.Fatalf("ReadFile after rename: %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		if err := vol.Remove(ctx, sid, "renamed.txt", false); err != nil {
			t.Fatalf("Remove: %v", err)
		}
	})

	t.Run("Remove recursive", func(t *testing.T) {
		if err := vol.Remove(ctx, sid, "sub", true); err != nil {
			t.Fatalf("RemoveAll: %v", err)
		}
	})

	t.Run("SyncPush", func(t *testing.T) {
		srcDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(srcDir, "push.txt"), []byte("pushed"), 0o644)
		result, err := vol.SyncPush(ctx, sid, srcDir, volume.WithForceFull())
		if err != nil {
			t.Fatalf("SyncPush: %v", err)
		}
		if result.FilesSynced < 1 {
			t.Errorf("synced = %d", result.FilesSynced)
		}
	})

	t.Run("SyncPull", func(t *testing.T) {
		dstDir := t.TempDir()
		result, err := vol.SyncPull(ctx, sid, dstDir)
		if err != nil {
			t.Fatalf("SyncPull: %v", err)
		}
		if result.FilesSynced < 1 {
			t.Errorf("synced = %d", result.FilesSynced)
		}
		data, err := os.ReadFile(filepath.Join(dstDir, "push.txt"))
		if err != nil {
			t.Fatalf("ReadFile pulled: %v", err)
		}
		if string(data) != "pushed" {
			t.Errorf("content = %q", data)
		}
	})

	t.Run("SyncPull with existing local files", func(t *testing.T) {
		_ = vol.WriteFile(ctx, sid, "extra.txt", []byte("extra"), 0o644)

		dstDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dstDir, "push.txt"), []byte("pushed"), 0o644)

		result, err := vol.SyncPull(ctx, sid, dstDir)
		if err != nil {
			t.Fatalf("SyncPull: %v", err)
		}
		if result.FilesSynced < 1 {
			t.Errorf("synced = %d", result.FilesSynced)
		}
	})

	t.Run("WriteFile large (multi-chunk)", func(t *testing.T) {
		largeData := make([]byte, 128*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		if err := vol.WriteFile(ctx, sid, "large.bin", largeData, 0o644); err != nil {
			t.Fatalf("WriteFile large: %v", err)
		}
		got, _, err := vol.ReadFile(ctx, sid, "large.bin")
		if err != nil {
			t.Fatalf("ReadFile large: %v", err)
		}
		if len(got) != len(largeData) {
			t.Errorf("len = %d, want %d", len(got), len(largeData))
		}
	})

	t.Run("SyncPush with excludes", func(t *testing.T) {
		srcDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(srcDir, "keep.txt"), []byte("keep"), 0o644)
		_ = os.WriteFile(filepath.Join(srcDir, "skip.log"), []byte("skip"), 0o644)

		result, err := vol.SyncPush(ctx, "exclude-remote", srcDir, volume.WithForceFull(), volume.WithExcludes("*.log"))
		if err != nil {
			t.Fatalf("SyncPush: %v", err)
		}
		if result.FilesSynced != 1 {
			t.Errorf("synced = %d, want 1", result.FilesSynced)
		}
		_ = vol.Remove(ctx, "exclude-remote", ".", true)
	})

	t.Run("SyncPull empty session", func(t *testing.T) {
		emptyDir := t.TempDir()
		_ = vol.MkdirAll(ctx, "empty-pull", ".")
		result, err := vol.SyncPull(ctx, "empty-pull", emptyDir)
		if err != nil {
			t.Fatalf("SyncPull: %v", err)
		}
		if result.FilesSynced != 0 {
			t.Errorf("synced = %d, want 0", result.FilesSynced)
		}
	})

	t.Run("SyncPush with RemotePath", func(t *testing.T) {
		srcDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(srcDir, "mod.go"), []byte("module test"), 0o644)
		result, err := vol.SyncPush(ctx, "rp-test", srcDir, volume.WithForceFull(), volume.WithRemotePath("packages/api"))
		if err != nil {
			t.Fatalf("SyncPush: %v", err)
		}
		if result.FilesSynced < 1 {
			t.Errorf("synced = %d", result.FilesSynced)
		}
		_ = vol.Remove(ctx, "rp-test", ".", true)
	})

	t.Run("SyncPull with RemotePath", func(t *testing.T) {
		rpSid := "rp-pull-test"
		_ = vol.MkdirAll(ctx, rpSid, "sub/deep")
		_ = vol.WriteFile(ctx, rpSid, "sub/deep/f.txt", []byte("deep"), 0o644)
		_ = vol.WriteFile(ctx, rpSid, "root.txt", []byte("root"), 0o644)

		dstDir := t.TempDir()
		result, err := vol.SyncPull(ctx, rpSid, dstDir, volume.WithForceFull(), volume.WithRemotePath("sub/deep"))
		if err != nil {
			t.Fatalf("SyncPull: %v", err)
		}
		if result.FilesSynced < 1 {
			t.Errorf("synced = %d", result.FilesSynced)
		}
		data, err := os.ReadFile(filepath.Join(dstDir, "f.txt"))
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "deep" {
			t.Errorf("content = %q", data)
		}
		_ = vol.Remove(ctx, rpSid, ".", true)
	})

	t.Run("Zip and Unzip", func(t *testing.T) {
		arcSid := "arc-zip-test"
		_ = vol.MkdirAll(ctx, arcSid, "src")
		_ = vol.WriteFile(ctx, arcSid, "src/a.txt", []byte("zip a"), 0o644)
		_ = vol.WriteFile(ctx, arcSid, "src/b.txt", []byte("zip b"), 0o644)

		zr, err := vol.Zip(ctx, arcSid, "src", "out.zip", nil)
		if err != nil {
			t.Fatalf("Zip: %v", err)
		}
		if zr.FilesCount != 2 {
			t.Errorf("zip files = %d, want 2", zr.FilesCount)
		}

		ur, err := vol.Unzip(ctx, arcSid, "out.zip", "extracted")
		if err != nil {
			t.Fatalf("Unzip: %v", err)
		}
		if ur.FilesCount != 2 {
			t.Errorf("unzip files = %d, want 2", ur.FilesCount)
		}

		data, _, err := vol.ReadFile(ctx, arcSid, "extracted/a.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "zip a" {
			t.Errorf("content = %q", data)
		}
		_ = vol.Remove(ctx, arcSid, ".", true)
	})

	t.Run("Zip with excludes", func(t *testing.T) {
		arcSid := "arc-zip-excl"
		_ = vol.MkdirAll(ctx, arcSid, "src")
		_ = vol.WriteFile(ctx, arcSid, "src/keep.txt", []byte("keep"), 0o644)
		_ = vol.WriteFile(ctx, arcSid, "src/skip.log", []byte("skip"), 0o644)

		zr, err := vol.Zip(ctx, arcSid, "src", "filtered.zip", []string{"*.log"})
		if err != nil {
			t.Fatalf("Zip: %v", err)
		}
		if zr.FilesCount != 1 {
			t.Errorf("zip files = %d, want 1", zr.FilesCount)
		}
		_ = vol.Remove(ctx, arcSid, ".", true)
	})

	t.Run("Gzip and Gunzip", func(t *testing.T) {
		arcSid := "arc-gzip-test"
		_ = vol.WriteFile(ctx, arcSid, "data.txt", []byte("gzip remote"), 0o644)

		gr, err := vol.Gzip(ctx, arcSid, "data.txt", "data.txt.gz")
		if err != nil {
			t.Fatalf("Gzip: %v", err)
		}
		if gr.FilesCount != 1 {
			t.Errorf("gzip files = %d", gr.FilesCount)
		}

		ur, err := vol.Gunzip(ctx, arcSid, "data.txt.gz", "restored.txt")
		if err != nil {
			t.Fatalf("Gunzip: %v", err)
		}
		if ur.FilesCount != 1 {
			t.Errorf("gunzip files = %d", ur.FilesCount)
		}

		data, _, err := vol.ReadFile(ctx, arcSid, "restored.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "gzip remote" {
			t.Errorf("content = %q", data)
		}
		_ = vol.Remove(ctx, arcSid, ".", true)
	})

	t.Run("Tar and Untar", func(t *testing.T) {
		arcSid := "arc-tar-test"
		_ = vol.MkdirAll(ctx, arcSid, "src")
		_ = vol.WriteFile(ctx, arcSid, "src/x.txt", []byte("tar remote"), 0o644)

		tr, err := vol.Tar(ctx, arcSid, "src", "out.tar", nil)
		if err != nil {
			t.Fatalf("Tar: %v", err)
		}
		if tr.FilesCount != 1 {
			t.Errorf("tar files = %d", tr.FilesCount)
		}

		ur, err := vol.Untar(ctx, arcSid, "out.tar", "extracted")
		if err != nil {
			t.Fatalf("Untar: %v", err)
		}
		if ur.FilesCount != 1 {
			t.Errorf("untar files = %d", ur.FilesCount)
		}

		data, _, err := vol.ReadFile(ctx, arcSid, "extracted/x.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "tar remote" {
			t.Errorf("content = %q", data)
		}
		_ = vol.Remove(ctx, arcSid, ".", true)
	})

	t.Run("Tgz and Untgz", func(t *testing.T) {
		arcSid := "arc-tgz-test"
		_ = vol.MkdirAll(ctx, arcSid, "src")
		_ = vol.WriteFile(ctx, arcSid, "src/f.txt", []byte("tgz remote"), 0o644)

		tr, err := vol.Tgz(ctx, arcSid, "src", "out.tgz", nil)
		if err != nil {
			t.Fatalf("Tgz: %v", err)
		}
		if tr.FilesCount != 1 {
			t.Errorf("tgz files = %d", tr.FilesCount)
		}

		ur, err := vol.Untgz(ctx, arcSid, "out.tgz", "extracted")
		if err != nil {
			t.Fatalf("Untgz: %v", err)
		}
		if ur.FilesCount != 1 {
			t.Errorf("untgz files = %d", ur.FilesCount)
		}

		data, _, err := vol.ReadFile(ctx, arcSid, "extracted/f.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "tgz remote" {
			t.Errorf("content = %q", data)
		}
		_ = vol.Remove(ctx, arcSid, ".", true)
	})

	t.Run("Abs dot", func(t *testing.T) {
		absSid := "abs-remote-test"
		_ = vol.MkdirAll(ctx, absSid, ".")
		got, err := vol.Abs(ctx, absSid, ".")
		if err != nil {
			t.Fatalf("Abs: %v", err)
		}
		if got == "" {
			t.Error("Abs returned empty")
		}
		_ = vol.Remove(ctx, absSid, ".", true)
	})

	t.Run("Abs relative", func(t *testing.T) {
		got, err := vol.Abs(ctx, sid, "sub/file.txt")
		if err != nil {
			t.Fatalf("Abs: %v", err)
		}
		if got == "" {
			t.Error("Abs returned empty")
		}
	})

	t.Run("Abs path traversal", func(t *testing.T) {
		_, err := vol.Abs(ctx, sid, "../../etc/passwd")
		if err == nil {
			t.Error("expected error for Abs path traversal")
		}
	})

	_ = vol.Remove(ctx, sid, ".", true)
}

func TestRemoteRemoveError(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	err := vol.Remove(context.Background(), "nonexistent-session", "nonexistent.txt", false)
	if err == nil {
		t.Error("expected error for remove nonexistent")
	}
}

func TestRemoteRenameError(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	err := vol.Rename(context.Background(), "nonexistent-session", "a.txt", "b.txt")
	if err == nil {
		t.Error("expected error for rename nonexistent")
	}
}

func TestRemoteMkdirAllAndStatError(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	_, err := vol.Stat(context.Background(), "stat-test", "nonexistent.txt")
	if err == nil {
		t.Error("expected error for stat nonexistent")
	}
}

func TestRemoteReadFileNotFound(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	_, _, err := vol.ReadFile(context.Background(), "notfound-session", "notfound.txt")
	if err == nil {
		t.Error("expected error for read nonexistent")
	}
}

func TestRemoteListDirNotFound(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	_, err := vol.ListDir(context.Background(), "notfound-session", "notfound-dir")
	if err == nil {
		t.Error("expected error for listdir nonexistent")
	}
}

func TestRemoteCopy(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	defer vol.Close()
	ctx := context.Background()
	sid := "copy-remote-test"

	_ = vol.WriteFile(ctx, sid, "src.txt", []byte("remote copy"), 0o644)

	result, err := vol.Copy(ctx, sid, "src.txt", "dst.txt", volume.WithForceFull())
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if result.FilesSynced < 1 {
		t.Errorf("synced = %d", result.FilesSynced)
	}

	data, _, err := vol.ReadFile(ctx, sid, "dst.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "remote copy" {
		t.Errorf("content = %q", data)
	}

	_ = vol.Remove(ctx, sid, ".", true)
}

func TestRemoteCopyDir(t *testing.T) {
	conn := dialRemoteTai(t)
	vol := volume.NewRemote(conn)
	defer vol.Close()
	ctx := context.Background()
	sid := "copy-remote-dir"

	_ = vol.MkdirAll(ctx, sid, "src/sub")
	_ = vol.WriteFile(ctx, sid, "src/a.txt", []byte("aaa"), 0o644)
	_ = vol.WriteFile(ctx, sid, "src/sub/b.txt", []byte("bbb"), 0o644)

	result, err := vol.Copy(ctx, sid, "src", "dst", volume.WithForceFull())
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if result.FilesSynced < 2 {
		t.Errorf("synced = %d", result.FilesSynced)
	}

	data, _, err := vol.ReadFile(ctx, sid, "dst/sub/b.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "bbb" {
		t.Errorf("content = %q", data)
	}

	_ = vol.Remove(ctx, sid, ".", true)
}
