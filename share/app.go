package share

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xiang/data"
)

// App 应用信息
var App AppInfo

// AppInfo 应用信息
type AppInfo struct {
	Name        string                 `json:"name,omitempty"`
	Short       string                 `json:"short,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Description string                 `json:"description,omitempty"`
	Icons       maps.MapStrSync        `json:"icons,omitempty"`
	Storage     AppStorage             `json:"storage,omitempty"`
	Option      map[string]interface{} `json:"option,omitempty"`
}

// AppStorage 应用存储
type AppStorage struct {
	Default string                 `json:"default"`
	Buckets map[string]string      `json:"buckets,omitempty"`
	S3      map[string]interface{} `json:"s3,omitempty"`
	OSS     *AppStorageOSS         `json:"oss,omitempty"`
	COS     map[string]interface{} `json:"cos,omitempty"`
}

// AppStorageOSS 阿里云存储
type AppStorageOSS struct {
	Endpoint    string `json:"endpoint,omitempty"`
	ID          string `json:"id,omitempty"`
	Secret      string `json:"secret,omitempty"`
	RoleArn     string `json:"roleArn,omitempty"`
	SessionName string `json:"sessionName,omitempty"`
}

// Script 脚本文件类型
type Script struct {
	Name    string
	Type    string
	Content []byte
	File    string
}

// AppRoot 应用目录
type AppRoot struct {
	APIs    string
	Flows   string
	Models  string
	Plugins string
	Tables  string
	Charts  string
	Screens string
	Data    string
}

// Public 输出公共信息
func (app AppInfo) Public() AppInfo {
	app.Storage.COS = nil
	app.Storage.OSS = nil
	app.Storage.S3 = nil
	return app
}

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
