package share

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/exception"
)

// Walk 遍历应用目录，读取文件列表
func Walk(root string, typeName string, cb func(root, filename string)) {
	root = strings.TrimPrefix(root, "fs://")
	root = strings.TrimPrefix(root, "file://")
	root = path.Join(root, "/")
	filepath.Walk(root, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(filename, typeName) {
			cb(root, filename)
		}
		return nil
	})
}

// SpecName 解析名称  root: "/tests/apis"  file: "/tests/apis/foo/bar.http.json"
func SpecName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/") // "foo/bar.http.json"
	namer := strings.Split(filename, ".")          // ["foo/bar", "http", "json"]
	nametypes := strings.Split(namer[0], "/")      // ["foo", "bar"]
	name := strings.Join(nametypes, ".")           // "foo.bar"
	return name
}

// ReadFile 读取文件
func ReadFile(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return content
}

// DirNotExists 校验目录是否存在
func DirNotExists(dir string) bool {
	dir = strings.TrimPrefix(dir, "fs://")
	dir = strings.TrimPrefix(dir, "file://")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return true
	}
	return false
}
