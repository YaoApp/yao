package volume

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func initTestRepo(t *testing.T, dir string) *git.Repository {
	t.Helper()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	return repo
}

func commitFile(t *testing.T, repo *git.Repository, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, _ := repo.Worktree()
	if _, err := wt.Add(name); err != nil {
		t.Fatal(err)
	}
	_, err := wt.Commit("add "+name, &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestFileLanguage(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"main.go", "go"},
		{"app.tsx", "typescript"},
		{"style.css", "css"},
		{"data.json", "json"},
		{"README.md", "markdown"},
		{"config.yaml", "yaml"},
		{"build.gradle", "groovy"},
		{"app.yao", "json"},
		{"unknown.xyz", "plaintext"},
		{"script.ps1", "powershell"},
		{"page.wxml", "xml"},
		{"Makefile", "plaintext"},
	}
	for _, c := range cases {
		got := fileLanguage(c.name)
		if got != c.want {
			t.Errorf("fileLanguage(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestIsBinaryContent(t *testing.T) {
	if isBinaryContent([]byte("hello world")) {
		t.Error("text should not be binary")
	}
	if !isBinaryContent([]byte{0x00, 0x01, 0x02}) {
		t.Error("null bytes should be binary")
	}
	if isBinaryContent([]byte{}) {
		t.Error("empty should not be binary")
	}
}

func TestStatusCodeToString(t *testing.T) {
	if statusCodeToString(git.Modified) != "M" {
		t.Errorf("Modified = %q", statusCodeToString(git.Modified))
	}
	if statusCodeToString(git.Untracked) != "?" {
		t.Errorf("Untracked = %q", statusCodeToString(git.Untracked))
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitListRepos
// ---------------------------------------------------------------------------

func TestLocalGitListRepos_Empty(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-list-empty"

	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	repos, err := vol.GitListRepos(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
}

func TestLocalGitListRepos_SingleRepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-list-single"

	repoDir := filepath.Join(dir, sid)
	initTestRepo(t, repoDir)

	repos, err := vol.GitListRepos(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].Path != "." {
		t.Errorf("path = %q, want '.'", repos[0].Path)
	}
}

func TestLocalGitListRepos_NestedRepos(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-list-nested"

	base := filepath.Join(dir, sid)
	_ = os.MkdirAll(base, 0o755)
	initTestRepo(t, filepath.Join(base, "frontend"))
	initTestRepo(t, filepath.Join(base, "backend"))

	repos, err := vol.GitListRepos(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

func TestLocalGitListRepos_HasChanges(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-list-changes"

	repoDir := filepath.Join(dir, sid)
	initTestRepo(t, repoDir)
	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("x"), 0o644)

	repos, err := vol.GitListRepos(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if !repos[0].HasChanges {
		t.Error("expected has_changes = true")
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitStatus
// ---------------------------------------------------------------------------

func TestLocalGitStatus_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-status-empty"

	initTestRepo(t, filepath.Join(dir, sid))

	result, err := vol.GitStatus(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if !result.IsEmpty {
		t.Error("expected is_empty = true")
	}
	if len(result.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(result.Files))
	}
}

func TestLocalGitStatus_WithChanges(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-status-changes"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "initial.txt", "init")

	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("new"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "initial.txt"), []byte("modified"), 0o644)

	result, err := vol.GitStatus(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if result.Branch == "" {
		t.Error("branch should not be empty")
	}
	if len(result.Files) < 2 {
		t.Errorf("expected >= 2 changed files, got %d", len(result.Files))
	}
}

func TestLocalGitStatus_InvalidPath(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.GitStatus(ctx, "test", "../../etc")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitFileDiff
// ---------------------------------------------------------------------------

func TestLocalGitFileDiff_UnstagedModification(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-unstaged"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "file.txt", "original content")

	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("modified content"), 0o644)

	result, err := vol.GitFileDiff(ctx, sid, ".", "file.txt", false)
	if err != nil {
		t.Fatalf("GitFileDiff: %v", err)
	}
	if result.Original != "original content" {
		t.Errorf("original = %q", result.Original)
	}
	if result.Modified != "modified content" {
		t.Errorf("modified = %q", result.Modified)
	}
	if result.IsBinary || result.IsNew || result.IsDeleted {
		t.Error("unexpected flags")
	}
}

func TestLocalGitFileDiff_StagedNew(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-staged-new"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "dummy.txt", "dummy")

	_ = os.WriteFile(filepath.Join(repoDir, "new.go"), []byte("package main"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.go")

	result, err := vol.GitFileDiff(ctx, sid, ".", "new.go", true)
	if err != nil {
		t.Fatalf("GitFileDiff staged: %v", err)
	}
	if !result.IsNew {
		t.Error("expected is_new = true")
	}
	if result.Language != "go" {
		t.Errorf("language = %q, want 'go'", result.Language)
	}
	if result.Modified != "package main" {
		t.Errorf("modified = %q", result.Modified)
	}
}

func TestLocalGitFileDiff_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-traversal"

	initTestRepo(t, filepath.Join(dir, sid))

	_, err := vol.GitFileDiff(ctx, sid, ".", "../../etc/passwd", false)
	if err == nil {
		t.Error("expected error for path traversal in file_path")
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitAdd + GitReset
// ---------------------------------------------------------------------------

func TestLocalGitAdd_SingleFile(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-add-single"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")
	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("new"), 0o644)

	err := vol.GitAdd(ctx, sid, ".", []string{"new.txt"})
	if err != nil {
		t.Fatalf("GitAdd: %v", err)
	}

	wt, _ := repo.Worktree()
	st, _ := wt.Status()
	fst := st["new.txt"]
	if fst.Staging != git.Added {
		t.Errorf("staging = %v, want Added", fst.Staging)
	}
}

func TestLocalGitAdd_All(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-add-all"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")
	_ = os.WriteFile(filepath.Join(repoDir, "a.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "b.txt"), []byte("b"), 0o644)

	err := vol.GitAdd(ctx, sid, ".", nil)
	if err != nil {
		t.Fatalf("GitAdd all: %v", err)
	}

	wt, _ := repo.Worktree()
	st, _ := wt.Status()
	for _, name := range []string{"a.txt", "b.txt"} {
		if st[name].Staging != git.Added {
			t.Errorf("%s staging = %v, want Added", name, st[name].Staging)
		}
	}
}

func TestLocalGitReset_SingleFile(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-reset-single"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("new"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.txt")

	err := vol.GitReset(ctx, sid, ".", []string{"new.txt"})
	if err != nil {
		t.Fatalf("GitReset: %v", err)
	}

	st, _ := wt.Status()
	fst := st["new.txt"]
	if fst.Staging != git.Untracked {
		t.Errorf("after reset: staging = %v, want Untracked", fst.Staging)
	}
}

func TestLocalGitReset_AllInEmptyRepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-reset-empty"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)

	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("new"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.txt")

	err := vol.GitReset(ctx, sid, ".", nil)
	if err != nil {
		t.Fatalf("GitReset all in empty: %v", err)
	}

	idx, _ := repo.Storer.Index()
	if len(idx.Entries) != 0 {
		t.Errorf("index should be empty after reset-all in empty repo, got %d entries", len(idx.Entries))
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitCommit
// ---------------------------------------------------------------------------

func TestLocalGitCommit_Success(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-commit-ok"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("staged"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.txt")

	result, err := vol.GitCommit(ctx, sid, ".", "test commit", "Author", "author@test.com", false)
	if err != nil {
		t.Fatalf("GitCommit: %v", err)
	}
	if result.CommitHash == "" {
		t.Error("expected non-empty commit hash")
	}
}

func TestLocalGitCommit_EmptyMessage(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-commit-empty-msg"

	initTestRepo(t, filepath.Join(dir, sid))

	_, err := vol.GitCommit(ctx, sid, ".", "", "", "", false)
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestLocalGitCommit_NothingToCommit(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-commit-nothing"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_, err := vol.GitCommit(ctx, sid, ".", "empty", "", "", false)
	if err == nil {
		t.Error("expected 'nothing to commit' error")
	}
}

func TestLocalGitCommit_AllowEmpty(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-commit-allow-empty"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	result, err := vol.GitCommit(ctx, sid, ".", "allow empty", "", "", true)
	if err != nil {
		t.Fatalf("GitCommit allow empty: %v", err)
	}
	if result.CommitHash == "" {
		t.Error("expected commit hash")
	}
}

func TestLocalGitCommit_DefaultAuthor(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-commit-default-author"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")
	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("x"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.txt")

	result, err := vol.GitCommit(ctx, sid, ".", "default author commit", "", "", false)
	if err != nil {
		t.Fatalf("GitCommit: %v", err)
	}
	if result.CommitHash == "" {
		t.Error("expected non-empty commit hash")
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitDiscardChanges
// ---------------------------------------------------------------------------

func TestLocalGitDiscard_SingleFile(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-discard-single"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "file.txt", "original")

	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("modified"), 0o644)

	err := vol.GitDiscardChanges(ctx, sid, ".", []string{"file.txt"})
	if err != nil {
		t.Fatalf("GitDiscardChanges: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(repoDir, "file.txt"))
	if string(data) != "original" {
		t.Errorf("content = %q, want 'original'", data)
	}
}

func TestLocalGitDiscard_UntrackedFile(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-discard-untracked"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_ = os.WriteFile(filepath.Join(repoDir, "untracked.txt"), []byte("x"), 0o644)

	err := vol.GitDiscardChanges(ctx, sid, ".", []string{"untracked.txt"})
	if err != nil {
		t.Fatalf("GitDiscardChanges untracked: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, "untracked.txt")); err == nil {
		t.Error("untracked file should be removed")
	}
}

func TestLocalGitDiscard_All(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-discard-all"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "file.txt", "original")

	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("modified"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "untracked.txt"), []byte("x"), 0o644)

	err := vol.GitDiscardChanges(ctx, sid, ".", nil)
	if err != nil {
		t.Fatalf("GitDiscardChanges all: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(repoDir, "file.txt"))
	if string(data) != "original" {
		t.Errorf("content = %q, want 'original'", data)
	}
	if _, err := os.Stat(filepath.Join(repoDir, "untracked.txt")); err == nil {
		t.Error("untracked file should be cleaned")
	}
}

func TestLocalGitDiscard_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-discard-empty-repo"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)

	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("x"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.txt")

	err := vol.GitDiscardChanges(ctx, sid, ".", nil)
	if err != nil {
		t.Fatalf("GitDiscardChanges empty repo: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, "new.txt")); err == nil {
		t.Error("file should be cleaned in empty repo discard-all")
	}
}

func TestLocalGitDiscard_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-discard-traversal"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	err := vol.GitDiscardChanges(ctx, sid, ".", []string{"../../etc/passwd"})
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — openRepo errors
// ---------------------------------------------------------------------------

func TestLocalGit_NotARepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "not-a-repo"

	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	_, err := vol.GitStatus(ctx, sid, ".")
	if err == nil {
		t.Error("expected error for non-repo directory")
	}
}

func TestLocalGit_PathTraversal_InOpenRepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.GitStatus(ctx, "test", "../../etc")
	if err == nil {
		t.Error("expected path traversal error")
	}
}

// ---------------------------------------------------------------------------
// resolveHead
// ---------------------------------------------------------------------------

func TestResolveHead_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	repo := initTestRepo(t, dir)

	branch, _, _, isEmpty := resolveHead(repo)
	if !isEmpty {
		t.Error("expected isEmpty=true for new repo")
	}
	if branch == "" {
		t.Error("branch should have a default name")
	}
}

func TestResolveHead_WithCommit(t *testing.T) {
	dir := t.TempDir()
	repo := initTestRepo(t, dir)
	commitFile(t, repo, dir, "f.txt", "x")

	branch, hash, isDetached, isEmpty := resolveHead(repo)
	if isEmpty {
		t.Error("should not be empty after commit")
	}
	if isDetached {
		t.Error("should not be detached")
	}
	if hash.IsZero() {
		t.Error("hash should not be zero")
	}
	if branch == "" {
		t.Error("branch name should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Remote Git mock tests
// ---------------------------------------------------------------------------

func TestMockRemoteGitListRepos(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	repos, err := vol.GitListRepos(context.Background(), "s1", ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	_ = repos
}

func TestMockRemoteGitStatus(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	result, err := vol.GitStatus(context.Background(), "s1", ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	_ = result
}

func TestMockRemoteGitFileDiff(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	result, err := vol.GitFileDiff(context.Background(), "s1", ".", "file.txt", false)
	if err != nil {
		t.Fatalf("GitFileDiff: %v", err)
	}
	_ = result
}

func TestMockRemoteGitAdd(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitAdd(context.Background(), "s1", ".", []string{"file.txt"})
	if err != nil {
		t.Fatalf("GitAdd: %v", err)
	}
}

func TestMockRemoteGitReset(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitReset(context.Background(), "s1", ".", []string{"file.txt"})
	if err != nil {
		t.Fatalf("GitReset: %v", err)
	}
}

func TestMockRemoteGitCommit(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	result, err := vol.GitCommit(context.Background(), "s1", ".", "msg", "Author", "a@b.com", false)
	if err != nil {
		t.Fatalf("GitCommit: %v", err)
	}
	_ = result
}

func TestMockRemoteGitDiscardChanges(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitDiscardChanges(context.Background(), "s1", ".", []string{"file.txt"})
	if err != nil {
		t.Fatalf("GitDiscardChanges: %v", err)
	}
}

func TestMockRemoteGitAddFail(t *testing.T) {
	mock := &mockVolumeServer{gitOpSuccess: false, gitOpMessage: "add failed"}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitAdd(context.Background(), "s1", ".", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteGitResetFail(t *testing.T) {
	mock := &mockVolumeServer{gitOpSuccess: false, gitOpMessage: "reset failed"}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitReset(context.Background(), "s1", ".", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteGitCommitFail(t *testing.T) {
	mock := &mockVolumeServer{gitCommitSuccess: false, gitCommitMessage: "nothing to commit"}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.GitCommit(context.Background(), "s1", ".", "msg", "", "", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteGitDiscardFail(t *testing.T) {
	mock := &mockVolumeServer{gitOpSuccess: false, gitOpMessage: "discard failed"}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitDiscardChanges(context.Background(), "s1", ".", nil)
	if err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitFileDiff staged modification & deleted & binary
// ---------------------------------------------------------------------------

func TestLocalGitFileDiff_StagedModification(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-staged-mod"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "file.go", "package main\n")

	_ = os.WriteFile(filepath.Join(repoDir, "file.go"), []byte("package main\n\nfunc init() {}\n"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("file.go")

	result, err := vol.GitFileDiff(ctx, sid, ".", "file.go", true)
	if err != nil {
		t.Fatalf("GitFileDiff staged mod: %v", err)
	}
	if result.IsNew || result.IsDeleted {
		t.Error("should not be new or deleted")
	}
	if result.Original != "package main\n" {
		t.Errorf("original = %q", result.Original)
	}
	if result.Modified != "package main\n\nfunc init() {}\n" {
		t.Errorf("modified = %q", result.Modified)
	}
}

func TestLocalGitFileDiff_StagedDeletion(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-staged-del"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "todelete.txt", "will be deleted")

	wt, _ := repo.Worktree()
	wt.Remove("todelete.txt")

	result, err := vol.GitFileDiff(ctx, sid, ".", "todelete.txt", true)
	if err != nil {
		t.Fatalf("GitFileDiff staged del: %v", err)
	}
	if !result.IsDeleted {
		t.Error("expected is_deleted = true")
	}
	if result.Original != "will be deleted" {
		t.Errorf("original = %q", result.Original)
	}
}

func TestLocalGitFileDiff_UntrackedNew(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-untracked-new"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("brand new"), 0o644)

	result, err := vol.GitFileDiff(ctx, sid, ".", "new.txt", false)
	if err != nil {
		t.Fatalf("GitFileDiff untracked: %v", err)
	}
	if !result.IsNew {
		t.Error("expected is_new = true")
	}
	if result.Modified != "brand new" {
		t.Errorf("modified = %q", result.Modified)
	}
}

func TestLocalGitFileDiff_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-empty-repo"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)

	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("content"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.txt")

	result, err := vol.GitFileDiff(ctx, sid, ".", "new.txt", true)
	if err != nil {
		t.Fatalf("GitFileDiff empty repo staged: %v", err)
	}
	if !result.IsNew {
		t.Error("expected is_new = true in empty repo")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitReset
// ---------------------------------------------------------------------------

func TestLocalGitReset_AllWithCommits(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-reset-all-commits"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_ = os.WriteFile(filepath.Join(repoDir, "staged.txt"), []byte("x"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("staged.txt")

	err := vol.GitReset(ctx, sid, ".", nil)
	if err != nil {
		t.Fatalf("GitReset all: %v", err)
	}

	st, _ := wt.Status()
	if fst, ok := st["staged.txt"]; ok && fst.Staging != git.Untracked {
		t.Errorf("after reset-all: staging = %v, want Untracked", fst.Staging)
	}
}

func TestLocalGitReset_PerFileInEmptyRepo(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-reset-perfile-empty"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)

	_ = os.WriteFile(filepath.Join(repoDir, "a.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "b.txt"), []byte("b"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("a.txt")
	wt.Add("b.txt")

	err := vol.GitReset(ctx, sid, ".", []string{"a.txt"})
	if err != nil {
		t.Fatalf("GitReset perfile empty: %v", err)
	}

	idx, _ := repo.Storer.Index()
	found := false
	for _, e := range idx.Entries {
		if e.Name == "a.txt" {
			t.Error("a.txt should have been removed from index")
		}
		if e.Name == "b.txt" {
			found = true
		}
	}
	if !found {
		t.Error("b.txt should remain in index")
	}
}

func TestLocalGitReset_StagedDeletion(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-reset-staged-del"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "file.txt", "content")

	wt, _ := repo.Worktree()
	wt.Remove("file.txt")

	err := vol.GitReset(ctx, sid, ".", []string{"file.txt"})
	if err != nil {
		t.Fatalf("GitReset staged deletion: %v", err)
	}

	st, _ := wt.Status()
	fst := st["file.txt"]
	if fst.Staging == git.Deleted {
		t.Error("after reset: file should not be staged as deleted")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitDiscardChanges with modified tracked
// ---------------------------------------------------------------------------

func TestLocalGitDiscard_WorktreeModification(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-discard-wt-mod"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "file.txt", "original")
	commitFile(t, repo, repoDir, "other.txt", "other")

	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("modified"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "untracked.txt"), []byte("new"), 0o644)

	err := vol.GitDiscardChanges(ctx, sid, ".", nil)
	if err != nil {
		t.Fatalf("GitDiscardChanges: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(repoDir, "file.txt"))
	if string(data) != "original" {
		t.Errorf("content = %q, want 'original'", data)
	}
	if _, err := os.Stat(filepath.Join(repoDir, "untracked.txt")); err == nil {
		t.Error("untracked file should be removed by Clean")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitListRepos with skip dirs
// ---------------------------------------------------------------------------

func TestLocalGitListRepos_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-list-skip"

	base := filepath.Join(dir, sid)
	_ = os.MkdirAll(base, 0o755)
	initTestRepo(t, filepath.Join(base, "app"))

	nmDir := filepath.Join(base, "node_modules", "pkg")
	_ = os.MkdirAll(nmDir, 0o755)
	initTestRepo(t, nmDir)

	repos, err := vol.GitListRepos(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	for _, r := range repos {
		if strings.Contains(r.Path, "node_modules") {
			t.Errorf("should not list repo inside node_modules: %s", r.Path)
		}
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo, got %d", len(repos))
	}
}

func TestLocalGitListRepos_WithRemote(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-list-remote"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "f.txt", "x")

	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/example/repo.git"},
	})
	if err != nil {
		t.Fatalf("CreateRemote: %v", err)
	}

	repos, err := vol.GitListRepos(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].RemoteURL != "https://github.com/example/repo.git" {
		t.Errorf("remote_url = %q", repos[0].RemoteURL)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: isBinaryContent edge cases
// ---------------------------------------------------------------------------

func TestIsBinaryContent_LargeText(t *testing.T) {
	data := make([]byte, 16384)
	for i := range data {
		data[i] = 'A'
	}
	if isBinaryContent(data) {
		t.Error("large ASCII should not be binary")
	}
}

func TestIsBinaryContent_BinaryInFirstChunk(t *testing.T) {
	data := make([]byte, 16384)
	for i := range data {
		data[i] = 'x'
	}
	data[100] = 0x00
	if !isBinaryContent(data) {
		t.Error("null byte should make it binary")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitStatus with comprehensive states
// ---------------------------------------------------------------------------

func TestLocalGitStatus_WithStagedAndUnstaged(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-status-mixed"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "tracked.txt", "tracked")

	_ = os.WriteFile(filepath.Join(repoDir, "tracked.txt"), []byte("modified"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("tracked.txt")

	_ = os.WriteFile(filepath.Join(repoDir, "untracked.txt"), []byte("new"), 0o644)

	result, err := vol.GitStatus(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if result.IsEmpty {
		t.Error("should not be empty")
	}
	if len(result.Files) < 2 {
		t.Errorf("expected >= 2 files, got %d", len(result.Files))
	}

	foundStaged := false
	foundUntracked := false
	for _, f := range result.Files {
		if f.Path == "tracked.txt" && f.IndexStatus == "M" {
			foundStaged = true
		}
		if f.Path == "untracked.txt" && f.WorktreeStatus == "?" {
			foundUntracked = true
		}
	}
	if !foundStaged {
		t.Error("should find staged modification")
	}
	if !foundUntracked {
		t.Error("should find untracked file")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: isBinaryContent with invalid UTF-8
// ---------------------------------------------------------------------------

func TestIsBinaryContent_InvalidUTF8(t *testing.T) {
	data := []byte{0x80, 0x81, 0xFE, 0xFF, 0x82}
	if !isBinaryContent(data) {
		t.Error("invalid UTF-8 should be detected as binary")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: countAheadBehind + collectCommitHashes
// ---------------------------------------------------------------------------

func TestCountAheadBehind_NoUpstream(t *testing.T) {
	dir := t.TempDir()
	repo := initTestRepo(t, dir)
	commitFile(t, repo, dir, "f.txt", "x")

	ahead, behind := countAheadBehind(repo)
	if ahead != 0 || behind != 0 {
		t.Errorf("ahead=%d behind=%d, expected 0,0", ahead, behind)
	}
}

func TestCountAheadBehind_WithUpstream(t *testing.T) {
	originDir := t.TempDir()
	originRepo := initTestRepo(t, originDir)
	commitFile(t, originRepo, originDir, "init.txt", "init")

	cloneDir := t.TempDir()
	repo, err := git.PlainClone(cloneDir, false, &git.CloneOptions{URL: originDir})
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	_ = os.WriteFile(filepath.Join(cloneDir, "local.txt"), []byte("local"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("local.txt")
	wt.Commit("local commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})

	ahead, behind := countAheadBehind(repo)
	if ahead != 1 {
		t.Errorf("ahead = %d, want 1", ahead)
	}
	if behind != 0 {
		t.Errorf("behind = %d, want 0", behind)
	}
}

func TestCollectCommitHashes(t *testing.T) {
	dir := t.TempDir()
	repo := initTestRepo(t, dir)
	commitFile(t, repo, dir, "a.txt", "a")
	commitFile(t, repo, dir, "b.txt", "b")

	head, _ := repo.Head()
	hashes := collectCommitHashes(repo, head.Hash())
	if len(hashes) != 2 {
		t.Errorf("expected 2 commits, got %d", len(hashes))
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitFileDiff binary in worktree
// ---------------------------------------------------------------------------

func TestLocalGitFileDiff_BinaryWorktree(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-binary-wt"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_ = os.WriteFile(filepath.Join(repoDir, "data.bin"), []byte{0x00, 0x01, 0x02, 0x03}, 0o644)

	result, err := vol.GitFileDiff(ctx, sid, ".", "data.bin", false)
	if err != nil {
		t.Fatalf("GitFileDiff binary: %v", err)
	}
	if !result.IsBinary {
		t.Error("expected is_binary = true")
	}
}

func TestLocalGitFileDiff_BinaryStaged(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-diff-binary-staged"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")

	_ = os.WriteFile(filepath.Join(repoDir, "data.bin"), []byte{0x00, 0x01, 0x02}, 0o644)
	wt, _ := repo.Worktree()
	wt.Add("data.bin")

	result, err := vol.GitFileDiff(ctx, sid, ".", "data.bin", true)
	if err != nil {
		t.Fatalf("GitFileDiff binary staged: %v", err)
	}
	if !result.IsBinary {
		t.Error("expected is_binary = true for staged binary")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitAdd with path validation, GitCommit with untracked only
// ---------------------------------------------------------------------------

func TestLocalGitAdd_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()

	_, err := vol.GitStatus(ctx, "test", "../../etc")
	if err == nil {
		t.Error("expected path traversal error in GitAdd")
	}
}

func TestLocalGitCommit_OnlyUntrackedFiles(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-commit-untracked-only"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "base.txt", "base")
	_ = os.WriteFile(filepath.Join(repoDir, "untracked.txt"), []byte("x"), 0o644)

	_, err := vol.GitCommit(ctx, sid, ".", "should fail", "", "", false)
	if err == nil {
		t.Error("expected 'nothing to commit' error when only untracked files exist")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: GitStatus in empty repo with staged files
// ---------------------------------------------------------------------------

func TestLocalGitStatus_EmptyRepoWithStaged(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-status-empty-staged"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	_ = os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("x"), 0o644)
	wt, _ := repo.Worktree()
	wt.Add("new.txt")

	result, err := vol.GitStatus(ctx, sid, ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if !result.IsEmpty {
		t.Error("expected is_empty = true")
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}
}

// Error-path remote Git tests (gRPC transport errors)
func TestErrRemoteGitListRepos(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.GitListRepos(context.Background(), "s1", ".")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitStatus(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.GitStatus(context.Background(), "s1", ".")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitFileDiff(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.GitFileDiff(context.Background(), "s1", ".", "file.txt", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitAdd(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitAdd(context.Background(), "s1", ".", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitReset(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitReset(context.Background(), "s1", ".", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitCommit(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.GitCommit(context.Background(), "s1", ".", "msg", "", "", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitDiscard(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitDiscardChanges(context.Background(), "s1", ".", nil)
	if err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitFetch / GitPull / GitPush (local-only scenarios)
// ---------------------------------------------------------------------------

func TestLocalGitFetch_NoRemote(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-fetch-noremote"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "f.txt", "x")

	err := vol.GitFetch(ctx, sid, ".", "")
	if err == nil {
		t.Error("expected error when no remote configured")
	}
}

func TestLocalGitPull_Rebase(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-pull-rebase"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "f.txt", "x")

	_, err := vol.GitPull(ctx, sid, ".", "", true)
	if err == nil {
		t.Error("expected error for rebase=true")
	}
	if !strings.Contains(err.Error(), "rebase not supported") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLocalGitPull_NoRemote(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-pull-noremote"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "f.txt", "x")

	_, err := vol.GitPull(ctx, sid, ".", "", false)
	if err == nil {
		t.Error("expected error when no remote configured")
	}
}

func TestLocalGitPush_NoRemote(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-push-noremote"

	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "f.txt", "x")

	err := vol.GitPush(ctx, sid, ".", "", false, false)
	if err == nil {
		t.Error("expected error when no remote configured")
	}
}

func TestLocalGitFetch_WithClone(t *testing.T) {
	originDir := t.TempDir()
	originRepo := initTestRepo(t, originDir)
	commitFile(t, originRepo, originDir, "init.txt", "init")

	dir := t.TempDir()
	sid := "git-fetch-clone"
	cloneDir := filepath.Join(dir, sid)

	_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{URL: originDir})
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	vol := NewLocal(dir)
	err = vol.GitFetch(context.Background(), sid, ".", "")
	if err != nil {
		t.Fatalf("GitFetch: %v", err)
	}
}

func TestLocalGitPull_UpToDate(t *testing.T) {
	originDir := t.TempDir()
	originRepo := initTestRepo(t, originDir)
	commitFile(t, originRepo, originDir, "init.txt", "init")

	dir := t.TempDir()
	sid := "git-pull-uptodate"
	cloneDir := filepath.Join(dir, sid)

	_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{URL: originDir})
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	vol := NewLocal(dir)
	result, err := vol.GitPull(context.Background(), sid, ".", "", false)
	if err != nil {
		t.Fatalf("GitPull: %v", err)
	}
	if result.HasConflicts {
		t.Error("should not have conflicts when up to date")
	}
}

func TestLocalGitPush_SetUpstream(t *testing.T) {
	originDir := t.TempDir()
	originRepo, _ := git.PlainInit(originDir, true)
	_ = originRepo

	dir := t.TempDir()
	sid := "git-push-upstream"
	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "f.txt", "x")

	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{originDir},
	})
	if err != nil {
		t.Fatalf("CreateRemote: %v", err)
	}

	vol := NewLocal(dir)
	err = vol.GitPush(context.Background(), sid, ".", "", false, true)
	if err != nil {
		t.Fatalf("GitPush: %v", err)
	}

	cfg, _ := repo.Config()
	ref, _ := repo.Head()
	branchName := ref.Name().Short()
	bc, ok := cfg.Branches[branchName]
	if !ok {
		t.Fatal("branch config not set after push --set-upstream")
	}
	if bc.Remote != "origin" {
		t.Errorf("remote = %q, want 'origin'", bc.Remote)
	}
}

// ---------------------------------------------------------------------------
// localStorage Git — GitStatus / GitListRepos with remote fields
// ---------------------------------------------------------------------------

func TestLocalGitStatus_WithRemoteFields(t *testing.T) {
	originDir := t.TempDir()
	originRepo := initTestRepo(t, originDir)
	commitFile(t, originRepo, originDir, "init.txt", "init")

	dir := t.TempDir()
	sid := "git-status-remote-fields"
	cloneDir := filepath.Join(dir, sid)

	clonedRepo, err := git.PlainClone(cloneDir, false, &git.CloneOptions{URL: originDir})
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	_ = os.WriteFile(filepath.Join(cloneDir, "local.txt"), []byte("local"), 0o644)
	wt, _ := clonedRepo.Worktree()
	wt.Add("local.txt")
	wt.Commit("local commit", &git.CommitOptions{
		Author: &object.Signature{Name: "T", Email: "t@t.com", When: time.Now()},
	})

	vol := NewLocal(dir)
	result, err := vol.GitStatus(context.Background(), sid, ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}

	if !result.HasUpstream {
		t.Error("expected has_upstream = true")
	}
	if result.RemoteName != "origin" {
		t.Errorf("remote_name = %q", result.RemoteName)
	}
	if result.Ahead != 1 {
		t.Errorf("ahead = %d, want 1", result.Ahead)
	}
}

func TestLocalGitStatus_RemoteWithoutUpstream(t *testing.T) {
	dir := t.TempDir()
	sid := "git-status-remote-noup"
	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "init.txt", "init")

	repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/example/repo.git"},
	})

	vol := NewLocal(dir)
	result, err := vol.GitStatus(context.Background(), sid, ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if result.HasUpstream {
		t.Error("expected has_upstream = false")
	}
	if result.RemoteName != "origin" {
		t.Errorf("remote_name = %q, want 'origin'", result.RemoteName)
	}
	if result.RemoteURL == "" {
		t.Error("expected non-empty remote_url for repo with origin but no upstream")
	}
	if result.UpstreamBranch != "" {
		t.Errorf("expected empty upstream_branch, got %q", result.UpstreamBranch)
	}
}

func TestLocalGitStatus_NoRemoteNoUpstream(t *testing.T) {
	dir := t.TempDir()
	sid := "git-status-no-remote"
	repoDir := filepath.Join(dir, sid)
	repo := initTestRepo(t, repoDir)
	commitFile(t, repo, repoDir, "init.txt", "init")

	vol := NewLocal(dir)
	result, err := vol.GitStatus(context.Background(), sid, ".")
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if result.HasUpstream {
		t.Error("expected has_upstream = false")
	}
	if result.RemoteName != "" {
		t.Errorf("expected empty remote_name, got %q", result.RemoteName)
	}
	if result.RemoteURL != "" {
		t.Errorf("expected empty remote_url, got %q", result.RemoteURL)
	}
}

func TestLocalGitListRepos_WithUpstream(t *testing.T) {
	originDir := t.TempDir()
	originRepo := initTestRepo(t, originDir)
	commitFile(t, originRepo, originDir, "init.txt", "init")

	dir := t.TempDir()
	sid := "git-listrepos-upstream"
	cloneDir := filepath.Join(dir, sid)

	_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{URL: originDir})
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	vol := NewLocal(dir)
	repos, err := vol.GitListRepos(context.Background(), sid, ".")
	if err != nil {
		t.Fatalf("GitListRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if !repos[0].HasUpstream {
		t.Error("expected has_upstream = true for cloned repo")
	}
}

// ---------------------------------------------------------------------------
// Mock remote Git — new RPCs
// ---------------------------------------------------------------------------

func TestMockRemoteGitConfigGet(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	values, err := vol.GitConfigGet(context.Background(), "s1", "user.name")
	if err != nil {
		t.Fatalf("GitConfigGet: %v", err)
	}
	if values["user.name"] != "MockUser" {
		t.Errorf("user.name = %q", values["user.name"])
	}
}

func TestMockRemoteGitConfigSet(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitConfigSet(context.Background(), "s1", "user.name", "NewUser")
	if err != nil {
		t.Fatalf("GitConfigSet: %v", err)
	}
}

func TestMockRemoteGitCredentialSet(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitCredentialSet(context.Background(), "s1", "github.com", "user", "token")
	if err != nil {
		t.Fatalf("GitCredentialSet: %v", err)
	}
}

func TestMockRemoteGitCredentialList(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	entries, err := vol.GitCredentialList(context.Background(), "s1")
	if err != nil {
		t.Fatalf("GitCredentialList: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestMockRemoteGitCredentialDelete(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitCredentialDelete(context.Background(), "s1", "github.com")
	if err != nil {
		t.Fatalf("GitCredentialDelete: %v", err)
	}
}

func TestMockRemoteGitSSHKeyImport(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitSSHKeyImport(context.Background(), "s1", "mykey", "privdata", "", "github.com")
	if err != nil {
		t.Fatalf("GitSSHKeyImport: %v", err)
	}
}

func TestMockRemoteGitSSHKeyList(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	keys, err := vol.GitSSHKeyList(context.Background(), "s1")
	if err != nil {
		t.Fatalf("GitSSHKeyList: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(keys))
	}
}

func TestMockRemoteGitSSHKeyDelete(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitSSHKeyDelete(context.Background(), "s1", "mykey")
	if err != nil {
		t.Fatalf("GitSSHKeyDelete: %v", err)
	}
}

func TestMockRemoteGitFetch(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitFetch(context.Background(), "s1", ".", "origin")
	if err != nil {
		t.Fatalf("GitFetch: %v", err)
	}
}

func TestMockRemoteGitPull(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	result, err := vol.GitPull(context.Background(), "s1", ".", "origin", false)
	if err != nil {
		t.Fatalf("GitPull: %v", err)
	}
	if result.HasConflicts {
		t.Error("expected no conflicts from mock")
	}
}

func TestMockRemoteGitPush(t *testing.T) {
	mock := &mockVolumeServer{}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitPush(context.Background(), "s1", ".", "origin", false, false)
	if err != nil {
		t.Fatalf("GitPush: %v", err)
	}
}

func TestMockRemoteGitFetchFail(t *testing.T) {
	mock := &mockVolumeServer{gitOpSuccess: false, gitOpMessage: "fetch failed"}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitFetch(context.Background(), "s1", ".", "")
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteGitPullFail(t *testing.T) {
	mock := &mockVolumeServer{gitPullSuccess: false, gitPullMessage: "pull failed"}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.GitPull(context.Background(), "s1", ".", "", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteGitPushFail(t *testing.T) {
	mock := &mockVolumeServer{gitOpSuccess: false, gitOpMessage: "push failed"}
	conn, cleanup := startMockServer(t, mock)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.GitPush(context.Background(), "s1", ".", "", false, false)
	if err == nil {
		t.Error("expected error")
	}
}

// error-path remote tests for new RPCs
func TestErrRemoteGitConfigGet(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	_, err := vol.GitConfigGet(context.Background(), "s1", "")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitConfigSet(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	err := vol.GitConfigSet(context.Background(), "s1", "k", "v")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitCredentialSet(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	err := vol.GitCredentialSet(context.Background(), "s1", "h", "u", "t")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitCredentialList(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	_, err := vol.GitCredentialList(context.Background(), "s1")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitCredentialDelete(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	err := vol.GitCredentialDelete(context.Background(), "s1", "h")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitSSHKeyImport(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	err := vol.GitSSHKeyImport(context.Background(), "s1", "n", "k", "", "")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitSSHKeyList(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	_, err := vol.GitSSHKeyList(context.Background(), "s1")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitSSHKeyDelete(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	err := vol.GitSSHKeyDelete(context.Background(), "s1", "n")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitFetch(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	err := vol.GitFetch(context.Background(), "s1", ".", "")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitPull(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	_, err := vol.GitPull(context.Background(), "s1", ".", "", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteGitPush(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()
	vol := NewRemote(conn)
	err := vol.GitPush(context.Background(), "s1", ".", "", false, false)
	if err == nil {
		t.Error("expected error")
	}
}
