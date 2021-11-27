package engine

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/api"
	"github.com/yaoapp/xiang/app"
	"github.com/yaoapp/xiang/chart"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/flow"
	"github.com/yaoapp/xiang/model"
	"github.com/yaoapp/xiang/page"
	"github.com/yaoapp/xiang/plugin"
	"github.com/yaoapp/xiang/query"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/table"
	"github.com/yaoapp/xiang/workflow"
)

// Load 根据配置加载 API, FLow, Model, Plugin
func Load(cfg config.Config) {

	share.DBConnect(cfg.Database) // 创建数据库连接

	app.Load(cfg) // 加载应用信息
	LoadEngine(cfg.Path)
	query.Load(cfg) // 加载数据分析引擎

	share.Load(cfg)    // 加载共享库 lib
	model.Load(cfg)    // 加载数据模型 model
	flow.Load(cfg)     // 加载业务逻辑 Flow
	plugin.Load(cfg)   // 加载业务插件 plugin
	table.Load(cfg)    // 加载数据表格 table
	chart.Load(cfg)    // 加载分析图表 chart
	page.Load(cfg)     // 加载页面 page
	workflow.Load(cfg) // 加载工作流  workflow
	api.Load(cfg)      // 加载业务接口 API

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
