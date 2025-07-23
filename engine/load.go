package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/api"
	"github.com/yaoapp/yao/cert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/connector"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/fs"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/moapi"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/neo"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/pack"
	"github.com/yaoapp/yao/pipe"
	"github.com/yaoapp/yao/plugin"
	"github.com/yaoapp/yao/query"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/schedule"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/socket"
	"github.com/yaoapp/yao/store"
	sui "github.com/yaoapp/yao/sui/api"
	"github.com/yaoapp/yao/task"
	"github.com/yaoapp/yao/websocket"
	"github.com/yaoapp/yao/widget"
	"github.com/yaoapp/yao/widgets"
)

// LoadHooks used to load custom widgets/processes
var LoadHooks = map[string]func(config.Config) error{}
var envRe = regexp.MustCompile(`\$ENV\.([0-9a-zA-Z_-]+)`)

// RegisterLoadHook register custom load hook
func RegisterLoadHook(name string, hook func(config.Config) error) error {
	if _, ok := LoadHooks[name]; ok {
		return fmt.Errorf("load hook %s already exists", name)
	}
	LoadHooks[name] = hook
	return nil
}

// LoadOption the load option
type LoadOption struct {
	Action           string `json:"action"`
	IgnoredAfterLoad bool   `json:"ignoredAfterLoad"`
	IsReload         bool   `json:"reload"`
}

// Warning the warning
type Warning struct {
	Widget string
	Error  error
}

// Load application engine
func Load(cfg config.Config, options LoadOption) (warnings []Warning, err error) {

	defer func() { err = exception.Catch(recover()) }()
	exception.Mode = cfg.Mode

	// SET XGEN_BASE
	adminRoot := "yao"
	if share.App.Optional != nil {
		if root, has := share.App.Optional["adminRoot"]; has {
			adminRoot = fmt.Sprintf("%v", root)
		}
	}
	os.Setenv("XGEN_BASE", adminRoot)

	// load the application
	err = loadApp(cfg.AppSource)
	if err != nil {
		printErr(cfg.Mode, "Load Application", err)
		warnings = append(warnings, Warning{Widget: "Load Application", Error: err})
	}

	// Make Database connections
	err = share.DBConnect(cfg.DB)
	if err != nil {
		// printErr(cfg.Mode, "DB", err)
		warnings = append(warnings, Warning{Widget: "DB", Error: err})
	}

	// Load Certs
	err = cert.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Cert", err)
		warnings = append(warnings, Warning{Widget: "Cert", Error: err})
	}

	// Load Connectors
	err = connector.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Connector", err)
		warnings = append(warnings, Warning{Widget: "Connector", Error: err})
	}

	// Load FileSystem
	err = fs.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "FileSystem", err)
		warnings = append(warnings, Warning{Widget: "FileSystem", Error: err})
	}

	// Load i18n
	err = i18n.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "i18n", err)
		warnings = append(warnings, Warning{Widget: "i18n", Error: err})
	}

	// start v8 runtime
	err = runtime.Start(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Runtime", err)
		warnings = append(warnings, Warning{Widget: "Runtime", Error: err})
	}

	// Load Query Engine
	err = query.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Query Engine", err)
		warnings = append(warnings, Warning{Widget: "Query Engine", Error: err})
	}

	// Load Scripts
	err = script.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Script", err)
		warnings = append(warnings, Warning{Widget: "Script", Error: err})
	}

	// Load Models
	err = model.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Model", err)
		warnings = append(warnings, Warning{Widget: "Model", Error: err})
	}

	// Load Data flows
	err = flow.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Flow", err)
		warnings = append(warnings, Warning{Widget: "Flow", Error: err})
	}

	// Load Stores
	err = store.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Store", err)
		warnings = append(warnings, Warning{Widget: "Store", Error: err})
	}

	// Load Plugins
	err = plugin.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Plugin", err)
		warnings = append(warnings, Warning{Widget: "Plugin", Error: err})
	}

	// Load WASM Application (experimental)

	// Load build-in widgets (table / form / chart / ...)
	err = widgets.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Widgets", err)
		warnings = append(warnings, Warning{Widget: "Widgets", Error: err})
	}

	// Load Importers
	// err = importer.Load(cfg)
	// if err != nil {
	// 	// printErr(cfg.Mode, "Plugin", err)
	// 	warnings = append(warnings, Warning{Widget: "Plugin", Error: err})
	// }

	// Load Apis
	err = api.Load(cfg) // 加载业务接口 API
	if err != nil {
		// printErr(cfg.Mode, "API", err)
		warnings = append(warnings, Warning{Widget: "API", Error: err})
	}

	// Load Sockets
	err = socket.Load(cfg) // Load sockets
	if err != nil {
		// printErr(cfg.Mode, "Socket", err)
		warnings = append(warnings, Warning{Widget: "Socket", Error: err})
	}

	// Load websockets (client mode)
	err = websocket.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "WebSocket", err)
		warnings = append(warnings, Warning{Widget: "WebSocket", Error: err})
	}

	// Load tasks
	err = task.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Task", err)
		warnings = append(warnings, Warning{Widget: "Task", Error: err})
	}

	// Load schedules
	err = schedule.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Schedule", err)
		warnings = append(warnings, Warning{Widget: "Schedule", Error: err})
	}

	// Load AIGC
	err = aigc.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "AIGC", err)
		warnings = append(warnings, Warning{Widget: "AIGC", Error: err})
	}

	// Load Custom Widget
	err = widget.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Widget", err)
		warnings = append(warnings, Warning{Widget: "Widget", Error: err})
	}

	// Load Custom Widget Instances
	err = widget.LoadInstances()
	if err != nil {
		// printErr(cfg.Mode, "Widget", err)
		warnings = append(warnings, Warning{Widget: "Widget", Error: err})
	}

	// Load SUI
	err = sui.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "SUI", err)
		warnings = append(warnings, Warning{Widget: "SUI", Error: err})
	}

	// Load Moapi
	err = moapi.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Moapi", err)
		warnings = append(warnings, Warning{Widget: "Moapi", Error: err})
	}

	// Load Pipe
	err = pipe.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Pipe", err)
		warnings = append(warnings, Warning{Widget: "Pipe", Error: err})
	}

	// Load Neo
	err = neo.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Neo", err)
		warnings = append(warnings, Warning{Widget: "Neo", Error: err})
	}

	for name, hook := range LoadHooks {
		err = hook(cfg)
		if err != nil {
			// printErr(cfg.Mode, name, err)
			warnings = append(warnings, Warning{Widget: name, Error: err})
		}
	}

	// Load Knowledge Base
	_, err = kb.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "Knowledge Base", err)
		warnings = append(warnings, Warning{Widget: "Knowledge Base", Error: err})
	}

	// Load OpenAPI
	_, err = openapi.Load(cfg)
	if err != nil {
		// printErr(cfg.Mode, "OpenAPI", err)
		warnings = append(warnings, Warning{Widget: "OpenAPI", Error: err})
	}

	// Execute AfterLoad Process if exists
	if share.App.AfterLoad != "" && !options.IgnoredAfterLoad {
		p, err := process.Of(share.App.AfterLoad, options)
		if err != nil {
			printErr(cfg.Mode, "AfterLoad", err)
			warnings = append(warnings, Warning{Widget: "AfterLoad", Error: err})
			return warnings, err
		}

		_, err = p.Exec()
		if err != nil {
			printErr(cfg.Mode, "AfterLoad", err)
			warnings = append(warnings, Warning{Widget: "AfterLoad", Error: err})
			return warnings, err
		}
	}

	return warnings, nil
}

// Unload application engine
func Unload() (err error) {
	defer func() { err = exception.Catch(recover()) }()

	// Stop Runtime
	err = runtime.Stop()

	// Close DB
	err = share.DBClose()

	// Close Query Engine
	err = query.Unload()

	// Close Connectors
	err = connector.Unload()

	// Recycle
	// api
	// models
	// flows
	// stores
	// scripts
	// connectors
	// filesystem
	// i18n
	// certs
	// plugins
	// importers
	// tasks
	// schedules
	// sockets
	// websockets
	// widgets
	// custom widget

	return err
}

// Reload the application engine
func Reload(cfg config.Config, options LoadOption) (err error) {

	defer func() { err = exception.Catch(recover()) }()
	exception.Mode = cfg.Mode

	// SET XGEN_BASE
	adminRoot := "yao"
	if share.App.Optional != nil {
		if root, has := share.App.Optional["adminRoot"]; has {
			adminRoot = fmt.Sprintf("%v", root)
		}
	}
	os.Setenv("XGEN_BASE", adminRoot)

	// load the application
	err = loadApp(cfg.AppSource)
	if err != nil {
		printErr(cfg.Mode, "Load Application", err)
	}

	// Load Certs
	err = cert.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Cert", err)
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

	// Load Query Engine
	err = query.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Query Engine", err)
	}

	// Load Scripts
	err = script.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Script", err)
	}

	// Load Models
	err = model.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Model", err)
	}

	// Load Data flows
	err = flow.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Flow", err)
	}

	// Load Stores
	err = store.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Store", err)
	}

	// Load Plugins
	err = plugin.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Plugin", err)
	}

	// Load WASM Application (experimental)

	// Load build-in widgets (table / form / chart / ...)
	err = widgets.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Widgets", err)
	}

	// Load Apis
	err = api.Load(cfg) // 加载业务接口 API
	if err != nil {
		printErr(cfg.Mode, "API", err)
	}

	// Load Sockets
	err = socket.Load(cfg) // Load sockets
	if err != nil {
		printErr(cfg.Mode, "Socket", err)
	}

	// Load websockets (client mode)
	err = websocket.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "WebSocket", err)
	}

	// Load tasks
	err = task.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Task", err)
	}

	// Load schedules
	err = schedule.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Schedule", err)
	}

	// Load Custom Widget
	err = widget.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Widget", err)
	}

	// Load AIGC
	err = aigc.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "AIGC", err)
	}

	// Load Neo
	err = neo.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Neo", err)
	}

	// Load OpenAPI
	_, err = openapi.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "OpenAPI", err)
	}

	// Execute AfterLoad Process if exists
	if share.App.AfterLoad != "" && !options.IgnoredAfterLoad {
		options.IsReload = true
		p, err := process.Of(share.App.AfterLoad, options)
		if err != nil {
			printErr(cfg.Mode, "AfterLoad", err)
			return err
		}

		_, err = p.Exec()
		if err != nil {
			printErr(cfg.Mode, "AfterLoad", err)
			return err
		}
	}

	return err
}

// Restart the application engine
func Restart(cfg config.Config, options LoadOption) error {
	err := Unload()
	if err != nil {
		return err
	}

	warnings, err := Load(cfg, options)
	if err != nil {
		return err
	}

	if len(warnings) > 0 {
		for _, warning := range warnings {
			printErr(cfg.Mode, warning.Widget, warning.Error)
		}
	}

	return nil
}

// loadApp load the application from bindata / pkg / disk
func loadApp(root string) error {

	var err error
	var app application.Application

	if share.BUILDIN {

		file, err := os.Executable()
		if err != nil {
			return err
		}

		// Load from cache
		app, err := application.OpenFromYazCache(file, pack.Cipher)

		if err != nil {

			// load from bin
			reader, err := data.ReadApp()
			if err != nil {
				return err
			}

			app, err = application.OpenFromYaz(reader, file, pack.Cipher) // Load app from Bin
			if err != nil {
				return err
			}
		}

		application.Load(app)
		config.Init() // Reset Config
		data.RemoveApp()

	} else if strings.HasSuffix(root, ".yaz") {
		app, err = application.OpenFromYazFile(root, pack.Cipher) // Load app from .yaz file
		if err != nil {
			return err
		}
		application.Load(app)
		config.Init() // Reset Config

	} else {
		app, err = application.OpenFromDisk(root) // Load app from Disk
		if err != nil {
			return err
		}
		application.Load(app)
	}

	var appData []byte
	var appFile string

	// Read app setting
	if has, _ := application.App.Exists("app.yao"); has {
		appFile = "app.yao"
		appData, err = application.App.Read("app.yao")
		if err != nil {
			return err
		}

	} else if has, _ := application.App.Exists("app.jsonc"); has {
		appFile = "app.jsonc"
		appData, err = application.App.Read("app.jsonc")
		if err != nil {
			return err
		}

	} else if has, _ := application.App.Exists("app.json"); has {
		appFile = "app.json"
		appData, err = application.App.Read("app.json")
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("app.yao or app.jsonc or app.json does not exists")
	}

	// Replace $ENV with os.Getenv
	appData = envRe.ReplaceAllFunc(appData, func(s []byte) []byte {
		key := string(s[5:])
		val := os.Getenv(key)
		if val == "" {
			return s
		}
		return []byte(val)
	})

	// Parse app.yao
	share.App = share.AppInfo{}
	err = application.Parse(appFile, appData, &share.App)
	if err != nil {
		return err
	}

	// Set default prefix
	if share.App.Prefix == "" {
		share.App.Prefix = "yao_"
	}

	return nil
}

func printErr(mode, widget string, err error) {
	message := fmt.Sprintf("[%s] %s", widget, err.Error())
	if !strings.Contains(message, "does not exists") && !strings.Contains(message, "no such file or directory") && mode == "development" {
		color.Red(message)
	}
}
