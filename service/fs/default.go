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
