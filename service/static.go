package service

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/share"
)

// AppFileServer static file server
var AppFileServer http.Handler

// XGenFileServerV1 XGen v1.0
var XGenFileServerV1 http.Handler = http.FileServer(data.XgenV1())

// AdminRoot cache
var AdminRoot = ""

// AdminRootLen cache
var AdminRootLen = 0

// Dir files
type Dir string

// SetupStatic setup static file server
func SetupStatic() error {

	// SetAdmin Root
	adminRoot()

	// Static file server
	AppFileServer = http.FileServer(Dir(filepath.Join(config.Conf.Root, "public")))

	return nil
}

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

	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))

	// Close dir views Disable directory listing
	stat, err := os.Stat(fullName)
	if err != nil {
		return nil, mapOpenError(err, fullName, filepath.Separator, os.Stat)
	}

	if stat.IsDir() {
		indexFile := filepath.Join(fullName, "index.html")
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			return nil, mapOpenError(fs.ErrNotExist, fullName, filepath.Separator, os.Stat)
		}
	}

	f, err := os.Open(fullName)
	if err != nil {
		return nil, mapOpenError(err, fullName, filepath.Separator, os.Stat)
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

// SetupAdmin setup admin static root
func adminRoot() (string, int) {
	if AdminRoot != "" {
		return AdminRoot, AdminRootLen
	}

	adminRoot := "/yao/"
	if share.App.AdminRoot != "" {
		root := strings.TrimPrefix(share.App.AdminRoot, "/")
		root = strings.TrimSuffix(root, "/")
		adminRoot = fmt.Sprintf("/%s/", root)
	}
	adminRootLen := len(adminRoot)
	AdminRoot = adminRoot
	AdminRootLen = adminRootLen
	return AdminRoot, AdminRootLen
}
