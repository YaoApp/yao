package share

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/data"
)

// Walk 遍历应用目录，读取文件列表
func Walk(root string, typeName string, cb func(root, filename string)) error {
	root = strings.TrimPrefix(root, "fs://")
	root = strings.TrimPrefix(root, "file://")
	root = path.Join(root, "/")
	err := filepath.Walk(root, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			log.With(log.F{"root": root, "type": typeName, "filename": filename}).Error(err.Error())
			return err
		}
		if strings.HasSuffix(filename, typeName) {
			cb(root, filename)
		}
		return nil
	})
	return err
}

// ID parse unique name root: "/tests/apis"  file: "/tests/apis/foo/bar.http.json"
func ID(root string, file string) string {
	return SpecName(root, file)
}

// File ID to file
func File(id string, ext string) string {
	ext = strings.TrimLeft(ext, ".")
	file := strings.ReplaceAll(id, ".", string(os.PathSeparator))
	return fmt.Sprintf("%s.%s", file, ext)
}

// SpecName 解析名称  root: "/tests/apis"  file: "/tests/apis/foo/bar.http.json"
func SpecName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/") // "foo/bar.http.json"
	namer := strings.Split(filename, ".")          // ["foo/bar", "http", "json"]
	nametypes := strings.Split(namer[0], "/")      // ["foo", "bar"]
	name := strings.Join(nametypes, ".")           // "foo.bar"
	return name
}

// ScriptName 解析数据处理脚本名称
func ScriptName(filename string) string {
	filename = strings.TrimSuffix(filename, ".js")
	namer := strings.Split(filename, ".") // ["foo/bar", "http", "json"]
	if len(namer) < 2 {
		return namer[0]
	}
	return namer[len(namer)-1]
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

// DirAbs 文件绝对路径
func DirAbs(dir string) string {
	dir = strings.TrimPrefix(dir, "fs://")
	dir = strings.TrimPrefix(dir, "file://")
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		log.Panic("获取绝对路径错误 %s %s", dir, err)
	}
	return dirAbs
}

// ************************************************
// 警告: 以下函数将被弃用
// ************************************************

// GetAppPlugins 遍历应用目录，读取文件列表
func GetAppPlugins(root string, typ string) []Script {
	files := []Script{}
	root = path.Join(root, "/")
	filepath.Walk(root, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(file, typ) {
			files = append(files, GetAppPluginFile(root, file))
		}
		return nil
	})
	return files
}

// GetAppPluginFile 读取文件
func GetAppPluginFile(root string, file string) Script {
	name := GetAppPluginFileName(root, file)
	return Script{
		Name: name,
		Type: "plugin",
		File: file,
	}
}

// GetAppPluginFileName 读取文件
func GetAppPluginFileName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	nametypes := strings.Split(namer[0], "/")
	name := strings.Join(nametypes, ".")
	return name
}

// GetAppFilesFS 遍历应用目录，读取文件列表
func GetAppFilesFS(root string, typ string) []Script {
	files := []Script{}
	root = path.Join(root, "/")
	filepath.Walk(root, func(filepath string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(filepath, typ) {
			files = append(files, GetAppFile(root, filepath))
		}

		return nil
	})
	return files
}

// GetAppFile 读取文件
func GetAppFile(root string, filepath string) Script {
	name := GetAppFileName(root, filepath)
	file, err := os.Open(filepath)
	if err != nil {
		exception.Err(err, 500).Throw()
	}

	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return Script{
		Name:    name,
		Type:    "app",
		Content: content,
	}
}

// GetAppFileName 读取文件
func GetAppFileName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	nametypes := strings.Split(namer[0], "/")
	name := strings.Join(nametypes, ".")
	return name
}

// GetAppFileBaseName 读取文件base
func GetAppFileBaseName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	return filepath.Join(root, namer[0])
}

// GetFilesFS 遍历目录，读取文件列表
func GetFilesFS(root string, typ string) []Script {
	files := []Script{}
	root = path.Join(root, "/")
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(path, typ) {
			files = append(files, GetFile(root, path))
		}
		return nil
	})
	return files
}

// GetFile 读取文件
func GetFile(root string, path string) Script {
	filename := strings.TrimPrefix(path, root+"/")
	name, typ := GetTypeName(filename)
	file, err := os.Open(path)
	if err != nil {
		exception.Err(err, 500).Throw()
	}

	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return Script{
		Name:    name,
		Type:    typ,
		Content: content,
	}
}

// GetFileName 读取文件
func GetFileName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	name, _ := GetTypeName(filename)
	return name
}

// GetFileBaseName 读取文件base
func GetFileBaseName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	return filepath.Join(root, namer[0])
}

// GetFilesBin 从 bindata 中读取文件列表
func GetFilesBin(root string, typ string) []Script {
	files := []Script{}
	binfiles := data.AssetNames()
	for _, path := range binfiles {
		if strings.HasSuffix(path, typ) {
			file := strings.TrimPrefix(path, root+"/")
			name, typ := GetTypeName(file)
			content, err := data.Asset(path)
			if err != nil {
				exception.Err(err, 500).Throw()
			}
			files = append(files, Script{
				Name:    name,
				Type:    typ,
				Content: content,
			})
		}
	}
	return files
}

// GetTypeName 读取类型
func GetTypeName(path string) (name string, typ string) {
	namer := strings.Split(path, ".")
	nametypes := strings.Split(namer[0], "/")
	name = strings.Join(nametypes[1:], ".")
	typ = nametypes[0]
	return name, typ
}
