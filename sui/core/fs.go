package core

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
)

// SuiFile is a custom implementation of http.File
type SuiFile struct {
	reader io.Reader
	size   int64
	name   string
}

// SuiFileInfo is a custom implementation of os.FileInfo
type SuiFileInfo struct {
	size int64
	name string
}

// Open is a custom implementation of http.FileSystem
func Open(c *gin.Context, path string, name string) (http.File, error) {
	root := application.App.Root()
	pathName := filepath.Join(root, path, name)
	data := []byte(fmt.Sprintf(`SUI Server: %s`, pathName))
	return &SuiFile{
		reader: bytes.NewReader(data),
		size:   int64(len(data)),
		name:   filepath.Base(name) + ".html",
	}, nil
}

// Close is a custom implementation of the Close method for SuiFile
func (file *SuiFile) Close() error {
	file.reader = nil
	return nil
}

// Read is a custom implementation of the Read method for SuiFile
func (file *SuiFile) Read(b []byte) (n int, err error) {
	// Use the custom SuiFile reader
	return file.reader.Read(b)
}

// Seek is a custom implementation of the Seek method for SuiFile
func (file *SuiFile) Seek(offset int64, whence int) (int64, error) {
	// Use the Seek method of the underlying os.File
	return 0, nil
}

// Readdir is a custom implementation of the Readdir method for SuiFile
func (file *SuiFile) Readdir(n int) ([]os.FileInfo, error) {
	// Use the Readdir method of the underlying os.File
	return nil, nil
}

// Stat is a custom implementation of the Stat method for SuiFile
func (file *SuiFile) Stat() (os.FileInfo, error) {
	return &SuiFileInfo{size: file.size, name: file.name}, nil
}

// Size is a custom implementation of os.FileInfo
func (info *SuiFileInfo) Size() int64 {
	return info.size
}

// Name is a custom implementation of os.FileInfo
func (info *SuiFileInfo) Name() string {
	return info.name
}

// Mode is a custom implementation of os.FileInfo
func (info *SuiFileInfo) Mode() os.FileMode {
	return 0
}

// ModTime is a custom implementation of os.FileInfo
func (info *SuiFileInfo) ModTime() time.Time {
	return time.Now()
}

// IsDir is a custom implementation of os.FileInfo
func (info *SuiFileInfo) IsDir() bool {
	return false
}

// Sys is a custom implementation of os.FileInfo
func (info *SuiFileInfo) Sys() interface{} {
	return nil
}
