package common

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application/ignore"
)

// DefaultIgnorePatterns are always excluded when packing, regardless of
// whether a .yaoignore file exists. The syntax is identical to .gitignore.
var DefaultIgnorePatterns = []string{
	".git/",
	".gitignore",
	".DS_Store",
	"Thumbs.db",
	"*.swp",
	"*.swo",
	"*.bak",
	"*.tmp",
	"*.log",
	"__debug_bin*",
	".vscode/",
	".cursor/",
	".idea/",
	"node_modules/",
	".yaoignore",
}

// PackDir creates a .yao.zip from a directory. All files under dir are stored
// under the "package/" prefix in the zip. extraFiles maps additional relative
// paths (under "package/") to their absolute source paths on disk.
// The manifest is written as "package/pkg.yao".
func PackDir(dir string, manifest *PkgManifest, extraFiles map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Sync Dependencies → RawDependencies before serialization
	manifest.PrepareMarshal()

	// Write pkg.yao manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal pkg.yao: %w", err)
	}
	f, err := w.Create("package/pkg.yao")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(data); err != nil {
		return nil, err
	}

	// Load ignore rules: built-in defaults first, then .yaoignore on top so
	// that user negation patterns (e.g. !important.tmp) can override defaults.
	gi := loadIgnoreRules(filepath.Join(dir, ".yaoignore"))

	// Walk the main directory
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			if rel != "." && gi.MatchesPath(rel+"/") {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "pkg.yao" {
			return nil
		}
		if gi.MatchesPath(rel) {
			return nil
		}
		return addFileToZip(w, "package/"+rel, path)
	}); err != nil {
		return nil, fmt.Errorf("walk dir %s: %w", dir, err)
	}

	// Add extra files (e.g., scripts collected from project root)
	for relPath, absPath := range extraFiles {
		zipPath := "package/" + filepath.ToSlash(relPath)
		if err := addFileToZip(w, zipPath, absPath); err != nil {
			return nil, fmt.Errorf("add extra file %s: %w", relPath, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnpackTo extracts the "package/" contents from a .yao.zip to destDir.
// Returns a list of extracted file paths relative to destDir.
func UnpackTo(zipData []byte, destDir string) ([]string, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	var extracted []string
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		if !strings.HasPrefix(name, "package/") {
			continue
		}
		rel := strings.TrimPrefix(name, "package/")
		if rel == "" || rel == "pkg.yao" {
			continue
		}

		dest := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			return nil, err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			return nil, copyErr
		}

		extracted = append(extracted, filepath.ToSlash(rel))
	}
	return extracted, nil
}

// ReadManifest reads and parses the pkg.yao from a .yao.zip byte slice.
func ReadManifest(zipData []byte) (*PkgManifest, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	for _, f := range r.File {
		if f.Name == "package/pkg.yao" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			var m PkgManifest
			if err := json.NewDecoder(rc).Decode(&m); err != nil {
				return nil, fmt.Errorf("decode pkg.yao: %w", err)
			}
			m.NormalizeDependencies()
			return &m, nil
		}
	}
	return nil, fmt.Errorf("pkg.yao not found in zip")
}

// ExtractFile reads a single file from the zip under "package/" prefix.
func ExtractFile(zipData []byte, relPath string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	target := "package/" + relPath
	for _, f := range r.File {
		if f.Name == target {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file %q not found in zip", relPath)
}

// ListZipFiles returns all file paths in the zip under "package/" prefix,
// excluding "package/pkg.yao".
func ListZipFiles(zipData []byte) ([]string, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	var files []string
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.HasPrefix(f.Name, "package/") {
			continue
		}
		rel := strings.TrimPrefix(f.Name, "package/")
		if rel == "" || rel == "pkg.yao" {
			continue
		}
		files = append(files, rel)
	}
	return files, nil
}

// loadIgnoreRules compiles ignore patterns with defaults first, then the
// .yaoignore file contents appended so user rules (including negations) win.
func loadIgnoreRules(yaoignorePath string) *ignore.GitIgnore {
	lines := make([]string, 0, len(DefaultIgnorePatterns)+16)
	lines = append(lines, DefaultIgnorePatterns...)

	if data, err := os.ReadFile(yaoignorePath); err == nil {
		for _, l := range strings.Split(string(data), "\n") {
			lines = append(lines, l)
		}
	}
	return ignore.CompileIgnoreLines(lines...)
}

func addFileToZip(w *zip.Writer, zipPath, srcPath string) error {
	f, err := w.Create(zipPath)
	if err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(f, src)
	return err
}
