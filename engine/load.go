package engine

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/api"
	"github.com/yaoapp/xiang/app"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/flow"
	"github.com/yaoapp/xiang/model"
	"github.com/yaoapp/xiang/plugin"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/table"
)

// Load 根据配置加载 API, FLow, Model, Plugin
func Load(cfg config.Config) {

	share.DBConnect(cfg.Database) // 创建数据库连接

	app.Load(cfg) // 加载应用信息
	LoadEngine(cfg.Path)

	model.Load(cfg)  // 加载数据模型 model
	api.Load(cfg)    // 加载业务接口 API
	flow.Load(cfg)   // 加载业务逻辑 Flow
	plugin.Load(cfg) // 加载业务插件 plugin
	table.Load(cfg)  // 加载数据表格 table

	// LoadApp(share.AppRoot{
	// 	APIs:    cfg.RootAPI,
	// 	Flows:   cfg.RootFLow,
	// 	Models:  cfg.RootModel,
	// 	Plugins: cfg.RootPlugin,
	// 	Tables:  cfg.RootTable,
	// 	Charts:  cfg.RootChart,
	// 	Screens: cfg.RootScreen,
	// 	Data:    cfg.RootData,
	// })

	// 加密密钥函数
	gou.LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, cfg.Database.AESKey), "AES")
	gou.LoadCrypt(`{}`, "PASSWORD")
}

// Reload 根据配置重新加载 API, FLow, Model, Plugin
func Reload(cfg config.Config) {
	gou.APIs = map[string]*gou.API{}
	gou.Models = map[string]*gou.Model{}
	gou.Flows = map[string]*gou.Flow{}
	gou.Plugins = map[string]*gou.Plugin{}
	Load(cfg)
}

// // DBConnect 建立数据库连接
// func DBConnect(dbconfig config.DatabaseConfig) {

// 	// 连接主库
// 	for i, dsn := range dbconfig.Primary {
// 		db := capsule.AddConn("primary", dbconfig.Driver, dsn)
// 		if i == 0 {
// 			db.SetAsGlobal()
// 		}
// 	}

// 	// 连接从库
// 	for _, dsn := range dbconfig.Secondary {
// 		capsule.AddReadConn("secondary", dbconfig.Driver, dsn)
// 	}
// }

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
			table.LoadTable(string(script.Content), "xiang."+script.Name)
			break
		}
	}
}

// LoadApp 加载应用的 API, Flow, Model 和 Plugin
func LoadApp(app share.AppRoot) {

	// api string, flow string, model string, plugin string
	// 创建应用目录
	// paths := []string{app.APIs, app.Flows, app.Models, app.Plugins, app.Charts, app.Tables, app.Screens, app.Data}
	// paths := []string{app.Flows, app.Models, app.Plugins, app.Charts, app.Tables, app.Screens, app.Data}
	// for _, p := range paths {
	// 	if !strings.HasPrefix(p, "fs://") && strings.Contains(p, "://") {
	// 		continue
	// 	}
	// 	root, err := filepath.Abs(strings.TrimPrefix(p, "fs://"))
	// 	if err != nil {
	// 		log.Panicf("创建目录失败(%s) %s", root, err)
	// 	}

	// 	if _, err := os.Stat(root); os.IsNotExist(err) {
	// 		err := os.MkdirAll(root, os.ModePerm)
	// 		if err != nil {
	// 			log.Panicf("创建目录失败(%s) %s", root, err)
	// 		}
	// 	}
	// }

	// // 加载API
	// if strings.HasPrefix(app.APIs, "fs://") || !strings.Contains(app.APIs, "://") {
	// 	root := strings.TrimPrefix(app.APIs, "fs://")
	// 	scripts := share.GetAppFilesFS(root, ".json")
	// 	for _, script := range scripts {
	// 		// 验证API 加载逻辑
	// 		gou.LoadAPI(string(script.Content), script.Name)
	// 	}
	// }

	// // 加载Flow
	// if strings.HasPrefix(app.Flows, "fs://") || !strings.Contains(app.Flows, "://") {
	// 	root := strings.TrimPrefix(app.Flows, "fs://")
	// 	scripts := share.GetAppFilesFS(root, ".json")
	// 	for _, script := range scripts {
	// 		gou.LoadFlow(string(script.Content), script.Name)
	// 	}
	// }

	// // 加载Model
	// if strings.HasPrefix(app.Models, "fs://") || !strings.Contains(app.Models, "://") {
	// 	root := strings.TrimPrefix(app.Models, "fs://")
	// 	scripts := share.GetAppFilesFS(root, ".json")
	// 	for _, script := range scripts {
	// 		gou.LoadModel(string(script.Content), script.Name)
	// 	}
	// }

	// 加载Plugin
	// if strings.HasPrefix(app.Plugins, "fs://") || !strings.Contains(app.Plugins, "://") {
	// 	root := strings.TrimPrefix(app.Plugins, "fs://")
	// 	scripts := share.GetAppPlugins(root, ".so")
	// 	for _, script := range scripts {
	// 		gou.LoadPlugin(script.File, script.Name)
	// 	}
	// }

	// 加载Table
	// if strings.HasPrefix(app.Tables, "fs://") || !strings.Contains(app.Tables, "://") {
	// 	root := strings.TrimPrefix(app.Tables, "fs://")
	// 	scripts := share.GetAppFilesFS(root, ".json")
	// 	for _, script := range scripts {
	// 		// 验证API 加载逻辑
	// 		table.Load(string(script.Content), script.Name)
	// 	}
	// }

}
