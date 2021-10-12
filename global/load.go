package global

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/data"
	"github.com/yaoapp/xiang/table"
	"github.com/yaoapp/xun/capsule"
)

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

// Load 根据配置加载 API, FLow, Model, Plugin
func Load(cfg config.Config) {

	AppInit(cfg)
	DBConnect(cfg.Database)
	LoadAppInfo(cfg.Root)
	LoadEngine(cfg.Path)
	LoadApp(AppRoot{
		APIs:    cfg.RootAPI,
		Flows:   cfg.RootFLow,
		Models:  cfg.RootModel,
		Plugins: cfg.RootPlugin,
		Tables:  cfg.RootTable,
		Charts:  cfg.RootChart,
		Screens: cfg.RootScreen,
		Data:    cfg.RootData,
	})

	// 加密密钥函数
	gou.LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, cfg.Database.AESKey), "AES")
	gou.LoadCrypt(`{}`, "PASSWORD")

	// 设定已加载配置
	Conf = cfg
}

// Reload 根据配置重新加载 API, FLow, Model, Plugin
func Reload(cfg config.Config) {
	gou.APIs = map[string]*gou.API{}
	gou.Models = map[string]*gou.Model{}
	gou.Flows = map[string]*gou.Flow{}
	gou.Plugins = map[string]*gou.Plugin{}
	Load(cfg)
}

// DBConnect 建立数据库连接
func DBConnect(dbconfig config.DatabaseConfig) {

	// 连接主库
	for i, dsn := range dbconfig.Primary {
		db := capsule.AddConn("primary", dbconfig.Driver, dsn)
		if i == 0 {
			db.SetAsGlobal()
		}
	}

	// 连接从库
	for _, dsn := range dbconfig.Secondary {
		capsule.AddReadConn("secondary", dbconfig.Driver, dsn)
	}
}

// AppInit 应用初始化
func AppInit(cfg config.Config) {

	if _, err := os.Stat(cfg.RootUI); os.IsNotExist(err) {
		err := os.MkdirAll(cfg.RootUI, os.ModePerm)
		if err != nil {
			log.Panicf("创建目录失败(%s) %s", cfg.RootUI, err)
		}
	}

	if _, err := os.Stat(cfg.RootDB); os.IsNotExist(err) {
		err := os.MkdirAll(cfg.RootDB, os.ModePerm)
		if err != nil {
			log.Panicf("创建目录失败(%s) %s", cfg.RootDB, err)
		}
	}

	if _, err := os.Stat(cfg.RootData); os.IsNotExist(err) {
		err := os.MkdirAll(cfg.RootData, os.ModePerm)
		if err != nil {
			log.Panicf("创建目录失败(%s) %s", cfg.RootData, err)
		}
	}
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

	// 加载 API, Flow, Models, Table, Chart, Screens
	for _, script := range scripts {
		switch script.Type {
		case "models":
			gou.LoadModel(string(script.Content), "xiang."+script.Name)
			break
		case "flows":
			gou.LoadFlow(string(script.Content), "xiang."+script.Name)
			break
		case "apis":
			gou.LoadAPI(string(script.Content), "xiang."+script.Name)
			break
		}
	}

	// 加载数据应用
	for _, script := range scripts {
		switch script.Type {
		case "tables":
			table.Load(string(script.Content), "xiang."+script.Name)
			break
		}
	}
}

// LoadApp 加载应用的 API, Flow, Model 和 Plugin
func LoadApp(app AppRoot) {

	// api string, flow string, model string, plugin string
	// 创建应用目录
	paths := []string{app.APIs, app.Flows, app.Models, app.Plugins, app.Charts, app.Screens, app.Data}
	for _, p := range paths {
		if !strings.HasPrefix(p, "fs://") && strings.Contains(p, "://") {
			continue
		}
		root, err := filepath.Abs(strings.TrimPrefix(p, "fs://"))
		if err != nil {
			log.Panicf("创建目录失败(%s) %s", root, err)
		}

		if _, err := os.Stat(root); os.IsNotExist(err) {
			err := os.MkdirAll(root, os.ModePerm)
			if err != nil {
				log.Panicf("创建目录失败(%s) %s", root, err)
			}
		}
	}

	// 加载API
	if strings.HasPrefix(app.APIs, "fs://") || !strings.Contains(app.APIs, "://") {
		root := strings.TrimPrefix(app.APIs, "fs://")
		scripts := getAppFilesFS(root, ".json")
		for _, script := range scripts {
			// 验证API 加载逻辑
			gou.LoadAPI(string(script.Content), script.Name)
		}
	}

	// 加载Flow
	if strings.HasPrefix(app.Flows, "fs://") || !strings.Contains(app.Flows, "://") {
		root := strings.TrimPrefix(app.Flows, "fs://")
		scripts := getAppFilesFS(root, ".json")
		for _, script := range scripts {
			gou.LoadFlow(string(script.Content), script.Name)
		}
	}

	// 加载Model
	if strings.HasPrefix(app.Models, "fs://") || !strings.Contains(app.Models, "://") {
		root := strings.TrimPrefix(app.Models, "fs://")
		scripts := getAppFilesFS(root, ".json")
		for _, script := range scripts {
			gou.LoadModel(string(script.Content), script.Name)
		}
	}

	// 加载Plugin
	if strings.HasPrefix(app.Plugins, "fs://") || !strings.Contains(app.Plugins, "://") {
		root := strings.TrimPrefix(app.Plugins, "fs://")
		scripts := getAppPlugins(root, ".so")
		for _, script := range scripts {
			gou.LoadPlugin(script.File, script.Name)
		}
	}

	// 加载Table
	if strings.HasPrefix(app.Tables, "fs://") || !strings.Contains(app.Tables, "://") {
		root := strings.TrimPrefix(app.Tables, "fs://")
		scripts := getAppFilesFS(root, ".json")
		for _, script := range scripts {
			// 验证API 加载逻辑
			table.Load(string(script.Content), script.Name)
		}
	}

}

// / getAppPluins 遍历应用目录，读取文件列表
func getAppPlugins(root string, typ string) []Script {
	files := []Script{}
	root = path.Join(root, "/")
	filepath.Walk(root, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			exception.Err(err, 500).Throw()
			return err
		}
		if strings.HasSuffix(file, typ) {
			files = append(files, getAppPluginFile(root, file))
		}
		return nil
	})
	return files
}

// getAppPluginFile 读取文件
func getAppPluginFile(root string, file string) Script {
	name := getAppPluginFileName(root, file)
	return Script{
		Name: name,
		Type: "plugin",
		File: file,
	}
}

// getAppFile 读取文件
func getAppPluginFileName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	nametypes := strings.Split(namer[0], "/")
	name := strings.Join(nametypes, ".")
	return name
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
			files = append(files, getAppFile(root, filepath))
		}

		return nil
	})
	return files
}

// getAppFile 读取文件
func getAppFile(root string, filepath string) Script {
	name := getAppFileName(root, filepath)
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

// getAppFile 读取文件
func getAppFileName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	nametypes := strings.Split(namer[0], "/")
	name := strings.Join(nametypes, ".")
	return name
}

// getAppFileBaseName 读取文件base
func getAppFileBaseName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	return filepath.Join(root, namer[0])
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
			files = append(files, getFile(root, path))
		}
		return nil
	})
	return files
}

// getFile 读取文件
func getFile(root string, path string) Script {
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
	return Script{
		Name:    name,
		Type:    typ,
		Content: content,
	}
}

// getFileName 读取文件
func getFileName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	name, _ := getTypeName(filename)
	return name
}

// getFileBaseName 读取文件base
func getFileBaseName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/")
	namer := strings.Split(filename, ".")
	return filepath.Join(root, namer[0])
}

// getFilesBin 从 bindata 中读取文件列表
func getFilesBin(root string, typ string) []Script {
	files := []Script{}
	binfiles := data.AssetNames()
	for _, path := range binfiles {
		if strings.HasSuffix(path, typ) {
			file := strings.TrimPrefix(path, root+"/")
			name, typ := getTypeName(file)
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

func getTypeName(path string) (name string, typ string) {
	namer := strings.Split(path, ".")
	nametypes := strings.Split(namer[0], "/")
	name = strings.Join(nametypes[1:], ".")
	typ = nametypes[0]
	return name, typ
}
