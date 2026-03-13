package workspace

import (
	"io"
	"io/fs"
	"testing"

	"github.com/yaoapp/yao/tai/volume"
)

func TestWorkspaceFS(t *testing.T) {
	dir := t.TempDir()
	vol := volume.NewLocal(dir)
	defer vol.Close()

	wfs := New(vol, "ws-test")
	defer wfs.Close()

	t.Run("WriteFile and ReadFile", func(t *testing.T) {
		if err := wfs.WriteFile("hello.txt", []byte("world"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		data, err := wfs.ReadFile("hello.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "world" {
			t.Errorf("got %q, want %q", data, "world")
		}
	})

	t.Run("Stat", func(t *testing.T) {
		info, err := wfs.Stat("hello.txt")
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if info.Name() != "hello.txt" {
			t.Errorf("name = %q, want %q", info.Name(), "hello.txt")
		}
		if info.Size() != 5 {
			t.Errorf("size = %d, want 5", info.Size())
		}
		if info.IsDir() {
			t.Error("expected file, not dir")
		}
		if info.Mode() == 0 {
			t.Error("mode should be nonzero")
		}
		if info.ModTime().IsZero() {
			t.Error("modtime should be nonzero")
		}
		if info.Sys() != nil {
			t.Error("Sys should be nil")
		}
	})

	t.Run("MkdirAll and ReadDir", func(t *testing.T) {
		if err := wfs.MkdirAll("sub/dir", 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		_ = wfs.WriteFile("sub/dir/file.txt", []byte("x"), 0o644)
		entries, err := wfs.ReadDir("sub/dir")
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		e := entries[0]
		if e.Name() != "file.txt" {
			t.Errorf("entry = %q, want %q", e.Name(), "file.txt")
		}
		if e.IsDir() {
			t.Error("entry should not be dir")
		}
		if e.Type()&fs.ModeDir != 0 {
			t.Error("Type should not include ModeDir")
		}
		info, err := e.Info()
		if err != nil {
			t.Fatalf("Info: %v", err)
		}
		if info.Name() != "file.txt" {
			t.Errorf("info name = %q", info.Name())
		}
	})

	t.Run("Open file and read", func(t *testing.T) {
		f, err := wfs.Open("hello.txt")
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer f.Close()

		// Stat via file
		finfo, err := f.Stat()
		if err != nil {
			t.Fatalf("file.Stat: %v", err)
		}
		if finfo.Name() != "hello.txt" {
			t.Errorf("name = %q", finfo.Name())
		}

		// Read all
		buf := make([]byte, 10)
		n, _ := f.Read(buf)
		if string(buf[:n]) != "world" {
			t.Errorf("read = %q, want %q", buf[:n], "world")
		}
		// Read past EOF
		_, err = f.Read(buf)
		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
	})

	t.Run("Open directory", func(t *testing.T) {
		f, err := wfs.Open("sub/dir")
		if err != nil {
			t.Fatalf("Open dir: %v", err)
		}
		defer f.Close()
		info, _ := f.Stat()
		if !info.IsDir() {
			t.Error("expected dir")
		}
		// Read on dir should error
		buf := make([]byte, 10)
		_, err = f.Read(buf)
		if err == nil {
			t.Error("expected error reading dir")
		}
	})

	t.Run("Rename", func(t *testing.T) {
		if err := wfs.Rename("hello.txt", "hi.txt"); err != nil {
			t.Fatalf("Rename: %v", err)
		}
		_, err := wfs.Stat("hi.txt")
		if err != nil {
			t.Fatalf("Stat after rename: %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		if err := wfs.Remove("hi.txt"); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		_, err := wfs.Stat("hi.txt")
		if err == nil {
			t.Error("expected error after remove")
		}
	})

	t.Run("RemoveAll", func(t *testing.T) {
		if err := wfs.RemoveAll("sub"); err != nil {
			t.Fatalf("RemoveAll: %v", err)
		}
		_, err := wfs.Stat("sub")
		if err == nil {
			t.Error("expected error after removeall")
		}
	})

	t.Run("Invalid path", func(t *testing.T) {
		_, err := wfs.Open("/absolute")
		if err == nil {
			t.Error("expected error for absolute path")
		}
		_, err = wfs.Stat("/absolute")
		if err == nil {
			t.Error("expected error for absolute path in Stat")
		}
		_, err = wfs.ReadFile("/absolute")
		if err == nil {
			t.Error("expected error for absolute path in ReadFile")
		}
		_, err = wfs.ReadDir("/absolute")
		if err == nil {
			t.Error("expected error for absolute path in ReadDir")
		}
	})

	t.Run("Open nonexistent", func(t *testing.T) {
		_, err := wfs.Open("nonexistent.txt")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("ReadFile nonexistent", func(t *testing.T) {
		_, err := wfs.ReadFile("nonexistent.txt")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("toFSInfo with slash", func(t *testing.T) {
		info := toFSInfo("sub/dir/file.txt", &volume.FileInfo{Path: "sub/dir/file.txt", Size: 1})
		if info.Name() != "file.txt" {
			t.Errorf("name = %q, want %q", info.Name(), "file.txt")
		}
	})

	t.Run("toFSInfo root", func(t *testing.T) {
		info := toFSInfo("", &volume.FileInfo{Path: "", IsDir: true})
		if info.Name() != "." {
			t.Errorf("name = %q, want %q", info.Name(), ".")
		}
	})
}

func TestGetRoot(t *testing.T) {
	dir := t.TempDir()
	vol := volume.NewLocal(dir)
	defer vol.Close()

	sid := "getroot-test"
	wfs := New(vol, sid)
	defer wfs.Close()

	root, err := wfs.GetRoot()
	if err != nil {
		t.Fatalf("GetRoot: %v", err)
	}
	want := dir + "/" + sid
	if root != want {
		t.Errorf("GetRoot() = %q, want %q", root, want)
	}
}

// Compile-time interface checks.
var (
	_ fs.FS         = (*workspaceFS)(nil)
	_ fs.StatFS     = (*workspaceFS)(nil)
	_ fs.ReadFileFS = (*workspaceFS)(nil)
	_ fs.ReadDirFS  = (*workspaceFS)(nil)
)
