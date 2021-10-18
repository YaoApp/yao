package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/data"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/table"
	"github.com/yaoapp/xiang/xfs"
	"github.com/yaoapp/xun/capsule"
)

// Load 根据配置加载 API, FLow, Model, Plugin
func Load(cfg config.Config) {

	AppInit(cfg)
	DBConnect(cfg.Database)
	LoadAppInfo(cfg.Root)
	LoadEngine(cfg.Path)
	LoadApp(share.AppRoot{
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
}

// LoadAppInfo 读取应用信息
func LoadAppInfo(root string) {
	info := defaultAppInfo()
	fs := xfs.New(root)
	if fs.MustExists("/app.json") {
		err := jsoniter.Unmarshal(fs.MustReadFile("/app.json"), &info)
		if err != nil {
			exception.New("解析应用失败 %s", 500, err).Throw()
		}
	}

	if fs.MustExists("/xiang/icons/icon.icns") {
		info.Icons.Set("icns", xfs.Encode(fs.MustReadFile("/xiang/icons/icon.icns")))
	}

	if fs.MustExists("/xiang/icons/icon.ico") {
		info.Icons.Set("ico", xfs.Encode(fs.MustReadFile("/xiang/icons/icon.ico")))
	}

	if fs.MustExists("/xiang/icons/icon.png") {
		info.Icons.Set("png", xfs.Encode(fs.MustReadFile("/xiang/icons/icon.png")))
	}

	share.App = info
}

// defaultAppInfo 读取默认应用信息
func defaultAppInfo() share.AppInfo {
	info := share.AppInfo{
		Icons: maps.MakeSync(),
	}
	err := jsoniter.Unmarshal(data.MustAsset("xiang/data/app.json"), &info)
	if err != nil {
		exception.New("解析默认应用失败 %s", 500, err).Throw()
	}

	info.Icons.Set("icns", xfs.Encode(data.MustAsset("xiang/data/icons/icon.icns")))
	info.Icons.Set("ico", xfs.Encode(data.MustAsset("xiang/data/icons/icon.ico")))
	info.Icons.Set("png", xfs.Encode(data.MustAsset("xiang/data/icons/icon.png")))

	return info
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

	var scripts []share.Script
	if strings.HasPrefix(from, "fs://") || !strings.Contains(from, "://") {
		root := strings.TrimPrefix(from, "fs://")
		scripts = share.GetFilesFS(root, ".json")
	} else if strings.HasPrefix(from, "bin://") {
		root := strings.TrimPrefix(from, "bin://")
		scripts = share.GetFilesBin(root, ".json")
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
func LoadApp(app share.AppRoot) {

	// api string, flow string, model string, plugin string
	// 创建应用目录
	paths := []string{app.APIs, app.Flows, app.Models, app.Plugins, app.Charts, app.Tables, app.Screens, app.Data}
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
		scripts := share.GetAppFilesFS(root, ".json")
		for _, script := range scripts {
			// 验证API 加载逻辑
			gou.LoadAPI(string(script.Content), script.Name)
		}
	}

	// 加载Flow
	if strings.HasPrefix(app.Flows, "fs://") || !strings.Contains(app.Flows, "://") {
		root := strings.TrimPrefix(app.Flows, "fs://")
		scripts := share.GetAppFilesFS(root, ".json")
		for _, script := range scripts {
			gou.LoadFlow(string(script.Content), script.Name)
		}
	}

	// 加载Model
	if strings.HasPrefix(app.Models, "fs://") || !strings.Contains(app.Models, "://") {
		root := strings.TrimPrefix(app.Models, "fs://")
		scripts := share.GetAppFilesFS(root, ".json")
		for _, script := range scripts {
			gou.LoadModel(string(script.Content), script.Name)
		}
	}

	// 加载Plugin
	if strings.HasPrefix(app.Plugins, "fs://") || !strings.Contains(app.Plugins, "://") {
		root := strings.TrimPrefix(app.Plugins, "fs://")
		scripts := share.GetAppPlugins(root, ".so")
		for _, script := range scripts {
			gou.LoadPlugin(script.File, script.Name)
		}
	}

	// 加载Table
	if strings.HasPrefix(app.Tables, "fs://") || !strings.Contains(app.Tables, "://") {
		root := strings.TrimPrefix(app.Tables, "fs://")
		scripts := share.GetAppFilesFS(root, ".json")
		for _, script := range scripts {
			// 验证API 加载逻辑
			table.Load(string(script.Content), script.Name)
		}
	}

}
