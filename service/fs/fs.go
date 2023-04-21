package fs

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application"
)

// Dir http root path
type Dir string

// DirPWA is the PWA path
type DirPWA string

// Open implements FileSystem using os.Open, opening files for reading rooted
// and relative to the directory d.
func (d Dir) Open(name string) (http.File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}

	dir := string(d)
	if dir == "" {
		dir = "."
	}

	name = filepath.FromSlash(path.Clean("/" + name))
	relName := filepath.Join(dir, name)

	// Close dir views Disable directory listing
	absName := filepath.Join(application.App.Root(), relName)
	stat, err := os.Stat(absName)
	if err != nil {
		return nil, mapOpenError(err, relName, filepath.Separator, os.Stat)
	}

	if stat.IsDir() {
		if _, err := os.Stat(filepath.Join(absName, "index.html")); os.IsNotExist(err) {
			return nil, mapOpenError(fs.ErrNotExist, relName, filepath.Separator, os.Stat)
		}
	}

	f, err := application.App.FS(string(d)).Open(name)
	if err != nil {
		return nil, mapOpenError(err, relName, filepath.Separator, os.Stat)
	}

	return f, nil
}

// Open implements FileSystem using os.Open, opening files for reading rooted
// and relative to the directory d.
func (d DirPWA) Open(name string) (http.File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}

	dir := string(d)
	if dir == "" {
		dir = "."
	}

	name = filepath.FromSlash(path.Clean("/" + name))
	relName := filepath.Join(dir, name)

	if filepath.Ext(relName) == "" && relName != dir {
		relName = filepath.Join(dir, "index.html")
		name = filepath.Join(string(os.PathSeparator), "index.html")
	}

	// Close dir views Disable directory listing
	absName := filepath.Join(application.App.Root(), relName)
	stat, err := os.Stat(absName)
	if err != nil {
		return nil, mapOpenError(err, relName, filepath.Separator, os.Stat)
	}

	if stat.IsDir() {
		if _, err := os.Stat(filepath.Join(absName, "index.html")); os.IsNotExist(err) {
			return nil, mapOpenError(fs.ErrNotExist, relName, filepath.Separator, os.Stat)
		}
	}

	f, err := application.App.FS(string(d)).Open(name)
	if err != nil {
		return nil, mapOpenError(err, relName, filepath.Separator, os.Stat)
	}

	return f, nil
}

// mapOpenError maps the provided non-nil error from opening name
// to a possibly better non-nil error. In particular, it turns OS-specific errors
// about opening files in non-directories into fs.ErrNotExist. See Issues 18984 and 49552.
func mapOpenError(originalErr error, name string, sep rune, stat func(string) (fs.FileInfo, error)) error {
	if errors.Is(originalErr, fs.ErrNotExist) || errors.Is(originalErr, fs.ErrPermission) {
		return originalErr
	}

	parts := strings.Split(name, string(sep))
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		fi, err := stat(strings.Join(parts[:i+1], string(sep)))
		if err != nil {
			return originalErr
		}
		if !fi.IsDir() {
			return fs.ErrNotExist
		}
	}
	return originalErr
}
