package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/service"
	"github.com/yaoapp/yao/share"
)

var startDebug = false
var startAlpha = false

var startCmd = &cobra.Command{
	Use:   "start",
	Short: L("Start Engine"),
	Long:  L("Start Engine"),
	Run: func(cmd *cobra.Command, args []string) {

		// recive interrupt signal
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		// defer service.Stop(func() { fmt.Println(L("Service stopped")) })
		Boot()

		if startDebug { // 强制 debug 模式启动
			config.Development()
		}

		mode := config.Conf.Mode
		err := engine.Load(config.Conf) // 加载脚本等
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
		port := fmt.Sprintf(":%d", config.Conf.Port)
		if port == ":80" {
			port = ""
		}

		host := config.Conf.Host
		if host == "0.0.0.0" {
			host = "127.0.0.1"
		}

		fmt.Println(color.WhiteString("\n---------------------------------"))
		fmt.Println(color.WhiteString(share.App.Name), color.WhiteString(share.App.Version), mode)
		fmt.Println(color.WhiteString("---------------------------------"))
		if !share.BUILDIN {
			root, _ := filepath.Abs(config.Conf.Root)
			fmt.Println(color.WhiteString(L("Root")), color.GreenString(" %s", root))
		}

		fmt.Println(color.WhiteString(L("Frontend")), color.GreenString(" http://%s%s/", host, port))
		fmt.Println(color.WhiteString(L("Dashboard")), color.GreenString(" http://%s%s/xiang/login/admin", host, port))
		fmt.Println(color.WhiteString(L("API")), color.GreenString(" http://%s%s/api", host, port))
		fmt.Println(color.WhiteString(L("Listening")), color.GreenString(" %s:%d", config.Conf.Host, config.Conf.Port))

		// development mode
		if mode == "development" {
			printApis(false)
			printTasks(false)
			printSchedules(false)

			// Watching
			fmt.Println(color.WhiteString("\n---------------------------------"))
			fmt.Println(color.WhiteString(L("Watching")))
			fmt.Println(color.WhiteString("---------------------------------"))
			service.Watch(config.Conf)
		}

		if mode == "production" {
			printApis(true)
			printTasks(true)
			printSchedules(true)
		}

		// Start server
		go service.Start()
		fmt.Println(color.GreenString(L("✨LISTENING✨")))

		for {
			select {
			case <-interrupt:
				service.Stop(func() {
					fmt.Println(color.GreenString(L("✨STOPPED✨")))
				})
				return
			}
		}
	},
}

func printSchedules(silent bool) {
	if silent {
		for name, sch := range gou.Schedules {
			process := fmt.Sprintf("Process: %s", sch.Process)
			if sch.TaskName != "" {
				process = fmt.Sprintf("Task: %s", sch.TaskName)
			}
			log.Info("[Schedule] %s %s %s %s", sch.Schedule, name, sch.Name, process)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Schedules List (%d)"), len(gou.Schedules)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for name, sch := range gou.Schedules {
		process := fmt.Sprintf("Process: %s", sch.Process)
		if sch.TaskName != "" {
			process = fmt.Sprintf("Task: %s", sch.TaskName)
		}
		fmt.Printf(color.CyanString("[Schedule] %s %s", sch.Schedule, name))
		fmt.Printf(color.WhiteString("\t%s\t%s\n", sch.Name, process))
	}
}

func printTasks(silent bool) {

	if silent {
		for _, t := range task.Tasks {
			log.Info("[Task] %s workers:%d", t.Option.Name, t.Option.WorkerNums)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Tasks List (%d)"), len(task.Tasks)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for _, t := range task.Tasks {
		fmt.Printf(color.CyanString("[Task] %s", t.Option.Name))
		fmt.Printf(color.WhiteString("\t workers: %d\n", t.Option.WorkerNums))
	}
}

func printApis(silent bool) {

	if silent {
		for _, api := range gou.APIs {
			if len(api.HTTP.Paths) <= 0 {
				continue
			}
			log.Info("[API] %s(%d)", api.Name, len(api.HTTP.Paths))
			for _, p := range api.HTTP.Paths {
				log.Info("%s %s %s", p.Method, filepath.Join("/api", api.HTTP.Group, p.Path), p.Process)
			}
		}
		for name, upgrader := range websocket.Upgraders { // WebSocket
			log.Info("[WebSocket] GET  /websocket/%s process:%s", name, upgrader.Process)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("API List")))
	fmt.Println(color.WhiteString("---------------------------------"))

	for _, api := range gou.APIs { // API信息
		if len(api.HTTP.Paths) <= 0 {
			continue
		}

		fmt.Printf(color.CyanString("\n%s(%d)\n", api.Name, len(api.HTTP.Paths)))
		for _, p := range api.HTTP.Paths {
			fmt.Println(
				colorMehtod(p.Method),
				color.WhiteString(filepath.Join("/api", api.HTTP.Group, p.Path)),
				"\tprocess:", p.Process)
		}
	}

	fmt.Printf(color.CyanString("\n%s(%d)\n", "WebSocket", len(websocket.Upgraders)))
	for name, upgrader := range websocket.Upgraders { // WebSocket
		fmt.Println(
			colorMehtod("GET"),
			color.WhiteString(filepath.Join("/websocket", name)),
			"\tprocess:", upgrader.Process)
	}
}

func colorMehtod(method string) string {
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		return color.GreenString("GET")
	case "POST":
		return color.YellowString("POST")
	default:
		return color.WhiteString(method)
	}
}

func init() {
	startCmd.PersistentFlags().BoolVarP(&startDebug, "debug", "", false, L("Development mode"))
	startCmd.PersistentFlags().BoolVarP(&startAlpha, "alpha", "", false, L("Enabled unstable features"))
}
