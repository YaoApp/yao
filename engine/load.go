package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/api"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/cert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/connector"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/fs"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/mcp"
	"github.com/yaoapp/yao/messenger"
	"github.com/yaoapp/yao/moapi"
	"github.com/yaoapp/yao/model"
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

// loadStep wraps a loading function with timing and progress reporting
func loadStep(name string, loadFunc func() error, callback func(string, string)) error {
	start := time.Now()
	err := loadFunc()
	duration := time.Since(start)

	if callback != nil {
		callback(name, duration.String())
	}

	return err
}

// Load application engine
func Load(cfg config.Config, options LoadOption, progressCallback ...func(string, string)) (warnings []Warning, err error) {

	defer func() { err = exception.Catch(recover()) }()

	var callback func(string, string)
	if len(progressCallback) > 0 {
		callback = progressCallback[0]
	}
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
	err = loadStep("Load Application", func() error {
		return loadApp(cfg.AppSource)
	}, callback)
	if err != nil {
		printErr(cfg.Mode, "Load Application", err)
		warnings = append(warnings, Warning{Widget: "Load Application", Error: err})
	}

	// Make Database connections
	err = loadStep("DB", func() error {
		return share.DBConnect(cfg.DB)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "DB", Error: err})
	}

	// Load Certs
	err = loadStep("Cert", func() error {
		return cert.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Cert", Error: err})
	}

	// Load Connectors
	err = loadStep("Connector", func() error {
		return connector.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Connector", Error: err})
	}

	// Load FileSystem
	err = loadStep("FileSystem", func() error {
		return fs.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "FileSystem", Error: err})
	}

	// Load i18n
	err = loadStep("i18n", func() error {
		return i18n.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "i18n", Error: err})
	}

	// start v8 runtime
	err = loadStep("Runtime", func() error {
		return runtime.Start(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Runtime", Error: err})
	}

	// Load Query Engine
	err = loadStep("Query Engine", func() error {
		return query.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Query Engine", Error: err})
	}

	// Load Scripts
	err = loadStep("Script", func() error {
		return script.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Script", Error: err})
	}

	// Load Models
	err = loadStep("Model", func() error {
		return model.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Model", Error: err})
	}

	// Load Data flows
	err = loadStep("Flow", func() error {
		return flow.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Flow", Error: err})
	}

	// Load Stores
	err = loadStep("Store", func() error {
		return store.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Store", Error: err})
	}

	// Load Uploaders
	err = loadStep("Uploader", func() error {
		return attachment.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Uploader", Error: err})
	}

	// Load Messengers
	err = loadStep("Messenger", func() error {
		return messenger.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Messenger", Error: err})
	}

	// Load Plugins
	err = loadStep("Plugin", func() error {
		return plugin.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Plugin", Error: err})
	}

	// Load WASM Application (experimental)

	// Load build-in widgets (table / form / chart / ...)
	err = loadStep("Widgets", func() error {
		return widgets.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Widgets", Error: err})
	}

	// Load Importers
	// err = importer.Load(cfg)
	// if err != nil {
	// 	// printErr(cfg.Mode, "Plugin", err)
	// 	warnings = append(warnings, Warning{Widget: "Plugin", Error: err})
	// }

	// Load Apis
	err = loadStep("API", func() error {
		return api.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "API", Error: err})
	}

	// Load Sockets
	err = loadStep("Socket", func() error {
		return socket.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Socket", Error: err})
	}

	// Load websockets (client mode)
	err = loadStep("WebSocket", func() error {
		return websocket.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "WebSocket", Error: err})
	}

	// Load tasks
	err = loadStep("Task", func() error {
		return task.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Task", Error: err})
	}

	// Load schedules
	err = loadStep("Schedule", func() error {
		return schedule.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Schedule", Error: err})
	}

	// Load AIGC
	err = loadStep("AIGC", func() error {
		return aigc.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "AIGC", Error: err})
	}

	// Load Custom Widget
	err = loadStep("Widget", func() error {
		return widget.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Widget", Error: err})
	}

	// Load Custom Widget Instances
	err = loadStep("Widget Instances", func() error {
		return widget.LoadInstances()
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Widget", Error: err})
	}

	// Load SUI
	err = loadStep("SUI", func() error {
		return sui.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "SUI", Error: err})
	}

	// Load Moapi
	err = loadStep("Moapi", func() error {
		return moapi.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Moapi", Error: err})
	}

	// Load Pipe
	err = loadStep("Pipe", func() error {
		return pipe.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Pipe", Error: err})
	}

	// Load MCP Clients
	err = loadStep("MCP", func() error {
		return mcp.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "MCP", Error: err})
	}

	// Load Knowledge Base
	err = loadStep("Knowledge Base", func() error {
		_, err := kb.Load(cfg)
		return err
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Knowledge Base", Error: err})
	}

	// Load Agent
	err = loadStep("Agent", func() error {
		return agent.Load(cfg)
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "Agent", Error: err})
	}

	for name, hook := range LoadHooks {
		err = hook(cfg)
		if err != nil {
			// printErr(cfg.Mode, name, err)
			warnings = append(warnings, Warning{Widget: name, Error: err})
		}
	}

	// Load OpenAPI
	err = loadStep("OpenAPI", func() error {
		_, err := openapi.Load(cfg)
		return err
	}, callback)
	if err != nil {
		warnings = append(warnings, Warning{Widget: "OpenAPI", Error: err})
	}

	// Execute AfterLoad Process if exists
	if share.App.AfterLoad != "" && !options.IgnoredAfterLoad {
		err = loadStep("AfterLoad", func() error {
			p, err := process.Of(share.App.AfterLoad, options)
			if err != nil {
				return err
			}
			_, err = p.Exec()
			return err
		}, callback)
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

	// Load Uploaders
	err = attachment.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Uploader", err)
	}

	// Load Messengers
	err = messenger.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Messenger", err)
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

	// Load MCP Clients
	err = mcp.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "MCP", err)
	}

	// Load Knowledge Base
	_, err = kb.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Knowledge Base", err)

	}

	// Load Agent
	err = agent.Load(cfg)
	if err != nil {
		printErr(cfg.Mode, "Agent", err)
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
