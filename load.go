package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

// Script 脚本文件类型
type Script struct {
	Name    string
	Type    string
	Content []byte
	File    string
}

// Load 根据配置加载 API, FLow, Model, Plugin
func Load(cfg Config) {
	LoadEngine(cfg.Path)
	LoadApp(cfg.RootAPI, cfg.RootFLow, cfg.RootModel, cfg.RootPlugin)
}

// LoadEngine 加载引擎的 API, Flow, Model 配置
func LoadEngine(from string) {

	var scripts []Script
	if strings.HasPrefix(from, "fs://") || !strings.Contains(from, "://") {
		root := strings.TrimPrefix(from, "fs://")
		scripts = getFilesFS(root, ".json")
	} else if strings.HasPrefix(from, "bin://") {
		root := strings.TrimPrefix(from, "bin://")
		scripts = getFilesBin(root, ".json")
	}

	if scripts == nil {
		exception.New("读取文件失败", 500, from).Throw()
	}

	if len(scripts) == 0 {
		exception.New("读取文件失败, 未找到任何可执行脚本", 500, from).Throw()
	}

	// 加载 API, Flow, Models
	for _, script := range scripts {
		switch script.Type {
		case "models":
			gou.LoadModel(string(script.Content), "xiang."+script.Name)
			break
		case "flows":
			gou.LoadFlow(string(script.Content), "xiang."+script.Name)
			break
		case "api":
			gou.LoadAPI(string(script.Content), "xiang."+script.Name)
			break
		}
	}
}

// LoadApp 加载应用的 API, Flow, Model 和 Plugin
func LoadApp(api string, flow string, model string, plugin string) {

	// 加载API
	if strings.HasPrefix(api, "fs://") || !strings.Contains(api, "://") {
		root := strings.TrimPrefix(api, "fs://")
		scripts := getAppFilesFS(root, ".json")
		for _, script := range scripts {
			gou.LoadAPI(string(script.Content), script.Name)
		}
	}

	// 加载Flow
	if strings.HasPrefix(flow, "fs://") || !strings.Contains(flow, "://") {
		root := strings.TrimPrefix(flow, "fs://")
		scripts := getAppFilesFS(root, ".json")
		for _, script := range scripts {
			gou.LoadFlow(string(script.Content), script.Name)
		}
	}

	// 加载Model
	if strings.HasPrefix(model, "fs://") || !strings.Contains(model, "://") {
		root := strings.TrimPrefix(model, "fs://")
		scripts := getAppFilesFS(root, ".json")
		for _, script := range scripts {
			gou.LoadModel(string(script.Content), script.Name)
		}
	}

	// 加载Plugin
	if strings.HasPrefix(plugin, "fs://") || !strings.Contains(plugin, "://") {
		root := strings.TrimPrefix(plugin, "fs://")
		scripts := getAppPlugins(root, ".so")
		for _, script := range scripts {
			gou.LoadPlugin(script.File, script.Name)
		}
	}
}

// / getAppPluins 遍历应用目录，读取文件列表
func getAppPlugins(root string, typ string) []Script {
	files := []Script{}
	root = path.Join(root, "/")
	filepath.Walk(root, func(filepath string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(filepath, typ) {
			filename := strings.TrimPrefix(filepath, root+"/")
			namer := strings.Split(filename, ".")
			nametypes := strings.Split(namer[0], "/")
			name := strings.Join(nametypes, ".")
			files = append(files, Script{
				Name: name,
				Type: "plugin",
				File: filepath,
			})
		}
		return nil
	})
	return files
}

// getAppFilesFS 遍历应用目录，读取文件列表
func getAppFilesFS(root string, typ string) []Script {
	files := []Script{}
	root = path.Join(root, "/")
	filepath.Walk(root, func(filepath string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(filepath, typ) {
			filename := strings.TrimPrefix(filepath, root+"/")

			namer := strings.Split(filename, ".")
			nametypes := strings.Split(namer[0], "/")
			name := strings.Join(nametypes, ".")

			file, err := os.Open(filepath)
			if err != nil {
				exception.Err(err, 500).Throw()
			}

			defer file.Close()
			content, err := ioutil.ReadAll(file)
			if err != nil {
				exception.Err(err, 500).Throw()
			}
			files = append(files, Script{
				Name:    name,
				Type:    "app",
				Content: content,
			})
		}
		return nil
	})
	return files
}

// getFilesFS 遍历目录，读取文件列表
func getFilesFS(root string, typ string) []Script {
	files := []Script{}
	root = path.Join(root, "/")
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(path, typ) {
			filename := strings.TrimPrefix(path, root+"/")
			name, typ := getTypeName(filename)

			file, err := os.Open(path)
			if err != nil {
				exception.Err(err, 500).Throw()
			}

			defer file.Close()
			content, err := ioutil.ReadAll(file)
			if err != nil {
				exception.Err(err, 500).Throw()
			}
			files = append(files, Script{
				Name:    name,
				Type:    typ,
				Content: content,
			})
		}
		return nil
	})
	return files
}

// getFilesBin 从 bindata 中读取文件列表
func getFilesBin(root string, typ string) []Script {
	files := []Script{}
	binfiles := AssetNames()
	for _, path := range binfiles {
		if strings.HasSuffix(path, typ) {
			file := strings.TrimPrefix(path, root+"/")
			name, typ := getTypeName(file)
			content, err := Asset(path)
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

func getTypeName(path string) (name string, typ string) {
	namer := strings.Split(path, ".")
	nametypes := strings.Split(namer[0], "/")
	name = strings.Join(nametypes[1:], ".")
	typ = nametypes[0]

	return name, typ
}
