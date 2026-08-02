package volume

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	gitMaxScanDepth  = 5
	gitMaxDiffSize   = 5 * 1024 * 1024 // 5 MB
	gitMaxCommitWalk = 1000
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (l *localStorage) openRepo(sessionID, repoPath string) (*git.Repository, string, error) {
	absPath, err := l.abs(sessionID, repoPath)
	if err != nil {
		return nil, "", err
	}
	repo, err := git.PlainOpen(absPath)
	if err != nil {
		return nil, "", err
	}
	if _, err := repo.Worktree(); err != nil {
		return nil, "", err
	}
	return repo, absPath, nil
}

func resolveHead(repo *git.Repository) (branch string, hash plumbing.Hash, isDetached, isEmpty bool) {
	ref, err := repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			isEmpty = true
			symRef, sErr := repo.Storer.Reference(plumbing.HEAD)
			if sErr == nil && symRef.Type() == plumbing.SymbolicReference {
				branch = symRef.Target().Short()
			} else {
				branch = "main"
			}
			return
		}
		branch = "unknown"
		return
	}
	hash = ref.Hash()
	if ref.Name().IsBranch() {
		branch = ref.Name().Short()
	} else {
		isDetached = true
		branch = hash.String()[:8]
	}
	return
}

func statusCodeToString(code git.StatusCode) string {
	return string(code)
}

func isBinaryContent(data []byte) bool {
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}
	for _, b := range check {
		if b == 0 {
			return true
		}
	}
	return len(data) > 0 && !utf8.Valid(data)
}

func readBlobContent(repo *git.Repository, hash plumbing.Hash) ([]byte, error) {
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return nil, err
	}
	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func blobSize(repo *git.Repository, hash plumbing.Hash) int64 {
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return 0
	}
	return blob.Size
}

func fileLanguage(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if lang, ok := extToLanguage[ext]; ok {
		return lang
	}
	return "plaintext"
}

func countAheadBehind(repo *git.Repository) (ahead, behind int) {
	ref, err := repo.Head()
	if err != nil || !ref.Name().IsBranch() {
		return 0, 0
	}
	cfg, err := repo.Config()
	if err != nil {
		return 0, 0
	}
	branchCfg, ok := cfg.Branches[ref.Name().Short()]
	if !ok || branchCfg.Remote == "" || branchCfg.Merge.String() == "" {
		return 0, 0
	}
	remoteRef, err := repo.Reference(
		plumbing.NewRemoteReferenceName(branchCfg.Remote, branchCfg.Merge.Short()),
		true,
	)
	if err != nil {
		return 0, 0
	}
	headHash := ref.Hash()
	upHash := remoteRef.Hash()
	if headHash == upHash {
		return 0, 0
	}
	headList := collectCommitHashes(repo, headHash)
	upList := collectCommitHashes(repo, upHash)
	upSet := make(map[plumbing.Hash]struct{}, len(upList))
	for _, h := range upList {
		upSet[h] = struct{}{}
	}
	headSet := make(map[plumbing.Hash]struct{}, len(headList))
	for _, h := range headList {
		headSet[h] = struct{}{}
	}
	for _, h := range headList {
		if _, ok := upSet[h]; !ok {
			ahead++
		}
	}
	for _, h := range upList {
		if _, ok := headSet[h]; !ok {
			behind++
		}
	}
	return
}

func collectCommitHashes(repo *git.Repository, from plumbing.Hash) []plumbing.Hash {
	iter, err := repo.Log(&git.LogOptions{From: from})
	if err != nil {
		return nil
	}
	defer iter.Close()
	hashes := make([]plumbing.Hash, 0, 64)
	for i := 0; i < gitMaxCommitWalk; i++ {
		c, err := iter.Next()
		if err != nil {
			break
		}
		hashes = append(hashes, c.Hash)
	}
	return hashes
}

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

func (l *localStorage) GitListRepos(_ context.Context, sessionID, basePath string) ([]GitRepo, error) {
	absBase, err := l.abs(sessionID, basePath)
	if err != nil {
		return nil, err
	}

	var repos []GitRepo

	_ = filepath.WalkDir(absBase, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		name := d.Name()

		if name == ".git" {
			repoDir := filepath.Dir(path)
			relPath, _ := filepath.Rel(absBase, repoDir)
			if relPath == "" {
				relPath = "."
			}

			repo, err := git.PlainOpen(repoDir)
			if err != nil {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			wt, err := repo.Worktree()
			if err != nil {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			branch, _, _, _ := resolveHead(repo)

			var remoteURL string
			remotes, rErr := repo.Remotes()
			if rErr == nil {
				for _, r := range remotes {
					if r.Config().Name == "origin" {
						urls := r.Config().URLs
						if len(urls) > 0 {
							remoteURL = urls[0]
						}
						break
					}
				}
			}

			hasChanges := false
			st, sErr := wt.Status()
			if sErr == nil && len(st) > 0 {
				hasChanges = true
			}

			repos = append(repos, GitRepo{
				Path:       relPath,
				Branch:     branch,
				RemoteURL:  remoteURL,
				HasChanges: hasChanges,
			})

			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			switch name {
			case "node_modules", "vendor", ".cache", "__pycache__", ".venv":
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(absBase, path)
			if rel != "." && strings.Count(rel, string(filepath.Separator)) >= gitMaxScanDepth {
				return filepath.SkipDir
			}
		}

		return nil
	})

	return repos, nil
}

func (l *localStorage) GitStatus(_ context.Context, sessionID, repoPath string) (*GitStatusResult, error) {
	repo, _, err := l.openRepo(sessionID, repoPath)
	if err != nil {
		return nil, err
	}

	branch, _, isDetached, isEmpty := resolveHead(repo)

	wt, _ := repo.Worktree()
	st, err := wt.Status()
	if err != nil {
		return nil, err
	}

	var files []GitChangedFile
	for path, fst := range st {
		if fst.Staging == git.Unmodified && fst.Worktree == git.Unmodified {
			continue
		}
		cf := GitChangedFile{
			Path:           path,
			IndexStatus:    statusCodeToString(fst.Staging),
			WorktreeStatus: statusCodeToString(fst.Worktree),
		}
		if fst.Extra != "" {
			cf.OldPath = fst.Extra
		}
		files = append(files, cf)
	}

	var ahead, behind int
	if !isEmpty && !isDetached {
		ahead, behind = countAheadBehind(repo)
	}

	return &GitStatusResult{
		Branch:     branch,
		Files:      files,
		Ahead:      ahead,
		Behind:     behind,
		IsDetached: isDetached,
		IsEmpty:    isEmpty,
	}, nil
}

func (l *localStorage) GitFileDiff(_ context.Context, sessionID, repoPath, filePath string, staged bool) (*GitFileDiffResult, error) {
	repo, absRepoPath, err := l.openRepo(sessionID, repoPath)
	if err != nil {
		return nil, err
	}

	_, headHash, _, isEmpty := resolveHead(repo)

	resp := &GitFileDiffResult{
		Language: fileLanguage(filePath),
	}

	if staged {
		if !isEmpty {
			commit, err := repo.CommitObject(headHash)
			if err == nil {
				tree, err := commit.Tree()
				if err == nil {
					f, err := tree.File(filePath)
					if err == nil {
						if f.Size > gitMaxDiffSize {
							resp.IsTooLarge = true
							return resp, nil
						}
						data, err := readBlobContent(repo, f.Hash)
						if err == nil {
							if isBinaryContent(data) {
								resp.IsBinary = true
								return resp, nil
							}
							resp.Original = string(data)
						}
					}
				}
			}
		}

		idx, err := repo.Storer.Index()
		if err == nil {
			for _, entry := range idx.Entries {
				if entry.Name == filePath {
					if blobSize(repo, entry.Hash) > gitMaxDiffSize {
						resp.IsTooLarge = true
						return resp, nil
					}
					data, err := readBlobContent(repo, entry.Hash)
					if err == nil {
						if isBinaryContent(data) {
							resp.IsBinary = true
							return resp, nil
						}
						resp.Modified = string(data)
					}
					break
				}
			}
		}
	} else {
		idx, err := repo.Storer.Index()
		if err == nil {
			for _, entry := range idx.Entries {
				if entry.Name == filePath {
					if blobSize(repo, entry.Hash) > gitMaxDiffSize {
						resp.IsTooLarge = true
						return resp, nil
					}
					data, err := readBlobContent(repo, entry.Hash)
					if err == nil {
						if isBinaryContent(data) {
							resp.IsBinary = true
							return resp, nil
						}
						resp.Original = string(data)
					}
					break
				}
			}
		}

		diskPath := filepath.Join(absRepoPath, filepath.Clean(filePath))
		if !strings.HasPrefix(diskPath, absRepoPath+string(filepath.Separator)) {
			return nil, os.ErrPermission
		}
		info, err := os.Stat(diskPath)
		if err == nil {
			if info.Size() > gitMaxDiffSize {
				resp.IsTooLarge = true
				return resp, nil
			}
			data, err := os.ReadFile(diskPath)
			if err == nil {
				if isBinaryContent(data) {
					resp.IsBinary = true
					return resp, nil
				}
				resp.Modified = string(data)
			}
		}
	}

	resp.IsNew = resp.Original == "" && resp.Modified != ""
	resp.IsDeleted = resp.Original != "" && resp.Modified == ""

	return resp, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

func (l *localStorage) GitAdd(_ context.Context, sessionID, repoPath string, files []string) error {
	repo, _, err := l.openRepo(sessionID, repoPath)
	if err != nil {
		return err
	}

	wt, _ := repo.Worktree()

	if len(files) == 0 {
		return wt.AddWithOptions(&git.AddOptions{All: true})
	}
	for _, f := range files {
		if _, err := wt.Add(f); err != nil {
			return err
		}
	}
	return nil
}

func (l *localStorage) GitReset(_ context.Context, sessionID, repoPath string, files []string) error {
	repo, _, err := l.openRepo(sessionID, repoPath)
	if err != nil {
		return err
	}

	_, headHash, _, isEmpty := resolveHead(repo)

	if len(files) == 0 {
		if isEmpty {
			idx, err := repo.Storer.Index()
			if err != nil {
				return err
			}
			idx.Entries = nil
			return repo.Storer.SetIndex(idx)
		}
		wt, _ := repo.Worktree()
		return wt.Reset(&git.ResetOptions{Commit: headHash, Mode: git.MixedReset})
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return err
	}

	resetSet := make(map[string]bool, len(files))
	for _, f := range files {
		resetSet[f] = true
	}

	if isEmpty {
		j := 0
		for _, e := range idx.Entries {
			if !resetSet[e.Name] {
				idx.Entries[j] = e
				j++
			}
		}
		idx.Entries = idx.Entries[:j]
	} else {
		commit, err := repo.CommitObject(headHash)
		if err != nil {
			return err
		}
		tree, err := commit.Tree()
		if err != nil {
			return err
		}

		j := 0
		for _, e := range idx.Entries {
			if !resetSet[e.Name] {
				idx.Entries[j] = e
				j++
				continue
			}
			delete(resetSet, e.Name)
			headFile, fErr := tree.File(e.Name)
			if fErr != nil {
				continue
			}
			e.Hash = headFile.Hash
			e.Mode = headFile.Mode
			if blob, bErr := repo.BlobObject(headFile.Hash); bErr == nil {
				e.Size = uint32(blob.Size)
			}
			idx.Entries[j] = e
			j++
		}
		idx.Entries = idx.Entries[:j]

		for name := range resetSet {
			headFile, fErr := tree.File(name)
			if fErr != nil {
				continue
			}
			blob, bErr := repo.BlobObject(headFile.Hash)
			if bErr != nil {
				continue
			}
			idx.Entries = append(idx.Entries, &index.Entry{
				Name: name,
				Hash: headFile.Hash,
				Mode: headFile.Mode,
				Size: uint32(blob.Size),
			})
		}
	}

	return repo.Storer.SetIndex(idx)
}

func (l *localStorage) GitCommit(_ context.Context, sessionID, repoPath, message, authorName, authorEmail string, allowEmpty bool) (*GitCommitResult, error) {
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("commit message required")
	}

	repo, _, err := l.openRepo(sessionID, repoPath)
	if err != nil {
		return nil, err
	}

	wt, _ := repo.Worktree()

	if !allowEmpty {
		st, err := wt.Status()
		if err != nil {
			return nil, err
		}
		hasStaged := false
		for _, fst := range st {
			if fst.Staging != git.Unmodified && fst.Staging != git.Untracked {
				hasStaged = true
				break
			}
		}
		if !hasStaged {
			return nil, fmt.Errorf("nothing to commit")
		}
	}

	if authorName == "" || authorEmail == "" {
		cfg, cErr := repo.Config()
		if cErr == nil {
			if authorName == "" && cfg.User.Name != "" {
				authorName = cfg.User.Name
			}
			if authorEmail == "" && cfg.User.Email != "" {
				authorEmail = cfg.User.Email
			}
		}
	}
	if authorName == "" {
		authorName = "Yao Workspace"
	}
	if authorEmail == "" {
		authorEmail = "workspace@yao.run"
	}

	hash, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		AllowEmptyCommits: allowEmpty,
	})
	if err != nil {
		return nil, err
	}

	return &GitCommitResult{
		CommitHash: hash.String(),
	}, nil
}

func (l *localStorage) GitDiscardChanges(_ context.Context, sessionID, repoPath string, files []string) error {
	repo, absRepoPath, err := l.openRepo(sessionID, repoPath)
	if err != nil {
		return err
	}

	_, _, _, isEmpty := resolveHead(repo)
	wt, _ := repo.Worktree()

	if len(files) == 0 {
		if isEmpty {
			idx, iErr := repo.Storer.Index()
			if iErr == nil {
				idx.Entries = nil
				_ = repo.Storer.SetIndex(idx)
			}
			return wt.Clean(&git.CleanOptions{Dir: true})
		}

		st, err := wt.Status()
		if err != nil {
			return err
		}
		idx, err := repo.Storer.Index()
		if err != nil {
			return err
		}
		idxMap := make(map[string]plumbing.Hash, len(idx.Entries))
		idxModes := make(map[string]uint32, len(idx.Entries))
		for _, e := range idx.Entries {
			idxMap[e.Name] = e.Hash
			idxModes[e.Name] = uint32(e.Mode)
		}
		for path, fst := range st {
			if fst.Worktree == git.Unmodified || fst.Worktree == git.Untracked {
				continue
			}
			hash, ok := idxMap[path]
			if !ok {
				continue
			}
			absFile := filepath.Join(absRepoPath, filepath.Clean(path))
			if !strings.HasPrefix(absFile, absRepoPath+string(filepath.Separator)) {
				continue
			}
			data, dErr := readBlobContent(repo, hash)
			if dErr != nil {
				continue
			}
			_ = os.MkdirAll(filepath.Dir(absFile), 0o755)
			perm := os.FileMode(0o644)
			if idxModes[path]&0o111 != 0 {
				perm = 0o755
			}
			_ = os.WriteFile(absFile, data, perm)
		}
		return wt.Clean(&git.CleanOptions{Dir: true})
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return err
	}

	idxMap := make(map[string]plumbing.Hash, len(idx.Entries))
	idxModes := make(map[string]uint32, len(idx.Entries))
	for _, e := range idx.Entries {
		idxMap[e.Name] = e.Hash
		idxModes[e.Name] = uint32(e.Mode)
	}

	for _, filePath := range files {
		absFile := filepath.Join(absRepoPath, filepath.Clean(filePath))
		if !strings.HasPrefix(absFile, absRepoPath+string(filepath.Separator)) {
			return os.ErrPermission
		}

		hash, tracked := idxMap[filePath]
		if tracked {
			data, dErr := readBlobContent(repo, hash)
			if dErr != nil {
				return dErr
			}
			_ = os.MkdirAll(filepath.Dir(absFile), 0o755)
			perm := os.FileMode(0o644)
			if idxModes[filePath]&0o111 != 0 {
				perm = 0o755
			}
			if err := os.WriteFile(absFile, data, perm); err != nil {
				return err
			}
		} else {
			_ = os.Remove(absFile)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Language map (extension → Monaco language identifier)
// ---------------------------------------------------------------------------

var extToLanguage = map[string]string{
	".go":         "go",
	".js":         "javascript",
	".jsx":        "javascript",
	".mjs":        "javascript",
	".ts":         "typescript",
	".tsx":        "typescript",
	".py":         "python",
	".java":       "java",
	".kt":         "kotlin",
	".kts":        "kotlin",
	".rs":         "rust",
	".rb":         "ruby",
	".php":        "php",
	".c":          "cpp",
	".h":          "cpp",
	".cpp":        "cpp",
	".cc":         "cpp",
	".cxx":        "cpp",
	".hpp":        "cpp",
	".cs":         "csharp",
	".swift":      "swift",
	".m":          "objective-c",
	".scala":      "scala",
	".dart":       "dart",
	".r":          "r",
	".lua":        "lua",
	".pl":         "perl",
	".pm":         "perl",
	".sh":         "shell",
	".bash":       "shell",
	".zsh":        "shell",
	".ps1":        "powershell",
	".bat":        "bat",
	".cmd":        "bat",
	".sql":        "sql",
	".html":       "html",
	".htm":        "html",
	".css":        "css",
	".less":       "less",
	".scss":       "scss",
	".json":       "json",
	".yaml":       "yaml",
	".yml":        "yaml",
	".xml":        "xml",
	".svg":        "xml",
	".md":         "markdown",
	".markdown":   "markdown",
	".ini":        "ini",
	".cfg":        "ini",
	".properties": "ini",
	".toml":       "ini",
	".graphql":    "graphql",
	".gql":        "graphql",
	".proto":      "protobuf",
	".clj":        "clojure",
	".cljs":       "clojure",
	".pug":        "pug",
	".hbs":        "handlebars",
	".yao":        "json",
	".gradle":     "groovy",
	".groovy":     "groovy",
	".wxml":       "xml",
	".wxss":       "css",
	".axml":       "xml",
	".acss":       "css",
}
