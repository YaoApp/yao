package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/api"
	"github.com/yaoapp/yao/app"
	"github.com/yaoapp/yao/cert"
	"github.com/yaoapp/yao/chart"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/connector"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/fs"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/importer"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/page"
	"github.com/yaoapp/yao/plugin"
	"github.com/yaoapp/yao/query"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/schedule"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/socket"
	"github.com/yaoapp/yao/store"
	"github.com/yaoapp/yao/studio"
	"github.com/yaoapp/yao/table"
	"github.com/yaoapp/yao/task"
	"github.com/yaoapp/yao/websocket"
	"github.com/yaoapp/yao/widget"
	"github.com/yaoapp/yao/widgets"
)

// Load 根据配置加载 API, FLow, Model, Plugin
func Load(cfg config.Config) (err error) {
	defer func() { err = exception.Catch(recover()) }()

	// Load Runtime
	err = runtime.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Runtime", err)
	}

	// 加载应用信息
	// 第一步: 加载应用信息
	app.Load(cfg)

	// 加密密钥函数
	gou.LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, cfg.DB.AESKey), "AES")
	gou.LoadCrypt(`{}`, "PASSWORD")

	// Load Certs
	err = cert.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Cert", err)
	}

	// Load connectors
	err = connector.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Connector", err)
	}

	// Load FileSystem
	err = fs.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "FileSystem", err)
	}

	// Load i18n
	err = i18n.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "i18n", err)
	}

	// Load Studio development mode only
	if cfg.Mode == "development" {
		err = studio.Load(cfg)
		if err != nil {
			printErr(cfg.Mode, "Studio", err)
		}
	}

	// 第二步: 建立数据库 & 会话连接
	err = share.DBConnect(cfg.DB) // 创建数据库连接
	if err != nil {
		printErr(cfg.Mode, "DB", err)
	}

	// share.SessionConnect(cfg.Session) // 创建会话服务器链接

	// 加载应用引擎
	if os.Getenv("YAO_DEV") != "" {
		LoadEngine(filepath.Join(os.Getenv("YAO_DEV"), "/yao"))
	} else {
		LoadEngine()
	}

	// 第三步: 加载数据分析引擎
	query.Load(cfg) // 加载数据分析引擎

	// 第四步: 加载共享库 & JS 处理器
	err = share.Load(cfg) // 加载共享库 lib
	if err != nil {
		printErr(cfg.Mode, "Lib", err)
	}

	err = script.Load(cfg) // 加载JS处理器 script
	if err != nil {
		printErr(cfg.Mode, "Script", err)
	}

	// 第五步: 加载数据模型等
	err = model.Load(cfg) // 加载数据模型 model
	if err != nil {
		printErr(cfg.Mode, "Model", err)
	}

	err = flow.Load(cfg) // 加载业务逻辑 Flow
	if err != nil {
		printErr(cfg.Mode, "Flow", err)
	}

	err = store.Load(cfg) // Load stores
	if err != nil {
		printErr(cfg.Mode, "Store", err)
	}

	err = plugin.Load(cfg) // 加载业务插件 plugin
	if err != nil {
		printErr(cfg.Mode, "Plugin", err)
	}

	// XGEN 1.0
	if share.App.XGen == "1.0" {

		// SET XGEN_BASE
		// adminRoot := "yao"
		// if share.App.Optional != nil {
		// 	if root, has := share.App.Optional["adminRoot"]; has {
		// 		adminRoot = fmt.Sprintf("%v", root)
		// 	}
		// }
		// os.Setenv("XGEN_BASE", adminRoot)

		// Load build-in widgets
		err = widgets.Load(cfg)
		if err != nil {
			printErr(cfg.Mode, "Widgets", err)
		}

		delete(gou.APIs, "xiang.table")
		delete(gou.APIs, "xiang.page")
		delete(gou.APIs, "xiang.chart")
		delete(gou.APIs, "xiang.xiang")
		delete(gou.APIs, "xiang.user")
		delete(gou.APIs, "xiang.storage")

	} else { // old version
		err = table.Load(cfg) // 加载数据表格 table
		if err != nil {
			printErr(cfg.Mode, "Table", err)
		}

		err = chart.Load(cfg) // 加载分析图表 chart
		if err != nil {
			printErr(cfg.Mode, "Chart", err)
		}

		err = page.Load(cfg) // 加载页面 page 忽略错误
		if err != nil {
			printErr(cfg.Mode, "Page", err)
		}
	}

	importer.Load(cfg) // 加载数据导入 imports

	err = api.Load(cfg) // 加载业务接口 API
	if err != nil {
		printErr(cfg.Mode, "API", err)
	}

	err = socket.Load(cfg) // Load sockets
	if err != nil {
		printErr(cfg.Mode, "Socket", err)
	}

	err = websocket.Load(cfg) // Load websockets (client)
	if err != nil {
		printErr(cfg.Mode, "WebSocket", err)
	}

	err = task.Load(cfg) // Load tasks
	if err != nil {
		printErr(cfg.Mode, "Task", err)
	}

	err = schedule.Load(cfg) // Load schedules
	if err != nil {
		printErr(cfg.Mode, "Schedule", err)
	}

	err = widget.Load(cfg) // Load widgets
	if err != nil {
		printErr(cfg.Mode, "Widget", err)
	}

	return nil
}

func printErr(mode, widget string, err error) {
	message := fmt.Sprintf("[%s] %s", widget, err.Error())
	if !strings.Contains(message, "does not exists") && mode == "development" {
		color.Red(message)
	}
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
func LoadEngine(from ...string) {
	var scripts []share.Script
	if len(from) > 0 {
		scripts = share.GetFilesFS(from[0], ".json")
	} else {
		scripts = share.GetFilesBin("yao", ".json")
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
