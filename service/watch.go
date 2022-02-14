package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/chart"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/page"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/table"
	"github.com/yaoapp/yao/workflow"
)

// Watch 监听应用目录文件变更
func Watch(cfg config.Config) {
	if os.Getenv("YAO_DEV") != "" {
		WatchEngine(filepath.Join(os.Getenv("YAO_DEV"), "/yao"))
	}
	WatchModel(filepath.Join(cfg.Root, "models"), "")
	WatchAPI(filepath.Join(cfg.Root, "apis"), "")
	WatchFlow(filepath.Join(cfg.Root, "flows"), "")
	WatchPlugin(filepath.Join(cfg.Root, "plugins"))
	WatchTable(filepath.Join(cfg.Root, "tables"), "")
	WatchChart(filepath.Join(cfg.Root, "charts"), "")
	WatchPage(filepath.Join(cfg.Root, "pages"), "")
	WatchWorkFlow(filepath.Join(cfg.Root, "workflows"), "")

	// 看板大屏
	WatchPage(filepath.Join(cfg.Root, "kanban"), "")
	WatchPage(filepath.Join(cfg.Root, "screen"), "")

	// 监听脚本 & libs更新
	WatchGlobal(filepath.Join(cfg.Root, "libs"))
	WatchGlobal(filepath.Join(cfg.Root, "scripts"))
}

// WatchEngine 监听监听引擎内建数据变更
func WatchEngine(root string) {
	root = share.DirAbs(root)
	WatchModel(filepath.Join(root, "models"), "xiang.")
	WatchAPI(filepath.Join(root, "apis"), "xiang.")
	WatchFlow(filepath.Join(root, "flows"), "xiang.")
	WatchTable(filepath.Join(root, "tables"), "xiang.")
}

// WatchGlobal 监听通用程序更新
func WatchGlobal(root string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {
		if !strings.HasSuffix(filename, ".json") && !strings.HasSuffix(filename, ".js") {
			return
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			err := engine.Load(config.Conf)
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}
			StopWithouttSession(func() {
				fmt.Println(color.GreenString("Service Restarted"))
				go StartWithouttSession()
			})
		}
	})
}

// WatchModel 监听业务接口更新
func WatchModel(root string, prefix string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {

		if !strings.HasSuffix(filename, ".json") {
			return
		}
		if op == "write" || op == "create" {
			name := prefix + share.SpecName(root, filename)
			content := share.ReadFile(filename)
			_, err := gou.LoadModelReturn(string(content), name) // Reload
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}
			fmt.Println(color.GreenString("Model %s Reloaded", name))
		} else if op == "remove" || op == "rename" {
			name := prefix + share.SpecName(root, filename)
			if _, has := gou.Models[name]; has {
				delete(gou.Models, name)
				fmt.Println(color.RedString("Model %s Removed", name))
			}
		}
	})
}

// WatchAPI 监听业务接口更新
func WatchAPI(root string, prefix string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {

		if !strings.HasSuffix(filename, ".json") {
			return
		}

		if op == "write" || op == "create" {
			name := prefix + share.SpecName(root, filename)
			content := share.ReadFile(filename)
			_, err := gou.LoadAPIReturn(string(content), name) // Reload
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}
			fmt.Println(color.GreenString("API %s Reloaded", name))

		} else if op == "remove" || op == "rename" {
			name := prefix + share.SpecName(root, filename)
			if _, has := gou.APIs[name]; has {
				delete(gou.APIs, name)
				fmt.Println(color.RedString("API %s Removed", name))
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			StopWithouttSession(func() {
				fmt.Println(color.GreenString("Service Restarted"))
				go StartWithouttSession()
			})
		}
	})
}

// WatchFlow 监听业务逻辑变更
func WatchFlow(root string, prefix string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {
		if !strings.HasSuffix(filename, ".json") && !strings.HasSuffix(filename, ".js") {
			return
		}

		if strings.HasSuffix(filename, ".js") {
			name := prefix + share.SpecName(root, filename)
			name = strings.ReplaceAll(name, ".", "/")
			filename = filepath.Join(root, name+".flow.json")
		}

		if op == "write" || op == "create" {
			name := prefix + share.SpecName(root, filename)
			content := share.ReadFile(filename)
			flow, err := gou.LoadFlowReturn(string(content), name) // Reload
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}

			if flow != nil { // Reload Script
				dir := filepath.Dir(filename)
				share.Walk(dir, ".js", func(root, filename string) {
					script := share.ScriptName(filename)
					content := share.ReadFile(filename)
					_, err := flow.LoadScriptReturn(string(content), script)
					if err != nil {
						fmt.Println(color.RedString("Fatal: %s", err.Error()))
						return
					}
				})
			}

			fmt.Println(color.GreenString("Flow %s Reloaded", name))

		} else if op == "remove" || op == "rename" {
			name := prefix + share.SpecName(root, filename)
			if _, has := gou.Flows[name]; has {
				delete(gou.Flows, name)
				fmt.Println(color.RedString("Flow %s Removed", name))
			}
		}
	})
}

// WatchPlugin 监听业务插件变更
func WatchPlugin(root string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {

		if !strings.HasSuffix(filename, ".so") {
			return
		}

		if op == "write" || op == "create" {
			name := share.SpecName(root, filename)
			_, err := gou.LoadPluginReturn(filename, name) // Reload
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}
			fmt.Println(color.GreenString("Plugin %s Reloaded", name))

		} else if op == "remove" || op == "rename" {
			name := share.SpecName(root, filename)
			if _, has := gou.Plugins[name]; has {
				delete(gou.Plugins, name)
				fmt.Println(color.RedString("Plugin %s Removed", name))
			}
		}
	})
}

// WatchTable 监听数据表格更新
func WatchTable(root string, prefix string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {

		if !strings.HasSuffix(filename, ".json") {
			return
		}

		if op == "write" || op == "create" {
			name := prefix + share.SpecName(root, filename)
			content := share.ReadFile(filename)
			_, err := table.LoadTable(string(content), name) // Reload Table
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}

			api, has := gou.APIs["xiang.table"]
			if has {
				_, err := gou.LoadAPIReturn(api.Source, api.Name)
				if err != nil {
					fmt.Println(color.RedString("Fatal: %s", err.Error()))
					return
				}
			}
			fmt.Println(color.GreenString("Table %s Reloaded", name))

		} else if op == "remove" || op == "rename" {
			name := prefix + share.SpecName(root, filename)
			if _, has := table.Tables[name]; has {
				delete(table.Tables, name)
				fmt.Println(color.RedString("Table %s Removed", name))
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			StopWithouttSession(func() {
				fmt.Println(color.GreenString("Service Restarted"))
				go StartWithouttSession()
			})
		}
	})
}

// WatchChart 监听分析图表更新
func WatchChart(root string, prefix string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {
		if !strings.HasSuffix(filename, ".json") && !strings.HasSuffix(filename, ".js") {
			return
		}

		if strings.HasSuffix(filename, ".js") {
			name := prefix + share.SpecName(root, filename)
			name = strings.ReplaceAll(name, ".", "/")
			filename = filepath.Join(root, name+".chart.json")
		}

		if op == "write" || op == "create" {
			name := prefix + share.SpecName(root, filename)
			content := share.ReadFile(filename)
			chart, err := chart.LoadChart(content, name) // Relaod
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}

			if chart != nil { // Reload Script
				dir := filepath.Dir(filename)
				share.Walk(dir, ".js", func(root, filename string) {
					script := share.ScriptName(filename)
					content := share.ReadFile(filename)
					_, err := chart.LoadScriptReturn(string(content), script)
					if err != nil {
						fmt.Println(color.RedString("Fatal: %s", err.Error()))
						return
					}
				})
			}

			api, has := gou.APIs["xiang.chart"]
			if has {
				_, err := gou.LoadAPIReturn(api.Source, api.Name)
				if err != nil {
					fmt.Println(color.RedString("Fatal: %s", err.Error()))
					return
				}
			}
			fmt.Println(color.GreenString("Chart %s Reloaded", name))

		} else if op == "remove" || op == "rename" {
			name := prefix + share.SpecName(root, filename)
			if _, has := chart.Charts[name]; has {
				delete(chart.Charts, name)
				fmt.Println(color.RedString("Chart %s Removed", name))
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			StopWithouttSession(func() {
				fmt.Println(color.GreenString("Service Restarted"))
				go StartWithouttSession()
			})
		}
	})
}

// WatchPage 监听页面更新
func WatchPage(root string, prefix string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {
		if !strings.HasSuffix(filename, ".json") && !strings.HasSuffix(filename, ".js") {
			return
		}

		if strings.HasSuffix(filename, ".js") {
			name := prefix + share.SpecName(root, filename)
			name = strings.ReplaceAll(name, ".", "/")
			filename = filepath.Join(root, name+".page.json")
		}

		if op == "write" || op == "create" {
			name := prefix + share.SpecName(root, filename)
			content := share.ReadFile(filename)
			page, err := page.LoadPage(content, name) // Relaod
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}

			if page != nil { // Reload Script
				dir := filepath.Dir(filename)
				share.Walk(dir, ".js", func(root, filename string) {
					script := share.ScriptName(filename)
					content := share.ReadFile(filename)
					_, err := page.LoadScriptReturn(string(content), script)
					if err != nil {
						fmt.Println(color.RedString("Fatal: %s", err.Error()))
						return
					}
				})
			}

			api, has := gou.APIs["xiang.page"]
			if has {
				_, err := gou.LoadAPIReturn(api.Source, api.Name)
				if err != nil {
					fmt.Println(color.RedString("Fatal: %s", err.Error()))
					return
				}
			}
			fmt.Println(color.GreenString("Page %s Reloaded", name))

		} else if op == "remove" || op == "rename" {
			name := prefix + share.SpecName(root, filename)
			if _, has := page.Pages[name]; has {
				delete(page.Pages, name)
				fmt.Println(color.RedString("Page %s Removed", name))
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			StopWithouttSession(func() {
				fmt.Println(color.GreenString("Service Restarted"))
				go StartWithouttSession()
			})
		}
	})
}

// WatchWorkFlow 监听工作流更新
func WatchWorkFlow(root string, prefix string) {
	if share.DirNotExists(root) {
		return
	}
	root = share.DirAbs(root)
	go share.Watch(root, func(op string, filename string) {
		if !strings.HasSuffix(filename, ".json") {
			return
		}

		if op == "write" || op == "create" {
			name := prefix + share.SpecName(root, filename)
			content := share.ReadFile(filename)
			_, err := workflow.LoadWorkFlow(content, name) // Relaod
			if err != nil {
				fmt.Println(color.RedString("Fatal: %s", err.Error()))
				return
			}

			api, has := gou.APIs["xiang.workflow."+name]
			if has {
				_, err := gou.LoadAPIReturn(api.Source, api.Name)
				if err != nil {
					fmt.Println(color.RedString("Fatal: %s", err.Error()))
					return
				}
			}
			fmt.Println(color.GreenString("Workflow %s Reloaded", name))

		} else if op == "remove" || op == "rename" {
			name := prefix + share.SpecName(root, filename)
			if _, has := workflow.WorkFlows[name]; has {
				delete(workflow.WorkFlows, name)
				fmt.Println(color.RedString("Workflow %s Removed", name))
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			StopWithouttSession(func() {
				fmt.Println(color.GreenString("Service Restarted"))
				go StartWithouttSession()
			})
		}
	})
}
