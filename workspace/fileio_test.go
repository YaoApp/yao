package workspace_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaoapp/yao/workspace"
)

func TestReadWriteFile(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			err := m.WriteFile(ctx, ws.ID, "hello.txt", []byte("hello world"), 0644)
			require.NoError(t, err)

			data, err := m.ReadFile(ctx, ws.ID, "hello.txt")
			require.NoError(t, err)
			assert.Equal(t, "hello world", string(data))
		})
	}
}

func TestWriteFile_NestedPath(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			err := m.WriteFile(ctx, ws.ID, "src/main.go", []byte("package main"), 0644)
			require.NoError(t, err)

			data, err := m.ReadFile(ctx, ws.ID, "src/main.go")
			require.NoError(t, err)
			assert.Equal(t, "package main", string(data))
		})
	}
}

func TestListDir(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			require.NoError(t, m.WriteFile(ctx, ws.ID, "a.txt", []byte("a"), 0644))
			require.NoError(t, m.WriteFile(ctx, ws.ID, "b.txt", []byte("b"), 0644))

			entries, err := m.ListDir(ctx, ws.ID, ".")
			require.NoError(t, err)
			names := make(map[string]bool)
			for _, e := range entries {
				names[e.Name] = true
			}
			assert.True(t, names["a.txt"])
			assert.True(t, names["b.txt"])
		})
	}
}

func TestRemoveFile(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			require.NoError(t, m.WriteFile(ctx, ws.ID, "tmp.txt", []byte("temp"), 0644))

			err := m.Remove(ctx, ws.ID, "tmp.txt")
			require.NoError(t, err)

			_, err = m.ReadFile(ctx, ws.ID, "tmp.txt")
			assert.Error(t, err)
		})
	}
}

func TestFS_ReadFile(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			require.NoError(t, m.WriteFile(ctx, ws.ID, "test.txt", []byte("via fs"), 0644))

			wfs, err := m.FS(ctx, ws.ID)
			require.NoError(t, err)

			data, err := fs.ReadFile(wfs, "test.txt")
			require.NoError(t, err)
			assert.Equal(t, "via fs", string(data))
		})
	}
}

func TestFS_WriteFile(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			wfs, err := m.FS(ctx, ws.ID)
			require.NoError(t, err)

			err = wfs.WriteFile("from-fs.txt", []byte("written via fs"), 0644)
			require.NoError(t, err)

			data, err := m.ReadFile(ctx, ws.ID, "from-fs.txt")
			require.NoError(t, err)
			assert.Equal(t, "written via fs", string(data))
		})
	}
}

func TestFS_MkdirAll(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			wfs, err := m.FS(ctx, ws.ID)
			require.NoError(t, err)

			err = wfs.MkdirAll("a/b/c", 0755)
			require.NoError(t, err)

			info, err := fs.Stat(wfs, "a/b/c")
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestFS_Rename(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			wfs, err := m.FS(ctx, ws.ID)
			require.NoError(t, err)

			err = wfs.WriteFile("old.txt", []byte("content"), 0644)
			require.NoError(t, err)

			err = wfs.Rename("old.txt", "new.txt")
			require.NoError(t, err)

			data, err := fs.ReadFile(wfs, "new.txt")
			require.NoError(t, err)
			assert.Equal(t, "content", string(data))

			_, err = fs.ReadFile(wfs, "old.txt")
			assert.Error(t, err)
		})
	}
}

func TestFS_WalkDir(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			wfs, err := m.FS(ctx, ws.ID)
			require.NoError(t, err)

			require.NoError(t, wfs.MkdirAll("src", 0755))
			require.NoError(t, wfs.WriteFile("src/main.go", []byte("package main"), 0644))
			require.NoError(t, wfs.WriteFile("src/util.go", []byte("package main"), 0644))

			var files []string
			err = fs.WalkDir(wfs, "src", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					files = append(files, path)
				}
				return nil
			})
			require.NoError(t, err)
			assert.Len(t, files, 2)
		})
	}
}

func TestFS_Remove(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			ctx := context.Background()
			wfs, err := m.FS(ctx, ws.ID)
			require.NoError(t, err)

			require.NoError(t, wfs.WriteFile("removeme.txt", []byte("bye"), 0644))

			err = wfs.Remove("removeme.txt")
			require.NoError(t, err)

			_, err = fs.ReadFile(wfs, "removeme.txt")
			assert.Error(t, err)
		})
	}
}

func TestFS_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.FS(context.Background(), "nonexistent")
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestManagerRename(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			require.NoError(t, m.WriteFile(ctx, ws.ID, "old.txt", []byte("rename me"), 0644))
			require.NoError(t, m.Rename(ctx, ws.ID, "old.txt", "new.txt"))

			data, err := m.ReadFile(ctx, ws.ID, "new.txt")
			require.NoError(t, err)
			assert.Equal(t, "rename me", string(data))

			_, err = m.ReadFile(ctx, ws.ID, "old.txt")
			assert.Error(t, err)
		})
	}
}

func TestManagerRename_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			err := m.Rename(context.Background(), "nonexistent", "a", "b")
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestManagerMkdirAll(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			require.NoError(t, m.MkdirAll(ctx, ws.ID, "a/b/c"))
			require.NoError(t, m.WriteFile(ctx, ws.ID, "a/b/c/test.txt", []byte("deep"), 0644))

			data, err := m.ReadFile(ctx, ws.ID, "a/b/c/test.txt")
			require.NoError(t, err)
			assert.Equal(t, "deep", string(data))
		})
	}
}

func TestManagerMkdirAll_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			err := m.MkdirAll(context.Background(), "nonexistent", "a/b")
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestManagerVolume(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			vol, wsID, err := m.Volume(ctx, ws.ID)
			require.NoError(t, err)
			assert.Equal(t, ws.ID, wsID)
			assert.NotNil(t, vol)

			require.NoError(t, vol.WriteFile(ctx, wsID, "via-vol.txt", []byte("volume direct"), 0o644))
			data, _, err := vol.ReadFile(ctx, wsID, "via-vol.txt")
			require.NoError(t, err)
			assert.Equal(t, "volume direct", string(data))
		})
	}
}

func TestManagerVolume_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, _, err := m.Volume(context.Background(), "nonexistent")
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}
