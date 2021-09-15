package global

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou"
)

var shutdown = make(chan bool)
var shutdownComplete = make(chan bool)

// ServiceStart 启动服务
func ServiceStart() {
	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host:   Conf.Service.Host,
			Port:   Conf.Service.Port,
			Allows: Conf.Service.Allow,
			Root:   "/api",
		},
		&shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)
}

// ServiceStop 关闭服务
func ServiceStop(onComplete func()) {
	shutdown <- true
	<-shutdownComplete
	onComplete()
}

// WatchChanges 监听配置文件变更
func WatchChanges() {
	watchEngine(Conf.Path)
	watchApp(Conf.RootAPI, Conf.RootFLow, Conf.RootModel, Conf.RootPlugin)
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
	go Watch(filepath.Join(rootAbs, "flows"), func(op string, file string) {

		if !strings.HasSuffix(file, ".json") {
			return
		}

		if strings.HasSuffix(file, ".js") {
			basName := getFileBaseName(root, file)
			file = basName + ".flow.json"
		}

		if op == "write" || op == "create" {
			script := getFile(root, file)
			gou.LoadFlow(string(script.Content), "xiang."+script.Name) // Reload
			log.Printf("Flow %s 已重新加载完毕", "xiang."+script.Name)
		} else if op == "remove" || op == "rename" {
			name := "xiang." + getFileName(root, file)
			if _, has := gou.Flows[name]; has {
				delete(gou.Flows, name)
				log.Printf("Flow %s 已经移除", name)
			}
		}
	})

	// 监听 models
	go Watch(filepath.Join(rootAbs, "models"), func(op string, file string) {

		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op == "write" || op == "create" {
			script := getFile(root, file)
			gou.LoadModel(string(script.Content), "xiang."+script.Name) // Reload
			log.Printf("Model %s 已重新加载完毕", "xiang."+script.Name)
		} else if op == "remove" || op == "rename" {
			name := "xiang." + getFileName(root, file)
			if _, has := gou.Models[name]; has {
				delete(gou.Models, name)
				log.Printf("Model %s 已经移除", name)
			}
		}
	})

	// 监听 apis
	go Watch(filepath.Join(rootAbs, "apis"), func(op string, file string) {

		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op == "write" || op == "create" {
			script := getFile(root, file)
			gou.LoadAPI(string(script.Content), "xiang."+script.Name) // Reload
			log.Printf("API %s 已重新加载完毕", "xiang."+script.Name)
		} else if op == "remove" || op == "rename" {
			name := "xiang." + getFileName(root, file)
			if _, has := gou.APIs[name]; has {
				delete(gou.APIs, name)
				log.Printf("API %s 已经移除", name)
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			ServiceStop(func() {
				log.Printf("服务器重启完毕")
				go ServiceStart()
			})
		}
	})
}

// watchApp 监听应用目录文件变更
func watchApp(api string, flow string, model string, plugin string) {
	watchAppAPI(api)
	watchAppFlow(flow)
	watchAppModel(model)
	watchAppPlugin(plugin)
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

	go Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".json") {
			return
		}

		if op == "write" || op == "create" {
			script := getAppFile(root, file)
			gou.LoadAPI(string(script.Content), script.Name) // Reload
			log.Printf("API %s 已重新加载完毕", script.Name)

		} else if op == "remove" || op == "rename" {
			name := getAppFileName(root, file)
			if _, has := gou.APIs[name]; has {
				delete(gou.APIs, name)
				log.Printf("API %s 已经移除", name)
			}
		}

		// 重启服务器
		if op == "write" || op == "create" || op == "remove" || op == "rename" {
			ServiceStop(func() {
				log.Printf("服务器重启完毕")
				go ServiceStart()
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
	go Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".json") && !strings.HasSuffix(file, ".js") {
			return
		}
		if strings.HasSuffix(file, ".js") {
			basName := getAppFileBaseName(root, file)
			file = basName + ".flow.json"
		}
		if op == "write" || op == "create" {
			script := getAppFile(root, file)
			gou.LoadFlow(string(script.Content), script.Name) // Reload
			log.Printf("Flow %s 已重新加载完毕", script.Name)
		} else if op == "remove" || op == "rename" {
			name := getAppFileName(root, file)
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
	go Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op == "write" || op == "create" {
			script := getAppFile(root, file)
			gou.LoadModel(string(script.Content), script.Name) // Reload
			log.Printf("Model %s 已重新加载完毕", script.Name)
		} else if op == "remove" || op == "rename" {
			name := getAppFileName(root, file)
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
	go Watch(rootAbs, func(op string, file string) {
		if !strings.HasSuffix(file, ".so") {
			return
		}

		if op == "write" || op == "create" {
			script := getAppPluginFile(root, file)
			gou.LoadPlugin(script.File, script.Name) // Reload
			log.Printf("Plugin %s 已重新加载完毕", script.Name)
		} else if op == "remove" || op == "rename" {
			name := getAppPluginFileName(root, file)
			if _, has := gou.Plugins[name]; has {
				delete(gou.Plugins, name)
				log.Printf("Plugin %s 已经移除", name)
			}
		}
	})
}
