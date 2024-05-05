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
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	ischedule "github.com/yaoapp/yao/schedule"
	"github.com/yaoapp/yao/service"
	"github.com/yaoapp/yao/setup"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/studio"
	itask "github.com/yaoapp/yao/task"
)

var startDebug = false
var startDisableWatching = false

var startCmd = &cobra.Command{
	Use:   "start",
	Short: L("Start Engine"),
	Long:  L("Start Engine"),
	Run: func(cmd *cobra.Command, args []string) {

		defer share.SessionStop()
		defer plugin.KillAll()

		// recive interrupt signal
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		Boot()

		// Setup
		if setup.Check() {
			go setup.Start()
			select {
			case <-setup.Done:
				setup.Stop()
				Boot()
				break
			case <-setup.Canceled:
				os.Exit(1)
				break
			}
		}

		// force debug
		if startDebug {
			config.Development()
		}

		// load the application engine
		err := engine.Load(config.Conf, engine.LoadOption{Action: "start"})
		if err != nil {
			fmt.Println(color.RedString(L("Load: %s"), err.Error()))
			os.Exit(1)
		}

		port := fmt.Sprintf(":%d", config.Conf.Port)
		if port == ":80" {
			port = ""
		}

		// variables for the service
		fs, err := fs.Get("system")
		if err != nil {
			fmt.Println(color.RedString(L("FileSystem: %s"), err.Error()))
			os.Exit(1)
		}

		mode := config.Conf.Mode
		host := config.Conf.Host
		dataRoot := fs.Root()
		runtimeMode := config.Conf.Runtime.Mode

		fmt.Println(color.WhiteString("\n--------------------------------------------"))
		fmt.Println(
			color.WhiteString(strings.TrimPrefix(share.App.Name, "::")),
			color.WhiteString(share.App.Version),
			mode,
		)
		fmt.Println(color.WhiteString("--------------------------------------------"))
		if !share.BUILDIN {
			root, _ := filepath.Abs(config.Conf.Root)
			fmt.Println(color.WhiteString(L("Root")), color.GreenString(" %s", root))
		}

		root, _ := adminRoot()
		urls := []string{fmt.Sprintf("http://%s:%s", host, port)}
		if host == "0.0.0.0" {
			urls, _ = setup.URLs(config.Conf)
		}

		fmt.Println(color.WhiteString(L("Runtime")), color.GreenString(" %s", runtimeMode))
		fmt.Println(color.WhiteString(L("Data")), color.GreenString(" %s", dataRoot))
		fmt.Println(color.WhiteString(L("Listening")), color.GreenString(" %s:%d", config.Conf.Host, config.Conf.Port))
		for _, url := range urls {
			fmt.Println(color.CyanString("\n%s", url))
			fmt.Println(color.WhiteString("--------------------------"))
			fmt.Println(color.WhiteString(L("Frontend")), color.GreenString(" %s", url))
			fmt.Println(color.WhiteString(L("Dashboard")), color.GreenString(" %s/%s/login/admin", url, strings.Trim(root, "/")))
			fmt.Println(color.WhiteString(L("API")), color.GreenString(" %s/api", url))
		}

		// print the messages under the development mode
		if mode == "development" {

			// Start Studio Server
			go func() {

				err = studio.Load(config.Conf)
				if err != nil {
					// fmt.Println(color.RedString(L("Studio Load: %s"), err.Error()))
					log.Error("Studio Load: %s", err.Error())
					return
				}

				err := studio.Start(config.Conf)
				if err != nil {
					log.Error("Studio Start: %s", err.Error())
					return
				}
			}()
			defer studio.Stop()

			printApis(false)
			printTasks(false)
			printSchedules(false)
			printConnectors(false)
			printStores(false)
			printStudio(false, host)
		}

		// Start Tasks
		itask.Start()
		defer itask.Stop()

		// Start Schedules
		ischedule.Start()
		defer ischedule.Stop()

		// Start HTTP Server
		srv, err := service.Start(config.Conf)
		defer func() {
			service.Stop(srv)
			fmt.Println(color.GreenString(L("✨EXITED✨")))
		}()

		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		// Start watching
		watchDone := make(chan uint8, 1)
		if mode == "development" && !startDisableWatching {
			// fmt.Println(color.WhiteString("\n---------------------------------"))
			// fmt.Println(color.WhiteString(L("Watching")))
			// fmt.Println(color.WhiteString("---------------------------------"))
			go service.Watch(srv, watchDone)
		}

		// Print the messages under the production mode
		if mode == "production" {
			printApis(true)
			printTasks(true)
			printSchedules(true)
			printConnectors(true)
			printStores(true)
		}

		for {
			select {
			case v := <-srv.Event():

				switch v {
				case http.READY:
					fmt.Println(color.GreenString(L("✨LISTENING✨")))
					break

				case http.CLOSED:
					fmt.Println(color.GreenString(L("✨EXITED✨")))
					watchDone <- 1
					return

				case http.ERROR:
					color.Red("Fatal: check the error information in the log")
					watchDone <- 1
					return

				default:
					fmt.Println("Signal:", v)
				}

			case <-interrupt:
				watchDone <- 1
				return
			}
		}
	},
}

func adminRoot() (string, int) {
	adminRoot := "/yao/"
	if share.App.AdminRoot != "" {
		root := strings.TrimPrefix(share.App.AdminRoot, "/")
		root = strings.TrimSuffix(root, "/")
		adminRoot = fmt.Sprintf("/%s/", root)
	}
	adminRootLen := len(adminRoot)
	return adminRoot, adminRootLen
}

func printConnectors(silent bool) {

	if len(connector.Connectors) == 0 {
		return
	}

	if silent {
		for name := range connector.Connectors {
			log.Info("[Connector] %s loaded", name)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Connectors List (%d)"), len(connector.Connectors)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for name := range connector.Connectors {
		fmt.Printf(color.CyanString("[Connector]"))
		fmt.Printf(color.WhiteString(" %s\t loaded\n", name))
	}
}

func printStores(silent bool) {
	if len(store.Pools) == 0 {
		return
	}

	if silent {
		for name := range store.Pools {
			log.Info("[Store] %s loaded", name)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Stores List (%d)"), len(connector.Connectors)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for name := range store.Pools {
		fmt.Printf(color.CyanString("[Store]"))
		fmt.Printf(color.WhiteString(" %s\t loaded\n", name))
	}
}

func printStudio(silent bool, host string) {

	if silent {
		log.Info("[Studio] http://%s:%d", host, config.Conf.Studio.Port)
		if config.Conf.Studio.Auto {
			log.Info("[Studio] Secret: %s", config.Conf.Studio.Secret)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Yao Studio Server")))
	fmt.Println(color.WhiteString("---------------------------------"))
	fmt.Printf(color.CyanString("HOST  : "))
	fmt.Printf(color.WhiteString(" %s\n", config.Conf.Host))
	fmt.Printf(color.CyanString("PORT  : "))
	fmt.Printf(color.WhiteString(" %d\n", config.Conf.Studio.Port))
	if config.Conf.Studio.Auto {
		fmt.Printf(color.CyanString("SECRET: "))
		fmt.Printf(color.WhiteString(" %s\n", config.Conf.Studio.Secret))
	}
}

func printSchedules(silent bool) {

	if len(schedule.Schedules) == 0 {
		return
	}

	if silent {
		for name, sch := range schedule.Schedules {
			process := fmt.Sprintf("Process: %s", sch.Process)
			if sch.TaskName != "" {
				process = fmt.Sprintf("Task: %s", sch.TaskName)
			}
			log.Info("[Schedule] %s %s %s %s", sch.Schedule, name, sch.Name, process)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Schedules List (%d)"), len(schedule.Schedules)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for name, sch := range schedule.Schedules {
		process := fmt.Sprintf("Process: %s", sch.Process)
		if sch.TaskName != "" {
			process = fmt.Sprintf("Task: %s", sch.TaskName)
		}
		fmt.Printf(color.CyanString("[Schedule] %s %s", sch.Schedule, name))
		fmt.Printf(color.WhiteString("\t%s\t%s\n", sch.Name, process))
	}
}

func printTasks(silent bool) {

	if len(task.Tasks) == 0 {
		return
	}

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
		for _, api := range api.APIs {
			if len(api.HTTP.Paths) <= 0 {
				continue
			}
			log.Info("[API] %s(%d)", api.ID, len(api.HTTP.Paths))
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

	for _, api := range api.APIs { // API信息
		if len(api.HTTP.Paths) <= 0 {
			continue
		}

		deprecated := ""
		if strings.HasPrefix(api.ID, "xiang.") {
			deprecated = " WILL BE DEPRECATED"
		}

		fmt.Printf("%s%s\n", color.CyanString("\n%s(%d)", api.ID, len(api.HTTP.Paths)), color.RedString(deprecated))
		for _, p := range api.HTTP.Paths {
			fmt.Println(
				colorMehtod(p.Method),
				color.WhiteString(filepath.Join("/api", api.HTTP.Group, p.Path)),
				"\tprocess:", p.Process)
		}
	}

	if len(websocket.Upgraders) > 0 {
		fmt.Printf(color.CyanString("\n%s(%d)\n", "WebSocket", len(websocket.Upgraders)))
		for name, upgrader := range websocket.Upgraders { // WebSocket
			fmt.Println(
				colorMehtod("GET"),
				color.WhiteString(filepath.Join("/websocket", name)),
				"\tprocess:", upgrader.Process)
		}
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
	startCmd.PersistentFlags().BoolVarP(&startDisableWatching, "disable-watching", "", false, L("Disable watching"))
}
