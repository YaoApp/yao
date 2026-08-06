package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initGitRepo(t *testing.T, dir string) *git.Repository {
	t.Helper()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	return repo
}

func gitCommit(t *testing.T, repo *git.Repository, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	wt, _ := repo.Worktree()
	_, err := wt.Add(name)
	require.NoError(t, err)
	_, err = wt.Commit("add "+name, &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)
}

func TestGitListRepos(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			initGitRepo(t, mountPath)

			repos, err := m.GitListRepos(ctx, ws.ID, true)
			require.NoError(t, err)
			assert.Len(t, repos, 1)
			assert.Equal(t, ".", repos[0].Path)
		})
	}
}

func TestGitListRepos_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.GitListRepos(context.Background(), "nonexistent-ws", true)
			assert.Error(t, err)
		})
	}
}

func TestGitListRepos_Cache(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			initGitRepo(t, mountPath)

			repos1, err := m.GitListRepos(ctx, ws.ID, true)
			require.NoError(t, err)
			assert.Len(t, repos1, 1)

			repos2, err := m.GitListRepos(ctx, ws.ID, false)
			require.NoError(t, err)
			assert.Len(t, repos2, 1)
			assert.Equal(t, repos1[0].Path, repos2[0].Path)

			repos3, err := m.GitListRepos(ctx, ws.ID, true)
			require.NoError(t, err)
			assert.Len(t, repos3, 1)
		})
	}
}

func TestGitStatus(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			repo := initGitRepo(t, mountPath)
			gitCommit(t, repo, mountPath, "init.txt", "init")

			result, err := m.GitStatus(ctx, ws.ID, ".")
			require.NoError(t, err)
			assert.NotEmpty(t, result.Branch)
			// .workspace.json metadata file may appear as untracked
			for _, f := range result.Files {
				if f.Path != ".workspace.json" {
					t.Errorf("unexpected changed file: %s", f.Path)
				}
			}
		})
	}
}

func TestGitStatus_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.GitStatus(context.Background(), "nonexistent-ws", ".")
			assert.Error(t, err)
		})
	}
}

func TestGitFileDiff(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			repo := initGitRepo(t, mountPath)
			gitCommit(t, repo, mountPath, "file.txt", "original")

			require.NoError(t, os.WriteFile(filepath.Join(mountPath, "file.txt"), []byte("modified"), 0o644))

			result, err := m.GitFileDiff(ctx, ws.ID, ".", "file.txt", false)
			require.NoError(t, err)
			assert.Equal(t, "original", result.Original)
			assert.Equal(t, "modified", result.Modified)
		})
	}
}

func TestGitFileDiff_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.GitFileDiff(context.Background(), "nonexistent-ws", ".", "file.txt", false)
			assert.Error(t, err)
		})
	}
}

func TestGitAdd(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			repo := initGitRepo(t, mountPath)
			gitCommit(t, repo, mountPath, "base.txt", "base")

			require.NoError(t, os.WriteFile(filepath.Join(mountPath, "new.txt"), []byte("new"), 0o644))

			err = m.GitAdd(ctx, ws.ID, ".", []string{"new.txt"})
			require.NoError(t, err)

			wt, _ := repo.Worktree()
			st, _ := wt.Status()
			assert.Equal(t, git.Added, st["new.txt"].Staging)
		})
	}
}

func TestGitAdd_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			err := m.GitAdd(context.Background(), "nonexistent-ws", ".", nil)
			assert.Error(t, err)
		})
	}
}

func TestGitReset(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			repo := initGitRepo(t, mountPath)
			gitCommit(t, repo, mountPath, "base.txt", "base")

			require.NoError(t, os.WriteFile(filepath.Join(mountPath, "staged.txt"), []byte("x"), 0o644))
			wt, _ := repo.Worktree()
			wt.Add("staged.txt")

			err = m.GitReset(ctx, ws.ID, ".", []string{"staged.txt"})
			require.NoError(t, err)

			st, _ := wt.Status()
			assert.Equal(t, git.Untracked, st["staged.txt"].Staging)
		})
	}
}

func TestGitReset_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			err := m.GitReset(context.Background(), "nonexistent-ws", ".", nil)
			assert.Error(t, err)
		})
	}
}

func TestGitCommit(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			repo := initGitRepo(t, mountPath)
			gitCommit(t, repo, mountPath, "base.txt", "base")

			require.NoError(t, os.WriteFile(filepath.Join(mountPath, "new.txt"), []byte("new"), 0o644))
			wt, _ := repo.Worktree()
			wt.Add("new.txt")

			result, err := m.GitCommit(ctx, ws.ID, ".", "test commit", "Test", "test@test.com", false)
			require.NoError(t, err)
			assert.NotEmpty(t, result.CommitHash)
		})
	}
}

func TestGitCommit_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.GitCommit(context.Background(), "nonexistent-ws", ".", "msg", "", "", false)
			assert.Error(t, err)
		})
	}
}

func TestGitDiscardChanges(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)
			ctx := context.Background()

			mountPath, err := m.MountPath(ctx, ws.ID)
			require.NoError(t, err)
			repo := initGitRepo(t, mountPath)
			gitCommit(t, repo, mountPath, "file.txt", "original")

			require.NoError(t, os.WriteFile(filepath.Join(mountPath, "file.txt"), []byte("modified"), 0o644))

			err = m.GitDiscardChanges(ctx, ws.ID, ".", []string{"file.txt"})
			require.NoError(t, err)

			data, _ := os.ReadFile(filepath.Join(mountPath, "file.txt"))
			assert.Equal(t, "original", string(data))
		})
	}
}

func TestGitDiscardChanges_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			err := m.GitDiscardChanges(context.Background(), "nonexistent-ws", ".", nil)
			assert.Error(t, err)
		})
	}
}
