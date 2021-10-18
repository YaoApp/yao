package service

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/table"
)

var shutdown = make(chan bool)
var shutdownComplete = make(chan bool)

// Start 启动服务
func Start() {
	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host:   config.Conf.Service.Host,
			Port:   config.Conf.Service.Port,
			Allows: config.Conf.Service.Allow,
			Root:   "/api",
		},
		&shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)
}

// Stop 关闭服务
func Stop(onComplete func()) {
	shutdown <- true
	<-shutdownComplete
	onComplete()
}

// WatchChanges 监听配置文件变更
func WatchChanges() {
	watchEngine(config.Conf.Path)
	watchApp(share.AppRoot{
		APIs:    config.Conf.RootAPI,
		Flows:   config.Conf.RootFLow,
		Models:  config.Conf.RootModel,
		Plugins: config.Conf.RootPlugin,
		Tables:  config.Conf.RootTable,
		Charts:  config.Conf.RootChart,
		Screens: config.Conf.RootScreen,
	})
}

// watchEngine 监听引擎目录文件变更
func watchEngine(from string) {
	if !strings.HasPrefix(from, "fs://") && strings.Contains(from, "://") {
		return
	}
	root := strings.TrimPrefix(from, "fs://")
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		log.Panicf("路径错误 %s %s", root, err)
	}

	// 监听 flows (这里应该重构)
	go share.Watch(filepath.Join(rootAbs, "flows"), func(op string, file string) {

		if !strings.HasSuffix(file, ".json") {
			return
		}

		if strings.HasSuffix(file, ".js") {
			basName := share.GetFileBaseName(root, file)
			file = basName + ".flow.json"
		}

		if op == "write" || op == "create" {
			script := share.GetFile(root, file)
			gou.LoadFlow(string(script.Content), "xiang."+script.Name) // Reload
			log.Printf("Flow %s 已重新加载完毕", "xiang."+script.Name)
		} else if op == "remove" || op == "rename" {
			name := "xiang." + share.GetFileName(root, file)
			if _, has := gou.Flows[name]; has {
				delete(gou.Flows, name)
				log.Printf("Flow %s 已经移除", name)
			}
		}
	})

	// 监听 models
	go share.Watch(filepath.Join(rootAbs, "models"), func(op string, file string) {

		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op == "write" || op == "create" {
			script := share.GetFile(root, file)
			gou.LoadModel(string(script.Content), "xiang."+script.Name) // Reload
			log.Printf("Model %s 已重新加载完毕", "xiang."+script.Name)
		} else if op == "remove" || op == "rename" {
			name := "xiang." + share.GetFileName(root, file)
			if _, has := gou.Models[name]; has {
				delete(gou.Models, name)
				log.Printf("Model %s 已经移除", name)
			}
		}
	})

	// 监听 apis
	go share.Watch(filepath.Join(rootAbs, "apis"), func(op string, file string) {

		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op == "write" || op == "create" {
			script := share.GetFile(root, file)
			gou.LoadAPI(string(script.Content), "xiang."+script.Name) // Reload
			log.Printf("API %s 已重新加载完毕", "xiang."+script.Name)

			// 打印最新API信息
			api := gou.APIs["xiang."+script.Name]
			log.Printf("%s(%d)", api.Name, len(api.HTTP.Paths))
			for _, p := range api.HTTP.Paths {
				log.Println(p.Method, filepath.Join("/api", api.HTTP.Group, p.Path), "\tprocess:", p.Process)
			}

		} else if op == "remove" || op == "rename" {
			name := "xiang." + share.GetFileName(root, file)
			if _, has := gou.APIs[name]; has {
				delete(gou.APIs, name)
				log.Printf("API %s 已经移除", name)
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			Stop(func() {
				log.Printf("服务器重启完毕")
				go Start()
			})
		}
	})

	// 监听 tables
	go share.Watch(filepath.Join(rootAbs, "tables"), func(op string, file string) {

		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op == "write" || op == "create" {
			script := share.GetFile(root, file)
			table.Load(string(script.Content), "xiang."+script.Name) // Reload
			api, has := gou.APIs["xiang.table"]
			if has {
				api.Reload()
			}

			log.Printf("数据表格 %s 已重新加载完毕", "xiang."+script.Name)

		} else if op == "remove" || op == "rename" {
			name := "xiang." + share.GetFileName(root, file)
			if _, has := table.Tables[name]; has {
				delete(table.Tables, name)
				log.Printf("数据表格 %s 已经移除", name)
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			Stop(func() {
				log.Printf("服务器重启完毕")
				go Start()
			})
		}
	})
}

// watchApp 监听应用目录文件变更
func watchApp(app share.AppRoot) {
	watchAppAPI(app.APIs)
	watchAppFlow(app.Flows)
	watchAppModel(app.Models)
	watchAppPlugin(app.Plugins)
	watchAppTable(app.Tables)
}

// watchAppTable 监听数据表格变更
func watchAppTable(rootTable string) {
	if !strings.HasPrefix(rootTable, "fs://") && strings.Contains(rootTable, "://") {
		return
	}
	root := strings.TrimPrefix(rootTable, "fs://")
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		log.Panicf("路径错误 %s %s", root, err)
	}

	go share.Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".json") {
			return
		}

		if op == "write" || op == "create" {
			script := share.GetAppFile(root, file)
			table.Load(string(script.Content), script.Name) // Reload
			api, has := gou.APIs["xiang.table"]
			if has {
				api.Reload()
			}

			log.Printf("数据表格 %s 已重新加载完毕", script.Name)

		} else if op == "remove" || op == "rename" {
			name := share.GetAppFileName(root, file)
			if _, has := gou.APIs[name]; has {
				delete(table.Tables, name)
				log.Printf("数据表格 %s 已经移除", name)
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			Stop(func() {
				log.Printf("服务器重启完毕")
				go Start()
			})
		}
	})
}

// watchAppAPI 监听API变更
func watchAppAPI(api string) {
	if !strings.HasPrefix(api, "fs://") && strings.Contains(api, "://") {
		return
	}
	root := strings.TrimPrefix(api, "fs://")
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		log.Panicf("路径错误 %s %s", root, err)
	}

	go share.Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".json") {
			return
		}

		if op == "write" || op == "create" {
			script := share.GetAppFile(root, file)
			gou.LoadAPI(string(script.Content), script.Name) // Reload
			log.Printf("API %s 已重新加载完毕", script.Name)

		} else if op == "remove" || op == "rename" {
			name := share.GetAppFileName(root, file)
			if _, has := gou.APIs[name]; has {
				delete(gou.APIs, name)
				log.Printf("API %s 已经移除", name)
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			Stop(func() {
				log.Printf("服务器重启完毕")
				go Start()
			})
		}
	})
}

// watchAppFlow 监听Flow变更
func watchAppFlow(flow string) {
	if !strings.HasPrefix(flow, "fs://") && strings.Contains(flow, "://") {
		return
	}
	root := strings.TrimPrefix(flow, "fs://")
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		log.Panicf("路径错误 %s %s", root, err)
	}
	go share.Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".json") && !strings.HasSuffix(file, ".js") {
			return
		}
		if strings.HasSuffix(file, ".js") {
			basName := share.GetAppFileBaseName(root, file)
			file = basName + ".flow.json"
		}
		if op == "write" || op == "create" {
			script := share.GetAppFile(root, file)
			gou.LoadFlow(string(script.Content), script.Name) // Reload
			log.Printf("Flow %s 已重新加载完毕", script.Name)
		} else if op == "remove" || op == "rename" {
			name := share.GetAppFileName(root, file)
			if _, has := gou.Flows[name]; has {
				delete(gou.Flows, name)
				log.Printf("Flow %s 已经移除", name)
			}
		}
	})
}

// watchAppModel 监听Model变更
func watchAppModel(model string) {
	if !strings.HasPrefix(model, "fs://") && strings.Contains(model, "://") {
		return
	}

	root := strings.TrimPrefix(model, "fs://")
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		log.Panicf("路径错误 %s %s", root, err)
	}
	go share.Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op == "write" || op == "create" {
			script := share.GetAppFile(root, file)
			gou.LoadModel(string(script.Content), script.Name) // Reload
			log.Printf("Model %s 已重新加载完毕", script.Name)
		} else if op == "remove" || op == "rename" {
			name := share.GetAppFileName(root, file)
			if _, has := gou.Models[name]; has {
				delete(gou.Models, name)
				log.Printf("Model %s 已经移除", name)
			}
		}
	})
}

// watchAppPlugin 监听Plugin变更
func watchAppPlugin(plugin string) {
	if !strings.HasPrefix(plugin, "fs://") && strings.Contains(plugin, "://") {
		return
	}
	root := strings.TrimPrefix(plugin, "fs://")
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		log.Panicf("路径错误 %s %s", root, err)
	}
	go share.Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".so") {
			return
		}

		if op == "write" || op == "create" {
			script := share.GetAppPluginFile(root, file)
			gou.LoadPlugin(script.File, script.Name) // Reload
			log.Printf("Plugin %s 已重新加载完毕", script.Name)
		} else if op == "remove" || op == "rename" {
			name := share.GetAppPluginFileName(root, file)
			if _, has := gou.Plugins[name]; has {
				delete(gou.Plugins, name)
				log.Printf("Plugin %s 已经移除", name)
			}
		}
	})
}
